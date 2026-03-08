# 文档导航

这个仓库现在只认两个入口：

- 项目入口：`README.md`
- 文档入口：`docs/README.md`

## 先看哪份

- 想快速了解项目：`docs/architecture/overview.md`
- 想看前端怎么组织：`docs/architecture/frontend.md`
- 想看后端怎么组织：`docs/architecture/backend-go.md`
- 想看数据怎么流动：`docs/architecture/data-flow.md`
- 想看开发命令和日常流程：`docs/operations/development.md`
- 想看数据库说明：`docs/operations/database.md`
- 想看 digest 配置：`docs/guides/digest.md`
- 想看阅读偏好：`docs/guides/reading-preferences.md`
- 想看内容处理链路：`docs/guides/content-processing.md`
- 想看编码安全：`docs/operations/encoding-safety.md`

## 文档分层

- `docs/architecture/` - 架构、目录、数据流
- `docs/guides/` - 功能使用和业务说明
- `docs/operations/` - 开发、数据库、编码安全、排障
- `docs/history/` - 经验沉淀和历史记录
- `docs/plans/` - 设计与实施计划

## 维护规则

- 文档只描述当前真实存在的目录和命令
- 根目录不再堆积功能说明文档
- 新文档先判断该放在 `architecture`、`guides`、`operations` 还是 `history`
