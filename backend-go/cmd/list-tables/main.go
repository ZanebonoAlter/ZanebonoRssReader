package main

import (
	"fmt"

	"my-robot-backend/internal/config"
	"my-robot-backend/pkg/database"
)

func main() {
	if err := config.LoadConfig("./configs"); err != nil {
		fmt.Printf("Warning: Failed to load config: %v\n", err)
	}

	if err := database.InitDB(config.AppConfig); err != nil {
		fmt.Printf("Failed to initialize database: %v\n", err)
		return
	}

	var tables []struct {
		Name string
	}

	database.DB.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name").Scan(&tables)

	fmt.Println("========================================")
	fmt.Println("  Current Tables in Database")
	fmt.Println("========================================")
	for i, t := range tables {
		fmt.Printf("%2d. %s\n", i+1, t.Name)
	}
	fmt.Println("========================================")
	fmt.Printf("Total: %d tables\n", len(tables))
}
