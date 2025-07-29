package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/git-gateway/handlers"
	"github.com/cloud-platform/collaborative-dev/internal/git-gateway/models"
	"github.com/cloud-platform/collaborative-dev/shared/database"
	"github.com/cloud-platform/collaborative-dev/shared/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// GitGatewayIntegrationTestSuite Git Gateway服务集成测试套件
type GitGatewayIntegrationTestSuite struct {
	suite.Suite
	db              *gorm.DB
	logger          *zap.Logger
	router          *gin.Engine
	server          *httptest.Server
	mockGitService  *MockGitService
	gitHandler      *handlers.GitHandler
	authToken       string
	testProjectID   uuid.UUID
	testRepositories []uuid.UUID
	testBranches    []string
	testCommits     []string
	testTags        []string
}

// MockGitService 实现GitService接口用于测试
type MockGitService struct {
	repositories map[uuid.UUID]*models.Repository
	branches     map[uuid.UUID][]models.Branch
	commits      map[uuid.UUID][]models.Commit
	tags         map[uuid.UUID][]models.Tag
}

// NewMockGitService 创建Mock Git服务
func NewMockGitService() *MockGitService {
	return &MockGitService{
		repositories: make(map[uuid.UUID]*models.Repository),
		branches:     make(map[uuid.UUID][]models.Branch),
		commits:      make(map[uuid.UUID][]models.Commit),
		tags:         make(map[uuid.UUID][]models.Tag),
	}
}

// 实现GitService接口的所有方法
func (m *MockGitService) CreateRepository(ctx context.Context, req *models.CreateRepositoryRequest, userID uuid.UUID) (*models.Repository, error) {
	projectID, _ := uuid.Parse(req.ProjectID)
	repo := &models.Repository{
		ID:            uuid.New(),
		ProjectID:     projectID,
		Name:          req.Name,
		Description:   req.Description,
		Visibility:    req.Visibility,
		Status:        models.RepositoryStatusActive,
		DefaultBranch: "main",
		CloneURL:      fmt.Sprintf("https://git.example.com/%s/%s.git", projectID.String(), req.Name),
		SSHURL:        fmt.Sprintf("git@git.example.com:%s/%s.git", projectID.String(), req.Name),
		Size:          0,
		CommitCount:   0,
		BranchCount:   1,
		TagCount:      0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	
	if req.DefaultBranch != nil {
		repo.DefaultBranch = *req.DefaultBranch
	}
	
	m.repositories[repo.ID] = repo
	return repo, nil
}

func (m *MockGitService) GetRepository(ctx context.Context, id uuid.UUID) (*models.Repository, error) {
	if repo, exists := m.repositories[id]; exists {
		return repo, nil
	}
	return nil, fmt.Errorf("repository not found")
}

func (m *MockGitService) ListRepositories(ctx context.Context, projectID *uuid.UUID, page, pageSize int) (*models.RepositoryListResponse, error) {
	var repos []models.Repository
	for _, repo := range m.repositories {
		if projectID == nil || repo.ProjectID == *projectID {
			repos = append(repos, *repo)
		}
	}
	
	return &models.RepositoryListResponse{
		Repositories: repos,
		Total:        int64(len(repos)),
		Page:         page,
		PageSize:     pageSize,
	}, nil
}

func (m *MockGitService) UpdateRepository(ctx context.Context, id uuid.UUID, req *models.UpdateRepositoryRequest) (*models.Repository, error) {
	repo, exists := m.repositories[id]
	if !exists {
		return nil, fmt.Errorf("repository not found")
	}
	
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
		repo.DefaultBranch = *req.DefaultBranch
	}
	
	repo.UpdatedAt = time.Now()
	return repo, nil
}

func (m *MockGitService) DeleteRepository(ctx context.Context, id uuid.UUID) error {
	if _, exists := m.repositories[id]; !exists {
		return fmt.Errorf("repository not found")
	}
	delete(m.repositories, id)
	return nil
}

func (m *MockGitService) CreateBranch(ctx context.Context, repositoryID uuid.UUID, req *models.CreateBranchRequest) (*models.Branch, error) {
	branch := &models.Branch{
		ID:           uuid.New(),
		RepositoryID: repositoryID,
		Name:         req.Name,
		CommitSHA:    req.FromSHA,
		IsDefault:    false,
		IsProtected:  req.Protected != nil && *req.Protected,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	if _, exists := m.branches[repositoryID]; !exists {
		m.branches[repositoryID] = []models.Branch{}
	}
	m.branches[repositoryID] = append(m.branches[repositoryID], *branch)
	return branch, nil
}

func (m *MockGitService) GetBranch(ctx context.Context, repositoryID uuid.UUID, name string) (*models.Branch, error) {
	if branches, exists := m.branches[repositoryID]; exists {
		for _, branch := range branches {
			if branch.Name == name {
				return &branch, nil
			}
		}
	}
	return nil, fmt.Errorf("branch not found")
}

func (m *MockGitService) ListBranches(ctx context.Context, repositoryID uuid.UUID) ([]models.Branch, error) {
	if branches, exists := m.branches[repositoryID]; exists {
		return branches, nil
	}
	return []models.Branch{}, nil
}

func (m *MockGitService) DeleteBranch(ctx context.Context, repositoryID uuid.UUID, name string) error {
	if branches, exists := m.branches[repositoryID]; exists {
		for i, branch := range branches {
			if branch.Name == name {
				m.branches[repositoryID] = append(branches[:i], branches[i+1:]...)
				return nil
			}
		}
	}
	return fmt.Errorf("branch not found")
}

func (m *MockGitService) SetDefaultBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) error {
	return nil
}

