package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PipelineStatus 流水线状态枚举
type PipelineStatus string

const (
	PipelineStatusPending   PipelineStatus = "pending"
	PipelineStatusRunning   PipelineStatus = "running"
	PipelineStatusSuccess   PipelineStatus = "success"
	PipelineStatusFailed    PipelineStatus = "failed"
	PipelineStatusCancelled PipelineStatus = "cancelled"
)


// TriggerType 触发类型枚举
type TriggerType string

const (
	TriggerTypeManual     TriggerType = "manual"
	TriggerTypePush       TriggerType = "push"
	TriggerTypePR         TriggerType = "pull_request"
	TriggerTypeScheduled  TriggerType = "scheduled"
	TriggerTypeWebhook    TriggerType = "webhook"
)

// Pipeline 流水线模型
type Pipeline struct {
	ID                 uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	RepositoryID       uuid.UUID  `json:"repository_id" gorm:"type:uuid;not null;index"`
	Name               string     `json:"name" gorm:"size:255;not null"`
	DefinitionFilePath string     `json:"definition_file_path" gorm:"size:512;not null"`
	Description        *string    `json:"description" gorm:"type:text"`
	IsActive           bool       `json:"is_active" gorm:"not null;default:true"`
	CreatedAt          time.Time  `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt          time.Time  `json:"updated_at" gorm:"not null;default:now()"`
	DeletedAt          *time.Time `json:"deleted_at" gorm:"index"`

	// 关联关系
	Repository    *Repository    `json:"repository,omitempty" gorm:"foreignKey:RepositoryID"`
	PipelineRuns  []PipelineRun  `json:"pipeline_runs,omitempty" gorm:"foreignKey:PipelineID"`
}

// PipelineRun 流水线执行记录模型
type PipelineRun struct {
	ID          uuid.UUID       `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	PipelineID  uuid.UUID       `json:"pipeline_id" gorm:"type:uuid;not null;index"`
	TriggerType TriggerType     `json:"trigger_type" gorm:"size:50;not null"`
	TriggerBy   *uuid.UUID      `json:"trigger_by" gorm:"type:uuid"`
	CommitSHA   string          `json:"commit_sha" gorm:"size:40;not null"`
	Branch      *string         `json:"branch" gorm:"size:255"`
	Status      PipelineStatus  `json:"status" gorm:"size:20;not null;default:'pending'"`
	StartedAt   *time.Time      `json:"started_at"`
	FinishedAt  *time.Time      `json:"finished_at"`
	Duration    *int64          `json:"duration"` // 持续时间（秒）
	Variables   map[string]string `json:"variables" gorm:"type:jsonb"`
	CreatedAt   time.Time       `json:"created_at" gorm:"not null;default:now()"`

	// 关联关系
	Pipeline    *Pipeline `json:"pipeline,omitempty" gorm:"foreignKey:PipelineID"`
	TriggerUser *User     `json:"trigger_user,omitempty" gorm:"foreignKey:TriggerBy"`
	Jobs        []Job     `json:"jobs,omitempty" gorm:"foreignKey:PipelineRunID"`
}


// Runner 执行器模型
type Runner struct {
	ID             uuid.UUID     `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	TenantID       uuid.UUID     `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Name           string        `json:"name" gorm:"size:255;not null"`
	Description    *string       `json:"description" gorm:"type:text"`
	Tags           []string      `json:"tags" gorm:"type:jsonb"`
	Status         RunnerStatus  `json:"status" gorm:"size:20;not null;default:'offline'"`
	Version        string        `json:"version" gorm:"size:50"`
	OS             string        `json:"os" gorm:"size:50"`
	Architecture   string        `json:"architecture" gorm:"size:50"`
	LastContactAt  *time.Time    `json:"last_contact_at"`
	CreatedAt      time.Time     `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt      time.Time     `json:"updated_at" gorm:"not null;default:now()"`

	// 关联关系
	Jobs []Job `json:"jobs,omitempty" gorm:"foreignKey:RunnerID"`
}

// RunnerStatus 执行器状态枚举
type RunnerStatus string

const (
	RunnerStatusOnline  RunnerStatus = "online"
	RunnerStatusOffline RunnerStatus = "offline"
	RunnerStatusIdle    RunnerStatus = "idle"
	RunnerStatusBusy    RunnerStatus = "busy"
)

// 引用的其他模型
type Repository struct {
	ID            uuid.UUID `json:"id" gorm:"type:uuid;primary_key"`
	ProjectID     uuid.UUID `json:"project_id"`
	Name          string    `json:"name"`
	Description   *string   `json:"description"`
	Visibility    string    `json:"visibility"`
	DefaultBranch string    `json:"default_branch"`
}

type User struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	DisplayName *string   `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url"`
}

// 请求和响应模型

// CreatePipelineRequest 创建流水线请求
type CreatePipelineRequest struct {
	RepositoryID       string  `json:"repository_id" binding:"required,uuid"`
	Name               string  `json:"name" binding:"required,min=1,max=255"`
	DefinitionFilePath string  `json:"definition_file_path" binding:"required,min=1,max=512"`
	Description        *string `json:"description" validate:"omitempty,max=2000"`
}

// UpdatePipelineRequest 更新流水线请求
type UpdatePipelineRequest struct {
	Name               *string `json:"name" validate:"omitempty,min=1,max=255"`
	DefinitionFilePath *string `json:"definition_file_path" validate:"omitempty,min=1,max=512"`
	Description        *string `json:"description" validate:"omitempty,max=2000"`
	IsActive           *bool   `json:"is_active"`
}

// TriggerPipelineRequest 触发流水线请求
type TriggerPipelineRequest struct {
	CommitSHA string            `json:"commit_sha" binding:"required,len=40"`
	Branch    *string           `json:"branch"`
	Variables map[string]string `json:"variables"`
}

