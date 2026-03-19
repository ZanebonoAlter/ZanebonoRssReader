package feeds

import (
	"testing"

	"my-robot-backend/internal/domain/models"
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

	fullPipelineFeed := models.Feed{ID: 1, FirecrawlEnabled: true, ArticleSummaryEnabled: true}
	fullPipelineArticle := service.buildArticleFromEntry(fullPipelineFeed, entry)
	if fullPipelineArticle.FirecrawlStatus != "pending" {
		t.Fatalf("firecrawl status = %q, want pending", fullPipelineArticle.FirecrawlStatus)
	}
	if fullPipelineArticle.SummaryStatus != "incomplete" {
		t.Fatalf("summary status = %q, want incomplete", fullPipelineArticle.SummaryStatus)
	}

	manualOnlyFeed := models.Feed{ID: 2, FirecrawlEnabled: false, ArticleSummaryEnabled: true}
	manualOnlyArticle := service.buildArticleFromEntry(manualOnlyFeed, entry)
	if manualOnlyArticle.FirecrawlStatus != "" {
		t.Fatalf("firecrawl status = %q, want empty", manualOnlyArticle.FirecrawlStatus)
	}
	if manualOnlyArticle.SummaryStatus != "complete" {
		t.Fatalf("summary status = %q, want complete", manualOnlyArticle.SummaryStatus)
	}

	disabledFeed := models.Feed{ID: 3, FirecrawlEnabled: false, ArticleSummaryEnabled: false}
	disabledArticle := service.buildArticleFromEntry(disabledFeed, entry)
	if disabledArticle.FirecrawlStatus != "" {
		t.Fatalf("disabled firecrawl status = %q, want empty", disabledArticle.FirecrawlStatus)
	}
	if disabledArticle.SummaryStatus != "complete" {
		t.Fatalf("disabled summary status = %q, want complete", disabledArticle.SummaryStatus)
	}
}
