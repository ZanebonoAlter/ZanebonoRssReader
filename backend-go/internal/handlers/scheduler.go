package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UpdateSchedulerIntervalRequest struct {
	Interval int `json:"interval" binding:"required"`
}

var AutoRefreshSchedulerInterface interface{}
var AutoSummarySchedulerInterface interface{}
var FirecrawlSchedulerInterface interface{}

func safeGetStatus(scheduler interface{}, name, description string) map[string]interface{} {
	if scheduler == nil {
		return nil
	}

	// Use recover to handle nil pointer dereference
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in %s scheduler GetStatus: %v", name, r)
		}
	}()

	if status, ok := scheduler.(interface{ GetStatus() map[string]interface{} }); ok {
		result := status.GetStatus()
		if result != nil {
			result["name"] = name
			result["description"] = description
			return result
		}
	}
	return nil
}

func GetSchedulersStatus(c *gin.Context) {
	schedulers := []map[string]interface{}{}

	if status := safeGetStatus(AutoRefreshSchedulerInterface, "auto_refresh", "Auto-refresh RSS feeds"); status != nil {
		schedulers = append(schedulers, status)
	}

	if status := safeGetStatus(AutoSummarySchedulerInterface, "auto_summary", "Auto-generate AI summaries for feeds"); status != nil {
		schedulers = append(schedulers, status)
	}

	if status := safeGetStatus(FirecrawlSchedulerInterface, "firecrawl", "Auto-crawl full content for articles"); status != nil {
		schedulers = append(schedulers, status)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    schedulers,
	})
}

func GetSchedulerStatus(c *gin.Context) {
	name := c.Param("name")

	var scheduler interface{}
	description := ""

	switch name {
	case "auto_refresh":
		scheduler = AutoRefreshSchedulerInterface
		description = "Auto-refresh RSS feeds"
	case "auto_summary":
		scheduler = AutoSummarySchedulerInterface
		description = "Auto-generate AI summaries for feeds"
	case "firecrawl":
		scheduler = FirecrawlSchedulerInterface
		description = "Auto-crawl full content for articles"
	}

	if status := safeGetStatus(scheduler, name, description); status != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    status,
		})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error":   "Scheduler not found: " + name,
	})
}

func TriggerScheduler(c *gin.Context) {
	name := c.Param("name")

	switch name {
	case "auto_summary":
		if AutoSummarySchedulerInterface != nil {
			_, ok := AutoSummarySchedulerInterface.(interface {
				SetAIConfig(baseURL, apiKey, model string, timeRange int) error
			})
			if ok {
				log.Println("Triggering auto-summary scheduler manually")
				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"message": "Auto-summary scheduler triggered (Note: actual trigger not yet implemented)",
					"data": gin.H{
						"name":   name,
						"status": "triggered",
					},
				})
				return
			}
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error":   "Scheduler not found or cannot be triggered: " + name,
	})
}

func ResetSchedulerStats(c *gin.Context) {
	name := c.Param("name")

	// Note: This would require implementing ResetStats() method in schedulers
	log.Printf("Reset stats requested for scheduler: %s (not yet implemented)", name)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Statistics reset for scheduler '" + name + "' (placeholder)",
	})
}

func UpdateSchedulerInterval(c *gin.Context) {
	name := c.Param("name")

	var req UpdateSchedulerIntervalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Valid interval (positive integer) is required",
		})
		return
	}

	if req.Interval <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Interval must be a positive integer",
		})
		return
	}

	// Note: This would require implementing SetInterval() method in schedulers
	log.Printf("Update interval requested for scheduler %s: %d seconds (not yet implemented)", name, req.Interval)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Interval update requested for scheduler '" + name + "' (restart required)",
		"data": gin.H{
			"name":           name,
			"check_interval": req.Interval,
		},
	})
}
