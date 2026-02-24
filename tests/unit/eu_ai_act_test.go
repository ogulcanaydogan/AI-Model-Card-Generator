package unit

import (
	"context"
	"testing"

	"github.com/yapay/ai-model-card-generator/pkg/compliance"
	"github.com/yapay/ai-model-card-generator/pkg/core"
)

func TestEUAIActCheckerPass(t *testing.T) {
	t.Parallel()
	checker := &compliance.EUAIActChecker{}
	card := core.ModelCard{
		Metadata: core.ModelMetadata{
			Name:        "demo-model",
			Owner:       "team",
			License:     "apache-2.0",
			Limitations: "limited to English",
		},
		Performance: core.PerformanceMetrics{Accuracy: 0.9, F1: 0.88},
		Fairness: core.FairnessMetrics{
			DemographicParityDiff: 0.1,
			EqualizedOddsDiff:     0.1,
			GroupStats:            []core.FairnessGroupStats{{Group: "a", Support: 10}},
		},
	}

	report, err := checker.Check(context.Background(), card, core.CheckOptions{})
	if err != nil {
		t.Fatalf("check returned error: %v", err)
	}
	if report.Status != "pass" {
		t.Fatalf("status = %s, want pass", report.Status)
	}
}

func TestEUAIActCheckerFail(t *testing.T) {
	t.Parallel()
	checker := &compliance.EUAIActChecker{}
	card := core.ModelCard{}

	report, err := checker.Check(context.Background(), card, core.CheckOptions{})
	if err != nil {
		t.Fatalf("check returned error: %v", err)
	}
	if report.Status != "fail" {
		t.Fatalf("status = %s, want fail", report.Status)
	}
	if len(report.RequiredGaps) == 0 {
		t.Fatalf("expected required gaps")
	}
}
