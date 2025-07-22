package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// JobStatus 作业状态枚举
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusSuccess   JobStatus = "success"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
	JobStatusSkipped   JobStatus = "skipped"
)

// JobType 作业类型枚举
type JobType string

const (
	JobTypeBuild   JobType = "build"
	JobTypeTest    JobType = "test"
	JobTypeDeploy  JobType = "deploy"
	JobTypeScript  JobType = "script"
	JobTypeCleanup JobType = "cleanup"
)

// JobPriority 作业优先级枚举
type JobPriority int

const (
	JobPriorityLow      JobPriority = 1
	JobPriorityNormal   JobPriority = 5
	JobPriorityHigh     JobPriority = 8
	JobPriorityCritical JobPriority = 10
)

// Job 作业模型
type Job struct {
	ID             uuid.UUID            `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name           string               `json:"name" gorm:"not null;index"`
	Description    string               `json:"description"`
	Type           JobType              `json:"type" gorm:"not null;index"`
	Status         JobStatus            `json:"status" gorm:"not null;index;default:'pending'"`
	Priority       JobPriority          `json:"priority" gorm:"not null;default:5"`
	
	// 关联关系
	PipelineRunID  uuid.UUID            `json:"pipeline_run_id" gorm:"type:uuid;not null;index"`
	PipelineRun    *PipelineRun         `json:"pipeline_run,omitempty" gorm:"foreignKey:PipelineRunID;constraint:OnDelete:CASCADE"`
	
	RunnerID       *uuid.UUID           `json:"runner_id" gorm:"type:uuid;index"`
	Runner         *Runner              `json:"runner,omitempty" gorm:"foreignKey:RunnerID"`
	
	AssignedRunnerID *uuid.UUID         `json:"assigned_runner_id" gorm:"type:uuid;index"`
	AssignedRunner   *Runner            `json:"assigned_runner,omitempty" gorm:"foreignKey:AssignedRunnerID"`
	
	// 作业配置
	Config         map[string]interface{} `json:"config" gorm:"type:jsonb"`
	Environment    map[string]string      `json:"environment" gorm:"type:jsonb"`
	Secrets        []string               `json:"secrets" gorm:"type:jsonb"`
	
	// 执行步骤
	Steps          []JobStep            `json:"steps" gorm:"type:jsonb"`
	
	// 依赖关系
	Dependencies   []uuid.UUID          `json:"dependencies" gorm:"type:jsonb"`
	
	// 资源要求
	Requirements   *JobRequirements     `json:"requirements" gorm:"type:jsonb"`
	
	// 执行信息
	StartedAt      *time.Time           `json:"started_at"`
	FinishedAt     *time.Time           `json:"finished_at"`
	Duration       *time.Duration       `json:"duration" gorm:"-"`
	ExitCode       *int                 `json:"exit_code"`
	ErrorMessage   string               `json:"error_message"`
	
	// 重试信息
	RetryCount     int                  `json:"retry_count" gorm:"default:0"`
	MaxRetries     int                  `json:"max_retries" gorm:"default:3"`
	
	// 日志和输出
	LogPath        string               `json:"log_path"`
	ArtifactPaths  []string             `json:"artifact_paths" gorm:"type:jsonb"`
	
	// 审计字段
	CreatedAt      time.Time            `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time            `json:"updated_at" gorm:"autoUpdateTime"`
	CreatedBy      *uuid.UUID           `json:"created_by" gorm:"type:uuid"`
	UpdatedBy      *uuid.UUID           `json:"updated_by" gorm:"type:uuid"`
}

