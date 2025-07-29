package unit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/client"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/compensation"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/models"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/repository"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/service"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/transaction"
)

// MockProjectRepository 项目存储库模拟
type MockProjectRepository struct {
	mock.Mock
}

func (m *MockProjectRepository) Create(ctx context.Context, project *models.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockProjectRepository) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.Project, error) {
	args := m.Called(ctx, id, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Project), args.Error(1)
}

func (m *MockProjectRepository) GetByKey(ctx context.Context, key string, tenantID uuid.UUID) (*models.Project, error) {
	args := m.Called(ctx, key, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Project), args.Error(1)
}

func (m *MockProjectRepository) Update(ctx context.Context, project *models.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockProjectRepository) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	args := m.Called(ctx, id, tenantID)
	return args.Error(0)
}

func (m *MockProjectRepository) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, filters map[string]interface{}) ([]models.Project, int64, error) {
	args := m.Called(ctx, tenantID, page, pageSize, filters)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]models.Project), args.Get(1).(int64), args.Error(2)
}

func (m *MockProjectRepository) AddMember(ctx context.Context, member *models.ProjectMember) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockProjectRepository) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	args := m.Called(ctx, projectID, userID)
	return args.Error(0)
}

func (m *MockProjectRepository) GetMembers(ctx context.Context, projectID uuid.UUID) ([]models.ProjectMember, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.ProjectMember), args.Error(1)
}

func (m *MockProjectRepository) GetMemberRole(ctx context.Context, projectID, userID uuid.UUID) (*models.Role, error) {
	args := m.Called(ctx, projectID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockProjectRepository) CheckUserAccess(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, projectID, userID)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockProjectRepository) GetUserProjects(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) ([]models.Project, error) {
	args := m.Called(ctx, userID, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Project), args.Error(1)
}

// MockDistributedTransactionManager 分布式事务管理器模拟
type MockDistributedTransactionManager struct {
	mock.Mock
}

func (m *MockDistributedTransactionManager) CreateRepositoryTransaction(
	ctx context.Context,
	projectID, userID, tenantID uuid.UUID,
	createReq *client.CreateRepositoryRequest,
) (*models.Repository, error) {
	args := m.Called(ctx, projectID, userID, tenantID, createReq)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Repository), args.Error(1)
}

func (m *MockDistributedTransactionManager) GetTransaction(transactionID uuid.UUID) (*transaction.DistributedTransaction, error) {
	args := m.Called(transactionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*transaction.DistributedTransaction), args.Error(1)
}

// ProjectServiceTestSuite 项目服务测试套件
type ProjectServiceTestSuite struct {
	suite.Suite
	service             service.ProjectService
	mockRepo            *MockProjectRepository
	mockGitClient       *MockGitGatewayClient
	mockTransactionMgr  *MockDistributedTransactionManager
	logger              *zap.Logger
	testProjectID       uuid.UUID
	testTenantID        uuid.UUID
	testUserID          uuid.UUID
	testRepositoryID    uuid.UUID
}

func (suite *ProjectServiceTestSuite) SetupSuite() {
	suite.logger = zaptest.NewLogger(suite.T())
	
	suite.testProjectID = uuid.New()
	suite.testTenantID = uuid.New()
	suite.testUserID = uuid.New()
	suite.testRepositoryID = uuid.New()
}

func (suite *ProjectServiceTestSuite) SetupTest() {
	suite.mockRepo = new(MockProjectRepository)
	suite.mockGitClient = new(MockGitGatewayClient)
	suite.mockTransactionMgr = new(MockDistributedTransactionManager)
	
	// 创建项目服务实例
	suite.service = service.NewProjectService(suite.mockRepo, suite.mockGitClient, suite.logger)
}

func (suite *ProjectServiceTestSuite) TearDownTest() {
	suite.mockRepo.AssertExpectations(suite.T())
	suite.mockGitClient.AssertExpectations(suite.T())
	suite.mockTransactionMgr.AssertExpectations(suite.T())
}

