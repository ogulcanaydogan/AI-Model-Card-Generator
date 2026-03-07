package unit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yapay/ai-model-card-generator/pkg/core"
	"github.com/yapay/ai-model-card-generator/pkg/generators"
)

func TestPipelineTemplateFileTakesPrecedence(t *testing.T) {
	evalPath := filepath.Join(t.TempDir(), "eval.csv")
	if err := os.WriteFile(evalPath, []byte("y_true,y_pred,group\n1,1,a\n"), 0o644); err != nil {
		t.Fatalf("write eval file: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "override.tmpl")
	templateContent := `# UNIT-OVERRIDE {{ .Metadata.Name }}

## Metadata
owner={{ .Metadata.Owner }}

## Performance
acc={{ printf "%.4f" .Performance.Accuracy }}

## Fairness
dp={{ printf "%.4f" .Fairness.DemographicParityDiff }}

## Compliance
{{ range .Compliance }}- {{ .Framework }}
{{ end }}`
	if err := os.WriteFile(templatePath, []byte(templateContent), 0o644); err != nil {
		t.Fatalf("write template file: %v", err)
	}

	outDir := filepath.Join(t.TempDir(), "out")
	pipeline := core.Pipeline{
		Extractors: map[string]core.Extractor{
			"custom": precedenceExtractor{},
		},
		Analyzers: nil,
		Generators: map[string]core.Generator{
			"md":   &generators.MarkdownGenerator{},
			"json": &generators.JSONGenerator{},
		},
		ComplianceCheckers: map[string]core.ComplianceChecker{
			"eu-ai-act": precedenceChecker{},
		},
		DefaultTemplatePath: "templates",
	}

	_, err := pipeline.Generate(context.Background(), core.GenerateOptions{
		Ref: core.ModelRef{
			Source: "custom",
			ID:     "demo",
			URI:    "unused",
		},
		EvalFile:             evalPath,
		Template:             "does-not-exist",
		TemplateFile:         templatePath,
		Formats:              []string{"md", "json"},
		OutDir:               outDir,
		Language:             "en",
		ComplianceFrameworks: []string{"eu-ai-act"},
	})
	if err != nil {
		t.Fatalf("pipeline generate with template file override: %v", err)
	}

	md, err := os.ReadFile(filepath.Join(outDir, "model_card.md"))
	if err != nil {
		t.Fatalf("read markdown output: %v", err)
	}
	if !strings.Contains(string(md), "# UNIT-OVERRIDE") {
		t.Fatalf("expected override template output, got: %s", string(md))
	}
}

type precedenceExtractor struct{}

func (precedenceExtractor) Name() string { return "precedence-extractor" }

func (precedenceExtractor) Extract(_ context.Context, _ core.ModelRef) (core.ModelMetadata, error) {
	return core.ModelMetadata{Name: "precedence-model", Owner: "owner"}, nil
}

type precedenceChecker struct{}

func (precedenceChecker) Framework() string { return "eu-ai-act" }

func (precedenceChecker) Check(_ context.Context, _ core.ModelCard, _ core.CheckOptions) (core.ComplianceReport, error) {
	return core.ComplianceReport{
		Framework: "eu-ai-act",
		Score:     100,
		Status:    "pass",
	}, nil
}
