package database

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

func sqliteLegacyMigrations() []Migration {
	return []Migration{
		{
			Version:     "20260403_0001",
			Driver:      "sqlite",
			Description: "Bootstrap the legacy SQLite schema and historical backfills needed by existing tests and old databases.",
			Up:          bootstrapSQLiteLegacySchema,
		},
	}
}

func bootstrapSQLiteLegacySchema(db *gorm.DB) error {
	if err := createSQLiteLegacyTables(db); err != nil {
		return err
	}

	if err := createSQLiteLegacyIndexes(db); err != nil {
		return err
	}

	if err := runSQLiteLegacyColumnMigrations(db); err != nil {
		return err
	}

	return nil
}

func createSQLiteLegacyTables(db *gorm.DB) error {
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
				summary_processing_started_at DATETIME,
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
		"firecrawl_jobs": `
			CREATE TABLE IF NOT EXISTS firecrawl_jobs (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				article_id INTEGER NOT NULL,
				status VARCHAR(20) NOT NULL DEFAULT 'pending',
				priority INTEGER DEFAULT 0,
				attempt_count INTEGER DEFAULT 0,
				max_attempts INTEGER DEFAULT 5,
				available_at DATETIME NOT NULL,
				leased_at DATETIME,
				lease_expires_at DATETIME,
				last_error TEXT,
				url_snapshot VARCHAR(1000),
				created_at DATETIME,
				updated_at DATETIME,
				FOREIGN KEY(article_id) REFERENCES articles(id) ON DELETE CASCADE
			)`,
		"tag_jobs": `
			CREATE TABLE IF NOT EXISTS tag_jobs (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				article_id INTEGER NOT NULL,
				status VARCHAR(20) NOT NULL DEFAULT 'pending',
				priority INTEGER DEFAULT 0,
				attempt_count INTEGER DEFAULT 0,
				max_attempts INTEGER DEFAULT 5,
				available_at DATETIME NOT NULL,
				leased_at DATETIME,
				lease_expires_at DATETIME,
				last_error TEXT,
				feed_name_snapshot VARCHAR(200),
				category_name_snapshot VARCHAR(100),
				force_retag BOOLEAN DEFAULT 0,
				reason VARCHAR(50),
				created_at DATETIME,
				updated_at DATETIME,
				FOREIGN KEY(article_id) REFERENCES articles(id) ON DELETE CASCADE
			)`,
	}

	for tableName, createSQL := range tables {
		if !sqliteTableExists(db, tableName) {
			log.Printf("Creating table: %s", tableName)
			if err := db.Exec(createSQL).Error; err != nil {
				return fmt.Errorf("failed to create table %s: %w", tableName, err)
			}
		}
	}

	return nil
}

func createSQLiteLegacyIndexes(db *gorm.DB) error {
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
		"firecrawl_jobs": {
			"CREATE INDEX IF NOT EXISTS idx_firecrawl_jobs_status_available_at ON firecrawl_jobs(status, available_at)",
			"CREATE INDEX IF NOT EXISTS idx_firecrawl_jobs_lease_expires_at ON firecrawl_jobs(lease_expires_at)",
			"CREATE INDEX IF NOT EXISTS idx_firecrawl_jobs_article_id ON firecrawl_jobs(article_id)",
		},
		"tag_jobs": {
			"CREATE INDEX IF NOT EXISTS idx_tag_jobs_status_available_at ON tag_jobs(status, available_at)",
			"CREATE INDEX IF NOT EXISTS idx_tag_jobs_lease_expires_at ON tag_jobs(lease_expires_at)",
			"CREATE INDEX IF NOT EXISTS idx_tag_jobs_article_id ON tag_jobs(article_id)",
		},
	}

	for tableName, idxList := range indexes {
		for _, idxSQL := range idxList {
			if err := db.Exec(idxSQL).Error; err != nil {
				log.Printf("Warning: Failed to create index for %s: %v", tableName, err)
			}
		}
	}

	return nil
}

