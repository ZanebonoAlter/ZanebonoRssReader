# 项目总览

## 当前真实架构

这个仓库当前的真实运行结构是：

- `front/` - Nuxt 4 前端
- `backend-go/` - Go + Gin 后端
- `docs/` - 正式文档
- `tests/` - 独立测试与验证材料

历史文档里提到的 Python 爬虫服务和若干脚本目录，不是这个 checkout 里的真实运行主线。

## 运行关系

```text
Nuxt Frontend (front, :3001)
        |
        v
Go Backend (backend-go, :5000)
        |
        v
SQLite (backend-go/rss_reader.db)
```

## 阅读入口

- 前端架构：`docs/architecture/frontend.md`
- 后端架构：`docs/architecture/backend-go.md`
- 数据流：`docs/architecture/data-flow.md`
- 开发命令：`docs/operations/development.md`

## 顶层目录职责

- `README.md` - 项目简介、启动方式、总导航
- `AGENTS.md` - 代理协作规则
- `docs/` - 所有长期维护文档
- `front/` - 前端源码和前端局部文档
- `backend-go/` - 后端源码和后端局部文档
- `tests/` - 与主应用分离的测试材料
