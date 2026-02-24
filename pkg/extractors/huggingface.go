package extractors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/yapay/ai-model-card-generator/pkg/core"
)

const defaultHFBaseURL = "https://huggingface.co"

// HuggingFaceExtractor retrieves model metadata from the Hugging Face Hub API.
type HuggingFaceExtractor struct {
	Client  *http.Client
	BaseURL string
}

// NewHuggingFaceExtractor creates a default extractor instance.
func NewHuggingFaceExtractor(baseURL string) *HuggingFaceExtractor {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultHFBaseURL
	}
	return &HuggingFaceExtractor{
		Client:  &http.Client{Timeout: 20 * time.Second},
		BaseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (e *HuggingFaceExtractor) Name() string {
	return "huggingface"
}

type hfModelResponse struct {
	ID       string         `json:"id"`
	ModelID  string         `json:"modelId"`
	Author   string         `json:"author"`
	Tags     []string       `json:"tags"`
	CardData map[string]any `json:"cardData"`
}

func (e *HuggingFaceExtractor) Extract(ctx context.Context, ref core.ModelRef) (core.ModelMetadata, error) {
	if strings.TrimSpace(ref.ID) == "" {
		return core.ModelMetadata{}, fmt.Errorf("model id is required")
	}

	if e.Client == nil {
		e.Client = &http.Client{Timeout: 20 * time.Second}
	}
	if strings.TrimSpace(e.BaseURL) == "" {
		e.BaseURL = defaultHFBaseURL
	}

	url := fmt.Sprintf("%s%s", e.BaseURL, path.Join("/api/models", ref.ID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return core.ModelMetadata{}, fmt.Errorf("create request: %w", err)
	}

	resp, err := e.Client.Do(req)
	if err != nil {
		return core.ModelMetadata{}, fmt.Errorf("request model metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return core.ModelMetadata{}, fmt.Errorf("huggingface API status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload hfModelResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return core.ModelMetadata{}, fmt.Errorf("decode metadata response: %w", err)
	}

	metadata := core.ModelMetadata{
		Name:    firstNonEmpty(payload.ModelID, payload.ID, ref.ID),
		Owner:   payload.Author,
		Tags:    payload.Tags,
		Metrics: map[string]float64{},
	}

	if payload.CardData != nil {
		if v, ok := payload.CardData["license"].(string); ok {
			metadata.License = v
		}
		if v, ok := payload.CardData["model_summary"].(string); ok {
			metadata.IntendedUse = v
		}
		if v, ok := payload.CardData["limitations"].(string); ok {
			metadata.Limitations = v
		}
		if v, ok := payload.CardData["datasets"].(string); ok {
			metadata.TrainingData = v
		}
		if v, ok := payload.CardData["eval_results"].(string); ok {
			metadata.EvalData = v
		}
	}

	if metadata.License == "" {
		for _, tag := range payload.Tags {
			if strings.HasPrefix(tag, "license:") {
				metadata.License = strings.TrimPrefix(tag, "license:")
				break
			}
		}
	}

	if metadata.Owner == "" {
		if idx := strings.Index(payload.ModelID, "/"); idx > 0 {
			metadata.Owner = payload.ModelID[:idx]
		}
	}

	return metadata, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
