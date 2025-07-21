package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RepositoryStatus 仓库状态枚举
type RepositoryStatus string

const (
	RepositoryStatusActive   RepositoryStatus = "active"
	RepositoryStatusArchived RepositoryStatus = "archived"
	RepositoryStatusDeleted  RepositoryStatus = "deleted"
)

// RepositoryVisibility 仓库可见性枚举
type RepositoryVisibility string

const (
	RepositoryVisibilityPublic   RepositoryVisibility = "public"
	RepositoryVisibilityPrivate  RepositoryVisibility = "private"
	RepositoryVisibilityInternal RepositoryVisibility = "internal"
)

// Repository Git仓库模型
type Repository struct {
	ID            uuid.UUID            `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	ProjectID     uuid.UUID            `json:"project_id" gorm:"type:uuid;not null;index"`
	Name          string               `json:"name" gorm:"size:255;not null"`
	Description   *string              `json:"description" gorm:"type:text"`
	Visibility    RepositoryVisibility `json:"visibility" gorm:"size:20;not null;default:'private'"`
	Status        RepositoryStatus     `json:"status" gorm:"size:20;not null;default:'active'"`
	DefaultBranch string               `json:"default_branch" gorm:"size:255;not null;default:'main'"`
	
	// Git配置
	GitPath       string  `json:"git_path" gorm:"size:512;not null"` // 实际Git仓库路径
	CloneURL      string  `json:"clone_url" gorm:"size:512"`         // 克隆URL
	SSHURL        string  `json:"ssh_url" gorm:"size:512"`           // SSH URL
	
	// 统计信息
	Size          int64   `json:"size" gorm:"default:0"`             // 仓库大小（字节）
	CommitCount   int64   `json:"commit_count" gorm:"default:0"`     // 提交数量
	BranchCount   int32   `json:"branch_count" gorm:"default:0"`     // 分支数量
	TagCount      int32   `json:"tag_count" gorm:"default:0"`        // 标签数量
	
	// 时间戳
	CreatedAt     time.Time  `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt     time.Time  `json:"updated_at" gorm:"not null;default:now()"`
	DeletedAt     *time.Time `json:"deleted_at" gorm:"index"`
	LastPushedAt  *time.Time `json:"last_pushed_at"`

	// 关联关系
	Project       *Project  `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	Branches      []Branch  `json:"branches,omitempty" gorm:"foreignKey:RepositoryID"`
	Commits       []Commit  `json:"commits,omitempty" gorm:"foreignKey:RepositoryID"`
	Tags          []Tag     `json:"tags,omitempty" gorm:"foreignKey:RepositoryID"`
	Webhooks      []Webhook `json:"webhooks,omitempty" gorm:"foreignKey:RepositoryID"`
}

// Branch 分支模型
type Branch struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	RepositoryID uuid.UUID  `json:"repository_id" gorm:"type:uuid;not null;index"`
	Name         string     `json:"name" gorm:"size:255;not null"`
	CommitSHA    string     `json:"commit_sha" gorm:"size:40;not null"`
	IsDefault    bool       `json:"is_default" gorm:"not null;default:false"`
	IsProtected  bool       `json:"is_protected" gorm:"not null;default:false"`
	CreatedAt    time.Time  `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt    time.Time  `json:"updated_at" gorm:"not null;default:now()"`
	DeletedAt    *time.Time `json:"deleted_at" gorm:"index"`

	// 关联关系
	Repository   *Repository `json:"repository,omitempty" gorm:"foreignKey:RepositoryID"`
	Commit       *Commit     `json:"commit,omitempty" gorm:"foreignKey:CommitSHA;references:SHA"`
}

// Commit 提交模型
type Commit struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	RepositoryID uuid.UUID  `json:"repository_id" gorm:"type:uuid;not null;index"`
	SHA          string     `json:"sha" gorm:"size:40;not null;uniqueIndex:idx_repo_commit"`
	Message      string     `json:"message" gorm:"type:text;not null"`
	Author       string     `json:"author" gorm:"size:255;not null"`
	AuthorEmail  string     `json:"author_email" gorm:"size:255;not null"`
	Committer    string     `json:"committer" gorm:"size:255;not null"`
	CommitterEmail string   `json:"committer_email" gorm:"size:255;not null"`
	
	// Git信息
	ParentSHAs   []string   `json:"parent_shas" gorm:"type:jsonb"`     // 父提交SHA列表
	TreeSHA      string     `json:"tree_sha" gorm:"size:40;not null"`  // 树对象SHA
	
	// 统计信息
	AddedLines   int32      `json:"added_lines" gorm:"default:0"`
	DeletedLines int32      `json:"deleted_lines" gorm:"default:0"`
	ChangedFiles int32      `json:"changed_files" gorm:"default:0"`
	
	// 时间戳
	CommittedAt  time.Time  `json:"committed_at" gorm:"not null"`
	CreatedAt    time.Time  `json:"created_at" gorm:"not null;default:now()"`

	// 关联关系
	Repository   *Repository `json:"repository,omitempty" gorm:"foreignKey:RepositoryID"`
	Files        []CommitFile `json:"files,omitempty" gorm:"foreignKey:CommitID"`
}

