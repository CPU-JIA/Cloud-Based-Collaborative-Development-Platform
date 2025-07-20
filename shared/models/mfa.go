package models

import (
	"time"

	"github.com/google/uuid"
)

// UserMFABackupCode 用户MFA备用验证码
type UserMFABackupCode struct {
	ID        uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID    uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`
	TenantID  uuid.UUID  `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Code      string     `json:"-" gorm:"type:varchar(100);not null"`
	Used      bool       `json:"used" gorm:"default:false"`
	UsedAt    *time.Time `json:"used_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`

	// 关联
	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TableName 表名
func (UserMFABackupCode) TableName() string {
	return "user_mfa_backup_codes"
}

// UserMFADevice 用户MFA设备记录
type UserMFADevice struct {
	ID         uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID     uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`
	TenantID   uuid.UUID  `json:"tenant_id" gorm:"type:uuid;not null;index"`
	DeviceName string     `json:"device_name" gorm:"type:varchar(100);not null"`
	DeviceType string     `json:"device_type" gorm:"type:varchar(50);not null"` // totp, sms, email
	Secret     string     `json:"-" gorm:"type:varchar(255)"`                   // TOTP密钥
	Phone      string     `json:"phone,omitempty" gorm:"type:varchar(20)"`      // SMS设备的手机号
	Email      string     `json:"email,omitempty" gorm:"type:varchar(255)"`     // Email设备的邮箱
	IsActive   bool       `json:"is_active" gorm:"default:true"`
	IsPrimary  bool       `json:"is_primary" gorm:"default:false"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	// 关联
	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TableName 表名
func (UserMFADevice) TableName() string {
	return "user_mfa_devices"
}

// MFAAttempt MFA认证尝试记录
type MFAAttempt struct {
	ID         uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID     uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`
	TenantID   uuid.UUID  `json:"tenant_id" gorm:"type:uuid;not null;index"`
	IPAddress  string     `json:"ip_address" gorm:"type:varchar(45);not null"`
	UserAgent  string     `json:"user_agent" gorm:"type:text"`
	DeviceID   *uuid.UUID `json:"device_id" gorm:"type:uuid"`
	Success    bool       `json:"success" gorm:"default:false"`
	FailReason string     `json:"fail_reason,omitempty" gorm:"type:varchar(255)"`
	CreatedAt  time.Time  `json:"created_at"`

	// 关联
	User   User           `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Device *UserMFADevice `json:"device,omitempty" gorm:"foreignKey:DeviceID"`
}

// TableName 表名
func (MFAAttempt) TableName() string {
	return "mfa_attempts"
}
