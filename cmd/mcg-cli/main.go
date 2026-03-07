package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/yapay/ai-model-card-generator/pkg/analyzers"
	"github.com/yapay/ai-model-card-generator/pkg/compliance"
	"github.com/yapay/ai-model-card-generator/pkg/core"
	"github.com/yapay/ai-model-card-generator/pkg/extractors"
	"github.com/yapay/ai-model-card-generator/pkg/generators"
	apisrv "github.com/yapay/ai-model-card-generator/pkg/server"
	cardtemplates "github.com/yapay/ai-model-card-generator/pkg/templates"
)

const toolVersion = "v1.0.0"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "generate":
		if err := runGenerate(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "generate failed: %v\n", err)
			os.Exit(1)
		}
	case "validate":
		if err := runValidate(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "validate failed: %v\n", err)
			os.Exit(1)
		}
	case "check":
		if err := runCheck(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "check failed: %v\n", err)
			os.Exit(1)
		}
	case "serve":
		if err := runServe(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "serve failed: %v\n", err)
			os.Exit(1)
		}
	case "template":
		if err := runTemplate(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "template failed: %v\n", err)
			os.Exit(1)
		}
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`mcg - AI Model Card Generator

Usage:
  mcg generate --model <id> --source <hf|mlflow|wandb|custom> --template <standard|eu-ai-act|minimal> --template-file <path> --eval-file <path> --formats <md,json,html,pdf> --out-dir <path> --lang <en> --compliance <eu-ai-act>
  mcg generate --batch <manifest.yaml> --workers <n> --fail-fast <true|false> --out-dir <path>
  mcg validate --schema schemas/model-card.v1.json --input <model-card.json|md>
  mcg check --framework <eu-ai-act|nist|iso42001> --input <model-card.json> --strict <false|true>
  mcg serve --addr :8080 --read-timeout 30s --write-timeout 180s
  mcg template init --name <name> --out <path> --base <standard|minimal|eu-ai-act>
  mcg template validate --input <template.tmpl>
  mcg template preview --input <template.tmpl> --card <model_card.json> --out <preview.md>
`)
}

func runGenerate(args []string) (err error) {
	auditor := core.NewAuditLoggerFromEnv()
	record, recErr := core.NewAuditRecord("cli", "generate", toolVersion, map[string]any{"args": args})
	if recErr != nil {
		return recErr
	}
	started := time.Now()
	defer func() {
		record.DurationMS = time.Since(started).Milliseconds()
		if err != nil {
			record.Status = "failed"
			record.Error = err.Error()
		} else {
			record.Status = "succeeded"
		}
		if auditErr := auditor.Append(record); auditErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; audit write failed: %v", err, auditErr)
				return
			}
			err = fmt.Errorf("audit write failed: %w", auditErr)
		}
	}()

	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	model := fs.String("model", "", "Model ID (e.g. bert-base-uncased for hf, entity/project/run_id for wandb, run:<run_id> for mlflow)")
	source := fs.String("source", "hf", "Model source: hf|mlflow|wandb|custom")
	templateName := fs.String("template", "standard", "Template: standard|eu-ai-act|minimal")
	templateFile := fs.String("template-file", "", "Custom template file path (.tmpl). Overrides --template when set")
	evalFile := fs.String("eval-file", "", "Evaluation CSV path")
	formats := fs.String("formats", "md,json,pdf", "Comma-separated output formats")
	outDir := fs.String("out-dir", "./artifacts", "Output directory")
	batchManifest := fs.String("batch", "", "Batch manifest YAML path")
	workers := fs.Int("workers", 4, "Parallel workers for batch mode")
	failFastValue := fs.String("fail-fast", "false", "Stop batch mode at first failed job (true|false)")
	lang := fs.String("lang", "en", "Output language")
	complianceArg := fs.String("compliance", "eu-ai-act", "Comma-separated compliance frameworks")
	uri := fs.String("uri", "", "Custom source metadata JSON path")
	hfBaseURL := fs.String("hf-base-url", "https://huggingface.co", "Hugging Face API base URL")
	if err := fs.Parse(args); err != nil {
		return err
	}

	pipeline := buildPipeline(*hfBaseURL)

	if strings.TrimSpace(*batchManifest) != "" {
		failFast, err := strconv.ParseBool(strings.TrimSpace(*failFastValue))
		if err != nil {
			return fmt.Errorf("invalid --fail-fast value %q: use true or false", *failFastValue)
		}
		record.Source = "batch"
		record.ModelRef = *batchManifest
		record.Frameworks = splitCSV(*complianceArg)

		reportPath, err := runGenerateBatch(pipeline, *batchManifest, *outDir, *workers, failFast)
		if reportPath != "" {
			record.ArtifactPaths = map[string]string{"batch_report": reportPath}
		}
		return err
	}

	if strings.TrimSpace(*model) == "" {
		return fmt.Errorf("--model is required")
	}
	if strings.EqualFold(strings.TrimSpace(*source), "wandb") {
		if _, err := extractors.ParseWandBModelID(*model); err != nil {
			return fmt.Errorf("invalid --model for wandb source: %w", err)
		}
	}
	if strings.EqualFold(strings.TrimSpace(*source), "mlflow") {
		if _, err := extractors.ParseMLflowModelID(*model); err != nil {
			return fmt.Errorf("invalid --model for mlflow source: %w", err)
		}
	}
	if strings.TrimSpace(*evalFile) == "" {
		return fmt.Errorf("--eval-file is required")
	}

	opts := core.GenerateOptions{
		Ref: core.ModelRef{
			Source: *source,
			ID:     *model,
			URI:    *uri,
		},
		EvalFile:             *evalFile,
		Template:             *templateName,
		TemplateFile:         *templateFile,
		Formats:              splitCSV(*formats),
		OutDir:               *outDir,
		Language:             *lang,
		ComplianceFrameworks: splitCSV(*complianceArg),
	}
	record.Source = strings.TrimSpace(*source)
	record.ModelRef = strings.TrimSpace(*model)
	record.Frameworks = opts.ComplianceFrameworks

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	card, err := pipeline.Generate(ctx, opts)
	if err != nil {
		return err
	}

	record.ArtifactPaths = map[string]string{}
	for format, path := range card.Artifacts.GeneratedFiles {
		record.ArtifactPaths[format] = path
	}
	record.ArtifactPaths["compliance_report"] = card.Artifacts.CompliancePath

	fmt.Printf("Generated model card for %s\n", card.Metadata.Name)
	for format, path := range card.Artifacts.GeneratedFiles {
		fmt.Printf("- %s: %s\n", format, path)
	}
	fmt.Printf("- compliance: %s\n", card.Artifacts.CompliancePath)
	return nil
}

