package database

import (
	"fmt"
	"log"
	"time"

	"my-robot-backend/internal/config"
	"my-robot-backend/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB(cfg *config.Config) error {
	var err error

	logLevel := logger.Silent
	if cfg.Server.Mode == "debug" {
		logLevel = logger.Info
	}

	cstZone := time.FixedZone("CST", 8*3600)
	DB, err = gorm.Open(sqlite.Open(cfg.Database.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().In(cstZone)
		},
	})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	log.Println("Database initialized successfully")

	if err := EnsureTables(); err != nil {
		log.Printf("Warning: Failed to ensure tables exist: %v", err)
	}

	return nil
}

func Migrate() error {
	return DB.AutoMigrate(
		&models.Category{},
		&models.Feed{},
		&models.Article{},
		&models.AISummary{},
		&models.AISummaryFeed{},
		&models.SchedulerTask{},
		&models.AISettings{},
		&models.ReadingBehavior{},
		&models.UserPreference{},
	)
}

func EnsureTables() error {
	tables := map[string]string{
		"categories": `
			CREATE TABLE IF NOT EXISTS categories (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name VARCHAR(100) NOT NULL UNIQUE,
				slug VARCHAR(50) UNIQUE,
				icon VARCHAR(50) DEFAULT 'folder',
				color VARCHAR(20) DEFAULT '#6366f1',
				description TEXT,
				created_at DATETIME
			)`,
		"feeds": `
			CREATE TABLE IF NOT EXISTS feeds (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				title VARCHAR(200) NOT NULL,
				description TEXT,
				url VARCHAR(500) NOT NULL UNIQUE,
				category_id INTEGER,
				icon VARCHAR(50) DEFAULT 'rss',
				color VARCHAR(20) DEFAULT '#8b5cf6',
				last_updated DATETIME,
				created_at DATETIME,
				max_articles INTEGER DEFAULT 100,
				refresh_interval INTEGER DEFAULT 60,
				refresh_status VARCHAR(20) DEFAULT 'idle',
				refresh_error TEXT,
				last_refresh_at DATETIME,
				ai_summary_enabled BOOLEAN DEFAULT 1,
				content_completion_enabled BOOLEAN DEFAULT 0,
				completion_on_refresh BOOLEAN DEFAULT 1,
				max_completion_retries INTEGER DEFAULT 3,
				firecrawl_enabled BOOLEAN DEFAULT 0,
				FOREIGN KEY(category_id) REFERENCES categories(id) ON DELETE CASCADE
			)`,
		"articles": `
			CREATE TABLE IF NOT EXISTS articles (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				feed_id INTEGER NOT NULL,
				title VARCHAR(500) NOT NULL,
				description TEXT,
				content TEXT,
				link VARCHAR(1000),
				image_url VARCHAR(1000),
				pub_date DATETIME,
				author VARCHAR(200),
				read BOOLEAN DEFAULT 0,
				favorite BOOLEAN DEFAULT 0,
				content_status VARCHAR(20) DEFAULT 'complete',
				full_content TEXT,
				content_fetched_at DATETIME,
				completion_attempts INTEGER DEFAULT 0,
				completion_error TEXT,
				ai_content_summary TEXT,
				firecrawl_status VARCHAR(20) DEFAULT 'pending',
				firecrawl_error TEXT,
				firecrawl_content TEXT,
				firecrawl_crawled_at DATETIME,
				created_at DATETIME,
				FOREIGN KEY(feed_id) REFERENCES feeds(id) ON DELETE CASCADE
			)`,
		"ai_summaries": `
			CREATE TABLE IF NOT EXISTS ai_summaries (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				feed_id INTEGER,
				category_id INTEGER,
				title VARCHAR(200) NOT NULL,
				summary TEXT NOT NULL,
				key_points TEXT,
				articles TEXT,
				article_count INTEGER DEFAULT 0,
				time_range INTEGER DEFAULT 180,
				created_at DATETIME,
				updated_at DATETIME,
				FOREIGN KEY(feed_id) REFERENCES feeds(id) ON DELETE CASCADE,
				FOREIGN KEY(category_id) REFERENCES categories(id)
			)`,
		"scheduler_tasks": `
			CREATE TABLE IF NOT EXISTS scheduler_tasks (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name VARCHAR(50) NOT NULL UNIQUE,
				description VARCHAR(200),
				check_interval INTEGER NOT NULL DEFAULT 60,
				last_execution_time DATETIME,
				next_execution_time DATETIME,
				status VARCHAR(20) DEFAULT 'idle',
				last_error TEXT,
				last_error_time DATETIME,
				total_executions INTEGER DEFAULT 0,
				successful_executions INTEGER DEFAULT 0,
				failed_executions INTEGER DEFAULT 0,
				consecutive_failures INTEGER DEFAULT 0,
				last_execution_duration REAL,
				last_execution_result TEXT,
				created_at DATETIME,
				updated_at DATETIME
			)`,
		"ai_settings": `
			CREATE TABLE IF NOT EXISTS ai_settings (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				key VARCHAR(100) NOT NULL UNIQUE,
				value TEXT,
				description VARCHAR(200),
				created_at DATETIME,
				updated_at DATETIME
			)`,
		"reading_behaviors": `
			CREATE TABLE IF NOT EXISTS reading_behaviors (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				article_id INTEGER NOT NULL,
				feed_id INTEGER,
				category_id INTEGER,
				session_id VARCHAR(100),
				event_type VARCHAR(50),
				scroll_depth INTEGER DEFAULT 0,
				reading_time INTEGER DEFAULT 0,
				created_at DATETIME,
				FOREIGN KEY(feed_id) REFERENCES feeds(id),
				FOREIGN KEY(article_id) REFERENCES articles(id)
			)`,
		"user_preferences": `
			CREATE TABLE IF NOT EXISTS user_preferences (
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
				FOREIGN KEY(feed_id) REFERENCES feeds(id) ON DELETE CASCADE,
				FOREIGN KEY(category_id) REFERENCES categories(id) ON DELETE CASCADE
			)`,
		"ai_summary_feeds": `
			CREATE TABLE IF NOT EXISTS ai_summary_feeds (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				summary_id INTEGER NOT NULL,
				feed_id INTEGER NOT NULL,
				feed_title VARCHAR(200),
				feed_icon VARCHAR(50),
				feed_color VARCHAR(20),
				article_count INTEGER DEFAULT 0,
				created_at DATETIME,
				FOREIGN KEY(summary_id) REFERENCES ai_summaries(id) ON DELETE CASCADE,
				FOREIGN KEY(feed_id) REFERENCES feeds(id) ON DELETE CASCADE
			)`,
		"ai_summary_queue": `
			CREATE TABLE IF NOT EXISTS ai_summary_queue (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				article_id INTEGER NOT NULL,
				status VARCHAR(20) DEFAULT 'pending',
				retry_count INTEGER DEFAULT 0,
				error_message TEXT,
				created_at DATETIME,
				updated_at DATETIME,
				FOREIGN KEY(article_id) REFERENCES articles(id) ON DELETE CASCADE
			)`,
	}

	indexes := map[string][]string{
		"feeds": {
			"CREATE INDEX IF NOT EXISTS idx_feeds_category_id ON feeds(category_id)",
		},
		"articles": {
			"CREATE INDEX IF NOT EXISTS idx_articles_feed_id ON articles(feed_id)",
		},
		"ai_summaries": {
			"CREATE INDEX IF NOT EXISTS idx_ai_summaries_feed_id ON ai_summaries(feed_id)",
			"CREATE INDEX IF NOT EXISTS idx_ai_summaries_category_id ON ai_summaries(category_id)",
		},
		"scheduler_tasks": {
			"CREATE INDEX IF NOT EXISTS idx_scheduler_tasks_status ON scheduler_tasks(status)",
			"CREATE INDEX IF NOT EXISTS idx_scheduler_tasks_name ON scheduler_tasks(name)",
		},
		"ai_settings": {
			"CREATE INDEX IF NOT EXISTS idx_ai_settings_key ON ai_settings(key)",
		},
		"reading_behaviors": {
			"CREATE INDEX IF NOT EXISTS idx_reading_behaviors_category_id ON reading_behaviors(category_id)",
			"CREATE INDEX IF NOT EXISTS idx_reading_behaviors_feed_id ON reading_behaviors(feed_id)",
			"CREATE INDEX IF NOT EXISTS idx_reading_behaviors_article_id ON reading_behaviors(article_id)",
			"CREATE INDEX IF NOT EXISTS idx_reading_behaviors_created_at ON reading_behaviors(created_at)",
			"CREATE INDEX IF NOT EXISTS idx_reading_behaviors_event_type ON reading_behaviors(event_type)",
			"CREATE INDEX IF NOT EXISTS idx_reading_behaviors_session_id ON reading_behaviors(session_id)",
		},
		"user_preferences": {
			"CREATE INDEX IF NOT EXISTS idx_user_preferences_feed_id ON user_preferences(feed_id)",
			"CREATE INDEX IF NOT EXISTS idx_user_preferences_category_id ON user_preferences(category_id)",
			"CREATE INDEX IF NOT EXISTS idx_user_preferences_last_interaction_at ON user_preferences(last_interaction_at)",
		},
		"ai_summary_feeds": {
			"CREATE INDEX IF NOT EXISTS idx_ai_summary_feeds_summary_id ON ai_summary_feeds(summary_id)",
			"CREATE INDEX IF NOT EXISTS idx_ai_summary_feeds_feed_id ON ai_summary_feeds(feed_id)",
		},
	}

	for tableName, createSQL := range tables {
		if !tableExists(tableName) {
			log.Printf("Creating table: %s", tableName)
			if err := DB.Exec(createSQL).Error; err != nil {
				return fmt.Errorf("failed to create table %s: %w", tableName, err)
			}
			log.Printf("✓ Table created: %s", tableName)
		}
	}

	for tableName, idxList := range indexes {
		for _, idxSQL := range idxList {
			if err := DB.Exec(idxSQL).Error; err != nil {
				log.Printf("Warning: Failed to create index for %s: %v", tableName, err)
			}
		}
	}

	if err := runMigrations(); err != nil {
		log.Printf("Warning: Failed to run migrations: %v", err)
	}

	return nil
}

