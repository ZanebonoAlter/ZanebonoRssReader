# Tag Cleanup Redesign — 三级清理策略

## 背景

原始 tag 层级清理仅处理 `depth >= 5` 的深树，但实际数据中仅 2 棵树满足条件，导致清理几乎不触发。66% 的标签未被层次化，30% 为僵尸标签。

## 数据现状

| 指标 | 数值 | 影响 |
|------|------|------|
| Active tags | 3,369 | 碎片化严重 |
| 孤立标签（无 abstract 关系） | 2,240 (66%) | 大量标签未被层次化 |
| 孤立且无文章（僵尸） | 1,006 (30%) | 最高优先级清理 |
| Abstract event 标签关联文章 | 全部 0 篇 | 抽象标签不参与文章关联 |
| depth >= 3 的树 | 覆盖更多 | 降低门槛后触发面扩大 |
| 多父节点冲突 | 15+ 个标签 | 需要精简 |

## 三级策略

### Phase 1: 僵尸标签清理（无 LLM）
- 条件：`status=active` + 无 abstract 关系 + 关联文章 = 0 + 超过 MinAgeDays
- 操作：批量标记为 `inactive`
- 函数：`CleanupZombieTags`, `BuildZombieTagSubQuery`

### Phase 2: 扁平化相似合并（LLM 辅助）
- 不依赖树结构，按 category 分批
- 从 abstract 标签池中检测相似/重复对
- 每批 <= 50 个标签，LLM 判断是否合并
- 函数：`CollectFlatTagBatch`, `BuildFlatMergePrompt`, `ExecuteFlatMerge`, `callFlatMergeLLM`

### Phase 3: 层次结构精简
- 降低深度门槛：`depth >= 5` -> `depth >= MinTreeDepthForCleanup (3)`
- 处理多父节点冲突：保留最相似关系
- 清理无叶子节点的 abstract 中间节点
- 清理引用了 `merged` 状态标签的 abstract 关系
- 函数：`CleanupOrphanedRelations`, `CleanupMultiParentConflicts`, `CleanupEmptyAbstractNodes`

### Phase 3b: 树结构清理（原有逻辑，降低门槛）
- 使用 `BuildTagForest` + `ProcessTree` 处理深树
- 深度阈值从 5 降低到 3

## 关键文件

| 文件 | 说明 |
|------|------|
| `topicanalysis/tag_cleanup.go` | Phase 1/2/3 核心逻辑 |
| `topicanalysis/tag_cleanup_test.go` | 测试 |
| `topicanalysis/hierarchy_cleanup.go` | BuildTagForest, ProcessTree, 深度常量 |
| `jobs/tag_hierarchy_cleanup.go` | 调度器，runCleanupCycle 集成三级策略 |

## 调度器执行顺序

`runCleanupCycle` 按以下顺序执行：
1. Phase 1: `CleanupZombieTags` — 无 LLM
2. Phase 2: `ExecuteFlatMerge("event")` + `ExecuteFlatMerge("keyword")` — LLM 辅助
3. Phase 3: `CleanupOrphanedRelations` -> `CleanupMultiParentConflicts` -> `CleanupEmptyAbstractNodes`
4. Phase 3b: `BuildTagForest` + `ProcessTree` — 原有深树处理

## Run Summary 字段

```json
{
  "trigger_source": "scheduled|manual",
  "zombie_count": 1006,
  "flat_merges_applied": 15,
  "orphaned_relations": 50,
  "multi_parent_resolved": 12,
  "empty_abstracts": 200,
  "trees_processed": 5,
  "tree_merges_applied": 3,
  "tree_abstracts_created": 2,
  "errors": 0
}
```
