package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	"go.uber.org/zap/zaptest"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MockProjectRepository 项目仓库模拟
type MockProjectRepository struct {
	mock.Mock
}

func (m *MockProjectRepository) Create(project *repository.Project) error {
	args := m.Called(project)
	return args.Error(0)
}

func (m *MockProjectRepository) GetByID(id string, tenantID string) (*repository.Project, error) {
	args := m.Called(id, tenantID)
	if project := args.Get(0); project != nil {
		return project.(*repository.Project), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectRepository) GetByKey(key string, tenantID string) (*repository.Project, error) {
	args := m.Called(key, tenantID)
	if project := args.Get(0); project != nil {
		return project.(*repository.Project), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectRepository) Update(project *repository.Project) error {
	args := m.Called(project)
	return args.Error(0)
}

func (m *MockProjectRepository) Delete(id string, tenantID string) error {
	args := m.Called(id, tenantID)
	return args.Error(0)
}

func (m *MockProjectRepository) List(tenantID string, page, pageSize int) ([]*repository.Project, int64, error) {
	args := m.Called(tenantID, page, pageSize)
	if projects := args.Get(0); projects != nil {
		return projects.([]*repository.Project), args.Get(1).(int64), args.Error(2)
	}
	return nil, 0, args.Error(2)
}

func (m *MockProjectRepository) GetUserProjects(userID string, tenantID string) ([]*repository.Project, error) {
	args := m.Called(userID, tenantID)
	if projects := args.Get(0); projects != nil {
		return projects.([]*repository.Project), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectRepository) CheckAccess(projectID string, userID string, tenantID string) (bool, error) {
	args := m.Called(projectID, userID, tenantID)
	return args.Bool(0), args.Error(1)
}

func (m *MockProjectRepository) AddMember(projectID string, userID string, roleID string, tenantID string) error {
	args := m.Called(projectID, userID, roleID, tenantID)
	return args.Error(0)
}

func (m *MockProjectRepository) RemoveMember(projectID string, userID string, tenantID string) error {
	args := m.Called(projectID, userID, tenantID)
	return args.Error(0)
}

func (m *MockProjectRepository) GetMembers(projectID string, tenantID string) ([]*repository.ProjectMember, error) {
	args := m.Called(projectID, tenantID)
	if members := args.Get(0); members != nil {
		return members.([]*repository.ProjectMember), args.Error(1)
	}
	return nil, args.Error(1)
}

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

// 完整的ProjectService测试套件
// TestProjectService_CreateProject 测试项目创建功能
func TestProjectService_CreateProject(t *testing.T) {
	tests := []struct {
		name          string
		req           *service.CreateProjectRequest
		setupMocks    func(*MockProjectRepository, *MockGitGatewayClient)
		expectedError string
	}{
		{
			name: "成功创建项目",
			req: &service.CreateProjectRequest{
				Name:        "测试项目",
				Key:         "TEST",
				Description: "这是一个测试项目",
				TenantID:    "tenant-1",
				CreatedBy:   "user-1",
			},
			setupMocks: func(mockRepo *MockProjectRepository, mockGit *MockGitGatewayClient) {
				mockRepo.On("GetByKey", "TEST", "tenant-1").Return(nil, gorm.ErrRecordNotFound)
				mockRepo.On("Create", mock.AnythingOfType("*repository.Project")).Return(nil)
				mockGit.On("CreateRepository", mock.Anything, mock.AnythingOfType("*client.CreateRepositoryRequest")).Return(&client.Repository{
					ID:   uuid.New(),
					Name: "TEST",
					URL:  "http://git.example.com/TEST",
				}, nil)
			},
		},
		{
			name: "项目标识符已存在",
			req: &service.CreateProjectRequest{
				Name:        "重复项目",
				Key:         "EXISTING",
				Description: "重复的项目",
				TenantID:    "tenant-1",
				CreatedBy:   "user-1",
			},
			setupMocks: func(mockRepo *MockProjectRepository, mockGit *MockGitGatewayClient) {
				existingProject := &repository.Project{
					ID:  uuid.New().String(),
					Key: "EXISTING",
				}
				mockRepo.On("GetByKey", "EXISTING", "tenant-1").Return(existingProject, nil)
			},
			expectedError: "项目标识符已存在",
		},
		{
			name: "无效的项目标识符格式",
			req: &service.CreateProjectRequest{
				Name:        "无效标识符项目",
				Key:         "invalid-key",
				Description: "无效标识符的项目",
				TenantID:    "tenant-1",
				CreatedBy:   "user-1",
			},
			setupMocks:    func(mockRepo *MockProjectRepository, mockGit *MockGitGatewayClient) {},
			expectedError: "项目标识符格式不正确",
		},
		{
			name: "项目名称为空",
			req: &service.CreateProjectRequest{
				Name:        "",
				Key:         "EMPTY",
				Description: "空名称项目",
				TenantID:    "tenant-1",
				CreatedBy:   "user-1",
			},
			setupMocks:    func(mockRepo *MockProjectRepository, mockGit *MockGitGatewayClient) {},
			expectedError: "项目名称不能为空",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockProjectRepository)
			mockGit := new(MockGitGatewayClient)
			logger := zaptest.NewLogger(t)

			projectService := service.NewProjectService(mockRepo, mockGit, logger)

			// Setup mocks
			tt.setupMocks(mockRepo, mockGit)

			// Execute
			result, err := projectService.CreateProject(context.Background(), tt.req)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.req.Name, result.Name)
				assert.Equal(t, tt.req.Key, result.Key)
			}

			// Verify all expectations were met
			mockRepo.AssertExpectations(t)
			mockGit.AssertExpectations(t)
		})
	}
}

