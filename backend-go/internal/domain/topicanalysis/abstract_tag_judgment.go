package topicanalysis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/domain/topictypes"
	"my-robot-backend/internal/platform/airouter"
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/jsonutil"
	"my-robot-backend/internal/platform/logging"

	"gorm.io/gorm"
)

type tagJudgment struct {
	Merge    *tagJudgmentMerge    `json:"merge,omitempty"`
	Abstract *tagJudgmentAbstract `json:"abstract,omitempty"`
}

type tagJudgmentMerge struct {
	Target   string   `json:"target"`
	Label    string   `json:"label"`
	Children []string `json:"children"`
	Reason   string   `json:"reason"`
}

type tagJudgmentAbstract struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Children    []string `json:"children"`
	Reason      string   `json:"reason"`
}

const mergeMinSimilarity = 0.85

func callLLMForTagJudgment(ctx context.Context, candidates []TagCandidate, newLabel string, category string, narrativeContext string, caller string) (*tagJudgment, error) {
	router := airouter.NewRouter()
	hasHighSim := shouldAllowMergeJudgment(candidates)
	prompt := buildTagJudgmentPrompt(candidates, newLabel, category, narrativeContext, hasHighSim)

	req := airouter.ChatRequest{
		Capability: airouter.CapabilityTopicTagging,
		Messages: []airouter.Message{
			{Role: "system", Content: "You are a tag taxonomy assistant. Respond only with valid JSON."},
			{Role: "user", Content: prompt},
		},
		JSONMode:    true,
		JSONSchema:  buildTagJudgmentSchema(),
		Temperature: func() *float64 { f := 0.3; return &f }(),
		Metadata: map[string]any{
			"operation":       "tag_judgment",
			"caller":          caller,
			"candidate_count": len(candidates),
			"new_label":       newLabel,
			"category":        category,
			"candidates":      buildCandidateSummary(candidates),
		},
	}

	result, err := router.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	logging.Infof("Tag judgment LLM response for %q: %s", newLabel, result.Content)

	return parseTagJudgmentResponse(result.Content, candidates)
}

func ensureNewLabelCandidateInAbstractJudgment(judgment *tagJudgment, candidates []TagCandidate, newLabel string) {
	if judgment == nil || judgment.Abstract == nil || len(judgment.Abstract.Children) == 0 {
		return
	}

	label := candidateLabelForNewLabel(candidates, newLabel)
	if label == "" || labelInSlice(judgment.Abstract.Children, label) {
		return
	}

	if judgment.Merge != nil {
		if judgment.Merge.Target == label || labelInSlice(judgment.Merge.Children, label) {
			return
		}
	}

	judgment.Abstract.Children = append(judgment.Abstract.Children, label)
}

func candidateLabelForNewLabel(candidates []TagCandidate, newLabel string) string {
	newSlug := topictypes.Slugify(newLabel)
	if newSlug == "" {
		return ""
	}
	for _, c := range candidates {
		if c.Tag == nil {
			continue
		}
		if c.Tag.Slug == newSlug || topictypes.Slugify(c.Tag.Label) == newSlug {
			return c.Tag.Label
		}
	}
	return ""
}

func labelInSlice(labels []string, target string) bool {
	for _, label := range labels {
		if label == target {
			return true
		}
	}
	return false
}

func selectMergeTarget(candidates []TagCandidate, mergeTarget string, mergeLabel string) *models.TopicTag {
	mergeTargetSlug := topictypes.Slugify(mergeTarget)
	for _, c := range candidates {
		if c.Tag != nil && c.Tag.Slug == mergeTargetSlug {
			return c.Tag
		}
	}

	mergeLabelSlug := topictypes.Slugify(mergeLabel)
	for _, c := range candidates {
		if c.Tag != nil && c.Tag.Slug == mergeLabelSlug {
			return c.Tag
		}
	}

	for _, c := range candidates {
		if c.Tag != nil && c.Tag.Label == mergeTarget {
			return c.Tag
		}
	}

	if mergeTargetSlug != "" || mergeLabelSlug != "" {
		for _, c := range candidates {
			if c.Tag != nil && c.Tag.Source != "abstract" {
				cSlug := topictypes.Slugify(c.Tag.Label)
				if mergeTargetSlug != "" && cSlug == mergeTargetSlug {
					return c.Tag
				}
				if mergeLabelSlug != "" && cSlug == mergeLabelSlug {
					return c.Tag
				}
			}
		}
	}

	logging.Warnf("selectMergeTarget: LLM target %q and label %q do not match any candidate, refusing merge", mergeTarget, mergeLabel)
	return nil
}

