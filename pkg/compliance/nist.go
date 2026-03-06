package compliance

import (
	"context"
	"fmt"
	"strings"

	"github.com/yapay/ai-model-card-generator/pkg/core"
)

// NISTChecker evaluates model card coverage against NIST AI RMF control families.
type NISTChecker struct{}

const (
	nistRequiredPenalty            = 15.0
	nistAdvisoryPenalty            = 5.0
	nistDemographicParityThreshold = 0.20
	nistEqualizedOddsThreshold     = 0.20
)

type nistFunctionAssessment struct {
	function           string
	requiredGaps       []string
	advisoryFindings   []string
	recommendedActions []string
}

func (c *NISTChecker) Framework() string {
	return "nist"
}

func (c *NISTChecker) Check(_ context.Context, card core.ModelCard, _ core.CheckOptions) (core.ComplianceReport, error) {
	assessments := []nistFunctionAssessment{
		c.checkGovern(card),
		c.checkMap(card),
		c.checkMeasure(card),
		c.checkManage(card),
	}

	requiredGaps := []string{}
	findings := []string{}
	recommendations := []string{}
	for _, assessment := range assessments {
		requiredGaps = append(requiredGaps, assessment.requiredGaps...)
		findings = append(findings, assessment.advisoryFindings...)
		recommendations = append(recommendations, assessment.recommendedActions...)
	}
	recommendations = dedupeStrings(recommendations)

	score := 100.0 - float64(len(requiredGaps))*nistRequiredPenalty - float64(len(findings))*nistAdvisoryPenalty
	if score < 0 {
		score = 0
	}

	status := "pass"
	if len(requiredGaps) > 0 {
		status = "fail"
	} else if len(findings) > 0 {
		status = "warn"
	}

	return core.ComplianceReport{
		Framework:          c.Framework(),
		Score:              score,
		Status:             status,
		Findings:           findings,
		RequiredGaps:       requiredGaps,
		RecommendedActions: recommendations,
	}, nil
}

func (c *NISTChecker) checkGovern(card core.ModelCard) nistFunctionAssessment {
	result := nistFunctionAssessment{function: "GOVERN"}

	if strings.TrimSpace(card.Metadata.Owner) == "" {
		result.requiredGaps = append(result.requiredGaps, "GOVERN: [GOV-1][required] Accountable owner/provider is missing")
		result.recommendedActions = append(result.recommendedActions, "GOVERN: [GOV-1] Set metadata.owner to document accountable ownership")
	}

	if strings.TrimSpace(card.Governance.Maintainer) == "" {
		result.requiredGaps = append(result.requiredGaps, "GOVERN: [GOV-2][required] Maintainer contact is missing")
		result.recommendedActions = append(result.recommendedActions, "GOVERN: [GOV-2] Populate governance.maintainer for operational accountability")
	}

	if card.Governance.GeneratedAt.IsZero() {
		result.advisoryFindings = append(result.advisoryFindings, "GOVERN: [GOV-3][advisory] Card generation timestamp is missing")
		result.recommendedActions = append(result.recommendedActions, "GOVERN: [GOV-3] Include governance.generated_at for traceability")
	}

	return result
}

func (c *NISTChecker) checkMap(card core.ModelCard) nistFunctionAssessment {
	result := nistFunctionAssessment{function: "MAP"}

	if strings.TrimSpace(card.Metadata.IntendedUse) == "" {
		result.requiredGaps = append(result.requiredGaps, "MAP: [MAP-1][required] Intended use is missing")
		result.recommendedActions = append(result.recommendedActions, "MAP: [MAP-1] Document metadata.intended_use with expected operating context")
	}

	if strings.TrimSpace(card.Metadata.Limitations) == "" {
		result.requiredGaps = append(result.requiredGaps, "MAP: [MAP-2][required] Limitations are missing")
		result.recommendedActions = append(result.recommendedActions, "MAP: [MAP-2] Document metadata.limitations and known failure boundaries")
	}

	if strings.TrimSpace(card.Metadata.TrainingData) == "" && strings.TrimSpace(card.Metadata.EvalData) == "" {
		result.requiredGaps = append(result.requiredGaps, "MAP: [MAP-3][required] Training/evaluation data context is missing")
		result.recommendedActions = append(result.recommendedActions, "MAP: [MAP-3] Add metadata.training_data or metadata.eval_data context")
	}

	if len(card.Metadata.Tags) == 0 {
		result.advisoryFindings = append(result.advisoryFindings, "MAP: [MAP-4][advisory] No model tags provided for context classification")
		result.recommendedActions = append(result.recommendedActions, "MAP: [MAP-4] Add metadata.tags to improve downstream governance and filtering")
	}

	return result
}

