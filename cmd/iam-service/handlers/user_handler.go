package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cloud-platform/collaborative-dev/cmd/iam-service/services"
	"github.com/cloud-platform/collaborative-dev/shared/api"
	"github.com/cloud-platform/collaborative-dev/shared/logger"
)

// UserHandler 用户管理处理器
type UserHandler struct {
	userService     *services.UserService
	userMgmtService *services.UserManagementService
	logger          logger.Logger
	respHandler     *api.ResponseHandler
}

// NewUserHandler 创建用户管理处理器实例
func NewUserHandler(userService *services.UserService, userMgmtService *services.UserManagementService, logger logger.Logger) *UserHandler {
	return &UserHandler{
		userService:     userService,
		userMgmtService: userMgmtService,
		logger:          logger,
		respHandler:     api.NewResponseHandler(),
	}
}

// GetUsers 获取用户列表
// @Summary 获取用户列表
// @Description 分页获取租户内的用户列表
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页号" default(1)
// @Param limit query int false "每页数量" default(20)
// @Param search query string false "搜索关键词"
// @Param is_active query bool false "是否激活"
// @Success 200 {object} StandardResponse{data=services.UserListResponse} "获取成功"
// @Failure 400 {object} StandardResponse "请求参数错误"
// @Failure 401 {object} StandardResponse "未授权"
// @Failure 500 {object} StandardResponse "服务器内部错误"
// @Router /api/v1/users [get]
func (h *UserHandler) GetUsers(c *gin.Context) {
	// 从上下文获取租户信息
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		h.respHandler.Unauthorized(c, "租户信息不存在")
		return
	}

	tenantUUID := tenantID.(uuid.UUID)

	// 解析查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")
	isActiveStr := c.Query("is_active")

	var isActive *bool
	if isActiveStr != "" {
		if activeVal, err := strconv.ParseBool(isActiveStr); err == nil {
			isActive = &activeVal
		}
	}

	// 构建查询请求
	req := &services.GetUsersRequest{
		TenantID: tenantUUID,
		Page:     page,
		Limit:    limit,
		Search:   search,
		IsActive: isActive,
	}

	// 获取用户列表
	result, err := h.userMgmtService.GetUsers(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("获取用户列表失败", "tenant_id", tenantUUID, "error", err)
		h.respHandler.InternalServerError(c, err.Error())
		return
	}

	h.respHandler.OK(c, "获取用户列表成功", result)
}

// CreateUser 创建用户
// @Summary 创建用户
// @Description 管理员创建新用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user body services.CreateUserRequest true "用户信息"
// @Success 201 {object} StandardResponse{data=models.User} "创建成功"
// @Failure 400 {object} StandardResponse "请求参数错误"
// @Failure 401 {object} StandardResponse "未授权"
// @Failure 409 {object} StandardResponse "用户已存在"
// @Failure 500 {object} StandardResponse "服务器内部错误"
// @Router /api/v1/users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req services.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("创建用户请求参数无效", "error", err)
		h.respHandler.BadRequest(c, "请求参数无效", nil)
		return
	}

	// 从上下文获取租户信息
	tenantID, _ := c.Get("tenant_id")
	tenantUUID := tenantID.(uuid.UUID)
	req.TenantID = tenantUUID

	// 创建用户
	user, err := h.userMgmtService.CreateUser(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("创建用户失败", "email", req.Email, "error", err)

		if contains(err.Error(), "已被注册") || contains(err.Error(), "已被使用") {
			h.respHandler.Conflict(c, err.Error())
		} else {
			h.respHandler.InternalServerError(c, err.Error())
		}
		return
	}

	h.logger.Info("用户创建成功", "email", req.Email, "user_id", user.ID)
	h.respHandler.Created(c, "用户创建成功", user.ToPublicUser())
}

