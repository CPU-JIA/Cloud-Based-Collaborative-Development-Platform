package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/git-gateway/models"
	"github.com/cloud-platform/collaborative-dev/internal/git-gateway/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// GitCommitInfo Git提交详细信息
type GitCommitInfo struct {
	SHA            string    `json:"sha"`
	Message        string    `json:"message"`
	Author         string    `json:"author"`
	AuthorEmail    string    `json:"author_email"`
	Committer      string    `json:"committer"`
	CommitterEmail string    `json:"committer_email"`
	CommittedAt    time.Time `json:"committed_at"`
	AddedLines     int       `json:"added_lines"`
	DeletedLines   int       `json:"deleted_lines"`
	ParentSHAs     []string  `json:"parent_shas"`
	TreeSHA        string    `json:"tree_sha"`
}

// FileChange 文件变更信息
type FileChange struct {
	Path         string `json:"path"`
	OldPath      string `json:"old_path,omitempty"`
	Status       string `json:"status"`
	AddedLines   int    `json:"added_lines"`
	DeletedLines int    `json:"deleted_lines"`
}

// GitService Git服务接口
type GitService interface {
	// 仓库管理
	CreateRepository(ctx context.Context, req *models.CreateRepositoryRequest, userID uuid.UUID) (*models.Repository, error)
	GetRepository(ctx context.Context, id uuid.UUID) (*models.Repository, error)
	ListRepositories(ctx context.Context, projectID *uuid.UUID, page, pageSize int) (*models.RepositoryListResponse, error)
	UpdateRepository(ctx context.Context, id uuid.UUID, req *models.UpdateRepositoryRequest) (*models.Repository, error)
	DeleteRepository(ctx context.Context, id uuid.UUID) error
	
	// 分支管理
	CreateBranch(ctx context.Context, repositoryID uuid.UUID, req *models.CreateBranchRequest) (*models.Branch, error)
	GetBranch(ctx context.Context, repositoryID uuid.UUID, name string) (*models.Branch, error)
	ListBranches(ctx context.Context, repositoryID uuid.UUID) ([]models.Branch, error)
	DeleteBranch(ctx context.Context, repositoryID uuid.UUID, name string) error
	SetDefaultBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) error
	MergeBranch(ctx context.Context, repositoryID uuid.UUID, targetBranch, sourceBranch string) error
	
	// 提交管理
	CreateCommit(ctx context.Context, repositoryID uuid.UUID, req *models.CreateCommitRequest) (*models.Commit, error)
	GetCommit(ctx context.Context, repositoryID uuid.UUID, sha string) (*models.Commit, error)
	ListCommits(ctx context.Context, repositoryID uuid.UUID, branch string, page, pageSize int) (*models.CommitListResponse, error)
	GetCommitDiff(ctx context.Context, repositoryID uuid.UUID, sha string) (*models.GitDiff, error)
	CompareBranches(ctx context.Context, repositoryID uuid.UUID, base, head string) (*models.GitDiff, error)
	
	// 标签管理
	CreateTag(ctx context.Context, repositoryID uuid.UUID, req *models.CreateTagRequest) (*models.Tag, error)
	GetTag(ctx context.Context, repositoryID uuid.UUID, name string) (*models.Tag, error)
	ListTags(ctx context.Context, repositoryID uuid.UUID) ([]models.Tag, error)
	DeleteTag(ctx context.Context, repositoryID uuid.UUID, name string) error
	
	// 文件操作
	GetFileContent(ctx context.Context, repositoryID uuid.UUID, branch, filePath string) ([]byte, error)
	GetDirectoryContent(ctx context.Context, repositoryID uuid.UUID, branch, dirPath string) ([]models.FileInfo, error)
	
	// 统计和搜索
	GetRepositoryStats(ctx context.Context, repositoryID uuid.UUID) (*models.RepositoryStats, error)
	SearchRepositories(ctx context.Context, query string, projectID *uuid.UUID, page, pageSize int) (*models.RepositoryListResponse, error)
}

// gitService Git服务实现
type gitService struct {
	repo      repository.GitRepository
	logger    *zap.Logger
	gitRoot   string // Git仓库根目录
}

// NewGitService 创建Git服务实例
func NewGitService(repo repository.GitRepository, logger *zap.Logger, gitRoot string) GitService {
	return &gitService{
		repo:    repo,
		logger:  logger,
		gitRoot: gitRoot,
	}
}

// 仓库管理实现

// CreateRepository 创建仓库
func (s *gitService) CreateRepository(ctx context.Context, req *models.CreateRepositoryRequest, userID uuid.UUID) (*models.Repository, error) {
	// 检查仓库名称是否已存在
	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}
	
	// 检查仓库是否已存在 - 实现幂等性
	existing, err := s.repo.GetRepositoryByProjectAndName(ctx, projectID, req.Name)
	if err == nil && existing != nil {
		// 幂等性：如果仓库已存在且配置匹配，直接返回已有仓库
		if s.isRepositoryConfigMatching(existing, req) {
			s.logger.Info("Repository already exists, returning existing repository", 
				zap.String("repository_id", existing.ID.String()),
				zap.String("name", req.Name))
			return existing, nil
		}
		// 如果配置不匹配，返回配置冲突错误
		return nil, fmt.Errorf("repository name '%s' already exists with different configuration", req.Name)
	}
	
	// 创建Git仓库路径
	repoPath := filepath.Join(s.gitRoot, projectID.String(), req.Name+".git")
	
	// 创建数据库记录
	defaultBranch := "main"
	if req.DefaultBranch != nil {
		defaultBranch = *req.DefaultBranch
	}
	
	repository := &models.Repository{
		ProjectID:     projectID,
		Name:          req.Name,
		Description:   req.Description,
		Visibility:    req.Visibility,
		Status:        models.RepositoryStatusActive,
		DefaultBranch: defaultBranch,
		GitPath:       repoPath,
		CloneURL:      s.generateCloneURL(projectID, req.Name),
		SSHURL:        s.generateSSHURL(projectID, req.Name),
		Size:          0,
		CommitCount:   0,
		BranchCount:   1, // 默认分支
		TagCount:      0,
	}
	
	if err := s.repo.CreateRepository(ctx, repository); err != nil {
		return nil, fmt.Errorf("failed to create repository record: %w", err)
	}
	
	// 创建物理Git仓库
	if err := s.initGitRepository(repoPath, defaultBranch, req.InitReadme); err != nil {
		// 回滚数据库记录
		s.repo.DeleteRepository(ctx, repository.ID)
		return nil, fmt.Errorf("failed to initialize git repository: %w", err)
	}
	
	// 创建默认分支记录
	if err := s.createDefaultBranch(ctx, repository.ID, defaultBranch); err != nil {
		s.logger.Error("Failed to create default branch record", zap.Error(err))
	}
	
	s.logger.Info("Repository created successfully", 
		zap.String("repository_id", repository.ID.String()),
		zap.String("name", req.Name))
	
	return repository, nil
}

