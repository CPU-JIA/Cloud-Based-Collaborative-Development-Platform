package models

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// User 用户模型
type User struct {
	ID                 uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID           uuid.UUID  `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Email              string     `json:"email" gorm:"type:varchar(255);uniqueIndex;not null"`
	Username           string     `json:"username" gorm:"type:varchar(100);uniqueIndex;not null"`
	PasswordHash       string     `json:"-" gorm:"type:varchar(255);not null"`
	FirstName          string     `json:"first_name" gorm:"type:varchar(100)"`
	LastName           string     `json:"last_name" gorm:"type:varchar(100)"`
	Avatar             string     `json:"avatar" gorm:"type:text"`
	Phone              string     `json:"phone" gorm:"type:varchar(20)"`
	IsActive           bool       `json:"is_active" gorm:"default:true"`
	IsEmailVerified    bool       `json:"is_email_verified" gorm:"default:false"`
	EmailVerifiedAt    *time.Time `json:"email_verified_at"`
	LastLoginAt        *time.Time `json:"last_login_at"`
	FailedLoginCount   int        `json:"failed_login_count" gorm:"default:0"`
	LockedUntil        *time.Time `json:"locked_until"`
	TwoFactorEnabled   bool       `json:"two_factor_enabled" gorm:"default:false"`
	TwoFactorSecret    string     `json:"-" gorm:"type:varchar(255)"`
	PasswordResetAt    *time.Time `json:"password_reset_at"`
	PasswordChangedAt  *time.Time `json:"password_changed_at"`
	MustChangePassword bool       `json:"must_change_password" gorm:"default:false"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	DeletedAt          *time.Time `json:"deleted_at" gorm:"index"`

	// 关联
	Tenant    Tenant        `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
	UserRoles []UserRole    `json:"user_roles,omitempty" gorm:"foreignKey:UserID"`
	Roles     []Role        `json:"roles,omitempty" gorm:"many2many:user_roles;"`
	Sessions  []UserSession `json:"sessions,omitempty" gorm:"foreignKey:UserID"`
}

// TableName 表名
func (User) TableName() string {
	return "users"
}

// Role 角色模型
type Role struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID    uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Name        string    `json:"name" gorm:"type:varchar(100);not null"`
	DisplayName string    `json:"display_name" gorm:"type:varchar(200)"`
	Description string    `json:"description" gorm:"type:text"`
	IsSystem    bool      `json:"is_system" gorm:"default:false"` // 系统角色不可删除
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// 关联
	Tenant          Tenant           `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
	RolePermissions []RolePermission `json:"role_permissions,omitempty" gorm:"foreignKey:RoleID"`
	Permissions     []Permission     `json:"permissions,omitempty" gorm:"many2many:role_permissions;"`
	UserRoles       []UserRole       `json:"user_roles,omitempty" gorm:"foreignKey:RoleID"`
}

// TableName 表名
func (Role) TableName() string {
	return "roles"
}

// Permission 权限模型
type Permission struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string    `json:"name" gorm:"type:varchar(100);uniqueIndex;not null"`
	DisplayName string    `json:"display_name" gorm:"type:varchar(200)"`
	Description string    `json:"description" gorm:"type:text"`
	Resource    string    `json:"resource" gorm:"type:varchar(100);not null"` // 资源名称
	Action      string    `json:"action" gorm:"type:varchar(50);not null"`    // 操作名称
	IsSystem    bool      `json:"is_system" gorm:"default:true"`              // 系统权限不可删除
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// 关联
	RolePermissions []RolePermission `json:"role_permissions,omitempty" gorm:"foreignKey:PermissionID"`
}

// TableName 表名
func (Permission) TableName() string {
	return "permissions"
}

