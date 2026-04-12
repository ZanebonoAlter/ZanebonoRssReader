package database

import (
	"fmt"

	"gorm.io/gorm"
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
	}
}
