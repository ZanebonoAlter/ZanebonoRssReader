# Phase 2: 关注标签与首页推送 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-13
**Phase:** 02-watched-tags-homepage-feed
**Areas discussed:** 关注交互位置与方式, 首页推送展示, 相关度排序, 无关注标签过渡, 关注标签数量限制, 文章时间范围策略, 侧边栏关注标签样式

---

## 关注交互位置与方式

| Option | Description | Selected |
|--------|-------------|----------|
| 主题图谱页 | 在现有主题图谱页的标签列表/详情面板中添加关注开关 | ✓ |
| 首页侧边栏 | 在首页侧边栏新增"关注标签"分组 | |
| 两处都可关注 | 主页和主题图谱页都可以切换关注 | |

**User's choice:** 主题图谱页
**Notes:** 用户进一步要求每个标签卡片有切换图标 + 独立面板展示已关注标签（按分类分组）。关注状态切换采用即时反馈策略。

---

## 首页推送展示方式

| Option | Description | Selected |
|--------|-------------|----------|
| 侧边栏新筛选入口 | 在侧边栏添加"关注标签"入口，点击后中间面板切换为关注文章流 | ✓ |
| 文章列表顶部 Tab | 中间文章列表顶部加 Tab 切换 | |
| 独立页面 | 关注文章流作为独立页面 | |

**User's choice:** 侧边栏新筛选入口

**Sub-decision: 展示方式**

| Option | Description | Selected |
|--------|-------------|----------|
| 单一混合流 | 一个入口显示所有关注标签文章 | |
| 按标签分别筛选 | 每个标签作为独立筛选器 | ✓ |

**User's choice:** 按标签分别筛选

**Sub-decision: 混合流入口**

| Option | Description | Selected |
|--------|-------------|----------|
| 有"全部关注"+ 各标签 | 侧边栏有混合流入口 + 每个标签筛选器 | ✓ |
| 只有各标签筛选器 | 无混合流 | |

**User's choice:** 有"全部关注"+ 各标签

---

## 相关度排序

| Option | Description | Selected |
|--------|-------------|----------|
| 标签数量排序 | 只按匹配的关注标签数量排序，简单可预测 | ✓ |
| 标签数 + embedding 加权 | 匹配标签数 + embedding 距离加权融合，更精准 | |

**User's choice:** 标签数量排序

**Sub-decision: 默认排序**

| Option | Description | Selected |
|--------|-------------|----------|
| 默认相关度 | "全部关注"混合流默认按匹配标签数排序 | ✓ |
| 默认时间倒序 | 与现有体验一致 | |

**User's choice:** 默认相关度

---

## 无关注标签时的过渡

| Option | Description | Selected |
|--------|-------------|----------|
| 静默回退 | 侧边栏标注"暂无关注"，主页保持默认时间线 | |
| 引导横幅 | 显示引导横幅，引导用户前往主题图谱页 | ✓ |

**User's choice:** 引导横幅

**Sub-decision: 显示时机**

| Option | Description | Selected |
|--------|-------------|----------|
| 仅首次显示 | 用户关闭后不再显示 | |
| 每次都显示 | 每次点击侧边栏"关注标签"入口且无关注标签时 | ✓ |

**User's choice:** 每次都显示

---

## 关注标签数量限制

| Option | Description | Selected |
|--------|-------------|----------|
| 无限制 | 用户自由决定 | ✓ |
| 有上限（~20 个） | 超出时提示整理 | |

**User's choice:** 无限制

---

## 文章时间范围策略

| Option | Description | Selected |
|--------|-------------|----------|
| 全部关联文章 | 不区分 watched_at 前后 | ✓ |
| 仅 watched_at 之后 | 只显示关注后的新文章 | |

**User's choice:** 全部关联文章

---

## 侧边栏关注标签样式

| Option | Description | Selected |
|--------|-------------|----------|
| 扁平列表 + 颜色点 | 每个标签带分类颜色点+标签名+文章数量 | ✓ |
| 按分类分组 | 按事件/人物/关键词分组，可折叠 | |
| 简约纯文本 | 只显示标签名 | |

**User's choice:** 扁平列表 + 颜色点，同时支持按分类筛选

---

## the agent's Discretion

- 侧边栏关注标签分组的具体视觉设计
- 切换关注状态的图标样式
- 关注面板在主题图谱页的具体位置
- 后端 API 请求/响应结构
- 分页加载策略

## Deferred Ideas

None — discussion stayed within phase scope.
