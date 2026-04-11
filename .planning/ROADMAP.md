# ROADMAP: Milestone v1.1 业务漏洞修复

## Overview

| Metric | Value |
|--------|-------|
| Milestone | v1.1 业务漏洞修复 |
| Phases | 6 |
| Requirements | 23 |
| Coverage | 100% ✓ |

## Phases

| # | Phase | Goal | Requirements | Success Criteria |
|---|-------|------|--------------|------------------|
| 1 | 并发控制修复 | scheduler并发执行不丢失任务、不重复执行 | CONC-01~05 | 4 |
| 2 | 标签流程统一 | 所有标签提取走统一队列，无绕过 | TAG-01~06 | 5 |
| 3 | 状态一致性 | 文章状态正确转换，无卡死 | STAT-01~05 | 4 |
| 4 | API规范化 | 前端API调用一致，状态同步正确 | API-01~04 | 3 |
| 5 | 错误处理完善 | panic不崩溃，错误可追踪 | ERR-01~05 | 4 |
| 6 | 恢复机制 | stale状态自动恢复，不永久卡死 | REC-01~04 | 3 |

## Phase Details

### Phase 1: 并发控制修复

**Goal:** scheduler并发执行不丢失任务、不重复执行

**Requirements:** CONC-01, CONC-02, CONC-03, CONC-04, CONC-05

**Plans:** 3 plans in 2 waves

Plans:
- [x] 01-01-PLAN.md — CONC-03: TriggerNow status_code consistency (Wave 1)
- [x] 01-02-PLAN.md — CONC-02: Firecrawl batch_id return (Wave 1)
- [x] 01-03-PLAN.md — CONC-01: Auto-refresh completion WebSocket (Wave 2)

**Success Criteria:**
1. 手动触发auto-refresh，观察日志确认所有feed刷新完成后再触发auto-summary
2. 手动触发firecrawl，前端收到明确的执行结果（成功/失败/已运行）
3. 连续点击scheduler trigger按钮，第二次收到"已在执行中"提示而非重复执行
4. 模拟feed刷新panic，确认其他feed继续刷新且panic被记录

**Files affected:**
- `backend-go/internal/jobs/auto_refresh.go`
- `backend-go/internal/jobs/firecrawl.go`
- `backend-go/internal/jobs/content_completion.go`
- `backend-go/internal/jobs/preference_update.go`
- `backend-go/internal/platform/ws/hub.go`

---

### Phase 2: 标签流程统一

**Goal:** 所有标签提取走统一队列，无绕过

**Requirements:** TAG-01, TAG-02, TAG-03, TAG-04, TAG-05, TAG-06

**Plans:** 1 plan in 1 wave

Plans:
- [ ] 02-01-PLAN.md — TAG-03/04: 异步API + 启动重试机制 (Wave 1)

**Success Criteria:**
1. Firecrawl完成后，文章标签通过TagJobQueue生成（查看tag_jobs表记录）
2. ContentCompletion完成后，文章标签通过TagJobQueue生成
3. 手动调用/articles/:id/tags API，tag_jobs表新增一条记录
4. TagQueue启动失败后自动重试成功（查看日志）
5. 同一文章同时触发多个标签任务，最终只生成一套标签

**Files affected:**
- `backend-go/internal/jobs/firecrawl.go`
- `backend-go/internal/domain/contentprocessing/content_completion_service.go`
- `backend-go/internal/domain/articles/handler.go`
- `backend-go/internal/domain/topicextraction/tag_queue.go`
- `backend-go/internal/domain/topicextraction/tag_job_queue.go`
- `backend-go/internal/domain/topicextraction/article_tagger.go`

---

### Phase 3: 状态一致性

**Goal:** 文章状态正确转换，无卡死

**Requirements:** STAT-01, STAT-02, STAT-03, STAT-04, STAT-05