func (m *MockGitService) MergeBranch(ctx context.Context, repositoryID uuid.UUID, targetBranch, sourceBranch string) error {
	return nil
}

func (m *MockGitService) CreateCommit(ctx context.Context, repositoryID uuid.UUID, req *models.CreateCommitRequest) (*models.Commit, error) {
	commit := &models.Commit{
		ID:             uuid.New(),
		RepositoryID:   repositoryID,
		SHA:            fmt.Sprintf("commit_sha_%d", time.Now().Unix()),
		Message:        req.Message,
		Author:         req.Author.Name,
		AuthorEmail:    req.Author.Email,
		Committer:      req.Author.Name,
		CommitterEmail: req.Author.Email,
		CommittedAt:    time.Now(),
		AddedLines:     10,
		DeletedLines:   5,
		ChangedFiles:   int32(len(req.Files)),
		CreatedAt:      time.Now(),
	}
	
	if _, exists := m.commits[repositoryID]; !exists {
		m.commits[repositoryID] = []models.Commit{}
	}
	m.commits[repositoryID] = append(m.commits[repositoryID], *commit)
	return commit, nil
}

func (m *MockGitService) GetCommit(ctx context.Context, repositoryID uuid.UUID, sha string) (*models.Commit, error) {
	if commits, exists := m.commits[repositoryID]; exists {
		for _, commit := range commits {
			if commit.SHA == sha {
				return &commit, nil
			}
		}
	}
	return nil, fmt.Errorf("commit not found")
}

