package database

import (
	"fmt"
	"sort"

	"gorm.io/gorm"

	"my-robot-backend/internal/domain/models"
)

type Migration struct {
	Version     string
	Driver      string
	Description string
	Up          func(db *gorm.DB) error
}

var migrationRegistry = registeredMigrations

func RunMigrations(db *gorm.DB, driver string) error {
	if db == nil {
		return fmt.Errorf("database connection is required")
	}

	normalizedDriver := normalizeDatabaseDriver(driver)
	if err := ensureSchemaMigrationsTable(db); err != nil {
		return err
	}

	appliedVersions, err := loadAppliedMigrationVersions(db, normalizedDriver)
	if err != nil {
		return err
	}

	for _, migration := range migrationsForDriver(normalizedDriver) {
		if appliedVersions[migration.Version] {
			continue
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := migration.Up(tx); err != nil {
				return fmt.Errorf("apply migration %s: %w", migration.Version, err)
			}

			if err := tx.Exec(
				"INSERT INTO schema_migrations (version, driver) VALUES (?, ?)",
				migration.Version,
				normalizedDriver,
			).Error; err != nil {
				return fmt.Errorf("record migration %s: %w", migration.Version, err)
			}

			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

func registeredMigrations() []Migration {
	migrations := append([]Migration{}, sqliteLegacyMigrations()...)
	migrations = append(migrations, postgresMigrations()...)
	return migrations
}

func migrationsForDriver(driver string) []Migration {
	filtered := make([]Migration, 0)
	for _, migration := range migrationRegistry() {
		if normalizeDatabaseDriver(migration.Driver) != driver {
			continue
		}
		filtered = append(filtered, migration)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Version < filtered[j].Version
	})

	return filtered
}

func ensureSchemaMigrationsTable(db *gorm.DB) error {
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) NOT NULL,
			driver VARCHAR(32) NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (driver, version)
		)
	`).Error; err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}

	return nil
}

func loadAppliedMigrationVersions(db *gorm.DB, driver string) (map[string]bool, error) {
	var versions []string
	if err := db.Raw("SELECT version FROM schema_migrations WHERE driver = ?", driver).Scan(&versions).Error; err != nil {
		return nil, fmt.Errorf("load applied migrations for %s: %w", driver, err)
	}

	applied := make(map[string]bool, len(versions))
	for _, version := range versions {
		applied[version] = true
	}

	return applied, nil
}

func autoMigrateModels(db *gorm.DB) error {
	return db.AutoMigrate(
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
		&models.FirecrawlJob{},
		&models.TagJob{},
	)
}
