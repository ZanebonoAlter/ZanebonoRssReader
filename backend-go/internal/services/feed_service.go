package services

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"my-robot-backend/internal/models"
	"my-robot-backend/pkg/database"
)

type FeedService struct {
	rssParser *RSSParser
}

func NewFeedService() *FeedService {
	return &FeedService{
		rssParser: NewRSSParser(),
	}
}

func (s *FeedService) RefreshFeed(feedID uint) error {
	var feed models.Feed
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
	feed.RefreshStatus = "success"
	feed.RefreshError = ""

	if feed.Icon == "" || feed.Icon == "rss" {
		if parsed.Image != "" {
			feed.Icon = parsed.Image
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
		err := database.DB.Where("feed_id = ? AND link = ?", feed.ID, entry.Link).First(&existingArticle).Error
		if err == nil {
			continue
		}

		if err != gorm.ErrRecordNotFound {
			continue
		}

		article := models.Article{
			FeedID:      feed.ID,
			Title:       entry.Title,
			Description: entry.Description,
			Content:     entry.Content,
			Link:        entry.Link,
			PubDate:     entry.PubDate,
			Author:      entry.Author,
		}

		if article.PubDate == nil {
			now := time.Now().In(time.FixedZone("CST", 8*3600))
			article.PubDate = &now
		}

		if err := database.DB.Create(&article).Error; err != nil {
			continue
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

func (s *FeedService) updateFeedError(feed *models.Feed, err error) {
	now := time.Now().In(time.FixedZone("CST", 8*3600))
	feed.RefreshStatus = "error"
	feed.RefreshError = err.Error()
	feed.LastRefreshAt = &now
	database.DB.Save(feed)
}

func (s *FeedService) cleanupOldArticles(feed *models.Feed) {
	var articles []models.Article
	if err := database.DB.Where("feed_id = ? AND favorite = ?", feed.ID, false).Order("pub_date DESC").Find(&articles).Error; err != nil {
		return
	}

	if len(articles) > feed.MaxArticles {
		toDelete := articles[feed.MaxArticles:]
		for _, article := range toDelete {
			database.DB.Delete(&article)
		}
	}
}

func (s *FeedService) FetchFeedPreview(feedURL string) (title, description string, err error) {
	return s.rssParser.FetchFeedMetadata(feedURL)
}
