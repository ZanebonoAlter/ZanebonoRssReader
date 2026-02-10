package schedulers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"my-robot-backend/internal/models"
	"my-robot-backend/internal/services"
	"my-robot-backend/pkg/database"
)

type AutoSummaryScheduler struct {
	cron           *cron.Cron
	checkInterval  time.Duration
	isRunning      bool
	aiConfig       *AIConfig
	executionMutex sync.Mutex
	isExecuting    bool
}

type AIConfig struct {
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key"`
	Model     string `json:"model"`
	TimeRange int    `json:"time_range"` // Time range in minutes for fetching articles
}

type GenerateSummaryRequest struct {
	CategoryID *uint  `json:"category_id"`
	TimeRange  int    `json:"time_range"`
	BaseURL    string `json:"base_url"`
	APIKey     string `json:"api_key"`
	Model      string `json:"model"`
}

func NewAutoSummaryScheduler(checkInterval int) *AutoSummaryScheduler {
	return &AutoSummaryScheduler{
		cron:          cron.New(),
		checkInterval: time.Duration(checkInterval) * time.Second,
		isRunning:     false,
	}
}

func (s *AutoSummaryScheduler) Start() error {
	if s.isRunning {
		return fmt.Errorf("auto-summary scheduler already running")
	}

	// Load AI config from database on startup
	if err := s.loadAIConfig(); err != nil {
		log.Printf("Warning: Failed to load AI config: %v", err)
	}

	scheduleExpr := fmt.Sprintf("@every %ds", int64(s.checkInterval.Seconds()))
	if _, err := s.cron.AddFunc(scheduleExpr, s.checkAndGenerateSummaries); err != nil {
		return fmt.Errorf("failed to schedule auto-summary: %w", err)
	}

	s.cron.Start()
	s.isRunning = true
	log.Printf("Auto-summary scheduler started with interval: %v", s.checkInterval)

	return nil
}

func (s *AutoSummaryScheduler) Stop() {
	if !s.isRunning {
		return
	}

	s.cron.Stop()
	s.isRunning = false
	log.Println("Auto-summary scheduler stopped")
}

func (s *AutoSummaryScheduler) SetAIConfig(baseURL, apiKey, model string, timeRange int) error {
	// Default to 180 minutes if not specified
	if timeRange <= 0 {
		timeRange = 180
	}

	config := &AIConfig{
		BaseURL:   baseURL,
		APIKey:    apiKey,
		Model:     model,
		TimeRange: timeRange,
	}

	s.aiConfig = config

	// Persist to database
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal AI config: %w", err)
	}

	var settings models.AISettings
	err = database.DB.Where("key = ?", "summary_config").First(&settings).Error

	if err == nil {
		settings.Value = string(configJSON)
		database.DB.Save(&settings)
	} else {
		settings = models.AISettings{
			Key:         "summary_config",
			Value:       string(configJSON),
			Description: "AI summary generation configuration",
		}
		database.DB.Create(&settings)
	}

	log.Println("AI configuration updated and saved to database")
	return nil
}

func (s *AutoSummaryScheduler) loadAIConfig() error {
	var settings models.AISettings
	err := database.DB.Where("key = ?", "summary_config").First(&settings).Error
	if err != nil {
		return fmt.Errorf("AI config not found in database")
	}

	var config AIConfig
	if err := json.Unmarshal([]byte(settings.Value), &config); err != nil {
		return fmt.Errorf("failed to parse AI config: %w", err)
	}

	s.aiConfig = &config
	log.Println("AI configuration loaded from database")
	return nil
}