// CommitFile 提交文件变更模型
type CommitFile struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	CommitID   uuid.UUID `json:"commit_id" gorm:"type:uuid;not null;index"`
	FilePath   string    `json:"file_path" gorm:"size:1024;not null"`
	Status     string    `json:"status" gorm:"size:20;not null"` // added, modified, deleted, renamed
	OldPath    *string   `json:"old_path" gorm:"size:1024"`      // 重命名前的路径
	
	// 变更统计
	AddedLines   int32   `json:"added_lines" gorm:"default:0"`
	DeletedLines int32   `json:"deleted_lines" gorm:"default:0"`
	
	// 关联关系
	Commit       *Commit `json:"commit,omitempty" gorm:"foreignKey:CommitID"`
}

// Tag 标签模型
type Tag struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	RepositoryID uuid.UUID  `json:"repository_id" gorm:"type:uuid;not null;index"`
	Name         string     `json:"name" gorm:"size:255;not null"`
	CommitSHA    string     `json:"commit_sha" gorm:"size:40;not null"`
	Message      *string    `json:"message" gorm:"type:text"`
	Tagger       string     `json:"tagger" gorm:"size:255"`
	TaggerEmail  string     `json:"tagger_email" gorm:"size:255"`
	TaggedAt     time.Time  `json:"tagged_at" gorm:"not null"`
	CreatedAt    time.Time  `json:"created_at" gorm:"not null;default:now()"`

	// 关联关系
	Repository   *Repository `json:"repository,omitempty" gorm:"foreignKey:RepositoryID"`
	Commit       *Commit     `json:"commit,omitempty" gorm:"foreignKey:CommitSHA;references:SHA"`
}

// Webhook Git仓库Webhook模型
type Webhook struct {
	ID           uuid.UUID    `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	RepositoryID uuid.UUID    `json:"repository_id" gorm:"type:uuid;not null;index"`
	URL          string       `json:"url" gorm:"size:512;not null"`
	Secret       *string      `json:"secret" gorm:"size:255"`
	Events       []string     `json:"events" gorm:"type:jsonb;not null"` // push, pull_request, etc.
	IsActive     bool         `json:"is_active" gorm:"not null;default:true"`
	CreatedAt    time.Time    `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt    time.Time    `json:"updated_at" gorm:"not null;default:now()"`

	// 关联关系
	Repository   *Repository  `json:"repository,omitempty" gorm:"foreignKey:RepositoryID"`
}

// 引用的其他模型（简化版）
type Project struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
}

// 请求和响应模型

// CreateRepositoryRequest 创建仓库请求
type CreateRepositoryRequest struct {
	ProjectID     string               `json:"project_id" binding:"required,uuid"`
	Name          string               `json:"name" binding:"required,min=1,max=255"`
	Description   *string              `json:"description" validate:"omitempty,max=2000"`
	Visibility    RepositoryVisibility `json:"visibility" binding:"required,oneof=public private internal"`
	DefaultBranch *string              `json:"default_branch" validate:"omitempty,min=1,max=255"`
	InitReadme    bool                 `json:"init_readme"`
}

// UpdateRepositoryRequest 更新仓库请求
type UpdateRepositoryRequest struct {
	Name          *string              `json:"name" validate:"omitempty,min=1,max=255"`
	Description   *string              `json:"description" validate:"omitempty,max=2000"`
	Visibility    *RepositoryVisibility `json:"visibility" validate:"omitempty,oneof=public private internal"`
	DefaultBranch *string              `json:"default_branch" validate:"omitempty,min=1,max=255"`
}

// CreateBranchRequest 创建分支请求
type CreateBranchRequest struct {
	Name      string `json:"name" binding:"required,min=1,max=255"`
	FromSHA   string `json:"from_sha" binding:"required,len=40"`
	Protected *bool  `json:"protected"`
}

// CreateCommitRequest 创建提交请求
type CreateCommitRequest struct {
	Branch    string                 `json:"branch" binding:"required"`
	Message   string                 `json:"message" binding:"required"`
	Author    CommitAuthor           `json:"author" binding:"required"`
	Files     []CreateCommitFile     `json:"files" binding:"required,min=1"`
}

type CommitAuthor struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

type CreateCommitFile struct {
	Path    string `json:"path" binding:"required"`
	Content string `json:"content" binding:"required"`
	Mode    string `json:"mode"`
}

// CreateTagRequest 创建标签请求
type CreateTagRequest struct {
	Name      string  `json:"name" binding:"required,min=1,max=255"`
	CommitSHA string  `json:"commit_sha" binding:"required,len=40"`
	Message   *string `json:"message"`
	Tagger    CommitAuthor `json:"tagger" binding:"required"`
}

// GitDiff Git差异信息
type GitDiff struct {
	FromSHA      string      `json:"from_sha"`
	ToSHA        string      `json:"to_sha"`
	Files        []DiffFile  `json:"files"`
	TotalAdded   int32       `json:"total_added"`
	TotalDeleted int32       `json:"total_deleted"`
}

