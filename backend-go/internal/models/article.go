package models

import (
	"time"
)

type Article struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	FeedID             uint       `gorm:"index;not null" json:"feed_id"`
	Title              string     `gorm:"size:500;not null" json:"title"`
	Description        string     `gorm:"type:text" json:"description"`
	Content            string     `gorm:"type:text" json:"content"`
	Link               string     `gorm:"size:1000" json:"link"`
	ImageURL           string     `gorm:"size:1000" json:"image_url"`
	PubDate            *time.Time `json:"pub_date"`
	Author             string     `gorm:"size:200" json:"author"`
	Read               bool       `gorm:"default:false" json:"read"`
	Favorite           bool       `gorm:"default:false" json:"favorite"`
	ContentStatus      string     `gorm:"size:20;default:complete" json:"content_status"`
	FullContent        string     `gorm:"type:text" json:"full_content"`
	ContentFetchedAt   *time.Time `json:"content_fetched_at"`
	CompletionAttempts int        `gorm:"default:0" json:"completion_attempts"`
	CompletionError    string     `gorm:"type:text" json:"completion_error"`
	AIContentSummary   string     `gorm:"type:text" json:"ai_content_summary"`
	FirecrawlEnabled   bool       `gorm:"default:false" json:"firecrawl_enabled"`
	FirecrawlStatus    string     `gorm:"size:20;default:pending" json:"firecrawl_status"`
	FirecrawlError     string     `gorm:"type:text" json:"firecrawl_error"`
	FirecrawlContent   string     `gorm:"type:text" json:"firecrawl_content"`
	FirecrawlCrawledAt *time.Time `json:"firecrawl_crawled_at"`
	CreatedAt          time.Time  `json:"created_at"`
	Feed               Feed       `gorm:"foreignKey:FeedID" json:"feed,omitempty"`
}

func (a *Article) ToDict() map[string]interface{} {
	return map[string]interface{}{
		"id":                   a.ID,
		"feed_id":              a.FeedID,
		"title":                a.Title,
		"description":          a.Description,
		"content":              a.Content,
		"link":                 a.Link,
		"image_url":            a.ImageURL,
		"pub_date":             FormatDatetimeCSTPtr(a.PubDate),
		"author":               a.Author,
		"read":                 a.Read,
		"favorite":             a.Favorite,
		"content_status":       a.ContentStatus,
		"full_content":         a.FullContent,
		"content_fetched_at":   FormatDatetimeCSTPtr(a.ContentFetchedAt),
		"completion_attempts":  a.CompletionAttempts,
		"completion_error":     a.CompletionError,
		"ai_content_summary":   a.AIContentSummary,
		"firecrawl_enabled":    a.FirecrawlEnabled,
		"firecrawl_status":     a.FirecrawlStatus,
		"firecrawl_error":      a.FirecrawlError,
		"firecrawl_content":    a.FirecrawlContent,
		"firecrawl_crawled_at": FormatDatetimeCSTPtr(a.FirecrawlCrawledAt),
		"created_at":           FormatDatetimeCST(a.CreatedAt),
	}
}
