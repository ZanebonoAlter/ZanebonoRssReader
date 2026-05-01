# 清理 topic_tag_relations 中指向非活跃标签的无效关系

## 问题

`topic_tag_relations` 中存在 26 条 abstract 关系，其 parent 或 child 标签已经是 inactive/merged 状态。

这些无效关系会导致：
- `BuildTagForest` 构建树时包含无效节点（已通过代码修复过滤）
- `CleanupOrphanedRelations` 未清理指向非活跃标签的关系
- 数据冗余，影响查询性能

## 现状

```sql
-- 当前无效关系数量
SELECT count(*)
FROM topic_tag_relations r
JOIN topic_tags p ON r.parent_id = p.id
JOIN topic_tags c ON r.child_id = c.id
WHERE r.relation_type = 'abstract'
  AND (p.status != 'active' OR c.status != 'active');
-- 结果: 26 条
```

## 修复方案

### 1. 数据清理 SQL

```sql
DELETE FROM topic_tag_relations
WHERE relation_type = 'abstract'
  AND (
    parent_id IN (SELECT id FROM topic_tags WHERE status != 'active')
    OR child_id IN (SELECT id FROM topic_tags WHERE status != 'active')
  );
```

### 2. 代码加固

在 `CleanupOrphanedRelations` (`internal/domain/topicanalysis/tag_cleanup.go`) 中增加对非活跃标签关系的清理逻辑，使其在每次 cleanup cycle 中自动清理。

### 3. 预防措施

在创建 abstract 关系时，添加 parent 和 child 标签的 status 校验。

## 优先级

中 - 不影响功能正确性（代码已过滤），但属于数据卫生问题。

## 相关代码

- `internal/domain/topicanalysis/hierarchy_cleanup.go` - `BuildTagForest` 已加 active 过滤
- `internal/domain/topicanalysis/tag_cleanup.go` - `CleanupOrphanedRelations`
- `internal/jobs/tag_hierarchy_cleanup.go` - cleanup scheduler