// GetRepository 获取仓库信息
func (s *gitService) GetRepository(ctx context.Context, id uuid.UUID) (*models.Repository, error) {
	return s.repo.GetRepositoryByID(ctx, id)
}

// ListRepositories 获取仓库列表
func (s *gitService) ListRepositories(ctx context.Context, projectID *uuid.UUID, page, pageSize int) (*models.RepositoryListResponse, error) {
	repos, total, err := s.repo.ListRepositories(ctx, projectID, page, pageSize)
	if err != nil {
		return nil, err
	}
	
	return &models.RepositoryListResponse{
		Repositories: repos,
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
	}, nil
}

// UpdateRepository 更新仓库
func (s *gitService) UpdateRepository(ctx context.Context, id uuid.UUID, req *models.UpdateRepositoryRequest) (*models.Repository, error) {
	// 检查仓库是否存在
	_, err := s.repo.GetRepositoryByID(ctx, id)
	if err != nil {
		return nil, err
	}
	
	updates := make(map[string]interface{})
	
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Visibility != nil {
		updates["visibility"] = *req.Visibility
	}
	if req.DefaultBranch != nil {
		updates["default_branch"] = *req.DefaultBranch
		// TODO: 更新Git仓库的默认分支
	}
	
	if len(updates) > 0 {
		if err := s.repo.UpdateRepository(ctx, id, updates); err != nil {
			return nil, err
		}
	}
	
	return s.repo.GetRepositoryByID(ctx, id)
}

// DeleteRepository 删除仓库
func (s *gitService) DeleteRepository(ctx context.Context, id uuid.UUID) error {
	repo, err := s.repo.GetRepositoryByID(ctx, id)
	if err != nil {
		return err
	}
	
	// 软删除数据库记录
	if err := s.repo.DeleteRepository(ctx, id); err != nil {
		return err
	}
	
	// TODO: 异步删除物理Git仓库文件
	go func() {
		if err := os.RemoveAll(repo.GitPath); err != nil {
			s.logger.Error("Failed to delete git repository files", 
				zap.String("repository_id", id.String()),
				zap.String("git_path", repo.GitPath),
				zap.Error(err))
		}
	}()
	
	s.logger.Info("Repository deleted", zap.String("repository_id", id.String()))
	return nil
}

// 分支管理实现

// CreateBranch 创建分支
func (s *gitService) CreateBranch(ctx context.Context, repositoryID uuid.UUID, req *models.CreateBranchRequest) (*models.Branch, error) {
	repo, err := s.repo.GetRepositoryByID(ctx, repositoryID)
	if err != nil {
		return nil, err
	}
	
	// 在Git仓库中创建分支
	if err := s.createGitBranch(repo.GitPath, req.Name, req.FromSHA); err != nil {
		return nil, fmt.Errorf("failed to create git branch: %w", err)
	}
	
	// 创建分支记录
	branch := &models.Branch{
		RepositoryID: repositoryID,
		Name:         req.Name,
		CommitSHA:    req.FromSHA,
		IsDefault:    false,
		IsProtected:  req.Protected != nil && *req.Protected,
	}
	
	if err := s.repo.CreateBranch(ctx, branch); err != nil {
		return nil, err
	}
	
	// 更新仓库分支计数
	s.updateRepositoryBranchCount(ctx, repositoryID)
	
	return branch, nil
}

// GetBranch 获取分支信息
func (s *gitService) GetBranch(ctx context.Context, repositoryID uuid.UUID, name string) (*models.Branch, error) {
	return s.repo.GetBranchByName(ctx, repositoryID, name)
}

// ListBranches 获取分支列表
func (s *gitService) ListBranches(ctx context.Context, repositoryID uuid.UUID) ([]models.Branch, error) {
	return s.repo.ListBranches(ctx, repositoryID)
}

// DeleteBranch 删除分支
func (s *gitService) DeleteBranch(ctx context.Context, repositoryID uuid.UUID, name string) error {
	// 检查是否为默认分支
	branch, err := s.repo.GetBranchByName(ctx, repositoryID, name)
	if err != nil {
		return err
	}
	
	if branch.IsDefault {
		return fmt.Errorf("cannot delete default branch")
	}
	
	// 从Git仓库删除分支
	repo, err := s.repo.GetRepositoryByID(ctx, repositoryID)
	if err != nil {
		return err
	}
	
	if err := s.deleteGitBranch(repo.GitPath, name); err != nil {
		return fmt.Errorf("failed to delete git branch: %w", err)
	}
	
	// 删除分支记录
	if err := s.repo.DeleteBranch(ctx, repositoryID, name); err != nil {
		return err
	}
	
	// 更新仓库分支计数
	s.updateRepositoryBranchCount(ctx, repositoryID)
	
	return nil
}

// SetDefaultBranch 设置默认分支
func (s *gitService) SetDefaultBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) error {
	return s.repo.SetDefaultBranch(ctx, repositoryID, branchName)
}

