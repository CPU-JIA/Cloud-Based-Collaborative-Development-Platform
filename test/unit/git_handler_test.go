package unit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/client"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/handler"
	"github.com/cloud-platform/collaborative-dev/shared/response"
)

// MockGitGatewayClient Git网关客户端模拟
type MockGitGatewayClient struct {
	mock.Mock
}

// Repository operations
func (m *MockGitGatewayClient) CreateRepository(ctx context.Context, req *client.CreateRepositoryRequest) (*client.Repository, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.Repository), args.Error(1)
}

func (m *MockGitGatewayClient) GetRepository(ctx context.Context, repositoryID uuid.UUID) (*client.Repository, error) {
	args := m.Called(ctx, repositoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.Repository), args.Error(1)
}

func (m *MockGitGatewayClient) UpdateRepository(ctx context.Context, repositoryID uuid.UUID, req *client.UpdateRepositoryRequest) (*client.Repository, error) {
	args := m.Called(ctx, repositoryID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.Repository), args.Error(1)
}

func (m *MockGitGatewayClient) DeleteRepository(ctx context.Context, repositoryID uuid.UUID) error {
	args := m.Called(ctx, repositoryID)
	return args.Error(0)
}

func (m *MockGitGatewayClient) ListRepositories(ctx context.Context, projectID *uuid.UUID, page, pageSize int) (*client.RepositoryListResponse, error) {
	args := m.Called(ctx, projectID, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.RepositoryListResponse), args.Error(1)
}

// Branch operations
func (m *MockGitGatewayClient) CreateBranch(ctx context.Context, repositoryID uuid.UUID, req *client.CreateBranchRequest) (*client.Branch, error) {
	args := m.Called(ctx, repositoryID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.Branch), args.Error(1)
}

func (m *MockGitGatewayClient) GetBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) (*client.Branch, error) {
	args := m.Called(ctx, repositoryID, branchName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.Branch), args.Error(1)
}

func (m *MockGitGatewayClient) ListBranches(ctx context.Context, repositoryID uuid.UUID) ([]client.Branch, error) {
	args := m.Called(ctx, repositoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]client.Branch), args.Error(1)
}

func (m *MockGitGatewayClient) DeleteBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) error {
	args := m.Called(ctx, repositoryID, branchName)
	return args.Error(0)
}

func (m *MockGitGatewayClient) SetDefaultBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) error {
	args := m.Called(ctx, repositoryID, branchName)
	return args.Error(0)
}

func (m *MockGitGatewayClient) MergeBranch(ctx context.Context, repositoryID uuid.UUID, targetBranch, sourceBranch string) error {
	args := m.Called(ctx, repositoryID, targetBranch, sourceBranch)
	return args.Error(0)
}

// Commit operations
func (m *MockGitGatewayClient) CreateCommit(ctx context.Context, repositoryID uuid.UUID, req *client.CreateCommitRequest) (*client.Commit, error) {
	args := m.Called(ctx, repositoryID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.Commit), args.Error(1)
}

func (m *MockGitGatewayClient) GetCommit(ctx context.Context, repositoryID uuid.UUID, sha string) (*client.Commit, error) {
	args := m.Called(ctx, repositoryID, sha)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.Commit), args.Error(1)
}

func (m *MockGitGatewayClient) ListCommits(ctx context.Context, repositoryID uuid.UUID, branch string, page, pageSize int) (*client.CommitListResponse, error) {
	args := m.Called(ctx, repositoryID, branch, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.CommitListResponse), args.Error(1)
}

func (m *MockGitGatewayClient) GetCommitDiff(ctx context.Context, repositoryID uuid.UUID, sha string) (*client.GitDiff, error) {
	args := m.Called(ctx, repositoryID, sha)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.GitDiff), args.Error(1)
}

func (m *MockGitGatewayClient) CompareBranches(ctx context.Context, repositoryID uuid.UUID, base, head string) (*client.GitDiff, error) {
	args := m.Called(ctx, repositoryID, base, head)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.GitDiff), args.Error(1)
}

