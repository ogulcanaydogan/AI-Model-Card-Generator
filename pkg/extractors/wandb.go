package extractors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/yapay/ai-model-card-generator/pkg/core"
)

const defaultWandBBaseURL = "https://api.wandb.ai"

// WandBRunRef is a normalized run reference parsed from `<entity>/<project>/<run_id>`.
type WandBRunRef struct {
	Entity  string
	Project string
	RunID   string
}

// ParseWandBModelID parses and validates the required model id format for wandb source.
func ParseWandBModelID(modelID string) (WandBRunRef, error) {
	parts := strings.Split(strings.TrimSpace(modelID), "/")
	if len(parts) != 3 {
		return WandBRunRef{}, fmt.Errorf("expected format <entity>/<project>/<run_id>")
	}
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return WandBRunRef{}, fmt.Errorf("expected format <entity>/<project>/<run_id>")
		}
	}
	return WandBRunRef{
		Entity:  parts[0],
		Project: parts[1],
		RunID:   parts[2],
	}, nil
}

// WeightsAndBiasesExtractor retrieves run metadata through the W&B API.
type WeightsAndBiasesExtractor struct {
	Client      *http.Client
	BaseURL     string
	APIToken    string
	FixturePath string
}

// NewWeightsAndBiasesExtractor creates an extractor with optional overrides.
func NewWeightsAndBiasesExtractor(baseURL, apiToken, fixturePath string) *WeightsAndBiasesExtractor {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultWandBBaseURL
	}
	return &WeightsAndBiasesExtractor{
		Client:      &http.Client{Timeout: 20 * time.Second},
		BaseURL:     strings.TrimRight(baseURL, "/"),
		APIToken:    strings.TrimSpace(apiToken),
		FixturePath: strings.TrimSpace(fixturePath),
	}
}

func (e *WeightsAndBiasesExtractor) Name() string {
	return "wandb"
}

type wandbGraphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type wandbGraphQLError struct {
	Message string `json:"message"`
}

type wandbRun struct {
	Name           string         `json:"name"`
	DisplayName    string         `json:"displayName"`
	Tags           []string       `json:"tags"`
	Notes          string         `json:"notes"`
	SummaryMetrics map[string]any `json:"summaryMetrics"`
	Config         map[string]any `json:"config"`
}

type wandbGraphQLResponse struct {
	Data struct {
		Project *struct {
			Run *wandbRun `json:"run"`
		} `json:"project"`
	} `json:"data"`
	Errors []wandbGraphQLError `json:"errors"`
}

