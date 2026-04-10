package database

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/config"
)

func TestInitDBUsesSQLiteDriver(t *testing.T) {
	fakeDB, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open fake db: %v", err)
	}

	originalDB := DB
	originalOpenSQLite := openSQLite
	originalOpenPostgres := openPostgres
	originalRunDatabaseMigrations := runDatabaseMigrations
	t.Cleanup(func() {
		DB = originalDB
		openSQLite = originalOpenSQLite
		openPostgres = originalOpenPostgres
		runDatabaseMigrations = originalRunDatabaseMigrations
	})

	sqliteCalled := false
	postgresCalled := false
	openSQLite = func(cfg *config.Config, gormCfg *gorm.Config) (*gorm.DB, error) {
		sqliteCalled = true
		return fakeDB, nil
	}
	openPostgres = func(cfg *config.Config, gormCfg *gorm.Config) (*gorm.DB, error) {
		postgresCalled = true
		return nil, fmt.Errorf("unexpected postgres call")
	}
	runDatabaseMigrations = func(db *gorm.DB, driver string) error {
		return nil
	}

	cfg := &config.Config{
		Server: config.ServerConfig{Mode: "test"},
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			DSN:    fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()),
		},
	}

	if err := InitDB(cfg); err != nil {
		t.Fatalf("InitDB returned error: %v", err)
	}

	if !sqliteCalled {
		t.Fatal("expected sqlite connector to be used")
	}
	if postgresCalled {
		t.Fatal("did not expect postgres connector to be used")
	}
	if DB != fakeDB {
		t.Fatal("expected global DB to be set from sqlite connector")
	}
}

func TestInitDBUsesPostgresDriver(t *testing.T) {
	fakeDB, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open fake db: %v", err)
	}

	originalDB := DB
	originalOpenSQLite := openSQLite
	originalOpenPostgres := openPostgres
	originalRunDatabaseMigrations := runDatabaseMigrations
	t.Cleanup(func() {
		DB = originalDB
		openSQLite = originalOpenSQLite
		openPostgres = originalOpenPostgres
		runDatabaseMigrations = originalRunDatabaseMigrations
	})

	sqliteCalled := false
	postgresCalled := false
	openSQLite = func(cfg *config.Config, gormCfg *gorm.Config) (*gorm.DB, error) {
		sqliteCalled = true
		return nil, fmt.Errorf("unexpected sqlite call")
	}
	openPostgres = func(cfg *config.Config, gormCfg *gorm.Config) (*gorm.DB, error) {
		postgresCalled = true
		return fakeDB, nil
	}
	runDatabaseMigrations = func(db *gorm.DB, driver string) error {
		return nil
	}

	cfg := &config.Config{
		Server: config.ServerConfig{Mode: "test"},
		Database: config.DatabaseConfig{
			Driver: "postgres",
			DSN:    "postgres://rss:rss@localhost:5432/rss_reader?sslmode=disable",
		},
	}

	if err := InitDB(cfg); err != nil {
		t.Fatalf("InitDB returned error: %v", err)
	}

	if sqliteCalled {
		t.Fatal("did not expect sqlite connector to be used")
	}
	if !postgresCalled {
		t.Fatal("expected postgres connector to be used")
	}
	if DB != fakeDB {
		t.Fatal("expected global DB to be set from postgres connector")
	}
}

func TestInitDBRejectsUnknownDriver(t *testing.T) {
	originalOpenSQLite := openSQLite
	originalOpenPostgres := openPostgres
	originalRunDatabaseMigrations := runDatabaseMigrations
	t.Cleanup(func() {
		openSQLite = originalOpenSQLite
		openPostgres = originalOpenPostgres
		runDatabaseMigrations = originalRunDatabaseMigrations
	})

	sqliteCalled := false
	postgresCalled := false
	openSQLite = func(cfg *config.Config, gormCfg *gorm.Config) (*gorm.DB, error) {
		sqliteCalled = true
		return nil, fmt.Errorf("unexpected sqlite call")
	}
	openPostgres = func(cfg *config.Config, gormCfg *gorm.Config) (*gorm.DB, error) {
		postgresCalled = true
		return nil, fmt.Errorf("unexpected postgres call")
	}
	runDatabaseMigrations = func(db *gorm.DB, driver string) error {
		return nil
	}

	cfg := &config.Config{
		Server: config.ServerConfig{Mode: "test"},
		Database: config.DatabaseConfig{
			Driver: "mysql",
			DSN:    "ignored",
		},
	}

	err := InitDB(cfg)
	if err == nil {
		t.Fatal("expected error for unknown database driver")
	}
	if !strings.Contains(err.Error(), "unknown database driver") {
		t.Fatalf("expected unknown driver error, got %v", err)
	}
	if sqliteCalled {
		t.Fatal("did not expect sqlite connector to be used")
	}
	if postgresCalled {
		t.Fatal("did not expect postgres connector to be used")
	}
}

