package database

import (
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"my-robot-backend/internal/platform/config"
)

var DB *gorm.DB

var openSQLite = connectSQLite
var openPostgres = connectPostgres
var runDatabaseMigrations = RunMigrations

var currentDatabaseDriver = defaultDatabaseDriver

const defaultDatabaseDriver = "sqlite"

func InitDB(cfg *config.Config) error {
	logLevel := logger.Silent
	if cfg.Server.Mode == "debug" {
		logLevel = logger.Info
	}

	cstZone := time.FixedZone("CST", 8*3600)
	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().In(cstZone)
		},
	}

	driver := normalizeDatabaseDriver(cfg.Database.Driver)
	db, err := openDatabase(cfg, gormCfg)
	if err != nil {
		return err
	}

	DB = db
	currentDatabaseDriver = driver

	if err := runDatabaseMigrations(db, driver); err != nil {
		return fmt.Errorf("run database migrations: %w", err)
	}

	log.Println("Database initialized successfully")
	return nil
}

func openDatabase(cfg *config.Config, gormCfg *gorm.Config) (*gorm.DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("database config is required")
	}

	switch normalizeDatabaseDriver(cfg.Database.Driver) {
	case defaultDatabaseDriver:
		return openSQLite(cfg, gormCfg)
	case "postgres":
		return openPostgres(cfg, gormCfg)
	default:
		return nil, fmt.Errorf("unknown database driver: %s", cfg.Database.Driver)
	}
}

func Migrate() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	return autoMigrateModels(DB)
}

func EnsureTables() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	return RunMigrations(DB, currentDatabaseDriver)
}

func normalizeDatabaseDriver(driver string) string {
	normalized := strings.ToLower(strings.TrimSpace(driver))
	if normalized == "" {
		return defaultDatabaseDriver
	}
	if normalized == "postgresql" {
		return "postgres"
	}
	return normalized
}
