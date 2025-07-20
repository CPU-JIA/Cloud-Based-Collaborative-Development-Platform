package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TenantStatus 租户状态枚举
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusPending   TenantStatus = "pending"
	TenantStatusDeleted   TenantStatus = "deleted"
)

// TenantPlan 租户计划类型
type TenantPlan string

const (
	TenantPlanFree         TenantPlan = "free"
	TenantPlanBasic        TenantPlan = "basic"
	TenantPlanProfessional TenantPlan = "professional"
	TenantPlanEnterprise   TenantPlan = "enterprise"
)

// Tenant 租户主实体
type Tenant struct {
	ID           uuid.UUID    `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name         string       `json:"name" gorm:"type:varchar(255);not null;index"`
	Domain       string       `json:"domain" gorm:"type:varchar(255);uniqueIndex;not null"`
	Status       TenantStatus `json:"status" gorm:"type:varchar(50);not null;default:'pending';index"`
	Plan         TenantPlan   `json:"plan" gorm:"type:varchar(50);not null;default:'free';index"`
	BillingEmail string       `json:"billing_email" gorm:"type:varchar(255);not null"`
	Description  string       `json:"description" gorm:"type:text"`

	// 联系信息
	ContactName  string `json:"contact_name" gorm:"type:varchar(255)"`
	ContactEmail string `json:"contact_email" gorm:"type:varchar(255)"`
	ContactPhone string `json:"contact_phone" gorm:"type:varchar(50)"`

	// 地址信息
	Address    string `json:"address" gorm:"type:text"`
	City       string `json:"city" gorm:"type:varchar(100)"`
	State      string `json:"state" gorm:"type:varchar(100)"`
	Country    string `json:"country" gorm:"type:varchar(100)"`
	PostalCode string `json:"postal_code" gorm:"type:varchar(20)"`

	// 系统字段
	CreatedAt time.Time      `json:"created_at" gorm:"index"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	Config       *TenantConfig       `json:"config,omitempty" gorm:"foreignKey:TenantID"`
	Subscription *TenantSubscription `json:"subscription,omitempty" gorm:"foreignKey:TenantID"`
	Branding     *TenantBranding     `json:"branding,omitempty" gorm:"foreignKey:TenantID"`
	Users        []User              `json:"-" gorm:"foreignKey:TenantID"`
}

// TenantConfig 租户配置
type TenantConfig struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;uniqueIndex"`

	// 资源限制
	MaxUsers         int `json:"max_users" gorm:"default:10"`
	MaxProjects      int `json:"max_projects" gorm:"default:5"`
	MaxStorage       int `json:"max_storage" gorm:"default:1024"` // MB
	MaxAPICallsDaily int `json:"max_api_calls_daily" gorm:"default:10000"`

	// 功能开关
	FeatureFlags map[string]interface{} `json:"feature_flags" gorm:"type:jsonb"`

	// 安全策略
	SecurityPolicy map[string]interface{} `json:"security_policy" gorm:"type:jsonb"`

	// 集成配置
	IntegrationSettings map[string]interface{} `json:"integration_settings" gorm:"type:jsonb"`

	// 系统字段
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联
	Tenant Tenant `json:"-" gorm:"foreignKey:TenantID"`
}

// SubscriptionStatus 订阅状态
type SubscriptionStatus string

const (
	SubscriptionStatusActive   SubscriptionStatus = "active"
	SubscriptionStatusCanceled SubscriptionStatus = "canceled"
	SubscriptionStatusExpired  SubscriptionStatus = "expired"
	SubscriptionStatusTrialing SubscriptionStatus = "trialing"
)

// BillingCycle 计费周期
type BillingCycle string

const (
	BillingCycleMonthly BillingCycle = "monthly"
	BillingCycleYearly  BillingCycle = "yearly"
)

// TenantSubscription 租户订阅信息
type TenantSubscription struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;uniqueIndex"`

	// 订阅信息
	PlanType     TenantPlan         `json:"plan_type" gorm:"type:varchar(50);not null"`
	Status       SubscriptionStatus `json:"status" gorm:"type:varchar(50);not null;default:'trialing'"`
	BillingCycle BillingCycle       `json:"billing_cycle" gorm:"type:varchar(20);not null;default:'monthly'"`

	// 时间信息
	TrialEndsAt        *time.Time `json:"trial_ends_at" gorm:"index"`
	CurrentPeriodStart time.Time  `json:"current_period_start" gorm:"index"`
	CurrentPeriodEnd   time.Time  `json:"current_period_end" gorm:"index"`
	ExpiresAt          *time.Time `json:"expires_at" gorm:"index"`

	// 计费信息
	Amount          float64 `json:"amount" gorm:"type:decimal(10,2)"`
	Currency        string  `json:"currency" gorm:"type:varchar(3);default:'USD'"`
	PaymentMethodID string  `json:"payment_method_id" gorm:"type:varchar(255)"`

	// 外部系统ID
	StripeCustomerID     string `json:"stripe_customer_id" gorm:"type:varchar(255);index"`
	StripeSubscriptionID string `json:"stripe_subscription_id" gorm:"type:varchar(255);uniqueIndex"`

	// 使用量统计
	UsageMetrics map[string]interface{} `json:"usage_metrics" gorm:"type:jsonb"`

	// 系统字段
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联
	Tenant Tenant `json:"-" gorm:"foreignKey:TenantID"`
}

