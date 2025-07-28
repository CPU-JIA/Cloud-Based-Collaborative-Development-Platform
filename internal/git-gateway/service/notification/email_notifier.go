package notification

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// EmailConfig 邮件配置
type EmailConfig struct {
	SMTPHost    string        `json:"smtp_host"`
	SMTPPort    int           `json:"smtp_port"`
	Username    string        `json:"username"`
	Password    string        `json:"password"`
	FromAddress string        `json:"from_address"`
	FromName    string        `json:"from_name"`
	UseTLS      bool          `json:"use_tls"`
	SkipVerify  bool          `json:"skip_verify"`
	MaxRetries  int           `json:"max_retries"`
	RetryDelay  time.Duration `json:"retry_delay"`
}

// EmailNotifier 邮件通知器
type EmailNotifier struct {
	config *EmailConfig
	logger *zap.Logger
	auth   smtp.Auth
}

// NewEmailNotifier 创建邮件通知器
func NewEmailNotifier(config *EmailConfig, logger *zap.Logger) *EmailNotifier {
	var auth smtp.Auth
	if config.Username != "" && config.Password != "" {
		auth = smtp.PlainAuth("", config.Username, config.Password, config.SMTPHost)
	}

	return &EmailNotifier{
		config: config,
		logger: logger,
		auth:   auth,
	}
}

// GetType 获取通知器类型
func (e *EmailNotifier) GetType() NotificationType {
	return NotificationTypeEmail
}

// IsAvailable 检查通知器是否可用
func (e *EmailNotifier) IsAvailable() bool {
	return e.config.SMTPHost != "" && e.config.SMTPPort > 0
}

// Send 发送邮件通知
func (e *EmailNotifier) Send(ctx context.Context, notification *Notification) (*NotificationResult, error) {
	if len(notification.Recipients) == 0 {
		return nil, fmt.Errorf("no recipients specified")
	}

	// 构建邮件内容
	message := e.buildMessage(notification)

	// 发送邮件（带重试）
	var lastErr error
	maxRetries := e.config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(e.config.RetryDelay):
			}
		}

		err := e.sendEmail(notification.Recipients, message)
		if err == nil {
			messageID := uuid.New().String()
			return &NotificationResult{
				Success:   true,
				MessageID: messageID,
				Timestamp: time.Now(),
			}, nil
		}

		lastErr = err
		e.logger.Warn("Failed to send email, retrying",
			zap.Int("attempt", attempt+1),
			zap.Error(err))
	}

	return &NotificationResult{
		Success:   false,
		Error:     lastErr.Error(),
		Timestamp: time.Now(),
	}, lastErr
}

// SendBatch 批量发送邮件
func (e *EmailNotifier) SendBatch(ctx context.Context, notifications []*Notification) ([]*NotificationResult, error) {
	results := make([]*NotificationResult, len(notifications))

	for i, notification := range notifications {
		result, err := e.Send(ctx, notification)
		if err != nil {
			e.logger.Error("Failed to send batch email",
				zap.Int("index", i),
				zap.Error(err))
		}
		results[i] = result
	}

	return results, nil
}

// buildMessage 构建邮件消息
func (e *EmailNotifier) buildMessage(notification *Notification) string {
	from := e.config.FromAddress
	if e.config.FromName != "" {
		from = fmt.Sprintf("%s <%s>", e.config.FromName, e.config.FromAddress)
	}

	headers := []string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", strings.Join(notification.Recipients, ", ")),
		fmt.Sprintf("Subject: %s", notification.Subject),
		fmt.Sprintf("Date: %s", time.Now().Format(time.RFC1123Z)),
		fmt.Sprintf("Message-ID: <%s@%s>", uuid.New().String(), e.config.SMTPHost),
	}

	// 设置内容类型
	if notification.HTML {
		headers = append(headers, "Content-Type: text/html; charset=UTF-8")
	} else {
		headers = append(headers, "Content-Type: text/plain; charset=UTF-8")
	}

	// 设置优先级
	switch notification.Priority {
	case "high":
		headers = append(headers, "X-Priority: 1", "Importance: High")
	case "low":
		headers = append(headers, "X-Priority: 5", "Importance: Low")
	default:
		headers = append(headers, "X-Priority: 3", "Importance: Normal")
	}

	// 组装消息
	message := strings.Join(headers, "\r\n") + "\r\n\r\n" + notification.Body

	return message
}

// sendEmail 实际发送邮件
func (e *EmailNotifier) sendEmail(recipients []string, message string) error {
	addr := fmt.Sprintf("%s:%d", e.config.SMTPHost, e.config.SMTPPort)

	if e.config.UseTLS {
		// 使用TLS发送
		tlsConfig := &tls.Config{
			ServerName:         e.config.SMTPHost,
			InsecureSkipVerify: e.config.SkipVerify,
		}

		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, e.config.SMTPHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer client.Close()

		// 认证
		if e.auth != nil {
			if err := client.Auth(e.auth); err != nil {
				return fmt.Errorf("SMTP authentication failed: %w", err)
			}
		}

		// 发送邮件
		if err := e.sendViaClient(client, recipients, message); err != nil {
			return err
		}
	} else {
		// 使用标准SMTP发送
		if err := smtp.SendMail(addr, e.auth, e.config.FromAddress, recipients, []byte(message)); err != nil {
			return fmt.Errorf("failed to send email: %w", err)
		}
	}

	return nil
}

// sendViaClient 通过SMTP客户端发送邮件
func (e *EmailNotifier) sendViaClient(client *smtp.Client, recipients []string, message string) error {
	// 设置发件人
	if err := client.Mail(e.config.FromAddress); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// 设置收件人
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// 发送数据
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	_, err = writer.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return nil
}
