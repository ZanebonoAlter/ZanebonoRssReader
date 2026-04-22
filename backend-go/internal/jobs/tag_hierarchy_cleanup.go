package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/robfig/cron/v3"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/domain/topicanalysis"
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/logging"
	"my-robot-backend/internal/platform/tracing"
)

// TagHierarchyCleanupScheduler cleans up deep tag hierarchies by merging duplicates and creating abstract tags
type TagHierarchyCleanupScheduler struct {
	cron           *cron.Cron
	checkInterval  time.Duration
	isRunning      atomic.Bool
	executionMutex sync.Mutex
	isExecuting    atomic.Bool
}

// TagHierarchyCleanupRunSummary records the results of a cleanup run
type TagHierarchyCleanupRunSummary struct {
	TriggerSource    string `json:"trigger_source"`
	StartedAt        string `json:"started_at"`
	FinishedAt       string `json:"finished_at"`
	TreesProcessed   int    `json:"trees_processed"`
	TagsProcessed    int    `json:"tags_processed"`
	MergesApplied    int    `json:"merges_applied"`
	FlatMergesApplied int   `json:"flat_merges_applied"`
	AbstractsCreated int    `json:"abstracts_created"`
	Errors           int    `json:"errors"`
	Reason           string `json:"reason"`
}

// NewTagHierarchyCleanupScheduler creates a new scheduler
func NewTagHierarchyCleanupScheduler(checkInterval int) *TagHierarchyCleanupScheduler {
	return &TagHierarchyCleanupScheduler{
		cron:          cron.New(),
		checkInterval: time.Duration(checkInterval) * time.Second,
	}
}

// Start begins the scheduler
func (s *TagHierarchyCleanupScheduler) Start() error {
	if s.isRunning.Load() {
		return fmt.Errorf("tag-hierarchy-cleanup scheduler already running")
	}

	s.initSchedulerTask()
	scheduleExpr := fmt.Sprintf("@every %ds", int64(s.checkInterval.Seconds()))
	if _, err := s.cron.AddFunc(scheduleExpr, s.cleanupHierarchy); err != nil {
		return fmt.Errorf("failed to schedule tag-hierarchy-cleanup: %w", err)
	}

	s.cron.Start()
	s.isRunning.Store(true)
	logging.Infof("Tag-hierarchy-cleanup scheduler started with interval: %v", s.checkInterval)
	return nil
}

// Stop halts the scheduler
func (s *TagHierarchyCleanupScheduler) Stop() {
	if !s.isRunning.Load() {
		return
	}
	s.cron.Stop()
	s.isRunning.Store(false)
	logging.Infoln("Tag-hierarchy-cleanup scheduler stopped")
}

// UpdateInterval changes the check interval
func (s *TagHierarchyCleanupScheduler) UpdateInterval(interval int) error {
	if interval <= 0 {
		return fmt.Errorf("interval must be positive")
	}

	wasRunning := s.isRunning.Load()
	if wasRunning {
		s.Stop()
	}

	s.cron = cron.New()
	s.checkInterval = time.Duration(interval) * time.Second

	if wasRunning {
		return s.Start()
	}

	var task models.SchedulerTask
	if err := database.DB.Where("name = ?", "tag_hierarchy_cleanup").First(&task).Error; err == nil {
		nextRun := time.Now().Add(s.checkInterval)
		database.DB.Model(&task).Updates(map[string]interface{}{
			"check_interval":      interval,
			"next_execution_time": &nextRun,
		})
	}

	return nil
}

// ResetStats resets scheduler statistics
func (s *TagHierarchyCleanupScheduler) ResetStats() error {
	var task models.SchedulerTask
	if err := database.DB.Where("name = ?", "tag_hierarchy_cleanup").First(&task).Error; err != nil {
		return err
	}

	nextRun := time.Now().Add(s.checkInterval)
	updates := map[string]interface{}{
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
		"next_execution_time":     &nextRun,
	}

	return database.DB.Model(&task).Updates(updates).Error
}

// TriggerNow manually triggers a cleanup run
func (s *TagHierarchyCleanupScheduler) TriggerNow() map[string]interface{} {
	if !s.executionMutex.TryLock() {
		return map[string]interface{}{
			"accepted":    false,
			"started":     false,
			"reason":      "already_running",
			"message":     "标签层级清理正在执行中，请稍后再试。",
			"status_code": http.StatusConflict,
		}
	}

	s.isExecuting.Store(true)
	go func() {
		defer s.executionMutex.Unlock()
		defer func() {
			s.isExecuting.Store(false)
			if r := recover(); r != nil {
				logging.Errorf("PANIC in manual tag-hierarchy-cleanup trigger: %v", r)
				s.updateSchedulerStatus("idle", fmt.Sprintf("Panic: %v", r), nil, nil)
			}
		}()
		s.runCleanupCycle("manual")
	}()

	return map[string]interface{}{
		"accepted": true,
		"started":  true,
		"reason":   "manual_run_started",
		"message":  "标签层级清理已经开始运行。",
	}
}

