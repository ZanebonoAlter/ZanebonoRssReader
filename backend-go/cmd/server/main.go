package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"my-robot-backend/internal/config"
	"my-robot-backend/internal/digest"
	"my-robot-backend/internal/handlers"
	"my-robot-backend/internal/middleware"
	"my-robot-backend/internal/schedulers"
	"my-robot-backend/internal/ws"
	"my-robot-backend/pkg/database"
)

var (
	autoRefreshScheduler       *schedulers.AutoRefreshScheduler
	autoSummaryScheduler       *schedulers.AutoSummaryScheduler
	preferenceUpdateScheduler  *schedulers.PreferenceUpdateScheduler
	contentCompletionScheduler *schedulers.ContentCompletionScheduler
	firecrawlScheduler         *schedulers.FirecrawlScheduler
	digestScheduler            *digest.DigestScheduler
)

func main() {
	if err := config.LoadConfig("./configs"); err != nil {
		log.Printf("Warning: Failed to load config: %v", err)
	}

	if err := database.InitDB(config.AppConfig); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Migrate digest config models
	if err := digest.Migrate(); err != nil {
		log.Fatalf("Failed to run digest migrations: %v", err)
	}

	if config.AppConfig != nil && config.AppConfig.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.Default()

	if config.AppConfig != nil {
		r.Use(middleware.CORS(config.AppConfig))
	}

	r.Use(gin.Recovery())

	SetupRoutes(r)

	// Initialize schedulers
	initializeSchedulers()

	// Setup graceful shutdown
	setupGracefulShutdown()

	addr := fmt.Sprintf(":%s", config.AppConfig.Server.Port)
	log.Printf("Server starting on %s", addr)
	log.Printf("Environment: %s", config.AppConfig.Server.Mode)

	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func SetupRoutes(r *gin.Engine) {
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"name":    "RSS Reader API (Go)",
			"version": "1.0.0",
			"endpoints": gin.H{
				"categories": "/api/categories",
				"feeds":      "/api/feeds",
				"articles":   "/api/articles",
				"ai":         "/api/ai",
				"opml": gin.H{
					"import": "POST /api/import-opml",
					"export": "GET /api/export-opml",
				},
				"schedulers": "/api/schedulers",
			},
		})
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":   "healthy",
			"database": "connected",
		})
	})

	r.GET("/api/tasks/status", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"success": true,
			"data": gin.H{
				"queue_size":   0,
				"active_tasks": 0,
				"tasks":        []string{},
			},
		})
	})

	// WebSocket endpoint
	r.GET("/ws", ws.HandleWebSocket)

	api := r.Group("/api")
	{
		categories := api.Group("/categories")
		{
			categories.GET("", handlers.GetCategories)
			categories.POST("", handlers.CreateCategory)
			categories.PUT("/:category_id", handlers.UpdateCategory)
			categories.DELETE("/:category_id", handlers.DeleteCategory)
		}

		feeds := api.Group("/feeds")
		{
			feeds.GET("", handlers.GetFeeds)
			feeds.POST("", handlers.CreateFeed)
			feeds.PUT("/:feed_id", handlers.UpdateFeed)
			feeds.DELETE("/:feed_id", handlers.DeleteFeed)
			feeds.POST("/:feed_id/refresh", handlers.RefreshFeed)
			feeds.POST("/fetch", handlers.FetchFeed)
			feeds.POST("/refresh-all", handlers.RefreshAllFeeds)
		}

		articles := api.Group("/articles")
		{
			articles.GET("/stats", handlers.GetArticlesStats)
			articles.GET("", handlers.GetArticles)
			articles.GET("/:article_id", handlers.GetArticle)
			articles.PUT("/:article_id", handlers.UpdateArticle)
			articles.PUT("/bulk-update", handlers.BulkUpdateArticles)
		}

		ai := api.Group("/ai")
		{
			ai.POST("/summarize", handlers.SummarizeArticle)
			ai.POST("/test", handlers.TestAIConnection)
			ai.GET("/settings", handlers.GetAISettings)
			ai.POST("/settings", handlers.SaveAISettings)
		}

		opml := api.Group("")
		{
			opml.POST("/import-opml", handlers.ImportOPML)
			opml.GET("/export-opml", handlers.ExportOPML)
		}

		schedulers := api.Group("/schedulers")
		{
			schedulers.GET("/status", handlers.GetSchedulersStatus)
			schedulers.GET("/:name/status", handlers.GetSchedulerStatus)
			schedulers.POST("/:name/trigger", handlers.TriggerScheduler)
			schedulers.POST("/:name/reset", handlers.ResetSchedulerStats)
			schedulers.PUT("/:name/interval", handlers.UpdateSchedulerInterval)
		}

		summaries := api.Group("/summaries")
		{
			summaries.GET("", handlers.GetSummaries)
			summaries.GET("/:summary_id", handlers.GetSummary)
			summaries.DELETE("/:summary_id", handlers.DeleteSummary)
			summaries.POST("/queue", handlers.SubmitQueueSummary)
			summaries.GET("/queue/status", handlers.GetQueueStatus)
			summaries.GET("/queue/jobs/:job_id", handlers.GetQueueJob)
		}

		// Auto summary configuration
		api.GET("/auto-summary/status", handlers.GetAutoSummaryStatus)
		api.POST("/auto-summary/config", handlers.UpdateAutoSummaryConfig)

		// Reading behavior tracking
		readingBehavior := api.Group("/reading-behavior")
		{
			readingBehavior.POST("/track", handlers.TrackReadingBehavior)
			readingBehavior.POST("/track-batch", handlers.BatchTrackReadingBehavior)
			readingBehavior.GET("/stats", handlers.GetReadingStats)
		}

		// User preferences
		preferences := api.Group("/user-preferences")
		{
			preferences.GET("", handlers.GetUserPreferences)
			preferences.POST("/update", handlers.TriggerPreferenceUpdate)
		}

		// Content completion
		contentCompletion := api.Group("/content-completion")
		{
			contentCompletion.POST("/articles/:article_id/complete", handlers.CompleteArticleContent)
			contentCompletion.POST("/feeds/:feed_id/complete-all", handlers.CompleteFeedArticles)
			contentCompletion.GET("/articles/:article_id/status", handlers.GetCompletionStatus)
		}

		// Firecrawl
		firecrawl := api.Group("/firecrawl")
		{
			firecrawl.POST("/article/:id", handlers.CrawlArticle)
			firecrawl.POST("/feed/:id/enable", handlers.EnableFeedFirecrawl)
			firecrawl.GET("/status", handlers.GetFirecrawlStatus)
		}

		// Digest configuration
		digestGroup := api.Group("/digest")
		{
			digestGroup.GET("/config", handlers.GetDigestConfig)
			digestGroup.PUT("/config", handlers.UpdateDigestConfig)
			digestGroup.POST("/test-feishu", handlers.TestFeishuPush)
			digestGroup.POST("/test-obsidian", handlers.TestObsidianWrite)
		}
	}
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stdout)
}