// GetUser 获取单个用户
// @Summary 获取用户详情
// @Description 根据用户ID获取用户详细信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用户ID"
// @Success 200 {object} StandardResponse{data=models.User} "获取成功"
// @Failure 400 {object} StandardResponse "请求参数错误"
// @Failure 401 {object} StandardResponse "未授权"
// @Failure 404 {object} StandardResponse "用户不存在"
// @Failure 500 {object} StandardResponse "服务器内部错误"
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	// 解析用户ID
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.respHandler.BadRequest(c, "用户ID格式错误", nil)
		return
	}

	// 从上下文获取租户信息
	tenantID, _ := c.Get("tenant_id")
	tenantUUID := tenantID.(uuid.UUID)

	// 获取用户信息
	user, err := h.userService.GetUserByID(c.Request.Context(), userID, tenantUUID)
	if err != nil {
		h.logger.Error("获取用户信息失败", "user_id", userID, "error", err)

		if contains(err.Error(), "不存在") {
			h.respHandler.NotFound(c, err.Error())
		} else {
			h.respHandler.InternalServerError(c, err.Error())
		}
		return
	}

	h.respHandler.OK(c, "获取用户信息成功", user.ToPublicUser())
}

// UpdateUser 更新用户
// @Summary 更新用户信息
// @Description 管理员更新用户信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用户ID"
// @Param user body services.AdminUpdateUserRequest true "用户信息"
// @Success 200 {object} StandardResponse{data=models.User} "更新成功"
// @Failure 400 {object} StandardResponse "请求参数错误"
// @Failure 401 {object} StandardResponse "未授权"
// @Failure 404 {object} StandardResponse "用户不存在"
// @Failure 500 {object} StandardResponse "服务器内部错误"
// @Router /api/v1/users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	// 解析用户ID
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.respHandler.BadRequest(c, "用户ID格式错误", nil)
		return
	}

	var req services.AdminUpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("更新用户请求参数无效", "error", err)
		h.respHandler.BadRequest(c, "请求参数无效", nil)
		return
	}

	// 从上下文获取租户信息
	tenantID, _ := c.Get("tenant_id")
	tenantUUID := tenantID.(uuid.UUID)

	// 更新用户信息
	user, err := h.userMgmtService.UpdateUser(c.Request.Context(), userID, tenantUUID, &req)
	if err != nil {
		h.logger.Error("更新用户信息失败", "user_id", userID, "error", err)

		if contains(err.Error(), "不存在") {
			h.respHandler.NotFound(c, err.Error())
		} else {
			h.respHandler.InternalServerError(c, err.Error())
		}
		return
	}

	h.logger.Info("用户信息更新成功", "user_id", userID)
	h.respHandler.OK(c, "用户信息更新成功", user.ToPublicUser())
}

// DeleteUser 删除用户
// @Summary 删除用户
// @Description 管理员删除用户（软删除）
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用户ID"
// @Success 200 {object} StandardResponse "删除成功"
// @Failure 400 {object} StandardResponse "请求参数错误"
// @Failure 401 {object} StandardResponse "未授权"
// @Failure 404 {object} StandardResponse "用户不存在"
// @Failure 500 {object} StandardResponse "服务器内部错误"
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	// 解析用户ID
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.respHandler.BadRequest(c, "用户ID格式错误", nil)
		return
	}

	// 从上下文获取租户信息和当前用户信息
	tenantID, _ := c.Get("tenant_id")
	currentUserID, _ := c.Get("user_id")

	tenantUUID := tenantID.(uuid.UUID)
	currentUserUUID := currentUserID.(uuid.UUID)

	// 不能删除自己
	if userID == currentUserUUID {
		h.respHandler.BadRequest(c, "不能删除自己的账户", nil)
		return
	}

	// 删除用户
	err = h.userMgmtService.DeleteUser(c.Request.Context(), userID, tenantUUID)
	if err != nil {
		h.logger.Error("删除用户失败", "user_id", userID, "error", err)

		if contains(err.Error(), "不存在") {
			h.respHandler.NotFound(c, err.Error())
		} else {
			h.respHandler.InternalServerError(c, err.Error())
		}
		return
	}

	h.logger.Info("用户删除成功", "user_id", userID)
	h.respHandler.OK(c, "用户删除成功", nil)
}

// 辅助函数
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				indexOf(s, substr) >= 0)))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
