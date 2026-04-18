package feeds

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	otelCodes "go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
	"my-robot-backend/internal/domain/contentprocessing"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/domain/topicextraction"
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/logging"
)

type FeedService struct {
	rssParser *RSSParser
}

func NewFeedService() *FeedService {
	return &FeedService{
		rssParser: NewRSSParser(),
	}
}

func (s *FeedService) RefreshFeed(ctx context.Context, feedID uint) (err error) {
	ctx, span := otel.Tracer("rss-reader-backend").Start(ctx, "FeedService.RefreshFeed")
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(otelCodes.Error, "error")
			span.RecordError(err)
		}
	}()
	/*line backend-go/internal/domain/feeds/service.go:26:2*/ var feed models.Feed
	if err := database.DB.First(&feed, feedID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("feed not found")
		}
		return err
	}

	parsed, err := s.rssParser.ParseFeedURL(feed.URL)
	if err != nil {
		s.updateFeedError(&feed, err)
		return err
	}

	now := time.Now().In(time.FixedZone("CST", 8*3600))
	feed.Title = parsed.Title
	feed.Description = parsed.Description
	feed.LastUpdated = &now
	feed.LastRefreshAt = &now
	feed.RefreshStatus = "success"
	feed.RefreshError = ""

	var firstArticleImage string
	for _, entry := range parsed.Entries {
		if entry.ImageURL != "" && firstArticleImage == "" {
			firstArticleImage = entry.ImageURL
			break
		}
	}

	if feed.Icon == "" || feed.Icon == "rss" || feed.Icon == "mdi:rss" {
		if parsed.Image != "" {
			feed.Icon = parsed.Image
		} else if firstArticleImage != "" {
			feed.Icon = firstArticleImage
		} else {
			feed.Icon = s.rssParser.FetchFaviconURL(feed.URL)
		}
	}

	var existingTitles []string
	database.DB.Model(&models.Article{}).
		Where("feed_id = ?", feed.ID).
		Pluck("title", &existingTitles)
	titleSet := make(map[string]bool, len(existingTitles))
	for _, t := range existingTitles {
		titleSet[t] = true
	}

	articlesAdded := 0
	for _, entry := range parsed.Entries {
		if entry.Link == "" {
			continue
		}

		if titleSet[entry.Title] {
			continue
		}

		article := s.buildArticleFromEntry(feed, entry)

		if article.PubDate == nil {
			now := time.Now().In(time.FixedZone("CST", 8*3600))
			article.PubDate = &now
		}

		if err := database.DB.Create(&article).Error; err != nil {
			continue
		}

		titleSet[entry.Title] = true

		if err := s.enqueueArticleProcessing(feed, article); err != nil {
			logging.Errorf("Error enqueueing processing for article %d (feed %d): %v", article.ID, feed.ID, err)
		}

		articlesAdded++
		if articlesAdded >= feed.MaxArticles {
			break
		}
	}

	s.cleanupOldArticles(&feed)

	if err := database.DB.Save(&feed).Error; err != nil {
		return err
	}

	return nil
}

func (s *FeedService) enqueueArticleProcessing(feed models.Feed, article models.Article) error {
	if shouldDelayArticleTagging(feed) {
		return contentprocessing.NewFirecrawlJobQueue(database.DB).Enqueue(article)
	}

	return topicextraction.NewTagJobQueue(database.DB).Enqueue(topicextraction.TagJobRequest{
		ArticleID:    article.ID,
		FeedName:     feed.Title,
		CategoryName: topicextraction.FeedCategoryName(feed),
		Reason:       "article_created",
	})
}

func (s *FeedService) updateFeedError(feed *models.Feed, err error) {
	now := time.Now().In(time.FixedZone("CST", 8*3600))
	feed.RefreshStatus = "error"
	feed.RefreshError = err.Error()
	feed.LastRefreshAt = &now
	database.DB.Save(feed)
}

func (s *FeedService) cleanupOldArticles(feed *models.Feed) {
	var articleCount int64
	database.DB.Model(&models.Article{}).Where("feed_id = ?", feed.ID).Count(&articleCount)

	if int(articleCount) <= feed.MaxArticles {
		return
	}

	var allArticles []struct {
		ID              uint
		Favorite        bool
		FirecrawlStatus string
		SummaryStatus   string
	}
	database.DB.Model(&models.Article{}).
		Select("id, favorite, firecrawl_status, summary_status").
		Where("feed_id = ?", feed.ID).
		Order("pub_date DESC").
		Find(&allArticles)

	keepIDs := make([]uint, 0)
	candidates := make([]uint, 0)

	for _, a := range allArticles {
		isActive := a.FirecrawlStatus == "pending" || a.FirecrawlStatus == "processing" ||
			a.SummaryStatus == "incomplete" || a.SummaryStatus == "pending"

		if a.Favorite || isActive {
			keepIDs = append(keepIDs, a.ID)
		} else {
			candidates = append(candidates, a.ID)
		}
	}

	remaining := feed.MaxArticles - len(keepIDs)
	if remaining > 0 {
		keepFromCandidates := candidates
		if len(candidates) > remaining {
			keepFromCandidates = candidates[:remaining]
		}
		keepIDs = append(keepIDs, keepFromCandidates...)
	}

	if len(keepIDs) == 0 {
		return
	}

	database.DB.Where("article_id IN (SELECT id FROM articles WHERE feed_id = ? AND id NOT IN ?)", feed.ID, keepIDs).Delete(&models.ReadingBehavior{})
	database.DB.Where("feed_id = ? AND id NOT IN ?", feed.ID, keepIDs).Delete(&models.Article{})
}

func (s *FeedService) FetchFeedPreview(feedURL string) (title, description string, err error) {
	return s.rssParser.FetchFeedMetadata(feedURL)
}

func (s *FeedService) buildArticleFromEntry(feed models.Feed, entry ParsedEntry) models.Article {
	article := models.Article{
		FeedID:        feed.ID,
		Title:         entry.Title,
		Description:   entry.Description,
		Content:       entry.Content,
		Link:          entry.Link,
		ImageURL:      entry.ImageURL,
		PubDate:       entry.PubDate,
		Author:        entry.Author,
		SummaryStatus: "complete",
	}

	if feed.FirecrawlEnabled {
		article.FirecrawlStatus = "pending"
		if feed.ArticleSummaryEnabled {
			article.SummaryStatus = "incomplete"
		}
	} else if feed.ArticleSummaryEnabled {
		// STAT-03: 只开启摘要不开启Firecrawl时，设置为pending等待手动触发摘要
		article.SummaryStatus = "pending"
	}

	return article
}

func shouldDelayArticleTagging(feed models.Feed) bool {
	return feed.FirecrawlEnabled
}
