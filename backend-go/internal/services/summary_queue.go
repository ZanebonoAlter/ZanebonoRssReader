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

// SummaryJobStatus 任务状态
type SummaryJobStatus string

const (
	JobPending    SummaryJobStatus = "pending"
	JobProcessing SummaryJobStatus = "processing"
	JobCompleted  SummaryJobStatus = "completed"
	JobFailed     SummaryJobStatus = "failed"
)

// SummaryJob 单个总结任务
type SummaryJob struct {
	ID           string           `json:"id"`
	BatchID      string           `json:"batch_id"`
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

// SummaryBatch 批次信息
type SummaryBatch struct {
	ID            string        `json:"id"`
	Status        string        `json:"status"` // pending/processing/completed
	TotalJobs     int           `json:"total_jobs"`
	CompletedJobs int           `json:"completed_jobs"`
	FailedJobs    int           `json:"failed_jobs"`
	CreatedAt     time.Time     `json:"created_at"`
	CompletedAt   *time.Time    `json:"completed_at,omitempty"`
	Jobs          []*SummaryJob `json:"jobs"`
}

// SummaryQueue 总结队列管理器
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

// GetSummaryQueue 获取队列单例
func GetSummaryQueue() *SummaryQueue {
	queueOnce.Do(func() {
		queueInstance = &SummaryQueue{
			stopChan: make(chan struct{}),
		}
	})
	return queueInstance
}

// AIConfig AI配置
type AIConfig struct {
	BaseURL   string
	APIKey    string
	Model     string
	TimeRange int
}

// SubmitBatch 提交新批次（覆盖旧批次）
func (q *SummaryQueue) SubmitBatch(categoryIDs []uint, config AIConfig) *SummaryBatch {
	q.mu.Lock()
	defer q.mu.Unlock()

	// 停止旧批次（如果有）
	if q.running {
		close(q.stopChan)
		q.stopChan = make(chan struct{})
	}

	// 获取分类名称
	categoryNames := make(map[uint]string)
	var categories []models.Category
	database.DB.Find(&categories)
	for _, cat := range categories {
		categoryNames[cat.ID] = cat.Name
	}

	batchID := generateBatchID()
	jobs := make([]*SummaryJob, 0, len(categoryIDs))
	now := time.Now()

	for _, catID := range categoryIDs {
		catName := categoryNames[catID]
		if catName == "" {
			catName = "未知分类"
		}
		job := &SummaryJob{
			ID:           generateJobID(),
			BatchID:      batchID,
			CategoryID:   &catID,
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

	// 启动处理
	q.running = true
	go q.processBatch(config)

	return q.currentBatch
}

// GetCurrentBatch 获取当前批次
func (q *SummaryQueue) GetCurrentBatch() *SummaryBatch {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.currentBatch
}

// GetJob 获取单个任务
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

// broadcastProgress 广播进度更新
func (q *SummaryQueue) broadcastProgress(currentJob *SummaryJob) {
	if q.currentBatch == nil {
		return
	}

	hub := ws.GetHub()

	// 构建所有任务的更新列表
	jobs := make([]ws.JobUpdate, len(q.currentBatch.Jobs))
	for i, job := range q.currentBatch.Jobs {
		jobs[i] = ws.JobUpdate{
			ID:           job.ID,
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

// processBatch 处理批次（单并发）
func (q *SummaryQueue) processBatch(config AIConfig) {
	q.mu.Lock()
	q.currentBatch.Status = "processing"
	q.mu.Unlock()

	// 广播开始处理
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

	// 广播完成
	q.broadcastProgress(nil)
}

// processJob 处理单个任务
func (q *SummaryQueue) processJob(job *SummaryJob, config AIConfig) {
	// 更新状态为处理中
	q.mu.Lock()
	job.Status = JobProcessing
	job.UpdatedAt = time.Now()
	q.mu.Unlock()

	// 广播开始处理
	q.broadcastProgress(job)

	// 执行总结
	result, err := q.generateSummaryForCategory(job.CategoryID, config)

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

	// 广播完成
	q.broadcastProgress(job)
}

// generateSummaryForCategory 为单个分类生成总结
func (q *SummaryQueue) generateSummaryForCategory(categoryID *uint, config AIConfig) (*models.AISummary, error) {
	timeRange := config.TimeRange
	if timeRange == 0 {
		timeRange = 180
	}

	timeThreshold := time.Now().Add(-time.Duration(timeRange) * time.Minute)

	// 获取文章
	var articles []models.Article
	if categoryID != nil {
		var feeds []models.Feed
		database.DB.Where("category_id = ? AND ai_summary_enabled = ?", *categoryID, true).Find(&feeds)

		feedIDs := make([]uint, len(feeds))
		for i, feed := range feeds {
			feedIDs[i] = feed.ID
		}

		database.DB.Where("feed_id IN ? AND pub_date >= ?", feedIDs, timeThreshold).
			Order("pub_date DESC").
			Find(&articles)
	} else {
		var feeds []models.Feed
		database.DB.Where("ai_summary_enabled = ?", true).Find(&feeds)

		feedIDs := make([]uint, len(feeds))
		for i, feed := range feeds {
			feedIDs[i] = feed.ID
		}

		database.DB.Where("feed_id IN ? AND pub_date >= ?", feedIDs, timeThreshold).
			Order("pub_date DESC").
			Find(&articles)
	}

	if len(articles) == 0 {
		return nil, &SummaryError{Code: "NO_ARTICLES", Message: "该分类下没有找到文章"}
	}

	// 准备文章内容
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

	// 生成标题
	categoryName := "全部分类"
	if categoryID != nil {
		var category models.Category
		database.DB.First(&category, *categoryID)
		categoryName = category.Name
	}

	title := categoryName + " - " + time.Now().Format("2006-01-02 15:04") + " 新闻汇总"

	// 准备提示词
	articlesText := joinStrings(articleTexts, "\n---\n")
	summaryPrompt := `请对以下来自"` + categoryName + `"分类的 ` + strconv.Itoa(len(articles)) + ` 篇文章进行汇总总结。

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
- 引文标注新闻来源（使用 > [来源名称](链接) 格式）

### 📰 其他重要新闻
列出其他重要新闻，每条包含：
- 新闻标题（用加粗）
- 简要说明（1-2句话）
- 引文标注新闻来源（使用 > [来源名称](链接) 格式）

## 核心观点
总结3-5个核心观点或趋势，每个观点用简洁的语言表达。

## 相关标签
#标签1 #标签2 #标签3

**重要提醒**：
1. 必须为每条新闻标注来源，使用引文格式
2. 来源格式：> [来源订阅源名称](文章链接)
3. 确保总结简洁明了，突出重点
4. 保持客观中立的语气`

	// 调用AI API
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

	// 保存到数据库
	articleIDs := make([]uint, len(articles))
	for i, article := range articles {
		articleIDs[i] = article.ID
	}
	articleIDsJSON, _ := json.Marshal(articleIDs)

	aiSummary := models.AISummary{
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

// SummaryError 自定义错误类型
type SummaryError struct {
	Code    string
	Message string
}

func (e *SummaryError) Error() string {
	return e.Message
}

// classifyError 分类错误
func classifyError(err error) string {
	if summaryErr, ok := err.(*SummaryError); ok {
		return summaryErr.Code
	}
	return "UNKNOWN"
}

// joinStrings 连接字符串
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

// generateBatchID 生成批次ID
func generateBatchID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

// generateJobID 生成任务ID
func generateJobID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36) + strconv.Itoa(int(time.Now().UnixNano()%1000))
}
