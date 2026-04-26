package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/domain/topicanalysis"
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/logging"
	"my-robot-backend/internal/platform/tracing"
)

type AutoTagMergeScheduler struct {
	cron           *cron.Cron
	checkInterval  time.Duration
	isRunning      bool
	executionMutex sync.Mutex
	isExecuting    bool
}

type AutoTagMergeRunSummary struct {
	TriggerSource string               `json:"trigger_source"`
	StartedAt     string               `json:"started_at"`
	FinishedAt    string               `json:"finished_at"`
	TotalPairs    int                  `json:"total_pairs"`
	MergedCount   int                  `json:"merged_count"`
	SkippedCount  int                  `json:"skipped_count"`
	FailedCount   int                  `json:"failed_count"`
	Reason        string               `json:"reason"`
	MergeDetails  []AutoTagMergeDetail `json:"merge_details,omitempty"`
}

type AutoTagMergeDetail struct {
	SourceTagID    uint    `json:"source_tag_id"`
	SourceLabel    string  `json:"source_label"`
	TargetTagID    uint    `json:"target_tag_id"`
	TargetLabel    string  `json:"target_label"`
	Similarity     float64 `json:"similarity"`
	SourceArticles int     `json:"source_articles"`
	TargetArticles int     `json:"target_articles"`
}

func NewAutoTagMergeScheduler(checkInterval int) *AutoTagMergeScheduler {
	return &AutoTagMergeScheduler{
		cron:          cron.New(),
		checkInterval: time.Duration(checkInterval) * time.Second,
		isRunning:     false,
	}
}

func (s *AutoTagMergeScheduler) Start() error {
	if s.isRunning {
		return fmt.Errorf("auto-tag-merge scheduler already running")
	}
	s.initSchedulerTask()

	scheduleExpr := fmt.Sprintf("@every %ds", int64(s.checkInterval.Seconds()))
	if _, err := s.cron.AddFunc(scheduleExpr, s.scanAndMergeTags); err != nil {
		return fmt.Errorf("failed to schedule auto-tag-merge: %w", err)
	}

	s.cron.Start()
	s.isRunning = true
	logging.Infof("Auto-tag-merge scheduler started with interval: %v", s.checkInterval)

	return nil
}

func (s *AutoTagMergeScheduler) Stop() {
	if !s.isRunning {
		return
	}

	s.cron.Stop()
	s.isRunning = false
	logging.Infoln("Auto-tag-merge scheduler stopped")
}

func (s *AutoTagMergeScheduler) UpdateInterval(interval int) error {
	if interval <= 0 {
		return fmt.Errorf("interval must be positive")
	}

	wasRunning := s.isRunning
	if wasRunning {
		s.Stop()
	}

	s.cron = cron.New()
	s.checkInterval = time.Duration(interval) * time.Second

	if wasRunning {
		return s.Start()
	}

	var task models.SchedulerTask
	if err := database.DB.Where("name = ?", "auto_tag_merge").First(&task).Error; err == nil {
		nextRun := time.Now().Add(s.checkInterval)
		database.DB.Model(&task).Updates(map[string]interface{}{
			"check_interval":      interval,
			"next_execution_time": &nextRun,
		})
	}

	return nil
}

