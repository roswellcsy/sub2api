package apistation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// FeishuWebhookClient sends alert messages to Feishu (Lark) via incoming webhook.
type FeishuWebhookClient struct {
	httpClient *http.Client
}

// NewFeishuWebhookClient creates a client with a 10s timeout.
func NewFeishuWebhookClient() *FeishuWebhookClient {
	return &FeishuWebhookClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// feishuCardMessage is the Feishu interactive card format.
type feishuCardMessage struct {
	MsgType string     `json:"msg_type"`
	Card    feishuCard `json:"card"`
}

type feishuCard struct {
	Header   feishuCardHeader    `json:"header"`
	Elements []feishuCardElement `json:"elements"`
}

type feishuCardHeader struct {
	Title    feishuCardText `json:"title"`
	Template string         `json:"template"`
}

type feishuCardText struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

type feishuCardElement struct {
	Tag     string          `json:"tag"`
	Content *feishuCardText `json:"content,omitempty"`
	Text    *feishuCardText `json:"text,omitempty"`
}

// SendAlert sends a card-style alert to the given Feishu webhook URL.
// template: "red" for errors, "orange" for warnings, "green" for info.
func (c *FeishuWebhookClient) SendAlert(ctx context.Context, webhookURL, title, content, template string) error {
	if webhookURL == "" {
		return fmt.Errorf("feishu webhook URL is empty")
	}
	if c == nil || c.httpClient == nil {
		c = NewFeishuWebhookClient()
	}
	if template == "" {
		template = "orange"
	}

	msg := feishuCardMessage{
		MsgType: "interactive",
		Card: feishuCard{
			Header: feishuCardHeader{
				Title:    feishuCardText{Tag: "plain_text", Content: title},
				Template: template,
			},
			Elements: []feishuCardElement{
				{
					Tag:  "markdown",
					Text: &feishuCardText{Tag: "lark_md", Content: content},
				},
			},
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal feishu message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create feishu request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send feishu alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("feishu webhook returned status %d", resp.StatusCode)
	}
	return nil
}
