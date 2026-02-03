package models

import (
	"crypto/md5"
	"encoding/hex"
	"time"
)

type Category struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;size:100;not null" json:"name"`
	Slug        string    `gorm:"uniqueIndex;size:50" json:"slug"`
	Icon        string    `gorm:"size:50;default:folder" json:"icon"`
	Color       string    `gorm:"size:20;default:#6366f1" json:"color"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	Feeds       []Feed    `gorm:"foreignKey:CategoryID;constraint:OnDelete:CASCADE" json:"feeds,omitempty"`
}

func (c *Category) ToDict() map[string]interface{} {
	feedCount := len(c.Feeds)
	return map[string]interface{}{
		"id":          c.ID,
		"name":        c.Name,
		"slug":        c.Slug,
		"icon":        c.Icon,
		"color":       c.Color,
		"description": c.Description,
		"created_at":  FormatDatetimeCST(c.CreatedAt),
		"feed_count":  feedCount,
	}
}

func GenerateSlug(name string) string {
	hash := md5.Sum([]byte(name))
	return hex.EncodeToString(hash[:])[:8]
}
