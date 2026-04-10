package database

import (
	"fmt"

	"gorm.io/gorm"
)

// First-batch Postgres bootstrap keeps the current payload shape intact while
// stabilizing the database cutover. ai_summaries.articles stays denormalized,
// and JSON-ish payload_json / metadata fields remain TEXT for now.
func bootstrapPostgresSchema(db *gorm.DB) error {
	if err := autoMigrateModels(db); err != nil {
		return fmt.Errorf("auto migrate postgres schema: %w", err)
	}

	if db.Dialector.Name() == "postgres" {
		for _, statement := range postgresColumnAdjustmentStatements() {
			if err := db.Exec(statement).Error; err != nil {
				return fmt.Errorf("apply postgres column adjustment %q: %w", statement, err)
			}
		}
	}

	for _, statement := range postgresBaselineIndexStatements() {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("apply postgres baseline index %q: %w", statement, err)
		}
	}

	return nil
}

func postgresColumnAdjustmentStatements() []string {
	return []string{
		"ALTER TABLE feeds ALTER COLUMN icon TYPE VARCHAR(1000)",
		"ALTER TABLE ai_summary_feeds ALTER COLUMN feed_icon TYPE VARCHAR(1000)",
	}
}

func postgresBaselineIndexStatements() []string {
	return []string{
		"CREATE INDEX IF NOT EXISTS idx_articles_feed_created_at ON articles(feed_id, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_articles_pub_date ON articles(pub_date)",
		"CREATE INDEX IF NOT EXISTS idx_ai_summaries_feed_created_at ON ai_summaries(feed_id, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_ai_summaries_category_created_at ON ai_summaries(category_id, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_article_topic_tags_topic_article ON article_topic_tags(topic_tag_id, article_id)",
		"CREATE INDEX IF NOT EXISTS idx_ai_summary_topics_topic_summary ON ai_summary_topics(topic_tag_id, summary_id)",
		"CREATE INDEX IF NOT EXISTS idx_reading_behaviors_feed_created_at ON reading_behaviors(feed_id, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_firecrawl_jobs_status_available_at ON firecrawl_jobs(status, available_at)",
		"CREATE INDEX IF NOT EXISTS idx_firecrawl_jobs_lease_expires_at ON firecrawl_jobs(lease_expires_at)",
		"CREATE INDEX IF NOT EXISTS idx_tag_jobs_status_available_at ON tag_jobs(status, available_at)",
		"CREATE INDEX IF NOT EXISTS idx_tag_jobs_lease_expires_at ON tag_jobs(lease_expires_at)",
	}
}
