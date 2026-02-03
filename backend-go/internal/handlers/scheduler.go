package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UpdateSchedulerIntervalRequest struct {
	Interval int `json:"interval" binding:"required"`
}

// Global scheduler references (will be set by main.go)
var AutoRefreshSchedulerInterface interface{}
var AutoSummarySchedulerInterface interface{}

func GetSchedulersStatus(c *gin.Context) {
	schedulers := []map[string]interface{}{}

	// Add auto-refresh scheduler status
	if AutoRefreshSchedulerInterface != nil {
		if status, ok := AutoRefreshSchedulerInterface.(interface{ GetStatus() map[string]interface{} }); ok {
			refreshStatus := status.GetStatus()
			refreshStatus["name"] = "auto_refresh"
			refreshStatus["description"] = "Auto-refresh RSS feeds"
			schedulers = append(schedulers, refreshStatus)
		}
	}

	// Add auto-summary scheduler status
	if AutoSummarySchedulerInterface != nil {
		if status, ok := AutoSummarySchedulerInterface.(interface{ GetStatus() map[string]interface{} }); ok {
			summaryStatus := status.GetStatus()
			summaryStatus["name"] = "auto_summary"
			summaryStatus["description"] = "Auto-generate AI summaries for categories"
			schedulers = append(schedulers, summaryStatus)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    schedulers,
	})
}

func GetSchedulerStatus(c *gin.Context) {
	name := c.Param("name")

	switch name {
	case "auto_refresh":
		if AutoRefreshSchedulerInterface != nil {
			if status, ok := AutoRefreshSchedulerInterface.(interface{ GetStatus() map[string]interface{} }); ok {
				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"data":    status.GetStatus(),
				})
				return
			}
		}

	case "auto_summary":
		if AutoSummarySchedulerInterface != nil {
			if status, ok := AutoSummarySchedulerInterface.(interface{ GetStatus() map[string]interface{} }); ok {
				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"data":    status.GetStatus(),
				})
				return
			}
		}
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
			// Check if it's the right type (even if we don't use the variable)
			_, ok := AutoSummarySchedulerInterface.(interface {
				SetAIConfig(baseURL, apiKey, model string) error
			})
			if ok {
				// Load AI config and trigger
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
