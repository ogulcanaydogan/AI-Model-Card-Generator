package generators

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yapay/ai-model-card-generator/pkg/core"
)

// JSONGenerator writes model cards in JSON format.
type JSONGenerator struct{}

func (g *JSONGenerator) Format() string {
	return "json"
}

func (g *JSONGenerator) Generate(_ context.Context, card core.ModelCard, _ string, outPath string) error {
	payload, err := json.MarshalIndent(card, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create JSON output dir: %w", err)
	}
	if err := os.WriteFile(outPath, payload, 0o644); err != nil {
		return fmt.Errorf("write JSON: %w", err)
	}
	return nil
}
