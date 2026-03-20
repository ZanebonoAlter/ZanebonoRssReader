package topicanalysis

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/domain/topictypes"
)

const (
	analysisWindowDaily  = "daily"
	analysisWindowWeekly = "weekly"
)

type AnalysisService interface {
	GetOrCreateAnalysis(tagID uint64, analysisType, windowType string, anchorDate time.Time) (*models.TopicTagAnalysis, error)
	GetAnalysis(tagID uint64, analysisType, windowType string, anchorDate time.Time) (*models.TopicTagAnalysis, error)
	RebuildAnalysis(tagID uint64, analysisType, windowType string, anchorDate time.Time) error
	GetAnalysisStatus(tagID uint64, analysisType, windowType string, anchorDate time.Time) (status string, progress float64, err error)
	EnqueueForSummary(summaryID uint64, priority int) error
}

type analysisService struct {
	db         *gorm.DB
	queue      AnalysisQueueWithSnapshot
	aiService  AIAnalyzer
	logger     *zap.Logger
	workerOnce sync.Once
}

var (
	analysisQueueInstance AnalysisQueueWithSnapshot
	analysisQueueOnce     sync.Once
	analysisServiceGlobal AnalysisService
	analysisServiceOnce   sync.Once
)

func getAnalysisQueue(db *gorm.DB, logger *zap.Logger) AnalysisQueueWithSnapshot {
	analysisQueueOnce.Do(func() {
		analysisQueueInstance = NewAnalysisQueue(db, logger)
	})
	return analysisQueueInstance
}

func NewAnalysisService(db *gorm.DB) AnalysisService {
	logger := zap.NewNop()
	svc := &analysisService{
		db:        db,
		queue:     getAnalysisQueue(db, logger),
		aiService: NewAIAnalysisService(logger),
		logger:    logger,
	}
	svc.startWorker()
	return svc
}

func GetAnalysisService(db *gorm.DB) AnalysisService {
	analysisServiceOnce.Do(func() {
		analysisServiceGlobal = NewAnalysisService(db)
	})
	return analysisServiceGlobal
}

func EnqueueTopicAnalysisForSummary(summaryID uint64, priority int, db *gorm.DB) error {
	service := GetAnalysisService(db)
	impl, ok := service.(interface {
		EnqueueForSummary(summaryID uint64, priority int) error
	})
	if !ok {
		return fmt.Errorf("analysis service does not support summary enqueue")
	}
	return impl.EnqueueForSummary(summaryID, priority)
}

func (s *analysisService) GetOrCreateAnalysis(tagID uint64, analysisType, windowType string, anchorDate time.Time) (*models.TopicTagAnalysis, error) {
	if err := validateAnalysisParams(analysisType, windowType); err != nil {
		return nil, err
	}

	anchor, err := normalizeAnalysisAnchor(windowType, anchorDate)
	if err != nil {
		return nil, err
	}

	return s.GetAnalysis(tagID, analysisType, windowType, anchor)
}

func (s *analysisService) GetAnalysis(tagID uint64, analysisType, windowType string, anchorDate time.Time) (*models.TopicTagAnalysis, error) {
	if err := validateAnalysisParams(analysisType, windowType); err != nil {
		return nil, err
	}

	anchor, err := normalizeAnalysisAnchor(windowType, anchorDate)
	if err != nil {
		return nil, err
	}

	var analysis models.TopicTagAnalysis
	err = s.db.Where("topic_tag_id = ? AND analysis_type = ? AND window_type = ? AND anchor_date = ?", tagID, analysisType, windowType, anchor).First(&analysis).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query analysis: %w", err)
	}
	return &analysis, nil
}

func (s *analysisService) RebuildAnalysis(tagID uint64, analysisType, windowType string, anchorDate time.Time) error {
	if err := validateAnalysisParams(analysisType, windowType); err != nil {
		return err
	}
	anchor, err := normalizeAnalysisAnchor(windowType, anchorDate)
	if err != nil {
		return err
	}
	return s.enqueue(tagID, analysisType, windowType, anchor, AnalysisPriorityHigh)
}

func (s *analysisService) GetAnalysisStatus(tagID uint64, analysisType, windowType string, anchorDate time.Time) (string, float64, error) {
	anchor, err := normalizeAnalysisAnchor(windowType, anchorDate)
	if err != nil {
		return "error", 0, err
	}

	if snapshot, ok := s.queue.GetLatestByKey(tagID, analysisType, windowType, anchor); ok {
		switch snapshot.Job.Status {
		case AnalysisStatusPending, AnalysisStatusProcessing:
			return snapshot.Job.Status, float64(snapshot.Progress) / 100, nil
		case AnalysisStatusFailed:
			return AnalysisStatusFailed, float64(snapshot.Progress) / 100, nil
		}
	}

	analysis, err := s.GetAnalysis(tagID, analysisType, windowType, anchor)
	if err != nil {
		return "error", 0, err
	}
	if analysis == nil {
		return "missing", 0, nil
	}
	return "ready", 1, nil
}