func runGenerateBatch(pipeline core.Pipeline, manifestPath, outDir string, workers int, failFast bool) (string, error) {
	manifest, err := core.LoadBatchManifest(manifestPath)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	report, err := pipeline.RunBatch(ctx, core.BatchRunOptions{
		Manifest: manifest,
		OutDir:   outDir,
		Workers:  workers,
		FailFast: failFast,
	})
	if err != nil {
		return "", err
	}

	reportPath := filepath.Join(outDir, "batch_report.json")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", core.Wrap("create batch out-dir", err)
	}
	if err := core.WriteBatchReport(reportPath, report); err != nil {
		return "", err
	}

	fmt.Printf("Batch generation finished: total=%d succeeded=%d failed=%d duration_ms=%d\n", report.Total, report.Succeeded, report.Failed, report.DurationMs)
	fmt.Printf("- report: %s\n", reportPath)
	for _, job := range report.Jobs {
		fmt.Printf("- %s [%s]", job.ID, job.Status)
		if job.Error != "" {
			fmt.Printf(" error=%s", job.Error)
		}
		fmt.Println()
	}

	if report.HasFailures() {
		return reportPath, fmt.Errorf("batch completed with %d failed job(s)", report.Failed)
	}
	return reportPath, nil
}

func runValidate(args []string) (err error) {
	auditor := core.NewAuditLoggerFromEnv()
	record, recErr := core.NewAuditRecord("cli", "validate", toolVersion, map[string]any{"args": args})
	if recErr != nil {
		return recErr
	}
	started := time.Now()
	defer func() {
		record.DurationMS = time.Since(started).Milliseconds()
		if err != nil {
			record.Status = "failed"
			record.Error = err.Error()
		} else {
			record.Status = "succeeded"
		}
		if auditErr := auditor.Append(record); auditErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; audit write failed: %v", err, auditErr)
				return
			}
			err = fmt.Errorf("audit write failed: %w", auditErr)
		}
	}()

	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	schema := fs.String("schema", filepath.Join("schemas", "model-card.v1.json"), "JSON schema path")
	input := fs.String("input", "", "Input model card (.json or .md)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*input) == "" {
		return fmt.Errorf("--input is required")
	}
	record.ModelRef = *input

	ext := strings.ToLower(filepath.Ext(*input))
	switch ext {
	case ".json":
		if err := core.ValidateJSONSchema(*schema, *input); err != nil {
			return err
		}
	case ".md":
		if err := validateMarkdownCard(*input); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported input extension: %s", ext)
	}

	fmt.Println("Validation succeeded")
	return nil
}

