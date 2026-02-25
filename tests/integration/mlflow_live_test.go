package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIMLflowLive(t *testing.T) {
	repoRoot := mustRepoRoot(t)
	trackingURI := strings.TrimSpace(os.Getenv("MLFLOW_TRACKING_URI"))
	runID := strings.TrimSpace(os.Getenv("MLFLOW_LIVE_RUN_ID"))
	if trackingURI == "" || runID == "" {
		t.Skip("MLFLOW_TRACKING_URI and MLFLOW_LIVE_RUN_ID are required for live mlflow integration test")
	}

	outDir := filepath.Join(t.TempDir(), "mlflow-live-artifacts")
	genOut, err := runCLIWithEnv(repoRoot, nil,
		"generate",
		"--model", "run:"+runID,
		"--source", "mlflow",
		"--eval-file", filepath.Join("examples", "eval_sample.csv"),
		"--formats", "json",
		"--out-dir", outDir,
	)
	if err != nil {
		t.Fatalf("live mlflow generate failed: %v\n%s", err, genOut)
	}

	jsonPath := filepath.Join(outDir, "model_card.json")
	if _, err := os.Stat(jsonPath); err != nil {
		t.Fatalf("expected generated json: %v", err)
	}

	checkOut, err := runCLI(repoRoot,
		"check",
		"--framework", "eu-ai-act",
		"--input", jsonPath,
		"--strict", "false",
	)
	if err != nil {
		t.Fatalf("live check failed: %v\n%s", err, checkOut)
	}
}
