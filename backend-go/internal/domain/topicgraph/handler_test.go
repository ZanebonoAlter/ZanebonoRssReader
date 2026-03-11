package topicgraph

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

func setupTopicGraphTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:topic_graph_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.AISettings{}, &models.Category{}, &models.Feed{}, &models.AISummary{}))
	require.NoError(t, db.AutoMigrate(&models.Article{}, &models.TopicTag{}, &models.AISummaryTopic{}))
	database.DB = db
}

func seedTopicGraphData(t *testing.T) {
	t.Helper()

	category := models.Category{Name: "AI", Slug: "ai", Color: "#3b6b87", Icon: "mdi:brain"}
	require.NoError(t, database.DB.Create(&category).Error)

	feedA := models.Feed{Title: "OpenAI Blog", URL: "https://example.com/openai", CategoryID: &category.ID, Icon: "mdi:robot", Color: "#3b6b87"}
	feedB := models.Feed{Title: "Latent Space", URL: "https://example.com/latent-space", CategoryID: &category.ID, Icon: "mdi:orbit", Color: "#bb6c37"}
	require.NoError(t, database.DB.Create(&feedA).Error)
	require.NoError(t, database.DB.Create(&feedB).Error)

	createdAt := time.Date(2026, 3, 11, 9, 30, 0, 0, time.FixedZone("CST", 8*3600))
	articles := []models.Article{
		{FeedID: feedA.ID, Title: "Runtime launch", Link: "https://example.com/runtime", CreatedAt: createdAt},
		{FeedID: feedA.ID, Title: "Toolchain memo", Link: "https://example.com/toolchain", CreatedAt: createdAt},
	}
	for i := range articles {
		require.NoError(t, database.DB.Create(&articles[i]).Error)
	}
	articleIDsJSON, err := json.Marshal([]uint{articles[0].ID, articles[1].ID})
	require.NoError(t, err)

	summaries := []models.AISummary{
		{
			FeedID:       &feedA.ID,
			CategoryID:   &category.ID,
			Title:        "OpenAI ships GPT-5 agent stack",
			Summary:      "OpenAI shipped a GPT-5 AI agent stack with multimodal planning and coding automation.",
			Articles:     string(articleIDsJSON),
			ArticleCount: 5,
			CreatedAt:    createdAt,
			UpdatedAt:    createdAt,
		},
		{
			FeedID:       &feedB.ID,
			CategoryID:   &category.ID,
			Title:        "Anthropic and OpenAI race on agents",
			Summary:      "Anthropic and OpenAI are both pushing AI agent tooling for enterprise coding workflows.",
			ArticleCount: 4,
			CreatedAt:    createdAt.Add(2 * time.Hour),
			UpdatedAt:    createdAt.Add(2 * time.Hour),
		},
	}

	for _, summary := range summaries {
		require.NoError(t, database.DB.Create(&summary).Error)
	}
}

func TestGetTopicGraphReturnsNodesAndEdges(t *testing.T) {
	setupTopicGraphTestDB(t)
	seedTopicGraphData(t)
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "type", Value: "daily"}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/topic-graph/daily?date=2026-03-11", http.NoBody)

	GetTopicGraph(ctx)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var body struct {
		Success bool `json:"success"`
		Data    struct {
			Nodes []map[string]any `json:"nodes"`
			Edges []map[string]any `json:"edges"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.True(t, body.Success)
	require.NotEmpty(t, body.Data.Nodes)
	require.NotEmpty(t, body.Data.Edges)
	require.Contains(t, collectNodeLabels(body.Data.Nodes), "AI Agent")
	require.Contains(t, collectNodeLabels(body.Data.Nodes), "OpenAI")

	var persistedCount int64
	require.NoError(t, database.DB.Model(&models.TopicTag{}).Count(&persistedCount).Error)
	require.Equal(t, int64(0), persistedCount)
}

func TestGetTopicDetailReturnsHistoryAndSummaries(t *testing.T) {
	setupTopicGraphTestDB(t)
	seedTopicGraphData(t)
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "slug", Value: "ai-agent"}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/topic-graph/topic/ai-agent?type=daily&date=2026-03-11", http.NoBody)

	GetTopicDetail(ctx)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var body struct {
		Success bool `json:"success"`
		Data    struct {
			Topic struct {
				Slug  string `json:"slug"`
				Label string `json:"label"`
			} `json:"topic"`
			Summaries   []map[string]any  `json:"summaries"`
			History     []map[string]any  `json:"history"`
			SearchLinks map[string]string `json:"search_links"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.True(t, body.Success)
	require.Equal(t, "ai-agent", body.Data.Topic.Slug)
	require.Equal(t, "AI Agent", body.Data.Topic.Label)
	require.NotEmpty(t, body.Data.Summaries)
	require.NotEmpty(t, body.Data.History)
	firstSummary := body.Data.Summaries[0]
	require.NotEmpty(t, firstSummary["feed_name"])
	require.NotEmpty(t, firstSummary["summary"])
	hasArticles := false
	for _, summary := range body.Data.Summaries {
		articles, ok := summary["articles"].([]any)
		if ok && len(articles) > 0 {
			hasArticles = true
			break
		}
	}
	require.True(t, hasArticles)
	searchLinks, ok := body.Data.SearchLinks["youtube_live"]
	require.True(t, ok)
	require.NotEmpty(t, searchLinks)
}

func collectNodeLabels(nodes []map[string]any) []string {
	labels := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if label, ok := node["label"].(string); ok {
			labels = append(labels, label)
		}
	}
	return labels
}
