package models

import "time"

type TopicTag struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Slug      string    `gorm:"size:120;uniqueIndex;not null" json:"slug"`
	Label     string    `gorm:"size:160;not null" json:"label"`
	Kind      string    `gorm:"size:20;default:topic" json:"kind"`
	Aliases   string    `gorm:"type:text" json:"aliases"`
	Source    string    `gorm:"size:20;default:heuristic" json:"source"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AISummaryTopic struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	SummaryID  uint      `gorm:"index;not null" json:"summary_id"`
	TopicTagID uint      `gorm:"index;not null" json:"topic_tag_id"`
	Score      float64   `gorm:"default:0" json:"score"`
	Source     string    `gorm:"size:20;default:heuristic" json:"source"`
	CreatedAt  time.Time `json:"created_at"`
	TopicTag   *TopicTag `gorm:"foreignKey:TopicTagID" json:"topic_tag,omitempty"`
}
