package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Sprint 迭代冲刺模型
type Sprint struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ProjectID   uuid.UUID `json:"project_id" gorm:"type:uuid;not null;index"`
	Name        string    `json:"name" gorm:"size:255;not null"`
	Description *string   `json:"description" gorm:"type:text"`
	Goal        *string   `json:"goal" gorm:"type:text"`
	Status      string    `json:"status" gorm:"size:20;not null;default:'planned'"`

	// 时间规划
	StartDate time.Time `json:"start_date" gorm:"not null"`
	EndDate   time.Time `json:"end_date" gorm:"not null"`

	// 容量规划
	Capacity int `json:"capacity" gorm:"default:0"` // 故事点容量

	// 审计字段
	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt *time.Time `json:"deleted_at" gorm:"index"`
	CreatedBy *uuid.UUID `json:"created_by" gorm:"type:uuid"`

	// 关联关系
	Project *Project    `json:"project,omitempty" gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
	Tasks   []AgileTask `json:"tasks,omitempty" gorm:"foreignKey:SprintID"`
	Creator *User       `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
}

// AgileTask 敏捷任务模型
type AgileTask struct {
	ID        uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ProjectID uuid.UUID  `json:"project_id" gorm:"type:uuid;not null;index"`
	SprintID  *uuid.UUID `json:"sprint_id" gorm:"type:uuid;index"`
	EpicID    *uuid.UUID `json:"epic_id" gorm:"type:uuid;index"`
	ParentID  *uuid.UUID `json:"parent_id" gorm:"type:uuid;index"`

	// 任务基本信息
	TaskNumber  int64   `json:"task_number" gorm:"not null;index"`
	Title       string  `json:"title" gorm:"size:500;not null"`
	Description *string `json:"description" gorm:"type:text"`
	Type        string  `json:"type" gorm:"size:50;not null;default:'story'"`
	Status      string  `json:"status" gorm:"size:50;not null;default:'todo'"`
	Priority    string  `json:"priority" gorm:"size:20;not null;default:'medium'"`

	// 敏捷估算
	StoryPoints      *int     `json:"story_points"`
	OriginalEstimate *float64 `json:"original_estimate"`            // 原始估算（小时）
	RemainingTime    *float64 `json:"remaining_time"`               // 剩余时间（小时）
	LoggedTime       float64  `json:"logged_time" gorm:"default:0"` // 已记录时间（小时）

	// 人员分配
	AssigneeID *uuid.UUID `json:"assignee_id" gorm:"type:uuid"`
	ReporterID uuid.UUID  `json:"reporter_id" gorm:"type:uuid;not null"`

	// 标签和分类
	Labels     []string `json:"labels" gorm:"type:jsonb"`
	Components []string `json:"components" gorm:"type:jsonb"`

	// 排序权重（用于看板拖拽排序）
	Rank string `json:"rank" gorm:"index"` // Lexorank算法排序字段

	// 业务字段
	AcceptanceCriteria []AcceptanceCriteria `json:"acceptance_criteria" gorm:"type:jsonb"`

	// 审计字段
	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt *time.Time `json:"deleted_at" gorm:"index"`

	// 关联关系
	Project     *Project         `json:"project,omitempty" gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
	Sprint      *Sprint          `json:"sprint,omitempty" gorm:"foreignKey:SprintID"`
	Epic        *Epic            `json:"epic,omitempty" gorm:"foreignKey:EpicID"`
	Parent      *AgileTask       `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
	Children    []AgileTask      `json:"children,omitempty" gorm:"foreignKey:ParentID"`
	Assignee    *User            `json:"assignee,omitempty" gorm:"foreignKey:AssigneeID"`
	Reporter    *User            `json:"reporter,omitempty" gorm:"foreignKey:ReporterID"`
	Comments    []TaskComment    `json:"comments,omitempty" gorm:"foreignKey:TaskID"`
	Attachments []TaskAttachment `json:"attachments,omitempty" gorm:"foreignKey:TaskID"`
	WorkLogs    []WorkLog        `json:"work_logs,omitempty" gorm:"foreignKey:TaskID"`
}