// MergeBranch 合并分支
func (s *gitService) MergeBranch(ctx context.Context, repositoryID uuid.UUID, targetBranch, sourceBranch string) error {
	repo, err := s.repo.GetRepositoryByID(ctx, repositoryID)
	if err != nil {
		return err
	}
	
	// 检查源分支和目标分支是否存在
	if _, err := s.repo.GetBranchByName(ctx, repositoryID, sourceBranch); err != nil {
		return fmt.Errorf("source branch '%s' not found", sourceBranch)
	}
	
	if _, err := s.repo.GetBranchByName(ctx, repositoryID, targetBranch); err != nil {
		return fmt.Errorf("target branch '%s' not found", targetBranch)
	}
	
	// 执行Git合并
	if err := s.mergeBranch(repo.GitPath, targetBranch, sourceBranch); err != nil {
		return fmt.Errorf("failed to merge branches: %w", err)
	}
	
	// 获取合并后的提交SHA
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repo.GitPath
	output, err := cmd.Output()
	if err != nil {
		s.logger.Error("Failed to get merge commit SHA", zap.Error(err))
	} else {
		commitSHA := fmt.Sprintf("%.40s", string(output))
		
		// 更新目标分支的提交SHA
		s.repo.UpdateBranch(ctx, repositoryID, targetBranch, map[string]interface{}{
			"commit_sha": commitSHA,
		})
	}
	
	s.logger.Info("Branch merged successfully", 
		zap.String("repository_id", repositoryID.String()),
		zap.String("target_branch", targetBranch),
		zap.String("source_branch", sourceBranch))
	
	return nil
}

// 提交管理实现

// CreateCommit 创建提交
func (s *gitService) CreateCommit(ctx context.Context, repositoryID uuid.UUID, req *models.CreateCommitRequest) (*models.Commit, error) {
	repo, err := s.repo.GetRepositoryByID(ctx, repositoryID)
	if err != nil {
		return nil, err
	}
	
	// 在Git仓库中创建提交
	commitSHA, err := s.createGitCommit(repo.GitPath, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create git commit: %w", err)
	}
	
	// 从Git获取提交详细信息
	gitCommit, err := s.getGitCommitInfo(repo.GitPath, commitSHA)
	if err != nil {
		s.logger.Error("Failed to get git commit info", zap.Error(err))
		gitCommit = &GitCommitInfo{
			SHA:            commitSHA,
			Message:        req.Message,
			Author:         req.Author.Name,
			AuthorEmail:    req.Author.Email,
			Committer:      req.Author.Name,
			CommitterEmail: req.Author.Email,
			CommittedAt:    time.Now(),
			AddedLines:     0,
			DeletedLines:   0,
			ParentSHAs:     []string{},
		}
	}
	
	// 创建提交记录
	commit := &models.Commit{
		RepositoryID:    repositoryID,
		SHA:             commitSHA,
		Message:         gitCommit.Message,
		Author:          gitCommit.Author,
		AuthorEmail:     gitCommit.AuthorEmail,
		Committer:       gitCommit.Committer,
		CommitterEmail:  gitCommit.CommitterEmail,
		CommittedAt:     gitCommit.CommittedAt,
		AddedLines:      int32(gitCommit.AddedLines),
		DeletedLines:    int32(gitCommit.DeletedLines),
		ChangedFiles:    int32(len(req.Files)),
	}
	
	if err := s.repo.CreateCommit(ctx, commit); err != nil {
		return nil, err
	}
	
	// 获取文件变更详情并创建记录
	fileChanges, err := s.getCommitFileChanges(repo.GitPath, commitSHA)
	if err != nil {
		s.logger.Error("Failed to get commit file changes", zap.Error(err))
		fileChanges = []FileChange{}
	}
	
	var commitFiles []models.CommitFile
	for _, change := range fileChanges {
		var oldPath *string
		if change.OldPath != "" {
			oldPath = &change.OldPath
		}
		
		commitFiles = append(commitFiles, models.CommitFile{
			CommitID:     commit.ID,
			FilePath:     change.Path,
			Status:       change.Status,
			OldPath:      oldPath,
			AddedLines:   int32(change.AddedLines),
			DeletedLines: int32(change.DeletedLines),
		})
	}
	
	if len(commitFiles) > 0 {
		if err := s.repo.CreateCommitFiles(ctx, commitFiles); err != nil {
			s.logger.Error("Failed to create commit files", zap.Error(err))
		}
	}
	
	// 更新分支指针
	if req.Branch != "" {
		s.repo.UpdateBranch(ctx, repositoryID, req.Branch, map[string]interface{}{
			"commit_sha": commitSHA,
		})
	}
	
	// 更新仓库统计
	s.updateRepositoryCommitCount(ctx, repositoryID)
	
	s.logger.Info("Commit created successfully", 
		zap.String("commit_sha", commitSHA),
		zap.String("repository_id", repositoryID.String()))
	
	return commit, nil
}

// GetCommit 获取提交信息
func (s *gitService) GetCommit(ctx context.Context, repositoryID uuid.UUID, sha string) (*models.Commit, error) {
	repo, err := s.repo.GetRepositoryByID(ctx, repositoryID)
	if err != nil {
		return nil, err
	}
	
	// 优先从Git获取最新的提交信息
	gitCommit, err := s.getGitCommitInfo(repo.GitPath, sha)
	if err != nil {
		s.logger.Error("Failed to get git commit info, falling back to database", zap.Error(err))
		// 回退到数据库查询
		return s.repo.GetCommitBySHA(ctx, repositoryID, sha)
	}
	
	// 转换为模型格式
	commit := &models.Commit{
		RepositoryID:    repositoryID,
		SHA:             gitCommit.SHA,
		Message:         gitCommit.Message,
		Author:          gitCommit.Author,
		AuthorEmail:     gitCommit.AuthorEmail,
		Committer:       gitCommit.Committer,
		CommitterEmail:  gitCommit.CommitterEmail,
		CommittedAt:     gitCommit.CommittedAt,
		AddedLines:      int32(gitCommit.AddedLines),
		DeletedLines:    int32(gitCommit.DeletedLines),
		ChangedFiles:    0, // 在需要时计算
	}
	
	// 获取文件变更数量
	if fileChanges, err := s.getCommitFileChanges(repo.GitPath, sha); err == nil {
		commit.ChangedFiles = int32(len(fileChanges))
	}
	
	s.logger.Debug("Retrieved commit info", 
		zap.String("repository_id", repositoryID.String()),
		zap.String("commit_sha", sha))
	
	return commit, nil
}