func TestInitDBReturnsMigrationError(t *testing.T) {
	fakeDB, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open fake db: %v", err)
	}

	originalDB := DB
	originalOpenSQLite := openSQLite
	originalOpenPostgres := openPostgres
	originalRunDatabaseMigrations := runDatabaseMigrations
	t.Cleanup(func() {
		DB = originalDB
		openSQLite = originalOpenSQLite
		openPostgres = originalOpenPostgres
		runDatabaseMigrations = originalRunDatabaseMigrations
	})

	openSQLite = func(cfg *config.Config, gormCfg *gorm.Config) (*gorm.DB, error) {
		return fakeDB, nil
	}
	openPostgres = func(cfg *config.Config, gormCfg *gorm.Config) (*gorm.DB, error) {
		return nil, fmt.Errorf("unexpected postgres call")
	}
	runDatabaseMigrations = func(db *gorm.DB, driver string) error {
		return errors.New("boom")
	}

	cfg := &config.Config{
		Server: config.ServerConfig{Mode: "test"},
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			DSN:    fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()),
		},
	}

	err = InitDB(cfg)
	if err == nil {
		t.Fatal("expected migration error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected migration error to be returned, got %v", err)
	}
}

func TestRunMigrationsCreatesSchemaMigrationsTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := RunMigrations(db, "sqlite"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	if !db.Migrator().HasTable("schema_migrations") {
		t.Fatal("expected schema_migrations table to exist")
	}
}

func TestRunMigrationsAppliesEachVersionOnce(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	originalMigrationRegistry := migrationRegistry
	t.Cleanup(func() {
		migrationRegistry = originalMigrationRegistry
	})

	versionCalls := map[string]int{}
	migrationRegistry = func() []Migration {
		return []Migration{
			{
				Version: "20260403_01",
				Driver:  "sqlite",
				Up: func(db *gorm.DB) error {
					versionCalls["20260403_01"]++
					return nil
				},
			},
			{
				Version: "20260403_02",
				Driver:  "sqlite",
				Up: func(db *gorm.DB) error {
					versionCalls["20260403_02"]++
					return nil
				},
			},
		}
	}

	if err := RunMigrations(db, "sqlite"); err != nil {
		t.Fatalf("first run migrations: %v", err)
	}
	if err := RunMigrations(db, "sqlite"); err != nil {
		t.Fatalf("second run migrations: %v", err)
	}

	if versionCalls["20260403_01"] != 1 {
		t.Fatalf("migration 20260403_01 ran %d times, want 1", versionCalls["20260403_01"])
	}
	if versionCalls["20260403_02"] != 1 {
		t.Fatalf("migration 20260403_02 ran %d times, want 1", versionCalls["20260403_02"])
	}

	var appliedCount int64
	if err := db.Raw("SELECT COUNT(*) FROM schema_migrations WHERE driver = ?", "sqlite").Scan(&appliedCount).Error; err != nil {
		t.Fatalf("count applied migrations: %v", err)
	}
	if appliedCount != 2 {
		t.Fatalf("applied migration count = %d, want 2", appliedCount)
	}
}

