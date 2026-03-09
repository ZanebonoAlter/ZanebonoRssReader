package digest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"my-robot-backend/internal/app/runtimeinfo"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
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
	WebhookURL string `json:"webhook_url"`
}

type TestObsidianRequest struct {
	VaultPath string `json:"vault_path"`
}

type digestPreviewSummary struct {
	ID           uint   `json:"id"`
	FeedID       *uint  `json:"feed_id"`
	FeedName     string `json:"feed_name"`
	FeedIcon     string `json:"feed_icon"`
	FeedColor    string `json:"feed_color"`
	CategoryID   uint   `json:"category_id"`
	CategoryName string `json:"category_name"`
	SummaryText  string `json:"summary_text"`
	ArticleCount int    `json:"article_count"`
	ArticleIDs   []uint `json:"article_ids"`
	CreatedAt    string `json:"created_at"`
}

type digestPreviewCategory struct {
	ID           uint                   `json:"id"`
	Name         string                 `json:"name"`
	FeedCount    int                    `json:"feed_count"`
	SummaryCount int                    `json:"summary_count"`
	Summaries    []digestPreviewSummary `json:"summaries"`
}

type digestPreviewResponse struct {
	Type              string                  `json:"type"`
	Title             string                  `json:"title"`
	PeriodLabel       string                  `json:"period_label"`
	GeneratedAt       string                  `json:"generated_at"`
	AnchorDate        string                  `json:"anchor_date"`
	CategoryCount     int                     `json:"category_count"`
	SummaryCount      int                     `json:"summary_count"`
	Markdown          string                  `json:"markdown"`
	Categories        []digestPreviewCategory `json:"categories"`
	DefaultCategoryID *uint                   `json:"default_category_id"`
	DefaultSummaryID  *uint                   `json:"default_summary_id"`
}

func validateTimeFormat(timeStr string) error {
	if timeStr == "" {
		return nil
	}
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid time format '%s', expected HH:MM", timeStr)
	}
	hour, minute, err := parseTimeParts(parts)
	if err != nil {
		return err
	}
	if hour < 0 || hour > 23 {
		return fmt.Errorf("hour must be between 0 and 23, got %d in time '%s'", hour, timeStr)
	}
	if minute < 0 || minute > 59 {
		return fmt.Errorf("minute must be between 0 and 59, got %d in time '%s'", minute, timeStr)
	}
	return nil
}

func parseTimeParts(parts []string) (int, int, error) {
	var hour, minute int
	if _, err := fmt.Sscanf(parts[0], "%d", &hour); err != nil {
		return 0, 0, fmt.Errorf("invalid hour in time '%s': %w", strings.Join(parts, ":"), err)
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &minute); err != nil {
		return 0, 0, fmt.Errorf("invalid minute in time '%s': %w", strings.Join(parts, ":"), err)
	}
	return hour, minute, nil
}

func validateWeekday(day int) error {
	if day < 0 || day > 6 {
		return fmt.Errorf("weekday must be between 0 and 6, got %d", day)
	}
	return nil
}

func defaultDigestConfig() DigestConfig {
	return DigestConfig{
		DailyEnabled:         false,
		DailyTime:            "09:00",
		WeeklyEnabled:        false,
		WeeklyDay:            1,
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
}

func ensureDigestConfig() (*DigestConfig, error) {
	var config DigestConfig
	if err := database.DB.First(&config).Error; err == nil {
		return &config, nil
	}

	config = defaultDigestConfig()
	if err := database.DB.Create(&config).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

func digestTitle(kind string) string {
	if kind == "weekly" {
		return "本周周报"
	}
	return "今日日报"
}

func digestPeriodLabel(kind string, date time.Time) string {
	cst := time.FixedZone("CST", 8*3600)
	current := date.In(cst)
	if kind == "weekly" {
		daysSinceMonday := (int(current.Weekday()) + 6) % 7
		start := current.AddDate(0, 0, -daysSinceMonday)
		end := start.AddDate(0, 0, 6)
		_, week := start.ISOWeek()
		return fmt.Sprintf("%d-W%d · %s 到 %s", start.Year(), week, start.Format("01-02"), end.Format("01-02"))
	}
	return current.Format("2006-01-02")
}

func parseDigestAnchorDate(raw string) (time.Time, error) {
	cst := time.FixedZone("CST", 8*3600)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Now().In(cst), nil
	}

	parsed, err := time.ParseInLocation("2006-01-02", raw, cst)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date '%s', expected YYYY-MM-DD", raw)
	}
	return parsed, nil
}
func totalSummaryCount(digests []CategoryDigest) int {
	total := 0
	for _, item := range digests {
		total += len(item.AISummaries)
	}
	return total
}

