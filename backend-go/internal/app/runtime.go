package app

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"my-robot-backend/internal/app/runtimeinfo"
	"my-robot-backend/internal/domain/contentprocessing"
	"my-robot-backend/internal/domain/digest"
	"my-robot-backend/internal/domain/topicanalysis"
	"my-robot-backend/internal/domain/topicextraction"
	"my-robot-backend/internal/jobs"
)

type Runtime struct {
	AutoRefresh            *jobs.AutoRefreshScheduler
	AutoSummary            *jobs.AutoSummaryScheduler
	PreferenceUpdate       *jobs.PreferenceUpdateScheduler
	ContentCompletion      *jobs.ContentCompletionScheduler
	Firecrawl              *jobs.FirecrawlScheduler
	Digest                 *digest.DigestScheduler
	BlockedArticleRecovery *jobs.BlockedArticleRecoveryScheduler
	AutoTagMerge           *jobs.AutoTagMergeScheduler
}

func StartRuntime() *Runtime {
	runtime := &Runtime{}

	// Start the tag queue worker for async article tagging
	if err := topicextraction.GetTagQueue().Start(); err != nil {
		log.Printf("Warning: Failed to start tag queue: %v", err)
	} else {
		log.Println("Tag queue started successfully")
	}

	// Start the embedding queue worker for async embedding generation
	topicanalysis.StartEmbeddingQueueWorker()
	log.Println("Embedding queue worker started successfully")
	topicanalysis.StartMergeReembeddingQueueWorker()
	log.Println("Merge re-embedding queue worker started successfully")

	runtime.AutoRefresh = jobs.NewAutoRefreshScheduler(60)
	if err := runtime.AutoRefresh.Start(); err != nil {
		log.Printf("Warning: Failed to start auto-refresh scheduler: %v", err)
	} else {
		log.Println("Auto-refresh scheduler started successfully")
	}

	autoSummaryInterval := 3600
	runtime.AutoSummary = jobs.NewAutoSummaryScheduler(autoSummaryInterval)
	if err := runtime.AutoSummary.Start(); err != nil {
		log.Printf("Warning: Failed to start auto-summary scheduler: %v", err)
	} else {
		log.Println("Auto-summary scheduler started successfully")
	}

	preferenceUpdateInterval := 1800
	runtime.PreferenceUpdate = jobs.NewPreferenceUpdateScheduler(preferenceUpdateInterval)
	if err := runtime.PreferenceUpdate.Start(); err != nil {
		log.Printf("Warning: Failed to start preference update scheduler: %v", err)
	} else {
		log.Println("Preference update scheduler started successfully")
	}

	runtime.Firecrawl = jobs.NewFirecrawlScheduler()
	if err := runtime.Firecrawl.Start(); err != nil {
		log.Printf("Warning: Failed to start firecrawl scheduler: %v", err)
	} else {
		log.Println("Firecrawl scheduler started successfully")
	}

	crawlServiceURL := os.Getenv("CRAWL_SERVICE_URL")
	if crawlServiceURL == "" {
		crawlServiceURL = "http://localhost:11235"
	}
	contentprocessing.InitContentCompletionHandler(crawlServiceURL)

	runtime.ContentCompletion = jobs.NewContentCompletionScheduler(
		contentprocessing.GetContentCompletionService(),
		60,
	)
	if err := runtime.ContentCompletion.Start(); err != nil {
		log.Printf("Warning: Failed to start content completion scheduler: %v", err)
	} else {
		log.Println("Content completion scheduler started successfully")
	}

	runtime.Digest = digest.NewDigestScheduler()
	if err := runtime.Digest.Start(); err != nil {
		log.Printf("Warning: Failed to start digest scheduler: %v", err)
	} else {
		log.Println("Digest scheduler started successfully")
	}

	// STAT-04: Blocked article recovery scheduler (hourly)
	runtime.BlockedArticleRecovery = jobs.NewBlockedArticleRecoveryScheduler(3600)
	if err := runtime.BlockedArticleRecovery.Start(); err != nil {
		log.Printf("Warning: Failed to start blocked article recovery scheduler: %v", err)
	} else {
		log.Println("Blocked article recovery scheduler started successfully")
	}

	// Auto tag merge scheduler (hourly)
	runtime.AutoTagMerge = jobs.NewAutoTagMergeScheduler(3600)
	if err := runtime.AutoTagMerge.Start(); err != nil {
		log.Printf("Warning: Failed to start auto tag merge scheduler: %v", err)
	} else {
		log.Println("Auto tag merge scheduler started successfully")
	}

	runtimeinfo.AutoRefreshSchedulerInterface = runtime.AutoRefresh
	runtimeinfo.AutoSummarySchedulerInterface = runtime.AutoSummary
	runtimeinfo.PreferenceUpdateSchedulerInterface = runtime.PreferenceUpdate
	runtimeinfo.AISummarySchedulerInterface = runtime.ContentCompletion
	runtimeinfo.FirecrawlSchedulerInterface = runtime.Firecrawl
	runtimeinfo.DigestSchedulerInterface = runtime.Digest
	runtimeinfo.AutoTagMergeSchedulerInterface = runtime.AutoTagMerge

	return runtime
}

func SetupGracefulShutdown(runtime *Runtime) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v, shutting down gracefully...", sig)

		done := make(chan struct{})
		go func() {
			log.Println("Stopping tag queue...")
			topicextraction.GetTagQueue().Stop()

			log.Println("Stopping embedding queue worker...")
			topicanalysis.StopEmbeddingQueueWorker()

			log.Println("Stopping merge re-embedding queue worker...")
			topicanalysis.StopMergeReembeddingQueueWorker()

			if runtime.AutoRefresh != nil {
				log.Println("Stopping auto-refresh scheduler...")
				runtime.AutoRefresh.Stop()
			}

			if runtime.AutoSummary != nil {
				log.Println("Stopping auto-summary scheduler...")
				runtime.AutoSummary.Stop()
			}

			if runtime.PreferenceUpdate != nil {
				log.Println("Stopping preference update scheduler...")
				runtime.PreferenceUpdate.Stop()
			}

			if runtime.ContentCompletion != nil {
				log.Println("Stopping content completion scheduler...")
				runtime.ContentCompletion.Stop()
			}

			if runtime.Firecrawl != nil {
				log.Println("Stopping firecrawl scheduler...")
				runtime.Firecrawl.Stop()
			}

			if runtime.Digest != nil {
				log.Println("Stopping digest scheduler...")
				runtime.Digest.Stop()
			}

			if runtime.BlockedArticleRecovery != nil {
				log.Println("Stopping blocked article recovery scheduler...")
				runtime.BlockedArticleRecovery.Stop()
			}

			if runtime.AutoTagMerge != nil {
				log.Println("Stopping auto tag merge scheduler...")
				runtime.AutoTagMerge.Stop()
			}

			close(done)
		}()

		select {
		case <-done:
			log.Println("Graceful shutdown completed")
		case <-time.After(30 * time.Second):
			log.Println("Graceful shutdown timed out after 30s, forcing exit")
		}
		os.Exit(0)
	}()
}
