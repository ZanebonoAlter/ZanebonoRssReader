package contentprocessing

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"my-robot-backend/internal/app/runtimeinfo"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/airouter"
	"my-robot-backend/internal/platform/database"
)

var completionService *ContentCompletionService

func InitContentCompletionHandler(crawlBaseURL string) {
	completionService = NewContentCompletionService(crawlBaseURL)
	loadCompletionAISettings()
}

func loadCompletionAISettings() {
	if completionService == nil {
		return
	}

	provider, _, err := airouter.NewRouter().ResolvePrimaryProvider(airouter.CapabilityArticleCompletion)
	if err == nil && provider != nil && provider.BaseURL != "" && provider.APIKey != "" && provider.Model != "" {
		completionService.SetAICredentials(provider.BaseURL, provider.APIKey, provider.Model)
		return
	}

	var settings models.AISettings
	if err := database.DB.Where("key = ?", "summary_config").First(&settings).Error; err != nil {
		return
	}

	var config struct {
		BaseURL string `json:"base_url"`
		APIKey  string `json:"api_key"`
		Model   string `json:"model"`
	}
	if err := json.Unmarshal([]byte(settings.Value), &config); err != nil {
		return
	}

	if config.BaseURL != "" && config.APIKey != "" && config.Model != "" {
		completionService.SetAICredentials(config.BaseURL, config.APIKey, config.Model)
	}
}

func SetCompletionAICredentials(baseURL, apiKey, model string) {
	if completionService != nil {
		completionService.SetAICredentials(baseURL, apiKey, model)
	}
}

func SetCompletionCrawlAPIToken(token string) {
	if completionService != nil {
		completionService.SetCrawlAPIToken(token)
	}
}

func CompleteArticleContent(c *gin.Context) {
	id := c.Param("article_id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Article ID is required"})
		return
	}

	var articleID uint
	if _, err := fmt.Sscanf(id, "%d", &articleID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid article ID"})
		return
	}

	if completionService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Content completion service not initialized"})
		return
	}

	var req struct {
		Force bool `json:"force"`
	}
	_ = c.ShouldBindJSON(&req)

	if err := completionService.CompleteArticleWithForce(articleID, req.Force); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Content completion initiated"})
}

func CompleteFeedArticles(c *gin.Context) {
	id := c.Param("feed_id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Feed ID is required"})
		return
	}

	var feedID uint
	if _, err := fmt.Sscanf(id, "%d", &feedID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid feed ID"})
		return
	}

	if completionService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Content completion service not initialized"})
		return
	}

	var articles []models.Article
	if err := database.DB.Where("feed_id = ? AND content_status IN ?", feedID, []string{"incomplete", "failed"}).Find(&articles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	completed := 0
	failed := 0

	for _, article := range articles {
		if err := completionService.CompleteArticleWithForce(article.ID, true); err != nil {
			failed++
		} else {
			completed++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"completed": completed,
		"failed":    failed,
		"total":     len(articles),
	})
}

func GetCompletionStatus(c *gin.Context) {
	id := c.Param("article_id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Article ID is required"})
		return
	}

	var article models.Article
	if err := database.DB.First(&article, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Article not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"content_status":       article.ContentStatus,
			"attempts":             article.CompletionAttempts,
			"error":                article.CompletionError,
			"fetched_at":           article.ContentFetchedAt,
			"ai_content_summary":   article.AIContentSummary,
			"full_content":         article.FullContent,
			"firecrawl_content":    article.FirecrawlContent,
			"firecrawl_status":     article.FirecrawlStatus,
			"firecrawl_error":      article.FirecrawlError,
			"firecrawl_crawled_at": article.FirecrawlCrawledAt,
		},
	})
}

func GetCompletionOverview(c *gin.Context) {
	if completionService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Content completion service not initialized"})
		return
	}

	overview, err := completionService.GetOverview()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	data := gin.H{
		"pending_count":    overview.PendingCount,
		"processing_count": overview.ProcessingCount,
		"completed_count":  overview.CompletedCount,
		"failed_count":     overview.FailedCount,
		"blocked_count":    overview.BlockedCount,
		"total_count":      overview.TotalCount,
		"ai_configured":    overview.AIConfigured,
		"blocked_reasons": gin.H{
			"waiting_for_firecrawl_count":     overview.BlockedReasons.WaitingForFirecrawlCount,
			"feed_disabled_count":             overview.BlockedReasons.FeedDisabledCount,
			"ai_unconfigured_count":           overview.BlockedReasons.AIUnconfiguredCount,
			"ready_but_missing_content_count": overview.BlockedReasons.ReadyButMissingContentCount,
		},
	}

	if scheduler, ok := runtimeinfo.AISummarySchedulerInterface.(interface{ GetStatus() map[string]interface{} }); ok {
		status := scheduler.GetStatus()
		for _, key := range []string{"is_executing", "current_article", "last_processed", "next_run", "last_error", "database_state", "overview"} {
			if value, exists := status[key]; exists {
				data[key] = value
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func GetContentCompletionService() *ContentCompletionService {
	return completionService
}
