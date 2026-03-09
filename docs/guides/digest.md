# Digest 指南

## 功能说明

Digest 用来生成每日或每周新闻汇总，并支持飞书推送和 Obsidian 导出。

## 基本配置

1. 进入 digest 页面或设置面板
2. 配置 daily / weekly 生成时间
3. 按需开启飞书推送
4. 按需开启 Obsidian 导出

## 飞书配置

1. 在飞书群里创建机器人
2. 复制 Webhook URL
3. 填入 digest 配置
4. 用测试接口验证

## Obsidian 配置

1. 准备一个 vault
2. 填写 vault 路径
3. 用测试写入验证

## 后端相关命令

```bash
go run cmd/migrate-digest/main.go
go run cmd/test-digest/main.go
```

## 相关接口

- `GET /api/digest/config`
- `PUT /api/digest/config`
- `GET /api/digest/status`
- `POST /api/digest/run/:type`
- `POST /api/digest/test-feishu`
- `POST /api/digest/test-obsidian`
