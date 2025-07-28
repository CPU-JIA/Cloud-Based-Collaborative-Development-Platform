package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AgileService 敏捷开发服务接口
type AgileService interface {
	// Sprint管理
	CreateSprint(ctx context.Context, req *CreateSprintRequest, userID, tenantID uuid.UUID) (*models.Sprint, error)
	GetSprint(ctx context.Context, sprintID uuid.UUID, userID, tenantID uuid.UUID) (*models.Sprint, error)
	ListSprints(ctx context.Context, projectID uuid.UUID, page, pageSize int, userID, tenantID uuid.UUID) (*SprintListResponse, error)
	UpdateSprint(ctx context.Context, sprintID uuid.UUID, req *UpdateSprintRequest, userID, tenantID uuid.UUID) (*models.Sprint, error)
	DeleteSprint(ctx context.Context, sprintID uuid.UUID, userID, tenantID uuid.UUID) error

	// Sprint状态管理
	StartSprint(ctx context.Context, sprintID uuid.UUID, userID, tenantID uuid.UUID) error
	CompleteSprint(ctx context.Context, sprintID uuid.UUID, userID, tenantID uuid.UUID) error

	// 敏捷任务管理
	CreateTask(ctx context.Context, req *CreateTaskRequest, userID, tenantID uuid.UUID) (*models.AgileTask, error)
	GetTask(ctx context.Context, taskID uuid.UUID, userID, tenantID uuid.UUID) (*models.AgileTask, error)
	ListTasks(ctx context.Context, filter *TaskFilter, page, pageSize int, userID, tenantID uuid.UUID) (*TaskListResponse, error)
	UpdateTask(ctx context.Context, taskID uuid.UUID, req *UpdateTaskRequest, userID, tenantID uuid.UUID) (*models.AgileTask, error)
	DeleteTask(ctx context.Context, taskID uuid.UUID, userID, tenantID uuid.UUID) error

	// 任务状态转换
	TransitionTask(ctx context.Context, taskID uuid.UUID, newStatus string, userID, tenantID uuid.UUID) error

	// 任务排序（拖拽）
	ReorderTasks(ctx context.Context, req *ReorderTasksRequest, userID, tenantID uuid.UUID) error
	MoveTask(ctx context.Context, req *TaskMoveRequest, userID, tenantID uuid.UUID) error
	BatchReorderTasks(ctx context.Context, req *BatchReorderRequest, userID, tenantID uuid.UUID) error
	RebalanceTaskRanks(ctx context.Context, projectID uuid.UUID, userID, tenantID uuid.UUID) error
	ValidateTaskOrder(ctx context.Context, projectID uuid.UUID, userID, tenantID uuid.UUID) error

	// 史诗管理
	CreateEpic(ctx context.Context, req *CreateEpicRequest, userID, tenantID uuid.UUID) (*models.Epic, error)
	GetEpic(ctx context.Context, epicID uuid.UUID, userID, tenantID uuid.UUID) (*models.Epic, error)
	ListEpics(ctx context.Context, projectID uuid.UUID, page, pageSize int, userID, tenantID uuid.UUID) (*EpicListResponse, error)
	UpdateEpic(ctx context.Context, epicID uuid.UUID, req *UpdateEpicRequest, userID, tenantID uuid.UUID) (*models.Epic, error)
	DeleteEpic(ctx context.Context, epicID uuid.UUID, userID, tenantID uuid.UUID) error

	// 看板管理
	CreateBoard(ctx context.Context, req *CreateBoardRequest, userID, tenantID uuid.UUID) (*models.Board, error)
	GetBoard(ctx context.Context, boardID uuid.UUID, userID, tenantID uuid.UUID) (*models.Board, error)
	ListBoards(ctx context.Context, projectID uuid.UUID, page, pageSize int, userID, tenantID uuid.UUID) (*BoardListResponse, error)
	UpdateBoard(ctx context.Context, boardID uuid.UUID, req *UpdateBoardRequest, userID, tenantID uuid.UUID) (*models.Board, error)
	DeleteBoard(ctx context.Context, boardID uuid.UUID, userID, tenantID uuid.UUID) error

	// 看板列管理
	CreateBoardColumn(ctx context.Context, req *CreateBoardColumnRequest, userID, tenantID uuid.UUID) (*models.BoardColumn, error)
	UpdateBoardColumn(ctx context.Context, columnID uuid.UUID, req *UpdateBoardColumnRequest, userID, tenantID uuid.UUID) (*models.BoardColumn, error)
	DeleteBoardColumn(ctx context.Context, columnID uuid.UUID, userID, tenantID uuid.UUID) error
	ReorderBoardColumns(ctx context.Context, req *ReorderColumnsRequest, userID, tenantID uuid.UUID) error

	// 看板任务移动
	BatchMoveTasks(ctx context.Context, req *BatchTaskMoveRequest, userID, tenantID uuid.UUID) error

	// 看板统计
	GetBoardStatistics(ctx context.Context, boardID uuid.UUID, userID, tenantID uuid.UUID) (*BoardStatistics, error)

	// 工作日志管理
	LogWork(ctx context.Context, req *LogWorkRequest, userID, tenantID uuid.UUID) (*models.WorkLog, error)
	GetWorkLogs(ctx context.Context, taskID uuid.UUID, userID, tenantID uuid.UUID) ([]models.WorkLog, error)
	DeleteWorkLog(ctx context.Context, workLogID uuid.UUID, userID, tenantID uuid.UUID) error

	// 任务评论管理
	AddComment(ctx context.Context, req *AddCommentRequest, userID, tenantID uuid.UUID) (*models.TaskComment, error)
	GetComments(ctx context.Context, taskID uuid.UUID, userID, tenantID uuid.UUID) ([]models.TaskComment, error)
	UpdateComment(ctx context.Context, commentID uuid.UUID, content string, userID, tenantID uuid.UUID) (*models.TaskComment, error)
	DeleteComment(ctx context.Context, commentID uuid.UUID, userID, tenantID uuid.UUID) error

	// 报表和统计
	GetSprintBurndown(ctx context.Context, sprintID uuid.UUID, userID, tenantID uuid.UUID) (*BurndownData, error)
	GetSprintBurndownData(ctx context.Context, sprintID uuid.UUID, userID, tenantID uuid.UUID) (*BurndownData, error)
	GetVelocityChart(ctx context.Context, projectID uuid.UUID, userID, tenantID uuid.UUID) (*VelocityData, error)
	GetTaskStatistics(ctx context.Context, projectID uuid.UUID, userID, tenantID uuid.UUID) (*TaskStatistics, error)
	GetProjectStatistics(ctx context.Context, projectID uuid.UUID, userID, tenantID uuid.UUID) (*ProjectStatistics, error)

	// 任务操作
	UpdateTaskStatus(ctx context.Context, taskID uuid.UUID, status string, userID, tenantID uuid.UUID) error
	AssignTask(ctx context.Context, taskID uuid.UUID, assigneeID *uuid.UUID, userID, tenantID uuid.UUID) error
	GetUserWorkload(ctx context.Context, targetUserID uuid.UUID, userID, tenantID uuid.UUID) (*UserWorkload, error)
}

// agileServiceImpl 敏捷开发服务实现
type agileServiceImpl struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewAgileService 创建敏捷开发服务
func NewAgileService(db *gorm.DB, logger *zap.Logger) AgileService {
	return &agileServiceImpl{
		db:     db,
		logger: logger,
	}
}

// Sprint管理实现

func (s *agileServiceImpl) CreateSprint(ctx context.Context, req *CreateSprintRequest, userID, tenantID uuid.UUID) (*models.Sprint, error) {
	// 验证项目访问权限
	if err := s.checkProjectAccess(ctx, req.ProjectID, userID, tenantID); err != nil {
		return nil, err
	}

	// 验证时间范围
	if req.EndDate.Before(req.StartDate) {
		return nil, errors.New("sprint end date must be after start date")
	}

	sprint := &models.Sprint{
		ProjectID:   req.ProjectID,
		Name:        req.Name,
		Description: req.Description,
		Goal:        req.Goal,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Capacity:    req.Capacity,
		Status:      models.SprintStatusPlanned,
		CreatedBy:   &userID,
	}

	if err := s.db.WithContext(ctx).Create(sprint).Error; err != nil {
		s.logger.Error("创建Sprint失败", zap.Error(err))
		return nil, fmt.Errorf("failed to create sprint: %w", err)
	}

	return sprint, nil
}

