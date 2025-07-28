package unit

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SimpleNotification 简化的通知模型用于测试
type SimpleNotification struct {
	ID        uuid.UUID              `json:"id"`
	TenantID  uuid.UUID              `json:"tenant_id"`
	UserID    uuid.UUID              `json:"user_id"`
	Type      string                 `json:"type"`
	Channel   string                 `json:"channel"`
	Title     string                 `json:"title"`
	Content   string                 `json:"content"`
	Data      map[string]interface{} `json:"data"`
	Status    string                 `json:"status"`
	Priority  string                 `json:"priority"`
	CreatedAt time.Time              `json:"created_at"`
	SentAt    *time.Time             `json:"sent_at"`
	ExpiresAt *time.Time             `json:"expires_at"`
}

// SimpleEmailNotification 简化的邮件通知模型用于测试
type SimpleEmailNotification struct {
	ID          uuid.UUID  `json:"id"`
	ToEmail     string     `json:"to_email"`
	FromEmail   string     `json:"from_email"`
	Subject     string     `json:"subject"`
	Body        string     `json:"body"`
	IsHTML      bool       `json:"is_html"`
	Priority    string     `json:"priority"`
	Status      string     `json:"status"`
	Attachments []string   `json:"attachments"`
	CreatedAt   time.Time  `json:"created_at"`
	SentAt      *time.Time `json:"sent_at"`
}