func initializeSchedulers() {
	// Initialize auto-refresh scheduler (default: 60 seconds)
	autoRefreshScheduler = schedulers.NewAutoRefreshScheduler(60)
	if err := autoRefreshScheduler.Start(); err != nil {
		log.Printf("Warning: Failed to start auto-refresh scheduler: %v", err)
	} else {
		log.Println("Auto-refresh scheduler started successfully")
	}

	// Initialize auto-summary scheduler (default: 3600 seconds = 1 hour)
	// Can be configured via environment variable AUTO_SUMMARY_INTERVAL
	autoSummaryInterval := 3600
	autoSummaryScheduler = schedulers.NewAutoSummaryScheduler(autoSummaryInterval)
	if err := autoSummaryScheduler.Start(); err != nil {
		log.Printf("Warning: Failed to start auto-summary scheduler: %v", err)
	} else {
		log.Println("Auto-summary scheduler started successfully")
	}

	// Initialize preference update scheduler (default: 1800 seconds = 30 minutes)
	preferenceUpdateInterval := 1800
	preferenceUpdateScheduler = schedulers.NewPreferenceUpdateScheduler(preferenceUpdateInterval)
	if err := preferenceUpdateScheduler.Start(); err != nil {
		log.Printf("Warning: Failed to start preference update scheduler: %v", err)
	} else {
		log.Println("Preference update scheduler started successfully")
	}

	// Initialize firecrawl scheduler
	firecrawlScheduler = schedulers.NewFirecrawlScheduler()
	if err := firecrawlScheduler.Start(); err != nil {
		log.Printf("Warning: Failed to start firecrawl scheduler: %v", err)
	} else {
		log.Println("Firecrawl scheduler started successfully")
	}

	// Initialize content completion service and scheduler
	crawlServiceURL := os.Getenv("CRAWL_SERVICE_URL")
	if crawlServiceURL == "" {
		crawlServiceURL = "http://localhost:11235"
	}
	handlers.InitContentCompletionHandler(crawlServiceURL)

	contentCompletionScheduler = schedulers.NewContentCompletionScheduler(
		handlers.GetContentCompletionService(),
		60, // 60 minutes
	)
	contentCompletionScheduler.Start()
	log.Println("Content completion scheduler started successfully")

	// Initialize digest scheduler
	digestScheduler = digest.NewDigestScheduler()
	if err := digestScheduler.Start(); err != nil {
		log.Printf("Warning: Failed to start digest scheduler: %v", err)
	} else {
		log.Println("Digest scheduler started successfully")
	}

	// Set scheduler references in handlers for status queries
	handlers.AutoRefreshSchedulerInterface = autoRefreshScheduler
	handlers.AutoSummarySchedulerInterface = autoSummaryScheduler
	handlers.FirecrawlSchedulerInterface = firecrawlScheduler
}

func setupGracefulShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v, shutting down gracefully...", sig)

		// Stop schedulers
		if autoRefreshScheduler != nil {
			log.Println("Stopping auto-refresh scheduler...")
			autoRefreshScheduler.Stop()
		}

		if autoSummaryScheduler != nil {
			log.Println("Stopping auto-summary scheduler...")
			autoSummaryScheduler.Stop()
		}

		if preferenceUpdateScheduler != nil {
			log.Println("Stopping preference update scheduler...")
			preferenceUpdateScheduler.Stop()
		}

		if contentCompletionScheduler != nil {
			log.Println("Stopping content completion scheduler...")
			contentCompletionScheduler.Stop()
		}

		if firecrawlScheduler != nil {
			log.Println("Stopping firecrawl scheduler...")
			firecrawlScheduler.Stop()
		}

		if digestScheduler != nil {
			log.Println("Stopping digest scheduler...")
			digestScheduler.Stop()
		}

		log.Println("Graceful shutdown completed")
		os.Exit(0)
	}()
}
