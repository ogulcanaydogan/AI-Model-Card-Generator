package extractors

import (
	"context"
	"fmt"

	"github.com/yapay/ai-model-card-generator/pkg/core"
)

// WeightsAndBiasesExtractor is a Phase 2 placeholder.
type WeightsAndBiasesExtractor struct{}

func (e *WeightsAndBiasesExtractor) Name() string {
	return "wandb"
}

func (e *WeightsAndBiasesExtractor) Extract(_ context.Context, ref core.ModelRef) (core.ModelMetadata, error) {
	return core.ModelMetadata{}, fmt.Errorf("wandb extractor not implemented for Phase 1 (model: %s)", ref.ID)
}
