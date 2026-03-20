package opennotebook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	transformationsPath       = "/api/transformations"
	executeTransformationPath = "/api/transformations/execute"
)

type Client struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

type SummarizeDigestRequest struct {
	Title          string `json:"title"`
	Content        string `json:"content"`
	TargetNotebook string `json:"target_notebook,omitempty"`
	PromptMode     string `json:"prompt_mode,omitempty"`
}

type SummarizeDigestResponse struct {
	SummaryMarkdown string `json:"summary_markdown"`
	RemoteID        string `json:"remote_id,omitempty"`
	RemoteURL       string `json:"remote_url,omitempty"`
}

type summarizeDigestPayload struct {
	Model string `json:"model"`
	SummarizeDigestRequest
}

type transformationSummary struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ApplyDefault bool   `json:"apply_default"`
}

type executeTransformationRequest struct {
	TransformationID string `json:"transformation_id"`
	InputText        string `json:"input_text"`
	ModelID          string `json:"model_id"`
}

type executeTransformationResponse struct {
	Output           string `json:"output"`
	TransformationID string `json:"transformation_id"`
	ModelID          string `json:"model_id"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func normalizeBaseURL(baseURL string) string {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return ""
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "http://" + trimmed
	}
	return strings.TrimRight(trimmed, "/")
}

func NewClient(baseURL, apiKey, model string) *Client {
	return &Client{
		BaseURL: normalizeBaseURL(baseURL),
		APIKey:  apiKey,
		Model:   model,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) SummarizeDigest(req SummarizeDigestRequest) (*SummarizeDigestResponse, error) {
	transformationID, err := c.resolveTransformationID(req.PromptMode)
	if err != nil {
		return nil, err
	}

	payload := executeTransformationRequest{
		TransformationID: transformationID,
		InputText:        req.Content,
		ModelID:          c.Model,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal open-notebook payload: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.BaseURL+executeTransformationPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create open-notebook request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(c.APIKey) != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("open-notebook request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read open-notebook response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		apiErr := errorResponse{}
		if err := json.Unmarshal(respBody, &apiErr); err == nil && strings.TrimSpace(apiErr.Error) != "" {
			return nil, fmt.Errorf("open-notebook request failed: %s", apiErr.Error)
		}
		return nil, fmt.Errorf("open-notebook request failed: status %d", resp.StatusCode)
	}

	result := &executeTransformationResponse{}
	if err := json.Unmarshal(respBody, result); err != nil {
		return nil, fmt.Errorf("parse open-notebook response: %w", err)
	}

	return &SummarizeDigestResponse{SummaryMarkdown: result.Output}, nil
}

func (c *Client) resolveTransformationID(promptMode string) (string, error) {
	transformations, err := c.listTransformations()
	if err != nil {
		return "", err
	}

	preferredNames := transformationNamesForPromptMode(promptMode)
	for _, name := range preferredNames {
		for _, item := range transformations {
			if strings.EqualFold(strings.TrimSpace(item.Name), name) {
				return item.ID, nil
			}
		}
	}

	for _, item := range transformations {
		if item.ApplyDefault {
			return item.ID, nil
		}
	}

	if len(transformations) > 0 {
		return transformations[0].ID, nil
	}

	return "", fmt.Errorf("no suitable transformation found")
}

func (c *Client) listTransformations() ([]transformationSummary, error) {
	httpReq, err := http.NewRequest(http.MethodGet, c.BaseURL+transformationsPath, nil)
	if err != nil {
		return nil, fmt.Errorf("create transformation list request: %w", err)
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("open-notebook request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read transformations response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("open-notebook request failed: status %d", resp.StatusCode)
	}

	items := []transformationSummary{}
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("parse transformations response: %w", err)
	}

	return items, nil
}

func transformationNamesForPromptMode(promptMode string) []string {
	switch strings.TrimSpace(promptMode) {
	case "", "digest_summary":
		return []string{"Simple Summary", "Dense Summary", "Key Insights"}
	default:
		return []string{promptMode, "Simple Summary", "Dense Summary", "Key Insights"}
	}
}
