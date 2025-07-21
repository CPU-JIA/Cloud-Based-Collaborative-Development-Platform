package handlers

import (
	"net/http"
	"strconv"

	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/models"
	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/service"
	"github.com/cloud-platform/collaborative-dev/shared/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PipelineHandler 流水线处理器
type PipelineHandler struct {
	service service.PipelineService
	logger  *zap.Logger
}

// NewPipelineHandler 创建流水线处理器实例
func NewPipelineHandler(service service.PipelineService, logger *zap.Logger) *PipelineHandler {
	return &PipelineHandler{
		service: service,
		logger:  logger,
	}
}

// 流水线管理接口

// CreatePipeline 创建流水线
// @Summary 创建流水线
// @Description 创建新的CI/CD流水线
// @Tags pipelines
// @Accept json
// @Produce json
// @Param request body models.CreatePipelineRequest true "创建流水线请求"
// @Success 201 {object} response.Response{data=models.Pipeline}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/pipelines [post]
func (h *PipelineHandler) CreatePipeline(c *gin.Context) {
	var req models.CreatePipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("参数绑定失败", zap.Error(err))
		response.Error(c, http.StatusBadRequest, "参数格式错误", err)
		return
	}

	// 从JWT中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "用户未认证", nil)
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "用户ID格式错误", nil)
		return
	}

	pipeline, err := h.service.CreatePipeline(c.Request.Context(), &req, userUUID)
	if err != nil {
		h.logger.Error("创建流水线失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "创建流水线失败", err)
		return
	}

	response.Success(c, http.StatusCreated, "流水线创建成功", pipeline)
}

// GetPipeline 获取流水线详情
// @Summary 获取流水线详情
// @Description 根据ID获取流水线详细信息
// @Tags pipelines
// @Produce json
// @Param id path string true "流水线ID"
// @Success 200 {object} response.Response{data=models.Pipeline}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/pipelines/{id} [get]
func (h *PipelineHandler) GetPipeline(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的流水线ID", err)
		return
	}

	pipeline, err := h.service.GetPipeline(c.Request.Context(), id)
	if err != nil {
		if err == service.ErrPipelineNotFound {
			response.Error(c, http.StatusNotFound, "流水线不存在", err)
			return
		}
		h.logger.Error("获取流水线失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "获取流水线失败", err)
		return
	}

	response.Success(c, http.StatusOK, "获取成功", pipeline)
}

// ListPipelines 获取流水线列表
// @Summary 获取流水线列表
// @Description 分页获取流水线列表
// @Tags pipelines
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页大小" default(20)
// @Param repository_id query string false "仓库ID"
// @Success 200 {object} response.Response{data=models.PipelineListResponse}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/pipelines [get]
func (h *PipelineHandler) ListPipelines(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	repositoryIDStr := c.Query("repository_id")

	var result *models.PipelineListResponse
	var err error

	if repositoryIDStr != "" {
		// 根据仓库ID查询
		repositoryID, parseErr := uuid.Parse(repositoryIDStr)
		if parseErr != nil {
			response.Error(c, http.StatusBadRequest, "无效的仓库ID", parseErr)
			return
		}
		result, err = h.service.GetPipelinesByRepository(c.Request.Context(), repositoryID, page, pageSize)
	} else {
		// 查询所有流水线
		result, err = h.service.ListPipelines(c.Request.Context(), page, pageSize)
	}

	if err != nil {
		h.logger.Error("获取流水线列表失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "获取流水线列表失败", err)
		return
	}

	response.Success(c, http.StatusOK, "获取成功", result)
}

// UpdatePipeline 更新流水线
// @Summary 更新流水线
// @Description 更新流水线信息
// @Tags pipelines
// @Accept json
// @Produce json
// @Param id path string true "流水线ID"
// @Param request body models.UpdatePipelineRequest true "更新流水线请求"
// @Success 200 {object} response.Response{data=models.Pipeline}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/pipelines/{id} [put]
func (h *PipelineHandler) UpdatePipeline(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的流水线ID", err)
		return
	}

	var req models.UpdatePipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("参数绑定失败", zap.Error(err))
		response.Error(c, http.StatusBadRequest, "参数格式错误", err)
		return
	}

	// 从JWT中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "用户未认证", nil)
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "用户ID格式错误", nil)
		return
	}

	pipeline, err := h.service.UpdatePipeline(c.Request.Context(), id, &req, userUUID)
	if err != nil {
		if err == service.ErrPipelineNotFound {
			response.Error(c, http.StatusNotFound, "流水线不存在", err)
			return
		}
		h.logger.Error("更新流水线失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "更新流水线失败", err)
		return
	}

	response.Success(c, http.StatusOK, "更新成功", pipeline)
}

