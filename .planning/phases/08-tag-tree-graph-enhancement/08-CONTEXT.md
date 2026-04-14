# Phase 08: 标签树增强与图谱交互优化 - Context

**Gathered:** 2026-04-14
**Status:** Ready for planning

<domain>
## Phase Boundary

完善标签树与图谱的交互体验：标签简介提取补充 LLM 上下文、时间维度筛选、抽象标签在图谱中可视化、合并预览功能迁移至设置页、节点手动归类。

**不包括：**
- 修改标签收敛/合并核心逻辑
- 新增标签提取流程
- 修改抽象标签提取的 LLM 调用方式

</domain>

<decisions>
## Implementation Decisions

### Description 提取
- **D-01:** 文章标签的 description 在 `findOrCreateTag` 创建新标签时同步生成，复用 JSON 模式保证结构化输出
  - 在 tagger 流程中，创建新标签后立即调用 LLM 生成 description
  - LLM 输入：标签名称 + slug + category + 关联文章的标题/摘要片段
  - 输出为结构化 JSON（description 字段）
- **D-02:** 抽象标签的 description 在 `ExtractAbstractTag` 调用 LLM 生成名称时顺带生成
  - 一次 LLM 调用同时返回 name + description
  - description 基于子标签的 description 聚合总结

### 时间筛选
- **D-03:** 标签树的时间筛选基于关联文章的发布时间（`articles.published_at`），不是标签创建时间
  - 筛选逻辑：某时间范围内有关联文章的标签为"活跃"标签
  - 时间范围支持：最近 7 天、30 天、自定义范围
- **D-04:** 不活跃的标签在筛选后置灰但保留在树中，不隐藏
  - 保留完整层级结构，视觉上降低透明度区分活跃/不活跃

### 图谱中抽象标签展示
- **D-05:** 抽象标签节点保持 category 颜色（event/keyword/person），加外发光/光环效果区分
  - 颜色不变：event=#f59e0b, person=#10b981, keyword=#6366f1
  - 光环效果通过 Three.js 材质实现（emissive 或 outline pass）
- **D-06:** 点击抽象标签节点弹出详情面板，显示子标签列表 + 文章时间线
  - 面板中可按子标签筛选时间线
  - 面板形式参考现有节点点击的行为（侧边/底部面板）

### 合并预览迁移
- **D-07:** TopicGraphPage 完全移除 TagMergePreview 入口，统一到 GlobalSettingsDialog 的 merge 队列旁边
  - TagMergePreview 组件本身不变，只变更挂载位置
  - 原有的按分类过滤功能保留（设置页可选择过滤范围）
- **D-08:** 合并完成后提示用户手动触发抽象层关系重建（非自动触发）
  - 弹出提示：'合并可能影响抽象层结构，建议重建'
  - 用户确认后执行重建逻辑

### Claude's Discretion
- description 字段的长度限制（由 LLM prompt 控制，不硬编码截断）
- 时间筛选的默认范围选择（agent 决定合理的默认值）
- 光环效果的具体实现方式（emissive 材质 vs outline pass）
- 详情面板的具体布局和动画
- 提示用户的弹窗样式和文案

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 后端 - 标签模型与服务
- `backend-go/internal/domain/models/topic_graph.go` — TopicTag 模型（需新增 description 字段）
- `backend-go/internal/domain/models/topic_tag_relation.go` — TopicTagRelation 模型
- `backend-go/internal/domain/topicanalysis/abstract_tag_service.go` — ExtractAbstractTag、GetHierarchy、DetachChildTag
- `backend-go/internal/domain/topicanalysis/embedding.go` — MergeTags（含 embedding 删除逻辑）、TagMatch
- `backend-go/internal/domain/topicextraction/tagger.go` — findOrCreateTag（description 生成集成点）

