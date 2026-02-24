package analyzers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yapay/ai-model-card-generator/pkg/core"
)

// FairnessAnalyzer delegates fairness computations to Python Fairlearn.
type FairnessAnalyzer struct {
	PythonBin  string
	ScriptPath string
}

func (a *FairnessAnalyzer) Name() string {
	return "fairness"
}

func (a *FairnessAnalyzer) Analyze(ctx context.Context, in core.AnalysisInput) (core.AnalysisResult, error) {
	pythonBin := firstNonEmpty(a.PythonBin, "python3")
	scriptPath := firstNonEmpty(a.ScriptPath, filepath.Join("scripts", "fairness_metrics.py"))

	tmpFile, err := os.CreateTemp("", "mcg-fairness-*.json")
	if err != nil {
		return core.AnalysisResult{}, fmt.Errorf("create temp output: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	cmd := exec.CommandContext(ctx, pythonBin, scriptPath, "--input", in.EvalFile, "--output", tmpPath)
	stderr := &strings.Builder{}
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return core.AnalysisResult{}, fmt.Errorf("python runtime not found. Install Python 3.11+ and run `pip install fairlearn pandas`")
		}
		output := strings.TrimSpace(stderr.String())
		if strings.Contains(strings.ToLower(output), "modulenotfounderror") {
			return core.AnalysisResult{}, fmt.Errorf("fairlearn dependency missing. Install with `pip install fairlearn pandas`")
		}
		return core.AnalysisResult{}, fmt.Errorf("run fairness script: %w (%s)", err, output)
	}

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return core.AnalysisResult{}, fmt.Errorf("read fairness output: %w", err)
	}

	var parsed struct {
		DemographicParityDiff float64 `json:"demographic_parity_diff"`
		EqualizedOddsDiff     float64 `json:"equalized_odds_diff"`
		GroupStats            []struct {
			Group             string  `json:"group"`
			SelectionRate     float64 `json:"selection_rate"`
			TruePositiveRate  float64 `json:"true_positive_rate"`
			FalsePositiveRate float64 `json:"false_positive_rate"`
			Support           int     `json:"support"`
		} `json:"group_stats"`
	}

	if err := json.Unmarshal(data, &parsed); err != nil {
		return core.AnalysisResult{}, fmt.Errorf("parse fairness output: %w", err)
	}

	groupStats := make([]core.FairnessGroupStats, 0, len(parsed.GroupStats))
	for _, s := range parsed.GroupStats {
		groupStats = append(groupStats, core.FairnessGroupStats{
			Group:             s.Group,
			SelectionRate:     s.SelectionRate,
			TruePositiveRate:  s.TruePositiveRate,
			FalsePositiveRate: s.FalsePositiveRate,
			Support:           s.Support,
		})
	}

	metrics := core.FairnessMetrics{
		DemographicParityDiff: parsed.DemographicParityDiff,
		EqualizedOddsDiff:     parsed.EqualizedOddsDiff,
		GroupStats:            groupStats,
	}

	return core.AnalysisResult{Fairness: &metrics}, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
