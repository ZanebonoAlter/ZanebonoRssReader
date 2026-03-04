package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"my-robot-backend/internal/digest"
	"my-robot-backend/internal/models"
	"my-robot-backend/pkg/database"
)

type UpdateDigestConfigRequest struct {
	DailyEnabled         bool   `json:"daily_enabled"`
	DailyTime            string `json:"daily_time"`
	WeeklyEnabled        bool   `json:"weekly_enabled"`
	WeeklyDay            int    `json:"weekly_day"`
	WeeklyTime           string `json:"weekly_time"`
	FeishuEnabled        bool   `json:"feishu_enabled"`
	FeishuWebhookURL     string `json:"feishu_webhook_url"`
	FeishuPushSummary    bool   `json:"feishu_push_summary"`
	FeishuPushDetails    bool   `json:"feishu_push_details"`
	ObsidianEnabled      bool   `json:"obsidian_enabled"`
	ObsidianVaultPath    string `json:"obsidian_vault_path"`
	ObsidianDailyDigest  bool   `json:"obsidian_daily_digest"`
	ObsidianWeeklyDigest bool   `json:"obsidian_weekly_digest"`
}

type TestFeishuRequest struct {
	WebhookURL string `json:"webhook_url" binding:"required"`
}

type TestObsidianRequest struct {
	VaultPath string `json:"vault_path" binding:"required"`
}

func GetDigestConfig(c *gin.Context) {
	var config digest.DigestConfig
	if err := database.DB.First(&config).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    config,
	})
}

func UpdateDigestConfig(c *gin.Context) {
	var req UpdateDigestConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
		})
		return
	}

	var config digest.DigestConfig
	err := database.DB.First(&config).Error

	if err == nil {
		config.DailyEnabled = req.DailyEnabled
		config.DailyTime = req.DailyTime
		config.WeeklyEnabled = req.WeeklyEnabled
		config.WeeklyDay = req.WeeklyDay
		config.WeeklyTime = req.WeeklyTime
		config.FeishuEnabled = req.FeishuEnabled
		config.FeishuWebhookURL = req.FeishuWebhookURL
		config.FeishuPushSummary = req.FeishuPushSummary
		config.FeishuPushDetails = req.FeishuPushDetails
		config.ObsidianEnabled = req.ObsidianEnabled
		config.ObsidianVaultPath = req.ObsidianVaultPath
		config.ObsidianDailyDigest = req.ObsidianDailyDigest
		config.ObsidianWeeklyDigest = req.ObsidianWeeklyDigest

		if err := database.DB.Save(&config).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
	} else {
		config = digest.DigestConfig{
			DailyEnabled:         req.DailyEnabled,
			DailyTime:            req.DailyTime,
			WeeklyEnabled:        req.WeeklyEnabled,
			WeeklyDay:            req.WeeklyDay,
			WeeklyTime:           req.WeeklyTime,
			FeishuEnabled:        req.FeishuEnabled,
			FeishuWebhookURL:     req.FeishuWebhookURL,
			FeishuPushSummary:    req.FeishuPushSummary,
			FeishuPushDetails:    req.FeishuPushDetails,
			ObsidianEnabled:      req.ObsidianEnabled,
			ObsidianVaultPath:    req.ObsidianVaultPath,
			ObsidianDailyDigest:  req.ObsidianDailyDigest,
			ObsidianWeeklyDigest: req.ObsidianWeeklyDigest,
		}

		if err := database.DB.Create(&config).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    config,
		"message": "Digest config updated successfully",
	})
}

func TestFeishuPush(c *gin.Context) {
	var req TestFeishuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "webhook_url is required",
		})
		return
	}

	notifier := digest.NewFeishuNotifier(req.WebhookURL)
	testTitle := "RSS Reader - 飞书推送测试"
	testContent := "这是一条测试消息，用于验证飞书 Webhook 配置是否正确。\n\n如果收到此消息，说明配置成功！"

	if err := notifier.SendSummary(testTitle, testContent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "飞书推送测试失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "飞书推送测试成功，请检查飞书是否收到消息",
	})
}

func TestObsidianWrite(c *gin.Context) {
	var req TestObsidianRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "vault_path is required",
		})
		return
	}

	testDigests := []digest.CategoryDigest{
		{
			CategoryName: "技术",
			FeedCount:    1,
			AISummaries: []models.AISummary{
				{
					ID:      1,
					Summary: "这是一条测试摘要，用于验证 Obsidian 导出功能是否正常工作。",
					Feed:    &models.Feed{Title: "测试订阅源"},
				},
			},
		},
	}

	exporter := digest.NewObsidianExporter(req.VaultPath)
	now := time.Now()

	testDir := now.Format("2006-01-02-test")
	testDigests[0].CategoryName = "测试-" + testDir

	if err := exporter.ExportDailyDigest(now, testDigests); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Obsidian 写入测试失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Obsidian 写入测试成功，请检查 Vault 目录中的 Daily/测试-* 文件",
	})
}
