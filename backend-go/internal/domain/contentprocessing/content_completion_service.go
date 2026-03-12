package contentprocessing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"my-robot-backend/internal/domain/models"
	platformai "my-robot-backend/internal/platform/ai"
	"my-robot-backend/internal/platform/airouter"
	"my-robot-backend/internal/platform/database"
)

type ContentCompletionService struct {
	crawlClient *Crawl4AIClient
	aiService   *platformai.AIService
	router      *airouter.Router
}

type ContentCompletionArticleRef struct {
	ID     uint   `json:"id"`
	FeedID uint   `json:"feed_id"`
	Title  string `json:"title"`
}

type ContentCompletionOverview struct {
	PendingCount           int                             `json:"pending_count"`
	ProcessingCount        int                             `json:"processing_count"`
	LiveProcessingCount    int                             `json:"live_processing_count"`
	StaleProcessingCount   int                             `json:"stale_processing_count"`
	CompletedCount         int                             `json:"completed_count"`
	FailedCount            int                             `json:"failed_count"`
	BlockedCount           int                             `json:"blocked_count"`
	TotalCount             int                             `json:"total_count"`
	AIConfigured           bool                            `json:"ai_configured"`
	BlockedReasons         ContentCompletionBlockedReasons `json:"blocked_reasons"`
	StaleProcessingArticle *ContentCompletionArticleRef    `json:"stale_processing_article"`
}

type ContentCompletionBlockedReasons struct {
	WaitingForFirecrawlCount    int `json:"waiting_for_firecrawl_count"`
	FeedDisabledCount           int `json:"feed_disabled_count"`
	AIUnconfiguredCount         int `json:"ai_unconfigured_count"`
	ReadyButMissingContentCount int `json:"ready_but_missing_content_count"`
}

func NewContentCompletionService(crawlBaseURL string) *ContentCompletionService {
	return &ContentCompletionService{
		crawlClient: NewCrawl4AIClient(crawlBaseURL),
		aiService:   platformai.NewAIService("", "", ""),
		router:      airouter.NewRouter(),
	}
}

func (s *ContentCompletionService) SetAICredentials(baseURL, apiKey, model string) {
	s.aiService = platformai.NewAIService(baseURL, apiKey, model)
}

func (s *ContentCompletionService) SetCrawlAPIToken(token string) {
	s.crawlClient.SetAPIToken(token)
}

func (s *ContentCompletionService) IsContentIncomplete(article *models.Article) bool {
	if article.ContentStatus == "complete" || article.ContentStatus == "failed" {
		return false
	}

	content := strings.TrimSpace(article.Content)
	if content == "" || content == strings.TrimSpace(article.Description) {
		return true
	}

	if len(content) < 200 {
		return true
	}

	return false
}

func (s *ContentCompletionService) CompleteArticle(articleID uint) error {
	return s.CompleteArticleWithForce(articleID, false)
}

func (s *ContentCompletionService) CompleteArticleWithForce(articleID uint, force bool) error {
	var article models.Article
	if err := database.DB.First(&article, articleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("article not found")
		}
		return fmt.Errorf("failed to fetch article: %w", err)
	}

	if article.ContentStatus == "complete" && !force {
		return nil
	}

	var feed models.Feed
	if err := database.DB.First(&feed, article.FeedID).Error; err != nil {
		return fmt.Errorf("failed to fetch feed: %w", err)
	}

	if !feed.ContentCompletionEnabled {
		return fmt.Errorf("AI summary not enabled for this feed")
	}

	if article.FirecrawlStatus != "completed" {
		return fmt.Errorf("firecrawl not completed for this article")
	}

	if article.CompletionAttempts >= feed.MaxCompletionRetries && !force {
		article.ContentStatus = "failed"
		article.CompletionError = "Max retries exceeded"
		database.DB.Save(&article)
		return fmt.Errorf("max completion retries exceeded")
	}

	if force {
		article.AIContentSummary = ""
		article.ContentFetchedAt = nil
	}

	article.ContentStatus = "pending"
	article.CompletionAttempts++
	article.CompletionError = ""
	database.DB.Save(&article)

	if s.aiService == nil || s.aiService.BaseURL == "" || s.aiService.APIKey == "" {
		if !s.hasRouteConfig() {
			article.ContentStatus = "failed"
			article.CompletionError = "AI service not configured"
			database.DB.Save(&article)
			return fmt.Errorf("AI service not configured")
		}
	}

	contentToSummarize := article.FirecrawlContent
	if contentToSummarize == "" {
		article.ContentStatus = "failed"
		article.CompletionError = "No firecrawl content available"
		database.DB.Save(&article)
		return fmt.Errorf("no firecrawl content available")
	}

	summary, err := s.summarizeContent(article.Title, contentToSummarize)
	if err != nil {
		article.ContentStatus = "failed"
		article.CompletionError = err.Error()
		database.DB.Save(&article)
		return fmt.Errorf("AI summarization failed: %w", err)
	}

	now := time.Now().In(time.FixedZone("CST", 8*3600))
	article.AIContentSummary = formatAISummary(summary)
	article.ContentStatus = "complete"
	article.CompletionError = ""
	article.ContentFetchedAt = &now

	if err := database.DB.Save(&article).Error; err != nil {
		return fmt.Errorf("failed to save article: %w", err)
	}

	return nil
}

