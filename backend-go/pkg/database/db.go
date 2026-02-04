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
