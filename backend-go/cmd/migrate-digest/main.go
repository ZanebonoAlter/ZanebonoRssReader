package main

import (
	"fmt"
	"log"
	"my-robot-backend/internal/domain/digest"
	"my-robot-backend/internal/platform/config"
	"my-robot-backend/internal/platform/database"
)

func main() {
	if err := config.LoadConfig("./configs"); err != nil {
		log.Printf("Warning: Failed to load config: %v", err)
	}

	if err := database.InitDB(config.AppConfig); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	if err := digest.Migrate(); err != nil {
		log.Fatalf("Failed to run digest migrations: %v", err)
	}

	fmt.Println("Digest migration completed successfully")
}
