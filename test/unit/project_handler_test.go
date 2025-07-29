package unit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

	"github.com/cloud-platform/collaborative-dev/internal/project-service/handlers"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/models"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/service"
	"github.com/cloud-platform/collaborative-dev/shared/response"
)

// MockProjectService 项目服务模拟
type MockProjectService struct {
	mock.Mock
}

func (m *MockProjectService) CreateProject(ctx context.Context, project *models.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockProjectService) GetProject(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.Project, error) {
	args := m.Called(ctx, id, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Project), args.Error(1)
}

func (m *MockProjectService) GetProjectByKey(ctx context.Context, key string, tenantID uuid.UUID) (*models.Project, error) {
	args := m.Called(ctx, key, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Project), args.Error(1)
}

func (m *MockProjectService) UpdateProject(ctx context.Context, project *models.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockProjectService) DeleteProject(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	args := m.Called(ctx, id, tenantID)
	return args.Error(0)
}

func (m *MockProjectService) ListProjects(ctx context.Context, tenantID uuid.UUID, req *service.ListProjectsRequest) ([]models.Project, int64, error) {
	args := m.Called(ctx, tenantID, req)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]models.Project), args.Get(1).(int64), args.Error(2)
}

func (m *MockProjectService) AddProjectMember(ctx context.Context, req *service.AddMemberRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockProjectService) RemoveProjectMember(ctx context.Context, req *service.RemoveMemberRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockProjectService) GetProjectMembers(ctx context.Context, projectID uuid.UUID, tenantID uuid.UUID) ([]models.ProjectMember, error) {
	args := m.Called(ctx, projectID, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.ProjectMember), args.Error(1)
}

func (m *MockProjectService) UpdateMemberRole(ctx context.Context, req *service.UpdateMemberRoleRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockProjectService) CreateRepository(ctx context.Context, req *service.CreateRepositoryRequest) (*models.Repository, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Repository), args.Error(1)
}

func (m *MockProjectService) GetRepositories(ctx context.Context, projectID uuid.UUID, tenantID uuid.UUID) ([]models.Repository, error) {
	args := m.Called(ctx, projectID, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Repository), args.Error(1)
}

// ProjectHandlerTestSuite 项目处理器测试套件
type ProjectHandlerTestSuite struct {
	suite.Suite
	handler        *handlers.ProjectHandler
	mockService    *MockProjectService
	router         *gin.Engine
	logger         *zap.Logger
	testProjectID  uuid.UUID
	testTenantID   uuid.UUID
	testUserID     uuid.UUID
}

func (suite *ProjectHandlerTestSuite) SetupSuite() {
	// 设置测试环境
	gin.SetMode(gin.TestMode)
	suite.logger = zaptest.NewLogger(suite.T())
	
	// 生成测试用的UUID
	suite.testProjectID = uuid.New()
	suite.testTenantID = uuid.New()
	suite.testUserID = uuid.New()
}

func (suite *ProjectHandlerTestSuite) SetupTest() {
	// 为每个测试创建新的mock和handler
	suite.mockService = new(MockProjectService)
	suite.handler = handlers.NewProjectHandler(suite.mockService, suite.logger)
	
	// 设置路由
	suite.router = gin.New()
	suite.setupRoutes()
}

func (suite *ProjectHandlerTestSuite) TearDownTest() {
	// 清理mock
	suite.mockService.AssertExpectations(suite.T())
}

func (suite *ProjectHandlerTestSuite) setupRoutes() {
	api := suite.router.Group("/api/v1")
	{
		projects := api.Group("/projects")
		{
			projects.POST("", suite.addAuthContext, suite.handler.CreateProject)
			projects.GET("", suite.addAuthContext, suite.handler.ListProjects)
			projects.GET("/:id", suite.addAuthContext, suite.handler.GetProject)
			projects.PUT("/:id", suite.addAuthContext, suite.handler.UpdateProject)
			projects.DELETE("/:id", suite.addAuthContext, suite.handler.DeleteProject)
			
			projects.GET("/:id/members", suite.addAuthContext, suite.handler.GetProjectMembers)
			projects.POST("/:id/members", suite.addAuthContext, suite.handler.AddProjectMember)
			projects.DELETE("/:id/members/:user_id", suite.addAuthContext, suite.handler.RemoveProjectMember)
			projects.PUT("/:id/members/:user_id/role", suite.addAuthContext, suite.handler.UpdateMemberRole)
			
			projects.GET("/:id/repositories", suite.addAuthContext, suite.handler.GetRepositories)
			projects.POST("/:id/repositories", suite.addAuthContext, suite.handler.CreateRepository)
		}
	}
}

