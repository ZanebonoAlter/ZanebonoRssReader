package jobs

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

func setupTagHierarchyCleanupSchedulerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	database.DB = db
	t.Cleanup(func() {
		database.DB = nil
	})

	if err := db.AutoMigrate(
		&models.SchedulerTask{},
		&models.TopicTag{},
		&models.TopicTagRelation{},
		&models.ArticleTopicTag{},
		&models.AISummaryTopic{},
	); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}

	return db
}

func TestRunCleanupCycleSummaryOmitsLegacyTreeFields(t *testing.T) {
	db := setupTagHierarchyCleanupSchedulerTestDB(t)

	scheduler := NewTagHierarchyCleanupScheduler(60)
	scheduler.runCleanupCycle("test")

	var task models.SchedulerTask
	if err := db.Where("name = ?", "tag_hierarchy_cleanup").First(&task).Error; err != nil {
		t.Fatalf("load scheduler task: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(task.LastExecutionResult), &payload); err != nil {
		t.Fatalf("unmarshal summary: %v", err)
	}

	for _, key := range []string{"trees_processed", "tags_processed", "abstracts_created", "phase4_trees", "phase4_merges", "phase4_reparents"} {
		if _, exists := payload[key]; exists {
			t.Fatalf("summary should not contain legacy key %q: %#v", key, payload)
		}
	}
	for _, key := range []string{"trees_reviewed", "merges_applied", "moves_applied", "tree_groups_created", "tree_groups_reused"} {
		if _, exists := payload[key]; !exists {
			t.Fatalf("summary missing expected key %q: %#v", key, payload)
		}
	}
	if _, exists := payload["flat_merges_applied"]; !exists {
		t.Fatalf("summary missing flat_merges_applied: %#v", payload)
	}
}
