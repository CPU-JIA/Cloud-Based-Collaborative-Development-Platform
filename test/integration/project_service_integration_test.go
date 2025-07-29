package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/client"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/compensation"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/handlers"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/models"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/repository"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/service"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/transaction"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/webhook"
	"github.com/cloud-platform/collaborative-dev/shared/database"
	"github.com/cloud-platform/collaborative-dev/shared/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ProjectServiceIntegrationTestSuite 项目服务集成测试套件
type ProjectServiceIntegrationTestSuite struct {
	suite.Suite
	db                 *gorm.DB
	logger             *zap.Logger
	router             *gin.Engine
	server             *httptest.Server
	repo               repository.ProjectRepository
	gitClient          *MockGitGatewayClient
	projectService     service.ProjectService
	projectHandler     *handlers.ProjectHandler
	compensationMgr    *compensation.CompensationManager
	transactionMgr     *transaction.DistributedTransactionManager
	webhookHandler     *webhook.WebhookHandler
	testTenantID       uuid.UUID
	testUserID         uuid.UUID
	testProject        *models.Project
	testRepository     *models.Repository
	testAdminUserID    uuid.UUID
	testMemberUserID   uuid.UUID
	authToken          string
	adminAuthToken     string
	memberAuthToken    string
}

// MockGitGatewayClient Mock Git网关客户端（用于集成测试）
type MockGitGatewayClient struct {
	repositories map[uuid.UUID]*client.Repository
	branches     map[uuid.UUID][]*client.Branch
	errors       map[string]error // 用于模拟错误场景
	callLog      []string         // 记录调用日志
}

// NewMockGitGatewayClient 创建Mock Git网关客户端
func NewMockGitGatewayClient() *MockGitGatewayClient {
	return &MockGitGatewayClient{
		repositories: make(map[uuid.UUID]*client.Repository),
		branches:     make(map[uuid.UUID][]*client.Branch),
		errors:       make(map[string]error),
		callLog:      make([]string, 0),
	}
}

// SetError 设置特定操作的错误
func (m *MockGitGatewayClient) SetError(operation string, err error) {
	m.errors[operation] = err
}

// GetCallLog 获取调用日志
func (m *MockGitGatewayClient) GetCallLog() []string {
	return m.callLog
}

// ClearCallLog 清空调用日志
func (m *MockGitGatewayClient) ClearCallLog() {
	m.callLog = make([]string, 0)
}

// CreateRepository 创建仓库（Mock实现）
func (m *MockGitGatewayClient) CreateRepository(ctx context.Context, req *client.CreateRepositoryRequest) (*client.Repository, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("CreateRepository: %s", req.Name))
	
	if err, exists := m.errors["CreateRepository"]; exists {
		return nil, err
	}

	projectID, _ := uuid.Parse(req.ProjectID)
	repo := &client.Repository{
		ID:            uuid.New(),
		ProjectID:     projectID,
		Name:          req.Name,
		Description:   req.Description,
		Visibility:    req.Visibility,
		DefaultBranch: func() string {
			if req.DefaultBranch != nil {
				return *req.DefaultBranch
			}
			return "main"
		}(),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	m.repositories[repo.ID] = repo
	
	// 创建默认分支
	defaultBranchName := "main"
	if req.DefaultBranch != nil {
		defaultBranchName = *req.DefaultBranch
	}
	
	branch := &client.Branch{
		ID:           uuid.New(),
		RepositoryID: repo.ID,
		Name:         defaultBranchName,
		CommitSHA:    "abc123",
		IsDefault:    true,
		IsProtected:  false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	m.branches[repo.ID] = []*client.Branch{branch}
	
	return repo, nil
}

// GetRepository 获取仓库（Mock实现）
func (m *MockGitGatewayClient) GetRepository(ctx context.Context, id uuid.UUID) (*client.Repository, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetRepository: %s", id.String()))
	
	if err, exists := m.errors["GetRepository"]; exists {
		return nil, err
	}

	repo, exists := m.repositories[id]
	if !exists {
		return nil, fmt.Errorf("repository not found")
	}
	
	return repo, nil
}

// ListRepositories 获取仓库列表（Mock实现）
func (m *MockGitGatewayClient) ListRepositories(ctx context.Context, projectID *uuid.UUID, page, pageSize int) (*client.RepositoryListResponse, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("ListRepositories: project=%s", projectID.String()))
	
	if err, exists := m.errors["ListRepositories"]; exists {
		return nil, err
	}

	var repos []*client.Repository
	for _, repo := range m.repositories {
		if projectID == nil || repo.ProjectID == *projectID {
			repos = append(repos, repo)
		}
	}

	// 简单分页
	startIndex := (page - 1) * pageSize
	endIndex := startIndex + pageSize
	if startIndex >= len(repos) {
		repos = []*client.Repository{}
	} else if endIndex > len(repos) {
		repos = repos[startIndex:]
	} else {
		repos = repos[startIndex:endIndex]
	}

	// Convert []*Repository to []Repository
	result := make([]client.Repository, len(repos))
	for i, repo := range repos {
		result[i] = *repo
	}

	return &client.RepositoryListResponse{
		Repositories: result,
		Total:        int64(len(m.repositories)),
		Page:         page,
		PageSize:     pageSize,
	}, nil
}

