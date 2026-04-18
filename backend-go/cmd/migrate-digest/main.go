package main

import (
	"my-robot-backend/internal/domain/digest"
	"my-robot-backend/internal/platform/config"
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/logging"
)

func main() {
	if err := config.LoadConfig("./configs"); err != nil {
		logging.Warnf("Warning: Failed to load config: %v", err)
	}

	if err := database.InitDB(config.AppConfig); err != nil {
		logging.Fatalf("Failed to initialize database: %v", err)
	}

	if err := digest.Migrate(); err != nil {
		logging.Fatalf("Failed to run digest migrations: %v", err)
	}

	logging.Infoln("Digest migration completed successfully")
}
