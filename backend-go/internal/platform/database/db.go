package database

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"my-robot-backend/internal/platform/config"
)

var DB *gorm.DB

var openPostgres = connectPostgres
var runDatabaseMigrations = RunMigrations

const defaultDatabaseDriver = "postgres"

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

	db, err := openPostgres(cfg, gormCfg)
	if err != nil {
		return err
	}

	DB = db

	if err := runDatabaseMigrations(db); err != nil {
		return fmt.Errorf("run database migrations: %w", err)
	}

	log.Println("Database initialized successfully")
	return nil
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

	return RunMigrations(DB)
}
