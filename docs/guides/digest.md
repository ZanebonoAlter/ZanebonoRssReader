# Digest 指南

## 功能说明

Digest 用来生成每日或每周新闻汇总，并支持飞书推送、Obsidian 导出和 Open Notebook 发送。

## 基本配置

1. 进入 Digest 页面或设置面板
2. 配置 daily / weekly 生成时间
3. 按需开启飞书推送（支持摘要模式和详情模式）
4. 按需开启 Obsidian 导出
5. 按需开启 Open Notebook 发送

## 飞书配置

1. 在飞书群里创建机器人
2. 复制 Webhook URL
3. 填入 Digest 配置
4. 选择推送格式（摘要 / 详情卡片）
5. 用测试接口验证

## Obsidian 配置

1. 准备一个 Vault
2. 填写 Vault 路径
3. 用测试写入验证

## Open Notebook 配置

1. 填写 Open Notebook 服务 Base URL
2. 填写 API Key 和 Model
3. 选择目标笔记本
4. 选择 Prompt 模式（默认 `digest_summary`）
5. 启用日报/周报自动发送

## 后端相关命令

```bash
go run cmd/migrate-digest/main.go
go run cmd/test-digest/main.go
```

## Digest 配置模型

Digest 配置存储在 `digest_configs` 表（`DigestConfig` 模型），主要字段：

- `daily_enabled` / `daily_time` — 日报开关和生成时间
- `weekly_enabled` / `weekly_day` / `weekly_time` — 周报开关、星期和时间
- `feishu_enabled` / `feishu_webhook_url` / `feishu_push_summary` / `feishu_push_details` — 飞书推送配置
- `obsidian_enabled` / `obsidian_vault_path` / `obsidian_daily_digest` / `obsidian_weekly_digest` — Obsidian 导出配置

Open Notebook 配置单独存储在 `ai_settings` 表的 `open_notebook_config` 键中。

## 相关接口

### 配置

- `GET /api/digest/config`
- `PUT /api/digest/config`
- `GET /api/digest/status`

### 预览与执行

- `GET /api/digest/preview/:type` — 预览日报/周报
- `POST /api/digest/run/:type` — 立即执行

### 测试

- `POST /api/digest/test-feishu`
- `POST /api/digest/test-obsidian`

### Open Notebook

- `GET /api/digest/open-notebook/config`
- `PUT /api/digest/open-notebook/config`
- `POST /api/digest/open-notebook/:type`
