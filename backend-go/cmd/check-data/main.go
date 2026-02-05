package main

import (
	"fmt"
	"log"

	"my-robot-backend/internal/config"
	"my-robot-backend/internal/models"
	"my-robot-backend/pkg/database"
)

func main() {
	if err := config.LoadConfig("./configs"); err != nil {
		log.Printf("Warning: Failed to load config: %v", err)
	}

	if err := database.InitDB(config.AppConfig); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	var categoryCount, feedCount, articleCount int64

	database.DB.Table("categories").Count(&categoryCount)
	database.DB.Table("feeds").Count(&feedCount)
	database.DB.Table("articles").Count(&articleCount)

	fmt.Println("========================================")
	fmt.Println("  Database Recovery Status")
	fmt.Println("========================================")
	fmt.Printf("Categories: %d\n", categoryCount)
	fmt.Printf("Feeds:      %d\n", feedCount)
	fmt.Printf("Articles:   %d\n", articleCount)
	fmt.Println()

	if categoryCount > 0 || feedCount > 0 || articleCount > 0 {
		fmt.Println("✅ Data recovered successfully!")

		if categoryCount > 0 {
			var categories []models.Category
			database.DB.Order("name ASC").Limit(5).Find(&categories)
			fmt.Println("\n📁 Sample categories:")
			for _, cat := range categories {
				fmt.Printf("   - %s (%s)\n", cat.Name, cat.Slug)
			}
		}
	} else {
		fmt.Println("❌ No data found - database is empty")
	}
}
