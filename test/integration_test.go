package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/client"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/handlers"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/repository"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MockGitGatewayClient Git网关客户端模拟
type MockGitGatewayClient struct {
	mock.Mock
}

func (m *MockGitGatewayClient) CreateRepository(ctx context.Context, req *client.CreateRepositoryRequest) (*client.Repository, error) {
	args := m.Called(ctx, req)
	if repo := args.Get(0); repo != nil {
		return repo.(*client.Repository), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockGitGatewayClient) GetRepository(ctx context.Context, repositoryID uuid.UUID) (*client.Repository, error) {
	args := m.Called(ctx, repositoryID)
	if repo := args.Get(0); repo != nil {
		return repo.(*client.Repository), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockGitGatewayClient) UpdateRepository(ctx context.Context, repositoryID uuid.UUID, req *client.UpdateRepositoryRequest) (*client.Repository, error) {
	args := m.Called(ctx, repositoryID, req)
	if repo := args.Get(0); repo != nil {
		return repo.(*client.Repository), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockGitGatewayClient) DeleteRepository(ctx context.Context, repositoryID uuid.UUID) error {
	args := m.Called(ctx, repositoryID)
	return args.Error(0)
}

func (m *MockGitGatewayClient) ListRepositories(ctx context.Context, projectID *uuid.UUID, page, pageSize int) (*client.RepositoryListResponse, error) {
	args := m.Called(ctx, projectID, page, pageSize)
	if resp := args.Get(0); resp != nil {
		return resp.(*client.RepositoryListResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

// 实现其他接口方法（简化版）
func (m *MockGitGatewayClient) CreateBranch(ctx context.Context, repositoryID uuid.UUID, req *client.CreateBranchRequest) (*client.Branch, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockGitGatewayClient) GetBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) (*client.Branch, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockGitGatewayClient) ListBranches(ctx context.Context, repositoryID uuid.UUID) ([]client.Branch, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockGitGatewayClient) DeleteBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) error {
	return fmt.Errorf("not implemented")
}

func (m *MockGitGatewayClient) SetDefaultBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) error {
	return fmt.Errorf("not implemented")
}

func (m *MockGitGatewayClient) MergeBranch(ctx context.Context, repositoryID uuid.UUID, targetBranch, sourceBranch string) error {
	return fmt.Errorf("not implemented")
}

func (m *MockGitGatewayClient) CreateCommit(ctx context.Context, repositoryID uuid.UUID, req *client.CreateCommitRequest) (*client.Commit, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockGitGatewayClient) GetCommit(ctx context.Context, repositoryID uuid.UUID, sha string) (*client.Commit, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockGitGatewayClient) ListCommits(ctx context.Context, repositoryID uuid.UUID, branch string, page, pageSize int) (*client.CommitListResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockGitGatewayClient) GetCommitDiff(ctx context.Context, repositoryID uuid.UUID, sha string) (*client.GitDiff, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockGitGatewayClient) CompareBranches(ctx context.Context, repositoryID uuid.UUID, base, head string) (*client.GitDiff, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockGitGatewayClient) CreateTag(ctx context.Context, repositoryID uuid.UUID, req *client.CreateTagRequest) (*client.Tag, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockGitGatewayClient) GetTag(ctx context.Context, repositoryID uuid.UUID, tagName string) (*client.Tag, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockGitGatewayClient) ListTags(ctx context.Context, repositoryID uuid.UUID) ([]client.Tag, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockGitGatewayClient) DeleteTag(ctx context.Context, repositoryID uuid.UUID, tagName string) error {
	return fmt.Errorf("not implemented")
}

func (m *MockGitGatewayClient) GetFileContent(ctx context.Context, repositoryID uuid.UUID, branch, filePath string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockGitGatewayClient) GetDirectoryContent(ctx context.Context, repositoryID uuid.UUID, branch, dirPath string) ([]client.FileInfo, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockGitGatewayClient) GetRepositoryStats(ctx context.Context, repositoryID uuid.UUID) (*client.RepositoryStats, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockGitGatewayClient) SearchRepositories(ctx context.Context, query string, projectID *uuid.UUID, page, pageSize int) (*client.RepositoryListResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// 简化的项目模型用于测试
type TestProject struct {
	ID          string  `gorm:"type:varchar(36);primary_key"`
	TenantID    string  `gorm:"type:varchar(36);not null"`
	Key         string  `gorm:"size:20;not null"`
	Name        string  `gorm:"size:255;not null"`
	Description *string `gorm:"type:text"`
	ManagerID   *string `gorm:"type:varchar(36)"`
	Status      string  `gorm:"size:20;not null;default:'active'"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time `gorm:"index"`
}

type TestProjectMember struct {
	ProjectID string `gorm:"type:varchar(36);primary_key"`
	UserID    string `gorm:"type:varchar(36);primary_key"`
	RoleID    string `gorm:"type:varchar(36);not null"`
	AddedAt   time.Time
	AddedBy   *string `gorm:"type:varchar(36)"`
}

type TestRepository struct {
	ID            string  `gorm:"type:varchar(36);primary_key"`
	ProjectID     string  `gorm:"type:varchar(36)"`
	Name          string  `gorm:"size:255;not null"`
	Description   *string `gorm:"type:text"`
	Visibility    string  `gorm:"size:20;not null"`
	DefaultBranch string  `gorm:"size:255;not null"`
}

type TestUser struct {
	ID          string  `gorm:"type:varchar(36);primary_key"`
	Username    string  `gorm:"size:255;not null"`
	Email       string  `gorm:"size:255;not null"`
	DisplayName *string `gorm:"size:255"`
	AvatarURL   *string `gorm:"type:text"`
	Status      string  `gorm:"size:20;not null;default:'active'"`
}

type TestRole struct {
	ID          string `gorm:"type:varchar(36);primary_key"`
	TenantID    string `gorm:"type:varchar(36);not null"`
	Name        string `gorm:"size:255;not null"`
	Scope       string `gorm:"size:50;not null"`
	Permissions string `gorm:"type:text"` // 简化为字符串
}

// 为测试模型添加TableName方法
type TestProjectWithTable struct {
	TestProject
}

func (TestProjectWithTable) TableName() string {
	return "projects"
}

type TestProjectMemberWithTable struct {
	TestProjectMember
}

func (TestProjectMemberWithTable) TableName() string {
	return "project_members"
}

type TestRepositoryWithTable struct {
	TestRepository
}

func (TestRepositoryWithTable) TableName() string {
	return "repositories"
}

type TestUserWithTable struct {
	TestUser
}

func (TestUserWithTable) TableName() string {
	return "users"
}

type TestRoleWithTable struct {
	TestRole
}

func (TestRoleWithTable) TableName() string {
	return "roles"
}

// 设置测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	// 使用SQLite内存数据库进行测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true, // 禁用外键约束以简化测试
	})
	assert.NoError(t, err)

	// 自动迁移测试表
	err = db.AutoMigrate(
		&TestProjectWithTable{},
		&TestProjectMemberWithTable{},
		&TestRepositoryWithTable{},
		&TestUserWithTable{},
		&TestRoleWithTable{},
	)
	assert.NoError(t, err)

	return db
}

// 设置测试路由
func setupTestRouter(projectService service.ProjectService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	logger := zap.NewNop()
	projectHandler := handlers.NewProjectHandler(projectService, nil, logger)
	
	// 添加认证中间件模拟
	router.Use(func(c *gin.Context) {
		// 使用固定的测试用户ID和租户ID
		userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		tenantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
		c.Set("user_id", userID)
		c.Set("tenant_id", tenantID)
		c.Next()
	})
	
	v1 := router.Group("/api/v1")
	{
		projects := v1.Group("/projects")
		{
			projects.POST("", projectHandler.CreateProject)
			projects.GET("/:id", projectHandler.GetProject)
			projects.POST("/:id/repositories", projectHandler.CreateRepository)
			projects.GET("/:id/repositories", projectHandler.ListRepositories)
		}
		
		repositories := v1.Group("/repositories")
		{
			repositories.GET("/:repository_id", projectHandler.GetRepository)
			repositories.PUT("/:repository_id", projectHandler.UpdateRepository)
			repositories.DELETE("/:repository_id", projectHandler.DeleteRepository)
		}
	}
	
	return router
}

// TestProjectGitIntegration 测试项目与Git网关集成
func TestProjectGitIntegration(t *testing.T) {
	// 设置测试数据库
	db := setupTestDB(t)
	
	// 创建模拟Git网关客户端
	mockGitClient := new(MockGitGatewayClient)
	
	// 设置logger
	logger := zap.NewNop()
	
	// 创建项目仓库和服务
	projectRepo := repository.NewProjectRepository(db)
	projectService := service.NewProjectService(projectRepo, mockGitClient, logger)
	
	// 设置路由
	router := setupTestRouter(projectService)
	
	t.Run("创建项目成功", func(t *testing.T) {
		// 创建项目请求
		createProjectReq := map[string]interface{}{
			"key":         "test-project",
			"name":        "测试项目",
			"description": "这是一个测试项目",
		}
		
		reqBody, _ := json.Marshal(createProjectReq)
		req := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
			
			if w.Code != http.StatusCreated {
				t.Logf("创建仓库失败，响应状态: %d, 响应体: %s", w.Code, w.Body.String())
			}
			assert.Equal(t, http.StatusCreated, w.Code)
			
			var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(201), response["code"]) // JSON数字会被解析为float64
		
		// 保存项目ID用于后续测试
		projectData := response["data"].(map[string]interface{})
		projectID := projectData["id"].(string)
		
		t.Run("为项目创建Git仓库", func(t *testing.T) {
			// 模拟Git网关返回成功创建的仓库
			repoID := uuid.New()
			mockRepo := &client.Repository{
				ID:          repoID,
				ProjectID:   uuid.MustParse(projectID),
				Name:        "test-repo",
				Description: stringPtr("测试仓库"),
				Visibility:  client.RepositoryVisibilityPrivate,
				DefaultBranch: "main",
			}
			
			mockGitClient.On("CreateRepository", mock.Anything, mock.MatchedBy(func(req *client.CreateRepositoryRequest) bool {
				return req.Name == "test-repo" && req.ProjectID == projectID
			})).Return(mockRepo, nil)
			
			// 模拟GetRepository调用（分布式事务确认阶段会调用）
			mockGitClient.On("GetRepository", mock.Anything, repoID).Return(mockRepo, nil)
			
			// 创建仓库请求
			createRepoReq := map[string]interface{}{
				"name":          "test-repo",
				"description":   "测试仓库",
				"visibility":    "private",
				"default_branch": "main",
				"init_readme":   true,
			}
			
			reqBody, _ := json.Marshal(createRepoReq)
			req := httptest.NewRequest("POST", "/api/v1/projects/"+projectID+"/repositories", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if w.Code != http.StatusCreated {
				t.Logf("创建仓库失败，响应状态: %d, 响应体: %s", w.Code, w.Body.String())
			}
			assert.Equal(t, http.StatusCreated, w.Code)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, float64(201), response["code"]) // JSON数字会被解析为float64
			
			// 验证返回的仓库数据
			repoData := response["data"].(map[string]interface{})
			assert.Equal(t, "test-repo", repoData["name"])
			assert.Equal(t, "private", repoData["visibility"])
			assert.Equal(t, "main", repoData["default_branch"])
			
			mockGitClient.AssertExpectations(t)
			
			t.Run("获取项目仓库列表", func(t *testing.T) {
				// 模拟Git网关返回仓库列表
				mockRepoList := &client.RepositoryListResponse{
					Repositories: []client.Repository{*mockRepo},
					Total:        1,
					Page:         1,
					PageSize:     20,
				}
				
				projectUUID := uuid.MustParse(projectID)
				mockGitClient.On("ListRepositories", mock.Anything, &projectUUID, 1, 20).Return(mockRepoList, nil)
				
				req := httptest.NewRequest("GET", "/api/v1/projects/"+projectID+"/repositories", nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				
				assert.Equal(t, http.StatusOK, w.Code)
				
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, float64(200), response["code"]) // 获取操作应该返回200 // JSON数字会被解析为float64
				
				// 验证仓库列表
				repoListData := response["data"].(map[string]interface{})
				repositories := repoListData["repositories"].([]interface{})
				assert.Len(t, repositories, 1)
				
				repo := repositories[0].(map[string]interface{})
				assert.Equal(t, "test-repo", repo["name"])
				
				mockGitClient.AssertExpectations(t)
			})
			
			t.Run("获取仓库详情", func(t *testing.T) {
				// 模拟Git网关返回仓库详情
				mockGitClient.On("GetRepository", mock.Anything, repoID).Return(mockRepo, nil)
				
				req := httptest.NewRequest("GET", "/api/v1/repositories/"+repoID.String(), nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				
				assert.Equal(t, http.StatusOK, w.Code)
				
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, float64(200), response["code"]) // 获取操作应该返回200 // JSON数字会被解析为float64
				
				// 验证仓库详情
				repoData := response["data"].(map[string]interface{})
				assert.Equal(t, "test-repo", repoData["name"])
				assert.Equal(t, repoID.String(), repoData["id"])
				
				mockGitClient.AssertExpectations(t)
			})
		})
	})
}

// TestGitGatewayClientIntegration 测试Git网关客户端集成
func TestGitGatewayClientIntegration(t *testing.T) {
	// 如果需要测试真实的HTTP通信，可以启动一个测试服务器
	t.Skip("需要运行真实的Git网关服务进行集成测试")
	
	// 创建真实的Git网关客户端
	gitClient := client.NewGitGatewayClient(&client.GitGatewayClientConfig{
		BaseURL: "http://localhost:8083",
		Timeout: 30 * time.Second,
		Logger:  zap.NewNop(),
	})
	
	ctx := context.Background()
	
	// 测试创建仓库
	createReq := &client.CreateRepositoryRequest{
		ProjectID:     uuid.New().String(),
		Name:          "integration-test-repo",
		Description:   stringPtr("集成测试仓库"),
		Visibility:    client.RepositoryVisibilityPrivate,
		DefaultBranch: stringPtr("main"),
		InitReadme:    true,
	}
	
	repo, err := gitClient.CreateRepository(ctx, createReq)
	if err != nil {
		t.Logf("创建仓库失败（预期行为，因为服务未运行）: %v", err)
		return
	}
	
	assert.NotNil(t, repo)
	assert.Equal(t, "integration-test-repo", repo.Name)
	
	// 测试获取仓库
	fetchedRepo, err := gitClient.GetRepository(ctx, repo.ID)
	assert.NoError(t, err)
	assert.Equal(t, repo.ID, fetchedRepo.ID)
	assert.Equal(t, repo.Name, fetchedRepo.Name)
	
	// 测试删除仓库
	err = gitClient.DeleteRepository(ctx, repo.ID)
	assert.NoError(t, err)
}

// 辅助函数定义在 test_helpers.go 中