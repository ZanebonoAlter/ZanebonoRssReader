package topicextraction

import (
	"testing"

	"github.com/stretchr/testify/require"

	"my-robot-backend/internal/domain/topictypes"
)

func TestExtractTopicsFindsCanonicalAITopicsAndEntities(t *testing.T) {
	result := ExtractTopics(topictypes.ExtractionInput{
		Title:        "OpenAI pushes GPT-5 agent workflow",
		Summary:      "OpenAI is shipping a new AI agent workflow around GPT-5 with multimodal planning and coding automation.",
		FeedName:     "Latent Space",
		CategoryName: "AI",
	})

	require.GreaterOrEqual(t, len(result), 3)
	require.Contains(t, topicLabels(result), "OpenAI")
	require.Contains(t, topicLabels(result), "AI Agent")
	require.Contains(t, topicLabels(result), "GPT-5")
	require.Contains(t, topicSlugs(result), "openai")
	require.Contains(t, topicSlugs(result), "ai-agent")
	require.Contains(t, topicSlugs(result), "gpt-5")
}

func TestExtractTopicsDeduplicatesAliases(t *testing.T) {
	result := ExtractTopics(topictypes.ExtractionInput{
		Title:        "OpenAI API update",
		Summary:      "OpenAI says the Open AI API now supports agent memory. OPENAI tooling remains the focus.",
		FeedName:     "OpenAI Blog",
		CategoryName: "AI",
	})

	labels := topicLabels(result)
	require.Equal(t, 1, countMatches(labels, "OpenAI"))
	openAI := findTopic(result, "OpenAI")
	require.NotNil(t, openAI)
	require.Greater(t, openAI.Score, 0.0)
}

func TestExtractTopicsFallsBackToFeedAndCategoryWhenTextIsSparse(t *testing.T) {
	result := ExtractTopics(topictypes.ExtractionInput{
		Title:        "Daily Brief",
		Summary:      "Short update.",
		FeedName:     "NVIDIA Research",
		CategoryName: "Infra",
	})

	require.Contains(t, topicLabels(result), "NVIDIA")
	require.Contains(t, topicLabels(result), "Infra")
}

func TestParseExtractedTagsAcceptsWrappedTagsObject(t *testing.T) {
	parsed, err := parseExtractedTags(`{"tags":[{"label":"OpenAI","category":"keyword","confidence":0.9,"aliases":["Open AI"]}]}`)

	require.NoError(t, err)
	require.Len(t, parsed, 1)
	require.Equal(t, "OpenAI", parsed[0].Label)
	require.Equal(t, "keyword", parsed[0].Category)
	require.Equal(t, 0.9, parsed[0].Confidence)
	require.Equal(t, []string{"Open AI"}, parsed[0].Aliases)
}

func TestParseExtractedTagsAcceptsSurroundingText(t *testing.T) {
	parsed, err := parseExtractedTags("Here are the extracted tags:\n```json\n[{\"label\":\"AI Agent\",\"category\":\"keyword\",\"confidence\":0.8}]\n```\nThese are the best matches.")

	require.NoError(t, err)
	require.Len(t, parsed, 1)
	require.Equal(t, "AI Agent", parsed[0].Label)
	require.Equal(t, "keyword", parsed[0].Category)
	require.Equal(t, 0.8, parsed[0].Confidence)
}

func topicLabels(items []topictypes.TopicTag) []string {
	labels := make([]string, 0, len(items))
	for _, item := range items {
		labels = append(labels, item.Label)
	}
	return labels
}

func topicSlugs(items []topictypes.TopicTag) []string {
	slugs := make([]string, 0, len(items))
	for _, item := range items {
		slugs = append(slugs, item.Slug)
	}
	return slugs
}

func countMatches(items []string, needle string) int {
	count := 0
	for _, item := range items {
		if item == needle {
			count++
		}
	}
	return count
}

func findTopic(items []topictypes.TopicTag, label string) *topictypes.TopicTag {
	for _, item := range items {
		if item.Label == label {
			return &item
		}
	}
	return nil
}
