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
var AISummarySchedulerInterface interface{}
var FirecrawlSchedulerInterface interface{}
var DigestSchedulerInterface interface{}

func safeGetStatus(scheduler interface{}, name, description string) map[string]interface{} {
	if scheduler == nil {
		return nil
	}

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

	if status := safeGetStatus(AISummarySchedulerInterface, "ai_summary", "AI summarize Firecrawl content"); status != nil {
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
	case "ai_summary":
		scheduler = AISummarySchedulerInterface
		description = "AI summarize Firecrawl content"
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
	case "auto_refresh":
		if AutoRefreshSchedulerInterface != nil {
			if scheduler, ok := AutoRefreshSchedulerInterface.(interface{ TriggerNow() map[string]interface{} }); ok {
				respondTriggerResult(c, name, scheduler.TriggerNow())
				return
			}
			if scheduler, ok := AutoRefreshSchedulerInterface.(interface{ Trigger() }); ok {
				log.Println("Triggering auto-refresh scheduler manually")
				scheduler.Trigger()
				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"message": "Auto-refresh scheduler triggered",
					"data": gin.H{
						"name":   name,
						"status": "triggered",
					},
				})
				return
			}
		}
	case "auto_summary":
		if AutoSummarySchedulerInterface != nil {
			if scheduler, ok := AutoSummarySchedulerInterface.(interface{ TriggerNow() map[string]interface{} }); ok {
				respondTriggerResult(c, name, scheduler.TriggerNow())
				return
			}
		}
	case "ai_summary":
		if AISummarySchedulerInterface != nil {
			if scheduler, ok := AISummarySchedulerInterface.(interface{ Trigger() }); ok {
				log.Println("Triggering ai_summary scheduler manually")
				scheduler.Trigger()
				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"message": "AI summary scheduler triggered",
					"data": gin.H{
						"name":   name,
						"status": "triggered",
					},
				})
				return
			}
		}
	case "firecrawl":
		if FirecrawlSchedulerInterface != nil {
			if scheduler, ok := FirecrawlSchedulerInterface.(interface{ Trigger() }); ok {
				log.Println("Triggering firecrawl scheduler manually")
				scheduler.Trigger()
				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"message": "Firecrawl scheduler triggered",
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

func respondTriggerResult(c *gin.Context, name string, result map[string]interface{}) {
	statusCode := http.StatusOK
	if rawCode, ok := result["status_code"].(int); ok {
		statusCode = rawCode
	}
	delete(result, "status_code")
	result["name"] = name

	accepted, _ := result["accepted"].(bool)
	message, _ := result["message"].(string)
	if accepted {
		c.JSON(statusCode, gin.H{
			"success": true,
			"message": message,
			"data":    result,
		})
		return
	}

	c.JSON(statusCode, gin.H{
		"success": false,
		"error":   message,
		"data":    result,
	})
}

func ResetSchedulerStats(c *gin.Context) {
	name := c.Param("name")

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
