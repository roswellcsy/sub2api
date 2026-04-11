package apistation

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// AuditRecord 记录一次上游请求的完整审计信息
type AuditRecord struct {
	RequestID       string            `json:"request_id"`
	AccountID       int64             `json:"account_id"`
	Model           string            `json:"model"`
	Timestamp       time.Time         `json:"timestamp"`
	UpstreamURL     string            `json:"upstream_url"`
	RequestHeaders  map[string]string `json:"request_headers"`  // 伪装相关 headers snapshot
	StatusCode      int               `json:"status_code"`
	ResponseHeaders map[string]string `json:"response_headers"` // rate-limit 相关
	DurationMs      int64             `json:"duration_ms"`
	ErrorMessage    string            `json:"error_message,omitempty"`
}

// auditRequestHeaderKeys 需要记录的请求 header keys (billing/fingerprint/beta/stainless/UA)
var auditRequestHeaderKeys = []string{
	"user-agent",
	"x-stainless-lang",
	"x-stainless-package-version",
	"x-stainless-os",
	"x-stainless-arch",
	"x-stainless-runtime",
	"x-stainless-runtime-version",
	"anthropic-version",
	"anthropic-beta",
}

// auditResponseHeaderKeys rate-limit 相关 response headers
var auditResponseHeaderKeys = []string{
	"retry-after",
	"x-ratelimit-limit-requests",
	"x-ratelimit-limit-tokens",
	"x-ratelimit-remaining-requests",
	"x-ratelimit-remaining-tokens",
	"x-ratelimit-reset-requests",
	"x-ratelimit-reset-tokens",
	"x-should-retry",
	"request-id",
	"cf-ray",
}

// sensitiveHeaderKeys 需要脱敏的 header keys
var sensitiveHeaderKeys = map[string]bool{
	"authorization": true,
	"x-api-key":     true,
	"cookie":        true,
}

// CaptureAuditRecord 从上游请求和响应构建审计记录
func CaptureAuditRecord(
	req *http.Request,
	resp *http.Response,
	accountID int64,
	model string,
	requestID string,
	durationMs int64,
	errMsg string,
) AuditRecord {
	record := AuditRecord{
		RequestID:       requestID,
		AccountID:       accountID,
		Model:           model,
		Timestamp:       time.Now(),
		DurationMs:      durationMs,
		ErrorMessage:    errMsg,
		RequestHeaders:  make(map[string]string),
		ResponseHeaders: make(map[string]string),
	}

	if req != nil {
		record.UpstreamURL = redactURL(req.URL.String())
		for _, key := range auditRequestHeaderKeys {
			if v := req.Header.Get(key); v != "" {
				record.RequestHeaders[key] = v
			}
		}
	}

	if resp != nil {
		record.StatusCode = resp.StatusCode
		for _, key := range auditResponseHeaderKeys {
			if v := resp.Header.Get(key); v != "" {
				record.ResponseHeaders[key] = v
			}
		}
	}

	return record
}

// AuditRecordToJSON 序列化审计记录为 JSON string (用于写入 ops_system_logs.extra)
func AuditRecordToJSON(record AuditRecord) string {
	data, err := json.Marshal(record)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// redactURL 脱敏 URL 中可能包含的 token/key
func redactURL(u string) string {
	// 只保留 scheme + host + path，去掉 query params
	if idx := strings.IndexByte(u, '?'); idx >= 0 {
		return u[:idx]
	}
	return u
}
