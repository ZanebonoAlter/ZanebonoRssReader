# Deferred Items

## 2026-04-11

- `front/app/features/topic-graph/components/TopicTimeline.test.ts` 在 `pnpm test:unit` 中仍然失败：`emits filter-change from header` 未发出预期事件。
- 该失败与本计划修改文件 (`front/app/api/scheduler.ts`, `front/app/stores/api.ts`) 无直接关系，本次未处理。
