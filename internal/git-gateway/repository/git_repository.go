package repository

import (
	"context"
	"github.com/cloud-platform/collaborative-dev/internal/git-gateway/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GitRepository Git仓库数据访问接口
type GitRepository interface {
	// 仓库管理
	CreateRepository(ctx context.Context, repo *models.Repository) error
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
	GetRepositoryByProjectAndName(ctx context.Context, projectID uuid.UUID, name string) (*models.Repository, error)
	ListRepositories(ctx context.Context, projectID *uuid.UUID, page, pageSize int) ([]models.Repository, int64, error)
	UpdateRepository(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error
	DeleteRepository(ctx context.Context, id uuid.UUID) error
	
	// 分支管理
	CreateBranch(ctx context.Context, branch *models.Branch) error
	GetBranchByName(ctx context.Context, repositoryID uuid.UUID, name string) (*models.Branch, error)
	ListBranches(ctx context.Context, repositoryID uuid.UUID) ([]models.Branch, error)
	UpdateBranch(ctx context.Context, repositoryID uuid.UUID, name string, updates map[string]interface{}) error
	DeleteBranch(ctx context.Context, repositoryID uuid.UUID, name string) error
	SetDefaultBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) error
	
	// 提交管理
	CreateCommit(ctx context.Context, commit *models.Commit) error
	GetCommitBySHA(ctx context.Context, repositoryID uuid.UUID, sha string) (*models.Commit, error)
	ListCommits(ctx context.Context, repositoryID uuid.UUID, branch string, page, pageSize int) ([]models.Commit, int64, error)
	GetCommitFiles(ctx context.Context, commitID uuid.UUID) ([]models.CommitFile, error)
	CreateCommitFiles(ctx context.Context, files []models.CommitFile) error
	
	// 标签管理
	CreateTag(ctx context.Context, tag *models.Tag) error
	GetTagByName(ctx context.Context, repositoryID uuid.UUID, name string) (*models.Tag, error)
	ListTags(ctx context.Context, repositoryID uuid.UUID) ([]models.Tag, error)
	DeleteTag(ctx context.Context, repositoryID uuid.UUID, name string) error
	
	// Webhook管理
	CreateWebhook(ctx context.Context, webhook *models.Webhook) error
	GetWebhookByID(ctx context.Context, id uuid.UUID) (*models.Webhook, error)
	ListWebhooks(ctx context.Context, repositoryID uuid.UUID) ([]models.Webhook, error)
	UpdateWebhook(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error
	DeleteWebhook(ctx context.Context, id uuid.UUID) error
	
	// 统计和查询
	GetRepositoryStats(ctx context.Context, repositoryID uuid.UUID) (*models.RepositoryStats, error)
	SearchRepositories(ctx context.Context, query string, projectID *uuid.UUID, page, pageSize int) ([]models.Repository, int64, error)
}

// gitRepository Git仓库数据访问实现
type gitRepository struct {
	db *gorm.DB
}

// NewGitRepository 创建Git仓库数据访问实例
func NewGitRepository(db *gorm.DB) GitRepository {
	return &gitRepository{
		db: db,
	}
}

// 仓库管理实现

// CreateRepository 创建仓库
func (r *gitRepository) CreateRepository(ctx context.Context, repo *models.Repository) error {
	return r.db.WithContext(ctx).Create(repo).Error
}

// GetRepositoryByID 通过ID获取仓库
func (r *gitRepository) GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error) {
	var repo models.Repository
	err := r.db.WithContext(ctx).
		Preload("Project").
		Where("id = ? AND deleted_at IS NULL", id).
		First(&repo).Error
	
	if err != nil {
		return nil, err
	}
	
	return &repo, nil
}

// GetRepositoryByProjectAndName 通过项目ID和名称获取仓库
func (r *gitRepository) GetRepositoryByProjectAndName(ctx context.Context, projectID uuid.UUID, name string) (*models.Repository, error) {
	var repo models.Repository
	err := r.db.WithContext(ctx).
		Preload("Project").
		Where("project_id = ? AND name = ? AND deleted_at IS NULL", projectID, name).
		First(&repo).Error
	
	if err != nil {
		return nil, err
	}
	
	return &repo, nil
}

// ListRepositories 获取仓库列表
func (r *gitRepository) ListRepositories(ctx context.Context, projectID *uuid.UUID, page, pageSize int) ([]models.Repository, int64, error) {
	var repos []models.Repository
	var total int64
	
	query := r.db.WithContext(ctx).Model(&models.Repository{}).Where("deleted_at IS NULL")
	
	if projectID != nil {
		query = query.Where("project_id = ?", *projectID)
	}
	
	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Preload("Project").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&repos).Error
	
	return repos, total, err
}

