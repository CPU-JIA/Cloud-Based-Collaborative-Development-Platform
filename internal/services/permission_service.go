package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/cloud-platform/collaborative-dev/internal/models"
)

// PermissionService 权限服务
type PermissionService struct {
	db *gorm.DB
}

// NewPermissionService 创建权限服务实例
func NewPermissionService(db *gorm.DB) *PermissionService {
	return &PermissionService{db: db}
}

// CreateFilePermission 创建文件权限
func (ps *PermissionService) CreateFilePermission(tenantID string, fileID *int, folderID *int, userID *int, roleID *int, permission string, grantedBy int, expiresAt *time.Time) (*models.FilePermission, error) {
	// 验证参数
	if fileID == nil && folderID == nil {
		return nil, errors.New("文件ID或文件夹ID不能同时为空")
	}
	if userID == nil && roleID == nil {
		return nil, errors.New("用户ID或角色ID不能同时为空")
	}

	fp := &models.FilePermission{
		TenantID:   tenantID,
		FileID:     fileID,
		FolderID:   folderID,
		UserID:     userID,
		RoleID:     roleID,
		Permission: permission,
		GrantedBy:  grantedBy,
		ExpiresAt:  expiresAt,
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := ps.db.Create(fp).Error; err != nil {
		return nil, fmt.Errorf("创建权限失败: %w", err)
	}

	return fp, nil
}

// CheckFilePermission 检查文件权限
func (ps *PermissionService) CheckFilePermission(tenantID string, fileID int, userID int, action string) (bool, error) {
	// 检查直接文件权限
	var directPermission models.FilePermission
	err := ps.db.Where("tenant_id = ? AND file_id = ? AND user_id = ? AND is_active = ?",
		tenantID, fileID, userID, true).First(&directPermission).Error

	if err == nil && directPermission.IsValid() && directPermission.HasPermission(action) {
		return true, nil
	}

	// 检查角色权限
	userRoles, err := ps.GetUserRoles(tenantID, userID)
	if err != nil {
		return false, err
	}

	for _, role := range userRoles {
		var rolePermission models.FilePermission
		err := ps.db.Where("tenant_id = ? AND file_id = ? AND role_id = ? AND is_active = ?",
			tenantID, fileID, role.ID, true).First(&rolePermission).Error

		if err == nil && rolePermission.IsValid() && rolePermission.HasPermission(action) {
			return true, nil
		}
	}

	// 检查文件夹继承权限
	var file models.File
	if err := ps.db.Where("id = ? AND tenant_id = ?", fileID, tenantID).First(&file).Error; err != nil {
		return false, err
	}

	if file.FolderID != nil {
		return ps.CheckFolderPermission(tenantID, *file.FolderID, userID, action)
	}

	return false, nil
}

// CheckFolderPermission 检查文件夹权限
func (ps *PermissionService) CheckFolderPermission(tenantID string, folderID int, userID int, action string) (bool, error) {
	// 检查直接文件夹权限
	var directPermission models.FilePermission
	err := ps.db.Where("tenant_id = ? AND folder_id = ? AND user_id = ? AND is_active = ?",
		tenantID, folderID, userID, true).First(&directPermission).Error

	if err == nil && directPermission.IsValid() && directPermission.HasPermission(action) {
		return true, nil
	}

	// 检查角色权限
	userRoles, err := ps.GetUserRoles(tenantID, userID)
	if err != nil {
		return false, err
	}

	for _, role := range userRoles {
		var rolePermission models.FilePermission
		err := ps.db.Where("tenant_id = ? AND folder_id = ? AND role_id = ? AND is_active = ?",
			tenantID, folderID, role.ID, true).First(&rolePermission).Error

		if err == nil && rolePermission.IsValid() && rolePermission.HasPermission(action) {
			return true, nil
		}
	}

	// 检查父文件夹权限（递归）
	var folder models.Folder
	if err := ps.db.Where("id = ? AND tenant_id = ?", folderID, tenantID).First(&folder).Error; err != nil {
		return false, err
	}

	if folder.ParentID != nil {
		return ps.CheckFolderPermission(tenantID, *folder.ParentID, userID, action)
	}

	return false, nil
}

// CreateShareLink 创建分享链接
func (ps *PermissionService) CreateShareLink(tenantID string, fileID *int, folderID *int, createdBy int, permission string, password string, expiresAt *time.Time, maxDownloads *int) (*models.ShareLink, error) {
	// 验证参数
	if fileID == nil && folderID == nil {
		return nil, errors.New("文件ID或文件夹ID不能同时为空")
	}

	// 生成唯一的分享令牌
	shareToken, err := ps.generateShareToken()
	if err != nil {
		return nil, fmt.Errorf("生成分享令牌失败: %w", err)
	}

	sl := &models.ShareLink{
		TenantID:     tenantID,
		FileID:       fileID,
		FolderID:     folderID,
		ShareToken:   shareToken,
		Password:     password,
		Permission:   permission,
		ExpiresAt:    expiresAt,
		MaxDownloads: maxDownloads,
		Downloads:    0,
		IsActive:     true,
		CreatedBy:    createdBy,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := ps.db.Create(sl).Error; err != nil {
		return nil, fmt.Errorf("创建分享链接失败: %w", err)
	}

	return sl, nil
}

// GetShareLink 获取分享链接
func (ps *PermissionService) GetShareLink(shareToken string) (*models.ShareLink, error) {
	var shareLink models.ShareLink
	err := ps.db.Preload("File").Preload("Folder").Where("share_token = ?", shareToken).First(&shareLink).Error
	if err != nil {
		return nil, fmt.Errorf("分享链接不存在或已失效")
	}

	if !shareLink.IsValid() {
		return nil, fmt.Errorf("分享链接已过期或失效")
	}

	return &shareLink, nil
}

// ValidateShareAccess 验证分享访问权限
func (ps *PermissionService) ValidateShareAccess(shareToken string, password string) (*models.ShareLink, error) {
	shareLink, err := ps.GetShareLink(shareToken)
	if err != nil {
		return nil, err
	}

	// 检查密码
	if shareLink.Password != "" && shareLink.Password != password {
		return nil, errors.New("密码错误")
	}

	return shareLink, nil
}

// IncrementShareDownload 增加分享下载次数
func (ps *PermissionService) IncrementShareDownload(shareToken string) error {
	return ps.db.Model(&models.ShareLink{}).Where("share_token = ?", shareToken).
		Update("downloads", gorm.Expr("downloads + ?", 1)).Error
}

// RevokeShareLink 撤销分享链接
func (ps *PermissionService) RevokeShareLink(shareToken string, userID int) error {
	var shareLink models.ShareLink
	if err := ps.db.Where("share_token = ?", shareToken).First(&shareLink).Error; err != nil {
		return fmt.Errorf("分享链接不存在")
	}

	// 检查权限（只有创建者可以撤销）
	if shareLink.CreatedBy != userID {
		return fmt.Errorf("无权限撤销此分享链接")
	}

	return ps.db.Model(&shareLink).Update("is_active", false).Error
}

// ListUserShareLinks 获取用户的分享链接列表
func (ps *PermissionService) ListUserShareLinks(tenantID string, userID int, page, pageSize int) ([]*models.ShareLink, int64, error) {
	var shareLinks []*models.ShareLink
	var total int64

	query := ps.db.Where("tenant_id = ? AND created_by = ?", tenantID, userID)

	// 统计总数
	query.Model(&models.ShareLink{}).Count(&total)

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Preload("File").Preload("Folder").
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&shareLinks).Error

	return shareLinks, total, err
}

// GetUserRoles 获取用户角色
func (ps *PermissionService) GetUserRoles(tenantID string, userID int) ([]*models.Role, error) {
	var roles []*models.Role

	err := ps.db.Table("roles").
		Joins("JOIN user_roles ON roles.id = user_roles.role_id").
		Where("user_roles.tenant_id = ? AND user_roles.user_id = ?", tenantID, userID).
		Find(&roles).Error

	return roles, err
}

// LogAccess 记录访问日志
func (ps *PermissionService) LogAccess(tenantID string, fileID *int, folderID *int, shareToken *string, userID *int, action string, ipAddress string, userAgent string, success bool, errorReason string) error {
	log := &models.AccessLog{
		TenantID:    tenantID,
		FileID:      fileID,
		FolderID:    folderID,
		ShareToken:  shareToken,
		UserID:      userID,
		Action:      action,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		Success:     success,
		ErrorReason: errorReason,
		CreatedAt:   time.Now(),
	}

	return ps.db.Create(log).Error
}

// GetAccessLogs 获取访问日志
func (ps *PermissionService) GetAccessLogs(tenantID string, fileID *int, folderID *int, page, pageSize int) ([]*models.AccessLog, int64, error) {
	var logs []*models.AccessLog
	var total int64

	query := ps.db.Where("tenant_id = ?", tenantID)

	if fileID != nil {
		query = query.Where("file_id = ?", *fileID)
	}
	if folderID != nil {
		query = query.Where("folder_id = ?", *folderID)
	}

	// 统计总数
	query.Model(&models.AccessLog{}).Count(&total)

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&logs).Error

	return logs, total, err
}

