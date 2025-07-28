package service

import (
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/models"
	"github.com/google/uuid"
)

// Sprint相关DTO

// CreateSprintRequest 创建Sprint请求
type CreateSprintRequest struct {
	ProjectID   uuid.UUID `json:"project_id" binding:"required"`
	Name        string    `json:"name" binding:"required,min=1,max=255"`
	Description *string   `json:"description,omitempty"`
	Goal        *string   `json:"goal,omitempty"`
	StartDate   time.Time `json:"start_date" binding:"required"`
	EndDate     time.Time `json:"end_date" binding:"required"`
	Capacity    int       `json:"capacity" binding:"min=0"`
}

// UpdateSprintRequest 更新Sprint请求
type UpdateSprintRequest struct {
	Name        *string    `json:"name,omitempty" binding:"omitempty,min=1,max=255"`
	Description *string    `json:"description,omitempty"`
	Goal        *string    `json:"goal,omitempty"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	Capacity    *int       `json:"capacity,omitempty" binding:"omitempty,min=0"`
}

// SprintListResponse Sprint列表响应
type SprintListResponse struct {
	Sprints  []models.Sprint `json:"sprints"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}

// 敏捷任务相关DTO

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	ProjectID          uuid.UUID                   `json:"project_id" binding:"required"`
	SprintID           *uuid.UUID                  `json:"sprint_id,omitempty"`
	EpicID             *uuid.UUID                  `json:"epic_id,omitempty"`
	ParentID           *uuid.UUID                  `json:"parent_id,omitempty"`
	Title              string                      `json:"title" binding:"required,min=1,max=500"`
	Description        *string                     `json:"description,omitempty"`
	Type               string                      `json:"type" binding:"required,oneof=story task bug epic subtask"`
	Priority           string                      `json:"priority" binding:"required,oneof=lowest low medium high highest"`
	StoryPoints        *int                        `json:"story_points,omitempty" binding:"omitempty,min=1,max=100"`
	OriginalEstimate   *float64                    `json:"original_estimate,omitempty" binding:"omitempty,min=0"`
	AssigneeID         *uuid.UUID                  `json:"assignee_id,omitempty"`
	Labels             []string                    `json:"labels,omitempty"`
	Components         []string                    `json:"components,omitempty"`
	AcceptanceCriteria []models.AcceptanceCriteria `json:"acceptance_criteria,omitempty"`
}

// UpdateTaskRequest 更新任务请求
type UpdateTaskRequest struct {
	Title              *string                     `json:"title,omitempty" binding:"omitempty,min=1,max=500"`
	Description        *string                     `json:"description,omitempty"`
	Status             *string                     `json:"status,omitempty" binding:"omitempty,oneof=todo in_progress in_review testing done cancelled"`
	Priority           *string                     `json:"priority,omitempty" binding:"omitempty,oneof=lowest low medium high highest"`
	StoryPoints        *int                        `json:"story_points,omitempty" binding:"omitempty,min=1,max=100"`
	OriginalEstimate   *float64                    `json:"original_estimate,omitempty" binding:"omitempty,min=0"`
	RemainingTime      *float64                    `json:"remaining_time,omitempty" binding:"omitempty,min=0"`
	AssigneeID         *uuid.UUID                  `json:"assignee_id,omitempty"`
	SprintID           *uuid.UUID                  `json:"sprint_id,omitempty"`
	EpicID             *uuid.UUID                  `json:"epic_id,omitempty"`
	Labels             []string                    `json:"labels,omitempty"`
	Components         []string                    `json:"components,omitempty"`
	AcceptanceCriteria []models.AcceptanceCriteria `json:"acceptance_criteria,omitempty"`
}

// TaskFilter 任务过滤器
type TaskFilter struct {
	ProjectID  uuid.UUID  `json:"project_id" binding:"required"`
	SprintID   *uuid.UUID `json:"sprint_id,omitempty"`
	EpicID     *uuid.UUID `json:"epic_id,omitempty"`
	AssigneeID *uuid.UUID `json:"assignee_id,omitempty"`
	Status     []string   `json:"status,omitempty"`
	Type       []string   `json:"type,omitempty"`
	Priority   []string   `json:"priority,omitempty"`
	SearchText *string    `json:"search_text,omitempty"`
}

