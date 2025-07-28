package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/cloud-platform/collaborative-dev/internal/git-gateway/models"
	"github.com/cloud-platform/collaborative-dev/internal/git-gateway/service"
	"github.com/cloud-platform/collaborative-dev/shared/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// GitHandler Git网关HTTP处理器
type GitHandler struct {
	gitService service.GitService
	logger     *zap.Logger
}

// NewGitHandler 创建Git网关处理器
func NewGitHandler(gitService service.GitService, logger *zap.Logger) *GitHandler {
	return &GitHandler{
		gitService: gitService,
		logger:     logger,
	}
}

// 仓库管理处理器

// CreateRepository 创建仓库
func (h *GitHandler) CreateRepository(c *gin.Context) {
	var req models.CreateRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// 从JWT中获取用户ID（这里简化处理）
	userIDStr, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	repository, err := h.gitService.CreateRepository(c.Request.Context(), &req, userID)
	if err != nil {
		h.logger.Error("Failed to create repository", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to create repository", err)
		return
	}

	response.Success(c, http.StatusCreated, "Repository created successfully", repository)
}

// GetRepository 获取仓库详情
func (h *GitHandler) GetRepository(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid repository ID", err)
		return
	}

	repository, err := h.gitService.GetRepository(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get repository", zap.Error(err))
		response.Error(c, http.StatusNotFound, "Repository not found", err)
		return
	}

	response.Success(c, http.StatusOK, "Repository retrieved successfully", repository)
}

// ListRepositories 获取仓库列表
func (h *GitHandler) ListRepositories(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var projectID *uuid.UUID
	if projectIDStr := c.Query("project_id"); projectIDStr != "" {
		if pid, err := uuid.Parse(projectIDStr); err == nil {
			projectID = &pid
		}
	}

	resp, err := h.gitService.ListRepositories(c.Request.Context(), projectID, page, pageSize)
	if err != nil {
		h.logger.Error("Failed to list repositories", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to list repositories", err)
		return
	}

	response.Success(c, http.StatusOK, "Repositories retrieved successfully", resp)
}

// UpdateRepository 更新仓库
func (h *GitHandler) UpdateRepository(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid repository ID", err)
		return
	}

	var req models.UpdateRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	repository, err := h.gitService.UpdateRepository(c.Request.Context(), id, &req)
	if err != nil {
		h.logger.Error("Failed to update repository", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to update repository", err)
		return
	}

	response.Success(c, http.StatusOK, "Repository updated successfully", repository)
}

// DeleteRepository 删除仓库
func (h *GitHandler) DeleteRepository(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid repository ID", err)
		return
	}

	if err := h.gitService.DeleteRepository(c.Request.Context(), id); err != nil {
		h.logger.Error("Failed to delete repository", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to delete repository", err)
		return
	}

	response.Success(c, http.StatusOK, "Repository deleted successfully", nil)
}

// GetRepositoryStats 获取仓库统计信息
func (h *GitHandler) GetRepositoryStats(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid repository ID", err)
		return
	}

	stats, err := h.gitService.GetRepositoryStats(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get repository stats", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to get repository stats", err)
		return
	}

	response.Success(c, http.StatusOK, "Repository stats retrieved successfully", stats)
}

// SearchRepositories 搜索仓库
func (h *GitHandler) SearchRepositories(c *gin.Context) {
	query := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var projectID *uuid.UUID
	if projectIDStr := c.Query("project_id"); projectIDStr != "" {
		if pid, err := uuid.Parse(projectIDStr); err == nil {
			projectID = &pid
		}
	}

	resp, err := h.gitService.SearchRepositories(c.Request.Context(), query, projectID, page, pageSize)
	if err != nil {
		h.logger.Error("Failed to search repositories", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to search repositories", err)
		return
	}

	response.Success(c, http.StatusOK, "Repositories searched successfully", resp)
}

// 分支管理处理器

// CreateBranch 创建分支
func (h *GitHandler) CreateBranch(c *gin.Context) {
	idStr := c.Param("id")
	repositoryID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid repository ID", err)
		return
	}

	var req models.CreateBranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	branch, err := h.gitService.CreateBranch(c.Request.Context(), repositoryID, &req)
	if err != nil {
		h.logger.Error("Failed to create branch", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to create branch", err)
		return
	}

	response.Success(c, http.StatusCreated, "Branch created successfully", branch)
}

// ListBranches 获取分支列表
func (h *GitHandler) ListBranches(c *gin.Context) {
	idStr := c.Param("id")
	repositoryID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid repository ID", err)
		return
	}

	branches, err := h.gitService.ListBranches(c.Request.Context(), repositoryID)
	if err != nil {
		h.logger.Error("Failed to list branches", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to list branches", err)
		return
	}

	response.Success(c, http.StatusOK, "Branches retrieved successfully", branches)
}

// GetBranch 获取分支详情
func (h *GitHandler) GetBranch(c *gin.Context) {
	idStr := c.Param("id")
	repositoryID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid repository ID", err)
		return
	}

	branchName := c.Param("branch")
	if branchName == "" {
		response.Error(c, http.StatusBadRequest, "Branch name is required", nil)
		return
	}

	branch, err := h.gitService.GetBranch(c.Request.Context(), repositoryID, branchName)
	if err != nil {
		h.logger.Error("Failed to get branch", zap.Error(err))
		response.Error(c, http.StatusNotFound, "Branch not found", err)
		return
	}

	response.Success(c, http.StatusOK, "Branch retrieved successfully", branch)
}

