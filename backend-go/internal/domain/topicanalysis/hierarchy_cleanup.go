package topicanalysis

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/domain/topictypes"
	"my-robot-backend/internal/platform/airouter"
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/jsonutil"
	"my-robot-backend/internal/platform/logging"

	"gorm.io/gorm"
)

// TreeNode represents a node in the tag hierarchy tree

const MinTreeDepthForCleanup = 3

type TreeNode struct {
	Tag          *models.TopicTag
	Depth        int
	Children     []*TreeNode
	Parent       *TreeNode
	ArticleCount int
}

// TreeCleanupResult holds the result of processing a single tree
type TreeCleanupResult struct {
	TreeRootID       uint
	TreeRootLabel    string
	TagsProcessed    int
	MergesApplied    int
	AbstractsCreated int
	Errors           []string
}

type HierarchyPhase4Result struct {
	TreesProcessed   int
	TagsProcessed    int
	MergesApplied    int
	ReparentsApplied int
	Errors           []string
}

// treeCleanupJudgment is the LLM's judgment for a batch of tags
type treeCleanupJudgment struct {
	Merges    []treeCleanupMerge    `json:"merges,omitempty"`
	Abstracts []treeCleanupAbstract `json:"abstracts,omitempty"`
	Notes     string                `json:"notes,omitempty"`
}

type treeCleanupMerge struct {
	SourceID uint   `json:"source_id"`
	TargetID uint   `json:"target_id"`
	Reason   string `json:"reason"`
}

type treeCleanupAbstract struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ChildrenIDs []uint `json:"children_ids"`
	Reason      string `json:"reason"`
}

// tagTreeInfo is used for LLM prompt
type tagTreeInfo struct {
	ID           uint   `json:"id"`
	Label        string `json:"label"`
	Description  string `json:"description"`
	Depth        int    `json:"depth"`
	ArticleCount int    `json:"article_count"`
	ChildrenIDs  []uint `json:"children_ids"`
	ParentID     *uint  `json:"parent_id,omitempty"`
}

// BuildTagForest builds all tag trees for a given category, filtering trees with depth >= MinTreeDepthForCleanup
func BuildTagForest(category string) ([]*TreeNode, error) {
	// Load all abstract relations
	var relations []models.TopicTagRelation
	if err := database.DB.Where("relation_type = ?", "abstract").Find(&relations).Error; err != nil {
		return nil, fmt.Errorf("query tag relations: %w", err)
	}

	if len(relations) == 0 {
		return nil, nil
	}

	// Build parent->children map
	childrenMap := make(map[uint][]uint)
	parentSet := make(map[uint]bool)
	childSet := make(map[uint]bool)

	for _, r := range relations {
		childrenMap[r.ParentID] = append(childrenMap[r.ParentID], r.ChildID)
		parentSet[r.ParentID] = true
		childSet[r.ChildID] = true
	}

	// Find root nodes (nodes that are parents but not children)
	var rootIDs []uint
	for parentID := range parentSet {
		if !childSet[parentID] {
			rootIDs = append(rootIDs, parentID)
		}
	}

	if len(rootIDs) == 0 {
		// Handle cycles: find entry points
		rootIDs = findCycleRoots(relations, parentSet)
	}

	// Load all tags in the hierarchy
	allTagIDs := make(map[uint]bool)
	for _, r := range relations {
		allTagIDs[r.ParentID] = true
		allTagIDs[r.ChildID] = true
	}

	tagIDs := make([]uint, 0, len(allTagIDs))
	for id := range allTagIDs {
		tagIDs = append(tagIDs, id)
	}

	var tags []models.TopicTag
	if err := database.DB.Where("id IN ? AND status = ? AND category = ?", tagIDs, "active", category).Find(&tags).Error; err != nil {
		return nil, fmt.Errorf("load tags: %w", err)
	}

	tagMap := make(map[uint]*models.TopicTag)
	for i := range tags {
		tagMap[tags[i].ID] = &tags[i]
	}

	// Build article counts
	articleCounts := countArticlesByTag(tagIDs, "")

	// Build trees
	var forest []*TreeNode
	for _, rootID := range rootIDs {
		rootTag, ok := tagMap[rootID]
		if !ok {
			continue
		}
		root := buildTreeNode(rootTag, 1, childrenMap, tagMap, articleCounts)
		depth := calculateTreeDepth(root)
		if depth >= MinTreeDepthForCleanup {
			forest = append(forest, root)
		}
	}

	return forest, nil
}