func (s *AutoTagMergeScheduler) ResetStats() error {
	var task models.SchedulerTask
	if err := database.DB.Where("name = ?", "auto_tag_merge").First(&task).Error; err != nil {
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

func (s *AutoTagMergeScheduler) TriggerNow() map[string]interface{} {
	if !s.executionMutex.TryLock() {
		return map[string]interface{}{
			"accepted":    false,
			"started":     false,
			"reason":      "already_running",
			"message":     "标签合并正在执行中，请稍后再试。",
			"status_code": http.StatusConflict,
		}
	}

	s.isExecuting = true
	go func() {
		defer s.executionMutex.Unlock()
		defer func() {
			s.isExecuting = false
			if r := recover(); r != nil {
				logging.Errorf("PANIC in manual auto-tag-merge trigger: %v", r)
				s.updateSchedulerStatus("idle", fmt.Sprintf("Panic: %v", r), nil, nil)
			}
		}()
		s.runMergeCycle("manual")
	}()

	return map[string]interface{}{
		"accepted": true,
		"started":  true,
		"reason":   "manual_run_started",
		"message":  "标签合并已经开始运行。",
	}
}

func (s *AutoTagMergeScheduler) initSchedulerTask() {
	var task models.SchedulerTask
	now := time.Now()
	nextRun := now.Add(s.checkInterval)

	if err := database.DB.Where("name = ?", "auto_tag_merge").First(&task).Error; err == nil {
		updates := map[string]interface{}{
			"description":         "Auto-merge similar tags based on embedding similarity",
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
		Name:              "auto_tag_merge",
		Description:       "Auto-merge similar tags based on embedding similarity",
		CheckInterval:     int(s.checkInterval.Seconds()),
		Status:            "idle",
		NextExecutionTime: &nextRun,
	}
	database.DB.Create(&task)
}

func (s *AutoTagMergeScheduler) scanAndMergeTags() {
	tracing.TraceSchedulerTick("auto_tag_merge", "cron", func(ctx context.Context) {
		if !s.executionMutex.TryLock() {
			logging.Infoln("Tag merge already in progress, skipping this cycle")
			return
		}
		s.isExecuting = true
		defer func() {
			s.executionMutex.Unlock()
			s.isExecuting = false
			if r := recover(); r != nil {
				logging.Errorf("PANIC in scanAndMergeTags: %v", r)
				s.updateSchedulerStatus("idle", fmt.Sprintf("Panic: %v", r), nil, nil)
			}
		}()

		s.runMergeCycle("scheduled")
	})
}

func (s *AutoTagMergeScheduler) runMergeCycle(triggerSource string) {
	logging.Infoln("Starting auto-tag-merge cycle")
	startTime := time.Now()
	summary := &AutoTagMergeRunSummary{
		TriggerSource: triggerSource,
		StartedAt:     startTime.Format(time.RFC3339),
	}
	s.updateSchedulerStatus("running", "", nil, nil)

	// Distance threshold: cosine distance < (1 - HighSimilarity) means similarity >= HighSimilarity
	distanceThreshold := 1.0 - topicanalysis.DefaultThresholds.HighSimilarity

	// Find similar tag pairs using pgvector cross-join
	type similarPair struct {
		SourceID    uint    `gorm:"column:source_id"`
		SourceLabel string  `gorm:"column:source_label"`
		TargetID    uint    `gorm:"column:target_id"`
		TargetLabel string  `gorm:"column:target_label"`
		Category    string  `gorm:"column:category"`
		Distance    float64 `gorm:"column:distance"`
	}

	var pairs []similarPair
	query := `
		SELECT
			t1.id AS source_id, t1.label AS source_label,
			t2.id AS target_id, t2.label AS target_label,
			t1.category,
			e1.embedding <=> e2.embedding AS distance
		FROM topic_tag_embeddings e1
		JOIN topic_tags t1 ON t1.id = e1.topic_tag_id
		JOIN topic_tag_embeddings e2 ON e2.topic_tag_id > e1.topic_tag_id
		JOIN topic_tags t2 ON t2.id = e2.topic_tag_id
		WHERE (t1.status = 'active' OR t1.status = '' OR t1.status IS NULL)
		  AND (t2.status = 'active' OR t2.status = '' OR t2.status IS NULL)
		  AND t1.category = t2.category
		  AND e1.embedding <=> e2.embedding < ?
		ORDER BY e1.embedding <=> e2.embedding ASC
		LIMIT 50
	`
	if err := database.DB.Raw(query, distanceThreshold).Scan(&pairs).Error; err != nil {
		logging.Errorf("Error querying similar tag pairs: %v", err)
		summary.Reason = "query_failed"
		summary.FinishedAt = time.Now().Format(time.RFC3339)
		s.updateSchedulerStatus("idle", err.Error(), &startTime, summary)
		return
	}

	summary.TotalPairs = len(pairs)

	if len(pairs) == 0 {
		logging.Infoln("No similar tag pairs found, skipping")
		summary.Reason = "no_similar_pairs"
		summary.FinishedAt = time.Now().Format(time.RFC3339)
		s.updateSchedulerStatus("idle", "", &startTime, summary)
		return
	}

	logging.Infof("Found %d similar tag pairs to evaluate", len(pairs))

	for _, pair := range pairs {
		// Load both tags to verify they are still active
		var tag1, tag2 models.TopicTag
		if err := database.DB.First(&tag1, pair.SourceID).Error; err != nil {
			logging.Warnf("Skipping pair (%d, %d): source tag not found", pair.SourceID, pair.TargetID)
			summary.SkippedCount++
			continue
		}
		if err := database.DB.First(&tag2, pair.TargetID).Error; err != nil {
			logging.Warnf("Skipping pair (%d, %d): target tag not found", pair.SourceID, pair.TargetID)
			summary.SkippedCount++
			continue
		}

		// Check both tags are still active (not merged by a previous pair in this cycle)
		if tag1.Status == "merged" || tag2.Status == "merged" {
			logging.Infoln(fmt.Sprintf("Skipping pair (%d, %d): one or both tags already merged", pair.SourceID, pair.TargetID))
			summary.SkippedCount++
			continue
		}

		// Count articles for each tag
		var count1, count2 int64
		database.DB.Model(&models.ArticleTopicTag{}).Where("topic_tag_id = ?", tag1.ID).Count(&count1)
		database.DB.Model(&models.ArticleTopicTag{}).Where("topic_tag_id = ?", tag2.ID).Count(&count2)

		// Determine source and target: tag with more articles is the target (keep it)
		var sourceID, targetID uint
		var sourceLabel, targetLabel string
		var sourceArticles, targetArticles int

		if count2 > count1 {
			// tag2 has more articles → keep tag2 as target, merge tag1 into it
			sourceID = tag1.ID
			sourceLabel = tag1.Label
			targetID = tag2.ID
			targetLabel = tag2.Label
			sourceArticles = int(count1)
			targetArticles = int(count2)
		} else if count1 > count2 {
			// tag1 has more articles → keep tag1 as target, merge tag2 into it
			sourceID = tag2.ID
			sourceLabel = tag2.Label
			targetID = tag1.ID
			targetLabel = tag1.Label
			sourceArticles = int(count2)
			targetArticles = int(count1)
		} else {
			// Equal article count → use smaller ID as source (deterministic)
			if tag1.ID < tag2.ID {
				sourceID = tag1.ID
				sourceLabel = tag1.Label
				targetID = tag2.ID
				targetLabel = tag2.Label
			} else {
				sourceID = tag2.ID
				sourceLabel = tag2.Label
				targetID = tag1.ID
				targetLabel = tag1.Label
			}
			sourceArticles = int(count1)
			targetArticles = int(count2)
		}

		similarity := 1.0 - pair.Distance

		logging.Infof("[AUTO-MERGE] merged tag '%s' (id=%d, %d articles) into '%s' (id=%d, %d articles), similarity=%.4f",
			sourceLabel, sourceID, sourceArticles,
			targetLabel, targetID, targetArticles,
			similarity,
		)

		if err := topicanalysis.MergeTags(sourceID, targetID); err != nil {
			logging.Errorf("Failed to merge tag %d into %d: %v", sourceID, targetID, err)
			summary.FailedCount++
			continue
		}

		summary.MergedCount++
		summary.MergeDetails = append(summary.MergeDetails, AutoTagMergeDetail{
			SourceTagID:    sourceID,
			SourceLabel:    sourceLabel,
			TargetTagID:    targetID,
			TargetLabel:    targetLabel,
			Similarity:     similarity,
			SourceArticles: sourceArticles,
			TargetArticles: targetArticles,
		})
	}

	duration := time.Since(startTime)
	logging.Infof("Auto-tag-merge cycle completed: %d merged, %d skipped, %d failed in %v",
		summary.MergedCount, summary.SkippedCount, summary.FailedCount, duration)

	summary.FinishedAt = time.Now().Format(time.RFC3339)
	if summary.MergedCount > 0 {
		summary.Reason = "tags_merged"
	} else if summary.FailedCount > 0 {
		summary.Reason = "merge_failed"
	} else {
		summary.Reason = "no_merges_needed"
	}
	s.updateSchedulerStatus("idle", "", &startTime, summary)
}

func (s *AutoTagMergeScheduler) updateSchedulerStatus(status, lastError string, startTime *time.Time, summary *AutoTagMergeRunSummary) {
	var task models.SchedulerTask
	err := database.DB.Where("name = ?", "auto_tag_merge").First(&task).Error

	now := time.Now()
	resultJSON := ""
	if summary != nil {
		if data, marshalErr := json.Marshal(summary); marshalErr == nil {
			resultJSON = string(data)
		}
	}

	if err == nil {
		task.Status = status
		task.LastError = lastError
		if lastError != "" {
			errTime := now
			task.LastErrorTime = &errTime
		}
		if resultJSON != "" {
			task.LastExecutionResult = resultJSON
		}

		if startTime != nil {
			duration := time.Since(*startTime)
			durationFloat := float64(duration.Seconds())
			task.LastExecutionDuration = &durationFloat
			task.LastExecutionTime = &now

			if status == "idle" && lastError == "" {
				task.SuccessfulExecutions++
				task.ConsecutiveFailures = 0
			} else if lastError != "" {
				task.FailedExecutions++
				task.ConsecutiveFailures++
			}
			task.TotalExecutions++

			nextExecution := now.Add(s.checkInterval)
			task.NextExecutionTime = &nextExecution
		}

		database.DB.Save(&task)
	} else {
		nextExecution := now.Add(s.checkInterval)
		var durationFloat *float64
		if startTime != nil {
			d := time.Since(*startTime)
			df := float64(d.Seconds())
			durationFloat = &df
		}

		task = models.SchedulerTask{
			Name:                  "auto_tag_merge",
			Description:           "Auto-merge similar tags based on embedding similarity",
			CheckInterval:         int(s.checkInterval.Seconds()),
			Status:                status,
			LastError:             lastError,
			NextExecutionTime:     &nextExecution,
			LastExecutionDuration: durationFloat,
			LastExecutionResult:   resultJSON,
		}

		if startTime != nil {
			task.LastExecutionTime = &now
		}

		database.DB.Create(&task)
	}
}

func (s *AutoTagMergeScheduler) GetStatus() SchedulerStatusResponse {
	entries := s.cron.Entries()

	var nextRun int64
	if len(entries) > 0 {
		nextRun = entries[0].Next.Unix()
	}

	var task models.SchedulerTask
	err := database.DB.Where("name = ?", "auto_tag_merge").First(&task).Error

	status := SchedulerStatusResponse{
		Name: "Auto Tag Merge",
		Status: func() string {
			if s.isExecuting {
				return "running"
			}
			if s.isRunning {
				return "idle"
			}
			return "stopped"
		}(),
		CheckInterval: int64(s.checkInterval.Seconds()),
		NextRun:       nextRun,
		IsExecuting:   s.isExecuting,
	}

	if err == nil {
		if task.NextExecutionTime != nil {
			status.NextRun = task.NextExecutionTime.Unix()
		}
	}

	return status
}

func (s *AutoTagMergeScheduler) IsExecuting() bool {
	return s.isExecuting
}

func parseAutoTagMergeRunSummary(task models.SchedulerTask) *AutoTagMergeRunSummary {
	if task.LastExecutionResult == "" {
		return nil
	}

	var summary AutoTagMergeRunSummary
	if err := json.Unmarshal([]byte(task.LastExecutionResult), &summary); err != nil {
		return nil
	}

	return &summary
}