func runCheck(args []string) (err error) {
	auditor := core.NewAuditLoggerFromEnv()
	record, recErr := core.NewAuditRecord("cli", "check", toolVersion, map[string]any{"args": args})
	if recErr != nil {
		return recErr
	}
	started := time.Now()
	defer func() {
		record.DurationMS = time.Since(started).Milliseconds()
		if err != nil {
			record.Status = "failed"
			record.Error = err.Error()
		} else {
			record.Status = "succeeded"
		}
		if auditErr := auditor.Append(record); auditErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; audit write failed: %v", err, auditErr)
				return
			}
			err = fmt.Errorf("audit write failed: %w", auditErr)
		}
	}()

	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	framework := fs.String("framework", "eu-ai-act", "Framework: eu-ai-act|nist|iso42001")
	input := fs.String("input", "", "Model card JSON path")
	strictValue := fs.String("strict", "false", "Fail process when required gaps exist (true|false)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*input) == "" {
		return fmt.Errorf("--input is required")
	}
	strict, err := strconv.ParseBool(strings.TrimSpace(*strictValue))
	if err != nil {
		return fmt.Errorf("invalid --strict value %q: use true or false", *strictValue)
	}
	record.ModelRef = *input

	card, err := core.LoadModelCard(*input)
	if err != nil {
		return err
	}

	checkers := complianceCheckers()
	frameworks := splitCSV(*framework)
	record.Frameworks = frameworks

	reports := make([]core.ComplianceReport, 0, len(frameworks))
	for _, fw := range frameworks {
		checker, ok := checkers[fw]
		if !ok {
			return fmt.Errorf("unsupported framework: %s", fw)
		}
		report, err := checker.Check(context.Background(), card, core.CheckOptions{Strict: strict})
		if err != nil {
			return err
		}
		reports = append(reports, report)
	}

	payload, err := json.MarshalIndent(reports, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(payload))

	if core.StrictComplianceExit(reports, strict) {
		return errors.New("strict compliance check failed")
	}
	return nil
}

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", ":8080", "HTTP listen address")
	readTimeout := fs.Duration("read-timeout", 30*time.Second, "HTTP read timeout")
	writeTimeout := fs.Duration("write-timeout", 180*time.Second, "HTTP write timeout")
	hfBaseURL := fs.String("hf-base-url", "https://huggingface.co", "Hugging Face API base URL")
	if err := fs.Parse(args); err != nil {
		return err
	}

	api := &apisrv.APIServer{
		Pipeline:    buildPipeline(*hfBaseURL),
		SchemaPath:  filepath.Join("schemas", "model-card.v1.json"),
		AuditLogger: core.NewAuditLoggerFromEnv(),
		ToolVersion: toolVersion,
	}

	server := &http.Server{
		Addr:         *addr,
		Handler:      api.Handler(),
		ReadTimeout:  *readTimeout,
		WriteTimeout: *writeTimeout,
	}
	fmt.Printf("mcg serve listening on %s\n", *addr)
	err := server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func runTemplate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("template subcommand is required: init|validate|preview")
	}

	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "init":
		return runTemplateInit(args[1:])
	case "validate":
		return runTemplateValidate(args[1:])
	case "preview":
		return runTemplatePreview(args[1:])
	default:
		return fmt.Errorf("unknown template subcommand: %s (expected init|validate|preview)", args[0])
	}
}

func runTemplateInit(args []string) error {
	fs := flag.NewFlagSet("template init", flag.ContinueOnError)
	name := fs.String("name", "", "Template display name")
	outPath := fs.String("out", "", "Output template path")
	base := fs.String("base", "standard", "Base template: standard|minimal|eu-ai-act")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if strings.TrimSpace(*name) == "" {
		return fmt.Errorf("--name is required")
	}
	if strings.TrimSpace(*outPath) == "" {
		return fmt.Errorf("--out is required")
	}
	if err := cardtemplates.InitTemplate(*name, *outPath, *base); err != nil {
		return err
	}
	fmt.Printf("Template created at %s\n", filepath.Clean(*outPath))
	return nil
}

func runTemplateValidate(args []string) error {
	fs := flag.NewFlagSet("template validate", flag.ContinueOnError)
	input := fs.String("input", "", "Template file path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*input) == "" {
		return fmt.Errorf("--input is required")
	}
	if err := cardtemplates.ValidateTemplateFile(*input); err != nil {
		return err
	}
	fmt.Println("Template validation succeeded")
	return nil
}

