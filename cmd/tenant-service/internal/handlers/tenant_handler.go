package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cloud-platform/collaborative-dev/cmd/tenant-service/internal/services"
	"github.com/cloud-platform/collaborative-dev/shared/api"
	"github.com/cloud-platform/collaborative-dev/shared/logger"
	"github.com/cloud-platform/collaborative-dev/shared/models"
)

// TenantHandler 租户处理器
type TenantHandler struct {
	tenantService services.TenantService
	logger        logger.Logger
	respHandler   *api.ResponseHandler
}

// NewTenantHandler 创建租户处理器
func NewTenantHandler(tenantService services.TenantService, logger logger.Logger) *TenantHandler {
	return &TenantHandler{
		tenantService: tenantService,
		logger:        logger,
		respHandler:   api.NewResponseHandler(),
	}
}

// CreateTenant 创建租户
// @Summary 创建租户
// @Description 创建新的租户
// @Tags tenants
// @Accept json
// @Produce json
// @Param tenant body services.CreateTenantRequest true "创建租户请求"
// @Success 201 {object} APIResponse{data=models.Tenant}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/tenants [post]
func (h *TenantHandler) CreateTenant(c *gin.Context) {
	var req services.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respHandler.ValidationError(c, "请求参数无效", map[string]string{
			"validation": err.Error(),
		})
		return
	}

	tenant, err := h.tenantService.CreateTenant(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("创建租户失败", "error", err, "request", req)
		h.respHandler.InternalServerError(c, err.Error())
		return
	}

	h.logger.Info("租户创建成功", "tenant_id", tenant.ID, "domain", tenant.Domain)
	h.respHandler.Created(c, "租户创建成功", tenant)
}

// GetTenant 获取租户详情
// @Summary 获取租户详情
// @Description 根据ID获取租户详细信息
// @Tags tenants
// @Produce json
// @Param id path string true "租户ID"
// @Success 200 {object} APIResponse{data=models.Tenant}
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/tenants/{id} [get]
func (h *TenantHandler) GetTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respHandler.BadRequest(c, "租户ID格式无效", nil)
		return
	}

	tenant, err := h.tenantService.GetTenant(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("获取租户失败", "error", err, "tenant_id", id)
		h.respHandler.NotFound(c, err.Error())
		return
	}

	h.respHandler.OK(c, "获取成功", tenant)
}

// GetTenantByDomain 根据域名获取租户
// @Summary 根据域名获取租户
// @Description 根据域名获取租户信息
// @Tags tenants
// @Produce json
// @Param domain query string true "域名"
// @Success 200 {object} APIResponse{data=models.Tenant}
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/tenants/by-domain [get]
func (h *TenantHandler) GetTenantByDomain(c *gin.Context) {
	domain := c.Query("domain")
	if domain == "" {
		h.respHandler.BadRequest(c, "域名参数不能为空", nil)
		return
	}

	tenant, err := h.tenantService.GetTenantByDomain(c.Request.Context(), domain)
	if err != nil {
		h.logger.Error("根据域名获取租户失败", "error", err, "domain", domain)
		h.respHandler.NotFound(c, err.Error())
		return
	}

	h.respHandler.OK(c, "获取成功", tenant)
}

// UpdateTenant 更新租户
// @Summary 更新租户
// @Description 更新租户信息
// @Tags tenants
// @Accept json
// @Produce json
// @Param id path string true "租户ID"
// @Param tenant body services.UpdateTenantRequest true "更新租户请求"
// @Success 200 {object} APIResponse{data=models.Tenant}
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/tenants/{id} [put]
func (h *TenantHandler) UpdateTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respHandler.BadRequest(c, "租户ID格式无效", nil)
		return
	}

	var req services.UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respHandler.ValidationError(c, "请求参数无效", map[string]string{
			"validation": err.Error(),
		})
		return
	}

	tenant, err := h.tenantService.UpdateTenant(c.Request.Context(), id, &req)
	if err != nil {
		h.logger.Error("更新租户失败", "error", err, "tenant_id", id)
		h.respHandler.InternalServerError(c, err.Error())
		return
	}

	h.logger.Info("租户更新成功", "tenant_id", tenant.ID)
	h.respHandler.OK(c, "更新成功", tenant)
}

// DeleteTenant 删除租户
// @Summary 删除租户
// @Description 软删除租户
// @Tags tenants
// @Produce json
// @Param id path string true "租户ID"
// @Success 204
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/tenants/{id} [delete]
func (h *TenantHandler) DeleteTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respHandler.BadRequest(c, "租户ID格式无效", nil)
		return
	}

	if err := h.tenantService.DeleteTenant(c.Request.Context(), id); err != nil {
		h.logger.Error("删除租户失败", "error", err, "tenant_id", id)
		h.respHandler.InternalServerError(c, err.Error())
		return
	}

	h.logger.Info("租户删除成功", "tenant_id", id)
	c.Status(http.StatusNoContent)
}

