package topicextraction

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/domain/topictypes"
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/ws"
)

const drainTimeout = 10 * time.Second

type TagQueue struct {
	stopChan     chan struct{}
	wg           sync.WaitGroup
	started      bool
	mu           sync.Mutex
	queue        *TagJobQueue
	pollInterval time.Duration
	lease        time.Duration
	batchSize    int
}

var (
	instance *TagQueue
	once     sync.Once
)

func GetTagQueue() *TagQueue {
	once.Do(func() {
		instance = &TagQueue{
			stopChan:     make(chan struct{}),
			queue:        NewTagJobQueue(database.DB),
			pollInterval: time.Second,
			lease:        10 * time.Minute,
			batchSize:    20,
		}
	})
	if instance.queue == nil {
		instance.queue = NewTagJobQueue(database.DB)
	}
	return instance
}

func (q *TagQueue) Enqueue(articleID uint, feedName, categoryName string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.started {
		return fmt.Errorf("tag queue is not started")
	}

	return q.queue.Enqueue(TagJobRequest{
		ArticleID:    articleID,
		FeedName:     feedName,
		CategoryName: categoryName,
		Reason:       "article_created",
	})
}

func (q *TagQueue) EnqueueAsync(articleID uint, feedName, categoryName string) {
	if err := q.Enqueue(articleID, feedName, categoryName); err != nil {
		log.Printf("[WARN] Failed to enqueue tag job for article %d: %v", articleID, err)
	}
}

func (q *TagQueue) Start() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.started {
		return fmt.Errorf("tag queue already started")
	}

	q.stopChan = make(chan struct{})
	q.queue = NewTagJobQueue(database.DB)
	q.started = true
	q.wg.Add(1)
	go q.worker()

	log.Println("Tag queue started successfully")
	return nil
}

func (q *TagQueue) Stop() {
	q.mu.Lock()
	if !q.started {
		q.mu.Unlock()
		return
	}
	q.started = false
	close(q.stopChan)
	q.mu.Unlock()

	q.wg.Wait()
	log.Println("Tag queue stopped")
}

func (q *TagQueue) worker() {
	defer q.wg.Done()

	ticker := time.NewTicker(q.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-q.stopChan:
			q.drainRemaining()
			return
		case <-ticker.C:
			q.processAvailableJobs()
		}
	}
}

func (q *TagQueue) drainRemaining() {
	ctx, cancel := context.WithTimeout(context.Background(), drainTimeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[INFO] Tag queue drain timed out after %v, remaining jobs will be processed on next start", drainTimeout)
			return
		default:
		}

		jobs, err := q.queue.Claim(q.batchSize, q.lease)
		if err != nil {
			log.Printf("[WARN] Failed to claim tag jobs during drain: %v", err)
			return
		}
		if len(jobs) == 0 {
			return
		}
		for _, job := range jobs {
			if ctx.Err() != nil {
				log.Printf("[INFO] Tag queue drain timed out, %d job(s) remaining for next start", len(jobs))
				return
			}
			q.processJob(job)
		}
	}
}

func (q *TagQueue) processAvailableJobs() {
	jobs, err := q.queue.Claim(q.batchSize, q.lease)
	if err != nil {
		log.Printf("[WARN] Failed to claim tag jobs: %v", err)
		return
	}

	for _, job := range jobs {
		q.processJob(job)
	}
}

func (q *TagQueue) processJob(job models.TagJob) {
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("panic: %v", r)
			log.Printf("[ERROR] Panic in tag job for article %d: %v", job.ArticleID, r)
			_ = q.queue.MarkFailed(job, msg, q.failureBackoff(job.AttemptCount))
		}
	}()

	var article models.Article
	if err := database.DB.First(&article, job.ArticleID).Error; err != nil {
		log.Printf("[WARN] Failed to fetch article %d for tagging: %v", job.ArticleID, err)
		_ = q.queue.MarkFailed(job, err.Error(), q.failureBackoff(job.AttemptCount))
		return
	}

	var err error
	if job.ForceRetag {
		err = RetagArticle(&article, job.FeedNameSnapshot, job.CategoryNameSnapshot)
	} else {
		err = TagArticle(&article, job.FeedNameSnapshot, job.CategoryNameSnapshot)
	}
	if err != nil {
		log.Printf("[WARN] Failed to tag article %d: %v", job.ArticleID, err)
		_ = q.queue.MarkFailed(job, err.Error(), q.failureBackoff(job.AttemptCount))
		return
	}

	if err := q.queue.MarkCompleted(job.ID); err != nil {
		log.Printf("[WARN] Failed to mark tag job %d completed: %v", job.ID, err)
		return
	}

	log.Printf("[DEBUG] Successfully tagged article %d", job.ArticleID)
	tags, broadcastErr := GetArticleTags(job.ArticleID)
	if broadcastErr != nil {
		log.Printf("[WARN] Failed to fetch tags for WebSocket broadcast: %v", broadcastErr)
		return
	}

	q.broadcastTagCompleted(job.ID, job.ArticleID, tags)
}

func (q *TagQueue) failureBackoff(attempt int) time.Duration {
	if attempt <= 1 {
		return time.Minute
	}
	backoff := time.Duration(1<<min(attempt-1, 4)) * time.Minute
	if backoff > 30*time.Minute {
		return 30 * time.Minute
	}
	return backoff
}

func (q *TagQueue) broadcastTagCompleted(jobID, articleID uint, tags []topictypes.TopicTag) {
	hub := ws.GetHub()
	tagItems := make([]ws.TagCompletedItem, len(tags))
	for i, tag := range tags {
		tagItems[i] = ws.TagCompletedItem{
			Slug:     tag.Slug,
			Label:    tag.Label,
			Category: tag.Category,
			Score:    tag.Score,
			Icon:     tag.Icon,
		}
	}

	msg := ws.TagCompletedMessage{
		Type:      "tag_completed",
		ArticleID: articleID,
		JobID:     jobID,
		Tags:      tagItems,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[WARN] Failed to marshal tag_completed message: %v", err)
		return
	}

	hub.BroadcastRaw(data)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
