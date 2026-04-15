# Phase 06: 标签合并交互界面 - Context

**Gathered:** 2026-04-13
**Status:** Ready for planning

<domain>
## Phase Boundary

用户可手动触发全量标签相似度扫描，预览候选合并对，编辑合并后标签名称，确认后执行合并，查看合并结果汇总。

**Scope anchor:**
- 手动触发扫描（返回预览，不自动执行）
- 预览界面展示候选对
- 名称编辑功能
- 逐对确认/跳过 + 批量合并
- 结果汇总展示

**Out of scope:**
- 自动合并逻辑（已由 scheduler 处理）
- 标签删除功能
- 合并后撤销机制

</domain>

<decisions>
## Implementation Decisions

### 预览界面布局
- **D-01:** 卡片列表展示候选合并对
  - 每张卡片显示：源标签名称、目标标签名称、相似度分数、源文章数、目标文章数
  - 可点击展开查看关联文章标题列表
  - 适合少量候选对（预期 ≤ 50 对）

### 名称编辑方式
- **D-02:** 内联编辑合并后名称
  - 卡片上显示编辑按钮（铅笔图标）
  - 点击后出现输入框，默认值为目标标签名称
  - 用户可输入任意新名称（不限于源或目标的名称）
  - 保存或取消按钮在输入框旁
  - 编辑就在预览界面完成，无需弹窗

### 确认流程
- **D-03:** 逐对确认 + 批量合并
  - 每张卡片有「合并」和「跳过」按钮
  - 页面顶部有「全部合并」按钮，一键处理所有未跳过的候选对
  - 点击合并后立即执行，卡片消失或标记为已处理
  - 点击跳过后卡片标记为已跳过，不再显示
  - 扫描结果缓存，用户可反复处理直到全部决定

### 结果展示
- **D-04:** 合并完成后显示汇总
  - 成功合并 N 个标签
  - 跳过 M 个标签对
  - 失败 K 个（如有）
  - 列出被合并的标签名称及其新名称
  - 提供关闭按钮，返回标签管理界面

### 功能定位
- **D-05:** 此功能为手动干预/预览入口
  - 自动合并已由 `auto_tag_merge.go` scheduler 处理
  - UI 允许用户手动触发扫描并预览结果
  - 用户可审视每个候选对并决定是否合并
  - 扫描逻辑复用 scheduler 的 `scanAndMergeTags` 但返回预览而非执行

### Agent's Discretion
- 触发按钮位置（在哪个页面入口）
- 空状态文案（无候选对时的提示）
- 进度指示（扫描进行中的 loading 状态）
- 卡片样式细节（颜色、图标、排版）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Backend - Merge Logic
- `backend-go/internal/jobs/auto_tag_merge.go` — 自动合并调度器，扫描逻辑可复用
- `backend-go/internal/domain/topicanalysis/embedding.go` — MergeTags 函数
- `backend-go/internal/domain/topicanalysis/config_service.go` — 相似度阈值配置

### Frontend - Existing Patterns
- `front/app/api/topicGraph.ts` — mergeTags API 已存在
- `front/app/api/mergeReembeddingQueue.ts` — 队列状态查询模式可参考
- `front/app/features/topic-graph/components/TopicAnalysisTabs.vue` — Tab 组件模式
- `front/app/components/dialog/EditFeedDialog.vue` — 内联编辑确认流程可参考

### Frontend - UI Conventions
- `front/AGENTS.md` — Vue 3 Composition API 规范
- `.planning/codebase/CONVENTIONS.md` — 前端编码约定

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `auto_tag_merge.go:scanAndMergeTags()` — 查询相似标签对的 SQL 可直接复用
- `topicanalysis.MergeTags(sourceID, targetID)` — 已实现合并逻辑
- `topicGraph.ts:mergeTags()` — 前端 API 已存在
- TopicAnalysisTabs.vue — Tab 组件布局模式

### Established Patterns
- 队列状态查询：`mergeReembeddingQueue.ts` 的 getStatus/getTasks 模式
- 内联确认流程：EditFeedDialog.vue 的 showDeleteConfirm 模式
- API 返回格式：`{ success, data, error, message }`

### Integration Points
- 后端新路由：`/topic-tags/merge-preview`（扫描返回候选对）
- 后端新路由：`/topic-tags/merge-with-name`（带名称的合并）
- 前端入口：可在 topics.vue 或 TopicGraphSidebar.vue 添加触发按钮

</code_context>

<specifics>
## Specific Ideas

- 用户可在卡片上直接编辑合并后名称，无需进入单独页面
- 扫描结果缓存，用户可逐步处理而非一次性决定全部
- 「全部合并」按钮仅合并未跳过的候选对，已跳过的排除

</specifics>

<deferred>
## Deferred Ideas

- 合并后撤销机制：技术复杂度高（需恢复 article_topic_tags 引用），后续迭代
- 合并历史记录查看：展示哪些标签被合并、何时合并，可后续迭代
- 扫描参数调整：允许用户自定义相似度阈值，可后续迭代

None — discussion stayed within phase scope

</deferred>

---

*Phase: 06-tag-merge-ui*
*Context gathered: 2026-04-13*