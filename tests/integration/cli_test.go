package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCLIGenerateValidateCheck(t *testing.T) {
	repoRoot := mustRepoRoot(t)
	outDir := filepath.Join(t.TempDir(), "artifacts")

	genOut, err := runCLI(repoRoot,
		"generate",
		"--model", "demo-model",
		"--source", "custom",
		"--uri", filepath.Join("tests", "fixtures", "custom_metadata.json"),
		"--eval-file", filepath.Join("examples", "eval_sample.csv"),
		"--formats", "md,json",
		"--out-dir", outDir,
	)
	if err != nil {
		t.Fatalf("generate failed: %v\n%s", err, genOut)
	}

	jsonPath := filepath.Join(outDir, "model_card.json")
	if _, err := os.Stat(jsonPath); err != nil {
		t.Fatalf("expected generated json: %v", err)
	}

	valOut, err := runCLI(repoRoot,
		"validate",
		"--schema", filepath.Join("schemas", "model-card.v1.json"),
		"--input", jsonPath,
	)
	if err != nil {
		t.Fatalf("validate failed: %v\n%s", err, valOut)
	}

	checkOut, err := runCLI(repoRoot,
		"check",
		"--framework", "eu-ai-act",
		"--input", jsonPath,
		"--strict", "false",
	)
	if err != nil {
		t.Fatalf("check failed: %v\n%s", err, checkOut)
	}
	if !strings.Contains(checkOut, "eu-ai-act") {
		t.Fatalf("check output missing framework: %s", checkOut)
	}
}

func TestCLIStrictCheckFails(t *testing.T) {
	repoRoot := mustRepoRoot(t)
	_, err := runCLI(repoRoot,
		"check",
		"--framework", "eu-ai-act",
		"--input", filepath.Join("tests", "fixtures", "strict_fail_model_card.json"),
		"--strict", "true",
	)
	if err == nil {
		t.Fatalf("expected strict check failure")
	}
}

func TestCLIWandBFixtureGenerateValidateCheck(t *testing.T) {
	repoRoot := mustRepoRoot(t)
	outDir := filepath.Join(t.TempDir(), "wandb-artifacts")

	genOut, err := runCLIWithEnv(repoRoot, []string{
		"MCG_WANDB_FIXTURE=tests/fixtures/wandb/run_fixture.json",
	},
		"generate",
		"--model", "acme/support/abc123",
		"--source", "wandb",
		"--eval-file", filepath.Join("examples", "eval_sample.csv"),
		"--formats", "md,json",
		"--out-dir", outDir,
	)
	if err != nil {
		t.Fatalf("wandb fixture generate failed: %v\n%s", err, genOut)
	}

	jsonPath := filepath.Join(outDir, "model_card.json")
	if _, err := os.Stat(jsonPath); err != nil {
		t.Fatalf("expected generated json: %v", err)
	}

	valOut, err := runCLI(repoRoot,
		"validate",
		"--schema", filepath.Join("schemas", "model-card.v1.json"),
		"--input", jsonPath,
	)
	if err != nil {
		t.Fatalf("validate failed: %v\n%s", err, valOut)
	}

	checkOut, err := runCLI(repoRoot,
		"check",
		"--framework", "eu-ai-act",
		"--input", jsonPath,
		"--strict", "false",
	)
	if err != nil {
		t.Fatalf("check failed: %v\n%s", err, checkOut)
	}
	if !strings.Contains(checkOut, "eu-ai-act") {
		t.Fatalf("check output missing framework: %s", checkOut)
	}
}

func runCLI(repoRoot string, args ...string) (string, error) {
	return runCLIWithEnv(repoRoot, nil, args...)
}

func runCLIWithEnv(repoRoot string, extraEnv []string, args ...string) (string, error) {
	cmdArgs := append([]string{"run", "./cmd/mcg-cli"}, args...)
	cmd := exec.Command("go", cmdArgs...)
	cmd.Dir = repoRoot
	env := append(os.Environ(),
		"MCG_FAIRNESS_SCRIPT=tests/fixtures/fairness_stub.py",
		"MCG_PYTHON_BIN=python3",
	)
	env = append(env, extraEnv...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func mustRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