// TenantBranding 租户品牌定制
type TenantBranding struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;uniqueIndex"`

	// Logo和图标
	LogoURL    string `json:"logo_url" gorm:"type:varchar(500)"`
	FaviconURL string `json:"favicon_url" gorm:"type:varchar(500)"`

	// 主题配色
	PrimaryColor   string `json:"primary_color" gorm:"type:varchar(7);default:'#1890ff'"` // HEX color
	SecondaryColor string `json:"secondary_color" gorm:"type:varchar(7);default:'#52c41a'"`
	AccentColor    string `json:"accent_color" gorm:"type:varchar(7);default:'#fa8c16'"`

	// 自定义域名
	CustomDomain    string `json:"custom_domain" gorm:"type:varchar(255);uniqueIndex"`
	CustomDomainSSL bool   `json:"custom_domain_ssl" gorm:"default:false"`

	// 自定义CSS
	CustomCSS string `json:"custom_css" gorm:"type:text"`

	// 邮件品牌
	EmailFromName string `json:"email_from_name" gorm:"type:varchar(255)"`
	EmailReplyTo  string `json:"email_reply_to" gorm:"type:varchar(255)"`

	// 其他设置
	BrandingSettings map[string]interface{} `json:"branding_settings" gorm:"type:jsonb"`

	// 系统字段
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联
	Tenant Tenant `json:"-" gorm:"foreignKey:TenantID"`
}

// TenantAuditLog 租户审计日志
type TenantAuditLog struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`

	// 操作信息
	Action   string `json:"action" gorm:"type:varchar(100);not null;index"` // create, update, delete, etc.
	Entity   string `json:"entity" gorm:"type:varchar(100);not null"`       // tenant, config, subscription, etc.
	EntityID string `json:"entity_id" gorm:"type:varchar(255);index"`

	// 用户信息
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;index"`
	UserEmail string    `json:"user_email" gorm:"type:varchar(255);index"`

	// 变更内容
	OldValues map[string]interface{} `json:"old_values" gorm:"type:jsonb"`
	NewValues map[string]interface{} `json:"new_values" gorm:"type:jsonb"`

	// 请求信息
	IPAddress string `json:"ip_address" gorm:"type:varchar(45);index"`
	UserAgent string `json:"user_agent" gorm:"type:text"`

	// 系统字段
	CreatedAt time.Time `json:"created_at" gorm:"index"`

	// 关联
	Tenant Tenant `json:"-" gorm:"foreignKey:TenantID"`
	User   User   `json:"-" gorm:"foreignKey:UserID"`
}

// BeforeCreate GORM钩子：创建前设置默认值
func (t *Tenant) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

// TableName 指定表名
func (Tenant) TableName() string {
	return "tenants"
}

func (TenantConfig) TableName() string {
	return "tenant_configs"
}

func (TenantSubscription) TableName() string {
	return "tenant_subscriptions"
}

func (TenantBranding) TableName() string {
	return "tenant_brandings"
}

func (TenantAuditLog) TableName() string {
	return "tenant_audit_logs"
}

// 验证方法

// IsActive 检查租户是否激活
func (t *Tenant) IsActive() bool {
	return t.Status == TenantStatusActive
}

// IsExpired 检查订阅是否过期
func (ts *TenantSubscription) IsExpired() bool {
	if ts.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*ts.ExpiresAt)
}

// IsInTrial 检查是否在试用期
func (ts *TenantSubscription) IsInTrial() bool {
	return ts.Status == SubscriptionStatusTrialing &&
		ts.TrialEndsAt != nil &&
		time.Now().Before(*ts.TrialEndsAt)
}

// CanUpgrade 检查是否可以升级计划
func (t *Tenant) CanUpgrade() bool {
	return t.Plan != TenantPlanEnterprise && t.Status == TenantStatusActive
}

// GetUsagePercentage 获取资源使用率
func (tc *TenantConfig) GetUsagePercentage(currentUsers, currentProjects int) map[string]float64 {
	usage := make(map[string]float64)

	if tc.MaxUsers > 0 {
		usage["users"] = float64(currentUsers) / float64(tc.MaxUsers) * 100
	}

	if tc.MaxProjects > 0 {
		usage["projects"] = float64(currentProjects) / float64(tc.MaxProjects) * 100
	}

	return usage
}

// IsFeatureEnabled 检查功能是否启用
func (tc *TenantConfig) IsFeatureEnabled(feature string) bool {
	if tc.FeatureFlags == nil {
		return false
	}

	enabled, exists := tc.FeatureFlags[feature]
	if !exists {
		return false
	}

	if enabledBool, ok := enabled.(bool); ok {
		return enabledBool
	}

	return false
}
