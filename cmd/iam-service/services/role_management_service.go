package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/cloud-platform/collaborative-dev/shared/database"
	"github.com/cloud-platform/collaborative-dev/shared/models"
)

// RoleManagementService 角色管理服务
type RoleManagementService struct {
	db *database.PostgresDB
}

// CreateRoleRequest 创建角色请求
type CreateRoleRequest struct {
	TenantID        uuid.UUID `json:"tenant_id"`
	Name            string    `json:"name" binding:"required,min=2,max=100"`
	DisplayName     string    `json:"display_name" binding:"required,min=2,max=200"`
	Description     string    `json:"description"`
	IsActive        *bool     `json:"is_active"` // 可选，默认true
	PermissionNames []string  `json:"permission_names"` // 分配的权限名称列表
}

// UpdateRoleRequest 更新角色请求
type UpdateRoleRequest struct {
	Name            *string  `json:"name"`
	DisplayName     *string  `json:"display_name"`
	Description     *string  `json:"description"`
	IsActive        *bool    `json:"is_active"`
	PermissionNames []string `json:"permission_names"` // 更新权限列表
}

// GetRolesRequest 获取角色列表请求
type GetRolesRequest struct {
	TenantID uuid.UUID `json:"tenant_id"`
	Page     int       `json:"page" binding:"min=1"`
	Limit    int       `json:"limit" binding:"min=1,max=100"`
	Search   string    `json:"search"`    // 搜索关键词（角色名、显示名、描述）
	IsActive *bool     `json:"is_active"` // 过滤激活状态
}

// RoleListResponse 角色列表响应
type RoleListResponse struct {
	Roles      []RoleWithPermissions `json:"roles"`
	Total      int64                 `json:"total"`
	Page       int                   `json:"page"`
	Limit      int                   `json:"limit"`
	TotalPages int                   `json:"total_pages"`
}

