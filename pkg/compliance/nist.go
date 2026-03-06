package compliance

import (
	"context"
	"fmt"
	"strings"

	"github.com/yapay/ai-model-card-generator/pkg/core"
)

// NISTChecker evaluates model card coverage against NIST AI RMF control families.
type NISTChecker struct{}

func (c *NISTChecker) Framework() string {
	return "nist"
}

func (c *NISTChecker) Check(_ context.Context, card core.ModelCard, _ core.CheckOptions) (core.ComplianceReport, error) {
	requiredGaps := []string{}
	findings := []string{}
	recommendations := []string{}

	governGaps, governFindings, governRecs := c.checkGovern(card)
	mapGaps, mapFindings, mapRecs := c.checkMap(card)
	measureGaps, measureFindings, measureRecs := c.checkMeasure(card)
	manageGaps, manageFindings, manageRecs := c.checkManage(card)

	requiredGaps = append(requiredGaps, governGaps...)
	requiredGaps = append(requiredGaps, mapGaps...)
	requiredGaps = append(requiredGaps, measureGaps...)
	requiredGaps = append(requiredGaps, manageGaps...)

	findings = append(findings, governFindings...)
	findings = append(findings, mapFindings...)
	findings = append(findings, measureFindings...)
	findings = append(findings, manageFindings...)

	recommendations = append(recommendations, governRecs...)
	recommendations = append(recommendations, mapRecs...)
	recommendations = append(recommendations, measureRecs...)
	recommendations = append(recommendations, manageRecs...)

	score := 100.0 - float64(len(requiredGaps))*12.0 - float64(len(findings))*4.0
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

func (c *NISTChecker) checkGovern(card core.ModelCard) ([]string, []string, []string) {
	required := []string{}
	findings := []string{}
	recs := []string{}

	if strings.TrimSpace(card.Metadata.Owner) == "" {
		required = append(required, "GOVERN: Accountable owner/provider is missing")
		recs = append(recs, "GOVERN: Set metadata.owner to document accountable ownership")
	}

	if strings.TrimSpace(card.Governance.Maintainer) == "" {
		findings = append(findings, "GOVERN: Maintainer contact is missing")
		recs = append(recs, "GOVERN: Populate governance.maintainer for operational accountability")
	}

	if card.Governance.GeneratedAt.IsZero() {
		findings = append(findings, "GOVERN: Card generation timestamp is missing")
		recs = append(recs, "GOVERN: Include governance.generated_at for traceability")
	}

	return required, findings, recs
}

func (c *NISTChecker) checkMap(card core.ModelCard) ([]string, []string, []string) {
	required := []string{}
	findings := []string{}
	recs := []string{}

	if strings.TrimSpace(card.Metadata.IntendedUse) == "" {
		required = append(required, "MAP: Intended use is missing")
		recs = append(recs, "MAP: Document metadata.intended_use with expected operating context")
	}

	if strings.TrimSpace(card.Metadata.Limitations) == "" {
		required = append(required, "MAP: Limitations are missing")
		recs = append(recs, "MAP: Document metadata.limitations and known failure boundaries")
	}

	if strings.TrimSpace(card.Metadata.TrainingData) == "" && strings.TrimSpace(card.Metadata.EvalData) == "" {
		required = append(required, "MAP: Training/evaluation data context is missing")
		recs = append(recs, "MAP: Add metadata.training_data or metadata.eval_data context")
	}

	if len(card.Metadata.Tags) == 0 {
		findings = append(findings, "MAP: No model tags provided for context classification")
		recs = append(recs, "MAP: Add metadata.tags to improve downstream governance and filtering")
	}

	return required, findings, recs
}

func (c *NISTChecker) checkMeasure(card core.ModelCard) ([]string, []string, []string) {
	required := []string{}
	findings := []string{}
	recs := []string{}

	if card.Performance.Accuracy == 0 && card.Performance.Precision == 0 &&
		card.Performance.Recall == 0 && card.Performance.F1 == 0 {
		required = append(required, "MEASURE: Performance evidence is missing")
		recs = append(recs, "MEASURE: Provide non-zero performance metrics with evaluation context")
	}

	if len(card.Fairness.GroupStats) == 0 {
		required = append(required, "MEASURE: Subgroup fairness evidence is missing")
		recs = append(recs, "MEASURE: Provide fairness.group_stats for subgroup analysis")
	}

	if card.Fairness.DemographicParityDiff > 0.2 {
		findings = append(findings, fmt.Sprintf("MEASURE: Demographic parity difference %.3f exceeds advisory threshold 0.2", card.Fairness.DemographicParityDiff))
		recs = append(recs, "MEASURE: Investigate subgroup thresholding/reweighting to reduce parity disparity")
	}

	if card.Fairness.EqualizedOddsDiff > 0.2 {
		findings = append(findings, fmt.Sprintf("MEASURE: Equalized odds difference %.3f exceeds advisory threshold 0.2", card.Fairness.EqualizedOddsDiff))
		recs = append(recs, "MEASURE: Improve error-rate parity via post-processing or retraining")
	}

	return required, findings, recs
}

func (c *NISTChecker) checkManage(card core.ModelCard) ([]string, []string, []string) {
	required := []string{}
	findings := []string{}
	recs := []string{}

	if len(card.RiskAssessment.KnownRisks) == 0 {
		required = append(required, "MANAGE: Known risk register is missing")
		recs = append(recs, "MANAGE: Populate risk_assessment.known_risks")
	}

	if len(card.RiskAssessment.Mitigations) == 0 {
		required = append(required, "MANAGE: Mitigation actions are missing")
		recs = append(recs, "MANAGE: Populate risk_assessment.mitigations")
	}

	if card.Carbon == nil || strings.EqualFold(card.Carbon.Method, "unavailable") || strings.TrimSpace(card.Carbon.Method) == "" {
		findings = append(findings, "MANAGE: Sustainability evidence is unavailable (carbon estimate)")
		recs = append(recs, "MANAGE: Provide carbon evidence via fixture or carbon bridge inputs")
	}

	return required, findings, recs
}