// TestProjectService_GetProject 测试获取项目功能
func TestProjectService_GetProject(t *testing.T) {
	tests := []struct {
		name          string
		projectID     string
		userID        string
		tenantID      string
		setupMocks    func(*MockProjectRepository)
		expectedError string
	}{
		{
			name:      "成功获取项目",
			projectID: "project-1",
			userID:    "user-1",
			tenantID:  "tenant-1",
			setupMocks: func(mockRepo *MockProjectRepository) {
				project := &repository.Project{
					ID:          "project-1",
					TenantID:    "tenant-1",
					Key:         "TEST",
					Name:        "测试项目",
					Description: "这是一个测试项目",
					Status:      "active",
				}
				mockRepo.On("GetByID", "project-1", "tenant-1").Return(project, nil)
				mockRepo.On("CheckAccess", "project-1", "user-1", "tenant-1").Return(true, nil)
			},
		},
		{
			name:      "项目不存在",
			projectID: "nonexistent",
			userID:    "user-1",
			tenantID:  "tenant-1",
			setupMocks: func(mockRepo *MockProjectRepository) {
				mockRepo.On("GetByID", "nonexistent", "tenant-1").Return(nil, gorm.ErrRecordNotFound)
			},
			expectedError: "项目不存在",
		},
		{
			name:      "无访问权限",
			projectID: "project-1",
			userID:    "user-2",
			tenantID:  "tenant-1",
			setupMocks: func(mockRepo *MockProjectRepository) {
				project := &repository.Project{
					ID:       "project-1",
					TenantID: "tenant-1",
					Key:      "TEST",
					Name:     "测试项目",
				}
				mockRepo.On("GetByID", "project-1", "tenant-1").Return(project, nil)
				mockRepo.On("CheckAccess", "project-1", "user-2", "tenant-1").Return(false, nil)
			},
			expectedError: "无权限访问此项目",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockProjectRepository)
			mockGit := new(MockGitGatewayClient)
			logger := zaptest.NewLogger(t)

			projectService := service.NewProjectService(mockRepo, mockGit, logger)

			// Setup mocks
			tt.setupMocks(mockRepo)

			// Execute
			result, err := projectService.GetProject(context.Background(), tt.projectID, tt.userID, tt.tenantID)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.projectID, result.ID)
			}

			// Verify all expectations were met
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestProjectService_UpdateProject 测试更新项目功能
func TestProjectService_UpdateProject(t *testing.T) {
	tests := []struct {
		name          string
		projectID     string
		req           *service.UpdateProjectRequest
		userID        string
		tenantID      string
		setupMocks    func(*MockProjectRepository)
		expectedError string
	}{
		{
			name:      "成功更新项目",
			projectID: "project-1",
			req: &service.UpdateProjectRequest{
				Name:        "更新的测试项目",
				Description: "更新后的描述",
			},
			userID:   "user-1",
			tenantID: "tenant-1",
			setupMocks: func(mockRepo *MockProjectRepository) {
				project := &repository.Project{
					ID:          "project-1",
					TenantID:    "tenant-1",
					Key:         "TEST",
					Name:        "测试项目",
					Description: "原描述",
					Status:      "active",
				}
				mockRepo.On("GetByID", "project-1", "tenant-1").Return(project, nil)
				mockRepo.On("CheckAccess", "project-1", "user-1", "tenant-1").Return(true, nil)
				mockRepo.On("Update", mock.AnythingOfType("*repository.Project")).Return(nil)
			},
		},
		{
			name:      "项目不存在",
			projectID: "nonexistent",
			req: &service.UpdateProjectRequest{
				Name:        "更新的名称",
				Description: "更新的描述",
			},
			userID:   "user-1",
			tenantID: "tenant-1",
			setupMocks: func(mockRepo *MockProjectRepository) {
				mockRepo.On("GetByID", "nonexistent", "tenant-1").Return(nil, gorm.ErrRecordNotFound)
			},
			expectedError: "项目不存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockProjectRepository)
			mockGit := new(MockGitGatewayClient)
			logger := zaptest.NewLogger(t)

			projectService := service.NewProjectService(mockRepo, mockGit, logger)

			// Setup mocks
			tt.setupMocks(mockRepo)

			// Execute
			result, err := projectService.UpdateProject(context.Background(), tt.projectID, tt.req, tt.userID, tt.tenantID)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.req.Name, result.Name)
				assert.Equal(t, tt.req.Description, result.Description)
			}

			// Verify all expectations were met
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestProjectService_DeleteProject 测试删除项目功能
func TestProjectService_DeleteProject(t *testing.T) {
	tests := []struct {
		name          string
		projectID     string
		userID        string
		tenantID      string
		setupMocks    func(*MockProjectRepository, *MockGitGatewayClient)
		expectedError string
	}{
		{
			name:      "成功删除项目",
			projectID: "project-1",
			userID:    "user-1",
			tenantID:  "tenant-1",
			setupMocks: func(mockRepo *MockProjectRepository, mockGit *MockGitGatewayClient) {
				project := &repository.Project{
					ID:       "project-1",
					TenantID: "tenant-1",
					Key:      "TEST",
					Name:     "测试项目",
					Status:   "active",
				}
				mockRepo.On("GetByID", "project-1", "tenant-1").Return(project, nil)
				mockRepo.On("CheckAccess", "project-1", "user-1", "tenant-1").Return(true, nil)
				mockRepo.On("Delete", "project-1", "tenant-1").Return(nil)
			},
		},
		{
			name:      "项目不存在",
			projectID: "nonexistent",
			userID:    "user-1",
			tenantID:  "tenant-1",
			setupMocks: func(mockRepo *MockProjectRepository, mockGit *MockGitGatewayClient) {
				mockRepo.On("GetByID", "nonexistent", "tenant-1").Return(nil, gorm.ErrRecordNotFound)
			},
			expectedError: "项目不存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockProjectRepository)
			mockGit := new(MockGitGatewayClient)
			logger := zaptest.NewLogger(t)

			projectService := service.NewProjectService(mockRepo, mockGit, logger)

			// Setup mocks
			tt.setupMocks(mockRepo, mockGit)

			// Execute
			err := projectService.DeleteProject(context.Background(), tt.projectID, tt.userID, tt.tenantID)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			// Verify all expectations were met
			mockRepo.AssertExpectations(t)
			mockGit.AssertExpectations(t)
		})
	}
}

