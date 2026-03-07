package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/yapay/ai-model-card-generator/pkg/compliance"
	"github.com/yapay/ai-model-card-generator/pkg/core"
	"github.com/yapay/ai-model-card-generator/pkg/extractors"
)

// APIServer exposes generate/validate/check over HTTP.
type APIServer struct {
	Pipeline    core.Pipeline
	SchemaPath  string
	AuditLogger *core.AuditLogger
	ToolVersion string

	RequireAuth bool
	APIKeys     map[string]struct{}

	RateLimitEnabled bool
	RateLimitRPM     int
	RateLimitBurst   int

	GenerateTimeout time.Duration
	ValidateTimeout time.Duration
	CheckTimeout    time.Duration

	LogWriter io.Writer

	rateLimiter *RateLimiter
}

type apiErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

type GenerateRequest struct {
	Ref                  core.ModelRef `json:"ref"`
	Source               string        `json:"source"`
	Model                string        `json:"model"`
	URI                  string        `json:"uri"`
	EvalFile             string        `json:"eval_file"`
	Template             string        `json:"template"`
	TemplateFile         string        `json:"template_file"`
	Formats              []string      `json:"formats"`
	OutDir               string        `json:"out_dir"`
	Language             string        `json:"language"`
	Lang                 string        `json:"lang"`
	Compliance           string        `json:"compliance"`
	ComplianceFrameworks []string      `json:"compliance_frameworks"`
}

type ValidateRequest struct {
	Schema string `json:"schema"`
	Input  string `json:"input"`
}

type CheckRequest struct {
	Framework  string   `json:"framework"`
	Frameworks []string `json:"frameworks"`
	Input      string   `json:"input"`
	Strict     bool     `json:"strict"`
}

// Handler builds the HTTP handler for API mode.
func (s *APIServer) Handler() http.Handler {
	s.ensureDefaults()
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)
	mux.HandleFunc("/generate", s.handleGenerate)
	mux.HandleFunc("/validate", s.handleValidate)
	mux.HandleFunc("/check", s.handleCheck)
	return s.withMiddlewares(mux)
}

func (s *APIServer) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *APIServer) handleReadyz(w http.ResponseWriter, _ *http.Request) {
	checks := map[string]string{}
	ready := true

	if s.AuditLogger == nil {
		ready = false
		checks["audit_logger"] = "missing"
	} else {
		checks["audit_logger"] = "ok"
	}
	if strings.TrimSpace(s.SchemaPath) == "" {
		ready = false
		checks["schema"] = "missing"
	} else if _, err := os.Stat(s.SchemaPath); err != nil {
		ready = false
		checks["schema"] = err.Error()
	} else {
		checks["schema"] = "ok"
	}
	if len(s.Pipeline.Extractors) == 0 {
		ready = false
		checks["extractors"] = "missing"
	} else {
		checks["extractors"] = "ok"
	}
	if len(s.Pipeline.Generators) == 0 {
		ready = false
		checks["generators"] = "missing"
	} else {
		checks["generators"] = "ok"
	}
	if len(s.Pipeline.ComplianceCheckers) == 0 {
		ready = false
		checks["compliance_checkers"] = "missing"
	} else {
		checks["compliance_checkers"] = "ok"
	}

	status := "ready"
	httpStatus := http.StatusOK
	if !ready {
		status = "not_ready"
		httpStatus = http.StatusServiceUnavailable
	}
	writeJSON(w, httpStatus, map[string]any{
		"status": status,
		"checks": checks,
	})
}

func (s *APIServer) handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var req GenerateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, fmt.Errorf("%w: %v", ErrInvalidInput, err), nil)
		return
	}

	opts, err := s.toGenerateOptions(req)
	if err != nil {
		writeAPIError(w, err, nil)
		return
	}

	record, err := core.NewAuditRecord("api", "generate", s.ToolVersion, req)
	if err != nil {
		writeAPIError(w, err, nil)
		return
	}
	record.Source = opts.Ref.Source
	record.ModelRef = opts.Ref.ID
	record.Frameworks = opts.ComplianceFrameworks

	started := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), s.GenerateTimeout)
	defer cancel()

	card, opErr := s.Pipeline.Generate(ctx, opts)
	if opErr == nil {
		record.ArtifactPaths = map[string]string{}
		for k, v := range card.Artifacts.GeneratedFiles {
			record.ArtifactPaths[k] = v
		}
		record.ArtifactPaths["compliance_report"] = card.Artifacts.CompliancePath
	}

	if err := s.appendAudit(record, started, opErr); err != nil {
		writeAPIError(w, err, nil)
		return
	}
	if opErr != nil {
		writeAPIError(w, opErr, nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"card":      card,
		"artifacts": card.Artifacts,
	})
}

