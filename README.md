# RSS Reader

基于 Go + Nuxt 4 的 RSS 阅读器，包含三栏阅读界面、AI 总结、内容处理、阅读偏好和 digest 能力。

## 当前真实架构

```text
front/       Nuxt 4 前端，默认 :3001
backend-go/  Go + Gin 后端，默认 :5000
docs/        正式文档入口
tests/       独立测试材料
```

这份 README 只描述当前仓库里真实存在的结构。

## 快速开始

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

## 访问地址

- 前端：`http://localhost:3001`
- 后端：`http://localhost:5000`

## 从哪开始读

- 项目总览：`docs/architecture/overview.md`
- 前端架构：`docs/architecture/frontend.md`
- 后端架构：`docs/architecture/backend-go.md`
- 数据流：`docs/architecture/data-flow.md`
- 开发命令：`docs/operations/development.md`
- 文档导航：`docs/README.md`

## 主要能力

- RSS feed 管理和自动刷新
- 文章阅读与内容处理
- AI 摘要与汇总
- 阅读偏好记录
- Digest 生成与外部推送

## 当前目录

```text
my-robot/
├── README.md
├── AGENTS.md
├── docs/
├── front/
├── backend-go/
└── tests/
```
