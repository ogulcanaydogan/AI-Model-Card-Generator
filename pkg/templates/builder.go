package templates

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/yapay/ai-model-card-generator/pkg/core"
)

// InitTemplate creates a custom template file from one of the built-in templates.
func InitTemplate(name, outPath, base string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("template name is required")
	}
	if strings.TrimSpace(outPath) == "" {
		return fmt.Errorf("output path is required")
	}

	basePath, err := BuiltInTemplatePath(base)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(basePath)
	if err != nil {
		return fmt.Errorf("read base template: %w", err)
	}

	header := fmt.Sprintf("{{/* Custom template: %s */}}\n", strings.TrimSpace(name))
	final := header + string(content)

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create template output directory: %w", err)
	}
	if err := os.WriteFile(outPath, []byte(final), 0o644); err != nil {
		return fmt.Errorf("write template: %w", err)
	}
	return nil
}

// ValidateTemplateFile parses and executes a template against a sample model card.
func ValidateTemplateFile(templatePath string) error {
	_, err := RenderTemplateFile(templatePath, sampleModelCard())
	return err
}

// RenderTemplateFile renders templatePath using a model card payload.
func RenderTemplateFile(templatePath string, card core.ModelCard) (string, error) {
	if strings.TrimSpace(templatePath) == "" {
		return "", fmt.Errorf("template path is required")
	}
	resolvedPath, err := resolvePathFromParents(templatePath)
	if err != nil {
		return "", fmt.Errorf("resolve template file: %w", err)
	}
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return "", fmt.Errorf("read template file: %w", err)
	}
	tmpl, err := template.New(filepath.Base(resolvedPath)).Option("missingkey=error").Parse(string(data))
	if err != nil {
		return "", fmt.Errorf("parse template file: %w", err)
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, card); err != nil {
		return "", fmt.Errorf("execute template file: %w", err)
	}
	return out.String(), nil
}

// WriteTemplatePreview renders a template with card data and writes the preview markdown to outPath.
func WriteTemplatePreview(templatePath, outPath string, card core.ModelCard) error {
	content, err := RenderTemplateFile(templatePath, card)
	if err != nil {
		return err
	}
	if strings.TrimSpace(outPath) == "" {
		return fmt.Errorf("output path is required")
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create preview output directory: %w", err)
	}
	if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write preview markdown: %w", err)
	}
	return nil
}

// BuiltInTemplatePath resolves supported built-in template names.
func BuiltInTemplatePath(base string) (string, error) {
	name := strings.ToLower(strings.TrimSpace(base))
	if name == "" {
		name = "standard"
	}
	switch name {
	case "standard", "minimal", "eu-ai-act":
		path, err := resolvePathFromParents(filepath.Join("templates", name+".tmpl"))
		if err != nil {
			return "", fmt.Errorf("resolve built-in template %q: %w", name, err)
		}
		return path, nil
	default:
		return "", fmt.Errorf("unsupported base template: %s", base)
	}
}

func sampleModelCard() core.ModelCard {
	return core.ModelCard{
		Version: "v1",
		Metadata: core.ModelMetadata{
			Name:         "sample-model",
			Owner:        "sample-owner",
			License:      "apache-2.0",
			Tags:         []string{"sample", "template"},
			IntendedUse:  "Template validation sample",
			Limitations:  "Not production-ready",
			TrainingData: "Sample training data",
			EvalData:     "Sample eval data",
			Metrics:      map[string]float64{"accuracy": 0.9},
		},
		Performance: core.PerformanceMetrics{
			Accuracy:  0.9,
			Precision: 0.91,
			Recall:    0.89,
			F1:        0.90,
			AUC:       0.92,
		},
		Fairness: core.FairnessMetrics{
			DemographicParityDiff: 0.02,
			EqualizedOddsDiff:     0.03,
			GroupStats: []core.FairnessGroupStats{
				{Group: "A", SelectionRate: 0.5, TruePositiveRate: 0.8, FalsePositiveRate: 0.2, Support: 100},
			},
		},
		Carbon: &core.CarbonEstimate{
			EstimatedKgCO2e: 1.23,
			Method:          "fixture",
		},
		RiskAssessment: core.RiskAssessment{
			KnownRisks:  []string{"Sample risk"},
			Mitigations: []string{"Sample mitigation"},
			BiasNotes:   []string{"Sample bias note"},
		},
		Governance: core.Governance{
			Maintainer:  "sample-owner",
			GeneratedAt: time.Unix(0, 0).UTC(),
			Language:    "en",
		},
		Compliance: []core.ComplianceReport{
			{
				Framework:          "eu-ai-act",
				Score:              95,
				Status:             "warn",
				Findings:           []string{"Sample finding"},
				RequiredGaps:       []string{"Sample required gap"},
				RecommendedActions: []string{"Sample action"},
			},
		},
	}
}

// ParseAndValidateTemplateContent validates a template string against a sample card.
func ParseAndValidateTemplateContent(content string) error {
	tmpl, err := template.New("inline").Option("missingkey=error").Parse(content)
	if err != nil {
		return fmt.Errorf("parse template content: %w", err)
	}
	if err := tmpl.Execute(io.Discard, sampleModelCard()); err != nil {
		return fmt.Errorf("execute template content: %w", err)
	}
	return nil
}

func resolvePathFromParents(path string) (string, error) {
	candidate := filepath.Clean(strings.TrimSpace(path))
	if candidate == "" {
		return "", fmt.Errorf("path is empty")
	}
	if filepath.IsAbs(candidate) {
		if _, err := os.Stat(candidate); err != nil {
			return "", err
		}
		return candidate, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	current := cwd
	for i := 0; i < 8; i++ {
		resolved := filepath.Join(current, candidate)
		if _, err := os.Stat(resolved); err == nil {
			return resolved, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", os.ErrNotExist
}