func (s *agileServiceImpl) GetSprint(ctx context.Context, sprintID uuid.UUID, userID, tenantID uuid.UUID) (*models.Sprint, error) {
	var sprint models.Sprint

	query := s.db.WithContext(ctx).
		Preload("Tasks").
		Preload("Creator")

	if err := query.First(&sprint, "id = ? AND deleted_at IS NULL", sprintID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("sprint not found")
		}
		return nil, fmt.Errorf("failed to get sprint: %w", err)
	}

	// 检查项目访问权限
	if err := s.checkProjectAccess(ctx, sprint.ProjectID, userID, tenantID); err != nil {
		return nil, err
	}

	return &sprint, nil
}

func (s *agileServiceImpl) ListSprints(ctx context.Context, projectID uuid.UUID, page, pageSize int, userID, tenantID uuid.UUID) (*SprintListResponse, error) {
	// 验证项目访问权限
	if err := s.checkProjectAccess(ctx, projectID, userID, tenantID); err != nil {
		return nil, err
	}

	var sprints []models.Sprint
	var total int64

	baseQuery := s.db.WithContext(ctx).
		Model(&models.Sprint{}).
		Where("project_id = ? AND deleted_at IS NULL", projectID)

	// 获取总数
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count sprints: %w", err)
	}

	// 获取分页数据
	offset := (page - 1) * pageSize
	if err := baseQuery.
		Preload("Tasks").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&sprints).Error; err != nil {
		return nil, fmt.Errorf("failed to list sprints: %w", err)
	}

	return &SprintListResponse{
		Sprints:  sprints,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *agileServiceImpl) UpdateSprint(ctx context.Context, sprintID uuid.UUID, req *UpdateSprintRequest, userID, tenantID uuid.UUID) (*models.Sprint, error) {
	var sprint models.Sprint

	if err := s.db.WithContext(ctx).First(&sprint, "id = ? AND deleted_at IS NULL", sprintID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("sprint not found")
		}
		return nil, fmt.Errorf("failed to get sprint: %w", err)
	}

	// 检查项目访问权限
	if err := s.checkProjectAccess(ctx, sprint.ProjectID, userID, tenantID); err != nil {
		return nil, err
	}

	// 更新字段
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = req.Description
	}
	if req.Goal != nil {
		updates["goal"] = req.Goal
	}
	if req.StartDate != nil {
		updates["start_date"] = *req.StartDate
	}
	if req.EndDate != nil {
		updates["end_date"] = *req.EndDate
	}
	if req.Capacity != nil {
		updates["capacity"] = *req.Capacity
	}

	if err := s.db.WithContext(ctx).Model(&sprint).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update sprint: %w", err)
	}

	return &sprint, nil
}

func (s *agileServiceImpl) DeleteSprint(ctx context.Context, sprintID uuid.UUID, userID, tenantID uuid.UUID) error {
	var sprint models.Sprint

	if err := s.db.WithContext(ctx).First(&sprint, "id = ? AND deleted_at IS NULL", sprintID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("sprint not found")
		}
		return fmt.Errorf("failed to get sprint: %w", err)
	}

	// 检查项目访问权限
	if err := s.checkProjectAccess(ctx, sprint.ProjectID, userID, tenantID); err != nil {
		return err
	}

	// 软删除
	if err := s.db.WithContext(ctx).Model(&sprint).Update("deleted_at", time.Now()).Error; err != nil {
		return fmt.Errorf("failed to delete sprint: %w", err)
	}

	return nil
}

func (s *agileServiceImpl) StartSprint(ctx context.Context, sprintID uuid.UUID, userID, tenantID uuid.UUID) error {
	var sprint models.Sprint

	if err := s.db.WithContext(ctx).First(&sprint, "id = ? AND deleted_at IS NULL", sprintID).Error; err != nil {
		return fmt.Errorf("sprint not found")
	}

	// 检查项目访问权限
	if err := s.checkProjectAccess(ctx, sprint.ProjectID, userID, tenantID); err != nil {
		return err
	}

	// 检查状态转换是否合法
	if sprint.Status != models.SprintStatusPlanned {
		return fmt.Errorf("can only start planned sprint")
	}

	// 更新状态
	if err := s.db.WithContext(ctx).Model(&sprint).Update("status", models.SprintStatusActive).Error; err != nil {
		return fmt.Errorf("failed to start sprint: %w", err)
	}

	return nil
}

func (s *agileServiceImpl) CompleteSprint(ctx context.Context, sprintID uuid.UUID, userID, tenantID uuid.UUID) error {
	var sprint models.Sprint

	if err := s.db.WithContext(ctx).First(&sprint, "id = ? AND deleted_at IS NULL", sprintID).Error; err != nil {
		return fmt.Errorf("sprint not found")
	}

	// 检查项目访问权限
	if err := s.checkProjectAccess(ctx, sprint.ProjectID, userID, tenantID); err != nil {
		return err
	}

	// 检查状态转换是否合法
	if sprint.Status != models.SprintStatusActive {
		return fmt.Errorf("can only complete active sprint")
	}

	// 更新状态
	if err := s.db.WithContext(ctx).Model(&sprint).Update("status", models.SprintStatusClosed).Error; err != nil {
		return fmt.Errorf("failed to complete sprint: %w", err)
	}

	return nil
}

// 敏捷任务管理实现

