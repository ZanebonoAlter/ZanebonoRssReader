package topicextraction

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"my-robot-backend/internal/domain/topictypes"
	"regexp"
	"sort"
	"strings"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/domain/topicanalysis"
	"my-robot-backend/internal/platform/airouter"
	"my-robot-backend/internal/platform/database"
)

var errNoAIAvailable = errors.New("no AI provider available for tagging")

// TagExtractor handles extracting and resolving tags from AI summaries
type TagExtractor struct {
	embeddingService *topicanalysis.EmbeddingService
	router           *airouter.Router
}

// NewTagExtractor creates a new tag extractor
func NewTagExtractor() *TagExtractor {
	return &TagExtractor{
		embeddingService: topicanalysis.NewEmbeddingService(),
		router:           airouter.NewRouter(),
	}
}

// ExtractionResult represents the result of tag extraction
type ExtractionResult struct {
	Tags    []topictypes.TopicTag
	Skipped []string // Tags that were skipped due to low confidence
	Errors  []string
	Source  string // "llm" or "heuristic"
}

// ExtractTags extracts tags from a summary using two-stage process:
// 1. AI extracts candidate tags with categories
// 2. For ambiguous candidates, AI decides whether to reuse or create
func (te *TagExtractor) ExtractTags(ctx context.Context, input topictypes.ExtractionInput) (*ExtractionResult, error) {
	// Step 1: Extract candidate tags
	candidates, err := te.extractCandidates(ctx, input)
	if err != nil {
		// Fall back to heuristic extraction
		return te.extractWithHeuristic(input, err)
	}

	if len(candidates) == 0 {
		return te.extractWithHeuristic(input, errors.New("no candidates extracted"))
	}

	// Step 2: Resolve each candidate against existing tags
	tags := make([]topictypes.TopicTag, 0, len(candidates))
	var skipped []string
	var errs []string

	for _, candidate := range candidates {
		tag, skip, err := te.resolveCandidate(ctx, candidate, input)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", candidate.Label, err))
			continue
		}
		if skip {
			skipped = append(skipped, candidate.Label)
			continue
		}
		tags = append(tags, *tag)
	}

	return &ExtractionResult{
		Tags:    tags,
		Skipped: skipped,
		Errors:  errs,
		Source:  "llm",
	}, nil
}

// extractCandidates extracts candidate tags from the summary
func (te *TagExtractor) extractCandidates(ctx context.Context, input topictypes.ExtractionInput) ([]topictypes.ExtractedTag, error) {
	systemPrompt := buildExtractionSystemPrompt()
	userPrompt := buildExtractionUserPrompt(input)

	maxTokens := 600
	temperature := 0.2
	metadata := map[string]any{
		"title": input.Title,
	}
	if input.FeedName != "" {
		metadata["feed_name"] = input.FeedName
	}
	if input.ArticleID != nil {
		metadata["article_id"] = *input.ArticleID
	}
	if input.SummaryID != nil {
		metadata["summary_id"] = *input.SummaryID
	}

	result, err := te.router.Chat(ctx, airouter.ChatRequest{
		Capability: airouter.CapabilityTopicTagging,
		Messages: []airouter.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: &temperature,
		MaxTokens:   &maxTokens,
		Metadata:    metadata,
		JSONMode:    true,
		JSONSchema:  tagExtractionSchema(),
	})
	if err != nil {
		return nil, fmt.Errorf("AI extraction failed: %w", err)
	}

	return parseExtractedTags(result.Content)
}