// buildTreeNode recursively builds a tree from the root
func buildTreeNode(tag *models.TopicTag, depth int, childrenMap map[uint][]uint, tagMap map[uint]*models.TopicTag, articleCounts map[uint]int) *TreeNode {
	node := &TreeNode{
		Tag:          tag,
		Depth:        depth,
		ArticleCount: articleCounts[tag.ID],
	}

	for _, childID := range childrenMap[tag.ID] {
		childTag, ok := tagMap[childID]
		if !ok {
			continue
		}
		childNode := buildTreeNode(childTag, depth+1, childrenMap, tagMap, articleCounts)
		childNode.Parent = node
		node.Children = append(node.Children, childNode)
	}

	return node
}

// calculateTreeDepth calculates the maximum depth of a tree
func calculateTreeDepth(node *TreeNode) int {
	if len(node.Children) == 0 {
		return 1
	}
	maxChildDepth := 0
	for _, child := range node.Children {
		d := calculateTreeDepth(child)
		if d > maxChildDepth {
			maxChildDepth = d
		}
	}
	return maxChildDepth + 1
}

// countNodes counts total nodes in a tree
func countNodes(node *TreeNode) int {
	count := 1
	for _, child := range node.Children {
		count += countNodes(child)
	}
	return count
}

// collectAllTags collects all tags in a tree
func collectAllTags(node *TreeNode) []*TreeNode {
	result := []*TreeNode{node}
	for _, child := range node.Children {
		result = append(result, collectAllTags(child)...)
	}
	return result
}

// findCycleRoots finds root nodes in cyclic graphs
func findCycleRoots(relations []models.TopicTagRelation, parentSet map[uint]bool) []uint {
	childToParent := make(map[uint]uint)
	for _, r := range relations {
		childToParent[r.ChildID] = r.ParentID
	}

	cycleRoots := make(map[uint]bool)
	globalVisited := make(map[uint]bool)

	for pid := range parentSet {
		if globalVisited[pid] {
			continue
		}
		path := make(map[uint]bool)
		current := pid
		for {
			if path[current] {
				cycleRoots[current] = true
				break
			}
			if globalVisited[current] {
				break
			}
			path[current] = true
			p, ok := childToParent[current]
			if !ok {
				break
			}
			current = p
		}
		for id := range path {
			globalVisited[id] = true
		}
	}

	var result []uint
	for id := range cycleRoots {
		result = append(result, id)
	}
	return result
}

func ExecuteHierarchyCleanupPhase4(category string) (*HierarchyPhase4Result, error) {
	forest, err := BuildTagForest(category)
	if err != nil {
		return nil, err
	}

	result := &HierarchyPhase4Result{}
	for _, root := range forest {
		treeResult, treeErr := cleanupDeepHierarchyTree(context.Background(), root)
		if treeErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("tree %d: %v", root.Tag.ID, treeErr))
			continue
		}
		result.TreesProcessed++
		result.TagsProcessed += treeResult.TagsProcessed
		result.MergesApplied += treeResult.MergesApplied
		result.ReparentsApplied += treeResult.AbstractsCreated
		result.Errors = append(result.Errors, treeResult.Errors...)
	}

	return result, nil
}

