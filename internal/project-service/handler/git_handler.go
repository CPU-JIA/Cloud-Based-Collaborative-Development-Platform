package handler

import (
	"net/http"
	"strconv"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// GitHandler Git操作处理器
type GitHandler struct {
	projectService service.ProjectService
	logger         *zap.Logger
}

// NewGitHandler 创建Git操作处理器
func NewGitHandler(projectService service.ProjectService, logger *zap.Logger) *GitHandler {
	return &GitHandler{
		projectService: projectService,
		logger:         logger,
	}
}

// 仓库管理

// CreateRepository 创建仓库
func (h *GitHandler) CreateRepository(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目ID"})
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	var req service.CreateRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	repository, err := h.projectService.CreateRepository(c.Request.Context(), projectID, &req, userID, tenantID)
	if err != nil {
		h.logger.Error("创建仓库失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": repository})
}

// GetRepository 获取仓库详情
func (h *GitHandler) GetRepository(c *gin.Context) {
	repositoryIDStr := c.Param("repositoryId")
	repositoryID, err := uuid.Parse(repositoryIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的仓库ID"})
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	repository, err := h.projectService.GetRepository(c.Request.Context(), repositoryID, userID, tenantID)
	if err != nil {
		h.logger.Error("获取仓库失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": repository})
}

// ListRepositories 获取项目仓库列表
func (h *GitHandler) ListRepositories(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目ID"})
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	repositories, err := h.projectService.ListRepositories(c.Request.Context(), projectID, page, pageSize, userID, tenantID)
	if err != nil {
		h.logger.Error("获取仓库列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": repositories})
}

// UpdateRepository 更新仓库
func (h *GitHandler) UpdateRepository(c *gin.Context) {
	repositoryIDStr := c.Param("repositoryId")
	repositoryID, err := uuid.Parse(repositoryIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的仓库ID"})
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	var req service.UpdateRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	repository, err := h.projectService.UpdateRepository(c.Request.Context(), repositoryID, &req, userID, tenantID)
	if err != nil {
		h.logger.Error("更新仓库失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": repository})
}

// DeleteRepository 删除仓库
func (h *GitHandler) DeleteRepository(c *gin.Context) {
	repositoryIDStr := c.Param("repositoryId")
	repositoryID, err := uuid.Parse(repositoryIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的仓库ID"})
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	err = h.projectService.DeleteRepository(c.Request.Context(), repositoryID, userID, tenantID)
	if err != nil {
		h.logger.Error("删除仓库失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "仓库删除成功"})
}

// 分支管理

// CreateBranch 创建分支
func (h *GitHandler) CreateBranch(c *gin.Context) {
	repositoryIDStr := c.Param("repositoryId")
	repositoryID, err := uuid.Parse(repositoryIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的仓库ID"})
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	var req service.CreateBranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	branch, err := h.projectService.CreateBranch(c.Request.Context(), repositoryID, &req, userID, tenantID)
	if err != nil {
		h.logger.Error("创建分支失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": branch})
}

// ListBranches 获取分支列表
func (h *GitHandler) ListBranches(c *gin.Context) {
	repositoryIDStr := c.Param("repositoryId")
	repositoryID, err := uuid.Parse(repositoryIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的仓库ID"})
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	branches, err := h.projectService.ListBranches(c.Request.Context(), repositoryID, page, pageSize, userID, tenantID)
	if err != nil {
		h.logger.Error("获取分支列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": branches})
}

// DeleteBranch 删除分支
func (h *GitHandler) DeleteBranch(c *gin.Context) {
	repositoryIDStr := c.Param("repositoryId")
	repositoryID, err := uuid.Parse(repositoryIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的仓库ID"})
		return
	}

	branchName := c.Param("branchName")
	if branchName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "分支名称不能为空"})
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	err = h.projectService.DeleteBranch(c.Request.Context(), repositoryID, branchName, userID, tenantID)
	if err != nil {
		h.logger.Error("删除分支失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "分支删除成功"})
}

// 合并请求管理

// CreatePullRequest 创建合并请求
func (h *GitHandler) CreatePullRequest(c *gin.Context) {
	repositoryIDStr := c.Param("repositoryId")
	repositoryID, err := uuid.Parse(repositoryIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的仓库ID"})
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	var req service.CreatePullRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	pullRequest, err := h.projectService.CreatePullRequest(c.Request.Context(), repositoryID, &req, userID, tenantID)
	if err != nil {
		h.logger.Error("创建合并请求失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": pullRequest})
}

// GetPullRequest 获取合并请求详情
func (h *GitHandler) GetPullRequest(c *gin.Context) {
	repositoryIDStr := c.Param("repositoryId")
	repositoryID, err := uuid.Parse(repositoryIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的仓库ID"})
		return
	}

	pullRequestIDStr := c.Param("pullRequestId")
	pullRequestID, err := uuid.Parse(pullRequestIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的合并请求ID"})
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	pullRequest, err := h.projectService.GetPullRequest(c.Request.Context(), repositoryID, pullRequestID, userID, tenantID)
	if err != nil {
		h.logger.Error("获取合并请求失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": pullRequest})
}

// 工具函数

// 工具函数已在agile_handler.go中定义，避免重复定义
