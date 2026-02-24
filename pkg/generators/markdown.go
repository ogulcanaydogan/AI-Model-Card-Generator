package generators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yapay/ai-model-card-generator/pkg/core"
)

// MarkdownGenerator writes model cards in Markdown format.
type MarkdownGenerator struct{}

func (g *MarkdownGenerator) Format() string {
	return "md"
}

func (g *MarkdownGenerator) Generate(_ context.Context, card core.ModelCard, templatePath, outPath string) error {
	content, err := renderMarkdown(card, templatePath)
	if err != nil {
		return fmt.Errorf("render markdown: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create markdown output dir: %w", err)
	}
	if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write markdown: %w", err)
	}
	return nil
}
