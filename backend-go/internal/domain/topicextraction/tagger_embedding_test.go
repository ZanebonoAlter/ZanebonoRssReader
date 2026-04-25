package topicextraction

import (
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

func setupTaggerEmbeddingTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	database.DB = db
	t.Cleanup(func() {
		database.DB = nil
	})

	if err := database.DB.AutoMigrate(
		&models.TopicTag{},
		&models.TopicTagRelation{},
	); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
}

func TestShouldDeleteAbstractChildEmbeddingPreservesNormalChildWithAbstractSibling(t *testing.T) {
	setupTaggerEmbeddingTestDB(t)

	parent := models.TopicTag{Slug: "parent", Label: "Parent", Category: "keyword", Source: "abstract", Status: "active"}
	abstractSibling := models.TopicTag{Slug: "abstract-sibling", Label: "Abstract Sibling", Category: "keyword", Source: "abstract", Status: "active"}
	child := models.TopicTag{Slug: "child", Label: "Child", Category: "keyword", Source: "llm", Status: "active"}
	for _, tag := range []*models.TopicTag{&parent, &abstractSibling, &child} {
		if err := database.DB.Create(tag).Error; err != nil {
			t.Fatalf("create tag %s: %v", tag.Label, err)
		}
	}

	for _, relation := range []models.TopicTagRelation{
		{ParentID: parent.ID, ChildID: abstractSibling.ID, RelationType: "abstract"},
		{ParentID: parent.ID, ChildID: child.ID, RelationType: "abstract"},
	} {
		if err := database.DB.Create(&relation).Error; err != nil {
			t.Fatalf("create relation: %v", err)
		}
	}

	deleteEmbedding := shouldDeleteAbstractChildEmbedding(child.ID, parent.ID)
	if deleteEmbedding {
		t.Fatal("expected embedding to be preserved when normal child has an abstract sibling")
	}
}

func TestShouldDeleteAbstractChildEmbeddingDeletesNormalChildWithoutAbstractSibling(t *testing.T) {
	setupTaggerEmbeddingTestDB(t)

	parent := models.TopicTag{Slug: "parent", Label: "Parent", Category: "keyword", Source: "abstract", Status: "active"}
	child := models.TopicTag{Slug: "child", Label: "Child", Category: "keyword", Source: "llm", Status: "active"}
	for _, tag := range []*models.TopicTag{&parent, &child} {
		if err := database.DB.Create(tag).Error; err != nil {
			t.Fatalf("create tag %s: %v", tag.Label, err)
		}
	}
	if err := database.DB.Create(&models.TopicTagRelation{ParentID: parent.ID, ChildID: child.ID, RelationType: "abstract"}).Error; err != nil {
		t.Fatalf("create relation: %v", err)
	}

	deleteEmbedding := shouldDeleteAbstractChildEmbedding(child.ID, parent.ID)
	if !deleteEmbedding {
		t.Fatal("expected embedding to be deleted when normal child has no abstract siblings")
	}
}
