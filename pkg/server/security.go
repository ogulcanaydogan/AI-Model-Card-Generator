package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultGenerateTimeout = 180 * time.Second
	defaultValidateTimeout = 60 * time.Second
	defaultCheckTimeout    = 60 * time.Second

	defaultRateLimitRPM   = 120
	defaultRateLimitBurst = 30
)

type contextKey string

const requestIDContextKey contextKey = "request_id"

// ParseAPIKeys parses comma-separated API keys into a lookup map.
func ParseAPIKeys(raw string) map[string]struct{} {
	parts := strings.Split(raw, ",")
	out := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		key := strings.TrimSpace(part)
		if key == "" {
			continue
		}
		out[key] = struct{}{}
	}
	return out
}

// RateLimiter is a simple token-bucket limiter keyed by route and client identity.
type RateLimiter struct {
	mu     sync.Mutex
	rpm    float64
	burst  float64
	bucket map[string]*tokenBucket
}

type tokenBucket struct {
	tokens float64
	last   time.Time
}

// NewRateLimiter creates a token bucket limiter.
func NewRateLimiter(rpm, burst int) *RateLimiter {
	if rpm <= 0 {
		rpm = defaultRateLimitRPM
	}
	if burst <= 0 {
		burst = defaultRateLimitBurst
	}
	return &RateLimiter{
		rpm:    float64(rpm),
		burst:  float64(burst),
		bucket: map[string]*tokenBucket{},
	}
}

// Allow reports whether the key can proceed at the current time.
func (r *RateLimiter) Allow(key string, now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	tb, ok := r.bucket[key]
	if !ok {
		r.bucket[key] = &tokenBucket{
			tokens: r.burst - 1,
			last:   now,
		}
		return true
	}

	elapsedSec := now.Sub(tb.last).Seconds()
	refillPerSec := r.rpm / 60.0
	tb.tokens += elapsedSec * refillPerSec
	if tb.tokens > r.burst {
		tb.tokens = r.burst
	}
	tb.last = now

	if tb.tokens < 1 {
		return false
	}
	tb.tokens -= 1
	return true
}

func (s *APIServer) ensureDefaults() {
	if s.LogWriter == nil {
		s.LogWriter = os.Stdout
	}
	if s.GenerateTimeout <= 0 {
		s.GenerateTimeout = defaultGenerateTimeout
	}
	if s.ValidateTimeout <= 0 {
		s.ValidateTimeout = defaultValidateTimeout
	}
	if s.CheckTimeout <= 0 {
		s.CheckTimeout = defaultCheckTimeout
	}
	if s.RateLimitRPM <= 0 {
		s.RateLimitRPM = defaultRateLimitRPM
	}
	if s.RateLimitBurst <= 0 {
		s.RateLimitBurst = defaultRateLimitBurst
	}
	if s.RateLimitEnabled && s.rateLimiter == nil {
		s.rateLimiter = NewRateLimiter(s.RateLimitRPM, s.RateLimitBurst)
	}
}

func (s *APIServer) withMiddlewares(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.ensureDefaults()

		started := time.Now().UTC()
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" {
			requestID = generateRequestID()
		}

		ctx := context.WithValue(r.Context(), requestIDContextKey, requestID)
		r = r.WithContext(ctx)

		rec := &statusRecorder{ResponseWriter: w}
		rec.Header().Set("X-Request-ID", requestID)
		clientIP := clientIPFromRequest(r)

		if isProtectedRoute(r) {
			if s.RequireAuth {
				apiKey := strings.TrimSpace(r.Header.Get("X-API-Key"))
				if apiKey == "" {
					writeAPIError(rec, ErrUnauthorized, nil)
					s.logRequest(started, requestID, r, rec, clientIP)
					return
				}
				if _, ok := s.APIKeys[apiKey]; !ok {
					writeAPIError(rec, ErrForbidden, nil)
					s.logRequest(started, requestID, r, rec, clientIP)
					return
				}
			}
			if s.RateLimitEnabled && s.rateLimiter != nil {
				limiterKey := fmt.Sprintf("%s|%s", clientIP, r.URL.Path)
				if !s.rateLimiter.Allow(limiterKey, started) {
					writeAPIError(rec, ErrRateLimited, nil)
					s.logRequest(started, requestID, r, rec, clientIP)
					return
				}
			}
		}

		next.ServeHTTP(rec, r)
		s.logRequest(started, requestID, r, rec, clientIP)
	})
}

func (s *APIServer) logRequest(started time.Time, requestID string, r *http.Request, rec *statusRecorder, clientIP string) {
	status := rec.StatusCode()
	if status == 0 {
		status = http.StatusOK
	}
	errorCode := strings.TrimSpace(rec.Header().Get("X-MCG-Error-Code"))
	logLine := map[string]any{
		"timestamp_utc": started.UTC().Format(time.RFC3339Nano),
		"request_id":    requestID,
		"method":        r.Method,
		"route":         r.URL.Path,
		"status":        status,
		"latency_ms":    time.Since(started).Milliseconds(),
		"client_ip":     clientIP,
		"error_code":    errorCode,
	}
	raw, err := json.Marshal(logLine)
	if err != nil {
		return
	}
	_, _ = io.WriteString(s.LogWriter, string(raw)+"\n")
}

func isProtectedRoute(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	switch r.URL.Path {
	case "/generate", "/validate", "/check":
		return true
	default:
		return false
	}
}

func clientIPFromRequest(r *http.Request) string {
	forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	if strings.TrimSpace(r.RemoteAddr) != "" {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return "unknown"
}

func generateRequestID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err == nil {
		return hex.EncodeToString(buf)
	}
	return strconv.FormatInt(time.Now().UTC().UnixNano(), 36)
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.statusCode = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(p []byte) (int, error) {
	if s.statusCode == 0 {
		s.statusCode = http.StatusOK
	}
	return s.ResponseWriter.Write(p)
}

func (s *statusRecorder) StatusCode() int {
	return s.statusCode
}

func requestIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(requestIDContextKey).(string)
	return strings.TrimSpace(v)
}
