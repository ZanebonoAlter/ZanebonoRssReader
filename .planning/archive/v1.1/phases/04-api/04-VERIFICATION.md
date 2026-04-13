---
phase: 04-api
verified: 2026-04-11T09:40:36Z
status: gaps_found
score: 3/5 must-haves verified
overrides_applied: 0
gaps:
  - truth: "所有scheduler status API返回相同字段结构"
    status: failed
    reason: "`/api/schedulers/status` 与 `/api/schedulers/:name/status` 已统一，但 `/api/digest/status` 仍返回 legacy digest 字段（running/daily_enabled/next_runs 等），缺少 `name/status/check_interval/next_run/is_executing` 契约。"
    artifacts:
      - path: "backend-go/internal/domain/digest/handler.go"
        issue: "`GetDigestStatus` 仍直接返回 `DigestSchedulerInterface.GetStatus() map[string]interface{}`。"
      - path: "backend-go/internal/domain/digest/scheduler.go"
        issue: "`DigestScheduler.GetStatus()` 仍返回 legacy map，字段结构与统一 scheduler status 契约不一致。"
    missing:
      - "让 digest status endpoint 复用统一的 `SchedulerStatusResponse` 契约，至少补齐 `name/status/check_interval/next_run/is_executing`。"
      - "补充 digest status 的统一契约测试，覆盖 `/api/digest/status`。"
  - truth: "Field types / semantics are consistent across all scheduler status responses"
    status: failed
    reason: "后端把 `name` 改成展示名（如 `Auto Refresh`）并把 `next_run` 改成 Unix 秒时间戳，但前端仍把 `scheduler.name` 当 slug 使用、把 `next_run` 当字符串时间处理，导致 scheduler UI 的触发、图标/热状态判断、下次执行时间展示与真实 API 契约脱节。"
    artifacts:
      - path: "backend-go/internal/jobs/handler.go"
        issue: "`SchedulerStatusResponse.Name` 当前承载展示名，而前端现有消费逻辑依赖 slug。"
      - path: "front/app/components/dialog/GlobalSettingsDialog.vue"
        issue: "`triggerScheduler(scheduler.name)` 会把展示名当作路由参数；`formatNextRun(nextRun: string | null | undefined)` 也把 `next_run` 当字符串解析。"
      - path: "front/app/utils/schedulerMeta.ts"
        issue: "`isHotScheduler/getSchedulerDisplayName/getSchedulerIcon` 都按 slug（`auto_refresh`/`firecrawl`）判断，无法匹配展示名。"
      - path: "front/app/types/scheduler.ts"
        issue: "`next_run` 类型仍声明为 `string | null`，与后端 `int64` 秒时间戳不一致。"
    missing:
      - "统一 scheduler 标识语义：要么后端返回稳定 slug 并另给 display label，要么前端全面改为消费展示名 + 独立 id。"
      - "统一 `next_run` 类型与格式化逻辑（RFC3339 或 Unix 秒二选一），并补充前后端联调测试。"
---

# Phase 4: API规范化 Verification Report

**Phase Goal:** 前端API调用一致，状态同步正确
**Verified:** 2026-04-11T09:40:36Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | 前端调用 scheduler trigger 使用统一的 apiClient | ✓ VERIFIED | `front/app/api/scheduler.ts:5-7,19-24` 统一走 `apiClient.post/get`；`front/app/api/scheduler.test.ts:24-43` 断言未调用 raw `fetch`；`pnpm test:unit -- app/api/scheduler.test.ts` 通过。 |
| 2 | 标记文章已读后，sidebar feed unread count 正确减少 | ✓ VERIFIED | `front/app/stores/api.ts:224-235,289-307` 通过 `syncFeedUnreadCount` 同步 `feeds/allFeeds`；已读时 `Math.max(0, current - 1)`，未读时 `current + 1`；`front/app/stores/api.test.ts:83-100` 与 `pnpm test:unit -- app/stores/api.test.ts` 通过。 |
| 3 | “全部标记已读”后，所有 feed（包括未分类）的 unread count 清零 | ✓ VERIFIED | `front/app/stores/api.ts:238-251,329-369` 用 `clearFeedUnreadCounts` 覆盖全量/feed/category/uncategorized 四种路径；`front/app/stores/api.test.ts:102-128` 覆盖未分类场景并通过。 |
| 4 | 所有 scheduler status API 返回相同字段结构 | ✗ FAILED | `backend-go/internal/jobs/handler.go:134-162` 已统一 `/api/schedulers/*`；但 `backend-go/internal/app/router.go:171` 仍暴露 `/api/digest/status`，而 `backend-go/internal/domain/digest/handler.go:607-640` + `backend-go/internal/domain/digest/scheduler.go:343-377` 返回 legacy digest 结构。 |
| 5 | Scheduler status 字段类型/语义在后端与前端消费侧保持一致 | ✗ FAILED | 后端 `SchedulerStatusResponse` 把 `name` 作为展示名、`next_run` 作为 `int64`（`backend-go/internal/jobs/handler.go:20-25`）；前端仍以 slug 语义消费 `name`（`front/app/utils/schedulerMeta.ts:3-29`、`front/app/components/dialog/GlobalSettingsDialog.vue:1107-1129,1193-1236`），并把 `next_run` 当字符串时间（`GlobalSettingsDialog.vue:508-518`、`front/app/types/scheduler.ts:86-106`）。 |

