package analyzers

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/yapay/ai-model-card-generator/pkg/core"
)

// BiasAnalyzer provides heuristic bias risk notes from evaluation data and metadata.
type BiasAnalyzer struct{}

func (a *BiasAnalyzer) Name() string {
	return "bias"
}

func (a *BiasAnalyzer) Analyze(_ context.Context, in core.AnalysisInput) (core.AnalysisResult, error) {
	notes := []string{}
	risks := []string{}

	if strings.TrimSpace(in.Metadata.Limitations) == "" {
		notes = append(notes, "Model limitations are not documented; potential hidden failure modes remain.")
	}

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
	if len(records) <= 1 {
		return core.AnalysisResult{BiasNotes: notes, RiskNotes: risks}, nil
	}

	headers := map[string]int{}
	for i, h := range records[0] {
		headers[strings.TrimSpace(strings.ToLower(h))] = i
	}
	groupIdx, ok := headers["group"]
	if !ok {
		notes = append(notes, "No group column found in evaluation data; fairness confidence is limited.")
		return core.AnalysisResult{BiasNotes: notes, RiskNotes: risks}, nil
	}

	groupCounts := map[string]int{}
	for _, row := range records[1:] {
		if groupIdx >= len(row) {
			continue
		}
		g := strings.TrimSpace(row[groupIdx])
		if g == "" {
			g = "unknown"
		}
		groupCounts[g]++
	}

	if len(groupCounts) < 2 {
		notes = append(notes, "Evaluation data has fewer than 2 groups; subgroup fairness cannot be assessed robustly.")
	}

	counts := make([]int, 0, len(groupCounts))
	for _, c := range groupCounts {
		counts = append(counts, c)
	}
	sort.Ints(counts)
	if len(counts) >= 2 && counts[0] > 0 {
		ratio := float64(counts[len(counts)-1]) / float64(counts[0])
		if ratio >= 2.0 {
			risks = append(risks, "Group representation imbalance exceeds 2x; fairness metrics may be unstable.")
		}
	}

	return core.AnalysisResult{BiasNotes: notes, RiskNotes: risks}, nil
}
