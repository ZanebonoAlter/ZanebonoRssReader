package jobs

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"my-robot-backend/internal/app/runtimeinfo"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/domain/summaries"
	"my-robot-backend/internal/platform/database"
)

type UpdateSchedulerIntervalRequest struct {
	Interval int `json:"interval" binding:"required"`
}

type SchedulerStatusResponse struct {
	Name          string `json:"name"`
	Status        string `json:"status"`
	CheckInterval int64  `json:"check_interval"`
	NextRun       int64  `json:"next_run"`
	IsExecuting   bool   `json:"is_executing"`
}

type schedulerDescriptor struct {
	Name        string
	Aliases     []string
	Description string
	TaskName    string
	Get         func() interface{}
}

func schedulerDescriptors() []schedulerDescriptor {
	return []schedulerDescriptor{
		{
			Name:        "auto_refresh",
			Description: "Auto-refresh RSS feeds",
			TaskName:    "auto_refresh",
			Get: func() interface{} {
				return runtimeinfo.AutoRefreshSchedulerInterface
			},
		},
		{
			Name:        "auto_summary",
			Description: "Auto-generate AI summaries for feeds",
			TaskName:    "auto_summary",
			Get: func() interface{} {
				return runtimeinfo.AutoSummarySchedulerInterface
			},
		},
		{
			Name:        "preference_update",
			Description: "Update reading preferences from behavior data",
			Get: func() interface{} {
				return runtimeinfo.PreferenceUpdateSchedulerInterface
			},
		},
		{
			Name:        "content_completion",
			Aliases:     []string{"ai_summary"},
			Description: "Complete article content and generate article summaries",
			TaskName:    "ai_summary",
			Get: func() interface{} {
				return runtimeinfo.AISummarySchedulerInterface
			},
		},
		{
			Name:        "firecrawl",
			Description: "Auto-crawl full content for articles",
			Get: func() interface{} {
				return runtimeinfo.FirecrawlSchedulerInterface
			},
		},
		{
			Name:        "digest",
			Description: "Run digest cron schedules",
			Get: func() interface{} {
				return runtimeinfo.DigestSchedulerInterface
			},
		},
	}
}

func resolveScheduler(name string) (*schedulerDescriptor, interface{}) {
	for _, descriptor := range schedulerDescriptors() {
		if descriptor.Name == name {
			return &descriptor, descriptor.Get()
		}
		for _, alias := range descriptor.Aliases {
			if alias == name {
				return &descriptor, descriptor.Get()
			}
		}
	}
	return nil, nil
}

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
	schedulers := make([]map[string]interface{}, 0)
	for _, descriptor := range schedulerDescriptors() {
		if status := safeGetStatus(descriptor.Get(), descriptor.Name, descriptor.Description); status != nil {
			schedulers = append(schedulers, status)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    schedulers,
	})
}

func GetSchedulerStatus(c *gin.Context) {
	name := c.Param("name")
	descriptor, scheduler := resolveScheduler(name)
	if descriptor == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Scheduler not found: " + name})
		return
	}

	if status := safeGetStatus(scheduler, descriptor.Name, descriptor.Description); status != nil {
		if name != descriptor.Name {
			status["requested_name"] = name
			status["alias_of"] = descriptor.Name
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": status})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Scheduler not found: " + name})
}

func TriggerScheduler(c *gin.Context) {
	requestedName := c.Param("name")
	descriptor, scheduler := resolveScheduler(requestedName)
	if descriptor == nil || scheduler == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Scheduler not found or cannot be triggered: " + requestedName})
		return
	}

	if triggerable, ok := scheduler.(interface{ TriggerNow() map[string]interface{} }); ok {
		respondTriggerResult(c, descriptor.Name, triggerable.TriggerNow())
		return
	}

	if triggerable, ok := scheduler.(interface{ Trigger() }); ok {
		log.Printf("Triggering %s scheduler manually", descriptor.Name)
		triggerable.Trigger()
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": descriptor.Description + " triggered",
			"data": gin.H{
				"name":   descriptor.Name,
				"status": "triggered",
			},
		})
		return
	}

	c.JSON(http.StatusConflict, gin.H{"success": false, "error": "Scheduler cannot be triggered: " + requestedName})
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
		c.JSON(statusCode, gin.H{"success": true, "message": message, "data": result})
		return
	}

	c.JSON(statusCode, gin.H{"success": false, "error": message, "data": result})
}

