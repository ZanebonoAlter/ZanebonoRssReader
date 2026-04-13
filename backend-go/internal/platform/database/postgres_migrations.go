package database

import (
	"fmt"
	"log"

	"gorm.io/gorm"
	"my-robot-backend/internal/domain/models"
)

func postgresMigrations() []Migration {
	return []Migration{
		{
			Version:     "20260403_0001",
			Description: "Enable pgvector support before any Postgres vector-aware schema changes.",
			Up: func(db *gorm.DB) error {
				if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
					return fmt.Errorf("create vector extension: %w", err)
				}
				return nil
			},
		},
		{
			Version:     "20260403_0002",
			Description: "Create the baseline Postgres schema used by the current runtime.",
			Up: func(db *gorm.DB) error {
				if err := bootstrapPostgresSchema(db); err != nil {
					return fmt.Errorf("bootstrap postgres schema: %w", err)
				}
				return nil
			},
		},
		{
			Version:     "20260403_0003",
			Description: "Staged groundwork for the later pgvector cutover: add the embedding vector column while runtime still reads the legacy JSON vector field.",
			Up: func(db *gorm.DB) error {
				if err := db.Exec("ALTER TABLE topic_tag_embeddings ADD COLUMN IF NOT EXISTS embedding vector(1536)").Error; err != nil {
					return fmt.Errorf("add topic_tag_embeddings.embedding column: %w", err)
				}
				return nil
			},
		},
		{
			Version:     "20260413_0001",
			Description: "Add HNSW index on topic_tag_embeddings.embedding for fast cosine similarity search.",
			Up: func(db *gorm.DB) error {
				if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_topic_tag_embeddings_embedding ON topic_tag_embeddings USING hnsw (embedding vector_cosine_ops)").Error; err != nil {
					return fmt.Errorf("create hnsw index on topic_tag_embeddings.embedding: %w", err)
				}
				return nil
			},
		},
		{
			Version:     "20260413_0002",
			Description: "Create embedding_config table with default configuration values.",
			Up: func(db *gorm.DB) error {
				if err := db.AutoMigrate(&models.EmbeddingConfig{}); err != nil {
					return fmt.Errorf("auto-migrate embedding_config: %w", err)
				}
				// Seed default config values (upsert)
				defaults := []models.EmbeddingConfig{
					{Key: "high_similarity_threshold", Value: "0.97", Description: "Auto-reuse existing tag if similarity >= this value"},
					{Key: "low_similarity_threshold", Value: "0.78", Description: "Auto-create new tag if similarity < this value"},
					{Key: "embedding_model", Value: "", Description: "Override embedding model name (empty = read from provider)"},
					{Key: "embedding_dimension", Value: "1536", Description: "Embedding vector dimension"},
				}
				for _, d := range defaults {
					var existing models.EmbeddingConfig
					if err := db.Where("key = ?", d.Key).First(&existing).Error; err != nil {
						if err := db.Create(&d).Error; err != nil {
							log.Printf("Warning: failed to seed embedding_config key %s: %v", d.Key, err)
						}
					}
				}
				return nil
			},
		},
		{
			Version:     "20260413_0003",
			Description: "Add status and merged_into_id columns to topic_tags for tag merge support.",
			Up: func(db *gorm.DB) error {
				if err := db.Exec("ALTER TABLE topic_tags ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active'").Error; err != nil {
					return fmt.Errorf("add topic_tags.status column: %w", err)
				}
				if err := db.Exec("ALTER TABLE topic_tags ADD COLUMN IF NOT EXISTS merged_into_id INTEGER REFERENCES topic_tags(id)").Error; err != nil {
					return fmt.Errorf("add topic_tags.merged_into_id column: %w", err)
				}
				if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_topic_tags_status ON topic_tags(status)").Error; err != nil {
					return fmt.Errorf("create idx_topic_tags_status: %w", err)
				}
				if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_topic_tags_merged_into_id ON topic_tags(merged_into_id)").Error; err != nil {
					return fmt.Errorf("create idx_topic_tags_merged_into_id: %w", err)
				}
				return nil
			},
		},
		{
			Version:     "20260413_0004",
			Description: "Create embedding_queue table for tracking embedding generation progress.",
			Up: func(db *gorm.DB) error {
				stmts := []string{
					`CREATE TABLE IF NOT EXISTS embedding_queue (
						id BIGSERIAL PRIMARY KEY,
						tag_id BIGINT NOT NULL REFERENCES topic_tags(id) ON DELETE CASCADE,
						status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
						error_message TEXT,
						retry_count INTEGER NOT NULL DEFAULT 0,
						created_at TIMESTAMP NOT NULL DEFAULT NOW(),
						started_at TIMESTAMP,
						completed_at TIMESTAMP
					)`,
					"CREATE INDEX IF NOT EXISTS idx_embedding_queue_status ON embedding_queue(status)",
					"CREATE INDEX IF NOT EXISTS idx_embedding_queue_tag_id ON embedding_queue(tag_id)",
				}
				for _, s := range stmts {
					if err := db.Exec(s).Error; err != nil {
						return fmt.Errorf("embedding_queue migration: %w", err)
					}
				}
				return nil
			},
		},
	}
}
