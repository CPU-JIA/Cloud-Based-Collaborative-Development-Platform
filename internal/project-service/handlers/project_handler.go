package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/models"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/service"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/webhook"
	"github.com/cloud-platform/collaborative-dev/shared/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ProjectHandler 项目处理器
type ProjectHandler struct {
	service        service.ProjectService
	webhookHandler *webhook.WebhookHandler
	logger         *zap.Logger
}

// NewProjectHandler 创建项目处理器实例
func NewProjectHandler(service service.ProjectService, webhookHandler *webhook.WebhookHandler, logger *zap.Logger) *ProjectHandler {
	return &ProjectHandler{
		service:        service,
		webhookHandler: webhookHandler,
		logger:         logger,
	}
}

// CreateProject 创建项目
// @Summary 创建项目
// @Description 创建新的项目
// @Tags projects
// @Accept json
// @Produce json
// @Param request body models.CreateProjectRequest true "创建项目请求"
// @Success 201 {object} response.Response{data=models.Project}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/projects [post]
// @Security Bearer
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	var req models.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数无效", err)
		return
	}

	// 获取用户信息
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "未授权访问", nil)
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "租户信息缺失", nil)
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		response.Error(c, http.StatusInternalServerError, "用户ID格式错误", nil)
		return
	}

	tenantUUID, ok := tenantID.(uuid.UUID)
	if !ok {
		response.Error(c, http.StatusInternalServerError, "租户ID格式错误", nil)
		return
	}

	// 创建项目
	project, err := h.service.CreateProject(c.Request.Context(), &req, userUUID, tenantUUID)
	if err != nil {
		h.logger.Error("创建项目失败", zap.Error(err))
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusCreated, "项目创建成功", project)
}

// GetProject 获取项目详情
// @Summary 获取项目详情
// @Description 根据ID获取项目详情
// @Tags projects
// @Produce json
// @Param id path string true "项目ID"
// @Success 200 {object} response.Response{data=models.Project}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/projects/{id} [get]
// @Security Bearer
func (h *ProjectHandler) GetProject(c *gin.Context) {
	projectIDStr := c.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "项目ID格式无效", nil)
		return
	}

	userID, tenantID, err := h.getUserAndTenantID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	project, err := h.service.GetProject(c.Request.Context(), projectID, userID, tenantID)
	if err != nil {
		if err.Error() == "项目不存在" || err.Error() == "无权限访问此项目" {
			response.Error(c, http.StatusNotFound, err.Error(), nil)
		} else {
			response.Error(c, http.StatusInternalServerError, err.Error(), nil)
		}
		return
	}

	response.Success(c, http.StatusOK, "获取项目成功", project)
}

// GetProjectByKey 根据key获取项目
// @Summary 根据key获取项目
// @Description 根据项目key获取项目详情
// @Tags projects
// @Produce json
// @Param key path string true "项目Key"
// @Success 200 {object} response.Response{data=models.Project}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/projects/key/{key} [get]
// @Security Bearer
func (h *ProjectHandler) GetProjectByKey(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		response.Error(c, http.StatusBadRequest, "项目key不能为空", nil)
		return
	}

	userID, tenantID, err := h.getUserAndTenantID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	project, err := h.service.GetProjectByKey(c.Request.Context(), key, userID, tenantID)
	if err != nil {
		if err.Error() == "项目不存在" || err.Error() == "无权限访问此项目" {
			response.Error(c, http.StatusNotFound, err.Error(), nil)
		} else {
			response.Error(c, http.StatusInternalServerError, err.Error(), nil)
		}
		return
	}

	response.Success(c, http.StatusOK, "获取项目成功", project)
}

// UpdateProject 更新项目
// @Summary 更新项目
// @Description 更新项目信息
// @Tags projects
// @Accept json
// @Produce json
// @Param id path string true "项目ID"
// @Param request body models.UpdateProjectRequest true "更新项目请求"
// @Success 200 {object} response.Response{data=models.Project}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/projects/{id} [put]
// @Security Bearer
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	projectIDStr := c.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "项目ID格式无效", nil)
		return
	}

	var req models.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数无效", err)
		return
	}

	userID, tenantID, err := h.getUserAndTenantID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	project, err := h.service.UpdateProject(c.Request.Context(), projectID, &req, userID, tenantID)
	if err != nil {
		if err.Error() == "项目不存在" {
			response.Error(c, http.StatusNotFound, err.Error(), nil)
		} else if err.Error() == "无权限更新此项目" {
			response.Error(c, http.StatusForbidden, err.Error(), nil)
		} else {
			response.Error(c, http.StatusBadRequest, err.Error(), nil)
		}
		return
	}

	response.Success(c, http.StatusOK, "项目更新成功", project)
}