// UpdateRepository 更新仓库（Mock实现）
func (m *MockGitGatewayClient) UpdateRepository(ctx context.Context, id uuid.UUID, req *client.UpdateRepositoryRequest) (*client.Repository, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("UpdateRepository: %s", id.String()))
	
	if err, exists := m.errors["UpdateRepository"]; exists {
		return nil, err
	}

	repo, exists := m.repositories[id]
	if !exists {
		return nil, fmt.Errorf("repository not found")
	}

	// 更新字段
	if req.Name != nil {
		repo.Name = *req.Name
	}
	if req.Description != nil {
		repo.Description = req.Description
	}
	if req.Visibility != nil {
		repo.Visibility = *req.Visibility
	}
	if req.DefaultBranch != nil {
		if req.DefaultBranch != nil {
			repo.DefaultBranch = *req.DefaultBranch
		}
	}
	repo.UpdatedAt = time.Now()

	return repo, nil
}

// DeleteRepository 删除仓库（Mock实现）
func (m *MockGitGatewayClient) DeleteRepository(ctx context.Context, id uuid.UUID) error {
	m.callLog = append(m.callLog, fmt.Sprintf("DeleteRepository: %s", id.String()))
	
	if err, exists := m.errors["DeleteRepository"]; exists {
		return err
	}

	delete(m.repositories, id)
	delete(m.branches, id)
	return nil
}

// CreateBranch 创建分支（Mock实现）
func (m *MockGitGatewayClient) CreateBranch(ctx context.Context, repositoryID uuid.UUID, req *client.CreateBranchRequest) (*client.Branch, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("CreateBranch: repo=%s, branch=%s", repositoryID.String(), req.Name))
	
	if err, exists := m.errors["CreateBranch"]; exists {
		return nil, err
	}

	branch := &client.Branch{
		ID:           uuid.New(),
		RepositoryID: repositoryID,
		Name:         req.Name,
		CommitSHA:    req.FromSHA,
		IsDefault:    false,
		IsProtected:  false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if _, exists := m.branches[repositoryID]; !exists {
		m.branches[repositoryID] = []*client.Branch{}
	}
	m.branches[repositoryID] = append(m.branches[repositoryID], branch)

	return branch, nil
}

// ListBranches 获取分支列表（Mock实现）
func (m *MockGitGatewayClient) ListBranches(ctx context.Context, repositoryID uuid.UUID) ([]client.Branch, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("ListBranches: %s", repositoryID.String()))
	
	if err, exists := m.errors["ListBranches"]; exists {
		return nil, err
	}

	branches, exists := m.branches[repositoryID]
	if !exists {
		return []client.Branch{}, nil
	}

	// Convert []*Branch to []Branch
	result := make([]client.Branch, len(branches))
	for i, branch := range branches {
		result[i] = *branch
	}

	return result, nil
}

// DeleteBranch 删除分支（Mock实现）
func (m *MockGitGatewayClient) DeleteBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("DeleteBranch: repo=%s, branch=%s", repositoryID.String(), branchName))
	
	if err, exists := m.errors["DeleteBranch"]; exists {
		return err
	}

	branches, exists := m.branches[repositoryID]
	if !exists {
		return fmt.Errorf("repository not found")
	}

	for i, branch := range branches {
		if branch.Name == branchName {
			m.branches[repositoryID] = append(branches[:i], branches[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("branch not found")
}

// CheckHealth 健康检查（Mock实现）
func (m *MockGitGatewayClient) CheckHealth(ctx context.Context) error {
	m.callLog = append(m.callLog, "CheckHealth")
	
	if err, exists := m.errors["CheckHealth"]; exists {
		return err
	}
	
	return nil
}

// Additional methods to implement the complete GitGatewayClient interface

// GetBranch 获取分支详情（Mock实现）
func (m *MockGitGatewayClient) GetBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) (*client.Branch, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetBranch: repo=%s, branch=%s", repositoryID.String(), branchName))
	
	if err, exists := m.errors["GetBranch"]; exists {
		return nil, err
	}

	branches, exists := m.branches[repositoryID]
	if !exists {
		return nil, fmt.Errorf("repository not found")
	}

	for _, branch := range branches {
		if branch.Name == branchName {
			return branch, nil
		}
	}

	return nil, fmt.Errorf("branch not found")
}

// SetDefaultBranch 设置默认分支（Mock实现）
func (m *MockGitGatewayClient) SetDefaultBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("SetDefaultBranch: repo=%s, branch=%s", repositoryID.String(), branchName))
	
	if err, exists := m.errors["SetDefaultBranch"]; exists {
		return err
	}

	branches, exists := m.branches[repositoryID]
	if !exists {
		return fmt.Errorf("repository not found")
	}

	// 重置所有分支的默认状态
	for _, branch := range branches {
		branch.IsDefault = false
	}

	// 设置新的默认分支
	for _, branch := range branches {
		if branch.Name == branchName {
			branch.IsDefault = true
			return nil
		}
	}

	return fmt.Errorf("branch not found")
}

