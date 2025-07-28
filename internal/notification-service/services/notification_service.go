package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/notification-service/models"
	"github.com/cloud-platform/collaborative-dev/internal/notification-service/repository"
	"github.com/cloud-platform/collaborative-dev/shared/logger"
	"github.com/google/uuid"
)

// NotificationService 通知服务
type NotificationService struct {
	notificationRepo *repository.NotificationRepository
	templateRepo     *repository.TemplateRepository
	ruleRepo         *repository.RuleRepository
	templateEngine   *TemplateEngine
	deliveryService  *DeliveryService
	logger           logger.Logger
}

// NewNotificationService 创建新的通知服务
func NewNotificationService(
	notificationRepo *repository.NotificationRepository,
	templateRepo *repository.TemplateRepository,
	ruleRepo *repository.RuleRepository,
	templateEngine *TemplateEngine,
	deliveryService *DeliveryService,
	appLogger logger.Logger,
) *NotificationService {
	return &NotificationService{
		notificationRepo: notificationRepo,
		templateRepo:     templateRepo,
		ruleRepo:         ruleRepo,
		templateEngine:   templateEngine,
		deliveryService:  deliveryService,
		logger:           appLogger,
	}
}

// CreateNotificationRequest 创建通知请求
type CreateNotificationRequest struct {
	UserID        *uuid.UUID       `json:"user_id,omitempty"`
	TenantID      uuid.UUID        `json:"tenant_id"`
	ProjectID     *uuid.UUID       `json:"project_id,omitempty"`
	Type          string           `json:"type"`
	Category      string           `json:"category"`
	Priority      string           `json:"priority"`
	Title         string           `json:"title,omitempty"`
	Content       string           `json:"content,omitempty"`
	Channels      *models.Channels `json:"channels,omitempty"`
	TemplateID    *uuid.UUID       `json:"template_id,omitempty"`
	EventData     json.RawMessage  `json:"event_data,omitempty"`
	Metadata      json.RawMessage  `json:"metadata,omitempty"`
	CorrelationID string           `json:"correlation_id,omitempty"`
	SourceEvent   string           `json:"source_event,omitempty"`
	CreatedBy     uuid.UUID        `json:"created_by"`
}

// CreateNotification 创建通知
func (ns *NotificationService) CreateNotification(ctx context.Context, req *CreateNotificationRequest) error {
	ns.logger.Info(fmt.Sprintf("Creating notification: type=%s, category=%s, tenant_id=%s",
		req.Type, req.Category, req.TenantID))

	// 1. 应用通知规则
	rules, err := ns.ruleRepo.GetActiveRulesByTypeAndTenant(ctx, req.Type, req.TenantID, req.UserID, req.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to get notification rules: %w", err)
	}

	// 如果没有匹配的规则，使用默认处理逻辑
	if len(rules) == 0 {
		return ns.createNotificationWithDefaults(ctx, req)
	}

	// 2. 为每个匹配的规则创建通知
	for _, rule := range rules {
		if err := ns.createNotificationFromRule(ctx, req, rule); err != nil {
			ns.logger.Error(fmt.Sprintf("Failed to create notification from rule %s: %v", rule.ID, err))
			continue
		}
	}

	return nil
}