func cleanupDeepHierarchyTree(ctx context.Context, root *TreeNode) (*TreeCleanupResult, error) {
	if root == nil || root.Tag == nil {
		return &TreeCleanupResult{}, nil
	}

	result := &TreeCleanupResult{
		TreeRootID:    root.Tag.ID,
		TreeRootLabel: root.Tag.Label,
	}

	nodes := collectAllTags(root)
	result.TagsProcessed = len(nodes)

	for _, node := range nodes {
		if node == nil || node.Tag == nil || node.Tag.Source != "abstract" {
			continue
		}

		if node.Depth >= MinTreeDepthForCleanup {
			candidates, err := findCrossLayerDuplicateCandidatesFn(ctx, node.Tag.ID, node.Tag.Category)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("cross-layer candidates for %d: %v", node.Tag.ID, err))
			} else {
				for _, candidate := range candidates {
					if candidate.Tag == nil {
						continue
					}
					shouldMerge, reason, judgeErr := judgeCrossLayerDuplicateFn(ctx, node.Tag.ID, candidate.Tag.ID)
					if judgeErr != nil {
						result.Errors = append(result.Errors, fmt.Sprintf("cross-layer judge %d->%d: %v", node.Tag.ID, candidate.Tag.ID, judgeErr))
						continue
					}
					if !shouldMerge {
						continue
					}
					if mergeErr := mergeTagsFn(node.Tag.ID, candidate.Tag.ID); mergeErr != nil {
						result.Errors = append(result.Errors, fmt.Sprintf("cross-layer merge %d->%d: %v", node.Tag.ID, candidate.Tag.ID, mergeErr))
						continue
					}
					logging.Infof("Hierarchy cleanup phase 4: merged %d into %d, reason=%s", node.Tag.ID, candidate.Tag.ID, reason)
					result.MergesApplied++
					break
				}
			}
		}

		if node.Depth > 4 && node.Parent != nil && node.Parent.Tag != nil {
			alternativeID, reason, err := aiJudgeAlternativePlacementFn(ctx, node.Tag.ID, node.Parent.Tag.ID)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("depth compression judge %d: %v", node.Tag.ID, err))
				continue
			}
			if alternativeID == 0 || alternativeID == node.Parent.Tag.ID || alternativeID == node.Tag.ID {
				continue
			}
			if linkErr := linkAbstractParentChild(node.Tag.ID, alternativeID); linkErr != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("depth compression link %d->%d: %v", node.Tag.ID, alternativeID, linkErr))
				continue
			}
			logging.Infof("Hierarchy cleanup phase 4: moved deep tag %d under %d, reason=%s", node.Tag.ID, alternativeID, reason)
			result.AbstractsCreated++
		}
	}

	return result, nil
}

// ProcessTree recursively processes a tag tree, splitting into batches of <= 50
func ProcessTree(node *TreeNode) (*TreeCleanupResult, error) {
	totalNodes := countNodes(node)

	// If the tree has <= 50 nodes, process it directly
	if totalNodes <= 50 {
		return processBatch(node)
	}

	// Otherwise, process each child subtree recursively
	result := &TreeCleanupResult{
		TreeRootID:    node.Tag.ID,
		TreeRootLabel: node.Tag.Label,
	}

	for _, child := range node.Children {
		childResult, err := ProcessTree(child)
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
			continue
		}
		result.TagsProcessed += childResult.TagsProcessed
		result.MergesApplied += childResult.MergesApplied
		result.AbstractsCreated += childResult.AbstractsCreated
		result.Errors = append(result.Errors, childResult.Errors...)
	}

	crossResult, err := processRootCrossLayer(node)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("cross-layer: %v", err))
	} else if crossResult != nil {
		result.TagsProcessed += crossResult.TagsProcessed
		result.MergesApplied += crossResult.MergesApplied
		result.Errors = append(result.Errors, crossResult.Errors...)
	}

	return result, nil
}

func processRootCrossLayer(root *TreeNode) (*TreeCleanupResult, error) {
	var deepNodes []*TreeNode
	collectDeepNodes(root, 3, &deepNodes)

	if len(deepNodes) == 0 {
		return nil, nil
	}

	batch := append([]*TreeNode{root}, deepNodes...)
	if len(batch) > 50 {
		batch = batch[:50]
	}

	prompt := buildCleanupPrompt(root, batch)
	judgment, err := callCleanupLLM(prompt)
	if err != nil {
		return nil, err
	}

	result := &TreeCleanupResult{
		TreeRootID:    root.Tag.ID,
		TreeRootLabel: root.Tag.Label,
		TagsProcessed: len(batch),
	}

	tagMap := make(map[uint]*TreeNode)
	for _, tag := range batch {
		tagMap[tag.Tag.ID] = tag
	}

	for _, merge := range judgment.Merges {
		if err := validateAndExecuteMerge(merge, tagMap); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("cross-layer merge %d->%d: %v", merge.SourceID, merge.TargetID, err))
			continue
		}
		result.MergesApplied++
	}

	return result, nil
}

func collectDeepNodes(node *TreeNode, minDepth int, result *[]*TreeNode) {
	if node.Depth >= minDepth {
		*result = append(*result, node)
	}
	for _, child := range node.Children {
		collectDeepNodes(child, minDepth, result)
	}
}

