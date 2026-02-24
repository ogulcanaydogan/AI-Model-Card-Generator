package extractors

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/yapay/ai-model-card-generator/pkg/core"
)

// CustomExtractor loads metadata from user-provided JSON files.
type CustomExtractor struct{}

func (e *CustomExtractor) Name() string {
	return "custom"
}

func (e *CustomExtractor) Extract(_ context.Context, ref core.ModelRef) (core.ModelMetadata, error) {
	if strings.TrimSpace(ref.URI) == "" {
		return core.ModelMetadata{}, fmt.Errorf("custom source requires --uri path to metadata JSON")
	}
	data, err := os.ReadFile(ref.URI)
	if err != nil {
		return core.ModelMetadata{}, fmt.Errorf("read custom metadata: %w", err)
	}
	var metadata core.ModelMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return core.ModelMetadata{}, fmt.Errorf("parse custom metadata: %w", err)
	}
	if metadata.Name == "" {
		metadata.Name = ref.ID
	}
	return metadata, nil
}
