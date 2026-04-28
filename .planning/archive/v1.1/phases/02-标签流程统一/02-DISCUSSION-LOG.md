# Phase 2: 标签流程统一 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-11
**Phase:** 02-标签流程统一
**Areas discussed:** TAG-03 API改造, TAG-04 启动重试

---

## TAG-03: 手动打标签API改造

| Option | Description | Selected |
|--------|-------------|----------|
| 同步等待 | enqueue后等待队列处理完成，返回新标签。用户立即看到结果 | |
| 异步返回job_id | enqueue后立即返回job_id，前端通过WebSocket监听或轮询查询结果 | ✓ |
| 异步仅确认 | enqueue后返回"已提交"，用户刷新页面查看结果 | |

**User's choice:** 异步返回job_id
**Notes:** 用户期望能追踪任务状态，不阻塞API响应

### TAG-03 前端监听机制

| Option | Description | Selected |
|--------|-------------|----------|
| WebSocket广播 | 前端监听tag_completed WebSocket消息，收到后刷新文章显示 | ✓ |
| 新增job查询API | 前端用job_id调用/api/articles/:id/tags/job/:jobId查询状态 | |
| 仅查询API | TagQueue没有WebSocket机制，直接用查询API | |

**User's choice:** WebSocket广播（推荐）
**Notes:** 与现有firecrawl_progress风格保持一致

---

## TAG-04: TagQueue启动重试机制

| Option | Description | Selected |
|--------|-------------|----------|
| 定时轮询重试 | 应用启动后，后台goroutine每30秒尝试Start()直到成功 | ✓ |
| enqueue触发重试 | 首次enqueue失败时触发重新Start() | |
| 失败记录不重试 | 记录启动失败日志，运维手动干预 | |

**User's choice:** 定时轮询重试（推荐）
**Notes:** 简单可靠，不依赖外部触发

### TAG-04 重试间隔和次数

| Option | Description | Selected |
|--------|-------------|----------|
| 30秒，最多10次 | 启动失败后30秒尝试重试，最大10次后放弃 | ✓ |
| 5秒，最多20次 | 启动失败后5秒尝试重试。更快恢复 | |
| 无限重试 | 无限重试直到成功 | |

**User's choice:** 30秒，最多10次（推荐）
**Notes:** 约5分钟后放弃，避免无限循环

---

## Agent's Discretion

- WebSocket消息字段细节（是否包含tag_count等）
- job查询API路径设计
- 重试goroutine与TagQueue singleton的同步机制

## Deferred Ideas

None — discussion stayed within phase scope