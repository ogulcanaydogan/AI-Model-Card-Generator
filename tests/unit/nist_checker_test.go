package unit

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/yapay/ai-model-card-generator/pkg/compliance"
	"github.com/yapay/ai-model-card-generator/pkg/core"
)

func TestNISTCheckerPass(t *testing.T) {
	t.Parallel()
	checker := &compliance.NISTChecker{}

	card := core.ModelCard{
		Metadata: core.ModelMetadata{
			Name:         "demo-model",
			Owner:        "ml-team",
			Tags:         []string{"nlp", "support"},
			IntendedUse:  "Customer support classification",
			Limitations:  "Not suitable for legal decisions",
			TrainingData: "Support ticket corpus v2",
			EvalData:     "Held-out eval sample",
		},
		Performance: core.PerformanceMetrics{
			Accuracy:  0.91,
			Precision: 0.89,
			Recall:    0.88,
			F1:        0.885,
		},
		Fairness: core.FairnessMetrics{
			DemographicParityDiff: 0.08,
			EqualizedOddsDiff:     0.07,
			GroupStats: []core.FairnessGroupStats{
				{Group: "a", SelectionRate: 0.5, TruePositiveRate: 0.9, FalsePositiveRate: 0.1, Support: 120},
				{Group: "b", SelectionRate: 0.48, TruePositiveRate: 0.87, FalsePositiveRate: 0.12, Support: 110},
			},
		},
		Carbon: &core.CarbonEstimate{
			EstimatedKgCO2e: 0.12,
			Method:          "fixture",
		},
		RiskAssessment: core.RiskAssessment{
			KnownRisks:  []string{"Data drift in new channels"},
			Mitigations: []string{"Monthly drift review and threshold recalibration"},
		},
		Governance: core.Governance{
			Maintainer:  "ml-owner@yapay.ai",
			GeneratedAt: time.Now().UTC(),
		},
	}

	report, err := checker.Check(context.Background(), card, core.CheckOptions{})
	if err != nil {
		t.Fatalf("check returned error: %v", err)
	}
	if report.Status != "pass" {
		t.Fatalf("status = %s, want pass", report.Status)
	}
	if len(report.RequiredGaps) != 0 {
		t.Fatalf("required_gaps should be empty: %+v", report.RequiredGaps)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("findings should be empty for pass: %+v", report.Findings)
	}
	if report.Score != 100 {
		t.Fatalf("score = %.2f, want 100", report.Score)
	}
}

func TestNISTCheckerWarn(t *testing.T) {
	t.Parallel()
	checker := &compliance.NISTChecker{}

	card := core.ModelCard{
		Metadata: core.ModelMetadata{
			Name:         "demo-model",
			Owner:        "ml-team",
			Tags:         []string{"nlp"},
			IntendedUse:  "Customer support classification",
			Limitations:  "English-focused",
			TrainingData: "Support ticket corpus v2",
		},
		Performance: core.PerformanceMetrics{
			Accuracy:  0.9,
			Precision: 0.87,
			Recall:    0.86,
			F1:        0.865,
		},
		Fairness: core.FairnessMetrics{
			DemographicParityDiff: 0.25,
			EqualizedOddsDiff:     0.19,
			GroupStats: []core.FairnessGroupStats{
				{Group: "a", SelectionRate: 0.6, TruePositiveRate: 0.91, FalsePositiveRate: 0.11, Support: 100},
				{Group: "b", SelectionRate: 0.45, TruePositiveRate: 0.82, FalsePositiveRate: 0.15, Support: 100},
			},
		},
		Carbon: &core.CarbonEstimate{
			EstimatedKgCO2e: 0,
			Method:          "unavailable",
		},
		RiskAssessment: core.RiskAssessment{
			KnownRisks:  []string{"Potential subgroup drift"},
			Mitigations: []string{"Weekly fairness monitoring"},
		},
		Governance: core.Governance{
			Maintainer:  "ml-owner@yapay.ai",
			GeneratedAt: time.Now().UTC(),
		},
	}

	report, err := checker.Check(context.Background(), card, core.CheckOptions{})
	if err != nil {
		t.Fatalf("check returned error: %v", err)
	}
	if report.Status != "warn" {
		t.Fatalf("status = %s, want warn", report.Status)
	}
	if len(report.RequiredGaps) != 0 {
		t.Fatalf("required_gaps should be empty: %+v", report.RequiredGaps)
	}
	if len(report.Findings) == 0 {
		t.Fatalf("expected advisory findings")
	}
	for _, finding := range report.Findings {
		if !strings.Contains(finding, "[advisory]") {
			t.Fatalf("finding must be advisory-tagged: %s", finding)
		}
	}
}

func TestNISTCheckerFail(t *testing.T) {
	t.Parallel()
	checker := &compliance.NISTChecker{}

	report, err := checker.Check(context.Background(), core.ModelCard{}, core.CheckOptions{})
	if err != nil {
		t.Fatalf("check returned error: %v", err)
	}
	if report.Status != "fail" {
		t.Fatalf("status = %s, want fail", report.Status)
	}
	if len(report.RequiredGaps) == 0 {
		t.Fatalf("expected required gaps")
	}
	for _, gap := range report.RequiredGaps {
		if !strings.Contains(gap, "[required]") {
			t.Fatalf("required gap must be required-tagged: %s", gap)
		}
	}
	assertContainsFunctionGap(t, report.RequiredGaps, "GOVERN:")
	assertContainsFunctionGap(t, report.RequiredGaps, "MAP:")
	assertContainsFunctionGap(t, report.RequiredGaps, "MEASURE:")
	assertContainsFunctionGap(t, report.RequiredGaps, "MANAGE:")
}

func assertContainsFunctionGap(t *testing.T, gaps []string, prefix string) {
	t.Helper()
	for _, gap := range gaps {
		if strings.HasPrefix(gap, prefix) {
			return
		}
	}
	t.Fatalf("required_gaps missing prefix %s: %+v", prefix, gaps)
}
