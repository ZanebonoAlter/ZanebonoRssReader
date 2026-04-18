package topicextraction

import (
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

var tagQueueStatusService *tagQueueStatusReader
var tagQueueStatusOnce sync.Once

type tagQueueStatusReader struct {
	db *gorm.DB
}

func getTagQueueStatusReader() *tagQueueStatusReader {
	tagQueueStatusOnce.Do(func() {
		tagQueueStatusService = &tagQueueStatusReader{db: database.DB}
	})
	return tagQueueStatusService
}

type tagQueueStatusCounts struct {
	Pending    int64 `json:"pending"`
	Processing int64 `json:"processing"`
	Completed  int64 `json:"completed"`
	Failed     int64 `json:"failed"`
	Total      int64 `json:"total"`
}

func GetTagQueueStatus(c *gin.Context) {
	reader := getTagQueueStatusReader()

	type statusRow struct {
		Status string
		Count  int64
	}
	var rows []statusRow
	err := reader.db.Model(&models.TagJob{}).
		Select("status, count(*) as count").
		Group("status").
		Scan(&rows).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	counts := tagQueueStatusCounts{}
	var total int64
	for _, r := range rows {
		total += r.Count
		switch r.Status {
		case string(models.JobStatusPending):
			counts.Pending = r.Count
		case string(models.JobStatusLeased):
			counts.Processing = r.Count
		case string(models.JobStatusCompleted):
			counts.Completed = r.Count
		case string(models.JobStatusFailed):
			counts.Failed = r.Count
		}
	}
	counts.Total = total

	c.JSON(http.StatusOK, gin.H{"success": true, "data": counts})
}

func GetTagQueueTasks(c *gin.Context) {
	reader := getTagQueueStatusReader()
	statusFilter := c.Query("status")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := reader.db.Model(&models.TagJob{}).
		Preload("Article", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, title")
		})

	if statusFilter != "" {
		query = query.Where("status = ?", statusFilter)
	}

	var total int64
	countQuery := reader.db.Model(&models.TagJob{})
	if statusFilter != "" {
		countQuery = countQuery.Where("status = ?", statusFilter)
	}
	countQuery.Count(&total)

	var jobs []models.TagJob
	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&jobs).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	type taskRow struct {
		ID                   uint    `json:"id"`
		ArticleID            uint    `json:"article_id"`
		ArticleTitle         string  `json:"article_title"`
		FeedNameSnapshot     string  `json:"feed_name_snapshot"`
		CategoryNameSnapshot string  `json:"category_name_snapshot"`
		Priority             int     `json:"priority"`
		Status               string  `json:"status"`
		AttemptCount         int     `json:"attempt_count"`
		MaxAttempts          int     `json:"max_attempts"`
		ForceRetag           bool    `json:"force_retag"`
		Reason               string  `json:"reason"`
		LastError            string  `json:"last_error,omitempty"`
		CreatedAt            string  `json:"created_at"`
		LeasedAt             *string `json:"leased_at,omitempty"`
	}

	tasks := make([]taskRow, 0, len(jobs))
	for _, j := range jobs {
		title := ""
		if j.Article != nil {
			title = j.Article.Title
		}
		var leasedAt *string
		if j.LeasedAt != nil {
			s := j.LeasedAt.Format("2006-01-02T15:04:05Z07:00")
			leasedAt = &s
		}
		tasks = append(tasks, taskRow{
			ID:                   j.ID,
			ArticleID:            j.ArticleID,
			ArticleTitle:         title,
			FeedNameSnapshot:     j.FeedNameSnapshot,
			CategoryNameSnapshot: j.CategoryNameSnapshot,
			Priority:             j.Priority,
			Status:               j.Status,
			AttemptCount:         j.AttemptCount,
			MaxAttempts:          j.MaxAttempts,
			ForceRetag:           j.ForceRetag,
			Reason:               j.Reason,
			LastError:            j.LastError,
			CreatedAt:            j.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			LeasedAt:             leasedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"tasks": tasks,
			"total": total,
		},
	})
}

func RetryTagQueueFailed(c *gin.Context) {
	reader := getTagQueueStatusReader()

	result := reader.db.Model(&models.TagJob{}).
		Where("status = ?", string(models.JobStatusFailed)).
		Updates(map[string]interface{}{
			"status":           string(models.JobStatusPending),
			"attempt_count":    0,
			"leased_at":        nil,
			"lease_expires_at": nil,
		})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "已重试 " + strconv.FormatInt(result.RowsAffected, 10) + " 个失败任务",
	})
}

func RegisterTagQueueRoutes(rg *gin.RouterGroup) {
	queue := rg.Group("/tag-queue")
	{
		queue.GET("/status", GetTagQueueStatus)
		queue.GET("/tasks", GetTagQueueTasks)
		queue.POST("/retry", RetryTagQueueFailed)
	}
}
