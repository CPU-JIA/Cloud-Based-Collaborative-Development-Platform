package client

import (
	"time"

	"github.com/google/uuid"
)

// 这些模型用于与Git网关服务通信，映射Git网关的数据结构

// Repository Git仓库模型（客户端版本）
type Repository struct {
	ID            uuid.UUID            `json:"id"`
	ProjectID     uuid.UUID            `json:"project_id"`
	Name          string               `json:"name"`
	Description   *string              `json:"description"`
	Visibility    RepositoryVisibility `json:"visibility"`
	Status        RepositoryStatus     `json:"status"`
	DefaultBranch string               `json:"default_branch"`
	
	// Git配置
	GitPath       string  `json:"git_path"`
	CloneURL      string  `json:"clone_url"`
	SSHURL        string  `json:"ssh_url"`
	
	// 统计信息
	Size          int64   `json:"size"`
	CommitCount   int64   `json:"commit_count"`
	BranchCount   int32   `json:"branch_count"`
	TagCount      int32   `json:"tag_count"`
	
	// 时间戳
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at"`
	LastPushedAt  *time.Time `json:"last_pushed_at"`
}

// Branch 分支模型
type Branch struct {
	ID           uuid.UUID  `json:"id"`
	RepositoryID uuid.UUID  `json:"repository_id"`
	Name         string     `json:"name"`
	CommitSHA    string     `json:"commit_sha"`
	IsDefault    bool       `json:"is_default"`
	IsProtected  bool       `json:"is_protected"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at"`
}

// Commit 提交模型
type Commit struct {
	ID             uuid.UUID  `json:"id"`
	RepositoryID   uuid.UUID  `json:"repository_id"`
	SHA            string     `json:"sha"`
	Message        string     `json:"message"`
	Author         string     `json:"author"`
	AuthorEmail    string     `json:"author_email"`
	Committer      string     `json:"committer"`
	CommitterEmail string     `json:"committer_email"`
	
	// Git信息
	ParentSHAs     []string   `json:"parent_shas"`
	TreeSHA        string     `json:"tree_sha"`
	
	// 统计信息
	AddedLines     int32      `json:"added_lines"`
	DeletedLines   int32      `json:"deleted_lines"`
	ChangedFiles   int32      `json:"changed_files"`
	
	// 时间戳
	CommittedAt    time.Time  `json:"committed_at"`
	CreatedAt      time.Time  `json:"created_at"`
	
	// 关联文件（可选）
	Files          []CommitFile `json:"files,omitempty"`
}

// CommitFile 提交文件变更模型
type CommitFile struct {
	ID           uuid.UUID `json:"id"`
	CommitID     uuid.UUID `json:"commit_id"`
	FilePath     string    `json:"file_path"`
	Status       string    `json:"status"`
	OldPath      *string   `json:"old_path"`
	AddedLines   int32     `json:"added_lines"`
	DeletedLines int32     `json:"deleted_lines"`
}

// Tag 标签模型
type Tag struct {
	ID           uuid.UUID  `json:"id"`
	RepositoryID uuid.UUID  `json:"repository_id"`
	Name         string     `json:"name"`
	CommitSHA    string     `json:"commit_sha"`
	Message      *string    `json:"message"`
	Tagger       string     `json:"tagger"`
	TaggerEmail  string     `json:"tagger_email"`
	TaggedAt     time.Time  `json:"tagged_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

// FileInfo 文件信息
type FileInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"` // file, directory
	Size int64  `json:"size"`
	Mode string `json:"mode"`
	SHA  string `json:"sha"`
}

// GitDiff Git差异信息
type GitDiff struct {
	FromSHA      string     `json:"from_sha"`
	ToSHA        string     `json:"to_sha"`
	Files        []DiffFile `json:"files"`
	TotalAdded   int32      `json:"total_added"`
	TotalDeleted int32      `json:"total_deleted"`
}

// DiffFile 差异文件
type DiffFile struct {
	Path         string  `json:"path"`
	OldPath      *string `json:"old_path"`
	Status       string  `json:"status"`
	AddedLines   int32   `json:"added_lines"`
	DeletedLines int32   `json:"deleted_lines"`
	Patch        string  `json:"patch"`
}

// RepositoryStats 仓库统计信息
type RepositoryStats struct {
	Size         int64      `json:"size"`
	CommitCount  int64      `json:"commit_count"`
	BranchCount  int64      `json:"branch_count"`
	TagCount     int64      `json:"tag_count"`
	LastPushedAt *time.Time `json:"last_pushed_at"`
}

// 枚举类型

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

// 请求模型

// CreateRepositoryRequest 创建仓库请求
type CreateRepositoryRequest struct {
	ProjectID     string               `json:"project_id"`
	Name          string               `json:"name"`
	Description   *string              `json:"description,omitempty"`
	Visibility    RepositoryVisibility `json:"visibility"`
	DefaultBranch *string              `json:"default_branch,omitempty"`
	InitReadme    bool                 `json:"init_readme"`
}

// UpdateRepositoryRequest 更新仓库请求
type UpdateRepositoryRequest struct {
	Name          *string               `json:"name,omitempty"`
	Description   *string               `json:"description,omitempty"`
	Visibility    *RepositoryVisibility `json:"visibility,omitempty"`
	DefaultBranch *string               `json:"default_branch,omitempty"`
}

// CreateBranchRequest 创建分支请求
type CreateBranchRequest struct {
	Name      string `json:"name"`
	FromSHA   string `json:"from_sha"`
	Protected *bool  `json:"protected,omitempty"`
}

// CreateCommitRequest 创建提交请求
type CreateCommitRequest struct {
	Branch  string             `json:"branch"`
	Message string             `json:"message"`
	Author  CommitAuthor       `json:"author"`
	Files   []CreateCommitFile `json:"files"`
}

// CommitAuthor 提交作者
type CommitAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// CreateCommitFile 创建提交文件
type CreateCommitFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Mode    string `json:"mode,omitempty"`
}

// CreateTagRequest 创建标签请求
type CreateTagRequest struct {
	Name      string       `json:"name"`
	CommitSHA string       `json:"commit_sha"`
	Message   *string      `json:"message,omitempty"`
	Tagger    CommitAuthor `json:"tagger"`
}

// 响应模型

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

// 工具方法

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