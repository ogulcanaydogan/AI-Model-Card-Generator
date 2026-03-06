package unit

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/yapay/ai-model-card-generator/pkg/analyzers"
	"github.com/yapay/ai-model-card-generator/pkg/core"
)

func TestCarbonAnalyzerFixture(t *testing.T) {
	t.Parallel()
	repoRoot := mustRepoRoot(t)

	analyzer := &analyzers.CarbonAnalyzer{
		FixturePath: filepath.Join(repoRoot, "tests", "fixtures", "carbon", "carbon_fixture.json"),
	}

	result, err := analyzer.Analyze(context.Background(), core.AnalysisInput{
		EvalFile: filepath.Join(repoRoot, "examples", "eval_sample.csv"),
	})
	if err != nil {
		t.Fatalf("analyze returned error: %v", err)
	}
	if result.Carbon == nil {
		t.Fatalf("expected carbon estimate")
	}
	if result.Carbon.Method != "fixture" {
		t.Fatalf("method = %q, want fixture", result.Carbon.Method)
	}
	if result.Carbon.EstimatedKgCO2e != 0.123456 {
		t.Fatalf("estimated_kg_co2e = %v, want 0.123456", result.Carbon.EstimatedKgCO2e)
	}
}

func TestCarbonAnalyzerBridgeFailureFallsBackToUnavailable(t *testing.T) {
	t.Parallel()
	repoRoot := mustRepoRoot(t)

	analyzer := &analyzers.CarbonAnalyzer{
		PythonBin:  "python3",
		ScriptPath: filepath.Join(repoRoot, "tests", "fixtures", "missing_carbon_script.py"),
	}

	result, err := analyzer.Analyze(context.Background(), core.AnalysisInput{
		EvalFile: filepath.Join(repoRoot, "examples", "eval_sample.csv"),
	})
	if err != nil {
		t.Fatalf("analyze should not fail on bridge failure: %v", err)
	}
	if result.Carbon == nil {
		t.Fatalf("expected carbon estimate")
	}
	if result.Carbon.Method != "unavailable" {
		t.Fatalf("method = %q, want unavailable", result.Carbon.Method)
	}
	if len(result.RiskNotes) == 0 {
		t.Fatalf("expected risk note for unavailable carbon evidence")
	}
}

func TestCarbonAnalyzerInvalidFixtureFails(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	fixturePath := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(fixturePath, []byte("{not-json}"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	analyzer := &analyzers.CarbonAnalyzer{FixturePath: fixturePath}
	_, err := analyzer.Analyze(context.Background(), core.AnalysisInput{EvalFile: "ignored.csv"})
	if err == nil {
		t.Fatalf("expected fixture parsing error")
	}
}
