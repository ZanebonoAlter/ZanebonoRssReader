package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"my-robot-backend/internal/models"
	"my-robot-backend/internal/services"
	"my-robot-backend/pkg/database"
)

var completionService *services.ContentCompletionService

func InitContentCompletionHandler(crawlBaseURL string) {
	completionService = services.NewContentCompletionService(crawlBaseURL)
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
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Article ID is required"})
		return
	}

	var articleID uint
	if _, err := fmt.Sscanf(id, "%d", &articleID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid article ID"})
		return
	}

	if err := completionService.CompleteArticle(articleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Content completion initiated"})
}

func CompleteFeedArticles(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Feed ID is required"})
		return
	}

	var feedID uint
	if _, err := fmt.Sscanf(id, "%d", &feedID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid feed ID"})
		return
	}

	var articles []models.Article
	if err := database.DB.Where("feed_id = ? AND content_status = ?", feedID, "incomplete").Find(&articles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	completed := 0
	failed := 0

	for _, article := range articles {
		if err := completionService.CompleteArticle(article.ID); err != nil {
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
	id := c.Param("id")
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
		"success":        true,
		"content_status": article.ContentStatus,
		"attempts":       article.CompletionAttempts,
		"error":          article.CompletionError,
		"fetched_at":     article.ContentFetchedAt,
	})
}

func GetContentCompletionService() *services.ContentCompletionService {
	return completionService
}
