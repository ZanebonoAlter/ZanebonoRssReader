# Phase 3: 状态一致性修复 - Context

**Gathered:** 2026-04-11
**Status:** Ready for planning

<domain>
## Phase Boundary

修复文章状态一致性相关问题，确保状态流转正确、阻塞任务可恢复、异常情况有告警。

**具体范围：**
- ✅ STAT-01: Feed删除时关联文章处理（现有CASCADE删除已满足"清理"需求，无需修改）
- ✅ STAT-02: CleanupOldArticles缺失feed处理（选择保留现有逻辑，feed不存在时跳过，因删除feed时文章已被CASCADE清理）
- ⚙️ STAT-03: buildArticleFromEntry逻辑修正：仅开启ArticleSummaryEnabled不开启Firecrawl时，设置summary_status="pending"而非"complete"
- ⚙️ STAT-04: Blocked文章（waiting_for_firecrawl）恢复机制：每小时检查feed状态变化，解除符合条件的阻塞
- ⚙️ STAT-05: ContentCompletion阻塞文章超过50篇时输出警告日志
</domain>

<decisions>
## Implementation Decisions

### STAT-01 关联文章处理
- **D-01:** 保留现有数据库ON DELETE CASCADE约束，删除feed时自动删除所有关联文章，无需额外状态标记逻辑

### STAT-02 旧文章清理
- **D-02:** 保持现有CleanupOldArticles逻辑不变，仅在feed存在时运行，删除feed时文章已被CASCADE清理无需额外处理

### STAT-03 文章初始状态修正
- **D-03:** 修改buildArticleFromEntry函数，当feed.ArticleSummaryEnabled为true且feed.FirecrawlEnabled为false时，将article.SummaryStatus设置为"pending"而非默认的"complete"

### STAT-04 阻塞文章恢复
- **D-04:** 新增独立定时任务，每小时运行一次，检查所有firecrawl_status为"waiting_for_firecrawl"或"blocked"的文章
- **D-05:** 检查对应feed的状态变化，当feed.FirecrawlEnabled变为true时，将文章firecrawl_status重置为"pending"重新入队处理
- **D-06:** 清理feed已被删除的阻塞文章（因CASCADE约束实际不存在此类数据，保留防御性判断）

### STAT-05 阻塞数量告警
- **D-07:** 使用固定阈值50篇，每小时阻塞检查时统计ContentCompletion状态为"blocked"的文章数量
- **D-08:** 超过阈值时输出WARN级别日志，包含当前阻塞数量信息
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements
- `.planning/REQUIREMENTS.md` §STAT-01~05 — 状态一致性需求详细定义

### Source Files (Affected)
- `backend-go/internal/domain/feeds/service.go`: buildArticleFromEntry函数修改（STAT-03）
- `backend-go/internal/jobs/firecrawl.go`: 阻塞状态逻辑参考，新增恢复任务
- `backend-go/internal/domain/contentprocessing/content_completion_service.go`: 阻塞状态逻辑参考，告警统计
- `backend-go/internal/app/runtime.go`: 注册新增的定时任务
</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- 现有定时任务调度框架：参考auto_refresh、firecrawl等scheduler的实现方式
- 数据库操作模式：GORM查询、更新模式与现有代码保持一致
- 日志规范：使用标准log.Printf输出，级别遵循现有约定
</code_context>

<specifics>
## Specific Implementation Ideas

1. **STAT-03修改：**
   在buildArticleFromEntry函数中新增判断：
   ```go
   if feed.ArticleSummaryEnabled && !feed.FirecrawlEnabled {
       article.SummaryStatus = "pending"
   }
   ```
   保持现有FirecrawlEnabled时的逻辑不变。

2. **STAT-04定时任务：**
   新增BlockedArticleRecoveryScheduler，每小时运行一次：
   - 查询所有firecrawl_status IN ("waiting_for_firecrawl", "blocked")的文章
   - 关联查询对应的feed，若feed已删除则跳过（实际不存在）
   - 若feed.FirecrawlEnabled已变为true，更新文章firecrawl_status为"pending"并入队Firecrawl处理

3. **STAT-05告警：**
   在定时任务中增加统计：
   - 查询content_completion_status为"blocked"的文章总数
   - 若总数>50，输出WARN日志：`[WARN] ContentCompletion blocked articles exceeded threshold: %d > 50`
</specifics>

<deferred>
## Deferred Ideas
- 阻塞告警阈值可配置化（当前使用固定50，后续如有需求可添加到用户配置）
- 阻塞恢复任务频率可配置化（当前固定每小时一次）
</deferred>

---
*Phase: 03-状态一致性修复*
*Context gathered: 2026-04-11*