// UserRole 用户角色关联模型
type UserRole struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	RoleID    uuid.UUID `json:"role_id" gorm:"type:uuid;not null;index"`
	TenantID  uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`
	CreatedAt time.Time `json:"created_at"`

	// 关联
	User   User   `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Role   Role   `json:"role,omitempty" gorm:"foreignKey:RoleID"`
	Tenant Tenant `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
}

// TableName 表名
func (UserRole) TableName() string {
	return "user_roles"
}

// RolePermission 角色权限关联模型
type RolePermission struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	RoleID       uuid.UUID `json:"role_id" gorm:"type:uuid;not null;index"`
	PermissionID uuid.UUID `json:"permission_id" gorm:"type:uuid;not null;index"`
	CreatedAt    time.Time `json:"created_at"`

	// 关联
	Role       Role       `json:"role,omitempty" gorm:"foreignKey:RoleID"`
	Permission Permission `json:"permission,omitempty" gorm:"foreignKey:PermissionID"`
}

// TableName 表名
func (RolePermission) TableName() string {
	return "role_permissions"
}

// UserSession 用户会话模型
type UserSession struct {
	ID                uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID            uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`
	TenantID          uuid.UUID  `json:"tenant_id" gorm:"type:uuid;not null;index"`
	SessionToken      string     `json:"-" gorm:"type:varchar(255);index;not null"` // 会话令牌哈希
	RefreshToken      string     `json:"-" gorm:"type:varchar(255);index"`          // 刷新令牌哈希
	UserAgent         string     `json:"user_agent" gorm:"type:text"`
	IPAddress         string     `json:"ip_address" gorm:"type:varchar(45)"`
	DeviceInfo        string     `json:"device_info" gorm:"type:text"`
	DeviceFingerprint string     `json:"device_fingerprint" gorm:"type:varchar(255);index"`
	IsActive          bool       `json:"is_active" gorm:"default:true;index"`
	LastActivity      time.Time  `json:"last_activity" gorm:"index"`
	ExpiresAt         time.Time  `json:"expires_at" gorm:"index"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	RevokedAt         *time.Time `json:"revoked_at" gorm:"index"`
	RevokeReason      string     `json:"revoke_reason,omitempty" gorm:"type:varchar(100)"`

	// 关联
	User   User   `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Tenant Tenant `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
}

// TableName 表名
func (UserSession) TableName() string {
	return "user_sessions"
}

// 注意：Tenant模型的完整定义在tenant.go中

// 用户方法

// SetPassword 设置密码（加密）
func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hashedPassword)
	return nil
}

// CheckPassword 验证密码
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// FullName 获取全名
func (u *User) FullName() string {
	if u.FirstName == "" && u.LastName == "" {
		return u.Username
	}
	return u.FirstName + " " + u.LastName
}

// IsLocked 检查用户是否被锁定
func (u *User) IsLocked() bool {
	return u.LockedUntil != nil && u.LockedUntil.After(time.Now())
}

// Lock 锁定用户
func (u *User) Lock(duration time.Duration) {
	lockUntil := time.Now().Add(duration)
	u.LockedUntil = &lockUntil
}

// Unlock 解锁用户
func (u *User) Unlock() {
	u.LockedUntil = nil
	u.FailedLoginCount = 0
}

// IncrementFailedLogin 增加失败登录次数
func (u *User) IncrementFailedLogin() {
	u.FailedLoginCount++
}

// ResetFailedLogin 重置失败登录次数
func (u *User) ResetFailedLogin() {
	u.FailedLoginCount = 0
}

// MarkEmailVerified 标记邮箱已验证
func (u *User) MarkEmailVerified() {
	u.IsEmailVerified = true
	now := time.Now()
	u.EmailVerifiedAt = &now
}

// UpdateLastLogin 更新最后登录时间
func (u *User) UpdateLastLogin() {
	now := time.Now()
	u.LastLoginAt = &now
	u.ResetFailedLogin()
}

// 权限检查方法

// HasRole 检查用户是否具有指定角色
func (u *User) HasRole(roleName string) bool {
	for _, role := range u.Roles {
		if role.Name == roleName {
			return true
		}
	}
	return false
}

// GetPermissions 获取用户所有权限
func (u *User) GetPermissions() []string {
	var permissions []string
	permissionSet := make(map[string]bool)

	for _, role := range u.Roles {
		for _, permission := range role.Permissions {
			key := permission.Resource + ":" + permission.Action
			if !permissionSet[key] {
				permissions = append(permissions, key)
				permissionSet[key] = true
			}
		}
	}

	return permissions
}

// HasPermission 检查用户是否具有指定权限
func (u *User) HasPermission(resource, action string) bool {
	for _, role := range u.Roles {
		for _, permission := range role.Permissions {
			if permission.Resource == resource && permission.Action == action {
				return true
			}
			// 检查通配符权限
			if permission.Resource == "*" || permission.Action == "*" {
				return true
			}
		}
	}
	return false
}

// ToPublicUser 转换为公开用户信息（不包含敏感信息）
func (u *User) ToPublicUser() map[string]interface{} {
	return map[string]interface{}{
		"id":                u.ID,
		"tenant_id":         u.TenantID,
		"email":             u.Email,
		"username":          u.Username,
		"first_name":        u.FirstName,
		"last_name":         u.LastName,
		"avatar":            u.Avatar,
		"is_active":         u.IsActive,
		"is_email_verified": u.IsEmailVerified,
		"last_login_at":     u.LastLoginAt,
		"created_at":        u.CreatedAt,
		"updated_at":        u.UpdatedAt,
	}
}
