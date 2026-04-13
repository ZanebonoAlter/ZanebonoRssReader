# 260413-p2t 后端队列 Tab 与标签合并重算队列

## 结论
- 设置弹窗新增独立“后端队列”Tab，不再把队列内容放在“通用设置”里。
- 后端现在同时维护两套独立 backlog：原始 embedding 队列，以及 `MergeTags` 成功后触发的 merge re-embedding 队列。
- `MergeTags(source, target)` 只会在事务提交成功后为 surviving `target` 入队，避免 worker 读取半提交状态。

## 后端实现要点
- 新表：`merge_reembedding_queues`
  - 字段：`source_tag_id`、`target_tag_id`、`status`、`error_message`、`retry_count`、时间戳
  - 索引：按 `status`、`source_tag_id`、`target_tag_id`
- 新服务：`MergeReembeddingQueueService`
  - `Enqueue(sourceTagID, targetTagID)`
  - `GetStatus()` / `GetTasks()` / `RetryFailed()`
  - `Start()` / `Stop()` / worker loop
- 去重规则：同一个 `target_tag_id` 只要还有 `pending/processing` 任务，就不再重复创建新任务。
- worker 处理逻辑：加载目标标签 → 重新生成 embedding → `SaveEmbedding` 覆盖目标标签 embedding → 更新任务状态。

## API 契约
- `GET /api/embedding/merge-reembedding/status`
- `GET /api/embedding/merge-reembedding/tasks?status=&limit=&offset=`
- `POST /api/embedding/merge-reembedding/retry`

返回风格与现有后端保持一致：`success/data/error/message`。

## MergeTags 链路变化
- 原有 5 步事务仍保持不变：迁移 article tags、迁移 summary tags、标记 source merged、删除 source embedding、重算 target feed_count。
- 新增的重算入队发生在事务 `commit` 之后，而不是事务内部。
- 如果入队失败，`MergeTags` 会返回错误，不会“假成功”。

## 前端呈现
- 新增 `MergeReembeddingQueuePanel.vue`，展示：
  - status cards
  - 轮询刷新
  - 状态筛选
  - source tag / target tag / 时间 / 错误信息表格
  - failed retry 按钮
- `GlobalSettingsDialog.vue` 中新增 `backend-queues` Tab，同时渲染：
  - `EmbeddingQueuePanel`
  - `MergeReembeddingQueuePanel`

## 回归验证
- `cd backend-go && go test ./internal/domain/topicanalysis -run "TestMergeReembeddingQueue|TestMergeTags.*Reembedding" -v`
- `cd backend-go && go build ./...`
- `cd front && pnpm exec nuxi typecheck && pnpm build`
