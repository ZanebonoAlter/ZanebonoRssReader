package main

import (
	"log"
	"my-robot-backend/internal/config"
	"my-robot-backend/internal/models"
	"my-robot-backend/pkg/database"
	"time"
)

func main() {
	if err := config.LoadConfig("./configs"); err != nil {
		log.Printf("Warning: Failed to load config: %v", err)
	}

	if err := database.InitDB(config.AppConfig); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 创建测试分类
	category := models.Category{
		Name:  "AI技术",
		Slug:  "ai-tech",
		Icon:  "robot",
		Color: "#6366f1",
	}
	database.DB.Create(&category)

	// 创建测试feed
	feed := models.Feed{
		Title:            "TechCrunch",
		URL:              "https://techcrunch.com/feed/",
		CategoryID:       &category.ID,
		AISummaryEnabled: true,
	}
	database.DB.Create(&feed)

	// 创建测试AI总结
	summary := models.AISummary{
		FeedID:       &feed.ID,
		CategoryID:   &category.ID,
		Title:        "TechCrunch - 2026年3月4日测试",
		Summary:      "## 核心主题\n\n这是一个测试总结...",
		ArticleCount: 5,
		TimeRange:    180,
		CreatedAt:    time.Now(),
	}
	database.DB.Create(&summary)

	log.Println("Test data created successfully")
}