func (c *NISTChecker) checkMeasure(card core.ModelCard) nistFunctionAssessment {
	result := nistFunctionAssessment{function: "MEASURE"}

	if !hasPerformanceEvidence(card.Performance) {
		result.requiredGaps = append(result.requiredGaps, "MEASURE: [MEA-1][required] Performance evidence is missing")
		result.recommendedActions = append(result.recommendedActions, "MEASURE: [MEA-1] Provide non-zero performance metrics with evaluation context")
	}

	if len(card.Fairness.GroupStats) == 0 {
		result.requiredGaps = append(result.requiredGaps, "MEASURE: [MEA-2][required] Subgroup fairness evidence is missing")
		result.recommendedActions = append(result.recommendedActions, "MEASURE: [MEA-2] Provide fairness.group_stats for subgroup analysis")
	}

	if len(card.Fairness.GroupStats) == 1 {
		result.advisoryFindings = append(result.advisoryFindings, "MEASURE: [MEA-3][advisory] Only one subgroup is present; comparative fairness evidence is weak")
		result.recommendedActions = append(result.recommendedActions, "MEASURE: [MEA-3] Evaluate at least two relevant groups for comparative fairness")
	}

	if card.Fairness.DemographicParityDiff > nistDemographicParityThreshold {
		result.advisoryFindings = append(result.advisoryFindings, fmt.Sprintf("MEASURE: [MEA-4][advisory] Demographic parity difference %.3f exceeds threshold %.2f", card.Fairness.DemographicParityDiff, nistDemographicParityThreshold))
		result.recommendedActions = append(result.recommendedActions, "MEASURE: [MEA-4] Investigate subgroup thresholding/reweighting to reduce parity disparity")
	}

	if card.Fairness.EqualizedOddsDiff > nistEqualizedOddsThreshold {
		result.advisoryFindings = append(result.advisoryFindings, fmt.Sprintf("MEASURE: [MEA-5][advisory] Equalized odds difference %.3f exceeds threshold %.2f", card.Fairness.EqualizedOddsDiff, nistEqualizedOddsThreshold))
		result.recommendedActions = append(result.recommendedActions, "MEASURE: [MEA-5] Improve error-rate parity via post-processing or retraining")
	}

	return result
}

func (c *NISTChecker) checkManage(card core.ModelCard) nistFunctionAssessment {
	result := nistFunctionAssessment{function: "MANAGE"}

	if len(card.RiskAssessment.KnownRisks) == 0 {
		result.requiredGaps = append(result.requiredGaps, "MANAGE: [MAN-1][required] Known risk register is missing")
		result.recommendedActions = append(result.recommendedActions, "MANAGE: [MAN-1] Populate risk_assessment.known_risks")
	}

	if len(card.RiskAssessment.Mitigations) == 0 {
		result.requiredGaps = append(result.requiredGaps, "MANAGE: [MAN-2][required] Mitigation actions are missing")
		result.recommendedActions = append(result.recommendedActions, "MANAGE: [MAN-2] Populate risk_assessment.mitigations")
	}

	if len(card.RiskAssessment.KnownRisks) > 0 && len(card.RiskAssessment.Mitigations) > 0 &&
		len(card.RiskAssessment.Mitigations) < len(card.RiskAssessment.KnownRisks) {
		result.requiredGaps = append(result.requiredGaps, "MANAGE: [MAN-3][required] Risk-to-mitigation coverage is incomplete")
		result.recommendedActions = append(result.recommendedActions, "MANAGE: [MAN-3] Provide at least one mitigation plan per known risk")
	}

	if card.Carbon == nil || strings.EqualFold(card.Carbon.Method, "unavailable") || strings.TrimSpace(card.Carbon.Method) == "" {
		result.advisoryFindings = append(result.advisoryFindings, "MANAGE: [MAN-4][advisory] Sustainability evidence is unavailable (carbon estimate)")
		result.recommendedActions = append(result.recommendedActions, "MANAGE: [MAN-4] Provide carbon evidence via fixture or carbon bridge inputs")
	}

	return result
}

func hasPerformanceEvidence(perf core.PerformanceMetrics) bool {
	return perf.Accuracy > 0 || perf.Precision > 0 || perf.Recall > 0 || perf.F1 > 0 || perf.AUC > 0
}

func dedupeStrings(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		normalized := strings.TrimSpace(item)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}
