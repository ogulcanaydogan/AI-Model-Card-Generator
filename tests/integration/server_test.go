package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yapay/ai-model-card-generator/pkg/analyzers"
	"github.com/yapay/ai-model-card-generator/pkg/compliance"
	"github.com/yapay/ai-model-card-generator/pkg/core"
	"github.com/yapay/ai-model-card-generator/pkg/extractors"
	"github.com/yapay/ai-model-card-generator/pkg/generators"
	apisrv "github.com/yapay/ai-model-card-generator/pkg/server"
)

func TestAPIServerGenerateValidateCheckAndAudit(t *testing.T) {
	repoRoot := mustRepoRoot(t)
	outDir := filepath.Join(t.TempDir(), "api-out")
	auditPath := filepath.Join(t.TempDir(), "audit", "runs.jsonl")

	api := &apisrv.APIServer{
		Pipeline: core.Pipeline{
			Extractors: map[string]core.Extractor{
				"custom": &extractors.CustomExtractor{},
			},
			Analyzers: []core.Analyzer{
				&analyzers.PerformanceAnalyzer{},
				&analyzers.FairnessAnalyzer{
					PythonBin:  "python3",
					ScriptPath: filepath.Join(repoRoot, "tests", "fixtures", "fairness_stub.py"),
				},
				&analyzers.BiasAnalyzer{},
				&analyzers.CarbonAnalyzer{
					FixturePath: filepath.Join(repoRoot, "tests", "fixtures", "carbon", "carbon_fixture.json"),
					PythonBin:   "python3",
					ScriptPath:  filepath.Join(repoRoot, "scripts", "carbon_metrics.py"),
				},
			},
			Generators: map[string]core.Generator{
				"md":   &generators.MarkdownGenerator{},
				"json": &generators.JSONGenerator{},
			},
			ComplianceCheckers: map[string]core.ComplianceChecker{
				"eu-ai-act": &compliance.EUAIActChecker{},
				"nist":      &compliance.NISTChecker{},
				"iso42001":  &compliance.ISO42001Checker{},
			},
			DefaultTemplatePath: filepath.Join(repoRoot, "templates"),
		},
		SchemaPath:  filepath.Join(repoRoot, "schemas", "model-card.v1.json"),
		AuditLogger: core.NewAuditLogger(auditPath),
		ToolVersion: "v1.0.0",
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	generatePayload := map[string]any{
		"source":                "custom",
		"model":                 "api-demo",
		"uri":                   filepath.Join(repoRoot, "tests", "fixtures", "custom_metadata.json"),
		"eval_file":             filepath.Join(repoRoot, "examples", "eval_sample.csv"),
		"template":              "standard",
		"formats":               []string{"json", "md"},
		"out_dir":               outDir,
		"language":              "en",
		"compliance_frameworks": []string{"eu-ai-act", "nist"},
	}
	generateResp := postJSON(t, srv.URL+"/generate", generatePayload, http.StatusOK)

	cardAny, ok := generateResp["card"].(map[string]any)
	if !ok {
		t.Fatalf("generate response missing card payload: %+v", generateResp)
	}
	artifactsAny, ok := cardAny["artifacts"].(map[string]any)
	if !ok {
		t.Fatalf("generate response missing artifacts: %+v", generateResp)
	}
	filesAny, ok := artifactsAny["generated_files"].(map[string]any)
	if !ok {
		t.Fatalf("generate response missing generated_files: %+v", generateResp)
	}
	jsonPath, ok := filesAny["json"].(string)
	if !ok || strings.TrimSpace(jsonPath) == "" {
		t.Fatalf("generate response missing json artifact path: %+v", generateResp)
	}

	validatePayload := map[string]any{
		"schema": filepath.Join(repoRoot, "schemas", "model-card.v1.json"),
		"input":  jsonPath,
	}
	postJSON(t, srv.URL+"/validate", validatePayload, http.StatusOK)

	checkPayload := map[string]any{
		"framework": "nist",
		"input":     jsonPath,
		"strict":    false,
	}
	checkResp := postJSON(t, srv.URL+"/check", checkPayload, http.StatusOK)
	if _, ok := checkResp["reports"]; !ok {
		t.Fatalf("check response missing reports: %+v", checkResp)
	}

	strictFailPayload := map[string]any{
		"framework": "nist",
		"input":     filepath.Join(repoRoot, "tests", "fixtures", "strict_fail_model_card.json"),
		"strict":    true,
	}
	strictFailResp := postJSON(t, srv.URL+"/check", strictFailPayload, http.StatusUnprocessableEntity)
	if strictFailResp["code"] != "compliance_failed" {
		t.Fatalf("unexpected strict-fail error code: %+v", strictFailResp)
	}

	auditRaw, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("read audit log: %v", err)
	}
	lines := splitNonEmptyLines(strings.TrimSpace(string(auditRaw)))
	if len(lines) < 4 {
		t.Fatalf("expected at least 4 audit entries, got %d", len(lines))
	}

	for _, line := range lines {
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("parse audit line: %v", err)
		}
		if rec["run_id"] == "" || rec["operation"] == "" || rec["mode"] == "" || rec["status"] == "" {
			t.Fatalf("audit record missing required fields: %+v", rec)
		}
	}
}

func postJSON(t *testing.T, url string, payload any, expectedStatus int) map[string]any {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post request: %v", err)
	}
	defer resp.Body.Close()

	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.StatusCode != expectedStatus {
		t.Fatalf("unexpected status %d, expected %d: %+v", resp.StatusCode, expectedStatus, out)
	}
	return out
}

func splitNonEmptyLines(input string) []string {
	raw := strings.Split(input, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
