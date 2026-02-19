package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"my-robot-backend/internal/models"
	"my-robot-backend/internal/services"
	"my-robot-backend/pkg/database"
)

type GenerateSummaryRequest struct {
	CategoryID *uint  `json:"category_id"`
	TimeRange  int    `json:"time_range"`
	BaseURL    string `json:"base_url"`
	APIKey     string `json:"api_key" binding:"required"`
	Model      string `json:"model"`
}

func GetSummaries(c *gin.Context) {
	categoryID, _ := strconv.Atoi(c.Query("category_id"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	query := database.DB.Model(&models.AISummary{})

	if categoryID > 0 {
		query = query.Where("category_id = ?", categoryID)
	}

	var total int64
	query.Count(&total)

	var summaries []models.AISummary
	offset := (page - 1) * perPage
	if err := query.Order("created_at DESC").Offset(offset).Limit(perPage).Find(&summaries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	data := make([]map[string]interface{}, len(summaries))
	for i, summary := range summaries {
		data[i] = summary.ToDict()
	}

	pages := int(total) / perPage
	if int(total)%perPage > 0 {
		pages++
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"pagination": gin.H{
			"page":     page,
			"per_page": perPage,
			"total":    total,
			"pages":    pages,
		},
	})
}

func GetSummary(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("summary_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid summary ID",
		})
		return
	}

	var summary models.AISummary
	if err := database.DB.Preload("Category").First(&summary, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Summary not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   err.Error(),
			})
		}
		return
	}

	summaryDict := summary.ToDict()

	// Parse article IDs and get details
	var articleIDs []uint
	if summary.Articles != "" {
		json.Unmarshal([]byte(summary.Articles), &articleIDs)
	}

	if len(articleIDs) > 0 {
		var articles []models.Article
		database.DB.Where("id IN ?", articleIDs).Find(&articles)

		articleDetails := make([]map[string]interface{}, len(articles))
		for i, article := range articles {
			articleDetails[i] = article.ToDict()
		}
		summaryDict["article_details"] = articleDetails
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summaryDict,
	})
}

func GenerateSummary(c *gin.Context) {
	var req GenerateSummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "API key is required",
		})
		return
	}

	categoryID := req.CategoryID
	timeRange := req.TimeRange
	if timeRange == 0 {
		timeRange = 180 // Default 3 hours
	}

	// Calculate time threshold
	timeThreshold := time.Now().Add(-time.Duration(timeRange) * time.Minute)

	// Get articles
	var articles []models.Article
	if categoryID != nil {
		// Get feeds in this category with AI summary enabled
		var feeds []models.Feed
		database.DB.Where("category_id = ? AND ai_summary_enabled = ?", *categoryID, true).Find(&feeds)

		feedIDs := make([]uint, len(feeds))
		for i, feed := range feeds {
			feedIDs[i] = feed.ID
		}

		// Get articles from these feeds
		database.DB.Where("feed_id IN ? AND pub_date >= ?", feedIDs, timeThreshold).
			Order("pub_date DESC").
			Find(&articles)
	} else {
		// Get all feeds with AI summary enabled
		var feeds []models.Feed
		database.DB.Where("ai_summary_enabled = ?", true).Find(&feeds)

		feedIDs := make([]uint, len(feeds))
		for i, feed := range feeds {
			feedIDs[i] = feed.ID
		}

		// Get articles from enabled feeds
		database.DB.Where("feed_id IN ? AND pub_date >= ?", feedIDs, timeThreshold).
			Order("pub_date DESC").
			Find(&articles)
	}

	if len(articles) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "在最近指定的时间范围内没有找到文章",
		})
		return
	}

	// Prepare article content
	articleTexts := make([]string, 0, len(articles))
	for i, article := range articles {
		if i >= 50 {
			break // Limit to 50 articles
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

	// Generate title
	categoryName := "全部分类"
	if categoryID != nil {
		var category models.Category
		database.DB.First(&category, *categoryID)
		categoryName = category.Name
	}

	title := categoryName + " - " + time.Now().Format("2006-01-02 15:04") + " 新闻汇总"

	// Prepare prompt for AI
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

	// Prepare AI request
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
		} `json:"error,omitempty"`
	}

	reqBody := openAIRequest{
		Model: req.Model,
		Messages: []openAIMessage{
			{Role: "system", Content: "你是一个专业的新闻分析助手，擅长汇总和分析多篇文章。"},
			{Role: "user", Content: summaryPrompt},
		},
		Temperature: 0.7,
		MaxTokens:   3000,
	}

	body, _ := json.Marshal(reqBody)
	httpReq, _ := http.NewRequest("POST", req.BaseURL+"/chat/completions", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "AI API调用失败: " + err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var openAIResp openAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "解析AI响应失败: " + err.Error(),
		})
		return
	}

	if openAIResp.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "AI API错误: " + openAIResp.Error.Message,
		})
		return
	}

	if len(openAIResp.Choices) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "AI未返回响应",
		})
		return
	}

	summaryText := openAIResp.Choices[0].Message.Content

	// Create AI summary record
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    aiSummary.ToDict(),
		"message": "成功生成 " + strconv.Itoa(len(articles)) + " 篇文章的汇总总结",
	})
}

