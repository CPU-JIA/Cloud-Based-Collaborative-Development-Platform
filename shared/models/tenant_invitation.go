package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InvitationStatus 邀请状态枚举
type InvitationStatus string

const (
	InvitationStatusPending  InvitationStatus = "pending"
	InvitationStatusAccepted InvitationStatus = "accepted"
	InvitationStatusDeclined InvitationStatus = "declined"
	InvitationStatusExpired  InvitationStatus = "expired"
	InvitationStatusCanceled InvitationStatus = "canceled"
)

// InvitationRole 邀请角色类型
type InvitationRole string

const (
	InvitationRoleAdmin     InvitationRole = "admin"
	InvitationRoleMember    InvitationRole = "member"
	InvitationRoleGuest     InvitationRole = "guest"
	InvitationRoleDeveloper InvitationRole = "developer"
	InvitationRoleViewer    InvitationRole = "viewer"
)

// TenantInvitation 租户成员邀请
type TenantInvitation struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`

	// 邀请信息
	InviterID uuid.UUID        `json:"inviter_id" gorm:"type:uuid;not null;index"` // 邀请人ID
	Email     string           `json:"email" gorm:"type:varchar(255);not null;index"`
	Role      InvitationRole   `json:"role" gorm:"type:varchar(50);not null;default:'member'"`
	Status    InvitationStatus `json:"status" gorm:"type:varchar(50);not null;default:'pending';index"`

	// 邀请内容
	Message     string   `json:"message" gorm:"type:text"`      // 个人邀请消息
	Permissions []string `json:"permissions" gorm:"type:jsonb"` // 自定义权限列表

	// 令牌信息
	Token     string    `json:"-" gorm:"type:varchar(255);uniqueIndex"` // 邀请令牌（不返回给客户端）
	ExpiresAt time.Time `json:"expires_at" gorm:"not null;index"`       // 过期时间

	// 响应信息
	AcceptedAt *time.Time `json:"accepted_at,omitempty" gorm:"index"`
	DeclinedAt *time.Time `json:"declined_at,omitempty" gorm:"index"`
	CanceledAt *time.Time `json:"canceled_at,omitempty" gorm:"index"`
	AcceptedBy *uuid.UUID `json:"accepted_by,omitempty" gorm:"type:uuid"` // 接受邀请的用户ID（如果邮箱对应多个用户）

	// 元数据
	InviteMetadata map[string]interface{} `json:"invite_metadata,omitempty" gorm:"type:jsonb"`

	// 系统字段
	CreatedAt time.Time      `json:"created_at" gorm:"index"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	Tenant   Tenant `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
	Inviter  User   `json:"inviter,omitempty" gorm:"foreignKey:InviterID"`
	Accepter *User  `json:"accepter,omitempty" gorm:"foreignKey:AcceptedBy"`
}

// BeforeCreate GORM钩子：创建前设置默认值
func (i *TenantInvitation) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}

	// 设置默认过期时间（7天）
	if i.ExpiresAt.IsZero() {
		i.ExpiresAt = time.Now().AddDate(0, 0, 7)
	}

	return nil
}

// TableName 指定表名
func (TenantInvitation) TableName() string {
	return "tenant_invitations"
}

// 验证方法

// IsExpired 检查邀请是否过期
func (i *TenantInvitation) IsExpired() bool {
	return time.Now().After(i.ExpiresAt)
}

// IsPending 检查邀请是否待处理
func (i *TenantInvitation) IsPending() bool {
	return i.Status == InvitationStatusPending && !i.IsExpired()
}

// CanAccept 检查邀请是否可以接受
func (i *TenantInvitation) CanAccept() bool {
	return i.Status == InvitationStatusPending && !i.IsExpired()
}

// CanCancel 检查邀请是否可以取消
func (i *TenantInvitation) CanCancel() bool {
	return i.Status == InvitationStatusPending
}

// GetRolePermissions 获取角色对应的权限列表
func (i *TenantInvitation) GetRolePermissions() []string {
	if len(i.Permissions) > 0 {
		return i.Permissions
	}

	// 返回角色默认权限
	switch i.Role {
	case InvitationRoleAdmin:
		return []string{
			"tenant.manage",
			"user.manage",
			"project.manage",
			"settings.manage",
			"billing.manage",
		}
	case InvitationRoleDeveloper:
		return []string{
			"project.read",
			"project.write",
			"code.read",
			"code.write",
			"ci_cd.manage",
		}
	case InvitationRoleMember:
		return []string{
			"project.read",
			"project.write",
			"user.read",
		}
	case InvitationRoleViewer:
		return []string{
			"project.read",
			"user.read",
		}
	case InvitationRoleGuest:
		return []string{
			"project.read",
		}
	default:
		return []string{}
	}
}

// Accept 接受邀请
func (i *TenantInvitation) Accept(userID uuid.UUID) error {
	if !i.CanAccept() {
		return fmt.Errorf("邀请无法接受：状态为 %s 或已过期", i.Status)
	}

	now := time.Now()
	i.Status = InvitationStatusAccepted
	i.AcceptedAt = &now
	i.AcceptedBy = &userID

	return nil
}

// Decline 拒绝邀请
func (i *TenantInvitation) Decline() error {
	if i.Status != InvitationStatusPending {
		return fmt.Errorf("只能拒绝待处理的邀请")
	}

	now := time.Now()
	i.Status = InvitationStatusDeclined
	i.DeclinedAt = &now

	return nil
}

// Cancel 取消邀请
func (i *TenantInvitation) Cancel() error {
	if !i.CanCancel() {
		return fmt.Errorf("无法取消此邀请")
	}

	now := time.Now()
	i.Status = InvitationStatusCanceled
	i.CanceledAt = &now

	return nil
}

// MarkExpired 标记为过期
func (i *TenantInvitation) MarkExpired() {
	if i.Status == InvitationStatusPending && i.IsExpired() {
		i.Status = InvitationStatusExpired
	}
}

// TenantMember 租户成员关系表
type TenantMember struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`
	UserID   uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`

	// 成员信息
	Role        InvitationRole `json:"role" gorm:"type:varchar(50);not null;default:'member'"`
	Permissions []string       `json:"permissions" gorm:"type:jsonb"`                            // 自定义权限
	Status      string         `json:"status" gorm:"type:varchar(50);not null;default:'active'"` // active, suspended, inactive

	// 加入信息
	JoinedAt     time.Time  `json:"joined_at" gorm:"not null;default:now()"`
	InvitedBy    *uuid.UUID `json:"invited_by,omitempty" gorm:"type:uuid;index"`
	InvitationID *uuid.UUID `json:"invitation_id,omitempty" gorm:"type:uuid;index"`

	// 最后活动
	LastActiveAt *time.Time `json:"last_active_at,omitempty" gorm:"index"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty" gorm:"index"`

	// 元数据
	MemberMetadata map[string]interface{} `json:"member_metadata,omitempty" gorm:"type:jsonb"`

	// 系统字段
	CreatedAt time.Time      `json:"created_at" gorm:"index"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系（加载时使用）
	Tenant     Tenant            `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
	User       User              `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Inviter    *User             `json:"inviter,omitempty" gorm:"foreignKey:InvitedBy"`
	Invitation *TenantInvitation `json:"invitation,omitempty" gorm:"foreignKey:InvitationID"`
}

// TableName 指定表名
func (TenantMember) TableName() string {
	return "tenant_members"
}

// IsActive 检查成员是否活跃
func (m *TenantMember) IsActive() bool {
	return m.Status == "active"
}

// HasPermission 检查是否具有特定权限
func (m *TenantMember) HasPermission(permission string) bool {
	// 检查自定义权限
	for _, p := range m.Permissions {
		if p == permission {
			return true
		}
	}

	// 检查角色默认权限
	rolePermissions := (&TenantInvitation{Role: m.Role}).GetRolePermissions()
	for _, p := range rolePermissions {
		if p == permission {
			return true
		}
	}

	return false
}

// UpdateLastActive 更新最后活动时间
func (m *TenantMember) UpdateLastActive() {
	now := time.Now()
	m.LastActiveAt = &now
}

// UpdateLastLogin 更新最后登录时间
func (m *TenantMember) UpdateLastLogin() {
	now := time.Now()
	m.LastLoginAt = &now
}
