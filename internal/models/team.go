package models

import (
	"time"
)

// Team 团队模型
type Team struct {
	ID          int       `json:"id" gorm:"primary_key"`
	TenantID    string    `json:"tenant_id" gorm:"not null;index"`
	ProjectID   int       `json:"project_id" gorm:"not null;index"`
	Name        string    `json:"name" gorm:"not null"`
	Description string    `json:"description"`
	Avatar      string    `json:"avatar"`
	Settings    string    `json:"settings" gorm:"type:json"`       // 团队设置JSON
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	CreatedBy   int       `json:"created_by" gorm:"not null"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	
	// 关联字段
	Members []TeamMember `json:"members,omitempty" gorm:"foreignKey:TeamID"`
	Roles   []Role       `json:"roles,omitempty" gorm:"foreignKey:ProjectID"`
}

// TeamMember 团队成员模型
type TeamMember struct {
	ID        int       `json:"id" gorm:"primary_key"`
	TenantID  string    `json:"tenant_id" gorm:"not null;index"`
	TeamID    int       `json:"team_id" gorm:"not null;index"`
	UserID    int       `json:"user_id" gorm:"not null;index"`
	RoleID    int       `json:"role_id" gorm:"not null;index"`
	Status    string    `json:"status" gorm:"not null;default:'active'"` // active, inactive, pending, rejected
	JoinedAt  time.Time `json:"joined_at"`
	InvitedBy int       `json:"invited_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	
	// 关联字段
	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Role Role `json:"role,omitempty" gorm:"foreignKey:RoleID"`
}

// TeamInvitation 团队邀请模型
type TeamInvitation struct {
	ID          int       `json:"id" gorm:"primary_key"`
	TenantID    string    `json:"tenant_id" gorm:"not null;index"`
	TeamID      int       `json:"team_id" gorm:"not null;index"`
	ProjectID   int       `json:"project_id" gorm:"not null;index"`
	Email       string    `json:"email" gorm:"not null"`
	RoleID      int       `json:"role_id" gorm:"not null;index"`
	Token       string    `json:"token" gorm:"unique;not null"`
	Status      string    `json:"status" gorm:"not null;default:'pending'"` // pending, accepted, rejected, expired
	ExpiresAt   time.Time `json:"expires_at"`
	Message     string    `json:"message"`
	InvitedBy   int       `json:"invited_by" gorm:"not null"`
	AcceptedBy  *int      `json:"accepted_by"`
	AcceptedAt  *time.Time `json:"accepted_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	
	// 关联字段
	Team      Team `json:"team,omitempty" gorm:"foreignKey:TeamID"`
	Role      Role `json:"role,omitempty" gorm:"foreignKey:RoleID"`
	InvitedUser User `json:"invited_user,omitempty" gorm:"foreignKey:InvitedBy"`
}

// PermissionRequest 权限申请模型
type PermissionRequest struct {
	ID           int       `json:"id" gorm:"primary_key"`
	TenantID     string    `json:"tenant_id" gorm:"not null;index"`
	ProjectID    int       `json:"project_id" gorm:"not null;index"`
	UserID       int       `json:"user_id" gorm:"not null;index"`
	RequestType  string    `json:"request_type" gorm:"not null"`              // role, file_permission, folder_permission
	TargetID     *int      `json:"target_id"`                                 // 目标ID（角色ID、文件ID、文件夹ID等）
	Permission   string    `json:"permission" gorm:"not null"`                // 申请的权限
	Reason       string    `json:"reason"`                                     // 申请理由
	Status       string    `json:"status" gorm:"not null;default:'pending'"` // pending, approved, rejected
	ReviewedBy   *int      `json:"reviewed_by"`                               // 审批人ID
	ReviewedAt   *time.Time `json:"reviewed_at"`                              // 审批时间
	ReviewReason string    `json:"review_reason"`                             // 审批意见
	ExpiresAt    *time.Time `json:"expires_at"`                               // 权限过期时间
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	
	// 关联字段
	User      User  `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Reviewer  *User `json:"reviewer,omitempty" gorm:"foreignKey:ReviewedBy"`
}