func parseSummaryArticleIDs(raw string) []uint {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []uint{}
	}

	var uintIDs []uint
	if err := json.Unmarshal([]byte(raw), &uintIDs); err == nil {
		return uintIDs
	}

	var intIDs []int
	if err := json.Unmarshal([]byte(raw), &intIDs); err == nil {
		result := make([]uint, 0, len(intIDs))
		for _, id := range intIDs {
			if id > 0 {
				result = append(result, uint(id))
			}
		}
		return result
	}

	var floatIDs []float64
	if err := json.Unmarshal([]byte(raw), &floatIDs); err == nil {
		result := make([]uint, 0, len(floatIDs))
		for _, id := range floatIDs {
			if id > 0 {
				result = append(result, uint(id))
			}
		}
		return result
	}

	return []uint{}
}

func buildDigestMarkdown(kind string, date time.Time, digests []CategoryDigest) string {
	var builder strings.Builder

	builder.WriteString("# ")
	builder.WriteString(digestTitle(kind))
	builder.WriteString("\n\n")
	builder.WriteString("- 时间：")
	builder.WriteString(digestPeriodLabel(kind, date))
	builder.WriteString("\n")
	builder.WriteString(fmt.Sprintf("- 分类数：%d\n", len(digests)))
	builder.WriteString(fmt.Sprintf("- 总结数：%d\n\n", totalSummaryCount(digests)))

	if len(digests) == 0 {
		builder.WriteString("当前时间窗还没有可展示的 AI 总结。")
		return builder.String()
	}

	for _, item := range digests {
		builder.WriteString("## ")
		builder.WriteString(item.CategoryName)
		builder.WriteString("\n\n")
		builder.WriteString(fmt.Sprintf("- 订阅源数：%d\n", item.FeedCount))
		builder.WriteString(fmt.Sprintf("- 总结数：%d\n\n", len(item.AISummaries)))

		for _, summary := range item.AISummaries {
			feedName := "未命名订阅"
			if summary.Feed != nil && strings.TrimSpace(summary.Feed.Title) != "" {
				feedName = summary.Feed.Title
			}

			builder.WriteString("### ")
			builder.WriteString(feedName)
			builder.WriteString("\n\n")
			builder.WriteString(summary.Summary)
			builder.WriteString("\n\n")
		}
	}

	return builder.String()
}

