package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cloud-platform/collaborative-dev/cmd/iam-service/services"
	"github.com/cloud-platform/collaborative-dev/shared/api"
	"github.com/cloud-platform/collaborative-dev/shared/logger"
)

// SessionHandler 会话处理器
type SessionHandler struct {
	sessionService *services.SessionManagementService
	logger         logger.Logger
	respHandler    *api.ResponseHandler
}

// NewSessionHandler 创建会话处理器
func NewSessionHandler(sessionService *services.SessionManagementService, logger logger.Logger) *SessionHandler {
	return &SessionHandler{
		sessionService: sessionService,
		logger:         logger,
		respHandler:    api.NewResponseHandler(),
	}
}

// GetSessions 获取用户会话列表
func (h *SessionHandler) GetSessions(c *gin.Context) {
	// 获取当前用户信息
	userID, exists := c.Get("user_id")
	if !exists {
		h.respHandler.Unauthorized(c, "用户未认证")
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		h.respHandler.BadRequest(c, "缺少租户信息", nil)
		return
	}

	// 获取查询参数
	onlyActive := c.DefaultQuery("only_active", "false") == "true"

	req := &services.GetUserSessionsRequest{
		UserID:     userID.(uuid.UUID),
		TenantID:   tenantID.(uuid.UUID),
		OnlyActive: onlyActive,
	}

	// 获取当前会话令牌
	authHeader := c.GetHeader("Authorization")
	var currentToken string
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		currentToken = authHeader[7:]
	}

	sessions, err := h.sessionService.GetUserSessions(c.Request.Context(), req, currentToken)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"user_id":   userID,
			"tenant_id": tenantID,
			"error":     err.Error(),
		}).Error("获取用户会话失败")

		h.respHandler.InternalServerError(c, err.Error())
		return
	}

	h.respHandler.OK(c, "获取会话列表成功", sessions)
}

// RevokeSession 撤销单个会话
func (h *SessionHandler) RevokeSession(c *gin.Context) {
	sessionIDStr := c.Param("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		h.respHandler.BadRequest(c, "无效的会话ID", nil)
		return
	}

	// 获取当前用户信息
	userID, exists := c.Get("user_id")
	if !exists {
		h.respHandler.Unauthorized(c, "用户未认证")
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		h.respHandler.BadRequest(c, "缺少租户信息", nil)
		return
	}

	err = h.sessionService.RevokeSession(c.Request.Context(), sessionID, userID.(uuid.UUID), tenantID.(uuid.UUID), "user_revoked")
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"user_id":    userID,
			"tenant_id":  tenantID,
			"session_id": sessionID,
			"error":      err.Error(),
		}).Error("撤销会话失败")

		h.respHandler.InternalServerError(c, err.Error())
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"user_id":    userID,
		"tenant_id":  tenantID,
		"session_id": sessionID,
	}).Info("会话撤销成功")

	h.respHandler.OK(c, "会话已撤销", nil)
}

// RevokeAllSessions 撤销用户所有其他会话
func (h *SessionHandler) RevokeAllSessions(c *gin.Context) {
	// 获取当前用户信息
	userID, exists := c.Get("user_id")
	if !exists {
		h.respHandler.Unauthorized(c, "用户未认证")
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		h.respHandler.BadRequest(c, "缺少租户信息", nil)
		return
	}

	// 解析请求参数
	type RevokeAllRequest struct {
		ExcludeCurrent bool `json:"exclude_current"`
	}

	var req RevokeAllRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果没有请求体，默认排除当前会话
		req.ExcludeCurrent = true
	}

	var excludeSessionID *uuid.UUID
	if req.ExcludeCurrent {
		// 这里需要获取当前会话ID，简化处理暂时设为nil
		// 在实际实现中应该从JWT或会话中获取当前会话ID
	}

	err := h.sessionService.RevokeUserSessions(c.Request.Context(), userID.(uuid.UUID), tenantID.(uuid.UUID), excludeSessionID, "user_revoked_all")
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"user_id":   userID,
			"tenant_id": tenantID,
			"error":     err.Error(),
		}).Error("撤销所有会话失败")

		h.respHandler.InternalServerError(c, err.Error())
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"user_id":         userID,
		"tenant_id":       tenantID,
		"exclude_current": req.ExcludeCurrent,
	}).Info("所有会话撤销成功")

	h.respHandler.OK(c, "所有其他会话已撤销", nil)
}

// GetSessionStats 获取会话统计信息（管理员功能）
func (h *SessionHandler) GetSessionStats(c *gin.Context) {
	// 检查管理员权限
	userRole, exists := c.Get("user_role")
	if !exists || userRole != "admin" {
		h.respHandler.Forbidden(c, "权限不足")
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		h.respHandler.BadRequest(c, "缺少租户信息", nil)
		return
	}

	// 解析时间范围参数
	startTimeStr := c.DefaultQuery("start_time", "")
	endTimeStr := c.DefaultQuery("end_time", "")

	var startTime, endTime time.Time
	var err error

	if startTimeStr != "" {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			h.respHandler.BadRequest(c, "开始时间格式无效", nil)
			return
		}
	} else {
		startTime = time.Now().AddDate(0, 0, -7) // 默认查询最近7天
	}

	if endTimeStr != "" {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			h.respHandler.BadRequest(c, "结束时间格式无效", nil)
			return
		}
	} else {
		endTime = time.Now()
	}

	stats, err := h.sessionService.GetSessionStats(c.Request.Context(), tenantID.(uuid.UUID), startTime, endTime)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"tenant_id":  tenantID,
			"start_time": startTime,
			"end_time":   endTime,
			"error":      err.Error(),
		}).Error("获取会话统计失败")

		h.respHandler.InternalServerError(c, err.Error())
		return
	}

	h.respHandler.OK(c, "获取会话统计成功", stats)
}

// AdminRevokeUserSessions 管理员强制撤销用户会话
func (h *SessionHandler) AdminRevokeUserSessions(c *gin.Context) {
	// 检查管理员权限
	userRole, exists := c.Get("user_role")
	if !exists || userRole != "admin" {
		h.respHandler.Forbidden(c, "权限不足")
		return
	}

	targetUserIDStr := c.Param("user_id")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		h.respHandler.BadRequest(c, "无效的用户ID", nil)
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		h.respHandler.BadRequest(c, "缺少租户信息", nil)
		return
	}

	adminUserID, _ := c.Get("user_id")

	err = h.sessionService.RevokeUserSessions(c.Request.Context(), targetUserID, tenantID.(uuid.UUID), nil, "admin_revoked")
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"admin_user_id":  adminUserID,
			"target_user_id": targetUserID,
			"tenant_id":      tenantID,
			"error":          err.Error(),
		}).Error("管理员撤销用户会话失败")

		h.respHandler.InternalServerError(c, err.Error())
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"admin_user_id":  adminUserID,
		"target_user_id": targetUserID,
		"tenant_id":      tenantID,
	}).Info("管理员强制撤销用户会话成功")

	h.respHandler.OK(c, "用户所有会话已撤销", nil)
}
