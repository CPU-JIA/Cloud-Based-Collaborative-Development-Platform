package handlers

import (
	"net/http"
	"strconv"

	"github.com/cloud-platform/collaborative-dev/internal/git-gateway/models"
	"github.com/cloud-platform/collaborative-dev/internal/git-gateway/service"
	"github.com/cloud-platform/collaborative-dev/shared/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// WebhookHandler Webhook处理器
type WebhookHandler struct {
	webhookService service.WebhookService
	logger         *zap.Logger
}

// NewWebhookHandler 创建Webhook处理器
func NewWebhookHandler(webhookService service.WebhookService, logger *zap.Logger) *WebhookHandler {
	return &WebhookHandler{
		webhookService: webhookService,
		logger:         logger,
	}
}

// 通用Git Webhook处理端点

// HandleGitWebhook 通用Git Webhook处理端点
func (h *WebhookHandler) HandleGitWebhook(c *gin.Context) {
	repositoryIDStr := c.Param("repository_id")
	repositoryID, err := uuid.Parse(repositoryIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid repository ID", err)
		return
	}

	// 获取事件类型（从Header或查询参数）
	eventType := c.GetHeader("X-Event-Type")
	if eventType == "" {
		eventType = c.Query("event_type")
	}

	if eventType == "" {
		response.Error(c, http.StatusBadRequest, "Missing event type", nil)
		return
	}

	// 获取签名（用于验证）
	signature := c.GetHeader("X-Hub-Signature")

	// 解析请求体
	var eventData map[string]interface{}
	if err := c.ShouldBindJSON(&eventData); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// 创建Webhook事件
	req := &models.CreateWebhookEventRequest{
		RepositoryID: repositoryID,
		EventType:    models.WebhookEventType(eventType),
		EventData:    eventData,
		Source:       "git",
		Signature:    signature,
	}

	event, err := h.webhookService.CreateWebhookEvent(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("处理Git Webhook失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to process webhook", err)
		return
	}

	response.Success(c, http.StatusOK, "Webhook processed successfully", gin.H{
		"event_id": event.ID,
		"status":   "processing",
	})
}

// GitHub兼容的Webhook端点

// HandleGitHubWebhook GitHub Webhook处理端点
func (h *WebhookHandler) HandleGitHubWebhook(c *gin.Context) {
	repositoryIDStr := c.Param("repository_id")
	repositoryID, err := uuid.Parse(repositoryIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid repository ID", err)
		return
	}

	// GitHub事件类型
	eventType := c.GetHeader("X-GitHub-Event")
	if eventType == "" {
		response.Error(c, http.StatusBadRequest, "Missing X-GitHub-Event header", nil)
		return
	}

	// GitHub签名
	signature := c.GetHeader("X-Hub-Signature-256")

	// 解析请求体
	var eventData map[string]interface{}
	if err := c.ShouldBindJSON(&eventData); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// 转换GitHub事件类型到内部事件类型
	internalEventType := h.mapGitHubEventType(eventType)

	req := &models.CreateWebhookEventRequest{
		RepositoryID: repositoryID,
		EventType:    internalEventType,
		EventData:    eventData,
		Source:       "github",
		Signature:    signature,
	}

	event, err := h.webhookService.CreateWebhookEvent(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("处理GitHub Webhook失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to process webhook", err)
		return
	}

	response.Success(c, http.StatusOK, "GitHub webhook processed successfully", gin.H{
		"event_id": event.ID,
		"status":   "processing",
	})
}

// GitLab兼容的Webhook端点

// HandleGitLabWebhook GitLab Webhook处理端点
func (h *WebhookHandler) HandleGitLabWebhook(c *gin.Context) {
	repositoryIDStr := c.Param("repository_id")
	repositoryID, err := uuid.Parse(repositoryIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid repository ID", err)
		return
	}

	// GitLab事件类型
	eventType := c.GetHeader("X-Gitlab-Event")
	if eventType == "" {
		response.Error(c, http.StatusBadRequest, "Missing X-Gitlab-Event header", nil)
		return
	}

	// GitLab Token
	token := c.GetHeader("X-Gitlab-Token")

	// 解析请求体
	var eventData map[string]interface{}
	if err := c.ShouldBindJSON(&eventData); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// 转换GitLab事件类型到内部事件类型
	internalEventType := h.mapGitLabEventType(eventType)

	req := &models.CreateWebhookEventRequest{
		RepositoryID: repositoryID,
		EventType:    internalEventType,
		EventData:    eventData,
		Source:       "gitlab",
		Signature:    token,
	}

	event, err := h.webhookService.CreateWebhookEvent(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("处理GitLab Webhook失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to process webhook", err)
		return
	}

	response.Success(c, http.StatusOK, "GitLab webhook processed successfully", gin.H{
		"event_id": event.ID,
		"status":   "processing",
	})
}

// Webhook事件管理端点

// CreateWebhookEvent 创建Webhook事件
func (h *WebhookHandler) CreateWebhookEvent(c *gin.Context) {
	var req models.CreateWebhookEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	event, err := h.webhookService.CreateWebhookEvent(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("创建Webhook事件失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to create webhook event", err)
		return
	}

	response.Success(c, http.StatusCreated, "Webhook event created successfully", event)
}

// GetWebhookEvent 获取Webhook事件
func (h *WebhookHandler) GetWebhookEvent(c *gin.Context) {
	eventIDStr := c.Param("event_id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid event ID", err)
		return
	}

	event, err := h.webhookService.GetWebhookEvent(c.Request.Context(), eventID)
	if err != nil {
		h.logger.Error("获取Webhook事件失败", zap.Error(err))
		response.Error(c, http.StatusNotFound, "Webhook event not found", err)
		return
	}

	response.Success(c, http.StatusOK, "Webhook event retrieved successfully", event)
}

