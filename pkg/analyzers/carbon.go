package analyzers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/yapay/ai-model-card-generator/pkg/core"
)

// CarbonAnalyzer computes/loads carbon estimate metadata.
type CarbonAnalyzer struct {
	PythonBin   string
	ScriptPath  string
	FixturePath string
}

func (a *CarbonAnalyzer) Name() string {
	return "carbon"
}

func (a *CarbonAnalyzer) Analyze(ctx context.Context, in core.AnalysisInput) (core.AnalysisResult, error) {
	if strings.TrimSpace(a.FixturePath) != "" {
		estimate, err := loadCarbonEstimate(a.FixturePath)
		if err != nil {
			return core.AnalysisResult{}, fmt.Errorf("load carbon fixture: %w", err)
		}
		return buildCarbonResult(estimate, ""), nil
	}

	pythonBin := firstNonEmpty(a.PythonBin, "python3")
	scriptPath := firstNonEmpty(a.ScriptPath, filepath.Join("scripts", "carbon_metrics.py"))
	estimate, err := runCarbonBridge(ctx, pythonBin, scriptPath, in.EvalFile)
	if err != nil {
		return unavailableCarbonResult(err), nil
	}
	return buildCarbonResult(estimate, ""), nil
}

func runCarbonBridge(ctx context.Context, pythonBin, scriptPath, evalFile string) (core.CarbonEstimate, error) {
	tmpFile, err := os.CreateTemp("", "mcg-carbon-*.json")
	if err != nil {
		return core.CarbonEstimate{}, fmt.Errorf("create temp output: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	cmd := exec.CommandContext(ctx, pythonBin, scriptPath, "--input", evalFile, "--output", tmpPath)
	stderr := &strings.Builder{}
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return core.CarbonEstimate{}, fmt.Errorf("python runtime not found")
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return core.CarbonEstimate{}, fmt.Errorf("run carbon script: %w", err)
		}
		return core.CarbonEstimate{}, fmt.Errorf("run carbon script: %w (%s)", err, msg)
	}

	estimate, err := loadCarbonEstimate(tmpPath)
	if err != nil {
		return core.CarbonEstimate{}, fmt.Errorf("parse carbon output: %w", err)
	}
	return estimate, nil
}

func loadCarbonEstimate(path string) (core.CarbonEstimate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return core.CarbonEstimate{}, fmt.Errorf("read file: %w", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return core.CarbonEstimate{}, fmt.Errorf("decode json: %w", err)
	}

	method := stringValue(payload["method"])
	if strings.TrimSpace(method) == "" {
		method = "unknown"
	}

	kg := firstNumber(payload, "estimated_kg_co2e", "kg_co2e", "emissions")
	if kg < 0 {
		kg = 0
	}

	return core.CarbonEstimate{
		EstimatedKgCO2e: kg,
		Method:          strings.ToLower(strings.TrimSpace(method)),
	}, nil
}

func unavailableCarbonResult(cause error) core.AnalysisResult {
	note := "Carbon estimate unavailable; provide fixture via MCG_CARBON_FIXTURE or configure scripts/carbon_metrics.py inputs."
	if cause != nil {
		note = fmt.Sprintf("Carbon estimate unavailable (%s); provide fixture via MCG_CARBON_FIXTURE or configure scripts/carbon_metrics.py inputs.", truncateString(cause.Error(), 160))
	}
	return core.AnalysisResult{
		Carbon: &core.CarbonEstimate{
			EstimatedKgCO2e: 0,
			Method:          "unavailable",
		},
		RiskNotes: []string{note},
	}
}

func buildCarbonResult(estimate core.CarbonEstimate, contextNote string) core.AnalysisResult {
	result := core.AnalysisResult{Carbon: &estimate}
	if strings.EqualFold(estimate.Method, "unavailable") {
		note := "Carbon estimate unavailable; include CodeCarbon-like measurement evidence for sustainability reporting."
		if strings.TrimSpace(contextNote) != "" {
			note = contextNote + " " + note
		}
		result.RiskNotes = []string{strings.TrimSpace(note)}
	}
	return result
}

func firstNumber(payload map[string]any, keys ...string) float64 {
	for _, key := range keys {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		switch v := raw.(type) {
		case float64:
			return v
		case float32:
			return float64(v)
		case int:
			return float64(v)
		case int32:
			return float64(v)
		case int64:
			return float64(v)
		case json.Number:
			if n, err := v.Float64(); err == nil {
				return n
			}
		case string:
			if n, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
				return n
			}
		}
	}
	return 0
}

func stringValue(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case fmt.Stringer:
		return s.String()
	default:
		return ""
	}
}

func truncateString(v string, max int) string {
	if max <= 0 || len(v) <= max {
		return v
	}
	return v[:max]
}
