---
status: testing
phase: 06-tag-merge-ui
source: [06-01-SUMMARY.md, 06-02-SUMMARY.md, 06-03-SUMMARY.md]
started: 2026-04-13T14:30:00Z
updated: 2026-04-13T14:55:00Z
---

## Current Test

number: 4
name: 内联编辑目标标签名
expected: |
  候选卡片上点击铅笔图标 → 标签名变为可编辑输入框 → 输入新名称并保存 → 显示更新后的名称。
awaiting: user response

## Tests

### 1. Cold Start Smoke Test
expected: Start Go backend from scratch. Server boots without errors. Navigate to topics page — page loads without console errors.
result: pass

### 2. Merge Preview API
expected: GET /api/topic-tags/merge-preview returns candidate pairs; POST /api/topic-tags/merge-with-name merges with custom name.
result: skipped
reason: 聚焦业务层 UI 测试

### 3. 标签合并预览弹窗与候选卡片
expected: |
  点击侧边栏"标签合并预览"按钮 → 弹窗打开 → 点击扫描 → 候选卡片出现，每张卡片显示：源→目标标签名、相似度分数、双方文章数。可展开查看文章标题（双栏布局）。
result: issue
reported: "修复映射后数据正确显示，但渲染样式很奇怪。另外扫描时间太长，应限定当前选中的feed或分类。"
severity: minor
fixed: true
fix: "tagMergePreview.ts 添加 snake_case→camelCase 映射"

### 4. 内联编辑目标标签名
expected: |
  候选卡片上点击铅笔图标 → 标签名变为可编辑输入框 → 输入新名称并保存 → 显示更新后的名称，有取消/保存控制。
result: [pending]

### 5. 合并与跳过操作
expected: |
  单张卡片：点击"合并"按钮显示加载状态，完成后标记已合并；点击"跳过"移除或灰化卡片。有"全部合并"按钮可批量处理剩余候选。
result: [pending]

### 6. 合并汇总与图谱刷新
expected: |
  所有候选处理完毕后显示汇总视图（合并数、跳过数、失败数）。关闭弹窗后，底层话题图谱自动刷新，反映合并后的标签。
result: [pending]

## Summary

total: 6
passed: 1
issues: 1
pending: 3
skipped: 1
blocked: 0

## Gaps

- truth: "候选卡片正确显示源/目标标签名、相似度、文章计数"
  status: fixed
  reason: "API返回snake_case，前端类型用camelCase，apiClient无映射"
  severity: blocker
  test: 3
  root_cause: "tagMergePreview.ts缺少snake_case→camelCase字段映射层"
  artifacts:
    - path: "front/app/api/tagMergePreview.ts"
      issue: "直接使用apiClient.get返回值，未做字段名映射"
  missing:
    - "添加Raw接口接收原始snake_case数据，map函数转为camelCase类型"
  fix_applied: true

- truth: "扫描限定当前选中的feed或分类，速度合理"
  status: failed
  reason: "用户反馈扫描时间太长，应限定当前选中的feed或分类范围"
  severity: minor
  test: 3
  root_cause: "preview API扫描全量标签，无scope过滤"
  artifacts:
    - path: "backend-go/internal/domain/topicanalysis/tag_merge_preview.go"
      issue: "ScanSimilarTagPairs查询全量标签无分类过滤"
  missing:
    - "前端传入当前feed/category作为scope参数"
    - "后端ScanSimilarTagPairs增加scope过滤"
