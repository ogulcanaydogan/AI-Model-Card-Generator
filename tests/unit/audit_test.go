package unit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yapay/ai-model-card-generator/pkg/core"
)

func TestAuditLoggerAppendWritesJSONL(t *testing.T) {
	auditPath := filepath.Join(t.TempDir(), "audit", "runs.jsonl")
	logger := core.NewAuditLogger(auditPath)

	record, err := core.NewAuditRecord("cli", "generate", "v1.0.0", map[string]any{
		"source": "custom",
		"model":  "demo",
	})
	if err != nil {
		t.Fatalf("new audit record: %v", err)
	}
	record.Status = "succeeded"
	record.DurationMS = 42
	record.Source = "custom"
	record.ModelRef = "demo"
	record.ArtifactPaths = map[string]string{"json": "artifacts/model_card.json"}

	if err := logger.Append(record); err != nil {
		t.Fatalf("append audit record: %v", err)
	}

	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("read audit log: %v", err)
	}
	lines := splitNonEmptyLines(string(data))
	if len(lines) != 1 {
		t.Fatalf("expected 1 audit line, got %d", len(lines))
	}

	var parsed core.AuditRecord
	if err := json.Unmarshal([]byte(lines[0]), &parsed); err != nil {
		t.Fatalf("parse audit line: %v", err)
	}
	if parsed.RunID == "" || parsed.InputHashSHA == "" || parsed.TimestampUTC == "" {
		t.Fatalf("missing required audit fields: %+v", parsed)
	}
	if parsed.Status != "succeeded" || parsed.Operation != "generate" || parsed.Mode != "cli" {
		t.Fatalf("unexpected audit values: %+v", parsed)
	}
}

func TestAuditLoggerAppendFailsForInvalidPath(t *testing.T) {
	dir := t.TempDir()
	logger := core.NewAuditLogger(dir)

	record, err := core.NewAuditRecord("api", "check", "v1.0.0", map[string]any{"framework": "nist"})
	if err != nil {
		t.Fatalf("new audit record: %v", err)
	}
	record.Status = "failed"

	if err := logger.Append(record); err == nil {
		t.Fatalf("expected append to fail when audit path points to a directory")
	}
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
