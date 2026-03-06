package unit

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/yapay/ai-model-card-generator/pkg/core"
)

func TestPipelineRetriesExternalExtractor(t *testing.T) {
	evalPath := filepath.Join(t.TempDir(), "eval.csv")
	if err := os.WriteFile(evalPath, []byte("y_true,y_pred,group\n1,1,a\n"), 0o644); err != nil {
		t.Fatalf("write eval file: %v", err)
	}

	extractor := &retryExtractor{failuresBeforeSuccess: 2}
	pipeline := core.Pipeline{
		Extractors: map[string]core.Extractor{
			"hf": extractor,
		},
		Generators: map[string]core.Generator{
			"json": retryJSONGenerator{},
		},
		ComplianceCheckers: map[string]core.ComplianceChecker{
			"eu-ai-act": retryComplianceChecker{},
		},
		DefaultTemplatePath: "templates",
	}

	_, err := pipeline.Generate(context.Background(), core.GenerateOptions{
		Ref: core.ModelRef{
			Source: "hf",
			ID:     "demo",
		},
		EvalFile:             evalPath,
		Template:             "standard",
		Formats:              []string{"json"},
		OutDir:               t.TempDir(),
		Language:             "en",
		ComplianceFrameworks: []string{"eu-ai-act"},
	})
	if err != nil {
		t.Fatalf("pipeline generate: %v", err)
	}
	if extractor.calls != 3 {
		t.Fatalf("expected 3 extractor calls (1 + 2 retries), got %d", extractor.calls)
	}
}

func TestPipelineDoesNotRetryCustomExtractor(t *testing.T) {
	evalPath := filepath.Join(t.TempDir(), "eval.csv")
	if err := os.WriteFile(evalPath, []byte("y_true,y_pred,group\n1,1,a\n"), 0o644); err != nil {
		t.Fatalf("write eval file: %v", err)
	}

	extractor := &retryExtractor{failuresBeforeSuccess: 2}
	pipeline := core.Pipeline{
		Extractors: map[string]core.Extractor{
			"custom": extractor,
		},
		Generators: map[string]core.Generator{
			"json": retryJSONGenerator{},
		},
		ComplianceCheckers: map[string]core.ComplianceChecker{
			"eu-ai-act": retryComplianceChecker{},
		},
		DefaultTemplatePath: "templates",
	}

	_, err := pipeline.Generate(context.Background(), core.GenerateOptions{
		Ref: core.ModelRef{
			Source: "custom",
			ID:     "demo",
		},
		EvalFile:             evalPath,
		Template:             "standard",
		Formats:              []string{"json"},
		OutDir:               t.TempDir(),
		Language:             "en",
		ComplianceFrameworks: []string{"eu-ai-act"},
	})
	if err == nil {
		t.Fatalf("expected generate to fail without retry on custom source")
	}
	if extractor.calls != 1 {
		t.Fatalf("expected 1 extractor call for custom source, got %d", extractor.calls)
	}
}

type retryExtractor struct {
	calls                 int
	failuresBeforeSuccess int
}

func (e *retryExtractor) Name() string { return "retry-extractor" }

func (e *retryExtractor) Extract(_ context.Context, _ core.ModelRef) (core.ModelMetadata, error) {
	e.calls++
	if e.calls <= e.failuresBeforeSuccess {
		return core.ModelMetadata{}, errors.New("transient failure")
	}
	return core.ModelMetadata{Name: "retry-model", Owner: "retry-owner"}, nil
}

type retryJSONGenerator struct{}

func (retryJSONGenerator) Format() string { return "json" }

func (retryJSONGenerator) Generate(_ context.Context, _ core.ModelCard, _ string, outPath string) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outPath, []byte("{}"), 0o644)
}

type retryComplianceChecker struct{}

func (retryComplianceChecker) Framework() string { return "eu-ai-act" }

func (retryComplianceChecker) Check(_ context.Context, _ core.ModelCard, _ core.CheckOptions) (core.ComplianceReport, error) {
	return core.ComplianceReport{
		Framework: "eu-ai-act",
		Score:     100,
		Status:    "pass",
	}, nil
}
