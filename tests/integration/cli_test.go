package integration

import (
	"encoding/json"
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
		"--compliance", "eu-ai-act,nist",
		"--out-dir", outDir,
	)
	if err != nil {
		t.Fatalf("generate failed: %v\n%s", err, genOut)
	}

	jsonPath := filepath.Join(outDir, "model_card.json")
	if _, err := os.Stat(jsonPath); err != nil {
		t.Fatalf("expected generated json: %v", err)
	}
	mdPath := filepath.Join(outDir, "model_card.md")
	if _, err := os.Stat(mdPath); err != nil {
		t.Fatalf("expected generated markdown: %v", err)
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
		"--framework", "nist",
		"--input", jsonPath,
		"--strict", "false",
	)
	if err != nil {
		t.Fatalf("nist check failed: %v\n%s", err, checkOut)
	}
	if !strings.Contains(checkOut, "\"framework\": \"nist\"") {
		t.Fatalf("check output missing nist framework: %s", checkOut)
	}
	if !strings.Contains(checkOut, "[evidence:") {
		t.Fatalf("check output missing evidence markers: %s", checkOut)
	}
	if !strings.Contains(checkOut, "[required]") || !strings.Contains(checkOut, "[advisory]") {
		t.Fatalf("check output missing required/advisory control markers: %s", checkOut)
	}
	if !strings.Contains(checkOut, "[MAN-") {
		t.Fatalf("check output missing expanded control identifier markers: %s", checkOut)
	}

	payload, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read generated json: %v", err)
	}
	var card map[string]any
	if err := json.Unmarshal(payload, &card); err != nil {
		t.Fatalf("parse generated json: %v", err)
	}
	carbon, ok := card["carbon"].(map[string]any)
	if !ok {
		t.Fatalf("generated json missing carbon block: %s", string(payload))
	}
	if _, ok := carbon["estimated_kg_co2e"]; !ok {
		t.Fatalf("generated carbon block missing estimated_kg_co2e: %+v", carbon)
	}

	mdPayload, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("read generated markdown: %v", err)
	}
	if !strings.Contains(string(mdPayload), "## Carbon / Sustainability") {
		t.Fatalf("generated markdown missing carbon section:\n%s", string(mdPayload))
	}
}

