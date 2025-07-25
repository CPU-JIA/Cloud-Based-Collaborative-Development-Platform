package models

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// FilePermission 文件权限模型
type FilePermission struct {
	ID         int       `json:"id" gorm:"primary_key"`
	TenantID   string    `json:"tenant_id" gorm:"not null;index"`
	FileID     *int      `json:"file_id" gorm:"index"`                    // 文件ID，为空表示文件夹权限
	FolderID   *int      `json:"folder_id" gorm:"index"`                  // 文件夹ID，为空表示文件权限
	UserID     *int      `json:"user_id" gorm:"index"`                    // 用户ID，为空表示角色权限
	RoleID     *int      `json:"role_id" gorm:"index"`                    // 角色ID，为空表示用户权限
	Permission string    `json:"permission" gorm:"not null"`              // 权限类型：read, write, delete, share, admin
	GrantedBy  int       `json:"granted_by" gorm:"not null"`              // 授权者ID
	ExpiresAt  *time.Time `json:"expires_at"`                             // 过期时间，为空表示永不过期
	IsActive   bool      `json:"is_active" gorm:"default:true"`           // 是否启用
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ShareLink 共享链接模型
type ShareLink struct {
	ID           int       `json:"id" gorm:"primary_key"`
	TenantID     string    `json:"tenant_id" gorm:"not null;index"`
	FileID       *int      `json:"file_id" gorm:"index"`                   // 文件ID
	FolderID     *int      `json:"folder_id" gorm:"index"`                 // 文件夹ID
	ShareToken   string    `json:"share_token" gorm:"unique;not null"`     // 分享令牌
	Password     string    `json:"password"`                               // 访问密码，可选
	Permission   string    `json:"permission" gorm:"not null;default:'read'"` // 分享权限：read, write
	ExpiresAt    *time.Time `json:"expires_at"`                             // 过期时间
	MaxDownloads *int      `json:"max_downloads"`                          // 最大下载次数
	Downloads    int       `json:"downloads" gorm:"default:0"`             // 当前下载次数
	IsActive     bool      `json:"is_active" gorm:"default:true"`          // 是否启用
	CreatedBy    int       `json:"created_by" gorm:"not null"`             // 创建者ID
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	
	// 关联字段
	File   *File   `json:"file,omitempty" gorm:"foreignKey:FileID"`
	Folder *Folder `json:"folder,omitempty" gorm:"foreignKey:FolderID"`
}

// AccessLog 访问日志模型
type AccessLog struct {
	ID          int       `json:"id" gorm:"primary_key"`
	TenantID    string    `json:"tenant_id" gorm:"not null;index"`
	FileID      *int      `json:"file_id" gorm:"index"`
	FolderID    *int      `json:"folder_id" gorm:"index"`
	ShareToken  *string   `json:"share_token" gorm:"index"`               // 分享令牌访问
	UserID      *int      `json:"user_id" gorm:"index"`                   // 用户ID，匿名访问时为空
	Action      string    `json:"action" gorm:"not null"`                 // 操作类型：view, download, upload, delete
	IPAddress   string    `json:"ip_address"`                             // 访问IP
	UserAgent   string    `json:"user_agent"`                             // 用户代理
	Success     bool      `json:"success" gorm:"default:true"`            // 是否成功
	ErrorReason string    `json:"error_reason"`                           // 失败原因
	CreatedAt   time.Time `json:"created_at"`
}

// Role 角色模型
type Role struct {
	ID          int       `json:"id" gorm:"primary_key"`
	TenantID    string    `json:"tenant_id" gorm:"not null;index"`
	ProjectID   int       `json:"project_id" gorm:"not null;index"`
	Name        string    `json:"name" gorm:"not null"`                   // 角色名称
	Description string    `json:"description"`                            // 角色描述
	Permissions []string  `json:"permissions" gorm:"-"`                   // 角色权限列表
	PermissionsData string `json:"-" gorm:"column:permissions"`           // 权限数据存储
	IsSystem    bool      `json:"is_system" gorm:"default:false"`         // 是否系统角色
	CreatedBy   int       `json:"created_by" gorm:"not null"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserRole 用户角色关联模型
type UserRole struct {
	ID        int       `json:"id" gorm:"primary_key"`
	TenantID  string    `json:"tenant_id" gorm:"not null;index"`
	UserID    int       `json:"user_id" gorm:"not null;index"`
	RoleID    int       `json:"role_id" gorm:"not null;index"`
	ProjectID int       `json:"project_id" gorm:"not null;index"`
	GrantedBy int       `json:"granted_by" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
}

// 权限常量
const (
	PermissionRead   = "read"
	PermissionWrite  = "write"
	PermissionDelete = "delete"
	PermissionShare  = "share"
	PermissionAdmin  = "admin"
)

// 操作类型常量
const (
	ActionView     = "view"
	ActionDownload = "download"
	ActionUpload   = "upload"
	ActionDelete   = "delete"
	ActionShare    = "share"
	ActionEdit     = "edit"
)

// 系统角色常量
const (
	RoleOwner      = "owner"       // 项目所有者
	RoleAdmin      = "admin"       // 项目管理员
	RoleMember     = "member"      // 项目成员
	RoleViewer     = "viewer"      // 只读访问者
	RoleGuest      = "guest"       // 访客
)

// HasPermission 检查是否有指定权限
func (fp *FilePermission) HasPermission(action string) bool {
	// 管理员权限包含所有权限
	if fp.Permission == PermissionAdmin {
		return true
	}
	
	// 检查具体权限
	switch action {
	case ActionView, ActionDownload:
		return fp.Permission == PermissionRead || fp.Permission == PermissionWrite || fp.Permission == PermissionAdmin
	case ActionUpload, ActionEdit:
		return fp.Permission == PermissionWrite || fp.Permission == PermissionAdmin
	case ActionDelete:
		return fp.Permission == PermissionDelete || fp.Permission == PermissionAdmin
	case ActionShare:
		return fp.Permission == PermissionShare || fp.Permission == PermissionAdmin
	default:
		return false
	}
}

// IsExpired 检查权限是否过期
func (fp *FilePermission) IsExpired() bool {
	if fp.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*fp.ExpiresAt)
}