// ListTenants 获取租户列表
// @Summary 获取租户列表
// @Description 分页获取租户列表，支持按状态和计划过滤
// @Tags tenants
// @Produce json
// @Param status query string false "状态过滤" Enums(active,suspended,pending,deleted)
// @Param plan query string false "计划过滤" Enums(free,basic,professional,enterprise)
// @Param offset query int false "偏移量" default(0)
// @Param limit query int false "限制数量" default(20) maximum(100)
// @Success 200 {object} APIResponse{data=services.ListTenantsResponse}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/tenants [get]
func (h *TenantHandler) ListTenants(c *gin.Context) {
	req := &services.ListTenantsRequest{}

	// 解析查询参数
	if status := c.Query("status"); status != "" {
		req.Status = models.TenantStatus(status)
	}

	if plan := c.Query("plan"); plan != "" {
		req.Plan = models.TenantPlan(plan)
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			req.Offset = offset
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			req.Limit = limit
		}
	}

	// 设置默认值
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	response, err := h.tenantService.ListTenants(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("获取租户列表失败", "error", err, "request", req)
		h.respHandler.InternalServerError(c, err.Error())
		return
	}

	h.respHandler.OK(c, "获取成功", response)
}

// SearchTenants 搜索租户
// @Summary 搜索租户
// @Description 根据关键词搜索租户
// @Tags tenants
// @Produce json
// @Param q query string true "搜索关键词"
// @Param offset query int false "偏移量" default(0)
// @Param limit query int false "限制数量" default(20) maximum(100)
// @Success 200 {object} APIResponse{data=[]models.Tenant}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/tenants/search [get]
func (h *TenantHandler) SearchTenants(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		h.respHandler.BadRequest(c, "搜索关键词不能为空", nil)
		return
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	tenants, err := h.tenantService.SearchTenants(c.Request.Context(), query, offset, limit)
	if err != nil {
		h.logger.Error("搜索租户失败", "error", err, "query", query)
		h.respHandler.InternalServerError(c, err.Error())
		return
	}

	h.respHandler.OK(c, "搜索成功", tenants)
}

// GetTenantStats 获取租户统计
// @Summary 获取租户统计
// @Description 获取租户统计信息
// @Tags tenants
// @Produce json
// @Success 200 {object} APIResponse{data=services.TenantStatsResponse}
// @Failure 500 {object} APIResponse
// @Router /api/v1/tenants/stats [get]
func (h *TenantHandler) GetTenantStats(c *gin.Context) {
	stats, err := h.tenantService.GetTenantStats(c.Request.Context())
	if err != nil {
		h.logger.Error("获取租户统计失败", "error", err)
		h.respHandler.InternalServerError(c, err.Error())
		return
	}

	h.respHandler.OK(c, "获取成功", stats)
}

// ActivateTenant 激活租户
// @Summary 激活租户
// @Description 激活指定租户
// @Tags tenants
// @Produce json
// @Param id path string true "租户ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/tenants/{id}/activate [post]
func (h *TenantHandler) ActivateTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respHandler.BadRequest(c, "租户ID格式无效", nil)
		return
	}

	if err := h.tenantService.ActivateTenant(c.Request.Context(), id); err != nil {
		h.logger.Error("激活租户失败", "error", err, "tenant_id", id)
		h.respHandler.InternalServerError(c, err.Error())
		return
	}

	h.logger.Info("租户激活成功", "tenant_id", id)
	h.respHandler.OK(c, "激活成功", nil)
}

// SuspendTenant 暂停租户
// @Summary 暂停租户
// @Description 暂停指定租户
// @Tags tenants
// @Accept json
// @Produce json
// @Param id path string true "租户ID"
// @Param body body SuspendTenantRequest true "暂停原因"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/tenants/{id}/suspend [post]
func (h *TenantHandler) SuspendTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respHandler.BadRequest(c, "租户ID格式无效", nil)
		return
	}

	var req SuspendTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respHandler.ValidationError(c, "请求参数无效", map[string]string{
			"validation": err.Error(),
		})
		return
	}

	if err := h.tenantService.SuspendTenant(c.Request.Context(), id, req.Reason); err != nil {
		h.logger.Error("暂停租户失败", "error", err, "tenant_id", id)
		h.respHandler.InternalServerError(c, err.Error())
		return
	}

	h.logger.Info("租户暂停成功", "tenant_id", id, "reason", req.Reason)
	h.respHandler.OK(c, "暂停成功", nil)
}

// GetTenantWithConfig 获取租户及配置
// @Summary 获取租户及配置
// @Description 获取租户详细信息，包括配置
// @Tags tenants
// @Produce json
// @Param id path string true "租户ID"
// @Success 200 {object} APIResponse{data=models.Tenant}
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/tenants/{id}/with-config [get]
func (h *TenantHandler) GetTenantWithConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respHandler.BadRequest(c, "租户ID格式无效", nil)
		return
	}

	tenant, err := h.tenantService.GetTenantWithConfig(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("获取租户配置失败", "error", err, "tenant_id", id)
		h.respHandler.NotFound(c, err.Error())
		return
	}

	h.respHandler.OK(c, "获取成功", tenant)
}

// GetTenantWithAll 获取租户完整信息
// @Summary 获取租户完整信息
// @Description 获取租户详细信息，包括所有关联数据
// @Tags tenants
// @Produce json
// @Param id path string true "租户ID"
// @Success 200 {object} APIResponse{data=models.Tenant}
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/tenants/{id}/complete [get]
func (h *TenantHandler) GetTenantWithAll(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respHandler.BadRequest(c, "租户ID格式无效", nil)
		return
	}

	tenant, err := h.tenantService.GetTenantWithAll(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("获取租户完整信息失败", "error", err, "tenant_id", id)
		h.respHandler.NotFound(c, err.Error())
		return
	}

	h.respHandler.OK(c, "获取成功", tenant)
}

// SuspendTenantRequest 暂停租户请求
type SuspendTenantRequest struct {
	Reason string `json:"reason" binding:"required"`
}