func TestCLICheckNISTStrictFails(t *testing.T) {
	repoRoot := mustRepoRoot(t)
	_, err := runCLI(repoRoot,
		"check",
		"--framework", "nist",
		"--input", filepath.Join("tests", "fixtures", "strict_fail_model_card.json"),
		"--strict", "true",
	)
	if err == nil {
		t.Fatalf("expected strict nist check failure")
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

func TestCLIMLflowFixtureGenerateValidateCheck(t *testing.T) {
	repoRoot := mustRepoRoot(t)
	outDir := filepath.Join(t.TempDir(), "mlflow-artifacts")

	genOut, err := runCLIWithEnv(repoRoot, []string{
		"MCG_MLFLOW_FIXTURE=tests/fixtures/mlflow/run_get_fixture.json",
	},
		"generate",
		"--model", "run:abc123",
		"--source", "mlflow",
		"--eval-file", filepath.Join("examples", "eval_sample.csv"),
		"--formats", "md,json",
		"--out-dir", outDir,
	)
	if err != nil {
		t.Fatalf("mlflow fixture generate failed: %v\n%s", err, genOut)
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

func TestCLIMLflowInvalidModelFormatFails(t *testing.T) {
	repoRoot := mustRepoRoot(t)
	output, err := runCLI(repoRoot,
		"generate",
		"--model", "abc123",
		"--source", "mlflow",
		"--eval-file", filepath.Join("examples", "eval_sample.csv"),
		"--formats", "json",
		"--out-dir", filepath.Join(t.TempDir(), "invalid-mlflow"),
	)
	if err == nil {
		t.Fatalf("expected generate failure for malformed mlflow model id")
	}
	if !strings.Contains(output, "invalid --model for mlflow source") {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestCLIBatchContinueOnErrorWritesReport(t *testing.T) {
	repoRoot := mustRepoRoot(t)
	outDir := filepath.Join(t.TempDir(), "batch-out")

	output, err := runCLIWithEnv(repoRoot, []string{
		"MCG_WANDB_FIXTURE=tests/fixtures/wandb/run_fixture.json",
	},
		"generate",
		"--batch", filepath.Join("tests", "fixtures", "batch", "manifest_continue.yaml"),
		"--out-dir", outDir,
		"--workers", "2",
		"--fail-fast", "false",
	)
	if err == nil {
		t.Fatalf("expected batch command to fail because one job is invalid")
	}
	if !strings.Contains(output, "batch completed with 1 failed job") {
		t.Fatalf("unexpected batch output: %s", output)
	}

	reportPath := filepath.Join(outDir, "batch_report.json")
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read batch report: %v", err)
	}

	type batchJob struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	var report struct {
		Total     int        `json:"total"`
		Succeeded int        `json:"succeeded"`
		Failed    int        `json:"failed"`
		Jobs      []batchJob `json:"jobs"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("parse batch report: %v", err)
	}
	if report.Total != 3 || report.Succeeded != 2 || report.Failed != 1 {
		t.Fatalf("unexpected batch summary: %+v", report)
	}

	statusByID := map[string]string{}
	for _, job := range report.Jobs {
		statusByID[job.ID] = job.Status
	}
	if statusByID["custom-ok"] != "succeeded" {
		t.Fatalf("expected custom-ok to succeed, got %q", statusByID["custom-ok"])
	}
	if statusByID["malformed-mlflow"] != "failed" {
		t.Fatalf("expected malformed-mlflow to fail, got %q", statusByID["malformed-mlflow"])
	}
	if statusByID["wandb-ok"] != "succeeded" {
		t.Fatalf("expected wandb-ok to succeed, got %q", statusByID["wandb-ok"])
	}

	if _, err := os.Stat(filepath.Join(outDir, "custom-ok", "model_card.json")); err != nil {
		t.Fatalf("expected custom-ok artifact: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "wandb-ok", "model_card.json")); err != nil {
		t.Fatalf("expected wandb-ok artifact: %v", err)
	}

	customMDPath := filepath.Join(outDir, "custom-ok", "model_card.md")
	customMD, err := os.ReadFile(customMDPath)
	if err != nil {
		t.Fatalf("read custom-ok markdown: %v", err)
	}
	if !strings.Contains(string(customMD), "# Batch Custom:") {
		t.Fatalf("expected batch custom template marker in markdown: %s", string(customMD))
	}
}

func TestCLIBatchFailFastStopsEarly(t *testing.T) {
	repoRoot := mustRepoRoot(t)
	outDir := filepath.Join(t.TempDir(), "batch-fail-fast")

	output, err := runCLI(repoRoot,
		"generate",
		"--batch", filepath.Join("tests", "fixtures", "batch", "manifest_fail_fast.yaml"),
		"--out-dir", outDir,
		"--workers", "1",
		"--fail-fast", "true",
	)
	if err == nil {
		t.Fatalf("expected fail-fast batch command to fail")
	}
	if !strings.Contains(output, "batch completed with 1 failed job") {
		t.Fatalf("unexpected fail-fast output: %s", output)
	}

	reportPath := filepath.Join(outDir, "batch_report.json")
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read batch report: %v", err)
	}

	type batchJob struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	var report struct {
		Failed int        `json:"failed"`
		Jobs   []batchJob `json:"jobs"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("parse batch report: %v", err)
	}
	if report.Failed != 1 {
		t.Fatalf("expected one failed job, got %+v", report)
	}

	statusByID := map[string]string{}
	for _, job := range report.Jobs {
		statusByID[job.ID] = job.Status
	}
	if statusByID["first-invalid"] != "failed" {
		t.Fatalf("expected first-invalid to fail, got %q", statusByID["first-invalid"])
	}
	if statusByID["should-skip"] != "skipped" {
		t.Fatalf("expected should-skip to be skipped, got %q", statusByID["should-skip"])
	}

	if _, err := os.Stat(filepath.Join(outDir, "should-skip", "model_card.json")); err == nil {
		t.Fatalf("did not expect artifact for skipped job")
	}
}

func TestCLITemplateInitValidatePreview(t *testing.T) {
	repoRoot := mustRepoRoot(t)
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "custom_from_cli.tmpl")
	previewPath := filepath.Join(tmpDir, "preview.md")

	initOut, err := runCLI(repoRoot,
		"template", "init",
		"--name", "CLI Template",
		"--out", templatePath,
		"--base", "standard",
	)
	if err != nil {
		t.Fatalf("template init failed: %v\n%s", err, initOut)
	}

	valOut, err := runCLI(repoRoot,
		"template", "validate",
		"--input", templatePath,
	)
	if err != nil {
		t.Fatalf("template validate failed: %v\n%s", err, valOut)
	}

	prevOut, err := runCLI(repoRoot,
		"template", "preview",
		"--input", templatePath,
		"--card", filepath.Join("tests", "fixtures", "strict_fail_model_card.json"),
		"--out", previewPath,
	)
	if err != nil {
		t.Fatalf("template preview failed: %v\n%s", err, prevOut)
	}

	preview, err := os.ReadFile(previewPath)
	if err != nil {
		t.Fatalf("read preview output: %v", err)
	}
	if !strings.Contains(string(preview), "## Metadata") {
		t.Fatalf("unexpected preview output: %s", string(preview))
	}
}

func TestCLIGenerateTemplateFileOverridesTemplate(t *testing.T) {
	repoRoot := mustRepoRoot(t)
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "override.tmpl")
	outDir := filepath.Join(tmpDir, "artifacts")

	templateContent := `# OVERRIDE {{ .Metadata.Name }}

## Metadata
- Owner: {{ .Metadata.Owner }}

## Performance
- Accuracy: {{ printf "%.4f" .Performance.Accuracy }}

## Fairness
- Demographic Parity Difference: {{ printf "%.4f" .Fairness.DemographicParityDiff }}

## Compliance
{{ range .Compliance }}- {{ .Framework }}: {{ .Status }}
{{ end }}`
	if err := os.WriteFile(templatePath, []byte(templateContent), 0o644); err != nil {
		t.Fatalf("write template file: %v", err)
	}

	output, err := runCLI(repoRoot,
		"generate",
		"--model", "demo-model",
		"--source", "custom",
		"--uri", filepath.Join("tests", "fixtures", "custom_metadata.json"),
		"--eval-file", filepath.Join("examples", "eval_sample.csv"),
		"--template", "minimal",
		"--template-file", templatePath,
		"--formats", "md,json",
		"--out-dir", outDir,
	)
	if err != nil {
		t.Fatalf("generate with template-file failed: %v\n%s", err, output)
	}

	mdPath := filepath.Join(outDir, "model_card.md")
	md, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("read markdown output: %v", err)
	}
	if !strings.Contains(string(md), "# OVERRIDE") {
		t.Fatalf("expected custom template output, got: %s", string(md))
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
		"MCG_CARBON_FIXTURE=tests/fixtures/carbon/carbon_fixture.json",
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
