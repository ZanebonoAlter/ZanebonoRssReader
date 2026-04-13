# 数据库说明

## 当前数据库

主分支仅支持 PostgreSQL 数据库驱动，默认配置为 `postgres`。SQLite 驱动已归档到 `sqlite` 独立分支，主分支不再维护。

| 驱动 | 用途 | 默认连接 |
|------|------|----------|
| `postgres` | 生产/开发使用，支持 pgvector 向量检索 | `host=postgres user=postgres password=postgres dbname=rss_reader port=5432 sslmode=disable TimeZone=Asia/Shanghai` |
| `sqlite` | 已归档（仅 `sqlite` 分支可用） | `backend-go/rss_reader.db` |

## 初始化方式

后端启动时自动执行版本化迁移。核心逻辑在：

- `backend-go/internal/platform/database/db.go` — 入口，按驱动分发连接
- `backend-go/internal/platform/database/migrator.go` — 版本化迁移框架（`schema_migrations` 追踪表）
- `backend-go/internal/platform/database/postgres_migrations.go` — PostgreSQL 迁移
- `backend-go/internal/platform/database/sqlite_legacy_migrations.go` — SQLite 迁移

> 从 SQLite 迁移到 PostgreSQL 的操作步骤见 [PostgreSQL 迁移操作手册](./postgres-migration.md)。

## 当前核心表

- `categories`
- `feeds`
- `articles`
- `ai_summaries`
- `ai_summary_feeds`
- `ai_summary_queue`
- `scheduler_tasks`
- `ai_settings`
- `reading_behaviors`
- `user_preferences`
- `digest_configs`
- `topic_tags`
- `topic_tag_embeddings`
- `topic_tag_analyses`
- `topic_analysis_cursors`
- `topic_analysis_jobs`

其中：

- 核心表和补充字段保证逻辑在 `backend-go/internal/platform/database/db.go`
- digest 相关表迁移在 `backend-go/internal/digest/`

## Topic 相关表说明

数据库里以 `topic_` 开头的表，主要服务于 Topic Graph、热点标签、主题分析这条链路。它们不是孤立功能，而是围绕 `topic_tags` 这份主题主数据逐层展开。

### `topic_tags`

- 主题标签主表，存放系统里所有主题实体
- 一条记录代表一个可复用的 topic/tag，核心字段包括：
  - `slug`：稳定标识，供接口和关联表引用
  - `label`：展示名称
  - `category`：标签分类，当前主要是 `event`、`person`、`keyword`
  - `aliases`：别名列表，便于复用已有标签
  - `is_canonical`：是否为规范标签
  - `source`：标签来源，如 `llm`、`heuristic`、`manual`
- 主要职责：
  - 为 `article_topic_tags`、`ai_summary_topics` 提供统一的标签字典
  - 作为 Topic Graph 节点和热点标签列表的数据来源
  - 作为 topic analysis、embedding 的主键锚点

### `topic_tag_embeddings`

- `topic_tags` 的向量扩展表，按 `topic_tag_id` 一对一保存 embedding
- 主要字段：
  - `vector`：向量内容（JSON 文本）
  - `dimension`：向量维度
  - `model`：生成 embedding 的模型
  - `text_hash`：由标签文本生成的哈希，用于判断是否需要重算
- 主要职责：
  - 在打标签时做相似标签匹配，尽量复用已有 topic，减少重复标签
  - 支撑“高相似直接复用 / 低相似新建 / 中间区间再判断”的匹配策略

### `topic_tag_analyses`

- 主题分析结果快照表，保存某个 topic 在某个时间窗上的分析结果
- 唯一键是：`topic_tag_id + analysis_type + window_type + anchor_date`
- 主要字段：
  - `analysis_type`：分析类型，例如事件、人物、关键词视角
  - `window_type`：时间窗，如 `daily`、`weekly`
  - `anchor_date`：锚点日期
  - `summary_count`：本次分析覆盖的摘要数量
  - `payload_json`：分析结果 JSON
  - `source`：结果来源，可能是 `ai` 或 `heuristic`
  - `version`：分析版本号
- 主要职责：
  - 给 `/api/topic-graph/analysis` 提供可直接读取的分析结果
  - 避免每次打开 Topic Graph 都重新做完整分析

### `topic_analysis_cursors`

- topic analysis 的增量更新游标表
- 唯一键是：`topic_tag_id + analysis_type + window_type`
- 主要字段：
  - `last_summary_id`：上次分析已处理到的最新 `ai_summaries.id`
  - `last_updated_at`：上次刷新时间
