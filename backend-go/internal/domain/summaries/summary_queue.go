package summaries

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/domain/preferences"
	"my-robot-backend/internal/domain/topicgraph"
	"my-robot-backend/internal/platform/database"
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
	go q.processBatch(config)

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

func (q *SummaryQueue) processBatch(config AIConfig) {
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

		q.processJob(job, config)
	}

	q.mu.Lock()
	q.currentBatch.Status = "completed"
	now := time.Now()
	q.currentBatch.CompletedAt = &now
	q.running = false
	q.mu.Unlock()

	q.broadcastProgress(nil)
}

func (q *SummaryQueue) processJob(job *SummaryJob, config AIConfig) {
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
	} else {
		job.Status = JobCompleted
		job.ResultID = &result.ID
		q.currentBatch.CompletedJobs++
	}

	q.mu.Unlock()

	q.broadcastProgress(job)
}

func (q *SummaryQueue) generateSummaryForFeed(feedID *uint, categoryID *uint, feedName string, categoryName string, config AIConfig) (*models.AISummary, error) {
	timeRange := config.TimeRange
	if timeRange == 0 {
		timeRange = 180
	}

	timeThreshold := time.Now().Add(-time.Duration(timeRange) * time.Minute)

	var articles []models.Article
	if feedID != nil {
		database.DB.Where("feed_id = ? AND pub_date >= ?", *feedID, timeThreshold).
			Order("pub_date DESC").
			Find(&articles)
	} else {
		return nil, &SummaryError{Code: "NO_FEED", Message: "未指定订阅源"}
	}

	if len(articles) == 0 {
		return nil, &SummaryError{Code: "NO_ARTICLES", Message: "该订阅源下没有找到文章"}
	}

	articleTexts := make([]string, 0, len(articles))
	for i, article := range articles {
		if i >= 80 {
			break
		}

		articleTexts = append(articleTexts, buildQueueArticleText(article))
	}

	displayName := feedName
	if displayName == "" {
		displayName = "未知订阅源"
	}

	title := displayName + " - " + time.Now().Format("2006-01-02 15:04") + " 新闻汇总"
	articlesText := joinStrings(articleTexts, "\n---\n")

	preferenceService := preferences.NewPreferenceService(database.DB)
	promptBuilder := NewAISummaryPromptBuilder(preferenceService, database.DB)
	summaryPrompt, promptContext, err := promptBuilder.BuildPersonalizedPrompt(displayName, categoryName, articlesText, len(articles), "zh")
	if err != nil {
		return nil, &SummaryError{Code: "PROMPT_BUILD_FAILED", Message: "构建总结提示词失败: " + err.Error()}
	}

	log.Printf(
		"Queue summary prompt built for feed=%s personalized=%t preferred_feeds=%d preferred_categories=%d",
		displayName,
		promptContext.Personalized,
		promptContext.FeedCount,
		promptContext.CategoryCount,
	)

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
			{Role: "user", Content: summaryPrompt},
		},
		Temperature: 0.7,
		MaxTokens:   16000,
	}

	body, _ := json.Marshal(reqBody)
	httpReq, err := http.NewRequest("POST", config.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, &SummaryError{Code: "REQUEST_FAILED", Message: "创建请求失败: " + err.Error()}
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+config.APIKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, &SummaryError{Code: "API_ERROR", Message: "AI API 调用失败: " + err.Error()}
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var openAIResp openAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return nil, &SummaryError{Code: "PARSE_ERROR", Message: "解析 AI 响应失败: " + err.Error()}
	}

	if openAIResp.Error != nil {
		return nil, &SummaryError{Code: "AI_ERROR", Message: "AI API 错误: " + openAIResp.Error.Message}
	}

	if len(openAIResp.Choices) == 0 {
		return nil, &SummaryError{Code: "NO_RESPONSE", Message: "AI 未返回内容"}
	}

	summaryText := openAIResp.Choices[0].Message.Content

	articleIDs := make([]uint, len(articles))
	for i, article := range articles {
		articleIDs[i] = article.ID
	}
	articleIDsJSON, _ := json.Marshal(articleIDs)

	aiSummary := models.AISummary{
		FeedID:       feedID,
		CategoryID:   categoryID,
		Title:        title,
		Summary:      summaryText,
		Articles:     string(articleIDsJSON),
		ArticleCount: len(articles),
		TimeRange:    timeRange,
	}

	if err := database.DB.Create(&aiSummary).Error; err != nil {
		return nil, &SummaryError{Code: "DB_ERROR", Message: "保存总结失败: " + err.Error()}
	}

	if err := topicgraph.TagSummary(&aiSummary); err != nil {
		log.Printf("[WARN] Failed to tag summary %d: %v", aiSummary.ID, err)
	}

	return &aiSummary, nil
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

func generateBatchID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

func generateJobID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36) + strconv.Itoa(int(time.Now().UnixNano()%1000))
}
