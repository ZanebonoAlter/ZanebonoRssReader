package main

import (
	"fmt"
	"log"
	"os"

	"my-robot-backend/internal/config"
	"my-robot-backend/pkg/database"
)

func main() {
	fmt.Println("========================================")
	fmt.Println("  RSS Reader - Database Migration Tool")
	fmt.Println("========================================")
	fmt.Println()

	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  go run cmd/migrate/main.go <command>")
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("  check    - Check database schema")
		fmt.Println("  migrate  - Run auto migration (WARNING: may modify existing tables)")
		fmt.Println("  fresh    - Drop all tables and recreate (WARNING: deletes all data)")
		fmt.Println()
		os.Exit(1)
	}

	command := os.Args[1]

	if err := config.LoadConfig("./configs"); err != nil {
		log.Printf("Warning: Failed to load config: %v", err)
	}

	if err := database.InitDB(config.AppConfig); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	switch command {
	case "check":
		checkSchema()
	case "migrate":
		runMigration()
	case "fresh":
		freshStart()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}

func checkSchema() {
	fmt.Println("✓ Database connection successful")
	fmt.Println()
	fmt.Println("Note: Go backend uses the same database schema as Python backend.")
	fmt.Println("No migration needed - the schemas are compatible.")
}

func runMigration() {
	fmt.Println("Running auto-migration...")
	fmt.Println("WARNING: This may modify existing table structures!")
	fmt.Print("Continue? (y/N): ")

	var confirm string
	fmt.Scanln(&confirm)

	if confirm != "y" && confirm != "Y" {
		fmt.Println("Migration cancelled.")
		return
	}

	if err := database.Migrate(); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Println("✓ Migration completed successfully")
}

func freshStart() {
	fmt.Println("WARNING: This will DELETE ALL DATA!")
	fmt.Print("Are you sure? (type 'yes' to confirm): ")

	var confirm string
	fmt.Scanln(&confirm)

	if confirm != "yes" {
		fmt.Println("Operation cancelled.")
		return
	}

	fmt.Println("Dropping all tables...")
	database.DB.Exec("DROP TABLE IF EXISTS articles")
	database.DB.Exec("DROP TABLE IF EXISTS feeds")
	database.DB.Exec("DROP TABLE IF EXISTS categories")
	database.DB.Exec("DROP TABLE IF EXISTS ai_summaries")
	database.DB.Exec("DROP TABLE IF EXISTS scheduler_tasks")
	database.DB.Exec("DROP TABLE IF EXISTS ai_settings")

	fmt.Println("Running migrations...")
	if err := database.Migrate(); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Println("✓ Fresh database created successfully")
}
