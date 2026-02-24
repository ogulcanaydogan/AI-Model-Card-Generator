package compliance

import (
	"context"

	"github.com/yapay/ai-model-card-generator/pkg/core"
)

// NISTChecker is a Phase 2 scaffold for AI RMF mapping.
type NISTChecker struct{}

func (c *NISTChecker) Framework() string {
	return "nist"
}

func (c *NISTChecker) Check(_ context.Context, card core.ModelCard, _ core.CheckOptions) (core.ComplianceReport, error) {
	score := 70.0
	if len(card.RiskAssessment.KnownRisks) > 0 {
		score -= 5
	}
	return core.ComplianceReport{
		Framework: c.Framework(),
		Score:     score,
		Status:    "warn",
		Findings: []string{
			"NIST AI RMF mapping is in scaffold mode for Phase 1",
		},
		RecommendedActions: []string{
			"Enable full GOVERN/MAP/MEASURE/MANAGE mapping in Phase 2",
		},
	}, nil
}
