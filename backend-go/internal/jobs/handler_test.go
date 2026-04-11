package jobs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"my-robot-backend/internal/app/runtimeinfo"
)

type stubTriggerScheduler struct {
	result map[string]interface{}
}

func (s stubTriggerScheduler) TriggerNow() map[string]interface{} {
	return s.result
}

type stubManagedScheduler struct {
	status             SchedulerStatusResponse
	taskStatus         map[string]interface{}
	triggerResult      map[string]interface{}
	updatedInterval    int
	resetCalled        bool
	triggerCalledCount int
}

func (s *stubManagedScheduler) GetStatus() SchedulerStatusResponse {
	return s.status
}

func (s *stubManagedScheduler) GetTaskStatusDetails() map[string]interface{} {
	return s.taskStatus
}

func (s *stubManagedScheduler) TriggerNow() map[string]interface{} {
	s.triggerCalledCount++
	if s.triggerResult != nil {
		return s.triggerResult
	}
	return map[string]interface{}{
		"accepted": true,
		"started":  true,
		"message":  "ok",
	}
}

func (s *stubManagedScheduler) ResetStats() error {
	s.resetCalled = true
	return nil
}

func (s *stubManagedScheduler) UpdateInterval(interval int) error {
	s.updatedInterval = interval
	return nil
}

func resetSchedulerInterfaces() {
	runtimeinfo.AutoRefreshSchedulerInterface = nil
	runtimeinfo.AutoSummarySchedulerInterface = nil
	runtimeinfo.PreferenceUpdateSchedulerInterface = nil
	runtimeinfo.AISummarySchedulerInterface = nil
	runtimeinfo.FirecrawlSchedulerInterface = nil
	runtimeinfo.DigestSchedulerInterface = nil
}

func TestTriggerSchedulerReturnsStructuredBlockedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetSchedulerInterfaces()
	runtimeinfo.AutoSummarySchedulerInterface = stubTriggerScheduler{result: map[string]interface{}{
		"accepted":    false,
		"started":     false,
		"reason":      "ai_config_missing",
		"message":     "AI config missing",
		"status_code": http.StatusBadRequest,
	}}
	defer func() {
		resetSchedulerInterfaces()
	}()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "name", Value: "auto_summary"}}

	TriggerScheduler(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", recorder.Code, recorder.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["success"] != false {
		t.Fatalf("success = %v, want false", body["success"])
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("data missing: %#v", body)
	}
	if data["reason"] != "ai_config_missing" {
		t.Fatalf("reason = %v, want ai_config_missing", data["reason"])
	}
}

func TestGetSchedulersStatusIncludesPreferenceUpdateAndDigest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetSchedulerInterfaces()
	defer resetSchedulerInterfaces()

	autoRefresh := &stubManagedScheduler{status: SchedulerStatusResponse{Name: "Auto Refresh", Status: "idle"}}
	autoSummary := &stubManagedScheduler{status: SchedulerStatusResponse{Name: "Auto Summary", Status: "idle"}}
	preference := &stubManagedScheduler{status: SchedulerStatusResponse{Name: "Preference Update", Status: "idle"}}
	completion := &stubManagedScheduler{status: SchedulerStatusResponse{Name: "Content Completion", Status: "idle"}}
	firecrawl := &stubManagedScheduler{status: SchedulerStatusResponse{Name: "Firecrawl Crawler", Status: "idle"}}
	digest := &stubManagedScheduler{status: SchedulerStatusResponse{Name: "Digest", Status: "running", IsExecuting: true}}

	runtimeinfo.AutoRefreshSchedulerInterface = autoRefresh
	runtimeinfo.AutoSummarySchedulerInterface = autoSummary
	runtimeinfo.PreferenceUpdateSchedulerInterface = preference
	runtimeinfo.AISummarySchedulerInterface = completion
	runtimeinfo.FirecrawlSchedulerInterface = firecrawl
	runtimeinfo.DigestSchedulerInterface = digest

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	GetSchedulersStatus(ctx)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	data := body["data"].([]any)

	names := map[string]bool{}
	for _, item := range data {
		entry := item.(map[string]any)
		names[entry["name"].(string)] = true
	}

	require.True(t, names["Auto Refresh"])
	require.True(t, names["Auto Summary"])
	require.True(t, names["Preference Update"])
	require.True(t, names["Content Completion"])
	require.True(t, names["Firecrawl Crawler"])
	require.True(t, names["Digest"])
}

func TestTriggerSchedulerSupportsContentCompletionAliasAndLegacyName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetSchedulerInterfaces()
	defer resetSchedulerInterfaces()

	completion := &stubManagedScheduler{triggerResult: map[string]interface{}{"accepted": true, "started": true, "message": "triggered"}}
	runtimeinfo.AISummarySchedulerInterface = completion

	for _, name := range []string{"content_completion", "ai_summary"} {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Params = gin.Params{{Key: "name", Value: name}}

		TriggerScheduler(ctx)

		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	}

	require.Equal(t, 2, completion.triggerCalledCount)
}

func TestResetSchedulerStatsCallsSchedulerImplementation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetSchedulerInterfaces()
	defer resetSchedulerInterfaces()

	autoRefresh := &stubManagedScheduler{}
	runtimeinfo.AutoRefreshSchedulerInterface = autoRefresh

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "name", Value: "auto_refresh"}}

	ResetSchedulerStats(ctx)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.True(t, autoRefresh.resetCalled)
}

func TestUpdateSchedulerIntervalCallsSchedulerImplementation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetSchedulerInterfaces()
	defer resetSchedulerInterfaces()

	completion := &stubManagedScheduler{}
	runtimeinfo.AISummarySchedulerInterface = completion

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "name", Value: "content_completion"}}
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/schedulers/content_completion/interval", strings.NewReader(`{"interval":120}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	UpdateSchedulerInterval(ctx)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, 120, completion.updatedInterval)
}

func TestGetTasksStatusAggregatesRuntimeWork(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetSchedulerInterfaces()
	defer resetSchedulerInterfaces()

	completion := &stubManagedScheduler{taskStatus: map[string]interface{}{
		"overview": map[string]interface{}{"pending_count": 3, "processing_count": 1},
	}}
	firecrawl := &stubManagedScheduler{taskStatus: map[string]interface{}{"status": "running", "queue_size": 2, "processing": 1}}
	runtimeinfo.AISummarySchedulerInterface = completion
	runtimeinfo.FirecrawlSchedulerInterface = firecrawl

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	GetTasksStatus(ctx)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	data := body["data"].(map[string]any)
	require.EqualValues(t, 2, data["active_tasks"])
	require.EqualValues(t, 5, data["queue_size"])

	tasks := data["tasks"].([]any)
	require.Len(t, tasks, 2)
}