// TestCreateProject 测试创建项目
func (suite *ProjectServiceTestSuite) TestCreateProject() {
	testCases := []struct {
		name           string
		request        *models.CreateProjectRequest
		setupMocks     func()
		expectError    bool
		expectedError  string
	}{
		{
			name: "成功创建项目",
			request: &models.CreateProjectRequest{
				Name:        "测试项目",
				Key:         "TEST_PROJECT",
				Description: "这是一个测试项目",
			},
			setupMocks: func() {
				suite.mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(project *models.Project) bool {
					return project.Name == "测试项目" && 
						   project.Key == "TEST_PROJECT" && 
						   project.TenantID == suite.testTenantID &&
						   project.ManagerID != nil && *project.ManagerID == suite.testUserID
				})).Return(nil)
			},
			expectError: false,
		},
		{
			name: "项目名称为空",
			request: &models.CreateProjectRequest{
				Key:         "TEST_PROJECT",
				Description: "这是一个测试项目",
			},
			setupMocks:    func() {},
			expectError:   true,
			expectedError: "项目名称不能为空",
		},
		{
			name: "项目键为空",
			request: &models.CreateProjectRequest{
				Name:        "测试项目",
				Description: "这是一个测试项目",
			},
			setupMocks:    func() {},
			expectError:   true,
			expectedError: "项目键不能为空",
		},
		{
			name: "项目键格式无效",
			request: &models.CreateProjectRequest{
				Name: "测试项目",
				Key:  "test-project-!@#",
			},
			setupMocks:    func() {},
			expectError:   true,
			expectedError: "项目键只能包含字母、数字和下划线",
		},
		{
			name: "项目名称过长",
			request: &models.CreateProjectRequest{
				Name: "这是一个非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常长的项目名称",
				Key:  "LONG_PROJECT",
			},
			setupMocks:    func() {},
			expectError:   true,
			expectedError: "项目名称长度不能超过100个字符",
		},
		{
			name: "数据库创建失败",
			request: &models.CreateProjectRequest{
				Name: "测试项目",
				Key:  "TEST_PROJECT",
			},
			setupMocks: func() {
				suite.mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Project")).
					Return(fmt.Errorf("数据库连接失败"))
			},
			expectError:   true,
			expectedError: "创建项目失败",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// 设置mocks
			tc.setupMocks()

			// 执行测试
			project, err := suite.service.CreateProject(
				context.Background(),
				tc.request,
				suite.testUserID,
				suite.testTenantID,
			)

			// 验证结果
			if tc.expectError {
				assert.Error(suite.T(), err)
				assert.Nil(suite.T(), project)
				if tc.expectedError != "" {
					assert.Contains(suite.T(), err.Error(), tc.expectedError)
				}
			} else {
				assert.NoError(suite.T(), err)
				assert.NotNil(suite.T(), project)
				assert.Equal(suite.T(), tc.request.Name, project.Name)
				assert.Equal(suite.T(), tc.request.Key, project.Key)
				assert.Equal(suite.T(), suite.testTenantID, project.TenantID)
				assert.NotNil(suite.T(), project.ManagerID)
				assert.Equal(suite.T(), suite.testUserID, *project.ManagerID)
			}
		})
	}
}