// Tag operations
func (m *MockGitGatewayClient) CreateTag(ctx context.Context, repositoryID uuid.UUID, req *client.CreateTagRequest) (*client.Tag, error) {
	args := m.Called(ctx, repositoryID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.Tag), args.Error(1)
}

func (m *MockGitGatewayClient) GetTag(ctx context.Context, repositoryID uuid.UUID, tagName string) (*client.Tag, error) {
	args := m.Called(ctx, repositoryID, tagName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.Tag), args.Error(1)
}

func (m *MockGitGatewayClient) ListTags(ctx context.Context, repositoryID uuid.UUID) ([]client.Tag, error) {
	args := m.Called(ctx, repositoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]client.Tag), args.Error(1)
}

func (m *MockGitGatewayClient) DeleteTag(ctx context.Context, repositoryID uuid.UUID, tagName string) error {
	args := m.Called(ctx, repositoryID, tagName)
	return args.Error(0)
}

// File operations
func (m *MockGitGatewayClient) GetFileContent(ctx context.Context, repositoryID uuid.UUID, branch, filePath string) ([]byte, error) {
	args := m.Called(ctx, repositoryID, branch, filePath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockGitGatewayClient) GetDirectoryContent(ctx context.Context, repositoryID uuid.UUID, branch, dirPath string) ([]client.FileInfo, error) {
	args := m.Called(ctx, repositoryID, branch, dirPath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]client.FileInfo), args.Error(1)
}

// Stats and search
func (m *MockGitGatewayClient) GetRepositoryStats(ctx context.Context, repositoryID uuid.UUID) (*client.RepositoryStats, error) {
	args := m.Called(ctx, repositoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.RepositoryStats), args.Error(1)
}

func (m *MockGitGatewayClient) SearchRepositories(ctx context.Context, query string, projectID *uuid.UUID, page, pageSize int) (*client.RepositoryListResponse, error) {
	args := m.Called(ctx, query, projectID, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.RepositoryListResponse), args.Error(1)
}

// GitHandlerTestSuite Git处理器测试套件
type GitHandlerTestSuite struct {
	suite.Suite
	handler          *handler.GitHandler
	mockGitClient    *MockGitGatewayClient
	router           *gin.Engine
	logger           *zap.Logger
	testRepositoryID uuid.UUID
	testProjectID    uuid.UUID
	testTenantID     uuid.UUID
	testUserID       uuid.UUID
}

func (suite *GitHandlerTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	suite.logger = zaptest.NewLogger(suite.T())
	
	suite.testRepositoryID = uuid.New()
	suite.testProjectID = uuid.New()
	suite.testTenantID = uuid.New()
	suite.testUserID = uuid.New()
}

func (suite *GitHandlerTestSuite) SetupTest() {
	suite.mockGitClient = new(MockGitGatewayClient)
	suite.handler = handler.NewGitHandler(suite.mockGitClient, suite.logger)
	
	suite.router = gin.New()
	suite.setupRoutes()
}

func (suite *GitHandlerTestSuite) TearDownTest() {
	suite.mockGitClient.AssertExpectations(suite.T())
}