func (s *analysisService) EnqueueForSummary(summaryID uint64, priority int) error {
	tagIDs, err := s.fetchTagIDsBySummaryID(summaryID)
	if err != nil {
		return err
	}
	if len(tagIDs) == 0 {
		return nil
	}

	anchor := normalizeQueueAnchorDate(time.Now().In(topictypes.TopicGraphCST))
	for _, tagID := range tagIDs {
		for _, analysisType := range []string{AnalysisTypeEvent, AnalysisTypePerson, AnalysisTypeKeyword} {
			if err := s.enqueue(tagID, analysisType, WindowTypeDaily, anchor, priority); err != nil {
				log.Printf("[WARN] enqueue topic analysis failed summary=%d tag=%d type=%s: %v", summaryID, tagID, analysisType, err)
			}
		}
	}
	return nil
}

func (s *analysisService) enqueue(tagID uint64, analysisType, windowType string, anchorDate time.Time, priority int) error {
	return s.queue.Enqueue(&AnalysisJob{
		TopicTagID:   tagID,
		AnalysisType: analysisType,
		WindowType:   windowType,
		AnchorDate:   anchorDate,
		Priority:     normalizePriority(priority),
	})
}

func (s *analysisService) startWorker() {
	s.workerOnce.Do(func() {
		go s.runQueueWorker()
	})
}

func (s *analysisService) runQueueWorker() {
	for {
		job, err := s.queue.Dequeue()
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "closed") {
				return
			}
			log.Printf("[WARN] analysis dequeue failed: %v", err)
			continue
		}
		if job == nil {
			time.Sleep(150 * time.Millisecond)
			continue
		}

		_ = s.queue.MarkProgress(job.ID, 25)
		if _, err := s.buildAndPersist(job.TopicTagID, job.AnalysisType, job.WindowType, job.AnchorDate); err != nil {
			_ = s.queue.UpdateStatus(job.ID, AnalysisStatusFailed, err.Error())
			continue
		}
		_ = s.queue.Complete(job.ID)
	}
}

func (s *analysisService) buildAndPersist(tagID uint64, analysisType, windowType string, anchorDate time.Time) (*models.TopicTagAnalysis, error) {
	if err := validateAnalysisParams(analysisType, windowType); err != nil {
		return nil, err
	}

	anchor, err := normalizeAnalysisAnchor(windowType, anchorDate)
	if err != nil {
		return nil, err
	}

	var tag models.TopicTag
	if err := s.db.First(&tag, tagID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("topic tag %d not found", tagID)
		}
		return nil, fmt.Errorf("failed to load topic tag: %w", err)
	}

	windowStart, windowEnd, _, err := topictypes.ResolveWindow(windowType, anchor)
	if err != nil {
		return nil, err
	}

	summaries, err := s.fetchSummariesByTag(tagID, windowStart, windowEnd)
	if err != nil {
		return nil, err
	}

	lookup := models.TopicTagAnalysis{TopicTagID: tagID, AnalysisType: analysisType, WindowType: windowType, AnchorDate: anchor}
	var analysis models.TopicTagAnalysis
	err = s.db.Where(&lookup).First(&analysis).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		analysis = lookup
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to load existing analysis: %w", err)
	}

	maxID := maxSummaryID(summaries)
	cursor, err := s.getCursor(tagID, analysisType, windowType)
	if err != nil {
		return nil, err
	}
	if analysis.ID != 0 && cursor != nil && cursor.LastSummaryID >= maxID {
		return &analysis, nil
	}

	payloadJSON, source, err := s.buildPayloadJSON(tag, analysisType, summaries, windowType, anchor)
	if err != nil {
		return nil, err
	}

	analysis.SummaryCount = len(summaries)
	analysis.PayloadJSON = payloadJSON
	analysis.Source = source
	if analysis.Version <= 0 {
		analysis.Version = 1
	} else {
		analysis.Version++
	}

	if analysis.ID == 0 {
		if err := s.db.Create(&analysis).Error; err != nil {
			return nil, fmt.Errorf("failed to create analysis: %w", err)
		}
	} else {
		if err := s.db.Save(&analysis).Error; err != nil {
			return nil, fmt.Errorf("failed to update analysis: %w", err)
		}
	}

	if err := s.updateCursor(tagID, analysisType, windowType, summaries); err != nil {
		return nil, err
	}
	return &analysis, nil
}