// TestGetProject 测试获取项目
func (suite *ProjectServiceTestSuite) TestGetProject() {
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
		name          string
		projectID     uuid.UUID
		setupMocks    func()
		expectError   bool
		expectedError string
	}{
		{
			name:      "成功获取项目",
			projectID: suite.testProjectID,
			setupMocks: func() {
				suite.mockRepo.On("GetByID", mock.Anything, suite.testProjectID, suite.testTenantID).
					Return(testProject, nil)
				suite.mockRepo.On("CheckUserAccess", mock.Anything, suite.testProjectID, suite.testUserID).
					Return(true, nil)
			},
			expectError: false,
		},
		{
			name:      "项目不存在",
			projectID: uuid.New(),
			setupMocks: func() {
				suite.mockRepo.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID"), suite.testTenantID).
					Return(nil, fmt.Errorf("项目不存在"))
			},
			expectError:   true,
			expectedError: "项目不存在",
		},
		{
			name:      "用户无权限访问",
			projectID: suite.testProjectID,
			setupMocks: func() {
				suite.mockRepo.On("GetByID", mock.Anything, suite.testProjectID, suite.testTenantID).
					Return(testProject, nil)
				suite.mockRepo.On("CheckUserAccess", mock.Anything, suite.testProjectID, suite.testUserID).
					Return(false, nil)
			},
			expectError:   true,
			expectedError: "无权限访问该项目",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// 设置mocks
			tc.setupMocks()

			// 执行测试
			project, err := suite.service.GetProject(
				context.Background(),
				tc.projectID,
				suite.testUserID,
				suite.testTenantID,
			)

			// 验证结果
			if tc.expectError {
				assert.Error(suite.T(), err)
				assert.Nil(suite.T(), project)
				if tc.expectedError != "" {
					assert.Contains(suite.T(), err.Error(), tc.expectedError)
				}
			} else {
				assert.NoError(suite.T(), err)
				assert.NotNil(suite.T(), project)
				assert.Equal(suite.T(), testProject.ID, project.ID)
				assert.Equal(suite.T(), testProject.Name, project.Name)
			}
		})
	}
}

// TestUpdateProject 测试更新项目
func (suite *ProjectServiceTestSuite) TestUpdateProject() {
	existingProject := &models.Project{
		ID:          suite.testProjectID,
		TenantID:    suite.testTenantID,
		Name:        "原项目名",
		Key:         "ORIGINAL_PROJECT",
		Description: "原描述",
		Status:      "active",
		ManagerID:   &suite.testUserID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	testCases := []struct {
		name          string
		request       *models.UpdateProjectRequest
		setupMocks    func()
		expectError   bool
		expectedError string
	}{
		{
			name: "成功更新项目",
			request: &models.UpdateProjectRequest{
				Name:        "更新后的项目名",
				Description: "更新后的描述",
			},
			setupMocks: func() {
				suite.mockRepo.On("GetByID", mock.Anything, suite.testProjectID, suite.testTenantID).
					Return(existingProject, nil)
				suite.mockRepo.On("CheckUserAccess", mock.Anything, suite.testProjectID, suite.testUserID).
					Return(true, nil)
				suite.mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(project *models.Project) bool {
					return project.ID == suite.testProjectID && 
						   project.Name == "更新后的项目名" &&
						   project.Description == "更新后的描述"
				})).Return(nil)
			},
			expectError: false,
		},
		{
			name: "项目不存在",
			request: &models.UpdateProjectRequest{
				Name: "不存在的项目",
			},
			setupMocks: func() {
				suite.mockRepo.On("GetByID", mock.Anything, suite.testProjectID, suite.testTenantID).
					Return(nil, fmt.Errorf("项目不存在"))
			},
			expectError:   true,
			expectedError: "项目不存在",
		},
		{
			name: "用户无权限更新",
			request: &models.UpdateProjectRequest{
				Name: "尝试更新",
			},
			setupMocks: func() {
				suite.mockRepo.On("GetByID", mock.Anything, suite.testProjectID, suite.testTenantID).
					Return(existingProject, nil)
				suite.mockRepo.On("CheckUserAccess", mock.Anything, suite.testProjectID, suite.testUserID).
					Return(false, nil)
			},
			expectError:   true,
			expectedError: "无权限更新该项目",
		},
		{
			name: "更新的项目名过长",
			request: &models.UpdateProjectRequest{
				Name: "这是一个非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常长的项目名称",
			},
			setupMocks: func() {
				suite.mockRepo.On("GetByID", mock.Anything, suite.testProjectID, suite.testTenantID).
					Return(existingProject, nil)
				suite.mockRepo.On("CheckUserAccess", mock.Anything, suite.testProjectID, suite.testUserID).
					Return(true, nil)
			},
			expectError:   true,
			expectedError: "项目名称长度不能超过100个字符",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// 设置mocks
			tc.setupMocks()

			// 执行测试
			project, err := suite.service.UpdateProject(
				context.Background(),
				suite.testProjectID,
				tc.request,
				suite.testUserID,
				suite.testTenantID,
			)

			// 验证结果
			if tc.expectError {
				assert.Error(suite.T(), err)
				assert.Nil(suite.T(), project)
				if tc.expectedError != "" {
					assert.Contains(suite.T(), err.Error(), tc.expectedError)
				}
			} else {
				assert.NoError(suite.T(), err)
				assert.NotNil(suite.T(), project)
				if tc.request.Name != "" {
					assert.Equal(suite.T(), tc.request.Name, project.Name)
				}
				if tc.request.Description != "" {
					assert.Equal(suite.T(), tc.request.Description, project.Description)
				}
			}
		})
	}
}