// DeleteBranch 删除分支
func (h *GitHandler) DeleteBranch(c *gin.Context) {
	idStr := c.Param("id")
	repositoryID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid repository ID", err)
		return
	}

	branchName := c.Param("branch")
	if branchName == "" {
		response.Error(c, http.StatusBadRequest, "Branch name is required", nil)
		return
	}

	if err := h.gitService.DeleteBranch(c.Request.Context(), repositoryID, branchName); err != nil {
		h.logger.Error("Failed to delete branch", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to delete branch", err)
		return
	}

	response.Success(c, http.StatusOK, "Branch deleted successfully", nil)
}

// SetDefaultBranch 设置默认分支
func (h *GitHandler) SetDefaultBranch(c *gin.Context) {
	idStr := c.Param("id")
	repositoryID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid repository ID", err)
		return
	}

	var req struct {
		BranchName string `json:"branch_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := h.gitService.SetDefaultBranch(c.Request.Context(), repositoryID, req.BranchName); err != nil {
		h.logger.Error("Failed to set default branch", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to set default branch", err)
		return
	}

	response.Success(c, http.StatusOK, "Default branch set successfully", nil)
}

// MergeBranch 合并分支
func (h *GitHandler) MergeBranch(c *gin.Context) {
	idStr := c.Param("id")
	repositoryID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid repository ID", err)
		return
	}

	var req struct {
		TargetBranch string `json:"target_branch" binding:"required"`
		SourceBranch string `json:"source_branch" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := h.gitService.MergeBranch(c.Request.Context(), repositoryID, req.TargetBranch, req.SourceBranch); err != nil {
		h.logger.Error("Failed to merge branch", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to merge branch", err)
		return
	}

	response.Success(c, http.StatusOK, "Branch merged successfully", nil)
}

// 提交管理处理器

// CreateCommit 创建提交
func (h *GitHandler) CreateCommit(c *gin.Context) {
	idStr := c.Param("id")
	repositoryID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid repository ID", err)
		return
	}

	var req models.CreateCommitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	commit, err := h.gitService.CreateCommit(c.Request.Context(), repositoryID, &req)
	if err != nil {
		h.logger.Error("Failed to create commit", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to create commit", err)
		return
	}

	response.Success(c, http.StatusCreated, "Commit created successfully", commit)
}

// ListCommits 获取提交列表
func (h *GitHandler) ListCommits(c *gin.Context) {
	idStr := c.Param("id")
	repositoryID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid repository ID", err)
		return
	}

	branch := c.Query("branch")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	resp, err := h.gitService.ListCommits(c.Request.Context(), repositoryID, branch, page, pageSize)
	if err != nil {
		h.logger.Error("Failed to list commits", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to list commits", err)
		return
	}

	response.Success(c, http.StatusOK, "Commits retrieved successfully", resp)
}

// GetCommit 获取提交详情
func (h *GitHandler) GetCommit(c *gin.Context) {
	idStr := c.Param("id")
	repositoryID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid repository ID", err)
		return
	}

	sha := c.Param("sha")
	if sha == "" {
		response.Error(c, http.StatusBadRequest, "Commit SHA is required", nil)
		return
	}

	commit, err := h.gitService.GetCommit(c.Request.Context(), repositoryID, sha)
	if err != nil {
		h.logger.Error("Failed to get commit", zap.Error(err))
		response.Error(c, http.StatusNotFound, "Commit not found", err)
		return
	}

	response.Success(c, http.StatusOK, "Commit retrieved successfully", commit)
}

// GetCommitDiff 获取提交差异
func (h *GitHandler) GetCommitDiff(c *gin.Context) {
	idStr := c.Param("id")
	repositoryID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid repository ID", err)
		return
	}

	sha := c.Param("sha")
	if sha == "" {
		response.Error(c, http.StatusBadRequest, "Commit SHA is required", nil)
		return
	}

	diff, err := h.gitService.GetCommitDiff(c.Request.Context(), repositoryID, sha)
	if err != nil {
		h.logger.Error("Failed to get commit diff", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to get commit diff", err)
		return
	}

	response.Success(c, http.StatusOK, "Commit diff retrieved successfully", diff)
}

