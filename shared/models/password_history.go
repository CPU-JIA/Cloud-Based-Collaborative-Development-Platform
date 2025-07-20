package models

import (
	"time"

	"github.com/google/uuid"
)

// UserPasswordHistory 用户密码历史记录
type UserPasswordHistory struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID       uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	TenantID     uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`
	PasswordHash string    `json:"-" gorm:"type:varchar(255);not null"`
	CreatedAt    time.Time `json:"created_at"`

	// 关联
	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TableName 表名
func (UserPasswordHistory) TableName() string {
	return "user_password_history"
}

// SecurityAuditLog 安全审计日志
type SecurityAuditLog struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID    uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`
	UserID      *uuid.UUID `json:"user_id" gorm:"type:uuid;index"`
	IPAddress   string    `json:"ip_address" gorm:"type:varchar(45);not null"`
	UserAgent   string    `json:"user_agent" gorm:"type:text"`
	Action      string    `json:"action" gorm:"type:varchar(100);not null;index"`
	Resource    string    `json:"resource" gorm:"type:varchar(255)"`
	Details     string    `json:"details" gorm:"type:text"`
	Success     bool      `json:"success" gorm:"default:true;index"`
	ErrorCode   string    `json:"error_code,omitempty" gorm:"type:varchar(50)"`
	ErrorMessage string   `json:"error_message,omitempty" gorm:"type:text"`
	Metadata    string    `json:"metadata,omitempty" gorm:"type:jsonb"`
	CreatedAt   time.Time `json:"created_at"`

	// 关联
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TableName 表名
func (SecurityAuditLog) TableName() string {
	return "security_audit_logs"
}

// LoginAttempt 登录尝试记录
type LoginAttempt struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID    uuid.UUID  `json:"tenant_id" gorm:"type:uuid;not null;index"`
	UserID      *uuid.UUID `json:"user_id" gorm:"type:uuid;index"`
	Email       string     `json:"email" gorm:"type:varchar(255);not null;index"`
	IPAddress   string     `json:"ip_address" gorm:"type:varchar(45);not null;index"`
	UserAgent   string     `json:"user_agent" gorm:"type:text"`
	Success     bool       `json:"success" gorm:"default:false;index"`
	FailReason  string     `json:"fail_reason,omitempty" gorm:"type:varchar(255)"`
	MFARequired bool       `json:"mfa_required" gorm:"default:false"`
	MFASuccess  *bool      `json:"mfa_success,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`

	// 关联
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TableName 表名
func (LoginAttempt) TableName() string {
	return "login_attempts"
}

// PasswordPolicy 租户密码策略配置
type PasswordPolicy struct {
	ID                       uuid.UUID     `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID                 uuid.UUID     `json:"tenant_id" gorm:"type:uuid;not null;unique"`
	MinLength                int           `json:"min_length" gorm:"default:8"`
	MaxLength                int           `json:"max_length" gorm:"default:128"`
	RequireUppercase         bool          `json:"require_uppercase" gorm:"default:true"`
	RequireLowercase         bool          `json:"require_lowercase" gorm:"default:true"`
	RequireNumbers           bool          `json:"require_numbers" gorm:"default:true"`
	RequireSpecialChars      bool          `json:"require_special_chars" gorm:"default:true"`
	MinSpecialChars          int           `json:"min_special_chars" gorm:"default:1"`
	MaxConsecutiveChars      int           `json:"max_consecutive_chars" gorm:"default:3"`
	MaxRepeatingChars        int           `json:"max_repeating_chars" gorm:"default:3"`
	PreventUsernameInclusion bool          `json:"prevent_username_inclusion" gorm:"default:true"`
	PreventEmailInclusion    bool          `json:"prevent_email_inclusion" gorm:"default:true"`
	PasswordExpiryDays       int           `json:"password_expiry_days" gorm:"default:90"`
	PasswordHistoryCount     int           `json:"password_history_count" gorm:"default:5"`
	AccountLockoutThreshold  int           `json:"account_lockout_threshold" gorm:"default:5"`
	AccountLockoutMinutes    int           `json:"account_lockout_minutes" gorm:"default:30"`
	PasswordComplexityScore  int           `json:"password_complexity_score" gorm:"default:60"`
	CreatedAt                time.Time     `json:"created_at"`
	UpdatedAt                time.Time     `json:"updated_at"`

	// 关联
	Tenant Tenant `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
}

// TableName 表名
func (PasswordPolicy) TableName() string {
	return "password_policies"
}