// TestDeleteProject 测试删除项目
func (suite *ProjectServiceTestSuite) TestDeleteProject() {
	existingProject := &models.Project{
		ID:        suite.testProjectID,
		TenantID:  suite.testTenantID,
		Name:      "测试项目",
		Key:       "TEST_PROJECT",
		Status:    "active",
		ManagerID: &suite.testUserID,
	}

	testCases := []struct {
		name          string
		setupMocks    func()
		expectError   bool
		expectedError string
	}{
		{
			name: "成功删除项目",
			setupMocks: func() {
				suite.mockRepo.On("GetByID", mock.Anything, suite.testProjectID, suite.testTenantID).
					Return(existingProject, nil)
				suite.mockRepo.On("CheckUserAccess", mock.Anything, suite.testProjectID, suite.testUserID).
					Return(true, nil)
				suite.mockRepo.On("Delete", mock.Anything, suite.testProjectID, suite.testTenantID).
					Return(nil)
			},
			expectError: false,
		},
		{
			name: "项目不存在",
			setupMocks: func() {
				suite.mockRepo.On("GetByID", mock.Anything, suite.testProjectID, suite.testTenantID).
					Return(nil, fmt.Errorf("项目不存在"))
			},
			expectError:   true,
			expectedError: "项目不存在",
		},
		{
			name: "用户无权限删除",
			setupMocks: func() {
				suite.mockRepo.On("GetByID", mock.Anything, suite.testProjectID, suite.testTenantID).
					Return(existingProject, nil)
				suite.mockRepo.On("CheckUserAccess", mock.Anything, suite.testProjectID, suite.testUserID).
					Return(false, nil)
			},
			expectError:   true,
			expectedError: "无权限删除该项目",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// 设置mocks
			tc.setupMocks()

			// 执行测试
			err := suite.service.DeleteProject(
				context.Background(),
				suite.testProjectID,
				suite.testUserID,
				suite.testTenantID,
			)

			// 验证结果
			if tc.expectError {
				assert.Error(suite.T(), err)
				if tc.expectedError != "" {
					assert.Contains(suite.T(), err.Error(), tc.expectedError)
				}
			} else {
				assert.NoError(suite.T(), err)
			}
		})
	}
}

// TestListProjects 测试获取项目列表
func (suite *ProjectServiceTestSuite) TestListProjects() {
	testProjects := []models.Project{
		{
			ID:        uuid.New(),
			TenantID:  suite.testTenantID,
			Name:      "项目1",
			Key:       "PROJECT_1",
			Status:    "active",
			ManagerID: &suite.testUserID,
			CreatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			TenantID:  suite.testTenantID,
			Name:      "项目2",
			Key:       "PROJECT_2",
			Status:    "active",
			ManagerID: &suite.testUserID,
			CreatedAt: time.Now(),
		},
	}

	testCases := []struct {
		name        string
		page        int
		pageSize    int
		filters     map[string]interface{}
		setupMocks  func()
		expectError bool
	}{
		{
			name:     "成功获取项目列表",
			page:     1,
			pageSize: 10,
			filters:  map[string]interface{}{},
			setupMocks: func() {
				suite.mockRepo.On("List", mock.Anything, suite.testTenantID, 1, 10, mock.AnythingOfType("map[string]interface {}")).
					Return(testProjects, int64(2), nil)
			},
			expectError: false,
		},
		{
			name:     "带搜索条件的项目列表",
			page:     1,
			pageSize: 10,
			filters:  map[string]interface{}{"search": "项目1"},
			setupMocks: func() {
				suite.mockRepo.On("List", mock.Anything, suite.testTenantID, 1, 10, mock.MatchedBy(func(filters map[string]interface{}) bool {
					return filters["search"] == "项目1"
				})).Return(testProjects[:1], int64(1), nil)
			},
			expectError: false,
		},
		{
			name:     "数据库查询失败",
			page:     1,
			pageSize: 10,
			filters:  map[string]interface{}{},
			setupMocks: func() {
				suite.mockRepo.On("List", mock.Anything, suite.testTenantID, 1, 10, mock.AnythingOfType("map[string]interface {}")).
					Return(nil, int64(0), fmt.Errorf("数据库连接失败"))
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// 设置mocks
			tc.setupMocks()

			// 执行测试
			response, err := suite.service.ListProjects(
				context.Background(),
				tc.page,
				tc.pageSize,
				tc.filters,
				suite.testUserID,
				suite.testTenantID,
			)

			// 验证结果
			if tc.expectError {
				assert.Error(suite.T(), err)
				assert.Nil(suite.T(), response)
			} else {
				assert.NoError(suite.T(), err)
				assert.NotNil(suite.T(), response)
				assert.Equal(suite.T(), tc.page, response.Page)
				assert.Equal(suite.T(), tc.pageSize, response.PageSize)
			}
		})
	}
}

