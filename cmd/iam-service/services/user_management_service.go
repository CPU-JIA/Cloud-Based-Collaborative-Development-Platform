package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/cloud-platform/collaborative-dev/shared/database"
	"github.com/cloud-platform/collaborative-dev/shared/models"
)

// UserManagementService 用户管理服务
type UserManagementService struct {
	db *database.PostgresDB
}

// CreateUserRequest 创建用户请求（管理员使用）
type CreateUserRequest struct {
	TenantID         uuid.UUID `json:"tenant_id"`
	Email            string    `json:"email" binding:"required,email"`
	Username         string    `json:"username" binding:"required,min=3,max=50"`
	Password         string    `json:"password" binding:"required,min=8"`
	FirstName        string    `json:"first_name" binding:"required"`
	LastName         string    `json:"last_name" binding:"required"`
	Phone            string    `json:"phone"`
	IsActive         *bool     `json:"is_active"` // 可选，默认true
	IsEmailVerified  *bool     `json:"is_email_verified"` // 可选，默认false
	RoleNames        []string  `json:"role_names"` // 分配的角色名称列表
}

// AdminUpdateUserRequest 管理员更新用户请求
type AdminUpdateUserRequest struct {
	Email           *string  `json:"email"`
	Username        *string  `json:"username"`
	FirstName       *string  `json:"first_name"`
	LastName        *string  `json:"last_name"`
	Phone           *string  `json:"phone"`
	Avatar          *string  `json:"avatar"`
	IsActive        *bool    `json:"is_active"`
	IsEmailVerified *bool    `json:"is_email_verified"`
	RoleNames       []string `json:"role_names"` // 更新角色
}

// GetUsersRequest 获取用户列表请求
type GetUsersRequest struct {
	TenantID uuid.UUID `json:"tenant_id"`
	Page     int       `json:"page" binding:"min=1"`
	Limit    int       `json:"limit" binding:"min=1,max=100"`
	Search   string    `json:"search"`    // 搜索关键词（邮箱、用户名、姓名）
	IsActive *bool     `json:"is_active"` // 过滤激活状态
	RoleName string    `json:"role_name"` // 按角色过滤
}

// UserListResponse 用户列表响应
type UserListResponse struct {
	Users      []UserWithRoles `json:"users"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	Limit      int             `json:"limit"`
	TotalPages int             `json:"total_pages"`
}

// UserWithRoles 用户信息（包含角色）
type UserWithRoles struct {
	ID              uuid.UUID `json:"id"`
	TenantID        uuid.UUID `json:"tenant_id"`
	Email           string    `json:"email"`
	Username        string    `json:"username"`
	FirstName       string    `json:"first_name"`
	LastName        string    `json:"last_name"`
	Avatar          string    `json:"avatar"`
	Phone           string    `json:"phone"`
	IsActive        bool      `json:"is_active"`
	IsEmailVerified bool      `json:"is_email_verified"`
	LastLoginAt     *time.Time `json:"last_login_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Roles           []string  `json:"roles"` // 角色名称列表
}

// NewUserManagementService 创建用户管理服务实例
func NewUserManagementService(db *database.PostgresDB) *UserManagementService {
	return &UserManagementService{
		db: db,
	}
}

// GetUsers 获取用户列表（分页）
func (s *UserManagementService) GetUsers(ctx context.Context, req *GetUsersRequest) (*UserListResponse, error) {
	tenantCtx := database.TenantContext{
		TenantID: req.TenantID,
	}

	// 构建查询
	query := s.db.WithContext(ctx, tenantCtx).
		Preload("Roles").
		Where("tenant_id = ?", req.TenantID)

	// 添加搜索条件
	if req.Search != "" {
		searchPattern := "%" + strings.ToLower(req.Search) + "%"
		query = query.Where(
			"LOWER(email) LIKE ? OR LOWER(username) LIKE ? OR LOWER(first_name) LIKE ? OR LOWER(last_name) LIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern,
		)
	}

	// 添加激活状态过滤
	if req.IsActive != nil {
		query = query.Where("is_active = ?", *req.IsActive)
	}

	// 按角色过滤
	if req.RoleName != "" {
		query = query.Joins("JOIN user_roles ur ON users.id = ur.user_id").
			Joins("JOIN roles r ON ur.role_id = r.id").
			Where("r.name = ? AND r.tenant_id = ?", req.RoleName, req.TenantID)
	}

	// 计算总数
	var total int64
	if err := query.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("统计用户总数失败: %w", err)
	}

	// 分页查询
	offset := (req.Page - 1) * req.Limit
	var users []models.User
	err := query.Offset(offset).Limit(req.Limit).
		Order("created_at DESC").
		Find(&users).Error

	if err != nil {
		return nil, fmt.Errorf("查询用户列表失败: %w", err)
	}

	// 转换为响应格式
	userList := make([]UserWithRoles, len(users))
	for i, user := range users {
		roles := make([]string, len(user.Roles))
		for j, role := range user.Roles {
			roles[j] = role.Name
		}

		userList[i] = UserWithRoles{
			ID:              user.ID,
			TenantID:        user.TenantID,
			Email:           user.Email,
			Username:        user.Username,
			FirstName:       user.FirstName,
			LastName:        user.LastName,
			Avatar:          user.Avatar,
			Phone:           user.Phone,
			IsActive:        user.IsActive,
			IsEmailVerified: user.IsEmailVerified,
			LastLoginAt:     user.LastLoginAt,
			CreatedAt:       user.CreatedAt,
			UpdatedAt:       user.UpdatedAt,
			Roles:           roles,
		}
	}

	// 计算总页数
	totalPages := int((total + int64(req.Limit) - 1) / int64(req.Limit))

	return &UserListResponse{
		Users:      userList,
		Total:      total,
		Page:       req.Page,
		Limit:      req.Limit,
		TotalPages: totalPages,
	}, nil
}

