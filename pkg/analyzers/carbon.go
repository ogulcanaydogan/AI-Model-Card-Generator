package analyzers

import (
	"context"

	"github.com/yapay/ai-model-card-generator/pkg/core"
)

// CarbonAnalyzer is a Phase 2 scaffold.
type CarbonAnalyzer struct{}

func (a *CarbonAnalyzer) Name() string {
	return "carbon"
}

func (a *CarbonAnalyzer) Analyze(_ context.Context, _ core.AnalysisInput) (core.AnalysisResult, error) {
	return core.AnalysisResult{
		Carbon: &core.CarbonEstimate{
			EstimatedKgCO2e: 0,
			Method:          "not-computed-phase-1",
		},
	}, nil
}
