package extractors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/yapay/ai-model-card-generator/pkg/core"
)

// ParseMLflowModelID validates and extracts run_id from `run:<run_id>` model references.
func ParseMLflowModelID(model string) (string, error) {
	candidate := strings.TrimSpace(model)
	if !strings.HasPrefix(strings.ToLower(candidate), "run:") {
		return "", fmt.Errorf("expected format run:<run_id>")
	}
	idx := strings.Index(candidate, ":")
	runID := strings.TrimSpace(candidate[idx+1:])
	if runID == "" {
		return "", fmt.Errorf("expected format run:<run_id>")
	}
	return runID, nil
}

// MLflowExtractor retrieves run metadata from MLflow Tracking API.
type MLflowExtractor struct {
	Client      *http.Client
	TrackingURI string
	Token       string
	Username    string
	Password    string
	FixturePath string
}

// NewMLflowExtractor creates an extractor with optional auth and fixture overrides.
func NewMLflowExtractor(trackingURI, token, username, password, fixturePath string) *MLflowExtractor {
	return &MLflowExtractor{
		Client:      &http.Client{Timeout: 20 * time.Second},
		TrackingURI: strings.TrimRight(strings.TrimSpace(trackingURI), "/"),
		Token:       strings.TrimSpace(token),
		Username:    strings.TrimSpace(username),
		Password:    strings.TrimSpace(password),
		FixturePath: strings.TrimSpace(fixturePath),
	}
}

func (e *MLflowExtractor) Name() string {
	return "mlflow"
}

type mlflowRunGetResponse struct {
	Run *struct {
		Info struct {
			RunID string `json:"run_id"`
		} `json:"info"`
		Data struct {
			Metrics []struct {
				Key   string  `json:"key"`
				Value float64 `json:"value"`
			} `json:"metrics"`
			Params []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			} `json:"params"`
			Tags []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			} `json:"tags"`
		} `json:"data"`
	} `json:"run"`
}

func (e *MLflowExtractor) Extract(ctx context.Context, ref core.ModelRef) (core.ModelMetadata, error) {
	runID, err := ParseMLflowModelID(ref.ID)
	if err != nil {
		return core.ModelMetadata{}, err
	}

	fixturePath := firstNonEmptyMLflow(strings.TrimSpace(e.FixturePath), strings.TrimSpace(os.Getenv("MCG_MLFLOW_FIXTURE")))
	if fixturePath != "" {
		payload, err := loadMLflowFixture(fixturePath)
		if err != nil {
			return core.ModelMetadata{}, fmt.Errorf("load mlflow fixture: %w", err)
		}
		return mapMLflowPayloadToMetadata(runID, payload), nil
	}

	if e.Client == nil {
		e.Client = &http.Client{Timeout: 20 * time.Second}
	}
	trackingURI := firstNonEmptyMLflow(strings.TrimSpace(e.TrackingURI), strings.TrimSpace(os.Getenv("MLFLOW_TRACKING_URI")))
	if trackingURI == "" {
		return core.ModelMetadata{}, fmt.Errorf("MLFLOW_TRACKING_URI is required for live mlflow extraction")
	}
	trackingURI = strings.TrimRight(trackingURI, "/")

	token := firstNonEmptyMLflow(strings.TrimSpace(e.Token), strings.TrimSpace(os.Getenv("MLFLOW_TRACKING_TOKEN")))
	username := firstNonEmptyMLflow(strings.TrimSpace(e.Username), strings.TrimSpace(os.Getenv("MLFLOW_TRACKING_USERNAME")))
	password := firstNonEmptyMLflow(strings.TrimSpace(e.Password), strings.TrimSpace(os.Getenv("MLFLOW_TRACKING_PASSWORD")))

	endpoint := fmt.Sprintf("%s/api/2.0/mlflow/runs/get?run_id=%s", trackingURI, url.QueryEscape(runID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return core.ModelMetadata{}, fmt.Errorf("create mlflow request: %w", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else if username != "" || password != "" {
		req.SetBasicAuth(username, password)
	}

	resp, err := e.Client.Do(req)
	if err != nil {
		return core.ModelMetadata{}, fmt.Errorf("request mlflow metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return core.ModelMetadata{}, fmt.Errorf("mlflow API status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload mlflowRunGetResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return core.ModelMetadata{}, fmt.Errorf("decode mlflow response: %w", err)
	}
	if payload.Run == nil {
		return core.ModelMetadata{}, fmt.Errorf("mlflow run not found: %s", runID)
	}

	return mapMLflowPayloadToMetadata(runID, payload), nil
}

func loadMLflowFixture(path string) (mlflowRunGetResponse, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return mlflowRunGetResponse{}, err
	}

	var payload mlflowRunGetResponse
	if err := json.Unmarshal(data, &payload); err != nil {
		return mlflowRunGetResponse{}, err
	}
	if payload.Run == nil {
		return mlflowRunGetResponse{}, fmt.Errorf("fixture missing run payload")
	}
	return payload, nil
}

func mapMLflowPayloadToMetadata(fallbackRunID string, payload mlflowRunGetResponse) core.ModelMetadata {
	runID := fallbackRunID
	if payload.Run != nil && strings.TrimSpace(payload.Run.Info.RunID) != "" {
		runID = strings.TrimSpace(payload.Run.Info.RunID)
	}

	tagMap := map[string]string{}
	tags := []string{}
	metrics := map[string]float64{}

	if payload.Run != nil {
		for _, metric := range payload.Run.Data.Metrics {
			if strings.TrimSpace(metric.Key) == "" {
				continue
			}
			metrics[metric.Key] = metric.Value
		}

		for _, tag := range payload.Run.Data.Tags {
			if strings.TrimSpace(tag.Key) == "" {
				continue
			}
			tagMap[tag.Key] = tag.Value
			tags = append(tags, tag.Key+":"+tag.Value)
		}

		for _, param := range payload.Run.Data.Params {
			if strings.TrimSpace(param.Key) == "" {
				continue
			}
			if value, err := strconv.ParseFloat(strings.TrimSpace(param.Value), 64); err == nil {
				if _, exists := metrics[param.Key]; !exists {
					metrics[param.Key] = value
				}
			}
		}
	}

	metadata := core.ModelMetadata{
		Name:         firstNonEmptyMLflow(tagMap["mlflow.runName"], runID),
		Owner:        firstNonEmptyMLflow(tagMap["mlflow.user"]),
		Tags:         tags,
		IntendedUse:  firstNonEmptyMLflow(tagMap["model_card.intended_use"], tagMap["intended_use"]),
		Limitations:  firstNonEmptyMLflow(tagMap["model_card.limitations"], tagMap["limitations"]),
		License:      firstNonEmptyMLflow(tagMap["model_card.license"], tagMap["license"]),
		TrainingData: firstNonEmptyMLflow(tagMap["model_card.training_data"], tagMap["training_data"]),
		EvalData:     firstNonEmptyMLflow(tagMap["model_card.eval_data"], tagMap["eval_data"]),
		Metrics:      metrics,
	}

	return metadata
}

func firstNonEmptyMLflow(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