func parseTagJudgmentResponse(content string, candidates []TagCandidate) (*tagJudgment, error) {
	content = jsonutil.SanitizeLLMJSON(content)

	var raw struct {
		Merge *struct {
			Target   string   `json:"target"`
			Label    string   `json:"label"`
			Children []string `json:"children"`
			Reason   string   `json:"reason"`
		} `json:"merge"`
		Abstract *struct {
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Children    []string `json:"children"`
			Reason      string   `json:"reason"`
		} `json:"abstract"`
	}

	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse tag judgment response: %w", err)
	}

	candidateLabels := make(map[string]bool, len(candidates))
	for _, c := range candidates {
		if c.Tag != nil {
			candidateLabels[c.Tag.Label] = true
		}
	}

	filterChildren := func(children []string) []string {
		var valid []string
		for _, ch := range children {
			ch = strings.TrimSpace(ch)
			if candidateLabels[ch] {
				valid = append(valid, ch)
			}
		}
		return valid
	}

	judgment := &tagJudgment{}
	usedLabels := make(map[string]bool)

	if raw.Merge != nil && raw.Merge.Target != "" {
		judgment.Merge = &tagJudgmentMerge{
			Target:   strings.TrimSpace(raw.Merge.Target),
			Label:    strings.TrimSpace(raw.Merge.Label),
			Children: filterChildren(raw.Merge.Children),
			Reason:   raw.Merge.Reason,
		}
		if judgment.Merge.Label == "" {
			judgment.Merge.Label = judgment.Merge.Target
		}
		usedLabels[judgment.Merge.Target] = true
		for _, ch := range judgment.Merge.Children {
			usedLabels[ch] = true
		}
	}

	if raw.Abstract != nil && raw.Abstract.Name != "" {
		abstractName := strings.TrimSpace(raw.Abstract.Name)
		if len(abstractName) > maxAbstractNameLen {
			abstractName = abstractName[:maxAbstractNameLen]
		}
		desc := strings.TrimSpace(raw.Abstract.Description)
		if len(desc) > 500 {
			desc = desc[:500]
		}
		var dedupedChildren []string
		for _, ch := range filterChildren(raw.Abstract.Children) {
			if !usedLabels[ch] {
				dedupedChildren = append(dedupedChildren, ch)
				usedLabels[ch] = true
			}
		}
		judgment.Abstract = &tagJudgmentAbstract{
			Name:        abstractName,
			Description: desc,
			Children:    dedupedChildren,
			Reason:      raw.Abstract.Reason,
		}
	}

	return judgment, nil
}

func buildCandidateList(candidates []TagCandidate) string {
	var parts []string
	for _, c := range candidates {
		if c.Tag != nil {
			tagType := "normal"
			if c.Tag.Source == "abstract" {
				tagType = "abstract"
			}
			desc := ""
			if c.Tag.Description != "" {
				runes := []rune(c.Tag.Description)
				if len(runes) > 200 {
					desc = fmt.Sprintf(" (description: %s...)", string(runes[:200]))
				} else {
					desc = fmt.Sprintf(" (description: %s)", c.Tag.Description)
				}
			}
			parts = append(parts, fmt.Sprintf("- %q (similarity: %.2f, type: %s, id=%d)%s", c.Tag.Label, c.Similarity, tagType, c.Tag.ID, desc))
		}
	}
	return strings.Join(parts, "\n")
}

func shouldAllowMergeJudgment(candidates []TagCandidate) bool {
	for _, candidate := range candidates {
		if candidate.Tag != nil && candidate.Similarity >= mergeMinSimilarity {
			return true
		}
	}
	return false
}

func buildTagJudgmentSchema() *airouter.JSONSchema {
	return &airouter.JSONSchema{Type: "object", Properties: map[string]airouter.SchemaProperty{
		"merge": {Type: "object", Description: "合并判断：仅当新标签与候选是完全同一概念时填写。父子、组织产品、生态、平台插件关系禁止 merge，应使用 abstract 或 null。omit if not applicable", Properties: map[string]airouter.SchemaProperty{
			"target":   {Type: "string", Description: "合并目标候选标签名称（必须是相似度≥0.85的候选）"},
			"label":    {Type: "string", Description: "合并后的统一名称"},
			"children": {Type: "array", Description: "与 target 完全同一概念的其他候选标签名称列表，不包括父子或生态相关标签", Items: &airouter.SchemaProperty{Type: "string"}},
			"reason":   {Type: "string", Description: "判断理由"},
		}, Required: []string{"target", "label", "children"}},
		"abstract": {Type: "object", Description: "抽象判断：需要为哪些候选创建抽象父标签。omit if not applicable", Properties: map[string]airouter.SchemaProperty{
			"name":        {Type: "string", Description: "抽象标签名称（1-160字）"},
			"description": {Type: "string", Description: "抽象标签中文客观描述（500字以内）"},
			"children":    {Type: "array", Description: "应作为该抽象标签子标签的候选名称列表", Items: &airouter.SchemaProperty{Type: "string"}},
			"reason":      {Type: "string", Description: "判断理由"},
		}, Required: []string{"name", "description", "children"}},
	}}
}