// resolveCandidate resolves a single candidate tag against existing tags
func (te *TagExtractor) resolveCandidate(ctx context.Context, candidate topictypes.ExtractedTag, input topictypes.ExtractionInput) (*topictypes.TopicTag, bool, error) {
	// Validate category
	category := validateCategory(candidate.Category)

	// Step 1: Check for exact slug match
	slug := slugify(candidate.Label)
	var existingTag models.TopicTag
	err := database.DB.Where("slug = ? AND category = ?", slug, category).First(&existingTag).Error
	if err == nil {
		// Exact match - reuse with updated score
		return &topictypes.TopicTag{
			Label:     existingTag.Label,
			Slug:      existingTag.Slug,
			Category:  existingTag.Category,
			Icon:      existingTag.Icon,
			Aliases:   parseAliases(existingTag.Aliases),
			Score:     candidate.Confidence,
			IsNew:     false,
			MatchedTo: existingTag.ID,
		}, false, nil
	}

	// Step 2: Check for alias match
	var aliasMatch models.TopicTag
	aliasMatchErr := database.DB.Where("category = ? AND ? IN (SELECT jsonb_array_elements_text(COALESCE(NULLIF(aliases, ''), '[]')::jsonb))", category, candidate.Label).First(&aliasMatch).Error
	if aliasMatchErr == nil {
		return &topictypes.TopicTag{
			Label:     aliasMatch.Label,
			Slug:      aliasMatch.Slug,
			Category:  aliasMatch.Category,
			Icon:      aliasMatch.Icon,
			Aliases:   parseAliases(aliasMatch.Aliases),
			Score:     candidate.Confidence,
			IsNew:     false,
			MatchedTo: aliasMatch.ID,
		}, false, nil
	}

	// Step 3: Vector similarity matching (if embedding service available)
	matchResult, err := te.embeddingService.TagMatch(ctx, candidate.Label, category, formatAliases(candidate.Aliases))
	if err == nil {
		switch matchResult.MatchType {
		case "exact":
			// Already handled above, but just in case
			return &topictypes.TopicTag{
				Label:     matchResult.ExistingTag.Label,
				Slug:      matchResult.ExistingTag.Slug,
				Category:  matchResult.ExistingTag.Category,
				Icon:      matchResult.ExistingTag.Icon,
				Aliases:   parseAliases(matchResult.ExistingTag.Aliases),
				Score:     candidate.Confidence,
				IsNew:     false,
				MatchedTo: matchResult.ExistingTag.ID,
			}, false, nil

		case "high_similarity":
			// Auto-reuse
			return &topictypes.TopicTag{
				Label:     matchResult.ExistingTag.Label,
				Slug:      matchResult.ExistingTag.Slug,
				Category:  matchResult.ExistingTag.Category,
				Icon:      matchResult.ExistingTag.Icon,
				Aliases:   parseAliases(matchResult.ExistingTag.Aliases),
				Score:     candidate.Confidence * matchResult.Similarity,
				IsNew:     false,
				MatchedTo: matchResult.ExistingTag.ID,
			}, false, nil

		case "low_similarity":
			// Auto-create new tag
			return &topictypes.TopicTag{
				Label:    candidate.Label,
				Slug:     slug,
				Category: category,
				Aliases:  candidate.Aliases,
				Score:    candidate.Confidence,
				IsNew:    true,
			}, false, nil

		case "ai_judgment":
			// Need AI to decide
			decision, err := te.aiJudgment(ctx, candidate, matchResult.Candidates, input)
			if err != nil {
				// On AI failure, default to creating new
				return &topictypes.TopicTag{
					Label:    candidate.Label,
					Slug:     slug,
					Category: category,
					Aliases:  candidate.Aliases,
					Score:    candidate.Confidence,
					IsNew:    true,
				}, false, nil
			}

			if decision.Decision == "reuse" && decision.ReuseTagID > 0 {
				// Find the tag to reuse
				for _, c := range matchResult.Candidates {
					if c.Tag.ID == decision.ReuseTagID {
						return &topictypes.TopicTag{
							Label:     c.Tag.Label,
							Slug:      c.Tag.Slug,
							Category:  c.Tag.Category,
							Icon:      c.Tag.Icon,
							Aliases:   parseAliases(c.Tag.Aliases),
							Score:     candidate.Confidence * c.Similarity,
							IsNew:     false,
							MatchedTo: c.Tag.ID,
						}, false, nil
					}
				}
			}

			// Create new tag
			label := candidate.Label
			if decision.NewLabel != "" {
				label = decision.NewLabel
			}
			cat := category
			if decision.NewCategory != "" {
				cat = validateCategory(decision.NewCategory)
			}
			return &topictypes.TopicTag{
				Label:    label,
				Slug:     slugify(label),
				Category: cat,
				Aliases:  candidate.Aliases,
				Score:    candidate.Confidence,
				IsNew:    true,
			}, false, nil
		}
	}

	// No embedding service or matching - create new tag
	return &topictypes.TopicTag{
		Label:    candidate.Label,
		Slug:     slug,
		Category: category,
		Aliases:  candidate.Aliases,
		Score:    candidate.Confidence,
		IsNew:    true,
	}, false, nil
}