func (s *ContentCompletionService) AutoCompletePendingArticles(limit int) ([]uint, []error) {
	var articles []models.Article

	err := database.DB.
		Joins("JOIN feeds ON feeds.id = articles.feed_id").
		Where("articles.firecrawl_status = ? AND articles.content_status = ?", "completed", "incomplete").
		Where("feeds.content_completion_enabled = ?", true).
		Preload("Feed").
		Limit(limit).
		Find(&articles).Error

	if err != nil {
		return nil, []error{fmt.Errorf("failed to fetch articles: %w", err)}
	}

	var completedIDs []uint
	var errors []error

	for _, article := range articles {
		if article.CompletionAttempts >= article.Feed.MaxCompletionRetries {
			continue
		}

		if err := s.CompleteArticle(article.ID); err != nil {
			errors = append(errors, fmt.Errorf("article %d: %w", article.ID, err))
		} else {
			completedIDs = append(completedIDs, article.ID)
		}
	}

	return completedIDs, errors
}

func (s *ContentCompletionService) CheckAndMarkIncompleteArticles(feedID uint) (int, error) {
	var articles []models.Article
	if err := database.DB.Where("feed_id = ?", feedID).Find(&articles).Error; err != nil {
		return 0, err
	}

	count := 0
	for _, article := range articles {
		if s.IsContentIncomplete(&article) && article.ContentStatus != "failed" {
			article.ContentStatus = "incomplete"
			database.DB.Save(&article)
			count++
		}
	}

	return count, nil
}

func (s *ContentCompletionService) GetOverview() (*ContentCompletionOverview, error) {
	overview := &ContentCompletionOverview{}
	overview.AIConfigured = (s.aiService != nil && s.aiService.BaseURL != "" && s.aiService.APIKey != "") || s.hasRouteConfig()

	countQuery := []struct {
		assign func(int64)
		query  func(*gorm.DB) *gorm.DB
	}{
		{
			assign: func(count int64) { overview.PendingCount = int(count) },
			query: func(db *gorm.DB) *gorm.DB {
				return db.Model(&models.Article{}).
					Joins("JOIN feeds ON feeds.id = articles.feed_id").
					Where("articles.firecrawl_status = ? AND articles.content_status = ?", "completed", "incomplete").
					Where("feeds.content_completion_enabled = ?", true)
			},
		},
		{
			assign: func(count int64) { overview.ProcessingCount = int(count) },
			query: func(db *gorm.DB) *gorm.DB {
				return db.Model(&models.Article{}).Where("content_status = ?", "pending")
			},
		},
		{
			assign: func(count int64) { overview.CompletedCount = int(count) },
			query: func(db *gorm.DB) *gorm.DB {
				return db.Model(&models.Article{}).Where("content_status = ?", "complete")
			},
		},
		{
			assign: func(count int64) { overview.FailedCount = int(count) },
			query: func(db *gorm.DB) *gorm.DB {
				return db.Model(&models.Article{}).Where("content_status = ?", "failed")
			},
		},
		{
			assign: func(count int64) { overview.BlockedCount = int(count) },
			query: func(db *gorm.DB) *gorm.DB {
				return db.Model(&models.Article{}).
					Joins("JOIN feeds ON feeds.id = articles.feed_id").
					Where("articles.content_status = ?", "incomplete").
					Where("articles.firecrawl_status <> ? OR feeds.content_completion_enabled = ?", "completed", false)
			},
		},
		{
			assign: func(count int64) { overview.TotalCount = int(count) },
			query: func(db *gorm.DB) *gorm.DB {
				return db.Model(&models.Article{})
			},
		},
	}

	for _, item := range countQuery {
		var count int64
		if err := item.query(database.DB).Count(&count).Error; err != nil {
			return nil, err
		}
		item.assign(count)
	}

	overview.StaleProcessingCount = overview.ProcessingCount
	overview.LiveProcessingCount = 0

	var staleArticle models.Article
	if err := database.DB.Where("content_status = ?", "pending").Order("created_at ASC").First(&staleArticle).Error; err == nil {
		overview.StaleProcessingArticle = ToArticleRef(staleArticle)
	}

	blockedQueries := []struct {
		assign func(int64)
		query  func(*gorm.DB) *gorm.DB
	}{
		{
			assign: func(count int64) { overview.BlockedReasons.WaitingForFirecrawlCount = int(count) },
			query: func(db *gorm.DB) *gorm.DB {
				return db.Model(&models.Article{}).
					Joins("JOIN feeds ON feeds.id = articles.feed_id").
					Where("articles.content_status = ?", "incomplete").
					Where("feeds.content_completion_enabled = ?", true).
					Where("articles.firecrawl_status <> ?", "completed")
			},
		},
		{
			assign: func(count int64) { overview.BlockedReasons.FeedDisabledCount = int(count) },
			query: func(db *gorm.DB) *gorm.DB {
				return db.Model(&models.Article{}).
					Joins("JOIN feeds ON feeds.id = articles.feed_id").
					Where("articles.content_status = ?", "incomplete").
					Where("feeds.content_completion_enabled = ?", false)
			},
		},
		{
			assign: func(count int64) { overview.BlockedReasons.ReadyButMissingContentCount = int(count) },
			query: func(db *gorm.DB) *gorm.DB {
				return db.Model(&models.Article{}).
					Joins("JOIN feeds ON feeds.id = articles.feed_id").
					Where("articles.firecrawl_status = ? AND articles.content_status = ?", "completed", "incomplete").
					Where("feeds.content_completion_enabled = ?", true).
					Where("TRIM(COALESCE(articles.firecrawl_content, '')) = ''")
			},
		},
	}

	for _, item := range blockedQueries {
		var count int64
		if err := item.query(database.DB).Count(&count).Error; err != nil {
			return nil, err
		}
		item.assign(count)
	}

	if !overview.AIConfigured {
		overview.BlockedReasons.AIUnconfiguredCount = overview.PendingCount
	}

	return overview, nil
}

