package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultBatchWorkers = 4
	batchManifestV1     = "v1"
)

// BatchManifest describes batch generation jobs.
type BatchManifest struct {
	Version  string        `json:"version" yaml:"version"`
	Defaults BatchDefaults `json:"defaults,omitempty" yaml:"defaults,omitempty"`
	Jobs     []BatchJob    `json:"jobs" yaml:"jobs"`
}

// BatchDefaults defines default values applied to each job.
type BatchDefaults struct {
	Template   string   `json:"template,omitempty" yaml:"template,omitempty"`
	Formats    []string `json:"formats,omitempty" yaml:"formats,omitempty"`
	Language   string   `json:"lang,omitempty" yaml:"lang,omitempty"`
	Compliance []string `json:"compliance,omitempty" yaml:"compliance,omitempty"`
	OutDir     string   `json:"out_dir,omitempty" yaml:"out_dir,omitempty"`
}

// BatchJob defines one generate execution in batch mode.
type BatchJob struct {
	ID         string   `json:"id" yaml:"id"`
	Source     string   `json:"source" yaml:"source"`
	Model      string   `json:"model" yaml:"model"`
	EvalFile   string   `json:"eval_file" yaml:"eval_file"`
	URI        string   `json:"uri,omitempty" yaml:"uri,omitempty"`
	Template   string   `json:"template,omitempty" yaml:"template,omitempty"`
	Formats    []string `json:"formats,omitempty" yaml:"formats,omitempty"`
	Language   string   `json:"lang,omitempty" yaml:"lang,omitempty"`
	Compliance []string `json:"compliance,omitempty" yaml:"compliance,omitempty"`
	OutDir     string   `json:"out_dir,omitempty" yaml:"out_dir,omitempty"`
}

// BatchRunOptions configures batch generation execution.
type BatchRunOptions struct {
	Manifest BatchManifest
	OutDir   string
	Workers  int
	FailFast bool
}

// BatchReport summarizes one batch run.
type BatchReport struct {
	Total      int              `json:"total"`
	Succeeded  int              `json:"succeeded"`
	Failed     int              `json:"failed"`
	DurationMs int64            `json:"duration_ms"`
	Jobs       []BatchJobReport `json:"jobs"`
}

// BatchJobReport stores one job execution result.
type BatchJobReport struct {
	ID             string            `json:"id"`
	Source         string            `json:"source,omitempty"`
	Model          string            `json:"model,omitempty"`
	Status         string            `json:"status"`
	Error          string            `json:"error,omitempty"`
	DurationMs     int64             `json:"duration_ms,omitempty"`
	ArtifactPaths  map[string]string `json:"artifact_paths,omitempty"`
	CompliancePath string            `json:"compliance_path,omitempty"`
}

type batchTask struct {
	index int
	opts  GenerateOptions
}

// LoadBatchManifest reads and validates a batch manifest YAML.
func LoadBatchManifest(path string) (BatchManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return BatchManifest{}, Wrap("read batch manifest", err)
	}
	var manifest BatchManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return BatchManifest{}, Wrap("parse batch manifest", err)
	}
	if strings.TrimSpace(manifest.Version) == "" {
		manifest.Version = batchManifestV1
	}
	if err := ValidateBatchManifest(manifest); err != nil {
		return BatchManifest{}, err
	}
	return manifest, nil
}

// ValidateBatchManifest validates static manifest constraints.
func ValidateBatchManifest(manifest BatchManifest) error {
	if !strings.EqualFold(strings.TrimSpace(manifest.Version), batchManifestV1) {
		return fmt.Errorf("unsupported batch manifest version: %s", manifest.Version)
	}
	if len(manifest.Jobs) == 0 {
		return fmt.Errorf("batch manifest must include at least one job")
	}

	seenIDs := map[string]struct{}{}
	for i, job := range manifest.Jobs {
		id := strings.TrimSpace(job.ID)
		if id == "" {
			return fmt.Errorf("jobs[%d].id is required", i)
		}
		if _, ok := seenIDs[id]; ok {
			return fmt.Errorf("jobs[%d].id duplicated: %s", i, id)
		}
		seenIDs[id] = struct{}{}
	}
	return nil
}