// SimpleSMSNotification 简化的短信通知模型用于测试
type SimpleSMSNotification struct {
	ID        uuid.UUID  `json:"id"`
	ToPhone   string     `json:"to_phone"`
	Content   string     `json:"content"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	SentAt    *time.Time `json:"sent_at"`
}

// SimpleWebhookNotification 简化的Webhook通知模型用于测试
type SimpleWebhookNotification struct {
	ID        uuid.UUID              `json:"id"`
	URL       string                 `json:"url"`
	Method    string                 `json:"method"`
	Headers   map[string]string      `json:"headers"`
	Payload   map[string]interface{} `json:"payload"`
	Status    string                 `json:"status"`
	Retries   int                    `json:"retries"`
	CreatedAt time.Time              `json:"created_at"`
	SentAt    *time.Time             `json:"sent_at"`
}

// SimpleNotificationTemplate 简化的通知模板模型用于测试
type SimpleNotificationTemplate struct {
	ID       uuid.UUID `json:"id"`
	TenantID uuid.UUID `json:"tenant_id"`
	Name     string    `json:"name"`
	Type     string    `json:"type"`
	Channel  string    `json:"channel"`
	Subject  string    `json:"subject"`
	Body     string    `json:"body"`
	IsActive bool      `json:"is_active"`
}

// SimpleCreateNotificationRequest 简化的创建通知请求用于测试
type SimpleCreateNotificationRequest struct {
	TenantID uuid.UUID              `json:"tenant_id"`
	UserID   uuid.UUID              `json:"user_id"`
	Type     string                 `json:"type"`
	Channel  string                 `json:"channel"`
	Title    string                 `json:"title"`
	Content  string                 `json:"content"`
	Data     map[string]interface{} `json:"data"`
	Priority string                 `json:"priority"`
}

// Notification Service 验证函数

// validateNotificationType 验证通知类型
func validateNotificationType(notificationType string) error {
	if notificationType == "" {
		return fmt.Errorf("通知类型不能为空")
	}

	validTypes := []string{
		"system",
		"user_action",
		"security_alert",
		"project_update",
		"task_assignment",
		"comment_mention",
		"approval_request",
		"deadline_reminder",
		"system_maintenance",
		"billing_notice",
	}

	for _, validType := range validTypes {
		if notificationType == validType {
			return nil
		}
	}

	return fmt.Errorf("无效的通知类型: %s", notificationType)
}

// validateNotificationChannel 验证通知渠道
func validateNotificationChannel(channel string) error {
	if channel == "" {
		return fmt.Errorf("通知渠道不能为空")
	}

	validChannels := []string{"email", "sms", "push", "webhook", "in_app"}
	for _, validChannel := range validChannels {
		if channel == validChannel {
			return nil
		}
	}

	return fmt.Errorf("无效的通知渠道: %s", channel)
}

// validateNotificationPriority 验证通知优先级
func validateNotificationPriority(priority string) error {
	if priority == "" {
		return fmt.Errorf("通知优先级不能为空")
	}

	validPriorities := []string{"low", "normal", "high", "urgent"}
	for _, validPriority := range validPriorities {
		if priority == validPriority {
			return nil
		}
	}

	return fmt.Errorf("无效的通知优先级: %s", priority)
}

// validateEmailAddress 验证邮箱地址
func validateEmailAddress(email string) error {
	if email == "" {
		return fmt.Errorf("邮箱地址不能为空")
	}

	_, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("邮箱地址格式不正确")
	}

	return nil
}

// validatePhoneNumber 验证电话号码
func validatePhoneNumber(phone string) error {
	if phone == "" {
		return fmt.Errorf("电话号码不能为空")
	}

	// 基础电话号码格式验证（支持国际格式）
	// 允许可选的+号开头，然后是1-9开头的数字，总长度2-15位
	phonePattern := regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	if !phonePattern.MatchString(phone) {
		return fmt.Errorf("电话号码格式不正确")
	}

	// 额外检查：最少4位数字（除了+号）
	cleanPhone := strings.TrimPrefix(phone, "+")
	if len(cleanPhone) < 4 {
		return fmt.Errorf("电话号码格式不正确")
	}

	return nil
}

// validateWebhookURL 验证Webhook URL
func validateWebhookURL(url string) error {
	if url == "" {
		return fmt.Errorf("Webhook URL不能为空")
	}

	// 基础URL格式验证
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("Webhook URL必须以http://或https://开头")
	}

	if len(url) > 2048 {
		return fmt.Errorf("Webhook URL长度不能超过2048字符")
	}

	return nil
}

// validateNotificationContent 验证通知内容
func validateNotificationContent(title, content string) error {
	if title == "" {
		return fmt.Errorf("通知标题不能为空")
	}

	if content == "" {
		return fmt.Errorf("通知内容不能为空")
	}

	if len(title) > 200 {
		return fmt.Errorf("通知标题长度不能超过200字符")
	}

	if len(content) > 10000 {
		return fmt.Errorf("通知内容长度不能超过10000字符")
	}

	return nil
}

// validateTemplateName 验证模板名称
func validateTemplateName(name string) error {
	if name == "" {
		return fmt.Errorf("模板名称不能为空")
	}

	if len(name) < 2 || len(name) > 100 {
		return fmt.Errorf("模板名称长度必须在2-100字符之间")
	}

	// 模板名称只能包含字母、数字、下划线和横线
	namePattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !namePattern.MatchString(name) {
		return fmt.Errorf("模板名称只能包含字母、数字、下划线和横线")
	}

	return nil
}

// validateCreateNotificationRequest 验证创建通知请求
func validateCreateNotificationRequest(req *SimpleCreateNotificationRequest) error {
	if req == nil {
		return fmt.Errorf("创建通知请求不能为nil")
	}

	if req.TenantID == uuid.Nil {
		return fmt.Errorf("租户ID不能为空")
	}

	if req.UserID == uuid.Nil {
		return fmt.Errorf("用户ID不能为空")
	}

	if err := validateNotificationType(req.Type); err != nil {
		return fmt.Errorf("通知类型: %v", err)
	}

	if err := validateNotificationChannel(req.Channel); err != nil {
		return fmt.Errorf("通知渠道: %v", err)
	}

	if err := validateNotificationContent(req.Title, req.Content); err != nil {
		return fmt.Errorf("通知内容: %v", err)
	}

	if err := validateNotificationPriority(req.Priority); err != nil {
		return fmt.Errorf("通知优先级: %v", err)
	}

	return nil
}

// TestNotificationTypeValidation 测试通知类型验证
func TestNotificationTypeValidation(t *testing.T) {
	tests := []struct {
		name          string
		notifyType    string
		expectedError string
	}{
		{"有效的系统通知", "system", ""},
		{"有效的用户操作通知", "user_action", ""},
		{"有效的安全警报", "security_alert", ""},
		{"有效的项目更新", "project_update", ""},
		{"有效的任务分配", "task_assignment", ""},
		{"有效的评论提及", "comment_mention", ""},
		{"有效的审批请求", "approval_request", ""},
		{"有效的截止日期提醒", "deadline_reminder", ""},
		{"有效的系统维护", "system_maintenance", ""},
		{"有效的计费通知", "billing_notice", ""},
		{"空通知类型", "", "通知类型不能为空"},
		{"无效的通知类型", "invalid_type", "无效的通知类型"},
		{"包含特殊字符的类型", "type@invalid", "无效的通知类型"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNotificationType(tt.notifyType)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestNotificationChannelValidation 测试通知渠道验证
func TestNotificationChannelValidation(t *testing.T) {
	tests := []struct {
		name          string
		channel       string
		expectedError string
	}{
		{"有效的邮件渠道", "email", ""},
		{"有效的短信渠道", "sms", ""},
		{"有效的推送渠道", "push", ""},
		{"有效的Webhook渠道", "webhook", ""},
		{"有效的应用内通知", "in_app", ""},
		{"空渠道", "", "通知渠道不能为空"},
		{"无效的渠道", "invalid_channel", "无效的通知渠道"},
		{"大写渠道名", "EMAIL", "无效的通知渠道"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNotificationChannel(tt.channel)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestNotificationPriorityValidation 测试通知优先级验证
func TestNotificationPriorityValidation(t *testing.T) {
	tests := []struct {
		name          string
		priority      string
		expectedError string
	}{
		{"低优先级", "low", ""},
		{"普通优先级", "normal", ""},
		{"高优先级", "high", ""},
		{"紧急优先级", "urgent", ""},
		{"空优先级", "", "通知优先级不能为空"},
		{"无效的优先级", "critical", "无效的通知优先级"},
		{"数字优先级", "1", "无效的通知优先级"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNotificationPriority(tt.priority)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestEmailAddressValidation 测试邮箱地址验证
func TestEmailAddressValidation(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		expectedError string
	}{
		{"有效的普通邮箱", "user@example.com", ""},
		{"有效的企业邮箱", "john.doe@company.com", ""},
		{"有效的带加号邮箱", "user+tag@example.com", ""},
		{"有效的子域名邮箱", "user@mail.example.com", ""},
		{"空邮箱", "", "邮箱地址不能为空"},
		{"无@符号", "userexample.com", "邮箱地址格式不正确"},
		{"多个@符号", "user@@example.com", "邮箱地址格式不正确"},
		{"缺少域名", "user@", "邮箱地址格式不正确"},
		{"缺少用户名", "@example.com", "邮箱地址格式不正确"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEmailAddress(tt.email)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestPhoneNumberValidation 测试电话号码验证
func TestPhoneNumberValidation(t *testing.T) {
	tests := []struct {
		name          string
		phone         string
		expectedError string
	}{
		{"有效的中国手机号", "13812345678", ""},
		{"有效的国际格式", "+8613812345678", ""},
		{"有效的美国号码", "+12345678901", ""},
		{"有效的英国号码", "+441234567890", ""},
		{"空电话号码", "", "电话号码不能为空"},
		{"包含字母", "138abcd5678", "电话号码格式不正确"},
		{"包含特殊字符", "138-1234-5678", "电话号码格式不正确"},
		{"以0开头", "01381234567", "电话号码格式不正确"},
		{"过短", "123", "电话号码格式不正确"},
		{"过长", "123456789012345678", "电话号码格式不正确"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePhoneNumber(tt.phone)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestWebhookURLValidation 测试Webhook URL验证
func TestWebhookURLValidation(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		expectedError string
	}{
		{"有效的HTTP URL", "http://example.com/webhook", ""},
		{"有效的HTTPS URL", "https://api.example.com/notifications", ""},
		{"有效的带端口URL", "https://localhost:8080/webhook", ""},
		{"有效的带路径参数URL", "https://example.com/webhook?token=abc123", ""},
		{"空URL", "", "Webhook URL不能为空"},
		{"无协议URL", "example.com/webhook", "Webhook URL必须以http://或https://开头"},
		{"FTP协议", "ftp://example.com/webhook", "Webhook URL必须以http://或https://开头"},
		{"过长URL", "https://example.com/" + strings.Repeat("a", 2048), "Webhook URL长度不能超过2048字符"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWebhookURL(tt.url)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestNotificationContentValidation 测试通知内容验证
func TestNotificationContentValidation(t *testing.T) {
	tests := []struct {
		name          string
		title         string
		content       string
		expectedError string
	}{
		{"有效的普通内容", "测试标题", "测试内容", ""},
		{"有效的长内容", "项目更新", strings.Repeat("内容", 100), ""},
		{"有效的英文内容", "Test Title", "Test content", ""},
		{"空标题", "", "测试内容", "通知标题不能为空"},
		{"空内容", "测试标题", "", "通知内容不能为空"},
		{"标题过长", strings.Repeat("标题", 101), "内容", "通知标题长度不能超过200字符"},
		{"内容过长", "标题", strings.Repeat("内容", 2501), "通知内容长度不能超过10000字符"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNotificationContent(tt.title, tt.content)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestTemplateNameValidation 测试模板名称验证
func TestTemplateNameValidation(t *testing.T) {
	tests := []struct {
		name          string
		templateName  string
		expectedError string
	}{
		{"有效的简单模板名", "welcome_email", ""},
		{"有效的带横线模板名", "user-registration", ""},
		{"有效的数字模板名", "template123", ""},
		{"有效的复合模板名", "project_update_v2", ""},
		{"空模板名", "", "模板名称不能为空"},
		{"模板名过短", "a", "模板名称长度必须在2-100字符之间"},
		{"模板名过长", strings.Repeat("a", 101), "模板名称长度必须在2-100字符之间"},
		{"包含特殊字符", "template@name", "模板名称只能包含字母、数字、下划线和横线"},
		{"包含空格", "template name", "模板名称只能包含字母、数字、下划线和横线"},
		{"包含中文", "模板名称", "模板名称只能包含字母、数字、下划线和横线"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTemplateName(tt.templateName)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestCreateNotificationRequestValidation 测试创建通知请求验证
func TestCreateNotificationRequestValidation(t *testing.T) {
	validTenantID := uuid.New()
	validUserID := uuid.New()

	tests := []struct {
		name          string
		req           *SimpleCreateNotificationRequest
		expectedError string
	}{
		{
			name: "有效的创建请求",
			req: &SimpleCreateNotificationRequest{
				TenantID: validTenantID,
				UserID:   validUserID,
				Type:     "user_action",
				Channel:  "email",
				Title:    "测试通知",
				Content:  "这是一个测试通知",
				Priority: "normal",
			},
			expectedError: "",
		},
		{
			name: "租户ID为空",
			req: &SimpleCreateNotificationRequest{
				TenantID: uuid.Nil,
				UserID:   validUserID,
				Type:     "user_action",
				Channel:  "email",
				Title:    "测试通知",
				Content:  "内容",
				Priority: "normal",
			},
			expectedError: "租户ID不能为空",
		},
		{
			name: "用户ID为空",
			req: &SimpleCreateNotificationRequest{
				TenantID: validTenantID,
				UserID:   uuid.Nil,
				Type:     "user_action",
				Channel:  "email",
				Title:    "测试通知",
				Content:  "内容",
				Priority: "normal",
			},
			expectedError: "用户ID不能为空",
		},
		{
			name: "通知类型无效",
			req: &SimpleCreateNotificationRequest{
				TenantID: validTenantID,
				UserID:   validUserID,
				Type:     "invalid_type",
				Channel:  "email",
				Title:    "测试通知",
				Content:  "内容",
				Priority: "normal",
			},
			expectedError: "通知类型: 无效的通知类型",
		},
		{
			name: "通知渠道无效",
			req: &SimpleCreateNotificationRequest{
				TenantID: validTenantID,
				UserID:   validUserID,
				Type:     "user_action",
				Channel:  "invalid_channel",
				Title:    "测试通知",
				Content:  "内容",
				Priority: "normal",
			},
			expectedError: "通知渠道: 无效的通知渠道",
		},
		{
			name: "通知标题为空",
			req: &SimpleCreateNotificationRequest{
				TenantID: validTenantID,
				UserID:   validUserID,
				Type:     "user_action",
				Channel:  "email",
				Title:    "",
				Content:  "内容",
				Priority: "normal",
			},
			expectedError: "通知内容: 通知标题不能为空",
		},
		{
			name: "通知优先级无效",
			req: &SimpleCreateNotificationRequest{
				TenantID: validTenantID,
				UserID:   validUserID,
				Type:     "user_action",
				Channel:  "email",
				Title:    "标题",
				Content:  "内容",
				Priority: "invalid_priority",
			},
			expectedError: "通知优先级: 无效的通知优先级",
		},
		{
			name:          "nil请求",
			req:           nil,
			expectedError: "创建通知请求不能为nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCreateNotificationRequest(tt.req)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestNotificationEdgeCases 测试通知边界情况
func TestNotificationEdgeCases(t *testing.T) {
	t.Run("极长的通知内容", func(t *testing.T) {
		longContent := strings.Repeat("非常长的通知内容", 1000)
		err := validateNotificationContent("标题", longContent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "通知内容长度不能超过10000字符")
	})

	t.Run("Unicode字符通知", func(t *testing.T) {
		err := validateNotificationContent("🔔 重要通知", "📧 您有新的邮件通知 ✅")
		assert.NoError(t, err)
	})

	t.Run("特殊电话号码格式", func(t *testing.T) {
		specialPhones := []string{
			"+8613812345678901", // 过长 (>15位)
			"+1",                // 过短 (<4位数字)
			"0013812345678",     // 以00开头
		}

		for _, phone := range specialPhones {
			err := validatePhoneNumber(phone)
			assert.Error(t, err, "电话号码 %s 应该无效", phone)
		}
	})

	t.Run("边界长度测试", func(t *testing.T) {
		// 测试边界长度 - 恰好200字符的标题
		exactly200Title := strings.Repeat("a", 200)
		err := validateNotificationContent(exactly200Title, "内容")
		assert.NoError(t, err)

		// 测试边界长度 - 恰好10000字符的内容
		exactly10000Content := strings.Repeat("a", 10000)
		err = validateNotificationContent("标题", exactly10000Content)
		assert.NoError(t, err)

		exactly2Template := "ab"
		err = validateTemplateName(exactly2Template)
		assert.NoError(t, err)

		exactly100Template := strings.Repeat("a", 100)
		err = validateTemplateName(exactly100Template)
		assert.NoError(t, err)
	})
}

// TestNotificationPerformance 测试通知性能
func TestNotificationPerformance(t *testing.T) {
	t.Run("批量邮箱验证", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 1000; i++ {
			email := fmt.Sprintf("user%d@example.com", i)
			err := validateEmailAddress(email)
			assert.NoError(t, err)
		}

		duration := time.Since(start)
		t.Logf("批量验证1000个邮箱耗时: %v", duration)

		// 验证应该在50毫秒内完成
		assert.Less(t, duration, 50*time.Millisecond)
	})

	t.Run("批量通知类型验证", func(t *testing.T) {
		start := time.Now()

		notificationTypes := []string{"system", "user_action", "security_alert", "project_update", "task_assignment"}
		for i := 0; i < 1000; i++ {
			notificationType := notificationTypes[i%len(notificationTypes)]
			err := validateNotificationType(notificationType)
			assert.NoError(t, err)
		}

		duration := time.Since(start)
		t.Logf("批量验证1000个通知类型耗时: %v", duration)

		// 验证应该在10毫秒内完成
		assert.Less(t, duration, 10*time.Millisecond)
	})

	t.Run("批量Webhook URL验证", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 1000; i++ {
			url := fmt.Sprintf("https://api%d.example.com/webhook", i)
			err := validateWebhookURL(url)
			assert.NoError(t, err)
		}

		duration := time.Since(start)
		t.Logf("批量验证1000个Webhook URL耗时: %v", duration)

		// 验证应该在20毫秒内完成
		assert.Less(t, duration, 20*time.Millisecond)
	})
}

// MockNotificationService 通知服务模拟实现
type MockNotificationService struct {
	notifications map[uuid.UUID]*SimpleNotification
	templates     map[uuid.UUID]*SimpleNotificationTemplate
	sentEmails    map[uuid.UUID]*SimpleEmailNotification
	sentSMS       map[uuid.UUID]*SimpleSMSNotification
	sentWebhooks  map[uuid.UUID]*SimpleWebhookNotification
}

// NewMockNotificationService 创建通知服务模拟
func NewMockNotificationService() *MockNotificationService {
	return &MockNotificationService{
		notifications: make(map[uuid.UUID]*SimpleNotification),
		templates:     make(map[uuid.UUID]*SimpleNotificationTemplate),
		sentEmails:    make(map[uuid.UUID]*SimpleEmailNotification),
		sentSMS:       make(map[uuid.UUID]*SimpleSMSNotification),
		sentWebhooks:  make(map[uuid.UUID]*SimpleWebhookNotification),
	}
}

// CreateNotification 模拟创建通知
func (m *MockNotificationService) CreateNotification(req *SimpleCreateNotificationRequest) (*SimpleNotification, error) {
	if err := validateCreateNotificationRequest(req); err != nil {
		return nil, err
	}

	notification := &SimpleNotification{
		ID:        uuid.New(),
		TenantID:  req.TenantID,
		UserID:    req.UserID,
		Type:      req.Type,
		Channel:   req.Channel,
		Title:     req.Title,
		Content:   req.Content,
		Data:      req.Data,
		Status:    "pending",
		Priority:  req.Priority,
		CreatedAt: time.Now(),
	}

	m.notifications[notification.ID] = notification
	return notification, nil
}

// SendEmailNotification 模拟发送邮件通知
func (m *MockNotificationService) SendEmailNotification(toEmail, fromEmail, subject, body string, isHTML bool) (*SimpleEmailNotification, error) {
	if err := validateEmailAddress(toEmail); err != nil {
		return nil, fmt.Errorf("收件人邮箱: %v", err)
	}

	if err := validateEmailAddress(fromEmail); err != nil {
		return nil, fmt.Errorf("发件人邮箱: %v", err)
	}

	if subject == "" {
		return nil, fmt.Errorf("邮件主题不能为空")
	}

	if body == "" {
		return nil, fmt.Errorf("邮件内容不能为空")
	}

	email := &SimpleEmailNotification{
		ID:        uuid.New(),
		ToEmail:   toEmail,
		FromEmail: fromEmail,
		Subject:   subject,
		Body:      body,
		IsHTML:    isHTML,
		Priority:  "normal",
		Status:    "sent",
		CreatedAt: time.Now(),
	}

	now := time.Now()
	email.SentAt = &now

	m.sentEmails[email.ID] = email
	return email, nil
}

// CreateTemplate 模拟创建通知模板
func (m *MockNotificationService) CreateTemplate(tenantID uuid.UUID, name, templateType, channel, subject, body string) (*SimpleNotificationTemplate, error) {
	if err := validateTemplateName(name); err != nil {
		return nil, err
	}

	if err := validateNotificationType(templateType); err != nil {
		return nil, fmt.Errorf("模板类型: %v", err)
	}

	if err := validateNotificationChannel(channel); err != nil {
		return nil, fmt.Errorf("模板渠道: %v", err)
	}

	// 检查模板名是否已存在
	for _, template := range m.templates {
		if template.Name == name && template.TenantID == tenantID {
			return nil, fmt.Errorf("模板名称 %s 已存在", name)
		}
	}

	template := &SimpleNotificationTemplate{
		ID:       uuid.New(),
		TenantID: tenantID,
		Name:     name,
		Type:     templateType,
		Channel:  channel,
		Subject:  subject,
		Body:     body,
		IsActive: true,
	}

	m.templates[template.ID] = template
	return template, nil
}

// TestMockNotificationService 测试通知服务模拟
func TestMockNotificationService(t *testing.T) {
	mockService := NewMockNotificationService()

	t.Run("创建通知成功", func(t *testing.T) {
		req := &SimpleCreateNotificationRequest{
			TenantID: uuid.New(),
			UserID:   uuid.New(),
			Type:     "user_action",
			Channel:  "email",
			Title:    "测试通知",
			Content:  "这是一个测试通知",
			Priority: "normal",
		}

		notification, err := mockService.CreateNotification(req)
		require.NoError(t, err)
		require.NotNil(t, notification)

		assert.Equal(t, req.TenantID, notification.TenantID)
		assert.Equal(t, req.UserID, notification.UserID)
		assert.Equal(t, req.Type, notification.Type)
		assert.Equal(t, req.Channel, notification.Channel)
		assert.Equal(t, req.Title, notification.Title)
		assert.Equal(t, req.Content, notification.Content)
		assert.Equal(t, "pending", notification.Status)
		assert.Equal(t, req.Priority, notification.Priority)
	})

	t.Run("发送邮件通知成功", func(t *testing.T) {
		email, err := mockService.SendEmailNotification(
			"user@example.com",
			"noreply@example.com",
			"测试邮件",
			"这是一封测试邮件",
			true,
		)

		require.NoError(t, err)
		require.NotNil(t, email)

		assert.Equal(t, "user@example.com", email.ToEmail)
		assert.Equal(t, "noreply@example.com", email.FromEmail)
		assert.Equal(t, "测试邮件", email.Subject)
		assert.Equal(t, "sent", email.Status)
		assert.True(t, email.IsHTML)
		assert.NotNil(t, email.SentAt)
	})

	t.Run("创建通知模板成功", func(t *testing.T) {
		tenantID := uuid.New()

		template, err := mockService.CreateTemplate(
			tenantID,
			"welcome_email",
			"user_action",
			"email",
			"欢迎使用我们的平台",
			"感谢您注册我们的平台！",
		)

		require.NoError(t, err)
		require.NotNil(t, template)

		assert.Equal(t, tenantID, template.TenantID)
		assert.Equal(t, "welcome_email", template.Name)
		assert.Equal(t, "user_action", template.Type)
		assert.Equal(t, "email", template.Channel)
		assert.True(t, template.IsActive)
	})

	t.Run("创建模板失败 - 名称重复", func(t *testing.T) {
		tenantID := uuid.New()

		// 第一次创建成功
		_, err := mockService.CreateTemplate(
			tenantID,
			"duplicate_template",
			"system",
			"email",
			"主题",
			"内容",
		)
		require.NoError(t, err)

		// 第二次创建失败
		template, err := mockService.CreateTemplate(
			tenantID,
			"duplicate_template",
			"system",
			"email",
			"主题2",
			"内容2",
		)

		assert.Error(t, err)
		assert.Nil(t, template)
		assert.Contains(t, err.Error(), "模板名称")
		assert.Contains(t, err.Error(), "已存在")
	})
}