// aiJudgment asks AI to decide on ambiguous tag matches
func (te *TagExtractor) aiJudgment(ctx context.Context, candidate topictypes.ExtractedTag, similarTags []topicanalysis.TagCandidate, input topictypes.ExtractionInput) (*topictypes.TagResolutionResponse, error) {
	// Build context for AI decision
	var similarInfo []topictypes.SimilarTagInfo
	for _, t := range similarTags {
		similarInfo = append(similarInfo, topictypes.SimilarTagInfo{
			ID:         t.Tag.ID,
			Label:      t.Tag.Label,
			Category:   t.Tag.Category,
			Aliases:    parseAliases(t.Tag.Aliases),
			Similarity: t.Similarity,
		})
	}

	req := topictypes.TagResolutionRequest{
		CandidateTag:   candidate,
		SimilarTags:    similarInfo,
		SummaryContext: fmt.Sprintf("标题: %s\n来源: %s", input.Title, input.FeedName),
	}

	systemPrompt := buildResolutionSystemPrompt()
	userPrompt := buildResolutionUserPrompt(req)

	maxTokens := 200
	temperature := 0.1
	metadata := map[string]any{
		"candidate": candidate.Label,
	}
	if input.FeedName != "" {
		metadata["feed_name"] = input.FeedName
	}
	if input.ArticleID != nil {
		metadata["article_id"] = *input.ArticleID
	}
	if input.SummaryID != nil {
		metadata["summary_id"] = *input.SummaryID
	}

	result, err := te.router.Chat(ctx, airouter.ChatRequest{
		Capability: airouter.CapabilityTopicTagging,
		Messages: []airouter.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: &temperature,
		MaxTokens:   &maxTokens,
		Metadata:    metadata,
		JSONMode:    true,
		JSONSchema:  tagResolutionSchema(),
	})
	if err != nil {
		return nil, fmt.Errorf("AI judgment failed: %w", err)
	}

	return parseResolutionResponse(result.Content)
}

// extractWithHeuristic falls back to rule-based extraction
func (te *TagExtractor) extractWithHeuristic(input topictypes.ExtractionInput, originalErr error) (*ExtractionResult, error) {
	tags := ExtractTopics(input)
	result := make([]topictypes.TopicTag, len(tags))
	for i, t := range tags {
		// Map old 'kind' to new 'category'
		category := "keyword"
		if t.Kind == "entity" {
			// Entities default to keyword category (organizations, products go here)
			// Future: could add heuristics to detect person/event
			category = "keyword"
		}
		result[i] = topictypes.TopicTag{
			Label:    t.Label,
			Slug:     t.Slug,
			Category: category,
			Score:    t.Score,
			IsNew:    true,
		}
	}
	return &ExtractionResult{
		Tags:   result,
		Source: "heuristic",
		Errors: []string{originalErr.Error()},
	}, nil
}

// Helper functions

