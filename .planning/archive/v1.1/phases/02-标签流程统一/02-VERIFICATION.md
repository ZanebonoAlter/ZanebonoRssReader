---
phase: 02-标签流程统一
verified: 2026-04-11T06:42:36.4548985Z
status: gaps_found
score: 4/6 must-haves verified
overrides_applied: 0
gaps:
  - truth: "手动调用/articles/:id/tags API，tag_jobs表新增一条记录"
    status: failed
    reason: "手动打标签已改为 enqueue，但 handler 入队后只按 pending 状态回查 job；当已有 leased 任务或新任务被 worker 快速 claim 时，会把成功入队误报为 500，无法稳定返回 job_id。"
    artifacts:
      - path: "backend-go/internal/domain/articles/handler.go"
        issue: "227-231 行只查询 status=pending；239-241 行硬编码返回 pending。"
      - path: "backend-go/internal/domain/articles/handler_test.go"
        issue: "仅覆盖首次 happy-path 入队，没有覆盖已有 leased 任务再次 POST /tags 的场景。"
    missing:
      - "让 Enqueue 直接返回 job 记录或 job_id，避免二次回查"
      - "至少把回查条件扩展到 pending + leased，并返回真实状态"
      - "补充 leased/快速 claim 场景的回归测试"
  - truth: "标签处理完成后前端收到WebSocket通知"
    status: failed
    reason: "后端已广播 tag_completed，但前端代码中没有任何 tag_completed 消费方，因此“前端收到通知”这一 must-have 不成立。"
    artifacts:
      - path: "backend-go/internal/domain/topicextraction/tag_queue.go"
        issue: "284-310 行只完成后端广播链路。"
      - path: "front/app/features/summaries/composables/useSummaryWebSocket.ts"
        issue: "18-27 行仅声明 progress 消息；79-84 行收到消息后直接按 Summary WSMessage 解析，没有 tag_completed 分支。"
    missing:
      - "新增前端 tag_completed WebSocket 消费逻辑"
      - "或在 VERIFICATION frontmatter 中添加 override，明确该前端接线被接受为本 phase 外工作"
---

# Phase 2: 标签流程统一 Verification Report

**Phase Goal:** 所有标签提取走统一队列（TagJobQueue），无绕过；手动打标签 API 异步入队；TagQueue 启动失败后台重试且不阻塞应用启动。
**Verified:** 2026-04-11T06:42:36.4548985Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Firecrawl完成后，文章标签通过TagJobQueue生成 | ✓ VERIFIED | `backend-go/internal/jobs/firecrawl.go:242-252` 在 crawl 完成后调用 `NewTagJobQueue(database.DB).Enqueue(...)`，`Reason: "firecrawl_completed"`，不再直接调用 `RetagArticle`。 |
| 2 | ContentCompletion完成后，文章标签通过TagJobQueue生成 | ✓ VERIFIED | `backend-go/internal/domain/contentprocessing/content_completion_service.go:198-205` 在 summary 完成后调用 `NewTagJobQueue(database.DB).Enqueue(...)`，`Reason: "summary_completed"`。 |
| 3 | 手动调用/articles/:id/tags API，tag_jobs表新增一条记录 | ✗ FAILED | `backend-go/internal/domain/articles/handler.go:216-223` 已 enqueue，但 `227-231` 只按 `status = pending` 回查；若已有 leased 任务或新任务被快速 claim，会进入 `failed to retrieve job_id` 返回 500。现有测试 `TestRetagArticleReturnsUpdatedTags` 仅覆盖首次 happy-path。 |
| 4 | TagQueue启动失败后后台重试且不阻塞应用启动 | ✓ VERIFIED | `backend-go/internal/domain/topicextraction/tag_queue.go:73-92` 首轮失败后 `go q.backgroundRetry()` 并立即 `return nil`；`119-149` 实现 30 秒间隔、最多 10 次的后台重试；`backend-go/internal/app/runtime.go:29-34` 启动时接入 `GetTagQueue().Start()`。 |
| 5 | 同一文章同时触发多个标签任务，最终只生成一套标签 | ✓ VERIFIED | `backend-go/internal/domain/topicextraction/tag_job_queue.go:33-55` 对 `pending/leased` 活跃任务做事务内复用/升级；`tag_job_queue_test.go:27-50` 的 `TestEnqueueTagJobUpgradesForceRetag` 验证第二次 enqueue 不会新增第二条 job；`article_tagger.go:34-45` 还会清理旧关联并跳过已存在标签。 |
| 6 | 标签处理完成后前端收到WebSocket通知 | ✗ FAILED | 后端已在 `backend-go/internal/domain/topicextraction/tag_queue.go:284-310` 广播 `tag_completed`；但 `front/app/features/summaries/composables/useSummaryWebSocket.ts:18-27,79-84` 仅处理 `progress` 消息，`front/app` 下 `grep "tag_completed"` 无匹配。 |

