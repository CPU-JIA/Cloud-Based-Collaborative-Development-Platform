package handlers

import (
	"net/http"
	"strconv"

	"github.com/cloud-platform/collaborative-dev/internal/notification-service/services"
	"github.com/cloud-platform/collaborative-dev/shared/logger"
	"github.com/cloud-platform/collaborative-dev/shared/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// NotificationHandler 通知处理器
type NotificationHandler struct {
	notificationService *services.NotificationService
	logger              logger.Logger
}

// NewNotificationHandler 创建新的通知处理器
func NewNotificationHandler(notificationService *services.NotificationService, appLogger logger.Logger) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
		logger:              appLogger,
	}
}

// GetNotifications 获取通知列表
// GET /api/v1/notifications
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	// 从JWT中获取用户信息
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", "USER_NOT_FOUND")
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", "TENANT_NOT_FOUND")
		return
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID", "INVALID_USER_ID")
		return
	}

	tenantUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid tenant ID", "INVALID_TENANT_ID")
		return
	}

	// 解析查询参数
	options := &services.GetNotificationsOptions{}

	if projectID := c.Query("project_id"); projectID != "" {
		projectUUID, err := uuid.Parse(projectID)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "Invalid project ID", "INVALID_PROJECT_ID")
			return
		}
		options.ProjectID = &projectUUID
	}

	options.Category = c.Query("category")
	options.Status = c.Query("status")
	options.Priority = c.Query("priority")
	options.SortBy = c.DefaultQuery("sort_by", "created_at")
	options.SortOrder = c.DefaultQuery("sort_order", "desc")

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 100 {
			options.Limit = l
		} else {
			options.Limit = 20 // 默认限制
		}
	} else {
		options.Limit = 20
	}

	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			options.Offset = o
		}
	}

	// 获取通知列表
	notifications, err := h.notificationService.GetNotifications(c.Request.Context(), userUUID, tenantUUID, options)
	if err != nil {
		h.logger.Error("Failed to get notifications:", err)
		response.Error(c, http.StatusInternalServerError, "Failed to get notifications", "GET_NOTIFICATIONS_FAILED")
		return
	}

	response.Success(c, http.StatusOK, "Success", map[string]interface{}{
		"notifications": notifications,
		"total":         len(notifications),
		"limit":         options.Limit,
		"offset":        options.Offset,
	})
}

// GetUnreadCount 获取未读通知数量
// GET /api/v1/notifications/unread/count
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	// 从JWT中获取用户信息
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", "USER_NOT_FOUND")
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", "TENANT_NOT_FOUND")
		return
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID", "INVALID_USER_ID")
		return
	}

	tenantUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid tenant ID", "INVALID_TENANT_ID")
		return
	}

	// 获取未读数量
	count, err := h.notificationService.GetUnreadCount(c.Request.Context(), userUUID, tenantUUID)
	if err != nil {
		h.logger.Error("Failed to get unread count:", err)
		response.Error(c, http.StatusInternalServerError, "Failed to get unread count", "GET_UNREAD_COUNT_FAILED")
		return
	}

	response.Success(c, http.StatusOK, "Success", map[string]interface{}{
		"unread_count": count,
	})
}

// MarkAsRead 标记通知为已读
// POST /api/v1/notifications/:id/read
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	// 获取通知ID
	notificationIDStr := c.Param("id")
	notificationID, err := uuid.Parse(notificationIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid notification ID", "INVALID_NOTIFICATION_ID")
		return
	}

	// 从JWT中获取用户信息
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", "USER_NOT_FOUND")
		return
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID", "INVALID_USER_ID")
		return
	}

	// 标记为已读
	if err := h.notificationService.MarkAsRead(c.Request.Context(), notificationID, userUUID); err != nil {
		h.logger.Error("Failed to mark notification as read:", err)
		response.Error(c, http.StatusInternalServerError, "Failed to mark as read", "MARK_READ_FAILED")
		return
	}

	response.Success(c, http.StatusOK, "Success", map[string]interface{}{
		"message": "Notification marked as read successfully",
	})
}

