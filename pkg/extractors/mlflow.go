package extractors

import (
	"context"
	"fmt"

	"github.com/yapay/ai-model-card-generator/pkg/core"
)

// MLflowExtractor is a Phase 2 placeholder.
type MLflowExtractor struct{}

func (e *MLflowExtractor) Name() string {
	return "mlflow"
}

func (e *MLflowExtractor) Extract(_ context.Context, ref core.ModelRef) (core.ModelMetadata, error) {
	return core.ModelMetadata{}, fmt.Errorf("mlflow extractor not implemented for Phase 1 (model: %s)", ref.ID)
}