// CreateUser 创建用户（管理员）
func (s *UserManagementService) CreateUser(ctx context.Context, req *CreateUserRequest) (*models.User, error) {
	tenantCtx := database.TenantContext{
		TenantID: req.TenantID,
	}

	// 检查邮箱是否已存在
	var existingUser models.User
	err := s.db.WithContext(ctx, tenantCtx).
		Where("email = ? AND tenant_id = ?", req.Email, req.TenantID).
		First(&existingUser).Error

	if err == nil {
		return nil, fmt.Errorf("邮箱已被注册")
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("检查邮箱失败: %w", err)
	}

	// 检查用户名是否已存在
	err = s.db.WithContext(ctx, tenantCtx).
		Where("username = ? AND tenant_id = ?", req.Username, req.TenantID).
		First(&existingUser).Error

	if err == nil {
		return nil, fmt.Errorf("用户名已被使用")
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("检查用户名失败: %w", err)
	}

	// 设置默认值
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	isEmailVerified := false
	if req.IsEmailVerified != nil {
		isEmailVerified = *req.IsEmailVerified
	}

	// 创建用户
	user := &models.User{
		TenantID:        req.TenantID,
		Email:           req.Email,
		Username:        req.Username,
		FirstName:       req.FirstName,
		LastName:        req.LastName,
		Phone:           req.Phone,
		IsActive:        isActive,
		IsEmailVerified: isEmailVerified,
	}

	// 设置密码
	err = user.SetPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	// 在事务中创建用户和分配角色
	err = s.db.WithContext(ctx, tenantCtx).Transaction(func(tx *gorm.DB) error {
		// 创建用户
		if err := tx.Create(user).Error; err != nil {
			return fmt.Errorf("创建用户失败: %w", err)
		}

		// 分配角色
		if len(req.RoleNames) > 0 {
			return s.assignRoles(ctx, tx, user.ID, req.TenantID, req.RoleNames)
		} else {
			// 如果没有指定角色，分配默认角色
			return s.assignDefaultRole(ctx, tx, user.ID, req.TenantID)
		}
	})

	if err != nil {
		return nil, err
	}

	// 重新加载用户信息（包含角色）
	err = s.db.WithContext(ctx, tenantCtx).
		Preload("Roles").
		Where("id = ?", user.ID).
		First(user).Error

	if err != nil {
		return nil, fmt.Errorf("重新加载用户信息失败: %w", err)
	}

	return user, nil
}

