# Phase 2: 关注标签与首页推送 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-15 (re-discussion)
**Phase:** 02-watched-tags-homepage-feed
**Mode:** discuss (full re-discussion based on current codebase state)
**Areas discussed:** 关注交互位置与方式, 首页推送展示, 相关度排序, 抽象标签支持, 无关注标签过渡

---

## 关注交互位置与方式

| Option | Description | Selected |
|--------|-------------|----------|
| 主题图谱页 | 关注操作集中在主题图谱页，标签卡片上切换 | ✓ |
| 首页侧边栏内 | 直接在侧边栏标签列表操作 | |
| 两个入口 | 主题图谱页主要入口 + 首页侧边栏可取消关注 | |

**User's choice:** 主题图谱页

**图标风格:**

| Option | Description | Selected |
|--------|-------------|----------|
| 眼睛图标 | "关注这个标签"，轻量 | |
| 心形图标 | 情感更强 | ✓ |
| 你来决定 | 不预先限定 | |

**User's choice:** 心形图标

**反馈策略:**

| Option | Description | Selected |
|--------|-------------|----------|
| 即时 UI 反馈 | 前端先更新 UI，后台同步 API | ✓ |
| 等 API 响应后更新 | 更保守但体验慢 | |

**User's choice:** 即时 UI 反馈

**关注数量限制:**

| Option | Description | Selected |
|--------|-------------|----------|
| 无上限 | 用户自由决定 | ✓ |
| 设置上限 | 避免文章流过于庞大 | |

**User's choice:** 无上限

**Notes:** 主题图谱页现在已有标签树、合并预览、图谱可视化等功能，关注操作入口仍然合适。

---

## 首页推送展示

**推送入口:**

| Option | Description | Selected |
|--------|-------------|----------|
| 侧边栏新增入口 | 在现有入口旁新增"关注标签"入口 | ✓ |
| 替换默认首页 | 默认显示关注标签文章流 | |
| 标签筛选器 | 文章列表顶部横向标签条 | |

**User's choice:** 侧边栏新增入口

**侧边栏组织:**

| Option | Description | Selected |
|--------|-------------|----------|
| 分组列表 | 侧边栏"关注标签"分组，展开后列出关注标签 | ✓ |
| 平级入口 + 内部筛选 | 与"全部文章"平级，内部再筛选 | |

**User's choice:** 分组列表

**文章流内容:**

| Option | Description | Selected |
|--------|-------------|----------|
| 全部文章 | 显示全部关联了关注标签的文章 | ✓ |
| 仅 watched_at 后的文章 | 避免历史文章过多 | |

**User's choice:** 全部文章

**侧边栏刷新:**

| Option | Description | Selected |
|--------|-------------|----------|
| 每次从后端加载 | 确保数据一致性 | ✓ |
| Store 缓存 + 增量更新 | 减少请求但需管理状态 | |

**User's choice:** 每次从后端加载

**Notes:** 操作流审查通过：主题图谱页关注 → 首页侧边栏查看（每次从后端加载关注列表）。

---

## 相关度排序

**"全部关注"混合流排序:**

| Option | Description | Selected |
|--------|-------------|----------|
| 仅标签数 | 匹配关注标签数排序 | ✓ |
| 标签数 + embedding 加权 | 更精细但复杂 | |
| 默认时间 + 可选相关度 | 用户可手动切换 | |

**User's choice:** 仅标签数

**单标签筛选排序:**

| Option | Description | Selected |
|--------|-------------|----------|
| 时间倒序 | 同标签下不需要复杂排序 | ✓ |
| 其他排序 | 同标签内也按相关度 | |

**User's choice:** 时间倒序

---

## 抽象标签支持

**抽象标签能否关注:**

| Option | Description | Selected |
|--------|-------------|----------|
| 可直接关注 | 关注后子标签文章全部出现 | ✓ |
| 只能关注具体标签 | 抽象标签仅展示用 | |
| 你来决定 | 后续再确定 | |

**User's choice:** 可直接关注

**相关度计算:**

| Option | Description | Selected |
|--------|-------------|----------|
| 仅统计直接标签数 | 简单明确 | |
| 抽象标签权重更高 | 匹配抽象标签权重更高 | ✓ |
| 你来决定 | 研究阶段再确定 | |

**User's choice:** 抽象标签权重更高

**Notes:** Phase 7 新增了抽象标签概念，这是原 CONTEXT.md 未考虑的重要变化。抽象标签可关注意味着需要 JOIN topic_tag_relations 查询子标签的文章。

---

## 无关注标签时的过渡

| Option | Description | Selected |
|--------|-------------|----------|
| 引导横幅 + 默认时间线 | 侧边栏显示引导横幅，首页保持默认 | ✓ |
| 轻量提示 | 仅标注"0 个关注标签" | |

**User's choice:** 引导横幅 + 默认时间线

---

## the agent's Discretion

- 侧边栏关注标签分组的具体视觉设计（折叠/展开、图标选择）
- 主题图谱页标签卡片上心形图标的具体位置和大小
- 后端 API 的请求/响应结构细节
- 分页加载策略（复用现有 useArticlePagination 模式）
- 抽象标签相关度的具体权重倍数

## Deferred Ideas

None — discussion stayed within phase scope.
