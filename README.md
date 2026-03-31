# RSS Reader

基于 Go + Nuxt 4 的 RSS 阅读器，三栏阅读界面，支持 AI 智能增强与内容汇总。

![主界面截图](img/image-main.png)

## ✨ 核心功能

### 主题图谱
- 你知道的，我一直想追踪一些事件的蛛丝马迹，比如事件之间的关联、事件的时间线发展（比如伊朗战争）
- 互联网没有记忆，很多事情会随着时间沉淀在互联网的大海深处，打捞非常困难
- 但是对于我们现在来说，使用AI去重、整理、打标签、梳理事件链路是一件相对来说有意义、并且有可行性的事情
- 嘿，你不想有一天对着你的孩子说————看，这是你爸妈经历的人生，排除掉垃圾信息后的人生

还在开发中，目前仅简单的关联效果
后续补充跨多时间段分析、单事件演变、事件聚合
rss的垃圾信息处理也在开发路上了
![主题图谱](img/image-topic.png)
![主题图谱](img/image-topic-timeline.png)
![主题图谱](img/image-topic-article.png)

### 📰 订阅管理

- Feed 管理：添加、编辑、删除、手动刷新、全量刷新
- 分类管理：自定义名称、图标、颜色
- OPML 导入导出
- 可配置自动刷新间隔

![订阅管理界面](img/image-feed.png)

### 📖 文章阅读

- FeedBro 风格三栏布局
- 收藏、已读标记、全屏阅读
- 预览模式与 iframe 模式切换
- 上一篇/下一篇快速导航

![文章阅读界面](img/image-article.png)

### 🤖 智能增强

- Firecrawl 全文抓取，补全 RSS 摘要内容
- AI 内容整理，生成结构化正文
- 内容源切换：原始内容 / Firecrawl 全文 / AI 整理稿

![内容增强状态面板](img/image-improve.png)

### 🧠 AI 总结

- 批量生成分类/Feed 级 AI 总结
- 按分类、订阅源、日期过滤
- WebSocket 实时显示生成进度

![AI 总结界面](img/image-ai-summary.png)

### 📊 阅读偏好

- 自动追踪阅读行为（打开、关闭、滚动、收藏）
- 偏好分数计算，优化排序
- 阅读统计展示

### 📰 Digest 汇总

- 日报/周报自动生成
- 飞书机器人推送
- Obsidian 笔记导出
- 可配置定时任务

![Digest 界面](img/image-digest.png)

## 🛠 技术栈

| 层级 | 技术 |
|------|------|
| 前端 | Nuxt 4 + Vue 3 + TypeScript + Pinia + Tailwind CSS v4 |
| 后端 | Go + Gin + GORM + SQLite |
| AI | OpenAI 兼容 API |

## 🚀 快速开始

### 前端

```bash
cd front
pnpm install
pnpm dev
```

### 后端

```bash
cd backend-go
go mod tidy
go run cmd/server/main.go
```

### 访问地址

- 前端：http://localhost:3001
- 后端：http://localhost:5000

## 📂 项目结构

```
my-robot/
├── front/        # Nuxt 4 前端
├── backend-go/   # Go + Gin 后端
├── docs/         # 项目文档
└── tests/        # 测试材料
```

## 📚 文档

- 项目总览：docs/architecture/overview.md
- 前端架构：docs/architecture/frontend.md
- 后端架构：docs/architecture/backend-go.md
- 功能说明：docs/guides/frontend-features.md
- 开发指南：docs/operations/development.md