func (s *AutoSummaryScheduler) checkAndGenerateSummaries() {
	// Load latest config from database
	if err := s.loadAIConfig(); err != nil {
		log.Printf("AI config not available, skipping summary generation: %v", err)
		s.updateSchedulerStatus("idle", "AI config not set", nil)
		return
	}

	// Check if already executing
	if !s.executionMutex.TryLock() {
		log.Println("Summary generation already in progress, skipping this cycle")
		return
	}
	defer s.executionMutex.Unlock()

	s.isExecuting = true
	defer func() {
		s.isExecuting = false
		// Global panic recovery
		if r := recover(); r != nil {
			log.Printf("❌ PANIC in checkAndGenerateSummaries: %v", r)
			s.updateSchedulerStatus("idle", fmt.Sprintf("Panic: %v", r), nil)
		}
	}()

	log.Println("Starting auto-summary generation cycle")

	startTime := time.Now()

	// Update scheduler status to running
	s.updateSchedulerStatus("running", "", nil)

	// Get all categories
	var categories []models.Category
	if err := database.DB.Find(&categories).Error; err != nil {
		log.Printf("Error fetching categories: %v", err)
		s.updateSchedulerStatus("idle", err.Error(), &startTime)
		return
	}

	if len(categories) == 0 {
		log.Println("No categories found, skipping summary generation")
		s.updateSchedulerStatus("idle", "No categories", &startTime)
		return
	}

	log.Printf("Found %d categories to process", len(categories))

	successCount := 0
	failedCount := 0

	// Generate summary for each category
	for i, category := range categories {
		log.Printf("Processing category %d/%d: %s (ID: %d)", i+1, len(categories), category.Name, category.ID)

		func() {
			// Add panic recovery for each category to prevent one failure from stopping all
			defer func() {
				if r := recover(); r != nil {
					log.Printf("❌ PANIC recovered while processing category %d (%s): %v", category.ID, category.Name, r)
					failedCount++
				}
			}()

			result, err := s.generateSummaryForCategory(category.ID)
			if err != nil {
				log.Printf("❌ Error generating summary for category %d (%s): %v", category.ID, category.Name, err)
				failedCount++
			} else if result {
				log.Printf("✓ Successfully generated summary for category %d (%s)", category.ID, category.Name)
				successCount++
			} else {
				log.Printf("⊘ Skipped category %d (%s) - no content to summarize", category.ID, category.Name)
			}
		}()
	}

	duration := time.Since(startTime)
	resultMsg := fmt.Sprintf("Completed: %d generated, %d skipped, %d failed in %v", successCount, len(categories)-successCount-failedCount, failedCount, duration)

	log.Printf("Auto-summary cycle completed: %s", resultMsg)
	s.updateSchedulerStatus("idle", "", &startTime)
}

func (s *AutoSummaryScheduler) generateSummaryForCategory(categoryID uint) (bool, error) {
	if s.aiConfig == nil {
		return false, fmt.Errorf("AI config not set")
	}

	categoryName := fmt.Sprintf("分类 ID %d", categoryID)
	log.Printf("Starting summary generation for %s", categoryName)

	// Use time range from config (default 180 minutes if not set)
	timeRange := s.aiConfig.TimeRange
	if timeRange <= 0 {
		timeRange = 180
	}
	timeThreshold := time.Now().Add(-time.Duration(timeRange) * time.Minute)
	log.Printf("Using time range: %d minutes (threshold: %s)", timeRange, timeThreshold.Format("2006-01-02 15:04:05"))

	// Get feeds in this category with AI summary enabled
	var feeds []models.Feed
	if err := database.DB.Where("category_id = ? AND ai_summary_enabled = ?", categoryID, true).Find(&feeds).Error; err != nil {
		return false, fmt.Errorf("failed to fetch feeds: %w", err)
	}

	if len(feeds) == 0 {
		log.Printf("No feeds with AI summary enabled found for category %d", categoryID)
		return false, nil // Not an error, just nothing to do
	}

	feedIDs := make([]uint, len(feeds))
	for i, feed := range feeds {
		feedIDs[i] = feed.ID
	}

	// Get articles from these feeds
	var articles []models.Article
	if err := database.DB.Where("feed_id IN ? AND pub_date >= ?", feedIDs, timeThreshold).
		Order("pub_date DESC").
		Find(&articles).Error; err != nil {
		return false, fmt.Errorf("failed to fetch articles: %w", err)
	}

	if len(articles) == 0 {
		log.Printf("No recent articles found for category %d in the last %d minutes", categoryID, timeRange)
		return false, nil
	}

	log.Printf("Found %d articles for category %d", len(articles), categoryID)

	// Prepare article content
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

	// Get category name
	var category models.Category
	database.DB.First(&category, categoryID)
	categoryName = category.Name

	title := categoryName + " - " + time.Now().Format("2006-01-02 15:04") + " 新闻汇总"
	articlesText := joinStrings(articleTexts, "\n---\n")

	prefService := services.NewPreferenceService(database.DB)
	promptBuilder := services.NewAISummaryPromptBuilder(prefService, database.DB)
	summaryPrompt, _ := promptBuilder.BuildPersonalizedPrompt(
		categoryName,
		articlesText,
		len(articles),
		"zh",
	)

	// Call AI API
	summaryText, err := s.callAI(summaryPrompt)
	if err != nil {
		return false, fmt.Errorf("AI API call failed: %w", err)
	}

	// Save to database
	articleIDs := make([]uint, len(articles))
	for i, article := range articles {
		articleIDs[i] = article.ID
	}
	articleIDsJSON, _ := json.Marshal(articleIDs)

	aiSummary := models.AISummary{
		CategoryID:   &categoryID,
		Title:        title,
		Summary:      summaryText,
		Articles:     string(articleIDsJSON),
		ArticleCount: len(articles),
		TimeRange:    timeRange,
	}

	if err := database.DB.Create(&aiSummary).Error; err != nil {
		return false, fmt.Errorf("failed to save summary: %w", err)
	}

	log.Printf("Successfully generated and saved summary for category %d (ID: %d)", categoryID, aiSummary.ID)
	return true, nil
}

