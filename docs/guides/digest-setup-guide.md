# 日报周报功能使用指南

## 功能概述

日报周报功能可以自动生成每日/每周的新闻汇总，并支持飞书推送和Obsidian导出。

## 配置步骤

### 1. 基础设置

1. 进入应用，点击左侧"日报周报"菜单
2. 点击"设置"按钮
3. 配置日报/周报的生成时间

### 2. 飞书推送

1. 在飞书群组中添加自定义机器人
2. 复制Webhook URL
3. 在设置中粘贴URL并启用推送
4. 点击"测试推送"验证

### 3. Obsidian导出

1. 安装Obsidian并创建Vault
2. 在设置中填写Vault完整路径
3. 点击"测试写入"验证
4. 启用日报/周报导出

## 文件结构

 导出到Obsidian的文件结构如下：

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

1. 检查Webhook URL是否正确
2. 确认机器人是否被移除
3. 查看后端日志

### Obsidian写入失败怎么办？

1. 检查路径是否有写入权限
2. 确认路径是否存在
3. 尝试使用绝对路径

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

## API端点

- `GET /api/digest/config` - 获取配置
- `PUT /api/digest/config` - 更新配置
- `POST /api/digest/test-feishu` - 测试飞书推送
- `POST /api/digest/test-obsidian` - 测试Obsidian写入