// ListWebhookEvents 列出Webhook事件
func (h *WebhookHandler) ListWebhookEvents(c *gin.Context) {
	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 构建过滤器
	filter := &models.WebhookEventFilter{}

	if repositoryIDStr := c.Query("repository_id"); repositoryIDStr != "" {
		if repositoryID, err := uuid.Parse(repositoryIDStr); err == nil {
			filter.RepositoryID = &repositoryID
		}
	}

	if source := c.Query("source"); source != "" {
		filter.Source = source
	}

	if processedStr := c.Query("processed"); processedStr != "" {
		if processed, err := strconv.ParseBool(processedStr); err == nil {
			filter.Processed = &processed
		}
	}

	// TODO: 解析时间范围参数

	resp, err := h.webhookService.ListWebhookEvents(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		h.logger.Error("列出Webhook事件失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to list webhook events", err)
		return
	}

	response.Success(c, http.StatusOK, "Webhook events retrieved successfully", resp)
}

// ProcessWebhookEvent 处理Webhook事件
func (h *WebhookHandler) ProcessWebhookEvent(c *gin.Context) {
	eventIDStr := c.Param("event_id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid event ID", err)
		return
	}

	if err := h.webhookService.ProcessWebhookEvent(c.Request.Context(), eventID); err != nil {
		h.logger.Error("处理Webhook事件失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to process webhook event", err)
		return
	}

	response.Success(c, http.StatusOK, "Webhook event processed successfully", nil)
}

// DeleteWebhookEvent 删除Webhook事件
func (h *WebhookHandler) DeleteWebhookEvent(c *gin.Context) {
	eventIDStr := c.Param("event_id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid event ID", err)
		return
	}

	if err := h.webhookService.DeleteWebhookEvent(c.Request.Context(), eventID); err != nil {
		h.logger.Error("删除Webhook事件失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to delete webhook event", err)
		return
	}

	response.Success(c, http.StatusOK, "Webhook event deleted successfully", nil)
}

// Webhook触发器管理端点

// CreateWebhookTrigger 创建钩子触发器
func (h *WebhookHandler) CreateWebhookTrigger(c *gin.Context) {
	var req models.CreateWebhookTriggerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	trigger, err := h.webhookService.CreateWebhookTrigger(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("创建钩子触发器失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to create webhook trigger", err)
		return
	}

	response.Success(c, http.StatusCreated, "Webhook trigger created successfully", trigger)
}

// GetWebhookTrigger 获取钩子触发器
func (h *WebhookHandler) GetWebhookTrigger(c *gin.Context) {
	triggerIDStr := c.Param("trigger_id")
	triggerID, err := uuid.Parse(triggerIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid trigger ID", err)
		return
	}

	trigger, err := h.webhookService.GetWebhookTrigger(c.Request.Context(), triggerID)
	if err != nil {
		h.logger.Error("获取钩子触发器失败", zap.Error(err))
		response.Error(c, http.StatusNotFound, "Webhook trigger not found", err)
		return
	}

	response.Success(c, http.StatusOK, "Webhook trigger retrieved successfully", trigger)
}