func (s *AutoSummaryScheduler) callAI(prompt string) (string, error) {
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
			Code    string `json:"code"`
		} `json:"error,omitempty"`
	}

	reqBody := openAIRequest{
		Model: s.aiConfig.Model,
		Messages: []openAIMessage{
			{Role: "system", Content: "你是一个专业的新闻分析助手，擅长汇总和分析多篇文章。"},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.7,
		MaxTokens:   3000,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", s.aiConfig.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.aiConfig.APIKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var openAIResp openAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		log.Printf("Failed to parse AI response (status %d): %s", resp.StatusCode, string(respBody))
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if openAIResp.Error != nil {
		return "", fmt.Errorf("AI API error: %s (type: %s, code: %s)",
			openAIResp.Error.Message, openAIResp.Error.Type, openAIResp.Error.Code)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	return openAIResp.Choices[0].Message.Content, nil
}

func (s *AutoSummaryScheduler) updateSchedulerStatus(status, lastError string, startTime *time.Time) {
	var task models.SchedulerTask
	err := database.DB.Where("name = ?", "auto_summary").First(&task).Error

	now := time.Now()

	if err == nil {
		task.Status = status
		task.LastError = lastError
		if lastError != "" {
			errTime := now
			task.LastErrorTime = &errTime
		}

		if startTime != nil {
			duration := time.Since(*startTime)
			durationFloat := float64(duration.Seconds())
			task.LastExecutionDuration = &durationFloat
			task.LastExecutionTime = &now

			if status == "idle" && lastError == "" {
				task.SuccessfulExecutions++
				task.ConsecutiveFailures = 0
			} else if lastError != "" {
				task.FailedExecutions++
				task.ConsecutiveFailures++
			}
			task.TotalExecutions++

			// Calculate next execution time
			nextExecution := now.Add(s.checkInterval)
			task.NextExecutionTime = &nextExecution
		}

		database.DB.Save(&task)
	} else {
		// Create new scheduler task record
		nextExecution := now.Add(s.checkInterval)
		var durationFloat *float64
		if startTime != nil {
			d := time.Since(*startTime)
			df := float64(d.Seconds())
			durationFloat = &df
		}

		task = models.SchedulerTask{
			Name:                  "auto_summary",
			Description:           "Auto-generate AI summaries for categories",
			CheckInterval:         int(s.checkInterval.Seconds()),
			Status:                status,
			LastError:             lastError,
			NextExecutionTime:     &nextExecution,
			LastExecutionDuration: durationFloat,
		}

		if startTime != nil {
			task.LastExecutionTime = &now
		}

		database.DB.Create(&task)
	}
}

func (s *AutoSummaryScheduler) GetStatus() map[string]interface{} {
	entries := s.cron.Entries()

	var nextRun time.Time
	if len(entries) > 0 {
		nextRun = entries[0].Next
	}

	// Get database state
	var task models.SchedulerTask
	err := database.DB.Where("name = ?", "auto_summary").First(&task).Error

	status := map[string]interface{}{
		"running":        s.isRunning,
		"check_interval": s.checkInterval.String(),
		"next_run":       nextRun.Format(time.RFC3339),
		"ai_configured":  s.aiConfig != nil,
		"is_executing":   s.isExecuting,
		"database_state": nil,
	}

	if err == nil {
		status["database_state"] = task.ToDict()
	}

	return status
}

func (s *AutoSummaryScheduler) IsExecuting() bool {
	return s.isExecuting
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
