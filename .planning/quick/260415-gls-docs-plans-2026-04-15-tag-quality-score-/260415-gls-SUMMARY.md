# Quick Task 260415-gls Summary

## 结果概览

已落地标签 `quality_score` 方案的后端计算/调度链路、后端排序与低质量标记输出、以及前端图谱消费与默认过滤交互。

## 本次代码提交

1. `7ebbb69` — `feat(260415-gls): add quality score scheduler backbone`
2. `96f4eb8` — `feat(260415-gls): sort topic APIs by quality score`
3. `2510e8f` — `feat(260415-gls): consume quality score in topic graph UI`

## 实际修改

### Task 1
- `backend-go/internal/domain/models/topic_graph.go`
- `backend-go/internal/domain/topicextraction/quality_score.go`
- `backend-go/internal/domain/topicextraction/quality_score_test.go`
- `backend-go/internal/jobs/tag_quality_score.go`
- `backend-go/internal/jobs/tag_quality_score_test.go`
- `backend-go/internal/app/runtime.go`
- `backend-go/internal/app/runtimeinfo/schedulers.go`
- `backend-go/internal/jobs/handler.go`

### Task 2
- `backend-go/internal/domain/topictypes/types.go`
- `backend-go/internal/domain/topicgraph/service.go`
- `backend-go/internal/domain/topicanalysis/abstract_tag_service.go`

### Task 3
- `front/app/api/topicGraph.ts`
- `front/app/api/abstractTags.ts`
- `front/app/types/topicTag.ts`
- `front/app/features/topic-graph/utils/buildTopicGraphViewModel.ts`
- `front/app/features/topic-graph/utils/buildTopicGraphViewModel.test.ts`
- `front/app/features/topic-graph/utils/buildDisplayedTopicGraph.test.ts`
- `front/app/features/topic-graph/components/TopicGraphCanvas.client.vue`
- `front/app/features/topic-graph/components/TopicGraphPage.vue`

## 验证命令

### Backend
- `cd backend-go && go test ./internal/domain/topicextraction/... ./internal/jobs/... -run "TestPercentileRankStableRange|TestComputeAllQualityScoresDefaultsAndEmptyAssociations|TestTagQualityScoreSchedulerManualTriggerLifecycle" -count=1`
- `cd backend-go && go test ./internal/domain/topicextraction/... -run "TestPercentile|TestCompute" -v && go build ./...`
- `cd backend-go && go build ./... && go test ./internal/domain/topicgraph/... ./internal/domain/topicanalysis/... -count=1`

### Frontend
- `cd front && pnpm test:unit -- app/features/topic-graph/utils/buildTopicGraphViewModel.test.ts`
- `cd front && pnpm exec nuxi typecheck`
- `cd front && pnpm build`

## 关键实现说明

- 后端新增 `ComputeAllQualityScores`，按频率、共现、来源分散度、语义默认分计算普通标签质量分，并在第二阶段按子标签质量分加权回写抽象标签。
- `tag_quality_score` 复用了现有 scheduler 模式：支持启动、状态查询、手动触发、重置统计，并接入统一 scheduler handler。
- 话题图谱与按分类热点标签改为 `quality_score` 优先排序，普通标签通过 `is_low_quality` 显式标记，抽象标签默认不标低质量。
- 前端 `buildTopicGraphViewModel` 改为优先消费 `quality_score`，节点大小/透明度随质量分变化；质量分缺失时回退到旧的 `weight` 逻辑。
- 热点标签列表改为默认隐藏低质量普通标签，但用户仍可通过“显示全部标签”显式展开查看；抽象标签始终可见。

## 与设计稿的等价实现说明

- 语义匹配分当前采用“缺少历史时默认 0.7”的保守实现；由于现有库内没有单独的匹配历史表，先不引入额外表结构，保持和设计稿行为等价的默认回退。
- 图谱节点透明度落在 view-model，并由 `TopicGraphCanvas.client.vue` 消费；这是为了把视觉策略仍收敛在前端图谱层，而不是把展示细节塞回 API。

## Deviations from Plan

- 由于仓库已有大量无关脏工作树，本次按任务选择性暂存并提交，仅把本 quick task 相关代码纳入 3 个代码提交；未触碰 docs artifact commit。

## Known Stubs

- None.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: scheduler-write-path | `backend-go/internal/jobs/tag_quality_score.go` | 新增定时批量写入 `topic_tags.quality_score`，已通过互斥与精确按 tag id 更新控制风险 |
