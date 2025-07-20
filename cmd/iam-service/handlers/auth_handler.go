package handlers

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cloud-platform/collaborative-dev/cmd/iam-service/services"
	"github.com/cloud-platform/collaborative-dev/shared/api"
	"github.com/cloud-platform/collaborative-dev/shared/logger"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	userService *services.UserService
	logger      logger.Logger
	respHandler *api.ResponseHandler
}

// NewAuthHandler 创建认证处理器实例
func NewAuthHandler(userService *services.UserService, logger logger.Logger) *AuthHandler {
	return &AuthHandler{
		userService: userService,
		logger:      logger,
		respHandler: api.NewResponseHandler(),
	}
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户通过邮箱和密码登录系统
// @Tags 认证
// @Accept json
// @Produce json
// @Param login body services.LoginRequest true "登录信息"
// @Success 200 {object} StandardResponse{data=services.LoginResponse} "登录成功"
// @Failure 400 {object} StandardResponse "请求参数错误"
// @Failure 401 {object} StandardResponse "认证失败"
// @Failure 500 {object} StandardResponse "服务器内部错误"
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("登录请求参数无效", "error", err)
		h.respHandler.BadRequest(c, "请求参数无效", nil)
		return
	}

	// 获取客户端信息
	clientIP := h.getClientIP(c)
	userAgent := c.GetHeader("User-Agent")

	// 执行登录
	loginResp, err := h.userService.Login(c.Request.Context(), &req, clientIP, userAgent)
	if err != nil {
		h.logger.Error("用户登录失败", "email", req.Email, "error", err)
		h.respHandler.Unauthorized(c, err.Error())
		return
	}

	h.logger.Info("用户登录成功", "email", req.Email, "user_id", loginResp.User["id"])
	h.respHandler.OK(c, "登录成功", loginResp)
}

// Register 用户注册
// @Summary 用户注册
// @Description 注册新用户账户
// @Tags 认证
// @Accept json
// @Produce json
// @Param register body services.RegisterRequest true "注册信息"
// @Success 201 {object} StandardResponse{data=models.User} "注册成功"
// @Failure 400 {object} StandardResponse "请求参数错误"
// @Failure 409 {object} StandardResponse "用户已存在"
// @Failure 500 {object} StandardResponse "服务器内部错误"
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req services.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("注册请求参数无效", "error", err)
		h.respHandler.BadRequest(c, "请求参数无效", nil)
		return
	}

	// 执行注册
	user, err := h.userService.Register(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("用户注册失败", "email", req.Email, "error", err)

		if strings.Contains(err.Error(), "已被注册") || strings.Contains(err.Error(), "已被使用") {
			h.respHandler.Conflict(c, err.Error())
		} else {
			h.respHandler.InternalServerError(c, err.Error())
		}
		return
	}

	h.logger.Info("用户注册成功", "email", req.Email, "user_id", user.ID)
	h.respHandler.Created(c, "注册成功", user.ToPublicUser())
}

// RefreshToken 刷新令牌
// @Summary 刷新访问令牌
// @Description 使用刷新令牌获取新的访问令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Param refresh body map[string]string true "刷新令牌" example({"refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."})
// @Success 200 {object} StandardResponse{data=auth.TokenPair} "刷新成功"
// @Failure 400 {object} StandardResponse "请求参数错误"
// @Failure 401 {object} StandardResponse "刷新令牌无效"
// @Failure 500 {object} StandardResponse "服务器内部错误"
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("刷新令牌请求参数无效", "error", err)
		h.respHandler.BadRequest(c, "请求参数无效", nil)
		return
	}

	refreshToken, exists := req["refresh_token"]
	if !exists || refreshToken == "" {
		h.respHandler.BadRequest(c, "刷新令牌不能为空", nil)
		return
	}

	// 执行令牌刷新
	tokenPair, err := h.userService.RefreshToken(c.Request.Context(), refreshToken)
	if err != nil {
		h.logger.Error("刷新令牌失败", "error", err)
		h.respHandler.Unauthorized(c, err.Error())
		return
	}

	h.logger.Info("令牌刷新成功")
	h.respHandler.OK(c, "令牌刷新成功", tokenPair)
}

// Logout 用户登出
// @Summary 用户登出
// @Description 用户登出并撤销会话
// @Tags 认证
// @Accept json
// @Produce json
// @Param logout body map[string]string true "刷新令牌" example({"refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."})
// @Success 200 {object} StandardResponse "登出成功"
// @Failure 400 {object} StandardResponse "请求参数错误"
// @Failure 500 {object} StandardResponse "服务器内部错误"
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("登出请求参数无效", "error", err)
		h.respHandler.BadRequest(c, "请求参数无效", nil)
		return
	}

	refreshToken, exists := req["refresh_token"]
	if !exists || refreshToken == "" {
		h.respHandler.BadRequest(c, "刷新令牌不能为空", nil)
		return
	}

	// 执行登出
	err := h.userService.Logout(c.Request.Context(), refreshToken)
	if err != nil {
		h.logger.Error("用户登出失败", "error", err)
		h.respHandler.InternalServerError(c, err.Error())
		return
	}

	h.logger.Info("用户登出成功")
	h.respHandler.OK(c, "登出成功", nil)
}

