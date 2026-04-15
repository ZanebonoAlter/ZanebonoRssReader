---
type: execute
mode: quick
autonomous: true
files_modified:
  - backend-go/internal/domain/models/topic_graph.go
  - backend-go/internal/domain/topictypes/types.go
  - backend-go/internal/domain/topicextraction/quality_score.go
  - backend-go/internal/domain/topicextraction/quality_score_test.go
  - backend-go/internal/jobs/tag_quality_score.go
  - backend-go/internal/app/runtime.go
  - backend-go/internal/app/runtimeinfo/schedulers.go
  - backend-go/internal/jobs/handler.go
  - backend-go/internal/domain/topicgraph/service.go
  - backend-go/internal/domain/topicanalysis/abstract_tag_service.go
  - front/app/api/topicGraph.ts
  - front/app/api/abstractTags.ts
  - front/app/types/topicTag.ts
  - front/app/features/topic-graph/utils/buildTopicGraphViewModel.ts
  - front/app/features/topic-graph/utils/buildTopicGraphViewModel.test.ts
  - front/app/features/topic-graph/components/TopicGraphPage.vue
must_haves:
  truths:
    - "标签具备持久化 quality_score，且可被定时重算。"
    - "话题图谱与热点标签列表按 quality_score 排序，而不是继续依赖原始 LLM score。"
    - "低质量普通标签默认隐藏，但用户可以显式切换查看全部。"
    - "图谱节点视觉强调随 quality_score 变化，抽象标签不被默认低质过滤。"
  artifacts:
    - path: "backend-go/internal/domain/topicextraction/quality_score.go"
      provides: "质量分计算与批量回写"
    - path: "backend-go/internal/jobs/tag_quality_score.go"
      provides: "每小时调度与手动触发入口"
    - path: "front/app/features/topic-graph/components/TopicGraphPage.vue"
      provides: "低质量标签默认隐藏与切换控制"
  key_links:
    - from: "backend-go/internal/jobs/tag_quality_score.go"
      to: "backend-go/internal/domain/topicextraction/quality_score.go"
      via: "ComputeAllQualityScores"
      pattern: "topicextraction\\.ComputeAllQualityScores"
    - from: "backend-go/internal/domain/topicgraph/service.go"
      to: "topic_tags.quality_score"
      via: "API 排序与低质量标记"
      pattern: "QualityScore|quality_score|IsLowQuality"
    - from: "front/app/features/topic-graph/components/TopicGraphPage.vue"
      to: "front/app/api/topicGraph.ts"
      via: "quality_score / is_low_quality 消费"
      pattern: "quality_score|is_low_quality"
---

<objective>
实现 `docs/plans/2026-04-15-tag-quality-score-design.md` 和 `docs/plans/2026-04-15-tag-quality-score-implementation.md` 中的标签质量分方案，补齐后端持久化/调度/排序链路，以及前端图谱与热点标签的消费逻辑。

Purpose: 让标签展示和图谱强调基于客观质量信号，而不是区分度很弱的原始 LLM 置信分。
Output: 可定时计算的 `quality_score`、按质量排序/标记低质量的 API、以及前端默认隐藏低质量标签并支持显示全部的交互。
</objective>

<context>
@.planning/STATE.md
@AGENTS.md
@docs/plans/2026-04-15-tag-quality-score-design.md
@docs/plans/2026-04-15-tag-quality-score-implementation.md
@backend-go/internal/jobs/auto_tag_merge.go
@backend-go/internal/app/runtime.go
@backend-go/internal/jobs/handler.go
@backend-go/internal/domain/topicgraph/service.go
@backend-go/internal/domain/topicanalysis/abstract_tag_service.go
@front/app/api/topicGraph.ts
@front/app/features/topic-graph/utils/buildTopicGraphViewModel.ts
@front/app/features/topic-graph/components/TopicGraphPage.vue

<interfaces>
From `backend-go/internal/domain/models/topic_graph.go`:

```go
type TopicTag struct {
    ID        uint
    Slug      string
    Label     string
    Category  string
    FeedCount int
    Status    string
    IsWatched bool
    WatchedAt *time.Time
}
```

From `backend-go/internal/domain/topictypes/types.go`:

```go
type TopicTag struct {
    ID         uint
    Label      string
    Slug       string
    Category   string
    Score      float64
    IsAbstract bool
    ChildSlugs []string
}
```

