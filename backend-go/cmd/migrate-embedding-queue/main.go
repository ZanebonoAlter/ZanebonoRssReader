package main

import (
	"fmt"
	"log"
	"my-robot-backend/internal/platform/config"
	"my-robot-backend/internal/platform/database"
)

func main() {
	if err := config.LoadConfig("./configs"); err != nil {
		log.Printf("Warning: Failed to load config: %v", err)
	}

	if err := database.InitDB(config.AppConfig); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	db := database.DB
	if db == nil {
		log.Fatal("Database connection is nil")
	}

	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS embedding_queue (
			id BIGSERIAL PRIMARY KEY,
			tag_id BIGINT NOT NULL REFERENCES topic_tags(id) ON DELETE CASCADE,
			status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
			error_message TEXT,
			retry_count INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			started_at TIMESTAMP,
			completed_at TIMESTAMP
		)
	`).Error; err != nil {
		log.Fatalf("Failed to create embedding_queue table: %v", err)
	}
	log.Println("✅ embedding_queue table created (or already exists)")

	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_embedding_queue_status ON embedding_queue(status)
	`).Error; err != nil {
		log.Fatalf("Failed to create idx_embedding_queue_status index: %v", err)
	}
	log.Println("✅ idx_embedding_queue_status index created (or already exists)")

	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_embedding_queue_tag_id ON embedding_queue(tag_id)
	`).Error; err != nil {
		log.Fatalf("Failed to create idx_embedding_queue_tag_id index: %v", err)
	}
	log.Println("✅ idx_embedding_queue_tag_id index created (or already exists)")

	fmt.Println("Embedding queue migration completed successfully")
}
