package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// SlackConfig Slack配置
type SlackConfig struct {
	WebhookURL string        `json:"webhook_url"`
	Channel    string        `json:"channel"`
	Username   string        `json:"username"`
	IconEmoji  string        `json:"icon_emoji"`
	IconURL    string        `json:"icon_url"`
	MaxRetries int           `json:"max_retries"`
	RetryDelay time.Duration `json:"retry_delay"`
}

// SlackNotifier Slack通知器
type SlackNotifier struct {
	config     *SlackConfig
	logger     *zap.Logger
	httpClient *http.Client
}

// SlackMessage Slack消息结构
type SlackMessage struct {
	Channel     string            `json:"channel,omitempty"`
	Username    string            `json:"username,omitempty"`
	Text        string            `json:"text"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	IconURL     string            `json:"icon_url,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
	Blocks      []SlackBlock      `json:"blocks,omitempty"`
}

// SlackAttachment Slack附件
type SlackAttachment struct {
	Fallback   string        `json:"fallback"`
	Color      string        `json:"color"`
	Title      string        `json:"title"`
	TitleLink  string        `json:"title_link,omitempty"`
	Text       string        `json:"text"`
	Fields     []SlackField  `json:"fields,omitempty"`
	Footer     string        `json:"footer,omitempty"`
	FooterIcon string        `json:"footer_icon,omitempty"`
	Timestamp  int64         `json:"ts,omitempty"`
	MarkdownIn []string      `json:"mrkdwn_in,omitempty"`
	Actions    []SlackAction `json:"actions,omitempty"`
}

// SlackField Slack字段
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// SlackAction Slack动作
type SlackAction struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	URL   string `json:"url"`
	Style string `json:"style,omitempty"`
}

// SlackBlock Slack块元素
type SlackBlock struct {
	Type string     `json:"type"`
	Text *SlackText `json:"text,omitempty"`
}

// SlackText Slack文本
type SlackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// NewSlackNotifier 创建Slack通知器
func NewSlackNotifier(config *SlackConfig, logger *zap.Logger) *SlackNotifier {
	return &SlackNotifier{
		config: config,
		logger: logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetType 获取通知器类型
func (s *SlackNotifier) GetType() NotificationType {
	return NotificationTypeSlack
}

// IsAvailable 检查通知器是否可用
func (s *SlackNotifier) IsAvailable() bool {
	return s.config.WebhookURL != ""
}

// Send 发送Slack通知
func (s *SlackNotifier) Send(ctx context.Context, notification *Notification) (*NotificationResult, error) {
	// 构建Slack消息
	message := s.buildSlackMessage(notification)

	// 发送消息（带重试）
	var lastErr error
	maxRetries := s.config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(s.config.RetryDelay):
			}
		}

		err := s.sendToSlack(ctx, message)
		if err == nil {
			messageID := uuid.New().String()
			return &NotificationResult{
				Success:   true,
				MessageID: messageID,
				Timestamp: time.Now(),
			}, nil
		}

		lastErr = err
		s.logger.Warn("Failed to send Slack message, retrying",
			zap.Int("attempt", attempt+1),
			zap.Error(err))
	}

	return &NotificationResult{
		Success:   false,
		Error:     lastErr.Error(),
		Timestamp: time.Now(),
	}, lastErr
}

// SendBatch 批量发送Slack通知
func (s *SlackNotifier) SendBatch(ctx context.Context, notifications []*Notification) ([]*NotificationResult, error) {
	results := make([]*NotificationResult, len(notifications))

	for i, notification := range notifications {
		result, err := s.Send(ctx, notification)
		if err != nil {
			s.logger.Error("Failed to send batch Slack message",
				zap.Int("index", i),
				zap.Error(err))
		}
		results[i] = result
	}

	return results, nil
}

// buildSlackMessage 构建Slack消息
func (s *SlackNotifier) buildSlackMessage(notification *Notification) *SlackMessage {
	message := &SlackMessage{
		Text: notification.Subject,
	}

	// 设置频道
	if s.config.Channel != "" {
		message.Channel = s.config.Channel
	}

	// 设置用户名
	if s.config.Username != "" {
		message.Username = s.config.Username
	}

	// 设置图标
	if s.config.IconEmoji != "" {
		message.IconEmoji = s.config.IconEmoji
	} else if s.config.IconURL != "" {
		message.IconURL = s.config.IconURL
	}

	// 创建附件
	attachment := SlackAttachment{
		Fallback:   notification.Body,
		Text:       notification.Body,
		Timestamp:  notification.Timestamp.Unix(),
		MarkdownIn: []string{"text", "pretext"},
	}

	// 根据优先级设置颜色
	switch notification.Priority {
	case "high":
		attachment.Color = "danger"
	case "low":
		attachment.Color = "good"
	default:
		attachment.Color = "warning"
	}

	// 添加元数据字段
	if len(notification.Metadata) > 0 {
		attachment.Fields = make([]SlackField, 0, len(notification.Metadata))
		for key, value := range notification.Metadata {
			attachment.Fields = append(attachment.Fields, SlackField{
				Title: key,
				Value: fmt.Sprintf("%v", value),
				Short: true,
			})
		}
	}

	// 添加页脚
	attachment.Footer = "Cloud Platform Notification"
	attachment.FooterIcon = "https://platform.slack-edge.com/img/default_application_icon.png"

	message.Attachments = []SlackAttachment{attachment}

	return message
}

// sendToSlack 发送消息到Slack
func (s *SlackNotifier) sendToSlack(ctx context.Context, message *SlackMessage) error {
	// 序列化消息
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", s.config.WebhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned status %d: %s", resp.StatusCode, string(body))
	}

	// Slack特定的错误检查
	if string(body) != "ok" {
		return fmt.Errorf("slack returned error: %s", string(body))
	}

	return nil
}
