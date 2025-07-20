package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/cloud-platform/collaborative-dev/shared/auth"
	"github.com/cloud-platform/collaborative-dev/shared/database"
	"github.com/cloud-platform/collaborative-dev/shared/models"
)

// UserService 用户服务
type UserService struct {
	db         *database.PostgresDB
	jwtService *auth.JWTService
	config     UserServiceConfig
}

// UserServiceConfig 用户服务配置
type UserServiceConfig struct {
	PasswordMinLength int
	MaxLoginAttempts  int
	LockoutDuration   time.Duration
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	User        map[string]interface{} `json:"user"`
	TokenPair   *auth.TokenPair        `json:"tokens"`
	RequiresMFA bool                   `json:"requires_mfa"`
	SessionID   uuid.UUID              `json:"session_id"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	TenantID  uuid.UUID `json:"tenant_id" binding:"required"`
	Email     string    `json:"email" binding:"required,email"`
	Username  string    `json:"username" binding:"required,min=3,max=50"`
	Password  string    `json:"password" binding:"required,min=8"`
	FirstName string    `json:"first_name" binding:"required"`
	LastName  string    `json:"last_name" binding:"required"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Avatar    string `json:"avatar"`
	Phone     string `json:"phone"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// NewUserService 创建用户服务实例
func NewUserService(db *database.PostgresDB, jwtService *auth.JWTService, config UserServiceConfig) *UserService {
	return &UserService{
		db:         db,
		jwtService: jwtService,
		config:     config,
	}
}

// Login 用户登录
func (s *UserService) Login(ctx context.Context, req *LoginRequest, clientIP, userAgent string) (*LoginResponse, error) {
	tenantCtx := database.TenantContext{} // 登录时暂不设置租户上下文

	// 查找用户
	var user models.User
	err := s.db.WithContext(ctx, tenantCtx).
		Preload("Roles").
		Preload("Roles.Permissions").
		Where("email = ? AND is_active = ?", req.Email, true).
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在或未激活")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	// 检查用户是否被锁定
	if user.IsLocked() {
		return nil, fmt.Errorf("账户已被锁定，请稍后再试")
	}

	// 验证密码
	if !user.CheckPassword(req.Password) {
		// 增加失败登录次数
		user.IncrementFailedLogin()

		// 检查是否需要锁定账户
		if user.FailedLoginCount >= s.config.MaxLoginAttempts {
			user.Lock(s.config.LockoutDuration)
		}

		// 更新用户信息
		s.db.WithContext(ctx, tenantCtx).Save(&user)

		return nil, fmt.Errorf("密码错误")
	}

	// 设置租户上下文
	tenantCtx.TenantID = user.TenantID
	tenantCtx.UserID = user.ID

	// 重置失败登录次数
	user.UpdateLastLogin()

	// 更新用户信息
	err = s.db.WithContext(ctx, tenantCtx).Save(&user).Error
	if err != nil {
		return nil, fmt.Errorf("更新用户登录信息失败: %w", err)
	}

	// 检查是否启用了双因子认证
	if user.TwoFactorEnabled {
		// TODO: 实现MFA逻辑
		return &LoginResponse{
			User:        user.ToPublicUser(),
			RequiresMFA: true,
		}, nil
	}

	// 获取用户权限
	permissions := user.GetPermissions()

	// 生成令牌对
	tokenPair, err := s.jwtService.GenerateTokenPair(
		user.ID,
		user.TenantID,
		user.Email,
		s.getUserRole(&user),
		permissions,
	)
	if err != nil {
		return nil, fmt.Errorf("生成令牌失败: %w", err)
	}

	// 创建会话记录
	session := &models.UserSession{
		UserID:       user.ID,
		TenantID:     user.TenantID,
		SessionToken: tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		UserAgent:    userAgent,
		IPAddress:    clientIP,
		ExpiresAt:    tokenPair.ExpiresAt,
		IsActive:     true,
	}

	err = s.db.WithContext(ctx, tenantCtx).Create(session).Error
	if err != nil {
		return nil, fmt.Errorf("创建会话失败: %w", err)
	}

	return &LoginResponse{
		User:        user.ToPublicUser(),
		TokenPair:   tokenPair,
		RequiresMFA: false,
		SessionID:   session.ID,
	}, nil
}

// Register 用户注册
func (s *UserService) Register(ctx context.Context, req *RegisterRequest) (*models.User, error) {
	tenantCtx := database.TenantContext{
		TenantID: req.TenantID,
	}

	// 检查邮箱是否已存在
	var existingUser models.User
	err := s.db.WithContext(ctx, tenantCtx).
		Where("email = ?", req.Email).
		First(&existingUser).Error

	if err == nil {
		return nil, fmt.Errorf("邮箱已被注册")
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("检查邮箱失败: %w", err)
	}

	// 检查用户名是否已存在
	err = s.db.WithContext(ctx, tenantCtx).
		Where("username = ?", req.Username).
		First(&existingUser).Error

	if err == nil {
		return nil, fmt.Errorf("用户名已被使用")
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("检查用户名失败: %w", err)
	}

	// 验证密码强度
	if len(req.Password) < s.config.PasswordMinLength {
		return nil, fmt.Errorf("密码长度至少需要%d个字符", s.config.PasswordMinLength)
	}

	// 创建用户
	user := &models.User{
		TenantID:  req.TenantID,
		Email:     req.Email,
		Username:  req.Username,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		IsActive:  true,
	}

	// 设置密码
	err = user.SetPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	// 保存用户
	err = s.db.WithContext(ctx, tenantCtx).Create(user).Error
	if err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	// 分配默认角色
	err = s.assignDefaultRole(ctx, user.ID, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("分配默认角色失败: %w", err)
	}

	return user, nil
}

// GetUserByID 根据ID获取用户
func (s *UserService) GetUserByID(ctx context.Context, userID, tenantID uuid.UUID) (*models.User, error) {
	tenantCtx := database.TenantContext{
		TenantID: tenantID,
		UserID:   userID,
	}

	var user models.User
	err := s.db.WithContext(ctx, tenantCtx).
		Preload("Roles").
		Preload("Roles.Permissions").
		Where("id = ? AND tenant_id = ?", userID, tenantID).
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	return &user, nil
}

// UpdateUser 更新用户信息
func (s *UserService) UpdateUser(ctx context.Context, userID, tenantID uuid.UUID, req *UpdateUserRequest) (*models.User, error) {
	tenantCtx := database.TenantContext{
		TenantID: tenantID,
		UserID:   userID,
	}

	// 获取现有用户
	user, err := s.GetUserByID(ctx, userID, tenantID)
	if err != nil {
		return nil, err
	}

	// 更新字段
	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}
	if req.LastName != "" {
		user.LastName = req.LastName
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}

	// 保存更新
	err = s.db.WithContext(ctx, tenantCtx).Save(user).Error
	if err != nil {
		return nil, fmt.Errorf("更新用户失败: %w", err)
	}

	return user, nil
}

// ChangePassword 修改密码
func (s *UserService) ChangePassword(ctx context.Context, userID, tenantID uuid.UUID, req *ChangePasswordRequest) error {
	tenantCtx := database.TenantContext{
		TenantID: tenantID,
		UserID:   userID,
	}

	// 获取用户
	user, err := s.GetUserByID(ctx, userID, tenantID)
	if err != nil {
		return err
	}

	// 验证当前密码
	if !user.CheckPassword(req.CurrentPassword) {
		return fmt.Errorf("当前密码错误")
	}

	// 验证新密码强度
	if len(req.NewPassword) < s.config.PasswordMinLength {
		return fmt.Errorf("新密码长度至少需要%d个字符", s.config.PasswordMinLength)
	}

	// 设置新密码
	err = user.SetPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("新密码加密失败: %w", err)
	}

	// 更新密码重置时间
	now := time.Now()
	user.PasswordResetAt = &now

	// 保存更新
	err = s.db.WithContext(ctx, tenantCtx).Save(user).Error
	if err != nil {
		return fmt.Errorf("更新密码失败: %w", err)
	}

	// 撤销所有现有会话（强制重新登录）
	err = s.revokeAllUserSessions(ctx, userID, tenantID)
	if err != nil {
		return fmt.Errorf("撤销用户会话失败: %w", err)
	}

	return nil
}

// RefreshToken 刷新令牌
func (s *UserService) RefreshToken(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	// 验证刷新令牌
	tokenPair, err := s.jwtService.RefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("刷新令牌失败: %w", err)
	}

	// 提取用户信息
	userID, tenantID, err := s.jwtService.ExtractUserInfo(tokenPair.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("提取用户信息失败: %w", err)
	}

	tenantCtx := database.TenantContext{
		TenantID: tenantID,
		UserID:   userID,
	}

	// 更新会话中的刷新令牌
	err = s.db.WithContext(ctx, tenantCtx).
		Model(&models.UserSession{}).
		Where("refresh_token = ? AND is_active = ?", refreshToken, true).
		Updates(map[string]interface{}{
			"refresh_token": tokenPair.RefreshToken,
			"expires_at":    tokenPair.ExpiresAt,
			"updated_at":    time.Now(),
		}).Error

	if err != nil {
		return nil, fmt.Errorf("更新会话失败: %w", err)
	}

	return tokenPair, nil
}

// Logout 用户登出
func (s *UserService) Logout(ctx context.Context, refreshToken string) error {
	// 撤销会话
	err := s.db.DB.WithContext(ctx).
		Model(&models.UserSession{}).
		Where("refresh_token = ?", refreshToken).
		Updates(map[string]interface{}{
			"is_active":  false,
			"revoked_at": time.Now(),
		}).Error

	if err != nil {
		return fmt.Errorf("撤销会话失败: %w", err)
	}

	return nil
}

// ValidateToken 验证令牌
func (s *UserService) ValidateToken(ctx context.Context, tokenString string) (*models.User, error) {
	// 验证JWT令牌
	claims, err := s.jwtService.ValidateToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("令牌验证失败: %w", err)
	}

	// 获取用户信息
	user, err := s.GetUserByID(ctx, claims.UserID, claims.TenantID)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	// 检查用户状态
	if !user.IsActive {
		return nil, fmt.Errorf("用户已被禁用")
	}

	if user.IsLocked() {
		return nil, fmt.Errorf("用户已被锁定")
	}

	return user, nil
}

// 私有方法

// getUserRole 获取用户的主要角色
func (s *UserService) getUserRole(user *models.User) string {
	if len(user.Roles) == 0 {
		return "user"
	}
	return user.Roles[0].Name
}

// assignDefaultRole 分配默认角色
func (s *UserService) assignDefaultRole(ctx context.Context, userID, tenantID uuid.UUID) error {
	tenantCtx := database.TenantContext{
		TenantID: tenantID,
		UserID:   userID,
	}

	// 查找默认用户角色
	var defaultRole models.Role
	err := s.db.WithContext(ctx, tenantCtx).
		Where("tenant_id = ? AND name = ?", tenantID, "user").
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

	err = s.db.WithContext(ctx, tenantCtx).Create(userRole).Error
	if err != nil {
		return fmt.Errorf("分配角色失败: %w", err)
	}

	return nil
}

// revokeAllUserSessions 撤销用户所有会话
func (s *UserService) revokeAllUserSessions(ctx context.Context, userID, tenantID uuid.UUID) error {
	tenantCtx := database.TenantContext{
		TenantID: tenantID,
		UserID:   userID,
	}

	err := s.db.WithContext(ctx, tenantCtx).
		Model(&models.UserSession{}).
		Where("user_id = ? AND is_active = ?", userID, true).
		Updates(map[string]interface{}{
			"is_active":  false,
			"revoked_at": time.Now(),
		}).Error

	return err
}
