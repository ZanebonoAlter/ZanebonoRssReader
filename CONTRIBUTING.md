<!-- generated-by: gsd-doc-writer -->

# 贡献指南

感谢你对 RSS Reader 项目的关注！本文档介绍如何参与贡献。

## 开发环境搭建

请先阅读以下文档完成本地环境配置：

- [快速上手](docs/guides/getting-started.md) — 前置条件与首次运行
- [开发指南](docs/operations/development.md) — 本地开发、构建、测试流程
- [配置说明](docs/guides/configuration.md) — 环境变量与配置项
- [项目总览](docs/architecture/overview.md) — 架构与运行关系

快速开始：

```bash
# 克隆仓库
git clone <repo-url>
cd my-robot

# 复制环境变量
cp .env.example .env

# 启动后端
cd backend-go
go mod tidy
go run cmd/server/main.go

# 启动前端（新终端）
cd front
pnpm install
pnpm dev
```

## 编码规范

本项目没有配置独立的 lint 或 format 工具（无 ESLint、Prettier、Biome），但有以下约定：

- **Go 代码**：遵循 `gofmt` 标准格式；导入分组为标准库 → 第三方 → 本地包。
- **前端代码**：使用 `<script setup lang="ts">` 的 Vue 3 Composition API；UTF-8 编码；遵循已有文件的分号风格。
- **代码组织**：
  - 前端 HTTP 逻辑放 `front/app/api/`，业务实现放 `front/app/features/`，类型定义集中在 `front/app/types/`
  - 后端路由在 `internal/app/router.go`，业务逻辑在 `internal/domain/*`
- **命名**：Go 导出符号 PascalCase，私有 lowerCamelCase；前端组件 PascalCase，composable 用 `use` 前缀。
- **数据映射**：`snake_case → camelCase` 转换集中在 API/store 层，不在模板或组件中做映射。
- **视觉风格**：保持 editorial / magazine 主题，不使用蓝紫色 SaaS 默认配色。

## 提交前检查

**前端改动**至少执行其中一项：

```bash
cd front
pnpm build
pnpm exec nuxi typecheck
pnpm test:unit
```

**后端改动**至少执行其中一项：

```bash
cd backend-go
go build ./...
go test ./...
```

**文档改动**如果涉及功能、接口、结构变化，需同步更新 `docs/` 下对应的架构和指南文档。

## PR 流程

1. **Fork 并创建分支**：从 `main` 分支拉取新分支，建议使用描述性名称（如 `feat/add-opml-export`、`fix/feed-refresh-error`）。
2. **确保检查通过**：运行上文"提交前检查"中的相关命令，确认构建和测试通过。
3. **保持变更聚焦**：一个 PR 解决一个问题或添加一个功能，避免混合多种不相关的改动。
4. **补充文档**：如果改动涉及 API 接口、数据流、运行时行为或 UI 结构的变化，请同步更新 `docs/` 下对应的文档。
5. **提交 PR**：在 PR 描述中说明改动目的、实现方式和测试情况。

## Issue 报告

项目未配置 GitHub Issue 模板。提交 Issue 时请包含：

- **复现步骤**：如何触发问题
- **预期行为**：你期望发生什么
- **实际行为**：实际发生了什么
- **环境信息**：Node.js 版本、Go 版本、操作系统
- **相关日志**：控制台输出或错误信息（如有）

## 许可证

本项目基于 [GNU General Public License v3.0](LICENSE) 开源。提交的贡献将遵循相同的许可证。
