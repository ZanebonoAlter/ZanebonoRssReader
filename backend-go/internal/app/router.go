package app

import (
	"github.com/gin-gonic/gin"
	"my-robot-backend/internal/handlers"
	"my-robot-backend/internal/ws"
)

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
			feeds.GET("/:feed_id", handlers.GetFeed)
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

		api.GET("/auto-summary/status", handlers.GetAutoSummaryStatus)
		api.POST("/auto-summary/config", handlers.UpdateAutoSummaryConfig)

		readingBehavior := api.Group("/reading-behavior")
		{
			readingBehavior.POST("/track", handlers.TrackReadingBehavior)
			readingBehavior.POST("/track-batch", handlers.BatchTrackReadingBehavior)
			readingBehavior.GET("/stats", handlers.GetReadingStats)
		}

		preferences := api.Group("/user-preferences")
		{
			preferences.GET("", handlers.GetUserPreferences)
			preferences.POST("/update", handlers.TriggerPreferenceUpdate)
		}

		contentCompletion := api.Group("/content-completion")
		{
			contentCompletion.POST("/articles/:article_id/complete", handlers.CompleteArticleContent)
			contentCompletion.POST("/feeds/:feed_id/complete-all", handlers.CompleteFeedArticles)
			contentCompletion.GET("/articles/:article_id/status", handlers.GetCompletionStatus)
			contentCompletion.GET("/overview", handlers.GetCompletionOverview)
		}

		firecrawl := api.Group("/firecrawl")
		{
			firecrawl.POST("/article/:id", handlers.CrawlArticle)
			firecrawl.POST("/feed/:id/enable", handlers.EnableFeedFirecrawl)
			firecrawl.GET("/status", handlers.GetFirecrawlStatus)
			firecrawl.POST("/settings", handlers.SaveFirecrawlSettings)
		}

		digestGroup := api.Group("/digest")
		{
			digestGroup.GET("/config", handlers.GetDigestConfig)
			digestGroup.GET("/status", handlers.GetDigestStatus)
			digestGroup.GET("/preview/:type", handlers.GetDigestPreview)
			digestGroup.PUT("/config", handlers.UpdateDigestConfig)
			digestGroup.POST("/run/:type", handlers.RunDigestNow)
			digestGroup.POST("/test-feishu", handlers.TestFeishuPush)
			digestGroup.POST("/test-obsidian", handlers.TestObsidianWrite)
		}
	}
}
