package models

import (
	"time"
)

type Article struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	FeedID      uint       `gorm:"index;not null" json:"feed_id"`
	Title       string     `gorm:"size:500;not null" json:"title"`
	Description string     `gorm:"type:text" json:"description"`
	Content     string     `gorm:"type:text" json:"content"`
	Link        string     `gorm:"size:1000" json:"link"`
	PubDate     *time.Time `json:"pub_date"`
	Author      string     `gorm:"size:200" json:"author"`
	Read        bool       `gorm:"default:false" json:"read"`
	Favorite    bool       `gorm:"default:false" json:"favorite"`
	CreatedAt   time.Time  `json:"created_at"`
	Feed        Feed       `gorm:"foreignKey:FeedID" json:"feed,omitempty"`
}

func (a *Article) ToDict() map[string]interface{} {
	return map[string]interface{}{
		"id":          a.ID,
		"feed_id":     a.FeedID,
		"title":       a.Title,
		"description": a.Description,
		"content":     a.Content,
		"link":        a.Link,
		"pub_date":    FormatDatetimeCSTPtr(a.PubDate),
		"author":      a.Author,
		"read":        a.Read,
		"favorite":    a.Favorite,
		"created_at":  FormatDatetimeCST(a.CreatedAt),
	}
}
