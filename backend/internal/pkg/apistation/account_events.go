package apistation

import (
	"encoding/json"
	"time"
)

// AccountStateEvent 记录账号状态变更事件
type AccountStateEvent struct {
	AccountID     int64     `json:"account_id"`
	OldStatus     string    `json:"old_status"`
	NewStatus     string    `json:"new_status"`
	Trigger       string    `json:"trigger"`       // 401/403/429/529/manual
	Reason        string    `json:"reason"`
	Timestamp     time.Time `json:"timestamp"`
	LastRequestID string    `json:"last_request_id,omitempty"`
}

// NewAccountStateEvent 创建状态变更事件
func NewAccountStateEvent(
	accountID int64,
	oldStatus, newStatus, trigger, reason, lastRequestID string,
) AccountStateEvent {
	return AccountStateEvent{
		AccountID:     accountID,
		OldStatus:     oldStatus,
		NewStatus:     newStatus,
		Trigger:       trigger,
		Reason:        reason,
		Timestamp:     time.Now(),
		LastRequestID: lastRequestID,
	}
}

// AccountStateEventToJSON 序列化为 JSON
func AccountStateEventToJSON(event AccountStateEvent) string {
	data, err := json.Marshal(event)
	if err != nil {
		return "{}"
	}
	return string(data)
}
