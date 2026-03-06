package core

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	defaultAuditPath = "artifacts/audit/runs.jsonl"
	defaultOperator  = "unknown"
)

// AuditRecord stores immutable run metadata for CLI/API operations.
type AuditRecord struct {
	RunID         string            `json:"run_id"`
	TimestampUTC  string            `json:"timestamp_utc"`
	Mode          string            `json:"mode"`
	Operation     string            `json:"operation"`
	InputHashSHA  string            `json:"input_hash_sha256"`
	ToolVersion   string            `json:"tool_version"`
	Operator      string            `json:"operator"`
	Status        string            `json:"status"`
	DurationMS    int64             `json:"duration_ms"`
	Frameworks    []string          `json:"frameworks,omitempty"`
	Source        string            `json:"source,omitempty"`
	ModelRef      string            `json:"model_ref,omitempty"`
	ArtifactPaths map[string]string `json:"artifact_paths,omitempty"`
	Error         string            `json:"error,omitempty"`
}

// AuditLogger appends JSONL records to an audit trail.
type AuditLogger struct {
	Path string
	mu   sync.Mutex
}

// NewAuditLogger creates an append-only JSONL writer.
func NewAuditLogger(path string) *AuditLogger {
	if strings.TrimSpace(path) == "" {
		path = defaultAuditPath
	}
	return &AuditLogger{Path: path}
}

// NewAuditLoggerFromEnv builds a logger from MCG_AUDIT_PATH.
func NewAuditLoggerFromEnv() *AuditLogger {
	path := strings.TrimSpace(os.Getenv("MCG_AUDIT_PATH"))
	return NewAuditLogger(path)
}

// NewAuditRecord creates a fresh record skeleton with deterministic metadata.
func NewAuditRecord(mode, operation, toolVersion string, input any) (AuditRecord, error) {
	hash, err := InputHashSHA256(input)
	if err != nil {
		return AuditRecord{}, err
	}
	return AuditRecord{
		RunID:        newRunID(),
		TimestampUTC: time.Now().UTC().Format(time.RFC3339Nano),
		Mode:         strings.TrimSpace(mode),
		Operation:    strings.TrimSpace(operation),
		InputHashSHA: hash,
		ToolVersion:  strings.TrimSpace(toolVersion),
		Operator:     operatorFromEnv(),
		Status:       "failed",
	}, nil
}

// InputHashSHA256 hashes any JSON-marshallable input.
func InputHashSHA256(input any) (string, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return "", Wrap("marshal audit input", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// Append writes one JSONL record. The write is serialized and append-only.
func (l *AuditLogger) Append(record AuditRecord) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if strings.TrimSpace(l.Path) == "" {
		return fmt.Errorf("audit path is empty")
	}
	if record.RunID == "" {
		return fmt.Errorf("audit record missing run_id")
	}
	if record.Operation == "" {
		return fmt.Errorf("audit record missing operation")
	}
	if record.Mode == "" {
		return fmt.Errorf("audit record missing mode")
	}
	if record.TimestampUTC == "" {
		return fmt.Errorf("audit record missing timestamp_utc")
	}

	if err := os.MkdirAll(filepath.Dir(l.Path), 0o755); err != nil {
		return Wrap("create audit directory", err)
	}

	data, err := json.Marshal(record)
	if err != nil {
		return Wrap("marshal audit record", err)
	}

	f, err := os.OpenFile(l.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return Wrap("open audit file", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return Wrap("append audit record", err)
	}
	return nil
}

func operatorFromEnv() string {
	operator := strings.TrimSpace(os.Getenv("MCG_OPERATOR"))
	if operator == "" {
		return defaultOperator
	}
	return operator
}

func newRunID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fallback to timestamp-based suffix when secure random fails.
		return fmt.Sprintf("run-%d", time.Now().UTC().UnixNano())
	}
	return "run-" + hex.EncodeToString(b[:])
}