func buildExtractionSystemPrompt() string {
	return `你是一个专业的新闻分析助手，负责从新闻摘要中提取结构化标签。

标签分为三类：
1. event（事件）：完整描述的新闻事件名词短语，必须具备语义完整性
   - 正确示例："苹果WWDC 2024发布会"、"央行禁止比特币交易"、"某景区门票涨价风波"
   - 错误示例："3月30"（裸日期）、"禁止交易"（无主体动作）、"门票涨价"（无归属状态）、"北京中关村"（裸地名）、"AI体验活动"（泛化活动名）
   
2. person（人物）：具体的个人姓名
   - 正确示例："Sam Altman"、"Elon Musk"
   - 错误示例："CEO"（泛称）、"发言人"（角色而非具体人）

3. keyword（关键词）：其他所有概念，包括组织、产品、技术、地点、主题、泛化活动等
   - 示例："苹果公司"、"ChatGPT"、"中关村"、"人工智能"、"AI体验活动"

提取规则：
- 提取3-8个标签
- event类标签必须是语义完整的名词短语，能独立传达事件内容
- 拒绝语义片段：不要把裸日期、无主体动作、无归属状态、裸地名、泛化活动名当作event，应归入keyword
- 无法判断语义完整性时，优先归入keyword类别
- 标签应该简洁、准确

每个标签输出格式：
{"label": "标签名称", "category": "event|person|keyword", "confidence": 0.0-1.0, "aliases": ["别名1"], "evidence": "提取依据"}`
}

func buildExtractionUserPrompt(input topictypes.ExtractionInput) string {
	return fmt.Sprintf(`请从以下新闻摘要中提取标签：

标题: %s
来源: %s
分类: %s

摘要内容:
%s

请返回JSON数组格式的标签列表。`, input.Title, input.FeedName, input.CategoryName, input.Summary)
}

func buildResolutionSystemPrompt() string {
	return `你是一个标签匹配助手，负责决定新提出的标签应该复用已有标签还是创建新标签。

决策标准：
1. 如果新标签与已有标签含义完全相同，复用已有标签
2. 如果新标签是已有标签的别名，复用已有标签
3. 如果标签含义明显不同，创建新标签
4. 如果标签存在细微差异（如版本号、地区），根据上下文判断

返回JSON格式：
{"decision": "reuse|create_new", "reuse_tag_id": 123, "reason": "决策理由", "new_label": "调整后的标签名", "new_category": "event|person|keyword"}`
}

func buildResolutionUserPrompt(req topictypes.TagResolutionRequest) string {
	var similarStr string
	for _, t := range req.SimilarTags {
		similarStr += fmt.Sprintf("- ID %d: \"%s\" (类别: %s, 相似度: %.2f)\n", t.ID, t.Label, t.Category, t.Similarity)
	}

	return fmt.Sprintf(`新标签候选：
- 名称: \"%s\"
- 类别: %s
- 置信度: %.2f

相似已有标签：
%s

上下文：
%s

请判断是否复用已有标签或创建新标签。`, req.CandidateTag.Label, req.CandidateTag.Category, req.CandidateTag.Confidence, similarStr, req.SummaryContext)
}

func parseExtractedTags(content string) ([]topictypes.ExtractedTag, error) {
	content = normalizeStructuredResponse(content)

	var raw []struct {
		Label      string   `json:"label"`
		Category   string   `json:"category"`
		Confidence float64  `json:"confidence"`
		Aliases    []string `json:"aliases"`
		Evidence   string   `json:"evidence"`
	}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		var wrapped struct {
			Tags []struct {
				Label      string   `json:"label"`
				Category   string   `json:"category"`
				Confidence float64  `json:"confidence"`
				Aliases    []string `json:"aliases"`
				Evidence   string   `json:"evidence"`
			} `json:"tags"`
		}
		if wrappedErr := json.Unmarshal([]byte(content), &wrapped); wrappedErr != nil {
			return nil, fmt.Errorf("failed to parse tags: %w", err)
		}
		raw = wrapped.Tags
	}

	result := make([]topictypes.ExtractedTag, 0, len(raw))
	for _, t := range raw {
		if strings.TrimSpace(t.Label) == "" {
			continue
		}
		cat := validateCategory(t.Category)
		conf := t.Confidence
		if conf <= 0 {
			conf = 0.7
		}
		result = append(result, topictypes.ExtractedTag{
			Label:      strings.TrimSpace(t.Label),
			Category:   cat,
			Confidence: conf,
			Aliases:    t.Aliases,
			Evidence:   t.Evidence,
		})
	}

	return result, nil
}