// addAuthContext 添加认证上下文中间件
func (suite *ProjectHandlerTestSuite) addAuthContext(c *gin.Context) {
	c.Set("tenant_id", suite.testTenantID.String())
	c.Set("user_id", suite.testUserID.String())
	c.Next()
}

// TestCreateProject 测试创建项目
func (suite *ProjectHandlerTestSuite) TestCreateProject() {
	testCases := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func()
		expectedStatus int
		expectedError  string
	}{
		{
			name: "成功创建项目",
			requestBody: map[string]interface{}{
				"name":        "测试项目",
				"key":         "TEST_PROJECT",
				"description": "这是一个测试项目",
			},
			setupMocks: func() {
				suite.mockService.On("CreateProject", mock.Anything, mock.MatchedBy(func(project *models.Project) bool {
					return project.Name == "测试项目" && 
						   project.Key == "TEST_PROJECT" && 
						   project.TenantID == suite.testTenantID &&
						   project.ManagerID != nil && *project.ManagerID == suite.testUserID
				})).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "项目名称为空",
			requestBody: map[string]interface{}{
				"key":         "TEST_PROJECT",
				"description": "这是一个测试项目",
			},
			setup{ocs: func() {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "项目名称不能为空",
		},
		{
			name: "项目键为空",
			requestBody: map[string]interface{}{
				"name":        "测试项目",
				"description": "这是一个测试项目",
			},
			setupMocks:     func() {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "项目键不能为空",
		},
		{
			name: "服务层创建失败",
			requestBody: map[string]interface{}{
				"name": "测试项目",
				"key":  "TEST_PROJECT",
			},
			setupMocks: func() {
				suite.mockService.On("CreateProject", mock.Anything, mock.AnythingOfType("*models.Project")).
					Return(fmt.Errorf("数据库连接失败"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "创建项目失败",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// 设置mocks
			tc.setupMocks()

			// 准备请求
			body, _ := json.Marshal(tc.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			suite.router.ServeHTTP(w, req)

			// 验证结果
			assert.Equal(suite.T(), tc.expectedStatus, w.Code)

			var response response.Response
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(suite.T(), err)

			if tc.expectedStatus == http.StatusCreated {
				assert.True(suite.T(), response.Success)
				assert.NotNil(suite.T(), response.Data)
			} else {
				assert.False(suite.T(), response.Success)
				if tc.expectedError != "" {
					assert.Contains(suite.T(), response.Message, tc.expectedError)
				}
			}
		})
	}
}

// TestGetProject 测试获取项目
func (suite *ProjectHandlerTestSuite) TestGetProject() {
	testProject := &models.Project{
		ID:          suite.testProjectID,
		TenantID:    suite.testTenantID,
		Name:        "测试项目",
		Key:         "TEST_PROJECT",
		Description: "这是一个测试项目",
		Status:      "active",
		ManagerID:   &suite.testUserID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	testCases := []struct {
		name           string
		projectID      string
		setupMocks     func()
		expectedStatus int
		expectedError  string
	}{
		{
			name:      "成功获取项目",
			projectID: suite.testProjectID.String(),
			setupMocks: func() {
				suite.mockService.On("GetProject", mock.Anything, suite.testProjectID, suite.testTenantID).
					Return(testProject, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "项目不存在",
			projectID: uuid.New().String(),
			setupMocks: func() {
				suite.mockService.On("GetProject", mock.Anything, mock.AnythingOfType("uuid.UUID"), suite.testTenantID).
					Return(nil, fmt.Errorf("项目不存在"))
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "项目不存在",
		},
		{
			name:           "无效的项目ID",
			projectID:      "invalid-uuid",
			setupMocks:     func() {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "无效的项目ID",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// 设置mocks
			tc.setupMocks()

			// 准备请求
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/projects/%s", tc.projectID), nil)
			w := httptest.NewRecorder()

			// 执行请求
			suite.router.ServeHTTP(w, req)

			// 验证结果
			assert.Equal(suite.T(), tc.expectedStatus, w.Code)

			var response response.Response
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(suite.T(), err)

			if tc.expectedStatus == http.StatusOK {
				assert.True(suite.T(), response.Success)
				assert.NotNil(suite.T(), response.Data)
			} else {
				assert.False(suite.T(), response.Success)
				if tc.expectedError != "" {
					assert.Contains(suite.T(), response.Message, tc.expectedError)
				}
			}
		})
	}
}

// TestListProjects 测试获取项目列表
func (suite *ProjectHandlerTestSuite) TestListProjects() {
	testProjects := []models.Project{
		{
			ID:          uuid.New(),
			TenantID:    suite.testTenantID,
			Name:        "项目1",
			Key:         "PROJECT_1",
			Status:      "active",
			ManagerID:   &suite.testUserID,
			CreatedAt:   time.Now(),
		},
		{
			ID:          uuid.New(),
			TenantID:    suite.testTenantID,
			Name:        "项目2",
			Key:         "PROJECT_2",
			Status:      "active",
			ManagerID:   &suite.testUserID,
			CreatedAt:   time.Now(),
		},
	}

	testCases := []struct {
		name           string
		queryParams    string
		setupMocks     func()
		expectedStatus int
	}{
		{
			name:        "成功获取项目列表",
			queryParams: "page=1&page_size=10",
			setupMocks: func() {
				suite.mockService.On("ListProjects", mock.Anything, suite.testTenantID, mock.MatchedBy(func(req *service.ListProjectsRequest) bool {
					return req.Page == 1 && req.PageSize == 10
				})).Return(testProjects, int64(2), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "带搜索条件的项目列表",
			queryParams: "page=1&page_size=10&search=项目1",
			setupMocks: func() {
				suite.mockService.On("ListProjects", mock.Anything, suite.testTenantID, mock.MatchedBy(func(req *service.ListProjectsRequest) bool {
					return req.Page == 1 && req.PageSize == 10 && req.Search == "项目1"
				})).Return(testProjects[:1], int64(1), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "默认分页参数",
			queryParams: "",
			setupMocks: func() {
				suite.mockService.On("ListProjects", mock.Anything, suite.testTenantID, mock.MatchedBy(func(req *service.ListProjectsRequest) bool {
					return req.Page == 1 && req.PageSize == 20
				})).Return(testProjects, int64(2), nil)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// 设置mocks
			tc.setupMocks()

			// 准备请求
			url := "/api/v1/projects"
			if tc.queryParams != "" {
				url += "?" + tc.queryParams
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			// 执行请求
			suite.router.ServeHTTP(w, req)

			// 验证结果
			assert.Equal(suite.T(), tc.expectedStatus, w.Code)

			var response response.Response
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(suite.T(), err)
			assert.True(suite.T(), response.Success)
			assert.NotNil(suite.T(), response.Data)
		})
	}
}

// TestUpdateProject 测试更新项目
func (suite *ProjectHandlerTestSuite) TestUpdateProject() {
	testCases := []struct {
		name           string
		projectID      string 
		requestBody    interface{}
		setupMocks     func()
		expectedStatus int
		expectedError  string
	}{
		{
			name:      "成功更新项目",
			projectID: suite.testProjectID.String(),
			requestBody: map[string]interface{}{
				"name":        "更新后的项目名",
				"description": "更新后的项目描述",
			},
			setupMocks: func() {
				suite.mockService.On("UpdateProject", mock.Anything, mock.MatchedBy(func(project *models.Project) bool {
					return project.ID == suite.testProjectID && 
						   project.Name == "更新后的项目名" &&
						   project.TenantID == suite.testTenantID
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "无效的项目ID",
			projectID:      "invalid-uuid",
			requestBody:    map[string]interface{}{"name": "更新项目"},
			setupMocks:     func() {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "无效的项目ID",
		},
		{
			name:      "项目不存在",
			projectID: uuid.New().String(),
			requestBody: map[string]interface{}{
				"name": "不存在的项目",
			},
			setupMocks: func() {
				suite.mockService.On("UpdateProject", mock.Anything, mock.AnythingOfType("*models.Project")).
					Return(fmt.Errorf("项目不存在"))
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "项目不存在",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// 设置mocks
			tc.setupMocks()

			// 准备请求
			body, _ := json.Marshal(tc.requestBody)
			req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/projects/%s", tc.projectID), bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			suite.router.ServeHTTP(w, req)

			// 验证结果
			assert.Equal(suite.T(), tc.expectedStatus, w.Code)

			var response response.Response
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(suite.T(), err)

			if tc.expectedStatus == http.StatusOK {
				assert.True(suite.T(), response.Success)
			} else {
				assert.False(suite.T(), response.Success)
				if tc.expectedError != "" {
					assert.Contains(suite.T(), response.Message, tc.expectedError)
				}
			}
		})
	}
}

// TestDeleteProject 测试删除项目
func (suite *ProjectHandlerTestSuite) TestDeleteProject() {
	testCases := []struct {
		name           string
		projectID      string
		setupMocks     func()
		expectedStatus int
		expectedError  string
	}{
		{
			name:      "成功删除项目",
			projectID: suite.testProjectID.String(),
			setupMocks: func() {
				suite.mockService.On("DeleteProject", mock.Anything, suite.testProjectID, suite.testTenantID).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "无效的项目ID",
			projectID:      "invalid-uuid",
			setupMocks:     func() {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "无效的项目ID",
		},
		{
			name:      "项目不存在",
			projectID: uuid.New().String(),
			setupMocks: func() {
				suite.mockService.On("DeleteProject", mock.Anything, mock.AnythingOfType("uuid.UUID"), suite.testTenantID).
					Return(fmt.Errorf("项目不存在"))
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "项目不存在",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// 设置mocks
			tc.setupMocks()

			// 准备请求
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/projects/%s", tc.projectID), nil)
			w := httptest.NewRecorder()

			// 执行请求
			suite.router.ServeHTTP(w, req)

			// 验证结果
			assert.Equal(suite.T(), tc.expectedStatus, w.Code)

			var response response.Response
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(suite.T(), err)

			if tc.expectedStatus == http.StatusOK {
				assert.True(suite.T(), response.Success)
			} else {
				assert.False(suite.T(), response.Success)
				if tc.expectedError != "" {
					assert.Contains(suite.T(), response.Message, tc.expectedError)
				}
			}
		})
	}
}

// TestProjectMemberOperations 测试项目成员操作
func (suite *ProjectHandlerTestSuite) TestProjectMemberOperations() {
	memberUserID := uuid.New()
	roleID := uuid.New()

	// 测试添加成员
	suite.Run("添加项目成员", func() {
		suite.mockService.On("AddProjectMember", mock.Anything, mock.MatchedBy(func(req *service.AddMemberRequest) bool {
			return req.ProjectID == suite.testProjectID && 
				   req.UserID == memberUserID &&
				   req.RoleID == roleID &&
				   req.TenantID == suite.testTenantID &&
				   req.AddedByUserID == suite.testUserID
		})).Return(nil)

		body, _ := json.Marshal(map[string]interface{}{
			"user_id": memberUserID.String(),
			"role_id": roleID.String(),
		})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/projects/%s/members", suite.testProjectID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusCreated, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
	})

	// 测试获取成员列表
	suite.Run("获取项目成员列表", func() {
		members := []models.ProjectMember{
			{
				ID:        uuid.New(),
				ProjectID: suite.testProjectID,
				UserID:    memberUserID,
				RoleID:    roleID,
				AddedAt:   time.Now(),
			},
		}

		suite.mockService.On("GetProjectMembers", mock.Anything, suite.testProjectID, suite.testTenantID).
			Return(members, nil)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/members", suite.testProjectID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
	})

	// 测试移除成员
	suite.Run("移除项目成员", func() {
		suite.mockService.On("RemoveProjectMember", mock.Anything, mock.MatchedBy(func(req *service.RemoveMemberRequest) bool {
			return req.ProjectID == suite.testProjectID && 
				   req.UserID == memberUserID &&
				   req.TenantID == suite.testTenantID
		})).Return(nil)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/projects/%s/members/%s", suite.testProjectID.String(), memberUserID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
	})
}

// TestRepositoryOperations 测试仓库操作
func (suite *ProjectHandlerTestSuite) TestRepositoryOperations() {
	// 测试创建仓库
	suite.Run("创建项目仓库", func() {
		repository := &models.Repository{
			ID:            uuid.New(),
			ProjectID:     suite.testProjectID,
			Name:          "test-repo",
			Description:   "测试仓库",
			Visibility:    "private",
			DefaultBranch: "main",
		}

		suite.mockService.On("CreateRepository", mock.Anything, mock.MatchedBy(func(req *service.CreateRepositoryRequest) bool {
			return req.ProjectID == suite.testProjectID && 
				   req.Name == "test-repo" &&
				   req.TenantID == suite.testTenantID &&
				   req.UserID == suite.testUserID
		})).Return(repository, nil)

		body, _ := json.Marshal(map[string]interface{}{
			"name":          "test-repo",
			"description":   "测试仓库",
			"visibility":    "private", 
			"init_readme":   true,
		})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/projects/%s/repositories", suite.testProjectID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusCreated, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
	})

	// 测试获取仓库列表
	suite.Run("获取项目仓库列表", func() {
		repositories := []models.Repository{
			{
				ID:          uuid.New(),
				ProjectID:   suite.testProjectID,
				Name:        "repo1",
				Description: "仓库1",
				Visibility:  "private",
			},
			{
				ID:          uuid.New(),
				ProjectID:   suite.testProjectID,
				Name:        "repo2",
				Description: "仓库2",
				Visibility:  "public",
			},
		}

		suite.mockService.On("GetRepositories", mock.Anything, suite.testProjectID, suite.testTenantID).
			Return(repositories, nil)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/repositories", suite.testProjectID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
	})
}

// TestErrorHandling 测试错误处理
func (suite *ProjectHandlerTestSuite) TestErrorHandling() {
	testCases := []struct {
		name             string
		method           string
		url              string
		body             interface{}
		setupMocks       func()
		expectedStatus   int
		expectedContains string
	}{
		{
			name:   "缺少认证上下文",
			method: http.MethodGet,
			url:    "/api/v1/projects",
			setupMocks: func() {
				// 移除认证中间件来模拟缺少认证上下文的情况
				router := gin.New()
				router.GET("/api/v1/projects", suite.handler.ListProjects)
				suite.router = router
			},
			expectedStatus:   http.StatusInternalServerError,
			expectedContains: "获取认证信息失败",
		},
		{
			name:   "JSON解析错误",
			method: http.MethodPost,
			url:    "/api/v1/projects",
			body:   `{"invalid": json}`,
			setupMocks: func() {},
			expectedStatus: http.StatusBadRequest,
			expectedContains: "请求体格式错误",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// 设置mocks
			tc.setupMocks()

			// 准备请求
			var req *http.Request
			if tc.body != nil {
				if bodyStr, ok := tc.body.(string); ok {
					req = httptest.NewRequest(tc.method, tc.url, strings.NewReader(bodyStr))
				} else {
					body, _ := json.Marshal(tc.body)
					req = httptest.NewRequest(tc.method, tc.url, bytes.NewBuffer(body))
				}
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tc.method, tc.url, nil)
			}
			
			w := httptest.NewRecorder()

			// 执行请求
			suite.router.ServeHTTP(w, req)

			// 验证结果
			assert.Equal(suite.T(), tc.expectedStatus, w.Code)
			if tc.expectedContains != "" {
				assert.Contains(suite.T(), w.Body.String(), tc.expectedContains)
			}
		})
	}
}

// TestConcurrentRequests 测试并发请求
func (suite *ProjectHandlerTestSuite) TestConcurrentRequests() {
	// 设置mock期望
	suite.mockService.On("GetProject", mock.Anything, mock.AnythingOfType("uuid.UUID"), suite.testTenantID).
		Return(&models.Project{
			ID:       suite.testProjectID,
			TenantID: suite.testTenantID,
			Name:     "测试项目",
			Key:      "TEST_PROJECT",
		}, nil).Times(10)

	// 并发执行10个请求
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/projects/%s", suite.testProjectID.String()), nil)
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)
			
			assert.Equal(suite.T(), http.StatusOK, w.Code)
			done <- true
		}()
	}

	// 等待所有请求完成
	for i := 0; i < 10; i++ {
		<-done
	}
}

// 运行测试套件
func TestProjectHandlerSuite(t *testing.T) {
	suite.Run(t, new(ProjectHandlerTestSuite))
}

// TestProjectHandlerValidation 测试输入验证
func TestProjectHandlerValidation(t *testing.T) {
	mockService := new(MockProjectService)
	logger := zaptest.NewLogger(t)
	handler := handlers.NewProjectHandler(mockService, logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// 添加认证中间件
	testTenantID := uuid.New()
	testUserID := uuid.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", testTenantID.String())
		c.Set("user_id", testUserID.String())
		c.Next()
	})
	
	router.POST("/projects", handler.CreateProject)

	tests := []struct {
		name           string
		payload        string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "空请求体",
			payload:        "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "请求体不能为空",
		},
		{
			name:           "项目名称过长",
			payload:        `{"name": "` + strings.Repeat("a", 101) + `", "key": "TEST"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "项目名称长度不能超过100个字符",
		},
		{
			name:           "项目键包含非法字符",
			payload:        `{"name": "测试", "key": "test-project-!@#"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "项目键只能包含字母、数字和下划线",
		},
		{
			name:           "描述过长",
			payload:        `{"name": "测试", "key": "TEST", "description": "` + strings.Repeat("a", 501) + `"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "项目描述长度不能超过500个字符",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(tt.payload))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}
		})
	}
}