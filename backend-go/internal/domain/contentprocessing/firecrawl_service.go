package contentprocessing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type FirecrawlConfig struct {
	APIUrl           string `json:"api_url"`
	APIKey           string `json:"api_key"`
	Enabled          bool   `json:"enabled"`
	Mode             string `json:"mode"`
	Timeout          int    `json:"timeout"`
	MaxContentLength int    `json:"max_content_length"`
}

type FirecrawlService struct {
	config *FirecrawlConfig
	client *http.Client
}

type ScrapeResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Content    string `json:"content"`
		HTML       string `json:"html"`
		Markdown   string `json:"markdown"`
		Screenshot string `json:"screenshot,omitempty"`
		Metadata   struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Language    string `json:"language"`
			SourceURL   string `json:"sourceURL"`
		} `json:"metadata"`
	} `json:"data"`
	Error string `json:"error,omitempty"`
}

func NewFirecrawlService(config *FirecrawlConfig) *FirecrawlService {
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 60
	}

	return &FirecrawlService{
		config: config,
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}
}

func (s *FirecrawlService) ScrapePage(url string) (*ScrapeResponse, error) {
	requestBody := map[string]interface{}{
		"url": url,
	}

	if s.config.Mode == "scrape" {
		requestBody["formats"] = []string{"markdown", "html"}
	}

	if s.config.Timeout > 0 {
		requestBody["timeout"] = s.config.Timeout * 1000
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := buildFirecrawlEndpoint(s.config.APIUrl, "/v1/scrape")
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.config.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Firecrawl API error: %s", string(respBody))
	}

	var result ScrapeResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("Firecrawl scrape failed: %s", result.Error)
	}

	maxLength := s.config.MaxContentLength
	if maxLength <= 0 {
		maxLength = 50000
	}

	if len(result.Data.Markdown) > maxLength {
		result.Data.Markdown = result.Data.Markdown[:maxLength]
	}

	return &result, nil
}

func buildFirecrawlEndpoint(baseURL, path string) string {
	trimmedBaseURL := strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(trimmedBaseURL, "/v1") && strings.HasPrefix(path, "/v1/") {
		path = strings.TrimPrefix(path, "/v1")
	}

	return trimmedBaseURL + path
}

func (s *FirecrawlService) ShouldUseFirecrawl(globalEnabled, feedEnabled bool, articleContent string) bool {
	if !s.config.Enabled || !globalEnabled {
		return false
	}

	if !feedEnabled {
		return false
	}

	if len(articleContent) > 2000 {
		return false
	}

	return true
}