// DeletePipeline 删除流水线
// @Summary 删除流水线
// @Description 软删除流水线
// @Tags pipelines
// @Produce json
// @Param id path string true "流水线ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/pipelines/{id} [delete]
func (h *PipelineHandler) DeletePipeline(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的流水线ID", err)
		return
	}

	// 从JWT中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "用户未认证", nil)
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "用户ID格式错误", nil)
		return
	}

	err = h.service.DeletePipeline(c.Request.Context(), id, userUUID)
	if err != nil {
		if err == service.ErrPipelineNotFound {
			response.Error(c, http.StatusNotFound, "流水线不存在", err)
			return
		}
		h.logger.Error("删除流水线失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "删除流水线失败", err)
		return
	}

	response.Success(c, http.StatusOK, "删除成功", nil)
}

// 流水线运行管理接口

// TriggerPipeline 触发流水线
// @Summary 触发流水线
// @Description 手动触发流水线执行
// @Tags pipeline-runs
// @Accept json
// @Produce json
// @Param id path string true "流水线ID"
// @Param request body models.TriggerPipelineRequest true "触发流水线请求"
// @Success 201 {object} response.Response{data=models.PipelineRun}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/pipelines/{id}/trigger [post]
func (h *PipelineHandler) TriggerPipeline(c *gin.Context) {
	idStr := c.Param("id")
	pipelineID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的流水线ID", err)
		return
	}

	var req models.TriggerPipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("参数绑定失败", zap.Error(err))
		response.Error(c, http.StatusBadRequest, "参数格式错误", err)
		return
	}

	// 从JWT中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "用户未认证", nil)
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "用户ID格式错误", nil)
		return
	}

	run, err := h.service.TriggerPipeline(c.Request.Context(), pipelineID, &req, userUUID)
	if err != nil {
		if err == service.ErrPipelineNotFound {
			response.Error(c, http.StatusNotFound, "流水线不存在", err)
			return
		}
		h.logger.Error("触发流水线失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "触发流水线失败", err)
		return
	}

	response.Success(c, http.StatusCreated, "流水线触发成功", run)
}

// GetPipelineRun 获取流水线运行详情
// @Summary 获取流水线运行详情
// @Description 根据ID获取流水线运行详细信息
// @Tags pipeline-runs
// @Produce json
// @Param id path string true "运行ID"
// @Success 200 {object} response.Response{data=models.PipelineRun}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/pipeline-runs/{id} [get]
func (h *PipelineHandler) GetPipelineRun(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的运行ID", err)
		return
	}

	run, err := h.service.GetPipelineRun(c.Request.Context(), id)
	if err != nil {
		if err == service.ErrPipelineRunNotFound {
			response.Error(c, http.StatusNotFound, "流水线运行不存在", err)
			return
		}
		h.logger.Error("获取流水线运行失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "获取流水线运行失败", err)
		return
	}

	response.Success(c, http.StatusOK, "获取成功", run)
}

// GetPipelineRuns 获取流水线运行列表
// @Summary 获取流水线运行列表
// @Description 根据流水线ID分页获取运行列表
// @Tags pipeline-runs
// @Produce json
// @Param id path string true "流水线ID"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页大小" default(20)
// @Success 200 {object} response.Response{data=models.PipelineRunListResponse}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/pipelines/{id}/runs [get]
func (h *PipelineHandler) GetPipelineRuns(c *gin.Context) {
	idStr := c.Param("id")
	pipelineID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的流水线ID", err)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.service.GetPipelineRunsByPipeline(c.Request.Context(), pipelineID, page, pageSize)
	if err != nil {
		h.logger.Error("获取流水线运行列表失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "获取流水线运行列表失败", err)
		return
	}

	response.Success(c, http.StatusOK, "获取成功", result)
}

// CancelPipelineRun 取消流水线运行
// @Summary 取消流水线运行
// @Description 取消正在进行的流水线运行
// @Tags pipeline-runs
// @Produce json
// @Param id path string true "运行ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/pipeline-runs/{id}/cancel [post]
func (h *PipelineHandler) CancelPipelineRun(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的运行ID", err)
		return
	}

	// 从JWT中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "用户未认证", nil)
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "用户ID格式错误", nil)
		return
	}

	err = h.service.CancelPipelineRun(c.Request.Context(), id, userUUID)
	if err != nil {
		if err == service.ErrPipelineRunNotFound {
			response.Error(c, http.StatusNotFound, "流水线运行不存在", err)
			return
		}
		h.logger.Error("取消流水线运行失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "取消流水线运行失败", err)
		return
	}

	response.Success(c, http.StatusOK, "取消成功", nil)
}

