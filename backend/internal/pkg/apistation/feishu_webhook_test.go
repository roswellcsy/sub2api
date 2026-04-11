package apistation

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFeishuWebhookClient_SendAlert(t *testing.T) {
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content type")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		receivedBody = body
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &FeishuWebhookClient{httpClient: server.Client()}
	err := client.SendAlert(context.Background(), server.URL, "Test Alert", "**Test** content", "red")
	if err != nil {
		t.Fatalf("SendAlert failed: %v", err)
	}
	if len(receivedBody) == 0 {
		t.Fatal("no body received")
	}
}

func TestFeishuWebhookClient_SendAlert_EmptyURL(t *testing.T) {
	client := NewFeishuWebhookClient()
	err := client.SendAlert(context.Background(), "", "Test", "content", "")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestFeishuWebhookClient_SendAlert_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &FeishuWebhookClient{httpClient: server.Client()}
	err := client.SendAlert(context.Background(), server.URL, "Test", "content", "")
	if err == nil {
		t.Fatal("expected error for server error")
	}
}