func buildTagJudgmentPrompt(candidates []TagCandidate, newLabel string, category string, narrativeContext string, hasHighSimCandidate bool) string {
	tagList := buildCandidateList(candidates)

	mergeRules := fmt.Sprintf(`
=== MERGE (very strict — same concept only) ===
- "target" MUST be the label of an EXISTING CANDIDATE TAG (from the list below). NEVER use the new tag's own label "%s" as target.
- "children" are other candidates that are ALSO the same concept (e.g. aliases, translations, different names for the same thing)
- Merge ONLY when the new tag and the target describe the EXACT SAME entity/event/person
- GOOD merges: "Tim Cook" ↔ "蒂姆·库克", "俄乌战争" ↔ "俄罗斯入侵乌克兰"
- NEVER merge parent/child, organization/product, ecosystem, platform/plugin, company/protocol, or brand/project relationships.
- For parent/child or ecosystem relationships, you MUST use abstract or null; merge is always wrong.
- BAD merges (DO NOT): "GPT-5发布" with "DeepSeekV4发布" — different events even if same domain
- BAD merges (DO NOT): "Anthropic" with "Anthropic 协议" — organization/product relationship, not the same concept
- BAD merges (DO NOT): "伊朗核问题" with "美众议院法案" — completely different topics`, newLabel)
	if hasHighSimCandidate {
		mergeRules += fmt.Sprintf(`
- The merge target MUST have similarity >= %.2f. Lower-similarity candidates should go to abstract or be left out.`, mergeMinSimilarity)
	} else {
		mergeRules += fmt.Sprintf(`
- CAUTION: No candidate has similarity >= %.2f. Merge is very unlikely to be correct — only use it if you are absolutely certain the new tag and target are the exact same concept despite low embedding similarity.
- In most cases when no high-similarity candidate exists, abstract or null is the correct answer.`, mergeMinSimilarity)
	}
	mergeShape := `  "merge": { ... },   // ONLY if the new tag IS THE SAME concept as a candidate
  "abstract": { ... } // ONLY if 2+ candidates are genuinely related but distinct`

	rules := fmt.Sprintf(`
You are comparing a NEW tag against existing candidate tags. Decide if they are the same concept, related, or unrelated.

Return ONE JSON object:
{
%s
}

If no real relationship exists, return: {"abstract": null}

%s

=== ABSTRACT (related but distinct concepts — encouraged when appropriate) ===
- Create when the new tag + 1+ candidates share a DIRECT thematic connection
- This is DIFFERENT from merge: the concepts are related but NOT the same — they should be grouped under a broader parent
- "name" must be a concise specific category, NOT a vague label like "新闻"/"技术"/"人物"
- Good: "AI大模型" for GPT-5+Claude+Gemini, "航天产业" for SpaceX+Starlink, "中东局势" for 伊朗核问题+伊美谈判
- Bad: grouping unrelated events that merely happened around the same time
- When 2+ candidates are in the same domain/topic area and similarity >= 0.65, abstract is usually appropriate

=== CRITICAL ===
- A candidate must appear in at most ONE section (merge.children or abstract.children)
- If candidates are clearly unrelated to the new tag AND to each other → return null

Existing candidates:`, mergeShape, mergeRules)

	categoryRules := ""
	switch category {
	case "person":
		categoryRules = `
Person-specific rules:
- merge: ONLY when different names for the SAME person (e.g. "Tim Cook" and "蒂姆·库克")
- abstract: CREATE abstract tags for people who share affiliations, roles, domains, or national relevance
  - Examples: "中国科技领袖" for 雷军+余承东, "AI公司CEO" for Sam Altman+Satya Nadella, "美国政治人物" for Trump+Biden
  - People in the same industry, organization, or country's public sphere are related enough for abstract
  - Do NOT create overly vague tags like "人物" — add thematic specificity (e.g. "中国科技领袖" not "人物")`
	case "event":
		categoryRules = `
Event-specific rules:
- merge: ONLY when clearly the SAME event with different descriptions (same actors, same incident)
- abstract: CREATE abstract tags ONLY when events are causally connected or part of the SAME event chain
  - GOOD abstracts: "2024年大选" for multiple events in the same election, "中东冲突升级" for events in the same conflict
  - BAD abstracts (DO NOT create): grouping unrelated news events that merely happened around the same time
  - CRITICAL: "same time period" alone is NOT sufficient for grouping. Events must share a direct causal link or be clearly part of the same story arc
  - Do NOT group events that merely share a broad category like "international news" or "tech news"
  - If a candidate event involves completely different countries, organizations, or topics → return null`
	default:
		categoryRules = `
Keyword-specific rules:
- merge: ONLY when tags are synonyms, translations, or different names for the EXACT SAME concept
- abstract: ONLY when keywords are directly related in a specific domain
  - Good: "前端框架" for React+Vue+Angular, "编程语言" for Python+Rust+Go
  - Bad: "技术" or "科技" as abstract for unrelated tech keywords
- Do NOT merge keywords that merely share a broad field (e.g. "AI" and "云计算" are different topics)
- When in doubt, return null`
	}

	prompt := fmt.Sprintf(`%s
%s

>>> NEW TAG TO CLASSIFY: "%s" (category: %s)

%s

Respond with JSON now.`, rules, tagList, newLabel, category, categoryRules)

	if narrativeContext != "" {
		prompt += fmt.Sprintf("\n\nAdditional context from narrative analysis:\n%s\nUse this context to help determine relationships.", narrativeContext)
	}

	return prompt
}

