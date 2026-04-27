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
	"my-robot-backend/internal/domain/topicextraction"
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/logging"
	"my-robot-backend/internal/platform/tracing"
)

// TagHierarchyCleanupScheduler runs a 4-phase tag cleanup cycle: zombie cleanup, flat merge, hierarchy pruning, and tree review
type TagHierarchyCleanupScheduler struct {
	cron           *cron.Cron
	checkInterval  time.Duration
	isRunning      atomic.Bool
	executionMutex sync.Mutex
	isExecuting    atomic.Bool
}

// TagHierarchyCleanupRunSummary records the results of a cleanup run
type TagHierarchyCleanupRunSummary struct {
	TriggerSource          string `json:"trigger_source"`
	StartedAt              string `json:"started_at"`
	FinishedAt             string `json:"finished_at"`
	ZombieDeactivated      int    `json:"zombie_deactivated"`
	FlatMergesApplied      int    `json:"flat_merges_applied"`
	OrphanedRelations      int    `json:"orphaned_relations"`
	MultiParentFixed       int    `json:"multi_parent_fixed"`
	EmptyAbstracts         int    `json:"empty_abstracts"`
	AdoptNarrowerProcessed int    `json:"adopt_narrower_processed"`
	AbstractUpdateProcessed int   `json:"abstract_update_processed"`
	TreesReviewed          int    `json:"trees_reviewed"`
	MergesApplied          int    `json:"merges_applied"`
	MovesApplied           int    `json:"moves_applied"`
	GroupsCreated          int    `json:"tree_groups_created"`
	GroupsReused           int    `json:"tree_groups_reused"`
	DescriptionBackfilled  int    `json:"description_backfilled"`
	Errors                 int    `json:"errors"`
	Reason                 string `json:"reason"`
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
			"description":         "7-phase tag cleanup: zombie, flat merge, hierarchy pruning, adopt narrower, abstract update, tree review, description backfill",
			"check_interval":      int(s.checkInterval.Seconds()),
			"next_execution_time": &nextRun,
		}
		if task.Status == "" || task.Status == "success" || task.Status == "failed" || task.Status == "running" {
			updates["status"] = "idle"
		}
		database.DB.Model(&task).Updates(updates)
		return
	}

	task = models.SchedulerTask{
		Name:              "tag_hierarchy_cleanup",
		Description:       "7-phase tag cleanup: zombie, flat merge, hierarchy pruning, adopt narrower, abstract update, tree review, description backfill",
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

	logging.Infoln("Starting tag cleanup cycle (7-phase)")

	// Phase 1: Zombie tag cleanup (no LLM)
	zombieCount, err := topicanalysis.CleanupZombieTags(topicanalysis.ZombieTagCriteria{
		MinAgeDays: 7,
		Categories: []string{"event", "keyword", "person"},
	})
	if err != nil {
		logging.Errorf("Phase 1 zombie cleanup failed: %v", err)
		summary.Errors++
	} else {
		summary.ZombieDeactivated = zombieCount
		logging.Infof("Phase 1: deactivated %d zombie tags", zombieCount)
	}

	// Phase 2: Flat merge (LLM-assisted)
	phaseStart := time.Now()
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
	logging.Infof("Phase 2 completed in %v (processed %d)", time.Since(phaseStart), summary.FlatMergesApplied)

	// Phase 3: Hierarchy pruning
	orphaned, err := topicanalysis.CleanupOrphanedRelations()
	if err != nil {
		logging.Errorf("Phase 3 orphaned relations cleanup failed: %v", err)
		summary.Errors++
	}
	summary.OrphanedRelations = orphaned
	logging.Infof("Phase 3: removed %d orphaned relations", orphaned)

	resolved, resolveErrors, err := topicanalysis.CleanupMultiParentConflicts()
	if err != nil {
		logging.Errorf("Phase 3 multi-parent cleanup failed: %v", err)
		summary.Errors++
	}
	summary.MultiParentFixed = resolved
	summary.Errors += len(resolveErrors)
	for _, errMsg := range resolveErrors {
		logging.Warnf("Phase 3 multi-parent: %s", errMsg)
	}
	logging.Infof("Phase 3: resolved %d multi-parent conflicts", resolved)

	emptied, err := topicanalysis.CleanupEmptyAbstractNodes()
	if err != nil {
		logging.Errorf("Phase 3 empty abstract cleanup failed: %v", err)
		summary.Errors++
	}
	summary.EmptyAbstracts = emptied
	logging.Infof("Phase 3: deactivated %d empty abstract tags", emptied)

	// Phase 4: Adopt narrower queue processing (before tree review)
	phaseStart = time.Now()
	adopted, err := topicanalysis.ProcessPendingAdoptNarrowerTasks()
	if err != nil {
		logging.Errorf("Phase 4 adopt narrower failed: %v", err)
		summary.Errors++
	} else {
		summary.AdoptNarrowerProcessed = adopted
		logging.Infof("Phase 4: processed %d adopt-narrower tasks", adopted)
	}
	logging.Infof("Phase 4 completed in %v (processed %d)", time.Since(phaseStart), adopted)

	// Phase 5: Abstract tag update queue processing (label/description refresh)
	phaseStart = time.Now()
	updated, err := topicanalysis.ProcessPendingAbstractTagUpdateTasks()
	if err != nil {
		logging.Errorf("Phase 5 abstract tag update failed: %v", err)
		summary.Errors++
	} else {
		summary.AbstractUpdateProcessed = updated
		logging.Infof("Phase 5: processed %d abstract-tag-update tasks", updated)
	}
	logging.Infof("Phase 5 completed in %v (processed %d)", time.Since(phaseStart), updated)

	// Phase 6: Tree review
	phaseStart = time.Now()
	for _, category := range []string{"event", "keyword", "person"} {
		reviewResult, reviewErr := topicanalysis.ReviewHierarchyTrees(category, 14, nil)
		if reviewErr != nil {
			logging.Errorf("Phase 6 tree review failed for %s: %v", category, reviewErr)
			summary.Errors++
			continue
		}
		summary.TreesReviewed += reviewResult.TreesReviewed
		summary.MergesApplied += reviewResult.MergesApplied
		summary.MovesApplied += reviewResult.MovesApplied
		summary.GroupsCreated += reviewResult.GroupsCreated
		summary.GroupsReused += reviewResult.GroupsReused
		summary.Errors += len(reviewResult.Errors)
		for _, errMsg := range reviewResult.Errors {
			logging.Warnf("Phase 6 %s: %s", category, errMsg)
		}
		logging.Infof("Phase 6 (%s): reviewed %d trees, %d merges, %d moves, %d groups created, %d groups reused", category, reviewResult.TreesReviewed, reviewResult.MergesApplied, reviewResult.MovesApplied, reviewResult.GroupsCreated, reviewResult.GroupsReused)
	}
	logging.Infof("Phase 6 completed in %v (processed %d)", time.Since(phaseStart), summary.TreesReviewed)

	// Phase 7: Description backfill for tags missing descriptions
	phaseStart = time.Now()
	backfilled, err := topicextraction.BackfillMissingDescriptions()
	if err != nil {
		logging.Errorf("Phase 7 description backfill failed: %v", err)
		summary.Errors++
	} else {
		summary.DescriptionBackfilled = backfilled
		logging.Infof("Phase 7: triggered description backfill for %d tags", backfilled)
	}
	logging.Infof("Phase 7 completed in %v (processed %d)", time.Since(phaseStart), backfilled)

	summary.FinishedAt = time.Now().Format(time.RFC3339)
	summary.Reason = fmt.Sprintf("zombie=%d, flat_merges=%d, orphaned_rels=%d, multi_parent=%d, empty_abstracts=%d, adopt_narrower=%d, abstract_update=%d, trees_reviewed=%d, merges=%d, moves=%d, groups_created=%d, groups_reused=%d, desc_backfilled=%d",
		summary.ZombieDeactivated, summary.FlatMergesApplied, summary.OrphanedRelations, summary.MultiParentFixed, summary.EmptyAbstracts, summary.AdoptNarrowerProcessed, summary.AbstractUpdateProcessed, summary.TreesReviewed, summary.MergesApplied, summary.MovesApplied, summary.GroupsCreated, summary.GroupsReused, summary.DescriptionBackfilled)

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
		Description:       "7-phase tag cleanup: zombie, flat merge, hierarchy pruning, adopt narrower, abstract update, tree review, description backfill",
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
