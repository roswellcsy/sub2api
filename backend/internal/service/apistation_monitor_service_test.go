package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apistation"
)

type apistationMonitorSettingRepoStub struct {
	values map[string]string
	err    error
}

func (s *apistationMonitorSettingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *apistationMonitorSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if v, ok := s.values[key]; ok {
		return v, nil
	}
	return "", ErrSettingNotFound
}

func (s *apistationMonitorSettingRepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *apistationMonitorSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *apistationMonitorSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *apistationMonitorSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *apistationMonitorSettingRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

func TestNewApistationMonitorService(t *testing.T) {
	svc := NewApistationMonitorService(nil, nil)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.feishu == nil {
		t.Fatal("expected non-nil feishu client")
	}
}

func TestApistationMonitorService_StartStop(t *testing.T) {
	svc := NewApistationMonitorService(nil, nil)
	svc.Start()
	svc.Stop()
}

func TestApistationMonitorService_NilSafe(t *testing.T) {
	var svc *ApistationMonitorService
	svc.Start()
	svc.Stop()
}

func TestApistationMonitorService_GetCheckInterval_Default(t *testing.T) {
	svc := NewApistationMonitorService(nil, nil)
	if got := svc.getCheckInterval(); got != 5*time.Minute {
		t.Fatalf("getCheckInterval() = %s, want %s", got, 5*time.Minute)
	}
}

func TestApistationMonitorService_CheckVersionDrift(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	settingSvc := &SettingService{
		settingRepo: &apistationMonitorSettingRepoStub{
			values: map[string]string{
				SettingKeyFeishuWebhookURL:      server.URL,
				SettingKeyCLIVersion:            "1.0.29",
				SettingKeyLatestKnownCLIVersion: "1.1.0",
			},
		},
	}
	svc := NewApistationMonitorService(nil, settingSvc)
	svc.checkVersionDrift()

	if got := calls.Load(); got != 1 {
		t.Fatalf("expected 1 alert, got %d", got)
	}
}

func TestApistationMonitorService_CheckBanWarnings(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	repo := &opsRepoMock{
		ListSystemLogsFn: func(ctx context.Context, filter *OpsSystemLogFilter) (*OpsSystemLogList, error) {
			return &OpsSystemLogList{
				Logs: []*OpsSystemLog{
					{Message: "account_state_change", Extra: map[string]any{"event": apistation.AccountStateEventToJSON(apistation.AccountStateEvent{AccountID: 42, Trigger: "auth", Reason: "403 forbidden"})}},
					{Message: "account_state_change", Extra: map[string]any{"event": apistation.AccountStateEventToJSON(apistation.AccountStateEvent{AccountID: 42, Trigger: "auth", Reason: "401 unauthorized"})}},
					{Message: "account_state_change", Extra: map[string]any{"event": apistation.AccountStateEventToJSON(apistation.AccountStateEvent{AccountID: 42, Trigger: "auth", Reason: "403 forbidden"})}},
				},
				Total:    3,
				Page:     1,
				PageSize: 3,
			}, nil
		},
	}
	opsSvc := NewOpsService(repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	settingSvc := &SettingService{
		settingRepo: &apistationMonitorSettingRepoStub{
			values: map[string]string{
				SettingKeyFeishuWebhookURL:     server.URL,
				SettingKeyBanAlertThreshold:    "3",
				SettingKeyMonitorCheckInterval: "60",
			},
		},
	}
	svc := NewApistationMonitorService(opsSvc, settingSvc)
	svc.checkBanWarnings()

	if got := calls.Load(); got != 1 {
		t.Fatalf("expected 1 alert, got %d", got)
	}
}
