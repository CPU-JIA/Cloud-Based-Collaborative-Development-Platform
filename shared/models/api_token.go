package models

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"time"
)

// APIToken represents a long-lived API access token
type APIToken struct {
	ID           uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TenantID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"tenant_id"`
	UserID       uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	Name         string         `gorm:"size:255;not null" json:"name"`
	Description  string         `gorm:"size:1000" json:"description"`
	TokenHash    string         `gorm:"size:255;not null;unique" json:"-"`    // 存储token的哈希值
	TokenPrefix  string         `gorm:"size:10;not null" json:"token_prefix"` // 用于显示的前缀
	Scopes       datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"scopes"`
	Permissions  datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"permissions"`
	Status       string         `gorm:"size:20;default:active" json:"status"` // active, revoked, expired, suspended
	LastUsedAt   *time.Time     `json:"last_used_at"`
	LastUsedIP   string         `gorm:"size:45" json:"last_used_ip"`
	UseCount     int64          `gorm:"default:0" json:"use_count"`
	RateLimitRPS int            `gorm:"default:100" json:"rate_limit_rps"` // 每秒请求限制
	ExpiresAt    *time.Time     `json:"expires_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	RevokedAt    *time.Time     `json:"revoked_at"`
	RevokedBy    *uuid.UUID     `gorm:"type:uuid" json:"revoked_by"`

	// Relations
	User          *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	RevokedByUser *User `gorm:"foreignKey:RevokedBy" json:"revoked_by_user,omitempty"`
}

// APITokenUsage tracks API token usage for analytics and monitoring
type APITokenUsage struct {
	ID           uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TenantID     uuid.UUID `gorm:"type:uuid;not null;index" json:"tenant_id"`
	TokenID      uuid.UUID `gorm:"type:uuid;not null;index" json:"token_id"`
	Endpoint     string    `gorm:"size:500;not null" json:"endpoint"`
	Method       string    `gorm:"size:10;not null" json:"method"`
	StatusCode   int       `gorm:"not null" json:"status_code"`
	ResponseTime int64     `gorm:"not null" json:"response_time"` // microseconds
	IPAddress    string    `gorm:"size:45" json:"ip_address"`
	UserAgent    string    `gorm:"size:1000" json:"user_agent"`
	RequestSize  int64     `gorm:"default:0" json:"request_size"`  // bytes
	ResponseSize int64     `gorm:"default:0" json:"response_size"` // bytes
	ErrorMessage string    `gorm:"size:1000" json:"error_message"`
	CreatedAt    time.Time `json:"created_at"`

	// Relations
	Token *APIToken `gorm:"foreignKey:TokenID" json:"token,omitempty"`
}

// APIScope represents available API scopes
type APIScope struct {
	ID          uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	Name        string         `gorm:"size:100;not null;unique" json:"name"`
	Description string         `gorm:"size:500" json:"description"`
	Category    string         `gorm:"size:50" json:"category"`
	Resources   datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"resources"`
	Actions     datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"actions"`
	IsSystem    bool           `gorm:"default:false" json:"is_system"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// APITokenScope represents the relationship between tokens and scopes
type APITokenScope struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TenantID  uuid.UUID `gorm:"type:uuid;not null;index" json:"tenant_id"`
	TokenID   uuid.UUID `gorm:"type:uuid;not null;index" json:"token_id"`
	ScopeID   uuid.UUID `gorm:"type:uuid;not null;index" json:"scope_id"`
	GrantedAt time.Time `json:"granted_at"`

	// Relations
	Token *APIToken `gorm:"foreignKey:TokenID" json:"token,omitempty"`
	Scope *APIScope `gorm:"foreignKey:ScopeID" json:"scope,omitempty"`
}

// APIRateLimit tracks rate limiting for API tokens
type APIRateLimit struct {
	ID           uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TokenID      uuid.UUID `gorm:"type:uuid;not null;index" json:"token_id"`
	WindowStart  time.Time `gorm:"not null;index" json:"window_start"`
	WindowSize   int       `gorm:"not null" json:"window_size"` // seconds
	RequestCount int       `gorm:"default:0" json:"request_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Relations
	Token *APIToken `gorm:"foreignKey:TokenID" json:"token,omitempty"`
}