From `front/app/api/topicGraph.ts`:

```ts
export interface TopicTag {
  id?: number
  label: string
  slug: string
  category: TopicCategory
  score: number
  is_abstract?: boolean
  child_slugs?: string[]
}
```
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: 建立 quality_score 数据模型、计算逻辑与定时调度</name>
  <files>backend-go/internal/domain/models/topic_graph.go, backend-go/internal/domain/topictypes/types.go, backend-go/internal/domain/topicextraction/quality_score.go, backend-go/internal/domain/topicextraction/quality_score_test.go, backend-go/internal/jobs/tag_quality_score.go, backend-go/internal/app/runtime.go, backend-go/internal/app/runtimeinfo/schedulers.go, backend-go/internal/jobs/handler.go</files>
  <behavior>
    - Test 1: percentile rank 在正常序列上返回 0-1 区间内的稳定结果。
    - Test 2: 当系统标签少于 3 个时，各维度走 0.5 默认值；当标签没有 article_topic_tags 关联时 quality_score 为 0。
    - Test 3: 调度器可像 auto_tag_merge 一样启动、手动触发、更新状态与重置统计。
  </behavior>
  <action>先补测试，再实现后端主链路。给 `models.TopicTag`、`topictypes.TopicTag`、`topicanalysis.TagHierarchyNode` 补 `QualityScore`；给对外标签类型补 `IsLowQuality`（普通标签 `quality_score < 0.3` 时为 true，抽象标签始终 false）。在 `quality_score.go` 实现可导出的 `ComputeAllQualityScores`：按设计稿计算频率分、共现分、来源分散度、语义匹配分；百分位归一化要处理空集合和少量标签场景（少于 3 个标签时维度默认 0.5），没有 article 关联的标签直接写 0；语义匹配在没有可用历史时默认 0.7；普通标签先算，抽象标签再基于 child 的 `QualityScore × ArticleCount` 加权平均。新增 `tag_quality_score` 调度器时复用 `auto_tag_merge` 的并发互斥、状态持久化、TriggerNow、ResetStats 和 runtime wiring 方式，并通过现有 `jobs/handler.go` 的统一调度器入口暴露手动触发能力，不要再加一套重复 handler。</action>
  <verify>
    <automated>cd backend-go &amp;&amp; go test ./internal/domain/topicextraction/... -run "TestPercentile|TestCompute" -v &amp;&amp; go build ./...</automated>
  </verify>
  <done>`topic_tags` 自动迁移出 `quality_score` 列；`ComputeAllQualityScores` 可被调度器调用；`tag_quality_score` 出现在统一 scheduler 状态/触发接口中；针对质量分核心公式的测试通过。</done>
</task>

<task type="auto">
  <name>Task 2: 让后端话题 API 按 quality_score 排序并显式标记低质量标签</name>
  <files>backend-go/internal/domain/topictypes/types.go, backend-go/internal/domain/topicgraph/service.go, backend-go/internal/domain/topicanalysis/abstract_tag_service.go</files>
  <action>更新 topic graph 与标签层级返回值，使 `quality_score` 和 `is_low_quality` 真正进入 API。`BuildTopicsByCategory` / `sortTagsByScoreMap` / 图谱 top topics 的最终展示顺序改为优先按 `QualityScore` 降序，原有 `Score` 只保留为原始置信或聚合信号，不再作为最终排序主键；`GetUnclassifiedTags` 改为 `quality_score DESC`；构造 `TagHierarchyNode` 时带上 `QualityScore`，以便前端可以消费，但不要把抽象标签误标成低质量。这里要保持现有 JSON snake_case 输出风格，且不要破坏现有 topic graph/abstract hierarchy 结构。</action>
  <verify>
    <automated>cd backend-go &amp;&amp; go build ./... &amp;&amp; go test ./internal/domain/topicgraph/... ./internal/domain/topicanalysis/... -count=1</automated>
  </verify>
  <done>图谱与热点标签 API 返回 `quality_score`；普通标签低于阈值会带 `is_low_quality: true`；后端排序主键切到 `quality_score`；抽象标签层级接口也能返回质量分。</done>
</task>

