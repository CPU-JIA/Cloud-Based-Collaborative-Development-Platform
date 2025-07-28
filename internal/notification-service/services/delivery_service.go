package services

import (
	"context"
	"fmt"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/notification-service/models"
	"github.com/cloud-platform/collaborative-dev/internal/notification-service/repository"
	"github.com/cloud-platform/collaborative-dev/shared/logger"
	"github.com/google/uuid"
)

// DeliveryService 通知发送服务
type DeliveryService struct {
	notificationRepo *repository.NotificationRepository
	deliveryLogRepo  *repository.DeliveryLogRepository
	emailService     EmailService
	webhookService   WebhookService
	inAppService     InAppService
	logger           logger.Logger
}

// NewDeliveryService 创建新的发送服务
func NewDeliveryService(
	notificationRepo *repository.NotificationRepository,
	deliveryLogRepo *repository.DeliveryLogRepository,
	emailService EmailService,
	webhookService WebhookService,
	inAppService InAppService,
	appLogger logger.Logger,
) *DeliveryService {
	return &DeliveryService{
		notificationRepo: notificationRepo,
		deliveryLogRepo:  deliveryLogRepo,
		emailService:     emailService,
		webhookService:   webhookService,
		inAppService:     inAppService,
		logger:           appLogger,
	}
}

// DeliverNotification 发送通知
func (ds *DeliveryService) DeliverNotification(ctx context.Context, notification *models.Notification) error {
	ds.logger.Info(fmt.Sprintf("Delivering notification: id=%s, type=%s", notification.ID, notification.Type))

	// 更新通知状态为处理中
	if err := ds.updateNotificationStatus(ctx, notification.ID, models.StatusProcessing); err != nil {
		return fmt.Errorf("failed to update notification status: %w", err)
	}

	// 记录发送开始时间
	startTime := time.Now()
	success := true
	var errors []error

	// 发送到各个渠道
	if notification.Channels.Email != nil && notification.Channels.Email.Enabled {
		if err := ds.deliverViaEmail(ctx, notification); err != nil {
			ds.logger.Error(fmt.Sprintf("Email delivery failed: %v", err))
			errors = append(errors, err)
			success = false
		}
	}

	if notification.Channels.Webhook != nil && notification.Channels.Webhook.Enabled {
		if err := ds.deliverViaWebhook(ctx, notification); err != nil {
			ds.logger.Error(fmt.Sprintf("Webhook delivery failed: %v", err))
			errors = append(errors, err)
			success = false
		}
	}

	if notification.Channels.InApp != nil && notification.Channels.InApp.Enabled {
		if err := ds.deliverViaInApp(ctx, notification); err != nil {
			ds.logger.Error(fmt.Sprintf("In-app delivery failed: %v", err))
			errors = append(errors, err)
			success = false
		}
	}

	// TODO: Push通知发送
	if notification.Channels.Push != nil && notification.Channels.Push.Enabled {
		ds.logger.Info("Push notification delivery not implemented yet")
	}

	// 更新通知状态
	if success {
		if err := ds.updateNotificationStatus(ctx, notification.ID, models.StatusSent); err != nil {
			return fmt.Errorf("failed to update notification status to sent: %w", err)
		}

		// 更新发送时间
		now := time.Now()
		notification.SentAt = &now
		if err := ds.notificationRepo.Update(ctx, notification); err != nil {
			ds.logger.Error(fmt.Sprintf("Failed to update sent timestamp: %v", err))
		}

		ds.logger.Info(fmt.Sprintf("Notification delivered successfully: id=%s, duration=%v",
			notification.ID, time.Since(startTime)))
	} else {
		if err := ds.updateNotificationStatus(ctx, notification.ID, models.StatusFailed); err != nil {
			return fmt.Errorf("failed to update notification status to failed: %w", err)
		}

		// 更新失败时间
		now := time.Now()
		notification.FailedAt = &now
		if err := ds.notificationRepo.Update(ctx, notification); err != nil {
			ds.logger.Error(fmt.Sprintf("Failed to update failed timestamp: %v", err))
		}

		return fmt.Errorf("notification delivery failed with %d errors", len(errors))
	}

	return nil
}

