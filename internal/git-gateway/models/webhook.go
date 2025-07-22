package models

import (
	"time"

	"github.com/google/uuid"
)

// User 用户模型（简化版，用于Webhook事件）
type User struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Name     string    `json:"name"`
}


// WebhookEvent Git钩子事件模型
type WebhookEvent struct {
	ID           uuid.UUID              `json:"id" db:"id"`
	RepositoryID uuid.UUID              `json:"repository_id" db:"repository_id"`
	EventType    WebhookEventType       `json:"event_type" db:"event_type"`
	EventData    map[string]interface{} `json:"event_data" db:"event_data"`
	Source       string                 `json:"source" db:"source"` // git, github, gitlab等
	Signature    string                 `json:"signature" db:"signature"`
	Processed    bool                   `json:"processed" db:"processed"`
	ProcessedAt  *time.Time             `json:"processed_at" db:"processed_at"`
	ErrorMessage string                 `json:"error_message" db:"error_message"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
}

// WebhookEventType 钩子事件类型
type WebhookEventType string

const (
	// 推送事件
	EventTypePush        WebhookEventType = "push"
	EventTypeTagPush     WebhookEventType = "tag_push"
	EventTypeBranchPush  WebhookEventType = "branch_push"
	
	// 拉取请求事件
	EventTypePullRequest WebhookEventType = "pull_request"
	EventTypeMergeRequest WebhookEventType = "merge_request"
	
	// 分支事件
	EventTypeBranchCreate WebhookEventType = "branch_create"
	EventTypeBranchDelete WebhookEventType = "branch_delete"
	
	// 标签事件  
	EventTypeTagCreate WebhookEventType = "tag_create"
	EventTypeTagDelete WebhookEventType = "tag_delete"
	
	// 提交事件
	EventTypeCommit WebhookEventType = "commit"
	
	// 仓库事件
	EventTypeRepoCreate WebhookEventType = "repository_create"
	EventTypeRepoDelete WebhookEventType = "repository_delete"
	EventTypeRepoUpdate WebhookEventType = "repository_update"
)

// PushEvent 推送事件数据
type PushEvent struct {
	Ref        string   `json:"ref"`         // refs/heads/main
	Before     string   `json:"before"`      // 推送前的SHA
	After      string   `json:"after"`       // 推送后的SHA
	Created    bool     `json:"created"`     // 是否是新建分支/标签
	Deleted    bool     `json:"deleted"`     // 是否是删除分支/标签
	Forced     bool     `json:"forced"`      // 是否是强制推送
	Compare    string   `json:"compare"`     // 比较链接
	Commits    []Commit `json:"commits"`     // 提交列表
	HeadCommit *Commit  `json:"head_commit"` // 头部提交
	Repository Repository `json:"repository"` // 仓库信息
	Pusher     User     `json:"pusher"`      // 推送者信息
}

// PullRequestEvent 拉取请求事件数据
type PullRequestEvent struct {
	Action      string      `json:"action"`       // opened, closed, merged等
	Number      int         `json:"number"`       // PR编号
	PullRequest PullRequest `json:"pull_request"` // PR详情
	Repository  Repository  `json:"repository"`   // 仓库信息
	Sender      User        `json:"sender"`       // 发送者信息
}

// BranchEvent 分支事件数据
type BranchEvent struct {
	Action     string     `json:"action"`     // created, deleted
	Ref        string     `json:"ref"`        // 分支引用
	RefType    string     `json:"ref_type"`   // branch, tag
	Repository Repository `json:"repository"` // 仓库信息
	Sender     User       `json:"sender"`     // 发送者信息
}

// TagEvent 标签事件数据  
type TagEvent struct {
	Action     string     `json:"action"`     // created, deleted
	Ref        string     `json:"ref"`        // 标签引用
	RefType    string     `json:"ref_type"`   // tag
	Repository Repository `json:"repository"` // 仓库信息
	Sender     User       `json:"sender"`     // 发送者信息
}

// WebhookTrigger 钩子触发器配置
type WebhookTrigger struct {
	ID           uuid.UUID        `json:"id" db:"id"`
	RepositoryID uuid.UUID        `json:"repository_id" db:"repository_id"`
	Name         string           `json:"name" db:"name"`
	EventTypes   []WebhookEventType `json:"event_types" db:"event_types"`
	Conditions   TriggerConditions `json:"conditions" db:"conditions"`
	Actions      TriggerActions   `json:"actions" db:"actions"`
	Enabled      bool             `json:"enabled" db:"enabled"`
	CreatedAt    time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at" db:"updated_at"`
}

// TriggerConditions 触发条件
type TriggerConditions struct {
	Branches     []string           `json:"branches"`      // 触发的分支
	Tags         []string           `json:"tags"`          // 触发的标签
	Paths        []string           `json:"paths"`         // 触发的文件路径
	Authors      []string           `json:"authors"`       // 触发的作者
	CommitMessage string            `json:"commit_message"` // 提交信息匹配
	FileChanges  FileChangeCondition `json:"file_changes"` // 文件变更条件
}

// FileChangeCondition 文件变更条件
type FileChangeCondition struct {
	Include []string `json:"include"` // 包含的路径模式
	Exclude []string `json:"exclude"` // 排除的路径模式
}