func TestRunMigrationsRoutesByDriver(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	originalMigrationRegistry := migrationRegistry
	t.Cleanup(func() {
		migrationRegistry = originalMigrationRegistry
	})

	sqliteCalls := 0
	postgresCalls := 0
	migrationRegistry = func() []Migration {
		return []Migration{
			{
				Version: "20260403_01",
				Driver:  "sqlite",
				Up: func(db *gorm.DB) error {
					sqliteCalls++
					return nil
				},
			},
			{
				Version: "20260403_02",
				Driver:  "postgres",
				Up: func(db *gorm.DB) error {
					postgresCalls++
					return nil
				},
			},
		}
	}

	if err := RunMigrations(db, "postgres"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	if sqliteCalls != 0 {
		t.Fatalf("sqlite migrations ran %d times, want 0", sqliteCalls)
	}
	if postgresCalls != 1 {
		t.Fatalf("postgres migrations ran %d times, want 1", postgresCalls)
	}
}

func TestInitDBPassesNormalizedPostgresDriverToMigrations(t *testing.T) {
	fakeDB, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open fake db: %v", err)
	}

	originalDB := DB
	originalOpenSQLite := openSQLite
	originalOpenPostgres := openPostgres
	originalRunDatabaseMigrations := runDatabaseMigrations
	t.Cleanup(func() {
		DB = originalDB
		openSQLite = originalOpenSQLite
		openPostgres = originalOpenPostgres
		runDatabaseMigrations = originalRunDatabaseMigrations
	})

	seenDriver := ""
	openSQLite = func(cfg *config.Config, gormCfg *gorm.Config) (*gorm.DB, error) {
		return nil, fmt.Errorf("unexpected sqlite call")
	}
	openPostgres = func(cfg *config.Config, gormCfg *gorm.Config) (*gorm.DB, error) {
		return fakeDB, nil
	}
	runDatabaseMigrations = func(db *gorm.DB, driver string) error {
		seenDriver = driver
		return nil
	}

	cfg := &config.Config{
		Server: config.ServerConfig{Mode: "test"},
		Database: config.DatabaseConfig{
			Driver: "postgresql",
			DSN:    "postgres://rss:rss@localhost:5432/rss_reader?sslmode=disable",
		},
	}

	if err := InitDB(cfg); err != nil {
		t.Fatalf("InitDB returned error: %v", err)
	}

	if seenDriver != "postgres" {
		t.Fatalf("migration driver = %q, want postgres", seenDriver)
	}
}

func TestPostgresMigrationsDocumentStagedEmbeddingCutover(t *testing.T) {
	migrations := postgresMigrations()
	if len(migrations) < 3 {
		t.Fatalf("postgres migrations count = %d, want at least 3", len(migrations))
	}

	last := migrations[len(migrations)-1]
	if !strings.Contains(strings.ToLower(last.Description), "staged") {
		t.Fatalf("expected staged rollout description, got %q", last.Description)
	}
	if !strings.Contains(strings.ToLower(last.Description), "vector") {
		t.Fatalf("expected vector column description, got %q", last.Description)
	}
	if !strings.Contains(strings.ToLower(last.Description), "json") {
		t.Fatalf("expected runtime json note, got %q", last.Description)
	}
}

func TestPostgresSchemaMigrationBootstrapsEmptyDatabaseWithBaselineIndexes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	bootstrapMigration := mustFindMigration(t, postgresMigrations(), "20260403_0002")
	if err := bootstrapMigration.Up(db); err != nil {
		t.Fatalf("run postgres bootstrap migration: %v", err)
	}

	assertSQLiteIndexColumns(t, db, "articles", "idx_articles_feed_created_at", []string{"feed_id", "created_at"})
	assertSQLiteIndexColumns(t, db, "articles", "idx_articles_pub_date", []string{"pub_date"})
	assertSQLiteIndexColumns(t, db, "ai_summaries", "idx_ai_summaries_feed_created_at", []string{"feed_id", "created_at"})
	assertSQLiteIndexColumns(t, db, "ai_summaries", "idx_ai_summaries_category_created_at", []string{"category_id", "created_at"})
	assertSQLiteIndexColumns(t, db, "article_topic_tags", "idx_article_topic_tags_topic_article", []string{"topic_tag_id", "article_id"})
	assertSQLiteIndexColumns(t, db, "ai_summary_topics", "idx_ai_summary_topics_topic_summary", []string{"topic_tag_id", "summary_id"})
	assertSQLiteIndexColumns(t, db, "reading_behaviors", "idx_reading_behaviors_feed_created_at", []string{"feed_id", "created_at"})
	assertSQLiteIndexColumns(t, db, "firecrawl_jobs", "idx_firecrawl_jobs_status_available_at", []string{"status", "available_at"})
	assertSQLiteIndexColumns(t, db, "firecrawl_jobs", "idx_firecrawl_jobs_lease_expires_at", []string{"lease_expires_at"})
	assertSQLiteIndexColumns(t, db, "tag_jobs", "idx_tag_jobs_status_available_at", []string{"status", "available_at"})
	assertSQLiteIndexColumns(t, db, "tag_jobs", "idx_tag_jobs_lease_expires_at", []string{"lease_expires_at"})
}

