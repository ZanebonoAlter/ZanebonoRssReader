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

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Capability  Capability
	Messages    []Message
	Temperature *float64
	MaxTokens   *int
	Metadata    map[string]any
}

type ChatResult struct {
	Content      string `json:"content"`
	ProviderID   uint   `json:"provider_id"`
	ProviderName string `json:"provider_name"`
	RouteName    string `json:"route_name"`
	UsedFallback bool   `json:"used_fallback"`
	AttemptCount int    `json:"attempt_count"`
}

type ProviderClient interface {
	Chat(ctx context.Context, provider models.AIProvider, req ChatRequest) (string, error)
}

type ProviderError struct {
	Message   string
	Code      string
	Retryable bool
}

func (e *ProviderError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

type openAICompatibleClient struct{}

func NewOpenAICompatibleClient() ProviderClient {
	return &openAICompatibleClient{}
}

func (c *openAICompatibleClient) Chat(ctx context.Context, provider models.AIProvider, req ChatRequest) (string, error) {
	temperature := 0.3
	if req.Temperature != nil {
		temperature = *req.Temperature
	} else if provider.Temperature != nil {
		temperature = *provider.Temperature
	}

	maxTokens := 16000
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	} else if provider.MaxTokens != nil {
		maxTokens = *provider.MaxTokens
	}

	body, err := json.Marshal(map[string]any{
		"model":       provider.Model,
		"messages":    req.Messages,
		"temperature": temperature,
		"max_tokens":  maxTokens,
	})
	if err != nil {
		return "", err
	}

	endpoint := strings.TrimRight(provider.BaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+provider.APIKey)

	timeout := time.Duration(provider.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	resp, err := (&http.Client{Timeout: timeout}).Do(httpReq)
	if err != nil {
		return "", &ProviderError{Message: err.Error(), Code: "network_error", Retryable: true}
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", &ProviderError{Message: err.Error(), Code: "read_error", Retryable: true}
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error,omitempty"`
	}
	if err := json.Unmarshal(responseBody, &parsed); err != nil {
		return "", &ProviderError{Message: fmt.Sprintf("failed to parse response: %v", err), Code: "parse_error", Retryable: resp.StatusCode >= 500}
	}

	if resp.StatusCode >= 400 {
		retryable := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
		message := string(responseBody)
		code := fmt.Sprintf("http_%d", resp.StatusCode)
		if parsed.Error != nil {
			message = parsed.Error.Message
			if parsed.Error.Code != "" {
				code = parsed.Error.Code
			}
		}
		return "", &ProviderError{Message: message, Code: code, Retryable: retryable}
	}

	if parsed.Error != nil {
		return "", &ProviderError{Message: parsed.Error.Message, Code: parsed.Error.Code, Retryable: false}
	}
	if len(parsed.Choices) == 0 {
		return "", &ProviderError{Message: "no response from AI", Code: "no_response", Retryable: true}
	}

	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}
