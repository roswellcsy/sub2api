package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupApistationHealthRouter(adminSvc service.AdminService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewApistationHealthHandler(adminSvc, nil)
	router.GET("/api/v1/admin/apistation/health", h.GetAccountHealth)
	return router
}

func TestNewApistationHealthHandler(t *testing.T) {
	h := NewApistationHealthHandler(nil, nil)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestApistationHealthHandler_GetAccountHealth_PaginatesAggregation(t *testing.T) {
	svc := newStubAdminService()
	now := time.Now().UTC()
	resetAt := now.Add(30 * time.Minute)
	overloadUntil := now.Add(15 * time.Minute)

	accounts := make([]service.Account, 0, 550)
	for i := 0; i < 500; i++ {
		accounts = append(accounts, service.Account{
			ID:          int64(i + 1),
			Name:        "error-account",
			Platform:    service.PlatformOpenAI,
			Status:      service.StatusError,
			Schedulable: false,
		})
	}
	for i := 0; i < 50; i++ {
		rateLimitedAt := now.Add(-time.Minute)
		accounts = append(accounts, service.Account{
			ID:               int64(501 + i),
			Name:             "active-account",
			Platform:         service.PlatformOpenAI,
			Status:           service.StatusActive,
			Schedulable:      true,
			RateLimitedAt:    &rateLimitedAt,
			RateLimitResetAt: &resetAt,
			OverloadUntil:    &overloadUntil,
		})
	}
	svc.accounts = accounts

	router := setupApistationHealthRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/apistation/health", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 2, svc.lastListAccounts.calls)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			TotalAccounts         int64             `json:"total_accounts"`
			FetchedAccountsCount  int               `json:"fetched_accounts_count"`
			ReturnedAccountsCount int               `json:"returned_accounts_count"`
			IsTruncated           bool              `json:"is_truncated"`
			ActiveCount           int64             `json:"active_count"`
			ErrorCount            int64             `json:"error_count"`
			RateLimitedCount      int64             `json:"rate_limited_count"`
			OverloadedCount       int64             `json:"overloaded_count"`
			SchedulableCount      int64             `json:"schedulable_count"`
			AvailabilityRate      float64           `json:"availability_rate"`
			Accounts              []json.RawMessage `json:"accounts"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, int64(550), resp.Data.TotalAccounts)
	require.Equal(t, 550, resp.Data.FetchedAccountsCount)
	require.Equal(t, 200, resp.Data.ReturnedAccountsCount)
	require.False(t, resp.Data.IsTruncated)
	require.Equal(t, int64(50), resp.Data.ActiveCount)
	require.Equal(t, int64(500), resp.Data.ErrorCount)
	require.Equal(t, int64(50), resp.Data.RateLimitedCount)
	require.Equal(t, int64(50), resp.Data.OverloadedCount)
	require.Equal(t, int64(50), resp.Data.SchedulableCount)
	require.InDelta(t, 9.09, resp.Data.AvailabilityRate, 0.001)
	require.Len(t, resp.Data.Accounts, 200)
}
