package app

import (
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
	"my-robot-backend/internal/platform/logging"
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
		logging.Warnf("Failed to start tag queue: %v", err)
	} else {
		logging.Infoln("Tag queue started successfully")
	}

	// Start the embedding queue worker for async embedding generation
	topicanalysis.StartEmbeddingQueueWorker()
	logging.Infoln("Embedding queue worker started successfully")
	topicanalysis.StartMergeReembeddingQueueWorker()
	logging.Infoln("Merge re-embedding queue worker started successfully")

	runtime.AutoRefresh = jobs.NewAutoRefreshScheduler(60)
	if err := runtime.AutoRefresh.Start(); err != nil {
		logging.Warnf("Failed to start auto-refresh scheduler: %v", err)
	} else {
		logging.Infoln("Auto-refresh scheduler started successfully")
	}

	autoSummaryInterval := 3600
	runtime.AutoSummary = jobs.NewAutoSummaryScheduler(autoSummaryInterval)
	if err := runtime.AutoSummary.Start(); err != nil {
		logging.Warnf("Failed to start auto-summary scheduler: %v", err)
	} else {
		logging.Infoln("Auto-summary scheduler started successfully")
	}

	preferenceUpdateInterval := 1800
	runtime.PreferenceUpdate = jobs.NewPreferenceUpdateScheduler(preferenceUpdateInterval)
	if err := runtime.PreferenceUpdate.Start(); err != nil {
		logging.Warnf("Failed to start preference update scheduler: %v", err)
	} else {
		logging.Infoln("Preference update scheduler started successfully")
	}

	runtime.Firecrawl = jobs.NewFirecrawlScheduler()
	if err := runtime.Firecrawl.Start(); err != nil {
		logging.Warnf("Failed to start firecrawl scheduler: %v", err)
	} else {
		logging.Infoln("Firecrawl scheduler started successfully")
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
		logging.Warnf("Failed to start content completion scheduler: %v", err)
	} else {
		logging.Infoln("Content completion scheduler started successfully")
	}

	runtime.Digest = digest.NewDigestScheduler()
	if err := runtime.Digest.Start(); err != nil {
		logging.Warnf("Failed to start digest scheduler: %v", err)
	} else {
		logging.Infoln("Digest scheduler started successfully")
	}

	// STAT-04: Blocked article recovery scheduler (hourly)
	runtime.BlockedArticleRecovery = jobs.NewBlockedArticleRecoveryScheduler(3600)
	if err := runtime.BlockedArticleRecovery.Start(); err != nil {
		logging.Warnf("Failed to start blocked article recovery scheduler: %v", err)
	} else {
		logging.Infoln("Blocked article recovery scheduler started successfully")
	}

	// Auto tag merge scheduler (hourly)
	runtime.AutoTagMerge = jobs.NewAutoTagMergeScheduler(3600)
	if err := runtime.AutoTagMerge.Start(); err != nil {
		logging.Warnf("Failed to start auto tag merge scheduler: %v", err)
	} else {
		logging.Infoln("Auto tag merge scheduler started successfully")
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
		logging.Infof("Received signal: %v, shutting down gracefully...", sig)

		done := make(chan struct{})
		go func() {
			logging.Infoln("Stopping tag queue...")
			topicextraction.GetTagQueue().Stop()

			logging.Infoln("Stopping embedding queue worker...")
			topicanalysis.StopEmbeddingQueueWorker()

			logging.Infoln("Stopping merge re-embedding queue worker...")
			topicanalysis.StopMergeReembeddingQueueWorker()

			if runtime.AutoRefresh != nil {
				logging.Infoln("Stopping auto-refresh scheduler...")
				runtime.AutoRefresh.Stop()
			}

			if runtime.AutoSummary != nil {
				logging.Infoln("Stopping auto-summary scheduler...")
				runtime.AutoSummary.Stop()
			}

			if runtime.PreferenceUpdate != nil {
				logging.Infoln("Stopping preference update scheduler...")
				runtime.PreferenceUpdate.Stop()
			}

			if runtime.ContentCompletion != nil {
				logging.Infoln("Stopping content completion scheduler...")
				runtime.ContentCompletion.Stop()
			}

			if runtime.Firecrawl != nil {
				logging.Infoln("Stopping firecrawl scheduler...")
				runtime.Firecrawl.Stop()
			}

			if runtime.Digest != nil {
				logging.Infoln("Stopping digest scheduler...")
				runtime.Digest.Stop()
			}

			if runtime.BlockedArticleRecovery != nil {
				logging.Infoln("Stopping blocked article recovery scheduler...")
				runtime.BlockedArticleRecovery.Stop()
			}

			if runtime.AutoTagMerge != nil {
				logging.Infoln("Stopping auto tag merge scheduler...")
				runtime.AutoTagMerge.Stop()
			}

			close(done)
		}()

		select {
		case <-done:
			logging.Infoln("Graceful shutdown completed")
		case <-time.After(30 * time.Second):
			logging.Warnln("Graceful shutdown timed out after 30s, forcing exit")
		}
		os.Exit(0)
	}()
}
