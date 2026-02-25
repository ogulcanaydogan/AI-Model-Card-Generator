package unit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yapay/ai-model-card-generator/pkg/core"
	"github.com/yapay/ai-model-card-generator/pkg/extractors"
)

func TestParseMLflowModelID(t *testing.T) {
	t.Parallel()
	runID, err := extractors.ParseMLflowModelID("run:abc123")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if runID != "abc123" {
		t.Fatalf("runID = %q, want abc123", runID)
	}
}

func TestParseMLflowModelIDInvalid(t *testing.T) {
	t.Parallel()
	_, err := extractors.ParseMLflowModelID("abc123")
	if err == nil {
		t.Fatalf("expected parse error")
	}
	if !strings.Contains(err.Error(), "run:<run_id>") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMLflowExtractorFixtureMapping(t *testing.T) {
	t.Parallel()
	repoRoot := mustRepoRoot(t)
	extractor := extractors.NewMLflowExtractor("", "", "", "", filepath.Join(repoRoot, "tests", "fixtures", "mlflow", "run_get_fixture.json"))

	metadata, err := extractor.Extract(context.Background(), core.ModelRef{Source: "mlflow", ID: "run:abc123"})
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}

	if metadata.Name != "support-router-mlflow" {
		t.Fatalf("name = %q, want support-router-mlflow", metadata.Name)
	}
	if metadata.Owner != "ml-team" {
		t.Fatalf("owner = %q, want ml-team", metadata.Owner)
	}
	if metadata.License != "apache-2.0" {
		t.Fatalf("license = %q, want apache-2.0", metadata.License)
	}
	if metadata.Metrics["accuracy"] != 0.92 {
		t.Fatalf("missing accuracy metric: %+v", metadata.Metrics)
	}
	if metadata.Metrics["max_depth"] != 8 {
		t.Fatalf("expected numeric param in metrics map: %+v", metadata.Metrics)
	}
}

func TestMLflowExtractorMissingRunInFixture(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	fixturePath := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(fixturePath, []byte(`{"status":"ok"}`), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	extractor := extractors.NewMLflowExtractor("", "", "", "", fixturePath)
	_, err := extractor.Extract(context.Background(), core.ModelRef{Source: "mlflow", ID: "run:abc123"})
	if err == nil {
		t.Fatalf("expected fixture error")
	}
	if !strings.Contains(err.Error(), "fixture missing run payload") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMLflowExtractorMissingTrackingURI(t *testing.T) {
	t.Parallel()
	extractor := extractors.NewMLflowExtractor("", "", "", "", "")

	_, err := extractor.Extract(context.Background(), core.ModelRef{Source: "mlflow", ID: "run:abc123"})
	if err == nil {
		t.Fatalf("expected missing tracking uri error")
	}
	if !strings.Contains(err.Error(), "MLFLOW_TRACKING_URI") {
		t.Fatalf("unexpected error: %v", err)
	}
}