func (e *WeightsAndBiasesExtractor) Extract(ctx context.Context, ref core.ModelRef) (core.ModelMetadata, error) {
	runRef, err := ParseWandBModelID(ref.ID)
	if err != nil {
		return core.ModelMetadata{}, err
	}

	fixturePath := firstNonEmptyString(strings.TrimSpace(e.FixturePath), strings.TrimSpace(os.Getenv("MCG_WANDB_FIXTURE")))
	if fixturePath != "" {
		run, err := loadWandBRunFromFixture(fixturePath)
		if err != nil {
			return core.ModelMetadata{}, fmt.Errorf("load wandb fixture: %w", err)
		}
		return mapWandBRunToMetadata(runRef, run), nil
	}

	if e.Client == nil {
		e.Client = &http.Client{Timeout: 20 * time.Second}
	}
	baseURL := firstNonEmptyString(strings.TrimSpace(e.BaseURL), strings.TrimSpace(os.Getenv("WANDB_BASE_URL")), defaultWandBBaseURL)
	baseURL = strings.TrimRight(baseURL, "/")

	apiToken := firstNonEmptyString(strings.TrimSpace(e.APIToken), strings.TrimSpace(os.Getenv("WANDB_API_KEY")))
	if apiToken == "" {
		return core.ModelMetadata{}, fmt.Errorf("WANDB_API_KEY is required for live wandb extraction")
	}

	query := `
query RunQuery($entity: String!, $project: String!, $run: String!) {
  project(name: $project, entityName: $entity) {
    run(name: $run) {
      name
      displayName
      tags
      notes
      summaryMetrics
      config
    }
  }
}
`
	reqPayload := wandbGraphQLRequest{
		Query: query,
		Variables: map[string]any{
			"entity":  runRef.Entity,
			"project": runRef.Project,
			"run":     runRef.RunID,
		},
	}
	body, err := json.Marshal(reqPayload)
	if err != nil {
		return core.ModelMetadata{}, fmt.Errorf("marshal wandb request: %w", err)
	}

	url := fmt.Sprintf("%s%s", baseURL, path.Join("/", "graphql"))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return core.ModelMetadata{}, fmt.Errorf("create wandb request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiToken)

	resp, err := e.Client.Do(req)
	if err != nil {
		return core.ModelMetadata{}, fmt.Errorf("request wandb metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return core.ModelMetadata{}, fmt.Errorf("wandb API status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed wandbGraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return core.ModelMetadata{}, fmt.Errorf("decode wandb response: %w", err)
	}
	if len(parsed.Errors) > 0 {
		return core.ModelMetadata{}, fmt.Errorf("wandb API error: %s", parsed.Errors[0].Message)
	}
	if parsed.Data.Project == nil || parsed.Data.Project.Run == nil {
		return core.ModelMetadata{}, fmt.Errorf("wandb run not found: %s/%s/%s", runRef.Entity, runRef.Project, runRef.RunID)
	}

	return mapWandBRunToMetadata(runRef, *parsed.Data.Project.Run), nil
}

func loadWandBRunFromFixture(path string) (wandbRun, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return wandbRun{}, err
	}

	var fixture struct {
		Run  *wandbRun `json:"run"`
		Data struct {
			Project *struct {
				Run *wandbRun `json:"run"`
			} `json:"project"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &fixture); err == nil {
		if fixture.Run != nil {
			return *fixture.Run, nil
		}
		if fixture.Data.Project != nil && fixture.Data.Project.Run != nil {
			return *fixture.Data.Project.Run, nil
		}
	}

	var gqlResp wandbGraphQLResponse
	if err := json.Unmarshal(data, &gqlResp); err == nil {
		if gqlResp.Data.Project != nil && gqlResp.Data.Project.Run != nil {
			return *gqlResp.Data.Project.Run, nil
		}
	}

	var direct wandbRun
	if err := json.Unmarshal(data, &direct); err == nil {
		if direct.Name != "" || direct.DisplayName != "" || len(direct.Tags) > 0 || len(direct.SummaryMetrics) > 0 || len(direct.Config) > 0 {
			return direct, nil
		}
	}

	return wandbRun{}, fmt.Errorf("unsupported fixture format")
}

func mapWandBRunToMetadata(runRef WandBRunRef, run wandbRun) core.ModelMetadata {
	config := run.Config
	metadata := core.ModelMetadata{
		Name:         firstNonEmptyString(run.DisplayName, run.Name, runRef.RunID),
		Owner:        runRef.Entity,
		Tags:         run.Tags,
		IntendedUse:  firstNonEmptyString(configValueAsString(config, "intended_use"), configValueAsString(config, "intendedUse"), configValueAsString(config, "purpose"), strings.TrimSpace(run.Notes)),
		Limitations:  firstNonEmptyString(configValueAsString(config, "limitations"), configValueAsString(config, "model_limitations"), configValueAsString(config, "known_limitations")),
		License:      firstNonEmptyString(configValueAsString(config, "license"), configValueAsString(config, "model_license")),
		TrainingData: firstNonEmptyString(configValueAsString(config, "training_data"), configValueAsString(config, "dataset"), configValueAsString(config, "datasets")),
		EvalData:     firstNonEmptyString(configValueAsString(config, "eval_data"), configValueAsString(config, "evaluation_data"), configValueAsString(config, "validation_data")),
		Metrics:      numericMetrics(run.SummaryMetrics),
	}
	if metadata.Metrics == nil {
		metadata.Metrics = map[string]float64{}
	}
	return metadata
}

func configValueAsString(config map[string]any, key string) string {
	if config == nil {
		return ""
	}
	raw, ok := config[key]
	if !ok {
		return ""
	}
	return anyToString(raw)
}

func anyToString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case bool:
		return strconv.FormatBool(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case int:
		return strconv.Itoa(v)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case map[string]any:
		if nested, ok := v["value"]; ok {
			return anyToString(nested)
		}
	case map[string]string:
		if nested, ok := v["value"]; ok {
			return strings.TrimSpace(nested)
		}
	}
	return ""
}

func numericMetrics(summary map[string]any) map[string]float64 {
	if len(summary) == 0 {
		return map[string]float64{}
	}
	out := make(map[string]float64)
	for key, raw := range summary {
		if value, ok := anyToFloat(raw); ok {
			out[key] = value
			continue
		}
		if nested, ok := raw.(map[string]any); ok {
			if value, ok := anyToFloat(nested["value"]); ok {
				out[key] = value
			}
		}
	}
	return out
}

func anyToFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		n, err := v.Float64()
		return n, err == nil
	case string:
		n, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return n, err == nil
	default:
		return 0, false
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
