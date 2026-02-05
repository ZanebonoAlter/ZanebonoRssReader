package models

import (
	"time"
)

type Feed struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	Title            string     `gorm:"size:200;not null" json:"title"`
	Description      string     `gorm:"type:text" json:"description"`
	URL              string     `gorm:"size:500;unique;not null" json:"url"`
	CategoryID       *uint      `gorm:"index" json:"category_id"`
	Icon             string     `gorm:"size:50;default:rss" json:"icon"`
	Color            string     `gorm:"size:20;default:#8b5cf6" json:"color"`
	LastUpdated      *time.Time `json:"last_updated"`
	CreatedAt        time.Time  `json:"created_at"`
	MaxArticles      int        `gorm:"default:100" json:"max_articles"`
	RefreshInterval  int        `gorm:"default:60" json:"refresh_interval"` // minutes
	RefreshStatus    string     `gorm:"size:20;default:idle" json:"refresh_status"`
	RefreshError     string     `gorm:"type:text" json:"refresh_error"`
	LastRefreshAt    *time.Time `json:"last_refresh_at"`
	AISummaryEnabled bool       `gorm:"default:true" json:"ai_summary_enabled"`
	Articles         []Article  `gorm:"foreignKey:FeedID;constraint:OnDelete:CASCADE" json:"articles,omitempty"`
}

func (f *Feed) ToDict(includeStats bool) map[string]interface{} {
	data := map[string]interface{}{
		"id":                 f.ID,
		"title":              f.Title,
		"description":        f.Description,
		"url":                f.URL,
		"category_id":        f.CategoryID,
		"icon":               f.Icon,
		"color":              f.Color,
		"last_updated":       FormatDatetimeCSTPtr(f.LastUpdated),
		"created_at":         FormatDatetimeCST(f.CreatedAt),
		"max_articles":       f.MaxArticles,
		"refresh_interval":   f.RefreshInterval,
		"refresh_status":     f.RefreshStatus,
		"refresh_error":      f.RefreshError,
		"last_refresh_at":    FormatDatetimeCSTPtr(f.LastRefreshAt),
		"ai_summary_enabled": f.AISummaryEnabled,
	}

	if includeStats {
		articleCount := len(f.Articles)
		unreadCount := 0
		for _, a := range f.Articles {
			if !a.Read {
				unreadCount++
			}
		}
		data["article_count"] = articleCount
		data["unread_count"] = unreadCount
	}

	return data
}