// JobStep 作业步骤
type JobStep struct {
	Name        string            `json:"name"`
	Commands    string            `json:"commands"`
	WorkingDir  string            `json:"working_dir,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Timeout     *time.Duration    `json:"timeout,omitempty"`
	AllowFailure bool             `json:"allow_failure,omitempty"`
	When        string            `json:"when,omitempty"` // always, on_success, on_failure, manual
}

// JobRequirements 作业资源要求
type JobRequirements struct {
	CPU       float64 `json:"cpu,omitempty"`       // CPU核心数
	Memory    int64   `json:"memory,omitempty"`    // 内存字节数
	Disk      int64   `json:"disk,omitempty"`      // 磁盘空间字节数
	GPU       int     `json:"gpu,omitempty"`       // GPU数量
	Network   bool    `json:"network,omitempty"`   // 是否需要网络访问
	Docker    bool    `json:"docker,omitempty"`    // 是否需要Docker
	
	// 标签要求（用于Runner匹配）
	Labels    map[string]string `json:"labels,omitempty"`
	
	// 操作系统要求
	OS        string   `json:"os,omitempty"`        // linux, windows, macos
	Arch      string   `json:"arch,omitempty"`      // amd64, arm64
	
	// 软件要求
	Software  []string `json:"software,omitempty"`  // 需要的软件列表
}

// BeforeCreate GORM钩子：创建前
func (j *Job) BeforeCreate(tx *gorm.DB) error {
	if j.ID == uuid.Nil {
		j.ID = uuid.New()
	}
	return nil
}

// AfterFind GORM钩子：查询后计算持续时间
func (j *Job) AfterFind(tx *gorm.DB) error {
	if j.StartedAt != nil && j.FinishedAt != nil {
		duration := j.FinishedAt.Sub(*j.StartedAt)
		j.Duration = &duration
	}
	return nil
}

// IsRunning 判断作业是否正在运行
func (j *Job) IsRunning() bool {
	return j.Status == JobStatusRunning
}

// IsCompleted 判断作业是否已完成
func (j *Job) IsCompleted() bool {
	return j.Status == JobStatusSuccess || 
		   j.Status == JobStatusFailed || 
		   j.Status == JobStatusCancelled
}

// IsRetryable 判断作业是否可重试
func (j *Job) IsRetryable() bool {
	return j.Status == JobStatusFailed && j.RetryCount < j.MaxRetries
}

// CanStart 判断作业是否可以开始执行
func (j *Job) CanStart() bool {
	return j.Status == JobStatusPending
}

// GetDurationString 获取持续时间字符串
func (j *Job) GetDurationString() string {
	if j.Duration != nil {
		return j.Duration.String()
	}
	
	if j.StartedAt != nil {
		if j.FinishedAt != nil {
			return j.FinishedAt.Sub(*j.StartedAt).String()
		}
		// 正在运行的作业
		return time.Since(*j.StartedAt).String()
	}
	
	return ""
}

// GetEstimatedDuration 获取预估持续时间
func (j *Job) GetEstimatedDuration() time.Duration {
	// 这里可以根据历史数据来估算
	// 简化实现，返回默认值
	switch j.Type {
	case JobTypeBuild:
		return 10 * time.Minute
	case JobTypeTest:
		return 5 * time.Minute
	case JobTypeDeploy:
		return 15 * time.Minute
	case JobTypeScript:
		return 3 * time.Minute
	case JobTypeCleanup:
		return 2 * time.Minute
	default:
		return 5 * time.Minute
	}
}

// JobExecution 作业执行记录（用于历史追踪）
type JobExecution struct {
	ID            uuid.UUID    `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	JobID         uuid.UUID    `json:"job_id" gorm:"type:uuid;not null;index"`
	Job           *Job         `json:"job,omitempty" gorm:"foreignKey:JobID;constraint:OnDelete:CASCADE"`
	
	RunnerID      uuid.UUID    `json:"runner_id" gorm:"type:uuid;not null;index"`
	Runner        *Runner      `json:"runner,omitempty" gorm:"foreignKey:RunnerID"`
	
	Status        JobStatus    `json:"status" gorm:"not null"`
	ExitCode      *int         `json:"exit_code"`
	ErrorMessage  string       `json:"error_message"`
	
	StartedAt     time.Time    `json:"started_at" gorm:"not null"`
	FinishedAt    *time.Time   `json:"finished_at"`
	Duration      time.Duration `json:"duration" gorm:"-"`
	
	LogPath       string       `json:"log_path"`
	ArtifactPaths []string     `json:"artifact_paths" gorm:"type:jsonb"`
	
	// 资源使用情况
	ResourceUsage map[string]interface{} `json:"resource_usage" gorm:"type:jsonb"`
	
	CreatedAt     time.Time    `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time    `json:"updated_at" gorm:"autoUpdateTime"`
}

// BeforeCreate GORM钩子：创建前
func (je *JobExecution) BeforeCreate(tx *gorm.DB) error {
	if je.ID == uuid.Nil {
		je.ID = uuid.New()
	}
	return nil
}

// AfterFind GORM钩子：查询后计算持续时间
func (je *JobExecution) AfterFind(tx *gorm.DB) error {
	if je.FinishedAt != nil {
		je.Duration = je.FinishedAt.Sub(je.StartedAt)
	} else {
		je.Duration = time.Since(je.StartedAt)
	}
	return nil
}

// JobQueue 作业队列模型
type JobQueue struct {
	ID          uuid.UUID    `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	JobID       uuid.UUID    `json:"job_id" gorm:"type:uuid;not null;index"`
	Job         *Job         `json:"job,omitempty" gorm:"foreignKey:JobID;constraint:OnDelete:CASCADE"`
	
	Priority    JobPriority  `json:"priority" gorm:"not null;index"`
	QueuedAt    time.Time    `json:"queued_at" gorm:"not null;autoCreateTime"`
	
	// 调度信息
	ScheduledAt *time.Time   `json:"scheduled_at"`
	RunnerID    *uuid.UUID   `json:"runner_id" gorm:"type:uuid"`
	Runner      *Runner      `json:"runner,omitempty" gorm:"foreignKey:RunnerID"`
	
	// 重试信息
	Attempts    int          `json:"attempts" gorm:"default:0"`
	LastError   string       `json:"last_error"`
	NextRetryAt *time.Time   `json:"next_retry_at"`
	
	CreatedAt   time.Time    `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time    `json:"updated_at" gorm:"autoUpdateTime"`
}

// BeforeCreate GORM钩子：创建前
func (jq *JobQueue) BeforeCreate(tx *gorm.DB) error {
	if jq.ID == uuid.Nil {
		jq.ID = uuid.New()
	}
	return nil
}

// TableName 指定表名
func (j *Job) TableName() string {
	return "jobs"
}

func (je *JobExecution) TableName() string {
	return "job_executions"
}

func (jq *JobQueue) TableName() string {
	return "job_queue"
}