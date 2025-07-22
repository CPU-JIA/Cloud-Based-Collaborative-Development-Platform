package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Event 通用事件模型
type Event struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	Source        string          `json:"source"`        // 事件来源服务
	Subject       string          `json:"subject"`       // 事件主题
	Time          time.Time       `json:"time"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	UserID        *uuid.UUID      `json:"user_id,omitempty"`
	ProjectID     *uuid.UUID      `json:"project_id,omitempty"`
	Data          json.RawMessage `json:"data"`
	DataSchema    string          `json:"data_schema,omitempty"`
	CorrelationID string          `json:"correlation_id,omitempty"`
}

// 项目事件数据结构

// TaskAssignedEventData 任务分配事件数据
type TaskAssignedEventData struct {
	TaskID        uuid.UUID `json:"task_id"`
	TaskTitle     string    `json:"task_title"`
	AssigneeID    uuid.UUID `json:"assignee_id"`
	AssigneeName  string    `json:"assignee_name"`
	AssignerID    uuid.UUID `json:"assigner_id"`
	AssignerName  string    `json:"assigner_name"`
	ProjectID     uuid.UUID `json:"project_id"`
	ProjectName   string    `json:"project_name"`
	Priority      string    `json:"priority"`
	DueDate       *time.Time `json:"due_date,omitempty"`
	Description   string    `json:"description,omitempty"`
}

// TaskCompletedEventData 任务完成事件数据
type TaskCompletedEventData struct {
	TaskID        uuid.UUID `json:"task_id"`
	TaskTitle     string    `json:"task_title"`
	CompletedByID uuid.UUID `json:"completed_by_id"`
	CompletedBy   string    `json:"completed_by"`
	ProjectID     uuid.UUID `json:"project_id"`
	ProjectName   string    `json:"project_name"`
	CompletedAt   time.Time `json:"completed_at"`
	Duration      int64     `json:"duration"` // 任务持续时间(小时)
}

// SprintStartedEventData Sprint开始事件数据
type SprintStartedEventData struct {
	SprintID      uuid.UUID `json:"sprint_id"`
	SprintName    string    `json:"sprint_name"`
	ProjectID     uuid.UUID `json:"project_id"`
	ProjectName   string    `json:"project_name"`
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
	Duration      int       `json:"duration"` // Sprint持续时间(天)
	TaskCount     int       `json:"task_count"`
	StoryPoints   int       `json:"story_points"`
	TeamMembers   []TeamMember `json:"team_members"`
}

// SprintCompletedEventData Sprint完成事件数据
type SprintCompletedEventData struct {
	SprintID           uuid.UUID `json:"sprint_id"`
	SprintName         string    `json:"sprint_name"`
	ProjectID          uuid.UUID `json:"project_id"`
	ProjectName        string    `json:"project_name"`
	CompletedAt        time.Time `json:"completed_at"`
	PlannedStoryPoints int       `json:"planned_story_points"`
	ActualStoryPoints  int       `json:"actual_story_points"`
	CompletionRate     float64   `json:"completion_rate"`
	VelocityAchieved   float64   `json:"velocity_achieved"`
}

// TeamMember 团队成员信息
type TeamMember struct {
	UserID   uuid.UUID `json:"user_id"`
	Name     string    `json:"name"`
	Role     string    `json:"role"`
	Email    string    `json:"email"`
}

// CI/CD事件数据结构

// BuildStartedEventData 构建开始事件数据
type BuildStartedEventData struct {
	PipelineID    uuid.UUID `json:"pipeline_id"`
	PipelineName  string    `json:"pipeline_name"`
	BuildNumber   int       `json:"build_number"`
	ProjectID     uuid.UUID `json:"project_id"`
	ProjectName   string    `json:"project_name"`
	Branch        string    `json:"branch"`
	Commit        string    `json:"commit"`
	CommitMessage string    `json:"commit_message"`
	TriggeredBy   uuid.UUID `json:"triggered_by"`
	TriggerName   string    `json:"trigger_name"`
	StartedAt     time.Time `json:"started_at"`
}

// BuildCompletedEventData 构建完成事件数据
type BuildCompletedEventData struct {
	PipelineID      uuid.UUID `json:"pipeline_id"`
	PipelineName    string    `json:"pipeline_name"`
	BuildNumber     int       `json:"build_number"`
	ProjectID       uuid.UUID `json:"project_id"`
	ProjectName     string    `json:"project_name"`
	Branch          string    `json:"branch"`
	Commit          string    `json:"commit"`
	Status          string    `json:"status"` // success, failed, cancelled
	Duration        int64     `json:"duration"` // 构建时间(秒)
	CompletedAt     time.Time `json:"completed_at"`
	TestResults     *TestResults `json:"test_results,omitempty"`
	ArtifactsCount  int       `json:"artifacts_count"`
	FailureReason   string    `json:"failure_reason,omitempty"`
}

// TestResults 测试结果
type TestResults struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

// DeploymentEventData 部署事件数据
type DeploymentEventData struct {
	DeploymentID   uuid.UUID `json:"deployment_id"`
	ProjectID      uuid.UUID `json:"project_id"`
	ProjectName    string    `json:"project_name"`
	Environment    string    `json:"environment"` // dev, staging, production
	Version        string    `json:"version"`
	Status         string    `json:"status"` // started, success, failed, rollback
	DeployedBy     uuid.UUID `json:"deployed_by"`
	DeployerName   string    `json:"deployer_name"`
	StartedAt      time.Time `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	Duration       *int64    `json:"duration,omitempty"` // 部署时间(秒)
	FailureReason  string    `json:"failure_reason,omitempty"`
}