// ListCommits 获取提交列表
func (s *gitService) ListCommits(ctx context.Context, repositoryID uuid.UUID, branch string, page, pageSize int) (*models.CommitListResponse, error) {
	repo, err := s.repo.GetRepositoryByID(ctx, repositoryID)
	if err != nil {
		return nil, err
	}
	
	// 从Git获取提交历史
	gitCommits, total, err := s.getGitCommitHistory(repo.GitPath, branch, page, pageSize)
	if err != nil {
		s.logger.Error("Failed to get git commit history", zap.Error(err))
		// 回退到数据库查询
		commits, dbTotal, dbErr := s.repo.ListCommits(ctx, repositoryID, branch, page, pageSize)
		if dbErr != nil {
			return nil, fmt.Errorf("failed to get commits from both git and database: git=%w, db=%w", err, dbErr)
		}
		return &models.CommitListResponse{
			Commits:  commits,
			Total:    dbTotal,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}
	
	// 转换Git提交为模型格式
	var commits []models.Commit
	for _, gitCommit := range gitCommits {
		commit := models.Commit{
			RepositoryID:    repositoryID,
			SHA:             gitCommit.SHA,
			Message:         gitCommit.Message,
			Author:          gitCommit.Author,
			AuthorEmail:     gitCommit.AuthorEmail,
			Committer:       gitCommit.Committer,
			CommitterEmail:  gitCommit.CommitterEmail,
			CommittedAt:     gitCommit.CommittedAt,
			AddedLines:      int32(gitCommit.AddedLines),
			DeletedLines:    int32(gitCommit.DeletedLines),
			ChangedFiles:    0, // 在需要时计算
		}
		commits = append(commits, commit)
	}
	
	s.logger.Debug("Retrieved commit history", 
		zap.String("repository_id", repositoryID.String()),
		zap.String("branch", branch),
		zap.Int("total", total),
		zap.Int("returned", len(commits)))
	
	return &models.CommitListResponse{
		Commits:  commits,
		Total:    int64(total),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetCommitDiff 获取提交差异
func (s *gitService) GetCommitDiff(ctx context.Context, repositoryID uuid.UUID, sha string) (*models.GitDiff, error) {
	repo, err := s.repo.GetRepositoryByID(ctx, repositoryID)
	if err != nil {
		return nil, err
	}
	
	return s.getGitDiff(repo.GitPath, sha+"^", sha)
}

// CompareBranches 比较分支
func (s *gitService) CompareBranches(ctx context.Context, repositoryID uuid.UUID, base, head string) (*models.GitDiff, error) {
	repo, err := s.repo.GetRepositoryByID(ctx, repositoryID)
	if err != nil {
		return nil, err
	}
	
	return s.getGitDiff(repo.GitPath, base, head)
}

// 标签管理实现

// CreateTag 创建标签
func (s *gitService) CreateTag(ctx context.Context, repositoryID uuid.UUID, req *models.CreateTagRequest) (*models.Tag, error) {
	repo, err := s.repo.GetRepositoryByID(ctx, repositoryID)
	if err != nil {
		return nil, err
	}
	
	// 在Git仓库中创建标签
	if err := s.createGitTag(repo.GitPath, req); err != nil {
		return nil, fmt.Errorf("failed to create git tag: %w", err)
	}
	
	// 创建标签记录
	tag := &models.Tag{
		RepositoryID: repositoryID,
		Name:         req.Name,
		CommitSHA:    req.CommitSHA,
		Message:      req.Message,
		Tagger:       req.Tagger.Name,
		TaggerEmail:  req.Tagger.Email,
		TaggedAt:     time.Now(),
	}
	
	if err := s.repo.CreateTag(ctx, tag); err != nil {
		return nil, err
	}
	
	// 更新仓库标签计数
	s.updateRepositoryTagCount(ctx, repositoryID)
	
	return tag, nil
}

// GetTag 获取标签信息
func (s *gitService) GetTag(ctx context.Context, repositoryID uuid.UUID, name string) (*models.Tag, error) {
	return s.repo.GetTagByName(ctx, repositoryID, name)
}

// ListTags 获取标签列表
func (s *gitService) ListTags(ctx context.Context, repositoryID uuid.UUID) ([]models.Tag, error) {
	return s.repo.ListTags(ctx, repositoryID)
}

// DeleteTag 删除标签
func (s *gitService) DeleteTag(ctx context.Context, repositoryID uuid.UUID, name string) error {
	repo, err := s.repo.GetRepositoryByID(ctx, repositoryID)
	if err != nil {
		return err
	}
	
	// 从Git仓库删除标签
	if err := s.deleteGitTag(repo.GitPath, name); err != nil {
		return fmt.Errorf("failed to delete git tag: %w", err)
	}
	
	// 删除标签记录
	if err := s.repo.DeleteTag(ctx, repositoryID, name); err != nil {
		return err
	}
	
	// 更新仓库标签计数
	s.updateRepositoryTagCount(ctx, repositoryID)
	
	return nil
}

// 文件操作实现

// GetFileContent 获取文件内容
func (s *gitService) GetFileContent(ctx context.Context, repositoryID uuid.UUID, branch, filePath string) ([]byte, error) {
	repo, err := s.repo.GetRepositoryByID(ctx, repositoryID)
	if err != nil {
		return nil, err
	}
	
	return s.getGitFileContent(repo.GitPath, branch, filePath)
}

// GetDirectoryContent 获取目录内容
func (s *gitService) GetDirectoryContent(ctx context.Context, repositoryID uuid.UUID, branch, dirPath string) ([]models.FileInfo, error) {
	repo, err := s.repo.GetRepositoryByID(ctx, repositoryID)
	if err != nil {
		return nil, err
	}
	
	return s.getGitDirectoryContent(repo.GitPath, branch, dirPath)
}

// 统计和搜索实现

// GetRepositoryStats 获取仓库统计信息
func (s *gitService) GetRepositoryStats(ctx context.Context, repositoryID uuid.UUID) (*models.RepositoryStats, error) {
	return s.repo.GetRepositoryStats(ctx, repositoryID)
}

// SearchRepositories 搜索仓库
func (s *gitService) SearchRepositories(ctx context.Context, query string, projectID *uuid.UUID, page, pageSize int) (*models.RepositoryListResponse, error) {
	repos, total, err := s.repo.SearchRepositories(ctx, query, projectID, page, pageSize)
	if err != nil {
		return nil, err
	}
	
	return &models.RepositoryListResponse{
		Repositories: repos,
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
	}, nil
}

// 私有辅助方法

// generateCloneURL 生成克隆URL
func (s *gitService) generateCloneURL(projectID uuid.UUID, repoName string) string {
	// TODO: 根据配置生成实际的URL
	return fmt.Sprintf("https://git.example.com/%s/%s.git", projectID.String(), repoName)
}

// generateSSHURL 生成SSH URL
func (s *gitService) generateSSHURL(projectID uuid.UUID, repoName string) string {
	// TODO: 根据配置生成实际的URL
	return fmt.Sprintf("git@git.example.com:%s/%s.git", projectID.String(), repoName)
}

// initGitRepository 初始化Git仓库
func (s *gitService) initGitRepository(repoPath, defaultBranch string, initReadme bool) error {
	// 创建目录
	if err := os.MkdirAll(filepath.Dir(repoPath), 0755); err != nil {
		return err
	}
	
	// 初始化裸仓库
	cmd := exec.Command("git", "init", "--bare", repoPath)
	if err := cmd.Run(); err != nil {
		return err
	}
	
	// 设置默认分支
	if defaultBranch != "master" {
		cmd = exec.Command("git", "symbolic-ref", "HEAD", "refs/heads/"+defaultBranch)
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			s.logger.Warn("Failed to set default branch", zap.Error(err))
		}
	}
	
	// TODO: 如果需要初始化README，创建初始提交
	
	return nil
}

// createDefaultBranch 创建默认分支记录
func (s *gitService) createDefaultBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) error {
	branch := &models.Branch{
		RepositoryID: repositoryID,
		Name:         branchName,
		CommitSHA:    "0000000000000000000000000000000000000000", // 空提交SHA
		IsDefault:    true,
		IsProtected:  false,
	}
	
	return s.repo.CreateBranch(ctx, branch)
}

// Git操作相关的私有方法

// createGitBranch 创建Git分支
func (s *gitService) createGitBranch(repoPath, branchName, fromSHA string) error {
	s.logger.Info("Creating git branch", 
		zap.String("repo_path", repoPath),
		zap.String("branch_name", branchName),
		zap.String("from_sha", fromSHA))
	
	// 检查分支是否已存在
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branchName)
	cmd.Dir = repoPath
	if err := cmd.Run(); err == nil {
		return fmt.Errorf("branch '%s' already exists", branchName)
	}
	
	// 创建分支
	var cmd2 *exec.Cmd
	if fromSHA != "" {
		// 从指定提交创建分支
		cmd2 = exec.Command("git", "branch", branchName, fromSHA)
	} else {
		// 从当前HEAD创建分支
		cmd2 = exec.Command("git", "branch", branchName)
	}
	cmd2.Dir = repoPath
	
	if output, err := cmd2.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create branch: %s", string(output))
	}
	
	s.logger.Info("Git branch created successfully", zap.String("branch_name", branchName))
	return nil
}

