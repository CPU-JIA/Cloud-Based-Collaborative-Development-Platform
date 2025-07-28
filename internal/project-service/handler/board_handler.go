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

// BoardHandler 看板管理处理器
type BoardHandler struct {
	agileService service.AgileService
	logger       *zap.Logger
}

// NewBoardHandler 创建看板处理器
func NewBoardHandler(agileService service.AgileService, logger *zap.Logger) *BoardHandler {
	return &BoardHandler{
		agileService: agileService,
		logger:       logger,
	}
}

// 看板管理

// CreateBoard 创建看板
func (h *BoardHandler) CreateBoard(c *gin.Context) {
	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	var req service.CreateBoardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request parameters", err)
		return
	}

	board, err := h.agileService.CreateBoard(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		h.logger.Error("创建看板失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to create board", err)
		return
	}

	response.Success(c, http.StatusCreated, "Board created successfully", board)
}

// GetBoard 获取看板详情
func (h *BoardHandler) GetBoard(c *gin.Context) {
	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid board ID", err)
		return
	}

	board, err := h.agileService.GetBoard(c.Request.Context(), id, userID, tenantID)
	if err != nil {
		h.logger.Error("获取看板失败", zap.Error(err))
		response.Error(c, http.StatusNotFound, "Board not found", err)
		return
	}

	response.Success(c, http.StatusOK, "Board retrieved successfully", board)
}

// ListBoards 获取看板列表
func (h *BoardHandler) ListBoards(c *gin.Context) {
	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

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

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	boards, err := h.agileService.ListBoards(c.Request.Context(), projectID, page, pageSize, userID, tenantID)
	if err != nil {
		h.logger.Error("获取看板列表失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to list boards", err)
		return
	}

	response.Success(c, http.StatusOK, "Boards retrieved successfully", boards)
}

// UpdateBoard 更新看板
func (h *BoardHandler) UpdateBoard(c *gin.Context) {
	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid board ID", err)
		return
	}

	var req service.UpdateBoardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request parameters", err)
		return
	}

	board, err := h.agileService.UpdateBoard(c.Request.Context(), id, &req, userID, tenantID)
	if err != nil {
		h.logger.Error("更新看板失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to update board", err)
		return
	}

	response.Success(c, http.StatusOK, "Board updated successfully", board)
}

// DeleteBoard 删除看板
func (h *BoardHandler) DeleteBoard(c *gin.Context) {
	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid board ID", err)
		return
	}

	if err := h.agileService.DeleteBoard(c.Request.Context(), id, userID, tenantID); err != nil {
		h.logger.Error("删除看板失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to delete board", err)
		return
	}

	response.Success(c, http.StatusOK, "Board deleted successfully", nil)
}

// 看板列管理

// CreateBoardColumn 创建看板列
func (h *BoardHandler) CreateBoardColumn(c *gin.Context) {
	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	var req service.CreateBoardColumnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request parameters", err)
		return
	}

	column, err := h.agileService.CreateBoardColumn(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		h.logger.Error("创建看板列失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to create board column", err)
		return
	}

	response.Success(c, http.StatusCreated, "Board column created successfully", column)
}

// UpdateBoardColumn 更新看板列
func (h *BoardHandler) UpdateBoardColumn(c *gin.Context) {
	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid column ID", err)
		return
	}

	var req service.UpdateBoardColumnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request parameters", err)
		return
	}

	column, err := h.agileService.UpdateBoardColumn(c.Request.Context(), id, &req, userID, tenantID)
	if err != nil {
		h.logger.Error("更新看板列失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to update board column", err)
		return
	}

	response.Success(c, http.StatusOK, "Board column updated successfully", column)
}

// DeleteBoardColumn 删除看板列
func (h *BoardHandler) DeleteBoardColumn(c *gin.Context) {
	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid column ID", err)
		return
	}

	if err := h.agileService.DeleteBoardColumn(c.Request.Context(), id, userID, tenantID); err != nil {
		h.logger.Error("删除看板列失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to delete board column", err)
		return
	}

	response.Success(c, http.StatusOK, "Board column deleted successfully", nil)
}

// ReorderBoardColumns 重新排序看板列
func (h *BoardHandler) ReorderBoardColumns(c *gin.Context) {
	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	var req service.ReorderColumnsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request parameters", err)
		return
	}

	if err := h.agileService.ReorderBoardColumns(c.Request.Context(), &req, userID, tenantID); err != nil {
		h.logger.Error("重新排序看板列失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to reorder board columns", err)
		return
	}

	response.Success(c, http.StatusOK, "Board columns reordered successfully", nil)
}

// 任务卡片管理

// MoveTaskToColumn 移动任务到指定列
func (h *BoardHandler) MoveTaskToColumn(c *gin.Context) {
	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	var req service.TaskMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request parameters", err)
		return
	}

	if err := h.agileService.MoveTask(c.Request.Context(), &req, userID, tenantID); err != nil {
		h.logger.Error("移动任务失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to move task", err)
		return
	}

	response.Success(c, http.StatusOK, "Task moved successfully", nil)
}

// BatchMoveTasksToColumn 批量移动任务
func (h *BoardHandler) BatchMoveTasksToColumn(c *gin.Context) {
	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	var req service.BatchTaskMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request parameters", err)
		return
	}

	if err := h.agileService.BatchMoveTasks(c.Request.Context(), &req, userID, tenantID); err != nil {
		h.logger.Error("批量移动任务失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to batch move tasks", err)
		return
	}

	response.Success(c, http.StatusOK, "Tasks moved successfully", nil)
}

// GetBoardStatistics 获取看板统计信息
func (h *BoardHandler) GetBoardStatistics(c *gin.Context) {
	userID := getUserIDFromContext(c)
	tenantID := getTenantIDFromContext(c)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid board ID", err)
		return
	}

	stats, err := h.agileService.GetBoardStatistics(c.Request.Context(), id, userID, tenantID)
	if err != nil {
		h.logger.Error("获取看板统计信息失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Failed to get board statistics", err)
		return
	}

	response.Success(c, http.StatusOK, "Board statistics retrieved successfully", stats)
}

// 注意：工具函数 getUserIDFromContext 和 getTenantIDFromContext 已在 agile_handler.go 中定义