// processBatch processes a batch of tags (<= 50 nodes)
func processBatch(root *TreeNode) (*TreeCleanupResult, error) {
	tags := collectAllTags(root)

	// Build prompt
	prompt := buildCleanupPrompt(root, tags)

	// Call LLM
	judgment, err := callCleanupLLM(prompt)
	if err != nil {
		return nil, err
	}

	result := &TreeCleanupResult{
		TreeRootID:    root.Tag.ID,
		TreeRootLabel: root.Tag.Label,
		TagsProcessed: len(tags),
	}

	// Create a tag map for quick lookup
	tagMap := make(map[uint]*TreeNode)
	for _, tag := range tags {
		tagMap[tag.Tag.ID] = tag
	}

	// Execute merges
	for _, merge := range judgment.Merges {
		if err := validateAndExecuteMerge(merge, tagMap); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("merge %d->%d: %v", merge.SourceID, merge.TargetID, err))
			continue
		}
		result.MergesApplied++
	}

	// Execute abstracts
	for _, abstract := range judgment.Abstracts {
		if err := validateAndExecuteAbstract(abstract, tagMap, root.Tag.Category); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("abstract %s: %v", abstract.Name, err))
			continue
		}
		result.AbstractsCreated++
	}

	return result, nil
}

// buildCleanupPrompt builds the LLM prompt for tree cleanup
func buildCleanupPrompt(root *TreeNode, tags []*TreeNode) string {
	// Collect tree info
	treeInfo := map[string]interface{}{
		"root_label": root.Tag.Label,
		"max_depth":  calculateTreeDepth(root),
		"total_tags": len(tags),
		"category":   root.Tag.Category,
	}

	// Collect tag info, sorted by depth then label
	var tagInfos []tagTreeInfo
	for _, tag := range tags {
		info := tagTreeInfo{
			ID:           tag.Tag.ID,
			Label:        tag.Tag.Label,
			Description:  truncateStr(tag.Tag.Description, 200),
			Depth:        tag.Depth,
			ArticleCount: tag.ArticleCount,
		}
		for _, child := range tag.Children {
			info.ChildrenIDs = append(info.ChildrenIDs, child.Tag.ID)
		}
		if tag.Parent != nil {
			pid := tag.Parent.Tag.ID
			info.ParentID = &pid
		}
		tagInfos = append(tagInfos, info)
	}

	sort.Slice(tagInfos, func(i, j int) bool {
		if tagInfos[i].Depth != tagInfos[j].Depth {
			return tagInfos[i].Depth < tagInfos[j].Depth
		}
		return tagInfos[i].Label < tagInfos[j].Label
	})

	// Build prompt data
	promptData := map[string]interface{}{
		"tree_info": treeInfo,
		"tags":      tagInfos,
	}

	promptJSON, _ := json.MarshalIndent(promptData, "", "  ")

	return fmt.Sprintf(`你是一位标签分类专家。请分析以下标签树，找出问题并提出清理建议。

当前标签树结构：
%s

请分析并返回以下格式的 JSON：
{
  "merges": [
    {
      "source_id": 123,
      "target_id": 456,
      "reason": "这两个标签描述的是同一个概念，应该合并"
    }
  ],
  "abstracts": [
    {
      "name": "新的抽象标签名称",
      "description": "对这个抽象标签的客观描述（500字以内）",
      "children_ids": [123, 456, 789],
      "reason": "这些标签共享一个共同的上层概念，应该被归入一个抽象父标签"
    }
  ],
  "notes": "其他观察（可选）"
}

规则：
1. merges 和 abstracts 都是可选的，可以为空数组
2. merge: 当两个非相邻层级的标签描述的是同一核心概念时使用
   - source_id: 被合并的标签（会被淘汰）
   - target_id: 保留的目标标签
   - 两个标签的 depth 差必须 >= 2
   - 优先保留更上层（depth 更小）的标签作为 target
3. abstract: 当一组标签（2个以上）共享一个共同的上层概念，但当前树中没有合适的抽象标签时使用
   - name: 新的抽象标签名称（1-160字符）
   - description: 客观描述（500字以内）
   - children_ids: 应该归入这个抽象标签的子标签 ID 列表（至少2个）
   - 新抽象标签会连接到这些子标签的最小公共祖先下
4. 不要修改直接父子关系（depth 差 = 1 的标签）
5. 如果树结构合理，没有需要修改的地方，返回空的 merges 和 abstracts
6. 只返回真正有把握的建议`, string(promptJSON))
}

