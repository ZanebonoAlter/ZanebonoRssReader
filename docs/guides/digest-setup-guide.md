# 日报周报功能使用指南

## 功能概述

日报周报功能可以自动生成每日/每周的新闻汇总，并支持飞书推送、Obsidian 导出和 Open Notebook 发送。

## 配置步骤

### 1. 基础设置

1. 进入应用，点击左侧"日报周报"菜单
2. 点击"设置"按钮
3. 配置日报/周报的生成时间

### 2. 飞书推送

1. 在飞书群组中添加自定义机器人
2. 复制 Webhook URL
3. 在设置中粘贴 URL 并启用推送
4. 选择推送格式：摘要模式或详情模式
5. 点击"测试推送"验证

### 3. Obsidian 导出

1. 安装 Obsidian 并创建 Vault
2. 在设置中填写 Vault 完整路径
3. 点击"测试写入"验证
4. 启用日报/周报导出

### 4. Open Notebook 发送

1. 在设置中填写 Open Notebook 服务地址（Base URL）
2. 填写 API Key 和 Model
3. 选择目标笔记本和 Prompt 模式
4. 启用日报/周报自动发送
5. 可选：启用"回写 Obsidian"，将 Open Notebook 处理后的内容写回 Obsidian Vault

## 文件结构

导出到 Obsidian 的文件结构如下：

```
ObsidianVault/
├── Daily/
│   ├── AI技术/
│   │   └── 2026-03-04-日报.md
│   └── 前端开发/
│       └── 2026-03-04-日报.md
├── Weekly/
│   ├── AI技术/
│   │   └── 2026-W9-周报.md
│   └── 前端开发/
│       └── 2026-W9-周报.md
└── Feeds/
    ├── TechCrunch/
    │   └── 2026-03-04.md
    └── ...
```

## 常见问题

### 飞书推送失败怎么办？

1. 检查 Webhook URL 是否正确
2. 确认机器人是否被移除
3. 查看后端日志

### Obsidian 写入失败怎么办？

1. 检查路径是否有写入权限
2. 确认路径是否存在
3. 尝试使用绝对路径

### Open Notebook 发送失败怎么办？

1. 确认服务地址可达
2. 检查 API Key 是否正确
3. 确认目标笔记本存在
4. 查看后端日志中的详细错误信息

## 数据库迁移

首次使用需要运行数据库迁移：

```bash
cd backend-go
go run cmd/migrate-digest/main.go
```

## 测试数据

生成测试数据：

```bash
cd backend-go
go run cmd/test-digest/main.go
```

## API 端点

### 配置

- `GET /api/digest/config` — 获取 Digest 配置
- `PUT /api/digest/config` — 更新 Digest 配置
- `GET /api/digest/status` — 获取 Digest 运行状态

### 预览与执行

- `GET /api/digest/preview/:type` — 预览日报/周报内容（支持 `?date=YYYY-MM-DD` 参数）
- `POST /api/digest/run/:type` — 立即执行日报/周报（支持 `?date=YYYY-MM-DD` 参数）

### 测试

- `POST /api/digest/test-feishu` — 测试飞书推送
- `POST /api/digest/test-obsidian` — 测试 Obsidian 写入

### Open Notebook

- `GET /api/digest/open-notebook/config` — 获取 Open Notebook 配置
- `PUT /api/digest/open-notebook/config` — 更新 Open Notebook 配置
- `POST /api/digest/open-notebook/:type` — 手动发送 Digest 到 Open Notebook（支持 `?date=YYYY-MM-DD` 参数）
