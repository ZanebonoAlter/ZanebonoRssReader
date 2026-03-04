package digest

import (
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"
	"my-robot-backend/pkg/database"
)

type DigestScheduler struct {
	cron      *cron.Cron
	isRunning bool
}

func NewDigestScheduler() *DigestScheduler {
	return &DigestScheduler{
		cron:      cron.New(),
		isRunning: false,
	}
}

func (s *DigestScheduler) Start() error {
	if s.isRunning {
		log.Println("Digest scheduler already running")
		return nil
	}

	config, err := s.LoadConfig()
	if err != nil {
		log.Printf("Failed to load digest config: %v", err)
		return err
	}

	if config.DailyEnabled {
		dailyExpr := fmt.Sprintf("0 %s * * *", config.DailyTime)
		if _, err := s.cron.AddFunc(dailyExpr, s.generateDailyDigest); err != nil {
			return fmt.Errorf("failed to schedule daily digest: %w", err)
		}
		log.Printf("Daily digest scheduled at %s", config.DailyTime)
	}

	if config.WeeklyEnabled {
		weeklyExpr := fmt.Sprintf("0 %s * * %s", config.WeeklyTime, s.weekdayToNumber(config.WeeklyDay))
		if _, err := s.cron.AddFunc(weeklyExpr, s.generateWeeklyDigest); err != nil {
			return fmt.Errorf("failed to schedule weekly digest: %w", err)
		}
		log.Printf("Weekly digest scheduled at %s on %s", config.WeeklyTime, s.weekdayToString(config.WeeklyDay))
	}

	s.cron.Start()
	s.isRunning = true
	log.Println("Digest scheduler started")
	return nil
}

func (s *DigestScheduler) Stop() {
	if !s.isRunning {
		return
	}
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.isRunning = false
	log.Println("Digest scheduler stopped")
}

func (s *DigestScheduler) LoadConfig() (*DigestConfig, error) {
	var config DigestConfig
	err := database.DB.First(&config).Error
	if err != nil {
		return nil, fmt.Errorf("failed to load digest config: %w", err)
	}
	return &config, nil
}

func (s *DigestScheduler) weekdayToNumber(day int) string {
	weekdayMap := map[int]string{
		0: "0",
		1: "1",
		2: "2",
		3: "3",
		4: "4",
		5: "5",
		6: "6",
	}
	return weekdayMap[day]
}

func (s *DigestScheduler) weekdayToString(day int) string {
	weekdayMap := map[int]string{
		0: "Sunday",
		1: "Monday",
		2: "Tuesday",
		3: "Wednesday",
		4: "Thursday",
		5: "Friday",
		6: "Saturday",
	}
	return weekdayMap[day]
}

func (s *DigestScheduler) generateDailyDigest() {
	log.Println("Starting daily digest generation")
}

func (s *DigestScheduler) generateWeeklyDigest() {
	log.Println("Starting weekly digest generation")
}

func (s *DigestScheduler) GetStatus() map[string]interface{} {
	entries := s.cron.Entries()

	var nextRun time.Time
	if len(entries) > 0 {
		nextRun = entries[0].Next
	}

	config, err := s.LoadConfig()
	if err != nil {
		log.Printf("Failed to load config for status: %v", err)
	}

	dailyTime := ""
	weeklyDay := 0
	weeklyTime := ""
	dailyEnabled := false
	weeklyEnabled := false

	if config != nil {
		dailyTime = config.DailyTime
		weeklyDay = config.WeeklyDay
		weeklyTime = config.WeeklyTime
		dailyEnabled = config.DailyEnabled
		weeklyEnabled = config.WeeklyEnabled
	}

	return map[string]interface{}{
		"running":        s.isRunning,
		"daily_enabled":  dailyEnabled,
		"weekly_enabled": weeklyEnabled,
		"daily_time":     dailyTime,
		"weekly_day":     weeklyDay,
		"weekly_time":    weeklyTime,
		"next_run":       nextRun.Format(time.RFC3339),
		"active_jobs":    len(entries),
	}
}