// MergeBatchJob merges defaults and validates one job for generation.
func MergeBatchJob(defaults BatchDefaults, job BatchJob, fallbackOutDir string) (GenerateOptions, error) {
	source := strings.ToLower(strings.TrimSpace(job.Source))
	model := strings.TrimSpace(job.Model)
	evalFile := strings.TrimSpace(job.EvalFile)
	uri := strings.TrimSpace(job.URI)

	if source == "" {
		return GenerateOptions{}, fmt.Errorf("job %q source is required", job.ID)
	}
	if model == "" {
		return GenerateOptions{}, fmt.Errorf("job %q model is required", job.ID)
	}
	if evalFile == "" {
		return GenerateOptions{}, fmt.Errorf("job %q eval_file is required", job.ID)
	}
	if !isSupportedBatchSource(source) {
		return GenerateOptions{}, fmt.Errorf("job %q has unsupported source: %s", job.ID, source)
	}
	if source == "custom" && uri == "" {
		return GenerateOptions{}, fmt.Errorf("job %q custom source requires uri", job.ID)
	}
	if source == "wandb" && !isValidWandBModelID(model) {
		return GenerateOptions{}, fmt.Errorf("job %q wandb model must be <entity>/<project>/<run_id>", job.ID)
	}
	if source == "mlflow" && !isValidMLflowModelID(model) {
		return GenerateOptions{}, fmt.Errorf("job %q mlflow model must be run:<run_id>", job.ID)
	}

	template := strings.TrimSpace(job.Template)
	if template == "" {
		template = strings.TrimSpace(defaults.Template)
	}
	if template == "" {
		template = "standard"
	}

	formats := cloneStringSlice(job.Formats)
	if len(formats) == 0 {
		formats = cloneStringSlice(defaults.Formats)
	}
	if len(formats) == 0 {
		formats = []string{"md", "json", "pdf"}
	}

	compliance := cloneStringSlice(job.Compliance)
	if len(compliance) == 0 {
		compliance = cloneStringSlice(defaults.Compliance)
	}
	if len(compliance) == 0 {
		compliance = []string{"eu-ai-act"}
	}

	lang := strings.TrimSpace(job.Language)
	if lang == "" {
		lang = strings.TrimSpace(defaults.Language)
	}
	if lang == "" {
		lang = "en"
	}

	outDir := strings.TrimSpace(job.OutDir)
	if outDir == "" {
		outRoot := strings.TrimSpace(defaults.OutDir)
		if outRoot == "" {
			outRoot = strings.TrimSpace(fallbackOutDir)
		}
		if outRoot == "" {
			outRoot = "./artifacts"
		}
		outDir = filepath.Join(outRoot, strings.TrimSpace(job.ID))
	}

	return GenerateOptions{
		Ref: ModelRef{
			Source: source,
			ID:     model,
			URI:    uri,
		},
		EvalFile:             evalFile,
		Template:             template,
		Formats:              formats,
		OutDir:               outDir,
		Language:             lang,
		ComplianceFrameworks: compliance,
	}, nil
}

