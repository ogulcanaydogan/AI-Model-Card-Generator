package generators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yapay/ai-model-card-generator/pkg/core"
)

// HTMLGenerator writes model cards in HTML format.
type HTMLGenerator struct{}

func (g *HTMLGenerator) Format() string {
	return "html"
}

func (g *HTMLGenerator) Generate(_ context.Context, card core.ModelCard, templatePath, outPath string) error {
	content, err := renderHTML(card, templatePath)
	if err != nil {
		return fmt.Errorf("render HTML: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create HTML output dir: %w", err)
	}
	if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write HTML: %w", err)
	}
	return nil
}
