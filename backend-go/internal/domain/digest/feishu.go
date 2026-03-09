package digest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type FeishuNotifier struct {
	webhookURL string
	client     *http.Client
}

type FeishuMessage struct {
	MsgType string                 `json:"msg_type"`
	Content map[string]interface{} `json:"content"`
}

func NewFeishuNotifier(webhookURL string) *FeishuNotifier {
	return &FeishuNotifier{
		webhookURL: webhookURL,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

func (f *FeishuNotifier) SendSummary(title, content string) error {
	message := FeishuMessage{
		MsgType: "text",
		Content: map[string]interface{}{
			"text": fmt.Sprintf("%s\n\n%s", title, content),
		},
	}
	return f.send(message)
}

func (f *FeishuNotifier) SendCard(title, content string) error {
	message := FeishuMessage{
		MsgType: "interactive",
		Content: map[string]interface{}{
			"config": map[string]interface{}{
				"wide_screen_mode": true,
			},
			"header": map[string]interface{}{
				"title": map[string]interface{}{
					"tag":     "plain_text",
					"content": title,
				},
			},
			"elements": []map[string]interface{}{
				{
					"tag": "div",
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": content,
					},
				},
			},
		},
	}
	return f.send(message)
}

func (f *FeishuNotifier) send(message FeishuMessage) error {
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", f.webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("feishu API returned status %d", resp.StatusCode)
	}

	return nil
}
