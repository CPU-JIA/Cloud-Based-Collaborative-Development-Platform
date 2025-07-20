package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cloud-platform/collaborative-dev/cmd/iam-service/services"
	"github.com/cloud-platform/collaborative-dev/shared/api"
	"github.com/cloud-platform/collaborative-dev/shared/logger"
)

// RoleHandler 角色管理处理器
type RoleHandler struct {
	roleService *services.RoleManagementService
	logger      logger.Logger
	respHandler *api.ResponseHandler
}

// NewRoleHandler 创建角色管理处理器实例
func NewRoleHandler(roleService *services.RoleManagementService, logger logger.Logger) *RoleHandler {
	return &RoleHandler{
		roleService: roleService,
		logger:      logger,
		respHandler: api.NewResponseHandler(),
	}
}

// GetRoles 获取角色列表
// @Summary 获取角色列表
// @Description 分页获取租户内的角色列表
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页号" default(1)
// @Param limit query int false "每页数量" default(20)
// @Param search query string false "搜索关键词"
// @Param is_active query bool false "是否激活"
// @Success 200 {object} StandardResponse{data=services.RoleListResponse} "获取成功"
// @Failure 400 {object} StandardResponse "请求参数错误"
// @Failure 401 {object} StandardResponse "未授权"
// @Failure 500 {object} StandardResponse "服务器内部错误"
// @Router /api/v1/roles [get]
func (h *RoleHandler) GetRoles(c *gin.Context) {
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
	req := &services.GetRolesRequest{
		TenantID: tenantUUID,
		Page:     page,
		Limit:    limit,
		Search:   search,
		IsActive: isActive,
	}

	// 获取角色列表
	result, err := h.roleService.GetRoles(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("获取角色列表失败", "tenant_id", tenantUUID, "error", err)
		h.respHandler.InternalServerError(c, err.Error())
		return
	}

	h.respHandler.OK(c, "获取角色列表成功", result)
}

// CreateRole 创建角色
// @Summary 创建角色
// @Description 管理员创建新角色
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param role body services.CreateRoleRequest true "角色信息"
// @Success 201 {object} StandardResponse{data=models.Role} "创建成功"
// @Failure 400 {object} StandardResponse "请求参数错误"
// @Failure 401 {object} StandardResponse "未授权"
// @Failure 409 {object} StandardResponse "角色已存在"
// @Failure 500 {object} StandardResponse "服务器内部错误"
// @Router /api/v1/roles [post]
func (h *RoleHandler) CreateRole(c *gin.Context) {
	var req services.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("创建角色请求参数无效", "error", err)
		h.respHandler.BadRequest(c, "请求参数无效", nil)
		return
	}

	// 从上下文获取租户信息
	tenantID, _ := c.Get("tenant_id")
	tenantUUID := tenantID.(uuid.UUID)
	req.TenantID = tenantUUID

	// 创建角色
	role, err := h.roleService.CreateRole(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("创建角色失败", "name", req.Name, "error", err)

		if contains(err.Error(), "已存在") {
			h.respHandler.Conflict(c, err.Error())
		} else {
			h.respHandler.InternalServerError(c, err.Error())
		}
		return
	}

	h.logger.Info("角色创建成功", "name", req.Name, "role_id", role.ID)
	h.respHandler.Created(c, "角色创建成功", role)
}

// GetRole 获取单个角色
// @Summary 获取角色详情
// @Description 根据角色ID获取角色详细信息
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "角色ID"
// @Success 200 {object} StandardResponse{data=models.Role} "获取成功"
// @Failure 400 {object} StandardResponse "请求参数错误"
// @Failure 401 {object} StandardResponse "未授权"
// @Failure 404 {object} StandardResponse "角色不存在"
// @Failure 500 {object} StandardResponse "服务器内部错误"
// @Router /api/v1/roles/{id} [get]
func (h *RoleHandler) GetRole(c *gin.Context) {
	// 解析角色ID
	roleIDStr := c.Param("id")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		h.respHandler.BadRequest(c, "角色ID格式错误", nil)
		return
	}

	// 从上下文获取租户信息
	tenantID, _ := c.Get("tenant_id")
	tenantUUID := tenantID.(uuid.UUID)

	// 获取角色信息
	role, err := h.roleService.GetRoleByID(c.Request.Context(), roleID, tenantUUID)
	if err != nil {
		h.logger.Error("获取角色信息失败", "role_id", roleID, "error", err)

		if contains(err.Error(), "不存在") {
			h.respHandler.NotFound(c, err.Error())
		} else {
			h.respHandler.InternalServerError(c, err.Error())
		}
		return
	}

	h.respHandler.OK(c, "获取角色信息成功", role)
}

