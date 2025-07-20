package models

import (
	"time"

	"github.com/cloud-platform/collaborative-dev/shared/models"
	"github.com/google/uuid"
)

// Type aliases for convenience
type TenantInvitationStatus = models.InvitationStatus

// TenantResponse 租户响应
type TenantResponse struct {
	ID           uuid.UUID           `json:"id"`
	Name         string              `json:"name"`
	Domain       string              `json:"domain"`
	Status       models.TenantStatus `json:"status"`
	Plan         models.TenantPlan   `json:"plan"`
	BillingEmail string              `json:"billing_email"`
	Description  string              `json:"description"`
	ContactName  string              `json:"contact_name"`
	ContactEmail string              `json:"contact_email"`
	ContactPhone string              `json:"contact_phone"`
	Address      string              `json:"address"`
	City         string              `json:"city"`
	State        string              `json:"state"`
	Country      string              `json:"country"`
	PostalCode   string              `json:"postal_code"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
}

// TenantWithConfigResponse 包含配置的租户响应
type TenantWithConfigResponse struct {
	*TenantResponse
	Config *models.TenantConfig `json:"config,omitempty"`
}

// TenantCompleteResponse 完整租户信息响应
type TenantCompleteResponse struct {
	*TenantResponse
	Config       *models.TenantConfig       `json:"config,omitempty"`
	Subscription *models.TenantSubscription `json:"subscription,omitempty"`
	Branding     *models.TenantBranding     `json:"branding,omitempty"`
	UserCount    int                        `json:"user_count"`
	ProjectCount int                        `json:"project_count"`
}

// TenantListResponse 租户列表响应
type TenantListResponse struct {
	Tenants    []TenantResponse `json:"tenants"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

// TenantStatsResponse 租户统计响应
type TenantStatsResponse struct {
	Total    int64                         `json:"total"`
	ByStatus map[models.TenantStatus]int64 `json:"by_status"`
	ByPlan   map[models.TenantPlan]int64   `json:"by_plan"`
	Recent   []TenantResponse              `json:"recent"`
	Growth   TenantGrowthStats             `json:"growth"`
	Revenue  TenantRevenueStats            `json:"revenue"`
}

// TenantGrowthStats 租户增长统计
type TenantGrowthStats struct {
	ThisMonth  int64   `json:"this_month"`
	LastMonth  int64   `json:"last_month"`
	GrowthRate float64 `json:"growth_rate"`
	YearToDate int64   `json:"year_to_date"`
}

// TenantRevenueStats 租户收入统计
type TenantRevenueStats struct {
	MonthlyRevenue float64                       `json:"monthly_revenue"`
	YearlyRevenue  float64                       `json:"yearly_revenue"`
	ByPlan         map[models.TenantPlan]float64 `json:"by_plan"`
	AverageRevenue float64                       `json:"average_revenue"`
}

// TenantUsageResponse 租户使用量响应
type TenantUsageResponse struct {
	TenantID     uuid.UUID              `json:"tenant_id"`
	CurrentUsers int                    `json:"current_users"`
	MaxUsers     int                    `json:"max_users"`
	UsagePercent map[string]float64     `json:"usage_percent"`
	Metrics      map[string]interface{} `json:"metrics"`
}

// InvitationResponse 邀请响应
type InvitationResponse struct {
	ID        uuid.UUID              `json:"id"`
	TenantID  uuid.UUID              `json:"tenant_id"`
	Email     string                 `json:"email"`
	Role      string                 `json:"role"`
	Status    TenantInvitationStatus `json:"status"`
	Token     string                 `json:"token"`
	ExpiresAt time.Time              `json:"expires_at"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// TenantDomainCheckResponse 域名检查响应
type TenantDomainCheckResponse struct {
	Domain    string `json:"domain"`
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

// TenantHealthResponse 租户健康状况响应
type TenantHealthResponse struct {
	TenantID      uuid.UUID              `json:"tenant_id"`
	Status        models.TenantStatus    `json:"status"`
	HealthScore   float64                `json:"health_score"`
	Issues        []string               `json:"issues"`
	Metrics       map[string]interface{} `json:"metrics"`
	LastCheckedAt time.Time              `json:"last_checked_at"`
}

// TenantActivityResponse 租户活动响应
type TenantActivityResponse struct {
	TenantID   uuid.UUID        `json:"tenant_id"`
	Activities []TenantActivity `json:"activities"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
}

// TenantActivity 租户活动
type TenantActivity struct {
	ID        uuid.UUID              `json:"id"`
	Action    string                 `json:"action"`
	Entity    string                 `json:"entity"`
	EntityID  string                 `json:"entity_id"`
	UserEmail string                 `json:"user_email"`
	Details   map[string]interface{} `json:"details"`
	CreatedAt time.Time              `json:"created_at"`
}

// BatchOperationResponse 批量操作响应
type BatchOperationResponse struct {
	Successful []uuid.UUID           `json:"successful"`
	Failed     []BatchOperationError `json:"failed"`
	Total      int                   `json:"total"`
}

// BatchOperationError 批量操作错误
type BatchOperationError struct {
	TenantID uuid.UUID `json:"tenant_id"`
	Error    string    `json:"error"`
}

// TenantMigrationResponse 租户迁移响应
type TenantMigrationResponse struct {
	TenantID     uuid.UUID `json:"tenant_id"`
	Status       string    `json:"status"`
	Progress     float64   `json:"progress"`
	EstimatedETA string    `json:"estimated_eta,omitempty"`
	Message      string    `json:"message"`
}