// DeleteProject 删除项目
// @Summary 删除项目
// @Description 删除指定项目
// @Tags projects
// @Produce json
// @Param id path string true "项目ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/projects/{id} [delete]
// @Security Bearer
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	projectIDStr := c.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "项目ID格式无效", nil)
		return
	}

	userID, tenantID, err := h.getUserAndTenantID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	err = h.service.DeleteProject(c.Request.Context(), projectID, userID, tenantID)
	if err != nil {
		if err.Error() == "项目不存在" {
			response.Error(c, http.StatusNotFound, err.Error(), nil)
		} else if err.Error() == "无权限删除此项目" {
			response.Error(c, http.StatusForbidden, err.Error(), nil)
		} else {
			response.Error(c, http.StatusInternalServerError, err.Error(), nil)
		}
		return
	}

	response.Success(c, http.StatusOK, "项目删除成功", nil)
}

// ListProjects 获取项目列表
// @Summary 获取项目列表
// @Description 获取用户有权限访问的项目列表
// @Tags projects
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param status query string false "项目状态" Enums(active, archived)
// @Param search query string false "搜索关键词"
// @Success 200 {object} response.Response{data=models.ProjectListResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/projects [get]
// @Security Bearer
func (h *ProjectHandler) ListProjects(c *gin.Context) {
	// 解析查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")
	search := c.Query("search")

	userID, tenantID, err := h.getUserAndTenantID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	// 构建过滤条件
	filters := make(map[string]interface{})
	if status != "" {
		filters["status"] = status
	}
	if search != "" {
		filters["search"] = search
	}

	result, err := h.service.ListProjects(c.Request.Context(), page, pageSize, filters, userID, tenantID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "获取项目列表成功", result)
}

// AddMember 添加项目成员
// @Summary 添加项目成员
// @Description 为项目添加新成员
// @Tags projects
// @Accept json
// @Produce json
// @Param id path string true "项目ID"
// @Param request body models.AddMemberRequest true "添加成员请求"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/projects/{id}/members [post]
// @Security Bearer
func (h *ProjectHandler) AddMember(c *gin.Context) {
	projectIDStr := c.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "项目ID格式无效", nil)
		return
	}

	var req models.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数无效", err)
		return
	}

	userID, tenantID, err := h.getUserAndTenantID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	err = h.service.AddMember(c.Request.Context(), projectID, &req, userID, tenantID)
	if err != nil {
		if err.Error() == "项目不存在" {
			response.Error(c, http.StatusNotFound, err.Error(), nil)
		} else if err.Error() == "无权限添加项目成员" {
			response.Error(c, http.StatusForbidden, err.Error(), nil)
		} else {
			response.Error(c, http.StatusBadRequest, err.Error(), nil)
		}
		return
	}

	response.Success(c, http.StatusOK, "成员添加成功", nil)
}

// RemoveMember 移除项目成员
// @Summary 移除项目成员
// @Description 从项目中移除成员
// @Tags projects
// @Produce json
// @Param id path string true "项目ID"
// @Param user_id path string true "用户ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/projects/{id}/members/{user_id} [delete]
// @Security Bearer
func (h *ProjectHandler) RemoveMember(c *gin.Context) {
	projectIDStr := c.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "项目ID格式无效", nil)
		return
	}

	memberUserIDStr := c.Param("user_id")
	memberUserID, err := uuid.Parse(memberUserIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "用户ID格式无效", nil)
		return
	}

	userID, tenantID, err := h.getUserAndTenantID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	err = h.service.RemoveMember(c.Request.Context(), projectID, memberUserID, userID, tenantID)
	if err != nil {
		if err.Error() == "项目不存在" || err.Error() == "成员不存在" {
			response.Error(c, http.StatusNotFound, err.Error(), nil)
		} else if err.Error() == "无权限移除项目成员" || err.Error() == "不能移除自己" {
			response.Error(c, http.StatusForbidden, err.Error(), nil)
		} else {
			response.Error(c, http.StatusInternalServerError, err.Error(), nil)
		}
		return
	}

	response.Success(c, http.StatusOK, "成员移除成功", nil)
}