// ListWebhookTriggers 列出钩子触发器
func (h *WebhookHandler) ListWebhookTriggers(c *gin.Context) {
	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 解析仓库ID过滤
	var repositoryID *uuid.UUID
	if repositoryIDStr := c.Query("repository_id"); repositoryIDStr != "" {
		if id, err := uuid.Parse(repositoryIDStr); err == nil {
			repositoryID = &id
		}
	}

	resp, err := h.webhookService.ListWebhookTriggers(c.Request.Context(), repositoryID, page, pageSize)
	if err != nil {
		h.logger.Error("列出钩子触发器失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to list webhook triggers", err)
		return
	}

	response.Success(c, http.StatusOK, "Webhook triggers retrieved successfully", resp)
}

// UpdateWebhookTrigger 更新钩子触发器
func (h *WebhookHandler) UpdateWebhookTrigger(c *gin.Context) {
	triggerIDStr := c.Param("trigger_id")
	triggerID, err := uuid.Parse(triggerIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid trigger ID", err)
		return
	}

	var req models.UpdateWebhookTriggerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	trigger, err := h.webhookService.UpdateWebhookTrigger(c.Request.Context(), triggerID, &req)
	if err != nil {
		h.logger.Error("更新钩子触发器失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to update webhook trigger", err)
		return
	}

	response.Success(c, http.StatusOK, "Webhook trigger updated successfully", trigger)
}

// DeleteWebhookTrigger 删除钩子触发器
func (h *WebhookHandler) DeleteWebhookTrigger(c *gin.Context) {
	triggerIDStr := c.Param("trigger_id")
	triggerID, err := uuid.Parse(triggerIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid trigger ID", err)
		return
	}

	if err := h.webhookService.DeleteWebhookTrigger(c.Request.Context(), triggerID); err != nil {
		h.logger.Error("删除钩子触发器失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to delete webhook trigger", err)
		return
	}

	response.Success(c, http.StatusOK, "Webhook trigger deleted successfully", nil)
}

// EnableWebhookTrigger 启用/禁用钩子触发器
func (h *WebhookHandler) EnableWebhookTrigger(c *gin.Context) {
	triggerIDStr := c.Param("trigger_id")
	triggerID, err := uuid.Parse(triggerIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid trigger ID", err)
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := h.webhookService.EnableWebhookTrigger(c.Request.Context(), triggerID, req.Enabled); err != nil {
		h.logger.Error("启用/禁用钩子触发器失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to enable/disable webhook trigger", err)
		return
	}

	action := "disabled"
	if req.Enabled {
		action = "enabled"
	}

	response.Success(c, http.StatusOK, "Webhook trigger "+action+" successfully", gin.H{
		"trigger_id": triggerID,
		"enabled":    req.Enabled,
	})
}

// 统计和监控端点

// GetWebhookStatistics 获取Webhook统计信息
func (h *WebhookHandler) GetWebhookStatistics(c *gin.Context) {
	var repositoryID *uuid.UUID
	if repositoryIDStr := c.Query("repository_id"); repositoryIDStr != "" {
		if id, err := uuid.Parse(repositoryIDStr); err == nil {
			repositoryID = &id
		}
	}

	stats, err := h.webhookService.GetWebhookStatistics(c.Request.Context(), repositoryID)
	if err != nil {
		h.logger.Error("获取Webhook统计信息失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to get webhook statistics", err)
		return
	}

	response.Success(c, http.StatusOK, "Webhook statistics retrieved successfully", stats)
}

// 辅助方法

// mapGitHubEventType 映射GitHub事件类型到内部事件类型
func (h *WebhookHandler) mapGitHubEventType(githubEvent string) models.WebhookEventType {
	switch githubEvent {
	case "push":
		return models.EventTypePush
	case "pull_request":
		return models.EventTypePullRequest
	case "create":
		return models.EventTypeBranchCreate // 或TagCreate，需要进一步判断
	case "delete":
		return models.EventTypeBranchDelete // 或TagDelete，需要进一步判断
	case "repository":
		return models.EventTypeRepoUpdate
	default:
		return models.WebhookEventType(githubEvent)
	}
}

// mapGitLabEventType 映射GitLab事件类型到内部事件类型
func (h *WebhookHandler) mapGitLabEventType(gitlabEvent string) models.WebhookEventType {
	switch gitlabEvent {
	case "Push Hook":
		return models.EventTypePush
	case "Merge Request Hook":
		return models.EventTypePullRequest
	case "Tag Push Hook":
		return models.EventTypeTagPush
	default:
		return models.WebhookEventType(gitlabEvent)
	}
}
