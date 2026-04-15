# 260414 Summary 文章级去重标记

## 背景

- `auto_summary` 和 `summary_queue` 之前都按 `ai_summaries.articles` 的整批文章 ID JSON 做去重。
- 只要时间窗内多出一篇新文章，整批文章集合就会变化，旧文章会再次被聚合进新 summary。
- 这会让高频更新 feed 在 180 分钟时间窗内反复总结同一批旧文章。

## 变更

- 在 `articles` 表新增两个字段：
  - `feed_summary_id`
  - `feed_summary_generated_at`
- `auto_summary` 和 `summary_queue` 现在只查询 `feed_summary_generated_at IS NULL` 的文章作为候选。
- 当命中已有 summary batch 时，不再只跳过 AI 调用，还会把这批文章回填到已有 summary 标记上。
- 当创建新 summary 成功后，立即把对应文章写上 summary 标记，后续调度不会再重复聚合这些文章。

## 数据库迁移

- 新增 PostgreSQL 迁移 `20260414_0003`
- SQL 内容：
  - `ALTER TABLE articles ADD COLUMN IF NOT EXISTS feed_summary_id BIGINT REFERENCES ai_summaries(id)`
  - `ALTER TABLE articles ADD COLUMN IF NOT EXISTS feed_summary_generated_at TIMESTAMP`
  - `CREATE INDEX IF NOT EXISTS idx_articles_feed_summary_id ON articles(feed_summary_id)`
  - `CREATE INDEX IF NOT EXISTS idx_articles_feed_summary_generated_at ON articles(feed_summary_generated_at)`

## 影响文件

- `backend-go/internal/domain/models/article.go`
- `backend-go/internal/domain/summaries/feed_summary_articles.go`
- `backend-go/internal/jobs/auto_summary.go`
- `backend-go/internal/domain/summaries/summary_queue.go`
- `backend-go/internal/platform/database/postgres_migrations.go`
- `docs/operations/database.md`
- `docs/operations/postgres-migration.md`

## 验证

- 新增测试覆盖两条 summary 路径：
  - 已有 batch 被跳过时会回填文章标记
  - 已标记文章不会再次进入新 summary
- 通过命令：
  - `go test ./internal/jobs ./internal/domain/summaries ./internal/platform/database ./cmd/migrate-db`

## 结果

- 重复聚合的判定粒度从“整批文章集合”改成“文章是否已经参与过 feed summary”。
- 新文章到来时，只会为未标记的新文章生成 summary，不再把同时间窗里的旧文章反复带入。
