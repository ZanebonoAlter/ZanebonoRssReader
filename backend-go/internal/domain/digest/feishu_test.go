package digest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFeishuNotifier(t *testing.T) {
	notifier := NewFeishuNotifier("https://test.webhook.url")
	assert.NotNil(t, notifier)
	assert.Equal(t, "https://test.webhook.url", notifier.webhookURL)
	assert.NotNil(t, notifier.client)
}

func TestSendSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var message FeishuMessage
		err := json.NewDecoder(r.Body).Decode(&message)
		require.NoError(t, err)

		assert.Equal(t, "text", message.MsgType)
		assert.Contains(t, message.Content["text"], "Test Title")
		assert.Contains(t, message.Content["text"], "Test Content")

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewFeishuNotifier(server.URL)
	err := notifier.SendSummary("Test Title", "Test Content")
	assert.NoError(t, err)
}

func TestSendCard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var message FeishuMessage
		err := json.NewDecoder(r.Body).Decode(&message)
		require.NoError(t, err)

		assert.Equal(t, "interactive", message.MsgType)

		config, ok := message.Content["config"].(map[string]interface{})
		require.True(t, ok)
		assert.True(t, config["wide_screen_mode"].(bool))

		header, ok := message.Content["header"].(map[string]interface{})
		require.True(t, ok)
		title, ok := header["title"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "plain_text", title["tag"])
		assert.Equal(t, "Card Title", title["content"])

		elements, ok := message.Content["elements"].([]interface{})
		require.True(t, ok)
		assert.Len(t, elements, 1)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewFeishuNotifier(server.URL)
	err := notifier.SendCard("Card Title", "Card Content")
	assert.NoError(t, err)
}

func TestSendSummaryErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	notifier := NewFeishuNotifier(server.URL)
	err := notifier.SendSummary("Test", "Content")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestSendCardErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	notifier := NewFeishuNotifier(server.URL)
	err := notifier.SendCard("Test", "Content")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 400")
}

func TestSendSummaryMessageFormat(t *testing.T) {
	title := "Daily Digest"
	content := "Article 1\nArticle 2"

	expectedFormat := title + "\n\n" + content

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var message FeishuMessage
		json.NewDecoder(r.Body).Decode(&message)

		text := message.Content["text"].(string)
		assert.Equal(t, expectedFormat, text)
		assert.True(t, strings.HasPrefix(text, title))
		assert.True(t, strings.Contains(text, content))

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewFeishuNotifier(server.URL)
	err := notifier.SendSummary(title, content)
	assert.NoError(t, err)
}

func TestSendCardStructure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var message FeishuMessage
		json.NewDecoder(r.Body).Decode(&message)

		assert.Equal(t, "interactive", message.MsgType)

		content := message.Content
		assert.Contains(t, content, "config")
		assert.Contains(t, content, "header")
		assert.Contains(t, content, "elements")

		config := content["config"].(map[string]interface{})
		assert.True(t, config["wide_screen_mode"].(bool))

		header := content["header"].(map[string]interface{})
		title := header["title"].(map[string]interface{})
		assert.Equal(t, "plain_text", title["tag"])
		assert.Equal(t, "Test Title", title["content"])

		elements := content["elements"].([]interface{})
		assert.Len(t, elements, 1)

		firstElement := elements[0].(map[string]interface{})
		assert.Equal(t, "div", firstElement["tag"])

		text := firstElement["text"].(map[string]interface{})
		assert.Equal(t, "lark_md", text["tag"])
		assert.Equal(t, "Test Content", text["content"])

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewFeishuNotifier(server.URL)
	err := notifier.SendCard("Test Title", "Test Content")
	assert.NoError(t, err)
}

func TestNetworkError(t *testing.T) {
	notifier := NewFeishuNotifier("http://invalid-url-that-does-not-exist.local:9999")
	err := notifier.SendSummary("Test", "Content")
	assert.Error(t, err)
}
