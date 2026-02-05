package models

import (
	"encoding/json"
	"time"
)

type AISummary struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	CategoryID   *uint     `gorm:"index" json:"category_id"`
	Title        string    `gorm:"size:200;not null" json:"title"`
	Summary      string    `gorm:"type:text;not null" json:"summary"`
	KeyPoints    string    `gorm:"type:text" json:"key_points"` // JSON array
	Articles     string    `gorm:"type:text" json:"articles"`   // JSON array of article IDs
	ArticleCount int       `gorm:"default:0" json:"article_count"`
	TimeRange    int       `gorm:"default:180" json:"time_range"` // minutes
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Category     *Category `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
}

func (a *AISummary) ToDict() map[string]interface{} {
	categoryName := "全部分类"
	if a.Category != nil {
		categoryName = a.Category.Name
	}

	return map[string]interface{}{
		"id":            a.ID,
		"category_id":   a.CategoryID,
		"title":         a.Title,
		"summary":       a.Summary,
		"key_points":    a.KeyPoints,
		"articles":      a.Articles,
		"article_count": a.ArticleCount,
		"time_range":    a.TimeRange,
		"created_at":    FormatDatetimeCST(a.CreatedAt),
		"updated_at":    FormatDatetimeCST(a.UpdatedAt),
		"category_name": categoryName,
	}
}

type SchedulerTask struct {
	ID                    uint       `gorm:"primaryKey" json:"id"`
	Name                  string     `gorm:"size:50;unique;not null;index" json:"name"`
	Description           string     `gorm:"size:200" json:"description"`
	CheckInterval         int        `gorm:"default:60;not null" json:"check_interval"` // seconds
	LastExecutionTime     *time.Time `json:"last_execution_time"`
	NextExecutionTime     *time.Time `json:"next_execution_time"`
	Status                string     `gorm:"size:20;default:idle;index" json:"status"`
	LastError             string     `gorm:"type:text" json:"last_error"`
	LastErrorTime         *time.Time `json:"last_error_time"`
	TotalExecutions       int        `gorm:"default:0" json:"total_executions"`
	SuccessfulExecutions  int        `gorm:"default:0" json:"successful_executions"`
	FailedExecutions      int        `gorm:"default:0" json:"failed_executions"`
	ConsecutiveFailures   int        `gorm:"default:0" json:"consecutive_failures"`
	LastExecutionDuration *float64   `json:"last_execution_duration"` // seconds
	LastExecutionResult   string     `gorm:"type:text" json:"last_execution_result"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

func (s *SchedulerTask) ToDict() map[string]interface{} {
	successRate := 0.0
	if s.TotalExecutions > 0 {
		successRate = float64(s.SuccessfulExecutions) / float64(s.TotalExecutions) * 100
	}

	return map[string]interface{}{
		"id":                      s.ID,
		"name":                    s.Name,
		"description":             s.Description,
		"check_interval":          s.CheckInterval,
		"last_execution_time":     FormatDatetimeCSTPtr(s.LastExecutionTime),
		"next_execution_time":     FormatDatetimeCSTPtr(s.NextExecutionTime),
		"status":                  s.Status,
		"last_error":              s.LastError,
		"last_error_time":         FormatDatetimeCSTPtr(s.LastErrorTime),
		"total_executions":        s.TotalExecutions,
		"successful_executions":   s.SuccessfulExecutions,
		"failed_executions":       s.FailedExecutions,
		"consecutive_failures":    s.ConsecutiveFailures,
		"last_execution_duration": s.LastExecutionDuration,
		"last_execution_result":   s.LastExecutionResult,
		"created_at":              FormatDatetimeCST(s.CreatedAt),
		"updated_at":              FormatDatetimeCST(s.UpdatedAt),
		"success_rate":            successRate,
	}
}

type AISettings struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Key         string    `gorm:"size:100;unique;not null;index" json:"key"`
	Value       string    `gorm:"type:text" json:"value"`
	Description string    `gorm:"size:200" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (a *AISettings) ToDict() map[string]interface{} {
	var valueJSON interface{}
	if a.Value != "" {
		json.Unmarshal([]byte(a.Value), &valueJSON)
	}

	return map[string]interface{}{
		"id":          a.ID,
		"key":         a.Key,
		"value":       valueJSON,
		"description": a.Description,
		"created_at":  FormatDatetimeCST(a.CreatedAt),
		"updated_at":  FormatDatetimeCST(a.UpdatedAt),
	}
}