func (s *agileServiceImpl) CreateTask(ctx context.Context, req *CreateTaskRequest, userID, tenantID uuid.UUID) (*models.AgileTask, error) {
	// 验证项目访问权限
	if err := s.checkProjectAccess(ctx, req.ProjectID, userID, tenantID); err != nil {
		return nil, err
	}

	// 生成排序权重（简化实现，实际应使用Lexorank算法）
	rank := s.generateRank(ctx, req.ProjectID)

	task := &models.AgileTask{
		ProjectID:          req.ProjectID,
		SprintID:           req.SprintID,
		EpicID:             req.EpicID,
		ParentID:           req.ParentID,
		Title:              req.Title,
		Description:        req.Description,
		Type:               req.Type,
		Status:             models.TaskStatusTodo,
		Priority:           req.Priority,
		StoryPoints:        req.StoryPoints,
		OriginalEstimate:   req.OriginalEstimate,
		AssigneeID:         req.AssigneeID,
		ReporterID:         userID,
		Labels:             req.Labels,
		Components:         req.Components,
		AcceptanceCriteria: req.AcceptanceCriteria,
		Rank:               rank,
	}

	if err := s.db.WithContext(ctx).Create(task).Error; err != nil {
		s.logger.Error("创建任务失败", zap.Error(err))
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// 加载关联数据
	if err := s.loadTaskAssociations(ctx, task); err != nil {
		s.logger.Warn("加载任务关联数据失败", zap.Error(err))
	}

	return task, nil
}

func (s *agileServiceImpl) GetTask(ctx context.Context, taskID uuid.UUID, userID, tenantID uuid.UUID) (*models.AgileTask, error) {
	var task models.AgileTask

	query := s.db.WithContext(ctx).
		Preload("Sprint").
		Preload("Epic").
		Preload("Parent").
		Preload("Children").
		Preload("Assignee").
		Preload("Reporter").
		Preload("Comments", "deleted_at IS NULL").
		Preload("Comments.Author").
		Preload("WorkLogs", "deleted_at IS NULL").
		Preload("WorkLogs.User")

	if err := query.First(&task, "id = ? AND deleted_at IS NULL", taskID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("task not found")
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// 检查项目访问权限
	if err := s.checkProjectAccess(ctx, task.ProjectID, userID, tenantID); err != nil {
		return nil, err
	}

	return &task, nil
}

func (s *agileServiceImpl) ListTasks(ctx context.Context, filter *TaskFilter, page, pageSize int, userID, tenantID uuid.UUID) (*TaskListResponse, error) {
	// 验证项目访问权限
	if err := s.checkProjectAccess(ctx, filter.ProjectID, userID, tenantID); err != nil {
		return nil, err
	}

	var tasks []models.AgileTask
	var total int64

	baseQuery := s.db.WithContext(ctx).
		Model(&models.AgileTask{}).
		Where("project_id = ? AND deleted_at IS NULL", filter.ProjectID)

	// 应用过滤条件
	s.applyTaskFilters(baseQuery, filter)

	// 获取总数
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count tasks: %w", err)
	}

	// 获取分页数据
	offset := (page - 1) * pageSize
	if err := baseQuery.
		Preload("Sprint").
		Preload("Epic").
		Preload("Assignee").
		Preload("Reporter").
		Order("rank ASC, created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	return &TaskListResponse{
		Tasks:    tasks,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *agileServiceImpl) UpdateTask(ctx context.Context, taskID uuid.UUID, req *UpdateTaskRequest, userID, tenantID uuid.UUID) (*models.AgileTask, error) {
	var task models.AgileTask

	if err := s.db.WithContext(ctx).First(&task, "id = ? AND deleted_at IS NULL", taskID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("task not found")
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// 检查项目访问权限
	if err := s.checkProjectAccess(ctx, task.ProjectID, userID, tenantID); err != nil {
		return nil, err
	}

	// 准备更新数据
	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = req.Description
	}
	if req.Status != nil {
		// 验证状态转换是否合法
		if !task.CanTransitionTo(*req.Status) {
			return nil, fmt.Errorf("invalid status transition from %s to %s", task.Status, *req.Status)
		}
		updates["status"] = *req.Status
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.StoryPoints != nil {
		updates["story_points"] = req.StoryPoints
	}
	if req.OriginalEstimate != nil {
		updates["original_estimate"] = req.OriginalEstimate
	}
	if req.RemainingTime != nil {
		updates["remaining_time"] = req.RemainingTime
	}
	if req.AssigneeID != nil {
		updates["assignee_id"] = req.AssigneeID
	}
	if req.SprintID != nil {
		updates["sprint_id"] = req.SprintID
	}
	if req.EpicID != nil {
		updates["epic_id"] = req.EpicID
	}
	if req.Labels != nil {
		updates["labels"] = req.Labels
	}
	if req.Components != nil {
		updates["components"] = req.Components
	}
	if req.AcceptanceCriteria != nil {
		updates["acceptance_criteria"] = req.AcceptanceCriteria
	}

	if err := s.db.WithContext(ctx).Model(&task).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	// 重新加载任务数据
	if err := s.loadTaskAssociations(ctx, &task); err != nil {
		s.logger.Warn("加载任务关联数据失败", zap.Error(err))
	}

	return &task, nil
}

// 工具方法

func (s *agileServiceImpl) checkProjectAccess(ctx context.Context, projectID, userID, tenantID uuid.UUID) error {
	var count int64
	err := s.db.WithContext(ctx).
		Table("projects p").
		Joins("LEFT JOIN project_members pm ON p.id = pm.project_id").
		Where("p.id = ? AND p.tenant_id = ? AND (p.manager_id = ? OR pm.user_id = ?)",
			projectID, tenantID, userID, userID).
		Count(&count).Error

	if err != nil {
		return fmt.Errorf("failed to check project access: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("no access to project")
	}

	return nil
}

func (s *agileServiceImpl) generateRank(ctx context.Context, projectID uuid.UUID) string {
	// 简化的排序权重生成，实际应使用Lexorank算法
	var maxRank string
	s.db.WithContext(ctx).
		Model(&models.AgileTask{}).
		Where("project_id = ? AND deleted_at IS NULL", projectID).
		Order("rank DESC").
		Limit(1).
		Pluck("rank", &maxRank)

	if maxRank == "" {
		return "1000000"
	}

	// 简化实现：将现有最大rank转换为数字并加1000
	if rank, err := strconv.Atoi(maxRank); err == nil {
		return strconv.Itoa(rank + 1000)
	}

	return "1000000"
}

func (s *agileServiceImpl) loadTaskAssociations(ctx context.Context, task *models.AgileTask) error {
	return s.db.WithContext(ctx).
		Preload("Sprint").
		Preload("Epic").
		Preload("Parent").
		Preload("Assignee").
		Preload("Reporter").
		First(task, task.ID).Error
}

func (s *agileServiceImpl) applyTaskFilters(query *gorm.DB, filter *TaskFilter) {
	if filter.SprintID != nil {
		query = query.Where("sprint_id = ?", *filter.SprintID)
	}
	if filter.EpicID != nil {
		query = query.Where("epic_id = ?", *filter.EpicID)
	}
	if filter.AssigneeID != nil {
		query = query.Where("assignee_id = ?", *filter.AssigneeID)
	}
	if filter.Status != nil && len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}
	if filter.Type != nil && len(filter.Type) > 0 {
		query = query.Where("type IN ?", filter.Type)
	}
	if filter.Priority != nil && len(filter.Priority) > 0 {
		query = query.Where("priority IN ?", filter.Priority)
	}
	if filter.SearchText != nil && *filter.SearchText != "" {
		searchPattern := "%" + *filter.SearchText + "%"
		query = query.Where("title ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)
	}
}

func (s *agileServiceImpl) DeleteTask(ctx context.Context, taskID uuid.UUID, userID, tenantID uuid.UUID) error {
	var task models.AgileTask

	if err := s.db.WithContext(ctx).First(&task, "id = ? AND deleted_at IS NULL", taskID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("task not found")
		}
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 检查项目访问权限
	if err := s.checkProjectAccess(ctx, task.ProjectID, userID, tenantID); err != nil {
		return err
	}

	// 软删除
	if err := s.db.WithContext(ctx).Model(&task).Update("deleted_at", time.Now()).Error; err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	return nil
}

func (s *agileServiceImpl) TransitionTask(ctx context.Context, taskID uuid.UUID, newStatus string, userID, tenantID uuid.UUID) error {
	var task models.AgileTask

	if err := s.db.WithContext(ctx).First(&task, "id = ? AND deleted_at IS NULL", taskID).Error; err != nil {
		return fmt.Errorf("task not found")
	}

	// 检查项目访问权限
	if err := s.checkProjectAccess(ctx, task.ProjectID, userID, tenantID); err != nil {
		return err
	}

	// 验证状态转换是否合法
	if !task.CanTransitionTo(newStatus) {
		return fmt.Errorf("invalid status transition from %s to %s", task.Status, newStatus)
	}

	// 更新状态
	if err := s.db.WithContext(ctx).Model(&task).Update("status", newStatus).Error; err != nil {
		return fmt.Errorf("failed to transition task: %w", err)
	}

	return nil
}

func (s *agileServiceImpl) ReorderTasks(ctx context.Context, req *ReorderTasksRequest, userID, tenantID uuid.UUID) error {
	// 验证项目访问权限
	if err := s.checkProjectAccess(ctx, req.ProjectID, userID, tenantID); err != nil {
		return err
	}

	// 批量更新任务排序
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, taskRank := range req.Tasks {
		if err := tx.Model(&models.AgileTask{}).
			Where("id = ? AND project_id = ? AND deleted_at IS NULL", taskRank.TaskID, req.ProjectID).
			Update("rank", taskRank.Rank).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to update task rank: %w", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit rank updates: %w", err)
	}

	return nil
}

// 史诗管理实现

func (s *agileServiceImpl) CreateEpic(ctx context.Context, req *CreateEpicRequest, userID, tenantID uuid.UUID) (*models.Epic, error) {
	// 验证项目访问权限
	if err := s.checkProjectAccess(ctx, req.ProjectID, userID, tenantID); err != nil {
		return nil, err
	}

	epic := &models.Epic{
		ProjectID:       req.ProjectID,
		Name:            req.Name,
		Description:     req.Description,
		Color:           req.Color,
		StartDate:       req.StartDate,
		EndDate:         req.EndDate,
		Goal:            req.Goal,
		SuccessCriteria: req.SuccessCriteria,
		Status:          models.EpicStatusOpen,
		CreatedBy:       &userID,
	}

	if err := s.db.WithContext(ctx).Create(epic).Error; err != nil {
		s.logger.Error("创建史诗失败", zap.Error(err))
		return nil, fmt.Errorf("failed to create epic: %w", err)
	}

	return epic, nil
}

func (s *agileServiceImpl) GetEpic(ctx context.Context, epicID uuid.UUID, userID, tenantID uuid.UUID) (*models.Epic, error) {
	var epic models.Epic

	query := s.db.WithContext(ctx).
		Preload("Tasks", "deleted_at IS NULL").
		Preload("Creator")

	if err := query.First(&epic, "id = ? AND deleted_at IS NULL", epicID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("epic not found")
		}
		return nil, fmt.Errorf("failed to get epic: %w", err)
	}

	// 检查项目访问权限
	if err := s.checkProjectAccess(ctx, epic.ProjectID, userID, tenantID); err != nil {
		return nil, err
	}

	return &epic, nil
}

func (s *agileServiceImpl) ListEpics(ctx context.Context, projectID uuid.UUID, page, pageSize int, userID, tenantID uuid.UUID) (*EpicListResponse, error) {
	// 验证项目访问权限
	if err := s.checkProjectAccess(ctx, projectID, userID, tenantID); err != nil {
		return nil, err
	}

	var epics []models.Epic
	var total int64

	baseQuery := s.db.WithContext(ctx).
		Model(&models.Epic{}).
		Where("project_id = ? AND deleted_at IS NULL", projectID)

	// 获取总数
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count epics: %w", err)
	}

	// 获取分页数据
	offset := (page - 1) * pageSize
	if err := baseQuery.
		Preload("Tasks", "deleted_at IS NULL").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&epics).Error; err != nil {
		return nil, fmt.Errorf("failed to list epics: %w", err)
	}

	return &EpicListResponse{
		Epics:    epics,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *agileServiceImpl) UpdateEpic(ctx context.Context, epicID uuid.UUID, req *UpdateEpicRequest, userID, tenantID uuid.UUID) (*models.Epic, error) {
	var epic models.Epic

	if err := s.db.WithContext(ctx).First(&epic, "id = ? AND deleted_at IS NULL", epicID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("epic not found")
		}
		return nil, fmt.Errorf("failed to get epic: %w", err)
	}

	// 检查项目访问权限
	if err := s.checkProjectAccess(ctx, epic.ProjectID, userID, tenantID); err != nil {
		return nil, err
	}

	// 更新字段
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = req.Description
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Color != nil {
		updates["color"] = req.Color
	}
	if req.StartDate != nil {
		updates["start_date"] = req.StartDate
	}
	if req.EndDate != nil {
		updates["end_date"] = req.EndDate
	}
	if req.Goal != nil {
		updates["goal"] = req.Goal
	}
	if req.SuccessCriteria != nil {
		updates["success_criteria"] = req.SuccessCriteria
	}

	if err := s.db.WithContext(ctx).Model(&epic).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update epic: %w", err)
	}

	return &epic, nil
}

