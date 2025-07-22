package fixtures

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TestTenant SQLite兼容的测试租户模型
type TestTenant struct {
	ID           uuid.UUID `json:"id" gorm:"type:text;primary_key"`
	Name         string    `json:"name" gorm:"type:varchar(255);not null"`
	Domain       string    `json:"domain" gorm:"type:varchar(255);not null;uniqueIndex"`
	Status       string    `json:"status" gorm:"type:varchar(50);not null;default:'pending'"`
	Plan         string    `json:"plan" gorm:"type:varchar(50);not null;default:'free'"`
	BillingEmail string    `json:"billing_email" gorm:"type:varchar(255);not null"`
	Description  string    `json:"description" gorm:"type:text"`

	// 简化的联系信息
	ContactName  string `json:"contact_name" gorm:"type:varchar(255)"`
	ContactEmail string `json:"contact_email" gorm:"type:varchar(255)"`
	ContactPhone string `json:"contact_phone" gorm:"type:varchar(50)"`

	// 系统字段
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// TestUser SQLite兼容的测试用户模型
type TestUser struct {
	ID       uuid.UUID `json:"id" gorm:"type:text;primary_key"`
	TenantID uuid.UUID `json:"tenant_id" gorm:"type:text;not null"`
	Email    string    `json:"email" gorm:"type:varchar(255);uniqueIndex;not null"`
	Username string    `json:"username" gorm:"type:varchar(100);uniqueIndex;not null"`
	
	// 认证相关
	PasswordHash string `json:"-" gorm:"type:varchar(255);not null"`
	Salt         string `json:"-" gorm:"type:varchar(255);not null"`
	
	// 状态字段
	IsActive     bool `json:"is_active" gorm:"default:true"`
	IsVerified   bool `json:"is_verified" gorm:"default:false"`
	MFAEnabled   bool `json:"mfa_enabled" gorm:"default:false"`
	MFASecret    string `json:"-" gorm:"type:varchar(255)"`
	
	// 系统字段
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// TestRole 测试角色模型
type TestRole struct {
	ID          uuid.UUID `json:"id" gorm:"type:text;primary_key"`
	Name        string    `json:"name" gorm:"type:varchar(100);not null"`
	Description string    `json:"description" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TestPermission 测试权限模型
type TestPermission struct {
	ID          uuid.UUID `json:"id" gorm:"type:text;primary_key"`
	Name        string    `json:"name" gorm:"type:varchar(100);not null"`
	Resource    string    `json:"resource" gorm:"type:varchar(100);not null"`
	Action      string    `json:"action" gorm:"type:varchar(50);not null"`
	Description string    `json:"description" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TestRefreshToken 测试刷新令牌模型
type TestRefreshToken struct {
	ID        uuid.UUID  `json:"id" gorm:"type:text;primary_key"`
	UserID    uuid.UUID  `json:"user_id" gorm:"type:text;not null"`
	Token     string     `json:"token" gorm:"type:varchar(255);uniqueIndex;not null"`
	ExpiresAt time.Time  `json:"expires_at" gorm:"not null"`
	IsRevoked bool       `json:"is_revoked" gorm:"default:false"`
	CreatedAt time.Time  `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at"`
}

// BeforeCreate 钩子：创建前设置UUID
func (t *TestTenant) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

func (u *TestUser) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

func (r *TestRole) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

func (p *TestPermission) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

func (rt *TestRefreshToken) BeforeCreate(tx *gorm.DB) error {
	if rt.ID == uuid.Nil {
		rt.ID = uuid.New()
	}
	return nil
}

// 表名映射
func (TestTenant) TableName() string       { return "tenants" }
func (TestUser) TableName() string         { return "users" }
func (TestRole) TableName() string         { return "roles" }
func (TestPermission) TableName() string   { return "permissions" }
func (TestRefreshToken) TableName() string { return "refresh_tokens" }