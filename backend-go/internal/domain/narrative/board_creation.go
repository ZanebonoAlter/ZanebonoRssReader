package narrative

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/airouter"
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/jsonutil"
	"my-robot-backend/internal/platform/logging"
)

func createBoardFromAbstractTree(tree AbstractTreeNode, date time.Time, categoryID uint) (*models.NarrativeBoard, error) {
	eventTagIDs := collectBoardEventTagIDs(tree)
	if len(eventTagIDs) == 0 {
		return nil, nil
	}

	abstractTagID := tree.ID
	prevBoardIDs := matchPreviousBoard(abstractTagID, date, categoryID)

	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	eventIDsJSON, _ := json.Marshal(eventTagIDs)
	abstractIDsJSON, _ := json.Marshal([]uint{abstractTagID})
	prevIDsJSON, _ := json.Marshal(prevBoardIDs)

	board := &models.NarrativeBoard{
		PeriodDate:      startOfDay,
		Name:            tree.Label,
		Description:     tree.Description,
		ScopeType:       models.NarrativeScopeTypeFeedCategory,
		ScopeCategoryID: &categoryID,
		EventTagIDs:     string(eventIDsJSON),
		AbstractTagIDs:  string(abstractIDsJSON),
		PrevBoardIDs:    string(prevIDsJSON),
		AbstractTagID:   &abstractTagID,
		IsSystem:        true,
	}

	if err := database.DB.Create(board).Error; err != nil {
		return nil, fmt.Errorf("save board from abstract tree %d: %w", tree.ID, err)
	}

	logging.Infof("board-creation: created board %d (%s) from abstract tree %d with %d event tags",
		board.ID, board.Name, tree.ID, len(eventTagIDs))
	return board, nil
}

func collectBoardEventTagIDs(tree AbstractTreeNode) []uint {
	var eventIDs []uint
	collectEventIDs(tree, &eventIDs)
	return eventIDs
}

func collectEventIDs(node AbstractTreeNode, ids *[]uint) {
	if node.Category == "event" {
		*ids = append(*ids, node.ID)
	}
	for _, child := range node.Children {
		collectEventIDs(child, ids)
	}
}

func matchPreviousBoard(abstractTagID uint, date time.Time, categoryID uint) []uint {
	yesterday := date.AddDate(0, 0, -1)
	startOfYesterday := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location())
	endOfYesterday := startOfYesterday.Add(24 * time.Hour)

	var boards []models.NarrativeBoard
	database.DB.Where("abstract_tag_id = ? AND period_date >= ? AND period_date < ? AND scope_category_id = ?",
		abstractTagID, startOfYesterday, endOfYesterday, categoryID).
		Find(&boards)

	if len(boards) == 0 {
		return nil
	}

	var ids []uint
	for _, b := range boards {
		ids = append(ids, b.ID)
	}
	return ids
}