func (s *agileServiceImpl) DeleteEpic(ctx context.Context, epicID uuid.UUID, userID, tenantID uuid.UUID) error {
	var epic models.Epic

	if err := s.db.WithContext(ctx).First(&epic, "id = ? AND deleted_at IS NULL", epicID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("epic not found")
		}
		return fmt.Errorf("failed to get epic: %w", err)
	}

	// 检查项目访问权限
	if err := s.checkProjectAccess(ctx, epic.ProjectID, userID, tenantID); err != nil {
		return err
	}

	// 软删除
	if err := s.db.WithContext(ctx).Model(&epic).Update("deleted_at", time.Now()).Error; err != nil {
		return fmt.Errorf("failed to delete epic: %w", err)
	}

	return nil
}

// 看板管理实现

func (s *agileServiceImpl) CreateBoard(ctx context.Context, req *CreateBoardRequest, userID, tenantID uuid.UUID) (*models.Board, error) {
	// 验证项目访问权限
	if err := s.checkProjectAccess(ctx, req.ProjectID, userID, tenantID); err != nil {
		return nil, err
	}

	board := &models.Board{
		ProjectID:   req.ProjectID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		CreatedBy:   &userID,
	}

	if err := s.db.WithContext(ctx).Create(board).Error; err != nil {
		s.logger.Error("创建看板失败", zap.Error(err))
		return nil, fmt.Errorf("failed to create board: %w", err)
	}

	return board, nil
}

func (s *agileServiceImpl) GetBoard(ctx context.Context, boardID uuid.UUID, userID, tenantID uuid.UUID) (*models.Board, error) {
	var board models.Board

	query := s.db.WithContext(ctx).
		Preload("Columns", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("Creator")

	if err := query.First(&board, "id = ? AND deleted_at IS NULL", boardID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("board not found")
		}
		return nil, fmt.Errorf("failed to get board: %w", err)
	}

	// 检查项目访问权限
	if err := s.checkProjectAccess(ctx, board.ProjectID, userID, tenantID); err != nil {
		return nil, err
	}

	return &board, nil
}

func (s *agileServiceImpl) ListBoards(ctx context.Context, projectID uuid.UUID, page, pageSize int, userID, tenantID uuid.UUID) (*BoardListResponse, error) {
	// 验证项目访问权限
	if err := s.checkProjectAccess(ctx, projectID, userID, tenantID); err != nil {
		return nil, err
	}

	var boards []models.Board
	var total int64

	// 计算总数
	if err := s.db.WithContext(ctx).Model(&models.Board{}).
		Where("project_id = ? AND deleted_at IS NULL", projectID).
		Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count boards: %w", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := s.db.WithContext(ctx).
		Where("project_id = ? AND deleted_at IS NULL", projectID).
		Preload("Columns", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&boards).Error; err != nil {
		return nil, fmt.Errorf("failed to list boards: %w", err)
	}

	return &BoardListResponse{
		Boards:   boards,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *agileServiceImpl) UpdateBoard(ctx context.Context, boardID uuid.UUID, req *UpdateBoardRequest, userID, tenantID uuid.UUID) (*models.Board, error) {
	var board models.Board

	if err := s.db.WithContext(ctx).First(&board, "id = ? AND deleted_at IS NULL", boardID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("board not found")
		}
		return nil, fmt.Errorf("failed to get board: %w", err)
	}

	// 检查项目访问权限
	if err := s.checkProjectAccess(ctx, board.ProjectID, userID, tenantID); err != nil {
		return nil, err
	}

	// 更新字段
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = req.Description
	}

	if err := s.db.WithContext(ctx).Model(&board).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update board: %w", err)
	}

	return &board, nil
}

func (s *agileServiceImpl) DeleteBoard(ctx context.Context, boardID uuid.UUID, userID, tenantID uuid.UUID) error {
	var board models.Board

	if err := s.db.WithContext(ctx).First(&board, "id = ? AND deleted_at IS NULL", boardID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("board not found")
		}
		return fmt.Errorf("failed to get board: %w", err)
	}

	// 检查项目访问权限
	if err := s.checkProjectAccess(ctx, board.ProjectID, userID, tenantID); err != nil {
		return err
	}

	// 软删除
	if err := s.db.WithContext(ctx).Model(&board).Update("deleted_at", time.Now()).Error; err != nil {
		return fmt.Errorf("failed to delete board: %w", err)
	}

	return nil
}

// GetSprintBurndownData 获取Sprint燃尽图数据
func (s *agileServiceImpl) GetSprintBurndownData(ctx context.Context, sprintID uuid.UUID, userID, tenantID uuid.UUID) (*BurndownData, error) {
	// 重用现有的GetSprintBurndown方法
	return s.GetSprintBurndown(ctx, sprintID, userID, tenantID)
}

// GetProjectStatistics 获取项目统计数据
func (s *agileServiceImpl) GetProjectStatistics(ctx context.Context, projectID uuid.UUID, userID, tenantID uuid.UUID) (*ProjectStatistics, error) {
	if err := s.checkProjectAccess(ctx, projectID, userID, tenantID); err != nil {
		return nil, err
	}

	// 获取任务统计
	var totalTasks, completedTasks, activeTasks int64

	// 总任务数
	if err := s.db.WithContext(ctx).Model(&models.AgileTask{}).
		Where("project_id = ? AND deleted_at IS NULL", projectID).
		Count(&totalTasks).Error; err != nil {
		return nil, fmt.Errorf("failed to count total tasks: %w", err)
	}

	// 已完成任务数
	if err := s.db.WithContext(ctx).Model(&models.AgileTask{}).
		Where("project_id = ? AND status = 'done' AND deleted_at IS NULL", projectID).
		Count(&completedTasks).Error; err != nil {
		return nil, fmt.Errorf("failed to count completed tasks: %w", err)
	}

	// 进行中任务数
	if err := s.db.WithContext(ctx).Model(&models.AgileTask{}).
		Where("project_id = ? AND status IN ('in_progress', 'in_review') AND deleted_at IS NULL", projectID).
		Count(&activeTasks).Error; err != nil {
		return nil, fmt.Errorf("failed to count active tasks: %w", err)
	}

	// 获取Sprint统计
	var totalSprints, activeSprints int64

	if err := s.db.WithContext(ctx).Model(&models.Sprint{}).
		Where("project_id = ? AND deleted_at IS NULL", projectID).
		Count(&totalSprints).Error; err != nil {
		return nil, fmt.Errorf("failed to count total sprints: %w", err)
	}

	if err := s.db.WithContext(ctx).Model(&models.Sprint{}).
		Where("project_id = ? AND status = 'active' AND deleted_at IS NULL", projectID).
		Count(&activeSprints).Error; err != nil {
		return nil, fmt.Errorf("failed to count active sprints: %w", err)
	}

	// 计算完成率
	var completionRate float64
	if totalTasks > 0 {
		completionRate = float64(completedTasks) / float64(totalTasks) * 100
	}

	statistics := &ProjectStatistics{
		TotalTasks:     totalTasks,
		CompletedTasks: completedTasks,
		ActiveTasks:    activeTasks,
		TotalSprints:   totalSprints,
		ActiveSprints:  activeSprints,
		CompletionRate: completionRate,
		ProjectID:      projectID,
		UpdatedAt:      time.Now(),
	}

	return statistics, nil
}