func (s *analysisService) buildPayloadJSON(tag models.TopicTag, analysisType string, summaries []models.AISummary, windowType string, anchor time.Time) (string, string, error) {
	aiParams := AnalysisParams{
		TopicTagID:   uint64(tag.ID),
		TopicLabel:   tag.Label,
		AnalysisType: analysisType,
		WindowType:   windowType,
		AnchorDate:   anchor,
		Summaries:    mapSummaryInfos(summaries),
	}
	result, err := s.aiService.Analyze(nil, aiParams)
	if err == nil && result != nil {
		bytes, marshalErr := json.Marshal(result)
		if marshalErr == nil {
			raw := strings.TrimSpace(string(bytes))
			if raw != "" {
				return raw, "ai", nil
			}
		}
	}

	payload, fallbackErr := s.buildPayload(tag, analysisType, summaries, windowType)
	if fallbackErr != nil {
		if err != nil {
			return "", "", fmt.Errorf("ai analysis failed: %w; fallback failed: %v", err, fallbackErr)
		}
		return "", "", fallbackErr
	}
	bytes, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return "", "", fmt.Errorf("failed to marshal payload: %w", marshalErr)
	}
	return string(bytes), "heuristic", nil
}

const maxSummariesPerTag = 20

func (s *analysisService) fetchSummariesByTag(tagID uint64, windowStart, windowEnd time.Time) ([]models.AISummary, error) {
	var summaries []models.AISummary
	err := s.db.Model(&models.AISummary{}).
		Joins("JOIN ai_summary_topics ast ON ast.summary_id = ai_summaries.id").
		Where("ast.topic_tag_id = ?", tagID).
		Where("ai_summaries.created_at >= ? AND ai_summaries.created_at < ?", windowStart, windowEnd).
		Preload("Feed").
		Preload("Category").
		Order("ai_summaries.created_at DESC").
		Limit(maxSummariesPerTag).
		Find(&summaries).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query summaries for tag: %w", err)
	}
	return summaries, nil
}

func (s *analysisService) fetchTagIDsBySummaryID(summaryID uint64) ([]uint64, error) {
	rows := make([]struct {
		TopicTagID uint64 `gorm:"column:topic_tag_id"`
	}, 0)
	err := s.db.Table("ai_summary_topics").Select("topic_tag_id").Where("summary_id = ?", summaryID).Group("topic_tag_id").Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to load summary tags: %w", err)
	}
	out := make([]uint64, 0, len(rows))
	for _, row := range rows {
		if row.TopicTagID > 0 {
			out = append(out, row.TopicTagID)
		}
	}
	return out, nil
}

func (s *analysisService) needsRefresh(tagID uint64, analysisType string, windowType string, anchorDate time.Time, currentSummaryCount uint64) (bool, error) {
	windowStart, windowEnd, _, err := topictypes.ResolveWindow(windowType, anchorDate)
	if err != nil {
		return false, err
	}
	summaries, err := s.fetchSummariesByTag(tagID, windowStart, windowEnd)
	if err != nil {
		return false, err
	}
	latestID := maxSummaryID(summaries)
	cursor, err := s.getCursor(tagID, analysisType, windowType)
	if err != nil {
		return false, err
	}
	if uint64(len(summaries)) != currentSummaryCount {
		return true, nil
	}
	if cursor == nil {
		return latestID > 0, nil
	}
	return latestID > cursor.LastSummaryID, nil
}

func (s *analysisService) getCursor(tagID uint64, analysisType string, windowType string) (*models.TopicAnalysisCursor, error) {
	var cursor models.TopicAnalysisCursor
	err := s.db.Where("topic_tag_id = ? AND analysis_type = ? AND window_type = ?", tagID, analysisType, windowType).First(&cursor).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query analysis cursor: %w", err)
	}
	return &cursor, nil
}

func maxSummaryID(summaries []models.AISummary) uint64 {
	var maxID uint64
	for _, summary := range summaries {
		if uint64(summary.ID) > maxID {
			maxID = uint64(summary.ID)
		}
	}
	return maxID
}

func mapSummaryInfos(summaries []models.AISummary) []SummaryInfo {
	out := make([]SummaryInfo, 0, len(summaries))
	for _, summary := range summaries {
		item := SummaryInfo{
			SummaryID: uint64(summary.ID),
			Title:     summary.Title,
			Summary:   summary.Summary,
			CreatedAt: summary.CreatedAt.In(topictypes.TopicGraphCST).Format("2006-01-02"),
		}
		if summary.Feed != nil {
			item.FeedName = summary.Feed.Title
		}
		if summary.Category != nil {
			item.CategoryName = summary.Category.Name
		}
		out = append(out, item)
	}
	return out
}

