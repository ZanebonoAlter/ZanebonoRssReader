package topicgraph

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/aisettings"
	"my-robot-backend/internal/platform/database"
)

var errTopicAIUnavailable = errors.New("topic AI unavailable")

type topicAIConfig struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
}

type topicAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

var extractTopicsWithAI = func(input ExtractionInput) ([]TopicTag, error) {
	configMap, _, err := aisettings.LoadSummaryConfig()
	if err != nil {
		return nil, err
	}

	config := topicAIConfig{}
	if len(configMap) > 0 {
		data, marshalErr := json.Marshal(configMap)
		if marshalErr != nil {
			return nil, marshalErr
		}
		if unmarshalErr := json.Unmarshal(data, &config); unmarshalErr != nil {
			return nil, unmarshalErr
		}
	}

	if strings.TrimSpace(config.BaseURL) == "" || strings.TrimSpace(config.APIKey) == "" || strings.TrimSpace(config.Model) == "" {
		return nil, errTopicAIUnavailable
	}

	body, err := json.Marshal(map[string]any{
		"model": config.Model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "Extract 3-8 concise topic or entity tags from the provided AI summary. Return JSON only as an array. Each item must contain label, kind (topic or entity), and score between 0 and 1.",
			},
			{
				"role":    "user",
				"content": fmt.Sprintf("Title: %s\nFeed: %s\nCategory: %s\nSummary:\n%s", input.Title, input.FeedName, input.CategoryName, input.Summary),
			},
		},
		"temperature": 0.1,
		"max_tokens":  400,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(config.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	resp, err := (&http.Client{Timeout: 45 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("topic AI request failed: %s", string(responseBody))
	}

	var parsed topicAIResponse
	if err := json.Unmarshal(responseBody, &parsed); err != nil {
		return nil, err
	}
	if parsed.Error != nil {
		return nil, errors.New(parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return nil, errors.New("topic AI returned no choices")
	}

	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var rawTags []struct {
		Label string  `json:"label"`
		Kind  string  `json:"kind"`
		Score float64 `json:"score"`
	}
	if err := json.Unmarshal([]byte(content), &rawTags); err != nil {
		return nil, err
	}

	result := make([]TopicTag, 0, len(rawTags))
	for _, item := range rawTags {
		label := strings.TrimSpace(item.Label)
		if label == "" {
			continue
		}
		kind := strings.ToLower(strings.TrimSpace(item.Kind))
		if kind != "entity" {
			kind = "topic"
		}
		if item.Score <= 0 {
			item.Score = 0.7
		}
		result = append(result, TopicTag{Label: label, Slug: slugify(label), Kind: kind, Score: item.Score})
	}

	return dedupeTopics(result), nil
}

func TagSummary(summary *models.AISummary) error {
	if summary == nil || summary.ID == 0 {
		return nil
	}

	input := ExtractionInput{
		Title:        summary.Title,
		Summary:      summary.Summary,
		FeedName:     feedLabel(*summary),
		CategoryName: categoryLabel(*summary),
	}

	topics, err := extractTopicsWithAI(input)
	source := "llm"
	if err != nil {
		topics = ExtractTopics(input)
		source = "heuristic"
	}
	if len(topics) == 0 {
		return nil
	}

	if err := database.DB.Where("summary_id = ?", summary.ID).Delete(&models.AISummaryTopic{}).Error; err != nil {
		return err
	}

	for _, topic := range dedupeTopics(topics) {
		tag := models.TopicTag{}
		err := database.DB.Where("slug = ?", topic.Slug).First(&tag).Error
		if err != nil {
			tag = models.TopicTag{Slug: topic.Slug, Label: topic.Label, Kind: topic.Kind, Source: source}
			if createErr := database.DB.Create(&tag).Error; createErr != nil {
				return createErr
			}
		} else {
			tag.Label = topic.Label
			tag.Kind = topic.Kind
			tag.Source = source
			if saveErr := database.DB.Save(&tag).Error; saveErr != nil {
				return saveErr
			}
		}

		link := models.AISummaryTopic{SummaryID: summary.ID, TopicTagID: tag.ID, Score: topic.Score, Source: source}
		if err := database.DB.Create(&link).Error; err != nil {
			return err
		}
	}

	return nil
}

func dedupeTopics(items []TopicTag) []TopicTag {
	best := make(map[string]TopicTag)
	for _, item := range items {
		if item.Slug == "" {
			continue
		}
		current, exists := best[item.Slug]
		if !exists || current.Score < item.Score {
			best[item.Slug] = item
		}
	}

	result := make([]TopicTag, 0, len(best))
	for _, item := range best {
		result = append(result, item)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Score == result[j].Score {
			return result[i].Label < result[j].Label
		}
		return result[i].Score > result[j].Score
	})
	return result
}
