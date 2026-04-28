# Phase 07: Middle-band 抽象标签提取 - Context

**Gathered:** 2026-04-13
**Status:** Ready for planning

<domain>
## Phase Boundary

在 embedding 相似度中间地带 (0.78-0.97) 提取共同抽象标签，避免无意义的新标签创建，减少标签碎片。

**不包括：**
- 修改 high/low 阈值逻辑
- 修改现有标签合并流程
- 全量标签聚类展示

</domain>

<decisions>
## Implementation Decisions

### 抽象标签提取策略
- **D-01:** 使用 LLM 分析候选标签的语义，生成抽象概念名称
- 复用现有 airouter 基础设施调用 LLM
- LLM 需要从候选标签中提取共同主题，并生成一个更抽象的标签名称

### 触发时机
- **D-02:** 实时触发 - 新文章入库时遇到 middle-band 相似度立即触发抽象标签提取
- 在 `tagger.go` 的 `ai_judgment` 分支中集成抽象标签逻辑
- 不影响现有 high/low 阈值的处理流程

### 抽象标签使用规则
- **D-03:** 文章只关联到子标签，抽象标签仅用于展示和聚合
- 保持现有 `article_topic_tags` 和 `ai_summary_topics` 关联逻辑不变
- 抽象标签不直接参与文章标签关联

- **D-04:** 用户可以手动编辑抽象标签的名称
- 提供 API 支持更新抽象标签的 label
- 前端在层级树中提供编辑入口

- **D-05:** 支持多级嵌套 - 抽象标签可以是另一个抽象标签的子标签
- `topic_tag_relations` 表支持任意层级的父子关系
- 前端层级树需要递归渲染

### 前端展示方式
- **D-06:** 层级树展示 - 树形结构展示标签层级，支持展开/折叠
- 复用现有 TopicGraphPage 作为入口
- 新增 TagHierarchy.vue 组件

### 抽象标签合并管理
- **D-07:** 新增专门的抽象标签管理页面，支持对抽象标签进行二次/多次合并
- 复用现有的 TagMergePreview 组件（相似度扫描 + 预览 + 自定义名称）
- 原有的标签合并功能移动到全局配置页面（Settings），作为系统级功能
- 抽象标签管理页面专注于层级结构的维护和优化

### Claude's Discretion
- 抽象标签的 category 继承自第一个子标签的 category
- 抽象标签默认为 active 状态，可被合并
- 相似度分数保留在关联表中用于后续分析

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 相关代码
- `backend-go/internal/domain/topicanalysis/embedding.go` — TagMatch 函数，三级匹配逻辑
- `backend-go/internal/domain/topicextraction/tagger.go` — 标签创建和 ai_judgment 处理
- `backend-go/internal/domain/models/topic_graph.go` — TopicTag 模型定义
- `backend-go/internal/platform/airouter/embedding.go` — Embedding 客户端
- `backend-go/internal/domain/topicanalysis/tag_merge_preview.go` — 标签合并预览扫描逻辑
- `backend-go/internal/domain/topicanalysis/tag_merge_preview_handler.go` — 预览和自定义合并 API
- `front/app/features/topic-graph/components/TagMergePreview.vue` — 合并预览组件（可复用）

### 项目文档
- `.planning/ROADMAP.md` § Phase 7 — 阶段目标和成功标准
- `.planning/REQUIREMENTS.md` § CONV-03 — 中间地带处理需求

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `airouter.Router` — 可复用的 LLM 调用框架，支持 CapabilityChat
- `EmbeddingService` — 已有相似度搜索和 TagMatch 逻辑
- `topictypes.Slugify` — 标签 slug 生成工具
- `database.DB` — GORM 数据库连接

### Established Patterns
- Handler 模式：验证参数 → 业务逻辑 → 返回 gin.H
- 事务模式：使用 `database.DB.Transaction` 处理复杂操作
- 异步模式：使用 goroutine 处理非阻塞任务（如 embedding 生成）

### Integration Points
- `tagger.go` 的 `ai_judgment` 分支 — 集成抽象标签提取逻辑
- `embedding.go` 的 `TagMatch` 函数 — 返回新的 match_type 或扩展处理
- `router.go` — 注册新的抽象标签管理 API

</code_context>

<specifics>
## Specific Ideas

### 数据模型
新建 `topic_tag_relations` 表：
```sql
CREATE TABLE topic_tag_relations (
  id SERIAL PRIMARY KEY,
  parent_id INTEGER NOT NULL REFERENCES topic_tags(id),
  child_id INTEGER NOT NULL REFERENCES topic_tags(id),
  relation_type VARCHAR(20) NOT NULL, -- abstract, synonym, related
  similarity_score FLOAT,
  created_at TIMESTAMP DEFAULT NOW(),
  UNIQUE(parent_id, child_id)
);
```

### 抽象标签提取流程
1. 检测到 middle-band 相似度 (0.78-0.97)
2. 收集候选标签（top 3 相似标签）
3. 调用 LLM 分析候选标签，生成抽象概念
4. 创建新的抽象标签（或复用已存在的抽象标签）
5. 建立父子关系（topic_tag_relations）
6. 返回抽象标签作为匹配结果

### API 设计
- `GET /api/tags/hierarchy` — 获取标签层级树
- `PUT /api/tags/:id/abstract-name` — 更新抽象标签名称
- `POST /api/tags/:id/detach` — 将子标签从抽象标签中分离
- `POST /api/tags/abstract/scan` — 扫描可合并的抽象标签对
- `POST /api/tags/abstract/merge` — 合并抽象标签（复用现有 MergeTags 逻辑）

### 前端页面结构
```
/pages/
├── topics.vue              # 标签图谱主页（现有）
├── topics/
│   ├── hierarchy.vue       # 标签层级树（新增）
│   └── abstract-merge.vue  # 抽象标签合并管理（新增）
└── settings/
    └── tags.vue            # 全局标签配置（移动现有合并功能）
```

### 组件复用
- `TagMergePreview.vue` — 复用于抽象标签合并页面
- 相同的扫描 → 预览 → 自定义名称 → 确认流程
- 区别：只展示和处理 abstract 类型的标签

</specifics>

<deferred>
## Deferred Ideas

- **全量标签聚类展示** — 未来可在标签管理界面展示完整的标签层级结构
- **抽象标签统计** — 展示抽象标签下的文章数量、时间分布等统计信息
- **批量抽象标签提取** — 定时任务批量处理历史标签的抽象化

</deferred>

---

*Phase: 07-middle-band*
*Context gathered: 2026-04-13*
