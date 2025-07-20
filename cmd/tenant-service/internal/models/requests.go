package models

import (
	"github.com/cloud-platform/collaborative-dev/shared/models"
	"github.com/google/uuid"
)

// CreateTenantRequest 创建租户请求
type CreateTenantRequest struct {
	Name         string            `json:"name" binding:"required,min=2,max=255" example:"示例公司"`
	Domain       string            `json:"domain" binding:"required,min=3,max=255,hostname" example:"example.com"`
	Plan         models.TenantPlan `json:"plan" binding:"required" example:"basic"`
	BillingEmail string            `json:"billing_email" binding:"required,email" example:"billing@example.com"`
	Description  string            `json:"description" example:"示例公司的开发协作平台"`
	ContactName  string            `json:"contact_name" binding:"required" example:"张三"`
	ContactEmail string            `json:"contact_email" binding:"required,email" example:"zhang.san@example.com"`
	ContactPhone string            `json:"contact_phone" example:"+86-138-0000-0000"`
	Address      string            `json:"address" example:"北京市朝阳区"`
	City         string            `json:"city" example:"北京"`
	State        string            `json:"state" example:"北京市"`
	Country      string            `json:"country" example:"中国"`
	PostalCode   string            `json:"postal_code" example:"100000"`
}

// UpdateTenantRequest 更新租户请求
type UpdateTenantRequest struct {
	Name         *string `json:"name,omitempty" binding:"omitempty,min=2,max=255"`
	Description  *string `json:"description,omitempty"`
	ContactName  *string `json:"contact_name,omitempty"`
	ContactEmail *string `json:"contact_email,omitempty" binding:"omitempty,email"`
	ContactPhone *string `json:"contact_phone,omitempty"`
	Address      *string `json:"address,omitempty"`
	City         *string `json:"city,omitempty"`
	State        *string `json:"state,omitempty"`
	Country      *string `json:"country,omitempty"`
	PostalCode   *string `json:"postal_code,omitempty"`
}

// TenantSearchRequest 租户搜索请求
type TenantSearchRequest struct {
	Query  string              `form:"query" binding:"omitempty,min=2" example:"示例"`
	Status models.TenantStatus `form:"status" binding:"omitempty" example:"active"`
	Plan   models.TenantPlan   `form:"plan" binding:"omitempty" example:"basic"`
	Limit  int                 `form:"limit" binding:"omitempty,min=1,max=100" example:"20"`
	Offset int                 `form:"offset" binding:"omitempty,min=0" example:"0"`
}

// UpdateTenantConfigRequest 更新租户配置请求
type UpdateTenantConfigRequest struct {
	MaxUsers            *int                    `json:"max_users,omitempty" binding:"omitempty,min=1"`
	MaxProjects         *int                    `json:"max_projects,omitempty" binding:"omitempty,min=1"`
	MaxStorage          *int                    `json:"max_storage,omitempty" binding:"omitempty,min=1"`
	MaxAPICallsDaily    *int                    `json:"max_api_calls_daily,omitempty" binding:"omitempty,min=1"`
	FeatureFlags        *map[string]interface{} `json:"feature_flags,omitempty"`
	SecurityPolicy      *map[string]interface{} `json:"security_policy,omitempty"`
	IntegrationSettings *map[string]interface{} `json:"integration_settings,omitempty"`
}

// UpdateTenantBrandingRequest 更新租户品牌请求
type UpdateTenantBrandingRequest struct {
	LogoURL          *string                 `json:"logo_url,omitempty" binding:"omitempty,url"`
	FaviconURL       *string                 `json:"favicon_url,omitempty" binding:"omitempty,url"`
	PrimaryColor     *string                 `json:"primary_color,omitempty" binding:"omitempty,hexcolor"`
	SecondaryColor   *string                 `json:"secondary_color,omitempty" binding:"omitempty,hexcolor"`
	AccentColor      *string                 `json:"accent_color,omitempty" binding:"omitempty,hexcolor"`
	CustomDomain     *string                 `json:"custom_domain,omitempty" binding:"omitempty,hostname"`
	CustomDomainSSL  *bool                   `json:"custom_domain_ssl,omitempty"`
	CustomCSS        *string                 `json:"custom_css,omitempty"`
	EmailFromName    *string                 `json:"email_from_name,omitempty"`
	EmailReplyTo     *string                 `json:"email_reply_to,omitempty" binding:"omitempty,email"`
	BrandingSettings *map[string]interface{} `json:"branding_settings,omitempty"`
}

// CreateInvitationRequest 创建邀请请求
type CreateInvitationRequest struct {
	Email   string `json:"email" binding:"required,email" example:"user@example.com"`
	Role    string `json:"role" binding:"required" example:"member"`
	Message string `json:"message" example:"欢迎加入我们的团队"`
}

// AcceptInvitationRequest 接受邀请请求
type AcceptInvitationRequest struct {
	Token    string `json:"token" binding:"required" example:"invitation-token"`
	Password string `json:"password" binding:"required,min=8" example:"password123"`
}

// TenantStatsFilter 租户统计筛选
type TenantStatsFilter struct {
	StartDate string              `form:"start_date" binding:"omitempty" example:"2024-01-01"`
	EndDate   string              `form:"end_date" binding:"omitempty" example:"2024-12-31"`
	Status    models.TenantStatus `form:"status" binding:"omitempty" example:"active"`
	Plan      models.TenantPlan   `form:"plan" binding:"omitempty" example:"basic"`
}

// BatchUpdateTenantsRequest 批量更新租户请求
type BatchUpdateTenantsRequest struct {
	TenantIDs []uuid.UUID            `json:"tenant_ids" binding:"required,min=1"`
	Updates   map[string]interface{} `json:"updates" binding:"required"`
}
