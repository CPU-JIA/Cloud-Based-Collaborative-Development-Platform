package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// IndexBasedReorderRequest 基于索引的任务重排序请求
type IndexBasedReorderRequest struct {
	TaskID      uuid.UUID  `json:"task_id" binding:"required"`       // 被移动的任务ID
	TargetIndex int        `json:"target_index" binding:"min=0"`     // 目标位置索引
	ColumnID    *string    `json:"column_id,omitempty"`              // 目标列ID（如果跨列移动）
	SprintID    *uuid.UUID `json:"sprint_id,omitempty"`             // 目标Sprint ID（如果跨Sprint移动）
}


// ReorderTasksByIndex 重新排序任务（基于索引位置）
func (s *agileServiceImpl) ReorderTasksByIndex(ctx context.Context, req *IndexBasedReorderRequest, userID, tenantID uuid.UUID) error {
	// 获取任务并验证权限
	var task models.AgileTask
	if err := s.db.WithContext(ctx).First(&task, "id = ? AND deleted_at IS NULL", req.TaskID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("task not found")
		}
		return fmt.Errorf("failed to get task: %w", err)
	}

	if err := s.checkProjectAccess(ctx, task.ProjectID, userID, tenantID); err != nil {
		return err
	}

	// 获取同一作用域内的任务列表
	var tasks []models.AgileTask
	query := s.db.WithContext(ctx).Where("project_id = ? AND deleted_at IS NULL", task.ProjectID)
	
	// 如果指定了Sprint，则限定在同一Sprint内
	if req.SprintID != nil {
		query = query.Where("sprint_id = ?", *req.SprintID)
	} else if task.SprintID != nil {
		query = query.Where("sprint_id = ?", *task.SprintID)
	} else {
		query = query.Where("sprint_id IS NULL")
	}

	// 如果指定了列，则限定在同一列内
	if req.ColumnID != nil {
		query = query.Where("status = ?", *req.ColumnID)
	} else {
		query = query.Where("status = ?", task.Status)
	}

	if err := query.Order("rank ASC").Find(&tasks).Error; err != nil {
		return fmt.Errorf("failed to get tasks: %w", err)
	}

	// 验证目标索引
	if req.TargetIndex < 0 || req.TargetIndex > len(tasks) {
		return fmt.Errorf("invalid target index: %d", req.TargetIndex)
	}

	// 准备现有排名列表（排除被移动的任务）
	existingRanks := make([]string, 0, len(tasks)-1)
	var currentTaskIndex = -1
	
	for i, t := range tasks {
		if t.ID == req.TaskID {
			currentTaskIndex = i
			continue
		}
		existingRanks = append(existingRanks, t.Rank)
	}

	if currentTaskIndex == -1 {
		return fmt.Errorf("task not found in current scope")
	}

	// 调整目标索引（因为排除了当前任务）
	adjustedTargetIndex := req.TargetIndex
	if req.TargetIndex > currentTaskIndex {
		adjustedTargetIndex--
	}

	// 计算新的Lexorank
	lexorankManager := NewLexorankManager()
	newRank, err := lexorankManager.CalculateRankForInsertion(adjustedTargetIndex, existingRanks)
	if err != nil {
		s.logger.Error("生成新排名失败", zap.Error(err))
		return fmt.Errorf("failed to calculate new rank: %w", err)
	}

	// 开始事务更新
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 更新任务信息
	updates := map[string]interface{}{
		"rank": newRank,
	}

	if req.ColumnID != nil {
		updates["status"] = *req.ColumnID
	}

	if req.SprintID != nil {
		updates["sprint_id"] = *req.SprintID
	}

	if err := tx.Model(&task).Updates(updates).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update task: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("任务排序完成",
		zap.String("task_id", req.TaskID.String()),
		zap.Int("target_index", req.TargetIndex),
		zap.String("new_rank", newRank))

	return nil
}

// 注意：MoveTask、BatchReorderTasks、RebalanceTaskRanks、ValidateTaskOrder 方法已在其他文件中实现

// GetTasksByRank 按排序获取任务列表
func (s *agileServiceImpl) GetTasksByRank(ctx context.Context, filter *TaskFilter, page, pageSize int, userID, tenantID uuid.UUID) (*TaskListResponse, error) {
	if err := s.checkProjectAccess(ctx, filter.ProjectID, userID, tenantID); err != nil {
		return nil, err
	}

	query := s.db.WithContext(ctx).Where("project_id = ? AND deleted_at IS NULL", filter.ProjectID)

	// 应用过滤条件
	if filter.SprintID != nil {
		query = query.Where("sprint_id = ?", *filter.SprintID)
	}
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}
	if filter.AssigneeID != nil {
		query = query.Where("assignee_id = ?", *filter.AssigneeID)
	}
	if len(filter.Type) > 0 {
		query = query.Where("type IN ?", filter.Type)
	}
	if len(filter.Priority) > 0 {
		query = query.Where("priority IN ?", filter.Priority)
	}
	if filter.SearchText != nil && *filter.SearchText != "" {
		searchPattern := "%" + *filter.SearchText + "%"
		query = query.Where("title ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)
	}

	// 按Rank排序
	query = query.Order("rank ASC")

	var total int64
	if err := query.Model(&models.AgileTask{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count tasks: %w", err)
	}

	var tasks []models.AgileTask
	offset := (page - 1) * pageSize
	if err := query.
		Preload("Sprint").
		Preload("Epic").
		Preload("Assignee").
		Preload("Reporter").
		Limit(pageSize).
		Offset(offset).
		Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	return &TaskListResponse{
		Tasks:    tasks,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}