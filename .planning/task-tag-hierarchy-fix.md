# TagHierarchy 时间过滤修复计划

## Bug 描述

TopicGraph 页面的标签层级（TagHierarchy）存在三个问题：

1. **默认时间范围是"全部"**：应默认使用父组件的时间锚点日期（selectedDate）
2. **后端过滤实现粗糙**：`GetTagHierarchy` 和 `GetUnclassifiedTags` 返回全量标签，仅标记 `is_active`，不过滤
3. **灰色标签排行靠前**：前端 `sortNodesByActivity` 只有 active/inactive 一级排序，无质量分二级排序

### 附带修复（已完成）

4. **时间线 pending 切换 bug**：点击"待整理"后切换日报看不到关联文章
   - 根因：`handleDigestSelect` / `handlePreviewDigest` 未清除 `selectedPendingNode`
   - 已修复：`TopicGraphPage.vue:933-944` 两处加了 `selectedPendingNode.value = false`

## 已完成

### 后端 `abstract_tag_service.go`

- [x] **`GetTagHierarchy`**：提前调用 `resolveActiveTagIDs`，当 `timeRange` 非空时裁剪 `relations` 只保留活跃子节点的边，重建 `tagIDSet` 后再加载标签。`IsActive` 字段：有 timeRange 时永远 true，无 timeRange 时按原逻辑
- [x] **`countArticlesByTag`**：新增批量计数函数，替代原来 N+1 查询。支持 1d/7d/30d/custom 时间范围的 GROUP BY 计数
- [x] **`GetUnclassifiedTags`**：当 `timeRange` 非空时过滤掉非活跃标签再返回，`IsActive` 永远 true

### 前端 bug 修复

- [x] `TopicGraphPage.vue` — `handleDigestSelect` 和 `handlePreviewDigest` 加 `selectedPendingNode.value = false`

## 未完成

### 前端 `TagHierarchy.vue`（已部分开始）

- [ ] **接受 `anchorDate` prop**（已加 prop 定义，但初始化和 watch 未完成）
  - `timeRange` 初始值应从 `anchorDate` 计算：`custom:${anchorDate}:${anchorDate}`
  - watch `anchorDate` 变化时更新 `timeRange`

  当前代码状态：prop 已加，但 `timeRange` ref 仍为 `''`。需要改为：
  ```ts
  const initialTimeRange = props.anchorDate ? `custom:${props.anchorDate}:${props.anchorDate}` : ''
  const timeRange = ref<string>(initialTimeRange)
  ```
  并加 watch：
  ```ts
  watch(() => props.anchorDate, (newDate) => {
    if (newDate) {
      timeRange.value = `custom:${newDate}:${newDate}`
    }
  })
  ```

### 前端 `TopicGraphPage.vue`

- [ ] **传 `selectedDate` 给 `TagHierarchy`**
  在标签层级 tab 的 `<TagHierarchy>` 上加 `:anchor-date="selectedDate"`
  位置约在 `TopicGraphPage.vue:1289-1294`

### 前端 `TagHierarchy.vue`

- [ ] **`sortNodesByActivity` 增加质量分二级排序**
  当前实现只按 `isActive` 排，没有二级排序。改为：
  ```ts
  function sortNodesByActivity(list: TagHierarchyNode[]): TagHierarchyNode[] {
    return [...list].sort((a, b) => {
      const aActive = a.isActive !== false
      const bActive = b.isActive !== false
      if (aActive !== bActive) return aActive ? -1 : 1
      const aScore = a.qualityScore ?? 0
      const bScore = b.qualityScore ?? 0
      if (bScore !== aScore) return bScore - aScore
      return (b.feedCount ?? 0) - (a.feedCount ?? 0)
    }).map(node => ({
      ...node,
      children: sortNodesByActivity(node.children),
    }))
  }
  ```

## 验证步骤

```bash
# 后端
cd backend-go && go test ./internal/domain/topicanalysis/... -v
cd backend-go && go build ./...

# 前端
cd front && pnpm exec nuxi typecheck
cd front && pnpm build
```

## 关键文件

| 文件 | 改动 |
|------|------|
| `backend-go/internal/domain/topicanalysis/abstract_tag_service.go` | 已改完 |
| `front/app/features/topic-graph/components/TagHierarchy.vue` | prop 已加，timeRange 初始化和 watch 待改 |
| `front/app/features/topic-graph/components/TopicGraphPage.vue` | 待传 selectedDate |
| `front/app/features/topic-graph/components/TagHierarchyRow.vue` | 无需改动（灰色样式 `opacity-40` 后端已不返回非活跃节点，自然消除） |
