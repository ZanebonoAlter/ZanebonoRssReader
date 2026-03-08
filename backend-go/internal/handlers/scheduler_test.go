package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type stubTriggerScheduler struct {
	result map[string]interface{}
}

func (s stubTriggerScheduler) TriggerNow() map[string]interface{} {
	return s.result
}

func TestTriggerSchedulerReturnsStructuredBlockedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	AutoSummarySchedulerInterface = stubTriggerScheduler{result: map[string]interface{}{
		"accepted":    false,
		"started":     false,
		"reason":      "ai_config_missing",
		"message":     "AI config missing",
		"status_code": http.StatusBadRequest,
	}}
	defer func() {
		AutoSummarySchedulerInterface = nil
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