// RetryPipelineRun 重试流水线运行
// @Summary 重试流水线运行
// @Description 重试失败的流水线运行
// @Tags pipeline-runs
// @Produce json
// @Param id path string true "运行ID"
// @Success 201 {object} response.Response{data=models.PipelineRun}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/pipeline-runs/{id}/retry [post]
func (h *PipelineHandler) RetryPipelineRun(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的运行ID", err)
		return
	}

	// 从JWT中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "用户未认证", nil)
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "用户ID格式错误", nil)
		return
	}

	newRun, err := h.service.RetryPipelineRun(c.Request.Context(), id, userUUID)
	if err != nil {
		if err == service.ErrPipelineRunNotFound {
			response.Error(c, http.StatusNotFound, "流水线运行不存在", err)
			return
		}
		h.logger.Error("重试流水线运行失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "重试流水线运行失败", err)
		return
	}

	response.Success(c, http.StatusCreated, "重试成功", newRun)
}

// 作业管理接口

// GetJob 获取作业详情
// @Summary 获取作业详情
// @Description 根据ID获取作业详细信息
// @Tags jobs
// @Produce json
// @Param id path string true "作业ID"
// @Success 200 {object} response.Response{data=models.Job}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/jobs/{id} [get]
func (h *PipelineHandler) GetJob(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的作业ID", err)
		return
	}

	job, err := h.service.GetJob(c.Request.Context(), id)
	if err != nil {
		if err == service.ErrJobNotFound {
			response.Error(c, http.StatusNotFound, "作业不存在", err)
			return
		}
		h.logger.Error("获取作业失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "获取作业失败", err)
		return
	}

	response.Success(c, http.StatusOK, "获取成功", job)
}

// GetJobs 获取作业列表
// @Summary 获取作业列表
// @Description 根据流水线运行ID获取作业列表
// @Tags jobs
// @Produce json
// @Param run_id path string true "流水线运行ID"
// @Success 200 {object} response.Response{data=[]models.Job}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/pipeline-runs/{run_id}/jobs [get]
func (h *PipelineHandler) GetJobs(c *gin.Context) {
	runIDStr := c.Param("run_id")
	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的运行ID", err)
		return
	}

	jobs, err := h.service.GetJobsByPipelineRun(c.Request.Context(), runID)
	if err != nil {
		h.logger.Error("获取作业列表失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "获取作业列表失败", err)
		return
	}

	response.Success(c, http.StatusOK, "获取成功", jobs)
}

// 执行器管理接口

// RegisterRunner 注册执行器
// @Summary 注册执行器
// @Description 注册新的CI/CD执行器
// @Tags runners
// @Accept json
// @Produce json
// @Param request body models.RegisterRunnerRequest true "注册执行器请求"
// @Success 201 {object} response.Response{data=models.Runner}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/runners [post]
func (h *PipelineHandler) RegisterRunner(c *gin.Context) {
	var req models.RegisterRunnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("参数绑定失败", zap.Error(err))
		response.Error(c, http.StatusBadRequest, "参数格式错误", err)
		return
	}

	// 从JWT中获取租户ID
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "租户未认证", nil)
		return
	}

	tenantUUID, ok := tenantID.(uuid.UUID)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "租户ID格式错误", nil)
		return
	}

	runner, err := h.service.RegisterRunner(c.Request.Context(), &req, tenantUUID)
	if err != nil {
		h.logger.Error("注册执行器失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "注册执行器失败", err)
		return
	}

	response.Success(c, http.StatusCreated, "执行器注册成功", runner)
}

// GetRunner 获取执行器详情
// @Summary 获取执行器详情
// @Description 根据ID获取执行器详细信息
// @Tags runners
// @Produce json
// @Param id path string true "执行器ID"
// @Success 200 {object} response.Response{data=models.Runner}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/runners/{id} [get]
func (h *PipelineHandler) GetRunner(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的执行器ID", err)
		return
	}

	runner, err := h.service.GetRunner(c.Request.Context(), id)
	if err != nil {
		if err == service.ErrRunnerNotFound {
			response.Error(c, http.StatusNotFound, "执行器不存在", err)
			return
		}
		h.logger.Error("获取执行器失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "获取执行器失败", err)
		return
	}

	response.Success(c, http.StatusOK, "获取成功", runner)
}