func DeleteSummary(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("summary_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid summary ID",
		})
		return
	}

	var summary models.AISummary
	if err := database.DB.First(&summary, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Summary not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   err.Error(),
			})
		}
		return
	}

	if err := database.DB.Delete(&summary).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Summary deleted successfully",
	})
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

type AutoSummaryConfig struct {
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key"`
	Model     string `json:"model"`
	TimeRange int    `json:"time_range"` // Time range in minutes for auto-summary
}

func AutoGenerateSummary(c *gin.Context) {
	var req GenerateSummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "API key is required",
		})
		return
	}

	go func() {
		generateSummaryWorker(req)
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"success": true,
		"message": "后台生成任务已启动",
	})
}

func generateSummaryWorker(req GenerateSummaryRequest) {
	categoryID := req.CategoryID
	timeRange := req.TimeRange
	if timeRange == 0 {
		timeRange = 180
	}

	timeThreshold := time.Now().Add(-time.Duration(timeRange) * time.Minute)

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
		return
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

	categoryName := "全部分类"
	if categoryID != nil {
		var category models.Category
		database.DB.First(&category, *categoryID)
		categoryName = category.Name
	}

	title := categoryName + " - " + time.Now().Format("2006-01-02 15:04") + " 新闻汇总"
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
		} `json:"error,omitempty"`
	}

	reqBody := openAIRequest{
		Model: req.Model,
		Messages: []openAIMessage{
			{Role: "system", Content: "你是一个专业的新闻分析助手，擅长汇总和分析多篇文章。"},
			{Role: "user", Content: summaryPrompt},
		},
		Temperature: 0.7,
		MaxTokens:   3000,
	}

	body, _ := json.Marshal(reqBody)
	httpReq, _ := http.NewRequest("POST", req.BaseURL+"/chat/completions", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var openAIResp openAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return
	}

	if openAIResp.Error != nil || len(openAIResp.Choices) == 0 {
		return
	}

	summaryText := openAIResp.Choices[0].Message.Content

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

	database.DB.Create(&aiSummary)
}

func GetAutoSummaryStatus(c *gin.Context) {
	var settings models.AISettings
	err := database.DB.Where("key = ?", "summary_config").First(&settings).Error

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"enabled": false,
				"status":  "not_configured",
			},
		})
		return
	}

	var config AutoSummaryConfig
	if err := json.Unmarshal([]byte(settings.Value), &config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to parse configuration",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled":    true,
			"status":     "configured",
			"base_url":   config.BaseURL,
			"model":      config.Model,
			"time_range": config.TimeRange,
		},
	})
}

func UpdateAutoSummaryConfig(c *gin.Context) {
	var req AutoSummaryConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
		})
		return
	}

	// Save to database
	configJSON, _ := json.Marshal(req)

	var settings models.AISettings
	err := database.DB.Where("key = ?", "summary_config").First(&settings).Error

	if err == nil {
		settings.Value = string(configJSON)
		settings.Description = "AI summary generation configuration (including auto-summary)"
		database.DB.Save(&settings)
	} else {
		settings = models.AISettings{
			Key:         "summary_config",
			Value:       string(configJSON),
			Description: "AI summary generation configuration (including auto-summary)",
		}
		database.DB.Create(&settings)
	}

	// Also update in-memory scheduler config if available
	if AutoSummarySchedulerInterface != nil {
		if scheduler, ok := AutoSummarySchedulerInterface.(interface {
			SetAIConfig(baseURL, apiKey, model string, timeRange int) error
		}); ok {
			if err := scheduler.SetAIConfig(req.BaseURL, req.APIKey, req.Model, req.TimeRange); err != nil {
				// Log but don't fail the request
				// The config is saved in DB, scheduler will pick it up on next cycle
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Auto summary configuration updated successfully",
	})
}

// ========== 队列相关 API ==========

// QueueSummaryRequest 队列总结请求
type QueueSummaryRequest struct {
	CategoryIDs []uint `json:"category_ids" binding:"required,min=1"`
	TimeRange   int    `json:"time_range"`
	BaseURL     string `json:"base_url"`
	APIKey      string `json:"api_key" binding:"required"`
	Model       string `json:"model"`
}

// SubmitQueueSummary 提交队列总结任务
func SubmitQueueSummary(c *gin.Context) {
	var req QueueSummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "请求参数错误: " + err.Error(),
		})
		return
	}

	queue := services.GetSummaryQueue()
	config := services.AIConfig{
		BaseURL:   req.BaseURL,
		APIKey:    req.APIKey,
		Model:     req.Model,
		TimeRange: req.TimeRange,
	}

	batch := queue.SubmitBatch(req.CategoryIDs, config)

	c.JSON(http.StatusAccepted, gin.H{
		"success": true,
		"message": "总结任务已加入队列",
		"data":    batch,
	})
}

// GetQueueStatus 获取队列状态
func GetQueueStatus(c *gin.Context) {
	queue := services.GetSummaryQueue()
	batch := queue.GetCurrentBatch()

	if batch == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    batch,
	})
}

// GetQueueJob 获取单个任务详情
func GetQueueJob(c *gin.Context) {
	jobID := c.Param("job_id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "任务ID不能为空",
		})
		return
	}

	queue := services.GetSummaryQueue()
	job := queue.GetJob(jobID)

	if job == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "任务不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    job,
	})
}
