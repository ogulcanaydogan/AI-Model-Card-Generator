package core

import "time"

// ModelRef identifies a model across supported registries.
type ModelRef struct {
	Source  string `json:"source"`
	ID      string `json:"id"`
	Version string `json:"version,omitempty"`
	URI     string `json:"uri,omitempty"`
}

// ModelMetadata contains normalized metadata extracted from registries.
type ModelMetadata struct {
	Name         string             `json:"name"`
	Owner        string             `json:"owner,omitempty"`
	License      string             `json:"license,omitempty"`
	Tags         []string           `json:"tags,omitempty"`
	IntendedUse  string             `json:"intended_use,omitempty"`
	Limitations  string             `json:"limitations,omitempty"`
	TrainingData string             `json:"training_data,omitempty"`
	EvalData     string             `json:"eval_data,omitempty"`
	Metrics      map[string]float64 `json:"metrics,omitempty"`
}

// PerformanceMetrics stores model quality metrics.
type PerformanceMetrics struct {
	Accuracy  float64 `json:"accuracy"`
	Precision float64 `json:"precision"`
	Recall    float64 `json:"recall"`
	F1        float64 `json:"f1"`
	AUC       float64 `json:"auc,omitempty"`
}

// FairnessGroupStats stores per-group prediction behavior.
type FairnessGroupStats struct {
	Group             string  `json:"group"`
	SelectionRate     float64 `json:"selection_rate"`
	TruePositiveRate  float64 `json:"true_positive_rate"`
	FalsePositiveRate float64 `json:"false_positive_rate"`
	Support           int     `json:"support"`
}

// FairnessMetrics stores fairness indicators.
type FairnessMetrics struct {
	DemographicParityDiff float64              `json:"demographic_parity_diff"`
	EqualizedOddsDiff     float64              `json:"equalized_odds_diff"`
	GroupStats            []FairnessGroupStats `json:"group_stats,omitempty"`
}

// CarbonEstimate stores environment impact data.
type CarbonEstimate struct {
	EstimatedKgCO2e float64 `json:"estimated_kg_co2e"`
	Method          string  `json:"method"`
}

// RiskAssessment stores identified risks and mitigations.
type RiskAssessment struct {
	KnownRisks  []string `json:"known_risks,omitempty"`
	Mitigations []string `json:"mitigations,omitempty"`
	BiasNotes   []string `json:"bias_notes,omitempty"`
}

// Governance stores accountability data.
type Governance struct {
	Maintainer  string    `json:"maintainer,omitempty"`
	GeneratedAt time.Time `json:"generated_at"`
	Language    string    `json:"language"`
}

// ComplianceReport stores compliance analysis results.
type ComplianceReport struct {
	Framework          string   `json:"framework"`
	Score              float64  `json:"score"`
	Status             string   `json:"status"`
	Findings           []string `json:"findings,omitempty"`
	RequiredGaps       []string `json:"required_gaps,omitempty"`
	RecommendedActions []string `json:"recommended_actions,omitempty"`
}

// Artifacts stores output file paths.
type Artifacts struct {
	GeneratedFiles map[string]string `json:"generated_files,omitempty"`
	CompliancePath string            `json:"compliance_path,omitempty"`
}

// ModelCard is the canonical model card payload.
type ModelCard struct {
	Version        string             `json:"version"`
	Metadata       ModelMetadata      `json:"metadata"`
	Performance    PerformanceMetrics `json:"performance"`
	Fairness       FairnessMetrics    `json:"fairness"`
	Carbon         *CarbonEstimate    `json:"carbon,omitempty"`
	RiskAssessment RiskAssessment     `json:"risk_assessment"`
	Governance     Governance         `json:"governance"`
	Compliance     []ComplianceReport `json:"compliance"`
	Artifacts      Artifacts          `json:"artifacts,omitempty"`
}

// AnalysisInput defines analyzer input.
type AnalysisInput struct {
	Ref      ModelRef
	Metadata ModelMetadata
	EvalFile string
}

// AnalysisResult contains analyzer outputs.
type AnalysisResult struct {
	Performance *PerformanceMetrics
	Fairness    *FairnessMetrics
	BiasNotes   []string
	Carbon      *CarbonEstimate
	RiskNotes   []string
	Raw         map[string]any
}

// CheckOptions configures compliance checks.
type CheckOptions struct {
	Strict bool
}

// GenerateOptions configures pipeline generation.
type GenerateOptions struct {
	Ref                  ModelRef
	EvalFile             string
	Template             string
	TemplateFile         string
	Formats              []string
	OutDir               string
	Language             string
	ComplianceFrameworks []string
}
