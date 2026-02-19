package services

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"my-robot-backend/internal/models"
	"my-robot-backend/pkg/database"
)

type ContentCompletionService struct {
	crawlClient *Crawl4AIClient
	aiService   *AIService
}

func NewContentCompletionService(crawlBaseURL string) *ContentCompletionService {
	return &ContentCompletionService{
		crawlClient: NewCrawl4AIClient(crawlBaseURL),
		aiService:   NewAIService("", "", ""),
	}
}

func (s *ContentCompletionService) SetAICredentials(baseURL, apiKey, model string) {
	s.aiService = NewAIService(baseURL, apiKey, model)
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
	var article models.Article
	if err := database.DB.First(&article, articleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("article not found")
		}
		return fmt.Errorf("failed to fetch article: %w", err)
	}

	if article.Link == "" {
		return fmt.Errorf("article has no link to crawl")
	}

	if article.ContentStatus == "complete" && article.FullContent != "" {
		return nil
	}

	var feed models.Feed
	if err := database.DB.First(&feed, article.FeedID).Error; err != nil {
		return fmt.Errorf("failed to fetch feed: %w", err)
	}

	if !feed.ContentCompletionEnabled {
		return fmt.Errorf("content completion not enabled for this feed")
	}

	if article.CompletionAttempts >= feed.MaxCompletionRetries {
		article.ContentStatus = "failed"
		article.CompletionError = "Max retries exceeded"
		database.DB.Save(&article)
		return fmt.Errorf("max completion retries exceeded")
	}

	article.ContentStatus = "pending"
	article.CompletionAttempts++
	database.DB.Save(&article)

	crawlResp, err := s.crawlClient.CrawlURL(article.Link, true)
	if err != nil {
		article.ContentStatus = "incomplete"
		article.CompletionError = err.Error()
		database.DB.Save(&article)
		return fmt.Errorf("crawl failed: %w", err)
	}

	if !crawlResp.Success {
		article.ContentStatus = "incomplete"
		article.CompletionError = crawlResp.Error
		database.DB.Save(&article)
		return fmt.Errorf("crawl failed: %s", crawlResp.Error)
	}

	now := time.Now().In(time.FixedZone("CST", 8*3600))
	article.FullContent = crawlResp.Markdown
	article.ContentFetchedAt = &now
	article.ContentStatus = "complete"
	article.CompletionError = ""

	if err := database.DB.Save(&article).Error; err != nil {
		return fmt.Errorf("failed to save article: %w", err)
	}

	if s.aiService != nil && s.aiService.BaseURL != "" && s.aiService.APIKey != "" {
		contentToSummarize := article.FullContent
		if contentToSummarize == "" {
			contentToSummarize = article.Content
		}

		if contentToSummarize != "" {
			summary, err := s.aiService.SummarizeArticle(article.Title, contentToSummarize, "zh")
			if err == nil && summary != nil {
				article.AIContentSummary = formatAISummary(summary)
				database.DB.Save(&article)
			}
		}
	}

	return nil
}

func (s *ContentCompletionService) AutoCompletePendingArticles(limit int) ([]uint, []error) {
	var articles []models.Article

	err := database.DB.
		Where("content_status = ?", "incomplete").
		Preload("Feed").
		Limit(limit).
		Find(&articles).Error

	if err != nil {
		return nil, []error{fmt.Errorf("failed to fetch articles: %w", err)}
	}

	var completedIDs []uint
	var errors []error

	for _, article := range articles {
		if !article.Feed.ContentCompletionEnabled {
			continue
		}

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

func formatAISummary(summary *AISummaryResponse) string {
	var result strings.Builder

	result.WriteString("## AI 总结\n\n")

	if summary.OneSentence != "" {
		result.WriteString(fmt.Sprintf("**一句话总结**: %s\n\n", summary.OneSentence))
	}

	if len(summary.KeyPoints) > 0 {
		result.WriteString("**核心观点**:\n")
		for _, point := range summary.KeyPoints {
			result.WriteString(fmt.Sprintf("- %s\n", point))
		}
		result.WriteString("\n")
	}

	if len(summary.Takeaways) > 0 {
		result.WriteString("**关键要点**:\n")
		for i, takeaway := range summary.Takeaways {
			result.WriteString(fmt.Sprintf("%d. %s\n", i+1, takeaway))
		}
		result.WriteString("\n")
	}

	if len(summary.Tags) > 0 {
		result.WriteString("**标签**: ")
		result.WriteString(strings.Join(summary.Tags, ", "))
	}

	return result.String()
}