type DiffFile struct {
	Path         string `json:"path"`
	OldPath      *string `json:"old_path"`
	Status       string `json:"status"`
	AddedLines   int32  `json:"added_lines"`
	DeletedLines int32  `json:"deleted_lines"`
	Patch        string `json:"patch"`
}

// 设置表名
func (Repository) TableName() string {
	return "repositories"
}

func (Branch) TableName() string {
	return "branches"
}

func (Commit) TableName() string {
	return "commits"
}

func (CommitFile) TableName() string {
	return "commit_files"
}

func (Tag) TableName() string {
	return "tags"
}

func (Webhook) TableName() string {
	return "webhooks"
}

// GORM钩子

// BeforeCreate 创建前的处理
func (r *Repository) BeforeCreate(tx *gorm.DB) error {
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

func (b *Branch) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		var newID uuid.UUID
		err := tx.Raw("SELECT uuid_generate_v7()").Scan(&newID).Error
		if err != nil {
			return err
		}
		b.ID = newID
	}
	return nil
}

func (c *Commit) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		var newID uuid.UUID
		err := tx.Raw("SELECT uuid_generate_v7()").Scan(&newID).Error
		if err != nil {
			return err
		}
		c.ID = newID
	}
	return nil
}

func (cf *CommitFile) BeforeCreate(tx *gorm.DB) error {
	if cf.ID == uuid.Nil {
		var newID uuid.UUID
		err := tx.Raw("SELECT uuid_generate_v7()").Scan(&newID).Error
		if err != nil {
			return err
		}
		cf.ID = newID
	}
	return nil
}

func (t *Tag) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		var newID uuid.UUID
		err := tx.Raw("SELECT uuid_generate_v7()").Scan(&newID).Error
		if err != nil {
			return err
		}
		t.ID = newID
	}
	return nil
}

func (w *Webhook) BeforeCreate(tx *gorm.DB) error {
	if w.ID == uuid.Nil {
		var newID uuid.UUID
		err := tx.Raw("SELECT uuid_generate_v7()").Scan(&newID).Error
		if err != nil {
			return err
		}
		w.ID = newID
	}
	return nil
}

// BeforeUpdate 更新前的处理
func (r *Repository) BeforeUpdate(tx *gorm.DB) error {
	r.UpdatedAt = time.Now()
	return nil
}

func (b *Branch) BeforeUpdate(tx *gorm.DB) error {
	b.UpdatedAt = time.Now()
	return nil
}

func (w *Webhook) BeforeUpdate(tx *gorm.DB) error {
	w.UpdatedAt = time.Now()
	return nil
}

// 业务方法

// IsActive 检查仓库是否活跃
func (r *Repository) IsActive() bool {
	return r.Status == RepositoryStatusActive
}

// IsPublic 检查仓库是否公开
func (r *Repository) IsPublic() bool {
	return r.Visibility == RepositoryVisibilityPublic
}

// IsPrivate 检查仓库是否私有
func (r *Repository) IsPrivate() bool {
	return r.Visibility == RepositoryVisibilityPrivate
}

// GetFullName 获取仓库全名
func (r *Repository) GetFullName() string {
	if r.Project != nil {
		return r.Project.Name + "/" + r.Name
	}
	return r.Name
}

// IsDefaultBranch 检查是否为默认分支
func (b *Branch) IsDefaultBranch() bool {
	return b.IsDefault
}

// IsProtectedBranch 检查分支是否受保护
func (b *Branch) IsProtectedBranch() bool {
	return b.IsProtected
}

// GetShortSHA 获取提交的短SHA
func (c *Commit) GetShortSHA() string {
	if len(c.SHA) >= 7 {
		return c.SHA[:7]
	}
	return c.SHA
}

// GetTotalChanges 获取总变更行数
func (c *Commit) GetTotalChanges() int32 {
	return c.AddedLines + c.DeletedLines
}

// IsAnnotated 检查是否为注释标签
func (t *Tag) IsAnnotated() bool {
	return t.Message != nil && *t.Message != ""
}

// 统计数据模型

// RepositoryStats 仓库统计信息
type RepositoryStats struct {
	Size         int64      `json:"size"`
	CommitCount  int64      `json:"commit_count"`
	BranchCount  int64      `json:"branch_count"`
	TagCount     int64      `json:"tag_count"`
	LastPushedAt *time.Time `json:"last_pushed_at"`
}

// RepositoryListResponse 仓库列表响应
type RepositoryListResponse struct {
	Repositories []Repository `json:"repositories"`
	Total        int64        `json:"total"`
	Page         int          `json:"page"`
	PageSize     int          `json:"page_size"`
}

// CommitListResponse 提交列表响应
type CommitListResponse struct {
	Commits  []Commit `json:"commits"`
	Total    int64    `json:"total"`
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
}

// FileInfo 文件信息
type FileInfo struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Type     string `json:"type"` // file, directory
	Size     int64  `json:"size"`
	Mode     string `json:"mode"`
	SHA      string `json:"sha"`
}