func findSimilarExistingAbstract(ctx context.Context, name, desc, category string, candidates []TagCandidate) *models.TopicTag {
	es := NewEmbeddingService()
	thresholds := es.GetThresholds()
	probe := &models.TopicTag{
		Label:       name,
		Description: desc,
		Category:    category,
		Source:      "abstract",
	}
	similar, err := es.FindSimilarTags(ctx, probe, category, 8, EmbeddingTypeSemantic)
	if err != nil {
		logging.Warnf("findSimilarExistingAbstract: embedding search failed: %v", err)
		return nil
	}

	existingAbstracts := make([]models.TopicTag, 0, len(similar))
	for _, candidate := range similar {
		if candidate.Tag == nil || candidate.Tag.Source != "abstract" {
			continue
		}
		if candidate.Similarity < thresholds.LowSimilarity {
			continue
		}
		existingAbstracts = append(existingAbstracts, *candidate.Tag)
		if len(existingAbstracts) == 5 {
			break
		}
	}
	if len(existingAbstracts) == 0 {
		return nil
	}

	candidateLabels := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.Tag != nil {
			candidateLabels = append(candidateLabels, candidate.Tag.Label)
		}
	}

	router := airouter.NewRouter()
	abstractInfo := make([]string, 0, len(existingAbstracts))
	for _, existing := range existingAbstracts {
		children := loadAbstractChildLabels(existing.ID, 5)
		abstractInfo = append(abstractInfo,
			fmt.Sprintf("- ID %d: %q (描述: %s, 子标签: %s)", existing.ID, existing.Label, truncateStr(existing.Description, 100), formatChildLabels(children)))
	}

	prompt := fmt.Sprintf(`一个新的抽象标签即将被创建，请检查以下已有抽象标签中是否有描述同一概念的。

即将创建的抽象标签: %q (描述: %s)
其候选子标签: %s

已有同 category 抽象标签:
%s

规则:
- 只有当已有标签描述的核心概念与新标签完全相同时才返回
- 不要把宽泛程度不同的标签当作同一概念

返回 JSON: {"reuse_id": 0 或 已有标签的ID, "reason": "简要说明"}`,
		name, truncateStr(desc, 200), formatChildLabels(candidateLabels), strings.Join(abstractInfo, "\n"))

	req := airouter.ChatRequest{
		Capability: airouter.CapabilityTopicTagging,
		Messages: []airouter.Message{
			{Role: "system", Content: "You are a tag taxonomy assistant. Respond only with valid JSON."},
			{Role: "user", Content: prompt},
		},
		JSONMode: true,
		JSONSchema: &airouter.JSONSchema{
			Type: "object",
			Properties: map[string]airouter.SchemaProperty{
				"reuse_id": {Type: "integer", Description: "复用的已有标签ID，0表示不匹配"},
				"reason":   {Type: "string", Description: "判断理由"},
			},
			Required: []string{"reuse_id", "reason"},
		},
		Temperature: func() *float64 { f := 0.2; return &f }(),
		Metadata: map[string]any{
			"operation":    "find_similar_existing_abstract",
			"new_name":     name,
			"category":     category,
			"candidates_n": len(candidates),
			"shortlist_n":  len(existingAbstracts),
		},
	}

	result, err := router.Chat(ctx, req)
	if err != nil {
		logging.Warnf("findSimilarExistingAbstract: LLM call failed: %v", err)
		return nil
	}

	var parsed struct {
		ReuseID uint   `json:"reuse_id"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(result.Content), &parsed); err != nil {
		logging.Warnf("findSimilarExistingAbstract: parse failed: %v", err)
		return nil
	}
	if parsed.ReuseID == 0 {
		return nil
	}

	for i := range existingAbstracts {
		if existingAbstracts[i].ID == parsed.ReuseID {
			logging.Infof("findSimilarExistingAbstract: found match %d (%q) for new %q: %s",
				existingAbstracts[i].ID, existingAbstracts[i].Label, name, parsed.Reason)
			return &existingAbstracts[i]
		}
	}

	logging.Warnf("findSimilarExistingAbstract: reuse_id %d not found in shortlist", parsed.ReuseID)
	return nil
}

func aiJudgeNarrowerConcept(ctx context.Context, parentTag *models.TopicTag, candidateTag *models.TopicTag) (bool, error) {
	parentChildren := loadAbstractChildLabels(parentTag.ID, 5)
	candidateChildren := loadAbstractChildLabels(candidateTag.ID, 5)

	router := airouter.NewRouter()
	prompt := fmt.Sprintf(`判断候选标签是否是目标标签的更窄（更具体）概念，应该作为其子标签。

目标标签（潜在的父标签）: %q (描述: %s)
目标标签的子标签: %s

候选标签（潜在的子标签）: %q (描述: %s)
候选标签的子标签: %s

规则:
- 如果候选标签描述的是目标标签范围内的一个具体方面、子集或特定场景，则它是更窄概念
- 如果两者是同一层级或候选更宽泛，返回 false
- 如果候选的子标签与目标的子标签高度重叠，说明是同一概念，返回 false

返回 JSON: {"narrower": true/false, "reason": "简要说明"}`,
		parentTag.Label, truncateStr(parentTag.Description, 200), formatChildLabels(parentChildren),
		candidateTag.Label, truncateStr(candidateTag.Description, 200), formatChildLabels(candidateChildren))

	req := airouter.ChatRequest{
		Capability: airouter.CapabilityTopicTagging,
		Messages: []airouter.Message{
			{Role: "system", Content: "You are a tag taxonomy assistant. Respond only with valid JSON."},
			{Role: "user", Content: prompt},
		},
		JSONMode: true,
		JSONSchema: &airouter.JSONSchema{
			Type: "object",
			Properties: map[string]airouter.SchemaProperty{
				"narrower": {Type: "boolean", Description: "候选标签是否是目标标签的更窄概念"},
				"reason":   {Type: "string", Description: "判断理由"},
			},
			Required: []string{"narrower", "reason"},
		},
		Temperature: func() *float64 { f := 0.2; return &f }(),
		Metadata: map[string]any{
			"operation":     "adopt_narrower_abstract",
			"parent_tag":    parentTag.ID,
			"candidate_tag": candidateTag.ID,
		},
	}

	result, err := router.Chat(ctx, req)
	if err != nil {
		return false, fmt.Errorf("LLM call failed: %w", err)
	}

	var parsed struct {
		Narrower bool   `json:"narrower"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(result.Content), &parsed); err != nil {
		return false, fmt.Errorf("parse LLM response: %w", err)
	}

	return parsed.Narrower, nil
}

