package unit

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/yapay/ai-model-card-generator/pkg/core"
)

func TestLoadBatchManifestParses(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "manifest.yaml")
	content := `version: v1
defaults:
  template: standard
  formats: [json]
jobs:
  - id: job-1
    source: custom
    model: demo
    uri: tests/fixtures/custom_metadata.json
    eval_file: examples/eval_sample.csv
`
	if err := os.WriteFile(manifestPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	manifest, err := core.LoadBatchManifest(manifestPath)
	if err != nil {
		t.Fatalf("load batch manifest: %v", err)
	}
	if manifest.Version != "v1" {
		t.Fatalf("unexpected manifest version: %s", manifest.Version)
	}
	if len(manifest.Jobs) != 1 || manifest.Jobs[0].ID != "job-1" {
		t.Fatalf("unexpected manifest jobs: %+v", manifest.Jobs)
	}
}

func TestMergeBatchJobValidation(t *testing.T) {
	defaultTemplateFile := filepath.Join("tests", "fixtures", "batch", "custom-template.tmpl")
	defaults := core.BatchDefaults{
		Template:     "standard",
		TemplateFile: defaultTemplateFile,
		Formats:      []string{"json"},
		Language:     "en",
		Compliance:   []string{"eu-ai-act"},
		OutDir:       filepath.Join(t.TempDir(), "out"),
	}

	if _, err := core.MergeBatchJob(defaults, core.BatchJob{
		ID:       "wandb-bad",
		Source:   "wandb",
		Model:    "bad-format",
		EvalFile: "examples/eval_sample.csv",
	}, "./artifacts"); err == nil {
		t.Fatalf("expected wandb format validation error")
	}

	if _, err := core.MergeBatchJob(defaults, core.BatchJob{
		ID:       "bad-source",
		Source:   "foobar",
		Model:    "demo",
		EvalFile: "examples/eval_sample.csv",
	}, "./artifacts"); err == nil {
		t.Fatalf("expected unsupported source validation error")
	}

	if _, err := core.MergeBatchJob(defaults, core.BatchJob{
		ID:       "mlflow-bad",
		Source:   "mlflow",
		Model:    "abc123",
		EvalFile: "examples/eval_sample.csv",
	}, "./artifacts"); err == nil {
		t.Fatalf("expected mlflow format validation error")
	}

	if _, err := core.MergeBatchJob(defaults, core.BatchJob{
		ID:       "custom-missing-uri",
		Source:   "custom",
		Model:    "demo",
		EvalFile: "examples/eval_sample.csv",
	}, "./artifacts"); err == nil {
		t.Fatalf("expected custom uri validation error")
	}

	merged, err := core.MergeBatchJob(defaults, core.BatchJob{
		ID:       "custom-ok",
		Source:   "custom",
		Model:    "demo",
		URI:      "tests/fixtures/custom_metadata.json",
		EvalFile: "examples/eval_sample.csv",
	}, "./artifacts")
	if err != nil {
		t.Fatalf("merge valid batch job: %v", err)
	}
	if merged.TemplateFile != defaultTemplateFile {
		t.Fatalf("expected template_file to be inherited, got %q", merged.TemplateFile)
	}
}

