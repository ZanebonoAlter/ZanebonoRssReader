package models

import "time"

const (
	AbstractTagUpdateQueueStatusPending    = "pending"
	AbstractTagUpdateQueueStatusProcessing = "processing"
	AbstractTagUpdateQueueStatusCompleted  = "completed"
	AbstractTagUpdateQueueStatusFailed     = "failed"
)

type AbstractTagUpdateQueue struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	AbstractTagID uint       `gorm:"not null;index" json:"abstract_tag_id"`
	TriggerReason string     `gorm:"size:50;not null" json:"trigger_reason"`
	Status        string     `gorm:"size:20;not null;default:pending;index" json:"status"`
	ErrorMessage  string     `gorm:"type:text" json:"error_message"`
	RetryCount    int        `gorm:"default:0" json:"retry_count"`
	CreatedAt     time.Time  `json:"created_at"`
	StartedAt     *time.Time `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at"`
	AbstractTag   *TopicTag  `gorm:"foreignKey:AbstractTagID" json:"abstract_tag,omitempty"`
}

func (AbstractTagUpdateQueue) TableName() string {
	return "abstract_tag_update_queues"
}
