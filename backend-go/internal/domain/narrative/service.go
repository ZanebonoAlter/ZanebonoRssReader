package narrative

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/logging"
)

type NarrativeService struct{}

func NewNarrativeService() *NarrativeService {
	return &NarrativeService{}
}

func (s *NarrativeService) DeleteByDate(date time.Time, scopeType string, categoryID *uint) (int, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := database.DB.Where("period = ? AND period_date >= ? AND period_date < ?", "daily", startOfDay, endOfDay)

	if scopeType != "" {
		query = query.Where("scope_type = ?", scopeType)
		if categoryID != nil {
			query = query.Where("scope_category_id = ?", *categoryID)
		}
	}

	result := query.Delete(&models.NarrativeSummary{})
	if result.Error != nil {
		return 0, fmt.Errorf("delete narratives for %s: %w", date.Format("2006-01-02"), result.Error)
	}

	logging.Infof("narrative: deleted %d existing narratives for %s (scope=%s)", result.RowsAffected, date.Format("2006-01-02"), scopeType)
	return int(result.RowsAffected), nil
}

func (s *NarrativeService) RegenerateAndSave(date time.Time) (int, error) {
	deleted, err := s.DeleteByDate(date, "", nil)
	if err != nil {
		return 0, err
	}
	logging.Infof("narrative: deleted %d old narratives before regenerating for %s", deleted, date.Format("2006-01-02"))

	saved, err := s.GenerateAndSave(date)
	if err != nil {
		return saved, err
	}

	catSaved, catErr := s.GenerateAndSaveForAllCategories(date)
	if catErr != nil {
		logging.Warnf("narrative: category regeneration failed during full regen: %v", catErr)
	}
	return saved + catSaved, nil
}

func (s *NarrativeService) RegenerateAndSaveForCategory(date time.Time, categoryID uint) (int, error) {
	deleted, err := s.DeleteByDate(date, models.NarrativeScopeTypeFeedCategory, &categoryID)
	if err != nil {
		return 0, err
	}
	logging.Infof("narrative: deleted %d old category narratives for category %d before regenerating", deleted, categoryID)

	var cat models.Category
	if err := database.DB.Where("id = ?", categoryID).First(&cat).Error; err != nil {
		return 0, fmt.Errorf("category %d not found: %w", categoryID, err)
	}

	return s.GenerateAndSaveForCategory(date, categoryID, cat.Name)
}

type ScopeSaveOpts struct {
	ScopeType   string
	CategoryID  *uint
	Label       string
}

