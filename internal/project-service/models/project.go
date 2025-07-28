package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Project 项目模型
type Project struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID    uuid.UUID  `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Key         string     `json:"key" gorm:"size:20;not null"`
	Name        string     `json:"name" gorm:"size:255;not null"`
	Description *string    `json:"description" gorm:"type:text"`
	ManagerID   *uuid.UUID `json:"manager_id" gorm:"type:uuid"`
	Status      string     `json:"status" gorm:"size:20;not null;default:'active'"`
	CreatedAt   time.Time  `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"not null;default:now()"`
	DeletedAt   *time.Time `json:"deleted_at" gorm:"index"`

	// 关联关系
	Tenant       *Tenant         `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
	Manager      *User           `json:"manager,omitempty" gorm:"foreignKey:ManagerID"`
	Members      []ProjectMember `json:"members,omitempty" gorm:"foreignKey:ProjectID"`
	Tasks        []Task          `json:"tasks,omitempty" gorm:"foreignKey:ProjectID"`
	Repositories []Repository    `json:"repositories,omitempty" gorm:"foreignKey:ProjectID"`
}

// ProjectMember 项目成员模型
type ProjectMember struct {
	ProjectID uuid.UUID  `json:"project_id" gorm:"type:uuid;primary_key"`
	UserID    uuid.UUID  `json:"user_id" gorm:"type:uuid;primary_key"`
	RoleID    uuid.UUID  `json:"role_id" gorm:"type:uuid;not null"`
	AddedAt   time.Time  `json:"added_at" gorm:"not null;default:now()"`
	AddedBy   *uuid.UUID `json:"added_by" gorm:"type:uuid"`

	// 关联关系
	Project     *Project `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	User        *User    `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Role        *Role    `json:"role,omitempty" gorm:"foreignKey:RoleID"`
	AddedByUser *User    `json:"added_by_user,omitempty" gorm:"foreignKey:AddedBy"`
}

// Tenant 租户模型（引用）
type Tenant struct {
	ID     uuid.UUID `json:"id" gorm:"type:uuid;primary_key"`
	Name   string    `json:"name"`
	Slug   string    `json:"slug"`
	Status string    `json:"status"`
}

// User 用户模型（引用）
type User struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	DisplayName *string   `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url"`
	Status      string    `json:"status"`
}

// Role 角色模型（引用）
type Role struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	Scope       string    `json:"scope"`
	Permissions []string  `json:"permissions" gorm:"type:jsonb"`
}

// Task 任务模型（引用）
type Task struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
	ProjectID   uuid.UUID  `json:"project_id"`
	TaskNumber  int64      `json:"task_number"`
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	StatusID    *uuid.UUID `json:"status_id"`
	AssigneeID  *uuid.UUID `json:"assignee_id"`
	CreatorID   uuid.UUID  `json:"creator_id"`
}

// Repository 仓库模型（引用）
type Repository struct {
	ID            uuid.UUID `json:"id" gorm:"type:uuid;primary_key"`
	ProjectID     uuid.UUID `json:"project_id"`
	Name          string    `json:"name"`
	Description   *string   `json:"description"`
	Visibility    string    `json:"visibility"`
	DefaultBranch string    `json:"default_branch"`
}

// Branch 分支模型
type Branch struct {
	ID           uuid.UUID  `json:"id"`
	RepositoryID uuid.UUID  `json:"repository_id"`
	Name         string     `json:"name"`
	CommitSHA    string     `json:"commit_sha"`
	IsDefault    bool       `json:"is_default"`
	IsProtected  bool       `json:"is_protected"`
	CreatedBy    uuid.UUID  `json:"created_by"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

// PullRequest 合并请求模型
type PullRequest struct {
	ID           uuid.UUID   `json:"id"`
	RepositoryID uuid.UUID   `json:"repository_id"`
	Number       int         `json:"number"`
	Title        string      `json:"title"`
	Description  *string     `json:"description,omitempty"`
	Status       string      `json:"status"`
	SourceBranch string      `json:"source_branch"`
	TargetBranch string      `json:"target_branch"`
	AuthorID     uuid.UUID   `json:"author_id"`
	AssigneeIDs  []uuid.UUID `json:"assignee_ids,omitempty"`
	ReviewerIDs  []uuid.UUID `json:"reviewer_ids,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
	MergedAt     *time.Time  `json:"merged_at,omitempty"`
	ClosedAt     *time.Time  `json:"closed_at,omitempty"`
}

// Commit 提交模型
type Commit struct {
	ID           uuid.UUID  `json:"id"`
	RepositoryID uuid.UUID  `json:"repository_id"`
	SHA          string     `json:"sha"`
	Message      string     `json:"message"`
	AuthorName   string     `json:"author_name"`
	AuthorEmail  string     `json:"author_email"`
	AuthorID     *uuid.UUID `json:"author_id,omitempty"`
	CommittedAt  time.Time  `json:"committed_at"`
	ParentSHAs   []string   `json:"parent_shas,omitempty"`
}

// WebhookEvent Webhook事件模型
type WebhookEvent struct {
	ID           uuid.UUID              `json:"id"`
	RepositoryID uuid.UUID              `json:"repository_id"`
	EventType    string                 `json:"event_type"`
	EventData    map[string]interface{} `json:"event_data"`
	ProcessedAt  *time.Time             `json:"processed_at,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// CreateProjectRequest 创建项目请求
type CreateProjectRequest struct {
	Key         string  `json:"key" binding:"required,min=2,max=20" validate:"alphanum"`
	Name        string  `json:"name" binding:"required,min=1,max=255"`
	Description *string `json:"description" validate:"max=2000"`
	ManagerID   *string `json:"manager_id" validate:"omitempty,uuid"`
}

// UpdateProjectRequest 更新项目请求
type UpdateProjectRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=1,max=255"`
	Description *string `json:"description" validate:"omitempty,max=2000"`
	ManagerID   *string `json:"manager_id" validate:"omitempty,uuid"`
	Status      *string `json:"status" validate:"omitempty,oneof=active archived"`
}

// ProjectListResponse 项目列表响应
type ProjectListResponse struct {
	Projects []Project `json:"projects"`
	Total    int64     `json:"total"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
}

// AddMemberRequest 添加成员请求
type AddMemberRequest struct {
	UserID string `json:"user_id" binding:"required,uuid"`
	RoleID string `json:"role_id" binding:"required,uuid"`
}

// TableName 设置表名
func (Project) TableName() string {
	return "projects"
}

func (ProjectMember) TableName() string {
	return "project_members"
}

// BeforeCreate GORM钩子：创建前的处理
func (p *Project) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		// 在代码中生成UUID，避免数据库兼容性问题
		p.ID = uuid.New()
	}
	return nil
}

// BeforeUpdate GORM钩子：更新前的处理
func (p *Project) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = time.Now()
	return nil
}

// IsActive 检查项目是否为活动状态
func (p *Project) IsActive() bool {
	return p.Status == "active" && p.DeletedAt == nil
}

// IsArchived 检查项目是否为归档状态
func (p *Project) IsArchived() bool {
	return p.Status == "archived"
}

// HasMember 检查用户是否为项目成员
func (p *Project) HasMember(userID uuid.UUID) bool {
	for _, member := range p.Members {
		if member.UserID == userID {
			return true
		}
	}
	return false
}

// GetMemberRole 获取用户在项目中的角色
func (p *Project) GetMemberRole(userID uuid.UUID) *Role {
	for _, member := range p.Members {
		if member.UserID == userID && member.Role != nil {
			return member.Role
		}
	}
	return nil
}
