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
		{name: "unauthorized", err: server.ErrUnauthorized, wantStatus: 401, wantCode: "unauthorized"},
		{name: "forbidden", err: server.ErrForbidden, wantStatus: 403, wantCode: "unauthorized"},
		{name: "rate-limited", err: server.ErrRateLimited, wantStatus: 429, wantCode: "rate_limited"},
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

func TestGenerateAuthAndRateLimitMiddleware(t *testing.T) {
	t.Run("missing-and-invalid-api-key", func(t *testing.T) {
		api := &server.APIServer{
			RequireAuth:      true,
			APIKeys:          server.ParseAPIKeys("secret-a,secret-b"),
			RateLimitEnabled: false,
		}
		body := []byte(`{"source":"wandb","model":"bad-id","eval_file":"examples/eval_sample.csv"}`)

		reqMissing := httptest.NewRequest(http.MethodPost, "/generate", bytes.NewReader(body))
		recMissing := httptest.NewRecorder()
		api.Handler().ServeHTTP(recMissing, reqMissing)
		assertAPIErrorCode(t, recMissing, http.StatusUnauthorized, "unauthorized")

		reqInvalid := httptest.NewRequest(http.MethodPost, "/generate", bytes.NewReader(body))
		reqInvalid.Header.Set("X-API-Key", "nope")
		recInvalid := httptest.NewRecorder()
		api.Handler().ServeHTTP(recInvalid, reqInvalid)
		assertAPIErrorCode(t, recInvalid, http.StatusForbidden, "unauthorized")

		reqValid := httptest.NewRequest(http.MethodPost, "/generate", bytes.NewReader(body))
		reqValid.Header.Set("X-API-Key", "secret-a")
		recValid := httptest.NewRecorder()
		api.Handler().ServeHTTP(recValid, reqValid)
		assertAPIErrorCode(t, recValid, http.StatusBadRequest, "invalid_input")
	})

	t.Run("rate-limited", func(t *testing.T) {
		api := &server.APIServer{
			RateLimitEnabled: true,
			RateLimitRPM:     1,
			RateLimitBurst:   1,
		}
		body := []byte(`{"source":"wandb","model":"bad-id","eval_file":"examples/eval_sample.csv"}`)

		reqFirst := httptest.NewRequest(http.MethodPost, "/generate", bytes.NewReader(body))
		recFirst := httptest.NewRecorder()
		api.Handler().ServeHTTP(recFirst, reqFirst)
		if recFirst.Code == http.StatusTooManyRequests {
			t.Fatalf("first request should not be limited")
		}

		reqSecond := httptest.NewRequest(http.MethodPost, "/generate", bytes.NewReader(body))
		recSecond := httptest.NewRecorder()
		api.Handler().ServeHTTP(recSecond, reqSecond)
		assertAPIErrorCode(t, recSecond, http.StatusTooManyRequests, "rate_limited")
	})
}

func TestRequestIDIsSetAndPropagated(t *testing.T) {
	api := &server.APIServer{}

	reqNoHeader := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recNoHeader := httptest.NewRecorder()
	api.Handler().ServeHTTP(recNoHeader, reqNoHeader)
	if got := recNoHeader.Header().Get("X-Request-ID"); got == "" {
		t.Fatalf("expected generated X-Request-ID header")
	}

	reqWithHeader := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	reqWithHeader.Header.Set("X-Request-ID", "req-fixed-123")
	recWithHeader := httptest.NewRecorder()
	api.Handler().ServeHTTP(recWithHeader, reqWithHeader)
	if got := recWithHeader.Header().Get("X-Request-ID"); got != "req-fixed-123" {
		t.Fatalf("expected request id propagation, got %q", got)
	}
}

func TestReadyzReturnsServiceUnavailableWhenDependenciesMissing(t *testing.T) {
	api := &server.APIServer{}
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for missing dependencies, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func assertAPIErrorCode(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("unexpected status: got=%d want=%d body=%s", rec.Code, wantStatus, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if payload["code"] != wantCode {
		t.Fatalf("unexpected error code: got=%v want=%s payload=%+v", payload["code"], wantCode, payload)
	}
}
