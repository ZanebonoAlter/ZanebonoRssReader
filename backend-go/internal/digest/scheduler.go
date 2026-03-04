package digest

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"my-robot-backend/pkg/database"
)

type DigestScheduler struct {
	cron      *cron.Cron
	isRunning bool
	mu        sync.Mutex
	config    *DigestConfig
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
		log.Println("Digest scheduler already running")
		return nil
	}

	config, err := s.loadOrCreateConfig()
	if err != nil {
		log.Printf("Failed to load digest config: %v", err)
		return err
	}
	s.config = config

	if config.DailyEnabled {
		dailyExpr := fmt.Sprintf("0 %s * * *", config.DailyTime)
		if _, err := s.cron.AddFunc(dailyExpr, s.generateDailyDigest); err != nil {
			return fmt.Errorf("failed to schedule daily digest: %w", err)
		}
		log.Printf("Daily digest scheduled at %s", config.DailyTime)
	}

	if config.WeeklyEnabled {
		cronDay := s.intToCronDay(config.WeeklyDay)
		weeklyExpr := fmt.Sprintf("0 %s * * %s", config.WeeklyTime, cronDay)
		if _, err := s.cron.AddFunc(weeklyExpr, s.generateWeeklyDigest); err != nil {
			return fmt.Errorf("failed to schedule weekly digest: %w", err)
		}
		log.Printf("Weekly digest scheduled at %s on day %d", config.WeeklyTime, config.WeeklyDay)
	}

	s.cron.Start()
	s.isRunning = true
	log.Println("Digest scheduler started")
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
	log.Println("Digest scheduler stopped")
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
		log.Println("Created default digest config")
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

func (s *DigestScheduler) generateDailyDigest() {
	log.Println("Starting daily digest generation")
}

func (s *DigestScheduler) generateWeeklyDigest() {
	log.Println("Starting weekly digest generation")
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
