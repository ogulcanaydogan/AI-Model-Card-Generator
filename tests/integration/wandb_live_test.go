package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIWandBLive(t *testing.T) {
	repoRoot := mustRepoRoot(t)
	apiKey := strings.TrimSpace(os.Getenv("WANDB_API_KEY"))
	modelID := strings.TrimSpace(os.Getenv("WANDB_LIVE_MODEL"))
	if apiKey == "" || modelID == "" {
		t.Skip("WANDB_API_KEY and WANDB_LIVE_MODEL are required for live wandb integration test")
	}

	outDir := filepath.Join(t.TempDir(), "wandb-live-artifacts")
	genOut, err := runCLIWithEnv(repoRoot, []string{"WANDB_API_KEY=" + apiKey},
		"generate",
		"--model", modelID,
		"--source", "wandb",
		"--eval-file", filepath.Join("examples", "eval_sample.csv"),
		"--formats", "json",
		"--out-dir", outDir,
	)
	if err != nil {
		t.Fatalf("live wandb generate failed: %v\n%s", err, genOut)
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