// Git事件数据结构

// PullRequestEventData Pull Request事件数据
type PullRequestEventData struct {
	PullRequestID    int       `json:"pull_request_id"`
	Title            string    `json:"title"`
	ProjectID        uuid.UUID `json:"project_id"`
	ProjectName      string    `json:"project_name"`
	Repository       string    `json:"repository"`
	SourceBranch     string    `json:"source_branch"`
	TargetBranch     string    `json:"target_branch"`
	AuthorID         uuid.UUID `json:"author_id"`
	AuthorName       string    `json:"author_name"`
	Action           string    `json:"action"` // opened, closed, merged, review_requested
	ReviewerID       *uuid.UUID `json:"reviewer_id,omitempty"`
	ReviewerName     string    `json:"reviewer_name,omitempty"`
	ChangesCount     int       `json:"changes_count"`
	CommitsCount     int       `json:"commits_count"`
	URL              string    `json:"url"`
}

// CodeReviewEventData 代码审查事件数据
type CodeReviewEventData struct {
	ReviewID        uuid.UUID `json:"review_id"`
	PullRequestID   int       `json:"pull_request_id"`
	ProjectID       uuid.UUID `json:"project_id"`
	ProjectName     string    `json:"project_name"`
	ReviewerID      uuid.UUID `json:"reviewer_id"`
	ReviewerName    string    `json:"reviewer_name"`
	AuthorID        uuid.UUID `json:"author_id"`
	AuthorName      string    `json:"author_name"`
	Action          string    `json:"action"` // approved, requested_changes, commented
	CommentsCount   int       `json:"comments_count"`
	ReviewedAt      time.Time `json:"reviewed_at"`
	URL             string    `json:"url"`
}

// 系统事件数据结构

// SystemAlertEventData 系统告警事件数据
type SystemAlertEventData struct {
	AlertID       string    `json:"alert_id"`
	AlertType     string    `json:"alert_type"` // performance, security, error, resource
	Severity      string    `json:"severity"` // low, medium, high, critical
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Service       string    `json:"service"`
	Environment   string    `json:"environment"`
	Metrics       map[string]interface{} `json:"metrics,omitempty"`
	TriggeredAt   time.Time `json:"triggered_at"`
	ResolvedAt    *time.Time `json:"resolved_at,omitempty"`
	ActionRequired bool     `json:"action_required"`
}