func aiJudgeBestParent(ctx context.Context, childTag *models.TopicTag, parents []parentWithInfo) (int, error) {
	var parentDescs []string
	for i, p := range parents {
		children := loadAbstractChildLabels(p.Parent.ID, 5)
		desc := fmt.Sprintf("父标签 %d: %q (描述: %s, 子标签: %s)", i+1, p.Parent.Label, truncateStr(p.Parent.Description, 150), formatChildLabels(children))
		parentDescs = append(parentDescs, desc)
	}

	childDesc := fmt.Sprintf("子标签: %q (描述: %s)", childTag.Label, truncateStr(childTag.Description, 200))

	router := airouter.NewRouter()
	prompt := fmt.Sprintf(`一个标签目前被多个抽象父标签收养，请判断哪个父标签是最合适的归属。

%s

%s

规则:
- 选择最能概括子标签概念的父标签
- 如果多个父标签都合适，选择子标签范围更具体的那个（更紧密的归属）
- 如果父标签之间有层级关系，选择最直接（最窄）的父标签

返回 JSON: {"best_index": 父标签编号(从1开始), "reason": "简要说明"}`,
		childDesc, strings.Join(parentDescs, "\n"))

	req := airouter.ChatRequest{
		Capability: airouter.CapabilityTopicTagging,
		Messages: []airouter.Message{
			{Role: "system", Content: "You are a tag taxonomy assistant. Respond only with valid JSON."},
			{Role: "user", Content: prompt},
		},
		JSONMode: true,
		JSONSchema: &airouter.JSONSchema{
			Type: "object",
			Properties: map[string]airouter.SchemaProperty{
				"best_index": {Type: "integer", Description: "最合适父标签的编号（从1开始）"},
				"reason":     {Type: "string", Description: "选择理由"},
			},
			Required: []string{"best_index", "reason"},
		},
		Temperature: func() *float64 { f := 0.2; return &f }(),
		Metadata: map[string]any{
			"operation": "resolve_multi_parent",
			"child_tag": childTag.ID,
		},
	}

	result, err := router.Chat(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("LLM call failed: %w", err)
	}

	var parsed struct {
		BestIndex int    `json:"best_index"`
		Reason    string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(result.Content), &parsed); err != nil {
		return 0, fmt.Errorf("parse LLM response: %w", err)
	}

	idx := parsed.BestIndex - 1
	if idx < 0 || idx >= len(parents) {
		return 0, fmt.Errorf("LLM returned invalid best_index %d (parents count: %d)", parsed.BestIndex, len(parents))
	}

	return idx, nil
}