func runSQLiteLegacyColumnMigrations(db *gorm.DB) error {
	if sqliteTableExists(db, "ai_summaries") && !sqliteColumnExists(db, "ai_summaries", "feed_id") {
		if err := db.Exec("ALTER TABLE ai_summaries ADD COLUMN feed_id INTEGER REFERENCES feeds(id) ON DELETE CASCADE").Error; err != nil {
			return fmt.Errorf("add ai_summaries.feed_id: %w", err)
		}
	}

	if !sqliteColumnExists(db, "feeds", "article_summary_enabled") {
		if err := db.Exec("ALTER TABLE feeds ADD COLUMN article_summary_enabled BOOLEAN DEFAULT 0").Error; err != nil {
			return fmt.Errorf("add feeds.article_summary_enabled: %w", err)
		}
		if sqliteColumnExists(db, "feeds", "content_completion_enabled") {
			if err := db.Exec("UPDATE feeds SET article_summary_enabled = COALESCE(content_completion_enabled, 0)").Error; err != nil {
				return fmt.Errorf("backfill feeds.article_summary_enabled: %w", err)
			}
		}
	}

	if !sqliteColumnExists(db, "articles", "summary_status") {
		if err := db.Exec("ALTER TABLE articles ADD COLUMN summary_status VARCHAR(20) DEFAULT 'complete'").Error; err != nil {
			return fmt.Errorf("add articles.summary_status: %w", err)
		}
		if sqliteColumnExists(db, "articles", "content_status") {
			if err := db.Exec("UPDATE articles SET summary_status = COALESCE(content_status, 'complete')").Error; err != nil {
				return fmt.Errorf("backfill articles.summary_status: %w", err)
			}
		}
	}

	if !sqliteColumnExists(db, "articles", "summary_generated_at") {
		if err := db.Exec("ALTER TABLE articles ADD COLUMN summary_generated_at DATETIME").Error; err != nil {
			return fmt.Errorf("add articles.summary_generated_at: %w", err)
		}
		if sqliteColumnExists(db, "articles", "content_fetched_at") {
			if err := db.Exec("UPDATE articles SET summary_generated_at = content_fetched_at WHERE summary_generated_at IS NULL").Error; err != nil {
				return fmt.Errorf("backfill articles.summary_generated_at: %w", err)
			}
		}
	}

	articleMigrations := []struct {
		colName string
		colType string
	}{
		{"image_url", "VARCHAR(1000)"},
		{"summary_status", "VARCHAR(20) DEFAULT 'complete'"},
		{"summary_generated_at", "DATETIME"},
		{"summary_processing_started_at", "DATETIME"},
		{"completion_attempts", "INTEGER DEFAULT 0"},
		{"completion_error", "TEXT"},
		{"ai_content_summary", "TEXT"},
		{"firecrawl_status", "VARCHAR(20) DEFAULT 'pending'"},
		{"firecrawl_error", "TEXT"},
		{"firecrawl_content", "TEXT"},
		{"firecrawl_crawled_at", "DATETIME"},
	}

	for _, migration := range articleMigrations {
		if sqliteColumnExists(db, "articles", migration.colName) {
			continue
		}

		sql := fmt.Sprintf("ALTER TABLE articles ADD COLUMN %s %s", migration.colName, migration.colType)
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("add articles.%s: %w", migration.colName, err)
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

	for _, migration := range feedMigrations {
		if sqliteColumnExists(db, "feeds", migration.colName) {
			continue
		}

		sql := fmt.Sprintf("ALTER TABLE feeds ADD COLUMN %s %s", migration.colName, migration.colType)
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("add feeds.%s: %w", migration.colName, err)
		}
	}

	if sqliteTableExists(db, "topic_tags") {
		topicTagMigrations := []struct {
			colName string
			colType string
		}{
			{"category", "VARCHAR(20) DEFAULT 'keyword'"},
			{"icon", "VARCHAR(100)"},
			{"is_canonical", "BOOLEAN DEFAULT 0"},
		}

		for _, migration := range topicTagMigrations {
			if sqliteColumnExists(db, "topic_tags", migration.colName) {
				continue
			}

			sql := fmt.Sprintf("ALTER TABLE topic_tags ADD COLUMN %s %s", migration.colName, migration.colType)
			if err := db.Exec(sql).Error; err != nil {
				return fmt.Errorf("add topic_tags.%s: %w", migration.colName, err)
			}
		}

		if sqliteIndexExists(db, "idx_topic_tags_category_slug") == false {
			if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_topic_tags_category_slug ON topic_tags(category, slug)").Error; err != nil {
				return fmt.Errorf("create idx_topic_tags_category_slug: %w", err)
			}
		}
	}

	return nil
}

func sqliteColumnExists(db *gorm.DB, tableName, columnName string) bool {
	var count int64
	db.Raw("SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?", tableName, columnName).Scan(&count)
	return count > 0
}

func sqliteTableExists(db *gorm.DB, tableName string) bool {
	var count int64
	db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", tableName).Scan(&count)
	return count > 0
}

func sqliteIndexExists(db *gorm.DB, indexName string) bool {
	var count int64
	db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?", indexName).Scan(&count)
	return count > 0
}
