package database

import (
	"reflect"
	"strings"
	"testing"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/config"
)

func TestInitDBConnectsToPostgres(t *testing.T) {
	if err := config.LoadConfig("../../configs"); err != nil {
		t.Fatalf("load config: %v", err)
	}

	if err := InitDB(config.AppConfig); err != nil {
		t.Fatalf("InitDB returned error: %v", err)
	}

	if DB == nil {
		t.Fatal("expected global DB to be initialized")
	}

	sqlDB, err := DB.DB()
	if err != nil {
		t.Fatalf("get underlying db: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("ping database: %v", err)
	}
}

func TestMigrateCreatesAllTables(t *testing.T) {
	if err := config.LoadConfig("../../configs"); err != nil {
		t.Fatalf("load config: %v", err)
	}

	if err := InitDB(config.AppConfig); err != nil {
		t.Fatalf("InitDB: %v", err)
	}

	if err := Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	expectedTables := []any{
		&models.FirecrawlJob{},
		&models.TagJob{},
		&models.AISettings{},
		&models.AIProvider{},
		&models.AIRoute{},
	}

	for _, table := range expectedTables {
		if !DB.Migrator().HasTable(table) {
			t.Fatalf("expected table %T to exist", table)
		}
	}
}

func TestPostgresMigrationsDocumentStagedEmbeddingCutover(t *testing.T) {
	migrations := postgresMigrations()
	if len(migrations) < 3 {
		t.Fatalf("postgres migrations count = %d, want at least 3", len(migrations))
	}

	migration := mustFindMigration(t, migrations, "20260403_0003")
	if !strings.Contains(strings.ToLower(migration.Description), "staged") {
		t.Fatalf("expected staged rollout description, got %q", migration.Description)
	}
	if !strings.Contains(strings.ToLower(migration.Description), "vector") {
		t.Fatalf("expected vector column description, got %q", migration.Description)
	}
	if !strings.Contains(strings.ToLower(migration.Description), "json") {
		t.Fatalf("expected runtime json note, got %q", migration.Description)
	}
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

func TestPostgresBootstrapExpandsLegacyShortIconColumns(t *testing.T) {
	statements := postgresColumnAdjustmentStatements()
	joined := strings.Join(statements, "\n")

	if !strings.Contains(joined, "ALTER TABLE feeds ALTER COLUMN icon TYPE VARCHAR(1000)") {
		t.Fatalf("expected feeds.icon widening statement, got %q", joined)
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
