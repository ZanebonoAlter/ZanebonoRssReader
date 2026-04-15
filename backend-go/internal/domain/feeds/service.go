package feeds

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	otelCodes "go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
	"my-robot-backend/internal/domain/contentprocessing"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/domain/topicextraction"
	"my-robot-backend/internal/platform/database"
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

	articlesAdded := 0
	for _, entry := range parsed.Entries {
		if entry.Link == "" {
			continue
		}

		var existingArticle models.Article
		err := database.DB.Where("feed_id = ? AND title = ?", feed.ID, entry.Title).First(&existingArticle).Error
		if err == nil {
			continue
		}

		if err != gorm.ErrRecordNotFound {
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

		if err := s.enqueueArticleProcessing(feed, article); err != nil {
			log.Printf("Error enqueueing processing for article %d (feed %d): %v", article.ID, feed.ID, err)
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
		ArticleID: article.ID,
		FeedName:  feed.Title,
		Reason:    "article_created",
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
	var articles []models.Article
	if err := database.DB.Omit("tag_count", "relevance_score").Where("feed_id = ?", feed.ID).Order("pub_date DESC").Find(&articles).Error; err != nil {
		return
	}

	if len(articles) <= feed.MaxArticles {
		return
	}

	articlesToDelete := len(articles) - feed.MaxArticles
	for i := len(articles) - 1; i >= 0 && articlesToDelete > 0; i-- {
		article := articles[i]
		if article.Favorite {
			continue
		}

		if article.FirecrawlStatus == "pending" || article.FirecrawlStatus == "processing" || article.SummaryStatus == "incomplete" || article.SummaryStatus == "pending" {
			continue
		}

		database.DB.Where("article_id = ?", article.ID).Delete(&models.ReadingBehavior{})
		database.DB.Delete(&article)
		articlesToDelete--
	}
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