func aiJudgeAbstractHierarchy(ctx context.Context, tag1ID, tag2ID uint) (parentID, childID uint, err error) {
	var tag1, tag2 models.TopicTag
	if err := database.DB.First(&tag1, tag1ID).Error; err != nil {
		return 0, 0, fmt.Errorf("load tag %d: %w", tag1ID, err)
	}
	if err := database.DB.First(&tag2, tag2ID).Error; err != nil {
		return 0, 0, fmt.Errorf("load tag %d: %w", tag2ID, err)
	}

	children1 := loadAbstractChildLabels(tag1ID, 8)
	children2 := loadAbstractChildLabels(tag2ID, 8)

	router := airouter.NewRouter()
	prompt := fmt.Sprintf(`Given two abstract topic tags, determine which concept is broader (more general) and which is more specific.

Tag A: %q (description: %s)
Tag A's children: %s

Tag B: %q (description: %s)
Tag B's children: %s

Respond with JSON:
{"parent": "A" or "B", "reason": "brief explanation"}

Rules:
- The parent should be the more general/broader concept
- Use the children tags to understand what each abstract tag actually covers
- If tag A's children are about China domestic events and tag B is about a foreign event, they are NOT the same category — do NOT establish a parent-child relation between them
- If children tags show the two abstract tags cover completely different domains or regions, choose the one that is broader but also prefer "A" if they are truly unrelated
- If they are equally broad, choose the one with a shorter/more concise label as parent
- If unclear, default to "A" as parent`, tag1.Label, truncateStr(tag1.Description, 200), formatChildLabels(children1),
		tag2.Label, truncateStr(tag2.Description, 200), formatChildLabels(children2))

	req := airouter.ChatRequest{
		Capability: airouter.CapabilityTopicTagging,
		Messages: []airouter.Message{
			{Role: "system", Content: "You are a tag taxonomy assistant. Respond only with valid JSON."},
			{Role: "user", Content: prompt},
		},
		JSONMode: true,
		JSONSchema: &airouter.JSONSchema{
			Type: "object",
			Properties: map[string]airouter.SchemaProperty{
				"parent": {Type: "string", Description: "更宽泛的标签标识，A 或 B"},
				"reason": {Type: "string", Description: "判断理由"},
			},
			Required: []string{"parent", "reason"},
		},
		Temperature: func() *float64 { f := 0.3; return &f }(),
	}

	result, err := router.Chat(ctx, req)
	if err != nil {
		return 0, 0, fmt.Errorf("LLM call failed: %w", err)
	}

	var parsed struct {
		Parent string `json:"parent"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(result.Content), &parsed); err != nil {
		return 0, 0, fmt.Errorf("parse LLM response: %w", err)
	}

	if strings.ToUpper(parsed.Parent) == "B" {
		return tag2ID, tag1ID, nil
	}
	return tag1ID, tag2ID, nil
}

func aiJudgeAlternativePlacement(ctx context.Context, tagID uint, suggestedParentID uint) (uint, string, error) {
	if database.DB == nil {
		return 0, "", fmt.Errorf("database not initialized")
	}

	var tag models.TopicTag
	if err := database.DB.First(&tag, tagID).Error; err != nil {
		return 0, "", err
	}

	var suggestedParent models.TopicTag
	if err := database.DB.First(&suggestedParent, suggestedParentID).Error; err != nil {
		return 0, "", err
	}

	siblings := loadAbstractChildLabels(suggestedParentID, 8)
	tagChildren := loadAbstractChildLabels(tagID, 5)

	router := airouter.NewRouter()
	prompt := fmt.Sprintf(`一个抽象标签即将被放置到层级树中，但目标位置会导致层级过深（超过4层）。
请判断该标签最合适的归属。

待放置标签: %q (描述: %s)
待放置标签路径: %s
该标签的子标签: %s

原定父标签: %q (描述: %s)
原定父标签路径: %s
原定父标签的子标签: %s

规则:
- 不要创建新的深层级
- 优先选择合并到已有标签，或放置到更浅的层级
- 如果该标签与原定父标签的某个子标签概念重叠，返回该子标签ID

返回 JSON: {"target_id": 目标标签ID或0表示不放置, "reason": "简要说明"}`,
		tag.Label, truncateStr(tag.Description, 200), loadTagPathString(tagID, 6), formatChildLabels(tagChildren),
		suggestedParent.Label, truncateStr(suggestedParent.Description, 200), loadTagPathString(suggestedParentID, 6), formatChildLabels(siblings))

	req := airouter.ChatRequest{
		Capability: airouter.CapabilityTopicTagging,
		Messages: []airouter.Message{
			{Role: "system", Content: "You are a tag taxonomy assistant. Respond only with valid JSON."},
			{Role: "user", Content: prompt},
		},
		JSONMode: true,
		JSONSchema: &airouter.JSONSchema{
			Type: "object",
			Properties: map[string]airouter.SchemaProperty{
				"target_id": {Type: "integer", Description: "目标标签ID，0表示不放置"},
				"reason":    {Type: "string", Description: "判断理由"},
			},
			Required: []string{"target_id", "reason"},
		},
		Temperature: func() *float64 { f := 0.2; return &f }(),
		Metadata: map[string]any{
			"operation":        "depth_limit_alternative",
			"tag_id":           tagID,
			"suggested_parent": suggestedParentID,
		},
	}

	result, err := router.Chat(ctx, req)
	if err != nil {
		return 0, "", fmt.Errorf("LLM call failed: %w", err)
	}

	var parsed struct {
		TargetID uint   `json:"target_id"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(result.Content), &parsed); err != nil {
		return 0, "", fmt.Errorf("parse response: %w", err)
	}

	return parsed.TargetID, parsed.Reason, nil
}