// MergeBranch 合并分支（Mock实现）
func (m *MockGitGatewayClient) MergeBranch(ctx context.Context, repositoryID uuid.UUID, targetBranch, sourceBranch string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("MergeBranch: repo=%s, target=%s, source=%s", repositoryID.String(), targetBranch, sourceBranch))
	
	if err, exists := m.errors["MergeBranch"]; exists {
		return err
	}

	return nil
}

// CompareBranches 比较分支（Mock实现）
func (m *MockGitGatewayClient) CompareBranches(ctx context.Context, repositoryID uuid.UUID, base, head string) (*client.GitDiff, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("CompareBranches: repo=%s, base=%s, head=%s", repositoryID.String(), base, head))
	
	if err, exists := m.errors["CompareBranches"]; exists {
		return nil, err
	}
	
	// Mock比较结果
	return &client.GitDiff{
		FromSHA: base,
		ToSHA:   head,
		Files: []client.DiffFile{
			{
				Path:         "README.md",
				Status:       "modified",
				AddedLines:   5,
				DeletedLines: 2,
				Patch:        "@@ -1,1 +1,3 @@\n-# Old Title\n+# New Title\n+\n+Updated README content",
			},
		},
		TotalAdded:   5,
		TotalDeleted: 2,
	}, nil
}

// CreateCommit 创建提交（Mock实现）
func (m *MockGitGatewayClient) CreateCommit(ctx context.Context, repositoryID uuid.UUID, req *client.CreateCommitRequest) (*client.Commit, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("CreateCommit: repo=%s, message=%s", repositoryID.String(), req.Message))
	
	if err, exists := m.errors["CreateCommit"]; exists {
		return nil, err
	}

	commit := &client.Commit{
		ID:             uuid.New(),
		RepositoryID:   repositoryID,
		SHA:            "abc123def456",
		Message:        req.Message,
		Author:         req.Author.Name,
		AuthorEmail:    req.Author.Email,
		Committer:      req.Author.Name,
		CommitterEmail: req.Author.Email,
		ParentSHAs:     []string{"parent123"},
		TreeSHA:        "tree456",
		AddedLines:     10,
		DeletedLines:   2,
		ChangedFiles:   int32(len(req.Files)),
		CommittedAt:    time.Now(),
		CreatedAt:      time.Now(),
	}

	return commit, nil
}

// GetCommit 获取提交详情（Mock实现）
func (m *MockGitGatewayClient) GetCommit(ctx context.Context, repositoryID uuid.UUID, sha string) (*client.Commit, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetCommit: repo=%s, sha=%s", repositoryID.String(), sha))
	
	if err, exists := m.errors["GetCommit"]; exists {
		return nil, err
	}

	commit := &client.Commit{
		ID:             uuid.New(),
		RepositoryID:   repositoryID,
		SHA:            sha,
		Message:        "Mock commit message",
		Author:         "Mock Author",
		AuthorEmail:    "author@example.com",
		Committer:      "Mock Committer",
		CommitterEmail: "committer@example.com",
		ParentSHAs:     []string{"parent123"},
		TreeSHA:        "tree456",
		AddedLines:     5,
		DeletedLines:   1,
		ChangedFiles:   2,
		CommittedAt:    time.Now(),
		CreatedAt:      time.Now(),
	}

	return commit, nil
}

// ListCommits 获取提交列表（Mock实现）
func (m *MockGitGatewayClient) ListCommits(ctx context.Context, repositoryID uuid.UUID, branch string, page, pageSize int) (*client.CommitListResponse, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("ListCommits: repo=%s, branch=%s", repositoryID.String(), branch))
	
	if err, exists := m.errors["ListCommits"]; exists {
		return nil, err
	}

	commit := client.Commit{
		ID:             uuid.New(),
		RepositoryID:   repositoryID,
		SHA:            "abc123def456",
		Message:        "Mock commit",
		Author:         "Mock Author",
		AuthorEmail:    "author@example.com",
		Committer:      "Mock Committer",
		CommitterEmail: "committer@example.com",
		ParentSHAs:     []string{"parent123"},
		TreeSHA:        "tree456",
		AddedLines:     5,
		DeletedLines:   1,
		ChangedFiles:   2,
		CommittedAt:    time.Now(),
		CreatedAt:      time.Now(),
	}

	return &client.CommitListResponse{
		Commits:  []client.Commit{commit},
		Total:    1,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetCommitDiff 获取提交差异（Mock实现）
func (m *MockGitGatewayClient) GetCommitDiff(ctx context.Context, repositoryID uuid.UUID, sha string) (*client.GitDiff, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetCommitDiff: repo=%s, sha=%s", repositoryID.String(), sha))
	
	if err, exists := m.errors["GetCommitDiff"]; exists {
		return nil, err
	}

	return &client.GitDiff{
		FromSHA: "parent123",
		ToSHA:   sha,
		Files: []client.DiffFile{
			{
				Path:         "file.txt",
				Status:       "modified",
				AddedLines:   5,
				DeletedLines: 1,
				Patch:        "@@ -1,1 +1,5 @@\n-old content\n+new content",
			},
		},
		TotalAdded:   5,
		TotalDeleted: 1,
	}, nil
}

// CreateTag 创建标签（Mock实现）
func (m *MockGitGatewayClient) CreateTag(ctx context.Context, repositoryID uuid.UUID, req *client.CreateTagRequest) (*client.Tag, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("CreateTag: repo=%s, tag=%s", repositoryID.String(), req.Name))
	
	if err, exists := m.errors["CreateTag"]; exists {
		return nil, err
	}

	tag := &client.Tag{
		ID:           uuid.New(),
		RepositoryID: repositoryID,
		Name:         req.Name,
		CommitSHA:    req.CommitSHA,
		Message:      req.Message,
		Tagger:       req.Tagger.Name,
		TaggerEmail:  req.Tagger.Email,
		TaggedAt:     time.Now(),
		CreatedAt:    time.Now(),
	}

	return tag, nil
}