// PipelineListResponse 流水线列表响应
type PipelineListResponse struct {
	Pipelines []Pipeline `json:"pipelines"`
	Total     int64      `json:"total"`
	Page      int        `json:"page"`
	PageSize  int        `json:"page_size"`
}

// PipelineRunListResponse 流水线运行列表响应
type PipelineRunListResponse struct {
	PipelineRuns []PipelineRun `json:"pipeline_runs"`
	Total        int64         `json:"total"`
	Page         int           `json:"page"`
	PageSize     int           `json:"page_size"`
}

// RegisterRunnerRequest 注册执行器请求
type RegisterRunnerRequest struct {
	Name         string   `json:"name" binding:"required,min=1,max=255"`
	Description  *string  `json:"description" validate:"omitempty,max=2000"`
	Tags         []string `json:"tags"`
	Version      string   `json:"version" binding:"required"`
	OS           string   `json:"os" binding:"required"`
	Architecture string   `json:"architecture" binding:"required"`
}

// UpdateRunnerRequest 更新执行器请求
type UpdateRunnerRequest struct {
	Name         *string   `json:"name" validate:"omitempty,min=1,max=255"`
	Description  *string   `json:"description" validate:"omitempty,max=2000"`
	Tags         *[]string `json:"tags"`
	Status       *string   `json:"status" validate:"omitempty,oneof=online offline idle busy"`
	Version      *string   `json:"version"`
	OS           *string   `json:"os"`
	Architecture *string   `json:"architecture"`
}

// 设置表名
func (Pipeline) TableName() string {
	return "pipelines"
}

func (PipelineRun) TableName() string {
	return "pipeline_runs"
}


func (Runner) TableName() string {
	return "runners"
}

// GORM钩子

// BeforeCreate 创建前的处理
func (p *Pipeline) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		var newID uuid.UUID
		err := tx.Raw("SELECT uuid_generate_v7()").Scan(&newID).Error
		if err != nil {
			return err
		}
		p.ID = newID
	}
	return nil
}

func (pr *PipelineRun) BeforeCreate(tx *gorm.DB) error {
	if pr.ID == uuid.Nil {
		var newID uuid.UUID
		err := tx.Raw("SELECT uuid_generate_v7()").Scan(&newID).Error
		if err != nil {
			return err
		}
		pr.ID = newID
	}
	return nil
}


func (r *Runner) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		var newID uuid.UUID
		err := tx.Raw("SELECT uuid_generate_v7()").Scan(&newID).Error
		if err != nil {
			return err
		}
		r.ID = newID
	}
	return nil
}

// BeforeUpdate 更新前的处理
func (p *Pipeline) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = time.Now()
	return nil
}

func (r *Runner) BeforeUpdate(tx *gorm.DB) error {
	r.UpdatedAt = time.Now()
	return nil
}

// 业务方法

// IsRunning 检查流水线是否正在运行
func (pr *PipelineRun) IsRunning() bool {
	return pr.Status == PipelineStatusRunning
}

// IsFinished 检查流水线是否已完成
func (pr *PipelineRun) IsFinished() bool {
	return pr.Status == PipelineStatusSuccess || 
		   pr.Status == PipelineStatusFailed || 
		   pr.Status == PipelineStatusCancelled
}

// CanCancel 检查流水线是否可以取消
func (pr *PipelineRun) CanCancel() bool {
	return pr.Status == PipelineStatusPending || pr.Status == PipelineStatusRunning
}

// GetDuration 获取执行持续时间
func (pr *PipelineRun) GetDuration() time.Duration {
	if pr.StartedAt == nil {
		return 0
	}
	
	endTime := time.Now()
	if pr.FinishedAt != nil {
		endTime = *pr.FinishedAt
	}
	
	return endTime.Sub(*pr.StartedAt)
}

// UpdateDuration 更新持续时间
func (pr *PipelineRun) UpdateDuration() {
	duration := pr.GetDuration()
	durationSeconds := int64(duration.Seconds())
	pr.Duration = &durationSeconds
}

// IsOnline 检查执行器是否在线
func (r *Runner) IsOnline() bool {
	return r.Status == RunnerStatusOnline || r.Status == RunnerStatusIdle || r.Status == RunnerStatusBusy
}

// IsAvailable 检查执行器是否可用（在线且空闲）
func (r *Runner) IsAvailable() bool {
	return r.Status == RunnerStatusOnline || r.Status == RunnerStatusIdle
}

// MatchesTags 检查执行器是否匹配指定标签
func (r *Runner) MatchesTags(requiredTags []string) bool {
	if len(requiredTags) == 0 {
		return true
	}
	
	runnerTagSet := make(map[string]bool)
	for _, tag := range r.Tags {
		runnerTagSet[tag] = true
	}
	
	for _, requiredTag := range requiredTags {
		if !runnerTagSet[requiredTag] {
			return false
		}
	}
	
	return true
}

// 统计数据模型

// PipelineStats 流水线统计信息
type PipelineStats struct {
	TotalRuns       int64   `json:"total_runs"`
	SuccessfulRuns  int64   `json:"successful_runs"`
	FailedRuns      int64   `json:"failed_runs"`
	SuccessRate     float64 `json:"success_rate"`
	AverageDuration int64   `json:"average_duration"` // 秒
}

// RunnerStats 执行器统计信息
type RunnerStats struct {
	TotalJobs          int64   `json:"total_jobs"`
	SuccessfulJobs     int64   `json:"successful_jobs"`
	FailedJobs         int64   `json:"failed_jobs"`
	SuccessRate        float64 `json:"success_rate"`
	AverageJobDuration int64   `json:"average_job_duration"` // 秒
}