// User 用户模型（扩展现有用户模型）
type User struct {
	ID           int       `json:"id" gorm:"primary_key"`
	TenantID     string    `json:"tenant_id" gorm:"not null;index"`
	Username     string    `json:"username" gorm:"unique;not null"`
	Email        string    `json:"email" gorm:"unique;not null"`
	DisplayName  string    `json:"display_name"`
	Avatar       string    `json:"avatar"`
	Phone        string    `json:"phone"`
	Department   string    `json:"department"`
	Position     string    `json:"position"`
	Status       string    `json:"status" gorm:"default:'active'"` // active, inactive, suspended
	LastLoginAt  *time.Time `json:"last_login_at"`
	PasswordHash string    `json:"password_hash" gorm:"not null"`
	IsSystem     bool      `json:"is_system" gorm:"default:false"`
	Settings     string    `json:"settings" gorm:"type:json"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	
	// 关联字段
	TeamMemberships []TeamMember        `json:"team_memberships,omitempty" gorm:"foreignKey:UserID"`
	Roles           []Role              `json:"roles,omitempty" gorm:"many2many:user_roles"`
	Permissions     []FilePermission    `json:"permissions,omitempty" gorm:"foreignKey:UserID"`
	ShareLinks      []ShareLink         `json:"share_links,omitempty" gorm:"foreignKey:CreatedBy"`
}

// TeamActivity 团队活动模型
type TeamActivity struct {
	ID          int       `json:"id" gorm:"primary_key"`
	TenantID    string    `json:"tenant_id" gorm:"not null;index"`
	TeamID      int       `json:"team_id" gorm:"not null;index"`
	UserID      int       `json:"user_id" gorm:"not null;index"`
	Action      string    `json:"action" gorm:"not null"`      // join, leave, role_change, permission_grant, etc.
	TargetType  string    `json:"target_type"`                 // user, role, file, folder
	TargetID    *int      `json:"target_id"`
	Details     string    `json:"details" gorm:"type:json"`    // 活动详情JSON
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	CreatedAt   time.Time `json:"created_at"`
	
	// 关联字段
	Team Team `json:"team,omitempty" gorm:"foreignKey:TeamID"`
	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// 团队成员状态常量
const (
	MemberStatusActive   = "active"
	MemberStatusInactive = "inactive"
	MemberStatusPending  = "pending"
	MemberStatusRejected = "rejected"
)

// 邀请状态常量
const (
	InvitationStatusPending  = "pending"
	InvitationStatusAccepted = "accepted"
	InvitationStatusRejected = "rejected"
	InvitationStatusExpired  = "expired"
)

// 权限申请状态常量
const (
	RequestStatusPending  = "pending"
	RequestStatusApproved = "approved"
	RequestStatusRejected = "rejected"
)

// 权限申请类型常量
const (
	RequestTypeRole           = "role"
	RequestTypeFilePermission = "file_permission"
	RequestTypeFolderPermission = "folder_permission"
)

// 团队活动类型常量
const (
	ActivityJoin          = "join"
	ActivityLeave         = "leave"
	ActivityRoleChange    = "role_change"
	ActivityPermissionGrant = "permission_grant"
	ActivityPermissionRevoke = "permission_revoke"
	ActivityFileShare     = "file_share"
	ActivityFileUpload    = "file_upload"
	ActivityFileDownload  = "file_download"
)

// 用户状态常量
const (
	UserStatusActive    = "active"
	UserStatusInactive  = "inactive"
	UserStatusSuspended = "suspended"
)

// 注意：角色名称和权限常量已在permission.go中定义

// 注意：GetDefaultRolePermissions方法已在permission.go中定义

// IsActive 检查团队成员是否活跃
func (tm *TeamMember) IsActive() bool {
	return tm.Status == MemberStatusActive
}

// IsExpired 检查邀请是否过期
func (ti *TeamInvitation) IsExpired() bool {
	return time.Now().After(ti.ExpiresAt)
}

// IsValid 检查邀请是否有效
func (ti *TeamInvitation) IsValid() bool {
	return ti.Status == InvitationStatusPending && !ti.IsExpired()
}

// CanApprove 检查用户是否可以审批权限申请
func (pr *PermissionRequest) CanApprove(userID int, userRoles []Role) bool {
	if pr.Status != RequestStatusPending {
		return false
	}
	
	// 检查用户是否有管理权限
	for _, role := range userRoles {
		for _, permission := range role.Permissions {
			if permission == PermissionAdmin {
				return true
			}
		}
	}
	
	return false
}

// GetMemberCount 获取团队成员数量
func (t *Team) GetMemberCount() int {
	var count int
	for _, member := range t.Members {
		if member.IsActive() {
			count++
		}
	}
	return count
}

// GetRoleDistribution 获取角色分布
func (t *Team) GetRoleDistribution() map[string]int {
	distribution := make(map[string]int)
	for _, member := range t.Members {
		if member.IsActive() {
			distribution[member.Role.Name]++
		}
	}
	return distribution
}

// HasPermission 检查用户在团队中是否有指定权限
func (tm *TeamMember) HasPermission(permission string) bool {
	if !tm.IsActive() {
		return false
	}
	
	for _, perm := range tm.Role.Permissions {
		if perm == permission || perm == PermissionAdmin {
			return true
		}
	}
	
	return false
}

// GetTeamRole 获取用户在团队中的角色
func (u *User) GetTeamRole(teamID int) *Role {
	for _, membership := range u.TeamMemberships {
		if membership.TeamID == teamID && membership.IsActive() {
			return &membership.Role
		}
	}
	return nil
}

// IsTeamMember 检查用户是否是团队成员
func (u *User) IsTeamMember(teamID int) bool {
	for _, membership := range u.TeamMemberships {
		if membership.TeamID == teamID && membership.IsActive() {
			return true
		}
	}
	return false
}