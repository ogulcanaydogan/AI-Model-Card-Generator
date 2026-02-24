package unit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yapay/ai-model-card-generator/pkg/core"
	"github.com/yapay/ai-model-card-generator/pkg/generators"
)

func TestMarkdownGenerator(t *testing.T) {
	t.Parallel()
	g := &generators.MarkdownGenerator{}
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "model_card.md")
	tmplPath := filepath.Join(tmpDir, "card.tmpl")
	if err := os.WriteFile(tmplPath, []byte("# {{ .Metadata.Name }}\n\n## Metadata\n"), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	card := core.ModelCard{Metadata: core.ModelMetadata{Name: "demo"}}
	if err := g.Generate(context.Background(), card, tmplPath, outPath); err != nil {
		t.Fatalf("generate markdown: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !strings.Contains(string(data), "# demo") {
		t.Fatalf("unexpected markdown output: %s", string(data))
	}
}
