package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cloud-platform/collaborative-dev/cmd/iam-service/services"
	"github.com/cloud-platform/collaborative-dev/shared/logger"
	"github.com/cloud-platform/collaborative-dev/shared/middleware"
)

// MFAHandler MFA处理器
type MFAHandler struct {
	mfaService *services.MFAManagementService
	logger     logger.Logger
}

// NewMFAHandler 创建MFA处理器
func NewMFAHandler(mfaService *services.MFAManagementService, logger logger.Logger) *MFAHandler {
	return &MFAHandler{
		mfaService: mfaService,
		logger:     logger,
	}
}

// EnableMFA 启用MFA
func (h *MFAHandler) EnableMFA(c *gin.Context) {
	var req services.EnableMFARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, StandardResponse{
			Success: false,
			Error:   "请求参数无效",
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	// 获取当前用户信息
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, StandardResponse{
			Success: false,
			Error:   "用户未认证",
			Code:    "UNAUTHORIZED",
		})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusBadRequest, StandardResponse{
			Success: false,
			Error:   "缺少租户信息",
			Code:    "MISSING_TENANT",
		})
		return
	}

	req.UserID = userID.(uuid.UUID)
	req.TenantID = tenantID.(uuid.UUID)

	// 获取客户端信息
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	response, err := h.mfaService.EnableMFA(c.Request.Context(), &req, ipAddress, userAgent)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"user_id":    req.UserID,
			"tenant_id":  req.TenantID,
			"error":      err.Error(),
			"ip_address": ipAddress,
		}).Error("启用MFA失败")

		c.JSON(http.StatusInternalServerError, StandardResponse{
			Success: false,
			Error:   "启用MFA失败",
			Code:    "ENABLE_MFA_FAILED",
			Message: err.Error(),
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"user_id":   req.UserID,
		"tenant_id": req.TenantID,
		"device_id": response.DeviceID,
	}).Info("MFA启用成功")

	c.JSON(http.StatusOK, StandardResponse{
		Success: true,
		Data:    response,
		Message: "MFA启用成功，请使用验证器应用扫描二维码完成设置",
	})
}

// VerifyMFASetup 验证MFA设置
func (h *MFAHandler) VerifyMFASetup(c *gin.Context) {
	var req services.VerifyMFARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, StandardResponse{
			Success: false,
			Error:   "请求参数无效",
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	// 获取当前用户信息
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, StandardResponse{
			Success: false,
			Error:   "用户未认证",
			Code:    "UNAUTHORIZED",
		})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusBadRequest, StandardResponse{
			Success: false,
			Error:   "缺少租户信息",
			Code:    "MISSING_TENANT",
		})
		return
	}

	req.UserID = userID.(uuid.UUID)
	req.TenantID = tenantID.(uuid.UUID)

	// 获取客户端信息
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	err := h.mfaService.VerifyMFASetup(c.Request.Context(), &req, ipAddress, userAgent)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"user_id":    req.UserID,
			"tenant_id":  req.TenantID,
			"error":      err.Error(),
			"ip_address": ipAddress,
		}).Error("MFA设置验证失败")

		c.JSON(http.StatusBadRequest, StandardResponse{
			Success: false,
			Error:   "MFA设置验证失败",
			Code:    "MFA_SETUP_VERIFICATION_FAILED",
			Message: err.Error(),
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"user_id":   req.UserID,
		"tenant_id": req.TenantID,
	}).Info("MFA设置验证成功")

	c.JSON(http.StatusOK, StandardResponse{
		Success: true,
		Message: "MFA设置完成，两步验证已启用",
	})
}

// VerifyMFA 验证MFA（登录时使用）
func (h *MFAHandler) VerifyMFA(c *gin.Context) {
	var req services.VerifyMFARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, StandardResponse{
			Success: false,
			Error:   "请求参数无效",
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	// 获取客户端信息
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	err := h.mfaService.VerifyMFA(c.Request.Context(), &req, ipAddress, userAgent)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"user_id":    req.UserID,
			"tenant_id":  req.TenantID,
			"error":      err.Error(),
			"ip_address": ipAddress,
		}).Error("MFA验证失败")

		c.JSON(http.StatusBadRequest, StandardResponse{
			Success: false,
			Error:   "MFA验证失败",
			Code:    "MFA_VERIFICATION_FAILED",
			Message: err.Error(),
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"user_id":   req.UserID,
		"tenant_id": req.TenantID,
	}).Info("MFA验证成功")

	c.JSON(http.StatusOK, StandardResponse{
		Success: true,
		Message: "MFA验证成功",
	})
}

// GetMFADevices 获取用户MFA设备
func (h *MFAHandler) GetMFADevices(c *gin.Context) {
	// 获取当前用户信息
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, StandardResponse{
			Success: false,
			Error:   "用户未认证",
			Code:    "UNAUTHORIZED",
		})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusBadRequest, StandardResponse{
			Success: false,
			Error:   "缺少租户信息",
			Code:    "MISSING_TENANT",
		})
		return
	}

	devices, err := h.mfaService.GetUserMFADevices(c.Request.Context(), userID.(uuid.UUID), tenantID.(uuid.UUID))
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"user_id":   userID,
			"tenant_id": tenantID,
			"error":     err.Error(),
		}).Error("获取MFA设备失败")

		c.JSON(http.StatusInternalServerError, StandardResponse{
			Success: false,
			Error:   "获取MFA设备失败",
			Code:    "GET_MFA_DEVICES_FAILED",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, StandardResponse{
		Success: true,
		Data:    devices,
		Message: "获取MFA设备成功",
	})
}

// DisableMFA 禁用MFA
func (h *MFAHandler) DisableMFA(c *gin.Context) {
	// 获取当前用户信息
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, StandardResponse{
			Success: false,
			Error:   "用户未认证",
			Code:    "UNAUTHORIZED",
		})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusBadRequest, StandardResponse{
			Success: false,
			Error:   "缺少租户信息",
			Code:    "MISSING_TENANT",
		})
		return
	}

	err := h.mfaService.DisableMFA(c.Request.Context(), userID.(uuid.UUID), tenantID.(uuid.UUID))
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"user_id":   userID,
			"tenant_id": tenantID,
			"error":     err.Error(),
		}).Error("禁用MFA失败")

		c.JSON(http.StatusInternalServerError, StandardResponse{
			Success: false,
			Error:   "禁用MFA失败",
			Code:    "DISABLE_MFA_FAILED",
			Message: err.Error(),
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"user_id":   userID,
		"tenant_id": tenantID,
	}).Info("MFA禁用成功")

	c.JSON(http.StatusOK, StandardResponse{
		Success: true,
		Message: "MFA已禁用",
	})
}