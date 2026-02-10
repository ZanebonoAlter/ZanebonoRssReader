package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"my-robot-backend/internal/config"
	"my-robot-backend/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// CurrentDBVersion 用于标记当前 Go 后端期望的数据库 schema 版本
// 后续如果有破坏性变更，可以按版本做增量迁移
const CurrentDBVersion = 1

func InitDB(cfg *config.Config) error {
	var err error

	// 对于 sqlite，DSN 一般就是文件路径，这里先简单判断一下文件是否存在
	isNewDB := false
	if cfg.Database.Driver == "sqlite" && cfg.Database.DSN != "" {
		if _, statErr := os.Stat(cfg.Database.DSN); os.IsNotExist(statErr) {
			isNewDB = true
			log.Printf("Database file %s does not exist, a new database will be created", cfg.Database.DSN)
		}
	}

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

	// 初始化 / 升级数据库版本记录
	if err := EnsureSchemaVersion(isNewDB); err != nil {
		log.Printf("Warning: Failed to ensure schema version: %v", err)
	}

	return nil
}

func Migrate() error {
	return DB.AutoMigrate(
		&models.Category{},
		&models.Feed{},
		&models.Article{},
		&models.AISummary{},
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
				pub_date DATETIME,
				author VARCHAR(200),
				read BOOLEAN DEFAULT 0,
				favorite BOOLEAN DEFAULT 0,
				created_at DATETIME,
				FOREIGN KEY(feed_id) REFERENCES feeds(id) ON DELETE CASCADE
			)`,
		"ai_summaries": `
			CREATE TABLE IF NOT EXISTS ai_summaries (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				category_id INTEGER,
				title VARCHAR(200) NOT NULL,
				summary TEXT NOT NULL,
				key_points TEXT,
				articles TEXT,
				article_count INTEGER DEFAULT 0,
				time_range INTEGER DEFAULT 180,
				created_at DATETIME,
				updated_at DATETIME,
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
	}

	indexes := map[string][]string{
		"feeds": {
			"CREATE INDEX IF NOT EXISTS idx_feeds_category_id ON feeds(category_id)",
		},
		"articles": {
			"CREATE INDEX IF NOT EXISTS idx_articles_feed_id ON articles(feed_id)",
		},
		"ai_summaries": {
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

	return nil
}

func tableExists(tableName string) bool {
	var count int64
	DB.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&count)
	return count > 0
}

// EnsureSchemaVersion 确保存在 schema 版本记录表，并根据需要执行增量迁移
func EnsureSchemaVersion(isNewDB bool) error {
	// 1. 创建版本表（如果不存在）
	createVersionTableSQL := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			version INTEGER NOT NULL,
			applied_at DATETIME NOT NULL
		)`

	if err := DB.Exec(createVersionTableSQL).Error; err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// 2. 读取当前数据库版本（没有记录则视为 0）
	var currentVersion int
	if err := DB.Raw("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&currentVersion).Error; err != nil {
		return fmt.Errorf("failed to read current schema version: %w", err)
	}

	if currentVersion == 0 {
		if isNewDB {
			log.Printf("New database detected, initializing schema version to %d", CurrentDBVersion)
		} else {
			log.Printf("Existing database without version record, treating as version 0 and migrating to %d", CurrentDBVersion)
		}
	}

	// 3. 如果版本已经最新，直接返回
	if currentVersion >= CurrentDBVersion {
		return nil
	}

	// 4. 执行从 currentVersion -> CurrentDBVersion 的逐版本迁移
	if err := applyMigrations(currentVersion, CurrentDBVersion); err != nil {
		return err
	}

	return nil
}

// applyMigrations 根据版本号顺序应用迁移
func applyMigrations(fromVersion, toVersion int) error {
	for v := fromVersion + 1; v <= toVersion; v++ {
		log.Printf("Applying database migration to version %d", v)

		switch v {
		case 1:
			// v1：当前已有的表结构（EnsureTables + AutoMigrate 已经覆盖）
			// 这里暂时不需要额外 SQL，将现有结构视为版本 1 的基线
		// 将来需要新增版本时，在这里追加：
		// case 2:
		//   // 执行 v1 -> v2 的结构变更
		//   if err := DB.Exec("ALTER TABLE ...").Error; err != nil {
		//       return fmt.Errorf("failed to apply migration v2: %w", err)
		//   }
		default:
			// 未知版本，防御性返回错误，避免误写入版本号
			return fmt.Errorf("no migration handler defined for version %d", v)
		}

		// 记录版本变更
		if err := recordMigration(v); err != nil {
			return err
		}
	}

	return nil
}

// recordMigration 写入一条迁移记录
func recordMigration(version int) error {
	now := time.Now()
	if err := DB.Exec(
		"INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)",
		version,
		now,
	).Error; err != nil {
		return fmt.Errorf("failed to record migration version %d: %w", version, err)
	}
	return nil
}