// TaskListResponse 任务列表响应
type TaskListResponse struct {
	Tasks    []models.AgileTask `json:"tasks"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

// ReorderTasksRequest 任务重排序请求
type ReorderTasksRequest struct {
	ProjectID uuid.UUID        `json:"project_id" binding:"required"`
	Tasks     []TaskRankUpdate `json:"tasks" binding:"required,min=1"`
}

// TaskRankUpdate 任务排序更新
type TaskRankUpdate struct {
	TaskID uuid.UUID `json:"task_id" binding:"required"`
	Rank   string    `json:"rank" binding:"required"`
}

// 史诗相关DTO

// CreateEpicRequest 创建史诗请求
type CreateEpicRequest struct {
	ProjectID       uuid.UUID  `json:"project_id" binding:"required"`
	Name            string     `json:"name" binding:"required,min=1,max=255"`
	Description     *string    `json:"description,omitempty"`
	Color           *string    `json:"color,omitempty" binding:"omitempty,len=7"`
	StartDate       *time.Time `json:"start_date,omitempty"`
	EndDate         *time.Time `json:"end_date,omitempty"`
	Goal            *string    `json:"goal,omitempty"`
	SuccessCriteria *string    `json:"success_criteria,omitempty"`
}

// UpdateEpicRequest 更新史诗请求
type UpdateEpicRequest struct {
	Name            *string    `json:"name,omitempty" binding:"omitempty,min=1,max=255"`
	Description     *string    `json:"description,omitempty"`
	Status          *string    `json:"status,omitempty" binding:"omitempty,oneof=open in_progress done cancelled"`
	Color           *string    `json:"color,omitempty" binding:"omitempty,len=7"`
	StartDate       *time.Time `json:"start_date,omitempty"`
	EndDate         *time.Time `json:"end_date,omitempty"`
	Goal            *string    `json:"goal,omitempty"`
	SuccessCriteria *string    `json:"success_criteria,omitempty"`
}

// EpicListResponse 史诗列表响应
type EpicListResponse struct {
	Epics    []models.Epic `json:"epics"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

// 看板相关DTO

// CreateBoardRequest 创建看板请求
type CreateBoardRequest struct {
	ProjectID   uuid.UUID `json:"project_id" binding:"required"`
	Name        string    `json:"name" binding:"required,min=1,max=255"`
	Description *string   `json:"description,omitempty"`
	Type        string    `json:"type" binding:"required,oneof=kanban scrum"`
}

// UpdateBoardRequest 更新看板请求
type UpdateBoardRequest struct {
	Name        *string `json:"name,omitempty" binding:"omitempty,min=1,max=255"`
	Description *string `json:"description,omitempty"`
}

// CreateBoardColumnRequest 创建看板列请求
type CreateBoardColumnRequest struct {
	BoardID  uuid.UUID `json:"board_id" binding:"required"`
	Name     string    `json:"name" binding:"required,min=1,max=100"`
	Position int       `json:"position" binding:"required,min=0"`
	WIPLimit *int      `json:"wip_limit,omitempty" binding:"omitempty,min=1"`
	Status   string    `json:"status" binding:"required"`
	Color    *string   `json:"color,omitempty" binding:"omitempty,len=7"`
}

// UpdateBoardColumnRequest 更新看板列请求
type UpdateBoardColumnRequest struct {
	Name     *string `json:"name,omitempty" binding:"omitempty,min=1,max=100"`
	Position *int    `json:"position,omitempty" binding:"omitempty,min=0"`
	WIPLimit *int    `json:"wip_limit,omitempty" binding:"omitempty,min=1"`
	Color    *string `json:"color,omitempty" binding:"omitempty,len=7"`
}

// 工作日志相关DTO

// LogWorkRequest 记录工作日志请求
type LogWorkRequest struct {
	TaskID      uuid.UUID `json:"task_id" binding:"required"`
	TimeSpent   float64   `json:"time_spent" binding:"required,min=0.1,max=24"`
	Description *string   `json:"description,omitempty"`
	WorkDate    time.Time `json:"work_date" binding:"required"`
}

// 任务评论相关DTO

// AddCommentRequest 添加评论请求
type AddCommentRequest struct {
	TaskID     uuid.UUID `json:"task_id" binding:"required"`
	Content    string    `json:"content" binding:"required,min=1"`
	IsInternal bool      `json:"is_internal,omitempty"`
}

// 报表和统计相关DTO

// BurndownData 燃尽图数据
type BurndownData struct {
	SprintID   uuid.UUID           `json:"sprint_id"`
	SprintName string              `json:"sprint_name"`
	StartDate  time.Time           `json:"start_date"`
	EndDate    time.Time           `json:"end_date"`
	DataPoints []BurndownDataPoint `json:"data_points"`
}

// BurndownDataPoint 燃尽图数据点
type BurndownDataPoint struct {
	Date                 time.Time `json:"date"`
	RemainingStoryPoints int       `json:"remaining_story_points"`
	IdealRemainingPoints int       `json:"ideal_remaining_points"`
	CompletedStoryPoints int       `json:"completed_story_points,omitempty"`
}

// VelocityData 速度图数据
type VelocityData struct {
	ProjectID uuid.UUID           `json:"project_id"`
	Sprints   []VelocityDataPoint `json:"sprints"`
	Average   float64             `json:"average_velocity"`
}

// VelocityDataPoint 速度图数据点
type VelocityDataPoint struct {
	SprintID        uuid.UUID `json:"sprint_id"`
	SprintName      string    `json:"sprint_name"`
	CompletedPoints int       `json:"completed_points"`
	CommittedPoints int       `json:"committed_points"`
	CompletedTasks  int       `json:"completed_tasks"`
	TotalTasks      int       `json:"total_tasks"`
}

// TaskStatistics 任务统计
type TaskStatistics struct {
	ProjectID            uuid.UUID            `json:"project_id"`
	TotalTasks           int64                `json:"total_tasks"`
	CompletedTasks       int64                `json:"completed_tasks"`
	InProgressTasks      int64                `json:"in_progress_tasks"`
	TotalStoryPoints     int64                `json:"total_story_points"`
	CompletedStoryPoints int64                `json:"completed_story_points"`
	TasksByStatus        map[string]int64     `json:"tasks_by_status"`
	TasksByType          map[string]int64     `json:"tasks_by_type"`
	TasksByPriority      map[string]int64     `json:"tasks_by_priority"`
	TasksByAssignee      []UserTaskStatistics `json:"tasks_by_assignee"`
	RecentActivity       []TaskActivity       `json:"recent_activity"`
}

// UserTaskStatistics 用户任务统计
type UserTaskStatistics struct {
	UserID               uuid.UUID `json:"user_id"`
	UserName             string    `json:"user_name"`
	TotalTasks           int64     `json:"total_tasks"`
	CompletedTasks       int64     `json:"completed_tasks"`
	InProgressTasks      int64     `json:"in_progress_tasks"`
	TotalStoryPoints     int64     `json:"total_story_points"`
	CompletedStoryPoints int64     `json:"completed_story_points"`
	TotalLoggedTime      float64   `json:"total_logged_time"`
}

// TaskActivity 任务活动
type TaskActivity struct {
	TaskID      uuid.UUID `json:"task_id"`
	TaskTitle   string    `json:"task_title"`
	UserID      uuid.UUID `json:"user_id"`
	UserName    string    `json:"user_name"`
	Action      string    `json:"action"` // created, updated, completed, commented
	Timestamp   time.Time `json:"timestamp"`
	Description string    `json:"description"`
}

// 看板相关响应DTO

// BoardWithTasks 包含任务的看板
type BoardWithTasks struct {
	models.Board
	Columns []BoardColumnWithTasks `json:"columns"`
}

// BoardColumnWithTasks 包含任务的看板列
type BoardColumnWithTasks struct {
	models.BoardColumn
	Tasks []models.AgileTask `json:"tasks"`
}

// 任务拖拽相关DTO

// MoveTaskRequest 移动任务请求
type MoveTaskRequest struct {
	TaskID         uuid.UUID `json:"task_id" binding:"required"`
	TargetColumnID uuid.UUID `json:"target_column_id" binding:"required"`
	Position       int       `json:"position" binding:"required,min=0"`
}

// 批量操作DTO

// BatchUpdateTasksRequest 批量更新任务请求
type BatchUpdateTasksRequest struct {
	TaskIDs []uuid.UUID       `json:"task_ids" binding:"required,min=1"`
	Updates UpdateTaskRequest `json:"updates"`
}

// AssignTasksRequest 批量分配任务请求
type AssignTasksRequest struct {
	TaskIDs    []uuid.UUID `json:"task_ids" binding:"required,min=1"`
	AssigneeID *uuid.UUID  `json:"assignee_id"`
}

// AddTasksToSprintRequest 批量添加任务到Sprint请求
type AddTasksToSprintRequest struct {
	TaskIDs  []uuid.UUID `json:"task_ids" binding:"required,min=1"`
	SprintID uuid.UUID   `json:"sprint_id" binding:"required"`
}

// 工作流相关DTO

// WorkflowTransition 工作流转换
type WorkflowTransition struct {
	FromStatus string              `json:"from_status"`
	ToStatus   string              `json:"to_status"`
	Name       string              `json:"name"`
	Conditions []WorkflowCondition `json:"conditions,omitempty"`
}

// WorkflowCondition 工作流条件
type WorkflowCondition struct {
	Type  string      `json:"type"` // assignee_required, approval_required, etc.
	Value interface{} `json:"value"`
}

// 时间追踪相关DTO

// TimeTrackingSummary 时间追踪摘要
type TimeTrackingSummary struct {
	TaskID           uuid.UUID `json:"task_id"`
	OriginalEstimate float64   `json:"original_estimate"`
	RemainingTime    float64   `json:"remaining_time"`
	LoggedTime       float64   `json:"logged_time"`
	Progress         float64   `json:"progress"` // 百分比
}

// 验收标准相关DTO

// UpdateAcceptanceCriteriaRequest 更新验收标准请求
type UpdateAcceptanceCriteriaRequest struct {
	TaskID   uuid.UUID                   `json:"task_id" binding:"required"`
	Criteria []models.AcceptanceCriteria `json:"criteria" binding:"required"`
}

// 子任务相关DTO

// CreateSubTaskRequest 创建子任务请求
type CreateSubTaskRequest struct {
	ParentID    uuid.UUID  `json:"parent_id" binding:"required"`
	Title       string     `json:"title" binding:"required,min=1,max=500"`
	Description *string    `json:"description,omitempty"`
	AssigneeID  *uuid.UUID `json:"assignee_id,omitempty"`
	Priority    string     `json:"priority" binding:"required,oneof=lowest low medium high highest"`
	Estimate    *float64   `json:"estimate,omitempty" binding:"omitempty,min=0"`
}

// 依赖关系相关DTO

// TaskDependency 任务依赖关系
type TaskDependency struct {
	ID          uuid.UUID `json:"id"`
	TaskID      uuid.UUID `json:"task_id"`
	DependsOnID uuid.UUID `json:"depends_on_id"`
	Type        string    `json:"type"` // blocks, relates_to
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   uuid.UUID `json:"created_by"`
}

// AddDependencyRequest 添加依赖请求
type AddDependencyRequest struct {
	TaskID      uuid.UUID `json:"task_id" binding:"required"`
	DependsOnID uuid.UUID `json:"depends_on_id" binding:"required"`
	Type        string    `json:"type" binding:"required,oneof=blocks relates_to"`
}

// TaskMoveRequest 精确任务移动请求
type TaskMoveRequest struct {
	TaskID         uuid.UUID  `json:"task_id" binding:"required"`
	PrevTaskID     *uuid.UUID `json:"prev_task_id,omitempty"`     // 前一个任务ID（如果为空则移到开头）
	NextTaskID     *uuid.UUID `json:"next_task_id,omitempty"`     // 后一个任务ID（如果为空则移到末尾）
	TargetStatus   *string    `json:"target_status,omitempty"`    // 目标状态（如果跨列移动）
	TargetSprintID *uuid.UUID `json:"target_sprint_id,omitempty"` // 目标Sprint（如果跨Sprint移动）
}

// BatchReorderRequest 批量重排序请求
type BatchReorderRequest struct {
	ProjectID uuid.UUID   `json:"project_id" binding:"required"`
	SprintID  *uuid.UUID  `json:"sprint_id,omitempty"`
	Status    *string     `json:"status,omitempty"`
	TaskIDs   []uuid.UUID `json:"task_ids" binding:"required,min=1"`
}

// 统计相关DTO

// ProjectStatistics 项目统计数据
type ProjectStatistics struct {
	ProjectID      uuid.UUID `json:"project_id"`
	TotalTasks     int64     `json:"total_tasks"`
	CompletedTasks int64     `json:"completed_tasks"`
	ActiveTasks    int64     `json:"active_tasks"`
	TotalSprints   int64     `json:"total_sprints"`
	ActiveSprints  int64     `json:"active_sprints"`
	CompletionRate float64   `json:"completion_rate"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// UserWorkload 用户工作负载
type UserWorkload struct {
	UserID              uuid.UUID      `json:"user_id"`
	TotalTasks          int            `json:"total_tasks"`
	TasksByStatus       map[string]int `json:"tasks_by_status"`
	TotalEstimatedHours int            `json:"total_estimated_hours"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

// 看板管理相关DTO

// BoardListResponse 看板列表响应
type BoardListResponse struct {
	Boards   []models.Board `json:"boards"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

// ReorderColumnsRequest 重新排序看板列请求
type ReorderColumnsRequest struct {
	BoardID   uuid.UUID   `json:"board_id" binding:"required"`
	ColumnIDs []uuid.UUID `json:"column_ids" binding:"required,min=1"`
}

// BatchTaskMoveRequest 批量移动任务请求
type BatchTaskMoveRequest struct {
	TaskIDs        []uuid.UUID `json:"task_ids" binding:"required,min=1"`
	TargetColumnID uuid.UUID   `json:"target_column_id" binding:"required"`
	NewPosition    *int        `json:"new_position,omitempty"`
}

// BoardStatistics 看板统计信息
type BoardStatistics struct {
	BoardID         uuid.UUID          `json:"board_id"`
	TotalTasks      int64              `json:"total_tasks"`
	CompletedTasks  int64              `json:"completed_tasks"`
	InProgressTasks int64              `json:"in_progress_tasks"`
	PendingTasks    int64              `json:"pending_tasks"`
	ColumnStats     []ColumnStatistics `json:"column_stats"`
}

// ColumnStatistics 看板列统计
type ColumnStatistics struct {
	ColumnID    uuid.UUID `json:"column_id"`
	ColumnName  string    `json:"column_name"`
	TaskCount   int64     `json:"task_count"`
	WIPLimit    *int      `json:"wip_limit,omitempty"`
	IsOverLimit bool      `json:"is_over_limit"`
}

// Sprint高级管理相关DTO

// ProjectMember 项目成员信息
type ProjectMember struct {
	UserID        uuid.UUID `json:"user_id"`
	Name          string    `json:"name"`
	Email         string    `json:"email"`
	AvgDailyHours float64   `json:"avg_daily_hours"`
}

// MemberCapacity 成员容量信息
type MemberCapacity struct {
	UserID         uuid.UUID `json:"user_id"`
	UserName       string    `json:"user_name"`
	AvgDailyHours  float64   `json:"avg_daily_hours"`
	TotalHours     float64   `json:"total_hours"`
	AllocatedHours float64   `json:"allocated_hours"`
	RemainingHours float64   `json:"remaining_hours"`
}

// SprintCapacityPlan Sprint容量规划
type SprintCapacityPlan struct {
	SprintID            uuid.UUID          `json:"sprint_id"`
	SprintName          string             `json:"sprint_name"`
	WorkDays            int                `json:"work_days"`
	TotalCapacityHours  float64            `json:"total_capacity_hours"`
	AllocatedHours      float64            `json:"allocated_hours"`
	RemainingHours      float64            `json:"remaining_hours"`
	CapacityUtilization float64            `json:"capacity_utilization"`
	MemberCapacities    []MemberCapacity   `json:"member_capacities"`
	Tasks               []models.AgileTask `json:"tasks"`
}

// TaskAssignmentSuggestion 任务分配建议
type TaskAssignmentSuggestion struct {
	TaskID          uuid.UUID `json:"task_id"`
	TaskTitle       string    `json:"task_title"`
	SuggestedUserID uuid.UUID `json:"suggested_user_id"`
	EstimateHours   float64   `json:"estimate_hours"`
	Reason          string    `json:"reason"`
	Confidence      float64   `json:"confidence"`
}

// TaskAllocationSuggestion 任务分配建议响应
type TaskAllocationSuggestion struct {
	SprintID               uuid.UUID                  `json:"sprint_id"`
	Suggestions            []TaskAssignmentSuggestion `json:"suggestions"`
	TotalTasks             int                        `json:"total_tasks"`
	AssignableTasks        int                        `json:"assignable_tasks"`
	UnassignableTasksCount int                        `json:"unassignable_tasks_count"`
}

// SprintProgress Sprint进度信息
type SprintProgress struct {
	SprintID             uuid.UUID `json:"sprint_id"`
	SprintName           string    `json:"sprint_name"`
	StartDate            time.Time `json:"start_date"`
	EndDate              time.Time `json:"end_date"`
	Status               string    `json:"status"`
	TotalTasks           int64     `json:"total_tasks"`
	CompletedTasks       int64     `json:"completed_tasks"`
	InProgressTasks      int64     `json:"in_progress_tasks"`
	TodoTasks            int64     `json:"todo_tasks"`
	TotalStoryPoints     int64     `json:"total_story_points"`
	CompletedStoryPoints int64     `json:"completed_story_points"`
	TimeProgress         float64   `json:"time_progress"`
	CompletionProgress   float64   `json:"completion_progress"`
	StoryPointProgress   float64   `json:"story_point_progress"`
	HealthStatus         string    `json:"health_status"`
	HealthScore          float64   `json:"health_score"`
}