// RoleWithPermissions 角色信息（包含权限）
type RoleWithPermissions struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	IsSystem    bool      `json:"is_system"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
	Permissions []string  `json:"permissions"` // 权限名称列表
	UserCount   int64     `json:"user_count"`  // 拥有该角色的用户数量
}

// NewRoleManagementService 创建角色管理服务实例
func NewRoleManagementService(db *database.PostgresDB) *RoleManagementService {
	return &RoleManagementService{
		db: db,
	}
}

// GetRoles 获取角色列表（分页）
func (s *RoleManagementService) GetRoles(ctx context.Context, req *GetRolesRequest) (*RoleListResponse, error) {
	tenantCtx := database.TenantContext{
		TenantID: req.TenantID,
	}

	// 构建查询
	query := s.db.WithContext(ctx, tenantCtx).
		Preload("Permissions").
		Where("tenant_id = ?", req.TenantID)

	// 添加搜索条件
	if req.Search != "" {
		searchPattern := "%" + strings.ToLower(req.Search) + "%"
		query = query.Where(
			"LOWER(name) LIKE ? OR LOWER(display_name) LIKE ? OR LOWER(description) LIKE ?",
			searchPattern, searchPattern, searchPattern,
		)
	}

	// 添加激活状态过滤
	if req.IsActive != nil {
		query = query.Where("is_active = ?", *req.IsActive)
	}

	// 计算总数
	var total int64
	if err := query.Model(&models.Role{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("统计角色总数失败: %w", err)
	}

	// 分页查询
	offset := (req.Page - 1) * req.Limit
	var roles []models.Role
	err := query.Offset(offset).Limit(req.Limit).
		Order("is_system DESC, name ASC").
		Find(&roles).Error

	if err != nil {
		return nil, fmt.Errorf("查询角色列表失败: %w", err)
	}

	// 获取每个角色的用户数量
	roleList := make([]RoleWithPermissions, len(roles))
	for i, role := range roles {
		// 统计用户数量
		var userCount int64
		s.db.WithContext(ctx, tenantCtx).
			Model(&models.UserRole{}).
			Where("role_id = ? AND tenant_id = ?", role.ID, req.TenantID).
			Count(&userCount)

		// 获取权限名称列表
		permissions := make([]string, len(role.Permissions))
		for j, permission := range role.Permissions {
			permissions[j] = permission.Name
		}

		roleList[i] = RoleWithPermissions{
			ID:          role.ID,
			TenantID:    role.TenantID,
			Name:        role.Name,
			DisplayName: role.DisplayName,
			Description: role.Description,
			IsSystem:    role.IsSystem,
			IsActive:    role.IsActive,
			CreatedAt:   role.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:   role.UpdatedAt.Format("2006-01-02 15:04:05"),
			Permissions: permissions,
			UserCount:   userCount,
		}
	}

	// 计算总页数
	totalPages := int((total + int64(req.Limit) - 1) / int64(req.Limit))

	return &RoleListResponse{
		Roles:      roleList,
		Total:      total,
		Page:       req.Page,
		Limit:      req.Limit,
		TotalPages: totalPages,
	}, nil
}

// CreateRole 创建角色
func (s *RoleManagementService) CreateRole(ctx context.Context, req *CreateRoleRequest) (*models.Role, error) {
	tenantCtx := database.TenantContext{
		TenantID: req.TenantID,
	}

	// 检查角色名是否已存在
	var existingRole models.Role
	err := s.db.WithContext(ctx, tenantCtx).
		Where("name = ? AND tenant_id = ?", req.Name, req.TenantID).
		First(&existingRole).Error

	if err == nil {
		return nil, fmt.Errorf("角色名已存在")
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("检查角色名失败: %w", err)
	}

	// 设置默认值
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	// 创建角色
	role := &models.Role{
		TenantID:    req.TenantID,
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		IsSystem:    false, // 管理员创建的角色都不是系统角色
		IsActive:    isActive,
	}

	// 在事务中创建角色和分配权限
	err = s.db.WithContext(ctx, tenantCtx).Transaction(func(tx *gorm.DB) error {
		// 创建角色
		if err := tx.Create(role).Error; err != nil {
			return fmt.Errorf("创建角色失败: %w", err)
		}

		// 分配权限
		if len(req.PermissionNames) > 0 {
			return s.assignPermissions(ctx, tx, role.ID, req.PermissionNames)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 重新加载角色信息（包含权限）
	err = s.db.WithContext(ctx, tenantCtx).
		Preload("Permissions").
		Where("id = ?", role.ID).
		First(role).Error

	if err != nil {
		return nil, fmt.Errorf("重新加载角色信息失败: %w", err)
	}

	return role, nil
}

// GetRoleByID 根据ID获取角色
func (s *RoleManagementService) GetRoleByID(ctx context.Context, roleID, tenantID uuid.UUID) (*models.Role, error) {
	tenantCtx := database.TenantContext{
		TenantID: tenantID,
	}

	var role models.Role
	err := s.db.WithContext(ctx, tenantCtx).
		Preload("Permissions").
		Where("id = ? AND tenant_id = ?", roleID, tenantID).
		First(&role).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("角色不存在")
		}
		return nil, fmt.Errorf("查询角色失败: %w", err)
	}

	return &role, nil
}

// UpdateRole 更新角色信息
func (s *RoleManagementService) UpdateRole(ctx context.Context, roleID, tenantID uuid.UUID, req *UpdateRoleRequest) (*models.Role, error) {
	tenantCtx := database.TenantContext{
		TenantID: tenantID,
	}

	// 获取现有角色
	role, err := s.GetRoleByID(ctx, roleID, tenantID)
	if err != nil {
		return nil, err
	}

	// 检查是否为系统角色
	if role.IsSystem {
		return nil, fmt.Errorf("系统角色不能修改")
	}

	// 在事务中更新角色信息
	err = s.db.WithContext(ctx, tenantCtx).Transaction(func(tx *gorm.DB) error {
		// 准备更新数据
		updates := make(map[string]interface{})

		if req.Name != nil {
			// 检查角色名是否被其他角色使用
			var existingRole models.Role
			err := tx.Where("name = ? AND tenant_id = ? AND id != ?", *req.Name, tenantID, roleID).
				First(&existingRole).Error
			if err == nil {
				return fmt.Errorf("角色名已被其他角色使用")
			} else if err != gorm.ErrRecordNotFound {
				return fmt.Errorf("检查角色名失败: %w", err)
			}
			updates["name"] = *req.Name
		}

		if req.DisplayName != nil {
			updates["display_name"] = *req.DisplayName
		}
		if req.Description != nil {
			updates["description"] = *req.Description
		}
		if req.IsActive != nil {
			updates["is_active"] = *req.IsActive
		}

		// 更新角色基本信息
		if len(updates) > 0 {
			if err := tx.Model(role).Updates(updates).Error; err != nil {
				return fmt.Errorf("更新角色信息失败: %w", err)
			}
		}

		// 更新权限
		if req.PermissionNames != nil {
			// 删除现有权限
			if err := tx.Where("role_id = ?", roleID).
				Delete(&models.RolePermission{}).Error; err != nil {
				return fmt.Errorf("删除角色权限失败: %w", err)
			}

			// 分配新权限
			if len(req.PermissionNames) > 0 {
				return s.assignPermissions(ctx, tx, roleID, req.PermissionNames)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 重新加载角色信息
	err = s.db.WithContext(ctx, tenantCtx).
		Preload("Permissions").
		Where("id = ?", roleID).
		First(role).Error

	if err != nil {
		return nil, fmt.Errorf("重新加载角色信息失败: %w", err)
	}

	return role, nil
}

// DeleteRole 删除角色
func (s *RoleManagementService) DeleteRole(ctx context.Context, roleID, tenantID uuid.UUID) error {
	tenantCtx := database.TenantContext{
		TenantID: tenantID,
	}

	// 获取角色信息
	role, err := s.GetRoleByID(ctx, roleID, tenantID)
	if err != nil {
		return err
	}

	// 检查是否为系统角色
	if role.IsSystem {
		return fmt.Errorf("系统角色不能删除")
	}

	// 检查是否有用户正在使用该角色
	var userRoleCount int64
	err = s.db.WithContext(ctx, tenantCtx).
		Model(&models.UserRole{}).
		Where("role_id = ? AND tenant_id = ?", roleID, tenantID).
		Count(&userRoleCount).Error

	if err != nil {
		return fmt.Errorf("检查角色使用情况失败: %w", err)
	}

	if userRoleCount > 0 {
		return fmt.Errorf("角色正在被 %d 个用户使用，无法删除", userRoleCount)
	}

	// 在事务中执行删除操作
	return s.db.WithContext(ctx, tenantCtx).Transaction(func(tx *gorm.DB) error {
		// 删除角色权限关联
		err := tx.Where("role_id = ?", roleID).
			Delete(&models.RolePermission{}).Error
		if err != nil {
			return fmt.Errorf("删除角色权限关联失败: %w", err)
		}

		// 删除角色
		err = tx.Delete(role).Error
		if err != nil {
			return fmt.Errorf("删除角色失败: %w", err)
		}

		return nil
	})
}

// GetAvailablePermissions 获取可用权限列表
func (s *RoleManagementService) GetAvailablePermissions(ctx context.Context) ([]models.Permission, error) {
	var permissions []models.Permission
	err := s.db.DB.WithContext(ctx).
		Order("resource ASC, action ASC").
		Find(&permissions).Error

	if err != nil {
		return nil, fmt.Errorf("查询权限列表失败: %w", err)
	}

	return permissions, nil
}

// 私有方法

// assignPermissions 分配权限
func (s *RoleManagementService) assignPermissions(ctx context.Context, tx *gorm.DB, roleID uuid.UUID, permissionNames []string) error {
	// 查找权限
	var permissions []models.Permission
	err := tx.Where("name IN ?", permissionNames).
		Find(&permissions).Error
	if err != nil {
		return fmt.Errorf("查找权限失败: %w", err)
	}

	if len(permissions) != len(permissionNames) {
		return fmt.Errorf("部分权限不存在")
	}

	// 创建角色权限关联
	for _, permission := range permissions {
		rolePermission := &models.RolePermission{
			RoleID:       roleID,
			PermissionID: permission.ID,
		}
		if err := tx.Create(rolePermission).Error; err != nil {
			return fmt.Errorf("分配权限失败: %w", err)
		}
	}

	return nil
}