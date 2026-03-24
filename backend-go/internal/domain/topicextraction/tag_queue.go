package topicextraction

import (
	"fmt"
	"log"
	"sync"
	"time"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

// TagTask represents a tag extraction task
type TagTask struct {
	ArticleID    uint
	FeedName     string
	CategoryName string
	CreatedAt    time.Time
}

// TagQueue manages asynchronous tag extraction tasks
type TagQueue struct {
	tasks    chan TagTask
	stopChan chan struct{}
	wg       sync.WaitGroup
	started  bool
	mu       sync.Mutex
}

var (
	instance *TagQueue
	once     sync.Once
)

// GetTagQueue returns the singleton instance of TagQueue
func GetTagQueue() *TagQueue {
	once.Do(func() {
		instance = &TagQueue{
			tasks:    make(chan TagTask, 1000), // Buffer up to 1000 tasks
			stopChan: make(chan struct{}),
		}
	})
	return instance
}

// Enqueue adds a tag task to the queue
// This is non-blocking and returns immediately
func (q *TagQueue) Enqueue(articleID uint, feedName, categoryName string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.started {
		return fmt.Errorf("tag queue is not started")
	}

	task := TagTask{
		ArticleID:    articleID,
		FeedName:     feedName,
		CategoryName: categoryName,
		CreatedAt:    time.Now(),
	}

	// Non-blocking send with select
	select {
	case q.tasks <- task:
		return nil
	default:
		// Queue is full, log a warning and return
		log.Printf("[WARN] Tag queue is full, dropping task for article %d", articleID)
		return fmt.Errorf("tag queue is full")
	}
}

// EnqueueAsync adds a tag task to the queue asynchronously (fire-and-forget)
// Use this when you don't need to know if the task was successfully queued
func (q *TagQueue) EnqueueAsync(articleID uint, feedName, categoryName string) {
	_ = q.Enqueue(articleID, feedName, categoryName)
}

// Start begins the tag worker
func (q *TagQueue) Start() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.started {
		return fmt.Errorf("tag queue already started")
	}

	q.started = true
	q.wg.Add(1)
	go q.worker()

	log.Println("Tag queue started successfully")
	return nil
}

// Stop gracefully shuts down the tag worker
func (q *TagQueue) Stop() {
	q.mu.Lock()
	if !q.started {
		q.mu.Unlock()
		return
	}
	q.started = false
	q.mu.Unlock()

	close(q.stopChan)
	q.wg.Wait()

	log.Println("Tag queue stopped")
}

// worker is the main loop that processes tag tasks
func (q *TagQueue) worker() {
	defer q.wg.Done()

	for {
		select {
		case <-q.stopChan:
			// Drain remaining tasks before stopping
			for {
				select {
				case task := <-q.tasks:
					q.processTask(task)
				default:
					return
				}
			}

		case task := <-q.tasks:
			q.processTask(task)
		}
	}
}

// processTask handles a single tag task
func (q *TagQueue) processTask(task TagTask) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ERROR] Panic in tag task for article %d: %v", task.ArticleID, r)
		}
	}()

	// Fetch the article from database
	var article models.Article
	if err := database.DB.First(&article, task.ArticleID).Error; err != nil {
		log.Printf("[WARN] Failed to fetch article %d for tagging: %v", task.ArticleID, err)
		return
	}

	// Perform the actual tagging
	if err := TagArticle(&article, task.FeedName, task.CategoryName); err != nil {
		log.Printf("[WARN] Failed to tag article %d: %v", task.ArticleID, err)
		return
	}

	log.Printf("[DEBUG] Successfully tagged article %d", task.ArticleID)
}
