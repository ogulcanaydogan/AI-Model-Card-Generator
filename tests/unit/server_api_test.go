package unit

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/yapay/ai-model-card-generator/pkg/core"
	"github.com/yapay/ai-model-card-generator/pkg/server"
)

func TestClassifyAPIError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "invalid-input", err: server.ErrInvalidInput, wantStatus: 400, wantCode: "invalid_input"},
		{name: "unsupported-source", err: core.ErrUnsupportedSource, wantStatus: 400, wantCode: "unsupported_source"},
		{name: "compliance-failed", err: server.ErrComplianceFailed, wantStatus: 422, wantCode: "compliance_failed"},
		{name: "internal", err: errors.New("boom"), wantStatus: 500, wantCode: "internal_error"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, code := server.ClassifyAPIError(tc.err)
			if status != tc.wantStatus || code != tc.wantCode {
				t.Fatalf("unexpected classification: status=%d code=%s", status, code)
			}
		})
	}
}

func TestGenerateValidationRejectsMalformedModel(t *testing.T) {
	api := &server.APIServer{
		SchemaPath:  "schemas/model-card.v1.json",
		ToolVersion: "v1.0.0",
		AuditLogger: core.NewAuditLogger(filepath.Join(t.TempDir(), "audit.jsonl")),
	}
	reqBody := map[string]any{
		"source":    "wandb",
		"model":     "bad-id",
		"eval_file": "examples/eval_sample.csv",
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/generate", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if payload["code"] != "invalid_input" {
		t.Fatalf("unexpected error code: %v", payload["code"])
	}
}