// deleteGitBranch 删除Git分支
func (s *gitService) deleteGitBranch(repoPath, branchName string) error {
	s.logger.Info("Deleting git branch", 
		zap.String("repo_path", repoPath),
		zap.String("branch_name", branchName))
	
	// 检查分支是否存在
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branchName)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("branch '%s' does not exist", branchName)
	}
	
	// 删除分支
	cmd = exec.Command("git", "branch", "-d", branchName)
	cmd.Dir = repoPath
	
	if _, err := cmd.CombinedOutput(); err != nil {
		// 尝试强制删除
		cmd = exec.Command("git", "branch", "-D", branchName)
		cmd.Dir = repoPath
		if output2, err2 := cmd.CombinedOutput(); err2 != nil {
			return fmt.Errorf("failed to delete branch: %s", string(output2))
		}
		s.logger.Warn("Branch force deleted", zap.String("branch_name", branchName))
	}
	
	s.logger.Info("Git branch deleted successfully", zap.String("branch_name", branchName))
	return nil
}

// mergeBranch 合并分支
func (s *gitService) mergeBranch(repoPath, targetBranch, sourceBranch string) error {
	s.logger.Info("Merging git branch", 
		zap.String("repo_path", repoPath),
		zap.String("target_branch", targetBranch),
		zap.String("source_branch", sourceBranch))
	
	// 切换到目标分支
	cmd := exec.Command("git", "checkout", targetBranch)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to checkout target branch: %s", string(output))
	}
	
	// 执行合并
	cmd = exec.Command("git", "merge", sourceBranch, "--no-ff", "-m", fmt.Sprintf("Merge branch '%s' into '%s'", sourceBranch, targetBranch))
	cmd.Dir = repoPath
	
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to merge branch: %s", string(output))
	}
	
	s.logger.Info("Git branch merged successfully", 
		zap.String("target_branch", targetBranch),
		zap.String("source_branch", sourceBranch))
	return nil
}

