package services

import (
	"context"
	"encoding/json"
	"fmt"
)

// MockEmailService Mock邮件服务
type MockEmailService struct{}

// SendEmail 发送邮件（Mock实现）
func (mes *MockEmailService) SendEmail(ctx context.Context, request *EmailRequest) error {
	// Mock实现：只是记录日志，不实际发送邮件
	fmt.Printf("Mock: Sending email to %v, subject: %s\n", request.To, request.Subject)
	return nil
}

// MockWebhookService Mock Webhook服务
type MockWebhookService struct{}

// SendWebhook 发送Webhook（Mock实现）
func (mws *MockWebhookService) SendWebhook(ctx context.Context, request *WebhookRequest) ([]byte, error) {
	// Mock实现：返回模拟响应
	fmt.Printf("Mock: Sending webhook to %s, method: %s\n", request.URL, request.Method)

	response := map[string]interface{}{
		"status":  "success",
		"message": "Webhook received",
		"data":    request.Notification,
	}

	return json.Marshal(response)
}

// MockInAppService Mock应用内通知服务
type MockInAppService struct{}

// SendInApp 发送应用内通知（Mock实现）
func (mias *MockInAppService) SendInApp(ctx context.Context, request *InAppRequest) error {
	// Mock实现：只是记录日志
	fmt.Printf("Mock: Sending in-app notification to user %v, title: %s\n",
		request.UserID, request.Notification.Title)
	return nil
}