<task type="auto" tdd="true">
  <name>Task 3: 前端消费 quality_score，默认隐藏低质量标签并增强图谱视觉层级</name>
  <files>front/app/api/topicGraph.ts, front/app/api/abstractTags.ts, front/app/types/topicTag.ts, front/app/features/topic-graph/utils/buildTopicGraphViewModel.ts, front/app/features/topic-graph/utils/buildTopicGraphViewModel.test.ts, front/app/features/topic-graph/components/TopicGraphPage.vue</files>
  <behavior>
    - Test 1: `buildTopicGraphViewModel` 优先按 `quality_score` 排序 top topics，并在质量分较低时缩小/降低节点透明度。
    - Test 2: 低质量普通标签默认不出现在热点列表中，打开“显示全部标签”后可见。
    - Test 3: 抽象标签即使分数低也不会被默认过滤。
  </behavior>
  <action>补齐前端类型与映射：`TopicTag`、`TagHierarchyNode`、abstract tag API mapper 都要支持 `quality_score` 与 `is_low_quality`。更新 `buildTopicGraphViewModel`，对 topic node 的 `size`/视觉强调优先使用 `quality_score`（没有时再退回 weight），并把 `topTopics` 的排序改成质量分优先。随后在 `TopicGraphPage.vue` 给热点标签区域增加一个清晰的“显示低质量标签/显示全部标签”开关，默认只展示 `!is_low_quality || is_abstract` 的标签；搜索、show all、fallback topic 列表都必须走同一套过滤/排序逻辑，避免界面不同区域表现不一致。同步修改现有 view-model 测试，必要时补最小组件级断言，但不要扩散到无关 UI 重构。</action>
  <verify>
    <automated>cd front &amp;&amp; pnpm test:unit -- app/features/topic-graph/utils/buildTopicGraphViewModel.test.ts &amp;&amp; pnpm exec nuxi typecheck &amp;&amp; pnpm build</automated>
  </verify>
  <done>前端类型完整消费 `quality_score` / `is_low_quality`；热点标签默认隐藏低质量普通标签且可切换显示全部；图谱节点视觉强调与标签质量一致；相关单测、typecheck、build 通过。</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| scheduler → database | 定时任务批量写入 `topic_tags.quality_score`，错误 SQL 或空值处理会污染展示排序 |
| API → frontend | `quality_score` / `is_low_quality` 是新的显示控制信号，映射错误会导致误隐藏或错误排序 |
| manual trigger → scheduler runtime | 手动触发入口可能被重复点击，必须沿用互斥保护避免并发重算 |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-quick-01 | T | `quality_score.go` SQL/回写 | mitigate | 对空集合、少量标签、零 article 标签做显式分支；更新时按 tag id 精确写入 |
| T-quick-02 | D | `tag_quality_score` 手动触发 | mitigate | 复用 `TryLock` + `isExecuting` 模式，重复触发返回冲突而不是并发执行 |
| T-quick-03 | I | API/前端映射 | mitigate | 保持 snake_case API 输出，在 `abstractTags.ts` / `topicGraph.ts` 明确映射并用 typecheck + 单测兜底 |
| T-quick-04 | R | 排序切换后的行为回归 | mitigate | 用 targeted tests 固定 `quality_score` 优先排序和抽象标签不过滤规则 |
</threat_model>

<verification>
- Backend: `cd backend-go && go test ./internal/domain/topicextraction/... -run "TestPercentile|TestCompute" -v && go build ./...`
- Backend broader check: `cd backend-go && go test ./internal/domain/topicgraph/... ./internal/domain/topicanalysis/... -count=1`
- Frontend: `cd front && pnpm test:unit -- app/features/topic-graph/utils/buildTopicGraphViewModel.test.ts && pnpm exec nuxi typecheck && pnpm build`
</verification>

<success_criteria>
- `tag_quality_score` 调度器可以启动、查看状态并手动触发。
- `topic_tags.quality_score` 被正常写入，普通标签和抽象标签都有合理分值。
- 热点标签 / 图谱排序改为质量分优先，API 返回 `quality_score` 和 `is_low_quality`。
- 前端默认隐藏低质量普通标签，但用户可显式显示全部，且图谱节点视觉层次跟随质量分。
</success_criteria>

<output>
执行完成后，在当前 quick 目录补一份实现总结，记录实际修改文件、验证命令和任何与设计稿不一致但保持行为等价的实现选择。
</output>