// UpdateTaskStatus 更新任务状态
func (s *agileServiceImpl) UpdateTaskStatus(ctx context.Context, taskID uuid.UUID, status string, userID, tenantID uuid.UUID) error {
	var task models.AgileTask

	if err := s.db.WithContext(ctx).First(&task, "id = ? AND deleted_at IS NULL", taskID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("task not found")
		}
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 检查项目访问权限
	if err := s.checkProjectAccess(ctx, task.ProjectID, userID, tenantID); err != nil {
		return err
	}

	// 更新状态
	if err := s.db.WithContext(ctx).Model(&task).Update("status", status).Error; err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	return nil
}

// AssignTask 分配任务
func (s *agileServiceImpl) AssignTask(ctx context.Context, taskID uuid.UUID, assigneeID *uuid.UUID, userID, tenantID uuid.UUID) error {
	var task models.AgileTask

	if err := s.db.WithContext(ctx).First(&task, "id = ? AND deleted_at IS NULL", taskID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("task not found")
		}
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 检查项目访问权限
	if err := s.checkProjectAccess(ctx, task.ProjectID, userID, tenantID); err != nil {
		return err
	}

	// 分配任务
	if err := s.db.WithContext(ctx).Model(&task).Update("assignee_id", assigneeID).Error; err != nil {
		return fmt.Errorf("failed to assign task: %w", err)
	}

	return nil
}

// GetUserWorkload 获取用户工作负载
func (s *agileServiceImpl) GetUserWorkload(ctx context.Context, targetUserID uuid.UUID, userID, tenantID uuid.UUID) (*UserWorkload, error) {
	// 获取用户的活跃任务
	var tasks []models.AgileTask

	if err := s.db.WithContext(ctx).
		Where("assignee_id = ? AND status NOT IN ('done', 'cancelled') AND deleted_at IS NULL", targetUserID).
		Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to get user tasks: %w", err)
	}

	// 统计不同状态的任务数量
	statusCounts := make(map[string]int)
	totalEstimatedHours := 0

	for _, task := range tasks {
		statusCounts[task.Status]++
		if task.OriginalEstimate != nil {
			totalEstimatedHours += int(*task.OriginalEstimate)
		}
	}

	workload := &UserWorkload{
		UserID:              targetUserID,
		TotalTasks:          len(tasks),
		TasksByStatus:       statusCounts,
		TotalEstimatedHours: totalEstimatedHours,
		UpdatedAt:           time.Now(),
	}

	return workload, nil
}

// BatchReorderTasks 批量重排序任务
func (s *agileServiceImpl) BatchReorderTasks(ctx context.Context, req *BatchReorderRequest, userID, tenantID uuid.UUID) error {
	// 验证项目访问权限
	if err := s.checkProjectAccess(ctx, req.ProjectID, userID, tenantID); err != nil {
		return err
	}

	// 批量更新任务排序
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 构建基础查询
	baseQuery := tx.Model(&models.AgileTask{}).
		Where("project_id = ? AND deleted_at IS NULL", req.ProjectID)

	// 添加过滤条件
	if req.SprintID != nil {
		baseQuery = baseQuery.Where("sprint_id = ?", *req.SprintID)
	}
	if req.Status != nil {
		baseQuery = baseQuery.Where("status = ?", *req.Status)
	}

	// 批量重新设置排序权重
	for i, taskID := range req.TaskIDs {
		newRank := fmt.Sprintf("batch_rank_%d_%d", time.Now().Unix(), i)
		if err := baseQuery.
			Where("id = ?", taskID).
			Update("rank", newRank).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to update task rank: %w", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit batch reorder: %w", err)
	}

	return nil
}

// RebalanceTaskRanks 重新平衡任务排序权重
func (s *agileServiceImpl) RebalanceTaskRanks(ctx context.Context, projectID uuid.UUID, userID, tenantID uuid.UUID) error {
	// 验证项目访问权限
	if err := s.checkProjectAccess(ctx, projectID, userID, tenantID); err != nil {
		return err
	}

	// 获取所有任务，按当前rank排序
	var tasks []models.AgileTask
	if err := s.db.WithContext(ctx).
		Where("project_id = ? AND deleted_at IS NULL", projectID).
		Order("rank ASC, created_at ASC").
		Find(&tasks).Error; err != nil {
		return fmt.Errorf("failed to get tasks for rebalancing: %w", err)
	}

	// 重新分配均匀的rank值
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for i, task := range tasks {
		newRank := fmt.Sprintf("rebalanced_%06d", i*1000)
		if err := tx.Model(&task).Update("rank", newRank).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to rebalance task rank: %w", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit rank rebalance: %w", err)
	}

	s.logger.Info("任务rank重新平衡完成", zap.String("project_id", projectID.String()), zap.Int("task_count", len(tasks)))
	return nil
}

// ValidateTaskOrder 验证任务排序的一致性
func (s *agileServiceImpl) ValidateTaskOrder(ctx context.Context, projectID uuid.UUID, userID, tenantID uuid.UUID) error {
	// 验证项目访问权限
	if err := s.checkProjectAccess(ctx, projectID, userID, tenantID); err != nil {
		return err
	}

	// 获取所有任务的rank值
	var ranks []string
	if err := s.db.WithContext(ctx).
		Model(&models.AgileTask{}).
		Where("project_id = ? AND deleted_at IS NULL", projectID).
		Pluck("rank", &ranks).Error; err != nil {
		return fmt.Errorf("failed to get task ranks: %w", err)
	}

	// 检查是否有重复的rank
	rankSet := make(map[string]bool)
	duplicates := []string{}

	for _, rank := range ranks {
		if rankSet[rank] {
			duplicates = append(duplicates, rank)
		}
		rankSet[rank] = true
	}

	if len(duplicates) > 0 {
		s.logger.Warn("发现重复的task rank值",
			zap.String("project_id", projectID.String()),
			zap.Strings("duplicates", duplicates))

		// 自动修复重复的rank值
		return s.RebalanceTaskRanks(ctx, projectID, userID, tenantID)
	}

	s.logger.Info("任务排序验证通过",
		zap.String("project_id", projectID.String()),
		zap.Int("task_count", len(ranks)))

	return nil
}

// 看板列管理实现

func (s *agileServiceImpl) CreateBoardColumn(ctx context.Context, req *CreateBoardColumnRequest, userID, tenantID uuid.UUID) (*models.BoardColumn, error) {
	// 验证看板访问权限
	var board models.Board
	if err := s.db.WithContext(ctx).First(&board, "id = ? AND deleted_at IS NULL", req.BoardID).Error; err != nil {
		return nil, fmt.Errorf("board not found")
	}

	if err := s.checkProjectAccess(ctx, board.ProjectID, userID, tenantID); err != nil {
		return nil, err
	}

	column := &models.BoardColumn{
		BoardID:  req.BoardID,
		Name:     req.Name,
		Position: req.Position,
		WIPLimit: req.WIPLimit,
		Status:   req.Status,
		Color:    req.Color,
	}

	if err := s.db.WithContext(ctx).Create(column).Error; err != nil {
		s.logger.Error("创建看板列失败", zap.Error(err))
		return nil, fmt.Errorf("failed to create board column: %w", err)
	}

	return column, nil
}

