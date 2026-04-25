package topicanalysis

import (
	"testing"

	"my-robot-backend/internal/domain/models"
)

func makeTag(id uint, label string) *models.TopicTag {
	return &models.TopicTag{ID: id, Label: label, Status: "active", Category: "event"}
}

func makeTagWithStatus(id uint, label string, status string) *models.TopicTag {
	return &models.TopicTag{ID: id, Label: label, Status: status, Category: "event"}
}

func TestCalculateTreeDepth_SingleNode(t *testing.T) {
	root := &TreeNode{Tag: makeTag(1, "root"), Depth: 1}
	if d := calculateTreeDepth(root); d != 1 {
		t.Errorf("expected 1, got %d", d)
	}
}

func TestCalculateTreeDepth_ThreeLevels(t *testing.T) {
	root := &TreeNode{Tag: makeTag(1, "root"), Depth: 1}
	child := &TreeNode{Tag: makeTag(2, "child"), Depth: 2, Parent: root}
	grandchild := &TreeNode{Tag: makeTag(3, "grandchild"), Depth: 3, Parent: child}
	root.Children = []*TreeNode{child}
	child.Children = []*TreeNode{grandchild}

	if d := calculateTreeDepth(root); d != 3 {
		t.Errorf("expected 3, got %d", d)
	}
}

func TestCalculateTreeDepth_FiveLevels(t *testing.T) {
	n1 := &TreeNode{Tag: makeTag(1, "a"), Depth: 1}
	n2 := &TreeNode{Tag: makeTag(2, "b"), Depth: 2, Parent: n1}
	n3 := &TreeNode{Tag: makeTag(3, "c"), Depth: 3, Parent: n2}
	n4 := &TreeNode{Tag: makeTag(4, "d"), Depth: 4, Parent: n3}
	n5 := &TreeNode{Tag: makeTag(5, "e"), Depth: 5, Parent: n4}
	n1.Children = []*TreeNode{n2}
	n2.Children = []*TreeNode{n3}
	n3.Children = []*TreeNode{n4}
	n4.Children = []*TreeNode{n5}

	if d := calculateTreeDepth(n1); d != 5 {
		t.Errorf("expected 5, got %d", d)
	}
}

func TestCountNodes(t *testing.T) {
	root := &TreeNode{Tag: makeTag(1, "root"), Depth: 1}
	c1 := &TreeNode{Tag: makeTag(2, "c1"), Depth: 2, Parent: root}
	c2 := &TreeNode{Tag: makeTag(3, "c2"), Depth: 2, Parent: root}
	root.Children = []*TreeNode{c1, c2}

	if n := countNodes(root); n != 3 {
		t.Errorf("expected 3, got %d", n)
	}
}

func TestCollectAllTags(t *testing.T) {
	root := &TreeNode{Tag: makeTag(1, "root"), Depth: 1}
	c1 := &TreeNode{Tag: makeTag(2, "c1"), Depth: 2, Parent: root}
	c2 := &TreeNode{Tag: makeTag(3, "c2"), Depth: 2, Parent: root}
	root.Children = []*TreeNode{c1, c2}

	tags := collectAllTags(root)
	if len(tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(tags))
	}
}

func TestIsDirectParentChild(t *testing.T) {
	parent := &TreeNode{Tag: makeTag(1, "parent"), Depth: 1}
	child := &TreeNode{Tag: makeTag(2, "child"), Depth: 2, Parent: parent}
	parent.Children = []*TreeNode{child}
	sibling := &TreeNode{Tag: makeTag(3, "sibling"), Depth: 2, Parent: parent}
	parent.Children = append(parent.Children, sibling)

	if !isDirectParentChild(parent, child) {
		t.Error("parent and child should be direct parent-child")
	}
	if !isDirectParentChild(child, parent) {
		t.Error("child and parent should be direct parent-child (reversed)")
	}
	if isDirectParentChild(child, sibling) {
		t.Error("siblings should NOT be direct parent-child")
	}
}

func TestValidateAndExecuteMerge_SameTag(t *testing.T) {
	node := &TreeNode{Tag: makeTag(1, "tag"), Depth: 1}
	tagMap := map[uint]*TreeNode{1: node}
	merge := treeCleanupMerge{SourceID: 1, TargetID: 1}

	err := validateAndExecuteMerge(merge, tagMap)
	if err == nil {
		t.Error("expected error for merging same tag")
	}
}

