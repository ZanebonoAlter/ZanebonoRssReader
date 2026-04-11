# PROJECT: RSS Reader 漏洞修复

## What This Is

RSS Reader 后端业务逻辑漏洞修复项目，目标解决定时任务触发、标签提取流程、状态一致性等核心业务问题。

## Core Value

**修复已发现的业务漏洞，确保系统稳定可靠运行**

用户已运行系统一段时间，发现了多个影响业务正确性的问题。修复这些问题比添加新功能更重要。

## Key Decisions

| Decision | Reason | Alternatives Considered |
|----------|--------|-------------------------|
| 使用GSD管理修复进度 | 可验证、可追踪、原子提交 | 直接在代码中修复（无记录） |
| 每个漏洞作为独立phase | 可独立验证、降低风险 | 批量修复（难以追踪） |
| 添加UAT测试 | 确保修复有效且不引入新问题 | 只改代码（无验证） |

## Current Milestone: v1.1 业务漏洞修复

**Goal:** 修复代码审查发现的6类业务漏洞，确保定时任务、标签提取、状态一致性正确运行

**Target features:**
- 定时任务并发控制与状态恢复
- 标签提取流程统一与队列管理
- 状态一致性检查与自动恢复
- API交互规范化
- 错误处理完善

## Active Requirements

See `.planning/REQUIREMENTS.md` for full list.

## Validated Requirements

**Phase 03 (状态一致性修复):**
- STAT-01: Feed删除时文章级联删除 (CASCADE实现)
- STAT-02: 文章清理不误删活跃文章
- STAT-03: Summary-only feed文章summary_status初始化为pending
- STAT-04: 阻塞文章自动恢复机制
- STAT-05: 阻塞数量超过阈值时WARN告警

## Out of Scope

| Requirement | Reason |
|-------------|--------|
| 新功能开发 | 本次只修复漏洞，不添加功能 |
| UI界面调整 | 漏洞修复是后端逻辑问题 |
| 性能优化 | 不是当前发现的漏洞 |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition:**
1. Requirements invalidated → Move to Out of Scope with reason
2. Requirements validated → Move to Validated with phase reference
3. New requirements emerged → Add to Active
4. Decisions to log → Add to Key Decisions

**After each milestone:**
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

## Context

**Codebase:** Go + Nuxt 4, SQLite, 单用户部署

**Key subsystems:**
- 定时任务: auto_refresh, auto_summary, firecrawl, content_completion, preference_update, digest
- 标签系统: TagQueue, TagJobQueue, TagArticle, RetagArticle
- 状态管理: Article states (firecrawl_status, summary_status), Feed states

**Known vulnerabilities:**
1. 并发控制不完整 → 重复执行风险
2. 标签提取绕过队列 → 流程混乱
3. 状态转换遗漏 → 文章卡死
4. API不一致 → 前端状态漂移
5. panic覆盖缺失 → 服务崩溃
6. stale状态无处理 → 任务卡死

---

*Last updated: 2026-04-11 (Phase 03 complete)*