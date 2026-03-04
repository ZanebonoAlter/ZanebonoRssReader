package digest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFeishuNotifier(t *testing.T) {
	notifier := NewFeishuNotifier("https://test.webhook.url")
	assert.NotNil(t, notifier)
	assert.Equal(t, "https://test.webhook.url", notifier.webhookURL)
	assert.NotNil(t, notifier.client)
}
