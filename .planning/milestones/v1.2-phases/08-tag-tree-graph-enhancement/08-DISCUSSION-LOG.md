# Phase 08: 标签树增强与图谱交互优化 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-14
**Phase:** 08-tag-tree-graph-enhancement
**Areas discussed:** Description 提取策略, 时间筛选维度, 图谱中抽象标签展示, 合并预览迁移到设置

---

## Description 提取策略

| Option | Description | Selected |
|--------|-------------|----------|
| 标签提取时同步生成 | findOrCreateTag 创建新标签时调用 LLM 生成 description | ✓ |
| 延迟批量生成 | 后台定时任务批量调用 LLM 补全 | |
| 按需生成 | 仅在 LLM 需要使用 description 时才触发生成 | |

**User's choice:** 标签提取时同步生成，复用 json 模式保证结构化输出
**Notes:** 用户明确要求复用 JSON 模式做 structured output

### Follow-up: 抽象标签 description

| Option | Description | Selected |
|--------|-------------|----------|
| ExtractAbstractTag 时顺带生成 | 调用 LLM 生成名称时同时生成 description，一次调用搞定 | ✓ |
| 单独生成 | 先只生成名称，description 后续补全 | |

**User's choice:** ExtractAbstractTag 时顺带生成

---

## 时间筛选维度

| Option | Description | Selected |
|--------|-------------|----------|
| 关联文章的发布时间 | 按 articles.published_at 筛选活跃标签 | ✓ |
| 标签的创建时间 | 按标签 created_at 筛选 | |
| 两者都支持 | 提供两个筛选维度 | |

**User's choice:** 关联文章的发布时间

### Follow-up: 不活跃标签处理

| Option | Description | Selected |
|--------|-------------|----------|
| 完全隐藏 | 不显示不活跃标签 | |
| 置灰但保留 | 降低透明度但仍显示在树中 | ✓ |
| 折叠到分组 | 收起到"历史标签"分组中 | |

**User's choice:** 置灰但保留

---

## 图谱中抽象标签展示

| Option | Description | Selected |
|--------|-------------|----------|
| 相同 category 颜色 + 光环 | 保持颜色加外发光效果 | ✓ |
| 统一用特殊颜色 | 所有抽象标签用同一特殊颜色 | |
| 相同颜色 + 特殊形状 | 颜色不变用不同形状区分 | |

**User's choice:** 相同 category 颜色 + 光环

### Follow-up: 点击交互

| Option | Description | Selected |
|--------|-------------|----------|
| 弹出详情面板 | 子标签列表 + 文章时间线，可按子标签筛选 | ✓ |
| 聚焦展开子节点 | 图谱中展开高亮子标签节点 | |
| 两者结合 | 默认聚焦 + 面板入口 | |

**User's choice:** 弹出详情面板

---

## 合并预览迁移到设置

| Option | Description | Selected |
|--------|-------------|----------|
| 完全移除 | TopicGraphPage 不再有合并预览入口 | ✓ |
| 保留快捷入口 | 保留按钮/链接打开设置页对应面板 | |

**User's choice:** 完全移除

### Follow-up: 抽象层重建触发

| Option | Description | Selected |
|--------|-------------|----------|
| 自动触发 | 合并后自动检查并重建 | |
| 提示用户手动触发 | 弹出提示，用户确认才重建 | ✓ |
| 不触发 | 合并不影响抽象层 | |

**User's choice:** 提示用户手动触发

---

## Remaining Items (not discussed in detail)

- 节点手动归类：按 ROADMAP 描述执行，弹窗显示 embedding 相近的抽象层供选择
- 子标签合并到抽象标签时删除原标签 embedding：按 ROADMAP 描述执行

## Agent's Discretion

- description 字段长度限制（由 LLM prompt 控制）
- 时间筛选默认范围
- 光环效果具体实现
- 详情面板布局和动画
- 提示弹窗样式

## Deferred Ideas

- description 的 LLM 质量评估
- 时间筛选的热度排序
- 图谱中抽象标签的动态展开/收起