func (s *TagHierarchyCleanupScheduler) initSchedulerTask() {
	var task models.SchedulerTask
	now := time.Now()
	nextRun := now.Add(s.checkInterval)

	if err := database.DB.Where("name = ?", "tag_hierarchy_cleanup").First(&task).Error; err == nil {
		updates := map[string]interface{}{
			"description":         "Auto-cleanup deep tag hierarchies by merging duplicates and creating abstract tags",
			"check_interval":      int(s.checkInterval.Seconds()),
			"next_execution_time": &nextRun,
		}
		if task.Status == "" || task.Status == "success" || task.Status == "failed" {
			updates["status"] = "idle"
		}
		database.DB.Model(&task).Updates(updates)
		return
	}

	task = models.SchedulerTask{
		Name:              "tag_hierarchy_cleanup",
		Description:       "Auto-cleanup deep tag hierarchies by merging duplicates and creating abstract tags",
		CheckInterval:     int(s.checkInterval.Seconds()),
		Status:            "idle",
		NextExecutionTime: &nextRun,
	}
	database.DB.Create(&task)
}

func (s *TagHierarchyCleanupScheduler) cleanupHierarchy() {
	tracing.TraceSchedulerTick("tag_hierarchy_cleanup", "cron", func(ctx context.Context) {
		_ = ctx
		if !s.executionMutex.TryLock() {
			logging.Infoln("Tag hierarchy cleanup already in progress, skipping this cycle")
			return
		}
		s.isExecuting.Store(true)
		defer func() {
			s.executionMutex.Unlock()
			s.isExecuting.Store(false)
			if r := recover(); r != nil {
				logging.Errorf("PANIC in cleanupHierarchy: %v", r)
				s.updateSchedulerStatus("idle", fmt.Sprintf("Panic: %v", r), nil, nil)
			}
		}()

		s.runCleanupCycle("scheduled")
	})
}

func (s *TagHierarchyCleanupScheduler) runCleanupCycle(triggerSource string) {
	startTime := time.Now()
	summary := &TagHierarchyCleanupRunSummary{
		TriggerSource: triggerSource,
		StartedAt:     startTime.Format(time.RFC3339),
	}
	s.updateSchedulerStatus("running", "", nil, nil)

	logging.Infoln("Starting tag cleanup cycle (3-phase)")

	// Phase 1: Zombie tag cleanup (no LLM)
	zombieCount, err := topicanalysis.CleanupZombieTags(topicanalysis.ZombieTagCriteria{
		MinAgeDays: 7,
		Categories: []string{"event", "keyword", "person"},
	})
	if err != nil {
		logging.Errorf("Phase 1 zombie cleanup failed: %v", err)
		summary.Errors++
	} else {
		logging.Infof("Phase 1: deactivated %d zombie tags", zombieCount)
	}

	// Phase 2: Flat merge (LLM-assisted)
	for _, category := range []string{"event", "keyword"} {
		merged, mergeErrors, err := topicanalysis.ExecuteFlatMerge(category, 50)
		if err != nil {
			logging.Errorf("Phase 2 flat merge failed for %s: %v", category, err)
			summary.Errors++
			continue
		}
		summary.FlatMergesApplied += merged
		summary.Errors += len(mergeErrors)
		for _, e := range mergeErrors {
			logging.Warnf("Phase 2 %s: %s", category, e)
		}
		logging.Infof("Phase 2 (%s): %d merges applied", category, merged)
	}

	// Phase 3: Hierarchy pruning
	orphaned, err := topicanalysis.CleanupOrphanedRelations()
	if err != nil {
		logging.Errorf("Phase 3 orphaned relations cleanup failed: %v", err)
		summary.Errors++
	}
	logging.Infof("Phase 3: removed %d orphaned relations", orphaned)

	resolved, _, err := topicanalysis.CleanupMultiParentConflicts()
	if err != nil {
		logging.Errorf("Phase 3 multi-parent cleanup failed: %v", err)
		summary.Errors++
	}
	logging.Infof("Phase 3: resolved %d multi-parent conflicts", resolved)

	emptied, err := topicanalysis.CleanupEmptyAbstractNodes()
	if err != nil {
		logging.Errorf("Phase 3 empty abstract cleanup failed: %v", err)
		summary.Errors++
	}
	logging.Infof("Phase 3: deactivated %d empty abstract tags", emptied)

	// Phase 3b: Tree-based cleanup with lowered threshold (original logic)
	categories := []string{"event", "keyword"}
	for _, category := range categories {
		forest, err := topicanalysis.BuildTagForest(category)
		if err != nil {
			logging.Errorf("BuildTagForest failed for %s: %v", category, err)
			summary.Errors++
			continue
		}
		if len(forest) == 0 {
			continue
		}
		logging.Infof("Phase 3b: found %d trees for %s", len(forest), category)
		for _, tree := range forest {
			result, err := topicanalysis.ProcessTree(tree)
			if err != nil {
				logging.Errorf("ProcessTree failed for %s: %v", tree.Tag.Label, err)
				summary.Errors++
				continue
			}
			summary.TreesProcessed++
			summary.TagsProcessed += result.TagsProcessed
			summary.MergesApplied += result.MergesApplied
			summary.AbstractsCreated += result.AbstractsCreated
			summary.Errors += len(result.Errors)
			for _, errMsg := range result.Errors {
				logging.Warnf("Tree %s: %s", result.TreeRootLabel, errMsg)
			}
		}
	}

	summary.FinishedAt = time.Now().Format(time.RFC3339)
	summary.Reason = fmt.Sprintf("zombie=%d, flat_merges=%d, orphaned_rels=%d, multi_parent=%d, empty_abstracts=%d, trees=%d, tree_merges=%d, tree_abstracts=%d",
		zombieCount, summary.FlatMergesApplied, orphaned, resolved, emptied,
		summary.TreesProcessed, summary.MergesApplied, summary.AbstractsCreated)

	logging.Infof("Tag cleanup cycle completed: %s", summary.Reason)

	if summary.Errors > 0 {
		s.updateSchedulerStatus("success_with_errors", "", &startTime, summary)
	} else {
		s.updateSchedulerStatus("success", "", &startTime, summary)
	}
}