// deliverViaEmail 通过邮件发送
func (ds *DeliveryService) deliverViaEmail(ctx context.Context, notification *models.Notification) error {
	emailChannel := notification.Channels.Email
	if emailChannel == nil || !emailChannel.Enabled {
		return nil
	}

	// 创建发送日志
	deliveryLog := &models.DeliveryLog{
		NotificationID: notification.ID,
		Channel:        models.ChannelEmail,
		Status:         models.DeliveryStatusSending,
		AttemptCount:   1,
	}

	// 发送给每个收件人
	for _, recipient := range emailChannel.To {
		deliveryLog.Recipient = recipient

		// 保存发送日志
		if err := ds.deliveryLogRepo.Create(ctx, deliveryLog); err != nil {
			ds.logger.Error(fmt.Sprintf("Failed to create delivery log: %v", err))
		}

		// 发送邮件
		emailRequest := &EmailRequest{
			To:      []string{recipient},
			CC:      emailChannel.CC,
			BCC:     emailChannel.BCC,
			Subject: emailChannel.Subject,
			Body:    notification.Content,
			IsHTML:  false,
		}

		// 如果有HTML模板，使用HTML格式
		if notification.Template != nil && notification.Template.HTMLTemplate != "" {
			// TODO: 渲染HTML模板
			emailRequest.IsHTML = true
		}

		if err := ds.emailService.SendEmail(ctx, emailRequest); err != nil {
			// 更新发送日志状态为失败
			now := time.Now()
			deliveryLog.Status = models.DeliveryStatusFailed
			deliveryLog.FailedAt = &now
			deliveryLog.ErrorMessage = err.Error()

			if updateErr := ds.deliveryLogRepo.Update(ctx, deliveryLog); updateErr != nil {
				ds.logger.Error(fmt.Sprintf("Failed to update delivery log: %v", updateErr))
			}

			return fmt.Errorf("failed to send email to %s: %w", recipient, err)
		}

		// 更新发送日志状态为已发送
		now := time.Now()
		deliveryLog.Status = models.DeliveryStatusSent
		deliveryLog.SentAt = &now

		if err := ds.deliveryLogRepo.Update(ctx, deliveryLog); err != nil {
			ds.logger.Error(fmt.Sprintf("Failed to update delivery log: %v", err))
		}
	}

	return nil
}

// deliverViaWebhook 通过Webhook发送
func (ds *DeliveryService) deliverViaWebhook(ctx context.Context, notification *models.Notification) error {
	webhookChannel := notification.Channels.Webhook
	if webhookChannel == nil || !webhookChannel.Enabled {
		return nil
	}

	// 创建发送日志
	deliveryLog := &models.DeliveryLog{
		NotificationID: notification.ID,
		Channel:        models.ChannelWebhook,
		Recipient:      webhookChannel.URL,
		Status:         models.DeliveryStatusSending,
		AttemptCount:   1,
	}

	// 保存发送日志
	if err := ds.deliveryLogRepo.Create(ctx, deliveryLog); err != nil {
		ds.logger.Error(fmt.Sprintf("Failed to create delivery log: %v", err))
	}

	// 构造Webhook请求
	webhookRequest := &WebhookRequest{
		URL:     webhookChannel.URL,
		Method:  webhookChannel.Method,
		Headers: webhookChannel.Headers,
		Payload: webhookChannel.Payload,
		Notification: map[string]interface{}{
			"id":         notification.ID,
			"type":       notification.Type,
			"category":   notification.Category,
			"priority":   notification.Priority,
			"title":      notification.Title,
			"content":    notification.Content,
			"created_at": notification.CreatedAt,
		},
	}

	// 发送Webhook
	response, err := ds.webhookService.SendWebhook(ctx, webhookRequest)
	if err != nil {
		// 更新发送日志状态为失败
		now := time.Now()
		deliveryLog.Status = models.DeliveryStatusFailed
		deliveryLog.FailedAt = &now
		deliveryLog.ErrorMessage = err.Error()

		if updateErr := ds.deliveryLogRepo.Update(ctx, deliveryLog); updateErr != nil {
			ds.logger.Error(fmt.Sprintf("Failed to update delivery log: %v", updateErr))
		}

		return fmt.Errorf("failed to send webhook: %w", err)
	}

	// 更新发送日志状态为已发送
	now := time.Now()
	deliveryLog.Status = models.DeliveryStatusSent
	deliveryLog.SentAt = &now
	deliveryLog.Response = response

	if err := ds.deliveryLogRepo.Update(ctx, deliveryLog); err != nil {
		ds.logger.Error(fmt.Sprintf("Failed to update delivery log: %v", err))
	}

	return nil
}