// DeleteNotification 删除通知
// DELETE /api/v1/notifications/:id
func (h *NotificationHandler) DeleteNotification(c *gin.Context) {
	// 获取通知ID
	notificationIDStr := c.Param("id")
	notificationID, err := uuid.Parse(notificationIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid notification ID", "INVALID_NOTIFICATION_ID")
		return
	}

	// 从JWT中获取用户信息
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", "USER_NOT_FOUND")
		return
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID", "INVALID_USER_ID")
		return
	}

	// 删除通知
	if err := h.notificationService.DeleteNotification(c.Request.Context(), notificationID, userUUID); err != nil {
		h.logger.Error("Failed to delete notification:", err)
		response.Error(c, http.StatusInternalServerError, "Failed to delete notification", "DELETE_NOTIFICATION_FAILED")
		return
	}

	response.Success(c, http.StatusOK, "Success", map[string]interface{}{
		"message": "Notification deleted successfully",
	})
}

// CreateNotification 创建通知 (管理员接口)
// POST /api/v1/notifications
func (h *NotificationHandler) CreateNotification(c *gin.Context) {
	// 从JWT中获取操作用户信息
	createdBy, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", "USER_NOT_FOUND")
		return
	}

	createdByUUID, err := uuid.Parse(createdBy.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID", "INVALID_USER_ID")
		return
	}

	// 解析请求体
	var req services.CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	// 设置创建者
	req.CreatedBy = createdByUUID

	// 验证必填字段
	if req.TenantID == uuid.Nil {
		response.Error(c, http.StatusBadRequest, "Tenant ID is required", "TENANT_ID_REQUIRED")
		return
	}

	if req.Type == "" {
		response.Error(c, http.StatusBadRequest, "Type is required", "TYPE_REQUIRED")
		return
	}

	if req.Category == "" {
		response.Error(c, http.StatusBadRequest, "Category is required", "CATEGORY_REQUIRED")
		return
	}

	// 创建通知
	if err := h.notificationService.CreateNotification(c.Request.Context(), &req); err != nil {
		h.logger.Error("Failed to create notification:", err)
		response.Error(c, http.StatusInternalServerError, "Failed to create notification", "CREATE_NOTIFICATION_FAILED")
		return
	}

	response.Success(c, http.StatusCreated, "Success", map[string]interface{}{
		"message": "Notification created successfully",
	})
}

// RetryNotification 重试失败的通知
// POST /api/v1/notifications/:id/retry
func (h *NotificationHandler) RetryNotification(c *gin.Context) {
	// 获取通知ID
	notificationIDStr := c.Param("id")
	notificationID, err := uuid.Parse(notificationIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid notification ID", "INVALID_NOTIFICATION_ID")
		return
	}

	// 重试通知
	if err := h.notificationService.RetryFailedNotification(c.Request.Context(), notificationID); err != nil {
		h.logger.Error("Failed to retry notification:", err)
		response.Error(c, http.StatusInternalServerError, "Failed to retry notification", "RETRY_NOTIFICATION_FAILED")
		return
	}

	response.Success(c, http.StatusOK, "Success", map[string]interface{}{
		"message": "Notification retry initiated successfully",
	})
}

// GetNotificationsByCorrelationID 根据关联ID获取通知
// GET /api/v1/notifications/correlation/:correlation_id
func (h *NotificationHandler) GetNotificationsByCorrelationID(c *gin.Context) {
	correlationID := c.Param("correlation_id")
	if correlationID == "" {
		response.Error(c, http.StatusBadRequest, "Correlation ID is required", "CORRELATION_ID_REQUIRED")
		return
	}

	// 从JWT中获取租户信息
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", "TENANT_NOT_FOUND")
		return
	}

	tenantUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid tenant ID", "INVALID_TENANT_ID")
		return
	}

	// 获取通知列表
	notifications, err := h.notificationService.GetNotificationsByCorrelationID(c.Request.Context(), correlationID, tenantUUID)
	if err != nil {
		h.logger.Error("Failed to get notifications by correlation ID:", err)
		response.Error(c, http.StatusInternalServerError, "Failed to get notifications", "GET_NOTIFICATIONS_FAILED")
		return
	}

	response.Success(c, http.StatusOK, "Success", map[string]interface{}{
		"notifications":  notifications,
		"correlation_id": correlationID,
		"total":          len(notifications),
	})
}