func TestPostgresSchemaMigrationRestoresBaselineConstraints(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}

	bootstrapMigration := mustFindMigration(t, postgresMigrations(), "20260403_0002")
	if err := bootstrapMigration.Up(db); err != nil {
		t.Fatalf("run postgres bootstrap migration: %v", err)
	}

	assertSQLiteUniqueIndexColumns(t, db, "article_topic_tags", []string{"article_id", "topic_tag_id"})
	assertSQLiteForeignKeyCascade(t, db, "article_topic_tags", "article_id", "articles")
	assertSQLiteForeignKeyCascade(t, db, "article_topic_tags", "topic_tag_id", "topic_tags")
	assertSQLiteForeignKeyCascade(t, db, "firecrawl_jobs", "article_id", "articles")
	assertSQLiteForeignKeyCascade(t, db, "tag_jobs", "article_id", "articles")
}

func TestPostgresSchemaMigrationKeepsFirstBatchTextColumns(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	bootstrapMigration := mustFindMigration(t, postgresMigrations(), "20260403_0002")
	if err := bootstrapMigration.Up(db); err != nil {
		t.Fatalf("run postgres bootstrap migration: %v", err)
	}

	assertSQLiteColumnType(t, db, "ai_summaries", "articles", "TEXT")
	assertSQLiteColumnType(t, db, "ai_providers", "metadata", "TEXT")
	assertSQLiteColumnType(t, db, "topic_tag_analyses", "payload_json", "TEXT")
}

func TestTopicTagAnalysisPayloadJSONExplicitlyStaysTextInModel(t *testing.T) {
	field, ok := reflect.TypeOf(models.TopicTagAnalysis{}).FieldByName("PayloadJSON")
	if !ok {
		t.Fatal("PayloadJSON field not found")
	}

	if !strings.Contains(field.Tag.Get("gorm"), "type:text") {
		t.Fatalf("expected PayloadJSON gorm tag to keep text storage, got %q", field.Tag.Get("gorm"))
	}
}

func TestArticleTimeFieldsAvoidSQLiteOnlyDatetimeType(t *testing.T) {
	articleType := reflect.TypeOf(models.Article{})
	for _, fieldName := range []string{"PubDate", "SummaryGeneratedAt", "SummaryProcessingStartedAt", "FirecrawlCrawledAt"} {
		field, ok := articleType.FieldByName(fieldName)
		if !ok {
			t.Fatalf("%s field not found", fieldName)
		}

		if strings.Contains(strings.ToLower(field.Tag.Get("gorm")), "type:datetime") {
			t.Fatalf("expected %s to avoid sqlite-only datetime tag, got %q", fieldName, field.Tag.Get("gorm"))
		}
	}
}

func TestFeedIconFieldsAllowURLLength(t *testing.T) {
	feedField, ok := reflect.TypeOf(models.Feed{}).FieldByName("Icon")
	if !ok {
		t.Fatal("Feed.Icon field not found")
	}
	if strings.Contains(feedField.Tag.Get("gorm"), "size:50") {
		t.Fatalf("expected Feed.Icon to allow long URL values, got %q", feedField.Tag.Get("gorm"))
	}

	summaryFeedField, ok := reflect.TypeOf(models.AISummaryFeed{}).FieldByName("FeedIcon")
	if !ok {
		t.Fatal("AISummaryFeed.FeedIcon field not found")
	}
	if strings.Contains(summaryFeedField.Tag.Get("gorm"), "size:50") {
		t.Fatalf("expected AISummaryFeed.FeedIcon to allow long URL values, got %q", summaryFeedField.Tag.Get("gorm"))
	}
}

