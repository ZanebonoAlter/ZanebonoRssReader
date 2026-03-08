package schedulers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
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
	TimeRange int    `json:"time_range"`
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

	settingsJSON, _, err := services.LoadSummaryConfig()
	if err != nil {
		return fmt.Errorf("failed to load AI config: %w", err)
	}

	settingsJSON["base_url"] = config.BaseURL
	settingsJSON["api_key"] = config.APIKey
	settingsJSON["model"] = config.Model
	settingsJSON["time_range"] = config.TimeRange

	if err := services.SaveSummaryConfig(settingsJSON, "AI summary generation configuration"); err != nil {
		return fmt.Errorf("failed to save AI config: %w", err)
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
	if err := s.loadAIConfig(); err != nil {
		log.Printf("AI config not available, skipping summary generation: %v", err)
		s.updateSchedulerStatus("idle", "AI config not set", nil)
		return
	}

	if !s.executionMutex.TryLock() {
		log.Println("Summary generation already in progress, skipping this cycle")
		return
	}
	defer s.executionMutex.Unlock()

	s.isExecuting = true
	defer func() {
		s.isExecuting = false
		if r := recover(); r != nil {
			log.Printf("[ERROR] PANIC in checkAndGenerateSummaries: %v", r)
			s.updateSchedulerStatus("idle", fmt.Sprintf("Panic: %v", r), nil)
		}
	}()

	log.Println("Starting auto-summary generation cycle")

	startTime := time.Now()
	s.updateSchedulerStatus("running", "", nil)

	var feeds []models.Feed
	if err := database.DB.Where("ai_summary_enabled = ?", true).Preload("Category").Find(&feeds).Error; err != nil {
		log.Printf("Error fetching feeds: %v", err)
		s.updateSchedulerStatus("idle", err.Error(), &startTime)
		return
	}

	if len(feeds) == 0 {
		log.Println("No feeds with AI summary enabled found, skipping")
		s.updateSchedulerStatus("idle", "No feeds enabled", &startTime)
		return
	}

	log.Printf("Found %d feeds to process", len(feeds))

	successCount := 0
	failedCount := 0

	for i, feed := range feeds {
		log.Printf("Processing feed %d/%d: %s (ID: %d)", i+1, len(feeds), feed.Title, feed.ID)

		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[ERROR] PANIC recovered while processing feed %d (%s): %v", feed.ID, feed.Title, r)
					failedCount++
				}
			}()

			result, err := s.generateSummaryForFeed(&feed)
			if err != nil {
				log.Printf("[ERROR] Error generating summary for feed %d (%s): %v", feed.ID, feed.Title, err)
				failedCount++
			} else if result {
				log.Printf("[OK] Successfully generated summary for feed %d (%s)", feed.ID, feed.Title)
				successCount++
			} else {
				log.Printf("[SKIP] Skipped feed %d (%s) - no content to summarize", feed.ID, feed.Title)
			}
		}()
	}

	duration := time.Since(startTime)
	resultMsg := fmt.Sprintf(
		"Completed: %d generated, %d skipped, %d failed in %v",
		successCount,
		len(feeds)-successCount-failedCount,
		failedCount,
		duration,
	)

	log.Printf("Auto-summary cycle completed: %s", resultMsg)
	s.updateSchedulerStatus("idle", "", &startTime)
}