**Score:** 3/5 truths verified

### Must-Have Alignment Audit

PLAN frontmatter 与真实代码存在多处漂移，需与代码事实区分：

- `04-01-PLAN.md` 的 artifact/key_link 断言不是代码现状：
  - 计划写的是 `~/api/client` 导入，但实际是相对导入 `./client`（`front/app/api/scheduler.ts:3`）。
  - 计划写的是 `updateUnreadCount()` helper，但实际实现是 `syncFeedUnreadCount()` / `clearFeedUnreadCounts()`（`front/app/stores/api.ts:224-251`）。
  - 计划引用的 `front/app/features/articles/components/ArticleContentActions.vue` 文件不存在；真实调用点在 `FeedLayoutShell.vue` / `AppSidebarView.vue`。
- `04-02-PLAN.md` 预期 `backend-go/internal/jobs/*_status.go`，实际统一逻辑落在现有 scheduler 文件与 `handler.go` 中；这是计划描述漂移，不是实现文件缺失。
- `04-02-SUMMARY.md` 声称“前端可用同一解析逻辑消费”与当前代码不符：`GlobalSettingsDialog.vue` 仍按旧 slug / string 时间语义消费 status 数据，未完成真正的前后端契约对齐。

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `front/app/api/scheduler.ts` | scheduler trigger/status API 统一走 `apiClient` | ✓ VERIFIED | 文件存在且无 raw `fetch`；计划里的导入路径字符串已过时，但目标达成。 |
| `front/app/stores/api.ts` | 已读/全部已读时同步本地 unread count | ✓ VERIFIED | 文件存在、实现非 stub；实际 helper 名与 PLAN 不同，但行为满足 API-02/API-03。 |
| `backend-go/internal/jobs/handler.go` | `/api/schedulers/*` 统一 status 响应 | ⚠️ PARTIAL | 统一 struct 已建立，但只覆盖 jobs handler 路由；未覆盖 digest 独立 status endpoint。 |
| `backend-go/internal/domain/digest/handler.go` | digest status 也应遵守统一 scheduler status 契约 | ✗ FAILED | 仍返回 legacy digest map。 |
| `backend-go/internal/domain/digest/scheduler.go` | digest runtime status 也应输出统一字段 | ✗ FAILED | `GetStatus()` 返回 `running/daily_enabled/next_runs/...`，不含统一五字段。 |
| `front/app/components/dialog/GlobalSettingsDialog.vue` | 前端可按统一契约正确消费 scheduler status | ✗ FAILED | 仍把 `name` 当 slug、`next_run` 当字符串，和后端新契约不一致。 |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `front/app/api/scheduler.ts` | `front/app/api/client.ts` | `apiClient.get/post` | ✓ WIRED | `scheduler.ts:3,5-24` 已实际依赖 `apiClient`。 |
| `front/app/features/shell/components/FeedLayoutShell.vue` | `front/app/stores/api.ts` | `apiStore.markAsRead/markAllAsRead(...)` | ✓ WIRED | `FeedLayoutShell.vue:136,368-379` 调用 store，状态同步链路存在。 |
| `front/app/features/shell/components/AppSidebarView.vue` | `front/app/stores/api.ts` | `apiStore.markAllAsRead({ feedId })` | ✓ WIRED | `AppSidebarView.vue:110-118` 直接触发批量已读。 |
| `backend-go/internal/jobs/handler.go` | scheduler implementations | `safeGetStatus()` + `GetStatus()` interface | ✓ WIRED | `handler.go:108-162` 正确聚合 auto_refresh/auto_summary/content_completion/preference_update/firecrawl。 |
| `backend-go/internal/domain/digest/handler.go` | `backend-go/internal/domain/digest/scheduler.go` | `DigestSchedulerInterface.GetStatus()` | ✗ NOT_WIRED (to unified contract) | 连通了，但仍是 legacy digest schema，不是统一 scheduler status schema。 |
| `backend-go/internal/jobs/handler.go` | `front/app/components/dialog/GlobalSettingsDialog.vue` | scheduler status JSON contract | ✗ CONTRACT_MISMATCH | 后端返回展示名+Unix 秒；前端按 slug+字符串时间消费，导致触发/展示逻辑失配。 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| --- | --- | --- | --- | --- |
| `front/app/stores/api.ts` | `feed.unreadCount` | `updateArticle()` / `markAllAsRead()` 成功后本地同步 | Yes | ✓ FLOWING |
| `front/app/components/dialog/GlobalSettingsDialog.vue` | `schedulerStatuses` | `useSchedulerApi().getSchedulersStatus()` → `backend-go/internal/jobs/handler.go:GetSchedulersStatus` | Partial | ⚠️ HOLLOW — 数据到达了，但 `name` / `next_run` 语义与前端消费不一致 |
| `backend-go/internal/domain/digest/handler.go` | `data` | `DigestScheduler.GetStatus()` | No | ✗ DISCONNECTED from unified contract |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| scheduler trigger helper 走 `apiClient` | `pnpm test:unit -- app/api/scheduler.test.ts` | 1 file, 1 test passed | ✓ PASS |
| article unread count 本地同步 | `pnpm test:unit -- app/stores/api.test.ts` | 1 file, 2 tests passed | ✓ PASS |
| jobs status handler 统一五字段 | `go test ./internal/jobs -run "TestSchedulerStatusFormat\|TestGetSchedulerStatusReturnsUnifiedResponseShape\|TestGetSchedulersStatusIncludesPreferenceUpdateAndDigest\|TestGetSchedulerStatusAliasUsesSameUnifiedShape" -v` | targeted Go tests passed | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| API-01 | `04-01-PLAN.md` | Scheduler trigger API统一使用apiClient，不直接使用fetch | ✓ SATISFIED | `front/app/api/scheduler.ts:5-24`；`scheduler.test.ts` 断言 `fetch` 未被调用。 |
| API-02 | `04-01-PLAN.md` | UpdateArticle成功后刷新相关feed的unreadCount，防止前端计数漂移 | ✓ SATISFIED | `front/app/stores/api.ts:305-307` 同步 `feeds/allFeeds`，并含非负保护。 |
| API-03 | `04-01-PLAN.md` | MarkAllAsRead的本地状态更新覆盖所有边界情况（未分类、空分类等） | ✓ SATISFIED | `front/app/stores/api.ts:342-369` 覆盖全量/feed/category/uncategorized；测试覆盖未分类。 |
| API-04 | `04-02-PLAN.md` | 所有scheduler status API返回格式一致（包含name, status, check_interval, next_run, is_executing） | ✗ BLOCKED | `/api/digest/status` 未统一；且后端新契约与 `GlobalSettingsDialog.vue` 的消费语义不一致。 |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `front/app/components/dialog/GlobalSettingsDialog.vue` | 508-518 | `formatNextRun(nextRun: string | null | undefined)` 仍按字符串时间解析 | 🛑 Blocker | 后端已返回 Unix 秒时间戳，UI 会把未来任务误判成“即将执行”或显示错误时间。 |
| `front/app/components/dialog/GlobalSettingsDialog.vue` | 1129 | `triggerScheduler(scheduler.name)` | 🛑 Blocker | `scheduler.name` 已被后端改成展示名时，trigger 路由参数不再稳定。 |
| `front/app/utils/schedulerMeta.ts` | 3-29 | 依赖 slug（`auto_refresh` / `auto_summary` / `firecrawl`）做展示/热状态判断 | ⚠️ Warning | 与后端 `name` 展示名语义冲突，导致图标、热轮询、摘要卡片条件判断失效。 |

### Gaps Summary

Phase 04 的前端 API 调用统一与 unread count 同步部分已经真实落地；API-01 / API-02 / API-03 有代码与测试双重证据支持。

但 Phase 04 的核心目标并未完全达成，因为 scheduler status 契约只在一半链路上被统一：

1. **后端仍残留未统一的 digest status API。** `/api/digest/status` 还是旧字段集合，不满足 REQUIREMENTS / ROADMAP 对“所有 scheduler status API 一致”的要求。
2. **前后端对 status 字段语义没有真正对齐。** 后端把 `name` 改成展示名、把 `next_run` 改成 Unix 秒；前端设置面板仍把 `name` 当 slug、把 `next_run` 当字符串时间。SUMMARY 把这部分描述成“前端可统一解析”，但真实代码并不支持这一点。

换句话说：**Phase 04 完成了部分任务，但没有完全实现“状态同步正确”的目标。** 当前最明显的风险点是 scheduler 设置页的触发与状态展示链路。

---

_Verified: 2026-04-11T09:40:36Z_
_Verifier: the agent (gsd-verifier)_