// callCleanupLLM calls LLM for tree cleanup judgment
func callCleanupLLM(prompt string) (*treeCleanupJudgment, error) {
	router := airouter.NewRouter()

	req := airouter.ChatRequest{
		Capability: airouter.CapabilityTopicTagging,
		Messages: []airouter.Message{
			{Role: "system", Content: "You are a tag taxonomy cleanup assistant. Respond only with valid JSON."},
			{Role: "user", Content: prompt},
		},
		JSONMode: true,
		JSONSchema: &airouter.JSONSchema{
			Type: "object",
			Properties: map[string]airouter.SchemaProperty{
				"merges": {
					Type: "array",
					Items: &airouter.SchemaProperty{
						Type: "object",
						Properties: map[string]airouter.SchemaProperty{
							"source_id": {Type: "integer"},
							"target_id": {Type: "integer"},
							"reason":    {Type: "string"},
						},
						Required: []string{"source_id", "target_id", "reason"},
					},
				},
				"abstracts": {
					Type: "array",
					Items: &airouter.SchemaProperty{
						Type: "object",
						Properties: map[string]airouter.SchemaProperty{
							"name":         {Type: "string"},
							"description":  {Type: "string"},
							"children_ids": {Type: "array", Items: &airouter.SchemaProperty{Type: "integer"}},
							"reason":       {Type: "string"},
						},
						Required: []string{"name", "description", "children_ids", "reason"},
					},
				},
				"notes": {Type: "string"},
			},
		},
		Temperature: func() *float64 { f := 0.2; return &f }(),
		Metadata: map[string]any{
			"operation": "tag_hierarchy_cleanup",
		},
	}

	result, err := router.Chat(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	content := jsonutil.SanitizeLLMJSON(result.Content)

	var judgment treeCleanupJudgment
	if err := json.Unmarshal([]byte(content), &judgment); err != nil {
		return nil, fmt.Errorf("parse LLM response: %w", err)
	}

	logging.Infof("Hierarchy cleanup LLM judgment: %d merges, %d abstracts",
		len(judgment.Merges), len(judgment.Abstracts))

	return &judgment, nil
}

// validateAndExecuteMerge validates and executes a merge suggestion
func validateAndExecuteMerge(merge treeCleanupMerge, tagMap map[uint]*TreeNode) error {
	source, ok := tagMap[merge.SourceID]
	if !ok {
		return fmt.Errorf("source tag %d not found", merge.SourceID)
	}

	target, ok := tagMap[merge.TargetID]
	if !ok {
		return fmt.Errorf("target tag %d not found", merge.TargetID)
	}

	// Check same tag
	if merge.SourceID == merge.TargetID {
		return fmt.Errorf("source and target are the same tag")
	}

	// Check active status
	if source.Tag.Status != "active" || target.Tag.Status != "active" {
		return fmt.Errorf("one or both tags are not active")
	}

	// Check direct parent-child
	if isDirectParentChild(source, target) {
		return fmt.Errorf("direct parent-child relationship")
	}

	// Check depth difference
	depthDiff := abs(source.Depth - target.Depth)
	if depthDiff < 2 {
		return fmt.Errorf("depth difference < 2")
	}

	// Execute merge
	logging.Infof("Hierarchy cleanup: merging tag %d (%s) into %d (%s), reason: %s",
		merge.SourceID, source.Tag.Label, merge.TargetID, target.Tag.Label, merge.Reason)

	if err := MergeTags(merge.SourceID, merge.TargetID); err != nil {
		return fmt.Errorf("merge failed: %w", err)
	}

	return nil
}

func validateAndExecuteAbstract(abstract treeCleanupAbstract, tagMap map[uint]*TreeNode, category string) error {
	if len(abstract.ChildrenIDs) < 2 {
		return fmt.Errorf("abstract tag needs at least 2 children, got %d", len(abstract.ChildrenIDs))
	}

	for _, childID := range abstract.ChildrenIDs {
		node, ok := tagMap[childID]
		if !ok {
			return fmt.Errorf("child tag %d not found in tree", childID)
		}
		if node.Tag.Status != "active" {
			return fmt.Errorf("child tag %d is not active", childID)
		}
	}

	return createAbstractTagDirectly(abstract, tagMap, category)
}

func createAbstractTagDirectly(abstract treeCleanupAbstract, tagMap map[uint]*TreeNode, category string) error {
	slug := topictypes.Slugify(abstract.Name)
	if slug == "" {
		return fmt.Errorf("generated empty slug for abstract name %q", abstract.Name)
	}

	for _, childID := range abstract.ChildrenIDs {
		node := tagMap[childID]
		if node.Tag.Slug == slug {
			logging.Infof("Hierarchy cleanup: abstract name %q (slug=%s) collides with candidate tag, skipping", abstract.Name, slug)
			return nil
		}
	}

	var candidates []TagCandidate
	for _, childID := range abstract.ChildrenIDs {
		node := tagMap[childID]
		candidates = append(candidates, TagCandidate{
			Tag:        node.Tag,
			Similarity: 0.9,
		})
	}

	var abstractTag *models.TopicTag
	if existingAbstract := findSimilarExistingAbstractFn(context.Background(), abstract.Name, abstract.Description, category, candidates); existingAbstract != nil {
		abstractTag = existingAbstract
	}

	var abstractChildren []*models.TopicTag

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if abstractTag == nil {
			var existing models.TopicTag
			if err := tx.Where("slug = ? AND category = ? AND status = ?", slug, category, "active").First(&existing).Error; err == nil {
				abstractTag = &existing
			}
		}

		if abstractTag == nil {
			abstractTag = &models.TopicTag{
				Slug:        slug,
				Label:       abstract.Name,
				Category:    category,
				Kind:        category,
				Source:      "abstract",
				Status:      "active",
				Description: abstract.Description,
			}
			if err := tx.Create(abstractTag).Error; err != nil {
				return fmt.Errorf("create abstract tag: %w", err)
			}
		}

		for _, childID := range abstract.ChildrenIDs {
			node := tagMap[childID]
			if node.Tag.ID == abstractTag.ID {
				continue
			}

			wouldCycle, err := wouldCreateCycle(tx, abstractTag.ID, node.Tag.ID)
			if err != nil {
				return fmt.Errorf("check cycle for child %d: %w", node.Tag.ID, err)
			}
			if wouldCycle {
				logging.Warnf("Hierarchy cleanup: skipping cyclic relation: abstract %d -> child %d", abstractTag.ID, node.Tag.ID)
				continue
			}

			var count int64
			tx.Model(&models.TopicTagRelation{}).
				Where("parent_id = ? AND child_id = ? AND relation_type = ?", abstractTag.ID, node.Tag.ID, "abstract").
				Count(&count)
			if count > 0 {
				abstractChildren = append(abstractChildren, node.Tag)
				continue
			}

			relation := models.TopicTagRelation{
				ParentID:        abstractTag.ID,
				ChildID:         node.Tag.ID,
				RelationType:    "abstract",
				SimilarityScore: 0.9,
			}
			if err := tx.Create(&relation).Error; err != nil {
				return fmt.Errorf("create tag relation for child %d: %w", node.Tag.ID, err)
			}
			abstractChildren = append(abstractChildren, node.Tag)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("abstract tag transaction failed: %w", err)
	}

	logging.Infof("Hierarchy cleanup: created abstract tag %d (%s) with %d children",
		abstractTag.ID, abstractTag.Label, len(abstractChildren))

	go func(tagID uint, name, cat string) {
		es := NewEmbeddingService()
		tag := &models.TopicTag{ID: tagID, Label: name, Category: cat}
		for _, embType := range []string{EmbeddingTypeIdentity, EmbeddingTypeSemantic} {
			emb, genErr := es.GenerateEmbedding(context.Background(), tag, embType)
			if genErr != nil {
				logging.Warnf("Hierarchy cleanup: failed to generate %s embedding for abstract tag %d: %v", embType, tagID, genErr)
				continue
			}
			emb.TopicTagID = tagID
			if saveErr := es.SaveEmbedding(emb); saveErr != nil {
				logging.Warnf("Hierarchy cleanup: failed to save %s embedding for abstract tag %d: %v", embType, tagID, saveErr)
			}
		}
		MatchAbstractTagHierarchy(context.Background(), tagID)
		adoptNarrowerAbstractChildren(context.Background(), tagID)
	}(abstractTag.ID, abstract.Name, category)

	go EnqueueAbstractTagUpdate(abstractTag.ID, "new_child_added")

	for _, child := range abstractChildren {
		go func(childID uint) {
			_, _ = resolveMultiParentConflict(childID)
		}(child.ID)
	}

	return nil
}

// isDirectParentChild checks if two nodes are direct parent and child
func isDirectParentChild(a, b *TreeNode) bool {
	if a.Parent == b || b.Parent == a {
		return true
	}
	return false
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