// Epic 史诗模型
type Epic struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ProjectID   uuid.UUID `json:"project_id" gorm:"type:uuid;not null;index"`
	Name        string    `json:"name" gorm:"size:255;not null"`
	Description *string   `json:"description" gorm:"type:text"`
	Status      string    `json:"status" gorm:"size:50;not null;default:'open'"`
	Color       *string   `json:"color" gorm:"size:7"` // 十六进制颜色代码

	// 时间计划
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`

	// 目标和指标
	Goal            *string `json:"goal" gorm:"type:text"`
	SuccessCriteria *string `json:"success_criteria" gorm:"type:text"`

	// 审计字段
	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt *time.Time `json:"deleted_at" gorm:"index"`
	CreatedBy *uuid.UUID `json:"created_by" gorm:"type:uuid"`

	// 关联关系
	Project *Project    `json:"project,omitempty" gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
	Tasks   []AgileTask `json:"tasks,omitempty" gorm:"foreignKey:EpicID"`
	Creator *User       `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
}

// AcceptanceCriteria 验收标准
type AcceptanceCriteria struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

// TaskComment 任务评论模型
type TaskComment struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TaskID     uuid.UUID `json:"task_id" gorm:"type:uuid;not null;index"`
	AuthorID   uuid.UUID `json:"author_id" gorm:"type:uuid;not null"`
	Content    string    `json:"content" gorm:"type:text;not null"`
	IsInternal bool      `json:"is_internal" gorm:"default:false"`

	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt *time.Time `json:"deleted_at" gorm:"index"`

	// 关联关系
	Task   *AgileTask `json:"task,omitempty" gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE"`
	Author *User      `json:"author,omitempty" gorm:"foreignKey:AuthorID"`
}

// TaskAttachment 任务附件模型
type TaskAttachment struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TaskID      uuid.UUID `json:"task_id" gorm:"type:uuid;not null;index"`
	FileName    string    `json:"file_name" gorm:"size:255;not null"`
	FileSize    int64     `json:"file_size" gorm:"not null"`
	ContentType string    `json:"content_type" gorm:"size:100;not null"`
	FilePath    string    `json:"file_path" gorm:"size:500;not null"`
	UploadedBy  uuid.UUID `json:"uploaded_by" gorm:"type:uuid;not null"`

	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
	DeletedAt *time.Time `json:"deleted_at" gorm:"index"`

	// 关联关系
	Task     *AgileTask `json:"task,omitempty" gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE"`
	Uploader *User      `json:"uploader,omitempty" gorm:"foreignKey:UploadedBy"`
}

// WorkLog 工作日志模型
type WorkLog struct {
	ID     uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TaskID uuid.UUID `json:"task_id" gorm:"type:uuid;not null;index"`
	UserID uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`

	TimeSpent   float64   `json:"time_spent" gorm:"not null"` // 工作时长（小时）
	Description *string   `json:"description" gorm:"type:text"`
	WorkDate    time.Time `json:"work_date" gorm:"not null"`

	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt *time.Time `json:"deleted_at" gorm:"index"`

	// 关联关系
	Task *AgileTask `json:"task,omitempty" gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE"`
	User *User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// Board 看板模型
type Board struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ProjectID   uuid.UUID `json:"project_id" gorm:"type:uuid;not null;index"`
	Name        string    `json:"name" gorm:"size:255;not null"`
	Description *string   `json:"description" gorm:"type:text"`
	Type        string    `json:"type" gorm:"size:50;not null;default:'kanban'"` // kanban, scrum

	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt *time.Time `json:"deleted_at" gorm:"index"`
	CreatedBy *uuid.UUID `json:"created_by" gorm:"type:uuid"`

	// 关联关系
	Project *Project      `json:"project,omitempty" gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
	Columns []BoardColumn `json:"columns,omitempty" gorm:"foreignKey:BoardID"`
	Creator *User         `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
}

