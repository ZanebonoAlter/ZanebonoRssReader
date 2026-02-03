package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type AIService struct {
	BaseURL string
	APIKey  string
	Model   string
	client  *http.Client
}

type AISummaryRequest struct {
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	Language string `json:"language"` // zh or en
}

type AISummaryResponse struct {
	OneSentence string   `json:"one_sentence"`
	KeyPoints   []string `json:"key_points"`
	Takeaways   []string `json:"takeaways"`
	Tags        []string `json:"tags"`
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func NewAIService(baseURL, apiKey, model string) *AIService {
	return &AIService{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (s *AIService) SummarizeArticle(title, content, language string) (*AISummaryResponse, error) {
	systemPrompt := s.getSystemPrompt(language)
	userContent := s.prepareArticleContent(title, content)

	req := openAIRequest{
		Model: s.Model,
		Messages: []openAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	resp, err := s.callOpenAI(req)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("AI API error: %s", resp.Error.Message)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from AI")
	}

	summaryText := resp.Choices[0].Message.Content
	return s.parseSummaryResponse(summaryText), nil
}

func (s *AIService) getSystemPrompt(language string) string {
	if language == "zh" {
		return `你是一个专业的文章分析助手。请对给定的文章进行智能总结，回复格式如下：

## 一句话总结
用一句话概括文章的核心内容。

## 核心观点
- 观点1
- 观点2
- 观点3

## 关键要点
1. 要点一
2. 要点二
3. 要点三

## 标签
#标签1 #标签2 #标签3

请确保总结简洁明了，突出重点。`
	}

	return `You are a professional article analysis assistant. Please provide an intelligent summary of the given article in the following format:

## One-Sentence Summary
A single sentence summarizing the core content of the article.

## Key Points
- Point 1
- Point 2
- Point 3

## Main Takeaways
1. Takeaway 1
2. Takeaway 2
3. Takeaway 3

## Tags
#tag1 #tag2 #tag3

Please ensure the summary is concise and highlights the key points.`
}

func (s *AIService) prepareArticleContent(title, content string) string {
	maxContentLength := 8000
	if len(content) > maxContentLength {
		content = content[:maxContentLength] + "..."
	}

	return fmt.Sprintf("标题：%s\n\n内容：%s", title, content)
}

func (s *AIService) parseSummaryResponse(responseText string) *AISummaryResponse {
	summary := &AISummaryResponse{
		KeyPoints: make([]string, 0),
		Takeaways: make([]string, 0),
		Tags:      make([]string, 0),
	}

	lines := splitLines(responseText)
	currentSection := ""

	for _, line := range lines {
		trimmed := trimSpace(line)
		if trimmed == "" {
			continue
		}

		if contains(trimmed, "一句话总结") || contains(trimmed, "One-Sentence Summary") {
			currentSection = "one_sentence"
			continue
		} else if contains(trimmed, "核心观点") || contains(trimmed, "Key Points") {
			currentSection = "key_points"
			continue
		} else if contains(trimmed, "关键要点") || contains(trimmed, "Main Takeaways") {
			currentSection = "takeaways"
			continue
		} else if contains(trimmed, "标签") || contains(trimmed, "Tags") {
			currentSection = "tags"
			continue
		}

		switch currentSection {
		case "one_sentence":
			summary.OneSentence = trimPrefix(trimmed, "•-*")
		case "key_points":
			point := trimPrefix(trimmed, "•-*")
			if point != "" {
				summary.KeyPoints = append(summary.KeyPoints, point)
			}
		case "takeaways":
			takeaway := trimPrefix(trimmed, "•-*123456789.).")
			if takeaway != "" {
				summary.Takeaways = append(summary.Takeaways, takeaway)
			}
		case "tags":
			tags := extractTags(trimmed)
			summary.Tags = append(summary.Tags, tags...)
		}
	}

	if summary.OneSentence == "" && responseText != "" {
		if len(responseText) > 200 {
			summary.OneSentence = responseText[:200]
		} else {
			summary.OneSentence = responseText
		}
	}

	return summary
}

func (s *AIService) callOpenAI(req openAIRequest) (*openAIResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", s.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.APIKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var openAIResp openAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return nil, err
	}

	return &openAIResp, nil
}

func (s *AIService) TestConnection() error {
	req := openAIRequest{
		Model: s.Model,
		Messages: []openAIMessage{
			{Role: "user", Content: "Hi"},
		},
		MaxTokens: 10,
	}

	resp, err := s.callOpenAI(req)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("API error: %s", resp.Error.Message)
	}

	return nil
}

func splitLines(s string) []string {
	lines := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func trimPrefix(s, chars string) string {
	chars = "-•*"
	start := 0
	for start < len(s) {
		found := false
		for _, c := range chars {
			if byte(c) == s[start] {
				start++
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return trimSpace(s[start:])
}

func extractTags(line string) []string {
	tags := make([]string, 0)
	start := 0
	for i := 0; i < len(line); i++ {
		if line[i] == '#' || i == len(line)-1 {
			if i > start {
				tag := trimSpace(line[start:i])
				if tag != "" {
					tags = append(tags, tag)
				}
			}
			start = i + 1
		}
	}
	return tags
}
