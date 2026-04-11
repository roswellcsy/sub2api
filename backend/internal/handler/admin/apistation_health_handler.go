package admin

import (
	"math"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// ApistationHealthHandler 处理 api-station 账号健康度查询
type ApistationHealthHandler struct {
	adminService service.AdminService
	opsService   *service.OpsService
}

// NewApistationHealthHandler creates a new health handler
func NewApistationHealthHandler(adminService service.AdminService, opsService *service.OpsService) *ApistationHealthHandler {
	return &ApistationHealthHandler{adminService: adminService, opsService: opsService}
}

// GetAccountHealth 获取账号池健康度概览
// GET /api/v1/admin/apistation/health
func (h *ApistationHealthHandler) GetAccountHealth(c *gin.Context) {
	if h.adminService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Admin service not available")
		return
	}

	ctx := c.Request.Context()

	accounts, total, err := h.adminService.ListAccounts(ctx, 1, 500, "", "", "", "", 0, "", "", "")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	var (
		activeCount      int64
		errorCount       int64
		rateLimitedCount int64
		overloadedCount  int64
		schedulableCount int64
	)

	type accountHealth struct {
		ID               int64      `json:"id"`
		Name             string     `json:"name"`
		Platform         string     `json:"platform"`
		Status           string     `json:"status"`
		Schedulable      bool       `json:"schedulable"`
		ErrorMessage     string     `json:"error_message,omitempty"`
		LastUsedAt       *time.Time `json:"last_used_at,omitempty"`
		RateLimitResetAt *time.Time `json:"rate_limit_reset_at,omitempty"`
		OverloadUntil    *time.Time `json:"overload_until,omitempty"`
	}

	now := time.Now()
	healthAccounts := make([]accountHealth, 0, len(accounts))

	for _, account := range accounts {
		healthAccounts = append(healthAccounts, accountHealth{
			ID:               account.ID,
			Name:             account.Name,
			Platform:         account.Platform,
			Status:           account.Status,
			Schedulable:      account.Schedulable,
			ErrorMessage:     account.ErrorMessage,
			LastUsedAt:       account.LastUsedAt,
			RateLimitResetAt: account.RateLimitResetAt,
			OverloadUntil:    account.OverloadUntil,
		})

		switch account.Status {
		case service.StatusActive:
			activeCount++
		case service.StatusError:
			errorCount++
		}

		if account.RateLimitedAt != nil && (account.RateLimitResetAt == nil || account.RateLimitResetAt.After(now)) {
			rateLimitedCount++
		}
		if account.OverloadUntil != nil && account.OverloadUntil.After(now) {
			overloadedCount++
		}
		if account.Schedulable {
			schedulableCount++
		}
	}

	availabilityRate := float64(0)
	if total > 0 {
		availabilityRate = math.Round(float64(schedulableCount)/float64(total)*10000) / 100
	}

	var recentErrorCount int64
	if h.opsService != nil {
		oneHourAgo := now.Add(-time.Hour).UTC()
		nowUTC := now.UTC()
		logs, err := h.opsService.ListSystemLogs(ctx, &service.OpsSystemLogFilter{
			Component: "account_state_change",
			Level:     "error",
			StartTime: &oneHourAgo,
			EndTime:   &nowUTC,
			Page:      1,
			PageSize:  1,
		})
		if err == nil {
			recentErrorCount = int64(logs.Total)
		}
	}

	response.Success(c, gin.H{
		"total_accounts":     total,
		"active_count":       activeCount,
		"error_count":        errorCount,
		"rate_limited_count": rateLimitedCount,
		"overloaded_count":   overloadedCount,
		"schedulable_count":  schedulableCount,
		"availability_rate":  availabilityRate,
		"recent_errors_1h":   recentErrorCount,
		"accounts":           healthAccounts,
	})
}