// GetTag 获取标签详情（Mock实现）
func (m *MockGitGatewayClient) GetTag(ctx context.Context, repositoryID uuid.UUID, tagName string) (*client.Tag, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetTag: repo=%s, tag=%s", repositoryID.String(), tagName))
	
	if err, exists := m.errors["GetTag"]; exists {
		return nil, err
	}

	tag := &client.Tag{
		ID:           uuid.New(),
		RepositoryID: repositoryID,
		Name:         tagName,
		CommitSHA:    "abc123",
		Message:      stringPtr("Mock tag message"),
		Tagger:       "Mock Tagger",
		TaggerEmail:  "tagger@example.com",
		TaggedAt:     time.Now(),
		CreatedAt:    time.Now(),
	}

	return tag, nil
}

// ListTags 获取标签列表（Mock实现）
func (m *MockGitGatewayClient) ListTags(ctx context.Context, repositoryID uuid.UUID) ([]client.Tag, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("ListTags: repo=%s", repositoryID.String()))
	
	if err, exists := m.errors["ListTags"]; exists {
		return nil, err
	}

	tag := client.Tag{
		ID:           uuid.New(),
		RepositoryID: repositoryID,
		Name:         "v1.0.0",
		CommitSHA:    "abc123",
		Message:      stringPtr("Release v1.0.0"),
		Tagger:       "Mock Tagger",
		TaggerEmail:  "tagger@example.com",
		TaggedAt:     time.Now(),
		CreatedAt:    time.Now(),
	}

	return []client.Tag{tag}, nil
}

// DeleteTag 删除标签（Mock实现）
func (m *MockGitGatewayClient) DeleteTag(ctx context.Context, repositoryID uuid.UUID, tagName string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("DeleteTag: repo=%s, tag=%s", repositoryID.String(), tagName))
	
	if err, exists := m.errors["DeleteTag"]; exists {
		return err
	}

	return nil
}

// GetFileContent 获取文件内容（Mock实现）
func (m *MockGitGatewayClient) GetFileContent(ctx context.Context, repositoryID uuid.UUID, branch, filePath string) ([]byte, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetFileContent: repo=%s, branch=%s, path=%s", repositoryID.String(), branch, filePath))
	
	if err, exists := m.errors["GetFileContent"]; exists {
		return nil, err
	}

	return []byte("Mock file content"), nil
}

// GetDirectoryContent 获取目录内容（Mock实现）
func (m *MockGitGatewayClient) GetDirectoryContent(ctx context.Context, repositoryID uuid.UUID, branch, dirPath string) ([]client.FileInfo, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetDirectoryContent: repo=%s, branch=%s, path=%s", repositoryID.String(), branch, dirPath))
	
	if err, exists := m.errors["GetDirectoryContent"]; exists {
		return nil, err
	}

	files := []client.FileInfo{
		{
			Name: "README.md",
			Path: "README.md",
			Type: "file",
			Size: 1024,
			Mode: "100644",
			SHA:  "abc123",
		},
		{
			Name: "src",
			Path: "src",
			Type: "directory",
			Size: 0,
			Mode: "040000",
			SHA:  "def456",
		},
	}

	return files, nil
}

// GetRepositoryStats 获取仓库统计信息（Mock实现）
func (m *MockGitGatewayClient) GetRepositoryStats(ctx context.Context, repositoryID uuid.UUID) (*client.RepositoryStats, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetRepositoryStats: repo=%s", repositoryID.String()))
	
	if err, exists := m.errors["GetRepositoryStats"]; exists {
		return nil, err
	}

	now := time.Now().Add(-1 * time.Hour)
	stats := &client.RepositoryStats{
		Size:         1024000,
		CommitCount:  50,
		BranchCount:  3,
		TagCount:     2,
		LastPushedAt: &now,
	}

	return stats, nil
}

