# 04-01 前端 API 一致性与未读数同步

## Summary

- `front/app/api/scheduler.ts` 的 scheduler trigger 统一改为 `apiClient.post`，不再维护独立的 raw fetch 分支。
- `front/app/stores/api.ts` 新增未读数同步 helper，在 `updateArticle` 与 `markAllAsRead` 后同步更新 `feeds` / `allFeeds` 两套 feed 集合。
- 新增 `front/app/api/scheduler.test.ts` 与 `front/app/stores/api.test.ts`，覆盖 scheduler trigger、单篇已读、全部已读（含未分类）回归场景。

## Why It Matters

- `API-01` 要求前端 scheduler trigger 与其余 API 请求保持一致的错误处理和返回结构。
- `API-02` / `API-03` 要求 sidebar 未读数在单篇已读与全部已读场景下不再漂移，且未分类 feed 也要被正确覆盖。

## Verification

- `pnpm test:unit -- app/api/scheduler.test.ts`
- `pnpm test:unit -- app/stores/api.test.ts`
- `pnpm exec nuxi typecheck`
- `pnpm test:unit`（本计划相关测试通过，但存在与 topic graph 相关的既有失败）