// GetMembers 获取项目成员列表
// @Summary 获取项目成员列表
// @Description 获取指定项目的成员列表
// @Tags projects
// @Produce json
// @Param id path string true "项目ID"
// @Success 200 {object} response.Response{data=[]models.ProjectMember}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/projects/{id}/members [get]
// @Security Bearer
func (h *ProjectHandler) GetMembers(c *gin.Context) {
	projectIDStr := c.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "项目ID格式无效", nil)
		return
	}

	userID, tenantID, err := h.getUserAndTenantID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	members, err := h.service.GetMembers(c.Request.Context(), projectID, userID, tenantID)
	if err != nil {
		if err.Error() == "无权限访问此项目" {
			response.Error(c, http.StatusForbidden, err.Error(), nil)
		} else {
			response.Error(c, http.StatusInternalServerError, err.Error(), nil)
		}
		return
	}

	response.Success(c, http.StatusOK, "获取成员列表成功", members)
}

// GetUserProjects 获取用户参与的项目列表
// @Summary 获取用户项目列表
// @Description 获取当前用户参与的所有项目
// @Tags projects
// @Produce json
// @Success 200 {object} response.Response{data=[]models.Project}
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/projects/my [get]
// @Security Bearer
func (h *ProjectHandler) GetUserProjects(c *gin.Context) {
	userID, tenantID, err := h.getUserAndTenantID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	projects, err := h.service.GetUserProjects(c.Request.Context(), userID, tenantID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "获取用户项目列表成功", projects)
}

// 辅助方法：获取用户ID和租户ID
func (h *ProjectHandler) getUserAndTenantID(c *gin.Context) (userID, tenantID uuid.UUID, err error) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		err = fmt.Errorf("未授权访问")
		return
	}

	tenantIDInterface, exists := c.Get("tenant_id")
	if !exists {
		err = fmt.Errorf("租户信息缺失")
		return
	}

	var ok bool
	userID, ok = userIDInterface.(uuid.UUID)
	if !ok {
		err = fmt.Errorf("用户ID格式错误")
		return
	}

	tenantID, ok = tenantIDInterface.(uuid.UUID)
	if !ok {
		err = fmt.Errorf("租户ID格式错误")
		return
	}

	return userID, tenantID, nil
}

// Git仓库管理处理器

// CreateRepository 创建Git仓库
// @Summary 创建Git仓库
// @Description 在指定项目中创建新的Git仓库
// @Tags repositories
// @Accept json
// @Produce json
// @Param id path string true "项目ID"
// @Param request body service.CreateRepositoryRequest true "创建仓库请求"
// @Success 201 {object} response.Response{data=models.Repository}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/projects/{id}/repositories [post]
// @Security Bearer
func (h *ProjectHandler) CreateRepository(c *gin.Context) {
	projectIDStr := c.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "项目ID格式无效", nil)
		return
	}

	var req service.CreateRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数无效", err)
		return
	}

	userID, tenantID, err := h.getUserAndTenantID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	repository, err := h.service.CreateRepository(c.Request.Context(), projectID, &req, userID, tenantID)
	if err != nil {
		h.logger.Error("创建仓库失败",
			zap.Error(err),
			zap.String("project_id", projectID.String()),
			zap.String("repository_name", req.Name))

		if err.Error() == "无权限在此项目中创建仓库" {
			response.Error(c, http.StatusForbidden, err.Error(), nil)
		} else {
			response.Error(c, http.StatusBadRequest, err.Error(), nil)
		}
		return
	}

	response.Success(c, http.StatusCreated, "仓库创建成功", repository)
}

// GetRepository 获取仓库详情
// @Summary 获取仓库详情
// @Description 根据ID获取Git仓库详情
// @Tags repositories
// @Produce json
// @Param repository_id path string true "仓库ID"
// @Success 200 {object} response.Response{data=models.Repository}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/repositories/{repository_id} [get]
// @Security Bearer
func (h *ProjectHandler) GetRepository(c *gin.Context) {
	repositoryIDStr := c.Param("repository_id")
	repositoryID, err := uuid.Parse(repositoryIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "仓库ID格式无效", nil)
		return
	}

	userID, tenantID, err := h.getUserAndTenantID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	repository, err := h.service.GetRepository(c.Request.Context(), repositoryID, userID, tenantID)
	if err != nil {
		if err.Error() == "无权限访问此仓库" {
			response.Error(c, http.StatusForbidden, err.Error(), nil)
		} else if err.Error() == "获取仓库信息失败: repository not found" {
			response.Error(c, http.StatusNotFound, "仓库不存在", nil)
		} else {
			response.Error(c, http.StatusInternalServerError, err.Error(), nil)
		}
		return
	}

	response.Success(c, http.StatusOK, "获取仓库成功", repository)
}

