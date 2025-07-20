package models

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken 刷新令牌模型
type RefreshToken struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID       uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`
	TenantID     uuid.UUID  `json:"tenant_id" gorm:"type:uuid;not null;index"`
	TokenHash    string     `json:"-" gorm:"type:varchar(255);uniqueIndex;not null"`   // 刷新令牌哈希
	JTI          string     `json:"jti" gorm:"type:varchar(255);uniqueIndex;not null"` // JWT ID，用于令牌撤销
	IsActive     bool       `json:"is_active" gorm:"default:true;index"`
	ExpiresAt    time.Time  `json:"expires_at" gorm:"index;not null"`
	LastUsedAt   *time.Time `json:"last_used_at" gorm:"index"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	RevokedAt    *time.Time `json:"revoked_at" gorm:"index"`
	RevokeReason string     `json:"revoke_reason,omitempty" gorm:"type:varchar(100)"`

	// 客户端信息
	UserAgent         string `json:"user_agent" gorm:"type:text"`
	IPAddress         string `json:"ip_address" gorm:"type:varchar(45)"`
	DeviceFingerprint string `json:"device_fingerprint" gorm:"type:varchar(255);index"`

	// 关联
	User   User   `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Tenant Tenant `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
}

// TableName 表名
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// IsExpired 检查令牌是否过期
func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

// IsRevoked 检查令牌是否被撤销
func (rt *RefreshToken) IsRevoked() bool {
	return rt.RevokedAt != nil
}

// IsValid 检查令牌是否有效（未过期且未撤销且激活）
func (rt *RefreshToken) IsValid() bool {
	return rt.IsActive && !rt.IsExpired() && !rt.IsRevoked()
}

// Revoke 撤销令牌
func (rt *RefreshToken) Revoke(reason string) {
	now := time.Now()
	rt.RevokedAt = &now
	rt.RevokeReason = reason
	rt.IsActive = false
}

// UpdateLastUsed 更新最后使用时间
func (rt *RefreshToken) UpdateLastUsed() {
	now := time.Now()
	rt.LastUsedAt = &now
}
