package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Notification 通知记录模型
type Notification struct {
	ID        uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    *uuid.UUID      `json:"user_id,omitempty" gorm:"type:uuid;index"`
	TenantID  uuid.UUID       `json:"tenant_id" gorm:"type:uuid;not null;index"`
	ProjectID *uuid.UUID      `json:"project_id,omitempty" gorm:"type:uuid;index"`
	
	// 通知基本信息
	Type        string      `json:"type" gorm:"not null;index"` // task_assigned, sprint_started, build_failed, etc.
	Category    string      `json:"category" gorm:"not null"`   // project, system, security, etc.
	Priority    string      `json:"priority" gorm:"not null;default:'medium'"` // low, medium, high, critical
	Title       string      `json:"title" gorm:"not null;size:500"`
	Content     string      `json:"content" gorm:"type:text"`
	
	// 发送渠道配置
	Channels    Channels    `json:"channels" gorm:"type:jsonb"`
	
	// 状态管理
	Status      string      `json:"status" gorm:"not null;default:'pending';index"` // pending, processing, sent, failed, cancelled
	SentAt      *time.Time  `json:"sent_at,omitempty"`
	FailedAt    *time.Time  `json:"failed_at,omitempty"`
	RetryCount  int         `json:"retry_count" gorm:"default:0"`
	MaxRetries  int         `json:"max_retries" gorm:"default:3"`
	
	// 元数据
	Metadata    json.RawMessage `json:"metadata,omitempty" gorm:"type:jsonb"`
	TemplateID  *uuid.UUID      `json:"template_id,omitempty" gorm:"type:uuid"`
	EventData   json.RawMessage `json:"event_data,omitempty" gorm:"type:jsonb"`
	
	// 追踪信息
	SourceEvent    string     `json:"source_event,omitempty" gorm:"index"`
	CorrelationID  string     `json:"correlation_id,omitempty" gorm:"index"`
	
	// 审计字段
	CreatedAt   time.Time   `json:"created_at" gorm:"not null"`
	UpdatedAt   time.Time   `json:"updated_at" gorm:"not null"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	CreatedBy   uuid.UUID   `json:"created_by" gorm:"type:uuid"`
	
	// 关联关系
	Template    *NotificationTemplate `json:"template,omitempty" gorm:"foreignKey:TemplateID"`
	DeliveryLogs []DeliveryLog        `json:"delivery_logs,omitempty" gorm:"foreignKey:NotificationID"`
}

// Channels 通知发送渠道配置
type Channels struct {
	Email     *EmailChannel     `json:"email,omitempty"`
	Webhook   *WebhookChannel   `json:"webhook,omitempty"`
	InApp     *InAppChannel     `json:"in_app,omitempty"`
	Push      *PushChannel      `json:"push,omitempty"`
}

// EmailChannel 邮件渠道配置
type EmailChannel struct {
	Enabled   bool     `json:"enabled"`
	To        []string `json:"to"`
	CC        []string `json:"cc,omitempty"`
	BCC       []string `json:"bcc,omitempty"`
	Subject   string   `json:"subject"`
	Template  string   `json:"template,omitempty"`
}

// WebhookChannel Webhook渠道配置
type WebhookChannel struct {
	Enabled bool              `json:"enabled"`
	URL     string            `json:"url"`
	Method  string            `json:"method"` // POST, PUT
	Headers map[string]string `json:"headers,omitempty"`
	Payload json.RawMessage   `json:"payload,omitempty"`
}

// InAppChannel 应用内通知渠道配置
type InAppChannel struct {
	Enabled bool   `json:"enabled"`
	Badge   bool   `json:"badge"`   // 是否显示徽章
	Sound   bool   `json:"sound"`   // 是否播放声音
	Popup   bool   `json:"popup"`   // 是否弹窗显示
}

// PushChannel 推送通知渠道配置
type PushChannel struct {
	Enabled bool              `json:"enabled"`
	Title   string            `json:"title"`
	Body    string            `json:"body"`
	Icon    string            `json:"icon,omitempty"`
	Data    map[string]string `json:"data,omitempty"`
}

// NotificationTemplate 通知模板模型
type NotificationTemplate struct {
	ID          uuid.UUID   `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID    uuid.UUID   `json:"tenant_id" gorm:"type:uuid;not null;index"`
	
	// 模板基本信息
	Name        string      `json:"name" gorm:"not null;size:255"`
	Type        string      `json:"type" gorm:"not null;index"` // 对应Notification.Type
	Category    string      `json:"category" gorm:"not null"`
	Description string      `json:"description,omitempty" gorm:"size:1000"`
	
	// 模板内容
	SubjectTemplate string      `json:"subject_template" gorm:"not null;size:500"`
	BodyTemplate    string      `json:"body_template" gorm:"type:text;not null"`
	HTMLTemplate    string      `json:"html_template,omitempty" gorm:"type:text"`
	
	// 模板配置
	Variables   json.RawMessage `json:"variables,omitempty" gorm:"type:jsonb"` // 支持的变量列表
	Language    string          `json:"language" gorm:"default:'zh-CN'"`
	Format      string          `json:"format" gorm:"default:'text'"` // text, html, markdown
	
	// 默认渠道配置
	DefaultChannels Channels `json:"default_channels" gorm:"type:jsonb"`
	
	// 状态管理
	IsActive    bool        `json:"is_active" gorm:"default:true"`
	Version     int         `json:"version" gorm:"default:1"`
	
	// 审计字段
	CreatedAt   time.Time   `json:"created_at" gorm:"not null"`
	UpdatedAt   time.Time   `json:"updated_at" gorm:"not null"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	CreatedBy   uuid.UUID   `json:"created_by" gorm:"type:uuid"`
	UpdatedBy   uuid.UUID   `json:"updated_by" gorm:"type:uuid"`
}

// NotificationRule 通知规则模型
type NotificationRule struct {
	ID          uuid.UUID   `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID    uuid.UUID   `json:"tenant_id" gorm:"type:uuid;not null;index"`
	UserID      *uuid.UUID  `json:"user_id,omitempty" gorm:"type:uuid;index"` // null表示全局规则
	ProjectID   *uuid.UUID  `json:"project_id,omitempty" gorm:"type:uuid;index"`
	
	// 规则配置
	Name        string      `json:"name" gorm:"not null;size:255"`
	Description string      `json:"description,omitempty" gorm:"size:1000"`
	EventTypes  []string    `json:"event_types" gorm:"type:jsonb;not null"` // 监听的事件类型
	Conditions  json.RawMessage `json:"conditions,omitempty" gorm:"type:jsonb"` // 触发条件
	
	// 通知配置
	TemplateID  uuid.UUID   `json:"template_id" gorm:"type:uuid;not null"`
	Channels    Channels    `json:"channels" gorm:"type:jsonb"`
	Priority    string      `json:"priority" gorm:"not null;default:'medium'"`
	
	// 频率控制
	RateLimit   *RateLimit  `json:"rate_limit,omitempty" gorm:"type:jsonb"`
	QuietHours  *QuietHours `json:"quiet_hours,omitempty" gorm:"type:jsonb"`
	
	// 状态管理
	IsActive    bool        `json:"is_active" gorm:"default:true"`
	
	// 审计字段
	CreatedAt   time.Time   `json:"created_at" gorm:"not null"`
	UpdatedAt   time.Time   `json:"updated_at" gorm:"not null"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	CreatedBy   uuid.UUID   `json:"created_by" gorm:"type:uuid"`
	UpdatedBy   uuid.UUID   `json:"updated_by" gorm:"type:uuid"`
	
	// 关联关系
	Template    *NotificationTemplate `json:"template,omitempty" gorm:"foreignKey:TemplateID"`
}

// RateLimit 频率限制配置
type RateLimit struct {
	MaxCount    int    `json:"max_count"`    // 最大通知数量
	TimeWindow  int    `json:"time_window"`  // 时间窗口(分钟)
	Strategy    string `json:"strategy"`     // throttle(节流), batch(批量), skip(跳过)
}

// QuietHours 免打扰时间配置
type QuietHours struct {
	Enabled   bool   `json:"enabled"`
	StartTime string `json:"start_time"` // HH:MM格式
	EndTime   string `json:"end_time"`   // HH:MM格式
	Timezone  string `json:"timezone"`   // 时区
	Weekdays  []int  `json:"weekdays"`   // 0-6, 0=Sunday
}

// DeliveryLog 发送日志模型
type DeliveryLog struct {
	ID             uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	NotificationID uuid.UUID  `json:"notification_id" gorm:"type:uuid;not null;index"`
	
	// 发送信息
	Channel       string     `json:"channel" gorm:"not null"` // email, webhook, in_app, push
	Recipient     string     `json:"recipient" gorm:"not null"`
	
	// 状态追踪
	Status        string     `json:"status" gorm:"not null"` // sending, sent, failed, bounced
	AttemptCount  int        `json:"attempt_count" gorm:"default:1"`
	
	// 时间记录
	SentAt        *time.Time `json:"sent_at,omitempty"`
	DeliveredAt   *time.Time `json:"delivered_at,omitempty"`
	FailedAt      *time.Time `json:"failed_at,omitempty"`
	
	// 错误信息
	ErrorMessage  string     `json:"error_message,omitempty" gorm:"type:text"`
	ErrorCode     string     `json:"error_code,omitempty"`
	
	// 外部追踪
	ExternalID    string     `json:"external_id,omitempty" gorm:"index"` // 第三方服务的消息ID
	
	// 响应数据
	Response      json.RawMessage `json:"response,omitempty" gorm:"type:jsonb"`
	
	// 审计字段
	CreatedAt     time.Time  `json:"created_at" gorm:"not null"`
	UpdatedAt     time.Time  `json:"updated_at" gorm:"not null"`
}

// UserNotificationSettings 用户通知偏好设置
type UserNotificationSettings struct {
	ID          uuid.UUID   `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID      uuid.UUID   `json:"user_id" gorm:"type:uuid;not null;uniqueIndex"`
	TenantID    uuid.UUID   `json:"tenant_id" gorm:"type:uuid;not null;index"`
	
	// 全局设置
	GloballyEnabled bool     `json:"globally_enabled" gorm:"default:true"`
	
	// 渠道偏好
	EmailEnabled    bool     `json:"email_enabled" gorm:"default:true"`
	InAppEnabled    bool     `json:"in_app_enabled" gorm:"default:true"`
	PushEnabled     bool     `json:"push_enabled" gorm:"default:true"`
	
	// 类别设置
	CategorySettings json.RawMessage `json:"category_settings" gorm:"type:jsonb"`
	
	// 免打扰设置
	QuietHours      *QuietHours `json:"quiet_hours,omitempty" gorm:"type:jsonb"`
	
	// 频率设置
	DigestMode      bool   `json:"digest_mode" gorm:"default:false"`      // 摘要模式
	DigestFrequency string `json:"digest_frequency" gorm:"default:'daily'"` // hourly, daily, weekly
	
	// 审计字段
	CreatedAt   time.Time   `json:"created_at" gorm:"not null"`
	UpdatedAt   time.Time   `json:"updated_at" gorm:"not null"`
	UpdatedBy   uuid.UUID   `json:"updated_by" gorm:"type:uuid"`
}

// TableName 指定表名
func (Notification) TableName() string {
	return "notifications"
}

func (NotificationTemplate) TableName() string {
	return "notification_templates"
}

func (NotificationRule) TableName() string {
	return "notification_rules"
}

func (DeliveryLog) TableName() string {
	return "delivery_logs"
}

func (UserNotificationSettings) TableName() string {
	return "user_notification_settings"
}