- 主要职责：
  - 判断某个 topic 的分析结果是否需要重建
  - 配合 `topic_tag_analyses` 实现“有新摘要再更新，没有新摘要直接复用旧快照”

### `topic_analysis_jobs`

- 主题分析任务队列表的持久化镜像，真实表名是复数 `topic_analysis_jobs`
- 这张表不是最终分析结果表，而是运行时 job 状态表
- 主要字段：
  - `topic_tag_id`：分析目标 topic
  - `analysis_type`：分析类型
  - `window_type`：时间窗
  - `anchor_date`：锚点日期
  - `priority`：优先级，数值越小优先级越高
  - `status`：`pending / processing / completed / failed`
  - `retry_count`：重试次数，当前最多 3 次
  - `progress`：运行进度
  - `error_message`：失败信息
- 主要职责：
  - 给前端状态轮询提供 job 级状态来源
  - 在使用内存队列时，把未完成 job 落库，方便服务重启后恢复 pending / processing 任务
  - 和 `topic_analysis_cursors` 分工不同：`jobs` 管“这次任务跑到哪”，`cursors` 管“这个 topic 历史上已经消费到哪个 summary”

## 与这些表强相关、但不以 `topic_` 开头的表

### `article_topic_tags`

- 文章与 topic 的关联表，也是当前文章标签的事实来源
- Topic Graph 的图节点、热点列表、digest 聚合标签、文章详情 tags，当前都直接或间接依赖这张表
- 文章打标签的主流程在文章入库后或文章补全完成后执行；summary 阶段只做兜底补齐

### `ai_summary_topics`

- 摘要与 topic 的关联表
- 主要给 `topic_tag_analyses` 提供分析输入：系统会按 topic 反查关联摘要，再生成事件 / 人物 / 关键词分析快照

## 一条简化的数据链路

1. 文章或摘要被打标签，先写入 `topic_tags`
2. 文章标签关系写入 `article_topic_tags`，摘要标签关系写入 `ai_summary_topics`
3. 若启用 embedding，相似性结果写入或读取 `topic_tag_embeddings`
4. Topic Graph 页面主要消费 `topic_tags + article_topic_tags`
5. Topic Analysis 则基于 `ai_summary_topics` 生成结果，落到 `topic_tag_analyses`
6. 增量刷新状态记录在 `topic_analysis_cursors`

## Topic Analysis 详细链路

这里要区分 3 个概念：

- `topic_tag_analyses`：最终分析结果快照
- `topic_analysis_cursors`：增量消费游标
- `topic_analysis_jobs`：运行中的任务状态

### 1. 入口：前端先查结果，不是先入队

当前真实链路不是“新 summary 产生后立刻自动创建 analysis job”。

更接近下面这个过程：

1. Topic Graph 页面打开某个 topic
2. 前端先请求 `/api/topic-graph/analysis`
3. 后端 `GetOrCreateAnalysis()` 当前实际只会查 `topic_tag_analyses`，不会自动创建 job
4. 如果查到了快照，直接返回 `payload_json`
5. 如果没查到，前端再请求 `/api/topic-graph/analysis/status`
6. 若状态是 `missing`，前端会调用 `/api/topic-graph/analysis/rebuild` 主动入队

所以现在 analysis 更像“按需生成 + 前端触发重建”，不是 summary 落库后立即后台全量生成。

### 2. 入队：创建 `topic_analysis_jobs`

当用户触发 rebuild 后：

1. `RebuildAnalysis()` 调用 `enqueue(...)`
2. 队列里创建一个 `AnalysisJob`
3. job 的去重键是：
   - `topic_tag_id`
   - `analysis_type`
   - `window_type`
   - `anchor_date`
4. 同一组键如果已有 pending / processing job，不会重复插入；只会在必要时提升优先级
5. 若当前使用的是 in-memory 队列，这个 job 会同步保存到 `topic_analysis_jobs`
6. 若配置了 `REDIS_URL`，则 job 改存 Redis；此时 `topic_analysis_jobs` 不再是主存储

### 3. 执行：worker 消费 job

`analysisService.startWorker()` 会启动队列 worker，持续做这些事：

1. `Dequeue()` 取出优先级最高的 job
2. job 状态从 `pending` 变为 `processing`
3. 进度先推进到约 10%，随后服务层再 `MarkProgress(job.ID, 25)`
4. 调用 `buildAndPersist(topicTagID, analysisType, windowType, anchorDate)`

如果成功：

- job 标记为 `completed`
- 进度写到 100%
- 对应去重键释放，下次允许再次创建同类 job

如果失败：

- job 会自动重试，最多 3 次
- 超过次数后标记 `failed`
- 错误信息写入 `error_message`