func (s *agileServiceImpl) UpdateBoardColumn(ctx context.Context, columnID uuid.UUID, req *UpdateBoardColumnRequest, userID, tenantID uuid.UUID) (*models.BoardColumn, error) {
	var column models.BoardColumn

	if err := s.db.WithContext(ctx).Preload("Board").First(&column, "id = ? AND deleted_at IS NULL", columnID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("board column not found")
		}
		return nil, fmt.Errorf("failed to get board column: %w", err)
	}

	// 检查项目访问权限
	if err := s.checkProjectAccess(ctx, column.Board.ProjectID, userID, tenantID); err != nil {
		return nil, err
	}

	// 更新字段
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Position != nil {
		updates["position"] = *req.Position
	}
	if req.WIPLimit != nil {
		updates["wip_limit"] = req.WIPLimit
	}
	if req.Color != nil {
		updates["color"] = req.Color
	}

	if err := s.db.WithContext(ctx).Model(&column).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update board column: %w", err)
	}

	return &column, nil
}

func (s *agileServiceImpl) DeleteBoardColumn(ctx context.Context, columnID uuid.UUID, userID, tenantID uuid.UUID) error {
	var column models.BoardColumn

	if err := s.db.WithContext(ctx).Preload("Board").First(&column, "id = ? AND deleted_at IS NULL", columnID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("board column not found")
		}
		return fmt.Errorf("failed to get board column: %w", err)
	}

	// 检查项目访问权限
	if err := s.checkProjectAccess(ctx, column.Board.ProjectID, userID, tenantID); err != nil {
		return err
	}

	// 软删除
	if err := s.db.WithContext(ctx).Model(&column).Update("deleted_at", time.Now()).Error; err != nil {
		return fmt.Errorf("failed to delete board column: %w", err)
	}

	return nil
}

func (s *agileServiceImpl) ReorderBoardColumns(ctx context.Context, req *ReorderColumnsRequest, userID, tenantID uuid.UUID) error {
	// 验证看板访问权限
	var board models.Board
	if err := s.db.WithContext(ctx).First(&board, "id = ? AND deleted_at IS NULL", req.BoardID).Error; err != nil {
		return fmt.Errorf("board not found")
	}

	if err := s.checkProjectAccess(ctx, board.ProjectID, userID, tenantID); err != nil {
		return err
	}

	// 批量更新列的位置
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for i, columnID := range req.ColumnIDs {
		if err := tx.Model(&models.BoardColumn{}).
			Where("id = ? AND board_id = ? AND deleted_at IS NULL", columnID, req.BoardID).
			Update("position", i).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to update column position: %w", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit column reorder: %w", err)
	}

	return nil
}

// 看板任务移动实现

func (s *agileServiceImpl) MoveTask(ctx context.Context, req *TaskMoveRequest, userID, tenantID uuid.UUID) error {
	var task models.AgileTask

	if err := s.db.WithContext(ctx).First(&task, "id = ? AND deleted_at IS NULL", req.TaskID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("task not found")
		}
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 检查项目访问权限
	if err := s.checkProjectAccess(ctx, task.ProjectID, userID, tenantID); err != nil {
		return err
	}

	// 准备更新数据
	updates := make(map[string]interface{})

	// 如果移动到不同状态
	if req.TargetStatus != nil && task.Status != *req.TargetStatus {
		// 验证状态转换是否合法
		if !task.CanTransitionTo(*req.TargetStatus) {
			return fmt.Errorf("invalid status transition from %s to %s", task.Status, *req.TargetStatus)
		}
		updates["status"] = *req.TargetStatus
	}

	// 如果移动到不同Sprint
	if req.TargetSprintID != nil {
		updates["sprint_id"] = req.TargetSprintID
	}

	// 计算新的rank (简化实现)
	lexorankManager := NewLexorankManager()

	var prevRank, nextRank *string

	if req.PrevTaskID != nil {
		var prevTask models.AgileTask
		if err := s.db.WithContext(ctx).First(&prevTask, "id = ?", *req.PrevTaskID).Error; err == nil {
			prevRank = &prevTask.Rank
		}
	}

	if req.NextTaskID != nil {
		var nextTask models.AgileTask
		if err := s.db.WithContext(ctx).First(&nextTask, "id = ?", *req.NextTaskID).Error; err == nil {
			nextRank = &nextTask.Rank
		}
	}

	newRank, err := lexorankManager.CalculateRankForPosition(prevRank, nextRank)
	if err != nil {
		s.logger.Warn("计算新rank失败，使用默认方案", zap.Error(err))
		newRank = fmt.Sprintf("move_%d", time.Now().Unix())
	}

	updates["rank"] = newRank

	// 执行更新
	if err := s.db.WithContext(ctx).Model(&task).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to move task: %w", err)
	}

	return nil
}

func (s *agileServiceImpl) BatchMoveTasks(ctx context.Context, req *BatchTaskMoveRequest, userID, tenantID uuid.UUID) error {
	// 验证所有任务的访问权限
	var tasks []models.AgileTask
	if err := s.db.WithContext(ctx).
		Where("id IN ? AND deleted_at IS NULL", req.TaskIDs).
		Find(&tasks).Error; err != nil {
		return fmt.Errorf("failed to get tasks: %w", err)
	}

	if len(tasks) != len(req.TaskIDs) {
		return fmt.Errorf("some tasks not found")
	}

	// 检查所有任务的项目访问权限
	for _, task := range tasks {
		if err := s.checkProjectAccess(ctx, task.ProjectID, userID, tenantID); err != nil {
			return fmt.Errorf("no access to task %s: %w", task.ID, err)
		}
	}

	// 获取目标列信息以确定新状态
	var targetColumn models.BoardColumn
	if err := s.db.WithContext(ctx).First(&targetColumn, "id = ? AND deleted_at IS NULL", req.TargetColumnID).Error; err != nil {
		return fmt.Errorf("target column not found")
	}

	// 批量移动任务
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	basePosition := 0
	if req.NewPosition != nil {
		basePosition = *req.NewPosition
	}

	for i, taskID := range req.TaskIDs {
		updates := map[string]interface{}{
			"status": targetColumn.Status,
			"rank":   fmt.Sprintf("batch_move_%d_%d", time.Now().Unix(), basePosition+i),
		}

		if err := tx.Model(&models.AgileTask{}).
			Where("id = ?", taskID).
			Updates(updates).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to batch move task: %w", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit batch move: %w", err)
	}

	return nil
}

// 看板统计实现

func (s *agileServiceImpl) GetBoardStatistics(ctx context.Context, boardID uuid.UUID, userID, tenantID uuid.UUID) (*BoardStatistics, error) {
	// 验证看板访问权限
	var board models.Board
	if err := s.db.WithContext(ctx).First(&board, "id = ? AND deleted_at IS NULL", boardID).Error; err != nil {
		return nil, fmt.Errorf("board not found")
	}

	if err := s.checkProjectAccess(ctx, board.ProjectID, userID, tenantID); err != nil {
		return nil, err
	}

	// 获取看板的所有列
	var columns []models.BoardColumn
	if err := s.db.WithContext(ctx).
		Where("board_id = ? AND deleted_at IS NULL", boardID).
		Order("position ASC").
		Find(&columns).Error; err != nil {
		return nil, fmt.Errorf("failed to get board columns: %w", err)
	}

	stats := &BoardStatistics{
		BoardID:     boardID,
		ColumnStats: make([]ColumnStatistics, 0, len(columns)),
	}

	// 统计每个列的任务数量
	for _, column := range columns {
		var taskCount int64
		if err := s.db.WithContext(ctx).Model(&models.AgileTask{}).
			Joins("INNER JOIN board_columns bc ON agile_tasks.status = bc.status").
			Where("bc.id = ? AND agile_tasks.deleted_at IS NULL", column.ID).
			Count(&taskCount).Error; err != nil {
			s.logger.Warn("统计列任务数量失败", zap.Error(err))
			continue
		}

		isOverLimit := false
		if column.WIPLimit != nil && taskCount > int64(*column.WIPLimit) {
			isOverLimit = true
		}

		columnStat := ColumnStatistics{
			ColumnID:    column.ID,
			ColumnName:  column.Name,
			TaskCount:   taskCount,
			WIPLimit:    column.WIPLimit,
			IsOverLimit: isOverLimit,
		}

		stats.ColumnStats = append(stats.ColumnStats, columnStat)
		stats.TotalTasks += taskCount

		// 根据状态分类任务
		switch column.Status {
		case "done", "completed":
			stats.CompletedTasks += taskCount
		case "in_progress", "in_review":
			stats.InProgressTasks += taskCount
		case "todo", "backlog":
			stats.PendingTasks += taskCount
		}
	}

	return stats, nil
}

// 工作日志管理实现