// createGitCommit 创建Git提交
func (s *gitService) createGitCommit(repoPath string, req *models.CreateCommitRequest) (string, error) {
	s.logger.Info("Creating git commit", 
		zap.String("repo_path", repoPath),
		zap.String("message", req.Message))
	
	// 检查仓库是否存在
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return "", fmt.Errorf("repository not found: %s", repoPath)
	}
	
	// 设置作者信息
	env := append(os.Environ(),
		"GIT_AUTHOR_NAME="+req.Author.Name,
		"GIT_AUTHOR_EMAIL="+req.Author.Email,
		"GIT_COMMITTER_NAME="+req.Author.Name,
		"GIT_COMMITTER_EMAIL="+req.Author.Email,
	)
	
	// 切换到指定分支
	if req.Branch != "" {
		cmd := exec.Command("git", "checkout", req.Branch)
		cmd.Dir = repoPath
		cmd.Env = env
		if output, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("failed to checkout branch: %s", string(output))
		}
	}
	
	var addedLines, deletedLines int
	
	// 处理文件变更
	for _, file := range req.Files {
		filePath := filepath.Join(repoPath, file.Path)
		
		// 获取变更前的文件内容进行差异计算
		var oldContent []byte
		if existingFile, err := os.ReadFile(filePath); err == nil {
			oldContent = existingFile
		}
		
		// 创建目录
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return "", fmt.Errorf("failed to create directory: %w", err)
		}
		
		// 写入新的文件内容
		newContent := []byte(file.Content)
		if err := os.WriteFile(filePath, newContent, 0644); err != nil {
			return "", fmt.Errorf("failed to write file: %w", err)
		}
		
		// 计算行变更数
		added, deleted := s.calculateLineDiff(oldContent, newContent)
		addedLines += added
		deletedLines += deleted
		
		// 添加到暂存区
		addCmd := exec.Command("git", "add", file.Path)
		addCmd.Dir = repoPath
		addCmd.Env = env
		if output, err := addCmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("failed to add file %s: %s", file.Path, string(output))
		}
	}
	
	// 检查是否有变更需要提交
	statusCmd := exec.Command("git", "status", "--porcelain", "--cached")
	statusCmd.Dir = repoPath
	statusCmd.Env = env
	statusOutput, err := statusCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to check git status: %w", err)
	}
	
	if len(strings.TrimSpace(string(statusOutput))) == 0 {
		return "", fmt.Errorf("no changes staged for commit")
	}
	
	// 创建提交
	commitCmd := exec.Command("git", "commit", "-m", req.Message)
	commitCmd.Dir = repoPath
	commitCmd.Env = env
	
	if output, err := commitCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to create commit: %s", string(output))
	}
	
	// 获取提交SHA
	shaCmd := exec.Command("git", "rev-parse", "HEAD")
	shaCmd.Dir = repoPath
	shaCmd.Env = env
	shaOutput, err := shaCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commit SHA: %w", err)
	}
	
	commitSHA := strings.TrimSpace(string(shaOutput))
	s.logger.Info("Git commit created successfully", 
		zap.String("commit_sha", commitSHA),
		zap.Int("added_lines", addedLines),
		zap.Int("deleted_lines", deletedLines))
	
	return commitSHA, nil
}

// createGitTag 创建Git标签
func (s *gitService) createGitTag(repoPath string, req *models.CreateTagRequest) error {
	s.logger.Info("Creating git tag", 
		zap.String("repo_path", repoPath),
		zap.String("tag_name", req.Name))
	
	// 设置标签创建者信息
	if req.Tagger != (models.CommitAuthor{}) {
		os.Setenv("GIT_COMMITTER_NAME", req.Tagger.Name)
		os.Setenv("GIT_COMMITTER_EMAIL", req.Tagger.Email)
	}
	
	var cmd *exec.Cmd
	if req.Message != nil && *req.Message != "" {
		// 创建带注释的标签
		cmd = exec.Command("git", "tag", "-a", req.Name, "-m", *req.Message, req.CommitSHA)
	} else {
		// 创建轻量级标签
		cmd = exec.Command("git", "tag", req.Name, req.CommitSHA)
	}
	cmd.Dir = repoPath
	
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create tag: %s", string(output))
	}
	
	s.logger.Info("Git tag created successfully", zap.String("tag_name", req.Name))
	return nil
}

// deleteGitTag 删除Git标签
func (s *gitService) deleteGitTag(repoPath, tagName string) error {
	s.logger.Info("Deleting git tag", 
		zap.String("repo_path", repoPath),
		zap.String("tag_name", tagName))
	
	cmd := exec.Command("git", "tag", "-d", tagName)
	cmd.Dir = repoPath
	
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete tag: %s", string(output))
	}
	
	s.logger.Info("Git tag deleted successfully", zap.String("tag_name", tagName))
	return nil
}

// getGitDiff 获取Git差异
func (s *gitService) getGitDiff(repoPath, from, to string) (*models.GitDiff, error) {
	s.logger.Info("Getting git diff", 
		zap.String("repo_path", repoPath),
		zap.String("from", from),
		zap.String("to", to))
	
	cmd := exec.Command("git", "diff", "--name-status", from+".."+to)
	cmd.Dir = repoPath
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get diff: %w", err)
	}
	
	// 解析差异输出
	var files []models.DiffFile
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if len(line) < 2 {
			continue
		}
		
		parts := line[2:] // 跳过状态字符和制表符
		status := string(line[0])
		
		diffFile := models.DiffFile{
			Path:   parts,
			Status: status,
		}
		files = append(files, diffFile)
	}
	
	return &models.GitDiff{
		FromSHA: from,
		ToSHA:   to,
		Files:   files,
	}, nil
}

// getGitFileContent 获取Git文件内容
func (s *gitService) getGitFileContent(repoPath, branch, filePath string) ([]byte, error) {
	s.logger.Info("Getting git file content", 
		zap.String("repo_path", repoPath),
		zap.String("branch", branch),
		zap.String("file_path", filePath))
	
	cmd := exec.Command("git", "show", branch+":"+filePath)
	cmd.Dir = repoPath
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get file content: %w", err)
	}
	
	return output, nil
}

