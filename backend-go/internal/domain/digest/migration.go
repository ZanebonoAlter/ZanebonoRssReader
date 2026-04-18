package digest

import (
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/logging"
)

// Migrate 执行摘要配置相关的数据库迁移
func Migrate() error {
	// 自动迁移 DigestConfig 表
	if err := database.DB.AutoMigrate(&DigestConfig{}); err != nil {
		logging.Errorf("Failed to migrate digest models: %v", err)
		return err
	}
	logging.Infof("Digest models migrated successfully")

	// 创建默认配置（如果表为空）
	var count int64
	if err := database.DB.Model(&DigestConfig{}).Count(&count).Error; err != nil {
		logging.Errorf("Failed to count digest configs: %v", err)
		return err
	}

	if count == 0 {
		defaultConfig := DigestConfig{
			DailyEnabled:         false,
			DailyTime:            "09:00",
			WeeklyEnabled:        false,
			WeeklyDay:            1, // Monday
			WeeklyTime:           "09:00",
			FeishuEnabled:        false,
			FeishuWebhookURL:     "",
			FeishuPushSummary:    true,
			FeishuPushDetails:    false,
			ObsidianEnabled:      false,
			ObsidianVaultPath:    "",
			ObsidianDailyDigest:  true,
			ObsidianWeeklyDigest: true,
		}

		if err := database.DB.Create(&defaultConfig).Error; err != nil {
			logging.Errorf("Failed to create default digest config: %v", err)
			return err
		}
		logging.Infof("Default digest config created")
	} else {
		logging.Infof("Digest config table already has %d record(s)", count)
	}

	return nil
}