// SearchRepositories 搜索仓库（Mock实现）
func (m *MockGitGatewayClient) SearchRepositories(ctx context.Context, query string, projectID *uuid.UUID, page, pageSize int) (*client.RepositoryListResponse, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("SearchRepositories: query=%s", query))
	
	if err, exists := m.errors["SearchRepositories"]; exists {
		return nil, err
	}

	var repos []*client.Repository
	for _, repo := range m.repositories {
		if projectID == nil || repo.ProjectID == *projectID {
			if strings.Contains(strings.ToLower(repo.Name), strings.ToLower(query)) {
				repos = append(repos, repo)
			}
		}
	}

	// Convert []*Repository to []Repository for search results
	result := make([]client.Repository, len(repos))
	for i, repo := range repos {
		result[i] = *repo
	}

	return &client.RepositoryListResponse{
		Repositories: result,
		Total:        int64(len(repos)),
		Page:         page,
		PageSize:     pageSize,
	}, nil
}

// MockEventProcessor Mock事件处理器（用于集成测试）
type MockEventProcessor struct {
	logger *zap.Logger
}

// ProcessRepositoryEvent 处理仓库事件
func (m *MockEventProcessor) ProcessRepositoryEvent(ctx context.Context, event *webhook.GitEvent, payload *webhook.RepositoryEvent) error {
	m.logger.Info("Processing repository event",
		zap.String("event_id", event.EventID),
		zap.String("action", payload.Action),
		zap.String("repository_id", payload.Repository.ID))
	return nil
}

// ProcessBranchEvent 处理分支事件
func (m *MockEventProcessor) ProcessBranchEvent(ctx context.Context, event *webhook.GitEvent, payload *webhook.BranchEvent) error {
	m.logger.Info("Processing branch event",
		zap.String("event_id", event.EventID),
		zap.String("action", payload.Action),
		zap.String("branch_name", payload.Branch.Name))
	return nil
}

// ProcessCommitEvent 处理提交事件
func (m *MockEventProcessor) ProcessCommitEvent(ctx context.Context, event *webhook.GitEvent, payload *webhook.CommitEvent) error {
	m.logger.Info("Processing commit event",
		zap.String("event_id", event.EventID),
		zap.String("action", payload.Action),
		zap.String("commit_sha", payload.Commit.SHA))
	return nil
}

// ProcessPushEvent 处理推送事件
func (m *MockEventProcessor) ProcessPushEvent(ctx context.Context, event *webhook.GitEvent, payload *webhook.PushEvent) error {
	m.logger.Info("Processing push event",
		zap.String("event_id", event.EventID),
		zap.String("repository_id", payload.RepositoryID),
		zap.String("branch", payload.Branch))
	return nil
}

// ProcessTagEvent 处理标签事件
func (m *MockEventProcessor) ProcessTagEvent(ctx context.Context, event *webhook.GitEvent, payload *webhook.TagEvent) error {
	m.logger.Info("Processing tag event",
		zap.String("event_id", event.EventID),
		zap.String("action", payload.Action),
		zap.String("tag_name", payload.Tag.Name))
	return nil
}

// SetupSuite 初始化测试套件
func (suite *ProjectServiceIntegrationTestSuite) SetupSuite() {
	// 设置环境变量
	os.Setenv("ENVIRONMENT", "test")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_NAME", "collaborative_dev_test")
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_PASSWORD", "postgres")
	
	var err error
	
	// 初始化日志
	suite.logger, err = zap.NewDevelopment()
	require.NoError(suite.T(), err, "日志初始化失败")
	
	// 初始化数据库连接（使用简化配置）
	dbConfig := database.Config{
		Host:     getSimpleEnvOrDefault("TEST_DB_HOST", "localhost"),
		Port:     5432,
		Name:     getSimpleEnvOrDefault("TEST_DB_NAME", "collaborative_dev_test"),
		User:     getSimpleEnvOrDefault("TEST_DB_USER", "postgres"),
		Password: getSimpleEnvOrDefault("TEST_DB_PASSWORD", "postgres"),
		SSLMode:  "disable",
	}
	
	pgDB, err := database.NewPostgresDB(dbConfig)
	require.NoError(suite.T(), err, "数据库连接失败")
	suite.db = pgDB.DB
	
	// 创建数据表（如果不存在）
	err = suite.createTables()
	require.NoError(suite.T(), err, "创建数据表失败")
	
	// 初始化Mock客户端
	suite.gitClient = NewMockGitGatewayClient()
	
	// 初始化仓库
	suite.repo = repository.NewProjectRepository(suite.db)
	
	// 初始化补偿管理器
	suite.compensationMgr = compensation.NewCompensationManager(suite.gitClient, suite.logger)
	
	// 初始化分布式事务管理器
	suite.transactionMgr = transaction.NewDistributedTransactionManager(
		suite.repo,
		suite.gitClient,
		suite.compensationMgr,
		suite.logger,
	)
	
	// 初始化项目服务
	suite.projectService = service.NewProjectServiceWithTransaction(
		suite.repo,
		suite.gitClient,
		suite.transactionMgr,
		suite.logger,
	)
	
	// 初始化Mock事件处理器
	eventProcessor := &MockEventProcessor{logger: suite.logger}
	
	// 初始化Webhook处理器
	suite.webhookHandler = webhook.NewWebhookHandler(eventProcessor, "", suite.logger)
	
	// 初始化项目处理器
	suite.projectHandler = handlers.NewProjectHandler(
		suite.projectService,
		suite.webhookHandler,
		suite.logger,
	)
	
	// 设置Gin为测试模式
	gin.SetMode(gin.TestMode)
	
	// 初始化路由
	suite.setupRoutes()
	
	// 创建测试服务器
	suite.server = httptest.NewServer(suite.router)
	
	// 初始化测试数据
	suite.setupTestData()
	
	suite.logger.Info("项目服务集成测试套件初始化完成")
}