// getGitDirectoryContent 获取Git目录内容
func (s *gitService) getGitDirectoryContent(repoPath, branch, dirPath string) ([]models.FileInfo, error) {
	s.logger.Info("Getting git directory content", 
		zap.String("repo_path", repoPath),
		zap.String("branch", branch),
		zap.String("dir_path", dirPath))
	
	cmd := exec.Command("git", "ls-tree", "-l", branch+":"+dirPath)
	cmd.Dir = repoPath
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get directory content: %w", err)
	}
	
	var files []models.FileInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		
		// 解析git ls-tree输出
		// 格式: <mode> <type> <object> <size> <name>
		parts := strings.Fields(line)
		if len(parts) < 5 {
			continue
		}
		
		fileInfo := models.FileInfo{
			Name: parts[4],
			Type: parts[1],
			Size: 0, // TODO: 解析大小
		}
		
		if parts[1] == "blob" {
			fileInfo.Type = "file"
		} else if parts[1] == "tree" {
			fileInfo.Type = "directory"
		}
		
		files = append(files, fileInfo)
	}
	
	return files, nil
}

// 统计更新方法

func (s *gitService) updateRepositoryBranchCount(ctx context.Context, repositoryID uuid.UUID) {
	branches, err := s.repo.ListBranches(ctx, repositoryID)
	if err != nil {
		return
	}
	
	s.repo.UpdateRepository(ctx, repositoryID, map[string]interface{}{
		"branch_count": len(branches),
	})
}

func (s *gitService) updateRepositoryCommitCount(ctx context.Context, repositoryID uuid.UUID) {
	repo, err := s.repo.GetRepositoryByID(ctx, repositoryID)
	if err != nil {
		s.logger.Error("Failed to get repository for commit count update", zap.Error(err))
		return
	}
	
	// 从Git获取总提交数
	commitCount, err := s.getGitTotalCommitCount(repo.GitPath, repo.DefaultBranch)
	if err != nil {
		s.logger.Error("Failed to get git commit count", zap.Error(err))
		return
	}
	
	// 更新数据库中的提交计数
	s.repo.UpdateRepository(ctx, repositoryID, map[string]interface{}{
		"commit_count": commitCount,
	})
	
	s.logger.Debug("Updated repository commit count", 
		zap.String("repository_id", repositoryID.String()),
		zap.Int64("commit_count", commitCount))
}

func (s *gitService) updateRepositoryTagCount(ctx context.Context, repositoryID uuid.UUID) {
	tags, err := s.repo.ListTags(ctx, repositoryID)
	if err != nil {
		return
	}
	
	s.repo.UpdateRepository(ctx, repositoryID, map[string]interface{}{
		"tag_count": len(tags),
	})
}

// calculateLineDiff 计算行变更差异
func (s *gitService) calculateLineDiff(oldContent, newContent []byte) (added int, deleted int) {
	oldLines := strings.Split(string(oldContent), "\n")
	newLines := strings.Split(string(newContent), "\n")
	
	// 简单的行差异计算
	if len(oldContent) == 0 {
		// 新文件
		return len(newLines), 0
	}
	
	if len(newContent) == 0 {
		// 删除文件
		return 0, len(oldLines)
	}
	
	// 计算变更行数的简化算法
	if len(newLines) > len(oldLines) {
		added = len(newLines) - len(oldLines)
	} else if len(oldLines) > len(newLines) {
		deleted = len(oldLines) - len(newLines)
	}
	
	// TODO: 实现更精确的diff算法
	return added, deleted
}

// getGitCommitInfo 从Git获取提交详细信息
func (s *gitService) getGitCommitInfo(repoPath, commitSHA string) (*GitCommitInfo, error) {
	s.logger.Debug("Getting git commit info", 
		zap.String("repo_path", repoPath),
		zap.String("commit_sha", commitSHA))
	
	// 获取提交基本信息
	cmd := exec.Command("git", "show", "--format=%H%n%s%n%B%n%an%n%ae%n%cn%n%ce%n%ct%n%P%n%T", "--no-patch", commitSHA)
	cmd.Dir = repoPath
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commit info: %w", err)
	}
	
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 9 {
		return nil, fmt.Errorf("invalid git show output")
	}
	
	// 解析提交时间
	timestampStr := lines[7]
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		timestamp = time.Now().Unix()
	}
	
	// 解析父提交SHA
	var parentSHAs []string
	if lines[8] != "" {
		parentSHAs = strings.Fields(lines[8])
	}
	
	commitInfo := &GitCommitInfo{
		SHA:            lines[0],
		Message:        lines[1],
		Author:         lines[3],
		AuthorEmail:    lines[4],
		Committer:      lines[5],
		CommitterEmail: lines[6],
		CommittedAt:    time.Unix(timestamp, 0),
		ParentSHAs:     parentSHAs,
		TreeSHA:        lines[9],
	}
	
	// 获取统计信息
	statsCmd := exec.Command("git", "show", "--numstat", "--format=", commitSHA)
	statsCmd.Dir = repoPath
	
	statsOutput, err := statsCmd.Output()
	if err == nil {
		addedLines, deletedLines := s.parseGitStats(string(statsOutput))
		commitInfo.AddedLines = addedLines
		commitInfo.DeletedLines = deletedLines
	}
	
	return commitInfo, nil
}

// parseGitStats 解析Git统计输出
func (s *gitService) parseGitStats(output string) (added, deleted int) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			if parts[0] != "-" {
				if num, err := strconv.Atoi(parts[0]); err == nil {
					added += num
				}
			}
			if parts[1] != "-" {
				if num, err := strconv.Atoi(parts[1]); err == nil {
					deleted += num
				}
			}
		}
	}
	
	return added, deleted
}

