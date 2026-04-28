# 项目文档 Wiki

RSS Reader 的全部文档入口。按需打开，避免一次加载过多上下文。

---

## 快速开始

| 文档 | 说明 |
|------|------|
| [../README.md](../README.md) | 项目简介与启动 |
| [guides/getting-started.md](guides/getting-started.md) | 开发环境搭建 |

---

## 架构

| 文档 | 说明 |
|------|------|
| [architecture/overview.md](architecture/overview.md) | 系统总览：技术栈、组件关系、核心子系统、数据流 |
| [architecture/backend-go.md](architecture/backend-go.md) | 后端分层、目录结构、数据模型 |
| [architecture/backend-runtime.md](architecture/backend-runtime.md) | 启动顺序、调度器管理、优雅退出 |
| [architecture/frontend.md](architecture/frontend.md) | Nuxt 4 分层、feature 组织、数据映射 |
| [architecture/frontend-components.md](architecture/frontend-components.md) | 各 feature 组件职责与交互 |
| [architecture/data-flow.md](architecture/data-flow.md) | 主链路、前端状态职责、定时任务链路 |
| [architecture/tracing.md](architecture/tracing.md) | OpenTelemetry 集成、埋点、查询 API |

**推荐阅读顺序**：overview → backend-go → backend-runtime → data-flow

---

## API 参考

每个领域一个文件，按需查阅：

| 文档 | 路由前缀 |
|------|----------|
| [api/_conventions.md](api/_conventions.md) | 通用约定（响应格式、分页、WebSocket） |
| [api/system.md](api/system.md) | `/`, `/health` |
| [api/categories.md](api/categories.md) | `/api/categories` |
| [api/feeds.md](api/feeds.md) | `/api/feeds` |
| [api/articles.md](api/articles.md) | `/api/articles` |
| [api/summaries.md](api/summaries.md) | `/api/summaries` |
| [api/ai-admin.md](api/ai-admin.md) | `/api/ai` |
| [api/opml.md](api/opml.md) | OPML 导入导出 |
| [api/schedulers.md](api/schedulers.md) | `/api/schedulers` |
| [api/content-completion.md](api/content-completion.md) | `/api/content-completion` |
| [api/firecrawl.md](api/firecrawl.md) | `/api/firecrawl` |
| [api/reading.md](api/reading.md) | `/api/reading-behavior`, `/api/user-preferences` |
| [api/topic-graph.md](api/topic-graph.md) | `/api/topic-graph`（含主题分析） |
| [api/digest.md](api/digest.md) | `/api/digest` |
| [api/traces.md](api/traces.md) | `/api/traces` |

完整索引见 [api/_index.md](api/_index.md)。

---

## 功能指南

| 文档 | 说明 |
|------|------|
| [guides/content-processing.md](guides/content-processing.md) | Firecrawl + AI 内容补全链路 |
| [guides/digest.md](guides/digest.md) | Digest 日报/周报配置 |
| [guides/digest-setup-guide.md](guides/digest-setup-guide.md) | Digest 配置步骤 |
| [guides/topic-graph.md](guides/topic-graph.md) | 主题图谱构建与分析 |
| [guides/reading-preferences.md](guides/reading-preferences.md) | 阅读偏好机制 |
| [guides/frontend-features.md](guides/frontend-features.md) | 前端功能说明 |
| [guides/configuration.md](guides/configuration.md) | 配置项说明 |
| [guides/deployment.md](guides/deployment.md) | 部署方式 |
| [guides/testing.md](guides/testing.md) | 测试指南 |

---

## 运维

| 文档 | 说明 |
|------|------|
| [operations/development.md](operations/development.md) | 构建、测试、验证命令 |
| [operations/database.md](operations/database.md) | 数据库说明 |
| [operations/postgres-migration.md](operations/postgres-migration.md) | PostgreSQL 迁移 |
| [operations/troubleshooting.md](operations/troubleshooting.md) | 排障指南 |

---

## 数据库

| 文档 | 说明 |
|------|------|
| [database/DATABASE_FIELDS.md](database/DATABASE_FIELDS.md) | 数据库字段详细说明 |

---

## 经验沉淀

| 文档 | 说明 |
|------|------|
| [experience/LESSONS_LEARNED.md](experience/LESSONS_LEARNED.md) | 踩坑记录与提交前检查清单 |
| [experience/ENCODING_SAFETY.md](experience/ENCODING_SAFETY.md) | Windows 编码安全 |

---

## 版本发布

`releases/` 目录保存每个里程碑的交付总结，包含需求覆盖、技术决策、技术债务等信息：
| 版本 | 说明 | 日期 |
|------|------|------|
| [v1.1](releases/MILESTONE_v1.1_SUMMARY.md) | 业务漏洞修复 | 2026-04-12 |

---

## 历史计划

`plans/` 目录保存已实施的设计与实施计划，供回溯参考。大部分文件按日期命名，如 `2026-03-04-ai-summary-enhancement-design.md`。

---

## 文档维护规则

- 文档只描述当前真实存在的目录和命令
- 新文档先判断该放在哪个目录（architecture / guides / api / operations / experience / plans）
- API 文档按领域拆分，每个文件对应一个路由前缀
- 前端文件保持 UTF-8，不要用 ANSI/GBK 重写