// RunBatch executes batch jobs with bounded concurrency and stable output order.
func (p *Pipeline) RunBatch(ctx context.Context, opts BatchRunOptions) (BatchReport, error) {
	if err := ValidateBatchManifest(opts.Manifest); err != nil {
		return BatchReport{}, err
	}

	workers := opts.Workers
	if workers <= 0 {
		workers = defaultBatchWorkers
	}

	rootOutDir := strings.TrimSpace(opts.OutDir)
	if rootOutDir == "" {
		rootOutDir = strings.TrimSpace(opts.Manifest.Defaults.OutDir)
	}
	if rootOutDir == "" {
		rootOutDir = "./artifacts"
	}
	if err := os.MkdirAll(rootOutDir, 0o755); err != nil {
		return BatchReport{}, Wrap("create batch output dir", err)
	}

	report := BatchReport{
		Total: len(opts.Manifest.Jobs),
		Jobs:  make([]BatchJobReport, len(opts.Manifest.Jobs)),
	}
	for i, job := range opts.Manifest.Jobs {
		report.Jobs[i] = BatchJobReport{
			ID:     strings.TrimSpace(job.ID),
			Source: strings.ToLower(strings.TrimSpace(job.Source)),
			Model:  strings.TrimSpace(job.Model),
			Status: "pending",
		}
	}

	started := time.Now()

	tasks := make([]batchTask, 0, len(opts.Manifest.Jobs))
	for i, job := range opts.Manifest.Jobs {
		generateOpts, err := MergeBatchJob(opts.Manifest.Defaults, job, rootOutDir)
		if err != nil {
			report.Jobs[i].Status = "failed"
			report.Jobs[i].Error = err.Error()
			if opts.FailFast {
				report.markPendingSkipped("skipped due fail-fast cancellation")
				report.computeSummary(time.Since(started))
				return report, nil
			}
			continue
		}
		tasks = append(tasks, batchTask{index: i, opts: generateOpts})
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var mu sync.Mutex
	var wg sync.WaitGroup
	jobCh := make(chan batchTask)

	worker := func() {
		defer wg.Done()
		for task := range jobCh {
			if opts.FailFast && runCtx.Err() != nil {
				mu.Lock()
				if report.Jobs[task.index].Status == "pending" {
					report.Jobs[task.index].Status = "skipped"
					report.Jobs[task.index].Error = "skipped due fail-fast cancellation"
				}
				mu.Unlock()
				continue
			}

			jobStart := time.Now()
			card, err := p.Generate(runCtx, task.opts)
			duration := time.Since(jobStart).Milliseconds()

			mu.Lock()
			if err != nil {
				report.Jobs[task.index].Status = "failed"
				report.Jobs[task.index].Error = err.Error()
				report.Jobs[task.index].DurationMs = duration
				if opts.FailFast {
					cancel()
				}
			} else {
				report.Jobs[task.index].Status = "succeeded"
				report.Jobs[task.index].DurationMs = duration
				report.Jobs[task.index].ArtifactPaths = card.Artifacts.GeneratedFiles
				report.Jobs[task.index].CompliancePath = card.Artifacts.CompliancePath
			}
			mu.Unlock()
		}
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go worker()
	}

	go func() {
		defer close(jobCh)
		for _, task := range tasks {
			select {
			case <-runCtx.Done():
				return
			case jobCh <- task:
			}
		}
	}()

	wg.Wait()
	if opts.FailFast {
		report.markPendingSkipped("skipped due fail-fast cancellation")
	}
	report.computeSummary(time.Since(started))
	return report, nil
}

// WriteBatchReport writes the final batch run report to disk.
func WriteBatchReport(path string, report BatchReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return Wrap("marshal batch report", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return Wrap("write batch report", err)
	}
	return nil
}

// HasFailures returns true when one or more jobs fail.
func (r BatchReport) HasFailures() bool {
	return r.Failed > 0
}

func (r *BatchReport) markPendingSkipped(reason string) {
	for i := range r.Jobs {
		if r.Jobs[i].Status == "pending" {
			r.Jobs[i].Status = "skipped"
			r.Jobs[i].Error = reason
		}
	}
}

func (r *BatchReport) computeSummary(duration time.Duration) {
	r.Succeeded = 0
	r.Failed = 0
	for _, job := range r.Jobs {
		switch job.Status {
		case "succeeded":
			r.Succeeded++
		case "failed":
			r.Failed++
		}
	}
	r.DurationMs = duration.Milliseconds()
}

func isSupportedBatchSource(source string) bool {
	switch source {
	case "hf", "wandb", "mlflow", "custom":
		return true
	default:
		return false
	}
}

func isValidWandBModelID(model string) bool {
	parts := strings.Split(strings.TrimSpace(model), "/")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if strings.TrimSpace(p) == "" {
			return false
		}
	}
	return true
}

func isValidMLflowModelID(model string) bool {
	m := strings.TrimSpace(model)
	if !strings.HasPrefix(m, "run:") {
		return false
	}
	return strings.TrimSpace(strings.TrimPrefix(m, "run:")) != ""
}

func cloneStringSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, v := range in {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
