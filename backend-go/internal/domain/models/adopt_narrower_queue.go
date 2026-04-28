package models

import "time"

const (
	AdoptNarrowerQueueStatusPending    = "pending"
	AdoptNarrowerQueueStatusProcessing = "processing"
	AdoptNarrowerQueueStatusCompleted  = "completed"
	AdoptNarrowerQueueStatusFailed     = "failed"
)

type AdoptNarrowerQueue struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	AbstractTagID uint       `gorm:"not null;index" json:"abstract_tag_id"`
	Source        string     `gorm:"size:50;not null" json:"source"`
	Status        string     `gorm:"size:20;not null;default:pending;index" json:"status"`
	ErrorMessage  string     `gorm:"type:text" json:"error_message"`
	RetryCount    int        `gorm:"default:0" json:"retry_count"`
	CreatedAt     time.Time  `json:"created_at"`
	StartedAt     *time.Time `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at"`
	AbstractTag   *TopicTag  `gorm:"foreignKey:AbstractTagID" json:"abstract_tag,omitempty"`
}

func (AdoptNarrowerQueue) TableName() string {
	return "adopt_narrower_queues"
}