func (s *AutoSummaryScheduler) generateSummaryForFeed(feed *models.Feed) (bool, error) {
	if s.aiConfig == nil {
		return false, fmt.Errorf("AI config not set")
	}

	timeRange := s.aiConfig.TimeRange
	if timeRange <= 0 {
		timeRange = 180
	}
	timeThreshold := time.Now().Add(-time.Duration(timeRange) * time.Minute)
	log.Printf("Using time range: %d minutes (threshold: %s)", timeRange, timeThreshold.Format("2006-01-02 15:04:05"))

	var articles []models.Article
	if err := database.DB.Where("feed_id = ? AND pub_date >= ?", feed.ID, timeThreshold).
		Order("pub_date DESC").
		Find(&articles).Error; err != nil {
		return false, fmt.Errorf("failed to fetch articles: %w", err)
	}

	if len(articles) == 0 {
		log.Printf("No recent articles found for feed %d in the last %d minutes", feed.ID, timeRange)
		return false, nil
	}

	log.Printf("Found %d articles for feed %d", len(articles), feed.ID)

	articleTexts := make([]string, 0, len(articles))
	for i, article := range articles {
		if i >= 80 {
			break
		}

		text := "Title: " + article.Title + "\n"
		if article.Description != "" {
			maxDesc := 1200
			if len(article.Description) < maxDesc {
				maxDesc = len(article.Description)
			}
			text += "Description: " + article.Description[:maxDesc] + "\n"
		}
		if article.Content != "" {
			maxContent := 2400
			if len(article.Content) < maxContent {
				maxContent = len(article.Content)
			}
			text += "Content: " + article.Content[:maxContent] + "\n"
		}
		text += "Link: " + article.Link + "\n"
		articleTexts = append(articleTexts, text)
	}

	feedName := feed.Title
	if feedName == "" {
		feedName = "Unknown feed"
	}

	categoryName := ""
	if feed.Category != nil {
		categoryName = feed.Category.Name
	}

	title := feedName + " - " + time.Now().Format("2006-01-02 15:04") + " News Summary"
	articlesText := joinStrings(articleTexts, "\n---\n")

	preferenceService := services.NewPreferenceService(database.DB)
	promptBuilder := services.NewAISummaryPromptBuilder(preferenceService, database.DB)
	summaryPrompt, promptContext, err := promptBuilder.BuildPersonalizedPrompt(feedName, categoryName, articlesText, len(articles), "en")
	if err != nil {
		return false, fmt.Errorf("failed to build prompt: %w", err)
	}

	log.Printf(
		"Auto summary prompt built for feed=%s personalized=%t preferred_feeds=%d preferred_categories=%d",
		feedName,
		promptContext.Personalized,
		promptContext.FeedCount,
		promptContext.CategoryCount,
	)

	summaryText, err := s.callAI(summaryPrompt)
	if err != nil {
		return false, fmt.Errorf("AI API call failed: %w", err)
	}

	articleIDs := make([]uint, len(articles))
	for i, article := range articles {
		articleIDs[i] = article.ID
	}
	articleIDsJSON, _ := json.Marshal(articleIDs)

	var categoryID *uint
	if feed.CategoryID != nil {
		catIDVal := *feed.CategoryID
		categoryID = &catIDVal
	}

	aiSummary := models.AISummary{
		FeedID:       &feed.ID,
		CategoryID:   categoryID,
		Title:        title,
		Summary:      summaryText,
		Articles:     string(articleIDsJSON),
		ArticleCount: len(articles),
		TimeRange:    timeRange,
	}

	if err := database.DB.Create(&aiSummary).Error; err != nil {
		return false, fmt.Errorf("failed to save summary: %w", err)
	}

	log.Printf("Successfully generated and saved summary for feed %d (ID: %d)", feed.ID, aiSummary.ID)
	return true, nil
}

func buildFeedSummaryPrompt(feedName string, categoryName string, articleCount int, articlesText string) string {
	catInfo := ""
	if categoryName != "" {
		catInfo = " (Category: " + categoryName + ")"
	}

	return `Please summarize the following ` + strconv.Itoa(articleCount) + ` articles from "` + feedName + `"` + catInfo + `.

Articles (newest first):
` + articlesText + `

Use this format:

## Core Theme
Summarize the main theme in one sentence.

## Important News

### Top Stories
List 2-3 key stories. For each story include:
- A bold title
- A short explanation in 2-3 sentences
- A source citation in the form > [Article Title](Link)

### Other News
List the other important stories. For each story include:
- A bold title
- A short explanation in 1-2 sentences
- A source citation in the form > [Article Title](Link)

## Key Takeaways
Summarize 3-5 important takeaways or trends.

## Tags
#` + feedName + ` #tag1 #tag2 #tag3

Important:
1. Every news item must include a source citation.
2. Use the format > [Article Title](Article Link).
3. Keep the summary concise and focused.
4. Stay objective and neutral.
5. Include the feed name as one of the tags: #` + feedName
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
			{Role: "system", Content: "You are a professional news analysis assistant who summarizes and compares multiple articles."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.7,
		MaxTokens:   16000,
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
		return "", fmt.Errorf(
			"AI API error: %s (type: %s, code: %s)",
			openAIResp.Error.Message,
			openAIResp.Error.Type,
			openAIResp.Error.Code,
		)
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

			nextExecution := now.Add(s.checkInterval)
			task.NextExecutionTime = &nextExecution
		}

		database.DB.Save(&task)
	} else {
		nextExecution := now.Add(s.checkInterval)
		var durationFloat *float64
		if startTime != nil {
			d := time.Since(*startTime)
			df := float64(d.Seconds())
			durationFloat = &df
		}

		task = models.SchedulerTask{
			Name:                  "auto_summary",
			Description:           "Auto-generate AI summaries for feeds",
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

	var task models.SchedulerTask
	err := database.DB.Where("name = ?", "auto_summary").First(&task).Error

	status := map[string]interface{}{
		"status": func() string {
			if s.isExecuting {
				return "running"
			}
			if s.isRunning {
				return "idle"
			}
			return "stopped"
		}(),
		"check_interval": int(s.checkInterval.Seconds()),
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