func (m *MockGitService) ListCommits(ctx context.Context, repositoryID uuid.UUID, branch string, page, pageSize int) (*models.CommitListResponse, error) {
	commits := []models.Commit{}
	if existingCommits, exists := m.commits[repositoryID]; exists {
		commits = existingCommits
	}
	
	return &models.CommitListResponse{
		Commits:  commits,
		Total:    int64(len(commits)),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (m *MockGitService) GetCommitDiff(ctx context.Context, repositoryID uuid.UUID, sha string) (*models.GitDiff, error) {
	return &models.GitDiff{
		FromSHA: sha + "^",
		ToSHA:   sha,
		Files:   []models.DiffFile{},
	}, nil
}

func (m *MockGitService) CompareBranches(ctx context.Context, repositoryID uuid.UUID, base, head string) (*models.GitDiff, error) {
	return &models.GitDiff{
		FromSHA: base,
		ToSHA:   head,
		Files:   []models.DiffFile{},
	}, nil
}

func (m *MockGitService) CreateTag(ctx context.Context, repositoryID uuid.UUID, req *models.CreateTagRequest) (*models.Tag, error) {
	tag := &models.Tag{
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
	
	if _, exists := m.tags[repositoryID]; !exists {
		m.tags[repositoryID] = []models.Tag{}
	}
	m.tags[repositoryID] = append(m.tags[repositoryID], *tag)
	return tag, nil
}

func (m *MockGitService) GetTag(ctx context.Context, repositoryID uuid.UUID, name string) (*models.Tag, error) {
	if tags, exists := m.tags[repositoryID]; exists {
		for _, tag := range tags {
			if tag.Name == name {
				return &tag, nil
			}
		}
	}
	return nil, fmt.Errorf("tag not found")
}

func (m *MockGitService) ListTags(ctx context.Context, repositoryID uuid.UUID) ([]models.Tag, error) {
	if tags, exists := m.tags[repositoryID]; exists {
		return tags, nil
	}
	return []models.Tag{}, nil
}

func (m *MockGitService) DeleteTag(ctx context.Context, repositoryID uuid.UUID, name string) error {
	if tags, exists := m.tags[repositoryID]; exists {
		for i, tag := range tags {
			if tag.Name == name {
				m.tags[repositoryID] = append(tags[:i], tags[i+1:]...)
				return nil
			}
		}
	}
	return fmt.Errorf("tag not found")
}

func (m *MockGitService) GetFileContent(ctx context.Context, repositoryID uuid.UUID, branch, filePath string) ([]byte, error) {
	return []byte("mock file content"), nil
}

func (m *MockGitService) GetDirectoryContent(ctx context.Context, repositoryID uuid.UUID, branch, dirPath string) ([]models.FileInfo, error) {
	return []models.FileInfo{
		{
			Name: "README.md",
			Type: "file",
			Size: 1024,
		},
		{
			Name: "src",
			Type: "directory",
			Size: 0,
		},
	}, nil
}

func (m *MockGitService) GetRepositoryStats(ctx context.Context, repositoryID uuid.UUID) (*models.RepositoryStats, error) {
	return &models.RepositoryStats{
		Size:        1024,
		CommitCount: 10,
		BranchCount: 3,
		TagCount:    2,
	}, nil
}

func (m *MockGitService) SearchRepositories(ctx context.Context, query string, projectID *uuid.UUID, page, pageSize int) (*models.RepositoryListResponse, error) {
	return m.ListRepositories(ctx, projectID, page, pageSize)
}

// SetupSuite 初始化测试套件
func (suite *GitGatewayIntegrationTestSuite) SetupSuite() {
	// 设置Gin为测试模式
	gin.SetMode(gin.TestMode)

	// 初始化日志器
	suite.logger, _ = zap.NewDevelopment()

	// 初始化测试数据库
	suite.setupTestDatabase()

	// 初始化Mock服务
	suite.mockGitService = NewMockGitService()

	// 初始化Git处理器
	suite.gitHandler = handlers.NewGitHandler(suite.mockGitService, suite.logger)

	// 设置路由
	suite.setupRouter()

	// 启动测试服务器
	suite.server = httptest.NewServer(suite.router)

	// 设置测试数据
	suite.authToken = "test-auth-token"
	suite.testProjectID = uuid.New()
	suite.testRepositories = []uuid.UUID{}
	suite.testBranches = []string{}
	suite.testCommits = []string{}
	suite.testTags = []string{}

	suite.logger.Info("Git Gateway集成测试套件初始化完成")
}

// setupTestDatabase 设置测试数据库
func (suite *GitGatewayIntegrationTestSuite) setupTestDatabase() {
	config := database.Config{
		Host:     "localhost",
		Port:     5432,
		Name:     "test_git_gateway",
		User:     "postgres",
		Password: "password",
		SSLMode:  "disable",
	}

	postgresDB, err := database.NewPostgresDB(config)
	if err != nil {
		suite.T().Skip("无法连接到测试数据库，跳过集成测试")
		return
	}

	suite.db = postgresDB.DB
}

// setupRouter 设置路由
func (suite *GitGatewayIntegrationTestSuite) setupRouter() {
	suite.router = gin.New()
	
	// 添加认证中间件模拟
	suite.router.Use(func(c *gin.Context) {
		c.Set("user_id", uuid.New().String())
		c.Set("tenant_id", uuid.New().String())
		c.Next()
	})

	// Git Gateway API路由
	api := suite.router.Group("/api/v1")
	{
		// 仓库管理
		api.POST("/repositories", suite.gitHandler.CreateRepository)
		api.GET("/repositories/:id", suite.gitHandler.GetRepository)
		api.GET("/repositories", suite.gitHandler.ListRepositories)
		api.PUT("/repositories/:id", suite.gitHandler.UpdateRepository)
		api.DELETE("/repositories/:id", suite.gitHandler.DeleteRepository)
		api.GET("/repositories/:id/stats", suite.gitHandler.GetRepositoryStats)
		api.GET("/repositories/search", suite.gitHandler.SearchRepositories)

		// 分支管理
		api.POST("/repositories/:id/branches", suite.gitHandler.CreateBranch)
		api.GET("/repositories/:id/branches/:name", suite.gitHandler.GetBranch)
		api.GET("/repositories/:id/branches", suite.gitHandler.ListBranches)
		api.DELETE("/repositories/:id/branches/:name", suite.gitHandler.DeleteBranch)
		api.PUT("/repositories/:id/branches/default", suite.gitHandler.SetDefaultBranch)
		api.POST("/repositories/:id/branches/merge", suite.gitHandler.MergeBranch)

		// 提交管理
		api.POST("/repositories/:id/commits", suite.gitHandler.CreateCommit)
		api.GET("/repositories/:id/commits/:sha", suite.gitHandler.GetCommit)
		api.GET("/repositories/:id/commits", suite.gitHandler.ListCommits)
		api.GET("/repositories/:id/commits/:sha/diff", suite.gitHandler.GetCommitDiff)
		api.GET("/repositories/:id/compare/:base...:head", suite.gitHandler.CompareBranches)

		// 标签管理
		api.POST("/repositories/:id/tags", suite.gitHandler.CreateTag)
		api.GET("/repositories/:id/tags/:name", suite.gitHandler.GetTag)
		api.GET("/repositories/:id/tags", suite.gitHandler.ListTags)
		api.DELETE("/repositories/:id/tags/:name", suite.gitHandler.DeleteTag)

		// 文件操作
		api.GET("/repositories/:id/files/:branch/*filepath", suite.gitHandler.GetFileContent)
		api.GET("/repositories/:id/tree/:branch/*dirpath", suite.gitHandler.GetDirectoryContent)
	}
}

// TearDownSuite 清理测试套件
func (suite *GitGatewayIntegrationTestSuite) TearDownSuite() {
	if suite.server != nil {
		suite.server.Close()
	}
	suite.logger.Info("Git Gateway集成测试套件清理完成")
}

// makeRequest 发送HTTP请求
func (suite *GitGatewayIntegrationTestSuite) makeRequest(method, path string, body interface{}, token string) (*http.Response, []byte) {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}

	req, err := http.NewRequest(method, suite.server.URL+path, bytes.NewBuffer(reqBody))
	suite.NoError(err)

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.NoError(err)

	respBody, err := io.ReadAll(resp.Body)
	suite.NoError(err)
	resp.Body.Close()

	return resp, respBody
}

// 测试仓库管理
func (suite *GitGatewayIntegrationTestSuite) TestRepositoryManagement() {
	suite.Run("创建仓库", func() {
		createReq := models.CreateRepositoryRequest{
			ProjectID:     suite.testProjectID.String(),
			Name:          "test-repo",
			Description:   gitStringPtr("测试仓库"),
			Visibility:    models.RepositoryVisibilityPrivate,
			DefaultBranch: gitStringPtr("main"),
			InitReadme:    true,
		}

		resp, body := suite.makeRequest("POST", "/api/v1/repositories", createReq, suite.authToken)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

		var createResponse response.Response
		err := json.Unmarshal(body, &createResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), http.StatusCreated, createResponse.Code)

		// 解析仓库数据
		repoData, _ := json.Marshal(createResponse.Data)
		var repository models.Repository
		err = json.Unmarshal(repoData, &repository)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), createReq.Name, repository.Name)

		// 保存测试数据
		suite.testRepositories = append(suite.testRepositories, repository.ID)
	})

	suite.Run("获取仓库", func() {
		if len(suite.testRepositories) == 0 {
			suite.T().Skip("没有可用的测试仓库")
			return
		}

		repositoryID := suite.testRepositories[0]
		resp, body := suite.makeRequest("GET", fmt.Sprintf("/api/v1/repositories/%s", repositoryID), nil, suite.authToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var getResponse response.Response
		err := json.Unmarshal(body, &getResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), http.StatusOK, getResponse.Code)
	})

	suite.Run("列出仓库", func() {
		resp, body := suite.makeRequest("GET", "/api/v1/repositories", nil, suite.authToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var listResponse response.Response
		err := json.Unmarshal(body, &listResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), http.StatusOK, listResponse.Code)
	})
}