// TestProjectMemberOperations 测试项目成员操作
func (suite *ProjectServiceTestSuite) TestProjectMemberOperations() {
	project := &models.Project{
		ID:        suite.testProjectID,
		TenantID:  suite.testTenantID,
		Name:      "测试项目",
		Status:    "active",
		ManagerID: &suite.testUserID,
	}

	memberUserID := uuid.New()
	roleID := uuid.New()

	// 测试添加成员
	suite.Run("添加项目成员", func() {
		request := &models.AddMemberRequest{
			UserID: memberUserID,
			RoleID: roleID,
		}

		suite.mockRepo.On("GetByID", mock.Anything, suite.testProjectID, suite.testTenantID).
			Return(project, nil)
		suite.mockRepo.On("CheckUserAccess", mock.Anything, suite.testProjectID, suite.testUserID).
			Return(true, nil)
		suite.mockRepo.On("AddMember", mock.Anything, mock.MatchedBy(func(member *models.ProjectMember) bool {
			return member.ProjectID == suite.testProjectID && 
				   member.UserID == memberUserID &&
				   member.RoleID == roleID &&
				   member.AddedByUserID == suite.testUserID
		})).Return(nil)

		err := suite.service.AddMember(
			context.Background(),
			suite.testProjectID,
			request,
			suite.testUserID,
			suite.testTenantID,
		)

		assert.NoError(suite.T(), err)
	})

	// 测试移除成员
	suite.Run("移除项目成员", func() {
		suite.mockRepo.On("GetByID", mock.Anything, suite.testProjectID, suite.testTenantID).
			Return(project, nil)
		suite.mockRepo.On("CheckUserAccess", mock.Anything, suite.testProjectID, suite.testUserID).
			Return(true, nil)
		suite.mockRepo.On("RemoveMember", mock.Anything, suite.testProjectID, memberUserID).
			Return(nil)

		err := suite.service.RemoveMember(
			context.Background(),
			suite.testProjectID,
			memberUserID,
			suite.testUserID,
			suite.testTenantID,
		)

		assert.NoError(suite.T(), err)
	})

	// 测试获取成员列表
	suite.Run("获取项目成员列表", func() {
		members := []models.ProjectMember{
			{
				ID:            uuid.New(),
				ProjectID:     suite.testProjectID,
				UserID:        memberUserID,
				RoleID:        roleID,
				AddedByUserID: suite.testUserID,
				AddedAt:       time.Now(),
			},
		}

		suite.mockRepo.On("GetByID", mock.Anything, suite.testProjectID, suite.testTenantID).
			Return(project, nil)
		suite.mockRepo.On("CheckUserAccess", mock.Anything, suite.testProjectID, suite.testUserID).
			Return(true, nil)
		suite.mockRepo.On("GetMembers", mock.Anything, suite.testProjectID).
			Return(members, nil)

		result, err := suite.service.GetMembers(
			context.Background(),
			suite.testProjectID,
			suite.testUserID,
			suite.testTenantID,
		)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.Len(suite.T(), result, 1)
		assert.Equal(suite.T(), memberUserID, result[0].UserID)
	})
}