func (s *agileServiceImpl) LogWork(ctx context.Context, req *LogWorkRequest, userID, tenantID uuid.UUID) (*models.WorkLog, error) {
	// 验证任务访问权限
	var task models.AgileTask
	if err := s.db.WithContext(ctx).First(&task, "id = ? AND deleted_at IS NULL", req.TaskID).Error; err != nil {
		return nil, fmt.Errorf("task not found")
	}

	if err := s.checkProjectAccess(ctx, task.ProjectID, userID, tenantID); err != nil {
		return nil, err
	}

	workLog := &models.WorkLog{
		TaskID:      req.TaskID,
		UserID:      userID,
		TimeSpent:   req.TimeSpent,
		Description: req.Description,
		WorkDate:    req.WorkDate,
	}

	if err := s.db.WithContext(ctx).Create(workLog).Error; err != nil {
		s.logger.Error("记录工作日志失败", zap.Error(err))
		return nil, fmt.Errorf("failed to log work: %w", err)
	}

	// 加载关联数据
	if err := s.db.WithContext(ctx).Preload("User").First(workLog, workLog.ID).Error; err != nil {
		s.logger.Warn("加载工作日志关联数据失败", zap.Error(err))
	}

	return workLog, nil
}

func (s *agileServiceImpl) GetWorkLogs(ctx context.Context, taskID uuid.UUID, userID, tenantID uuid.UUID) ([]models.WorkLog, error) {
	// 验证任务访问权限
	var task models.AgileTask
	if err := s.db.WithContext(ctx).First(&task, "id = ? AND deleted_at IS NULL", taskID).Error; err != nil {
		return nil, fmt.Errorf("task not found")
	}

	if err := s.checkProjectAccess(ctx, task.ProjectID, userID, tenantID); err != nil {
		return nil, err
	}

	var workLogs []models.WorkLog
	if err := s.db.WithContext(ctx).
		Preload("User").
		Where("task_id = ? AND deleted_at IS NULL", taskID).
		Order("work_date DESC, created_at DESC").
		Find(&workLogs).Error; err != nil {
		return nil, fmt.Errorf("failed to get work logs: %w", err)
	}

	return workLogs, nil
}

func (s *agileServiceImpl) DeleteWorkLog(ctx context.Context, workLogID uuid.UUID, userID, tenantID uuid.UUID) error {
	var workLog models.WorkLog

	if err := s.db.WithContext(ctx).Preload("Task").First(&workLog, "id = ? AND deleted_at IS NULL", workLogID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("work log not found")
		}
		return fmt.Errorf("failed to get work log: %w", err)
	}

	// 检查项目访问权限
	if err := s.checkProjectAccess(ctx, workLog.Task.ProjectID, userID, tenantID); err != nil {
		return err
	}

	// 只有创建者或项目管理员能删除工作日志
	if workLog.UserID != userID {
		// TODO: 检查是否为项目管理员
		return fmt.Errorf("permission denied")
	}

	// 软删除
	if err := s.db.WithContext(ctx).Model(&workLog).Update("deleted_at", time.Now()).Error; err != nil {
		return fmt.Errorf("failed to delete work log: %w", err)
	}

	return nil
}

// 任务评论管理实现

func (s *agileServiceImpl) AddComment(ctx context.Context, req *AddCommentRequest, userID, tenantID uuid.UUID) (*models.TaskComment, error) {
	// 验证任务访问权限
	var task models.AgileTask
	if err := s.db.WithContext(ctx).First(&task, "id = ? AND deleted_at IS NULL", req.TaskID).Error; err != nil {
		return nil, fmt.Errorf("task not found")
	}

	if err := s.checkProjectAccess(ctx, task.ProjectID, userID, tenantID); err != nil {
		return nil, err
	}

	comment := &models.TaskComment{
		TaskID:     req.TaskID,
		AuthorID:   userID,
		Content:    req.Content,
		IsInternal: req.IsInternal,
	}

	if err := s.db.WithContext(ctx).Create(comment).Error; err != nil {
		s.logger.Error("添加任务评论失败", zap.Error(err))
		return nil, fmt.Errorf("failed to add comment: %w", err)
	}

	// 加载关联数据
	if err := s.db.WithContext(ctx).Preload("Author").First(comment, comment.ID).Error; err != nil {
		s.logger.Warn("加载评论关联数据失败", zap.Error(err))
	}

	return comment, nil
}

func (s *agileServiceImpl) GetComments(ctx context.Context, taskID uuid.UUID, userID, tenantID uuid.UUID) ([]models.TaskComment, error) {
	// 验证任务访问权限
	var task models.AgileTask
	if err := s.db.WithContext(ctx).First(&task, "id = ? AND deleted_at IS NULL", taskID).Error; err != nil {
		return nil, fmt.Errorf("task not found")
	}

	if err := s.checkProjectAccess(ctx, task.ProjectID, userID, tenantID); err != nil {
		return nil, err
	}

	var comments []models.TaskComment
	if err := s.db.WithContext(ctx).
		Preload("Author").
		Where("task_id = ? AND deleted_at IS NULL", taskID).
		Order("created_at ASC").
		Find(&comments).Error; err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}

	return comments, nil
}

func (s *agileServiceImpl) UpdateComment(ctx context.Context, commentID uuid.UUID, content string, userID, tenantID uuid.UUID) (*models.TaskComment, error) {
	var comment models.TaskComment

	if err := s.db.WithContext(ctx).Preload("Task").First(&comment, "id = ? AND deleted_at IS NULL", commentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("comment not found")
		}
		return nil, fmt.Errorf("failed to get comment: %w", err)
	}

	// 检查项目访问权限
	if err := s.checkProjectAccess(ctx, comment.Task.ProjectID, userID, tenantID); err != nil {
		return nil, err
	}

	// 只有作者能编辑评论
	if comment.AuthorID != userID {
		return nil, fmt.Errorf("permission denied")
	}

	// 更新评论内容
	if err := s.db.WithContext(ctx).Model(&comment).Update("content", content).Error; err != nil {
		return nil, fmt.Errorf("failed to update comment: %w", err)
	}

	// 重新加载关联数据
	if err := s.db.WithContext(ctx).Preload("Author").First(&comment, comment.ID).Error; err != nil {
		s.logger.Warn("加载评论关联数据失败", zap.Error(err))
	}

	return &comment, nil
}

func (s *agileServiceImpl) DeleteComment(ctx context.Context, commentID uuid.UUID, userID, tenantID uuid.UUID) error {
	var comment models.TaskComment

	if err := s.db.WithContext(ctx).Preload("Task").First(&comment, "id = ? AND deleted_at IS NULL", commentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("comment not found")
		}
		return fmt.Errorf("failed to get comment: %w", err)
	}

	// 检查项目访问权限
	if err := s.checkProjectAccess(ctx, comment.Task.ProjectID, userID, tenantID); err != nil {
		return err
	}

	// 只有作者或项目管理员能删除评论
	if comment.AuthorID != userID {
		// TODO: 检查是否为项目管理员
		return fmt.Errorf("permission denied")
	}

	// 软删除
	if err := s.db.WithContext(ctx).Model(&comment).Update("deleted_at", time.Now()).Error; err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	return nil
}

// 报表和统计实现

func (s *agileServiceImpl) GetSprintBurndown(ctx context.Context, sprintID uuid.UUID, userID, tenantID uuid.UUID) (*BurndownData, error) {
	// 获取Sprint信息
	var sprint models.Sprint
	if err := s.db.WithContext(ctx).First(&sprint, "id = ? AND deleted_at IS NULL", sprintID).Error; err != nil {
		return nil, fmt.Errorf("sprint not found")
	}

	if err := s.checkProjectAccess(ctx, sprint.ProjectID, userID, tenantID); err != nil {
		return nil, err
	}

	// 获取Sprint中的所有任务
	var tasks []models.AgileTask
	if err := s.db.WithContext(ctx).
		Where("sprint_id = ? AND deleted_at IS NULL", sprintID).
		Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to get sprint tasks: %w", err)
	}

	// 计算总故事点数
	totalStoryPoints := 0
	for _, task := range tasks {
		if task.StoryPoints != nil {
			totalStoryPoints += *task.StoryPoints
		}
	}

	// 生成燃尽图数据点（简化实现，实际应该基于历史数据）
	dataPoints := make([]BurndownDataPoint, 0)

	// 计算Sprint的工作日数
	workDays := calculateWorkDays(sprint.StartDate, sprint.EndDate)

	// 生成理想燃尽线
	for i := 0; i <= workDays; i++ {
		currentDate := sprint.StartDate.AddDate(0, 0, i)
		if currentDate.After(sprint.EndDate) {
			break
		}

		// 理想剩余点数（线性递减）
		idealRemaining := totalStoryPoints - (totalStoryPoints*i)/workDays

		// 实际剩余点数（简化实现，应该基于历史完成数据）
		actualRemaining := totalStoryPoints // 这里应该查询历史数据

		dataPoint := BurndownDataPoint{
			Date:                 currentDate,
			RemainingStoryPoints: actualRemaining,
			IdealRemainingPoints: idealRemaining,
		}

		dataPoints = append(dataPoints, dataPoint)
	}

	burndownData := &BurndownData{
		SprintID:   sprintID,
		SprintName: sprint.Name,
		StartDate:  sprint.StartDate,
		EndDate:    sprint.EndDate,
		DataPoints: dataPoints,
	}

	return burndownData, nil
}