func runTemplatePreview(args []string) error {
	fs := flag.NewFlagSet("template preview", flag.ContinueOnError)
	input := fs.String("input", "", "Template file path")
	cardPath := fs.String("card", "", "Model card JSON path")
	outPath := fs.String("out", "", "Output markdown path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*input) == "" {
		return fmt.Errorf("--input is required")
	}
	if strings.TrimSpace(*cardPath) == "" {
		return fmt.Errorf("--card is required")
	}
	if strings.TrimSpace(*outPath) == "" {
		return fmt.Errorf("--out is required")
	}

	card, err := core.LoadModelCard(*cardPath)
	if err != nil {
		return err
	}
	if err := cardtemplates.WriteTemplatePreview(*input, *outPath, card); err != nil {
		return err
	}
	fmt.Printf("Template preview written to %s\n", filepath.Clean(*outPath))
	return nil
}

func buildPipeline(hfBaseURL string) core.Pipeline {
	fairnessScript := os.Getenv("MCG_FAIRNESS_SCRIPT")
	if strings.TrimSpace(fairnessScript) == "" {
		fairnessScript = filepath.Join("scripts", "fairness_metrics.py")
	}
	pythonBin := os.Getenv("MCG_PYTHON_BIN")
	if strings.TrimSpace(pythonBin) == "" {
		pythonBin = "python3"
	}
	carbonPythonBin := os.Getenv("MCG_CARBON_PYTHON_BIN")
	if strings.TrimSpace(carbonPythonBin) == "" {
		carbonPythonBin = pythonBin
	}
	carbonScript := os.Getenv("MCG_CARBON_SCRIPT")
	if strings.TrimSpace(carbonScript) == "" {
		carbonScript = filepath.Join("scripts", "carbon_metrics.py")
	}
	carbonFixture := os.Getenv("MCG_CARBON_FIXTURE")
	wandbBaseURL := os.Getenv("WANDB_BASE_URL")
	wandbAPIKey := os.Getenv("WANDB_API_KEY")
	wandbFixture := os.Getenv("MCG_WANDB_FIXTURE")
	mlflowTrackingURI := os.Getenv("MLFLOW_TRACKING_URI")
	mlflowTrackingToken := os.Getenv("MLFLOW_TRACKING_TOKEN")
	mlflowTrackingUsername := os.Getenv("MLFLOW_TRACKING_USERNAME")
	mlflowTrackingPassword := os.Getenv("MLFLOW_TRACKING_PASSWORD")
	mlflowFixture := os.Getenv("MCG_MLFLOW_FIXTURE")

	return core.Pipeline{
		Extractors: map[string]core.Extractor{
			"hf":     extractors.NewHuggingFaceExtractor(hfBaseURL),
			"mlflow": extractors.NewMLflowExtractor(mlflowTrackingURI, mlflowTrackingToken, mlflowTrackingUsername, mlflowTrackingPassword, mlflowFixture),
			"wandb":  extractors.NewWeightsAndBiasesExtractor(wandbBaseURL, wandbAPIKey, wandbFixture),
			"custom": &extractors.CustomExtractor{},
		},
		Analyzers: []core.Analyzer{
			&analyzers.PerformanceAnalyzer{},
			&analyzers.FairnessAnalyzer{PythonBin: pythonBin, ScriptPath: fairnessScript},
			&analyzers.BiasAnalyzer{},
			&analyzers.CarbonAnalyzer{PythonBin: carbonPythonBin, ScriptPath: carbonScript, FixturePath: carbonFixture},
		},
		Generators: map[string]core.Generator{
			"md":   &generators.MarkdownGenerator{},
			"html": &generators.HTMLGenerator{},
			"pdf":  &generators.PDFGenerator{},
			"json": &generators.JSONGenerator{},
		},
		ComplianceCheckers:  complianceCheckers(),
		DefaultTemplatePath: "templates",
	}
}

func complianceCheckers() map[string]core.ComplianceChecker {
	return map[string]core.ComplianceChecker{
		"eu-ai-act": &compliance.EUAIActChecker{},
		"nist":      &compliance.NISTChecker{},
		"iso42001":  &compliance.ISO42001Checker{},
	}
}

func validateMarkdownCard(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := strings.ToLower(string(data))
	requiredSections := []string{"## metadata", "## performance", "## fairness", "## compliance"}
	for _, section := range requiredSections {
		if !strings.Contains(content, section) {
			return fmt.Errorf("markdown model card missing required section: %s", section)
		}
	}
	return nil
}

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		n := strings.TrimSpace(strings.ToLower(p))
		if n != "" {
			out = append(out, n)
		}
	}
	return out
}