// TestProjectService_AddMember 测试添加项目成员功能
func TestProjectService_AddMember(t *testing.T) {
	tests := []struct {
		name          string
		projectID     string
		req           *service.AddMemberRequest
		tenantID      string
		setupMocks    func(*MockProjectRepository)
		expectedError string
	}{
		{
			name:      "成功添加成员",
			projectID: "project-1",
			req: &service.AddMemberRequest{
				UserID: "user-2",
				RoleID: "role-dev",
			},
			tenantID: "tenant-1",
			setupMocks: func(mockRepo *MockProjectRepository) {
				project := &repository.Project{
					ID:       "project-1",
					TenantID: "tenant-1",
					Status:   "active",
				}
				mockRepo.On("GetByID", "project-1", "tenant-1").Return(project, nil)
				mockRepo.On("AddMember", "project-1", "user-2", "role-dev", "tenant-1").Return(nil)
			},
		},
		{
			name:      "项目不存在",
			projectID: "nonexistent",
			req: &service.AddMemberRequest{
				UserID: "user-2",
				RoleID: "role-dev",
			},
			tenantID: "tenant-1",
			setupMocks: func(mockRepo *MockProjectRepository) {
				mockRepo.On("GetByID", "nonexistent", "tenant-1").Return(nil, gorm.ErrRecordNotFound)
			},
			expectedError: "项目不存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockProjectRepository)
			mockGit := new(MockGitGatewayClient)
			logger := zaptest.NewLogger(t)

			projectService := service.NewProjectService(mockRepo, mockGit, logger)

			// Setup mocks
			tt.setupMocks(mockRepo)

			// Execute
			err := projectService.AddMember(context.Background(), tt.projectID, tt.req, tt.tenantID)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			// Verify all expectations were met
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestProjectService_ListProjects 测试列表项目功能
func TestProjectService_ListProjects(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		page       int
		pageSize   int
		setupMocks func(*MockProjectRepository)
	}{
		{
			name:     "成功获取项目列表",
			tenantID: "tenant-1",
			page:     1,
			pageSize: 10,
			setupMocks: func(mockRepo *MockProjectRepository) {
				projects := []*repository.Project{
					{
						ID:       "project-1",
						TenantID: "tenant-1",
						Key:      "TEST1",
						Name:     "测试项目1",
						Status:   "active",
					},
					{
						ID:       "project-2",
						TenantID: "tenant-1",
						Key:      "TEST2",
						Name:     "测试项目2",
						Status:   "active",
					},
				}
				mockRepo.On("List", "tenant-1", 1, 10).Return(projects, int64(2), nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockProjectRepository)
			mockGit := new(MockGitGatewayClient)
			logger := zaptest.NewLogger(t)

			projectService := service.NewProjectService(mockRepo, mockGit, logger)

			// Setup mocks
			tt.setupMocks(mockRepo)

			// Execute
			result, err := projectService.ListProjects(context.Background(), tt.tenantID, tt.page, tt.pageSize)

			// Assert
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Len(t, result.Projects, 2)
			assert.Equal(t, int64(2), result.Total)

			// Verify all expectations were met
			mockRepo.AssertExpectations(t)
		})
	}
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
				ID:            repoID,
				ProjectID:     uuid.MustParse(projectID),
				Name:          "test-repo",
				Description:   stringPtr("测试仓库"),
				Visibility:    client.RepositoryVisibilityPrivate,
				DefaultBranch: "main",
			}

			mockGitClient.On("CreateRepository", mock.Anything, mock.MatchedBy(func(req *client.CreateRepositoryRequest) bool {
				return req.Name == "test-repo" && req.ProjectID == projectID
			})).Return(mockRepo, nil)

			// 模拟GetRepository调用（分布式事务确认阶段会调用）
			mockGitClient.On("GetRepository", mock.Anything, repoID).Return(mockRepo, nil)

			// 创建仓库请求
			createRepoReq := map[string]interface{}{
				"name":           "test-repo",
				"description":    "测试仓库",
				"visibility":     "private",
				"default_branch": "main",
				"init_readme":    true,
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
