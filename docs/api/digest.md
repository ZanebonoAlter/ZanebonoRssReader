# Digest 汇总

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/digest/config` | 获取配置 |
| PUT | `/api/digest/config` | 更新配置 |
| GET | `/api/digest/status` | 运行状态 |
| GET | `/api/digest/preview/:type` | 预览 |
| POST | `/api/digest/run/:type` | 立即执行 |
| GET | `/api/digest/open-notebook/config` | Open Notebook 配置 |
| PUT | `/api/digest/open-notebook/config` | 更新 Open Notebook 配置 |
| POST | `/api/digest/open-notebook/:type` | 发送到 Open Notebook |
| POST | `/api/digest/test-feishu` | 测试飞书推送 |
| POST | `/api/digest/test-obsidian` | 测试 Obsidian 写入 |

---

### GET /api/digest/config

```json
{
  "success": true,
  "data": {
    "daily_enabled": false,
    "daily_time": "09:00",
    "weekly_enabled": false,
    "weekly_day": 1,
    "weekly_time": "09:00",
    "feishu_enabled": false,
    "feishu_webhook_url": "",
    "feishu_push_summary": true,
    "feishu_push_details": false,
    "obsidian_enabled": false,
    "obsidian_vault_path": "",
    "obsidian_daily_digest": true,
    "obsidian_weekly_digest": true
  }
}
```

### PUT /api/digest/config

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `daily_enabled` | bool | 是 | 启用日报 |
| `daily_time` | string | 是 | `HH:MM` |
| `weekly_enabled` | bool | 是 | 启用周报 |
| `weekly_day` | int | 是 | 0-6，0=周日 |
| `weekly_time` | string | 是 | `HH:MM` |
| `feishu_enabled` | bool | 是 | 启用飞书 |
| `feishu_webhook_url` | string | 是 | Webhook URL |
| `feishu_push_summary` | bool | 是 | 推送摘要 |
| `feishu_push_details` | bool | 是 | 推送详情卡片 |
| `obsidian_enabled` | bool | 是 | 启用 Obsidian |
| `obsidian_vault_path` | string | 是 | Vault 路径 |
| `obsidian_daily_digest` | bool | 是 | 导出日报 |
| `obsidian_weekly_digest` | bool | 是 | 导出周报 |

### GET /api/digest/status

Digest 调度器运行状态。

### GET /api/digest/preview/:type

`type`：`daily` 或 `weekly`。查询参数：`date`（`YYYY-MM-DD`，可选）。

返回 `type`, `title`, `period_label`, `generated_at`, `anchor_date`, `category_count`, `summary_count`, `markdown`, `categories` 等。

### POST /api/digest/run/:type

`type`：`daily` 或 `weekly`。查询参数：`date`（可选）。

```json
{
  "success": true,
  "message": "已执行当前 digest 流程",
  "data": {
    "preview": { ... },
    "sent_to_feishu": true,
    "exported_to_obsidian": false,
    "sent_to_open_notebook": false
  }
}
```

### GET /api/digest/open-notebook/config

Open Notebook 集成配置。

### PUT /api/digest/open-notebook/config

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `enabled` | bool | 是 | 是否启用 |
| `base_url` | string | 是 | 地址 |
| `api_key` | string | 否 | API Key |
| `model` | string | 否 | 模型名 |
| `target_notebook` | string | 否 | 目标笔记本 |
| `prompt_mode` | string | 否 | 默认 `digest_summary` |
| `auto_send_daily` | bool | 是 | 自动发送日报 |
| `auto_send_weekly` | bool | 是 | 自动发送周报 |
| `export_back_to_obsidian` | bool | 是 | 导出到 Obsidian |

### POST /api/digest/open-notebook/:type

手动发送到 Open Notebook。`type`：`daily`/`weekly`。查询参数：`date`（可选）。

### POST /api/digest/test-feishu

测试飞书 Webhook。可选 body：`{ "webhook_url": "..." }`。留空用已保存的。

### POST /api/digest/test-obsidian

测试 Obsidian 写入。可选 body：`{ "vault_path": "..." }`。留空用已保存的。