// UpdateUser 更新用户信息（管理员）
func (s *UserManagementService) UpdateUser(ctx context.Context, userID, tenantID uuid.UUID, req *AdminUpdateUserRequest) (*models.User, error) {
	tenantCtx := database.TenantContext{
		TenantID: tenantID,
		UserID:   userID,
	}

	// 获取现有用户
	var user models.User
	err := s.db.WithContext(ctx, tenantCtx).
		Preload("Roles").
		Where("id = ? AND tenant_id = ?", userID, tenantID).
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	// 在事务中更新用户信息
	err = s.db.WithContext(ctx, tenantCtx).Transaction(func(tx *gorm.DB) error {
		// 准备更新数据
		updates := make(map[string]interface{})

		if req.Email != nil {
			// 检查邮箱是否被其他用户使用
			var existingUser models.User
			err := tx.Where("email = ? AND tenant_id = ? AND id != ?", *req.Email, tenantID, userID).
				First(&existingUser).Error
			if err == nil {
				return fmt.Errorf("邮箱已被其他用户使用")
			} else if err != gorm.ErrRecordNotFound {
				return fmt.Errorf("检查邮箱失败: %w", err)
			}
			updates["email"] = *req.Email
		}

		if req.Username != nil {
			// 检查用户名是否被其他用户使用
			var existingUser models.User
			err := tx.Where("username = ? AND tenant_id = ? AND id != ?", *req.Username, tenantID, userID).
				First(&existingUser).Error
			if err == nil {
				return fmt.Errorf("用户名已被其他用户使用")
			} else if err != gorm.ErrRecordNotFound {
				return fmt.Errorf("检查用户名失败: %w", err)
			}
			updates["username"] = *req.Username
		}

		if req.FirstName != nil {
			updates["first_name"] = *req.FirstName
		}
		if req.LastName != nil {
			updates["last_name"] = *req.LastName
		}
		if req.Phone != nil {
			updates["phone"] = *req.Phone
		}
		if req.Avatar != nil {
			updates["avatar"] = *req.Avatar
		}
		if req.IsActive != nil {
			updates["is_active"] = *req.IsActive
		}
		if req.IsEmailVerified != nil {
			updates["is_email_verified"] = *req.IsEmailVerified
			if *req.IsEmailVerified {
				updates["email_verified_at"] = time.Now()
			}
		}

		// 更新用户基本信息
		if len(updates) > 0 {
			if err := tx.Model(&user).Updates(updates).Error; err != nil {
				return fmt.Errorf("更新用户信息失败: %w", err)
			}
		}

		// 更新角色
		if req.RoleNames != nil {
			// 删除现有角色
			if err := tx.Where("user_id = ? AND tenant_id = ?", userID, tenantID).
				Delete(&models.UserRole{}).Error; err != nil {
				return fmt.Errorf("删除用户角色失败: %w", err)
			}

			// 分配新角色
			if len(req.RoleNames) > 0 {
				return s.assignRoles(ctx, tx, userID, tenantID, req.RoleNames)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 重新加载用户信息
	err = s.db.WithContext(ctx, tenantCtx).
		Preload("Roles").
		Where("id = ?", userID).
		First(&user).Error

	if err != nil {
		return nil, fmt.Errorf("重新加载用户信息失败: %w", err)
	}

	return &user, nil
}

// DeleteUser 删除用户（软删除）
func (s *UserManagementService) DeleteUser(ctx context.Context, userID, tenantID uuid.UUID) error {
	tenantCtx := database.TenantContext{
		TenantID: tenantID,
		UserID:   userID,
	}

	// 检查用户是否存在
	var user models.User
	err := s.db.WithContext(ctx, tenantCtx).
		Where("id = ? AND tenant_id = ?", userID, tenantID).
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("用户不存在")
		}
		return fmt.Errorf("查询用户失败: %w", err)
	}

	// 在事务中执行删除操作
	return s.db.WithContext(ctx, tenantCtx).Transaction(func(tx *gorm.DB) error {
		// 撤销所有用户会话
		err := tx.Model(&models.UserSession{}).
			Where("user_id = ? AND tenant_id = ?", userID, tenantID).
			Updates(map[string]interface{}{
				"is_active":  false,
				"revoked_at": time.Now(),
			}).Error
		if err != nil {
			return fmt.Errorf("撤销用户会话失败: %w", err)
		}

		// 删除用户角色关联
		err = tx.Where("user_id = ? AND tenant_id = ?", userID, tenantID).
			Delete(&models.UserRole{}).Error
		if err != nil {
			return fmt.Errorf("删除用户角色关联失败: %w", err)
		}

		// 软删除用户
		err = tx.Delete(&user).Error
		if err != nil {
			return fmt.Errorf("删除用户失败: %w", err)
		}

		return nil
	})
}

// 私有方法

// assignRoles 分配角色
func (s *UserManagementService) assignRoles(ctx context.Context, tx *gorm.DB, userID, tenantID uuid.UUID, roleNames []string) error {
	// 查找角色
	var roles []models.Role
	err := tx.Where("name IN ? AND tenant_id = ?", roleNames, tenantID).
		Find(&roles).Error
	if err != nil {
		return fmt.Errorf("查找角色失败: %w", err)
	}

	if len(roles) != len(roleNames) {
		return fmt.Errorf("部分角色不存在")
	}

	// 创建用户角色关联
	for _, role := range roles {
		userRole := &models.UserRole{
			UserID:   userID,
			RoleID:   role.ID,
			TenantID: tenantID,
		}
		if err := tx.Create(userRole).Error; err != nil {
			return fmt.Errorf("分配角色失败: %w", err)
		}
	}

	return nil
}

// assignDefaultRole 分配默认角色
func (s *UserManagementService) assignDefaultRole(ctx context.Context, tx *gorm.DB, userID, tenantID uuid.UUID) error {
	// 查找默认用户角色
	var defaultRole models.Role
	err := tx.Where("tenant_id = ? AND name = ?", tenantID, "user").
		First(&defaultRole).Error

	if err != nil {
		return fmt.Errorf("查找默认角色失败: %w", err)
	}

	// 分配角色
	userRole := &models.UserRole{
		UserID:   userID,
		RoleID:   defaultRole.ID,
		TenantID: tenantID,
	}

	err = tx.Create(userRole).Error
	if err != nil {
		return fmt.Errorf("分配默认角色失败: %w", err)
	}

	return nil
}