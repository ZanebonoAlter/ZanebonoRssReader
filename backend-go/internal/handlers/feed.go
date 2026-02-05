package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"my-robot-backend/internal/models"
	"my-robot-backend/internal/services"
	"my-robot-backend/pkg/database"
)

type CreateFeedRequest struct {
	Title            string `json:"title"`
	Description      string `json:"description"`
	URL              string `json:"url" binding:"required"`
	CategoryID       *uint  `json:"category_id"`
	Icon             string `json:"icon"`
	Color            string `json:"color"`
	MaxArticles      int    `json:"max_articles"`
	RefreshInterval  int    `json:"refresh_interval"`
	AISummaryEnabled bool   `json:"ai_summary_enabled"`
}

type UpdateFeedRequest struct {
	Title            string `json:"title"`
	Description      string `json:"description"`
	URL              string `json:"url"`
	CategoryID       *uint  `json:"category_id"`
	Icon             string `json:"icon"`
	Color            string `json:"color"`
	MaxArticles      int    `json:"max_articles"`
	RefreshInterval  int    `json:"refresh_interval"`
	AISummaryEnabled bool   `json:"ai_summary_enabled"`
}

func GetFeeds(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	categoryID, _ := strconv.Atoi(c.Query("category_id"))
	uncategorized := c.Query("uncategorized") == "true"

	query := database.DB.Model(&models.Feed{})

	if categoryID > 0 {
		query = query.Where("category_id = ?", categoryID)
	}

	if uncategorized {
		query = query.Where("category_id IS NULL")
	}

	var total int64
	query.Count(&total)

	var feeds []models.Feed
	if perPage >= 10000 {
		if err := query.Order("title ASC").Find(&feeds).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		data := make([]map[string]interface{}, len(feeds))
		for i, feed := range feeds {
			database.DB.Preload("Articles").First(&feed, feed.ID)
			data[i] = feed.ToDict(true)
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    data,
			"pagination": gin.H{
				"page":     1,
				"per_page": len(feeds),
				"total":    total,
				"pages":    1,
			},
		})
		return
	}

	offset := (page - 1) * perPage
	if err := query.Order("title ASC").Offset(offset).Limit(perPage).Find(&feeds).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	data := make([]map[string]interface{}, len(feeds))
	for i, feed := range feeds {
		database.DB.Preload("Articles").First(&feed, feed.ID)
		data[i] = feed.ToDict(true)
	}

	pages := int(total) / perPage
	if int(total)%perPage > 0 {
		pages++
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"pagination": gin.H{
			"page":     page,
			"per_page": perPage,
			"total":    total,
			"pages":    pages,
		},
	})
}

func CreateFeed(c *gin.Context) {
	var req CreateFeedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Feed URL is required",
		})
		return
	}

	var existing models.Feed
	if err := database.DB.Where("url = ?", req.URL).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"error":   "Feed with this URL already exists",
		})
		return
	}

	now := time.Now()
	feed := models.Feed{
		Title:            req.Title,
		Description:      req.Description,
		URL:              req.URL,
		CategoryID:       req.CategoryID,
		Icon:             req.Icon,
		Color:            req.Color,
		MaxArticles:      req.MaxArticles,
		RefreshInterval:  req.RefreshInterval,
		AISummaryEnabled: req.AISummaryEnabled,
		LastUpdated:      &now,
	}

	if feed.Title == "" {
		feed.Title = "Untitled Feed"
	}
	if feed.Icon == "" {
		feed.Icon = "rss"
	}
	if feed.Color == "" {
		feed.Color = "#8b5cf6"
	}
	if feed.MaxArticles == 0 {
		feed.MaxArticles = 100
	}
	if feed.RefreshInterval == 0 {
		feed.RefreshInterval = 60
	}

	if err := database.DB.Create(&feed).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	database.DB.Preload("Articles").First(&feed, feed.ID)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    feed.ToDict(true),
		"message": "Feed created successfully",
	})
}

func UpdateFeed(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("feed_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid feed ID",
		})
		return
	}

	var feed models.Feed
	if err := database.DB.First(&feed, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Feed not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   err.Error(),
			})
		}
		return
	}

	// Read raw body BEFORE binding to check which fields are present
	rawBody, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Failed to read request body",
		})
		return
	}

	var bodyMap map[string]interface{}
	if err := json.Unmarshal(rawBody, &bodyMap); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid JSON",
		})
		return
	}

	var req UpdateFeedRequest
	if err := json.Unmarshal(rawBody, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
		})
		return
	}

	updates := make(map[string]interface{})
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.URL != "" {
		updates["url"] = req.URL
	}
	// Only update category_id if explicitly provided (not zero/nil)
	if req.CategoryID != nil {
		updates["category_id"] = *req.CategoryID
	}
	if req.Icon != "" {
		updates["icon"] = req.Icon
	}
	if req.Color != "" {
		updates["color"] = req.Color
	}
	if req.MaxArticles > 0 {
		updates["max_articles"] = req.MaxArticles
	}
	if req.RefreshInterval >= 0 {
		updates["refresh_interval"] = req.RefreshInterval
	}
	// Check if ai_summary_enabled exists in request body
	if val, exists := bodyMap["ai_summary_enabled"]; exists {
		// Use the actual value from bodyMap to preserve boolean type
		updates["ai_summary_enabled"] = val
	}

	if err := database.DB.Model(&feed).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	database.DB.Preload("Articles").First(&feed, uint(id))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    feed.ToDict(true),
	})
}

func DeleteFeed(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("feed_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid feed ID",
		})
		return
	}

	var feed models.Feed
	if err := database.DB.First(&feed, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Feed not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   err.Error(),
			})
		}
		return
	}

	if err := database.DB.Delete(&feed).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Feed deleted successfully",
	})
}

func RefreshFeed(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("feed_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid feed ID",
		})
		return
	}

	var feed models.Feed
	if err := database.DB.First(&feed, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Feed not found",
		})
		return
	}

	now := time.Now()
	feed.RefreshStatus = "refreshing"
	feed.LastRefreshAt = &now
	feed.RefreshError = ""
	database.DB.Save(&feed)

	go func() {
		refreshFeedWorker(uint(id))
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"success": true,
		"message": "Started refreshing feed in background",
	})
}

func refreshFeedWorker(feedID uint) {
	feedService := services.NewFeedService()
	if err := feedService.RefreshFeed(feedID); err != nil {
		return
	}
}

func FetchFeed(c *gin.Context) {
	var req struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Feed URL is required",
		})
		return
	}

	feedService := services.NewFeedService()
	title, description, err := feedService.FetchFeedPreview(req.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"title":       title,
			"description": description,
		},
	})
}

func RefreshAllFeeds(c *gin.Context) {
	var feeds []models.Feed
	if err := database.DB.Find(&feeds).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	if len(feeds) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "No feeds to refresh",
		})
		return
	}

	for _, feed := range feeds {
		now := time.Now()
		feed.RefreshStatus = "refreshing"
		feed.LastRefreshAt = &now
		feed.RefreshError = ""
		database.DB.Save(&feed)
	}

	go func() {
		refreshAllFeedsWorker()
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"success": true,
		"message": "Started refreshing all feeds in background",
		"data": gin.H{
			"total_feeds": len(feeds),
		},
	})
}

func refreshAllFeedsWorker() {
	var feeds []models.Feed
	database.DB.Find(&feeds)

	feedService := services.NewFeedService()
	for _, feed := range feeds {
		if err := feedService.RefreshFeed(feed.ID); err != nil {
			continue
		}
	}
}
