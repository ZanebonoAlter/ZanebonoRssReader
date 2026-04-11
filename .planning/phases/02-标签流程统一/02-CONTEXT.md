# Phase 2: 标签流程统一 - Context

**Gathered:** 2026-04-11
**Status:** Ready for planning

<domain>
## Phase Boundary

所有标签提取走统一队列（TagJobQueue），无绕过。

**实际修复范围（基于代码分析）：**

| Requirement | 当前状态 | 需修复 |
|-------------|---------|--------|
| TAG-01 | firecrawl.go:242-252 已正确使用TagJobQueue.Enqueue | 否 |
| TAG-02 | content_completion_service.go:198-205 已正确使用TagJobQueue.Enqueue | 否 |
| TAG-03 | articles/handler.go:216 直接调用RetagArticle绕过队列 | **是** |
| TAG-04 | tag_queue.go:70-86 Start失败后无自动恢复 | **是** |
| TAG-05 | tag_job_queue.go:34-59 transaction幂等检查已存在 | 否 |
| TAG-06 | article_tagger.go:34-38 RetagArticle已清理旧标签 | 否 |

**Phase只修复TAG-03和TAG-04。**

</domain>

<decisions>
## Implementation Decisions

### TAG-03: 手动打标签API改造
- **D-01:** `/articles/:id/tags` API改为异步enqueue到TagJobQueue，不再直接调用RetagArticle
- **D-02:** API立即返回job_id，前端可通过job_id查询或监听WebSocket获取结果
- **D-03:** 新增WebSocket消息类型 `tag_completed`，TagQueue处理完成后广播通知前端
- **D-04:** WebSocket消息格式：`{type: "tag_completed", article_id: N, job_id: "...", tags: [...]}`

### TAG-04: TagQueue启动重试机制
- **D-05:** 启动失败后后台goroutine定时轮询重试，不阻塞应用启动
- **D-06:** 重试间隔：30秒
- **D-07:** 最大重试次数：10次（约5分钟后放弃）
- **D-08:** 每次重试记录日志 `[INFO] TagQueue retry attempt N/10`
- **D-09:** 成功后记录 `[INFO] TagQueue started after N retry attempts`

### Agent's Discretion
- WebSocket消息字段细节（是否包含tag_count等）
- job查询API路径设计（如 `/api/articles/:id/tags/job/:jobId`）
- 重试goroutine与TagQueue singleton的同步机制

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements
- `.planning/REQUIREMENTS.md` §TAG-01~06 — 标签流程需求定义（注意：部分已正确实现）

### Source Files (Affected)
- `backend-go/internal/domain/articles/handler.go:200-241` — 手动打标签API实现（需改造）
- `backend-go/internal/domain/topicextraction/tag_queue.go:70-86` — TagQueue Start()方法（需添加重试）
- `backend-go/internal/domain/topicextraction/tag_job_queue.go` — TagJobQueue实现（参考Enqueue格式）
- `backend-go/internal/platform/ws/hub.go` — WebSocket Hub（参考BroadcastRaw用法）
- `backend-go/internal/jobs/firecrawl.go:311-329` — WebSocket broadcastProgress参考实现

### Reference (Already Correct - Verify Only)
- `backend-go/internal/jobs/firecrawl.go:242-252` — TAG-01实现（已正确）
- `backend-go/internal/domain/contentprocessing/content_completion_service.go:198-205` — TAG-02实现（已正确）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `TagJobQueue.Enqueue()` — transaction-based enqueue，支持幂等检查
- `ws.GetHub().BroadcastRaw()` — WebSocket广播机制
- `singleton pattern` — GetTagQueue()使用sync.Once

### Established Patterns
- WebSocket消息格式：`{type: "...", ...fields}` + json.Marshal + BroadcastRaw
- 定时轮询：goroutine + ticker + select stopChan
- Async API返回：`gin.H{"success": true, "job_id": "...", "message": "..."}`

### Integration Points
- HTTP Handler: articles/handler.go TagArticle endpoint
- WebSocket Hub: ws/hub.go BroadcastRaw
- TagQueue: singleton，启动时初始化

</code_context>

<specifics>
## Specific Ideas

- TAG-03 WebSocket消息可参考firecrawl.go的firecrawl_progress格式，保持一致风格
- TAG-04重试goroutine可在GetTagQueue()初始化时启动，或在runtime.go中添加ensureTagQueueRunning辅助函数
- job查询API可新增 `/api/tag-jobs/:id` 返回job状态，但优先使用WebSocket减少API开销

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 02-标签流程统一*
*Context gathered: 2026-04-11*