func (s *TagHierarchyCleanupScheduler) updateSchedulerStatus(status, lastError string, startTime *time.Time, summary *TagHierarchyCleanupRunSummary) {
	now := time.Now()
	nextExecution := now.Add(s.checkInterval)
	resultJSON := ""
	if summary != nil {
		if encoded, err := json.Marshal(summary); err == nil {
			resultJSON = string(encoded)
		}
	}

	updates := map[string]interface{}{
		"status":              status,
		"last_error":          lastError,
		"next_execution_time": &nextExecution,
	}
	if startTime != nil {
		duration := float64(time.Since(*startTime).Seconds())
		updates["last_execution_time"] = &now
		updates["last_execution_duration"] = &duration
		updates["last_execution_result"] = resultJSON
	}

	var task models.SchedulerTask
	if err := database.DB.Where("name = ?", "tag_hierarchy_cleanup").First(&task).Error; err == nil {
		updates["total_executions"] = task.TotalExecutions
		updates["successful_executions"] = task.SuccessfulExecutions
		updates["failed_executions"] = task.FailedExecutions
		updates["consecutive_failures"] = task.ConsecutiveFailures

		if startTime != nil {
			updates["total_executions"] = task.TotalExecutions + 1
			if status == "success" || status == "success_with_errors" {
				updates["successful_executions"] = task.SuccessfulExecutions + 1
				updates["consecutive_failures"] = 0
			} else if status == "failed" {
				updates["failed_executions"] = task.FailedExecutions + 1
				updates["consecutive_failures"] = task.ConsecutiveFailures + 1
				updates["last_error_time"] = &now
			}
		}

		database.DB.Model(&task).Updates(updates)
		return
	}

	task = models.SchedulerTask{
		Name:              "tag_hierarchy_cleanup",
		Description:       "Auto-cleanup deep tag hierarchies by merging duplicates and creating abstract tags",
		CheckInterval:     int(s.checkInterval.Seconds()),
		Status:            status,
		LastError:         lastError,
		NextExecutionTime: &nextExecution,
	}
	if startTime != nil {
		duration := float64(time.Since(*startTime).Seconds())
		task.LastExecutionTime = &now
		task.LastExecutionDuration = &duration
		task.LastExecutionResult = resultJSON
		task.TotalExecutions = 1
		if status == "success" || status == "success_with_errors" {
			task.SuccessfulExecutions = 1
		} else if status == "failed" {
			task.FailedExecutions = 1
			task.ConsecutiveFailures = 1
			task.LastErrorTime = &now
		}
	}
	database.DB.Create(&task)
}

// GetStatus returns the current scheduler status
func (s *TagHierarchyCleanupScheduler) GetStatus() SchedulerStatusResponse {
	entries := s.cron.Entries()
	var nextRun int64
	if len(entries) > 0 {
		nextRun = entries[0].Next.Unix()
	}

	var task models.SchedulerTask
	err := database.DB.Where("name = ?", "tag_hierarchy_cleanup").First(&task).Error

	status := SchedulerStatusResponse{
		Name: "Tag Hierarchy Cleanup",
		Status: func() string {
			if s.isExecuting.Load() {
				return "running"
			}
			if s.isRunning.Load() {
				return "idle"
			}
			return "stopped"
		}(),
		CheckInterval: int64(s.checkInterval.Seconds()),
		NextRun:       nextRun,
		IsExecuting:   s.isExecuting.Load(),
	}
	if err == nil && task.NextExecutionTime != nil {
		status.NextRun = task.NextExecutionTime.Unix()
	}
	return status
}