**Score:** 4/6 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `backend-go/internal/domain/articles/handler.go` | RetagArticleHandler 异步入队并返回 job 元数据 | ⚠ PARTIAL | 已通过 `TagJobQueue.Enqueue` 入队并返回 `job_id` happy-path；但 `227-231` 仅查 `pending`，leased/快速 claim 场景会误报 500。 |
| `backend-go/internal/domain/topicextraction/tag_queue.go` | 非阻塞启动重试 + 完成后广播 tag_completed | ✓ VERIFIED | `Start/tryStart/backgroundRetry` 与 `broadcastTagCompleted` 均已接通。 |
| `backend-go/internal/platform/ws/hub.go` | 定义 `TagCompletedMessage` 契约 | ✓ VERIFIED | `84-99` 定义 `TagCompletedMessage` / `TagCompletedItem`。 |
| `front/app/features/summaries/composables/useSummaryWebSocket.ts` | 消费 `tag_completed` 通知并让前端真正收到完成事件 | ✗ MISSING | 仅声明/解析 summary `progress` 消息，没有 `tag_completed` 分支。 |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `backend-go/internal/domain/articles/handler.go` | `TagJobQueue.Enqueue` | `queue.Enqueue(topicextraction.TagJobRequest{...})` | ✓ WIRED | `handler.go:216-223` 明确通过 queue enqueue，`grep "RetagArticle\(" backend-go/internal/domain/articles/handler.go` 无匹配。 |
| `backend-go/internal/domain/topicextraction/tag_queue.go` | `ws.GetHub().BroadcastRaw` | `broadcastTagCompleted()` | ✓ WIRED | `tag_queue.go:284-310` 通过 `ws.GetHub()` → `json.Marshal` → `hub.BroadcastRaw(data)`。 |
| `backend-go/internal/app/runtime.go` | `TagQueue.Start()` | `topicextraction.GetTagQueue().Start()` | ✓ WIRED | `runtime.go:29-34` 在应用启动时接入 TagQueue。 |
| `tag_completed` WebSocket 事件 | 前端消费方 | WebSocket message handling | ✗ NOT_WIRED | `front/app` 无 `tag_completed` 消费；现有 `useSummaryWebSocket` 只支持 `progress`。 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| --- | --- | --- | --- | --- |
| `backend-go/internal/domain/articles/handler.go` | `job_id` / `status` | `database.DB.Where("article_id = ? AND status = ?", article.ID, pending).First(&tagJob)` | Partial | ⚠ HOLLOW — happy-path 有真实数据，但 leased/快速 claim 场景会查不到，导致成功入队后仍返回 500。 |
| `backend-go/internal/domain/topicextraction/tag_queue.go` | `tags` | `GetArticleTags(job.ArticleID)` after `TagArticle/RetagArticle` + `MarkCompleted` | Yes | ✓ FLOWING |
| `backend-go/internal/jobs/firecrawl.go` | `TagJobRequest{ArticleID, FeedName, ForceRetag, Reason}` | crawl 完成后的文章/订阅数据 | Yes | ✓ FLOWING |
| `backend-go/internal/domain/contentprocessing/content_completion_service.go` | `TagJobRequest{ArticleID, FeedName, ForceRetag, Reason}` | summary 完成后的文章/订阅数据 | Yes | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| 手动打标签 happy-path 返回 job_id 并写入 tag_jobs | `go test ./internal/domain/articles -run TestRetagArticleReturnsUpdatedTags -v` | PASS | ✓ PASS |
| `tag_completed` WebSocket 负载字段契约 | `go test ./internal/platform/ws -run TestTagCompletedMessageMarshal -v` | PASS | ✓ PASS |
| 同文章重复 enqueue 复用活跃 job | `go test ./internal/domain/topicextraction -run TestEnqueueTagJobUpgradesForceRetag -v` | PASS | ✓ PASS |
| 后端相关包可编译 | `go build ./...` | PASS | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| TAG-03 | `02-01-PLAN.md` | 手动打标签API `/articles/:id/tags` 改为通过TagJobQueue enqueue，保持流程一致 | ✓ SATISFIED | `backend-go/internal/domain/articles/handler.go:216-223` 已改为 `TagJobQueue.Enqueue(...)`，不再直接调用 `RetagArticle`。 |
| TAG-04 | `02-01-PLAN.md` | TagQueue启动失败时自动重试，不永久卡在"not started"状态 | ✓ SATISFIED | `backend-go/internal/domain/topicextraction/tag_queue.go:73-149` 已实现首轮失败后立即返回 + 后台重试。 |
| TAG-01 | —（ORPHANED） | Firecrawl完成后统一通过TagJobQueue异步enqueue标签任务，不直接调用RetagArticle | ✓ SATISFIED / ORPHANED | `backend-go/internal/jobs/firecrawl.go:242-252`。该 requirement 属于 Phase 2 traceability，但未出现在任何 PLAN frontmatter 的 `requirements`。 |
| TAG-02 | —（ORPHANED） | ContentCompletion完成后统一通过TagJobQueue异步enqueue标签任务，不直接调用RetagArticle | ✓ SATISFIED / ORPHANED | `backend-go/internal/domain/contentprocessing/content_completion_service.go:198-205`。该 requirement 未被 PLAN frontmatter 声明。 |
| TAG-05 | —（ORPHANED） | TagArticle增加幂等检查，防止同时多个标签任务重复处理同一文章 | ✓ SATISFIED / ORPHANED | `backend-go/internal/domain/topicextraction/tag_job_queue.go:33-55` + `tag_job_queue_test.go:27-50`。该 requirement 未被 PLAN frontmatter 声明。 |
| TAG-06 | —（ORPHANED） | RetagArticle完成时清理文章现有标签关联，防止残留 | ✓ SATISFIED / ORPHANED | `backend-go/internal/domain/topicextraction/article_tagger.go:34-38` 在 force retag 时先删旧 `ArticleTopicTag`。该 requirement 未被 PLAN frontmatter 声明。 |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `backend-go/internal/domain/articles/handler.go` | 227-241 | 入队后二次回查仅限 `pending`，并硬编码返回 `pending` | 🛑 Blocker | 活跃 `leased` 任务或被快速 claim 的新任务会让 API 误报 500，破坏手动打标签的稳定异步契约。 |
| `docs/api/articles.md` | 72 | 文档要求“监听 tag_completed 或轮询 job 状态”，但代码库没有 tag job 查询 API | ⚠️ Warning | 失败态没有可用回传闭环，客户端可能一直等不到终态。 |
| `backend-go/internal/app/runtime.go` | 30-34 | `Start()` 首轮失败也返回 nil，runtime 仍统一打印“Tag queue started successfully” | ⚠️ Warning | 启动日志会在“后台重试中”场景下产生误导，不利于运维判断。 |