func (s *agileServiceImpl) GetVelocityChart(ctx context.Context, projectID uuid.UUID, userID, tenantID uuid.UUID) (*VelocityData, error) {
	if err := s.checkProjectAccess(ctx, projectID, userID, tenantID); err != nil {
		return nil, err
	}

	// 获取项目的已完成Sprint
	var sprints []models.Sprint
	if err := s.db.WithContext(ctx).
		Where("project_id = ? AND status = 'closed' AND deleted_at IS NULL", projectID).
		Order("end_date DESC").
		Limit(10). // 最近10个Sprint
		Find(&sprints).Error; err != nil {
		return nil, fmt.Errorf("failed to get completed sprints: %w", err)
	}

	velocityData := &VelocityData{
		ProjectID: projectID,
		Sprints:   make([]VelocityDataPoint, 0),
	}

	totalVelocity := 0.0

	for _, sprint := range sprints {
		// 统计Sprint中的任务
		var completedPoints, committedPoints int
		var completedTasks, totalTasks int64

		// 已完成的故事点数
		if err := s.db.WithContext(ctx).
			Model(&models.AgileTask{}).
			Where("sprint_id = ? AND status = 'done' AND deleted_at IS NULL", sprint.ID).
			Select("COALESCE(SUM(story_points), 0)").
			Scan(&completedPoints).Error; err != nil {
			s.logger.Warn("统计已完成故事点失败", zap.Error(err))
		}

		// 承诺的故事点数（所有任务）
		if err := s.db.WithContext(ctx).
			Model(&models.AgileTask{}).
			Where("sprint_id = ? AND deleted_at IS NULL", sprint.ID).
			Select("COALESCE(SUM(story_points), 0)").
			Scan(&committedPoints).Error; err != nil {
			s.logger.Warn("统计承诺故事点失败", zap.Error(err))
		}

		// 已完成任务数
		if err := s.db.WithContext(ctx).
			Model(&models.AgileTask{}).
			Where("sprint_id = ? AND status = 'done' AND deleted_at IS NULL", sprint.ID).
			Count(&completedTasks).Error; err != nil {
			s.logger.Warn("统计已完成任务数失败", zap.Error(err))
		}

		// 总任务数
		if err := s.db.WithContext(ctx).
			Model(&models.AgileTask{}).
			Where("sprint_id = ? AND deleted_at IS NULL", sprint.ID).
			Count(&totalTasks).Error; err != nil {
			s.logger.Warn("统计总任务数失败", zap.Error(err))
		}

		velocityPoint := VelocityDataPoint{
			SprintID:        sprint.ID,
			SprintName:      sprint.Name,
			CompletedPoints: completedPoints,
			CommittedPoints: committedPoints,
			CompletedTasks:  int(completedTasks),
			TotalTasks:      int(totalTasks),
		}

		velocityData.Sprints = append(velocityData.Sprints, velocityPoint)
		totalVelocity += float64(completedPoints)
	}

	// 计算平均速度
	if len(sprints) > 0 {
		velocityData.Average = totalVelocity / float64(len(sprints))
	}

	return velocityData, nil
}

func (s *agileServiceImpl) GetTaskStatistics(ctx context.Context, projectID uuid.UUID, userID, tenantID uuid.UUID) (*TaskStatistics, error) {
	if err := s.checkProjectAccess(ctx, projectID, userID, tenantID); err != nil {
		return nil, err
	}

	statistics := &TaskStatistics{
		ProjectID:       projectID,
		TasksByStatus:   make(map[string]int64),
		TasksByType:     make(map[string]int64),
		TasksByPriority: make(map[string]int64),
		TasksByAssignee: make([]UserTaskStatistics, 0),
		RecentActivity:  make([]TaskActivity, 0),
	}

	// 获取基本统计
	baseQuery := s.db.WithContext(ctx).Model(&models.AgileTask{}).
		Where("project_id = ? AND deleted_at IS NULL", projectID)

	// 总任务数
	if err := baseQuery.Count(&statistics.TotalTasks).Error; err != nil {
		return nil, fmt.Errorf("failed to count total tasks: %w", err)
	}

	// 已完成任务数
	if err := baseQuery.Where("status = 'done'").Count(&statistics.CompletedTasks).Error; err != nil {
		return nil, fmt.Errorf("failed to count completed tasks: %w", err)
	}

	// 进行中任务数
	if err := baseQuery.Where("status IN ('in_progress', 'in_review')").Count(&statistics.InProgressTasks).Error; err != nil {
		return nil, fmt.Errorf("failed to count in-progress tasks: %w", err)
	}

	// 总故事点数
	var totalStoryPoints, completedStoryPoints int64
	if err := s.db.WithContext(ctx).
		Model(&models.AgileTask{}).
		Where("project_id = ? AND deleted_at IS NULL", projectID).
		Select("COALESCE(SUM(story_points), 0)").
		Scan(&totalStoryPoints).Error; err != nil {
		s.logger.Warn("统计总故事点失败", zap.Error(err))
	}
	statistics.TotalStoryPoints = totalStoryPoints

	if err := s.db.WithContext(ctx).
		Model(&models.AgileTask{}).
		Where("project_id = ? AND status = 'done' AND deleted_at IS NULL", projectID).
		Select("COALESCE(SUM(story_points), 0)").
		Scan(&completedStoryPoints).Error; err != nil {
		s.logger.Warn("统计已完成故事点失败", zap.Error(err))
	}
	statistics.CompletedStoryPoints = completedStoryPoints

	// 按状态统计
	var statusStats []struct {
		Status string
		Count  int64
	}
	if err := s.db.WithContext(ctx).
		Model(&models.AgileTask{}).
		Where("project_id = ? AND deleted_at IS NULL", projectID).
		Group("status").
		Select("status, COUNT(*) as count").
		Scan(&statusStats).Error; err != nil {
		s.logger.Warn("按状态统计任务失败", zap.Error(err))
	} else {
		for _, stat := range statusStats {
			statistics.TasksByStatus[stat.Status] = stat.Count
		}
	}

	// 按类型统计
	var typeStats []struct {
		Type  string
		Count int64
	}
	if err := s.db.WithContext(ctx).
		Model(&models.AgileTask{}).
		Where("project_id = ? AND deleted_at IS NULL", projectID).
		Group("type").
		Select("type, COUNT(*) as count").
		Scan(&typeStats).Error; err != nil {
		s.logger.Warn("按类型统计任务失败", zap.Error(err))
	} else {
		for _, stat := range typeStats {
			statistics.TasksByType[stat.Type] = stat.Count
		}
	}

	// 按优先级统计
	var priorityStats []struct {
		Priority string
		Count    int64
	}
	if err := s.db.WithContext(ctx).
		Model(&models.AgileTask{}).
		Where("project_id = ? AND deleted_at IS NULL", projectID).
		Group("priority").
		Select("priority, COUNT(*) as count").
		Scan(&priorityStats).Error; err != nil {
		s.logger.Warn("按优先级统计任务失败", zap.Error(err))
	} else {
		for _, stat := range priorityStats {
			statistics.TasksByPriority[stat.Priority] = stat.Count
		}
	}

	return statistics, nil
}

// 辅助函数

// calculateWorkDays 计算两个日期之间的工作日数（不包含周末）
func calculateWorkDays(startDate, endDate time.Time) int {
	days := 0
	for d := startDate; d.Before(endDate) || d.Equal(endDate); d = d.AddDate(0, 0, 1) {
		weekday := d.Weekday()
		if weekday != time.Saturday && weekday != time.Sunday {
			days++
		}
	}
	return days
}