### 前端 - 标签树与图谱
- `front/app/features/topic-graph/components/TagHierarchy.vue` — 标签树主组件（需加时间筛选）
- `front/app/features/topic-graph/components/TagHierarchyRow.vue` — 递归行渲染
- `front/app/features/topic-graph/components/TopicGraphCanvas.client.vue` — 3D 图谱渲染（需加抽象标签视觉区分）
- `front/app/features/topic-graph/utils/buildTopicGraphViewModel.ts` — 图谱节点构建（颜色/大小逻辑）
- `front/app/features/topic-graph/components/TagMergePreview.vue` — 合并预览组件（需迁移位置）

### 前端 - 设置页面
- `front/app/components/dialog/GlobalSettingsDialog.vue` — 全局设置对话框（合并预览目标位置）
- `front/app/features/ai/components/MergeReembeddingQueuePanel.vue` — merge 队列面板（合并预览旁边）

### 前端 - API 层
- `front/app/api/abstractTag.ts` — 抽象标签相关 API
- `front/app/api/tagMergePreview.ts` — 合并预览 API
- `front/app/types/topicTag.ts` — TagHierarchyNode 类型定义（需扩展 description）

### Phase 7 决策
- `.planning/phases/07-middle-band/07-CONTEXT.md` — 抽象标签提取策略、层级关系、递归树基础

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `abstract_tag_service.go:ExtractAbstractTag()` — 已有 LLM 调用生成抽象标签名称，可扩展为同时生成 description
- `TagHierarchy.vue` + `TagHierarchyRow.vue` — 递归树组件已有，加时间筛选/置灰即可
- `TopicGraphCanvas.client.vue` — Three.js 节点材质已有 category 颜色映射，加光环效果只需扩展 `buildNodeObject()`
- `TagMergePreview.vue` — 组件逻辑完整，迁移位置不需要改内部逻辑
- `MergeReembeddingQueuePanel.vue` — 已在 GlobalSettingsDialog 中，合并预览放在旁边即可

### Established Patterns
- **JSON 模式 LLM 调用**: 用于 structured output（如标签提取），description 生成可复用
- **详情面板模式**: 图谱节点点击弹出侧面板展示详情（参考现有 TopicGraphSidebar 行为）
- **GlobalSettingsDialog Tab 模式**: 新面板以 Tab 或子面板形式加入现有设置对话框
- **置灰/透明度模式**: 前端通过 CSS opacity 区分活跃/不活跃状态

### Integration Points
- `tagger.go:findOrCreateTag()` — 新标签创建后触发 description 生成
- `abstract_tag_service.go:ExtractAbstractTag()` — 扩展 LLM prompt 同时返回 description
- `TopicGraphCanvas.client.vue:buildNodeObject()` — 扩展材质属性加光环
- `TopicGraphPage.vue` — 移除 TagMergePreview 挂载
- `GlobalSettingsDialog.vue` — 新增 TagMergePreview 挂载 + 重建提示逻辑
- `TagHierarchy.vue` — 新增时间筛选参数传递给 API

</code_context>

<specifics>
## Specific Ideas

- description 的 LLM prompt 应包含关联文章的标题/摘要片段，提供足够上下文
- 时间筛选的 API 参数可复用现有的文章查询逻辑（按 published_at 范围过滤），后端返回标签活跃度标记
- 3D 图谱光环效果优先用 emissive 材质（性能好），outline pass 作为备选
- 合并后重建提示用 toast 或 inline 提示，不阻断用户操作流程

</specifics>

<deferred>
## Deferred Ideas

- **description 的 LLM 质量评估**: 未来可加质量检查，description 过短或不准确时重新生成
- **时间筛选的热度排序**: 在时间范围内按文章数量排序标签，识别热点主题
- **图谱中抽象标签的动态展开/收起**: 交互式展开抽象标签的子节点，减少图谱初始信息量

</deferred>

---

*Phase: 08-tag-tree-graph-enhancement*
*Context gathered: 2026-04-14*
