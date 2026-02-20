package services

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"my-robot-backend/internal/models"
	"my-robot-backend/internal/ws"
	"my-robot-backend/pkg/database"
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

func (q *SummaryQueue) SubmitBatch(categoryIDs []uint, config AIConfig) *SummaryBatch {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.running {
		close(q.stopChan)
		q.stopChan = make(chan struct{})
	}

	var feeds []models.Feed
	if len(categoryIDs) > 0 {
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
		if i >= 50 {
			break
		}

		text := "标题: " + article.Title + "\n"
		if article.Description != "" {
			maxDesc := 500
			if len(article.Description) < maxDesc {
				maxDesc = len(article.Description)
			}
			text += "描述: " + article.Description[:maxDesc] + "\n"
		}
		if article.Content != "" {
			maxContent := 1000
			if len(article.Content) < maxContent {
				maxContent = len(article.Content)
			}
			text += "内容: " + article.Content[:maxContent] + "\n"
		}
		text += "链接: " + article.Link + "\n"
		articleTexts = append(articleTexts, text)
	}

	displayName := feedName
	if displayName == "" {
		displayName = "未知订阅源"
	}

	title := displayName + " - " + time.Now().Format("2006-01-02 15:04") + " 新闻汇总"

	articlesText := joinStrings(articleTexts, "\n---\n")
	summaryPrompt := buildFeedSummaryPrompt(displayName, categoryName, len(articles), articlesText)

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
			{Role: "system", Content: "你是一个专业的新闻分析助手，擅长汇总和分析多篇文章。"},
			{Role: "user", Content: summaryPrompt},
		},
		Temperature: 0.7,
		MaxTokens:   3000,
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
		return nil, &SummaryError{Code: "API_ERROR", Message: "AI API调用失败: " + err.Error()}
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var openAIResp openAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return nil, &SummaryError{Code: "PARSE_ERROR", Message: "解析AI响应失败: " + err.Error()}
	}

	if openAIResp.Error != nil {
		return nil, &SummaryError{Code: "AI_ERROR", Message: "AI API错误: " + openAIResp.Error.Message}
	}

	if len(openAIResp.Choices) == 0 {
		return nil, &SummaryError{Code: "NO_RESPONSE", Message: "AI未返回响应"}
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

	return &aiSummary, nil
}

func buildFeedSummaryPrompt(feedName string, categoryName string, articleCount int, articlesText string) string {
	catInfo := ""
	if categoryName != "" {
		catInfo = "（属于分类：" + categoryName + "）"
	}

	return `请对以下来自"` + feedName + `"订阅源` + catInfo + `的 ` + strconv.Itoa(articleCount) + ` 篇文章进行汇总总结。

文章列表（按时间倒序）：
` + articlesText + `

请提供以下格式的总结：

## 核心主题
用一句话概括这批文章的核心主题和趋势。

## 重要新闻

### 🔥 热点事件
列出2-3个最重要的事件，每个事件包含：
- 事件标题（用加粗）
- 简要说明（2-3句话）
- 引文标注新闻来源（使用 > [文章标题](链接) 格式）

### 📰 其他重要新闻
列出其他重要新闻，每条包含：
- 新闻标题（用加粗）
- 简要说明（1-2句话）
- 引文标注新闻来源（使用 > [文章标题](链接) 格式）

## 核心观点
总结3-5个核心观点或趋势，每个观点用简洁的语言表达。

## 相关标签
#` + feedName + ` #标签1 #标签2 #标签3

**重要提醒**：
1. 必须为每条新闻标注来源，使用引文格式
2. 来源格式：> [文章标题](文章链接)
3. 确保总结简洁明了，突出重点
4. 保持客观中立的语气
5. 标签中必须包含订阅源名称：#` + feedName
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
