package topicanalysis

import (
	"context"
	"time"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/logging"

	"gorm.io/gorm"
)

// ProcessPendingAdoptNarrowerTasks processes all pending adopt-narrower queue tasks.
// Intended to be called from a scheduled task before tree review.
// Returns the number of tasks processed.
func ProcessPendingAdoptNarrowerTasks() (int, error) {
	var tasks []models.AdoptNarrowerQueue
	if err := database.DB.
		Where("status = ?", models.AdoptNarrowerQueueStatusPending).
		Order("created_at ASC").
		Limit(50).
		Find(&tasks).Error; err != nil {
		return 0, err
	}

	if len(tasks) == 0 {
		return 0, nil
	}

	logging.Infof("adopt narrower batch: found %d pending tasks", len(tasks))

	processed := 0
	for _, task := range tasks {
		if err := adoptNarrowerAbstractChildren(context.Background(), task.AbstractTagID); err != nil {
			logging.Warnf("adopt narrower batch: failed for tag %d: %v", task.AbstractTagID, err)
			markAdoptNarrowerFailed(task.ID, err.Error())
			continue
		}

		now := time.Now()
		if err := database.DB.Model(&models.AdoptNarrowerQueue{}).
			Where("id = ?", task.ID).
			Updates(map[string]interface{}{
				"status":       models.AdoptNarrowerQueueStatusCompleted,
				"completed_at": now,
			}).Error; err != nil {
			logging.Warnf("adopt narrower batch: failed to mark task %d completed: %v", task.ID, err)
		}

		processed++
	}

	logging.Infof("adopt narrower batch: processed %d/%d tasks", processed, len(tasks))
	return processed, nil
}

// ProcessPendingAbstractTagUpdateTasks processes all pending abstract-tag-update queue tasks.
// Intended to be called from a scheduled task before tree review.
// Returns the number of tasks processed.
func ProcessPendingAbstractTagUpdateTasks() (int, error) {
	var tasks []models.AbstractTagUpdateQueue
	if err := database.DB.
		Where("status = ?", models.AbstractTagUpdateQueueStatusPending).
		Order("created_at ASC").
		Limit(50).
		Find(&tasks).Error; err != nil {
		return 0, err
	}

	if len(tasks) == 0 {
		return 0, nil
	}

	logging.Infof("abstract tag update batch: found %d pending tasks", len(tasks))

	svc := NewAbstractTagUpdateQueueService(nil)
	processed := 0
	for _, task := range tasks {
		if err := svc.refreshAbstractTag(task.AbstractTagID); err != nil {
			logging.Warnf("abstract tag update batch: failed for tag %d: %v", task.AbstractTagID, err)
			markAbstractTagUpdateFailed(task.ID, err.Error())
			continue
		}

		now := time.Now()
		if err := database.DB.Model(&models.AbstractTagUpdateQueue{}).
			Where("id = ?", task.ID).
			Updates(map[string]interface{}{
				"status":       models.AbstractTagUpdateQueueStatusCompleted,
				"completed_at": now,
			}).Error; err != nil {
			logging.Warnf("abstract tag update batch: failed to mark task %d completed: %v", task.ID, err)
		}

		processed++
	}

	logging.Infof("abstract tag update batch: processed %d/%d tasks", processed, len(tasks))
	return processed, nil
}

func markAdoptNarrowerFailed(taskID uint, errMsg string) {
	now := time.Now()
	if err := database.DB.Model(&models.AdoptNarrowerQueue{}).
		Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"status":        models.AdoptNarrowerQueueStatusFailed,
			"error_message": errMsg,
			"completed_at":  now,
			"retry_count":   gorm.Expr("retry_count + 1"),
		}).Error; err != nil {
		logging.Warnf("adopt narrower batch: failed to mark task %d failed: %v", taskID, err)
	}
}

func markAbstractTagUpdateFailed(taskID uint, errMsg string) {
	now := time.Now()
	if err := database.DB.Model(&models.AbstractTagUpdateQueue{}).
		Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"status":        models.AbstractTagUpdateQueueStatusFailed,
			"error_message": errMsg,
			"completed_at":  now,
			"retry_count":   gorm.Expr("retry_count + 1"),
		}).Error; err != nil {
		logging.Warnf("abstract tag update batch: failed to mark task %d failed: %v", taskID, err)
	}
}
