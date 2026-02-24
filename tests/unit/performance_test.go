package unit

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/yapay/ai-model-card-generator/pkg/analyzers"
	"github.com/yapay/ai-model-card-generator/pkg/core"
)

func TestPerformanceAnalyzer(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "eval.csv")
	content := "y_true,y_pred,group\n1,1,a\n1,0,a\n0,1,b\n0,0,b\n"
	if err := os.WriteFile(csvPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write eval csv: %v", err)
	}

	analyzer := &analyzers.PerformanceAnalyzer{}
	res, err := analyzer.Analyze(context.Background(), core.AnalysisInput{EvalFile: csvPath})
	if err != nil {
		t.Fatalf("analyze returned error: %v", err)
	}
	if res.Performance == nil {
		t.Fatalf("performance metrics missing")
	}

	if got := res.Performance.Accuracy; got != 0.5 {
		t.Fatalf("accuracy = %v, want 0.5", got)
	}
	if got := res.Performance.Precision; got != 0.5 {
		t.Fatalf("precision = %v, want 0.5", got)
	}
	if got := res.Performance.Recall; got != 0.5 {
		t.Fatalf("recall = %v, want 0.5", got)
	}
	if got := res.Performance.F1; got != 0.5 {
		t.Fatalf("f1 = %v, want 0.5", got)
	}
}