// createNotificationWithDefaults 使用默认设置创建通知
func (ns *NotificationService) createNotificationWithDefaults(ctx context.Context, req *CreateNotificationRequest) error {
	// 查找默认模板
	template, err := ns.templateRepo.GetDefaultTemplateByType(ctx, req.Type, req.TenantID)
	if err != nil {
		ns.logger.Warn(fmt.Sprintf("No default template found for type %s, creating basic notification", req.Type))
	}

	// 创建通知记录
	notification := &models.Notification{
		UserID:        req.UserID,
		TenantID:      req.TenantID,
		ProjectID:     req.ProjectID,
		Type:          req.Type,
		Category:      req.Category,
		Priority:      req.Priority,
		Title:         req.Title,
		Content:       req.Content,
		Status:        models.StatusPending,
		EventData:     req.EventData,
		Metadata:      req.Metadata,
		SourceEvent:   req.SourceEvent,
		CorrelationID: req.CorrelationID,
		CreatedBy:     req.CreatedBy,
		MaxRetries:    3,
	}

	// 设置渠道配置
	if req.Channels != nil {
		notification.Channels = *req.Channels
	} else {
		// 使用默认渠道配置
		notification.Channels = models.Channels{
			InApp: &models.InAppChannel{
				Enabled: true,
				Badge:   true,
			},
		}
	}

	// 如果有模板，设置模板ID并渲染内容
	if template != nil {
		notification.TemplateID = &template.ID

		if err := ns.renderNotificationContent(ctx, notification, template); err != nil {
			ns.logger.Error(fmt.Sprintf("Failed to render template content: %v", err))
		}
	}

	// 保存通知记录
	if err := ns.notificationRepo.Create(ctx, notification); err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	// 异步发送通知
	go func() {
		if err := ns.deliveryService.DeliverNotification(context.Background(), notification); err != nil {
			ns.logger.Error(fmt.Sprintf("Failed to deliver notification %s: %v", notification.ID, err))
		}
	}()

	ns.logger.Info(fmt.Sprintf("Notification created successfully: id=%s", notification.ID))
	return nil
}

// createNotificationFromRule 根据规则创建通知
func (ns *NotificationService) createNotificationFromRule(ctx context.Context, req *CreateNotificationRequest, rule *models.NotificationRule) error {
	// 检查规则条件
	if !ns.checkRuleConditions(req, rule) {
		ns.logger.Debug(fmt.Sprintf("Rule conditions not met for rule %s", rule.ID))
		return nil
	}

	// 检查频率限制
	if rule.RateLimit != nil {
		allowed, err := ns.checkRateLimit(ctx, req, rule)
		if err != nil {
			return fmt.Errorf("failed to check rate limit: %w", err)
		}
		if !allowed {
			ns.logger.Info(fmt.Sprintf("Rate limit exceeded for rule %s", rule.ID))
			return nil
		}
	}

	// 检查免打扰时间
	if rule.QuietHours != nil && rule.QuietHours.Enabled {
		if ns.isInQuietHours(rule.QuietHours) {
			ns.logger.Info(fmt.Sprintf("Notification suppressed due to quiet hours for rule %s", rule.ID))
			return nil
		}
	}

	// 获取模板
	template, err := ns.templateRepo.GetByID(ctx, rule.TemplateID)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	// 创建通知记录
	notification := &models.Notification{
		UserID:        req.UserID,
		TenantID:      req.TenantID,
		ProjectID:     req.ProjectID,
		Type:          req.Type,
		Category:      req.Category,
		Priority:      rule.Priority, // 使用规则中定义的优先级
		Status:        models.StatusPending,
		EventData:     req.EventData,
		Metadata:      req.Metadata,
		SourceEvent:   req.SourceEvent,
		CorrelationID: req.CorrelationID,
		CreatedBy:     req.CreatedBy,
		TemplateID:    &template.ID,
		Channels:      rule.Channels,
		MaxRetries:    3,
	}

	// 渲染通知内容
	if err := ns.renderNotificationContent(ctx, notification, template); err != nil {
		return fmt.Errorf("failed to render notification content: %w", err)
	}

	// 保存通知记录
	if err := ns.notificationRepo.Create(ctx, notification); err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	// 异步发送通知
	go func() {
		if err := ns.deliveryService.DeliverNotification(context.Background(), notification); err != nil {
			ns.logger.Error(fmt.Sprintf("Failed to deliver notification %s: %v", notification.ID, err))
		}
	}()

	ns.logger.Info(fmt.Sprintf("Notification created from rule: id=%s, rule_id=%s", notification.ID, rule.ID))
	return nil
}