// ValidateToken 验证令牌
// @Summary 验证访问令牌
// @Description 验证访问令牌的有效性并返回用户信息
// @Tags 认证
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} StandardResponse{data=models.User} "令牌有效"
// @Failure 401 {object} StandardResponse "令牌无效"
// @Failure 500 {object} StandardResponse "服务器内部错误"
// @Router /api/v1/auth/validate [get]
func (h *AuthHandler) ValidateToken(c *gin.Context) {
	// 从请求头获取令牌
	tokenString := h.extractTokenFromHeader(c)
	if tokenString == "" {
		h.respHandler.Unauthorized(c, "缺少访问令牌")
		return
	}

	// 验证令牌
	user, err := h.userService.ValidateToken(c.Request.Context(), tokenString)
	if err != nil {
		h.logger.Error("令牌验证失败", "error", err)
		h.respHandler.Unauthorized(c, err.Error())
		return
	}

	h.logger.Info("令牌验证成功", "user_id", user.ID)
	h.respHandler.OK(c, "令牌有效", user.ToPublicUser())
}

// GetProfile 获取当前用户信息
// @Summary 获取当前用户信息
// @Description 获取当前登录用户的详细信息
// @Tags 用户
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} StandardResponse{data=models.User} "获取成功"
// @Failure 401 {object} StandardResponse "未授权"
// @Failure 500 {object} StandardResponse "服务器内部错误"
// @Router /api/v1/auth/profile [get]
func (h *AuthHandler) GetProfile(c *gin.Context) {
	// 从上下文获取用户信息（由JWT中间件设置）
	userID, exists := c.Get("user_id")
	if !exists {
		h.respHandler.Unauthorized(c, "用户信息不存在")
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		h.respHandler.Unauthorized(c, "租户信息不存在")
		return
	}

	// 类型转换
	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		h.respHandler.InternalServerError(c, "用户ID格式错误")
		return
	}

	tenantUUID, ok := tenantID.(uuid.UUID)
	if !ok {
		h.respHandler.InternalServerError(c, "租户ID格式错误")
		return
	}

	// 获取用户信息
	user, err := h.userService.GetUserByID(c.Request.Context(), userUUID, tenantUUID)
	if err != nil {
		h.logger.Error("获取用户信息失败", "user_id", userUUID, "error", err)
		h.respHandler.InternalServerError(c, err.Error())
		return
	}

	h.respHandler.OK(c, "获取用户信息成功", user.ToPublicUser())
}

// UpdateProfile 更新用户信息
// @Summary 更新当前用户信息
// @Description 更新当前登录用户的个人信息
// @Tags 用户
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param profile body services.UpdateUserRequest true "用户信息"
// @Success 200 {object} StandardResponse{data=models.User} "更新成功"
// @Failure 400 {object} StandardResponse "请求参数错误"
// @Failure 401 {object} StandardResponse "未授权"
// @Failure 500 {object} StandardResponse "服务器内部错误"
// @Router /api/v1/auth/profile [put]
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	var req services.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("更新用户信息请求参数无效", "error", err)
		h.respHandler.BadRequest(c, "请求参数无效", nil)
		return
	}

	// 从上下文获取用户信息
	userID, _ := c.Get("user_id")
	tenantID, _ := c.Get("tenant_id")

	userUUID := userID.(uuid.UUID)
	tenantUUID := tenantID.(uuid.UUID)

	// 更新用户信息
	user, err := h.userService.UpdateUser(c.Request.Context(), userUUID, tenantUUID, &req)
	if err != nil {
		h.logger.Error("更新用户信息失败", "user_id", userUUID, "error", err)
		h.respHandler.InternalServerError(c, err.Error())
		return
	}

	h.logger.Info("用户信息更新成功", "user_id", userUUID)
	h.respHandler.OK(c, "用户信息更新成功", user.ToPublicUser())
}

// ChangePassword 修改密码
// @Summary 修改密码
// @Description 修改当前用户的密码
// @Tags 用户
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param password body services.ChangePasswordRequest true "密码信息"
// @Success 200 {object} StandardResponse "修改成功"
// @Failure 400 {object} StandardResponse "请求参数错误"
// @Failure 401 {object} StandardResponse "未授权"
// @Failure 500 {object} StandardResponse "服务器内部错误"
// @Router /api/v1/auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req services.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("修改密码请求参数无效", "error", err)
		h.respHandler.BadRequest(c, "请求参数无效", nil)
		return
	}

	// 从上下文获取用户信息
	userID, _ := c.Get("user_id")
	tenantID, _ := c.Get("tenant_id")

	userUUID := userID.(uuid.UUID)
	tenantUUID := tenantID.(uuid.UUID)

	// 修改密码
	err := h.userService.ChangePassword(c.Request.Context(), userUUID, tenantUUID, &req)
	if err != nil {
		h.logger.Error("修改密码失败", "user_id", userUUID, "error", err)

		if strings.Contains(err.Error(), "当前密码错误") {
			h.respHandler.BadRequest(c, err.Error(), nil)
		} else {
			h.respHandler.InternalServerError(c, err.Error())
		}
		return
	}

	h.logger.Info("密码修改成功", "user_id", userUUID)
	h.respHandler.OK(c, "密码修改成功，请重新登录", nil)
}

// 辅助方法

// getClientIP 获取客户端IP地址
func (h *AuthHandler) getClientIP(c *gin.Context) string {
	// 优先从代理头中获取真实IP
	if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
		// X-Forwarded-For可能包含多个IP，取第一个
		ips := strings.Split(ip, ",")
		return strings.TrimSpace(ips[0])
	}

	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		return ip
	}

	// 从连接中获取IP
	return c.ClientIP()
}

// extractTokenFromHeader 从请求头中提取JWT令牌
func (h *AuthHandler) extractTokenFromHeader(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}

	// Bearer token格式: "Bearer <token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}
