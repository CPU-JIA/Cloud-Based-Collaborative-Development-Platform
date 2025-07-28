package notification

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// NotificationType 通知类型
type NotificationType string

const (
	NotificationTypeEmail    NotificationType = "email"
	NotificationTypeSlack    NotificationType = "slack"
	NotificationTypeWebhook  NotificationType = "webhook"
	NotificationTypeDingTalk NotificationType = "dingtalk"
	NotificationTypeWeChat   NotificationType = "wechat"
)

// Notification 通知内容
type Notification struct {
	Type       NotificationType       `json:"type"`
	Recipients []string               `json:"recipients"`
	Subject    string                 `json:"subject"`
	Body       string                 `json:"body"`
	HTML       bool                   `json:"html"`
	Priority   string                 `json:"priority"` // low, normal, high
	Metadata   map[string]interface{} `json:"metadata"`
	Timestamp  time.Time              `json:"timestamp"`
}

// NotificationResult 通知结果
type NotificationResult struct {
	Success   bool      `json:"success"`
	MessageID string    `json:"message_id,omitempty"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Notifier 通知器接口
type Notifier interface {
	// Send 发送通知
	Send(ctx context.Context, notification *Notification) (*NotificationResult, error)
	// SendBatch 批量发送通知
	SendBatch(ctx context.Context, notifications []*Notification) ([]*NotificationResult, error)
	// GetType 获取通知器类型
	GetType() NotificationType
	// IsAvailable 检查通知器是否可用
	IsAvailable() bool
}

// NotificationManager 通知管理器
type NotificationManager struct {
	notifiers map[NotificationType]Notifier
	logger    *zap.Logger
}

// NewNotificationManager 创建通知管理器
func NewNotificationManager(logger *zap.Logger) *NotificationManager {
	return &NotificationManager{
		notifiers: make(map[NotificationType]Notifier),
		logger:    logger,
	}
}

// RegisterNotifier 注册通知器
func (m *NotificationManager) RegisterNotifier(notifier Notifier) error {
	if notifier == nil {
		return fmt.Errorf("notifier cannot be nil")
	}

	notifierType := notifier.GetType()
	if _, exists := m.notifiers[notifierType]; exists {
		return fmt.Errorf("notifier type %s already registered", notifierType)
	}

	m.notifiers[notifierType] = notifier
	m.logger.Info("Notifier registered", zap.String("type", string(notifierType)))

	return nil
}

// Send 发送通知
func (m *NotificationManager) Send(ctx context.Context, notification *Notification) (*NotificationResult, error) {
	notifier, exists := m.notifiers[notification.Type]
	if !exists {
		return nil, fmt.Errorf("notifier type %s not found", notification.Type)
	}

	if !notifier.IsAvailable() {
		return nil, fmt.Errorf("notifier type %s is not available", notification.Type)
	}

	m.logger.Debug("Sending notification",
		zap.String("type", string(notification.Type)),
		zap.String("subject", notification.Subject),
		zap.Int("recipients", len(notification.Recipients)))

	result, err := notifier.Send(ctx, notification)
	if err != nil {
		m.logger.Error("Failed to send notification",
			zap.String("type", string(notification.Type)),
			zap.Error(err))
		return nil, err
	}

	return result, nil
}

// SendMultiple 使用多种方式发送通知
func (m *NotificationManager) SendMultiple(ctx context.Context, types []NotificationType, baseNotification *Notification) map[NotificationType]*NotificationResult {
	results := make(map[NotificationType]*NotificationResult)

	for _, notifyType := range types {
		// 复制通知并设置类型
		notification := *baseNotification
		notification.Type = notifyType

		result, err := m.Send(ctx, &notification)
		if err != nil {
			results[notifyType] = &NotificationResult{
				Success:   false,
				Error:     err.Error(),
				Timestamp: time.Now(),
			}
		} else {
			results[notifyType] = result
		}
	}

	return results
}

// GetAvailableNotifiers 获取可用的通知器列表
func (m *NotificationManager) GetAvailableNotifiers() []NotificationType {
	var available []NotificationType

	for notifyType, notifier := range m.notifiers {
		if notifier.IsAvailable() {
			available = append(available, notifyType)
		}
	}

	return available
}
