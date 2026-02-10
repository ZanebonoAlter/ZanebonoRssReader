package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"my-robot-backend/internal/models"
	"my-robot-backend/internal/services"
	"my-robot-backend/pkg/database"
)

type TrackBehaviorRequest struct {
	ArticleID   uint   `json:"article_id" binding:"required"`
	FeedID      uint   `json:"feed_id" binding:"required"`
	CategoryID  *uint  `json:"category_id"`
	SessionID   string `json:"session_id" binding:"required"`
	EventType   string `json:"event_type" binding:"required"`
	ScrollDepth int    `json:"scroll_depth"`
	ReadingTime int    `json:"reading_time"`
}

type BatchTrackBehaviorRequest struct {
	Events []TrackBehaviorRequest `json:"events" binding:"required"`
}

func TrackReadingBehavior(c *gin.Context) {
	var req TrackBehaviorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
		})
		return
	}

	categoryID := req.CategoryID

	if categoryID == nil || *categoryID == 0 {
		var feed models.Feed
		if err := database.DB.Select("category_id").First(&feed, req.FeedID).Error; err == nil {
			categoryID = feed.CategoryID
		} else {
			categoryID = nil // Explicitly set to nil if feed not found
		}
	}

	// Debug log
	log.Printf("[TrackBehavior] FeedID: %d, Request.CategoryID: %v, Final.CategoryID: %v",
		req.FeedID, req.CategoryID, categoryID)

	behavior := models.ReadingBehavior{
		ArticleID:   req.ArticleID,
		FeedID:      req.FeedID,
		CategoryID:  categoryID,
		SessionID:   req.SessionID,
		EventType:   req.EventType,
		ScrollDepth: req.ScrollDepth,
		ReadingTime: req.ReadingTime,
		CreatedAt:   time.Now(),
	}

	if err := database.DB.Create(&behavior).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    behavior.ToDict(),
	})
}

func BatchTrackReadingBehavior(c *gin.Context) {
	var req BatchTrackBehaviorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
		})
		return
	}

	feedCategoryMap := make(map[uint]*uint)

	behaviors := make([]models.ReadingBehavior, len(req.Events))
	for i, event := range req.Events {
		categoryID := event.CategoryID

		if categoryID == nil || *categoryID == 0 {
			if cachedCategoryID, exists := feedCategoryMap[event.FeedID]; exists {
				categoryID = cachedCategoryID
			} else {
				var feed models.Feed
				if err := database.DB.Select("category_id").First(&feed, event.FeedID).Error; err == nil {
					categoryID = feed.CategoryID
					feedCategoryMap[event.FeedID] = categoryID
				}
			}
		}

		behaviors[i] = models.ReadingBehavior{
			ArticleID:   event.ArticleID,
			FeedID:      event.FeedID,
			CategoryID:  categoryID,
			SessionID:   event.SessionID,
			EventType:   event.EventType,
			ScrollDepth: event.ScrollDepth,
			ReadingTime: event.ReadingTime,
			CreatedAt:   time.Now(),
		}
	}

	if err := database.DB.Create(&behaviors).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": len(behaviors),
	})
}

func GetReadingStats(c *gin.Context) {
	var stats struct {
		TotalArticles      int64   `json:"total_articles"`
		TotalReadingTime   int     `json:"total_reading_time"`
		AvgReadingTime     float64 `json:"avg_reading_time"`
		AvgScrollDepth     float64 `json:"avg_scroll_depth"`
		MostActiveFeedID   uint    `json:"most_active_feed_id"`
		MostActiveCategory uint    `json:"most_active_category"`
	}

	database.DB.Model(&models.ReadingBehavior{}).
		Distinct("article_id").
		Count(&stats.TotalArticles)

	database.DB.Model(&models.ReadingBehavior{}).
		Select("COALESCE(SUM(reading_time), 0)").
		Scan(&stats.TotalReadingTime)

	var avgTime sql.NullFloat64
	database.DB.Model(&models.ReadingBehavior{}).
		Where("reading_time > 0").
		Select("AVG(reading_time)").
		Scan(&avgTime)
	if avgTime.Valid {
		stats.AvgReadingTime = avgTime.Float64
	}

	var avgDepth sql.NullFloat64
	database.DB.Model(&models.ReadingBehavior{}).
		Where("scroll_depth > 0").
		Select("AVG(scroll_depth)").
		Scan(&avgDepth)
	if avgDepth.Valid {
		stats.AvgScrollDepth = avgDepth.Float64
	}

	type FeedCount struct {
		FeedID uint
		Count  int64
	}
	var feedCounts []FeedCount
	database.DB.Model(&models.ReadingBehavior{}).
		Select("feed_id, COUNT(*) as count").
		Group("feed_id").
		Order("count DESC").
		Limit(1).
		Scan(&feedCounts)
	if len(feedCounts) > 0 {
		stats.MostActiveFeedID = feedCounts[0].FeedID
	}

	type CategoryCount struct {
		CategoryID *uint
		Count      int64
	}
	var categoryCounts []CategoryCount
	database.DB.Model(&models.ReadingBehavior{}).
		Select("category_id, COUNT(*) as count").
		Where("category_id IS NOT NULL").
		Group("category_id").
		Order("count DESC").
		Limit(1).
		Scan(&categoryCounts)
	if len(categoryCounts) > 0 && categoryCounts[0].CategoryID != nil {
		stats.MostActiveCategory = *categoryCounts[0].CategoryID
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

func GetUserPreferences(c *gin.Context) {
	preferenceType := c.Query("type")

	var preferences []models.UserPreference
	query := database.DB.Model(&models.UserPreference{})

	if preferenceType == "feed" {
		query = query.Where("feed_id IS NOT NULL")
	} else if preferenceType == "category" {
		query = query.Where("category_id IS NOT NULL")
	}

	if err := query.
		Preload("Feed").
		Preload("Category").
		Order("preference_score DESC").
		Find(&preferences).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	data := make([]map[string]interface{}, len(preferences))
	for i, pref := range preferences {
		data[i] = pref.ToDict()
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

func TriggerPreferenceUpdate(c *gin.Context) {
	go func() {
		service := services.NewPreferenceService(database.DB)
		service.UpdateAllPreferences()
	}()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Preference update triggered",
	})
}
