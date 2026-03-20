package models

import "time"

// TopicTagAnalysis 主题标签分析快照
type TopicTagAnalysis struct {
	ID           uint64    `gorm:"primaryKey"`
	TopicTagID   uint64    `gorm:"index:idx_tag_analysis_date,unique"`
	AnalysisType string    `gorm:"index:idx_tag_analysis_date,unique"` // event, person, keyword
	WindowType   string    `gorm:"index:idx_tag_analysis_date,unique"` // daily, weekly
	AnchorDate   time.Time `gorm:"index:idx_tag_analysis_date,unique"`
	SummaryCount int
	PayloadJSON  string // 存储分析结果的JSON
	Source       string // ai, heuristic, cached
	Version      int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// TopicAnalysisCursor 分析游标（用于增量更新）
type TopicAnalysisCursor struct {
	ID            uint64 `gorm:"primaryKey"`
	TopicTagID    uint64 `gorm:"uniqueIndex:idx_cursor_tag_type_window"`
	AnalysisType  string `gorm:"uniqueIndex:idx_cursor_tag_type_window"`
	WindowType    string `gorm:"uniqueIndex:idx_cursor_tag_type_window"`
	LastSummaryID uint64
	LastUpdatedAt time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