func (suite *GitHandlerTestSuite) setupRoutes() {
	api := suite.router.Group("/api/v1")
	{
		git := api.Group("/git")
		{
			// Repository routes
			repos := git.Group("/repositories")
			{
				repos.GET("/:id", suite.addAuthContext, suite.handler.GetRepository)
				repos.PUT("/:id", suite.addAuthContext, suite.handler.UpdateRepository)
				repos.DELETE("/:id", suite.addAuthContext, suite.handler.DeleteRepository)
				repos.GET("/:id/stats", suite.addAuthContext, suite.handler.GetRepositoryStats)
				
				// Branch routes
				repos.GET("/:id/branches", suite.addAuthContext, suite.handler.ListBranches)
				repos.POST("/:id/branches", suite.addAuthContext, suite.handler.CreateBranch)
				repos.GET("/:id/branches/:name", suite.addAuthContext, suite.handler.GetBranch)
				repos.DELETE("/:id/branches/:name", suite.addAuthContext, suite.handler.DeleteBranch)
				repos.PUT("/:id/default-branch", suite.addAuthContext, suite.handler.SetDefaultBranch)
				repos.POST("/:id/merge", suite.addAuthContext, suite.handler.MergeBranches)
				
				// Commit routes
				repos.GET("/:id/commits", suite.addAuthContext, suite.handler.ListCommits)
				repos.POST("/:id/commits", suite.addAuthContext, suite.handler.CreateCommit)
				repos.GET("/:id/commits/:sha", suite.addAuthContext, suite.handler.GetCommit)
				repos.GET("/:id/commits/:sha/diff", suite.addAuthContext, suite.handler.GetCommitDiff)
				repos.GET("/:id/compare", suite.addAuthContext, suite.handler.CompareBranches)
				
				// Tag routes
				repos.GET("/:id/tags", suite.addAuthContext, suite.handler.ListTags)
				repos.POST("/:id/tags", suite.addAuthContext, suite.handler.CreateTag)
				repos.GET("/:id/tags/:name", suite.addAuthContext, suite.handler.GetTag)
				repos.DELETE("/:id/tags/:name", suite.addAuthContext, suite.handler.DeleteTag)
				
				// File routes
				repos.GET("/:id/files", suite.addAuthContext, suite.handler.GetFileContent)
				repos.GET("/:id/tree", suite.addAuthContext, suite.handler.GetDirectoryContent)
			}
		}
	}
}

func (suite *GitHandlerTestSuite) addAuthContext(c *gin.Context) {
	c.Set("tenant_id", suite.testTenantID.String())
	c.Set("user_id", suite.testUserID.String())
	c.Next()
}