### Human Verification Required

### 1. TagQueue 启动失败后的真实恢复链路

**Test:** 在本地让数据库连接/`tag_jobs` 可用性在启动瞬间失败，再恢复依赖，观察应用是否先正常启动、随后 TagQueue 自动恢复。
**Expected:** 应用主进程不被阻塞；日志先出现 initial start failed / retry attempt，依赖恢复后出现 started after N retry attempts。
**Why human:** 需要真实运行时注入启动故障并观察日志时序，静态代码与单测无法证明。 

### 2. 手动打标签端到端 WebSocket 完成体验

**Test:** 启动后端与前端，调用 `POST /api/articles/:id/tags`，同时连接 `/ws`，确认前端页面或调试控制台是否真正消费 `tag_completed`。
**Expected:** 后端广播后，前端有明确的完成态处理，而不只是网络层收到原始消息。
**Why human:** 当前自动化只证明后端能广播；“前端收到并使用”涉及运行中的 UI / WS 交互。 

### Gaps Summary

Phase 02 的核心后端方向大体成立：Firecrawl、ContentCompletion、手动 API 都已走 `TagJobQueue`，TagQueue 也具备“首次失败立即返回、后台重试”的非阻塞启动骨架，TAG-01/02/03/04/05/06 在代码层大多都能找到对应实现。

但从“目标倒推”的角度，仍有两个会阻断结论为 `passed` 的缺口：

1. **手动打标签 API 的异步契约不稳定。** `RetagArticleHandler` 入队后只回查 `pending` job，导致已成功入队但被 worker 抢占为 `leased` 的场景可能误报 500。这意味着“手动 API 稳定异步入队并返回 job_id”并未真正闭环。
2. **Plan 声明的“前端收到 WebSocket 通知”没有实现。** 后端只完成了广播侧，前端没有消费 `tag_completed`。这看起来是有意把前端接线留到后续；若团队接受该偏差，应在 VERIFICATION frontmatter 增加 override，否则该 must-have 仍算失败。

可接受的 override 建议（仅针对第 2 项 intentional deviation）：

```yaml
overrides:
  - must_have: "标签处理完成后前端收到WebSocket通知"
    reason: "本 phase 只交付后端异步契约与广播能力，前端消费逻辑明确留到后续工作"
    accepted_by: "{name}"
    accepted_at: "2026-04-11T06:42:36Z"
```

在修复第 1 项之前，本 phase 仍不能判定为 passed。

---

_Verified: 2026-04-11T06:42:36.4548985Z_
_Verifier: the agent (gsd-verifier)_