func (s *NarrativeService) GenerateAndSave(date time.Time) (int, error) {
	tagInputs, err := CollectTagInputs(date)
	if err != nil {
		return 0, fmt.Errorf("collect tag inputs: %w", err)
	}
	if len(tagInputs) == 0 {
		logging.Infof("narrative: no tag inputs for %s, skipping", date.Format("2006-01-02"))
		return 0, nil
	}

	prevNarratives, err := CollectPreviousNarratives(date, "", nil)
	if err != nil {
		return 0, fmt.Errorf("collect previous narratives: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	outputs, err := GenerateNarratives(ctx, tagInputs, prevNarratives)
	if err != nil {
		return 0, fmt.Errorf("generate narratives: %w", err)
	}
	if len(outputs) == 0 {
		logging.Infof("narrative: no narratives generated for %s", date.Format("2006-01-02"))
		return 0, nil
	}

	saved, err := saveNarratives(outputs, date, nil)
	if err != nil {
		return 0, fmt.Errorf("save narratives: %w", err)
	}

	markEndedNarratives(date, outputs, prevNarratives)

	go feedbackNarrativesToTags(outputs)
	go GenerateWatchedTagNarratives(date)

	logging.Infof("narrative: saved %d global narratives for %s", saved, date.Format("2006-01-02"))
	return saved, nil
}

func (s *NarrativeService) GenerateAndSaveForCategory(date time.Time, categoryID uint, categoryLabel string) (int, error) {
	tagInputs, err := CollectTagInputsByCategory(date, categoryID)
	if err != nil {
		return 0, fmt.Errorf("collect tag inputs for category %d: %w", categoryID, err)
	}
	if len(tagInputs) == 0 {
		logging.Infof("narrative: no tag inputs for category %d on %s, skipping", categoryID, date.Format("2006-01-02"))
		return 0, nil
	}

	articleCount := 0
	for _, t := range tagInputs {
		articleCount += t.ArticleCount
	}
	if articleCount < 5 {
		logging.Infof("narrative: category %d has only %d articles on %s, below threshold", categoryID, articleCount, date.Format("2006-01-02"))
		return 0, nil
	}

	validTagCount := 0
	for _, t := range tagInputs {
		if t.ArticleCount > 0 {
			validTagCount++
		}
	}
	if validTagCount < 3 {
		logging.Infof("narrative: category %d has only %d valid tags on %s, below threshold", categoryID, validTagCount, date.Format("2006-01-02"))
		return 0, nil
	}

	prevNarratives, err := CollectPreviousNarratives(date, models.NarrativeScopeTypeFeedCategory, &categoryID)
	if err != nil {
		return 0, fmt.Errorf("collect previous narratives for category %d: %w", categoryID, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	outputs, err := GenerateNarratives(ctx, tagInputs, prevNarratives)
	if err != nil {
		return 0, fmt.Errorf("generate narratives for category %d: %w", categoryID, err)
	}

	if len(outputs) > 5 {
		outputs = outputs[:5]
	}
	if len(outputs) == 0 {
		logging.Infof("narrative: no narratives generated for category %d on %s", categoryID, date.Format("2006-01-02"))
		return 0, nil
	}

	catID := categoryID
	opts := &ScopeSaveOpts{
		ScopeType:  models.NarrativeScopeTypeFeedCategory,
		CategoryID: &catID,
		Label:      categoryLabel,
	}

	saved, err := saveNarratives(outputs, date, opts)
	if err != nil {
		return 0, fmt.Errorf("save category narratives: %w", err)
	}

	logging.Infof("narrative: saved %d narratives for category %d (%s) on %s", saved, categoryID, categoryLabel, date.Format("2006-01-02"))
	return saved, nil
}

func (s *NarrativeService) GenerateAndSaveForAllCategories(date time.Time) (int, error) {
	categories, err := CollectActiveCategories(date)
	if err != nil {
		return 0, fmt.Errorf("collect active categories: %w", err)
	}
	if len(categories) == 0 {
		logging.Infof("narrative: no active categories for %s", date.Format("2006-01-02"))
		return 0, nil
	}

	totalSaved := 0
	for _, cat := range categories {
		saved, err := s.GenerateAndSaveForCategory(date, cat.ID, cat.Name)
		if err != nil {
			logging.Warnf("narrative: failed to generate for category %d (%s): %v", cat.ID, cat.Name, err)
			continue
		}
		totalSaved += saved
	}

	logging.Infof("narrative: saved %d category narratives across %d categories for %s", totalSaved, len(categories), date.Format("2006-01-02"))
	return totalSaved, nil
}

func saveNarratives(outputs []NarrativeOutput, date time.Time, scopeOpts *ScopeSaveOpts) (int, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	records := make([]models.NarrativeSummary, 0, len(outputs))
	for _, out := range outputs {
		parentIDsJSON, _ := json.Marshal(out.ParentIDs)
		tagIDsJSON, _ := json.Marshal(out.RelatedTagIDs)

		articleIDs := resolveArticleIDs(out.RelatedTagIDs, date)
		articleIDsJSON, _ := json.Marshal(articleIDs)

		generation := resolveGeneration(out, date)

		status := out.Status
		if status == "" {
			status = models.NarrativeStatusEmerging
		}

		record := models.NarrativeSummary{
			Title:             out.Title,
			Summary:           out.Summary,
			Status:            status,
			Period:            "daily",
			PeriodDate:        startOfDay,
			Generation:        generation,
			ParentIDs:         string(parentIDsJSON),
			RelatedTagIDs:     string(tagIDsJSON),
			RelatedArticleIDs: string(articleIDsJSON),
			Source:            "ai",
			ScopeType:         models.NarrativeScopeTypeGlobal,
		}
		if scopeOpts != nil {
			record.ScopeType = scopeOpts.ScopeType
			record.ScopeCategoryID = scopeOpts.CategoryID
			record.ScopeLabel = scopeOpts.Label
		}
		records = append(records, record)
	}

	if err := database.DB.CreateInBatches(records, 20).Error; err != nil {
		logging.Warnf("narrative: failed to batch save narratives: %v", err)
		saved := 0
		for _, record := range records {
			if err := database.DB.Create(&record).Error; err != nil {
				logging.Warnf("narrative: failed to save '%s': %v", record.Title, err)
				continue
			}
			saved++
		}
		return saved, nil
	}
	return len(records), nil
}

func resolveGeneration(out NarrativeOutput, date time.Time) int {
	if len(out.ParentIDs) == 0 {
		return 0
	}

	var prevNarratives []models.NarrativeSummary
	database.DB.Where("id IN ?", out.ParentIDs).Find(&prevNarratives)

	maxGen := -1
	for _, n := range prevNarratives {
		if n.Generation > maxGen {
			maxGen = n.Generation
		}
	}
	if maxGen < 0 {
		return 0
	}
	return maxGen + 1
}

func resolveArticleIDs(tagIDs []uint, date time.Time) []uint64 {
	if len(tagIDs) == 0 {
		return nil
	}

	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var articleIDs []uint64
	if err := database.DB.Model(&models.ArticleTopicTag{}).
		Select("DISTINCT article_topic_tags.article_id").
		Joins("JOIN articles ON articles.id = article_topic_tags.article_id").
		Where("article_topic_tags.topic_tag_id IN ? AND articles.pub_date >= ? AND articles.pub_date < ?", tagIDs, startOfDay, endOfDay).
		Pluck("article_topic_tags.article_id", &articleIDs).Error; err != nil {
		logging.Warnf("narrative: resolveArticleIDs failed: %v", err)
	}

	return articleIDs
}

func markEndedNarratives(date time.Time, currentOutputs []NarrativeOutput, prev []PreviousNarrative) {
	if len(prev) == 0 {
		return
	}

	referencedParentIDs := make(map[uint64]bool)
	for _, out := range currentOutputs {
		for _, pid := range out.ParentIDs {
			referencedParentIDs[uint64(pid)] = true
		}
	}

	var endedIDs []uint64
	for _, p := range prev {
		if !referencedParentIDs[p.ID] && p.Status != models.NarrativeStatusEnding {
			endedIDs = append(endedIDs, p.ID)
		}
	}

	if len(endedIDs) == 0 {
		return
	}

	result := database.DB.Model(&models.NarrativeSummary{}).
		Where("id IN ? AND status != ?", endedIDs, models.NarrativeStatusEnding).
		Update("status", models.NarrativeStatusEnding)

	logging.Infof("narrative: marked %d previous narratives as ending", result.RowsAffected)
}

type NarrativeListItem struct {
	ID          uint64     `json:"id"`
	Title       string     `json:"title"`
	Summary     string     `json:"summary"`
	Status      string     `json:"status"`
	Period      string     `json:"period"`
	PeriodDate  string     `json:"period_date"`
	Generation  int        `json:"generation"`
	ParentIDs   []uint64   `json:"parent_ids"`
	RelatedTags []TagBrief `json:"related_tags"`
	ChildIDs    []uint64   `json:"child_ids"`
}

type TagBrief struct {
	ID       uint   `json:"id"`
	Slug     string `json:"slug"`
	Label    string `json:"label"`
	Category string `json:"category"`
	Kind     string `json:"kind,omitempty"`
}

type TimelineDay struct {
	Date       string              `json:"date"`
	Narratives []NarrativeListItem `json:"narratives"`
}

func (s *NarrativeService) GetTimeline(anchorDate time.Time, days int, scopeType string, categoryID *uint) ([]TimelineDay, error) {
	if days <= 0 {
		days = 7
	}
	if days > 30 {
		days = 30
	}

	startOfAnchor := time.Date(anchorDate.Year(), anchorDate.Month(), anchorDate.Day(), 0, 0, 0, 0, anchorDate.Location())
	rangeStart := startOfAnchor.AddDate(0, 0, -(days - 1))
	rangeEnd := startOfAnchor.Add(24 * time.Hour)

	query := database.DB.
		Where("period = ? AND period_date >= ? AND period_date < ?", "daily", rangeStart, rangeEnd)

	if scopeType != "" {
		query = query.Where("scope_type = ?", scopeType)
		if categoryID != nil {
			query = query.Where("scope_category_id = ?", *categoryID)
		}
	} else {
		query = query.Where("scope_type = ?", models.NarrativeScopeTypeGlobal)
	}

	var narratives []models.NarrativeSummary
	if err := query.
		Order("period_date ASC, generation ASC, id ASC").
		Find(&narratives).Error; err != nil {
		return nil, fmt.Errorf("query narrative timeline: %w", err)
	}

	grouped := make(map[string][]models.NarrativeSummary)
	for _, n := range narratives {
		key := n.PeriodDate.Format("2006-01-02")
		grouped[key] = append(grouped[key], n)
	}

	allItems := toListItems(narratives)
	itemByID := make(map[uint64]NarrativeListItem)
	for _, item := range allItems {
		itemByID[item.ID] = item
	}

	var result []TimelineDay
	for d := rangeStart; d.Before(rangeEnd); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		dayItems := make([]NarrativeListItem, 0)
		if ns, ok := grouped[key]; ok {
			for _, n := range ns {
				if item, found := itemByID[n.ID]; found {
					dayItems = append(dayItems, item)
				}
			}
		}
		result = append(result, TimelineDay{
			Date:       key,
			Narratives: dayItems,
		})
	}

	return result, nil
}

func (s *NarrativeService) GetByDate(date time.Time, scopeType string, categoryID *uint) ([]NarrativeListItem, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := database.DB.
		Where("period = ? AND period_date >= ? AND period_date < ?", "daily", startOfDay, endOfDay)

	if scopeType != "" {
		query = query.Where("scope_type = ?", scopeType)
		if categoryID != nil {
			query = query.Where("scope_category_id = ?", *categoryID)
		}
	} else {
		query = query.Where("scope_type = ?", models.NarrativeScopeTypeGlobal)
	}

	var narratives []models.NarrativeSummary
	if err := query.
		Order("generation ASC, id ASC").
		Find(&narratives).Error; err != nil {
		return nil, fmt.Errorf("query narratives by date: %w", err)
	}

	return toListItems(narratives), nil
}

type NarrativeScopeItem struct {
	CategoryID      uint   `json:"category_id"`
	CategoryName    string `json:"category_name"`
	CategoryIcon    string `json:"category_icon"`
	CategoryColor   string `json:"category_color"`
	NarrativeCount  int    `json:"narrative_count"`
	LastGeneratedAt string `json:"last_generated_at"`
}

type NarrativeScopesResponse struct {
	Date        string               `json:"date"`
	GlobalCount int                  `json:"global_count"`
	Categories  []NarrativeScopeItem `json:"categories"`
}

func (s *NarrativeService) GetScopes(date time.Time) (*NarrativeScopesResponse, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var globalCount int64
	database.DB.Model(&models.NarrativeSummary{}).
		Where("scope_type = ? AND period_date >= ? AND period_date < ?", models.NarrativeScopeTypeGlobal, startOfDay, endOfDay).
		Count(&globalCount)

	type catRow struct {
		ScopeCategoryID uint   `json:"scope_category_id"`
		ScopeLabel      string `json:"scope_label"`
		Cnt             int    `json:"cnt"`
	}
	var catRows []catRow
	database.DB.Model(&models.NarrativeSummary{}).
		Select("scope_category_id, scope_label, COUNT(*) as cnt").
		Where("scope_type = ? AND period_date >= ? AND period_date < ?", models.NarrativeScopeTypeFeedCategory, startOfDay, endOfDay).
		Group("scope_category_id, scope_label").
		Scan(&catRows)

	var items []NarrativeScopeItem
	if len(catRows) > 0 {
		catIDs := make([]uint, 0, len(catRows))
		for _, row := range catRows {
			catIDs = append(catIDs, row.ScopeCategoryID)
		}
		var categories []models.Category
		database.DB.Where("id IN ?", catIDs).Find(&categories)
		catMap := make(map[uint]models.Category, len(categories))
		for _, c := range categories {
			catMap[c.ID] = c
		}

		type lastRow struct {
			ScopeCategoryID uint   `json:"scope_category_id"`
			CreatedAt       string `json:"created_at"`
		}
		var lastRows []lastRow
		database.DB.Model(&models.NarrativeSummary{}).
			Select("scope_category_id, MAX(created_at) as created_at").
			Where("scope_type = ? AND scope_category_id IN ? AND period_date >= ? AND period_date < ?",
				models.NarrativeScopeTypeFeedCategory, catIDs, startOfDay, endOfDay).
			Group("scope_category_id").
			Scan(&lastRows)
		lastMap := make(map[uint]string, len(lastRows))
		for _, lr := range lastRows {
			lastMap[lr.ScopeCategoryID] = lr.CreatedAt
		}

		for _, row := range catRows {
			cat, ok := catMap[row.ScopeCategoryID]
			if ok {
				items = append(items, NarrativeScopeItem{
					CategoryID:      cat.ID,
					CategoryName:    cat.Name,
					CategoryIcon:    cat.Icon,
					CategoryColor:   cat.Color,
					NarrativeCount:  row.Cnt,
					LastGeneratedAt: lastMap[cat.ID],
				})
			} else {
				items = append(items, NarrativeScopeItem{
					CategoryID:      row.ScopeCategoryID,
					CategoryName:    row.ScopeLabel,
					NarrativeCount:  row.Cnt,
					LastGeneratedAt: lastMap[row.ScopeCategoryID],
				})
			}
		}
	}

	return &NarrativeScopesResponse{
		Date:        startOfDay.Format("2006-01-02"),
		GlobalCount: int(globalCount),
		Categories:  items,
	}, nil
}

func (s *NarrativeService) GetNarrativeTree(narrativeID uint64) (*NarrativeListItem, error) {
	var narrative models.NarrativeSummary
	if err := database.DB.Where("id = ?", narrativeID).First(&narrative).Error; err != nil {
		return nil, fmt.Errorf("query narrative %d: %w", narrativeID, err)
	}

	items := toListItems([]models.NarrativeSummary{narrative})
	if len(items) == 0 {
		return nil, fmt.Errorf("failed to build list item for narrative %d", narrativeID)
	}
	return &items[0], nil
}

func (s *NarrativeService) GetNarrativeHistory(narrativeID uint64) ([]NarrativeListItem, error) {
	var narrative models.NarrativeSummary
	if err := database.DB.Where("id = ?", narrativeID).First(&narrative).Error; err != nil {
		return nil, fmt.Errorf("query narrative %d: %w", narrativeID, err)
	}

	var history []models.NarrativeSummary
	visited := make(map[uint64]bool)
	walkHistory(narrativeID, &history, visited)

	return toListItems(history), nil
}

func walkHistory(id uint64, history *[]models.NarrativeSummary, visited map[uint64]bool) {
	walkHistoryDepth(id, history, visited, 0, 30)
}

func walkHistoryDepth(id uint64, history *[]models.NarrativeSummary, visited map[uint64]bool, depth, maxDepth int) {
	if depth > maxDepth || visited[id] {
		return
	}
	visited[id] = true

	var narrative models.NarrativeSummary
	if err := database.DB.Where("id = ?", id).First(&narrative).Error; err != nil {
		return
	}

	var parentIDs []uint64
	if narrative.ParentIDs != "" {
		json.Unmarshal([]byte(narrative.ParentIDs), &parentIDs)
	}

	for _, pid := range parentIDs {
		walkHistoryDepth(pid, history, visited, depth+1, maxDepth)
	}

	*history = append(*history, narrative)
}

func toListItems(narratives []models.NarrativeSummary) []NarrativeListItem {
	if len(narratives) == 0 {
		return nil
	}

	tagIDSet := make(map[uint]bool)
	for _, n := range narratives {
		var tagIDs []uint
		if n.RelatedTagIDs != "" {
			json.Unmarshal([]byte(n.RelatedTagIDs), &tagIDs)
		}
		for _, id := range tagIDs {
			tagIDSet[id] = true
		}
	}

	tagBriefMap := make(map[uint]TagBrief)
	if len(tagIDSet) > 0 {
		tagIDs := make([]uint, 0, len(tagIDSet))
		for id := range tagIDSet {
			tagIDs = append(tagIDs, id)
		}
		var tags []models.TopicTag
		database.DB.Where("id IN ?", tagIDs).Find(&tags)
		for _, t := range tags {
			tagBriefMap[t.ID] = TagBrief{ID: t.ID, Slug: t.Slug, Label: t.Label, Category: t.Category, Kind: t.Kind}
		}
	}

	childMap := make(map[uint64][]uint64)
	for _, n := range narratives {
		var parentIDs []uint64
		if n.ParentIDs != "" {
			json.Unmarshal([]byte(n.ParentIDs), &parentIDs)
		}
		for _, pid := range parentIDs {
			childMap[pid] = append(childMap[pid], n.ID)
		}
	}

	items := make([]NarrativeListItem, 0, len(narratives))
	for _, n := range narratives {
		var parentIDs []uint64
		if n.ParentIDs != "" {
			json.Unmarshal([]byte(n.ParentIDs), &parentIDs)
		}

		var tagIDs []uint
		if n.RelatedTagIDs != "" {
			json.Unmarshal([]byte(n.RelatedTagIDs), &tagIDs)
		}

		tagBriefs := make([]TagBrief, 0, len(tagIDs))
		for _, tid := range tagIDs {
			if brief, ok := tagBriefMap[tid]; ok {
				tagBriefs = append(tagBriefs, brief)
			}
		}

		childIDs := childMap[n.ID]
		if childIDs == nil {
			childIDs = []uint64{}
		}

		items = append(items, NarrativeListItem{
			ID:          n.ID,
			Title:       n.Title,
			Summary:     n.Summary,
			Status:      n.Status,
			Period:      n.Period,
			PeriodDate:  n.PeriodDate.Format("2006-01-02"),
			Generation:  n.Generation,
			ParentIDs:   parentIDs,
			RelatedTags: tagBriefs,
			ChildIDs:    childIDs,
		})
	}

	return items
}