// ListRunners 获取执行器列表
// @Summary 获取执行器列表
// @Description 分页获取执行器列表
// @Tags runners
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页大小" default(20)
// @Success 200 {object} response.Response{data=[]models.Runner}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/runners [get]
func (h *PipelineHandler) ListRunners(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 从JWT中获取租户ID
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "租户未认证", nil)
		return
	}

	tenantUUID, ok := tenantID.(uuid.UUID)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "租户ID格式错误", nil)
		return
	}

	runners, total, err := h.service.ListRunners(c.Request.Context(), tenantUUID, page, pageSize)
	if err != nil {
		h.logger.Error("获取执行器列表失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "获取执行器列表失败", err)
		return
	}

	result := gin.H{
		"runners":   runners,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}

	response.Success(c, http.StatusOK, "获取成功", result)
}

// UpdateRunner 更新执行器
// @Summary 更新执行器
// @Description 更新执行器信息
// @Tags runners
// @Accept json
// @Produce json
// @Param id path string true "执行器ID"
// @Param request body models.UpdateRunnerRequest true "更新执行器请求"
// @Success 200 {object} response.Response{data=models.Runner}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/runners/{id} [put]
func (h *PipelineHandler) UpdateRunner(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的执行器ID", err)
		return
	}

	var req models.UpdateRunnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("参数绑定失败", zap.Error(err))
		response.Error(c, http.StatusBadRequest, "参数格式错误", err)
		return
	}

	runner, err := h.service.UpdateRunner(c.Request.Context(), id, &req)
	if err != nil {
		if err == service.ErrRunnerNotFound {
			response.Error(c, http.StatusNotFound, "执行器不存在", err)
			return
		}
		h.logger.Error("更新执行器失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "更新执行器失败", err)
		return
	}

	response.Success(c, http.StatusOK, "更新成功", runner)
}

// UnregisterRunner 注销执行器
// @Summary 注销执行器
// @Description 注销执行器
// @Tags runners
// @Produce json
// @Param id path string true "执行器ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/runners/{id} [delete]
func (h *PipelineHandler) UnregisterRunner(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的执行器ID", err)
		return
	}

	err = h.service.UnregisterRunner(c.Request.Context(), id)
	if err != nil {
		if err == service.ErrRunnerNotFound {
			response.Error(c, http.StatusNotFound, "执行器不存在", err)
			return
		}
		h.logger.Error("注销执行器失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "注销执行器失败", err)
		return
	}

	response.Success(c, http.StatusOK, "注销成功", nil)
}

// HeartbeatRunner 执行器心跳
// @Summary 执行器心跳
// @Description 执行器定期发送心跳以保持在线状态
// @Tags runners
// @Accept json
// @Produce json
// @Param id path string true "执行器ID"
// @Param status body string true "执行器状态"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/runners/{id}/heartbeat [post]
func (h *PipelineHandler) HeartbeatRunner(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的执行器ID", err)
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("参数绑定失败", zap.Error(err))
		response.Error(c, http.StatusBadRequest, "参数格式错误", err)
		return
	}

	status := models.RunnerStatus(req.Status)
	err = h.service.HeartbeatRunner(c.Request.Context(), id, status)
	if err != nil {
		h.logger.Error("执行器心跳失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "执行器心跳失败", err)
		return
	}

	response.Success(c, http.StatusOK, "心跳成功", nil)
}

// 统计接口

// GetPipelineStats 获取流水线统计信息
// @Summary 获取流水线统计信息
// @Description 获取流水线的运行统计信息
// @Tags statistics
// @Produce json
// @Param id path string true "流水线ID"
// @Param days query int false "统计天数" default(30)
// @Success 200 {object} response.Response{data=models.PipelineStats}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/pipelines/{id}/stats [get]
func (h *PipelineHandler) GetPipelineStats(c *gin.Context) {
	idStr := c.Param("id")
	pipelineID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的流水线ID", err)
		return
	}

	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))

	stats, err := h.service.GetPipelineStats(c.Request.Context(), pipelineID, days)
	if err != nil {
		h.logger.Error("获取流水线统计失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "获取流水线统计失败", err)
		return
	}

	response.Success(c, http.StatusOK, "获取成功", stats)
}

// GetRunnerStats 获取执行器统计信息
// @Summary 获取执行器统计信息
// @Description 获取执行器的运行统计信息
// @Tags statistics
// @Produce json
// @Param id path string true "执行器ID"
// @Success 200 {object} response.Response{data=models.RunnerStats}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/runners/{id}/stats [get]
func (h *PipelineHandler) GetRunnerStats(c *gin.Context) {
	idStr := c.Param("id")
	runnerID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的执行器ID", err)
		return
	}

	stats, err := h.service.GetRunnerStats(c.Request.Context(), runnerID)
	if err != nil {
		h.logger.Error("获取执行器统计失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "获取执行器统计失败", err)
		return
	}

	response.Success(c, http.StatusOK, "获取成功", stats)
}