// TestRepositoryOperations 测试仓库操作
func (suite *ProjectServiceTestSuite) TestRepositoryOperations() {
	project := &models.Project{
		ID:        suite.testProjectID,
		TenantID:  suite.testTenantID,
		Name:      "测试项目",
		Status:    "active",
		ManagerID: &suite.testUserID,
	}

	// 测试创建仓库（使用分布式事务）
	suite.Run("创建项目仓库", func() {
		request := &service.CreateRepositoryRequest{
			Name:          "test-repo",
			Description:   "测试仓库",
			Visibility:    "private",
			DefaultBranch: "main",
			InitReadme:    true,
		}

		repository := &models.Repository{
			ID:            suite.testRepositoryID,
			ProjectID:     suite.testProjectID,
			Name:          "test-repo",
			Description:   "测试仓库",
			Visibility:    "private",
			DefaultBranch: "main",
		}

		suite.mockRepo.On("GetByID", mock.Anything, suite.testProjectID, suite.testTenantID).
			Return(project, nil)
		suite.mockRepo.On("CheckUserAccess", mock.Anything, suite.testProjectID, suite.testUserID).
			Return(true, nil)

		// 模拟分布式事务管理器的调用
		// 注意：这里简化了实现，实际应该通过依赖注入使用mock的事务管理器
		suite.mockGitClient.On("CreateRepository", mock.Anything, mock.MatchedBy(func(req *client.CreateRepositoryRequest) bool {
			return req.Name == "test-repo" && req.ProjectID == suite.testProjectID.String()
		})).Return(&client.Repository{
			ID:            suite.testRepositoryID,
			ProjectID:     suite.testProjectID,
			Name:          "test-repo",
			Description:   &request.Description,
			Visibility:    client.RepositoryVisibilityPrivate,
			DefaultBranch: "main",
		}, nil)

		result, err := suite.service.CreateRepository(
			context.Background(),
			suite.testProjectID,
			request,
			suite.testUserID,
			suite.testTenantID,
		)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.Equal(suite.T(), "test-repo", result.Name)
		assert.Equal(suite.T(), suite.testProjectID, result.ProjectID)
	})
}

// TestAccessControl 测试访问控制
func (suite *ProjectServiceTestSuite) TestAccessControl() {
	// 测试权限检查
	suite.Run("检查项目访问权限", func() {
		suite.mockRepo.On("CheckUserAccess", mock.Anything, suite.testProjectID, suite.testUserID).
			Return(true, nil)

		hasAccess, err := suite.service.CheckProjectAccess(
			context.Background(),
			suite.testProjectID,
			suite.testUserID,
		)

		assert.NoError(suite.T(), err)
		assert.True(suite.T(), hasAccess)
	})

	// 测试获取用户项目列表
	suite.Run("获取用户项目列表", func() {
		userProjects := []models.Project{
			{
				ID:        suite.testProjectID,
				TenantID:  suite.testTenantID,
				Name:      "用户项目1",
				Key:       "USER_PROJECT_1",
				Status:    "active",
				ManagerID: &suite.testUserID,
			},
		}

		suite.mockRepo.On("GetUserProjects", mock.Anything, suite.testUserID, suite.testTenantID).
			Return(userProjects, nil)

		projects, err := suite.service.GetUserProjects(
			context.Background(),
			suite.testUserID,
			suite.testTenantID,
		)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), projects)
		assert.Len(suite.T(), projects, 1)
		assert.Equal(suite.T(), "用户项目1", projects[0].Name)
	})
}