func judgeCrossLayerDuplicate(ctx context.Context, sourceID uint, candidateID uint) (bool, string, error) {
	if database.DB == nil {
		return false, "", fmt.Errorf("database not initialized")
	}

	var sourceTag models.TopicTag
	if err := database.DB.First(&sourceTag, sourceID).Error; err != nil {
		return false, "", err
	}
	var candidateTag models.TopicTag
	if err := database.DB.First(&candidateTag, candidateID).Error; err != nil {
		return false, "", err
	}

	router := airouter.NewRouter()
	prompt := fmt.Sprintf(`请判断以下两个标签是否描述同一概念，并且应该合并。

标签 A: %q
- 描述: %s
- 层级路径: %s
- 子标签: %s

标签 B: %q
- 描述: %s
- 层级路径: %s
- 子标签: %s

规则:
- 只回答是否应合并，不要讨论层级摆放
- 只有在两个标签的核心概念相同、保留为两个节点会造成重复时，才返回 should_merge=true

返回 JSON: {"should_merge": true/false, "reason": "简要说明"}`,
		sourceTag.Label, truncateStr(sourceTag.Description, 200), loadTagPathString(sourceID, 6), formatChildLabels(loadAbstractChildLabels(sourceID, 5)),
		candidateTag.Label, truncateStr(candidateTag.Description, 200), loadTagPathString(candidateID, 6), formatChildLabels(loadAbstractChildLabels(candidateID, 5)))

	req := airouter.ChatRequest{
		Capability: airouter.CapabilityTopicTagging,
		Messages: []airouter.Message{
			{Role: "system", Content: "You are a tag taxonomy assistant. Respond only with valid JSON."},
			{Role: "user", Content: prompt},
		},
		JSONMode: true,
		JSONSchema: &airouter.JSONSchema{
			Type: "object",
			Properties: map[string]airouter.SchemaProperty{
				"should_merge": {Type: "boolean", Description: "是否应合并"},
				"reason":       {Type: "string", Description: "判断理由"},
			},
			Required: []string{"should_merge", "reason"},
		},
		Temperature: func() *float64 { f := 0.1; return &f }(),
		Metadata: map[string]any{
			"operation":    "judge_cross_layer_duplicate",
			"source_id":    sourceID,
			"candidate_id": candidateID,
		},
	}

	result, err := router.Chat(ctx, req)
	if err != nil {
		return false, "", fmt.Errorf("LLM call failed: %w", err)
	}

	var parsed struct {
		ShouldMerge bool   `json:"should_merge"`
		Reason      string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(result.Content), &parsed); err != nil {
		return false, "", fmt.Errorf("parse response: %w", err)
	}

	return parsed.ShouldMerge, parsed.Reason, nil
}