func ResetSchedulerStats(c *gin.Context) {
	requestedName := c.Param("name")
	descriptor, scheduler := resolveScheduler(requestedName)
	if descriptor == nil || scheduler == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Scheduler not found: " + requestedName})
		return
	}

	if resettable, ok := scheduler.(interface{ ResetStats() error }); ok {
		if err := resettable.ResetStats(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("Statistics reset for scheduler '%s'", descriptor.Name)})
		return
	}

	if descriptor.TaskName != "" {
		if err := resetSchedulerTask(descriptor.TaskName); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("Statistics reset for scheduler '%s'", descriptor.Name)})
		return
	}

	c.JSON(http.StatusConflict, gin.H{"success": false, "error": "Scheduler stats cannot be reset: " + requestedName})
}

func UpdateSchedulerInterval(c *gin.Context) {
	requestedName := c.Param("name")
	descriptor, scheduler := resolveScheduler(requestedName)
	if descriptor == nil || scheduler == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Scheduler not found: " + requestedName})
		return
	}

	var req UpdateSchedulerIntervalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Valid interval (positive integer) is required"})
		return
	}
	if req.Interval <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Interval must be a positive integer"})
		return
	}

	updatable, ok := scheduler.(interface{ UpdateInterval(int) error })
	if !ok {
		c.JSON(http.StatusConflict, gin.H{"success": false, "error": "Scheduler interval cannot be updated: " + requestedName})
		return
	}

	if err := updatable.UpdateInterval(req.Interval); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Interval updated for scheduler '%s'", descriptor.Name),
		"data": gin.H{
			"name":           descriptor.Name,
			"check_interval": req.Interval,
		},
	})
}

func GetTasksStatus(c *gin.Context) {
	tasks := make([]gin.H, 0)
	queueSize := 0
	activeTasks := 0

	if batch := summaries.GetSummaryQueue().GetCurrentBatch(); batch != nil {
		pendingJobs := batch.TotalJobs - batch.CompletedJobs - batch.FailedJobs
		if pendingJobs < 0 {
			pendingJobs = 0
		}
		queueSize += pendingJobs
		activeTasks++
		tasks = append(tasks, gin.H{
			"type":           "summary_queue",
			"status":         batch.Status,
			"batch_id":       batch.ID,
			"total_jobs":     batch.TotalJobs,
			"completed_jobs": batch.CompletedJobs,
			"failed_jobs":    batch.FailedJobs,
			"pending_jobs":   pendingJobs,
		})
	}

	if status := safeGetStatus(runtimeinfo.AISummarySchedulerInterface, "content_completion", "Complete article content and generate article summaries"); status != nil {
		if overview, ok := status["overview"].(map[string]interface{}); ok {
			pendingCount := asInt(overview["pending_count"])
			processingCount := asInt(overview["processing_count"])
			if pendingCount > 0 || processingCount > 0 {
				queueSize += pendingCount
				activeTasks++
				tasks = append(tasks, gin.H{
					"type":             "content_completion",
					"status":           status["status"],
					"pending_count":    pendingCount,
					"processing_count": processingCount,
					"overview":         overview,
				})
			}
		}
	}

	if status := safeGetStatus(runtimeinfo.FirecrawlSchedulerInterface, "firecrawl", "Auto-crawl full content for articles"); status != nil {
		queueCount := asInt(status["queue_size"])
		processingCount := asInt(status["processing"])
		if queueCount > 0 || processingCount > 0 {
			queueSize += queueCount
			activeTasks++
			tasks = append(tasks, gin.H{
				"type":             "firecrawl",
				"status":           status["status"],
				"queue_size":       queueCount,
				"processing_count": processingCount,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"queue_size":   queueSize,
			"active_tasks": activeTasks,
			"tasks":        tasks,
		},
	})
}

func resetSchedulerTask(taskName string) error {
	var task models.SchedulerTask
	if err := database.DB.Where("name = ?", taskName).First(&task).Error; err != nil {
		return err
	}

	return database.DB.Model(&task).Updates(map[string]interface{}{
		"status":                  "idle",
		"last_error":              "",
		"last_error_time":         nil,
		"total_executions":        0,
		"successful_executions":   0,
		"failed_executions":       0,
		"consecutive_failures":    0,
		"last_execution_time":     nil,
		"last_execution_duration": nil,
		"last_execution_result":   "",
	}).Error
}

func asInt(value interface{}) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}