// renderNotificationContent 渲染通知内容
func (ns *NotificationService) renderNotificationContent(ctx context.Context, notification *models.Notification, template *models.NotificationTemplate) error {
	// 准备模板数据
	templateData, err := ns.prepareTemplateData(notification)
	if err != nil {
		return fmt.Errorf("failed to prepare template data: %w", err)
	}

	// 渲染标题
	if title, err := ns.templateEngine.RenderText(template.SubjectTemplate, templateData); err != nil {
		ns.logger.Error(fmt.Sprintf("Failed to render title template: %v", err))
	} else {
		notification.Title = title
	}

	// 渲染内容
	if content, err := ns.templateEngine.RenderText(template.BodyTemplate, templateData); err != nil {
		ns.logger.Error(fmt.Sprintf("Failed to render content template: %v", err))
	} else {
		notification.Content = content
	}

	return nil
}

// prepareTemplateData 准备模板数据
func (ns *NotificationService) prepareTemplateData(notification *models.Notification) (map[string]interface{}, error) {
	data := map[string]interface{}{
		"notification": map[string]interface{}{
			"id":         notification.ID,
			"type":       notification.Type,
			"category":   notification.Category,
			"priority":   notification.Priority,
			"created_at": notification.CreatedAt,
		},
		"tenant_id":  notification.TenantID,
		"user_id":    notification.UserID,
		"project_id": notification.ProjectID,
	}

	// 解析事件数据
	if notification.EventData != nil {
		var eventData map[string]interface{}
		if err := json.Unmarshal(notification.EventData, &eventData); err != nil {
			ns.logger.Warn(fmt.Sprintf("Failed to unmarshal event data: %v", err))
		} else {
			data["event"] = eventData
		}
	}

	// 解析元数据
	if notification.Metadata != nil {
		var metadata map[string]interface{}
		if err := json.Unmarshal(notification.Metadata, &metadata); err != nil {
			ns.logger.Warn(fmt.Sprintf("Failed to unmarshal metadata: %v", err))
		} else {
			data["metadata"] = metadata
		}
	}

	// 添加常用变量
	data["now"] = time.Now()
	data["today"] = time.Now().Format("2006-01-02")

	return data, nil
}

// checkRuleConditions 检查规则条件
func (ns *NotificationService) checkRuleConditions(req *CreateNotificationRequest, rule *models.NotificationRule) bool {
	// 检查事件类型是否匹配
	eventTypeMatched := false
	for _, eventType := range rule.EventTypes {
		if eventType == req.Type {
			eventTypeMatched = true
			break
		}
	}

	if !eventTypeMatched {
		return false
	}

	// TODO: 实现更复杂的条件检查逻辑
	// 例如：检查事件数据中的特定字段、用户角色、项目权限等

	return true
}

// checkRateLimit 检查频率限制
func (ns *NotificationService) checkRateLimit(ctx context.Context, req *CreateNotificationRequest, rule *models.NotificationRule) (bool, error) {
	if rule.RateLimit == nil {
		return true, nil
	}

	// 计算时间窗口开始时间
	windowStart := time.Now().Add(-time.Duration(rule.RateLimit.TimeWindow) * time.Minute)

	// 查询时间窗口内的通知数量
	count, err := ns.notificationRepo.CountNotificationsByRule(ctx, rule.ID, windowStart)
	if err != nil {
		return false, err
	}

	// 检查是否超过限制
	if count >= rule.RateLimit.MaxCount {
		switch rule.RateLimit.Strategy {
		case "throttle":
			return false, nil // 丢弃
		case "batch":
			// TODO: 实现批量策略
			return false, nil
		case "skip":
			return false, nil // 跳过
		default:
			return false, nil
		}
	}

	return true, nil
}

// isInQuietHours 检查是否在免打扰时间
func (ns *NotificationService) isInQuietHours(quietHours *models.QuietHours) bool {
	if !quietHours.Enabled {
		return false
	}

	// TODO: 实现时区转换和时间范围检查
	// 这里需要根据用户时区进行转换
	_ = time.Now() // 临时处理未使用变量

	return false
}

