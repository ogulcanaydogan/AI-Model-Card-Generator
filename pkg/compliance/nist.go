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
	nistRequiredPenalty            = 12.0
	nistAdvisoryPenalty            = 4.0
	nistDemographicParityThreshold = 0.20
	nistEqualizedOddsThreshold     = 0.20
)

const (
	nistRequired = "required"
	nistAdvisory = "advisory"
)

type nistControl struct {
	Function     string
	ID           string
	Requirement  string
	Evidence     string
	Description  string
	Remediation  string
	EvaluateFunc func(card core.ModelCard) nistControlEvaluation
}

type nistControlEvaluation struct {
	Passed bool
	Detail string
}

func (c *NISTChecker) Framework() string {
	return "nist"
}

func (c *NISTChecker) Check(_ context.Context, card core.ModelCard, _ core.CheckOptions) (core.ComplianceReport, error) {
	requiredGaps := []string{}
	findings := []string{}
	recommendations := []string{}

	for _, control := range nistControlCatalog() {
		evaluation := control.EvaluateFunc(card)
		if evaluation.Passed {
			continue
		}

		message := formatControlMessage(control, evaluation.Detail)
		switch control.Requirement {
		case nistRequired:
			requiredGaps = append(requiredGaps, message)
		default:
			findings = append(findings, message)
		}
		recommendations = append(recommendations, formatControlRecommendation(control))
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

func nistControlCatalog() []nistControl {
	return []nistControl{
		{
			Function:    "GOVERN",
			ID:          "GOV-1",
			Requirement: nistRequired,
			Evidence:    "metadata.owner",
			Description: "Accountable owner/provider is missing",
			Remediation: "Set metadata.owner to document accountable ownership",
			EvaluateFunc: func(card core.ModelCard) nistControlEvaluation {
				return nistControlEvaluation{Passed: strings.TrimSpace(card.Metadata.Owner) != ""}
			},
		},
		{
			Function:    "GOVERN",
			ID:          "GOV-2",
			Requirement: nistRequired,
			Evidence:    "governance.maintainer",
			Description: "Maintainer contact is missing",
			Remediation: "Populate governance.maintainer for operational accountability",
			EvaluateFunc: func(card core.ModelCard) nistControlEvaluation {
				return nistControlEvaluation{Passed: strings.TrimSpace(card.Governance.Maintainer) != ""}
			},
		},
		{
			Function:    "GOVERN",
			ID:          "GOV-3",
			Requirement: nistAdvisory,
			Evidence:    "governance.generated_at",
			Description: "Card generation timestamp is missing",
			Remediation: "Include governance.generated_at for traceability",
			EvaluateFunc: func(card core.ModelCard) nistControlEvaluation {
				return nistControlEvaluation{Passed: !card.Governance.GeneratedAt.IsZero()}
			},
		},
		{
			Function:    "GOVERN",
			ID:          "GOV-4",
			Requirement: nistAdvisory,
			Evidence:    "metadata.license",
			Description: "License declaration is missing",
			Remediation: "Populate metadata.license to support governance and legal review",
			EvaluateFunc: func(card core.ModelCard) nistControlEvaluation {
				return nistControlEvaluation{Passed: strings.TrimSpace(card.Metadata.License) != ""}
			},
		},
		{
			Function:    "MAP",
			ID:          "MAP-1",
			Requirement: nistRequired,
			Evidence:    "metadata.intended_use",
			Description: "Intended use is missing",
			Remediation: "Document metadata.intended_use with expected operating context",
			EvaluateFunc: func(card core.ModelCard) nistControlEvaluation {
				return nistControlEvaluation{Passed: strings.TrimSpace(card.Metadata.IntendedUse) != ""}
			},
		},
		{
			Function:    "MAP",
			ID:          "MAP-2",
			Requirement: nistRequired,
			Evidence:    "metadata.limitations",
			Description: "Limitations are missing",
			Remediation: "Document metadata.limitations and known failure boundaries",
			EvaluateFunc: func(card core.ModelCard) nistControlEvaluation {
				return nistControlEvaluation{Passed: strings.TrimSpace(card.Metadata.Limitations) != ""}
			},
		},
		{
			Function:    "MAP",
			ID:          "MAP-3",
			Requirement: nistRequired,
			Evidence:    "metadata.training_data|metadata.eval_data",
			Description: "Training/evaluation data context is missing",
			Remediation: "Add metadata.training_data or metadata.eval_data context",
			EvaluateFunc: func(card core.ModelCard) nistControlEvaluation {
				hasContext := strings.TrimSpace(card.Metadata.TrainingData) != "" || strings.TrimSpace(card.Metadata.EvalData) != ""
				return nistControlEvaluation{Passed: hasContext}
			},
		},
		{
			Function:    "MAP",
			ID:          "MAP-4",
			Requirement: nistAdvisory,
			Evidence:    "metadata.tags",
			Description: "No model tags provided for context classification",
			Remediation: "Add metadata.tags to improve downstream governance and filtering",
			EvaluateFunc: func(card core.ModelCard) nistControlEvaluation {
				return nistControlEvaluation{Passed: len(card.Metadata.Tags) > 0}
			},
		},
		{
			Function:    "MAP",
			ID:          "MAP-5",
			Requirement: nistAdvisory,
			Evidence:    "metadata.name",
			Description: "Model name is missing",
			Remediation: "Set metadata.name to a clear model identifier",
			EvaluateFunc: func(card core.ModelCard) nistControlEvaluation {
				return nistControlEvaluation{Passed: strings.TrimSpace(card.Metadata.Name) != ""}
			},
		},
		{
			Function:    "MEASURE",
			ID:          "MEA-1",
			Requirement: nistRequired,
			Evidence:    "performance.{accuracy,precision,recall,f1,auc}",
			Description: "Performance evidence is missing",
			Remediation: "Provide non-zero performance metrics with evaluation context",
			EvaluateFunc: func(card core.ModelCard) nistControlEvaluation {
				return nistControlEvaluation{Passed: hasPerformanceEvidence(card.Performance)}
			},
		},
		{
			Function:    "MEASURE",
			ID:          "MEA-2",
			Requirement: nistRequired,
			Evidence:    "fairness.group_stats",
			Description: "Subgroup fairness evidence is missing",
			Remediation: "Provide fairness.group_stats for subgroup analysis",
			EvaluateFunc: func(card core.ModelCard) nistControlEvaluation {
				return nistControlEvaluation{Passed: len(card.Fairness.GroupStats) > 0}
			},
		},
		{
			Function:    "MEASURE",
			ID:          "MEA-3",
			Requirement: nistAdvisory,
			Evidence:    "fairness.group_stats[*].group",
			Description: "Comparative subgroup fairness evidence is weak",
			Remediation: "Evaluate at least two relevant groups for comparative fairness",
			EvaluateFunc: func(card core.ModelCard) nistControlEvaluation {
				if len(card.Fairness.GroupStats) == 0 {
					return nistControlEvaluation{Passed: true}
				}
				if len(card.Fairness.GroupStats) >= 2 {
					return nistControlEvaluation{Passed: true}
				}
				return nistControlEvaluation{
					Passed: false,
					Detail: "only one subgroup is present",
				}
			},
		},
		{
			Function:    "MEASURE",
			ID:          "MEA-4",
			Requirement: nistAdvisory,
			Evidence:    "fairness.demographic_parity_diff",
			Description: "Demographic parity difference exceeds threshold",
			Remediation: "Investigate subgroup thresholding/reweighting to reduce parity disparity",
			EvaluateFunc: func(card core.ModelCard) nistControlEvaluation {
				if card.Fairness.DemographicParityDiff <= nistDemographicParityThreshold {
					return nistControlEvaluation{Passed: true}
				}
				return nistControlEvaluation{
					Passed: false,
					Detail: fmt.Sprintf("value %.3f exceeds %.2f", card.Fairness.DemographicParityDiff, nistDemographicParityThreshold),
				}
			},
		},
		{
			Function:    "MEASURE",
			ID:          "MEA-5",
			Requirement: nistAdvisory,
			Evidence:    "fairness.equalized_odds_diff",
			Description: "Equalized odds difference exceeds threshold",
			Remediation: "Improve error-rate parity via post-processing or retraining",
			EvaluateFunc: func(card core.ModelCard) nistControlEvaluation {
				if card.Fairness.EqualizedOddsDiff <= nistEqualizedOddsThreshold {
					return nistControlEvaluation{Passed: true}
				}
				return nistControlEvaluation{
					Passed: false,
					Detail: fmt.Sprintf("value %.3f exceeds %.2f", card.Fairness.EqualizedOddsDiff, nistEqualizedOddsThreshold),
				}
			},
		},
		{
			Function:    "MEASURE",
			ID:          "MEA-6",
			Requirement: nistAdvisory,
			Evidence:    "metadata.metrics",
			Description: "Extractor-level metric context is missing",
			Remediation: "Include source metrics in metadata.metrics for traceability",
			EvaluateFunc: func(card core.ModelCard) nistControlEvaluation {
				return nistControlEvaluation{Passed: len(card.Metadata.Metrics) > 0}
			},
		},
		{
			Function:    "MANAGE",
			ID:          "MAN-1",
			Requirement: nistRequired,
			Evidence:    "risk_assessment.known_risks",
			Description: "Known risk register is missing",
			Remediation: "Populate risk_assessment.known_risks",
			EvaluateFunc: func(card core.ModelCard) nistControlEvaluation {
				return nistControlEvaluation{Passed: len(card.RiskAssessment.KnownRisks) > 0}
			},
		},
		{
			Function:    "MANAGE",
			ID:          "MAN-2",
			Requirement: nistRequired,
			Evidence:    "risk_assessment.mitigations",
			Description: "Mitigation actions are missing",
			Remediation: "Populate risk_assessment.mitigations",
			EvaluateFunc: func(card core.ModelCard) nistControlEvaluation {
				return nistControlEvaluation{Passed: len(card.RiskAssessment.Mitigations) > 0}
			},
		},
		{
			Function:    "MANAGE",
			ID:          "MAN-3",
			Requirement: nistRequired,
			Evidence:    "risk_assessment.known_risks|risk_assessment.mitigations",
			Description: "Risk-to-mitigation coverage is incomplete",
			Remediation: "Provide at least one mitigation plan per known risk",
			EvaluateFunc: func(card core.ModelCard) nistControlEvaluation {
				risks := len(card.RiskAssessment.KnownRisks)
				mitigations := len(card.RiskAssessment.Mitigations)
				if risks == 0 || mitigations == 0 {
					return nistControlEvaluation{Passed: true}
				}
				if mitigations >= risks {
					return nistControlEvaluation{Passed: true}
				}
				return nistControlEvaluation{
					Passed: false,
					Detail: fmt.Sprintf("%d mitigations for %d risks", mitigations, risks),
				}
			},
		},
		{
			Function:    "MANAGE",
			ID:          "MAN-4",
			Requirement: nistAdvisory,
			Evidence:    "carbon.method",
			Description: "Sustainability evidence is unavailable",
			Remediation: "Provide carbon evidence via fixture or carbon bridge inputs",
			EvaluateFunc: func(card core.ModelCard) nistControlEvaluation {
				return nistControlEvaluation{Passed: hasCarbonEvidence(card)}
			},
		},
		{
			Function:    "MANAGE",
			ID:          "MAN-5",
			Requirement: nistAdvisory,
			Evidence:    "risk_assessment.bias_notes",
			Description: "Bias monitoring notes are missing",
			Remediation: "Add risk_assessment.bias_notes with monitoring observations",
			EvaluateFunc: func(card core.ModelCard) nistControlEvaluation {
				return nistControlEvaluation{Passed: len(card.RiskAssessment.BiasNotes) > 0}
			},
		},
		{
			Function:    "MANAGE",
			ID:          "MAN-6",
			Requirement: nistAdvisory,
			Evidence:    "governance.language",
			Description: "Governance language metadata is missing",
			Remediation: "Set governance.language for localization traceability",
			EvaluateFunc: func(card core.ModelCard) nistControlEvaluation {
				return nistControlEvaluation{Passed: strings.TrimSpace(card.Governance.Language) != ""}
			},
		},
	}
}

func formatControlMessage(control nistControl, detail string) string {
	message := fmt.Sprintf(
		"%s: [%s][%s][evidence:%s] %s",
		control.Function,
		control.ID,
		control.Requirement,
		control.Evidence,
		control.Description,
	)
	if strings.TrimSpace(detail) != "" {
		message = fmt.Sprintf("%s (%s)", message, strings.TrimSpace(detail))
	}
	return message
}

func formatControlRecommendation(control nistControl) string {
	return fmt.Sprintf("%s: [%s] %s", control.Function, control.ID, control.Remediation)
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

func hasCarbonEvidence(card core.ModelCard) bool {
	if card.Carbon == nil {
		return false
	}
	method := strings.TrimSpace(card.Carbon.Method)
	if method == "" {
		return false
	}
	return !strings.EqualFold(method, "unavailable")
}
