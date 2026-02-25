package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yapay/ai-model-card-generator/pkg/analyzers"
	"github.com/yapay/ai-model-card-generator/pkg/compliance"
	"github.com/yapay/ai-model-card-generator/pkg/core"
	"github.com/yapay/ai-model-card-generator/pkg/extractors"
	"github.com/yapay/ai-model-card-generator/pkg/generators"
)

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
  mcg generate --model <id> --source <hf|mlflow|wandb|custom> --template <standard|eu-ai-act|minimal> --eval-file <path> --formats <md,json,html,pdf> --out-dir <path> --lang <en> --compliance <eu-ai-act>
  mcg validate --schema schemas/model-card.v1.json --input <model-card.json|md>
  mcg check --framework <eu-ai-act|nist|iso42001> --input <model-card.json> --strict <false|true>
`)
}

func runGenerate(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	model := fs.String("model", "", "Model ID (e.g. bert-base-uncased for hf, entity/project/run_id for wandb, run:<run_id> for mlflow)")
	source := fs.String("source", "hf", "Model source: hf|mlflow|wandb|custom")
	templateName := fs.String("template", "standard", "Template: standard|eu-ai-act|minimal")
	evalFile := fs.String("eval-file", "", "Evaluation CSV path")
	formats := fs.String("formats", "md,json,pdf", "Comma-separated output formats")
	outDir := fs.String("out-dir", "./artifacts", "Output directory")
	lang := fs.String("lang", "en", "Output language")
	complianceArg := fs.String("compliance", "eu-ai-act", "Comma-separated compliance frameworks")
	uri := fs.String("uri", "", "Custom source metadata JSON path")
	hfBaseURL := fs.String("hf-base-url", "https://huggingface.co", "Hugging Face API base URL")
	if err := fs.Parse(args); err != nil {
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

	fairnessScript := os.Getenv("MCG_FAIRNESS_SCRIPT")
	if strings.TrimSpace(fairnessScript) == "" {
		fairnessScript = filepath.Join("scripts", "fairness_metrics.py")
	}
	pythonBin := os.Getenv("MCG_PYTHON_BIN")
	if strings.TrimSpace(pythonBin) == "" {
		pythonBin = "python3"
	}
	wandbBaseURL := os.Getenv("WANDB_BASE_URL")
	wandbAPIKey := os.Getenv("WANDB_API_KEY")
	wandbFixture := os.Getenv("MCG_WANDB_FIXTURE")
	mlflowTrackingURI := os.Getenv("MLFLOW_TRACKING_URI")
	mlflowTrackingToken := os.Getenv("MLFLOW_TRACKING_TOKEN")
	mlflowTrackingUsername := os.Getenv("MLFLOW_TRACKING_USERNAME")
	mlflowTrackingPassword := os.Getenv("MLFLOW_TRACKING_PASSWORD")
	mlflowFixture := os.Getenv("MCG_MLFLOW_FIXTURE")

	pipeline := core.Pipeline{
		Extractors: map[string]core.Extractor{
			"hf":     extractors.NewHuggingFaceExtractor(*hfBaseURL),
			"mlflow": extractors.NewMLflowExtractor(mlflowTrackingURI, mlflowTrackingToken, mlflowTrackingUsername, mlflowTrackingPassword, mlflowFixture),
			"wandb":  extractors.NewWeightsAndBiasesExtractor(wandbBaseURL, wandbAPIKey, wandbFixture),
			"custom": &extractors.CustomExtractor{},
		},
		Analyzers: []core.Analyzer{
			&analyzers.PerformanceAnalyzer{},
			&analyzers.FairnessAnalyzer{PythonBin: pythonBin, ScriptPath: fairnessScript},
			&analyzers.BiasAnalyzer{},
		},
		Generators: map[string]core.Generator{
			"md":   &generators.MarkdownGenerator{},
			"html": &generators.HTMLGenerator{},
			"pdf":  &generators.PDFGenerator{},
			"json": &generators.JSONGenerator{},
		},
		ComplianceCheckers: map[string]core.ComplianceChecker{
			"eu-ai-act": &compliance.EUAIActChecker{},
			"nist":      &compliance.NISTChecker{},
			"iso42001":  &compliance.ISO42001Checker{},
		},
		DefaultTemplatePath: "templates",
	}

	opts := core.GenerateOptions{
		Ref: core.ModelRef{
			Source: *source,
			ID:     *model,
			URI:    *uri,
		},
		EvalFile:             *evalFile,
		Template:             *templateName,
		Formats:              splitCSV(*formats),
		OutDir:               *outDir,
		Language:             *lang,
		ComplianceFrameworks: splitCSV(*complianceArg),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	card, err := pipeline.Generate(ctx, opts)
	if err != nil {
		return err
	}

	fmt.Printf("Generated model card for %s\n", card.Metadata.Name)
	for format, path := range card.Artifacts.GeneratedFiles {
		fmt.Printf("- %s: %s\n", format, path)
	}
	fmt.Printf("- compliance: %s\n", card.Artifacts.CompliancePath)
	return nil
}

func runValidate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	schema := fs.String("schema", filepath.Join("schemas", "model-card.v1.json"), "JSON schema path")
	input := fs.String("input", "", "Input model card (.json or .md)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*input) == "" {
		return fmt.Errorf("--input is required")
	}

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

func runCheck(args []string) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	framework := fs.String("framework", "eu-ai-act", "Framework: eu-ai-act|nist|iso42001")
	input := fs.String("input", "", "Model card JSON path")
	strict := fs.Bool("strict", false, "Fail process when required gaps exist")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*input) == "" {
		return fmt.Errorf("--input is required")
	}

	card, err := core.LoadModelCard(*input)
	if err != nil {
		return err
	}

	checkers := map[string]core.ComplianceChecker{
		"eu-ai-act": &compliance.EUAIActChecker{},
		"nist":      &compliance.NISTChecker{},
		"iso42001":  &compliance.ISO42001Checker{},
	}

	frameworks := splitCSV(*framework)
	reports := make([]core.ComplianceReport, 0, len(frameworks))
	for _, fw := range frameworks {
		checker, ok := checkers[fw]
		if !ok {
			return fmt.Errorf("unsupported framework: %s", fw)
		}
		report, err := checker.Check(context.Background(), card, core.CheckOptions{Strict: *strict})
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

	if core.StrictComplianceExit(reports, *strict) {
		return errors.New("strict compliance check failed")
	}
	return nil
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
