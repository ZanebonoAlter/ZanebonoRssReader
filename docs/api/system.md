# 系统信息

### GET /

返回 API 名称和版本。

```json
{
  "name": "RSS Reader API (Go)",
  "version": "1.0.0",
  "endpoints": { ... }
}
```

### GET /health

健康检查。

```json
{
  "status": "healthy",
  "database": "connected"
}
```