// BoardColumn 看板列模型
type BoardColumn struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	BoardID  uuid.UUID `json:"board_id" gorm:"type:uuid;not null;index"`
	Name     string    `json:"name" gorm:"size:100;not null"`
	Position int       `json:"position" gorm:"not null"`
	WIPLimit *int      `json:"wip_limit"`                      // Work In Progress 限制
	Status   string    `json:"status" gorm:"size:50;not null"` // 对应的任务状态
	Color    *string   `json:"color" gorm:"size:7"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// 关联关系
	Board *Board `json:"board,omitempty" gorm:"foreignKey:BoardID;constraint:OnDelete:CASCADE"`
}

// 枚举常量定义

// 任务类型常量
const (
	TaskTypeStory   = "story"   // 用户故事
	TaskTypeTask    = "task"    // 任务
	TaskTypeBug     = "bug"     // 缺陷
	TaskTypeEpic    = "epic"    // 史诗
	TaskTypeSubTask = "subtask" // 子任务
)

// 任务状态常量
const (
	TaskStatusTodo       = "todo"        // 待办
	TaskStatusInProgress = "in_progress" // 进行中
	TaskStatusInReview   = "in_review"   // 代码评审中
	TaskStatusTesting    = "testing"     // 测试中
	TaskStatusDone       = "done"        // 完成
	TaskStatusCancelled  = "cancelled"   // 已取消
)

// 优先级常量
const (
	PriorityLowest  = "lowest"
	PriorityLow     = "low"
	PriorityMedium  = "medium"
	PriorityHigh    = "high"
	PriorityHighest = "highest"
)

// Sprint状态常量
const (
	SprintStatusPlanned = "planned" // 计划中
	SprintStatusActive  = "active"  // 进行中
	SprintStatusClosed  = "closed"  // 已关闭
)

// Epic状态常量
const (
	EpicStatusOpen       = "open"        // 开放
	EpicStatusInProgress = "in_progress" // 进行中
	EpicStatusDone       = "done"        // 完成
	EpicStatusCancelled  = "cancelled"   // 取消
)

// GORM钩子函数
func (s *Sprint) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

func (at *AgileTask) BeforeCreate(tx *gorm.DB) error {
	if at.ID == uuid.Nil {
		at.ID = uuid.New()
	}
	// 生成任务编号
	if at.TaskNumber == 0 {
		// 这里应该通过服务层来生成唯一的任务编号
		// 现在只是占位逻辑
		at.TaskNumber = time.Now().Unix()
	}
	return nil
}

func (e *Epic) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}

func (tc *TaskComment) BeforeCreate(tx *gorm.DB) error {
	if tc.ID == uuid.Nil {
		tc.ID = uuid.New()
	}
	return nil
}

func (ta *TaskAttachment) BeforeCreate(tx *gorm.DB) error {
	if ta.ID == uuid.Nil {
		ta.ID = uuid.New()
	}
	return nil
}

func (wl *WorkLog) BeforeCreate(tx *gorm.DB) error {
	if wl.ID == uuid.Nil {
		wl.ID = uuid.New()
	}
	return nil
}

func (b *Board) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

func (bc *BoardColumn) BeforeCreate(tx *gorm.DB) error {
	if bc.ID == uuid.Nil {
		bc.ID = uuid.New()
	}
	return nil
}

// 表名设置
func (Sprint) TableName() string {
	return "sprints"
}

func (AgileTask) TableName() string {
	return "agile_tasks"
}

func (Epic) TableName() string {
	return "epics"
}

func (TaskComment) TableName() string {
	return "task_comments"
}

func (TaskAttachment) TableName() string {
	return "task_attachments"
}

func (WorkLog) TableName() string {
	return "work_logs"
}

func (Board) TableName() string {
	return "boards"
}

func (BoardColumn) TableName() string {
	return "board_columns"
}

// 业务方法

// IsActive 检查Sprint是否为活动状态
func (s *Sprint) IsActive() bool {
	return s.Status == SprintStatusActive && s.DeletedAt == nil
}

// IsCompleted 检查Sprint是否已完成
func (s *Sprint) IsCompleted() bool {
	return s.Status == SprintStatusClosed
}

// GetDuration 获取Sprint持续时间
func (s *Sprint) GetDuration() time.Duration {
	return s.EndDate.Sub(s.StartDate)
}

// GetProgress 获取Sprint进度（百分比）
func (s *Sprint) GetProgress() float64 {
	if len(s.Tasks) == 0 {
		return 0
	}

	completed := 0
	for _, task := range s.Tasks {
		if task.Status == TaskStatusDone {
			completed++
		}
	}

	return float64(completed) / float64(len(s.Tasks)) * 100
}

// IsCompleted 检查任务是否已完成
func (at *AgileTask) IsCompleted() bool {
	return at.Status == TaskStatusDone
}

// IsInProgress 检查任务是否进行中
func (at *AgileTask) IsInProgress() bool {
	return at.Status == TaskStatusInProgress
}

// GetRemainingStoryPoints 获取剩余故事点
func (at *AgileTask) GetRemainingStoryPoints() int {
	if at.IsCompleted() {
		return 0
	}
	if at.StoryPoints == nil {
		return 0
	}
	return *at.StoryPoints
}

// HasParent 检查是否有父任务
func (at *AgileTask) HasParent() bool {
	return at.ParentID != nil
}

// IsSubTask 检查是否为子任务
func (at *AgileTask) IsSubTask() bool {
	return at.Type == TaskTypeSubTask || at.HasParent()
}

// CanTransitionTo 检查是否可以转换到指定状态
func (at *AgileTask) CanTransitionTo(newStatus string) bool {
	// 简化的状态转换逻辑，实际应该根据工作流配置
	validTransitions := map[string][]string{
		TaskStatusTodo:       {TaskStatusInProgress, TaskStatusCancelled},
		TaskStatusInProgress: {TaskStatusTodo, TaskStatusInReview, TaskStatusTesting, TaskStatusDone, TaskStatusCancelled},
		TaskStatusInReview:   {TaskStatusInProgress, TaskStatusTesting, TaskStatusDone},
		TaskStatusTesting:    {TaskStatusInProgress, TaskStatusDone, TaskStatusInReview},
		TaskStatusDone:       {TaskStatusInProgress}, // 允许重新打开
		TaskStatusCancelled:  {TaskStatusTodo},
	}

	validTargets, exists := validTransitions[at.Status]
	if !exists {
		return false
	}

	for _, valid := range validTargets {
		if valid == newStatus {
			return true
		}
	}
	return false
}
