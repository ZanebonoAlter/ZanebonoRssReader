package summaries

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/domain/preferences"
	"my-robot-backend/internal/domain/topicextraction"
	"my-robot-backend/internal/platform/airouter"
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/tracing"
	"my-robot-backend/internal/platform/ws"
)

type SummaryJobStatus string

const (
	JobPending    SummaryJobStatus = "pending"
	JobProcessing SummaryJobStatus = "processing"
	JobCompleted  SummaryJobStatus = "completed"
	JobFailed     SummaryJobStatus = "failed"
)

type SummaryJob struct {
	ID           string           `json:"id"`
	BatchID      string           `json:"batch_id"`
	FeedID       *uint            `json:"feed_id"`
	FeedName     string           `json:"feed_name"`
	FeedIcon     string           `json:"feed_icon"`
	FeedColor    string           `json:"feed_color"`
	CategoryID   *uint            `json:"category_id"`
	CategoryName string           `json:"category_name"`
	Status       SummaryJobStatus `json:"status"`
	ErrorMessage string           `json:"error_message,omitempty"`
	ErrorCode    string           `json:"error_code,omitempty"`
	ResultID     *uint            `json:"result_id,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
	CompletedAt  *time.Time       `json:"completed_at,omitempty"`
}

type SummaryBatch struct {
	ID            string        `json:"id"`
	Status        string        `json:"status"`
	TotalJobs     int           `json:"total_jobs"`
	CompletedJobs int           `json:"completed_jobs"`
	FailedJobs    int           `json:"failed_jobs"`
	CreatedAt     time.Time     `json:"created_at"`
	CompletedAt   *time.Time    `json:"completed_at,omitempty"`
	Jobs          []*SummaryJob `json:"jobs"`
}

type SummaryQueue struct {
	mu           sync.RWMutex
	currentBatch *SummaryBatch
	running      bool
	stopChan     chan struct{}
}

var (
	queueInstance *SummaryQueue
	queueOnce     sync.Once
)

func GetSummaryQueue() *SummaryQueue {
	queueOnce.Do(func() {
		queueInstance = &SummaryQueue{
			stopChan: make(chan struct{}),
		}
	})
	return queueInstance
}

type AIConfig struct {
	BaseURL   string
	APIKey    string
	Model     string
	TimeRange int
}

var requestQueueSummaryChat = func(prompt string, metadata map[string]any) (string, error) {
	maxTokens := 16000
	temperature := 0.7
	result, err := airouter.NewRouter().Chat(context.Background(), airouter.ChatRequest{
		Capability: airouter.CapabilitySummary,
		Messages: []airouter.Message{
			{Role: "system", Content: "你是一名专业的新闻分析助手，擅长总结并比较多篇文章。"},
			{Role: "user", Content: prompt},
		},
		Temperature: &temperature,
		MaxTokens:   &maxTokens,
		Metadata:    metadata,
	})
	if err != nil {
		return "", err
	}
	return result.Content, nil
}

func (q *SummaryQueue) SubmitBatch(categoryIDs []uint, feedIDs []uint, config AIConfig) *SummaryBatch {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.running {
		close(q.stopChan)
		q.stopChan = make(chan struct{})
	}

	var feeds []models.Feed
	if len(feedIDs) > 0 {
		database.DB.Where("id IN ? AND ai_summary_enabled = ?", feedIDs, true).
			Preload("Category").
			Find(&feeds)
	} else if len(categoryIDs) > 0 {
		database.DB.Where("category_id IN ? AND ai_summary_enabled = ?", categoryIDs, true).
			Preload("Category").
			Find(&feeds)
	} else {
		database.DB.Where("ai_summary_enabled = ?", true).
			Preload("Category").
			Find(&feeds)
	}

	batchID := generateBatchID()
	jobs := make([]*SummaryJob, 0, len(feeds))
	now := time.Now()

	for _, feed := range feeds {
		feedIDCopy := feed.ID
		var catID *uint
		var catName string
		if feed.Category != nil && feed.CategoryID != nil {
			catIDVal := *feed.CategoryID
			catID = &catIDVal
			catName = feed.Category.Name
		}

		job := &SummaryJob{
			ID:           generateJobID(),
			BatchID:      batchID,
			FeedID:       &feedIDCopy,
			FeedName:     feed.Title,
			FeedIcon:     feed.Icon,
			FeedColor:    feed.Color,
			CategoryID:   catID,
			CategoryName: catName,
			Status:       JobPending,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		jobs = append(jobs, job)
	}

	q.currentBatch = &SummaryBatch{
		ID:            batchID,
		Status:        "pending",
		TotalJobs:     len(jobs),
		CompletedJobs: 0,
		FailedJobs:    0,
		CreatedAt:     now,
		Jobs:          jobs,
	}

	q.running = true
	go q.processBatch(context.Background(), config)

	return q.currentBatch
}

func (q *SummaryQueue) GetCurrentBatch() *SummaryBatch {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.currentBatch
}

func (q *SummaryQueue) GetJob(jobID string) *SummaryJob {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.currentBatch == nil {
		return nil
	}

	for _, job := range q.currentBatch.Jobs {
		if job.ID == jobID {
			return job
		}
	}
	return nil
}

func (q *SummaryQueue) broadcastProgress(currentJob *SummaryJob) {
	if q.currentBatch == nil {
		return
	}

	hub := ws.GetHub()

	jobs := make([]ws.JobUpdate, len(q.currentBatch.Jobs))
	for i, job := range q.currentBatch.Jobs {
		jobs[i] = ws.JobUpdate{
			ID:           job.ID,
			FeedID:       job.FeedID,
			FeedName:     job.FeedName,
			FeedIcon:     job.FeedIcon,
			FeedColor:    job.FeedColor,
			CategoryID:   job.CategoryID,
			CategoryName: job.CategoryName,
			Status:       string(job.Status),
			ErrorMessage: job.ErrorMessage,
			ErrorCode:    job.ErrorCode,
			ResultID:     job.ResultID,
		}
	}

	msg := &ws.SummaryProgressMessage{
		Type:      "progress",
		BatchID:   q.currentBatch.ID,
		Status:    q.currentBatch.Status,
		TotalJobs: q.currentBatch.TotalJobs,
		Completed: q.currentBatch.CompletedJobs,
		Failed:    q.currentBatch.FailedJobs,
		Jobs:      jobs,
	}

	if currentJob != nil {
		msg.CurrentJob = &ws.JobUpdate{
			ID:           currentJob.ID,
			FeedID:       currentJob.FeedID,
			FeedName:     currentJob.FeedName,
			FeedIcon:     currentJob.FeedIcon,
			FeedColor:    currentJob.FeedColor,
			CategoryID:   currentJob.CategoryID,
			CategoryName: currentJob.CategoryName,
			Status:       string(currentJob.Status),
			ErrorMessage: currentJob.ErrorMessage,
			ErrorCode:    currentJob.ErrorCode,
			ResultID:     currentJob.ResultID,
		}
	}

	hub.BroadcastProgress(msg)
}

func (q *SummaryQueue) processBatch(ctx context.Context, config AIConfig) {
	ctx, span := tracing.Tracer("summaries").Start(ctx, "SummaryQueue.processBatch",
		trace.WithAttributes(attribute.String("batch.id", q.currentBatch.ID)),
	)
	defer span.End()

	span.AddEvent("input", trace.WithAttributes(
		attribute.Int("feeds.count", q.currentBatch.TotalJobs),
		attribute.String("config.model", config.Model),
	))

	q.mu.Lock()
	q.currentBatch.Status = "processing"
	q.mu.Unlock()

	q.broadcastProgress(nil)

	for _, job := range q.currentBatch.Jobs {
		select {
		case <-q.stopChan:
			return
		default:
		}

		q.processJob(ctx, job, config)
	}

	q.mu.Lock()
	q.currentBatch.Status = "completed"
	now := time.Now()
	q.currentBatch.CompletedAt = &now
	q.running = false
	q.mu.Unlock()

	span.AddEvent("output", trace.WithAttributes(
		attribute.Int("summaries.created", q.currentBatch.CompletedJobs),
		attribute.Int("summaries.failed", q.currentBatch.FailedJobs),
	))
	span.SetStatus(codes.Ok, "")

	q.broadcastProgress(nil)
}

func (q *SummaryQueue) processJob(ctx context.Context, job *SummaryJob, config AIConfig) {
	feedID := 0
	if job.FeedID != nil {
		feedID = int(*job.FeedID)
	}
	ctx, span := tracing.Tracer("summaries").Start(ctx, "SummaryQueue.processJob",
		trace.WithAttributes(
			attribute.Int("job.feed_id", feedID),
			attribute.String("job.status", string(job.Status)),
		),
	)
	defer span.End()

	span.AddEvent("input", trace.WithAttributes(
		attribute.String("job.feed_name", job.FeedName),
		attribute.String("job.category_name", job.CategoryName),
	))

	q.mu.Lock()
	job.Status = JobProcessing
	job.UpdatedAt = time.Now()
	q.mu.Unlock()

	q.broadcastProgress(job)

	result, err := q.generateSummaryForFeed(job.FeedID, job.CategoryID, job.FeedName, job.CategoryName, config)

	q.mu.Lock()

	job.UpdatedAt = time.Now()
	now := time.Now()
	job.CompletedAt = &now

	if err != nil {
		job.Status = JobFailed
		job.ErrorMessage = err.Error()
		job.ErrorCode = classifyError(err)
		q.currentBatch.FailedJobs++
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		job.Status = JobCompleted
		job.ResultID = &result.ID
		q.currentBatch.CompletedJobs++
		span.AddEvent("output", trace.WithAttributes(
			attribute.Int("result.id", int(result.ID)),
		))
		span.SetStatus(codes.Ok, "")
	}

	q.mu.Unlock()

	q.broadcastProgress(job)
}

const maxQueueArticlesPerSummary = 10

func (q *SummaryQueue) generateSummaryForFeed(feedID *uint, categoryID *uint, feedName string, categoryName string, config AIConfig) (*models.AISummary, error) {
	timeRange := config.TimeRange
	if timeRange == 0 {
		timeRange = 180
	}

	timeThreshold := time.Now().Add(-time.Duration(timeRange) * time.Minute)

	var articles []models.Article
	if feedID != nil {
		database.DB.Omit("tag_count", "relevance_score").Where("feed_id = ? AND created_at >= ? AND feed_summary_generated_at IS NULL", *feedID, timeThreshold).
			Order("created_at DESC").
			Find(&articles)
	} else {
		return nil, &SummaryError{Code: "NO_FEED", Message: "未指定订阅源"}
	}

	if len(articles) == 0 {
		return nil, &SummaryError{Code: "NO_ARTICLES", Message: "该订阅源下没有找到文章"}
	}

	log.Printf("Found %d articles for feed %d", len(articles), *feedID)

	displayName := feedName
	if displayName == "" {
		displayName = "未知订阅源"
	}

	batches := chunkQueueArticles(articles, maxQueueArticlesPerSummary)
	totalBatches := len(batches)
	var lastSummary *models.AISummary

	for batchIndex, batch := range batches {
		batchNum := batchIndex + 1
		log.Printf("Processing queue batch %d/%d with %d articles for feed %d", batchNum, totalBatches, len(batch), *feedID)

		articleTexts := make([]string, 0, len(batch))
		batchArticleIDs := make([]uint, len(batch))
		for _, article := range batch {
			articleTexts = append(articleTexts, buildQueueArticleText(article))
		}
		for i, article := range batch {
			batchArticleIDs[i] = article.ID
		}
		articleIDsJSON, _ := json.Marshal(batchArticleIDs)

		existingSummary, err := findQueueSummaryBatch(feedID, string(articleIDsJSON))
		if err != nil {
			return nil, &SummaryError{Code: "DB_ERROR", Message: "检查已有总结失败: " + err.Error()}
		}
		if existingSummary != nil {
			log.Printf("Skipping existing queue summary for feed %d batch %d/%d (ID: %d)", *feedID, batchNum, totalBatches, existingSummary.ID)
			if err := MarkArticlesWithFeedSummary(batchArticleIDs, existingSummary); err != nil {
				log.Printf("[WARN] Failed to mark articles with existing summary %d: %v", existingSummary.ID, err)
			}
			if err := topicextraction.TagSummary(existingSummary); err != nil {
				log.Printf("[WARN] Failed to backfill tags for existing summary %d: %v", existingSummary.ID, err)
			}
			if err := topicextraction.BackfillArticleTags(batch, displayName, categoryName); err != nil {
				log.Printf("[WARN] Failed to backfill article tags for existing feed %d batch %d: %v", *feedID, batchNum, err)
			}
			lastSummary = existingSummary
			continue
		}

		title := displayName + " - " + time.Now().Format("2006-01-02 15:04")
		if totalBatches > 1 {
			title = fmt.Sprintf("%s (第%d/%d部分)", title, batchNum, totalBatches)
		}
		title = title + " 新闻汇总"

		articlesText := joinStrings(articleTexts, "\n---\n")

		preferenceService := preferences.NewPreferenceService(database.DB)
		promptBuilder := NewAISummaryPromptBuilder(preferenceService, database.DB)
		summaryPrompt, promptContext, err := promptBuilder.BuildPersonalizedPrompt(displayName, categoryName, articlesText, len(batch), "zh")
		if err != nil {
			return nil, &SummaryError{Code: "PROMPT_BUILD_FAILED", Message: "构建总结提示词失败: " + err.Error()}
		}

		log.Printf(
			"Queue summary prompt built for feed=%s batch=%d/%d personalized=%t preferred_feeds=%d preferred_categories=%d",
			displayName, batchNum, totalBatches,
			promptContext.Personalized,
			promptContext.FeedCount,
			promptContext.CategoryCount,
		)

		summaryText, err := callQueueSummaryModel(config, summaryPrompt, buildQueueSummaryRequestMeta(feedID, displayName, categoryName, timeRange, batchNum, totalBatches, batchArticleIDs))
		if err != nil {
			return nil, err
		}

		aiSummary := models.AISummary{
			FeedID:       feedID,
			CategoryID:   categoryID,
			Title:        title,
			Summary:      summaryText,
			Articles:     string(articleIDsJSON),
			ArticleCount: len(batch),
			TimeRange:    timeRange,
		}

		if err := database.DB.Create(&aiSummary).Error; err != nil {
			return nil, &SummaryError{Code: "DB_ERROR", Message: "保存总结失败: " + err.Error()}
		}
		if err := MarkArticlesWithFeedSummary(batchArticleIDs, &aiSummary); err != nil {
			log.Printf("[WARN] Failed to mark articles with summary %d: %v", aiSummary.ID, err)
		}
		if err := topicextraction.TagSummary(&aiSummary); err != nil {
			log.Printf("[WARN] Failed to tag summary %d: %v", aiSummary.ID, err)
		}

		if err := topicextraction.BackfillArticleTags(batch, displayName, categoryName); err != nil {
			log.Printf("[WARN] Failed to backfill article tags for feed %d batch %d: %v", *feedID, batchNum, err)
		}

		lastSummary = &aiSummary
		log.Printf("Successfully generated queue summary for feed %d batch %d/%d (ID: %d)", *feedID, batchNum, totalBatches, aiSummary.ID)
	}

	log.Printf("Completed %d summary batches for feed %d, total %d articles", totalBatches, *feedID, len(articles))
	return lastSummary, nil
}

func chunkQueueArticles(articles []models.Article, chunkSize int) [][]models.Article {
	if len(articles) <= chunkSize {
		return [][]models.Article{articles}
	}

	chunks := make([][]models.Article, 0, (len(articles)+chunkSize-1)/chunkSize)
	for i := 0; i < len(articles); i += chunkSize {
		end := i + chunkSize
		if end > len(articles) {
			end = len(articles)
		}
		chunks = append(chunks, articles[i:end])
	}
	return chunks
}

func buildQueueArticleText(article models.Article) string {
	text := "标题: " + article.Title + "\n"
	if article.Description != "" {
		text += "描述: " + truncateQueueArticleText(article.Description, 1200) + "\n"
	}

	content := strings.TrimSpace(article.FirecrawlContent)
	if content == "" {
		content = strings.TrimSpace(article.Content)
	}
	if content != "" {
		text += "内容: " + truncateQueueArticleText(content, 2400) + "\n"
	}

	text += "链接: " + article.Link + "\n"
	return text
}

func truncateQueueArticleText(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	return text[:limit]
}

func buildQueueSummaryRequestMeta(feedID *uint, feedName string, categoryName string, timeRange int, batchNum int, totalBatches int, articleIDs []uint) map[string]any {
	meta := map[string]any{
		"article_count": len(articleIDs),
		"article_ids":   articleIDs,
		"batch_num":     batchNum,
		"category_name": categoryName,
		"feed_name":     feedName,
		"source":        "summary_queue",
		"time_range":    timeRange,
		"total_batches": totalBatches,
	}
	if feedID != nil {
		meta["feed_id"] = *feedID
	}
	return meta
}

func findQueueSummaryBatch(feedID *uint, articleIDsJSON string) (*models.AISummary, error) {
	if feedID == nil || articleIDsJSON == "" {
		return nil, nil
	}

	var summaries []models.AISummary
	err := database.DB.Where("feed_id = ? AND articles = ?", *feedID, articleIDsJSON).Order("id DESC").Limit(1).Find(&summaries).Error
	if err != nil {
		return nil, err
	}
	if len(summaries) == 0 {
		return nil, nil
	}

	return &summaries[0], nil
}

type SummaryError struct {
	Code    string
	Message string
}

func (e *SummaryError) Error() string {
	return e.Message
}

func classifyError(err error) string {
	if summaryErr, ok := err.(*SummaryError); ok {
		return summaryErr.Code
	}
	return "UNKNOWN"
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

func callQueueSummaryModel(config AIConfig, prompt string, metadata map[string]any) (string, error) {
	if strings.TrimSpace(config.BaseURL) != "" && strings.TrimSpace(config.APIKey) != "" && strings.TrimSpace(config.Model) != "" {
		return callDirectSummaryModel(config, prompt)
	}
	result, err := requestQueueSummaryChat(prompt, metadata)
	if err != nil {
		return "", &SummaryError{Code: "AI_ERROR", Message: "AI 路由调用失败: " + err.Error()}
	}
	if strings.TrimSpace(result) == "" {
		return "", &SummaryError{Code: "NO_RESPONSE", Message: "AI 未返回内容"}
	}
	return result, nil
}

func callDirectSummaryModel(config AIConfig, prompt string) (string, error) {
	type openAIMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	type openAIRequest struct {
		Model       string          `json:"model"`
		Messages    []openAIMessage `json:"messages"`
		Temperature float64         `json:"temperature"`
		MaxTokens   int             `json:"max_tokens"`
	}

	type openAIResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error,omitempty"`
	}

	reqBody := openAIRequest{
		Model: config.Model,
		Messages: []openAIMessage{
			{Role: "system", Content: "你是一名专业的新闻分析助手，擅长总结并比较多篇文章。"},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.7,
		MaxTokens:   16000,
	}

	body, _ := json.Marshal(reqBody)
	httpReq, err := http.NewRequest("POST", strings.TrimRight(config.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", &SummaryError{Code: "REQUEST_FAILED", Message: "创建请求失败: " + err.Error()}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+config.APIKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", &SummaryError{Code: "API_ERROR", Message: "AI API 调用失败: " + err.Error()}
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var openAIResp openAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return "", &SummaryError{Code: "PARSE_ERROR", Message: "解析 AI 响应失败: " + err.Error()}
	}
	if openAIResp.Error != nil {
		return "", &SummaryError{Code: "AI_ERROR", Message: "AI API 错误: " + openAIResp.Error.Message}
	}
	if len(openAIResp.Choices) == 0 {
		return "", &SummaryError{Code: "NO_RESPONSE", Message: "AI 未返回内容"}
	}
	return openAIResp.Choices[0].Message.Content, nil
}

func generateBatchID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

func generateJobID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36) + strconv.Itoa(int(time.Now().UnixNano()%1000))
}
