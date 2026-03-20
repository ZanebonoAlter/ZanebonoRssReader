package airouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"my-robot-backend/internal/domain/models"
)

// EmbeddingRequest represents a request to generate embeddings
type EmbeddingRequest struct {
	Input          []string `json:"input"`
	Model          string   `json:"model"`
	EncodingFormat string   `json:"encoding_format,omitempty"` // optional: "float" or "base64"
	Dimensions     int      `json:"dimensions,omitempty"`      // optional: reduce dimensions
}

// EmbeddingResult represents the result of an embedding request
type EmbeddingResult struct {
	Embeddings [][]float64 `json:"embeddings"`
	Model      string      `json:"model"`
	Dimensions int         `json:"dimensions"`
	Provider   string      `json:"provider"`
}

// EmbeddingClient generates embeddings using AI providers
type EmbeddingClient struct{}

// NewEmbeddingClient creates a new embedding client
func NewEmbeddingClient() *EmbeddingClient {
	return &EmbeddingClient{}
}

// Embed generates embeddings for the given texts
func (c *EmbeddingClient) Embed(ctx context.Context, provider models.AIProvider, texts []string, model string) (*EmbeddingResult, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts to embed")
	}

	if model == "" {
		model = "text-embedding-ada-002"
	}

	body := EmbeddingRequest{
		Input: texts,
		Model: model,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	endpoint := strings.TrimRight(provider.BaseURL, "/") + "/embeddings"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if provider.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+provider.APIKey)
	}

	timeout := time.Duration(provider.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	resp, err := (&http.Client{Timeout: timeout}).Do(httpReq)
	if err != nil {
		return nil, &ProviderError{Message: err.Error(), Code: "network_error", Retryable: true}
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &ProviderError{Message: err.Error(), Code: "read_error", Retryable: true}
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error *struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}
		_ = json.Unmarshal(responseBody, &errResp)
		msg := string(responseBody)
		if errResp.Error != nil && errResp.Error.Message != "" {
			msg = errResp.Error.Message
		}
		retryable := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
		return nil, &ProviderError{Message: msg, Code: fmt.Sprintf("http_%d", resp.StatusCode), Retryable: retryable}
	}

	var parsed struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
		Model string `json:"model"`
	}
	if err := json.Unmarshal(responseBody, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse embedding response: %w", err)
	}

	if len(parsed.Data) == 0 {
		return nil, &ProviderError{Message: "no embeddings in response", Code: "no_response", Retryable: false}
	}

	// Sort by index and extract embeddings
	embeddings := make([][]float64, len(parsed.Data))
	for _, item := range parsed.Data {
		if item.Index < 0 || item.Index >= len(embeddings) {
			continue
		}
		embeddings[item.Index] = item.Embedding
	}

	dimensions := 0
	if len(embeddings) > 0 && len(embeddings[0]) > 0 {
		dimensions = len(embeddings[0])
	}

	return &EmbeddingResult{
		Embeddings: embeddings,
		Model:      parsed.Model,
		Dimensions: dimensions,
		Provider:   provider.Name,
	}, nil
}

// CosineSimilarity calculates the cosine similarity between two embedding vectors
func CosineSimilarity(a, b []float64) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vector dimensions don't match: %d vs %d", len(a), len(b))
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0, nil
	}

	return dotProduct / (sqrt(normA) * sqrt(normB)), nil
}

// Borrowed from math package for efficiency
func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	// Newton's method for sqrt
	z := x
	for i := 0; i < 100; i++ {
		z = z - (z*z-x)/(2*z)
		if z*z-x < 1e-10 && -(z*z-x) < 1e-10 {
			break
		}
	}
	return z
}