// UpdateRepository 更新仓库
func (r *gitRepository) UpdateRepository(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&models.Repository{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(updates).Error
}

// DeleteRepository 删除仓库（软删除）
func (r *gitRepository) DeleteRepository(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&models.Repository{}).
		Where("id = ?", id).
		Update("deleted_at", gorm.Expr("NOW()")).Error
}

// 分支管理实现

// CreateBranch 创建分支
func (r *gitRepository) CreateBranch(ctx context.Context, branch *models.Branch) error {
	return r.db.WithContext(ctx).Create(branch).Error
}

// GetBranchByName 通过名称获取分支
func (r *gitRepository) GetBranchByName(ctx context.Context, repositoryID uuid.UUID, name string) (*models.Branch, error) {
	var branch models.Branch
	err := r.db.WithContext(ctx).
		Where("repository_id = ? AND name = ? AND deleted_at IS NULL", repositoryID, name).
		First(&branch).Error
	
	if err != nil {
		return nil, err
	}
	
	return &branch, nil
}

// ListBranches 获取分支列表
func (r *gitRepository) ListBranches(ctx context.Context, repositoryID uuid.UUID) ([]models.Branch, error) {
	var branches []models.Branch
	err := r.db.WithContext(ctx).
		Where("repository_id = ? AND deleted_at IS NULL", repositoryID).
		Order("is_default DESC, name ASC").
		Find(&branches).Error
	
	return branches, err
}

// UpdateBranch 更新分支
func (r *gitRepository) UpdateBranch(ctx context.Context, repositoryID uuid.UUID, name string, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&models.Branch{}).
		Where("repository_id = ? AND name = ? AND deleted_at IS NULL", repositoryID, name).
		Updates(updates).Error
}

// DeleteBranch 删除分支（软删除）
func (r *gitRepository) DeleteBranch(ctx context.Context, repositoryID uuid.UUID, name string) error {
	return r.db.WithContext(ctx).
		Model(&models.Branch{}).
		Where("repository_id = ? AND name = ?", repositoryID, name).
		Update("deleted_at", gorm.Expr("NOW()")).Error
}

// SetDefaultBranch 设置默认分支
func (r *gitRepository) SetDefaultBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) error {
	// 开启事务
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 清除当前默认分支
		if err := tx.Model(&models.Branch{}).
			Where("repository_id = ? AND is_default = true", repositoryID).
			Update("is_default", false).Error; err != nil {
			return err
		}
		
		// 设置新默认分支
		if err := tx.Model(&models.Branch{}).
			Where("repository_id = ? AND name = ?", repositoryID, branchName).
			Update("is_default", true).Error; err != nil {
			return err
		}
		
		// 更新仓库默认分支
		return tx.Model(&models.Repository{}).
			Where("id = ?", repositoryID).
			Update("default_branch", branchName).Error
	})
}

// 提交管理实现

// CreateCommit 创建提交
func (r *gitRepository) CreateCommit(ctx context.Context, commit *models.Commit) error {
	return r.db.WithContext(ctx).Create(commit).Error
}

// GetCommitBySHA 通过SHA获取提交
func (r *gitRepository) GetCommitBySHA(ctx context.Context, repositoryID uuid.UUID, sha string) (*models.Commit, error) {
	var commit models.Commit
	err := r.db.WithContext(ctx).
		Preload("Files").
		Where("repository_id = ? AND sha = ?", repositoryID, sha).
		First(&commit).Error
	
	if err != nil {
		return nil, err
	}
	
	return &commit, nil
}

// ListCommits 获取提交列表
func (r *gitRepository) ListCommits(ctx context.Context, repositoryID uuid.UUID, branch string, page, pageSize int) ([]models.Commit, int64, error) {
	var commits []models.Commit
	var total int64
	
	query := r.db.WithContext(ctx).Model(&models.Commit{}).Where("repository_id = ?", repositoryID)
	
	// 如果指定分支，需要额外的逻辑来获取该分支的提交
	// 这里简化处理，实际需要根据Git历史来过滤
	
	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Order("committed_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&commits).Error
	
	return commits, total, err
}

// GetCommitFiles 获取提交文件变更
func (r *gitRepository) GetCommitFiles(ctx context.Context, commitID uuid.UUID) ([]models.CommitFile, error) {
	var files []models.CommitFile
	err := r.db.WithContext(ctx).
		Where("commit_id = ?", commitID).
		Find(&files).Error
	
	return files, err
}

// CreateCommitFiles 创建提交文件变更
func (r *gitRepository) CreateCommitFiles(ctx context.Context, files []models.CommitFile) error {
	if len(files) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&files).Error
}