func TestRunBatchContinueOnError(t *testing.T) {
	repoRoot := unitRepoRoot(t)
	evalPath := filepath.Join(repoRoot, "examples", "eval_sample.csv")

	pipeline := newBatchTestPipeline(t)
	report, err := pipeline.RunBatch(context.Background(), core.BatchRunOptions{
		Manifest: core.BatchManifest{
			Version: "v1",
			Defaults: core.BatchDefaults{
				Template:   "standard",
				Formats:    []string{"json"},
				Language:   "en",
				Compliance: []string{"eu-ai-act"},
			},
			Jobs: []core.BatchJob{
				{ID: "job-a", Source: "custom", Model: "ok-a", URI: "unused", EvalFile: evalPath},
				{ID: "job-b", Source: "custom", Model: "fail", URI: "unused", EvalFile: evalPath},
				{ID: "job-c", Source: "custom", Model: "ok-c", URI: "unused", EvalFile: evalPath},
			},
		},
		OutDir:   t.TempDir(),
		Workers:  3,
		FailFast: false,
	})
	if err != nil {
		t.Fatalf("run batch: %v", err)
	}

	if report.Total != 3 || report.Succeeded != 2 || report.Failed != 1 {
		t.Fatalf("unexpected batch summary: %+v", report)
	}
	if report.Jobs[0].ID != "job-a" || report.Jobs[1].ID != "job-b" || report.Jobs[2].ID != "job-c" {
		t.Fatalf("job order changed: %+v", report.Jobs)
	}
	if report.Jobs[0].Status != "succeeded" || report.Jobs[1].Status != "failed" || report.Jobs[2].Status != "succeeded" {
		t.Fatalf("unexpected job statuses: %+v", report.Jobs)
	}
}

func TestRunBatchFailFastSkipsPending(t *testing.T) {
	repoRoot := unitRepoRoot(t)
	evalPath := filepath.Join(repoRoot, "examples", "eval_sample.csv")

	pipeline := newBatchTestPipeline(t)
	report, err := pipeline.RunBatch(context.Background(), core.BatchRunOptions{
		Manifest: core.BatchManifest{
			Version: "v1",
			Defaults: core.BatchDefaults{
				Template:   "standard",
				Formats:    []string{"json"},
				Language:   "en",
				Compliance: []string{"eu-ai-act"},
			},
			Jobs: []core.BatchJob{
				{ID: "job-a", Source: "custom", Model: "fail", URI: "unused", EvalFile: evalPath},
				{ID: "job-b", Source: "custom", Model: "ok-b", URI: "unused", EvalFile: evalPath},
			},
		},
		OutDir:   t.TempDir(),
		Workers:  1,
		FailFast: true,
	})
	if err != nil {
		t.Fatalf("run batch: %v", err)
	}

	if report.Failed != 1 {
		t.Fatalf("expected one failed job, got %+v", report)
	}
	if report.Jobs[0].Status != "failed" {
		t.Fatalf("expected first job failed, got %s", report.Jobs[0].Status)
	}
	if report.Jobs[1].Status != "skipped" {
		t.Fatalf("expected second job skipped, got %s", report.Jobs[1].Status)
	}
}

func newBatchTestPipeline(t *testing.T) core.Pipeline {
	t.Helper()
	return core.Pipeline{
		Extractors: map[string]core.Extractor{
			"custom": batchStubExtractor{},
		},
		Analyzers: nil,
		Generators: map[string]core.Generator{
			"json": batchStubGenerator{},
		},
		ComplianceCheckers: map[string]core.ComplianceChecker{
			"eu-ai-act": batchStubChecker{},
		},
		DefaultTemplatePath: "templates",
	}
}

type batchStubExtractor struct{}

func (batchStubExtractor) Name() string { return "batch-stub-extractor" }

func (batchStubExtractor) Extract(_ context.Context, ref core.ModelRef) (core.ModelMetadata, error) {
	if ref.ID == "fail" {
		return core.ModelMetadata{}, errors.New("forced extractor failure")
	}
	return core.ModelMetadata{Name: ref.ID, Owner: "batch-test"}, nil
}

type batchStubGenerator struct{}

func (batchStubGenerator) Format() string { return "json" }

func (batchStubGenerator) Generate(_ context.Context, _ core.ModelCard, _ string, outPath string) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outPath, []byte("{}"), 0o644)
}

type batchStubChecker struct{}

func (batchStubChecker) Framework() string { return "eu-ai-act" }

func (batchStubChecker) Check(_ context.Context, _ core.ModelCard, _ core.CheckOptions) (core.ComplianceReport, error) {
	return core.ComplianceReport{
		Framework: "eu-ai-act",
		Score:     100,
		Status:    "pass",
	}, nil
}

func unitRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
