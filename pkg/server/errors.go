package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/yapay/ai-model-card-generator/pkg/core"
)

var (
	// ErrInvalidInput identifies malformed requests or unsupported request values.
	ErrInvalidInput = errors.New("invalid input")
	// ErrComplianceFailed identifies strict compliance failures.
	ErrComplianceFailed = errors.New("compliance failed")
)

// ClassifyAPIError maps internal errors to stable API error codes and status codes.
func ClassifyAPIError(err error) (statusCode int, code string) {
	switch {
	case err == nil:
		return http.StatusOK, ""
	case errors.Is(err, ErrInvalidInput):
		return http.StatusBadRequest, "invalid_input"
	case errors.Is(err, core.ErrUnsupportedSource):
		return http.StatusBadRequest, "unsupported_source"
	case errors.Is(err, core.ErrUnsupportedFormat):
		return http.StatusBadRequest, "invalid_input"
	case errors.Is(err, core.ErrMissingEvalFile):
		return http.StatusBadRequest, "invalid_input"
	case errors.Is(err, core.ErrComplianceFramework):
		return http.StatusBadRequest, "invalid_input"
	case errors.Is(err, core.ErrSchemaValidationFail):
		return http.StatusBadRequest, "invalid_input"
	case errors.Is(err, ErrComplianceFailed):
		return http.StatusUnprocessableEntity, "compliance_failed"
	default:
		low := strings.ToLower(err.Error())
		if strings.Contains(low, "strict compliance check failed") {
			return http.StatusUnprocessableEntity, "compliance_failed"
		}
		if strings.Contains(low, "unsupported framework") {
			return http.StatusBadRequest, "invalid_input"
		}
		if strings.Contains(low, "invalid --") || strings.Contains(low, "expected format") {
			return http.StatusBadRequest, "invalid_input"
		}
		return http.StatusInternalServerError, "internal_error"
	}
}