// UpdateRole 更新角色
// @Summary 更新角色信息
// @Description 管理员更新角色信息
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "角色ID"
// @Param role body services.UpdateRoleRequest true "角色信息"
// @Success 200 {object} StandardResponse{data=models.Role} "更新成功"
// @Failure 400 {object} StandardResponse "请求参数错误"
// @Failure 401 {object} StandardResponse "未授权"
// @Failure 404 {object} StandardResponse "角色不存在"
// @Failure 500 {object} StandardResponse "服务器内部错误"
// @Router /api/v1/roles/{id} [put]
func (h *RoleHandler) UpdateRole(c *gin.Context) {
	// 解析角色ID
	roleIDStr := c.Param("id")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		h.respHandler.BadRequest(c, "角色ID格式错误", nil)
		return
	}

	var req services.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("更新角色请求参数无效", "error", err)
		h.respHandler.BadRequest(c, "请求参数无效", nil)
		return
	}

	// 从上下文获取租户信息
	tenantID, _ := c.Get("tenant_id")
	tenantUUID := tenantID.(uuid.UUID)

	// 更新角色信息
	role, err := h.roleService.UpdateRole(c.Request.Context(), roleID, tenantUUID, &req)
	if err != nil {
		h.logger.Error("更新角色信息失败", "role_id", roleID, "error", err)

		if contains(err.Error(), "不存在") {
			h.respHandler.NotFound(c, err.Error())
		} else if contains(err.Error(), "系统角色") {
			h.respHandler.BadRequest(c, err.Error(), nil)
		} else {
			h.respHandler.InternalServerError(c, err.Error())
		}
		return
	}

	h.logger.Info("角色信息更新成功", "role_id", roleID)
	h.respHandler.OK(c, "角色信息更新成功", role)
}

// DeleteRole 删除角色
// @Summary 删除角色
// @Description 管理员删除角色
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "角色ID"
// @Success 200 {object} StandardResponse "删除成功"
// @Failure 400 {object} StandardResponse "请求参数错误"
// @Failure 401 {object} StandardResponse "未授权"
// @Failure 404 {object} StandardResponse "角色不存在"
// @Failure 500 {object} StandardResponse "服务器内部错误"
// @Router /api/v1/roles/{id} [delete]
func (h *RoleHandler) DeleteRole(c *gin.Context) {
	// 解析角色ID
	roleIDStr := c.Param("id")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		h.respHandler.BadRequest(c, "角色ID格式错误", nil)
		return
	}

	// 从上下文获取租户信息
	tenantID, _ := c.Get("tenant_id")
	tenantUUID := tenantID.(uuid.UUID)

	// 删除角色
	err = h.roleService.DeleteRole(c.Request.Context(), roleID, tenantUUID)
	if err != nil {
		h.logger.Error("删除角色失败", "role_id", roleID, "error", err)

		if contains(err.Error(), "不存在") {
			h.respHandler.NotFound(c, err.Error())
		} else if contains(err.Error(), "系统角色") || contains(err.Error(), "正在使用") {
			h.respHandler.BadRequest(c, err.Error(), nil)
		} else {
			h.respHandler.InternalServerError(c, err.Error())
		}
		return
	}

	h.logger.Info("角色删除成功", "role_id", roleID)
	h.respHandler.OK(c, "角色删除成功", nil)
}
