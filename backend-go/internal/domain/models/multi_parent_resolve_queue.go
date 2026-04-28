package models

import "time"

const (
	MultiParentResolveStatusPending    = "pending"
	MultiParentResolveStatusProcessing = "processing"
	MultiParentResolveStatusCompleted  = "completed"
	MultiParentResolveStatusFailed     = "failed"
)

type MultiParentResolveQueue struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	ChildTagID   uint       `gorm:"not null;index" json:"child_tag_id"`
	Source       string     `gorm:"size:50;not null" json:"source"`
	Status       string     `gorm:"size:20;not null;default:pending;index" json:"status"`
	ErrorMessage string     `gorm:"type:text" json:"error_message"`
	RetryCount   int        `gorm:"default:0" json:"retry_count"`
	CreatedAt    time.Time  `json:"created_at"`
	StartedAt    *time.Time `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at"`
}

func (MultiParentResolveQueue) TableName() string {
	return "multi_parent_resolve_queues"
}