func (s *ContentCompletionService) hasRouteConfig() bool {
	if s.router == nil {
		return false
	}
	provider, _, err := s.router.ResolvePrimaryProvider(airouter.CapabilityArticleCompletion)
	return err == nil && provider != nil && strings.TrimSpace(provider.APIKey) != ""
}

func (s *ContentCompletionService) summarizeContent(title, content string) (*platformai.AISummaryResponse, error) {
	if s.router != nil {
		maxTokens := 16000
		result, err := s.router.Chat(context.Background(), airouter.ChatRequest{
			Capability: airouter.CapabilityArticleCompletion,
			Messages: []airouter.Message{
				{Role: "system", Content: s.aiService.GetSystemPrompt("zh")},
				{Role: "user", Content: s.aiService.PrepareArticleContent(title, content)},
			},
			MaxTokens: &maxTokens,
			Metadata: map[string]any{
				"title": title,
			},
		})
		if err == nil {
			return platformai.ParseSummaryMarkdown(result.Content), nil
		}
		if s.aiService == nil || s.aiService.BaseURL == "" || s.aiService.APIKey == "" {
			return nil, err
		}
	}

	return s.aiService.SummarizeArticle(title, content, "zh")
}

func ToArticleRef(article models.Article) *ContentCompletionArticleRef {
	return &ContentCompletionArticleRef{
		ID:     article.ID,
		FeedID: article.FeedID,
		Title:  article.Title,
	}
}

func formatAISummary(summary *platformai.AISummaryResponse) string {
	if summary == nil {
		return ""
	}

	if strings.TrimSpace(summary.Markdown) != "" {
		return strings.TrimSpace(summary.Markdown)
	}

	var result strings.Builder
	result.WriteString("# 内容整理\n\n")

	if summary.OneSentence != "" {
		result.WriteString(fmt.Sprintf("> %s\n\n", summary.OneSentence))
	}

	if len(summary.KeyPoints) > 0 {
		result.WriteString("## 关键点\n\n")
		for _, point := range summary.KeyPoints {
			result.WriteString(fmt.Sprintf("- %s\n", point))
		}
		result.WriteString("\n")
	}

	if len(summary.Takeaways) > 0 {
		result.WriteString("## 补充说明\n\n")
		for i, takeaway := range summary.Takeaways {
			result.WriteString(fmt.Sprintf("%d. %s\n", i+1, takeaway))
		}
		result.WriteString("\n")
	}

	if len(summary.Tags) > 0 {
		result.WriteString("## 标签\n\n")
		for _, tag := range summary.Tags {
			result.WriteString(fmt.Sprintf("- %s\n", tag))
		}
	}

	return result.String()
}
