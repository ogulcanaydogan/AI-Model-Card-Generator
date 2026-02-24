package compliance

import (
	"context"
	"fmt"
	"strings"

	"github.com/yapay/ai-model-card-generator/pkg/core"
)

// EUAIActChecker evaluates model card coverage against practical EU AI Act documentation controls.
type EUAIActChecker struct{}

func (c *EUAIActChecker) Framework() string {
	return "eu-ai-act"
}

func (c *EUAIActChecker) Check(_ context.Context, card core.ModelCard, _ core.CheckOptions) (core.ComplianceReport, error) {
	gaps := []string{}
	findings := []string{}
	recommendations := []string{}

	if strings.TrimSpace(card.Metadata.Name) == "" {
		gaps = append(gaps, "Model identity is missing")
	}
	if strings.TrimSpace(card.Metadata.Owner) == "" {
		gaps = append(gaps, "Accountable provider/owner is missing")
	}
	if strings.TrimSpace(card.Metadata.License) == "" {
		findings = append(findings, "License information is not documented")
		recommendations = append(recommendations, "Add license information for transparent distribution terms")
	}
	if strings.TrimSpace(card.Metadata.Limitations) == "" {
		gaps = append(gaps, "Model limitations are not documented")
		recommendations = append(recommendations, "Document known limitations and operational boundaries")
	}

	if card.Performance.Accuracy == 0 && card.Performance.F1 == 0 {
		gaps = append(gaps, "Performance metrics are missing or empty")
		recommendations = append(recommendations, "Provide evaluation metrics with dataset context")
	}

	if len(card.Fairness.GroupStats) == 0 {
		findings = append(findings, "No subgroup fairness breakdown found")
		recommendations = append(recommendations, "Include group-level fairness evidence")
	}
	if card.Fairness.DemographicParityDiff > 0.2 {
		findings = append(findings, fmt.Sprintf("Demographic parity difference %.3f exceeds advisory threshold 0.2", card.Fairness.DemographicParityDiff))
		recommendations = append(recommendations, "Investigate representation, thresholding, or reweighting for impacted groups")
	}
	if card.Fairness.EqualizedOddsDiff > 0.2 {
		findings = append(findings, fmt.Sprintf("Equalized odds difference %.3f exceeds advisory threshold 0.2", card.Fairness.EqualizedOddsDiff))
		recommendations = append(recommendations, "Apply post-processing or retraining to reduce error-rate disparity")
	}

	score := 100.0 - float64(len(gaps))*15.0 - float64(len(findings))*5.0
	if score < 0 {
		score = 0
	}

	status := "pass"
	if len(gaps) > 0 {
		status = "fail"
	} else if len(findings) > 0 {
		status = "warn"
	}

	return core.ComplianceReport{
		Framework:          c.Framework(),
		Score:              score,
		Status:             status,
		Findings:           findings,
		RequiredGaps:       gaps,
		RecommendedActions: recommendations,
	}, nil
}