// generateShareToken 生成分享令牌
func (ps *PermissionService) generateShareToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateRole 创建角色
func (ps *PermissionService) CreateRole(tenantID string, projectID int, name string, description string, permissions []string, createdBy int) (*models.Role, error) {
	role := &models.Role{
		TenantID:    tenantID,
		ProjectID:   projectID,
		Name:        name,
		Description: description,
		Permissions: permissions,
		IsSystem:    false,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := ps.db.Create(role).Error; err != nil {
		return nil, fmt.Errorf("创建角色失败: %w", err)
	}

	return role, nil
}

// AssignUserRole 分配用户角色
func (ps *PermissionService) AssignUserRole(tenantID string, userID int, roleID int, projectID int, grantedBy int) error {
	userRole := &models.UserRole{
		TenantID:  tenantID,
		UserID:    userID,
		RoleID:    roleID,
		ProjectID: projectID,
		GrantedBy: grantedBy,
		CreatedAt: time.Now(),
	}

	return ps.db.Create(userRole).Error
}

// RemoveUserRole 移除用户角色
func (ps *PermissionService) RemoveUserRole(tenantID string, userID int, roleID int, projectID int) error {
	return ps.db.Where("tenant_id = ? AND user_id = ? AND role_id = ? AND project_id = ?",
		tenantID, userID, roleID, projectID).Delete(&models.UserRole{}).Error
}

// CheckUserPermission 检查用户权限
func (ps *PermissionService) CheckUserPermission(userID int, projectID int, permission string) (bool, error) {
	// 查询用户在项目中的角色
	var roles []models.Role
	err := ps.db.Table("roles").
		Joins("JOIN team_members ON team_members.role_id = roles.id").
		Joins("JOIN teams ON team_members.team_id = teams.id").
		Where("team_members.user_id = ? AND teams.project_id = ? AND team_members.status = ?",
			userID, projectID, models.MemberStatusActive).
		Find(&roles).Error

	if err != nil {
		return false, err
	}

	// 检查角色权限
	for _, role := range roles {
		// 获取角色的默认权限
		rolePermissions := models.GetDefaultRolePermissions(role.Name)

		// 检查是否有所需权限
		for _, perm := range rolePermissions {
			if perm == permission {
				return true, nil
			}
			// admin权限包含所有权限
			if perm == models.PermissionAdmin {
				return true, nil
			}
		}
	}

	return false, nil
}
