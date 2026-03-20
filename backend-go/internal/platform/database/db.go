package database

import (
	"fmt"
	"log"
	"time"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/config"

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
		&models.TopicTag{},
		&models.TopicTagEmbedding{},
		&models.TopicTagAnalysis{},
		&models.TopicAnalysisCursor{},
		&models.AISummaryTopic{},
		&models.ArticleTopicTag{},
		&models.AISummaryFeed{},
		&models.SchedulerTask{},
		&models.AISettings{},
		&models.AIProvider{},
		&models.AIRoute{},
		&models.AIRouteProvider{},
		&models.AICallLog{},
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
				article_summary_enabled BOOLEAN DEFAULT 0,
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
				summary_status VARCHAR(20) DEFAULT 'complete',
				summary_generated_at DATETIME,
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
		"topic_tags": `
			CREATE TABLE IF NOT EXISTS topic_tags (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				slug VARCHAR(120) NOT NULL,
				label VARCHAR(160) NOT NULL,
				category VARCHAR(20) NOT NULL DEFAULT 'keyword',
				icon VARCHAR(100),
				aliases TEXT,
				is_canonical BOOLEAN DEFAULT 0,
				source VARCHAR(20) DEFAULT 'llm',
				kind VARCHAR(20) DEFAULT 'keyword',
				created_at DATETIME,
				updated_at DATETIME
			)`,
		"topic_tag_embeddings": `
			CREATE TABLE IF NOT EXISTS topic_tag_embeddings (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				topic_tag_id INTEGER NOT NULL UNIQUE,
				vector TEXT NOT NULL,
				dimension INTEGER NOT NULL,
				model VARCHAR(50) NOT NULL,
				text_hash VARCHAR(64),
				created_at DATETIME,
				updated_at DATETIME,
				FOREIGN KEY(topic_tag_id) REFERENCES topic_tags(id) ON DELETE CASCADE
			)`,
		"topic_tag_analyses": `
			CREATE TABLE IF NOT EXISTS topic_tag_analyses (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				topic_tag_id INTEGER NOT NULL,
				analysis_type VARCHAR(32) NOT NULL,
				window_type VARCHAR(32) NOT NULL,
				anchor_date DATETIME NOT NULL,
				summary_count INTEGER DEFAULT 0,
				payload_json TEXT,
				source VARCHAR(20) DEFAULT 'heuristic',
				version INTEGER DEFAULT 1,
				created_at DATETIME,
				updated_at DATETIME,
				FOREIGN KEY(topic_tag_id) REFERENCES topic_tags(id) ON DELETE CASCADE,
				UNIQUE(topic_tag_id, analysis_type, window_type, anchor_date)
			)`,
		"topic_analysis_cursors": `
			CREATE TABLE IF NOT EXISTS topic_analysis_cursors (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				topic_tag_id INTEGER NOT NULL,
				analysis_type VARCHAR(32) NOT NULL,
				window_type VARCHAR(32) NOT NULL,
				last_summary_id INTEGER DEFAULT 0,
				last_updated_at DATETIME,
				created_at DATETIME,
				updated_at DATETIME,
				FOREIGN KEY(topic_tag_id) REFERENCES topic_tags(id) ON DELETE CASCADE,
				UNIQUE(topic_tag_id, analysis_type, window_type)
			)`,
		"ai_summary_topics": `
			CREATE TABLE IF NOT EXISTS ai_summary_topics (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				summary_id INTEGER NOT NULL,
				topic_tag_id INTEGER NOT NULL,
				score REAL DEFAULT 0,
				source VARCHAR(20) DEFAULT 'heuristic',
				created_at DATETIME,
				FOREIGN KEY(summary_id) REFERENCES ai_summaries(id) ON DELETE CASCADE,
				FOREIGN KEY(topic_tag_id) REFERENCES topic_tags(id) ON DELETE CASCADE
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
		"ai_providers": `
			CREATE TABLE IF NOT EXISTS ai_providers (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name VARCHAR(100) NOT NULL UNIQUE,
				provider_type VARCHAR(50) NOT NULL DEFAULT 'openai_compatible',
				base_url VARCHAR(500) NOT NULL,
				api_key TEXT NOT NULL,
				model VARCHAR(100) NOT NULL,
				enabled BOOLEAN NOT NULL DEFAULT 1,
				timeout_seconds INTEGER NOT NULL DEFAULT 120,
				max_tokens INTEGER,
				temperature REAL,
				metadata TEXT,
				created_at DATETIME,
				updated_at DATETIME
			)`,
		"ai_routes": `
			CREATE TABLE IF NOT EXISTS ai_routes (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name VARCHAR(100) NOT NULL,
				capability VARCHAR(50) NOT NULL,
				enabled BOOLEAN NOT NULL DEFAULT 1,
				strategy VARCHAR(50) NOT NULL DEFAULT 'ordered_failover',
				description VARCHAR(255),
				created_at DATETIME,
				updated_at DATETIME,
				UNIQUE(capability, name)
			)`,
		"ai_route_providers": `
			CREATE TABLE IF NOT EXISTS ai_route_providers (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				route_id INTEGER NOT NULL,
				provider_id INTEGER NOT NULL,
				priority INTEGER NOT NULL DEFAULT 100,
				enabled BOOLEAN NOT NULL DEFAULT 1,
				created_at DATETIME,
				updated_at DATETIME,
				FOREIGN KEY(route_id) REFERENCES ai_routes(id) ON DELETE CASCADE,
				FOREIGN KEY(provider_id) REFERENCES ai_providers(id) ON DELETE CASCADE,
				UNIQUE(route_id, provider_id)
			)`,
		"ai_call_logs": `
			CREATE TABLE IF NOT EXISTS ai_call_logs (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				capability VARCHAR(50) NOT NULL,
				route_name VARCHAR(100) NOT NULL,
				provider_name VARCHAR(100) NOT NULL,
				success BOOLEAN NOT NULL,
				is_fallback BOOLEAN NOT NULL DEFAULT 0,
				latency_ms INTEGER,
				error_code VARCHAR(100),
				error_message TEXT,
				request_meta TEXT,
				created_at DATETIME
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
		"article_topic_tags": `
			CREATE TABLE IF NOT EXISTS article_topic_tags (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				article_id INTEGER NOT NULL,
				topic_tag_id INTEGER NOT NULL,
				score REAL DEFAULT 0,
				source VARCHAR(20) DEFAULT 'llm',
				created_at DATETIME,
				updated_at DATETIME,
				FOREIGN KEY(article_id) REFERENCES articles(id) ON DELETE CASCADE,
				FOREIGN KEY(topic_tag_id) REFERENCES topic_tags(id) ON DELETE CASCADE,
				UNIQUE(article_id, topic_tag_id)
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
		"ai_providers": {
			"CREATE INDEX IF NOT EXISTS idx_ai_providers_enabled ON ai_providers(enabled)",
			"CREATE INDEX IF NOT EXISTS idx_ai_providers_provider_type ON ai_providers(provider_type)",
		},
		"ai_routes": {
			"CREATE INDEX IF NOT EXISTS idx_ai_routes_capability ON ai_routes(capability)",
			"CREATE INDEX IF NOT EXISTS idx_ai_routes_enabled ON ai_routes(enabled)",
		},
		"ai_route_providers": {
			"CREATE INDEX IF NOT EXISTS idx_ai_route_providers_route_id ON ai_route_providers(route_id)",
			"CREATE INDEX IF NOT EXISTS idx_ai_route_providers_provider_id ON ai_route_providers(provider_id)",
			"CREATE INDEX IF NOT EXISTS idx_ai_route_providers_priority ON ai_route_providers(priority)",
		},
		"ai_call_logs": {
			"CREATE INDEX IF NOT EXISTS idx_ai_call_logs_capability ON ai_call_logs(capability)",
			"CREATE INDEX IF NOT EXISTS idx_ai_call_logs_success ON ai_call_logs(success)",
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
		"topic_tags": {
			"CREATE INDEX IF NOT EXISTS idx_topic_tags_category ON topic_tags(category)",
			"CREATE INDEX IF NOT EXISTS idx_topic_tags_category_slug ON topic_tags(category, slug)",
		},
		"topic_tag_embeddings": {
			"CREATE INDEX IF NOT EXISTS idx_topic_tag_embeddings_topic_tag_id ON topic_tag_embeddings(topic_tag_id)",
		},
		"topic_tag_analyses": {
			"CREATE INDEX IF NOT EXISTS idx_topic_tag_analyses_tag_id ON topic_tag_analyses(topic_tag_id)",
			"CREATE INDEX IF NOT EXISTS idx_topic_tag_analyses_lookup ON topic_tag_analyses(topic_tag_id, analysis_type, window_type, anchor_date)",
		},
		"topic_analysis_cursors": {
			"CREATE INDEX IF NOT EXISTS idx_topic_analysis_cursors_tag_id ON topic_analysis_cursors(topic_tag_id)",
			"CREATE INDEX IF NOT EXISTS idx_topic_analysis_cursors_lookup ON topic_analysis_cursors(topic_tag_id, analysis_type, window_type)",
		},
		"article_topic_tags": {
			"CREATE INDEX IF NOT EXISTS idx_article_topic_tags_article_id ON article_topic_tags(article_id)",
			"CREATE INDEX IF NOT EXISTS idx_article_topic_tags_topic_tag_id ON article_topic_tags(topic_tag_id)",
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
	if tableExists("ai_summaries") && !columnExists("ai_summaries", "feed_id") {
		log.Println("Adding feed_id column to ai_summaries table...")
		if err := DB.Exec("ALTER TABLE ai_summaries ADD COLUMN feed_id INTEGER REFERENCES feeds(id) ON DELETE CASCADE").Error; err != nil {
			log.Printf("Warning: Failed to add feed_id column: %v", err)
		} else {
			log.Println("✓ feed_id column added to ai_summaries")
		}
	}

	if !columnExists("feeds", "article_summary_enabled") {
		log.Println("Adding article_summary_enabled column to feeds table...")
		if err := DB.Exec("ALTER TABLE feeds ADD COLUMN article_summary_enabled BOOLEAN DEFAULT 0").Error; err != nil {
			log.Printf("Warning: Failed to add article_summary_enabled column: %v", err)
		} else {
			if columnExists("feeds", "content_completion_enabled") {
				if err := DB.Exec("UPDATE feeds SET article_summary_enabled = COALESCE(content_completion_enabled, 0)").Error; err != nil {
					log.Printf("Warning: Failed to backfill article_summary_enabled: %v", err)
				}
			}
			log.Println("✓ article_summary_enabled column added to feeds")
		}
	}

	if !columnExists("articles", "summary_status") {
		log.Println("Adding summary_status column to articles table...")
		if err := DB.Exec("ALTER TABLE articles ADD COLUMN summary_status VARCHAR(20) DEFAULT 'complete'").Error; err != nil {
			log.Printf("Warning: Failed to add summary_status column: %v", err)
		} else {
			if columnExists("articles", "content_status") {
				if err := DB.Exec("UPDATE articles SET summary_status = COALESCE(content_status, 'complete')").Error; err != nil {
					log.Printf("Warning: Failed to backfill summary_status: %v", err)
				}
			}
			log.Println("✓ summary_status column added to articles")
		}
	}

	if !columnExists("articles", "summary_generated_at") {
		log.Println("Adding summary_generated_at column to articles table...")
		if err := DB.Exec("ALTER TABLE articles ADD COLUMN summary_generated_at DATETIME").Error; err != nil {
			log.Printf("Warning: Failed to add summary_generated_at column: %v", err)
		} else {
			if columnExists("articles", "content_fetched_at") {
				if err := DB.Exec("UPDATE articles SET summary_generated_at = content_fetched_at WHERE summary_generated_at IS NULL").Error; err != nil {
					log.Printf("Warning: Failed to backfill summary_generated_at: %v", err)
				}
			}
			log.Println("✓ summary_generated_at column added to articles")
		}
	}

	articleMigrations := []struct {
		colName string
		colType string
	}{
		{"image_url", "VARCHAR(1000)"},
		{"summary_status", "VARCHAR(20) DEFAULT 'complete'"},
		{"summary_generated_at", "DATETIME"},
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
		{"article_summary_enabled", "BOOLEAN DEFAULT 0"},
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

	// topic_tags migrations for new category system
	if tableExists("topic_tags") {
		topicTagMigrations := []struct {
			colName string
			colType string
		}{
			{"category", "VARCHAR(20) DEFAULT 'keyword'"},
			{"icon", "VARCHAR(100)"},
			{"is_canonical", "BOOLEAN DEFAULT 0"},
		}

		for _, m := range topicTagMigrations {
			if !columnExists("topic_tags", m.colName) {
				log.Printf("Adding %s column to topic_tags table...", m.colName)
				sql := fmt.Sprintf("ALTER TABLE topic_tags ADD COLUMN %s %s", m.colName, m.colType)
				if err := DB.Exec(sql).Error; err != nil {
					log.Printf("Warning: Failed to add %s column: %v", m.colName, err)
				} else {
					log.Printf("✓ %s column added to topic_tags", m.colName)
				}
			}
		}

		// Migrate existing kind values to category
		if columnExists("topic_tags", "category") {
			var needsMigration int64
			DB.Raw("SELECT COUNT(*) FROM topic_tags WHERE category = 'keyword' AND kind IS NOT NULL AND kind != ''").Scan(&needsMigration)
			if needsMigration > 0 {
				log.Printf("Migrating %d existing topic_tags to new category system...", needsMigration)
				log.Printf("✓ Existing topic_tags migrated to category system")
			}
		}

		var indexExists int64
		DB.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_topic_tags_category_slug'").Scan(&indexExists)
		if indexExists == 0 {
			log.Println("Creating composite index for topic_tags...")
			if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_topic_tags_category_slug ON topic_tags(category, slug)").Error; err != nil {
				log.Printf("Warning: Failed to create composite index: %v", err)
			} else {
				log.Println("✓ Composite index created for topic_tags")
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