func TestValidateAndExecuteMerge_DirectParentChild(t *testing.T) {
	parent := &TreeNode{Tag: makeTag(1, "parent"), Depth: 1}
	child := &TreeNode{Tag: makeTag(2, "child"), Depth: 2, Parent: parent}
	parent.Children = []*TreeNode{child}
	tagMap := map[uint]*TreeNode{1: parent, 2: child}

	merge := treeCleanupMerge{SourceID: 2, TargetID: 1}
	err := validateAndExecuteMerge(merge, tagMap)
	if err == nil {
		t.Error("expected error for direct parent-child merge")
	}
}

func TestValidateAndExecuteMerge_DepthDiffLessThan2(t *testing.T) {
	n1 := &TreeNode{Tag: makeTag(1, "a"), Depth: 2}
	n2 := &TreeNode{Tag: makeTag(2, "b"), Depth: 3}
	tagMap := map[uint]*TreeNode{1: n1, 2: n2}

	merge := treeCleanupMerge{SourceID: 2, TargetID: 1}
	err := validateAndExecuteMerge(merge, tagMap)
	if err == nil {
		t.Error("expected error for depth diff < 2")
	}
}

func TestValidateAndExecuteMerge_InactiveTag(t *testing.T) {
	n1 := &TreeNode{Tag: makeTag(1, "a"), Depth: 1}
	n2 := &TreeNode{Tag: makeTagWithStatus(2, "b", "inactive"), Depth: 5}
	tagMap := map[uint]*TreeNode{1: n1, 2: n2}

	merge := treeCleanupMerge{SourceID: 2, TargetID: 1}
	err := validateAndExecuteMerge(merge, tagMap)
	if err == nil {
		t.Error("expected error for inactive tag")
	}
}

func TestValidateAndExecuteMerge_TagNotFound(t *testing.T) {
	n1 := &TreeNode{Tag: makeTag(1, "a"), Depth: 1}
	tagMap := map[uint]*TreeNode{1: n1}

	merge := treeCleanupMerge{SourceID: 999, TargetID: 1}
	err := validateAndExecuteMerge(merge, tagMap)
	if err == nil {
		t.Error("expected error for missing source tag")
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		input, expected int
	}{
		{5, 5},
		{-3, 3},
		{0, 0},
	}
	for _, tt := range tests {
		if got := abs(tt.input); got != tt.expected {
			t.Errorf("abs(%d) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestCollectDeepNodes(t *testing.T) {
	root := &TreeNode{Tag: makeTag(1, "root"), Depth: 1}
	c1 := &TreeNode{Tag: makeTag(2, "c1"), Depth: 2, Parent: root}
	c2 := &TreeNode{Tag: makeTag(3, "c2"), Depth: 3, Parent: c1}
	c3 := &TreeNode{Tag: makeTag(4, "c3"), Depth: 4, Parent: c2}
	root.Children = []*TreeNode{c1}
	c1.Children = []*TreeNode{c2}
	c2.Children = []*TreeNode{c3}

	var deep []*TreeNode
	collectDeepNodes(root, 3, &deep)
	if len(deep) != 2 {
		t.Errorf("expected 2 deep nodes (depth>=3), got %d", len(deep))
	}
}

func TestCollectDeepNodes_None(t *testing.T) {
	root := &TreeNode{Tag: makeTag(1, "root"), Depth: 1}
	c1 := &TreeNode{Tag: makeTag(2, "c1"), Depth: 2, Parent: root}
	root.Children = []*TreeNode{c1}

	var deep []*TreeNode
	collectDeepNodes(root, 3, &deep)
	if len(deep) != 0 {
		t.Errorf("expected 0 deep nodes, got %d", len(deep))
	}
}

func TestMinTreeDepthConstant(t *testing.T) {
	if MinTreeDepthForCleanup != 3 {
		t.Errorf("expected MinTreeDepthForCleanup=3, got %d", MinTreeDepthForCleanup)
	}
}

func TestExecuteHierarchyCleanupPhase4SkipsShallowTrees(t *testing.T) {
	setupAbstractTagServiceTestDB(t)

	result, err := ExecuteHierarchyCleanupPhase4("event")
	if err != nil {
		t.Fatalf("ExecuteHierarchyCleanupPhase4 returned error: %v", err)
	}
	if result.TreesProcessed != 0 {
		t.Fatalf("expected no deep trees processed, got %+v", result)
	}
}
