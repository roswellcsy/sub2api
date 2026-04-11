package apistation

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
)

func TestCaptureAuditRecord(t *testing.T) {
	req := &http.Request{
		URL: &url.URL{Scheme: "https", Host: "api.anthropic.com", Path: "/v1/messages", RawQuery: "key=secret"},
		Header: http.Header{
			"User-Agent":                  {"claude-code/1.0.33"},
			"X-Stainless-Lang":            {"js"},
			"X-Stainless-Package-Version": {"0.30.1"},
			"Anthropic-Beta":              {"max-tokens-3-5-sonnet-2024-07-15"},
			"Authorization":               {"Bearer sk-ant-xxx"},
		},
	}
	resp := &http.Response{
		StatusCode: 200,
		Header: http.Header{
			"Request-Id": {"req-123"},
			"Cf-Ray":     {"abc-def"},
		},
	}

	record := CaptureAuditRecord(req, resp, 42, "claude-sonnet-4-6", "req-test-001", 1500, "")

	if record.AccountID != 42 {
		t.Errorf("AccountID = %d, want 42", record.AccountID)
	}
	if record.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", record.StatusCode)
	}
	if record.DurationMs != 1500 {
		t.Errorf("DurationMs = %d, want 1500", record.DurationMs)
	}

	// URL should be redacted (no query params)
	if record.UpstreamURL != "https://api.anthropic.com/v1/messages" {
		t.Errorf("UpstreamURL = %s, want redacted", record.UpstreamURL)
	}

	// Whitelisted headers captured
	if record.RequestHeaders["user-agent"] != "claude-code/1.0.33" {
		t.Errorf("user-agent not captured")
	}
	if record.RequestHeaders["anthropic-beta"] != "max-tokens-3-5-sonnet-2024-07-15" {
		t.Errorf("anthropic-beta not captured")
	}

	// Sensitive headers NOT captured
	if _, exists := record.RequestHeaders["authorization"]; exists {
		t.Errorf("authorization should not be captured")
	}

	// Response headers
	if record.ResponseHeaders["request-id"] != "req-123" {
		t.Errorf("response request-id not captured")
	}

	// JSON serialization
	jsonStr := AuditRecordToJSON(record)
	var parsed AuditRecord
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}
	if parsed.RequestID != "req-test-001" {
		t.Errorf("Parsed RequestID = %s, want req-test-001", parsed.RequestID)
	}
}

func TestAccountStateEvent(t *testing.T) {
	event := NewAccountStateEvent(42, "active", "error", "401", "auth failed", "req-123")
	if event.AccountID != 42 || event.OldStatus != "active" || event.NewStatus != "error" {
		t.Errorf("Event fields mismatch: %+v", event)
	}

	jsonStr := AccountStateEventToJSON(event)
	var parsed AccountStateEvent
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}
	if parsed.Trigger != "401" {
		t.Errorf("Parsed Trigger = %s, want 401", parsed.Trigger)
	}
}