func normalizeStructuredResponse(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	if strings.HasPrefix(content, "[") || strings.HasPrefix(content, "{") {
		return content
	}

	arrayStart := strings.Index(content, "[")
	arrayEnd := strings.LastIndex(content, "]")
	if arrayStart >= 0 && arrayEnd > arrayStart {
		return strings.TrimSpace(content[arrayStart : arrayEnd+1])
	}

	objectStart := strings.Index(content, "{")
	objectEnd := strings.LastIndex(content, "}")
	if objectStart >= 0 && objectEnd > objectStart {
		return strings.TrimSpace(content[objectStart : objectEnd+1])
	}

	return content
}

func parseResolutionResponse(content string) (*topictypes.TagResolutionResponse, error) {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var resp struct {
		Decision    string `json:"decision"`
		ReuseTagID  uint   `json:"reuse_tag_id"`
		Reason      string `json:"reason"`
		NewLabel    string `json:"new_label"`
		NewCategory string `json:"new_category"`
	}
	if err := json.Unmarshal([]byte(content), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse resolution: %w", err)
	}

	return &topictypes.TagResolutionResponse{
		Decision:    resp.Decision,
		ReuseTagID:  resp.ReuseTagID,
		Reason:      resp.Reason,
		NewLabel:    resp.NewLabel,
		NewCategory: resp.NewCategory,
	}, nil
}

func validateCategory(cat string) string {
	cat = strings.ToLower(strings.TrimSpace(cat))
	switch cat {
	case "event", "person", "keyword":
		return cat
	default:
		return "keyword"
	}
}

func parseAliases(aliases string) []string {
	if aliases == "" {
		return nil
	}
	var result []string
	if err := json.Unmarshal([]byte(aliases), &result); err != nil {
		// Legacy: comma-separated
		for _, a := range strings.Split(aliases, ",") {
			if s := strings.TrimSpace(a); s != "" {
				result = append(result, s)
			}
		}
	}
	return result
}

func formatAliases(aliases []string) string {
	if len(aliases) == 0 {
		return ""
	}
	data, _ := json.Marshal(aliases)
	return string(data)
}

// slugify creates a URL-safe slug from a string
func slugifyWithPunc(value string) string {
	// Lowercase
	slug := strings.ToLower(value)

	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove or replace special characters
	reg := regexp.MustCompile(`[^a-z0-9\u4e00-\u9fff\-]`)
	slug = reg.ReplaceAllString(slug, "")

	return slug
}

// dedupeTags removes duplicate tags based on slug and category
func dedupeTags(tags []topictypes.TopicTag) []topictypes.TopicTag {
	best := make(map[string]topictypes.TopicTag)
	for _, t := range tags {
		key := t.Category + ":" + t.Slug
		current, exists := best[key]
		if !exists || current.Score < t.Score {
			best[key] = t
		}
	}

	result := make([]topictypes.TopicTag, 0, len(best))
	for _, t := range best {
		result = append(result, t)
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Score == result[j].Score {
			return result[i].Label < result[j].Label
		}
		return result[i].Score > result[j].Score
	})

	return result
}

func tagExtractionSchema() *airouter.JSONSchema {
	return &airouter.JSONSchema{
		Type: "array",
		Items: &airouter.JSONSchema{
			Type: "object",
			Properties: map[string]airouter.SchemaProperty{
				"label":      {Type: "string", Description: "标签名称"},
				"category":   {Type: "string", Description: "event, person 或 keyword"},
				"confidence": {Type: "number", Description: "置信度 0.0-1.0"},
				"aliases":    {Type: "array", Items: &airouter.SchemaProperty{Type: "string"}},
				"evidence":   {Type: "string", Description: "提取依据"},
			},
			Required: []string{"label", "category"},
		},
	}
}

func tagResolutionSchema() *airouter.JSONSchema {
	return &airouter.JSONSchema{
		Type: "object",
		Properties: map[string]airouter.SchemaProperty{
			"decision":     {Type: "string", Description: "reuse 或 create_new"},
			"reuse_tag_id": {Type: "integer", Description: "复用的标签ID"},
			"reason":       {Type: "string", Description: "决策理由"},
			"new_label":    {Type: "string", Description: "调整后的标签名"},
			"new_category": {Type: "string", Description: "event, person 或 keyword"},
		},
		Required: []string{"decision", "reason"},
	}
}
