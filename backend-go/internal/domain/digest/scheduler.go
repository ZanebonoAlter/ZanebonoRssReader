package digest

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/logging"
)

func parseTime(timeStr string) (hour, minute int, err error) {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid time format, expected HH:MM")
	}
	hour, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid hour: %w", err)
	}
	minute, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid minute: %w", err)
	}
	if hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf("hour must be between 0 and 23")
	}
	if minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("minute must be between 0 and 59")
	}
	return hour, minute, nil
}

type DigestScheduler struct {
	cron           *cron.Cron
	isRunning      bool
	mu             sync.Mutex
	executionMutex sync.Mutex
	config         *DigestConfig
}

func NewDigestScheduler() *DigestScheduler {
	return &DigestScheduler{
		cron:      cron.New(),
		isRunning: false,
		config:    nil,
	}
}

func (s *DigestScheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		logging.Infof("Digest scheduler already running")
		return nil
	}

	return s.reloadLocked()
}

func (s *DigestScheduler) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.reloadLocked()
}

func (s *DigestScheduler) reloadLocked() error {
	if s.cron != nil && s.isRunning {
		ctx := s.cron.Stop()
		<-ctx.Done()
	}

	s.cron = cron.New()

	config, err := s.loadOrCreateConfig()
	if err != nil {
		logging.Errorf("Failed to load digest config: %v", err)
		return err
	}
	s.config = config

	if config.DailyEnabled {
		hour, minute, err := parseTime(config.DailyTime)
		if err != nil {
			return fmt.Errorf("invalid daily time format: %w", err)
		}
		dailyExpr := fmt.Sprintf("%d %d * * *", minute, hour)
		if _, err := s.cron.AddFunc(dailyExpr, s.generateDailyDigest); err != nil {
			return fmt.Errorf("failed to schedule daily digest: %w", err)
		}
		logging.Infof("Daily digest scheduled at %s", config.DailyTime)
	}

	if config.WeeklyEnabled {
		hour, minute, err := parseTime(config.WeeklyTime)
		if err != nil {
			return fmt.Errorf("invalid weekly time format: %w", err)
		}
		if config.WeeklyDay < 0 || config.WeeklyDay > 6 {
			return fmt.Errorf("invalid weekly day: %d (must be 0-6)", config.WeeklyDay)
		}
		cronDay := s.intToCronDay(config.WeeklyDay)
		weeklyExpr := fmt.Sprintf("%d %d * * %s", minute, hour, cronDay)
		if _, err := s.cron.AddFunc(weeklyExpr, s.generateWeeklyDigest); err != nil {
			return fmt.Errorf("failed to schedule weekly digest: %w", err)
		}
		logging.Infof("Weekly digest scheduled at %s on day %d", config.WeeklyTime, config.WeeklyDay)
	}

	s.cron.Start()
	s.isRunning = true
	logging.Infof("Digest scheduler started")
	return nil
}

func (s *DigestScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.isRunning = false
	logging.Infof("Digest scheduler stopped")
}