func (s *APIServer) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var req ValidateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, fmt.Errorf("%w: %v", ErrInvalidInput, err), nil)
		return
	}
	if strings.TrimSpace(req.Input) == "" {
		writeAPIError(w, fmt.Errorf("%w: input is required", ErrInvalidInput), nil)
		return
	}
	schema := strings.TrimSpace(req.Schema)
	if schema == "" {
		schema = s.SchemaPath
	}

	record, err := core.NewAuditRecord("api", "validate", s.ToolVersion, req)
	if err != nil {
		writeAPIError(w, err, nil)
		return
	}
	started := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), s.ValidateTimeout)
	defer cancel()

	ext := strings.ToLower(filepath.Ext(req.Input))
	var opErr error
	switch ext {
	case ".json":
		if err := ctx.Err(); err != nil {
			opErr = err
			break
		}
		opErr = core.ValidateJSONSchema(schema, req.Input)
	case ".md":
		if err := ctx.Err(); err != nil {
			opErr = err
			break
		}
		opErr = validateMarkdownCard(req.Input)
	default:
		opErr = fmt.Errorf("%w: unsupported input extension: %s", ErrInvalidInput, ext)
	}

	if err := s.appendAudit(record, started, opErr); err != nil {
		writeAPIError(w, err, nil)
		return
	}
	if opErr != nil {
		writeAPIError(w, opErr, nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"valid":  true,
		"schema": schema,
		"input":  req.Input,
	})
}

func (s *APIServer) handleCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var req CheckRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, fmt.Errorf("%w: %v", ErrInvalidInput, err), nil)
		return
	}
	if strings.TrimSpace(req.Input) == "" {
		writeAPIError(w, fmt.Errorf("%w: input is required", ErrInvalidInput), nil)
		return
	}

	frameworks := req.Frameworks
	if len(frameworks) == 0 {
		frameworks = splitCSV(req.Framework)
	}
	if len(frameworks) == 0 {
		frameworks = []string{"eu-ai-act"}
	}

	record, err := core.NewAuditRecord("api", "check", s.ToolVersion, req)
	if err != nil {
		writeAPIError(w, err, nil)
		return
	}
	record.Frameworks = frameworks
	started := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), s.CheckTimeout)
	defer cancel()

	card, opErr := core.LoadModelCard(req.Input)
	if opErr != nil {
		if err := s.appendAudit(record, started, opErr); err != nil {
			writeAPIError(w, err, nil)
			return
		}
		writeAPIError(w, opErr, nil)
		return
	}
	record.Source = card.Metadata.Owner
	record.ModelRef = card.Metadata.Name

	checkers := map[string]core.ComplianceChecker{
		"eu-ai-act": &compliance.EUAIActChecker{},
		"nist":      &compliance.NISTChecker{},
		"iso42001":  &compliance.ISO42001Checker{},
	}

	reports := make([]core.ComplianceReport, 0, len(frameworks))
	for _, fw := range frameworks {
		checker, ok := checkers[strings.ToLower(strings.TrimSpace(fw))]
		if !ok {
			opErr = fmt.Errorf("%w: unsupported framework: %s", ErrInvalidInput, fw)
			break
		}
		if err := ctx.Err(); err != nil {
			opErr = err
			break
		}
		report, err := checker.Check(ctx, card, core.CheckOptions{Strict: req.Strict})
		if err != nil {
			opErr = err
			break
		}
		reports = append(reports, report)
	}

	details := map[string]any{
		"reports": reports,
	}
	if opErr == nil && core.StrictComplianceExit(reports, req.Strict) {
		opErr = fmt.Errorf("%w: strict compliance check failed", ErrComplianceFailed)
		details["strict_failed"] = true
	}

	if err := s.appendAudit(record, started, opErr); err != nil {
		writeAPIError(w, err, nil)
		return
	}
	if opErr != nil {
		writeAPIError(w, opErr, details)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"reports":       reports,
		"strict_failed": false,
	})
}