func buildPreviewCategories(digests []CategoryDigest) ([]digestPreviewCategory, *uint, *uint) {
	categories := make([]digestPreviewCategory, 0, len(digests))
	var defaultCategoryID *uint
	var defaultSummaryID *uint

	for _, item := range digests {
		summaries := make([]digestPreviewSummary, 0, len(item.AISummaries))
		for _, summary := range item.AISummaries {
			feedName := "未命名订阅"
			feedIcon := "mdi:rss"
			feedColor := "#3b6b87"

			if summary.Feed != nil {
				if strings.TrimSpace(summary.Feed.Title) != "" {
					feedName = summary.Feed.Title
				}
				if strings.TrimSpace(summary.Feed.Icon) != "" {
					feedIcon = summary.Feed.Icon
				}
				if strings.TrimSpace(summary.Feed.Color) != "" {
					feedColor = summary.Feed.Color
				}
			}

			articleIDs := parseSummaryArticleIDs(summary.Articles)
			summaryItem := digestPreviewSummary{
				ID:           summary.ID,
				FeedID:       summary.FeedID,
				FeedName:     feedName,
				FeedIcon:     feedIcon,
				FeedColor:    feedColor,
				CategoryID:   item.CategoryID,
				CategoryName: item.CategoryName,
				SummaryText:  summary.Summary,
				ArticleCount: summary.ArticleCount,
				ArticleIDs:   articleIDs,
				CreatedAt:    models.FormatDatetimeCST(summary.CreatedAt),
			}
			summaries = append(summaries, summaryItem)

			if defaultCategoryID == nil {
				categoryID := item.CategoryID
				defaultCategoryID = &categoryID
			}
			if defaultSummaryID == nil {
				summaryID := summary.ID
				defaultSummaryID = &summaryID
			}
		}

		category := digestPreviewCategory{
			ID:           item.CategoryID,
			Name:         item.CategoryName,
			FeedCount:    item.FeedCount,
			SummaryCount: len(summaries),
			Summaries:    summaries,
		}
		categories = append(categories, category)
	}

	sort.SliceStable(categories, func(i, j int) bool {
		if categories[i].SummaryCount == categories[j].SummaryCount {
			return categories[i].Name < categories[j].Name
		}
		return categories[i].SummaryCount > categories[j].SummaryCount
	})

	return categories, defaultCategoryID, defaultSummaryID
}

func buildPreview(kind string, date time.Time) (*digestPreviewResponse, *DigestConfig, []CategoryDigest, error) {
	config, err := ensureDigestConfig()
	if err != nil {
		return nil, nil, nil, err
	}

	generator := NewDigestGenerator(config)
	var digests []CategoryDigest

	switch kind {
	case "daily":
		digests, err = generator.GenerateDailyDigest(date)
	case "weekly":
		digests, err = generator.GenerateWeeklyDigest(date)
	default:
		return nil, nil, nil, fmt.Errorf("unsupported digest type: %s", kind)
	}
	if err != nil {
		return nil, nil, nil, err
	}

	categories, defaultCategoryID, defaultSummaryID := buildPreviewCategories(digests)
	preview := &digestPreviewResponse{
		Type:              kind,
		Title:             digestTitle(kind),
		PeriodLabel:       digestPeriodLabel(kind, date),
		GeneratedAt:       models.FormatDatetimeCST(time.Now()),
		AnchorDate:        date.In(time.FixedZone("CST", 8*3600)).Format("2006-01-02"),
		CategoryCount:     len(categories),
		SummaryCount:      totalSummaryCount(digests),
		Markdown:          buildDigestMarkdown(kind, date, digests),
		Categories:        categories,
		DefaultCategoryID: defaultCategoryID,
		DefaultSummaryID:  defaultSummaryID,
	}

	return preview, config, digests, nil
}

func GetDigestConfig(c *gin.Context) {
	config, err := ensureDigestConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    config,
	})
}

func GetDigestStatus(c *gin.Context) {
	if runtimeinfo.DigestSchedulerInterface != nil {
		if scheduler, ok := runtimeinfo.DigestSchedulerInterface.(interface{ GetStatus() map[string]interface{} }); ok {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    scheduler.GetStatus(),
			})
			return
		}
	}

	config, err := ensureDigestConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"running":        false,
			"daily_enabled":  config.DailyEnabled,
			"weekly_enabled": config.WeeklyEnabled,
			"daily_time":     config.DailyTime,
			"weekly_day":     config.WeeklyDay,
			"weekly_time":    config.WeeklyTime,
			"next_runs":      []string{},
			"active_jobs":    0,
		},
	})
}

