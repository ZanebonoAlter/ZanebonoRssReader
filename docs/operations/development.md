# 开发指南

## 环境要求

- Node.js 18+
- pnpm 10+
- Go 1.21+

## 前端

```bash
cd front
pnpm install
pnpm dev
pnpm build
pnpm exec nuxi typecheck
```

默认开发地址：`http://localhost:3001`

## 后端

```bash
cd backend-go
go mod tidy
go run cmd/server/main.go
go test ./...
```

默认 API 地址：`http://localhost:5000`

## 开发顺序

1. 启动后端
2. 启动前端
3. 修改代码时同步更新文档入口
4. 提交前至少跑对应类型检查或测试
