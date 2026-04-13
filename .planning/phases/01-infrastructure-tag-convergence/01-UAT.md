---
status: complete
phase: 01-infrastructure-tag-convergence
source: 01-01-SUMMARY.md, 01-02-SUMMARY.md, 01-03-SUMMARY.md
started: 2026-04-13T16:00:00Z
updated: 2026-04-13T21:36:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Cold Start Smoke Test
expected: 停止后端服务，清除临时状态（缓存、锁文件等）。从零启动 Go 后端 (go run cmd/server/main.go)。服务器启动无错误，数据库 migration 自动完成（embedding_config 表、pgvector HNSW 索引、status/merged_into_id 列），基础健康检查请求返回正常响应。
result: issue
reported: "ERROR: null value in column \"driver\" of relation \"schema_migrations\" violates not-null constraint (SQLSTATE 23502)"
severity: blocker
fixed: "migrator.go — added driver column to INSERT and CREATE TABLE DDL"

### 2. Embedding 配置 API 可读写
expected: GET /api/embedding/config 返回 4 条默认配置项（thresholds、model、dimension 等）。PUT /api/embedding/config/:key 可更新某个配置值（如收敛阈值），再次 GET 确认值已更新。PUT 非法值（如阈值 >1.0）返回验证错误。
result: issue
reported: "后端 API 已实现但缺少前端配置界面，用户无法通过 Web UI 查看和修改 embedding 配置"
severity: major

### 3. 新文章入库时语义相近标签自动复用
expected: 添加一篇新文章触发标签提取时，如果已存在语义高度相近的标签（相似度 ≥ 阈值），系统复用已有标签而不创建新标签。检查标签列表确认没有产生语义重复标签。
result: pass
note: "修复后验证通过 — ensureVectorDimension 自动将列从 vector(1536) 改为 vector(2560)，embedding 成功入库"

### 4. Embedding 不可用时优雅降级
expected: 当 embedding provider 不可用（API key 未配置或服务不可达）时，新文章标签提取仍正常工作，使用精确匹配（slug+category），不报错不中断。日志中有 WARN 级别的 fallback 提示。
result: pass

### 5. 标签合并后关联引用正确迁移
expected: 对一个标签执行合并操作后，源标签的所有文章关联（article_topic_tags）迁移到目标标签。源标签状态变为 merged，merged_into_id 指向目标标签。旧标签保留在数据库中（非物理删除）。
result: issue
reported: "标签合并应该是自动的，或者有个按钮，现在啥都没有"
severity: major

### 6. 合并标签在匹配查询中被过滤
expected: 已合并（status=merged）的标签不会出现在标签匹配结果中。TagMatch 和 FindSimilarTags 都过滤掉 merged 状态的标签，新文章入库不会匹配到已合并标签。
result: blocked
blocked_by: prior-phase
reason: "依赖 Test 5 标签合并功能先完成，目前没有合并过的标签可验证"

### 7. 新建标签异步生成 embedding
expected: 创建新标签后，标签立即可用（不阻塞）。后台异步生成 embedding 并存入 pgvector 列。稍后查询该标签的 embedding 数据确认已生成。生成失败不影响标签使用。
result: pass

## Summary

total: 7
passed: 3
issues: 3
pending: 0
skipped: 0

## Gaps

- truth: "从零启动 Go 后端，服务器启动无错误，数据库 migration 自动完成"
  status: fixed
  reason: "User reported: ERROR: null value in column \"driver\" of relation \"schema_migrations\" violates not-null constraint (SQLSTATE 23502) — migration 20260413_0001 INSERT INTO schema_migrations (version) VALUES ('20260413_0001') failed, server exit status 1"
  severity: blocker
  test: 1
  root_cause: "migrator.go:42 INSERT INTO schema_migrations (version) VALUES (?) 缺少 driver 列。schema_migrations 表由其他迁移框架创建，含 driver (NOT NULL) 列且 PK 为 (driver, version)。ensureSchemaMigrationsTable 的 CREATE TABLE IF NOT EXISTS 因表已存在被跳过，未检测到额外列。"
  artifacts:
    - path: "backend-go/internal/platform/database/migrator.go"
      issue: "INSERT INTO schema_migrations (version) VALUES (?) — missing driver column"
  missing:
    - "Change INSERT to include driver column: INSERT INTO schema_migrations (version, driver) VALUES (?, 'postgres')"
    - "Also update ensureSchemaMigrationsTable to detect existing table schema or skip if table already has driver column"

- truth: "Embedding 配置可通过前端界面查看和修改"
  status: failed
  reason: "User reported: 后端 API 已实现但缺少前端配置界面，用户无法通过 Web UI 查看和修改 embedding 配置"
  severity: major
  test: 2
  root_cause: "Phase 1 只实现了后端 API (embedding_config_handler.go)，未规划配套前端配置页面"
  artifacts:
    - path: "backend-go/internal/domain/topicanalysis/embedding_config_handler.go"
      issue: "API exists but no frontend consumer"
  missing:
    - "Frontend settings page section for embedding config (threshold, model, dimension)"

- truth: "标签合并操作可通过 UI 触发（自动或手动按钮）"
  status: failed
  reason: "User reported: 标签合并应该是自动的，或者有个按钮，现在啥都没有"
  severity: major
  test: 5

