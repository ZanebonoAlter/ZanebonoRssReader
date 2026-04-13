# Phase 1: 并发控制修复 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-11
**Phase:** 01-并发控制修复
**Areas discussed:** Auto-refresh等待机制, Firecrawl TriggerNow返回值, TriggerNow格式一致性, Digest reload优雅停止

---

## Auto-refresh等待机制 (CONC-01)

| Option | Description | Selected |
|--------|-------------|----------|
| Fire-and-forget（现状） | TriggerNow()返回后feeds继续刷新，auto-summary在刷新完成后触发。前端只看到"触发成功"但不知道何时完成。 | |
| 等待完成（阻塞前端） | TriggerNow()阻塞等待feeds刷新完成再返回实际结果。前端感知真实结果但用户体验差。 | |
| 异步+完成通知（推荐） | TriggerNow()立即返回，通过WebSocket或状态API让前端感知完成。不阻塞前端且能知道结果。 | ✓ |

**User's choice:** 异步+完成通知（推荐）
**Notes:** 需要新增完成通知机制，可参考Firecrawl WebSocket broadcastProgress模式

---

## Firecrawl TriggerNow返回值 (CONC-02)

| Option | Description | Selected |
|--------|-------------|----------|
| 等待完成（阻塞前端） | TriggerNow()等待runCrawlCycle完成再返回completed/failed计数。前端感知真实结果但可能等很久。 | |
| 现有WebSocket通知（现状） | TriggerNow()立即返回，前端通过WebSocket firecrawl_progress监听进度。已有机制。 | |
| 异步+batch查询（推荐） | TriggerNow()立即返回batch_id，前端通过batch_id查询执行状态。新增API，结合WebSocket使用。 | ✓ |

**User's choice:** 异步+batch查询（推荐）
**Notes:** batchID已在runCrawlCycle中生成，需新增batch状态查询API

---

## TriggerNow格式一致性 (CONC-03)

| Option | Description | Selected |
|--------|-------------|----------|
| 统一必填字段（推荐） | accepted, started, reason, message, status_code必填，成功时可选effectful/summary等扩展。保持灵活扩展。 | ✓ |
| 严格统一全部字段 | 所有scheduler返回完全相同字段结构。更严格但可能限制扩展。 | |
| 保持现状 | 现有格式已足够一致，无需修改。 | |

**User's choice:** 统一必填字段（推荐）
**Notes:** 最小修改，主要统一status_code使用http常量而非硬编码409

---

## Digest reload优雅停止 (CONC-05)

| Option | Description | Selected |
|--------|-------------|----------|
| 现有实现已正确（推荐） | cron.Stop().Done()等待执行任务完成，AddFunc重新添加时保持原schedule时间。 | ✓ |
| 显式保存/恢复pending任务 | reload前检查pending tasks保存到临时变量，reload后恢复。额外保险但可能过度设计。 | |
| 添加测试验证 | 实际测试验证reload后定时任务是否保持原schedule。 | |

**User's choice:** 现有实现已正确（推荐）
**Notes:** CONC-05无需代码修改，现有cron库行为已正确处理

---

## Agent's Discretion

- WebSocket消息格式细节设计
- batch状态查询API具体字段
- 前端监听机制实现方式

## Deferred Ideas

None — discussion stayed within phase scope