// 测试分支管理
func (suite *GitGatewayIntegrationTestSuite) TestBranchManagement() {
	suite.Run("创建分支", func() {
		if len(suite.testRepositories) == 0 {
			suite.T().Skip("没有可用的测试仓库")
			return
		}

		repositoryID := suite.testRepositories[0]
		createReq := models.CreateBranchRequest{
			Name:      "feature/test",
			FromSHA:   "initial_commit",
			Protected: gitBoolPtr(false),
		}

		resp, body := suite.makeRequest("POST", fmt.Sprintf("/api/v1/repositories/%s/branches", repositoryID), createReq, suite.authToken)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

		var createResponse response.Response
		err := json.Unmarshal(body, &createResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), http.StatusCreated, createResponse.Code)

		// 保存测试数据
		suite.testBranches = append(suite.testBranches, createReq.Name)
	})

	suite.Run("获取分支", func() {
		if len(suite.testRepositories) == 0 || len(suite.testBranches) == 0 {
			suite.T().Skip("没有可用的测试数据")
			return
		}

		repositoryID := suite.testRepositories[0]
		branchName := suite.testBranches[0]

		resp, body := suite.makeRequest("GET", fmt.Sprintf("/api/v1/repositories/%s/branches/%s", repositoryID, branchName), nil, suite.authToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var getResponse response.Response
		err := json.Unmarshal(body, &getResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), http.StatusOK, getResponse.Code)
	})
}

// TestGitGatewayIntegration 运行所有Git Gateway集成测试
func TestGitGatewayIntegration(t *testing.T) {
	suite.Run(t, new(GitGatewayIntegrationTestSuite))
}

// 辅助函数
func gitStringPtr(s string) *string {
	return &s
}

func gitBoolPtr(b bool) *bool {
	return &b
}