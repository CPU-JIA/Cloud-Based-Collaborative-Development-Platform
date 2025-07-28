package handler

import (
	"net/http"
	"strconv"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/service"
	"github.com/cloud-platform/collaborative-dev/shared/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AgileHandler 敏捷开发处理器
type AgileHandler struct {
	agileService service.AgileService
	logger       *zap.Logger
}

// NewAgileHandler 创建敏捷开发处理器
func NewAgileHandler(agileService service.AgileService, logger *zap.Logger) *AgileHandler {
	return &AgileHandler{
		agileService: agileService,
		logger:       logger,
	}
}

// Sprint管理接口

// CreateSprint 创建Sprint
func (h *AgileHandler) CreateSprint(c *gin.Context) {
	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	var req service.CreateSprintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	sprint, err := h.agileService.CreateSprint(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		h.logger.Error("创建Sprint失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to create sprint", err)
		return
	}

	response.Success(c, http.StatusCreated, "Sprint created successfully", sprint)
}

// GetSprint 获取Sprint详情
func (h *AgileHandler) GetSprint(c *gin.Context) {
	sprintIDStr := c.Param("sprintId")
	sprintID, err := uuid.Parse(sprintIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid sprint ID", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	sprint, err := h.agileService.GetSprint(c.Request.Context(), sprintID, userID, tenantID)
	if err != nil {
		h.logger.Error("获取Sprint失败", zap.Error(err))
		response.Error(c, http.StatusNotFound, "Sprint not found", err)
		return
	}

	response.Success(c, http.StatusOK, "Sprint retrieved successfully", sprint)
}

// ListSprints 获取Sprint列表
func (h *AgileHandler) ListSprints(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid project ID", err)
		return
	}

	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	sprints, err := h.agileService.ListSprints(c.Request.Context(), projectID, page, pageSize, userID, tenantID)
	if err != nil {
		h.logger.Error("获取Sprint列表失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to list sprints", err)
		return
	}

	response.Success(c, http.StatusOK, "Sprints retrieved successfully", sprints)
}

// UpdateSprint 更新Sprint
func (h *AgileHandler) UpdateSprint(c *gin.Context) {
	sprintIDStr := c.Param("sprintId")
	sprintID, err := uuid.Parse(sprintIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid sprint ID", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	var req service.UpdateSprintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	sprint, err := h.agileService.UpdateSprint(c.Request.Context(), sprintID, &req, userID, tenantID)
	if err != nil {
		h.logger.Error("更新Sprint失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to update sprint", err)
		return
	}

	response.Success(c, http.StatusOK, "Sprint updated successfully", sprint)
}

// DeleteSprint 删除Sprint
func (h *AgileHandler) DeleteSprint(c *gin.Context) {
	sprintIDStr := c.Param("sprintId")
	sprintID, err := uuid.Parse(sprintIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid sprint ID", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	if err := h.agileService.DeleteSprint(c.Request.Context(), sprintID, userID, tenantID); err != nil {
		h.logger.Error("删除Sprint失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to delete sprint", err)
		return
	}

	response.Success(c, http.StatusOK, "Sprint deleted successfully", nil)
}

// StartSprint 启动Sprint
func (h *AgileHandler) StartSprint(c *gin.Context) {
	sprintIDStr := c.Param("sprintId")
	sprintID, err := uuid.Parse(sprintIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid sprint ID", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	if err := h.agileService.StartSprint(c.Request.Context(), sprintID, userID, tenantID); err != nil {
		h.logger.Error("启动Sprint失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to start sprint", err)
		return
	}

	response.Success(c, http.StatusOK, "Sprint started successfully", gin.H{
		"sprint_id": sprintID,
		"status":    "active",
	})
}

// CompleteSprint 完成Sprint
func (h *AgileHandler) CompleteSprint(c *gin.Context) {
	sprintIDStr := c.Param("sprintId")
	sprintID, err := uuid.Parse(sprintIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid sprint ID", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	if err := h.agileService.CompleteSprint(c.Request.Context(), sprintID, userID, tenantID); err != nil {
		h.logger.Error("完成Sprint失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to complete sprint", err)
		return
	}

	response.Success(c, http.StatusOK, "Sprint completed successfully", gin.H{
		"sprint_id": sprintID,
		"status":    "closed",
	})
}

// CloseSprint 关闭Sprint (别名方法)
func (h *AgileHandler) CloseSprint(c *gin.Context) {
	h.CompleteSprint(c)
}

// 敏捷任务管理接口

// CreateTask 创建任务
func (h *AgileHandler) CreateTask(c *gin.Context) {
	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	var req service.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	task, err := h.agileService.CreateTask(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		h.logger.Error("创建任务失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to create task", err)
		return
	}

	response.Success(c, http.StatusCreated, "Task created successfully", task)
}

// GetTask 获取任务详情
func (h *AgileHandler) GetTask(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	task, err := h.agileService.GetTask(c.Request.Context(), taskID, userID, tenantID)
	if err != nil {
		h.logger.Error("获取任务失败", zap.Error(err))
		response.Error(c, http.StatusNotFound, "Task not found", err)
		return
	}

	response.Success(c, http.StatusOK, "Task retrieved successfully", task)
}

// ListTasks 获取任务列表
func (h *AgileHandler) ListTasks(c *gin.Context) {
	// 解析项目ID
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		response.Error(c, http.StatusBadRequest, "Project ID is required", nil)
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid project ID", err)
		return
	}

	// 构建过滤器
	filter := &service.TaskFilter{
		ProjectID: projectID,
	}

	// 解析可选过滤参数
	if sprintIDStr := c.Query("sprint_id"); sprintIDStr != "" {
		if sprintID, err := uuid.Parse(sprintIDStr); err == nil {
			filter.SprintID = &sprintID
		}
	}

	if epicIDStr := c.Query("epic_id"); epicIDStr != "" {
		if epicID, err := uuid.Parse(epicIDStr); err == nil {
			filter.EpicID = &epicID
		}
	}

	if assigneeIDStr := c.Query("assignee_id"); assigneeIDStr != "" {
		if assigneeID, err := uuid.Parse(assigneeIDStr); err == nil {
			filter.AssigneeID = &assigneeID
		}
	}

	if statuses := c.QueryArray("status"); len(statuses) > 0 {
		filter.Status = statuses
	}

	if types := c.QueryArray("type"); len(types) > 0 {
		filter.Type = types
	}

	if priorities := c.QueryArray("priority"); len(priorities) > 0 {
		filter.Priority = priorities
	}

	if searchText := c.Query("search"); searchText != "" {
		filter.SearchText = &searchText
	}

	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	tasks, err := h.agileService.ListTasks(c.Request.Context(), filter, page, pageSize, userID, tenantID)
	if err != nil {
		h.logger.Error("获取任务列表失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to list tasks", err)
		return
	}

	response.Success(c, http.StatusOK, "Tasks retrieved successfully", tasks)
}

// UpdateTask 更新任务
func (h *AgileHandler) UpdateTask(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	var req service.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	task, err := h.agileService.UpdateTask(c.Request.Context(), taskID, &req, userID, tenantID)
	if err != nil {
		h.logger.Error("更新任务失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to update task", err)
		return
	}

	response.Success(c, http.StatusOK, "Task updated successfully", task)
}

// TransitionTask 任务状态转换
func (h *AgileHandler) TransitionTask(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	var req struct {
		Status string `json:"status" binding:"required,oneof=todo in_progress in_review testing done cancelled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	if err := h.agileService.TransitionTask(c.Request.Context(), taskID, req.Status, userID, tenantID); err != nil {
		h.logger.Error("任务状态转换失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to transition task", err)
		return
	}

	response.Success(c, http.StatusOK, "Task status transitioned successfully", gin.H{
		"task_id": taskID,
		"status":  req.Status,
	})
}

// ReorderTasks 方法已在下方重新定义，带有更详细的注释和错误处理

// 工作日志管理接口

// LogWork 记录工作日志
func (h *AgileHandler) LogWork(c *gin.Context) {
	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	var req service.LogWorkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	workLog, err := h.agileService.LogWork(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		h.logger.Error("记录工作日志失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to log work", err)
		return
	}

	response.Success(c, http.StatusCreated, "Work logged successfully", workLog)
}

// GetWorkLogs 获取工作日志
func (h *AgileHandler) GetWorkLogs(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	workLogs, err := h.agileService.GetWorkLogs(c.Request.Context(), taskID, userID, tenantID)
	if err != nil {
		h.logger.Error("获取工作日志失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to get work logs", err)
		return
	}

	response.Success(c, http.StatusOK, "Work logs retrieved successfully", workLogs)
}

// DeleteWorkLog 删除工作日志
func (h *AgileHandler) DeleteWorkLog(c *gin.Context) {
	workLogIDStr := c.Param("workLogId")
	workLogID, err := uuid.Parse(workLogIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid work log ID", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	if err := h.agileService.DeleteWorkLog(c.Request.Context(), workLogID, userID, tenantID); err != nil {
		h.logger.Error("删除工作日志失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to delete work log", err)
		return
	}

	response.Success(c, http.StatusOK, "Work log deleted successfully", nil)
}

// 任务评论管理接口

// AddComment 添加评论
func (h *AgileHandler) AddComment(c *gin.Context) {
	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	var req service.AddCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	comment, err := h.agileService.AddComment(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		h.logger.Error("添加评论失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to add comment", err)
		return
	}

	response.Success(c, http.StatusCreated, "Comment added successfully", comment)
}

// GetComments 获取评论列表
func (h *AgileHandler) GetComments(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	comments, err := h.agileService.GetComments(c.Request.Context(), taskID, userID, tenantID)
	if err != nil {
		h.logger.Error("获取评论列表失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to get comments", err)
		return
	}

	response.Success(c, http.StatusOK, "Comments retrieved successfully", comments)
}

// UpdateComment 更新评论
func (h *AgileHandler) UpdateComment(c *gin.Context) {
	commentIDStr := c.Param("commentId")
	commentID, err := uuid.Parse(commentIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid comment ID", err)
		return
	}

	var req struct {
		Content string `json:"content" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	comment, err := h.agileService.UpdateComment(c.Request.Context(), commentID, req.Content, userID, tenantID)
	if err != nil {
		h.logger.Error("更新评论失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to update comment", err)
		return
	}

	response.Success(c, http.StatusOK, "Comment updated successfully", comment)
}

// DeleteComment 删除评论
func (h *AgileHandler) DeleteComment(c *gin.Context) {
	commentIDStr := c.Param("commentId")
	commentID, err := uuid.Parse(commentIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid comment ID", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	if err := h.agileService.DeleteComment(c.Request.Context(), commentID, userID, tenantID); err != nil {
		h.logger.Error("删除评论失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to delete comment", err)
		return
	}

	response.Success(c, http.StatusOK, "Comment deleted successfully", nil)
}

// 报表和统计接口

// GetSprintBurndown 获取Sprint燃尽图
func (h *AgileHandler) GetSprintBurndown(c *gin.Context) {
	sprintIDStr := c.Param("sprintId")
	sprintID, err := uuid.Parse(sprintIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid sprint ID", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	burndownData, err := h.agileService.GetSprintBurndown(c.Request.Context(), sprintID, userID, tenantID)
	if err != nil {
		h.logger.Error("获取燃尽图数据失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to get burndown data", err)
		return
	}

	response.Success(c, http.StatusOK, "Burndown data retrieved successfully", burndownData)
}

// GetVelocityChart 获取速度图
func (h *AgileHandler) GetVelocityChart(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid project ID", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	velocityData, err := h.agileService.GetVelocityChart(c.Request.Context(), projectID, userID, tenantID)
	if err != nil {
		h.logger.Error("获取速度图数据失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to get velocity data", err)
		return
	}

	response.Success(c, http.StatusOK, "Velocity data retrieved successfully", velocityData)
}

// GetTaskStatistics 获取任务统计
func (h *AgileHandler) GetTaskStatistics(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid project ID", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	statistics, err := h.agileService.GetTaskStatistics(c.Request.Context(), projectID, userID, tenantID)
	if err != nil {
		h.logger.Error("获取任务统计失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to get task statistics", err)
		return
	}

	response.Success(c, http.StatusOK, "Task statistics retrieved successfully", statistics)
}

// 任务拖拽排序接口

// ReorderTasks 重新排序任务
// @Summary 重新排序任务
// @Description 基于目标索引重新排序任务，支持跨列和跨Sprint移动
// @Tags Task
// @Accept json
// @Produce json
// @Param request body service.ReorderTasksRequest true "重排序请求"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/tasks/reorder [post]
func (h *AgileHandler) ReorderTasks(c *gin.Context) {
	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	var req service.ReorderTasksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	err := h.agileService.ReorderTasks(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		h.logger.Error("任务重排序失败", zap.Error(err))
		if err.Error() == "task not found" {
			response.Error(c, http.StatusNotFound, "Task not found", err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to reorder tasks", err)
		return
	}

	response.Success(c, http.StatusOK, "Tasks reordered successfully", nil)
}

// MoveTask 精确移动任务
// @Summary 精确移动任务
// @Description 基于前后任务位置精确移动任务
// @Tags Task
// @Accept json
// @Produce json
// @Param request body service.TaskMoveRequest true "移动请求"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/tasks/move [post]
func (h *AgileHandler) MoveTask(c *gin.Context) {
	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	var req service.TaskMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	err := h.agileService.MoveTask(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		h.logger.Error("任务移动失败", zap.Error(err))
		if err.Error() == "task not found" {
			response.Error(c, http.StatusNotFound, "Task not found", err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to move task", err)
		return
	}

	response.Success(c, http.StatusOK, "Task moved successfully", nil)
}

// BatchReorderTasks 批量重排序任务
// @Summary 批量重排序任务
// @Description 批量设置任务的新排序
// @Tags Task
// @Accept json
// @Produce json
// @Param request body service.BatchReorderRequest true "批量重排序请求"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/tasks/batch-reorder [post]
func (h *AgileHandler) BatchReorderTasks(c *gin.Context) {
	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	var req service.BatchReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	err := h.agileService.BatchReorderTasks(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		h.logger.Error("批量任务重排序失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to batch reorder tasks", err)
		return
	}

	response.Success(c, http.StatusOK, "Tasks batch reordered successfully", nil)
}

// RebalanceTaskRanks 重新平衡任务排名
// @Summary 重新平衡任务排名
// @Description 重新平衡项目中所有任务的排名，修复排序问题
// @Tags Task
// @Produce json
// @Param project_id path string true "项目ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/projects/{project_id}/tasks/rebalance [post]
func (h *AgileHandler) RebalanceTaskRanks(c *gin.Context) {
	projectIDStr := c.Param("project_id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid project ID", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	err = h.agileService.RebalanceTaskRanks(c.Request.Context(), projectID, userID, tenantID)
	if err != nil {
		h.logger.Error("重新平衡任务排名失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to rebalance task ranks", err)
		return
	}

	response.Success(c, http.StatusOK, "Task ranks rebalanced successfully", nil)
}

// ValidateTaskOrder 验证任务排序
// @Summary 验证任务排序
// @Description 验证项目中任务排序是否正确
// @Tags Task
// @Produce json
// @Param project_id path string true "项目ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/projects/{project_id}/tasks/validate-order [get]
func (h *AgileHandler) ValidateTaskOrder(c *gin.Context) {
	projectIDStr := c.Param("project_id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid project ID", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	err = h.agileService.ValidateTaskOrder(c.Request.Context(), projectID, userID, tenantID)
	if err != nil {
		h.logger.Error("任务排序验证失败", zap.Error(err))
		response.Error(c, http.StatusBadRequest, "Task order validation failed", err)
		return
	}

	response.Success(c, http.StatusOK, "Task order is valid", nil)
}

// 缺失的方法实现

// GetSprintBurndownData 获取Sprint燃尽图数据
func (h *AgileHandler) GetSprintBurndownData(c *gin.Context) {
	sprintIDStr := c.Param("sprintId")
	sprintID, err := uuid.Parse(sprintIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid sprint ID", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	burndownData, err := h.agileService.GetSprintBurndownData(c.Request.Context(), sprintID, userID, tenantID)
	if err != nil {
		h.logger.Error("获取燃尽图数据失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to get burndown data", err)
		return
	}

	response.Success(c, http.StatusOK, "Burndown data retrieved successfully", burndownData)
}

// CreateEpic 创建Epic
func (h *AgileHandler) CreateEpic(c *gin.Context) {
	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	var req service.CreateEpicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	epic, err := h.agileService.CreateEpic(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		h.logger.Error("创建Epic失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to create epic", err)
		return
	}

	response.Success(c, http.StatusCreated, "Epic created successfully", epic)
}

// ListEpics 获取Epic列表
func (h *AgileHandler) ListEpics(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid project ID", err)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	epics, err := h.agileService.ListEpics(c.Request.Context(), projectID, page, pageSize, userID, tenantID)
	if err != nil {
		h.logger.Error("获取Epic列表失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to list epics", err)
		return
	}

	response.Success(c, http.StatusOK, "Epics retrieved successfully", epics)
}

// GetEpic 获取Epic详情
func (h *AgileHandler) GetEpic(c *gin.Context) {
	epicIDStr := c.Param("epicId")
	epicID, err := uuid.Parse(epicIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid epic ID", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	epic, err := h.agileService.GetEpic(c.Request.Context(), epicID, userID, tenantID)
	if err != nil {
		h.logger.Error("获取Epic失败", zap.Error(err))
		response.Error(c, http.StatusNotFound, "Epic not found", err)
		return
	}

	response.Success(c, http.StatusOK, "Epic retrieved successfully", epic)
}

// GetProjectStatistics 获取项目统计数据
func (h *AgileHandler) GetProjectStatistics(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid project ID", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	statistics, err := h.agileService.GetProjectStatistics(c.Request.Context(), projectID, userID, tenantID)
	if err != nil {
		h.logger.Error("获取项目统计失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to get project statistics", err)
		return
	}

	response.Success(c, http.StatusOK, "Project statistics retrieved successfully", statistics)
}

// DeleteTask 删除任务
func (h *AgileHandler) DeleteTask(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	if err := h.agileService.DeleteTask(c.Request.Context(), taskID, userID, tenantID); err != nil {
		h.logger.Error("删除任务失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to delete task", err)
		return
	}

	response.Success(c, http.StatusOK, "Task deleted successfully", nil)
}

// UpdateTaskStatus 更新任务状态
func (h *AgileHandler) UpdateTaskStatus(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	var req struct {
		Status string `json:"status" binding:"required,oneof=todo in_progress in_review testing done cancelled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	if err := h.agileService.UpdateTaskStatus(c.Request.Context(), taskID, req.Status, userID, tenantID); err != nil {
		h.logger.Error("更新任务状态失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to update task status", err)
		return
	}

	response.Success(c, http.StatusOK, "Task status updated successfully", gin.H{
		"task_id": taskID,
		"status":  req.Status,
	})
}

// AssignTask 分配任务
func (h *AgileHandler) AssignTask(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	var req struct {
		AssigneeID *uuid.UUID `json:"assignee_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	if err := h.agileService.AssignTask(c.Request.Context(), taskID, req.AssigneeID, userID, tenantID); err != nil {
		h.logger.Error("分配任务失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to assign task", err)
		return
	}

	response.Success(c, http.StatusOK, "Task assigned successfully", gin.H{
		"task_id":     taskID,
		"assignee_id": req.AssigneeID,
	})
}

// AddTaskComment 添加任务评论
func (h *AgileHandler) AddTaskComment(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	var req struct {
		Content string `json:"content" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	commentReq := &service.AddCommentRequest{
		TaskID:  taskID,
		Content: req.Content,
	}

	comment, err := h.agileService.AddComment(c.Request.Context(), commentReq, userID, tenantID)
	if err != nil {
		h.logger.Error("添加任务评论失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to add comment", err)
		return
	}

	response.Success(c, http.StatusCreated, "Comment added successfully", comment)
}

// ListTaskComments 获取任务评论列表
func (h *AgileHandler) ListTaskComments(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	comments, err := h.agileService.GetComments(c.Request.Context(), taskID, userID, tenantID)
	if err != nil {
		h.logger.Error("获取任务评论失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to get comments", err)
		return
	}

	response.Success(c, http.StatusOK, "Comments retrieved successfully", comments)
}

// ListWorkLogs 获取工作日志列表
func (h *AgileHandler) ListWorkLogs(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	workLogs, err := h.agileService.GetWorkLogs(c.Request.Context(), taskID, userID, tenantID)
	if err != nil {
		h.logger.Error("获取工作日志失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to get work logs", err)
		return
	}

	response.Success(c, http.StatusOK, "Work logs retrieved successfully", workLogs)
}

// GetUserWorkload 获取用户工作负载
func (h *AgileHandler) GetUserWorkload(c *gin.Context) {
	userIDStr := c.Param("userId")
	targetUserID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	workload, err := h.agileService.GetUserWorkload(c.Request.Context(), targetUserID, userID, tenantID)
	if err != nil {
		h.logger.Error("获取用户工作负载失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to get user workload", err)
		return
	}

	response.Success(c, http.StatusOK, "User workload retrieved successfully", workload)
}

// 工具函数

// getUserIDFromContext 从上下文中获取用户ID
func getUserIDFromContext(c *gin.Context) uuid.UUID {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil
	}

	switch v := userID.(type) {
	case uuid.UUID:
		return v
	case string:
		if id, err := uuid.Parse(v); err == nil {
			return id
		}
	}

	return uuid.Nil
}

// getTenantIDFromContext 从上下文中获取租户ID
func getTenantIDFromContext(c *gin.Context) uuid.UUID {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return uuid.Nil
	}

	switch v := tenantID.(type) {
	case uuid.UUID:
		return v
	case string:
		if id, err := uuid.Parse(v); err == nil {
			return id
		}
	}

	return uuid.Nil
}