func mergeOrLinkSimilarAbstract(ctx context.Context, tag1ID, tag2ID uint) error {
	var tag1, tag2 models.TopicTag
	if err := database.DB.First(&tag1, tag1ID).Error; err != nil {
		return fmt.Errorf("load tag %d: %w", tag1ID, err)
	}
	if err := database.DB.First(&tag2, tag2ID).Error; err != nil {
		return fmt.Errorf("load tag %d: %w", tag2ID, err)
	}

	children1 := loadAbstractChildLabels(tag1ID, 5)
	children2 := loadAbstractChildLabels(tag2ID, 5)

	router := airouter.NewRouter()
	prompt := fmt.Sprintf(`两个抽象标签非常相似，请判断它们的关系。

标签 A: %q (描述: %s)
A 的子标签: %s

标签 B: %q (描述: %s)
B 的子标签: %s

判断:
- 如果它们描述的是完全相同的概念（只是表述不同），返回 "merge"
- 如果 A 是 B 的上位概念（更宽泛），返回 "parent_A"
- 如果 B 是 A 的上位概念（更宽泛），返回 "parent_B"

返回 JSON: {"action": "merge"|"parent_A"|"parent_B", "target": "A"|"B", "reason": "简要说明"}`,
		tag1.Label, truncateStr(tag1.Description, 200), formatChildLabels(children1),
		tag2.Label, truncateStr(tag2.Description, 200), formatChildLabels(children2))

	req := airouter.ChatRequest{
		Capability: airouter.CapabilityTopicTagging,
		Messages: []airouter.Message{
			{Role: "system", Content: "You are a tag taxonomy assistant. Respond only with valid JSON."},
			{Role: "user", Content: prompt},
		},
		JSONMode: true,
		JSONSchema: &airouter.JSONSchema{
			Type: "object",
			Properties: map[string]airouter.SchemaProperty{
				"action": {Type: "string", Description: "merge, parent_A, 或 parent_B"},
				"target": {Type: "string", Description: "A 或 B，merge 时保留的标签"},
				"reason": {Type: "string", Description: "判断理由"},
			},
			Required: []string{"action", "target", "reason"},
		},
		Temperature: func() *float64 { f := 0.2; return &f }(),
		Metadata: map[string]any{
			"operation": "merge_or_link_similar_abstract",
			"tag_a":     tag1ID,
			"tag_b":     tag2ID,
		},
	}

	result, err := router.Chat(ctx, req)
	if err != nil {
		return fmt.Errorf("LLM call failed: %w", err)
	}

	var parsed struct {
		Action string `json:"action"`
		Target string `json:"target"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(result.Content), &parsed); err != nil {
		return fmt.Errorf("parse LLM response: %w", err)
	}

	switch parsed.Action {
	case "merge":
		sourceID, targetID := tag1ID, tag2ID
		sourceLabel, targetLabel := tag1.Label, tag2.Label
		if strings.EqualFold(parsed.Target, "A") {
			sourceID, targetID = tag2ID, tag1ID
			sourceLabel, targetLabel = tag2.Label, tag1.Label
		}
		logging.Infof("mergeOrLinkSimilarAbstract: merging %d (%s) into %d (%s), reason: %s",
			sourceID, sourceLabel, targetID, targetLabel, parsed.Reason)
		return MergeTags(sourceID, targetID)
	case "parent_A":
		return linkAbstractParentChild(tag2ID, tag1ID)
	case "parent_B":
		return linkAbstractParentChild(tag1ID, tag2ID)
	default:
		return fmt.Errorf("unknown action %q from LLM", parsed.Action)
	}
}

func reparentOrLinkAbstractChild(ctx context.Context, childID, newParentID uint) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		wouldCycle, err := wouldCreateCycle(tx, newParentID, childID)
		if err != nil {
			return fmt.Errorf("cycle check: %w", err)
		}
		if wouldCycle {
			return fmt.Errorf("would create cycle: parent=%d, child=%d", newParentID, childID)
		}

		var count int64
		tx.Model(&models.TopicTagRelation{}).
			Where("parent_id = ? AND child_id = ? AND relation_type = ?", newParentID, childID, "abstract").
			Count(&count)
		if count > 0 {
			return nil
		}

		var oldParentRelation models.TopicTagRelation
		if err := tx.Where("child_id = ? AND relation_type = ?", childID, "abstract").First(&oldParentRelation).Error; err == nil {
			var oldParentTag models.TopicTag
			if err := tx.First(&oldParentTag, oldParentRelation.ParentID).Error; err != nil {
				return fmt.Errorf("load old parent tag: %w", err)
			}

			var newParentTag models.TopicTag
			if err := tx.First(&newParentTag, newParentID).Error; err != nil {
				return fmt.Errorf("load new parent tag: %w", err)
			}

			narrower, err := aiJudgeNarrowerConceptFn(ctx, &newParentTag, &oldParentTag)
			if err != nil {
				return fmt.Errorf("judge old parent narrower: %w", err)
			}
			if !narrower {
				return fmt.Errorf("child %d already has parent %d which is not narrower than %d", childID, oldParentRelation.ParentID, newParentID)
			}

			oldParentCycle, err := wouldCreateCycle(tx, newParentID, oldParentRelation.ParentID)
			if err != nil {
				return fmt.Errorf("cycle check for old parent: %w", err)
			}
			if oldParentCycle {
				return fmt.Errorf("would create cycle via old parent: parent=%d, old_parent=%d", newParentID, oldParentRelation.ParentID)
			}

			relation := models.TopicTagRelation{
				ParentID:     newParentID,
				ChildID:      oldParentRelation.ParentID,
				RelationType: "abstract",
			}
			return tx.Where("parent_id = ? AND child_id = ? AND relation_type = ?", newParentID, oldParentRelation.ParentID, "abstract").FirstOrCreate(&relation).Error
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("find old parent: %w", err)
		}

		relation := models.TopicTagRelation{
			ParentID:     newParentID,
			ChildID:      childID,
			RelationType: "abstract",
		}
		return tx.Create(&relation).Error
	})
}
