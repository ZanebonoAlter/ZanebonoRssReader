package summaries

import (
	"fmt"
	"time"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

func MarkArticlesWithFeedSummary(articleIDs []uint, summary *models.AISummary) error {
	if summary == nil || summary.ID == 0 || len(articleIDs) == 0 {
		return nil
	}

	summarizedAt := summary.CreatedAt
	if summarizedAt.IsZero() {
		summarizedAt = time.Now()
	}

	if err := database.DB.Model(&models.Article{}).
		Where("id IN ?", articleIDs).
		Updates(map[string]any{
			"feed_summary_id":           summary.ID,
			"feed_summary_generated_at": summarizedAt,
		}).Error; err != nil {
		return fmt.Errorf("mark articles with feed summary %d: %w", summary.ID, err)
	}

	return nil
}