func (s *analysisService) buildPayload(tag models.TopicTag, analysisType string, summaries []models.AISummary, windowType string) (map[string]any, error) {
	switch analysisType {
	case AnalysisTypeEvent:
		return s.buildEventPayload(tag, summaries), nil
	case AnalysisTypePerson:
		return s.buildPersonPayload(tag, summaries, windowType), nil
	case AnalysisTypeKeyword:
		return s.buildKeywordPayload(tag, summaries, windowType), nil
	default:
		return nil, fmt.Errorf("unsupported analysis type: %s", analysisType)
	}
}

func (s *analysisService) buildEventPayload(tag models.TopicTag, summaries []models.AISummary) map[string]any {
	articlesBySummary := topictypes.FetchArticlesForSummaries(summaries)
	timeline := make([]map[string]any, 0, len(summaries))
	keyMoments := make([]string, 0, minInt(len(summaries), 5))

	for i, summary := range summaries {
		sources := make([]map[string]any, 0, len(articlesBySummary[summary.ID]))
		for _, article := range articlesBySummary[summary.ID] {
			sources = append(sources, map[string]any{"article_id": article.ID, "title": article.Title, "link": article.Link})
		}

		timeline = append(timeline, map[string]any{
			"date":            summary.CreatedAt.In(topictypes.TopicGraphCST).Format("2006-01-02"),
			"title":           summary.Title,
			"summary":         truncateText(summary.Summary, 240),
			"source_articles": sources,
		})
		if i < 5 {
			keyMoments = append(keyMoments, summary.Title)
		}
	}

	return map[string]any{
		"timeline":       timeline,
		"key_moments":    keyMoments,
		"related_topics": s.fetchRelatedTopicLabels(uint64(tag.ID), 8),
	}
}

func (s *analysisService) buildPersonPayload(tag models.TopicTag, summaries []models.AISummary, windowType string) map[string]any {
	articlesBySummary := topictypes.FetchArticlesForSummaries(summaries)
	appearances := make([]map[string]any, 0, len(summaries))

	for _, summary := range summaries {
		item := map[string]any{
			"date":  summary.CreatedAt.In(topictypes.TopicGraphCST).Format("2006-01-02"),
			"scene": truncateText(summary.Title, 96),
			"quote": firstSentence(summary.Summary),
		}
		if cards := articlesBySummary[summary.ID]; len(cards) > 0 {
			item["article_title"] = cards[0].Title
			item["article_link"] = cards[0].Link
		}
		appearances = append(appearances, item)
	}

	background := "暂无背景资料"
	if len(summaries) > 0 {
		background = truncateText(summaries[0].Summary, 280)
	}

	return map[string]any{
		"profile": map[string]any{
			"name":       tag.Label,
			"role":       "相关人物",
			"background": background,
		},
		"appearances": appearances,
		"trend_data":  buildTrendData(summaries, windowType),
	}
}

func (s *analysisService) buildKeywordPayload(tag models.TopicTag, summaries []models.AISummary, windowType string) map[string]any {
	contextExamples := make([]map[string]any, 0, minInt(len(summaries), 6))
	for i, summary := range summaries {
		if i >= 6 {
			break
		}
		contextExamples = append(contextExamples, map[string]any{
			"text":   truncateText(summary.Summary, 220),
			"source": summary.Title,
		})
	}

	coOccurrence := s.fetchCoOccurrence(uint64(tag.ID), 12)
	relatedTopics := make([]string, 0, minInt(len(coOccurrence), 8))
	for i, item := range coOccurrence {
		if i >= 8 {
			break
		}
		name, _ := item["keyword"].(string)
		relatedTopics = append(relatedTopics, name)
	}

	return map[string]any{
		"trend_data":       buildTrendData(summaries, windowType),
		"related_topics":   relatedTopics,
		"co_occurrence":    coOccurrence,
		"context_examples": contextExamples,
	}
}