// createTables 创建测试所需的数据表
func (suite *ProjectServiceIntegrationTestSuite) createTables() error {
	// 创建项目表
	err := suite.db.Exec(`
		CREATE TABLE IF NOT EXISTS projects (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL,
			key VARCHAR(20) UNIQUE NOT NULL,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			status VARCHAR(20) DEFAULT 'active',
			manager_id UUID,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		return err
	}

	// 创建项目成员表
	err = suite.db.Exec(`
		CREATE TABLE IF NOT EXISTS project_members (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			project_id UUID NOT NULL,
			user_id UUID NOT NULL,
			role_id UUID NOT NULL,
			added_by UUID,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(project_id, user_id)
		)
	`).Error
	if err != nil {
		return err
	}

	return nil
}

// setupRoutes 设置路由
func (suite *ProjectServiceIntegrationTestSuite) setupRoutes() {
	suite.router = gin.New()
	
	// 添加中间件
	suite.router.Use(gin.Recovery())
	suite.router.Use(suite.mockAuthMiddleware())
	
	// API路由组
	api := suite.router.Group("/api/v1")
	
	// 项目管理路由
	projects := api.Group("/projects")
	{
		projects.POST("", suite.projectHandler.CreateProject)
		projects.GET("", suite.projectHandler.ListProjects)
		projects.GET("/my", suite.projectHandler.GetUserProjects)
		projects.GET("/:id", suite.projectHandler.GetProject)
		projects.PUT("/:id", suite.projectHandler.UpdateProject)
		projects.DELETE("/:id", suite.projectHandler.DeleteProject)
		projects.GET("/key/:key", suite.projectHandler.GetProjectByKey)
		
		// 成员管理
		projects.POST("/:id/members", suite.projectHandler.AddMember)
		projects.GET("/:id/members", suite.projectHandler.GetMembers)
		projects.DELETE("/:id/members/:user_id", suite.projectHandler.RemoveMember)
		
		// 仓库管理
		projects.POST("/:id/repositories", suite.projectHandler.CreateRepository)
		projects.GET("/:id/repositories", suite.projectHandler.ListRepositories)
	}
	
	// 仓库管理路由
	repositories := api.Group("/repositories")
	{
		repositories.GET("/:repository_id", suite.projectHandler.GetRepository)
		repositories.PUT("/:repository_id", suite.projectHandler.UpdateRepository)
		repositories.DELETE("/:repository_id", suite.projectHandler.DeleteRepository)
	}
	
	// Webhook路由
	webhooks := api.Group("/webhooks")
	{
		webhooks.POST("/git", suite.projectHandler.HandleGitWebhook)
		webhooks.GET("/health", suite.projectHandler.GetWebhookHealth)
	}
}

// mockAuthMiddleware Mock认证中间件
func (suite *ProjectServiceIntegrationTestSuite) mockAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		
		switch authHeader {
		case "Bearer admin-token":
			c.Set("user_id", suite.testAdminUserID)
			c.Set("tenant_id", suite.testTenantID)
		case "Bearer member-token":
			c.Set("user_id", suite.testMemberUserID)
			c.Set("tenant_id", suite.testTenantID)
		case "Bearer user-token":
			c.Set("user_id", suite.testUserID)
			c.Set("tenant_id", suite.testTenantID)
		default:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// setupTestData 设置测试数据
func (suite *ProjectServiceIntegrationTestSuite) setupTestData() {
	// 生成测试ID
	suite.testTenantID = uuid.New()
	suite.testUserID = uuid.New()
	suite.testAdminUserID = uuid.New()
	suite.testMemberUserID = uuid.New()
	
	// 设置认证令牌
	suite.authToken = "Bearer user-token"
	suite.adminAuthToken = "Bearer admin-token"
	suite.memberAuthToken = "Bearer member-token"
	
	// 清理现有数据
	suite.cleanupTestData()
}

// cleanupTestData 清理测试数据
func (suite *ProjectServiceIntegrationTestSuite) cleanupTestData() {
	// 清理数据库
	suite.db.Exec("DELETE FROM project_members WHERE 1=1")
	suite.db.Exec("DELETE FROM projects WHERE 1=1")
	
	// 清理Mock客户端
	suite.gitClient.repositories = make(map[uuid.UUID]*client.Repository)
	suite.gitClient.branches = make(map[uuid.UUID][]*client.Branch)
	suite.gitClient.errors = make(map[string]error)
	suite.gitClient.ClearCallLog()
}

// TearDownSuite 清理测试套件
func (suite *ProjectServiceIntegrationTestSuite) TearDownSuite() {
	if suite.server != nil {
		suite.server.Close()
	}
	
	// 清理测试数据
	suite.cleanupTestData()
	
	// 关闭数据库连接
	if sqlDB, err := suite.db.DB(); err == nil {
		sqlDB.Close()
	}
	
	suite.logger.Info("项目服务集成测试套件清理完成")
}

// SetupTest 每个测试前的初始化
func (suite *ProjectServiceIntegrationTestSuite) SetupTest() {
	// 清理测试数据
	suite.cleanupTestData()
}

// makeRequest 发送HTTP请求的辅助方法
func (suite *ProjectServiceIntegrationTestSuite) makeRequest(method, path string, body interface{}, authToken string) (*http.Response, []byte) {
	var bodyReader io.Reader
	
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		require.NoError(suite.T(), err, "序列化请求体失败")
		bodyReader = bytes.NewBuffer(bodyBytes)
	}
	
	req, err := http.NewRequest(method, suite.server.URL+path, bodyReader)
	require.NoError(suite.T(), err, "创建请求失败")
	
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	
	if authToken != "" {
		req.Header.Set("Authorization", authToken)
	}
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(suite.T(), err, "发送请求失败")
	
	responseBody, err := io.ReadAll(resp.Body)
	require.NoError(suite.T(), err, "读取响应失败")
	resp.Body.Close()
	
	return resp, responseBody
}

// TestProjectLifecycle 测试项目完整生命周期
func (suite *ProjectServiceIntegrationTestSuite) TestProjectLifecycle() {
	suite.Run("完整项目生命周期测试", func() {
		// 1. 创建项目
		createReq := models.CreateProjectRequest{
			Key:         "test-project",
			Name:        "测试项目",
			Description: stringPtr("这是一个测试项目"),
		}
		
		resp, body := suite.makeRequest("POST", "/api/v1/projects", createReq, suite.authToken)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
		
		var createResponse response.Response
		err := json.Unmarshal(body, &createResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 200, createResponse.Code)
		
		// 解析创建的项目
		projectData, err := json.Marshal(createResponse.Data)
		assert.NoError(suite.T(), err)
		
		var project models.Project
		err = json.Unmarshal(projectData, &project)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), "test-project", project.Key)
		assert.Equal(suite.T(), "测试项目", project.Name)
		
		projectID := project.ID
		
		// 2. 获取项目详情
		resp, body = suite.makeRequest("GET", fmt.Sprintf("/api/v1/projects/%s", projectID), nil, suite.authToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var getResponse response.Response
		err = json.Unmarshal(body, &getResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 200, getResponse.Code)
		
		// 3. 更新项目
		updateReq := models.UpdateProjectRequest{
			Name:        stringPtr("更新的测试项目"),
			Description: stringPtr("这是一个更新的测试项目"),
		}
		
		resp, body = suite.makeRequest("PUT", fmt.Sprintf("/api/v1/projects/%s", projectID), updateReq, suite.authToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var updateResponse response.Response
		err = json.Unmarshal(body, &updateResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 200, updateResponse.Code)
		
		// 4. 获取项目列表
		resp, body = suite.makeRequest("GET", "/api/v1/projects", nil, suite.authToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var listResponse response.Response
		err = json.Unmarshal(body, &listResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 200, listResponse.Code)
		
		// 5. 根据key获取项目
		resp, body = suite.makeRequest("GET", "/api/v1/projects/key/test-project", nil, suite.authToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var getByKeyResponse response.Response
		err = json.Unmarshal(body, &getByKeyResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 200, getByKeyResponse.Code)
		
		// 6. 删除项目
		resp, body = suite.makeRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s", projectID), nil, suite.authToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var deleteResponse response.Response
		err = json.Unmarshal(body, &deleteResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 200, deleteResponse.Code)
		
		// 7. 验证项目已删除
		resp, body = suite.makeRequest("GET", fmt.Sprintf("/api/v1/projects/%s", projectID), nil, suite.authToken)
		assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)
	})
}

// TestProjectMemberManagement 测试项目成员管理
func (suite *ProjectServiceIntegrationTestSuite) TestProjectMemberManagement() {
	suite.Run("项目成员管理测试", func() {
		// 1. 创建项目（使用管理员账户）
		createReq := models.CreateProjectRequest{
			Key:         "member-test",
			Name:        "成员管理测试项目",
			Description: stringPtr("用于测试成员管理功能"),
		}
		
		resp, body := suite.makeRequest("POST", "/api/v1/projects", createReq, suite.adminAuthToken)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
		
		var createResponse response.Response
		err := json.Unmarshal(body, &createResponse)
		assert.NoError(suite.T(), err)
		
		projectData, err := json.Marshal(createResponse.Data)
		assert.NoError(suite.T(), err)
		
		var project models.Project
		err = json.Unmarshal(projectData, &project)
		assert.NoError(suite.T(), err)
		
		projectID := project.ID
		
		// 2. 添加项目成员
		addMemberReq := models.AddMemberRequest{
			UserID: suite.testMemberUserID.String(),
			RoleID: uuid.New().String(), // 使用随机角色ID
		}
		
		resp, body = suite.makeRequest("POST", fmt.Sprintf("/api/v1/projects/%s/members", projectID), addMemberReq, suite.adminAuthToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var addMemberResponse response.Response
		err = json.Unmarshal(body, &addMemberResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 200, addMemberResponse.Code)
		
		// 3. 获取项目成员列表
		resp, body = suite.makeRequest("GET", fmt.Sprintf("/api/v1/projects/%s/members", projectID), nil, suite.adminAuthToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var getMembersResponse response.Response
		err = json.Unmarshal(body, &getMembersResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 200, getMembersResponse.Code)
		
		// 4. 移除项目成员
		resp, body = suite.makeRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s/members/%s", projectID, suite.testMemberUserID), nil, suite.adminAuthToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var removeMemberResponse response.Response
		err = json.Unmarshal(body, &removeMemberResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 200, removeMemberResponse.Code)
		
		// 5. 验证成员已移除
		resp, body = suite.makeRequest("GET", fmt.Sprintf("/api/v1/projects/%s/members", projectID), nil, suite.adminAuthToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		err = json.Unmarshal(body, &getMembersResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 200, getMembersResponse.Code)
	})
}

// TestRepositoryLifecycle 测试仓库完整生命周期
func (suite *ProjectServiceIntegrationTestSuite) TestRepositoryLifecycle() {
	suite.Run("仓库生命周期测试", func() {
		// 1. 创建项目
		createProjectReq := models.CreateProjectRequest{
			Key:         "repo-test",
			Name:        "仓库测试项目",
			Description: stringPtr("用于测试仓库管理功能"),
		}
		
		resp, body := suite.makeRequest("POST", "/api/v1/projects", createProjectReq, suite.authToken)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
		
		var createProjectResponse response.Response
		err := json.Unmarshal(body, &createProjectResponse)
		assert.NoError(suite.T(), err)
		
		projectData, err := json.Marshal(createProjectResponse.Data)
		assert.NoError(suite.T(), err)
		
		var project models.Project
		err = json.Unmarshal(projectData, &project)
		assert.NoError(suite.T(), err)
		
		projectID := project.ID
		
		// 2. 创建仓库
		createRepoReq := service.CreateRepositoryRequest{
			Name:        "test-repo",
			Description: stringPtr("测试仓库"),
			Visibility:  "private",
			InitReadme:  true,
		}
		
		resp, body = suite.makeRequest("POST", fmt.Sprintf("/api/v1/projects/%s/repositories", projectID), createRepoReq, suite.authToken)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
		
		var createRepoResponse response.Response
		err = json.Unmarshal(body, &createRepoResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 200, createRepoResponse.Code)
		
		// 解析创建的仓库
		repoData, err := json.Marshal(createRepoResponse.Data)
		assert.NoError(suite.T(), err)
		
		var repository models.Repository
		err = json.Unmarshal(repoData, &repository)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), "test-repo", repository.Name)
		
		repositoryID := repository.ID
		
		// 验证Git网关被调用
		callLog := suite.gitClient.GetCallLog()
		assert.Contains(suite.T(), callLog, "CreateRepository: test-repo")
		
		// 3. 获取仓库详情
		resp, body = suite.makeRequest("GET", fmt.Sprintf("/api/v1/repositories/%s", repositoryID), nil, suite.authToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var getRepoResponse response.Response
		err = json.Unmarshal(body, &getRepoResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 200, getRepoResponse.Code)
		
		// 4. 获取项目仓库列表
		resp, body = suite.makeRequest("GET", fmt.Sprintf("/api/v1/projects/%s/repositories", projectID), nil, suite.authToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var listRepoResponse response.Response
		err = json.Unmarshal(body, &listRepoResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 200, listRepoResponse.Code)
		
		// 5. 更新仓库
		updateRepoReq := service.UpdateRepositoryRequest{
			Name:        stringPtr("updated-test-repo"),
			Description: stringPtr("更新的测试仓库"),
		}
		
		resp, body = suite.makeRequest("PUT", fmt.Sprintf("/api/v1/repositories/%s", repositoryID), updateRepoReq, suite.authToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var updateRepoResponse response.Response
		err = json.Unmarshal(body, &updateRepoResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 200, updateRepoResponse.Code)
		
		// 6. 删除仓库
		resp, body = suite.makeRequest("DELETE", fmt.Sprintf("/api/v1/repositories/%s", repositoryID), nil, suite.authToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var deleteRepoResponse response.Response
		err = json.Unmarshal(body, &deleteRepoResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 200, deleteRepoResponse.Code)
		
		// 验证Git网关删除调用
		callLog = suite.gitClient.GetCallLog()
		assert.Contains(suite.T(), callLog, fmt.Sprintf("DeleteRepository: %s", repositoryID))
		
		// 7. 验证仓库已删除
		resp, body = suite.makeRequest("GET", fmt.Sprintf("/api/v1/repositories/%s", repositoryID), nil, suite.authToken)
		assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)
	})
}

// stringPtr 返回字符串指针的辅助函数
func stringPtr(s string) *string {
	return &s
}


// 运行集成测试套件
func TestProjectServiceIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ProjectServiceIntegrationTestSuite))
}