// SecurityEventData 安全事件数据
type SecurityEventData struct {
	EventType     string    `json:"event_type"` // login_failure, suspicious_activity, permission_change
	UserID        *uuid.UUID `json:"user_id,omitempty"`
	UserName      string    `json:"user_name,omitempty"`
	IPAddress     string    `json:"ip_address"`
	UserAgent     string    `json:"user_agent,omitempty"`
	Location      string    `json:"location,omitempty"`
	RiskLevel     string    `json:"risk_level"` // low, medium, high, critical
	Description   string    `json:"description"`
	ActionTaken   string    `json:"action_taken,omitempty"`
	RequiresAction bool     `json:"requires_action"`
	DetectedAt    time.Time `json:"detected_at"`
}

// UserEventData 用户事件数据
type UserEventData struct {
	EventType   string    `json:"event_type"` // user_created, user_updated, user_deleted, role_changed
	TargetUserID uuid.UUID `json:"target_user_id"`
	TargetUserName string  `json:"target_user_name"`
	ActorID     *uuid.UUID `json:"actor_id,omitempty"`
	ActorName   string    `json:"actor_name,omitempty"`
	Changes     map[string]interface{} `json:"changes,omitempty"`
	Reason      string    `json:"reason,omitempty"`
}

// 常用事件类型常量
const (
	// 项目事件
	EventTypeTaskAssigned        = "project.task.assigned"
	EventTypeTaskCompleted       = "project.task.completed"
	EventTypeTaskOverdue         = "project.task.overdue"
	EventTypeSprintStarted       = "project.sprint.started"
	EventTypeSprintCompleted     = "project.sprint.completed"
	EventTypeMilestoneReached    = "project.milestone.reached"
	
	// CI/CD事件
	EventTypeBuildStarted        = "cicd.build.started"
	EventTypeBuildCompleted      = "cicd.build.completed"
	EventTypeBuildFailed         = "cicd.build.failed"
	EventTypeDeploymentStarted   = "cicd.deployment.started"
	EventTypeDeploymentCompleted = "cicd.deployment.completed"
	EventTypeDeploymentFailed    = "cicd.deployment.failed"
	
	// Git事件
	EventTypePullRequestOpened   = "git.pr.opened"
	EventTypePullRequestMerged   = "git.pr.merged"
	EventTypePullRequestClosed   = "git.pr.closed"
	EventTypeCodeReviewRequested = "git.review.requested"
	EventTypeCodeReviewCompleted = "git.review.completed"
	
	// 系统事件
	EventTypeSystemAlert         = "system.alert"
	EventTypeSecurityAlert       = "security.alert"
	EventTypeUserCreated         = "user.created"
	EventTypeUserUpdated         = "user.updated"
	EventTypePermissionChanged   = "user.permission.changed"
)

// 通知类别常量
const (
	CategoryProject     = "project"
	CategoryCICD        = "cicd"
	CategoryGit         = "git"
	CategorySystem      = "system"
	CategorySecurity    = "security"
	CategoryUser        = "user"
)

// 通知优先级常量
const (
	PriorityLow      = "low"
	PriorityMedium   = "medium"
	PriorityHigh     = "high"
	PriorityCritical = "critical"
)

// 通知状态常量
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusSent       = "sent"
	StatusFailed     = "failed"
	StatusCancelled  = "cancelled"
)

// 发送渠道常量
const (
	ChannelEmail   = "email"
	ChannelWebhook = "webhook"
	ChannelInApp   = "in_app"
	ChannelPush    = "push"
)

// 发送状态常量
const (
	DeliveryStatusSending   = "sending"
	DeliveryStatusSent      = "sent"
	DeliveryStatusDelivered = "delivered"
	DeliveryStatusFailed    = "failed"
	DeliveryStatusBounced   = "bounced"
)