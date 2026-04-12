# 定时任务 Schedulers

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/schedulers/status` | 所有调度器状态 |
| GET | `/api/schedulers/:name/status` | 指定调度器状态 |
| POST | `/api/schedulers/:name/trigger` | 手动触发 |
| POST | `/api/schedulers/:name/reset` | 重置统计 |
| PUT | `/api/schedulers/:name/interval` | 更新间隔 |

---

### 支持的调度器

| 名称 | 别名 | 说明 |
|------|------|------|
| `auto_refresh` | - | 自动刷新 RSS |
| `auto_summary` | - | 自动生成 AI 总结 |
| `preference_update` | - | 更新阅读偏好 |
| `content_completion` | `ai_summary` | 文章内容补全 |
| `firecrawl` | - | Firecrawl 全文抓取 |
| `digest` | - | Digest 日报/周报 |

### PUT /api/schedulers/:name/interval

```json
{ "interval": 30 }
```

`interval`：正整数，单位取决于调度器。
