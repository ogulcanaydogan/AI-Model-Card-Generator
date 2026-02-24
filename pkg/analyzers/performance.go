package analyzers

import (
	"context"
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/yapay/ai-model-card-generator/pkg/core"
)

// PerformanceAnalyzer computes classification metrics from eval CSV.
type PerformanceAnalyzer struct{}

type scorePair struct {
	score float64
	label bool
}

func (a *PerformanceAnalyzer) Name() string {
	return "performance"
}

func (a *PerformanceAnalyzer) Analyze(_ context.Context, in core.AnalysisInput) (core.AnalysisResult, error) {
	f, err := os.Open(in.EvalFile)
	if err != nil {
		return core.AnalysisResult{}, fmt.Errorf("open eval file: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return core.AnalysisResult{}, fmt.Errorf("read eval csv: %w", err)
	}
	if len(records) < 2 {
		return core.AnalysisResult{}, fmt.Errorf("eval csv must contain header and at least one row")
	}

	headers := map[string]int{}
	for idx, h := range records[0] {
		headers[strings.TrimSpace(strings.ToLower(h))] = idx
	}
	for _, required := range []string{"y_true", "y_pred", "group"} {
		if _, ok := headers[required]; !ok {
			return core.AnalysisResult{}, fmt.Errorf("eval csv missing required column: %s", required)
		}
	}

	var tp, tn, fp, fn float64
	scores := []scorePair{}

	for _, row := range records[1:] {
		if len(row) < len(records[0]) {
			continue
		}
		yTrue, err := parseBinaryLabel(row[headers["y_true"]])
		if err != nil {
			return core.AnalysisResult{}, fmt.Errorf("parse y_true: %w", err)
		}
		yPred, err := parseBinaryLabel(row[headers["y_pred"]])
		if err != nil {
			return core.AnalysisResult{}, fmt.Errorf("parse y_pred: %w", err)
		}

		switch {
		case yTrue && yPred:
			tp++
		case !yTrue && !yPred:
			tn++
		case !yTrue && yPred:
			fp++
		case yTrue && !yPred:
			fn++
		}

		if scoreIdx, ok := headers["y_score"]; ok {
			s, err := strconv.ParseFloat(strings.TrimSpace(row[scoreIdx]), 64)
			if err == nil {
				scores = append(scores, scorePair{score: s, label: yTrue})
			}
		}
	}

	total := tp + tn + fp + fn
	if total == 0 {
		return core.AnalysisResult{}, fmt.Errorf("no valid evaluation rows found")
	}

	metrics := core.PerformanceMetrics{
		Accuracy:  safeDiv(tp+tn, total),
		Precision: safeDiv(tp, tp+fp),
		Recall:    safeDiv(tp, tp+fn),
		F1:        0,
	}
	if metrics.Precision+metrics.Recall > 0 {
		metrics.F1 = 2 * metrics.Precision * metrics.Recall / (metrics.Precision + metrics.Recall)
	}
	if len(scores) > 0 {
		metrics.AUC = computeAUC(scores)
	}

	return core.AnalysisResult{Performance: &metrics}, nil
}

func safeDiv(numerator, denominator float64) float64 {
	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}

func parseBinaryLabel(v string) (bool, error) {
	s := strings.ToLower(strings.TrimSpace(v))
	switch s {
	case "1", "true", "yes", "positive", "pos":
		return true, nil
	case "0", "false", "no", "negative", "neg":
		return false, nil
	default:
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return false, fmt.Errorf("unsupported binary label value: %q", v)
		}
		return n >= 0.5, nil
	}
}

func computeAUC(scores []scorePair) float64 {
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].score == scores[j].score {
			if scores[i].label == scores[j].label {
				return false
			}
			return scores[i].label && !scores[j].label
		}
		return scores[i].score < scores[j].score
	})

	var posCount, negCount int
	for _, s := range scores {
		if s.label {
			posCount++
		} else {
			negCount++
		}
	}
	if posCount == 0 || negCount == 0 {
		return 0
	}

	// Mann-Whitney U based AUC.
	rank := 1.0
	var posRankSum float64
	for i := 0; i < len(scores); {
		j := i + 1
		for j < len(scores) && math.Abs(scores[j].score-scores[i].score) < 1e-12 {
			j++
		}
		avgRank := (rank + float64(j)) / 2.0
		for k := i; k < j; k++ {
			if scores[k].label {
				posRankSum += avgRank
			}
		}
		rank = float64(j + 1)
		i = j
	}

	u := posRankSum - float64(posCount*(posCount+1))/2.0
	return u / float64(posCount*negCount)
}