// deliverViaInApp 通过应用内通知发送
func (ds *DeliveryService) deliverViaInApp(ctx context.Context, notification *models.Notification) error {
	inAppChannel := notification.Channels.InApp
	if inAppChannel == nil || !inAppChannel.Enabled {
		return nil
	}

	// 创建发送日志
	deliveryLog := &models.DeliveryLog{
		NotificationID: notification.ID,
		Channel:        models.ChannelInApp,
		Status:         models.DeliveryStatusSending,
		AttemptCount:   1,
	}

	if notification.UserID != nil {
		deliveryLog.Recipient = notification.UserID.String()
	}

	// 保存发送日志
	if err := ds.deliveryLogRepo.Create(ctx, deliveryLog); err != nil {
		ds.logger.Error(fmt.Sprintf("Failed to create delivery log: %v", err))
	}

	// 构造应用内通知请求
	inAppRequest := &InAppRequest{
		UserID:       notification.UserID,
		Notification: notification,
		ShowBadge:    inAppChannel.Badge,
		ShowPopup:    inAppChannel.Popup,
		PlaySound:    inAppChannel.Sound,
	}

	// 发送应用内通知
	if err := ds.inAppService.SendInApp(ctx, inAppRequest); err != nil {
		// 更新发送日志状态为失败
		now := time.Now()
		deliveryLog.Status = models.DeliveryStatusFailed
		deliveryLog.FailedAt = &now
		deliveryLog.ErrorMessage = err.Error()

		if updateErr := ds.deliveryLogRepo.Update(ctx, deliveryLog); updateErr != nil {
			ds.logger.Error(fmt.Sprintf("Failed to update delivery log: %v", updateErr))
		}

		return fmt.Errorf("failed to send in-app notification: %w", err)
	}

	// 更新发送日志状态为已发送
	now := time.Now()
	deliveryLog.Status = models.DeliveryStatusSent
	deliveryLog.SentAt = &now

	if err := ds.deliveryLogRepo.Update(ctx, deliveryLog); err != nil {
		ds.logger.Error(fmt.Sprintf("Failed to update delivery log: %v", err))
	}

	return nil
}

// updateNotificationStatus 更新通知状态
func (ds *DeliveryService) updateNotificationStatus(ctx context.Context, notificationID uuid.UUID, status string) error {
	return ds.notificationRepo.UpdateStatus(ctx, notificationID, status)
}

// 接口定义

// EmailService 邮件服务接口
type EmailService interface {
	SendEmail(ctx context.Context, request *EmailRequest) error
}

// EmailRequest 邮件请求
type EmailRequest struct {
	To      []string `json:"to"`
	CC      []string `json:"cc,omitempty"`
	BCC     []string `json:"bcc,omitempty"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	IsHTML  bool     `json:"is_html"`
}

// WebhookService Webhook服务接口
type WebhookService interface {
	SendWebhook(ctx context.Context, request *WebhookRequest) ([]byte, error)
}

// WebhookRequest Webhook请求
type WebhookRequest struct {
	URL          string                 `json:"url"`
	Method       string                 `json:"method"`
	Headers      map[string]string      `json:"headers,omitempty"`
	Payload      []byte                 `json:"payload,omitempty"`
	Notification map[string]interface{} `json:"notification"`
}

// InAppService 应用内通知服务接口
type InAppService interface {
	SendInApp(ctx context.Context, request *InAppRequest) error
}

// InAppRequest 应用内通知请求
type InAppRequest struct {
	UserID       *uuid.UUID           `json:"user_id"`
	Notification *models.Notification `json:"notification"`
	ShowBadge    bool                 `json:"show_badge"`
	ShowPopup    bool                 `json:"show_popup"`
	PlaySound    bool                 `json:"play_sound"`
}
