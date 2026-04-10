package database

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"my-robot-backend/internal/platform/config"
)

func connectSQLite(cfg *config.Config, gormCfg *gorm.Config) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(cfg.Database.DSN), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect sqlite database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sqlite database instance: %w", err)
	}

	if journalMode := cfg.Database.SQLite.JournalMode; journalMode != "" {
		db.Exec(fmt.Sprintf("PRAGMA journal_mode=%s", journalMode))
	}
	if busyTimeout := cfg.Database.SQLite.BusyTimeoutMS; busyTimeout > 0 {
		db.Exec(fmt.Sprintf("PRAGMA busy_timeout=%d", busyTimeout))
	}
	if cfg.Database.SQLite.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.Database.SQLite.MaxIdleConns)
	}
	if cfg.Database.SQLite.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.Database.SQLite.MaxOpenConns)
	}

	return db, nil
}
