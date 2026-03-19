package database

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestRunMigrationsBackfillsRenamedSummaryColumns(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	DB = db

	if err := DB.Exec(`
		CREATE TABLE feeds (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT,
			url TEXT,
			content_completion_enabled BOOLEAN DEFAULT 0,
			firecrawl_enabled BOOLEAN DEFAULT 0,
			completion_on_refresh BOOLEAN DEFAULT 1,
			max_completion_retries INTEGER DEFAULT 3
		)
	`).Error; err != nil {
		t.Fatalf("create legacy feeds table: %v", err)
	}

	if err := DB.Exec(`
		CREATE TABLE articles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			feed_id INTEGER NOT NULL,
			title TEXT,
			content TEXT,
			content_status VARCHAR(20) DEFAULT 'complete',
			content_fetched_at DATETIME,
			completion_attempts INTEGER DEFAULT 0,
			completion_error TEXT,
			ai_content_summary TEXT,
			firecrawl_status VARCHAR(20) DEFAULT 'pending',
			firecrawl_error TEXT,
			firecrawl_content TEXT,
			firecrawl_crawled_at DATETIME,
			created_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create legacy articles table: %v", err)
	}

	if err := DB.Exec(`
		INSERT INTO feeds (id, title, url, content_completion_enabled, firecrawl_enabled, completion_on_refresh, max_completion_retries)
		VALUES (1, 'Feed', 'https://example.com/rss', 1, 1, 1, 5)
	`).Error; err != nil {
		t.Fatalf("seed legacy feed: %v", err)
	}

	if err := DB.Exec(`
		INSERT INTO articles (id, feed_id, title, content, content_status, content_fetched_at, completion_attempts, completion_error, ai_content_summary, firecrawl_status, firecrawl_content)
		VALUES (1, 1, 'Article', 'body', 'failed', '2026-03-19 10:00:00', 2, 'boom', 'summary', 'completed', 'crawl body')
	`).Error; err != nil {
		t.Fatalf("seed legacy article: %v", err)
	}

	if err := runMigrations(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	if !columnExists("feeds", "article_summary_enabled") {
		t.Fatal("expected article_summary_enabled column to exist")
	}
	if !columnExists("articles", "summary_status") {
		t.Fatal("expected summary_status column to exist")
	}
	if !columnExists("articles", "summary_generated_at") {
		t.Fatal("expected summary_generated_at column to exist")
	}

	var articleSummaryEnabled int
	if err := DB.Raw("SELECT article_summary_enabled FROM feeds WHERE id = 1").Scan(&articleSummaryEnabled).Error; err != nil {
		t.Fatalf("load article_summary_enabled: %v", err)
	}
	if articleSummaryEnabled != 1 {
		t.Fatalf("article_summary_enabled = %d, want 1", articleSummaryEnabled)
	}

	var summaryStatus string
	if err := DB.Raw("SELECT summary_status FROM articles WHERE id = 1").Scan(&summaryStatus).Error; err != nil {
		t.Fatalf("load summary_status: %v", err)
	}
	if summaryStatus != "failed" {
		t.Fatalf("summary_status = %q, want failed", summaryStatus)
	}

	var summaryGeneratedAt string
	if err := DB.Raw("SELECT summary_generated_at FROM articles WHERE id = 1").Scan(&summaryGeneratedAt).Error; err != nil {
		t.Fatalf("load summary_generated_at: %v", err)
	}
	if summaryGeneratedAt == "" {
		t.Fatal("expected summary_generated_at to be backfilled")
	}
}
