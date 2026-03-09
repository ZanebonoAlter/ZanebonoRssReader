package digest

import (
	"time"
)

// DigestConfig 存储每日/每周摘要的配置
type DigestConfig struct {
	ID uint `gorm:"primaryKey" json:"id"`

	// 每日摘要配置
	DailyEnabled bool   `gorm:"default:false" json:"daily_enabled"`
	DailyTime    string `gorm:"size:5;default:'09:00'" json:"daily_time"` // 格式: "HH:MM"

	// 每周摘要配置
	WeeklyEnabled bool   `gorm:"default:false" json:"weekly_enabled"`
	WeeklyDay     int    `gorm:"default:1;comment:0=Sunday,1=Monday,...,6=Saturday" json:"weekly_day"`
	WeeklyTime    string `gorm:"size:5;default:'09:00'" json:"weekly_time"` // 格式: "HH:MM"

	// 飞书推送配置
	FeishuEnabled     bool   `gorm:"default:false" json:"feishu_enabled"`
	FeishuWebhookURL  string `gorm:"type:text" json:"feishu_webhook_url"`
	FeishuPushSummary bool   `gorm:"default:true" json:"feishu_push_summary"`
	FeishuPushDetails bool   `gorm:"default:false" json:"feishu_push_details"`

	// Obsidian 配置
	ObsidianEnabled      bool   `gorm:"default:false" json:"obsidian_enabled"`
	ObsidianVaultPath    string `gorm:"type:text" json:"obsidian_vault_path"`
	ObsidianDailyDigest  bool   `gorm:"default:true" json:"obsidian_daily_digest"`
	ObsidianWeeklyDigest bool   `gorm:"default:true" json:"obsidian_weekly_digest"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (DigestConfig) TableName() string {
	return "digest_configs"
}