### 4. 构建分析结果：读取摘要并更新快照

`buildAndPersist(...)` 的主逻辑是：

1. 先加载 `topic_tags`，确认 topic 存在
2. 根据 `window_type + anchor_date` 计算时间窗
3. 通过 `ai_summary_topics` 反查这个 topic 关联的 `ai_summaries`
4. 尝试读取现有 `topic_tag_analyses` 快照
5. 计算当前窗口里的最大 `summary_id`
6. 再读取 `topic_analysis_cursors`

关键判断在这里：

- 如果已有快照
- 且 cursor 存在
- 且 `cursor.last_summary_id >= 当前窗口内最大 summary_id`

那么说明自上次分析后没有新摘要，这次 job 会直接复用旧快照，不重算 payload。

否则才会继续真正生成分析内容。

### 5. 生成 payload：AI 优先，启发式兜底

当需要重算时：

1. 组装 `AnalysisParams`
2. 调用 `AIAnalysisService.Analyze(...)`
3. 如果 AI 成功，结果序列化后写入 `topic_tag_analyses.payload_json`，`source = ai`
4. 如果 AI 失败，就走 `buildPayload(...)` 的启发式兜底逻辑，`source = heuristic`
5. 最后更新：
   - `summary_count`
   - `payload_json`
   - `source`
   - `version`

### 6. cursor 的作用：记录“已经吃到哪里”

`topic_analysis_cursors` 不是任务队列表，而是增量消费检查点。

它解决的问题是：

- 不是“当前有没有 job 在跑”
- 而是“这个 topic 这个分析维度，历史上已经处理到哪个 summary 了”

典型用法：

1. 本次分析完成后，记录 `last_summary_id`
2. 下次同一个 topic / analysis_type / window_type 再被请求时
3. 用当前窗口里的最新 summary id 与 cursor 对比
4. 如果没有更大的 summary id，就直接复用现有快照

所以：

- `topic_analysis_jobs` 管一次任务的生命周期
- `topic_analysis_cursors` 管长期增量边界

### 7. 前端如何感知状态

Topic Graph 底部分析面板当前大致按这个顺序工作：

1. 先拉 `/api/topic-graph/analysis`
2. 没数据时再拉 `/api/topic-graph/analysis/status`
3. 如果状态是 `pending / processing`，前端每约 1.8 秒轮询一次
4. 如果状态是 `missing`，前端调用 `rebuild`
5. 如果状态变成 `ready`，前端再次拉取正式分析结果

这里后端的 `status` 来源是：

- 先看队列快照 `GetLatestByKey(...)`，也就是 job 状态
- 队列里没有时，再看 `topic_tag_analyses` 是否已有快照
- 有快照返回 `ready`
- 没快照返回 `missing`

### 8. 当前链路的一个现实点

- 文档和设计里容易把 topic analysis 理解成“summary 生成后自动入队”
- 但当前代码里，`EnqueueTopicAnalysisForSummary(...)` 还没有被 summary 主流程实际调用
- 所以现在更准确的描述应是：
  - summary 先写入 `ai_summary_topics`
  - analysis 在前端访问或手动 rebuild 时按需生成
  - `topic_analysis_cursors` 用来避免重复重算
  - `topic_analysis_jobs` 用来承接按需重建期间的任务状态

## 当前 schema 特点

这个项目现在不是只靠单次 `AutoMigrate`。

实际流程是：

1. 启动时初始化数据库连接
2. 执行 `EnsureTables()` 保证缺表存在
3. 对旧表补充新增字段
4. 再由 digest 迁移补 digest 自己的表

也就是说，数据库演进是“GORM + 手写 SQL 保底 + 独立子系统迁移”三种方式并存。

## 当前新增能力相关字段

### `feeds`

- `article_summary_enabled`
- `completion_on_refresh`
- `max_completion_retries`
- `firecrawl_enabled`

### `articles`

- `image_url`
- `summary_status`
- `summary_generated_at`
- `completion_attempts`
- `completion_error`
- `ai_content_summary`
- `firecrawl_status`
- `firecrawl_error`
- `firecrawl_content`
- `firecrawl_crawled_at`

## 常用命令

```bash
cd backend-go
go run cmd/server/main.go
go run cmd/migrate-digest/main.go
go run cmd/test-digest/main.go
```

## 说明

- 当前项目以 Go 后端为主
- 文档以当前 checkout 里的真实数据库逻辑为准
- 不再把不存在的历史后端当作正式运行依赖描述
- 数据库文档以 `docs/operations/database.md` 为准，不再依赖 `backend-go/DATABASE.md` 的旧叙述