// TestRepositoryOperations 测试仓库操作
func (suite *GitHandlerTestSuite) TestRepositoryOperations() {
	// 测试获取仓库信息
	suite.Run("获取仓库信息", func() {
		repository := &client.Repository{
			ID:            suite.testRepositoryID,
			ProjectID:     suite.testProjectID,
			Name:          "test-repo",
			Description:   &[]string{"测试仓库"}[0],
			Visibility:    client.RepositoryVisibilityPrivate,
			Status:        client.RepositoryStatusActive,
			DefaultBranch: "main",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		suite.mockGitClient.On("GetRepository", mock.Anything, suite.testRepositoryID).
			Return(repository, nil)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/git/repositories/%s", suite.testRepositoryID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
		assert.NotNil(suite.T(), response.Data)
	})

	// 测试更新仓库
	suite.Run("更新仓库", func() {
		updateReq := &client.UpdateRepositoryRequest{
			Name:        &[]string{"updated-repo"}[0],
			Description: &[]string{"更新后的仓库描述"}[0],
			Visibility:  &client.RepositoryVisibilityPublic,
		}

		updatedRepository := &client.Repository{
			ID:          suite.testRepositoryID,
			ProjectID:   suite.testProjectID,
			Name:        "updated-repo",
			Description: &[]string{"更新后的仓库描述"}[0],
			Visibility:  client.RepositoryVisibilityPublic,
			UpdatedAt:   time.Now(),
		}

		suite.mockGitClient.On("UpdateRepository", mock.Anything, suite.testRepositoryID, mock.MatchedBy(func(req *client.UpdateRepositoryRequest) bool {
			return req.Name != nil && *req.Name == "updated-repo" &&
				   req.Description != nil && *req.Description == "更新后的仓库描述" &&
				   req.Visibility != nil && *req.Visibility == client.RepositoryVisibilityPublic
		})).Return(updatedRepository, nil)

		body, _ := json.Marshal(map[string]interface{}{
			"name":        "updated-repo",
			"description": "更新后的仓库描述",
			"visibility":  "public",
		})
		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/git/repositories/%s", suite.testRepositoryID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
	})

	// 测试删除仓库
	suite.Run("删除仓库", func() {
		suite.mockGitClient.On("DeleteRepository", mock.Anything, suite.testRepositoryID).
			Return(nil)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/git/repositories/%s", suite.testRepositoryID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
	})

	// 测试获取仓库统计信息
	suite.Run("获取仓库统计信息", func() {
		stats := &client.RepositoryStats{
			Size:         1024000,
			CommitCount:  150,
			BranchCount:  5,
			TagCount:     3,
			LastPushedAt: &time.Now(),
		}

		suite.mockGitClient.On("GetRepositoryStats", mock.Anything, suite.testRepositoryID).
			Return(stats, nil)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/git/repositories/%s/stats", suite.testRepositoryID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
		assert.NotNil(suite.T(), response.Data)
	})
}

// TestBranchOperations 测试分支操作
func (suite *GitHandlerTestSuite) TestBranchOperations() {
	// 测试获取分支列表
	suite.Run("获取分支列表", func() {
		branches := []client.Branch{
			{
				ID:           uuid.New(),
				RepositoryID: suite.testRepositoryID,
				Name:         "main",
				CommitSHA:    "abc123def456",
				IsDefault:    true,
				IsProtected:  true,
				CreatedAt:    time.Now(),
			},
			{
				ID:           uuid.New(),
				RepositoryID: suite.testRepositoryID,
				Name:         "develop",
				CommitSHA:    "def456ghi789",
				IsDefault:    false,
				IsProtected:  false,
				CreatedAt:    time.Now(),
			},
		}

		suite.mockGitClient.On("ListBranches", mock.Anything, suite.testRepositoryID).
			Return(branches, nil)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/git/repositories/%s/branches", suite.testRepositoryID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
		assert.NotNil(suite.T(), response.Data)
	})

	// 测试创建分支
	suite.Run("创建分支", func() {
		newBranch := &client.Branch{
			ID:           uuid.New(),
			RepositoryID: suite.testRepositoryID,
			Name:         "feature/new-feature",
			CommitSHA:    "abc123def456",
			IsDefault:    false,
			IsProtected:  false,
			CreatedAt:    time.Now(),
		}

		suite.mockGitClient.On("CreateBranch", mock.Anything, suite.testRepositoryID, mock.MatchedBy(func(req *client.CreateBranchRequest) bool {
			return req.Name == "feature/new-feature" && req.FromSHA == "abc123def456"
		})).Return(newBranch, nil)

		body, _ := json.Marshal(map[string]interface{}{
			"name":     "feature/new-feature",
			"from_sha": "abc123def456",
		})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/git/repositories/%s/branches", suite.testRepositoryID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusCreated, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
	})

	// 测试获取分支详情
	suite.Run("获取分支详情", func() {
		branch := &client.Branch{
			ID:           uuid.New(),
			RepositoryID: suite.testRepositoryID,
			Name:         "main",
			CommitSHA:    "abc123def456",
			IsDefault:    true,
			IsProtected:  true,
			CreatedAt:    time.Now(),
		}

		suite.mockGitClient.On("GetBranch", mock.Anything, suite.testRepositoryID, "main").
			Return(branch, nil)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/git/repositories/%s/branches/main", suite.testRepositoryID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
	})

	// 测试删除分支
	suite.Run("删除分支", func() {
		suite.mockGitClient.On("DeleteBranch", mock.Anything, suite.testRepositoryID, "feature/old-feature").
			Return(nil)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/git/repositories/%s/branches/feature/old-feature", suite.testRepositoryID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
	})

	// 测试设置默认分支
	suite.Run("设置默认分支", func() {
		suite.mockGitClient.On("SetDefaultBranch", mock.Anything, suite.testRepositoryID, "develop").
			Return(nil)

		body, _ := json.Marshal(map[string]interface{}{
			"branch_name": "develop",
		})
		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/git/repositories/%s/default-branch", suite.testRepositoryID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
	})

	// 测试合并分支
	suite.Run("合并分支", func() {
		suite.mockGitClient.On("MergeBranch", mock.Anything, suite.testRepositoryID, "main", "feature/new-feature").
			Return(nil)

		body, _ := json.Marshal(map[string]interface{}{
			"target_branch": "main",
			"source_branch": "feature/new-feature",
		})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/git/repositories/%s/merge", suite.testRepositoryID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
	})
}

// TestCommitOperations 测试提交操作
func (suite *GitHandlerTestSuite) TestCommitOperations() {
	// 测试获取提交列表
	suite.Run("获取提交列表", func() {
		commits := []client.Commit{
			{
				ID:           uuid.New(),
				RepositoryID: suite.testRepositoryID,
				SHA:          "abc123def456",
				Message:      "Initial commit",
				Author:       "John Doe",
				AuthorEmail:  "john@example.com",
				CommittedAt:  time.Now(),
				AddedLines:   100,
				DeletedLines: 0,
				ChangedFiles: 5,
			},
			{
				ID:           uuid.New(),
				RepositoryID: suite.testRepositoryID,
				SHA:          "def456ghi789",
				Message:      "Add new feature",
				Author:       "Jane Smith",
				AuthorEmail:  "jane@example.com",
				CommittedAt:  time.Now(),
				AddedLines:   50,
				DeletedLines: 10,
				ChangedFiles: 3,
			},
		}

		commitsResponse := &client.CommitListResponse{
			Commits:  commits,
			Total:    2,
			Page:     1,
			PageSize: 20,
		}

		suite.mockGitClient.On("ListCommits", mock.Anything, suite.testRepositoryID, "main", 1, 20).
			Return(commitsResponse, nil)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/git/repositories/%s/commits?branch=main&page=1&page_size=20", suite.testRepositoryID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
		assert.NotNil(suite.T(), response.Data)
	})

	// 测试创建提交
	suite.Run("创建提交", func() {
		newCommit := &client.Commit{
			ID:           uuid.New(),
			RepositoryID: suite.testRepositoryID,
			SHA:          "ghi789jkl012",
			Message:      "Fix bug in authentication",
			Author:       "Developer",
			AuthorEmail:  "dev@example.com",
			CommittedAt:  time.Now(),
			AddedLines:   20,
			DeletedLines: 5,
			ChangedFiles: 2,
		}

		suite.mockGitClient.On("CreateCommit", mock.Anything, suite.testRepositoryID, mock.MatchedBy(func(req *client.CreateCommitRequest) bool {
			return req.Branch == "main" && 
				   req.Message == "Fix bug in authentication" &&
				   req.Author.Name == "Developer" &&
				   len(req.Files) == 2
		})).Return(newCommit, nil)

		body, _ := json.Marshal(map[string]interface{}{
			"branch":  "main",
			"message": "Fix bug in authentication",
			"author": map[string]interface{}{
				"name":  "Developer",
				"email": "dev@example.com",
			},
			"files": []map[string]interface{}{
				{
					"path":    "src/auth.go",
					"content": "package auth\n\n// Fixed authentication logic",
				},
				{
					"path":    "src/auth_test.go",
					"content": "package auth\n\n// Updated tests",
				},
			},
		})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/git/repositories/%s/commits", suite.testRepositoryID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusCreated, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
	})

	// 测试获取提交详情
	suite.Run("获取提交详情", func() {
		commit := &client.Commit{
			ID:           uuid.New(),
			RepositoryID: suite.testRepositoryID,
			SHA:          "abc123def456",
			Message:      "Initial commit",
			Author:       "John Doe",
			AuthorEmail:  "john@example.com",
			CommittedAt:  time.Now(),
			AddedLines:   100,
			DeletedLines: 0,
			ChangedFiles: 5,
		}

		suite.mockGitClient.On("GetCommit", mock.Anything, suite.testRepositoryID, "abc123def456").
			Return(commit, nil)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/git/repositories/%s/commits/abc123def456", suite.testRepositoryID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
	})

	// 测试获取提交差异
	suite.Run("获取提交差异", func() {
		diff := &client.GitDiff{
			FromSHA:      "parent123",
			ToSHA:        "abc123def456",
			TotalAdded:   100,
			TotalDeleted: 0,
			Files: []client.DiffFile{
				{
					Path:         "README.md",
					Status:       "added",
					AddedLines:   50,
					DeletedLines: 0,
					Patch:        "+# Project Title\n+This is a new project",
				},
				{
					Path:         "main.go",
					Status:       "added",
					AddedLines:   50,
					DeletedLines: 0,
					Patch:        "+package main\n+\n+func main() { }",
				},
			},
		}

		suite.mockGitClient.On("GetCommitDiff", mock.Anything, suite.testRepositoryID, "abc123def456").
			Return(diff, nil)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/git/repositories/%s/commits/abc123def456/diff", suite.testRepositoryID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
	})

	// 测试比较分支
	suite.Run("比较分支", func() {
		diff := &client.GitDiff{
			FromSHA:      "main123",
			ToSHA:        "feature456",
			TotalAdded:   75,
			TotalDeleted: 25,
			Files: []client.DiffFile{
				{
					Path:         "feature.go",
					Status:       "added",
					AddedLines:   50,
					DeletedLines: 0,
					Patch:        "+package feature\n+\n+func NewFeature() { }",
				},
				{
					Path:         "old_code.go",
					Status:       "modified",
					AddedLines:   25,
					DeletedLines: 25,
					Patch:        "-old code\n+new code",
				},
			},
		}

		suite.mockGitClient.On("CompareBranches", mock.Anything, suite.testRepositoryID, "main", "feature/new-feature").
			Return(diff, nil)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/git/repositories/%s/compare?base=main&head=feature/new-feature", suite.testRepositoryID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
	})
}

// TestTagOperations 测试标签操作
func (suite *GitHandlerTestSuite) TestTagOperations() {
	// 测试获取标签列表
	suite.Run("获取标签列表", func() {
		tags := []client.Tag{
			{
				ID:           uuid.New(),
				RepositoryID: suite.testRepositoryID,
				Name:         "v1.0.0",
				CommitSHA:    "abc123def456",
				Message:      &[]string{"Release version 1.0.0"}[0],
				Tagger:       "Release Manager",
				TaggerEmail:  "release@example.com",
				TaggedAt:     time.Now(),
				CreatedAt:    time.Now(),
			},
			{
				ID:           uuid.New(),
				RepositoryID: suite.testRepositoryID,
				Name:         "v0.9.0",
				CommitSHA:    "def456ghi789",
				Message:      &[]string{"Beta release"}[0],
				Tagger:       "Developer",
				TaggerEmail:  "dev@example.com",
				TaggedAt:     time.Now().Add(-24 * time.Hour),
				CreatedAt:    time.Now().Add(-24 * time.Hour),
			},
		}

		suite.mockGitClient.On("ListTags", mock.Anything, suite.testRepositoryID).
			Return(tags, nil)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/git/repositories/%s/tags", suite.testRepositoryID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
		assert.NotNil(suite.T(), response.Data)
	})

	// 测试创建标签
	suite.Run("创建标签", func() {
		newTag := &client.Tag{
			ID:           uuid.New(),
			RepositoryID: suite.testRepositoryID,
			Name:         "v1.1.0",
			CommitSHA:    "ghi789jkl012",
			Message:      &[]string{"Release version 1.1.0 with bug fixes"}[0],
			Tagger:       "Release Manager",
			TaggerEmail:  "release@example.com",
			TaggedAt:     time.Now(),
			CreatedAt:    time.Now(),
		}

		suite.mockGitClient.On("CreateTag", mock.Anything, suite.testRepositoryID, mock.MatchedBy(func(req *client.CreateTagRequest) bool {
			return req.Name == "v1.1.0" && 
				   req.CommitSHA == "ghi789jkl012" &&
				   req.Message != nil && *req.Message == "Release version 1.1.0 with bug fixes" &&
				   req.Tagger.Name == "Release Manager"
		})).Return(newTag, nil)

		body, _ := json.Marshal(map[string]interface{}{
			"name":       "v1.1.0",
			"commit_sha": "ghi789jkl012",
			"message":    "Release version 1.1.0 with bug fixes",
			"tagger": map[string]interface{}{
				"name":  "Release Manager",
				"email": "release@example.com",
			},
		})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/git/repositories/%s/tags", suite.testRepositoryID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusCreated, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
	})

	// 测试获取标签详情
	suite.Run("获取标签详情", func() {
		tag := &client.Tag{
			ID:           uuid.New(),
			RepositoryID: suite.testRepositoryID,
			Name:         "v1.0.0",
			CommitSHA:    "abc123def456",
			Message:      &[]string{"Release version 1.0.0"}[0],
			Tagger:       "Release Manager",
			TaggerEmail:  "release@example.com",
			TaggedAt:     time.Now(),
			CreatedAt:    time.Now(),
		}

		suite.mockGitClient.On("GetTag", mock.Anything, suite.testRepositoryID, "v1.0.0").
			Return(tag, nil)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/git/repositories/%s/tags/v1.0.0", suite.testRepositoryID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
	})

	// 测试删除标签
	suite.Run("删除标签", func() {
		suite.mockGitClient.On("DeleteTag", mock.Anything, suite.testRepositoryID, "v0.9.0").
			Return(nil)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/git/repositories/%s/tags/v0.9.0", suite.testRepositoryID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
	})
}

// TestFileOperations 测试文件操作
func (suite *GitHandlerTestSuite) TestFileOperations() {
	// 测试获取文件内容
	suite.Run("获取文件内容", func() {
		fileContent := []byte("package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}")

		suite.mockGitClient.On("GetFileContent", mock.Anything, suite.testRepositoryID, "main", "main.go").
			Return(fileContent, nil)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/git/repositories/%s/files?branch=main&path=main.go", suite.testRepositoryID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
		assert.NotNil(suite.T(), response.Data)
	})

	// 测试获取目录内容
	suite.Run("获取目录内容", func() {
		dirContent := []client.FileInfo{
			{
				Name: "main.go",
				Path: "main.go",
				Type: "file",
				Size: 156,
				Mode: "100644",
				SHA:  "abc123def456",
			},
			{
				Name: "README.md",
				Path: "README.md",
				Type: "file",
				Size: 1024,
				Mode: "100644",
				SHA:  "def456ghi789",
			},
			{
				Name: "src",
				Path: "src",
				Type: "directory",
				Size: 0,
				Mode: "040000",
				SHA:  "ghi789jkl012",
			},
		}

		suite.mockGitClient.On("GetDirectoryContent", mock.Anything, suite.testRepositoryID, "main", "/").
			Return(dirContent, nil)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/git/repositories/%s/tree?branch=main&path=/", suite.testRepositoryID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
		assert.NotNil(suite.T(), response.Data)
	})
}

// TestErrorHandling 测试错误处理
func (suite *GitHandlerTestSuite) TestErrorHandling() {
	testCases := []struct {
		name             string
		method           string
		url              string
		setupMocks       func()
		expectedStatus   int
		expectedContains string
	}{
		{
			name:   "仓库不存在",
			method: http.MethodGet,
			url:    fmt.Sprintf("/api/v1/git/repositories/%s", uuid.New().String()),
			setupMocks: func() {
				suite.mockGitClient.On("GetRepository", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return(nil, fmt.Errorf("repository not found"))
			},
			expectedStatus:   http.StatusNotFound,
			expectedContains: "repository not found",
		},
		{
			name:             "无效的仓库ID",
			method:           http.MethodGet,
			url:              "/api/v1/git/repositories/invalid-uuid",
			setupMocks:       func() {},
			expectedStatus:   http.StatusBadRequest,
			expectedContains: "无效的仓库ID",
		},
		{
			name:   "Git网关服务不可用",
			method: http.MethodGet,
			url:    fmt.Sprintf("/api/v1/git/repositories/%s", suite.testRepositoryID.String()),
			setupMocks: func() {
				suite.mockGitClient.On("GetRepository", mock.Anything, suite.testRepositoryID).
					Return(nil, fmt.Errorf("service unavailable"))
			},
			expectedStatus:   http.StatusInternalServerError,
			expectedContains: "service unavailable",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.setupMocks()

			req := httptest.NewRequest(tc.method, tc.url, nil)
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tc.expectedStatus, w.Code)
			if tc.expectedContains != "" {
				assert.Contains(suite.T(), w.Body.String(), tc.expectedContains)
			}
		})
	}
}

// 运行测试套件
func TestGitHandlerSuite(t *testing.T) {
	suite.Run(t, new(GitHandlerTestSuite))
}