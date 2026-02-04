package services

import (
	"math"
	"time"

	"gorm.io/gorm"
	"my-robot-backend/internal/models"
)

type PreferenceService struct {
	db *gorm.DB
}

func NewPreferenceService(db *gorm.DB) *PreferenceService {
	return &PreferenceService{db: db}
}

func (s *PreferenceService) UpdateAllPreferences() error {
	if err := s.updateFeedPreferences(); err != nil {
		return err
	}
	return s.updateCategoryPreferences()
}

func (s *PreferenceService) updateFeedPreferences() error {
	type FeedStats struct {
		FeedID           uint
		TotalEvents      int64
		TotalReadingTime int
		AvgScrollDepth   float64
		LastInteraction  time.Time
	}

	var feedStats []FeedStats
	if err := s.db.Model(&models.ReadingBehavior{}).
		Select(`
			feed_id,
			COUNT(*) as total_events,
			COALESCE(SUM(reading_time), 0) as total_reading_time,
			COALESCE(AVG(scroll_depth), 0) as avg_scroll_depth,
			MAX(created_at) as last_interaction
		`).
		Where("feed_id IS NOT NULL").
		Group("feed_id").
		Scan(&feedStats).Error; err != nil {
		return err
	}

	for _, stats := range feedStats {
		preferenceScore := s.calculatePreferenceScore(
			stats.TotalEvents,
			stats.TotalReadingTime,
			stats.AvgScrollDepth,
			stats.LastInteraction,
		)

		avgReadingTime := 0
		if stats.TotalEvents > 0 {
			avgReadingTime = stats.TotalReadingTime / int(stats.TotalEvents)
		}

		var pref models.UserPreference
		if err := s.db.Where("feed_id = ?", stats.FeedID).First(&pref).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				pref = models.UserPreference{
					FeedID:            &stats.FeedID,
					PreferenceScore:   preferenceScore,
					AvgReadingTime:    avgReadingTime,
					InteractionCount:  int(stats.TotalEvents),
					ScrollDepthAvg:    stats.AvgScrollDepth,
					LastInteractionAt: &stats.LastInteraction,
				}
				s.db.Create(&pref)
			} else {
				return err
			}
		} else {
			pref.PreferenceScore = preferenceScore
			pref.AvgReadingTime = avgReadingTime
			pref.InteractionCount = int(stats.TotalEvents)
			pref.ScrollDepthAvg = stats.AvgScrollDepth
			pref.LastInteractionAt = &stats.LastInteraction
			s.db.Save(&pref)
		}
	}

	return nil
}

func (s *PreferenceService) updateCategoryPreferences() error {
	type CategoryStats struct {
		CategoryID       *uint
		TotalEvents      int64
		TotalReadingTime int
		AvgScrollDepth   float64
		LastInteraction  time.Time
	}

	var categoryStats []CategoryStats
	if err := s.db.Model(&models.ReadingBehavior{}).
		Select(`
			category_id,
			COUNT(*) as total_events,
			COALESCE(SUM(reading_time), 0) as total_reading_time,
			COALESCE(AVG(scroll_depth), 0) as avg_scroll_depth,
			MAX(created_at) as last_interaction
		`).
		Where("category_id IS NOT NULL").
		Group("category_id").
		Scan(&categoryStats).Error; err != nil {
		return err
	}

	for _, stats := range categoryStats {
		preferenceScore := s.calculatePreferenceScore(
			stats.TotalEvents,
			stats.TotalReadingTime,
			stats.AvgScrollDepth,
			stats.LastInteraction,
		)

		avgReadingTime := 0
		if stats.TotalEvents > 0 {
			avgReadingTime = stats.TotalReadingTime / int(stats.TotalEvents)
		}

		var pref models.UserPreference
		if err := s.db.Where("category_id = ?", stats.CategoryID).First(&pref).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				pref = models.UserPreference{
					CategoryID:        stats.CategoryID,
					PreferenceScore:   preferenceScore,
					AvgReadingTime:    avgReadingTime,
					InteractionCount:  int(stats.TotalEvents),
					ScrollDepthAvg:    stats.AvgScrollDepth,
					LastInteractionAt: &stats.LastInteraction,
				}
				s.db.Create(&pref)
			} else {
				return err
			}
		} else {
			pref.PreferenceScore = preferenceScore
			pref.AvgReadingTime = avgReadingTime
			pref.InteractionCount = int(stats.TotalEvents)
			pref.ScrollDepthAvg = stats.AvgScrollDepth
			pref.LastInteractionAt = &stats.LastInteraction
			s.db.Save(&pref)
		}
	}

	return nil
}

func (s *PreferenceService) calculatePreferenceScore(
	totalEvents int64,
	totalReadingTime int,
	avgScrollDepth float64,
	lastInteraction time.Time,
) float64 {
	if totalEvents == 0 {
		return 0
	}

	scrollScore := avgScrollDepth / 100.0 * 0.4

	avgTime := float64(totalReadingTime) / float64(totalEvents)
	readingTimeScore := math.Min(avgTime/180.0, 1.0) * 0.3

	eventsScore := math.Min(float64(totalEvents)/50.0, 1.0) * 0.3

	baseScore := scrollScore + readingTimeScore + eventsScore

	daysSinceInteraction := time.Since(lastInteraction).Hours() / 24
	decayFactor := math.Exp(-daysSinceInteraction / 30.0)

	finalScore := baseScore * decayFactor

	return math.Max(0, math.Min(finalScore, 1.0))
}

func (s *PreferenceService) GetUserFeedPreferences() ([]models.UserPreference, error) {
	var preferences []models.UserPreference
	err := s.db.Where("feed_id IS NOT NULL").
		Preload("Feed").
		Order("preference_score DESC").
		Find(&preferences).Error
	return preferences, err
}

func (s *PreferenceService) GetUserCategoryPreferences() ([]models.UserPreference, error) {
	var preferences []models.UserPreference
	err := s.db.Where("category_id IS NOT NULL").
		Preload("Category").
		Order("preference_score DESC").
		Find(&preferences).Error
	return preferences, err
}

func (s *PreferenceService) GetTopPreferredFeeds(limit int) ([]uint, error) {
	type FeedIDScore struct {
		FeedID uint
		Score  float64
	}

	var results []FeedIDScore
	err := s.db.Model(&models.UserPreference{}).
		Select("feed_id, preference_score as score").
		Where("feed_id IS NOT NULL").
		Order("preference_score DESC").
		Limit(limit).
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	feedIDs := make([]uint, len(results))
	for i, r := range results {
		feedIDs[i] = r.FeedID
	}

	return feedIDs, nil
}

func (s *PreferenceService) GetTopPreferredCategories(limit int) ([]uint, error) {
	type CategoryIDScore struct {
		CategoryID *uint
		Score      float64
	}

	var results []CategoryIDScore
	err := s.db.Model(&models.UserPreference{}).
		Select("category_id, preference_score as score").
		Where("category_id IS NOT NULL").
		Order("preference_score DESC").
		Limit(limit).
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	categoryIDs := make([]uint, 0, len(results))
	for _, r := range results {
		if r.CategoryID != nil {
			categoryIDs = append(categoryIDs, *r.CategoryID)
		}
	}

	return categoryIDs, nil
}
