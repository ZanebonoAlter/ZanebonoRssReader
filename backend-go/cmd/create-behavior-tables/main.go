package main

import (
	"fmt"
	"log"

	"my-robot-backend/internal/config"
	"my-robot-backend/pkg/database"
)

func main() {
	if err := config.LoadConfig("./configs"); err != nil {
		log.Printf("Warning: Failed to load config: %v", err)
	}

	if err := database.InitDB(config.AppConfig); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	fmt.Println("Creating reading_behavior and user_preferences tables...")

	sqlStatements := []string{
		`CREATE TABLE IF NOT EXISTS reading_behaviors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			article_id INTEGER NOT NULL,
			feed_id INTEGER,
			category_id INTEGER,
			session_id VARCHAR(100),
			event_type VARCHAR(20),
			scroll_depth INTEGER DEFAULT 0,
			reading_time INTEGER DEFAULT 0,
			created_at DATETIME,
			FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE,
			FOREIGN KEY (feed_id) REFERENCES feeds(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_reading_behaviors_article_id ON reading_behaviors(article_id)`,
		`CREATE INDEX IF NOT EXISTS idx_reading_behaviors_feed_id ON reading_behaviors(feed_id)`,
		`CREATE INDEX IF NOT EXISTS idx_reading_behaviors_category_id ON reading_behaviors(category_id)`,
		`CREATE INDEX IF NOT EXISTS idx_reading_behaviors_session_id ON reading_behaviors(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_reading_behaviors_created_at ON reading_behaviors(created_at)`,

		`CREATE TABLE IF NOT EXISTS user_preferences (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			feed_id INTEGER,
			category_id INTEGER,
			preference_score REAL DEFAULT 0,
			avg_reading_time INTEGER DEFAULT 0,
			interaction_count INTEGER DEFAULT 0,
			scroll_depth_avg REAL DEFAULT 0,
			last_interaction_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME,
			FOREIGN KEY (feed_id) REFERENCES feeds(id) ON DELETE CASCADE,
			FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_preferences_feed_id ON user_preferences(feed_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_preferences_category_id ON user_preferences(category_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_preferences_last_interaction_at ON user_preferences(last_interaction_at)`,
	}

	for i, sql := range sqlStatements {
		if err := database.DB.Exec(sql).Error; err != nil {
			log.Printf("Error executing statement %d: %v\nSQL: %s", i+1, err, sql)
			continue
		}
		fmt.Printf("✓ Statement %d executed successfully\n", i+1)
	}

	fmt.Println("\n✓ Migration completed successfully!")
	fmt.Println("New tables: reading_behaviors, user_preferences")
}