func GetDigestPreview(c *gin.Context) {
	kind := c.Param("type")
	anchorDate, err := parseDigestAnchorDate(c.Query("date"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	preview, _, _, err := buildPreview(kind, anchorDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    preview,
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

	if req.DailyEnabled {
		if err := validateTimeFormat(req.DailyTime); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": fmt.Sprintf("Invalid daily time: %v", err)})
			return
		}
	}

	if req.WeeklyEnabled {
		if err := validateTimeFormat(req.WeeklyTime); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": fmt.Sprintf("Invalid weekly time: %v", err)})
			return
		}
		if err := validateWeekday(req.WeeklyDay); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": fmt.Sprintf("Invalid weekly day: %v", err)})
			return
		}
	}

	config, err := ensureDigestConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

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

	if err := database.DB.Save(config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	if runtimeinfo.DigestSchedulerInterface != nil {
		if scheduler, ok := runtimeinfo.DigestSchedulerInterface.(interface{ Reload() error }); ok {
			if err := scheduler.Reload(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error":   "Config saved, but scheduler reload failed: " + err.Error(),
				})
				return
			}
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
	if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request body"})
		return
	}

	config, err := ensureDigestConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	webhookURL := strings.TrimSpace(req.WebhookURL)
	if webhookURL == "" {
		webhookURL = strings.TrimSpace(config.FeishuWebhookURL)
	}
	if webhookURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "请先填写飞书 Webhook URL"})
		return
	}

	notifier := NewFeishuNotifier(webhookURL)
	testTitle := "RSS Reader - 飞书测试"
	testContent := "这是一条测试消息。\n\n如果你看到了它，说明飞书机器人已经通了。"

	if err := notifier.SendSummary(testTitle, testContent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "飞书推送测试失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "飞书推送测试成功，请去群里看一眼"})
}

func TestObsidianWrite(c *gin.Context) {
	var req TestObsidianRequest
	if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request body"})
		return
	}

	config, err := ensureDigestConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	vaultPath := strings.TrimSpace(req.VaultPath)
	if vaultPath == "" {
		vaultPath = strings.TrimSpace(config.ObsidianVaultPath)
	}
	if vaultPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "请先填写 Obsidian Vault 路径"})
		return
	}

	testDigests := []CategoryDigest{
		{
			CategoryName: "测试导出",
			FeedCount:    1,
			AISummaries: []models.AISummary{{
				ID:      1,
				Summary: "这是一条测试摘要，用来确认 Obsidian 写入链路没断。",
				Feed:    &models.Feed{Title: "测试订阅源"},
			}},
		},
	}

	exporter := NewObsidianExporter(vaultPath)
	if err := exporter.ExportDailyDigest(time.Now(), testDigests); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Obsidian 写入测试失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Obsidian 写入测试成功，去 Vault 的 Daily/测试导出 目录看看"})
}

func RunDigestNow(c *gin.Context) {
	kind := c.Param("type")
	anchorDate, err := parseDigestAnchorDate(c.Query("date"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	preview, config, digests, err := buildPreview(kind, anchorDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	sentToFeishu := false
	exportedToObsidian := false

	if config.FeishuEnabled && strings.TrimSpace(config.FeishuWebhookURL) != "" {
		notifier := NewFeishuNotifier(config.FeishuWebhookURL)
		title := preview.Title + " · " + preview.PeriodLabel
		content := preview.Markdown
		if config.FeishuPushDetails {
			err = notifier.SendCard(title, content)
		} else {
			err = notifier.SendSummary(title, content)
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "飞书发送失败: " + err.Error()})
			return
		}
		sentToFeishu = true
	}

	if config.ObsidianEnabled && strings.TrimSpace(config.ObsidianVaultPath) != "" {
		exporter := NewObsidianExporter(config.ObsidianVaultPath)
		switch kind {
		case "daily":
			if config.ObsidianDailyDigest {
				if err := exporter.ExportDailyDigest(anchorDate, digests); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Obsidian 导出失败: " + err.Error()})
					return
				}
				exportedToObsidian = true
			}
		case "weekly":
			if config.ObsidianWeeklyDigest {
				if err := exporter.ExportWeeklyDigest(anchorDate, digests); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Obsidian 导出失败: " + err.Error()})
					return
				}
				exportedToObsidian = true
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "已执行当前 digest 流程",
		"data": gin.H{
			"preview":              preview,
			"sent_to_feishu":       sentToFeishu,
			"exported_to_obsidian": exportedToObsidian,
		},
	})
}
