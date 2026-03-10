package digest

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestDigestSchedulerAutoSendToOpenNotebook(t *testing.T) {
	setupDigestHandlerTestDB(t)
	gin.SetMode(gin.TestMode)
	require.NoError(t, saveOpenNotebookConfigRecord(openNotebookConfig{
		Enabled:        true,
		BaseURL:        "https://open-notebook.example",
		APIKey:         "secret",
		Model:          "gpt-4.1-mini",
		PromptMode:     "digest_summary",
		AutoSendWeekly: true,
	}))

	originalFactory := openNotebookClientFactory
	defer func() { openNotebookClientFactory = originalFactory }()

	stubClient := &stubOpenNotebookClient{response: &openNotebookRunResponse{SummaryMarkdown: "# 二次总结"}}
	openNotebookClientFactory = func(config openNotebookConfig) openNotebookClient {
		return stubClient
	}

	scheduler := NewDigestScheduler()
	ok := scheduler.autoSendToOpenNotebook("weekly", time.Date(2026, 3, 10, 9, 0, 0, 0, time.FixedZone("CST", 8*3600)), []CategoryDigest{})

	require.True(t, ok)
	require.NotNil(t, stubClient.request)
	require.Equal(t, "本周周报", stubClient.request.Title)
	require.Contains(t, stubClient.request.Content, "# 本周周报")
}

func TestDigestSchedulerAutoSendToOpenNotebookSkipsWhenDisabled(t *testing.T) {
	setupDigestHandlerTestDB(t)
	gin.SetMode(gin.TestMode)
	require.NoError(t, saveOpenNotebookConfigRecord(openNotebookConfig{
		Enabled:        true,
		BaseURL:        "https://open-notebook.example",
		AutoSendWeekly: false,
	}))

	originalFactory := openNotebookClientFactory
	defer func() { openNotebookClientFactory = originalFactory }()

	called := false
	openNotebookClientFactory = func(config openNotebookConfig) openNotebookClient {
		called = true
		return &stubOpenNotebookClient{}
	}

	scheduler := NewDigestScheduler()
	ok := scheduler.autoSendToOpenNotebook("weekly", time.Now(), []CategoryDigest{})

	require.False(t, ok)
	require.False(t, called)
}