**Success Criteria:**
1. 删除feed后，其文章firecrawl_status/summary_status显示"abandoned"
2. 新feed（无max_articles配置）的文章超过100篇时自动清理旧文章
3. 创建只开ArticleSummaryEnabled的feed，新文章summary_status为"pending"
4. Feed从FirecrawlEnabled=false改为true后，blocked文章解除waiting_for_firecrawl状态
5. ContentCompletion overview显示blocked超过50篇时，日志有warning

**Files affected:**
- `backend-go/internal/domain/feeds/service.go`
- `backend-go/internal/domain/models/article.go`
- `backend-go/internal/domain/contentprocessing/content_completion_service.go`
- `backend-go/internal/jobs/content_completion.go`

---

### Phase 4: API规范化

**Goal:** 前端API调用一致，状态同步正确

**Requirements:** API-01, API-02, API-03, API-04

**Success Criteria:**
1. 前端调用scheduler trigger使用统一的apiClient（查看scheduler.ts代码）
2. 标记文章已读后，sidebar feed unread count正确减少
3. "全部标记已读"后，所有feed（包括未分类）的unread count清零
4. 所有scheduler status API返回相同字段结构

**Files affected:**
- `front/app/api/scheduler.ts`
- `front/app/stores/api.ts`
- `backend-go/internal/jobs/handler.go`
- `backend-go/internal/jobs/auto_refresh.go`
- `backend-go/internal/jobs/firecrawl.go`
- `backend-go/internal/jobs/content_completion.go`
- `backend-go/internal/jobs/auto_summary.go`

---

### Phase 5: 错误处理完善

**Goal:** panic不崩溃，错误可追踪

**Requirements:** ERR-01, ERR-02, ERR-03, ERR-04, ERR-05

**Success Criteria:**
1. Firecrawl处理过程中发生panic，scheduler继续运行且lastError记录panic原因
2. Preference update发生panic，scheduler继续运行
3. Digest生成发生panic，下次定时任务继续触发
4. Firecrawl/preference_update错误持久化到scheduler_tasks表
5. Digest执行有数据库记录（新建digest_scheduler_tasks表或复用scheduler_tasks）

**Files affected:**
- `backend-go/internal/jobs/firecrawl.go`
- `backend-go/internal/jobs/preference_update.go`
- `backend-go/internal/domain/digest/scheduler.go`
- `backend-go/internal/domain/models/scheduler_task.go`

---

### Phase 6: 恢复机制

**Goal:** stale状态自动恢复，不永久卡死

**Requirements:** REC-01, REC-02, REC-03, REC-04

**Success Criteria:**
1. Feed刷新卡住超过5分钟被重置，日志显示feed_id和stale时长
2. Firecrawl job processing超过lease时间被重置为pending，重新入队
3. ContentCompletion article processing超过30分钟被重置，重新处理
4. TagQueue job失败5次后标记failed，不再无限重试

**Files affected:**
- `backend-go/internal/jobs/auto_refresh.go`
- `backend-go/internal/jobs/firecrawl.go`
- `backend-go/internal/jobs/content_completion.go`
- `backend-go/internal/domain/topicextraction/tag_queue.go`
- `backend-go/internal/domain/contentprocessing/firecrawl_job_queue.go`

---

## Dependencies

```
Phase 1 ──┐
          │
Phase 2 ──┤── Phase 5 (错误处理依赖前面phase的panic recovery)
          │
Phase 3 ──┤── Phase 6 (恢复机制依赖状态检查逻辑)
          │
Phase 4 ──┘
```

建议执行顺序: 1 → 2 → 3 → 4 → 5 → 6

---

## Verification

**After all phases:**
1. 运行完整integration test suite (`tests/workflow/`)
2. 手动测试各scheduler trigger行为
3. 检查scheduler_tasks表有完整执行记录
4. 检查tag_jobs表有正确任务流转
5. 无文章卡在blocked/stale状态超过阈值

---

*Generated by GSD roadmap workflow*