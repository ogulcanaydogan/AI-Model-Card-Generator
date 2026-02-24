package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xeipuuv/gojsonschema"
)

// Pipeline orchestrates extraction, analysis, compliance checks, and output generation.
type Pipeline struct {
	Extractors          map[string]Extractor
	Analyzers           []Analyzer
	Generators          map[string]Generator
	ComplianceCheckers  map[string]ComplianceChecker
	DefaultTemplatePath string
}

// Generate creates a model card and writes requested output artifacts.
func (p *Pipeline) Generate(ctx context.Context, opts GenerateOptions) (ModelCard, error) {
	extractor, ok := p.Extractors[strings.ToLower(opts.Ref.Source)]
	if !ok {
		return ModelCard{}, fmt.Errorf("%w: %s", ErrUnsupportedSource, opts.Ref.Source)
	}

	if opts.EvalFile == "" {
		return ModelCard{}, ErrMissingEvalFile
	}

	if _, err := os.Stat(opts.EvalFile); err != nil {
		return ModelCard{}, Wrap("stat eval file", err)
	}

	metadata, err := extractor.Extract(ctx, opts.Ref)
	if err != nil {
		return ModelCard{}, Wrap("extract metadata", err)
	}

	analysisInput := AnalysisInput{
		Ref:      opts.Ref,
		Metadata: metadata,
		EvalFile: opts.EvalFile,
	}

	perf := PerformanceMetrics{}
	fairness := FairnessMetrics{}
	biasNotes := []string{}
	riskNotes := []string{}

	for _, analyzer := range p.Analyzers {
		result, err := analyzer.Analyze(ctx, analysisInput)
		if err != nil {
			return ModelCard{}, Wrap("analyze "+analyzer.Name(), err)
		}
		if result.Performance != nil {
			perf = *result.Performance
		}
		if result.Fairness != nil {
			fairness = *result.Fairness
		}
		if len(result.BiasNotes) > 0 {
			biasNotes = append(biasNotes, result.BiasNotes...)
		}
		if len(result.RiskNotes) > 0 {
			riskNotes = append(riskNotes, result.RiskNotes...)
		}
	}

	card := ModelCard{
		Version:     "v1",
		Metadata:    metadata,
		Performance: perf,
		Fairness:    fairness,
		RiskAssessment: RiskAssessment{
			KnownRisks:  riskNotes,
			Mitigations: []string{"Collect representative evaluation data", "Monitor drift and subgroup behavior", "Document model limitations"},
			BiasNotes:   biasNotes,
		},
		Governance: Governance{
			Maintainer:  metadata.Owner,
			GeneratedAt: time.Now().UTC(),
			Language:    defaultIfEmpty(opts.Language, "en"),
		},
	}

	frameworks := opts.ComplianceFrameworks
	if len(frameworks) == 0 {
		frameworks = []string{"eu-ai-act"}
	}

	reports := make([]ComplianceReport, 0, len(frameworks))
	for _, framework := range frameworks {
		checker, ok := p.ComplianceCheckers[strings.ToLower(framework)]
		if !ok {
			return ModelCard{}, fmt.Errorf("%w: %s", ErrComplianceFramework, framework)
		}
		report, err := checker.Check(ctx, card, CheckOptions{Strict: false})
		if err != nil {
			return ModelCard{}, Wrap("compliance check", err)
		}
		reports = append(reports, report)
	}
	card.Compliance = reports

	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return ModelCard{}, Wrap("create output dir", err)
	}

	generatedFiles := map[string]string{}
	templatePath := filepath.Join(p.DefaultTemplatePath, opts.Template+".tmpl")
	if opts.Template == "" {
		templatePath = filepath.Join(p.DefaultTemplatePath, "standard.tmpl")
	}

	for _, format := range opts.Formats {
		normalized := strings.ToLower(strings.TrimSpace(format))
		generator, ok := p.Generators[normalized]
		if !ok {
			return ModelCard{}, fmt.Errorf("%w: %s", ErrUnsupportedFormat, normalized)
		}
		ext := normalized
		if normalized == "md" {
			ext = "md"
		}
		outPath := filepath.Join(opts.OutDir, "model_card."+ext)
		if err := generator.Generate(ctx, card, templatePath, outPath); err != nil {
			return ModelCard{}, Wrap("generate "+normalized, err)
		}
		generatedFiles[normalized] = outPath
	}

	compliancePath := filepath.Join(opts.OutDir, "compliance_report.json")
	reportBytes, err := json.MarshalIndent(reports, "", "  ")
	if err != nil {
		return ModelCard{}, Wrap("marshal compliance report", err)
	}
	if err := os.WriteFile(compliancePath, reportBytes, 0o644); err != nil {
		return ModelCard{}, Wrap("write compliance report", err)
	}

	card.Artifacts = Artifacts{
		GeneratedFiles: generatedFiles,
		CompliancePath: compliancePath,
	}

	// Ensure JSON artifact always reflects final card payload.
	if _, hasJSON := generatedFiles["json"]; !hasJSON {
		if gen, ok := p.Generators["json"]; ok {
			jsonPath := filepath.Join(opts.OutDir, "model_card.json")
			if err := gen.Generate(ctx, card, templatePath, jsonPath); err != nil {
				return ModelCard{}, Wrap("generate json", err)
			}
			card.Artifacts.GeneratedFiles["json"] = jsonPath
		}
	}

	return card, nil
}

// ValidateJSONSchema validates JSON documents against a JSON schema.
func ValidateJSONSchema(schemaPath, inputPath string) error {
	schemaLoader := gojsonschema.NewReferenceLoader("file://" + filepath.Clean(schemaPath))
	documentLoader := gojsonschema.NewReferenceLoader("file://" + filepath.Clean(inputPath))

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return Wrap("run schema validation", err)
	}
	if result.Valid() {
		return nil
	}

	messages := make([]string, 0, len(result.Errors()))
	for _, desc := range result.Errors() {
		messages = append(messages, desc.String())
	}
	return fmt.Errorf("%w: %s", ErrSchemaValidationFail, strings.Join(messages, "; "))
}

// LoadModelCard loads a model card from JSON.
func LoadModelCard(path string) (ModelCard, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ModelCard{}, Wrap("read model card", err)
	}
	var card ModelCard
	if err := json.Unmarshal(data, &card); err != nil {
		return ModelCard{}, Wrap("parse model card", err)
	}
	return card, nil
}

// SaveModelCard writes a model card as JSON.
func SaveModelCard(path string, card ModelCard) error {
	data, err := json.MarshalIndent(card, "", "  ")
	if err != nil {
		return Wrap("marshal model card", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return Wrap("write model card", err)
	}
	return nil
}

// StrictComplianceExit returns true if strict mode should fail process.
func StrictComplianceExit(reports []ComplianceReport, strict bool) bool {
	if !strict {
		return false
	}
	for _, r := range reports {
		if strings.EqualFold(r.Status, "fail") && len(r.RequiredGaps) > 0 {
			return true
		}
	}
	return false
}

func defaultIfEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

// EnsureErrorsJoin keeps compatibility for older versions if needed.
func EnsureErrorsJoin(errs ...error) error {
	return errors.Join(errs...)
}
