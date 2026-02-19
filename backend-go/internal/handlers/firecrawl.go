package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"my-robot-backend/internal/models"
	"my-robot-backend/internal/services"
	"my-robot-backend/pkg/database"
)

func CrawlArticle(c *gin.Context) {
	articleID := c.Param("id")

	var article models.Article
	if err := database.DB.First(&article, articleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Article not found",
		})
		return
	}

	var feed models.Feed
	if err := database.DB.First(&feed, article.FeedID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Feed not found",
		})
		return
	}

	if !feed.FirecrawlEnabled {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Firecrawl not enabled for this feed",
		})
		return
	}

	config, err := services.GetFirecrawlConfig()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	if !config.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Firecrawl is not enabled globally",
		})
		return
	}

	firecrawlService := services.NewFirecrawlService(config)

	article.FirecrawlStatus = "processing"
	database.DB.Save(&article)

	result, err := firecrawlService.ScrapePage(article.Link)
	if err != nil {
		article.FirecrawlStatus = "failed"
		article.FirecrawlError = err.Error()
		database.DB.Save(&article)

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	article.FirecrawlStatus = "completed"
	article.FirecrawlContent = result.Data.Markdown
	now := time.Now()
	article.FirecrawlCrawledAt = &now
	database.DB.Save(&article)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"firecrawl_content": result.Data.Markdown,
			"firecrawl_status":  "completed",
		},
	})
}

func EnableFeedFirecrawl(c *gin.Context) {
	feedID := c.Param("id")

	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	var feed models.Feed
	if err := database.DB.First(&feed, feedID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Feed not found",
		})
		return
	}

	feed.FirecrawlEnabled = req.Enabled
	database.DB.Save(&feed)

	if req.Enabled {
		database.DB.Model(&models.Article{}).
			Where("feed_id = ?", feed.ID).
			Updates(map[string]interface{}{
				"firecrawl_enabled": true,
				"firecrawl_status":  "pending",
			})
	} else {
		database.DB.Model(&models.Article{}).
			Where("feed_id = ?", feed.ID).
			Update("firecrawl_enabled", false)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"firecrawl_enabled": feed.FirecrawlEnabled,
		},
	})
}

func GetFirecrawlStatus(c *gin.Context) {
	config, err := services.GetFirecrawlConfig()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"enabled": false,
				"error":   err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled":            config.Enabled,
			"api_url":            config.APIUrl,
			"mode":               config.Mode,
			"timeout":            config.Timeout,
			"max_content_length": config.MaxContentLength,
			"api_key_configured": config.APIKey != "",
		},
	})
}