func (s *analysisService) fetchRelatedTopicLabels(tagID uint64, limit int) []string {
	type row struct{ Label string }
	rows := make([]row, 0)
	_ = s.db.Table("ai_summary_topics ast").
		Select("tt.label").
		Joins("JOIN ai_summary_topics ast2 ON ast2.summary_id = ast.summary_id").
		Joins("JOIN topic_tags tt ON tt.id = ast2.topic_tag_id").
		Where("ast.topic_tag_id = ?", tagID).
		Where("ast2.topic_tag_id <> ?", tagID).
		Group("tt.id, tt.label").
		Order("COUNT(*) DESC").
		Limit(limit).
		Scan(&rows).Error

	result := make([]string, 0, len(rows))
	for _, r := range rows {
		if strings.TrimSpace(r.Label) != "" {
			result = append(result, r.Label)
		}
	}
	return result
}

func (s *analysisService) fetchCoOccurrence(tagID uint64, limit int) []map[string]any {
	type row struct {
		Keyword string
		Count   int
	}
	rows := make([]row, 0)
	_ = s.db.Table("ai_summary_topics ast").
		Select("tt.label AS keyword, COUNT(*) AS count").
		Joins("JOIN ai_summary_topics ast2 ON ast2.summary_id = ast.summary_id").
		Joins("JOIN topic_tags tt ON tt.id = ast2.topic_tag_id").
		Where("ast.topic_tag_id = ?", tagID).
		Where("ast2.topic_tag_id <> ?", tagID).
		Group("tt.id, tt.label").
		Order("COUNT(*) DESC").
		Limit(limit).
		Scan(&rows).Error

	maxCount := 0
	for _, r := range rows {
		if r.Count > maxCount {
			maxCount = r.Count
		}
	}

	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		score := 0.0
		if maxCount > 0 {
			score = float64(r.Count) / float64(maxCount)
		}
		out = append(out, map[string]any{"keyword": r.Keyword, "score": score})
	}
	return out
}

func (s *analysisService) updateCursor(tagID uint64, analysisType, windowType string, summaries []models.AISummary) error {
	lookup := models.TopicAnalysisCursor{TopicTagID: tagID, AnalysisType: analysisType, WindowType: windowType}
	var cursor models.TopicAnalysisCursor
	err := s.db.Where(&lookup).First(&cursor).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		cursor = lookup
	} else if err != nil {
		return fmt.Errorf("failed to query analysis cursor: %w", err)
	}

	cursor.LastSummaryID = maxSummaryID(summaries)
	cursor.LastUpdatedAt = time.Now().In(topictypes.TopicGraphCST)

	if cursor.ID == 0 {
		if err := s.db.Create(&cursor).Error; err != nil {
			return fmt.Errorf("failed to create analysis cursor: %w", err)
		}
		return nil
	}
	if err := s.db.Save(&cursor).Error; err != nil {
		return fmt.Errorf("failed to update analysis cursor: %w", err)
	}
	return nil
}

func validateAnalysisParams(analysisType, windowType string) error {
	switch analysisType {
	case AnalysisTypeEvent, AnalysisTypePerson, AnalysisTypeKeyword:
	default:
		return fmt.Errorf("unsupported analysis type: %s", analysisType)
	}
	switch windowType {
	case analysisWindowDaily, analysisWindowWeekly:
	default:
		return fmt.Errorf("unsupported window type: %s", windowType)
	}
	return nil
}

func normalizeAnalysisAnchor(windowType string, anchorDate time.Time) (time.Time, error) {
	windowStart, _, _, err := topictypes.ResolveWindow(windowType, anchorDate)
	if err != nil {
		return time.Time{}, err
	}
	return windowStart, nil
}

func buildTrendData(summaries []models.AISummary, windowType string) []map[string]any {
	counter := map[string]int{}
	for _, summary := range summaries {
		key := summary.CreatedAt.In(topictypes.TopicGraphCST).Format("2006-01-02")
		counter[key]++
	}
	trend := make([]map[string]any, 0, len(counter))
	for date, count := range counter {
		trend = append(trend, map[string]any{"date": date, "count": count})
	}
	sort.SliceStable(trend, func(i, j int) bool {
		di, _ := trend[i]["date"].(string)
		dj, _ := trend[j]["date"].(string)
		return di < dj
	})
	if windowType == analysisWindowWeekly && len(trend) > 7 {
		return trend[len(trend)-7:]
	}
	return trend
}

func truncateText(value string, maxLen int) string {
	trimmed := strings.TrimSpace(value)
	if maxLen <= 0 || len(trimmed) <= maxLen {
		return trimmed
	}
	if maxLen <= 3 {
		return trimmed[:maxLen]
	}
	return trimmed[:maxLen-3] + "..."
}

func firstSentence(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	for _, sep := range []string{"。", "!", "?", "."} {
		if idx := strings.Index(trimmed, sep); idx > 0 {
			return strings.TrimSpace(trimmed[:idx+len(sep)])
		}
	}
	return truncateText(trimmed, 120)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