func runMigrations() error {
	if !columnExists("ai_summaries", "feed_id") {
		log.Println("Adding feed_id column to ai_summaries table...")
		if err := DB.Exec("ALTER TABLE ai_summaries ADD COLUMN feed_id INTEGER REFERENCES feeds(id) ON DELETE CASCADE").Error; err != nil {
			log.Printf("Warning: Failed to add feed_id column: %v", err)
		} else {
			log.Println("✓ feed_id column added to ai_summaries")
		}
	}

	articleMigrations := []struct {
		colName string
		colType string
	}{
		{"image_url", "VARCHAR(1000)"},
		{"content_status", "VARCHAR(20) DEFAULT 'complete'"},
		{"full_content", "TEXT"},
		{"content_fetched_at", "DATETIME"},
		{"completion_attempts", "INTEGER DEFAULT 0"},
		{"completion_error", "TEXT"},
		{"ai_content_summary", "TEXT"},
		{"firecrawl_status", "VARCHAR(20) DEFAULT 'pending'"},
		{"firecrawl_error", "TEXT"},
		{"firecrawl_content", "TEXT"},
		{"firecrawl_crawled_at", "DATETIME"},
	}

	for _, m := range articleMigrations {
		if !columnExists("articles", m.colName) {
			log.Printf("Adding %s column to articles table...", m.colName)
			sql := fmt.Sprintf("ALTER TABLE articles ADD COLUMN %s %s", m.colName, m.colType)
			if err := DB.Exec(sql).Error; err != nil {
				log.Printf("Warning: Failed to add %s column: %v", m.colName, err)
			} else {
				log.Printf("✓ %s column added to articles", m.colName)
			}
		}
	}

	feedMigrations := []struct {
		colName string
		colType string
	}{
		{"content_completion_enabled", "BOOLEAN DEFAULT 0"},
		{"completion_on_refresh", "BOOLEAN DEFAULT 1"},
		{"max_completion_retries", "INTEGER DEFAULT 3"},
		{"firecrawl_enabled", "BOOLEAN DEFAULT 0"},
	}

	for _, m := range feedMigrations {
		if !columnExists("feeds", m.colName) {
			log.Printf("Adding %s column to feeds table...", m.colName)
			sql := fmt.Sprintf("ALTER TABLE feeds ADD COLUMN %s %s", m.colName, m.colType)
			if err := DB.Exec(sql).Error; err != nil {
				log.Printf("Warning: Failed to add %s column: %v", m.colName, err)
			} else {
				log.Printf("✓ %s column added to feeds", m.colName)
			}
		}
	}

	return nil
}

func columnExists(tableName, columnName string) bool {
	var count int64
	DB.Raw("SELECT COUNT(*) FROM pragma_table_info(?) WHERE name=?", tableName, columnName).Scan(&count)
	return count > 0
}

func tableExists(tableName string) bool {
	var count int64
	DB.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&count)
	return count > 0
}
