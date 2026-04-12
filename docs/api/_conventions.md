# API 通用约定

基路径：`/api`，后端默认 `http://localhost:5000`。

## 响应格式

所有接口返回 JSON，统一信封：

```json
{
  "success": true,
  "data": { ... },
  "message": "操作描述（可选）"
}
```

错误响应：

```json
{
  "success": false,
  "error": "错误描述"
}
```

## 分页

列表接口支持 `page`（默认 `1`）和 `per_page`（默认 `20`）：

```json
{
  "success": true,
  "data": [ ... ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 150,
    "pages": 8
  }
}
```

## 认证

个人/单用户部署，无认证系统。

## WebSocket

实时端点：`ws://localhost:5000/ws`，用于 AI 总结进度推送等。
