# Open Notebook Digest Integration Design

**Date:** 2026-03-10
**Status:** Approved

## Goal

把现有 digest 生成出的 Markdown 交给 open-notebook 做二次总结。

第一版先做手动触发和结果回显。
稳定后再接入自动运行。

## Why This Route

- 现有 digest 链路已经能稳定产出 Markdown
- digest 页面已经有配置、预览、手动运行和导出能力
- 这条路线不碰现有 AI summary 主链路
- 风险更小，验证更快

## Current Architecture Fit

现有关键节点：

- digest preview 构建：`backend-go/internal/domain/digest/handler.go`
- digest markdown 拼装：`backend-go/internal/domain/digest/handler.go`
- digest 手动执行：`backend-go/internal/domain/digest/handler.go`
- Obsidian 导出：`backend-go/internal/domain/digest/obsidian.go`
- digest 前端 API：`front/app/api/digest.ts`
- digest 视图：`front/app/features/digest/components/DigestListView.vue`
- digest 设置：`front/app/features/digest/components/DigestSettings.vue`

## Recommended Architecture

### 1. Source Of Truth

open-notebook 的输入直接使用 digest preview 产出的 `preview.markdown`。

这样有几个好处：

- 不重复拼装内容
- 手动运行和预览内容保持一致
- 后续自动化也能复用同一份输入

### 2. New Backend Layer

新增平台客户端层：

- `backend-go/internal/platform/opennotebook/client.go`

职责：

- 封装 open-notebook HTTP 调用
- 统一请求超时、鉴权、错误解析
- 返回结构化结果给 digest 域

### 3. Digest Domain Integration

在 digest 域新增一层编排：

- 读取当前 digest preview
- 组装 open-notebook 请求
- 返回二次总结内容
- 后续可选导出到 Obsidian

建议先新增 handler，不急着加复杂 service。

### 4. Configuration Strategy

open-notebook 配置不要硬塞进 `digest_configs`。

建议复用现有 `AISettings` 存储方式，新增独立 key：

- `open_notebook_config`

建议字段：

- `enabled`
- `base_url`
- `api_key`
- `model`
- `target_notebook`
- `prompt_mode`
- `auto_send_daily`
- `auto_send_weekly`
- `export_back_to_obsidian`

这样做的原因：

- digest 配置仍然只管调度、飞书、Obsidian
- open-notebook 保持成独立外部集成
- 后续替换供应方时影响更小

## Phase Plan

### Phase 1: Manual Send + UI Feedback

目标：先验证 API 和结果质量。

后端新增：

- `GET /api/digest/open-notebook/config`
- `PUT /api/digest/open-notebook/config`
- `POST /api/digest/open-notebook/:type`

请求参数：

- `type`: `daily` | `weekly`
- query `date`: `YYYY-MM-DD`

行为：

1. 后端调用 `buildPreview(type, anchorDate)`
2. 取 `preview.markdown`
3. 发送给 open-notebook
4. 返回：
   - 原始 preview 元信息
   - open-notebook 的二次总结
   - 远端返回的元数据

前端新增：

- digest 设置区增加 open-notebook 配置块
- digest 列表页增加 “发送到 Open Notebook” 按钮
- digest 详情区增加 “二次总结结果” 面板

第一版先不落库。

### Phase 2: Manual Run Hook

目标：让 `RunDigestNow()` 跑完后可选自动发送。

行为：

- 如果 open-notebook 对应开关已开
- 在 digest 手动执行成功后自动把当前 markdown 发过去
- 响应里返回 `sent_to_open_notebook`

### Phase 3: Scheduled Auto Send

目标：接入 daily / weekly scheduler。

行为：

- 每日 digest 成功后自动发送 daily
- 每周 digest 成功后自动发送 weekly
- 失败不阻断现有飞书和 Obsidian 链路
- 错误写日志，状态暴露给前端

### Phase 4: History And Retry

目标：补运行记录，方便排错。

可新增表：`digest_open_notebook_runs`

建议字段：

- `id`
- `digest_type`
- `anchor_date`
- `status`
- `source_markdown_hash`
- `request_excerpt`
- `response_excerpt`
- `remote_id`
- `remote_url`
- `error_message`
- `created_at`

## Prompt/Product Shape

open-notebook 第一版定位成“二次浓缩摘要”。

不要替代原 digest。
不要覆盖原 Markdown。

建议固定输出结构：

1. 一句话结论
2. 今天/本周最值得看
3. 重复出现的话题
4. 值得后续追踪
5. 原文入口

这样能同时满足：

- 飞书快读
- notebook 沉淀
- 后续首页摘要卡片

## API Sketch

### `GET /api/digest/open-notebook/config`

返回 open-notebook 配置。

### `PUT /api/digest/open-notebook/config`

保存 open-notebook 配置。

### `POST /api/digest/open-notebook/:type?date=YYYY-MM-DD`

返回：

```json
{
  "success": true,
  "data": {
    "digest_type": "daily",
    "anchor_date": "2026-03-10",
    "source_markdown": "# 今日日报...",
    "summary_markdown": "# 今日一页版...",
    "remote_id": "optional",
    "remote_url": "optional"
  }
}
```

## Error Handling

- 配置缺失：返回 400，提示先配置 open-notebook
- digest 没内容：仍允许发送，但 UI 提示“当前时间窗没有总结”
- open-notebook 失败：返回清晰错误，不影响原 digest 预览
- 自动发送失败：不阻断 digest 主流程，只记录失败

## Risks

### Low Risk

- 手动发送现有 preview markdown
- 独立配置读写
- 结果在前端单独展示

### Medium Risk

- 自动接入 `RunDigestNow()`
- 自动接入 scheduler 状态反馈

### High Risk

- 把 open-notebook 混进 AI summary 队列
- 让 open-notebook 替代原 digest 存储模型

第一版不要碰高风险项。

## Testing Strategy

后端：

- client 请求单测
- digest open-notebook handler 单测
- 配置读写单测

前端：

- 手动验证设置保存
- 手动验证发送按钮
- 手动验证结果面板渲染

## Success Criteria

- 用户能在 digest 页面配置 open-notebook
- 用户能把当前 daily/weekly digest 发送给 open-notebook
- 页面能看到返回的二次总结
- 原始 digest、飞书、Obsidian 功能不受影响
- 失败时能知道卡在哪