// TriggerActions 触发动作
type TriggerActions struct {
	StartPipeline   *PipelineAction   `json:"start_pipeline"`   // 启动流水线
	SendNotification *NotificationAction `json:"send_notification"` // 发送通知
	CallWebhook     *WebhookAction    `json:"call_webhook"`     // 调用Webhook
}

// PipelineAction 流水线动作
type PipelineAction struct {
	PipelineID   uuid.UUID                 `json:"pipeline_id"`
	Variables    map[string]interface{}    `json:"variables"`
	Environment  string                   `json:"environment"`
	Parameters   map[string]interface{}   `json:"parameters"`
}

// NotificationAction 通知动作
type NotificationAction struct {
	Type     string   `json:"type"`     // email, slack, webhook
	Recipients []string `json:"recipients"`
	Template string   `json:"template"`
	Subject  string   `json:"subject"`
}

// WebhookAction Webhook动作
type WebhookAction struct {
	URL       string                 `json:"url"`
	Method    string                 `json:"method"`
	Headers   map[string]string      `json:"headers"`
	Body      map[string]interface{} `json:"body"`
	Secret    string                 `json:"secret"`
}

// WebhookDelivery 钩子投递记录
type WebhookDelivery struct {
	ID           uuid.UUID         `json:"id" db:"id"`
	WebhookID    uuid.UUID         `json:"webhook_id" db:"webhook_id"`
	EventID      uuid.UUID         `json:"event_id" db:"event_id"`
	URL          string            `json:"url" db:"url"`
	Method       string            `json:"method" db:"method"`
	Status       DeliveryStatus    `json:"status" db:"status"`
	StatusCode   int               `json:"status_code" db:"status_code"`
	RequestBody  string            `json:"request_body" db:"request_body"`
	ResponseBody string            `json:"response_body" db:"response_body"`
	Duration     int64             `json:"duration" db:"duration"` // 毫秒
	Attempts     int               `json:"attempts" db:"attempts"`
	MaxAttempts  int               `json:"max_attempts" db:"max_attempts"`
	NextRetryAt  *time.Time        `json:"next_retry_at" db:"next_retry_at"`
	CreatedAt    time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at" db:"updated_at"`
}

// DeliveryStatus 投递状态
type DeliveryStatus string

const (
	DeliveryStatusPending DeliveryStatus = "pending"
	DeliveryStatusSuccess DeliveryStatus = "success"
	DeliveryStatusFailed  DeliveryStatus = "failed"
	DeliveryStatusRetrying DeliveryStatus = "retrying"
)

// CreateWebhookEventRequest 创建Webhook事件请求
type CreateWebhookEventRequest struct {
	RepositoryID uuid.UUID              `json:"repository_id" binding:"required"`
	EventType    WebhookEventType       `json:"event_type" binding:"required"`
	EventData    map[string]interface{} `json:"event_data" binding:"required"`
	Source       string                 `json:"source"`
	Signature    string                 `json:"signature"`
}

// CreateWebhookTriggerRequest 创建钩子触发器请求
type CreateWebhookTriggerRequest struct {
	RepositoryID uuid.UUID         `json:"repository_id" binding:"required"`
	Name         string            `json:"name" binding:"required"`
	EventTypes   []WebhookEventType `json:"event_types" binding:"required"`
	Conditions   TriggerConditions `json:"conditions"`
	Actions      TriggerActions    `json:"actions" binding:"required"`
	Enabled      bool              `json:"enabled"`
}

// UpdateWebhookTriggerRequest 更新钩子触发器请求
type UpdateWebhookTriggerRequest struct {
	Name       string            `json:"name"`
	EventTypes []WebhookEventType `json:"event_types"`
	Conditions TriggerConditions `json:"conditions"`
	Actions    TriggerActions    `json:"actions"`
	Enabled    *bool             `json:"enabled"`
}

// WebhookEventFilter 事件过滤器
type WebhookEventFilter struct {
	RepositoryID *uuid.UUID        `json:"repository_id"`
	EventTypes   []WebhookEventType `json:"event_types"`
	Source       string            `json:"source"`
	Processed    *bool             `json:"processed"`
	StartTime    *time.Time        `json:"start_time"`
	EndTime      *time.Time        `json:"end_time"`
}

// WebhookEventListResponse 事件列表响应
type WebhookEventListResponse struct {
	Events     []WebhookEvent `json:"events"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

// WebhookTriggerListResponse 触发器列表响应
type WebhookTriggerListResponse struct {
	Triggers   []WebhookTrigger `json:"triggers"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

// WebhookDeliveryListResponse 投递记录列表响应
type WebhookDeliveryListResponse struct {
	Deliveries []WebhookDelivery `json:"deliveries"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalPages int               `json:"total_pages"`
}

// WebhookStatistics Webhook统计信息
type WebhookStatistics struct {
	TotalEvents       int64                         `json:"total_events"`
	ProcessedEvents   int64                         `json:"processed_events"`
	FailedEvents      int64                         `json:"failed_events"`
	EventsByType      map[WebhookEventType]int64    `json:"events_by_type"`
	EventsBySource    map[string]int64              `json:"events_by_source"`
	DeliveryByStatus  map[DeliveryStatus]int64      `json:"delivery_by_status"`
	TriggersByRepo    map[uuid.UUID]int64           `json:"triggers_by_repo"`
	RecentEvents      []WebhookEvent                `json:"recent_events"`
	AverageProcessTime float64                      `json:"average_process_time"`
	SuccessRate       float64                       `json:"success_rate"`
}