// 标签管理实现

// CreateTag 创建标签
func (r *gitRepository) CreateTag(ctx context.Context, tag *models.Tag) error {
	return r.db.WithContext(ctx).Create(tag).Error
}

// GetTagByName 通过名称获取标签
func (r *gitRepository) GetTagByName(ctx context.Context, repositoryID uuid.UUID, name string) (*models.Tag, error) {
	var tag models.Tag
	err := r.db.WithContext(ctx).
		Where("repository_id = ? AND name = ?", repositoryID, name).
		First(&tag).Error
	
	if err != nil {
		return nil, err
	}
	
	return &tag, nil
}

// ListTags 获取标签列表
func (r *gitRepository) ListTags(ctx context.Context, repositoryID uuid.UUID) ([]models.Tag, error) {
	var tags []models.Tag
	err := r.db.WithContext(ctx).
		Where("repository_id = ?", repositoryID).
		Order("tagged_at DESC").
		Find(&tags).Error
	
	return tags, err
}

// DeleteTag 删除标签
func (r *gitRepository) DeleteTag(ctx context.Context, repositoryID uuid.UUID, name string) error {
	return r.db.WithContext(ctx).
		Where("repository_id = ? AND name = ?", repositoryID, name).
		Delete(&models.Tag{}).Error
}

// Webhook管理实现

// CreateWebhook 创建Webhook
func (r *gitRepository) CreateWebhook(ctx context.Context, webhook *models.Webhook) error {
	return r.db.WithContext(ctx).Create(webhook).Error
}

// GetWebhookByID 通过ID获取Webhook
func (r *gitRepository) GetWebhookByID(ctx context.Context, id uuid.UUID) (*models.Webhook, error) {
	var webhook models.Webhook
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&webhook).Error
	
	if err != nil {
		return nil, err
	}
	
	return &webhook, nil
}

// ListWebhooks 获取Webhook列表
func (r *gitRepository) ListWebhooks(ctx context.Context, repositoryID uuid.UUID) ([]models.Webhook, error) {
	var webhooks []models.Webhook
	err := r.db.WithContext(ctx).
		Where("repository_id = ?", repositoryID).
		Order("created_at DESC").
		Find(&webhooks).Error
	
	return webhooks, err
}

// UpdateWebhook 更新Webhook
func (r *gitRepository) UpdateWebhook(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&models.Webhook{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// DeleteWebhook 删除Webhook
func (r *gitRepository) DeleteWebhook(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&models.Webhook{}).Error
}

// 统计和查询实现

// GetRepositoryStats 获取仓库统计信息
func (r *gitRepository) GetRepositoryStats(ctx context.Context, repositoryID uuid.UUID) (*models.RepositoryStats, error) {
	var stats models.RepositoryStats
	
	// 获取分支数量
	if err := r.db.WithContext(ctx).
		Model(&models.Branch{}).
		Where("repository_id = ? AND deleted_at IS NULL", repositoryID).
		Count(&stats.BranchCount).Error; err != nil {
		return nil, err
	}
	
	// 获取提交数量
	if err := r.db.WithContext(ctx).
		Model(&models.Commit{}).
		Where("repository_id = ?", repositoryID).
		Count(&stats.CommitCount).Error; err != nil {
		return nil, err
	}
	
	// 获取标签数量
	if err := r.db.WithContext(ctx).
		Model(&models.Tag{}).
		Where("repository_id = ?", repositoryID).
		Count(&stats.TagCount).Error; err != nil {
		return nil, err
	}
	
	// 获取仓库大小等其他信息
	var repo models.Repository
	if err := r.db.WithContext(ctx).
		Select("size, last_pushed_at").
		Where("id = ?", repositoryID).
		First(&repo).Error; err != nil {
		return nil, err
	}
	
	stats.Size = repo.Size
	stats.LastPushedAt = repo.LastPushedAt
	
	return &stats, nil
}

// SearchRepositories 搜索仓库
func (r *gitRepository) SearchRepositories(ctx context.Context, query string, projectID *uuid.UUID, page, pageSize int) ([]models.Repository, int64, error) {
	var repos []models.Repository
	var total int64
	
	dbQuery := r.db.WithContext(ctx).Model(&models.Repository{}).Where("deleted_at IS NULL")
	
	if projectID != nil {
		dbQuery = dbQuery.Where("project_id = ?", *projectID)
	}
	
	if query != "" {
		searchPattern := "%" + query + "%"
		dbQuery = dbQuery.Where("name ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)
	}
	
	// 获取总数
	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	// 分页查询
	offset := (page - 1) * pageSize
	err := dbQuery.
		Preload("Project").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&repos).Error
	
	return repos, total, err
}