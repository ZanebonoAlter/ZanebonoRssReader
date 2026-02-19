package schedulers

import (
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"
	"my-robot-backend/internal/models"
	"my-robot-backend/internal/services"
	"my-robot-backend/pkg/database"
)

type ContentCompletionScheduler struct {
	cron              *cron.Cron
	completionService *services.ContentCompletionService
	checkInterval     time.Duration
	taskName          string
}

func NewContentCompletionScheduler(completionService *services.ContentCompletionService, checkIntervalMinutes int) *ContentCompletionScheduler {
	taskName := "content_completion"

	scheduler := &ContentCompletionScheduler{
		cron:              cron.New(),
		completionService: completionService,
		checkInterval:     time.Duration(checkIntervalMinutes) * time.Minute,
		taskName:          taskName,
	}

	interval := fmt.Sprintf("@every %dm", checkIntervalMinutes)
	_, err := scheduler.cron.AddFunc(interval, scheduler.checkAndCompleteArticles)
	if err != nil {
		log.Printf("Failed to schedule content completion: %v", err)
	}

	return scheduler
}

func (s *ContentCompletionScheduler) Start() error {
	s.cron.Start()
	log.Printf("Content completion scheduler started (interval: %v)", s.checkInterval)
	s.initSchedulerTask()
	return nil
}

func (s *ContentCompletionScheduler) Stop() {
	s.cron.Stop()
	log.Println("Content completion scheduler stopped")
}

func (s *ContentCompletionScheduler) checkAndCompleteArticles() {
	var task models.SchedulerTask
	if err := database.DB.Where("name = ?", s.taskName).First(&task).Error; err != nil {
		log.Printf("Scheduler task not found: %v", err)
		return
	}

	now := time.Now().In(time.FixedZone("CST", 8*3600))
	task.Status = "running"
	task.LastExecutionTime = &now
	database.DB.Save(&task)

	startTime := time.Now()
	completedIDs, errors := s.completionService.AutoCompletePendingArticles(50)
	duration := time.Since(startTime).Seconds()

	task.LastExecutionDuration = &duration

	if len(errors) > 0 {
		task.Status = "error"
		task.FailedExecutions++
		task.ConsecutiveFailures++
		task.LastError = errors[0].Error()
		task.LastErrorTime = &now
		log.Printf("Content completion completed with errors: %d completed, %d failed", len(completedIDs), len(errors))
	} else {
		task.Status = "idle"
		task.SuccessfulExecutions++
		task.ConsecutiveFailures = 0
		task.LastError = ""
		log.Printf("Content completion completed successfully: %d articles processed", len(completedIDs))
	}

	task.TotalExecutions++
	task.LastExecutionResult = ""
	if len(completedIDs) > 0 {
		task.LastExecutionResult = "Completed articles: " + string(rune(len(completedIDs)))
	}

	nextRun := now.Add(s.checkInterval)
	task.NextExecutionTime = &nextRun
	database.DB.Save(&task)
}

func (s *ContentCompletionScheduler) initSchedulerTask() {
	var task models.SchedulerTask
	err := database.DB.Where("name = ?", s.taskName).First(&task).Error

	if err == nil {
		return
	}

	now := time.Now().In(time.FixedZone("CST", 8*3600))
	nextRun := now.Add(s.checkInterval)

	task = models.SchedulerTask{
		Name:                 s.taskName,
		Description:          "自动补全不完整的文章内容",
		CheckInterval:        int(s.checkInterval.Seconds()),
		Status:               "idle",
		NextExecutionTime:    &nextRun,
		TotalExecutions:      0,
		SuccessfulExecutions: 0,
		FailedExecutions:     0,
		ConsecutiveFailures:  0,
	}

	database.DB.Create(&task)
	log.Println("Content completion scheduler task initialized")
}