// IsValid 检查权限是否有效
func (fp *FilePermission) IsValid() bool {
	return fp.IsActive && !fp.IsExpired()
}

// IsExpired 检查分享链接是否过期
func (sl *ShareLink) IsExpired() bool {
	if sl.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*sl.ExpiresAt)
}

// IsDownloadLimitReached 检查是否达到下载限制
func (sl *ShareLink) IsDownloadLimitReached() bool {
	if sl.MaxDownloads == nil {
		return false
	}
	return sl.Downloads >= *sl.MaxDownloads
}

// IsValid 检查分享链接是否有效
func (sl *ShareLink) IsValid() bool {
	return sl.IsActive && !sl.IsExpired() && !sl.IsDownloadLimitReached()
}

// GetDefaultRolePermissions 获取角色默认权限
func GetDefaultRolePermissions(roleName string) []string {
	switch roleName {
	case RoleOwner:
		return []string{PermissionRead, PermissionWrite, PermissionDelete, PermissionShare, PermissionAdmin}
	case RoleAdmin:
		return []string{PermissionRead, PermissionWrite, PermissionDelete, PermissionShare}
	case RoleMember:
		return []string{PermissionRead, PermissionWrite, PermissionShare}
	case RoleViewer:
		return []string{PermissionRead}
	case RoleGuest:
		return []string{PermissionRead}
	default:
		return []string{PermissionRead}
	}
}

// BeforeCreate GORM钩子：创建前序列化权限
func (r *Role) BeforeCreate(tx *gorm.DB) error {
	if len(r.Permissions) > 0 {
		r.PermissionsData = strings.Join(r.Permissions, ",")
	}
	return nil
}

// BeforeUpdate GORM钩子：更新前序列化权限
func (r *Role) BeforeUpdate(tx *gorm.DB) error {
	if len(r.Permissions) > 0 {
		r.PermissionsData = strings.Join(r.Permissions, ",")
	}
	return nil
}

// AfterFind GORM钩子：查询后反序列化权限
func (r *Role) AfterFind(tx *gorm.DB) error {
	if r.PermissionsData != "" {
		r.Permissions = strings.Split(r.PermissionsData, ",")
	}
	return nil
}