package articles

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

func setupArticlesHandlerTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:articles_handler_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Category{}, &models.Feed{}, &models.Article{}, &models.TopicTag{}, &models.ArticleTopicTag{}))
	database.DB = db
}

func TestGetArticleReturnsArticleTags(t *testing.T) {
	setupArticlesHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	category := models.Category{Name: "AI", Slug: "ai", Color: "#3b6b87", Icon: "mdi:brain"}
	require.NoError(t, database.DB.Create(&category).Error)

	feed := models.Feed{Title: "OpenAI Blog", URL: "https://example.com/openai", CategoryID: &category.ID}
	require.NoError(t, database.DB.Create(&feed).Error)

	article := models.Article{
		FeedID:    feed.ID,
		Title:     "GPT-5 agent runtime",
		Link:      "https://example.com/gpt5-agent-runtime",
		CreatedAt: time.Date(2026, 3, 22, 9, 0, 0, 0, time.FixedZone("CST", 8*3600)),
	}
	require.NoError(t, database.DB.Create(&article).Error)

	topicTag := models.TopicTag{Label: "AI Agent", Slug: "ai-agent", Category: models.TagCategoryKeyword, Kind: "keyword", Icon: "mdi:robot"}
	require.NoError(t, database.DB.Create(&topicTag).Error)
	require.NoError(t, database.DB.Create(&models.ArticleTopicTag{ArticleID: article.ID, TopicTagID: topicTag.ID, Score: 0.92, Source: "llm"}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "article_id", Value: fmt.Sprintf("%d", article.ID)}}
	ctx.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/articles/%d", article.ID), http.NoBody)

	GetArticle(ctx)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var body struct {
		Success bool `json:"success"`
		Data    struct {
			ID   uint `json:"id"`
			Tags []struct {
				Slug     string  `json:"slug"`
				Label    string  `json:"label"`
				Category string  `json:"category"`
				Score    float64 `json:"score"`
				Icon     string  `json:"icon"`
			} `json:"tags"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.True(t, body.Success)
	require.Equal(t, article.ID, body.Data.ID)
	require.Len(t, body.Data.Tags, 1)
	require.Equal(t, "ai-agent", body.Data.Tags[0].Slug)
	require.Equal(t, "AI Agent", body.Data.Tags[0].Label)
	require.Equal(t, models.TagCategoryKeyword, body.Data.Tags[0].Category)
	require.Equal(t, 0.92, body.Data.Tags[0].Score)
	require.Equal(t, "mdi:robot", body.Data.Tags[0].Icon)
}

func TestGetArticlesReturnsTagCount(t *testing.T) {
	setupArticlesHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	feed := models.Feed{Title: "OpenAI Blog", URL: "https://example.com/openai"}
	require.NoError(t, database.DB.Create(&feed).Error)

	article := models.Article{
		FeedID:    feed.ID,
		Title:     "Runtime launch",
		Link:      "https://example.com/runtime",
		CreatedAt: time.Now(),
	}
	require.NoError(t, database.DB.Create(&article).Error)

	tagA := models.TopicTag{Label: "AI Agent", Slug: "ai-agent", Category: models.TagCategoryKeyword, Kind: "keyword"}
	tagB := models.TopicTag{Label: "OpenAI", Slug: "openai", Category: models.TagCategoryKeyword, Kind: "keyword"}
	require.NoError(t, database.DB.Create(&tagA).Error)
	require.NoError(t, database.DB.Create(&tagB).Error)
	require.NoError(t, database.DB.Create(&models.ArticleTopicTag{ArticleID: article.ID, TopicTagID: tagA.ID, Score: 1, Source: "llm"}).Error)
	require.NoError(t, database.DB.Create(&models.ArticleTopicTag{ArticleID: article.ID, TopicTagID: tagB.ID, Score: 0.8, Source: "llm"}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/articles", http.NoBody)

	GetArticles(ctx)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var body struct {
		Success bool `json:"success"`
		Data    []struct {
			ID       uint `json:"id"`
			TagCount int  `json:"tag_count"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.True(t, body.Success)
	require.NotEmpty(t, body.Data)
	require.Equal(t, article.ID, body.Data[0].ID)
	require.Equal(t, 2, body.Data[0].TagCount)
}

func TestRetagArticleReturnsUpdatedTags(t *testing.T) {
	setupArticlesHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	feed := models.Feed{Title: "OpenAI Feed", URL: "https://example.com/openai"}
	require.NoError(t, database.DB.Create(&feed).Error)

	article := models.Article{
		FeedID:           feed.ID,
		Title:            "Daily brief",
		Link:             "https://example.com/daily-brief",
		Description:      "Old short description",
		AIContentSummary: "OpenAI launched a new AI agent workflow.",
		CreatedAt:        time.Now(),
	}
	require.NoError(t, database.DB.Create(&article).Error)

	oldTag := models.TopicTag{Label: "Legacy", Slug: "legacy", Category: models.TagCategoryKeyword, Kind: "keyword"}
	require.NoError(t, database.DB.Create(&oldTag).Error)
	require.NoError(t, database.DB.Create(&models.ArticleTopicTag{ArticleID: article.ID, TopicTagID: oldTag.ID, Score: 1, Source: "manual"}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "article_id", Value: fmt.Sprintf("%d", article.ID)}}
	ctx.Request = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/articles/%d/tags", article.ID), http.NoBody)

	RetagArticleHandler(ctx)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var body struct {
		Success bool `json:"success"`
		Data    struct {
			TagCount int `json:"tag_count"`
			Tags     []struct {
				Slug string `json:"slug"`
			} `json:"tags"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.True(t, body.Success)
	require.NotZero(t, body.Data.TagCount)
	require.NotEmpty(t, body.Data.Tags)
	require.NotEqual(t, "legacy", body.Data.Tags[0].Slug)

	var linkCount int64
	require.NoError(t, database.DB.Model(&models.ArticleTopicTag{}).Where("article_id = ?", article.ID).Count(&linkCount).Error)
	require.Equal(t, int64(body.Data.TagCount), linkCount)
}
