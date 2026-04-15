# Phase 1: 基础设施与标签收敛 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-13
**Phase:** 01-基础设施与标签收敛
**Areas discussed:** 存量 embedding 迁移策略, 合并历史追溯程度, 阈值配置交互方式, embedding 模型切换策略

---

## 存量 embedding 迁移策略

| Option | Description | Selected |
|--------|-------------|----------|
| SQL 就地转换 | 写 SQL 把 JSON text 解析为 vector 类型。零停机，保留原始向量值 | |
| 全量重新生成 | 删旧向量，用新模型重算所有标签 embedding | |
| 就地转换 + 标记需重算 | SQL 转格式保底，同时记录当前模型名 | |

**User's choice:** 不需要迁移 — 目前没有存量 embedding 数据（功能未生效），直接从零建立 pgvector 列。旧文章支持后续批量重新计算。
**Notes:** 简化了迁移工作，不需要写数据转换脚本。

---

## 合并历史追溯程度

| Option | Description | Selected |
|--------|-------------|----------|
| 简单标记 + 目标ID | TopicTag 加 status 字段 + merged_into_id。简洁，保证引用不悬空 | ✓ |
| 完整合并事件日志 | 额外建 tag_merge_events 表记录源/目标/相似度/时间 | |
| 融入 aliases 即可 | 旧标签 label 写入目标 aliases，无额外字段 | |

**User's choice:** 简单标记 + 目标ID
**Notes:** 不需要额外的 merge_events 日志表，保持简洁。

---

## 阈值配置交互方式

| Option | Description | Selected |
|--------|-------------|----------|
| 融入现有 preferences | 复用 preferences 系统存储阈值 | |
| 独立配置表 | 新建 embedding_config 表，存储阈值、模型名、维度等 | ✓ |
| 仅 API，无前端 UI | 只提供 API 端点，纯命令行/API 调试 | |

**User's choice:** 独立配置表
**Notes:** embedding 相关配置（模型、维度、阈值）是独立配置域，不混入通用 preferences。

---

## embedding 模型切换策略

| Option | Description | Selected |
|--------|-------------|----------|
| 标记过期 + 后台重算 | 模型切换后标记现有 embedding 过期，后台任务异步重算 | ✓ |
| 切换时同步全量重算 | 切换时自动触发全量重算，停机时间取决于标签数量 | |
| 查询时实时重算 | 每次查询检测模型是否匹配，不匹配就重算单个标签 | |

**User's choice:** 标记过期 + 后台重算
**Notes:** 模型切换是低频操作，短暂降级可接受。切换期间相似度匹配降级为创建新标签，不影响标签创建流程。

---

## the agent's Discretion

- pgvector 列的具体维度和索引参数选择
- 嵌入生成的批处理策略
- 后台重算任务的并发和速率控制
- API 端点的具体请求/响应结构

## Deferred Ideas

None — discussion stayed within phase scope.
