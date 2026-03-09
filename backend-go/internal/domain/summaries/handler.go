package summaries

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"my-robot-backend/internal/app/runtimeinfo"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/aisettings"
	"my-robot-backend/internal/platform/database"
)

type GenerateSummaryRequest struct {
	FeedID     *uint  `json:"feed_id"`
	CategoryID *uint  `json:"category_id"`
	TimeRange  int    `json:"time_range"`
	BaseURL    string `json:"base_url"`
	APIKey     string `json:"api_key" binding:"required"`
	Model      string `json:"model"`
}

func GetSummaries(c *gin.Context) {
	feedID, _ := strconv.Atoi(c.Query("feed_id"))
	categoryID, _ := strconv.Atoi(c.Query("category_id"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	query := database.DB.Model(&models.AISummary{})

	if feedID > 0 {
		query = query.Where("feed_id = ?", feedID)
	}
	if categoryID > 0 {
		query = query.Where("category_id = ?", categoryID)
	}

	var total int64
	query.Count(&total)

	var summaries []models.AISummary
	offset := (page - 1) * perPage
	if err := query.Preload("Feed").Preload("Category").Order("created_at DESC").Offset(offset).Limit(perPage).Find(&summaries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	data := make([]map[string]interface{}, len(summaries))
	for i, summary := range summaries {
		data[i] = summary.ToDict()
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

func GetSummary(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("summary_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid summary ID",
		})
		return
	}

	var summary models.AISummary
	if err := database.DB.Preload("Category").Preload("Feed").First(&summary, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Summary not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   err.Error(),
			})
		}
		return
	}

	summaryDict := summary.ToDict()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summaryDict,
	})
}

func DeleteSummary(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("summary_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid summary ID",
		})
		return
	}

	var summary models.AISummary
	if err := database.DB.First(&summary, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Summary not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   err.Error(),
			})
		}
		return
	}

	if err := database.DB.Delete(&summary).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Summary deleted successfully",
	})
}

type AutoSummaryConfig struct {
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key"`
	Model     string `json:"model"`
	TimeRange int    `json:"time_range"`
}

func GetAutoSummaryStatus(c *gin.Context) {
	var settings models.AISettings
	err := database.DB.Where("key = ?", "summary_config").First(&settings).Error

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"enabled": false,
				"status":  "not_configured",
			},
		})
		return
	}

	var config AutoSummaryConfig
	if err := settings.ParseValue(&config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to parse configuration",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled":    true,
			"status":     "configured",
			"base_url":   config.BaseURL,
			"model":      config.Model,
			"time_range": config.TimeRange,
		},
	})
}

func UpdateAutoSummaryConfig(c *gin.Context) {
	var req AutoSummaryConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
		})
		return
	}

	configJSON, _, err := aisettings.LoadSummaryConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	configJSON["base_url"] = req.BaseURL
	configJSON["api_key"] = req.APIKey
	configJSON["model"] = req.Model
	configJSON["time_range"] = req.TimeRange

	if err := aisettings.SaveSummaryConfig(configJSON, "AI summary generation configuration (including auto-summary)"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	if runtimeinfo.AutoSummarySchedulerInterface != nil {
		if scheduler, ok := runtimeinfo.AutoSummarySchedulerInterface.(interface {
			SetAIConfig(baseURL, apiKey, model string, timeRange int) error
		}); ok {
			scheduler.SetAIConfig(req.BaseURL, req.APIKey, req.Model, req.TimeRange)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Auto summary configuration updated successfully",
	})
}

type QueueSummaryRequest struct {
	CategoryIDs []uint `json:"category_ids"`
	FeedIDs     []uint `json:"feed_ids"`
	TimeRange   int    `json:"time_range"`
	BaseURL     string `json:"base_url"`
	APIKey      string `json:"api_key" binding:"required"`
	Model       string `json:"model"`
}

func SubmitQueueSummary(c *gin.Context) {
	var req QueueSummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "请求参数错误: " + err.Error(),
		})
		return
	}

	if len(req.CategoryIDs) == 0 && len(req.FeedIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "请选择分类或订阅源",
		})
		return
	}

	queue := GetSummaryQueue()
	config := AIConfig{
		BaseURL:   req.BaseURL,
		APIKey:    req.APIKey,
		Model:     req.Model,
		TimeRange: req.TimeRange,
	}

	batch := queue.SubmitBatch(req.CategoryIDs, config)

	c.JSON(http.StatusAccepted, gin.H{
		"success": true,
		"message": "Summary job queued successfully",
		"data":    batch,
	})
}

func GetQueueStatus(c *gin.Context) {
	queue := GetSummaryQueue()
	batch := queue.GetCurrentBatch()

	if batch == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    batch,
	})
}

func GetQueueJob(c *gin.Context) {
	jobID := c.Param("job_id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "任务ID不能为空",
		})
		return
	}

	queue := GetSummaryQueue()
	job := queue.GetJob(jobID)

	if job == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Job not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    job,
	})
}

type FormatDatetimeCST func(time.Time) string

var FormatDatetimeCSTImpl = func(t time.Time) string {
	return t.In(time.FixedZone("CST", 8*3600)).Format("2006-01-02 15:04:05")
}
