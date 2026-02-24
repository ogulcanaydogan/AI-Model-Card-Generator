package unit

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/yapay/ai-model-card-generator/pkg/core"
	"github.com/yapay/ai-model-card-generator/pkg/extractors"
)

func TestParseWandBModelID(t *testing.T) {
	t.Parallel()

	ref, err := extractors.ParseWandBModelID("acme/support/abc123")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if ref.Entity != "acme" || ref.Project != "support" || ref.RunID != "abc123" {
		t.Fatalf("unexpected parse result: %+v", ref)
	}
}

func TestParseWandBModelIDInvalid(t *testing.T) {
	t.Parallel()

	_, err := extractors.ParseWandBModelID("acme/support")
	if err == nil {
		t.Fatalf("expected parse error")
	}
	if !strings.Contains(err.Error(), "expected format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWandBExtractorFixtureMapping(t *testing.T) {
	t.Parallel()
	repoRoot := mustRepoRoot(t)
	extractor := extractors.NewWeightsAndBiasesExtractor("", "", filepath.Join(repoRoot, "tests", "fixtures", "wandb", "run_fixture.json"))

	metadata, err := extractor.Extract(context.Background(), core.ModelRef{Source: "wandb", ID: "acme/support/abc123"})
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}

	if metadata.Name != "support-router-v2" {
		t.Fatalf("name = %q, want support-router-v2", metadata.Name)
	}
	if metadata.Owner != "acme" {
		t.Fatalf("owner = %q, want acme", metadata.Owner)
	}
	if metadata.IntendedUse == "" || metadata.Limitations == "" {
		t.Fatalf("expected intended use and limitations from fixture config")
	}
	if metadata.Metrics["accuracy"] != 0.93 {
		t.Fatalf("accuracy metric missing or wrong: %+v", metadata.Metrics)
	}
	if _, ok := metadata.Metrics["status"]; ok {
		t.Fatalf("non-numeric summary metric should not be included")
	}
}

func TestWandBExtractorMissingAPIKey(t *testing.T) {
	t.Parallel()
	extractor := extractors.NewWeightsAndBiasesExtractor("", "", "")

	_, err := extractor.Extract(context.Background(), core.ModelRef{Source: "wandb", ID: "acme/support/abc123"})
	if err == nil {
		t.Fatalf("expected missing API key error")
	}
	if !strings.Contains(err.Error(), "WANDB_API_KEY") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func mustRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