// getCommitFileChanges 获取提交的文件变更列表
func (s *gitService) getCommitFileChanges(repoPath, commitSHA string) ([]FileChange, error) {
	s.logger.Debug("Getting commit file changes", 
		zap.String("repo_path", repoPath),
		zap.String("commit_sha", commitSHA))
	
	// 获取文件变更状态和统计
	cmd := exec.Command("git", "show", "--numstat", "--name-status", "--format=", commitSHA)
	cmd.Dir = repoPath
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get file changes: %w", err)
	}
	
	var changes []FileChange
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	
	// 解析numstat输出
	var statLines []string
	var statusLines []string
	
	// 分离numstat和name-status输出
	inStats := true
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		if strings.HasPrefix(line, "A\t") || strings.HasPrefix(line, "M\t") || strings.HasPrefix(line, "D\t") {
			inStats = false
		}
		
		if inStats {
			statLines = append(statLines, line)
		} else {
			statusLines = append(statusLines, line)
		}
	}
	
	// 解析文件状态
	for i, statusLine := range statusLines {
		if statusLine == "" {
			continue
		}
		
		parts := strings.Fields(statusLine)
		if len(parts) < 2 {
			continue
		}
		
		status := parts[0]
		filePath := parts[1]
		var oldPath string
		
		// 处理重命名的情况
		if strings.HasPrefix(status, "R") && len(parts) >= 3 {
			oldPath = filePath
			filePath = parts[2]
		}
		
		change := FileChange{
			Path:    filePath,
			OldPath: oldPath,
			Status:  s.normalizeFileStatus(status),
		}
		
		// 匹配对应的统计信息
		if i < len(statLines) {
			statParts := strings.Fields(statLines[i])
			if len(statParts) >= 2 {
				if statParts[0] != "-" {
					if num, err := strconv.Atoi(statParts[0]); err == nil {
						change.AddedLines = num
					}
				}
				if statParts[1] != "-" {
					if num, err := strconv.Atoi(statParts[1]); err == nil {
						change.DeletedLines = num
					}
				}
			}
		}
		
		changes = append(changes, change)
	}
	
	return changes, nil
}

// normalizeFileStatus 标准化文件状态
func (s *gitService) normalizeFileStatus(gitStatus string) string {
	switch {
	case strings.HasPrefix(gitStatus, "A"):
		return "added"
	case strings.HasPrefix(gitStatus, "M"):
		return "modified"
	case strings.HasPrefix(gitStatus, "D"):
		return "deleted"
	case strings.HasPrefix(gitStatus, "R"):
		return "renamed"
	case strings.HasPrefix(gitStatus, "C"):
		return "copied"
	default:
		return "modified"
	}
}

// getGitCommitHistory 从Git获取提交历史
func (s *gitService) getGitCommitHistory(repoPath, branch string, page, pageSize int) ([]GitCommitInfo, int, error) {
	s.logger.Debug("Getting git commit history", 
		zap.String("repo_path", repoPath),
		zap.String("branch", branch),
		zap.Int("page", page),
		zap.Int("page_size", pageSize))
	
	// 确定分支参数
	branchArg := "HEAD"
	if branch != "" {
		branchArg = branch
	}
	
	// 计算跳过的提交数
	skip := (page - 1) * pageSize
	
	// 获取提交历史
	cmd := exec.Command("git", "log", "--format=%H%n%s%n%an%n%ae%n%cn%n%ce%n%ct%n%P", 
		"--skip="+strconv.Itoa(skip), "--max-count="+strconv.Itoa(pageSize), branchArg)
	cmd.Dir = repoPath
	
	output, err := cmd.Output()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get commit history: %w", err)
	}
	
	// 解析提交信息
	var commits []GitCommitInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	
	for i := 0; i < len(lines); i += 8 {
		if i+7 >= len(lines) {
			break
		}
		
		// 解析提交时间
		timestampStr := lines[i+6]
		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			timestamp = time.Now().Unix()
		}
		
		// 解析父提交SHA
		var parentSHAs []string
		if lines[i+7] != "" {
			parentSHAs = strings.Fields(lines[i+7])
		}
		
		commit := GitCommitInfo{
			SHA:            lines[i],
			Message:        lines[i+1],
			Author:         lines[i+2],
			AuthorEmail:    lines[i+3],
			Committer:      lines[i+4],
			CommitterEmail: lines[i+5],
			CommittedAt:    time.Unix(timestamp, 0),
			ParentSHAs:     parentSHAs,
		}
		
		// 获取统计信息（可选，性能考虑）
		if pageSize <= 20 { // 只对小批量获取统计信息
			if statsCmd := exec.Command("git", "show", "--numstat", "--format=", commit.SHA); true {
				statsCmd.Dir = repoPath
				if statsOutput, err := statsCmd.Output(); err == nil {
					added, deleted := s.parseGitStats(string(statsOutput))
					commit.AddedLines = added
					commit.DeletedLines = deleted
				}
			}
		}
		
		commits = append(commits, commit)
	}
	
	// 获取总提交数
	total, err := s.getGitTotalCommitCount(repoPath, branchArg)
	if err != nil {
		s.logger.Warn("Failed to get total commit count", zap.Error(err))
		total = int64(len(commits)) // 降级处理
	}
	
	return commits, int(total), nil
}

// getGitTotalCommitCount 获取Git总提交数
func (s *gitService) getGitTotalCommitCount(repoPath, branch string) (int64, error) {
	s.logger.Debug("Getting git total commit count", 
		zap.String("repo_path", repoPath),
		zap.String("branch", branch))
	
	branchArg := "HEAD"
	if branch != "" {
		branchArg = branch
	}
	
	cmd := exec.Command("git", "rev-list", "--count", branchArg)
	cmd.Dir = repoPath
	
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get commit count: %w", err)
	}
	
	countStr := strings.TrimSpace(string(output))
	count, err := strconv.ParseInt(countStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid commit count: %w", err)
	}
	
	return count, nil
}

// isRepositoryConfigMatching 检查仓库配置是否匹配（用于幂等性）
func (s *gitService) isRepositoryConfigMatching(existing *models.Repository, req *models.CreateRepositoryRequest) bool {
	// 检查基本配置是否匹配
	if existing.Visibility != req.Visibility {
		return false
	}
	
	// 检查描述是否匹配
	if (existing.Description == nil) != (req.Description == nil) {
		return false
	}
	if existing.Description != nil && req.Description != nil && *existing.Description != *req.Description {
		return false
	}
	
	// 检查默认分支是否匹配
	expectedDefaultBranch := "main"
	if req.DefaultBranch != nil {
		expectedDefaultBranch = *req.DefaultBranch
	}
	if existing.DefaultBranch != expectedDefaultBranch {
		return false
	}
	
	// 检查仓库状态是否为活跃状态
	if existing.Status != models.RepositoryStatusActive {
		return false
	}
	
	return true
}