package core

import "context"

// Extractor retrieves metadata from a model registry.
type Extractor interface {
	Name() string
	Extract(ctx context.Context, ref ModelRef) (ModelMetadata, error)
}

// Analyzer computes model quality and risk metrics.
type Analyzer interface {
	Name() string
	Analyze(ctx context.Context, in AnalysisInput) (AnalysisResult, error)
}

// Generator creates model-card artifacts.
type Generator interface {
	Format() string
	Generate(ctx context.Context, card ModelCard, templatePath, outPath string) error
}

// ComplianceChecker evaluates a model card against a framework.
type ComplianceChecker interface {
	Framework() string
	Check(ctx context.Context, card ModelCard, opts CheckOptions) (ComplianceReport, error)
}