// TestErrorScenarios 测试错误场景
func (suite *ProjectServiceTestSuite) TestErrorScenarios() {
	// 测试验证错误
	suite.Run("验证失败场景", func() {
		// 空的项目名称
		request := &models.CreateProjectRequest{
			Key: "TEST_PROJECT",
		}

		project, err := suite.service.CreateProject(
			context.Background(),
			request,
			suite.testUserID,
			suite.testTenantID,
		)

		assert.Error(suite.T(), err)
		assert.Nil(suite.T(), project)
		assert.Contains(suite.T(), err.Error(), "项目名称不能为空")
	})

	// 测试数据库连接错误
	suite.Run("数据库连接失败", func() {
		request := &models.CreateProjectRequest{
			Name: "测试项目",
			Key:  "TEST_PROJECT",
		}

		suite.mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Project")).
			Return(fmt.Errorf("database connection failed"))

		project, err := suite.service.CreateProject(
			context.Background(),
			request,
			suite.testUserID,
			suite.testTenantID,
		)

		assert.Error(suite.T(), err)
		assert.Nil(suite.T(), project)
		assert.Contains(suite.T(), err.Error(), "创建项目失败")
	})
}

// TestConcurrentOperations 测试并发操作
func (suite *ProjectServiceTestSuite) TestConcurrentOperations() {
	project := &models.Project{
		ID:        suite.testProjectID,
		TenantID:  suite.testTenantID,
		Name:      "测试项目",
		Status:    "active",
		ManagerID: &suite.testUserID,
	}

	// 设置并发获取项目的mock
	suite.mockRepo.On("GetByID", mock.Anything, suite.testProjectID, suite.testTenantID).
		Return(project, nil).Times(10)
	suite.mockRepo.On("CheckUserAccess", mock.Anything, suite.testProjectID, suite.testUserID).
		Return(true, nil).Times(10)

	// 并发执行10个获取项目操作
	done := make(chan bool, 10)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func() {
			result, err := suite.service.GetProject(
				context.Background(),
				suite.testProjectID,
				suite.testUserID,
				suite.testTenantID,
			)

			if err != nil {
				errors <- err
			} else {
				assert.NotNil(suite.T(), result)
				assert.Equal(suite.T(), suite.testProjectID, result.ID)
			}
			done <- true
		}()
	}

	// 等待所有操作完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 检查是否有错误
	select {
	case err := <-errors:
		suite.T().Errorf("并发操作出现错误: %v", err)
	default:
		// 没有错误，测试通过
	}
}

// 运行测试套件
func TestProjectServiceSuite(t *testing.T) {
	suite.Run(t, new(ProjectServiceTestSuite))
}

// TestValidationHelpers 测试验证辅助函数
func TestValidationHelpers(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockRepo := new(MockProjectRepository)
	mockGitClient := new(MockGitGatewayClient)
	service := service.NewProjectService(mockRepo, mockGitClient, logger)

	// 由于validateProjectKey是私有方法，我们通过公共方法间接测试
	testCases := []struct {
		name        string
		projectKey  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "有效的项目键",
			projectKey:  "VALID_PROJECT_KEY",
			expectError: false,
		},
		{
			name:        "包含小写字母的项目键",
			projectKey:  "Valid_Project_Key",
			expectError: false,
		},
		{
			name:        "包含数字的项目键",
			projectKey:  "PROJECT_123",
			expectError: false,
		},
		{
			name:        "包含特殊字符的项目键",
			projectKey:  "PROJECT-KEY!",
			expectError: true,
			errorMsg:    "项目键只能包含字母、数字和下划线",
		},
		{
			name:        "空的项目键",
			projectKey:  "",
			expectError: true,
			errorMsg:    "项目键不能为空",
		},
		{
			name:        "过长的项目键",
			projectKey:  "THIS_IS_A_VERY_VERY_VERY_VERY_VERY_VERY_VERY_VERY_VERY_VERY_VERY_VERY_LONG_PROJECT_KEY_THAT_EXCEEDS_THE_MAXIMUM_LENGTH",
			expectError: true,
			errorMsg:    "项目键长度不能超过50个字符",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request := &models.CreateProjectRequest{
				Name: "测试项目",
				Key:  tc.projectKey,
			}

			if !tc.expectError {
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Project")).
					Return(nil).Once()
			}

			_, err := service.CreateProject(
				context.Background(),
				request,
				uuid.New(),
				uuid.New(),
			)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}