func TestPostgresBootstrapExpandsLegacyShortIconColumns(t *testing.T) {
	statements := postgresColumnAdjustmentStatements()
	joined := strings.Join(statements, "\n")

	if !strings.Contains(joined, "ALTER TABLE feeds ALTER COLUMN icon TYPE VARCHAR(1000)") {
		t.Fatalf("expected feeds.icon widening statement, got %q", joined)
	}
	if !strings.Contains(joined, "ALTER TABLE ai_summary_feeds ALTER COLUMN feed_icon TYPE VARCHAR(1000)") {
		t.Fatalf("expected ai_summary_feeds.feed_icon widening statement, got %q", joined)
	}
}

func TestSQLiteLegacyBootstrapStillWorksForExistingTests(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.Exec(`
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

	if err := db.Exec(`
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

	if err := db.Exec(`
		INSERT INTO feeds (id, title, url, content_completion_enabled, firecrawl_enabled, completion_on_refresh, max_completion_retries)
		VALUES (1, 'Feed', 'https://example.com/rss', 1, 1, 1, 5)
	`).Error; err != nil {
		t.Fatalf("seed legacy feed: %v", err)
	}

	if err := db.Exec(`
		INSERT INTO articles (id, feed_id, title, content, content_status, content_fetched_at, completion_attempts, completion_error, ai_content_summary, firecrawl_status, firecrawl_content)
		VALUES (1, 1, 'Article', 'body', 'failed', '2026-03-19 10:00:00', 2, 'boom', 'summary', 'completed', 'crawl body')
	`).Error; err != nil {
		t.Fatalf("seed legacy article: %v", err)
	}

	if err := RunMigrations(db, "sqlite"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	if !sqliteColumnExists(db, "feeds", "article_summary_enabled") {
		t.Fatal("expected article_summary_enabled column to exist")
	}
	if !sqliteColumnExists(db, "articles", "summary_status") {
		t.Fatal("expected summary_status column to exist")
	}
	if !sqliteColumnExists(db, "articles", "summary_generated_at") {
		t.Fatal("expected summary_generated_at column to exist")
	}

	var articleSummaryEnabled int
	if err := db.Raw("SELECT article_summary_enabled FROM feeds WHERE id = 1").Scan(&articleSummaryEnabled).Error; err != nil {
		t.Fatalf("load article_summary_enabled: %v", err)
	}
	if articleSummaryEnabled != 1 {
		t.Fatalf("article_summary_enabled = %d, want 1", articleSummaryEnabled)
	}

	var summaryStatus string
	if err := db.Raw("SELECT summary_status FROM articles WHERE id = 1").Scan(&summaryStatus).Error; err != nil {
		t.Fatalf("load summary_status: %v", err)
	}
	if summaryStatus != "failed" {
		t.Fatalf("summary_status = %q, want failed", summaryStatus)
	}

	var summaryGeneratedAt string
	if err := db.Raw("SELECT summary_generated_at FROM articles WHERE id = 1").Scan(&summaryGeneratedAt).Error; err != nil {
		t.Fatalf("load summary_generated_at: %v", err)
	}
	if summaryGeneratedAt == "" {
		t.Fatal("expected summary_generated_at to be backfilled")
	}
}

func TestMigrateCreatesJobQueueTables(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	DB = db
	if err := Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if !DB.Migrator().HasTable(&models.FirecrawlJob{}) {
		t.Fatal("expected firecrawl_jobs table to exist")
	}
	if !DB.Migrator().HasTable(&models.TagJob{}) {
		t.Fatal("expected tag_jobs table to exist")
	}
}

func mustFindMigration(t *testing.T, migrations []Migration, version string) Migration {
	t.Helper()

	for _, migration := range migrations {
		if migration.Version == version {
			return migration
		}
	}

	t.Fatalf("migration %s not found", version)
	return Migration{}
}

func assertSQLiteIndexColumns(t *testing.T, db *gorm.DB, tableName string, indexName string, wantColumns []string) {
	t.Helper()

	indexInfoRows, err := db.Raw(fmt.Sprintf("PRAGMA index_info('%s')", indexName)).Rows()
	if err != nil {
		t.Fatalf("load index info for %s: %v", indexName, err)
	}
	defer indexInfoRows.Close()

	type indexInfo struct {
		Seqno int
		Cid   int
		Name  string
	}

	columns := make([]string, 0, len(wantColumns))
	for indexInfoRows.Next() {
		var row indexInfo
		if err := db.ScanRows(indexInfoRows, &row); err != nil {
			t.Fatalf("scan index info for %s: %v", indexName, err)
		}
		columns = append(columns, row.Name)
	}

	if len(columns) == 0 {
		t.Fatalf("expected index %s on table %s to exist", indexName, tableName)
	}
	if len(columns) != len(wantColumns) {
		t.Fatalf("index %s columns = %v, want %v", indexName, columns, wantColumns)
	}
	for i := range wantColumns {
		if columns[i] != wantColumns[i] {
			t.Fatalf("index %s columns = %v, want %v", indexName, columns, wantColumns)
		}
	}
}

func assertSQLiteUniqueIndexColumns(t *testing.T, db *gorm.DB, tableName string, wantColumns []string) {
	t.Helper()

	rows, err := db.Raw(fmt.Sprintf("PRAGMA index_list('%s')", tableName)).Rows()
	if err != nil {
		t.Fatalf("load index list for %s: %v", tableName, err)
	}
	defer rows.Close()

	type indexListRow struct {
		Seq     int
		Name    string
		Unique  int
		Origin  string
		Partial int
	}

	for rows.Next() {
		var row indexListRow
		if err := db.ScanRows(rows, &row); err != nil {
			t.Fatalf("scan index list for %s: %v", tableName, err)
		}
		if row.Unique != 1 {
			continue
		}

		indexInfoRows, err := db.Raw(fmt.Sprintf("PRAGMA index_info('%s')", row.Name)).Rows()
		if err != nil {
			t.Fatalf("load index info for %s: %v", row.Name, err)
		}

		columns := make([]string, 0, len(wantColumns))
		for indexInfoRows.Next() {
			var info struct {
				Seqno int
				Cid   int
				Name  string
			}
			if err := db.ScanRows(indexInfoRows, &info); err != nil {
				indexInfoRows.Close()
				t.Fatalf("scan index info for %s: %v", row.Name, err)
			}
			columns = append(columns, info.Name)
		}
		indexInfoRows.Close()

		if reflect.DeepEqual(columns, wantColumns) {
			return
		}
	}

	t.Fatalf("expected unique index on %s with columns %v", tableName, wantColumns)
}

func assertSQLiteForeignKeyCascade(t *testing.T, db *gorm.DB, tableName string, fromColumn string, targetTable string) {
	t.Helper()

	rows, err := db.Raw(fmt.Sprintf("PRAGMA foreign_key_list('%s')", tableName)).Rows()
	if err != nil {
		t.Fatalf("load foreign keys for %s: %v", tableName, err)
	}
	defer rows.Close()

	type foreignKeyRow struct {
		ID       int
		Seq      int
		Table    string
		From     string
		To       string
		OnUpdate string
		OnDelete string
		Match    string
	}

	for rows.Next() {
		var row foreignKeyRow
		if err := db.ScanRows(rows, &row); err != nil {
			t.Fatalf("scan foreign keys for %s: %v", tableName, err)
		}
		if row.From == fromColumn && row.Table == targetTable {
			if !strings.EqualFold(row.OnDelete, "cascade") {
				t.Fatalf("foreign key %s.%s delete action = %q, want cascade", tableName, fromColumn, row.OnDelete)
			}
			return
		}
	}

	t.Fatalf("expected foreign key on %s.%s referencing %s", tableName, fromColumn, targetTable)
}

func assertSQLiteColumnType(t *testing.T, db *gorm.DB, tableName string, columnName string, wantType string) {
	t.Helper()

	rows, err := db.Raw(fmt.Sprintf("PRAGMA table_info('%s')", tableName)).Rows()
	if err != nil {
		t.Fatalf("load table info for %s: %v", tableName, err)
	}
	defer rows.Close()

	type tableInfo struct {
		Cid       int
		Name      string
		Type      string
		NotNull   int
		DfltValue *string
		Pk        int
	}

	for rows.Next() {
		var row tableInfo
		if err := db.ScanRows(rows, &row); err != nil {
			t.Fatalf("scan table info for %s: %v", tableName, err)
		}
		if row.Name == columnName {
			if !strings.EqualFold(row.Type, wantType) {
				t.Fatalf("column %s.%s type = %s, want %s", tableName, columnName, row.Type, wantType)
			}
			return
		}
	}

	t.Fatalf("column %s.%s not found", tableName, columnName)
}