func (s *DigestScheduler) loadOrCreateConfig() (*DigestConfig, error) {
	var config DigestConfig
	err := database.DB.First(&config).Error
	if err != nil {
		defaultConfig := DigestConfig{
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
		if err := database.DB.Create(&defaultConfig).Error; err != nil {
			return nil, fmt.Errorf("failed to create default digest config: %w", err)
		}
		logging.Infof("Created default digest config")
		return &defaultConfig, nil
	}
	return &config, nil
}

func (s *DigestScheduler) intToCronDay(day int) string {
	cronDayMap := map[int]string{
		0: "0",
		1: "1",
		2: "2",
		3: "3",
		4: "4",
		5: "5",
		6: "6",
	}
	return cronDayMap[day]
}

func (s *DigestScheduler) sendFeishuDigest(title string, digests []CategoryDigest, config *DigestConfig) error {
	notifier := NewFeishuNotifier(config.FeishuWebhookURL)

	if config.FeishuPushDetails {
		content := s.generateDetailedDigestMessage(digests)
		return notifier.SendCard(title, content)
	}

	content := s.generateSummaryMessage(digests)
	return notifier.SendSummary(title, content)
}

func (s *DigestScheduler) exportToObsidian(isDaily bool, date time.Time, digests []CategoryDigest, config *DigestConfig) error {
	exporter := NewObsidianExporter(config.ObsidianVaultPath)

	if isDaily {
		return exporter.ExportDailyDigest(date, digests)
	}

	return exporter.ExportWeeklyDigest(date, digests)
}

func (s *DigestScheduler) autoSendToOpenNotebook(kind string, date time.Time, digests []CategoryDigest) bool {
	config, _, err := loadOpenNotebookConfigRecord()
	if err != nil {
		logging.Errorf("Failed to load Open Notebook config for %s digest: %v", kind, err)
		return false
	}
	if !shouldAutoSendOpenNotebook(kind, config) {
		return false
	}

	preview := &digestPreviewResponse{
		Type:       kind,
		Title:      digestTitle(kind),
		AnchorDate: date.In(time.FixedZone("CST", 8*3600)).Format("2006-01-02"),
		Markdown:   buildDigestMarkdown(kind, date, digests),
	}

	if _, err := sendDigestPreviewToOpenNotebook(kind, preview, config); err != nil {
		logging.Errorf("Failed to auto-send %s digest to Open Notebook: %v", kind, err)
		return false
	}

	logging.Infof("%s digest sent to Open Notebook successfully", kind)
	return true
}

func (s *DigestScheduler) generateSummaryMessage(digests []CategoryDigest) string {
	var message string
	message += "日报周报概览\n\n"
	message += fmt.Sprintf("共 %d 个分类有新内容\n\n", len(digests))

	for _, digest := range digests {
		message += fmt.Sprintf("- %s\n", digest.CategoryName)
		message += fmt.Sprintf("  %d 条总结\n\n", len(digest.AISummaries))
	}

	return message
}

func (s *DigestScheduler) generateDetailedDigestMessage(digests []CategoryDigest) string {
	var content string

	for _, digest := range digests {
		content += fmt.Sprintf("## %s\n\n", digest.CategoryName)
		content += fmt.Sprintf("**订阅源数量**: %d\n", digest.FeedCount)
		content += fmt.Sprintf("**总结数量**: %d\n\n", len(digest.AISummaries))

		for _, summary := range digest.AISummaries {
			feedName := "未知订阅源"
			if summary.Feed != nil {
				feedName = summary.Feed.Title
			}
			content += fmt.Sprintf("### %s\n\n", feedName)
			content += fmt.Sprintf("%s\n\n", summary.Summary)
		}
	}

	return content
}

func (s *DigestScheduler) generateDailyDigest() {
	if !s.executionMutex.TryLock() {
		logging.Infof("Digest generation already running, skipping daily cycle")
		return
	}
	defer s.executionMutex.Unlock()

	logging.Infof("Starting daily digest generation")

	config, err := s.loadOrCreateConfig()
	if err != nil {
		logging.Errorf("Failed to load config for daily digest: %v", err)
		return
	}

	generator := NewDigestGenerator(config)
	digests, err := generator.GenerateDailyDigest(time.Now())
	if err != nil {
		logging.Errorf("Failed to generate daily digest: %v", err)
		return
	}

	logging.Infof("Generated daily digest with %d categories", len(digests))

	if config.FeishuEnabled && config.FeishuWebhookURL != "" {
		if err := s.sendFeishuDigest("每日日报", digests, config); err != nil {
			logging.Errorf("Failed to send daily digest to Feishu: %v", err)
		} else {
			logging.Infof("Daily digest sent to Feishu successfully")
		}
	}

	if config.ObsidianEnabled && config.ObsidianVaultPath != "" && config.ObsidianDailyDigest {
		if err := s.exportToObsidian(true, time.Now(), digests, config); err != nil {
			logging.Errorf("Failed to export daily digest to Obsidian: %v", err)
		} else {
			logging.Infof("Daily digest exported to Obsidian successfully")
		}
	}

	s.autoSendToOpenNotebook("daily", time.Now(), digests)
}

func (s *DigestScheduler) generateWeeklyDigest() {
	if !s.executionMutex.TryLock() {
		logging.Infof("Digest generation already running, skipping weekly cycle")
		return
	}
	defer s.executionMutex.Unlock()

	logging.Infof("Starting weekly digest generation")

	config, err := s.loadOrCreateConfig()
	if err != nil {
		logging.Errorf("Failed to load config for weekly digest: %v", err)
		return
	}

	generator := NewDigestGenerator(config)
	digests, err := generator.GenerateWeeklyDigest(time.Now())
	if err != nil {
		logging.Errorf("Failed to generate weekly digest: %v", err)
		return
	}

	logging.Infof("Generated weekly digest with %d categories", len(digests))

	if config.FeishuEnabled && config.FeishuWebhookURL != "" {
		if err := s.sendFeishuDigest("每周周报", digests, config); err != nil {
			logging.Errorf("Failed to send weekly digest to Feishu: %v", err)
		} else {
			logging.Infof("Weekly digest sent to Feishu successfully")
		}
	}

	if config.ObsidianEnabled && config.ObsidianVaultPath != "" && config.ObsidianWeeklyDigest {
		if err := s.exportToObsidian(false, time.Now(), digests, config); err != nil {
			logging.Errorf("Failed to export weekly digest to Obsidian: %v", err)
		} else {
			logging.Infof("Weekly digest exported to Obsidian successfully")
		}
	}

	s.autoSendToOpenNotebook("weekly", time.Now(), digests)
}

func (s *DigestScheduler) GetStatus() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries := s.cron.Entries()
	nextRuns := make([]string, 0, len(entries))
	for _, entry := range entries {
		nextRuns = append(nextRuns, entry.Next.Format(time.RFC3339))
	}

	dailyTime := ""
	weeklyDay := 0
	weeklyTime := ""
	dailyEnabled := false
	weeklyEnabled := false

	if s.config != nil {
		dailyTime = s.config.DailyTime
		weeklyDay = s.config.WeeklyDay
		weeklyTime = s.config.WeeklyTime
		dailyEnabled = s.config.DailyEnabled
		weeklyEnabled = s.config.WeeklyEnabled
	}

	return map[string]interface{}{
		"running":        s.isRunning,
		"daily_enabled":  dailyEnabled,
		"weekly_enabled": weeklyEnabled,
		"daily_time":     dailyTime,
		"weekly_day":     weeklyDay,
		"weekly_time":    weeklyTime,
		"next_runs":      nextRuns,
		"active_jobs":    len(entries),
	}
}