// GetNotifications 获取通知列表
func (ns *NotificationService) GetNotifications(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID, options *GetNotificationsOptions) ([]*models.Notification, error) {
	// 转换选项
	repoOptions := &repository.GetNotificationsOptions{
		ProjectID: options.ProjectID,
		Category:  options.Category,
		Status:    options.Status,
		Priority:  options.Priority,
		Limit:     options.Limit,
		Offset:    options.Offset,
		SortBy:    options.SortBy,
		SortOrder: options.SortOrder,
	}
	return ns.notificationRepo.GetByUser(ctx, userID, tenantID, repoOptions)
}

// GetNotificationsOptions 获取通知选项
type GetNotificationsOptions struct {
	ProjectID *uuid.UUID `json:"project_id,omitempty"`
	Category  string     `json:"category,omitempty"`
	Status    string     `json:"status,omitempty"`
	Priority  string     `json:"priority,omitempty"`
	Limit     int        `json:"limit,omitempty"`
	Offset    int        `json:"offset,omitempty"`
	SortBy    string     `json:"sort_by,omitempty"`    // created_at, priority
	SortOrder string     `json:"sort_order,omitempty"` // asc, desc
}

// MarkAsRead 标记通知为已读
func (ns *NotificationService) MarkAsRead(ctx context.Context, notificationID uuid.UUID, userID uuid.UUID) error {
	notification, err := ns.notificationRepo.GetByID(ctx, notificationID)
	if err != nil {
		return fmt.Errorf("failed to get notification: %w", err)
	}

	// 验证权限
	if notification.UserID == nil || *notification.UserID != userID {
		return fmt.Errorf("permission denied")
	}

	// 更新状态
	return ns.notificationRepo.UpdateStatus(ctx, notificationID, "read")
}

// GetUnreadCount 获取未读通知数量
func (ns *NotificationService) GetUnreadCount(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) (int, error) {
	return ns.notificationRepo.GetUnreadCount(ctx, userID, tenantID)
}

// DeleteNotification 删除通知
func (ns *NotificationService) DeleteNotification(ctx context.Context, notificationID uuid.UUID, userID uuid.UUID) error {
	notification, err := ns.notificationRepo.GetByID(ctx, notificationID)
	if err != nil {
		return fmt.Errorf("failed to get notification: %w", err)
	}

	// 验证权限
	if notification.UserID == nil || *notification.UserID != userID {
		return fmt.Errorf("permission denied")
	}

	return ns.notificationRepo.SoftDelete(ctx, notificationID)
}

// RetryFailedNotification 重试失败的通知
func (ns *NotificationService) RetryFailedNotification(ctx context.Context, notificationID uuid.UUID) error {
	notification, err := ns.notificationRepo.GetByID(ctx, notificationID)
	if err != nil {
		return fmt.Errorf("failed to get notification: %w", err)
	}

	if notification.Status != models.StatusFailed {
		return fmt.Errorf("notification is not in failed status")
	}

	if notification.RetryCount >= notification.MaxRetries {
		return fmt.Errorf("maximum retry count exceeded")
	}

	// 重置状态
	notification.Status = models.StatusPending
	notification.RetryCount++

	if err := ns.notificationRepo.Update(ctx, notification); err != nil {
		return fmt.Errorf("failed to update notification: %w", err)
	}

	// 重新发送
	go func() {
		if err := ns.deliveryService.DeliverNotification(context.Background(), notification); err != nil {
			ns.logger.Error(fmt.Sprintf("Failed to retry notification %s: %v", notification.ID, err))
		}
	}()

	return nil
}

// GetNotificationsByCorrelationID 根据关联ID获取通知
func (ns *NotificationService) GetNotificationsByCorrelationID(ctx context.Context, correlationID string, tenantID uuid.UUID) ([]*models.Notification, error) {
	return ns.notificationRepo.GetByCorrelationID(ctx, correlationID, tenantID)
}