// CompareBranches 比较分支
func (h *GitHandler) CompareBranches(c *gin.Context) {
	idStr := c.Param("id")
	repositoryID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid repository ID", err)
		return
	}

	base := c.Query("base")
	head := c.Query("head")

	if base == "" || head == "" {
		response.Error(c, http.StatusBadRequest, "Both base and head parameters are required", nil)
		return
	}

	diff, err := h.gitService.CompareBranches(c.Request.Context(), repositoryID, base, head)
	if err != nil {
		h.logger.Error("Failed to compare branches", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to compare branches", err)
		return
	}

	response.Success(c, http.StatusOK, "Branches compared successfully", diff)
}

// 标签管理处理器

// CreateTag 创建标签
func (h *GitHandler) CreateTag(c *gin.Context) {
	idStr := c.Param("id")
	repositoryID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid repository ID", err)
		return
	}

	var req models.CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	tag, err := h.gitService.CreateTag(c.Request.Context(), repositoryID, &req)
	if err != nil {
		h.logger.Error("Failed to create tag", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to create tag", err)
		return
	}

	response.Success(c, http.StatusCreated, "Tag created successfully", tag)
}

// ListTags 获取标签列表
func (h *GitHandler) ListTags(c *gin.Context) {
	idStr := c.Param("id")
	repositoryID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid repository ID", err)
		return
	}

	tags, err := h.gitService.ListTags(c.Request.Context(), repositoryID)
	if err != nil {
		h.logger.Error("Failed to list tags", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to list tags", err)
		return
	}

	response.Success(c, http.StatusOK, "Tags retrieved successfully", tags)
}

// GetTag 获取标签详情
func (h *GitHandler) GetTag(c *gin.Context) {
	idStr := c.Param("id")
	repositoryID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid repository ID", err)
		return
	}

	tagName := c.Param("tag")
	if tagName == "" {
		response.Error(c, http.StatusBadRequest, "Tag name is required", nil)
		return
	}

	tag, err := h.gitService.GetTag(c.Request.Context(), repositoryID, tagName)
	if err != nil {
		h.logger.Error("Failed to get tag", zap.Error(err))
		response.Error(c, http.StatusNotFound, "Tag not found", err)
		return
	}

	response.Success(c, http.StatusOK, "Tag retrieved successfully", tag)
}

// DeleteTag 删除标签
func (h *GitHandler) DeleteTag(c *gin.Context) {
	idStr := c.Param("id")
	repositoryID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid repository ID", err)
		return
	}

	tagName := c.Param("tag")
	if tagName == "" {
		response.Error(c, http.StatusBadRequest, "Tag name is required", nil)
		return
	}

	if err := h.gitService.DeleteTag(c.Request.Context(), repositoryID, tagName); err != nil {
		h.logger.Error("Failed to delete tag", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to delete tag", err)
		return
	}

	response.Success(c, http.StatusOK, "Tag deleted successfully", nil)
}

// 文件操作处理器

// GetFileContent 获取文件内容
func (h *GitHandler) GetFileContent(c *gin.Context) {
	idStr := c.Param("id")
	repositoryID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid repository ID", err)
		return
	}

	branch := c.Query("branch")
	filePath := c.Query("path")

	if branch == "" {
		response.Error(c, http.StatusBadRequest, "Branch parameter is required", nil)
		return
	}

	if filePath == "" {
		response.Error(c, http.StatusBadRequest, "Path parameter is required", nil)
		return
	}

	content, err := h.gitService.GetFileContent(c.Request.Context(), repositoryID, branch, filePath)
	if err != nil {
		h.logger.Error("Failed to get file content", zap.Error(err))
		response.Error(c, http.StatusNotFound, "File not found", err)
		return
	}

	// 检查文件内容类型
	contentType := "text/plain"
	if strings.HasSuffix(strings.ToLower(filePath), ".json") {
		contentType = "application/json"
	} else if strings.HasSuffix(strings.ToLower(filePath), ".xml") {
		contentType = "application/xml"
	} else if strings.HasSuffix(strings.ToLower(filePath), ".html") || strings.HasSuffix(strings.ToLower(filePath), ".htm") {
		contentType = "text/html"
	}

	c.Data(http.StatusOK, contentType, content)
}

// GetDirectoryContent 获取目录内容
func (h *GitHandler) GetDirectoryContent(c *gin.Context) {
	idStr := c.Param("id")
	repositoryID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid repository ID", err)
		return
	}

	branch := c.Query("branch")
	dirPath := c.DefaultQuery("path", "")

	if branch == "" {
		response.Error(c, http.StatusBadRequest, "Branch parameter is required", nil)
		return
	}

	files, err := h.gitService.GetDirectoryContent(c.Request.Context(), repositoryID, branch, dirPath)
	if err != nil {
		h.logger.Error("Failed to get directory content", zap.Error(err))
		response.Error(c, http.StatusNotFound, "Directory not found", err)
		return
	}

	response.Success(c, http.StatusOK, "Directory content retrieved successfully", gin.H{"files": files})
}
