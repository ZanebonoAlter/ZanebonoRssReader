package digest

import (
	"bytes"
	"encoding/json"
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

type stubOpenNotebookClient struct {
	response *openNotebookRunResponse
	err      error
	request  *openNotebookSendRequest
}

func (s *stubOpenNotebookClient) SummarizeDigest(req openNotebookSendRequest) (*openNotebookRunResponse, error) {
	s.request = &req
	return s.response, s.err
}

func setupDigestHandlerTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:digest_handler_test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.AISettings{}, &DigestConfig{}))
	database.DB = db
}

func TestGetOpenNotebookConfigReturnsDefaults(t *testing.T) {
	setupDigestHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	GetOpenNotebookConfig(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, true, body["success"])

	data := body["data"].(map[string]any)
	require.Equal(t, false, data["enabled"])
	require.Equal(t, "digest_summary", data["prompt_mode"])
}

func TestUpdateOpenNotebookConfigPersistsValues(t *testing.T) {
	setupDigestHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	body := []byte(`{"enabled":true,"base_url":"https://open-notebook.example","api_key":"secret","model":"gpt-4.1-mini","target_notebook":"digest-lab","prompt_mode":"digest_summary","auto_send_daily":true,"auto_send_weekly":false,"export_back_to_obsidian":true}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/digest/open-notebook/config", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	UpdateOpenNotebookConfig(ctx)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var bodyMap map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &bodyMap))
	data := bodyMap["data"].(map[string]any)
	require.Equal(t, true, data["enabled"])
	require.Equal(t, "https://open-notebook.example", data["base_url"])
	require.Equal(t, "digest-lab", data["target_notebook"])

	stored, settings, err := loadOpenNotebookConfigRecord()
	require.NoError(t, err)
	require.NotNil(t, settings)
	require.Equal(t, true, stored.Enabled)
	require.Equal(t, "secret", stored.APIKey)
	require.Equal(t, true, stored.ExportBackToObsidian)
}

func TestSendDigestToOpenNotebookReturnsSummary(t *testing.T) {
	setupDigestHandlerTestDB(t)
	gin.SetMode(gin.TestMode)
	require.NoError(t, saveOpenNotebookConfigRecord(openNotebookConfig{
		Enabled:        true,
		BaseURL:        "https://open-notebook.example",
		APIKey:         "secret",
		Model:          "gpt-4.1-mini",
		TargetNotebook: "digest-lab",
		PromptMode:     "digest_summary",
	}))

	originalPreviewBuilder := digestPreviewBuilder
	originalClientFactory := openNotebookClientFactory
	defer func() {
		digestPreviewBuilder = originalPreviewBuilder
		openNotebookClientFactory = originalClientFactory
	}()

	digestPreviewBuilder = func(kind string, date time.Time) (*digestPreviewResponse, *DigestConfig, []CategoryDigest, error) {
		return &digestPreviewResponse{
			Type:       kind,
			AnchorDate: date.Format("2006-01-02"),
			Title:      "今日日报",
			Markdown:   "# digest source",
		}, &DigestConfig{}, nil, nil
	}

	stubClient := &stubOpenNotebookClient{response: &openNotebookRunResponse{
		SummaryMarkdown: "# 二次总结",
		RemoteID:        "note-1",
		RemoteURL:       "https://open-notebook.example/note-1",
	}}
	openNotebookClientFactory = func(config openNotebookConfig) openNotebookClient {
		return stubClient
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "type", Value: "daily"}}
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/digest/open-notebook/daily?date=2026-03-10", http.NoBody)

	SendDigestToOpenNotebook(ctx)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.NotNil(t, stubClient.request)
	require.Equal(t, "今日日报", stubClient.request.Title)
	require.Equal(t, "# digest source", stubClient.request.Content)
	require.Equal(t, "digest-lab", stubClient.request.TargetNotebook)

	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	data := body["data"].(map[string]any)
	require.Equal(t, "daily", data["digest_type"])
	require.Equal(t, "2026-03-10", data["anchor_date"])
	require.Equal(t, "# digest source", data["source_markdown"])
	require.Equal(t, "# 二次总结", data["summary_markdown"])
	require.Equal(t, "note-1", data["remote_id"])
}

func TestRunDigestNowAutoSendsOpenNotebookWithoutBlockingSuccess(t *testing.T) {
	setupDigestHandlerTestDB(t)
	gin.SetMode(gin.TestMode)
	require.NoError(t, saveOpenNotebookConfigRecord(openNotebookConfig{
		Enabled:       true,
		BaseURL:       "https://open-notebook.example",
		APIKey:        "secret",
		Model:         "gpt-4.1-mini",
		PromptMode:    "digest_summary",
		AutoSendDaily: true,
	}))

	originalPreviewBuilder := digestPreviewBuilder
	originalClientFactory := openNotebookClientFactory
	defer func() {
		digestPreviewBuilder = originalPreviewBuilder
		openNotebookClientFactory = originalClientFactory
	}()

	digestPreviewBuilder = func(kind string, date time.Time) (*digestPreviewResponse, *DigestConfig, []CategoryDigest, error) {
		return &digestPreviewResponse{
			Type:       kind,
			AnchorDate: date.Format("2006-01-02"),
			Title:      "今日日报",
			Markdown:   "# digest source",
		}, &DigestConfig{}, nil, nil
	}

	stubClient := &stubOpenNotebookClient{response: &openNotebookRunResponse{SummaryMarkdown: "# 二次总结"}}
	openNotebookClientFactory = func(config openNotebookConfig) openNotebookClient {
		return stubClient
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "type", Value: "daily"}}
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/digest/run/daily?date=2026-03-10", http.NoBody)

	RunDigestNow(ctx)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.NotNil(t, stubClient.request)

	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	data := body["data"].(map[string]any)
	require.Equal(t, true, data["sent_to_open_notebook"])
}

func TestRunDigestNowIgnoresOpenNotebookFailure(t *testing.T) {
	setupDigestHandlerTestDB(t)
	gin.SetMode(gin.TestMode)
	require.NoError(t, saveOpenNotebookConfigRecord(openNotebookConfig{
		Enabled:       true,
		BaseURL:       "https://open-notebook.example",
		APIKey:        "secret",
		Model:         "gpt-4.1-mini",
		PromptMode:    "digest_summary",
		AutoSendDaily: true,
	}))

	originalPreviewBuilder := digestPreviewBuilder
	originalClientFactory := openNotebookClientFactory
	defer func() {
		digestPreviewBuilder = originalPreviewBuilder
		openNotebookClientFactory = originalClientFactory
	}()

	digestPreviewBuilder = func(kind string, date time.Time) (*digestPreviewResponse, *DigestConfig, []CategoryDigest, error) {
		return &digestPreviewResponse{
			Type:       kind,
			AnchorDate: date.Format("2006-01-02"),
			Title:      "今日日报",
			Markdown:   "# digest source",
		}, &DigestConfig{}, nil, nil
	}

	stubClient := &stubOpenNotebookClient{err: assertAnError("boom")}
	openNotebookClientFactory = func(config openNotebookConfig) openNotebookClient {
		return stubClient
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "type", Value: "daily"}}
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/digest/run/daily?date=2026-03-10", http.NoBody)

	RunDigestNow(ctx)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	data := body["data"].(map[string]any)
	require.Equal(t, false, data["sent_to_open_notebook"])
}

type testError string

func (e testError) Error() string { return string(e) }

func assertAnError(message string) error { return testError(message) }
