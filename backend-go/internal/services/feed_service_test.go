package services

import (
	"testing"

	"my-robot-backend/internal/models"
)

func TestBuildArticleFromEntryTracksOnlyRunnableStates(t *testing.T) {
	service := NewFeedService()
	entry := ParsedEntry{
		Title:       "Fresh News",
		Description: "desc",
		Content:     "content",
		Link:        "https://example.com/article",
		Author:      "bot",
	}

	fullPipelineFeed := models.Feed{ID: 1, FirecrawlEnabled: true, ContentCompletionEnabled: true}
	fullPipelineArticle := service.buildArticleFromEntry(fullPipelineFeed, entry)
	if fullPipelineArticle.FirecrawlStatus != "pending" {
		t.Fatalf("firecrawl status = %q, want pending", fullPipelineArticle.FirecrawlStatus)
	}
	if fullPipelineArticle.ContentStatus != "incomplete" {
		t.Fatalf("content status = %q, want incomplete", fullPipelineArticle.ContentStatus)
	}

	manualOnlyFeed := models.Feed{ID: 2, FirecrawlEnabled: false, ContentCompletionEnabled: true}
	manualOnlyArticle := service.buildArticleFromEntry(manualOnlyFeed, entry)
	if manualOnlyArticle.FirecrawlStatus != "" {
		t.Fatalf("firecrawl status = %q, want empty", manualOnlyArticle.FirecrawlStatus)
	}
	if manualOnlyArticle.ContentStatus != "complete" {
		t.Fatalf("content status = %q, want complete", manualOnlyArticle.ContentStatus)
	}

	disabledFeed := models.Feed{ID: 3, FirecrawlEnabled: false, ContentCompletionEnabled: false}
	disabledArticle := service.buildArticleFromEntry(disabledFeed, entry)
	if disabledArticle.FirecrawlStatus != "" {
		t.Fatalf("disabled firecrawl status = %q, want empty", disabledArticle.FirecrawlStatus)
	}
	if disabledArticle.ContentStatus != "complete" {
		t.Fatalf("disabled content status = %q, want complete", disabledArticle.ContentStatus)
	}
}
