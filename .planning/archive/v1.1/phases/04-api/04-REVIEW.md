---
phase: 04-api
reviewed: 2026-04-11T10:45:00Z
depth: standard
files_reviewed: 14
files_reviewed_list:
  - front/app/api/scheduler.ts
  - front/app/api/scheduler.test.ts
  - front/app/stores/api.ts
  - front/app/stores/api.test.ts
  - front/vitest.config.ts
  - backend-go/internal/jobs/handler.go
  - backend-go/internal/jobs/auto_refresh.go
  - backend-go/internal/jobs/auto_summary.go
  - backend-go/internal/jobs/content_completion.go
  - backend-go/internal/jobs/firecrawl.go
  - backend-go/internal/jobs/preference_update.go
  - backend-go/internal/jobs/scheduler_status_response_test.go
  - backend-go/internal/jobs/content_completion_test.go
  - backend-go/internal/jobs/handler_test.go
findings:
  critical: 0
  warning: 2
  info: 0
  total: 2
status: issues_found
---

# Phase 04: Code Review

## Summary

审查了 Phase 04 的前后端 scheduler API 统一改动，以及配套测试。后端五字段契约本身实现一致，但当前变更把已有前端消费契约一起收窄了，导致 Scheduler 设置页出现明显兼容性回归。

## Findings

### WR-01 `name` 从内部标识改成展示名，前端现有分支逻辑和后续请求会直接失效

- file: `backend-go/internal/jobs/handler.go:20-26,37-91`
- file: `backend-go/internal/jobs/auto_refresh.go:290-304`
- file: `backend-go/internal/jobs/auto_summary.go:860-874`
- file: `backend-go/internal/jobs/content_completion.go:449-467`
- file: `backend-go/internal/jobs/firecrawl.go:337-345`
- file: `backend-go/internal/jobs/preference_update.go:167-173`
- file: `front/app/components/dialog/GlobalSettingsDialog.vue:337-345,402-403,1126-1141,1193-1229`
- file: `front/app/utils/schedulerMeta.ts:3-29,59-82`
- issue: 这次后端把 status payload 里的 `name` 统一写成展示名，例如 `Auto Refresh`、`Content Completion`、`Firecrawl Crawler`。但前端当前把 `name` 当作稳定 slug 使用：轮询热任务判断依赖 `isHotScheduler(item.name)`，内容补全面板依赖 `isContentCompletionScheduler(scheduler.name)`，自动总结/后台刷新卡片依赖 `scheduler.name === 'auto_summary'/'auto_refresh'`，按钮点击又直接 `triggerScheduler(scheduler.name)`，反馈读取也用 `getSchedulerFeedback(scheduler.name)`。一旦返回展示名，这些逻辑会同时失效。
- impact: Scheduler 面板会出现至少四类回归：1) `手动执行` 按钮把展示名当路由参数发给 `/schedulers/:name/trigger`，后端返回 404；2) 触发反馈按 slug 存、按展示名读，反馈条不会显示；3) `isHotScheduler()` 不再识别热任务，轮询退回慢速分支；4) 自动总结、后台刷新、内容补全面板条件判断失效，详细卡片不再渲染。
- fix: 保持 API 中的 `name` 为稳定 slug，把展示名放到独立字段，例如 `display_name`；或者在前端 API 层把展示名映射回 slug，但这会让触发接口、反馈缓存、图标/颜色映射都继续依赖额外转换，复杂度更高。

### WR-02 status 接口被收窄为五字段后，现有设置页依赖的扩展数据全部丢失

- file: `backend-go/internal/jobs/handler.go:108-158,362-395`
- file: `backend-go/internal/jobs/handler_test.go:244-309`
- file: `backend-go/internal/jobs/content_completion.go:470-540`
- file: `front/app/components/dialog/GlobalSettingsDialog.vue:398-403,1144-1257,1266-1416`
- file: `front/app/types/scheduler.ts:86-106`
- issue: `GetSchedulersStatus`/`GetSchedulerStatus` 现在只透传 `SchedulerStatusResponse` 五个字段，并且测试还断言返回键必须精确等于 `check_interval/is_executing/name/next_run/status`。但前端当前的 scheduler 面板明显依赖 richer payload：`scheduler.database_state` 用于执行次数/成功率/上次执行时间，`last_run_summary` 用于后台刷新和自动总结摘要，`overview/current_article/stale_processing_article` 用于内容补全面板，`ai_configured` 用于自动总结状态提示。`content_completion.go` 虽然实现了 `GetTaskStatusDetails()`，但 `GetSchedulersStatus` 根本没有把这些 details 合并回响应，所以前端拿不到任何扩展数据。
- impact: Scheduler 设置页列表虽然还能显示基础状态，但详情区会整体退化：统计卡、最近运行摘要、内容补全实时文章、AI 配置状态都无法显示；`shouldShowContentCompletionPanel()` 也会因为 `overview/current_article` 缺失而常驻返回 `false`。
- fix: 两种修复方向二选一即可闭环：1) 保持现有 `/schedulers/status` 契约向后兼容，在统一五字段的基础上继续附带已有扩展字段；2) 新增专门的 detail endpoint，并同步改前端为“列表接口 + detail 接口”模式。当前代码和测试都还没有完成这次契约迁移的前后端联动，所以不能只改后端 shape。

## Test Notes

- 本次为静态代码审查，未额外执行自动化测试。
- 现有后端测试主要验证了统一后的五字段 shape，但没有覆盖前端兼容性，因此上述回归不会被当前测试集捕获。