func createMiscBoardsFromEvents(events []TagInput, date time.Time, categoryID uint) ([]models.NarrativeBoard, error) {
	if len(events) == 0 {
		return nil, nil
	}

	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	if len(events) <= 3 {
		eventIDs := make([]uint, 0, len(events))
		for _, e := range events {
			eventIDs = append(eventIDs, e.ID)
		}

		prevBoardIDs := matchMiscPreviousBoard("其他动态", date, categoryID)

		eventIDsJSON, _ := json.Marshal(eventIDs)
		abstractIDsJSON, _ := json.Marshal([]uint{})
		prevIDsJSON, _ := json.Marshal(prevBoardIDs)

		board := models.NarrativeBoard{
			PeriodDate:      startOfDay,
			Name:            "其他动态",
			Description:     "当日未分类事件汇总",
			ScopeType:       models.NarrativeScopeTypeFeedCategory,
			ScopeCategoryID: &categoryID,
			EventTagIDs:     string(eventIDsJSON),
			AbstractTagIDs:  string(abstractIDsJSON),
			PrevBoardIDs:    string(prevIDsJSON),
		}

		if err := database.DB.Create(&board).Error; err != nil {
			return nil, fmt.Errorf("save misc board: %w", err)
		}

		logging.Infof("board-creation: created single misc board %d (%s) with %d events",
			board.ID, board.Name, len(events))
		return []models.NarrativeBoard{board}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	partitions, err := partitionMiscEvents(ctx, events, date, categoryID)
	if err != nil {
		logging.Warnf("board-creation: partitionMiscEvents failed: %v, falling back to single board", err)
		fallback := make([]TagInput, 0, 3)
		for i := range events {
			fallback = append(fallback, events[i])
			if len(fallback) >= 3 {
				break
			}
		}
		return createMiscBoardsFromEvents(fallback, date, categoryID)
	}

	if len(partitions) == 0 {
		return nil, nil
	}

	var boards []models.NarrativeBoard
	for _, p := range partitions {
		eventIDsJSON, _ := json.Marshal(p.EventTagIDs)
		abstractIDsJSON, _ := json.Marshal([]uint{})
		prevIDsJSON, _ := json.Marshal(p.PrevBoardIDs)

		board := models.NarrativeBoard{
			PeriodDate:      startOfDay,
			Name:            p.Name,
			Description:     p.Description,
			ScopeType:       models.NarrativeScopeTypeFeedCategory,
			ScopeCategoryID: &categoryID,
			EventTagIDs:     string(eventIDsJSON),
			AbstractTagIDs:  string(abstractIDsJSON),
			PrevBoardIDs:    string(prevIDsJSON),
		}
		boards = append(boards, board)
	}

	if len(boards) > 0 {
		if err := database.DB.CreateInBatches(boards, 20).Error; err != nil {
			return nil, fmt.Errorf("save misc boards: %w", err)
		}
	}

	return boards, nil
}

func partitionMiscEvents(ctx context.Context, events []TagInput, date time.Time, categoryID uint) ([]BoardPartition, error) {
	if len(events) <= 3 {
		return nil, fmt.Errorf("partitionMiscEvents requires >3 events, got %d", len(events))
	}

	prevBoards, err := CollectPreviousDayBoards(date, models.NarrativeScopeTypeFeedCategory, &categoryID)
	if err != nil {
		logging.Warnf("board-creation: failed to collect previous day boards for misc partition: %v", err)
	}

	validEventIDs := make(map[uint]bool, len(events))
	for _, e := range events {
		validEventIDs[e.ID] = true
	}

	prevBoardIDSet := make(map[uint]bool)
	for _, b := range prevBoards {
		prevBoardIDSet[b.ID] = true
	}

	prompt := buildMiscPartitionPrompt(events, prevBoards)

	temperature := 0.4
	maxTokens := 6000
	result, err := airouter.NewRouter().Chat(ctx, airouter.ChatRequest{
		Capability: airouter.CapabilityTopicTagging,
		Messages: []airouter.Message{
			{Role: "system", Content: miscPartitionSystemPrompt},
			{Role: "user", Content: prompt},
		},
		Temperature: &temperature,
		MaxTokens:   &maxTokens,
		JSONMode:    true,
		JSONSchema: &airouter.JSONSchema{
			Type: "object",
			Properties: map[string]airouter.SchemaProperty{
				"boards": {
					Type: "array",
					Items: &airouter.SchemaProperty{
						Type: "object",
						Properties: map[string]airouter.SchemaProperty{
							"name":           {Type: "string", Description: "看板名称，不超过20字"},
							"description":    {Type: "string", Description: "看板描述，50-100字"},
							"event_tag_ids":  {Type: "array", Items: &airouter.SchemaProperty{Type: "integer"}, Description: "分配到此看板的事件标签ID"},
							"prev_board_ids": {Type: "array", Items: &airouter.SchemaProperty{Type: "integer"}, Description: "延续的昨日看板ID"},
						},
						Required: []string{"name", "description", "event_tag_ids", "prev_board_ids"},
					},
				},
			},
			Required: []string{"boards"},
		},
		Metadata: map[string]any{
			"operation":   "misc_event_partition",
			"category_id": categoryID,
			"event_count": len(events),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("misc partition AI call failed: %w", err)
	}

	logging.Infof("board-creation: misc partition response length=%d, first_500=%s", len(result.Content), truncateStr(result.Content, 500))

	content := jsonutil.SanitizeLLMJSON(result.Content)
	var raw struct {
		Boards []BoardPartition `json:"boards"`
	}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("parse misc partition response: %w", err)
	}

	var valid []BoardPartition
	assigned := make(map[uint]bool)
	for _, b := range raw.Boards {
		var filteredEventIDs []uint
		for _, id := range b.EventTagIDs {
			if !validEventIDs[id] || assigned[id] {
				continue
			}
			filteredEventIDs = append(filteredEventIDs, id)
			assigned[id] = true
		}
		if len(filteredEventIDs) == 0 {
			continue
		}

		var filteredPrevIDs []uint
		for _, id := range b.PrevBoardIDs {
			if prevBoardIDSet[id] {
				filteredPrevIDs = append(filteredPrevIDs, id)
			}
		}
		if filteredPrevIDs == nil {
			filteredPrevIDs = []uint{}
		}

		valid = append(valid, BoardPartition{
			Name:          b.Name,
			Description:   b.Description,
			EventTagIDs:   filteredEventIDs,
			AbstractTagIDs: []uint{},
			PrevBoardIDs:  filteredPrevIDs,
		})
	}

	return valid, nil
}

const miscPartitionSystemPrompt = `你是一名专业的新闻叙事看板分析师。你的任务是将未归入任何抽象树的独立事件标签分组到叙事看板中。

每个看板代表一个独立的故事线或主题领域。规则：
1. 每个事件标签只能出现在一个看板中（不可重复分配）
2. 为每个看板提供简洁的名称（中文，不超过20字）和描述（中文，50-100字）
3. 如果某个事件标签与前一天的某个看板延续，标注 prev_board_ids
4. 每个看板必须至少包含一个事件标签
5. 不要为了凑数而强行合并不相关的事件
6. 数量不固定，有几条写几条，没有就返回空数组

输出要求：
1. 顶层必须是 JSON 对象，且只能包含一个字段：boards
2. boards 必须是 JSON 数组；没有符合条件的看板时，返回 {"boards":[]}
3. boards 数组中的每个元素都必须包含 name、description、event_tag_ids、prev_board_ids 字段
4. event_tag_ids 和 prev_board_ids 必须始终输出数组，即使为空也要输出 []
5. 只返回一个合法 JSON 对象，不要输出 Markdown 代码块、解释文字、前后缀，禁止输出第二个 JSON 块`

func buildMiscPartitionPrompt(events []TagInput, prevBoards []PreviousBoardBrief) string {
	var buf strings.Builder
	buf.WriteString("## 今日未分类事件标签\n\n")
	for _, t := range events {
		entry := fmt.Sprintf("- [EventTagID:%d] %s (文章数:%d", t.ID, t.Label, t.ArticleCount)
		if t.Description != "" {
			entry += fmt.Sprintf(", 描述:%s", t.Description)
		}
		entry += ")\n"
		buf.WriteString(entry)
	}

	if len(prevBoards) > 0 {
		buf.WriteString("\n## 昨日看板（供延续参考）\n\n")
		for _, b := range prevBoards {
			buf.WriteString(fmt.Sprintf("- [PrevBoardID:%d] %s: %s\n", b.ID, b.Name, b.Description))
		}
	}

	buf.WriteString("\n请将以上事件标签分组到叙事看板中。确保每个事件标签只出现在一个看板中。\n")
	return buf.String()
}

func matchMiscPreviousBoard(name string, date time.Time, categoryID uint) []uint {
	yesterday := date.AddDate(0, 0, -1)
	startOfYesterday := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location())
	endOfYesterday := startOfYesterday.Add(24 * time.Hour)

	var boards []models.NarrativeBoard
	database.DB.Where("name = ? AND period_date >= ? AND period_date < ? AND scope_category_id = ? AND abstract_tag_id IS NULL",
		name, startOfYesterday, endOfYesterday, categoryID).
		Find(&boards)

	if len(boards) == 0 {
		return nil
	}

	var ids []uint
	for _, b := range boards {
		ids = append(ids, b.ID)
	}
	return ids
}
