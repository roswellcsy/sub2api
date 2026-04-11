package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// ApistationAuditHandler 处理 api-station 审计日志查询
type ApistationAuditHandler struct {
	opsService *service.OpsService
}

// NewApistationAuditHandler creates a new audit handler
func NewApistationAuditHandler(opsService *service.OpsService) *ApistationAuditHandler {
	return &ApistationAuditHandler{opsService: opsService}
}

// GetAccountAudit 获取指定账号的审计日志（请求审计 + 状态变更事件）
// GET /api/v1/admin/audit/account/:id
func (h *ApistationAuditHandler) GetAccountAudit(c *gin.Context) {
	if h.opsService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Ops service not available")
		return
	}

	accountIDStr := strings.TrimSpace(c.Param("id"))
	accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
	if err != nil || accountID <= 0 {
		response.BadRequest(c, "invalid account id")
		return
	}

	pageSize := 50
	if l := strings.TrimSpace(c.Query("limit")); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			pageSize = n
		}
	}

	ctx := c.Request.Context()

	auditLogs, err := h.opsService.ListSystemLogs(ctx, &service.OpsSystemLogFilter{
		Component: "request_audit",
		AccountID: &accountID,
		Page:      1,
		PageSize:  pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	stateEvents, err := h.opsService.ListSystemLogs(ctx, &service.OpsSystemLogFilter{
		Component: "account_state_change",
		AccountID: &accountID,
		Page:      1,
		PageSize:  pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{
		"account_id":   accountID,
		"audit_logs":   auditLogs.Logs,
		"state_events": stateEvents.Logs,
	})
}

// GetRecentErrors 获取最近的错误事件汇总（account_state_change 中的 error 级别）
// GET /api/v1/admin/audit/recent-errors
func (h *ApistationAuditHandler) GetRecentErrors(c *gin.Context) {
	if h.opsService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Ops service not available")
		return
	}

	pageSize := 100
	if l := strings.TrimSpace(c.Query("limit")); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			pageSize = n
		}
	}

	ctx := c.Request.Context()

	logs, err := h.opsService.ListSystemLogs(ctx, &service.OpsSystemLogFilter{
		Component: "account_state_change",
		Level:     "error",
		Page:      1,
		PageSize:  pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{
		"errors": logs.Logs,
		"total":  logs.Total,
	})
}