// Predefined API scopes
var SystemAPIScopes = []APIScope{
	{
		Name:        "users:read",
		Description: "Read user information",
		Category:    "users",
		Resources:   datatypes.JSON(`["users"]`),
		Actions:     datatypes.JSON(`["read", "list"]`),
		IsSystem:    true,
	},
	{
		Name:        "users:write",
		Description: "Create and update users",
		Category:    "users",
		Resources:   datatypes.JSON(`["users"]`),
		Actions:     datatypes.JSON(`["create", "update"]`),
		IsSystem:    true,
	},
	{
		Name:        "users:delete",
		Description: "Delete users",
		Category:    "users",
		Resources:   datatypes.JSON(`["users"]`),
		Actions:     datatypes.JSON(`["delete"]`),
		IsSystem:    true,
	},
	{
		Name:        "projects:read",
		Description: "Read project information",
		Category:    "projects",
		Resources:   datatypes.JSON(`["projects", "tasks", "sprints"]`),
		Actions:     datatypes.JSON(`["read", "list"]`),
		IsSystem:    true,
	},
	{
		Name:        "projects:write",
		Description: "Create and update projects",
		Category:    "projects",
		Resources:   datatypes.JSON(`["projects", "tasks", "sprints"]`),
		Actions:     datatypes.JSON(`["create", "update"]`),
		IsSystem:    true,
	},
	{
		Name:        "projects:admin",
		Description: "Full project administration",
		Category:    "projects",
		Resources:   datatypes.JSON(`["projects", "tasks", "sprints", "members"]`),
		Actions:     datatypes.JSON(`["create", "read", "update", "delete", "manage"]`),
		IsSystem:    true,
	},
	{
		Name:        "repos:read",
		Description: "Read repository information",
		Category:    "repositories",
		Resources:   datatypes.JSON(`["repositories", "commits", "branches"]`),
		Actions:     datatypes.JSON(`["read", "list", "clone"]`),
		IsSystem:    true,
	},
	{
		Name:        "repos:write",
		Description: "Push to repositories",
		Category:    "repositories",
		Resources:   datatypes.JSON(`["repositories", "commits", "branches"]`),
		Actions:     datatypes.JSON(`["push", "create_branch", "merge"]`),
		IsSystem:    true,
	},
	{
		Name:        "repos:admin",
		Description: "Full repository administration",
		Category:    "repositories",
		Resources:   datatypes.JSON(`["repositories", "commits", "branches", "hooks", "settings"]`),
		Actions:     datatypes.JSON(`["create", "read", "update", "delete", "admin"]`),
		IsSystem:    true,
	},
	{
		Name:        "cicd:read",
		Description: "Read CI/CD pipeline information",
		Category:    "cicd",
		Resources:   datatypes.JSON(`["pipelines", "builds", "deployments"]`),
		Actions:     datatypes.JSON(`["read", "list"]`),
		IsSystem:    true,
	},
	{
		Name:        "cicd:trigger",
		Description: "Trigger CI/CD pipelines",
		Category:    "cicd",
		Resources:   datatypes.JSON(`["pipelines", "builds"]`),
		Actions:     datatypes.JSON(`["trigger", "cancel"]`),
		IsSystem:    true,
	},
	{
		Name:        "cicd:admin",
		Description: "Full CI/CD administration",
		Category:    "cicd",
		Resources:   datatypes.JSON(`["pipelines", "builds", "deployments", "runners"]`),
		Actions:     datatypes.JSON(`["create", "read", "update", "delete", "manage"]`),
		IsSystem:    true,
	},
	{
		Name:        "notifications:read",
		Description: "Read notifications",
		Category:    "notifications",
		Resources:   datatypes.JSON(`["notifications"]`),
		Actions:     datatypes.JSON(`["read", "list"]`),
		IsSystem:    true,
	},
	{
		Name:        "notifications:write",
		Description: "Send notifications",
		Category:    "notifications",
		Resources:   datatypes.JSON(`["notifications"]`),
		Actions:     datatypes.JSON(`["create", "send"]`),
		IsSystem:    true,
	},
	{
		Name:        "kb:read",
		Description: "Read knowledge base",
		Category:    "knowledge",
		Resources:   datatypes.JSON(`["documents", "pages", "attachments"]`),
		Actions:     datatypes.JSON(`["read", "search"]`),
		IsSystem:    true,
	},
	{
		Name:        "kb:write",
		Description: "Create and edit knowledge base",
		Category:    "knowledge",
		Resources:   datatypes.JSON(`["documents", "pages", "attachments"]`),
		Actions:     datatypes.JSON(`["create", "update", "upload"]`),
		IsSystem:    true,
	},
	{
		Name:        "admin:read",
		Description: "Read administrative information",
		Category:    "admin",
		Resources:   datatypes.JSON(`["tenants", "settings", "logs", "metrics"]`),
		Actions:     datatypes.JSON(`["read", "list"]`),
		IsSystem:    true,
	},
	{
		Name:        "admin:write",
		Description: "Administrative actions",
		Category:    "admin",
		Resources:   datatypes.JSON(`["tenants", "settings", "users", "permissions"]`),
		Actions:     datatypes.JSON(`["create", "update", "manage"]`),
		IsSystem:    true,
	},
}

// TableName methods for custom table names
func (APIToken) TableName() string {
	return "api_tokens"
}

func (APITokenUsage) TableName() string {
	return "api_token_usage"
}

func (APIScope) TableName() string {
	return "api_scopes"
}

func (APITokenScope) TableName() string {
	return "api_token_scopes"
}

func (APIRateLimit) TableName() string {
	return "api_rate_limits"
}
