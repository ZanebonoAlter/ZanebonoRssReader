package topicgraph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/airouter"
	"my-robot-backend/internal/platform/database"
)

var errNoAIAvailable = errors.New("no AI provider available for tagging")

// TagExtractor handles extracting and resolving tags from AI summaries
type TagExtractor struct {
	embeddingService *EmbeddingService
	router           *airouter.Router
}

// NewTagExtractor creates a new tag extractor
func NewTagExtractor() *TagExtractor {
	return &TagExtractor{
		embeddingService: NewEmbeddingService(),
		router:           airouter.NewRouter(),
	}
}

// ExtractionResult represents the result of tag extraction
type ExtractionResult struct {
	Tags    []TopicTag
	Skipped []string // Tags that were skipped due to low confidence
	Errors  []string
	Source  string // "llm" or "heuristic"
}

// ExtractTags extracts tags from a summary using two-stage process:
// 1. AI extracts candidate tags with categories
// 2. For ambiguous candidates, AI decides whether to reuse or create
func (te *TagExtractor) ExtractTags(ctx context.Context, input ExtractionInput) (*ExtractionResult, error) {
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
	tags := make([]TopicTag, 0, len(candidates))
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
func (te *TagExtractor) extractCandidates(ctx context.Context, input ExtractionInput) ([]ExtractedTag, error) {
	systemPrompt := buildExtractionSystemPrompt()
	userPrompt := buildExtractionUserPrompt(input)

	maxTokens := 600
	temperature := 0.2

	result, err := te.router.Chat(ctx, airouter.ChatRequest{
		Capability: airouter.CapabilityTopicTagging,
		Messages: []airouter.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: &temperature,
		MaxTokens:   &maxTokens,
		Metadata: map[string]any{
			"title": input.Title,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("AI extraction failed: %w", err)
	}

	return parseExtractedTags(result.Content)
}

// resolveCandidate resolves a single candidate tag against existing tags
func (te *TagExtractor) resolveCandidate(ctx context.Context, candidate ExtractedTag, input ExtractionInput) (*TopicTag, bool, error) {
	// Validate category
	category := validateCategory(candidate.Category)

	// Step 1: Check for exact slug match
	slug := slugify(candidate.Label)
	var existingTag models.TopicTag
	err := database.DB.Where("slug = ? AND category = ?", slug, category).First(&existingTag).Error
	if err == nil {
		// Exact match - reuse with updated score
		return &TopicTag{
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
	aliasMatchErr := database.DB.Where("category = ? AND ? IN (SELECT value FROM json_each(aliases))", category, candidate.Label).First(&aliasMatch).Error
	if aliasMatchErr == nil {
		return &TopicTag{
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
			return &TopicTag{
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
			return &TopicTag{
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
			return &TopicTag{
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
				return &TopicTag{
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
						return &TopicTag{
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
			return &TopicTag{
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
	return &TopicTag{
		Label:    candidate.Label,
		Slug:     slug,
		Category: category,
		Aliases:  candidate.Aliases,
		Score:    candidate.Confidence,
		IsNew:    true,
	}, false, nil
}

// aiJudgment asks AI to decide on ambiguous tag matches
func (te *TagExtractor) aiJudgment(ctx context.Context, candidate ExtractedTag, similarTags []TagCandidate, input ExtractionInput) (*TagResolutionResponse, error) {
	// Build context for AI decision
	var similarInfo []SimilarTagInfo
	for _, t := range similarTags {
		similarInfo = append(similarInfo, SimilarTagInfo{
			ID:         t.Tag.ID,
			Label:      t.Tag.Label,
			Category:   t.Tag.Category,
			Aliases:    parseAliases(t.Tag.Aliases),
			Similarity: t.Similarity,
		})
	}

	req := TagResolutionRequest{
		CandidateTag:   candidate,
		SimilarTags:    similarInfo,
		SummaryContext: fmt.Sprintf("标题: %s\n来源: %s", input.Title, input.FeedName),
	}

	systemPrompt := buildResolutionSystemPrompt()
	userPrompt := buildResolutionUserPrompt(req)

	maxTokens := 200
	temperature := 0.1

	result, err := te.router.Chat(ctx, airouter.ChatRequest{
		Capability: airouter.CapabilityTopicTagging,
		Messages: []airouter.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: &temperature,
		MaxTokens:   &maxTokens,
		Metadata: map[string]any{
			"candidate": candidate.Label,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("AI judgment failed: %w", err)
	}

	return parseResolutionResponse(result.Content)
}

// extractWithHeuristic falls back to rule-based extraction
func (te *TagExtractor) extractWithHeuristic(input ExtractionInput, originalErr error) (*ExtractionResult, error) {
	tags := ExtractTopics(input)
	result := make([]TopicTag, len(tags))
	for i, t := range tags {
		// Map old 'kind' to new 'category'
		category := "keyword"
		if t.Kind == "entity" {
			// Entities default to keyword category (organizations, products go here)
			// Future: could add heuristics to detect person/event
			category = "keyword"
		}
		result[i] = TopicTag{
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
1. event（事件）：具有明确时间、地点的新闻事件，如"WWDC 2024"、"GPT-4发布"
2. person（人物）：具体的个人，如"Sam Altman"、"Elon Musk"
3. keyword（关键词）：其他所有概念，包括组织、产品、技术、主题等

每个标签输出格式：
{"label": "标签名称", "category": "event|person|keyword", "confidence": 0.0-1.0, "aliases": ["别名1", "别名2"], "evidence": "提取依据"}

提取规则：
- 提取3-8个标签
- 标签应该简洁、准确
- confidence 表示提取的置信度（0-1）
- aliases 包含标签的常见别名或变体
- 优先提取文章核心主题相关的标签
- 组织名称归入 keyword 类别`
}

func buildExtractionUserPrompt(input ExtractionInput) string {
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

func buildResolutionUserPrompt(req TagResolutionRequest) string {
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

func parseExtractedTags(content string) ([]ExtractedTag, error) {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var raw []struct {
		Label      string   `json:"label"`
		Category   string   `json:"category"`
		Confidence float64  `json:"confidence"`
		Aliases    []string `json:"aliases"`
		Evidence   string   `json:"evidence"`
	}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse tags: %w", err)
	}

	result := make([]ExtractedTag, 0, len(raw))
	for _, t := range raw {
		if strings.TrimSpace(t.Label) == "" {
			continue
		}
		cat := validateCategory(t.Category)
		conf := t.Confidence
		if conf <= 0 {
			conf = 0.7
		}
		result = append(result, ExtractedTag{
			Label:      strings.TrimSpace(t.Label),
			Category:   cat,
			Confidence: conf,
			Aliases:    t.Aliases,
			Evidence:   t.Evidence,
		})
	}

	return result, nil
}

func parseResolutionResponse(content string) (*TagResolutionResponse, error) {
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

	return &TagResolutionResponse{
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
func dedupeTags(tags []TopicTag) []TopicTag {
	best := make(map[string]TopicTag)
	for _, t := range tags {
		key := t.Category + ":" + t.Slug
		current, exists := best[key]
		if !exists || current.Score < t.Score {
			best[key] = t
		}
	}

	result := make([]TopicTag, 0, len(best))
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