// ListRepositories 获取项目仓库列表
// @Summary 获取项目仓库列表
// @Description 获取指定项目的Git仓库列表
// @Tags repositories
// @Produce json
// @Param id path string true "项目ID"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} response.Response{data=service.RepositoryListResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /api/v1/projects/{id}/repositories [get]
// @Security Bearer
func (h *ProjectHandler) ListRepositories(c *gin.Context) {
	projectIDStr := c.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "项目ID格式无效", nil)
		return
	}

	// 解析查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	userID, tenantID, err := h.getUserAndTenantID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	result, err := h.service.ListRepositories(c.Request.Context(), projectID, page, pageSize, userID, tenantID)
	if err != nil {
		if err.Error() == "无权限访问此项目的仓库" {
			response.Error(c, http.StatusForbidden, err.Error(), nil)
		} else {
			response.Error(c, http.StatusInternalServerError, err.Error(), nil)
		}
		return
	}

	response.Success(c, http.StatusOK, "获取仓库列表成功", result)
}

// UpdateRepository 更新仓库
// @Summary 更新仓库
// @Description 更新Git仓库信息
// @Tags repositories
// @Accept json
// @Produce json
// @Param repository_id path string true "仓库ID"
// @Param request body service.UpdateRepositoryRequest true "更新仓库请求"
// @Success 200 {object} response.Response{data=models.Repository}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/repositories/{repository_id} [put]
// @Security Bearer
func (h *ProjectHandler) UpdateRepository(c *gin.Context) {
	repositoryIDStr := c.Param("repository_id")
	repositoryID, err := uuid.Parse(repositoryIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "仓库ID格式无效", nil)
		return
	}

	var req service.UpdateRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数无效", err)
		return
	}

	userID, tenantID, err := h.getUserAndTenantID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	repository, err := h.service.UpdateRepository(c.Request.Context(), repositoryID, &req, userID, tenantID)
	if err != nil {
		if err.Error() == "无权限更新此仓库" {
			response.Error(c, http.StatusForbidden, err.Error(), nil)
		} else if err.Error() == "获取仓库信息失败: repository not found" {
			response.Error(c, http.StatusNotFound, "仓库不存在", nil)
		} else {
			response.Error(c, http.StatusBadRequest, err.Error(), nil)
		}
		return
	}

	response.Success(c, http.StatusOK, "仓库更新成功", repository)
}

// DeleteRepository 删除仓库
// @Summary 删除仓库
// @Description 删除指定的Git仓库
// @Tags repositories
// @Produce json
// @Param repository_id path string true "仓库ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/repositories/{repository_id} [delete]
// @Security Bearer
func (h *ProjectHandler) DeleteRepository(c *gin.Context) {
	repositoryIDStr := c.Param("repository_id")
	repositoryID, err := uuid.Parse(repositoryIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "仓库ID格式无效", nil)
		return
	}

	userID, tenantID, err := h.getUserAndTenantID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	err = h.service.DeleteRepository(c.Request.Context(), repositoryID, userID, tenantID)
	if err != nil {
		if err.Error() == "无权限删除此仓库" {
			response.Error(c, http.StatusForbidden, err.Error(), nil)
		} else if err.Error() == "获取仓库信息失败: repository not found" {
			response.Error(c, http.StatusNotFound, "仓库不存在", nil)
		} else {
			response.Error(c, http.StatusInternalServerError, err.Error(), nil)
		}
		return
	}

	response.Success(c, http.StatusOK, "仓库删除成功", nil)
}

// Git Webhook处理器

// HandleGitWebhook 处理Git webhook事件
// @Summary 处理Git webhook事件
// @Description 接收并处理来自Git网关的webhook事件
// @Tags webhooks
// @Accept json
// @Produce json
// @Param payload body webhook.GitEvent true "Git事件负载"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/webhooks/git [post]
func (h *ProjectHandler) HandleGitWebhook(c *gin.Context) {
	if h.webhookHandler == nil {
		response.Error(c, http.StatusInternalServerError, "Webhook处理器未初始化", nil)
		return
	}

	h.webhookHandler.HandleWebhook(c)
}

// GetWebhookHealth webhook健康检查
// @Summary Webhook健康检查
// @Description 检查webhook处理服务的健康状态
// @Tags webhooks
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/webhooks/health [get]
func (h *ProjectHandler) GetWebhookHealth(c *gin.Context) {
	if h.webhookHandler == nil {
		response.Error(c, http.StatusInternalServerError, "Webhook处理器未初始化", nil)
		return
	}

	h.webhookHandler.GetHealthCheck(c)
}
