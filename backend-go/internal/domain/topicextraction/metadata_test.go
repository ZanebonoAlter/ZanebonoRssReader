package topicextraction

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/airouter"
	"my-robot-backend/internal/platform/database"
)

func setupTopicExtractionTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	database.DB = db
	if err := database.DB.AutoMigrate(
		&models.Feed{},
		&models.AISummary{},
		&models.TopicTag{},
		&models.AISummaryTopic{},
		&models.Article{},
		&models.ArticleTopicTag{},
		&models.AIProvider{},
		&models.AIRoute{},
		&models.AIRouteProvider{},
		&models.AICallLog{},
	); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
}

func TestTagSummaryLogsSummaryMetadata(t *testing.T) {
	setupTopicExtractionTestDB(t)

	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"[{\"label\":\"OpenAI\",\"category\":\"person\",\"confidence\":0.9}]"}}]}`))
	}))
	defer aiServer.Close()

	provider := models.AIProvider{Name: "tag-primary", ProviderType: airouter.ProviderTypeOpenAICompatible, BaseURL: aiServer.URL, APIKey: "token", Model: "test-model", Enabled: true}
	if err := database.DB.Create(&provider).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}
	route := models.AIRoute{Name: airouter.DefaultRouteName, Capability: string(airouter.CapabilityTopicTagging), Enabled: true, Strategy: "ordered_failover"}
	if err := database.DB.Create(&route).Error; err != nil {
		t.Fatalf("create route: %v", err)
	}
	if err := database.DB.Create(&models.AIRouteProvider{RouteID: route.ID, ProviderID: provider.ID, Priority: 1, Enabled: true}).Error; err != nil {
		t.Fatalf("create route provider: %v", err)
	}

	feed := models.Feed{Title: "Feed", URL: "https://example.com/feed"}
	if err := database.DB.Create(&feed).Error; err != nil {
		t.Fatalf("create feed: %v", err)
	}
	summary := models.AISummary{FeedID: &feed.ID, Title: "Summary title", Summary: "Summary body", Articles: "[1,2]", ArticleCount: 2}
	if err := database.DB.Create(&summary).Error; err != nil {
		t.Fatalf("create summary: %v", err)
	}
	summary.Feed = &feed

	if err := TagSummary(&summary); err != nil {
		t.Fatalf("tag summary: %v", err)
	}

	var callLog models.AICallLog
	if err := database.DB.First(&callLog).Error; err != nil {
		t.Fatalf("load call log: %v", err)
	}
	if callLog.Capability != string(airouter.CapabilityTopicTagging) {
		t.Fatalf("capability = %q, want %q", callLog.Capability, airouter.CapabilityTopicTagging)
	}
	if callLog.RequestMeta != `{"feed_name":"Feed","summary_id":1,"title":"Summary title"}` && callLog.RequestMeta != fmt.Sprintf(`{"feed_name":"Feed","summary_id":%d,"title":"Summary title"}`, summary.ID) {
		t.Fatalf("request_meta = %s", callLog.RequestMeta)
	}
}