func (s *APIServer) toGenerateOptions(req GenerateRequest) (core.GenerateOptions, error) {
	source := strings.ToLower(strings.TrimSpace(firstNonEmpty(req.Source, req.Ref.Source)))
	model := strings.TrimSpace(firstNonEmpty(req.Model, req.Ref.ID))
	uri := strings.TrimSpace(firstNonEmpty(req.URI, req.Ref.URI))
	evalFile := strings.TrimSpace(req.EvalFile)

	if source == "" {
		return core.GenerateOptions{}, fmt.Errorf("%w: source is required", ErrInvalidInput)
	}
	if model == "" {
		return core.GenerateOptions{}, fmt.Errorf("%w: model is required", ErrInvalidInput)
	}
	if evalFile == "" {
		return core.GenerateOptions{}, fmt.Errorf("%w: eval_file is required", ErrInvalidInput)
	}
	switch source {
	case "hf":
		// no-op
	case "wandb":
		if _, err := extractors.ParseWandBModelID(model); err != nil {
			return core.GenerateOptions{}, fmt.Errorf("%w: invalid wandb model id: %v", ErrInvalidInput, err)
		}
	case "mlflow":
		if _, err := extractors.ParseMLflowModelID(model); err != nil {
			return core.GenerateOptions{}, fmt.Errorf("%w: invalid mlflow model id: %v", ErrInvalidInput, err)
		}
	case "custom":
		if uri == "" {
			return core.GenerateOptions{}, fmt.Errorf("%w: custom source requires uri", ErrInvalidInput)
		}
	default:
		return core.GenerateOptions{}, fmt.Errorf("%w: %w: %s", ErrInvalidInput, core.ErrUnsupportedSource, source)
	}

	template := strings.TrimSpace(req.Template)
	if template == "" {
		template = "standard"
	}
	formats := req.Formats
	if len(formats) == 0 {
		formats = []string{"md", "json", "pdf"}
	}
	frameworks := req.ComplianceFrameworks
	if len(frameworks) == 0 {
		frameworks = splitCSV(req.Compliance)
	}
	if len(frameworks) == 0 {
		frameworks = []string{"eu-ai-act"}
	}
	language := strings.TrimSpace(firstNonEmpty(req.Language, req.Lang))
	if language == "" {
		language = "en"
	}
	outDir := strings.TrimSpace(req.OutDir)
	if outDir == "" {
		outDir = filepath.Join("artifacts", "api", strconv.FormatInt(time.Now().UTC().UnixNano(), 10))
	}

	return core.GenerateOptions{
		Ref: core.ModelRef{
			Source: source,
			ID:     model,
			URI:    uri,
		},
		EvalFile:             evalFile,
		Template:             template,
		TemplateFile:         strings.TrimSpace(req.TemplateFile),
		Formats:              formats,
		OutDir:               outDir,
		Language:             language,
		ComplianceFrameworks: frameworks,
	}, nil
}

func (s *APIServer) appendAudit(record core.AuditRecord, started time.Time, opErr error) error {
	if s.AuditLogger == nil {
		return fmt.Errorf("audit logger is not configured")
	}
	record.DurationMS = time.Since(started).Milliseconds()
	if opErr != nil {
		record.Status = "failed"
		record.Error = opErr.Error()
	} else {
		record.Status = "succeeded"
	}
	if err := s.AuditLogger.Append(record); err != nil {
		return fmt.Errorf("audit write failed: %w", err)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeAPIError(w http.ResponseWriter, err error, details any) {
	statusCode, code := ClassifyAPIError(err)
	if code != "" {
		w.Header().Set("X-MCG-Error-Code", code)
	}
	writeJSON(w, statusCode, apiErrorResponse{
		Code:    code,
		Message: err.Error(),
		Details: details,
	})
}

func writeMethodNotAllowed(w http.ResponseWriter) {
	writeAPIError(w, fmt.Errorf("%w: method not allowed", ErrInvalidInput), nil)
}

func decodeJSON(r *http.Request, out any) error {
	if r.Body == nil {
		return errors.New("request body is required")
	}
	defer r.Body.Close()

	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	return nil
}

func validateMarkdownCard(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := strings.ToLower(string(data))
	requiredSections := []string{"## metadata", "## performance", "## fairness", "## compliance"}
	for _, section := range requiredSections {
		if !strings.Contains(content, section) {
			return fmt.Errorf("%w: markdown model card missing required section: %s", ErrInvalidInput, section)
		}
	}
	return nil
}

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		n := strings.TrimSpace(strings.ToLower(p))
		if n != "" {
			out = append(out, n)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
