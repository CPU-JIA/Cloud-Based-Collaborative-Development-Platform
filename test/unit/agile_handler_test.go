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

	"github.com/cloud-platform/collaborative-dev/internal/project-service/handler"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/models"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/service"
	"github.com/cloud-platform/collaborative-dev/shared/response"
)

// MockAgileService 敏捷服务模拟
type MockAgileService struct {
	mock.Mock
}

// Sprint operations
func (m *MockAgileService) CreateSprint(ctx context.Context, req *service.CreateSprintRequest) (*models.Sprint, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Sprint), args.Error(1)
}

func (m *MockAgileService) GetSprint(ctx context.Context, sprintID, tenantID uuid.UUID) (*models.Sprint, error) {
	args := m.Called(ctx, sprintID, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Sprint), args.Error(1)
}

func (m *MockAgileService) UpdateSprint(ctx context.Context, req *service.UpdateSprintRequest) (*models.Sprint, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Sprint), args.Error(1)
}

func (m *MockAgileService) DeleteSprint(ctx context.Context, sprintID, tenantID uuid.UUID) error {
	args := m.Called(ctx, sprintID, tenantID)
	return args.Error(0)
}

func (m *MockAgileService) ListSprints(ctx context.Context, projectID, tenantID uuid.UUID, req *service.ListSprintsRequest) (*service.SprintListResponse, error) {
	args := m.Called(ctx, projectID, tenantID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.SprintListResponse), args.Error(1)
}

func (m *MockAgileService) StartSprint(ctx context.Context, sprintID, tenantID uuid.UUID) error {
	args := m.Called(ctx, sprintID, tenantID)
	return args.Error(0)
}

func (m *MockAgileService) CompleteSprint(ctx context.Context, sprintID, tenantID uuid.UUID) error {
	args := m.Called(ctx, sprintID, tenantID)
	return args.Error(0)
}

// Epic operations
func (m *MockAgileService) CreateEpic(ctx context.Context, req *service.CreateEpicRequest) (*models.Epic, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Epic), args.Error(1)
}

func (m *MockAgileService) GetEpic(ctx context.Context, epicID, tenantID uuid.UUID) (*models.Epic, error) {
	args := m.Called(ctx, epicID, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Epic), args.Error(1)
}

func (m *MockAgileService) UpdateEpic(ctx context.Context, req *service.UpdateEpicRequest) (*models.Epic, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Epic), args.Error(1)
}

func (m *MockAgileService) DeleteEpic(ctx context.Context, epicID, tenantID uuid.UUID) error {
	args := m.Called(ctx, epicID, tenantID)
	return args.Error(0)
}

func (m *MockAgileService) ListEpics(ctx context.Context, projectID, tenantID uuid.UUID, req *service.ListEpicsRequest) (*service.EpicListResponse, error) {
	args := m.Called(ctx, projectID, tenantID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.EpicListResponse), args.Error(1)
}

// Task operations
func (m *MockAgileService) CreateTask(ctx context.Context, req *service.CreateTaskRequest) (*models.AgileTask, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AgileTask), args.Error(1)
}

func (m *MockAgileService) GetTask(ctx context.Context, taskID, tenantID uuid.UUID) (*models.AgileTask, error) {
	args := m.Called(ctx, taskID, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AgileTask), args.Error(1)
}

func (m *MockAgileService) UpdateTask(ctx context.Context, req *service.UpdateTaskRequest) (*models.AgileTask, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AgileTask), args.Error(1)
}

func (m *MockAgileService) DeleteTask(ctx context.Context, taskID, tenantID uuid.UUID) error {
	args := m.Called(ctx, taskID, tenantID)
	return args.Error(0)
}

func (m *MockAgileService) ListTasks(ctx context.Context, projectID, tenantID uuid.UUID, req *service.ListTasksRequest) (*service.TaskListResponse, error) {
	args := m.Called(ctx, projectID, tenantID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.TaskListResponse), args.Error(1)
}

func (m *MockAgileService) UpdateTaskStatus(ctx context.Context, req *service.UpdateTaskStatusRequest) (*models.AgileTask, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AgileTask), args.Error(1)
}

func (m *MockAgileService) AssignTask(ctx context.Context, req *service.AssignTaskRequest) (*models.AgileTask, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AgileTask), args.Error(1)
}

func (m *MockAgileService) ReorderTasks(ctx context.Context, req *service.ReorderTasksRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

// Board operations
func (m *MockAgileService) CreateBoard(ctx context.Context, req *service.CreateBoardRequest) (*models.Board, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Board), args.Error(1)
}

func (m *MockAgileService) GetBoard(ctx context.Context, boardID, tenantID uuid.UUID) (*models.Board, error) {
	args := m.Called(ctx, boardID, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Board), args.Error(1)
}

func (m *MockAgileService) UpdateBoard(ctx context.Context, req *service.UpdateBoardRequest) (*models.Board, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Board), args.Error(1)
}

func (m *MockAgileService) DeleteBoard(ctx context.Context, boardID, tenantID uuid.UUID) error {
	args := m.Called(ctx, boardID, tenantID)
	return args.Error(0)
}

func (m *MockAgileService) ListBoards(ctx context.Context, projectID, tenantID uuid.UUID, req *service.ListBoardsRequest) (*service.BoardListResponse, error) {
	args := m.Called(ctx, projectID, tenantID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.BoardListResponse), args.Error(1)
}

// Analytics operations
func (m *MockAgileService) GetSprintBurndown(ctx context.Context, sprintID, tenantID uuid.UUID) (*service.BurndownChart, error) {
	args := m.Called(ctx, sprintID, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.BurndownChart), args.Error(1)
}

func (m *MockAgileService) GetVelocityChart(ctx context.Context, projectID, tenantID uuid.UUID, req *service.VelocityChartRequest) (*service.VelocityChart, error) {
	args := m.Called(ctx, projectID, tenantID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.VelocityChart), args.Error(1)
}

// AgileHandlerTestSuite 敏捷处理器测试套件
type AgileHandlerTestSuite struct {
	suite.Suite
	handler       *handler.AgileHandler
	mockService   *MockAgileService
	router        *gin.Engine
	logger        *zap.Logger
	testProjectID uuid.UUID
	testTenantID  uuid.UUID
	testUserID    uuid.UUID
	testSprintID  uuid.UUID
	testEpicID    uuid.UUID
	testTaskID    uuid.UUID
	testBoardID   uuid.UUID
}

func (suite *AgileHandlerTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	suite.logger = zaptest.NewLogger(suite.T())
	
	suite.testProjectID = uuid.New()
	suite.testTenantID = uuid.New()
	suite.testUserID = uuid.New()
	suite.testSprintID = uuid.New()
	suite.testEpicID = uuid.New()
	suite.testTaskID = uuid.New()
	suite.testBoardID = uuid.New()
}

func (suite *AgileHandlerTestSuite) SetupTest() {
	suite.mockService = new(MockAgileService)
	suite.handler = handler.NewAgileHandler(suite.mockService, suite.logger)
	
	suite.router = gin.New()
	suite.setupRoutes()
}

func (suite *AgileHandlerTestSuite) TearDownTest() {
	suite.mockService.AssertExpectations(suite.T())
}

func (suite *AgileHandlerTestSuite) setupRoutes() {
	api := suite.router.Group("/api/v1")
	{
		projects := api.Group("/projects/:project_id")
		{
			// Sprint routes
			sprints := projects.Group("/sprints")
			{
				sprints.POST("", suite.addAuthContext, suite.handler.CreateSprint)
				sprints.GET("", suite.addAuthContext, suite.handler.ListSprints)
				sprints.GET("/:id", suite.addAuthContext, suite.handler.GetSprint)
				sprints.PUT("/:id", suite.addAuthContext, suite.handler.UpdateSprint)
				sprints.DELETE("/:id", suite.addAuthContext, suite.handler.DeleteSprint)
				sprints.POST("/:id/start", suite.addAuthContext, suite.handler.StartSprint)
				sprints.POST("/:id/complete", suite.addAuthContext, suite.handler.CompleteSprint)
				sprints.GET("/:id/burndown", suite.addAuthContext, suite.handler.GetSprintBurndown)
			}
			
			// Epic routes
			epics := projects.Group("/epics")
			{
				epics.POST("", suite.addAuthContext, suite.handler.CreateEpic)
				epics.GET("", suite.addAuthContext, suite.handler.ListEpics)
				epics.GET("/:id", suite.addAuthContext, suite.handler.GetEpic)
				epics.PUT("/:id", suite.addAuthContext, suite.handler.UpdateEpic)
				epics.DELETE("/:id", suite.addAuthContext, suite.handler.DeleteEpic)
			}
			
			// Task routes
			tasks := projects.Group("/tasks")
			{
				tasks.POST("", suite.addAuthContext, suite.handler.CreateTask)
				tasks.GET("", suite.addAuthContext, suite.handler.ListTasks)
				tasks.GET("/:id", suite.addAuthContext, suite.handler.GetTask)
				tasks.PUT("/:id", suite.addAuthContext, suite.handler.UpdateTask)
				tasks.DELETE("/:id", suite.addAuthContext, suite.handler.DeleteTask)
				tasks.PUT("/:id/status", suite.addAuthContext, suite.handler.UpdateTaskStatus)
				tasks.PUT("/:id/assign", suite.addAuthContext, suite.handler.AssignTask)
				tasks.POST("/reorder", suite.addAuthContext, suite.handler.ReorderTasks)
			}
			
			// Board routes
			boards := projects.Group("/boards")
			{
				boards.POST("", suite.addAuthContext, suite.handler.CreateBoard)
				boards.GET("", suite.addAuthContext, suite.handler.ListBoards)
				boards.GET("/:id", suite.addAuthContext, suite.handler.GetBoard)
				boards.PUT("/:id", suite.addAuthContext, suite.handler.UpdateBoard)
				boards.DELETE("/:id", suite.addAuthContext, suite.handler.DeleteBoard)
			}
			
			// Analytics routes
			analytics := projects.Group("/analytics")
			{
				analytics.GET("/velocity", suite.addAuthContext, suite.handler.GetVelocityChart)
			}
		}
	}
}

func (suite *AgileHandlerTestSuite) addAuthContext(c *gin.Context) {
	c.Set("tenant_id", suite.testTenantID.String())
	c.Set("user_id", suite.testUserID.String())
	c.Next()
}

// TestSprintOperations 测试Sprint操作
func (suite *AgileHandlerTestSuite) TestSprintOperations() {
	// 测试创建Sprint
	suite.Run("创建Sprint", func() {
		startDate := time.Now().AddDate(0, 0, 1)
		endDate := time.Now().AddDate(0, 0, 15)
		
		sprint := &models.Sprint{
			ID:          suite.testSprintID,
			ProjectID:   suite.testProjectID,
			Name:        "Sprint 1",
			Description: &[]string{"第一个Sprint"}[0],
			Goal:        &[]string{"完成基础功能"}[0],
			Status:      models.SprintStatusPlanned,
			StartDate:   startDate,
			EndDate:     endDate,
			Capacity:    100,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		suite.mockService.On("CreateSprint", mock.Anything, mock.MatchedBy(func(req *service.CreateSprintRequest) bool {
			return req.ProjectID == suite.testProjectID &&
				   req.Name == "Sprint 1" &&
				   req.TenantID == suite.testTenantID &&
				   req.CreatedBy == suite.testUserID
		})).Return(sprint, nil)

		body, _ := json.Marshal(map[string]interface{}{
			"name":        "Sprint 1",
			"description": "第一个Sprint",
			"goal":        "完成基础功能",
			"start_date":  startDate.Format(time.RFC3339),
			"end_date":    endDate.Format(time.RFC3339),
			"capacity":    100,
		})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/projects/%s/sprints", suite.testProjectID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusCreated, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
		assert.NotNil(suite.T(), response.Data)
	})

	// 测试获取Sprint列表
	suite.Run("获取Sprint列表", func() {
		sprints := []models.Sprint{
			{
				ID:        uuid.New(),
				ProjectID: suite.testProjectID,
				Name:      "Sprint 1",
				Status:    models.SprintStatusActive,
				StartDate: time.Now().AddDate(0, 0, -7),
				EndDate:   time.Now().AddDate(0, 0, 7),
				Capacity:  100,
			},
			{
				ID:        uuid.New(),
				ProjectID: suite.testProjectID,
				Name:      "Sprint 2",
				Status:    models.SprintStatusPlanned,
				StartDate: time.Now().AddDate(0, 0, 8),
				EndDate:   time.Now().AddDate(0, 0, 22),
				Capacity:  120,
			},
		}

		sprintListResponse := &service.SprintListResponse{
			Sprints:  sprints,
			Total:    2,
			Page:     1,
			PageSize: 20,
		}

		suite.mockService.On("ListSprints", mock.Anything, suite.testProjectID, suite.testTenantID, mock.MatchedBy(func(req *service.ListSprintsRequest) bool {
			return req.Page == 1 && req.PageSize == 20
		})).Return(sprintListResponse, nil)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/sprints?page=1&page_size=20", suite.testProjectID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
		assert.NotNil(suite.T(), response.Data)
	})

	// 测试获取Sprint详情
	suite.Run("获取Sprint详情", func() {
		sprint := &models.Sprint{
			ID:          suite.testSprintID,
			ProjectID:   suite.testProjectID,
			Name:        "Sprint 1",
			Description: &[]string{"第一个Sprint"}[0],
			Status:      models.SprintStatusActive,
			StartDate:   time.Now().AddDate(0, 0, -7),
			EndDate:     time.Now().AddDate(0, 0, 7),
			Capacity:    100,
		}

		suite.mockService.On("GetSprint", mock.Anything, suite.testSprintID, suite.testTenantID).
			Return(sprint, nil)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/sprints/%s", suite.testProjectID.String(), suite.testSprintID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
		assert.NotNil(suite.T(), response.Data)
	})

	// 测试启动Sprint
	suite.Run("启动Sprint", func() {
		suite.mockService.On("StartSprint", mock.Anything, suite.testSprintID, suite.testTenantID).
			Return(nil)

		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/projects/%s/sprints/%s/start", suite.testProjectID.String(), suite.testSprintID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
	})

	// 测试完成Sprint
	suite.Run("完成Sprint", func() {
		suite.mockService.On("CompleteSprint", mock.Anything, suite.testSprintID, suite.testTenantID).
			Return(nil)

		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/projects/%s/sprints/%s/complete", suite.testProjectID.String(), suite.testSprintID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
	})

	// 测试删除Sprint
	suite.Run("删除Sprint", func() {
		suite.mockService.On("DeleteSprint", mock.Anything, suite.testSprintID, suite.testTenantID).
			Return(nil)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/projects/%s/sprints/%s", suite.testProjectID.String(), suite.testSprintID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
	})
}

// TestEpicOperations 测试Epic操作
func (suite *AgileHandlerTestSuite) TestEpicOperations() {
	// 测试创建Epic
	suite.Run("创建Epic", func() {
		epic := &models.Epic{
			ID:              suite.testEpicID,
			ProjectID:       suite.testProjectID,
			Name:            "用户管理模块",
			Description:     &[]string{"完整的用户管理功能"}[0],
			Status:          models.EpicStatusOpen,
			Color:           &[]string{"#2196F3"}[0],
			Goal:            &[]string{"提供完整的用户管理解决方案"}[0],
			SuccessCriteria: &[]string{"用户可以注册、登录、管理个人信息"}[0],
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		suite.mockService.On("CreateEpic", mock.Anything, mock.MatchedBy(func(req *service.CreateEpicRequest) bool {
			return req.ProjectID == suite.testProjectID &&
				   req.Name == "用户管理模块" &&
				   req.TenantID == suite.testTenantID &&
				   req.CreatedBy == suite.testUserID
		})).Return(epic, nil)

		body, _ := json.Marshal(map[string]interface{}{
			"name":             "用户管理模块",
			"description":      "完整的用户管理功能",
			"color":            "#2196F3",
			"goal":             "提供完整的用户管理解决方案",
			"success_criteria": "用户可以注册、登录、管理个人信息",
		})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/projects/%s/epics", suite.testProjectID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusCreated, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
		assert.NotNil(suite.T(), response.Data)
	})

	// 测试获取Epic列表
	suite.Run("获取Epic列表", func() {
		epics := []models.Epic{
			{
				ID:        uuid.New(),
				ProjectID: suite.testProjectID,
				Name:      "用户管理模块",
				Status:    models.EpicStatusInProgress,
				Color:     &[]string{"#2196F3"}[0],
			},
			{
				ID:        uuid.New(),
				ProjectID: suite.testProjectID,
				Name:      "支付模块",
				Status:    models.EpicStatusOpen,
				Color:     &[]string{"#4CAF50"}[0],
			},
		}

		epicListResponse := &service.EpicListResponse{
			Epics:    epics,
			Total:    2,
			Page:     1,
			PageSize: 20,
		}

		suite.mockService.On("ListEpics", mock.Anything, suite.testProjectID, suite.testTenantID, mock.MatchedBy(func(req *service.ListEpicsRequest) bool {
			return req.Page == 1 && req.PageSize == 20
		})).Return(epicListResponse, nil)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/epics?page=1&page_size=20", suite.testProjectID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
		assert.NotNil(suite.T(), response.Data)
	})

	// 测试获取Epic详情
	suite.Run("获取Epic详情", func() {
		epic := &models.Epic{
			ID:              suite.testEpicID,
			ProjectID:       suite.testProjectID,
			Name:            "用户管理模块",
			Description:     &[]string{"完整的用户管理功能"}[0],
			Status:          models.EpicStatusInProgress,
			Color:           &[]string{"#2196F3"}[0],
			Goal:            &[]string{"提供完整的用户管理解决方案"}[0],
			SuccessCriteria: &[]string{"用户可以注册、登录、管理个人信息"}[0],
		}

		suite.mockService.On("GetEpic", mock.Anything, suite.testEpicID, suite.testTenantID).
			Return(epic, nil)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/epics/%s", suite.testProjectID.String(), suite.testEpicID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
		assert.NotNil(suite.T(), response.Data)
	})

	// 测试删除Epic
	suite.Run("删除Epic", func() {
		suite.mockService.On("DeleteEpic", mock.Anything, suite.testEpicID, suite.testTenantID).
			Return(nil)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/projects/%s/epics/%s", suite.testProjectID.String(), suite.testEpicID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
	})
}

// TestTaskOperations 测试任务操作
func (suite *AgileHandlerTestSuite) TestTaskOperations() {
	// 测试创建任务
	suite.Run("创建任务", func() {
		acceptanceCriteria := []models.AcceptanceCriteria{
			{
				ID:          "ac-1",
				Description: "用户可以成功登录",
				Completed:   false,
			},
			{
				ID:          "ac-2",
				Description: "显示欢迎消息",
				Completed:   false,
			},
		}

		task := &models.AgileTask{
			ID:                 suite.testTaskID,
			ProjectID:          suite.testProjectID,
			SprintID:           &suite.testSprintID,
			EpicID:             &suite.testEpicID,
			TaskNumber:         1001,
			Title:              "实现用户登录功能",
			Description:        &[]string{"开发用户登录接口和前端页面"}[0],
			Type:               models.TaskTypeStory,
			Status:             models.TaskStatusTodo,
			Priority:           models.PriorityHigh,
			StoryPoints:        &[]int{8}[0],
			OriginalEstimate:   &[]float64{16.0}[0],
			RemainingTime:      &[]float64{16.0}[0],
			LoggedTime:         0,
			ReporterID:         suite.testUserID,
			Labels:             []string{"frontend", "backend", "authentication"},
			Components:         []string{"auth-service", "web-ui"},
			Rank:               "0|i00007:",
			AcceptanceCriteria: acceptanceCriteria,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}

		suite.mockService.On("CreateTask", mock.Anything, mock.MatchedBy(func(req *service.CreateTaskRequest) bool {
			return req.ProjectID == suite.testProjectID &&
				   req.Title == "实现用户登录功能" &&
				   req.TenantID == suite.testTenantID &&
				   req.ReporterID == suite.testUserID
		})).Return(task, nil)

		body, _ := json.Marshal(map[string]interface{}{
			"sprint_id":    suite.testSprintID.String(),
			"epic_id":      suite.testEpicID.String(),
			"title":        "实现用户登录功能",
			"description":  "开发用户登录接口和前端页面",
			"type":         "story",
			"priority":     "high",
			"story_points": 8,
			"original_estimate": 16.0,
			"labels":       []string{"frontend", "backend", "authentication"},
			"components":   []string{"auth-service", "web-ui"},
			"acceptance_criteria": []map[string]interface{}{
				{
					"id":          "ac-1",
					"description": "用户可以成功登录",
					"completed":   false,
				},
				{
					"id":          "ac-2",
					"description": "显示欢迎消息",
					"completed":   false,
				},
			},
		})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/projects/%s/tasks", suite.testProjectID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusCreated, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
		assert.NotNil(suite.T(), response.Data)
	})

	// 测试获取任务列表
	suite.Run("获取任务列表", func() {
		tasks := []models.AgileTask{
			{
				ID:          uuid.New(),
				ProjectID:   suite.testProjectID,
				TaskNumber:  1001,
				Title:       "实现用户登录功能",
				Type:        models.TaskTypeStory,
				Status:      models.TaskStatusInProgress,
				Priority:    models.PriorityHigh,
				StoryPoints: &[]int{8}[0],
				ReporterID:  suite.testUserID,
			},
			{
				ID:          uuid.New(),
				ProjectID:   suite.testProjectID,
				TaskNumber:  1002,
				Title:       "修复登录页面样式问题",
				Type:        models.TaskTypeBug,
				Status:      models.TaskStatusTodo,
				Priority:    models.PriorityMedium,
				StoryPoints: &[]int{3}[0],
				ReporterID:  suite.testUserID,
			},
		}

		taskListResponse := &service.TaskListResponse{
			Tasks:    tasks,
			Total:    2,
			Page:     1,
			PageSize: 20,
		}

		suite.mockService.On("ListTasks", mock.Anything, suite.testProjectID, suite.testTenantID, mock.MatchedBy(func(req *service.ListTasksRequest) bool {
			return req.Page == 1 && req.PageSize == 20
		})).Return(taskListResponse, nil)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/tasks?page=1&page_size=20", suite.testProjectID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
		assert.NotNil(suite.T(), response.Data)
	})

	// 测试更新任务状态
	suite.Run("更新任务状态", func() {
		updatedTask := &models.AgileTask{
			ID:            suite.testTaskID,
			ProjectID:     suite.testProjectID,
			Title:         "实现用户登录功能",
			Status:        models.TaskStatusInProgress,
			RemainingTime: &[]float64{12.0}[0],
		}

		suite.mockService.On("UpdateTaskStatus", mock.Anything, mock.MatchedBy(func(req *service.UpdateTaskStatusRequest) bool {
			return req.TaskID == suite.testTaskID &&
				   req.Status == models.TaskStatusInProgress &&
				   req.TenantID == suite.testTenantID
		})).Return(updatedTask, nil)

		body, _ := json.Marshal(map[string]interface{}{
			"status":         "in_progress",
			"remaining_time": 12.0,
		})
		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/projects/%s/tasks/%s/status", suite.testProjectID.String(), suite.testTaskID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
		assert.NotNil(suite.T(), response.Data)
	})

	// 测试分配任务
	suite.Run("分配任务", func() {
		assigneeID := uuid.New()
		assignedTask := &models.AgileTask{
			ID:         suite.testTaskID,
			ProjectID:  suite.testProjectID,
			Title:      "实现用户登录功能",
			AssigneeID: &assigneeID,
		}

		suite.mockService.On("AssignTask", mock.Anything, mock.MatchedBy(func(req *service.AssignTaskRequest) bool {
			return req.TaskID == suite.testTaskID &&
				   req.AssigneeID != nil && *req.AssigneeID == assigneeID &&
				   req.TenantID == suite.testTenantID
		})).Return(assignedTask, nil)

		body, _ := json.Marshal(map[string]interface{}{
			"assignee_id": assigneeID.String(),
		})
		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/projects/%s/tasks/%s/assign", suite.testProjectID.String(), suite.testTaskID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
		assert.NotNil(suite.T(), response.Data)
	})

	// 测试任务重新排序
	suite.Run("任务重新排序", func() {
		suite.mockService.On("ReorderTasks", mock.Anything, mock.MatchedBy(func(req *service.ReorderTasksRequest) bool {
			return req.ProjectID == suite.testProjectID &&
				   req.TenantID == suite.testTenantID &&
				   len(req.TaskOrders) == 2
		})).Return(nil)

		body, _ := json.Marshal(map[string]interface{}{
			"task_orders": []map[string]interface{}{
				{
					"task_id": uuid.New().String(),
					"rank":    "0|i00001:",
				},
				{
					"task_id": uuid.New().String(),
					"rank":    "0|i00002:",
				},
			},
		})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/projects/%s/tasks/reorder", suite.testProjectID.String()), bytes.NewBuffer(body))
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

// TestBoardOperations 测试看板操作
func (suite *AgileHandlerTestSuite) TestBoardOperations() {
	// 测试创建看板
	suite.Run("创建看板", func() {
		board := &models.Board{
			ID:          suite.testBoardID,
			ProjectID:   suite.testProjectID,
			Name:        "开发看板",
			Description: &[]string{"主要开发任务看板"}[0],
			Type:        "kanban",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		suite.mockService.On("CreateBoard", mock.Anything, mock.MatchedBy(func(req *service.CreateBoardRequest) bool {
			return req.ProjectID == suite.testProjectID &&
				   req.Name == "开发看板" &&
				   req.TenantID == suite.testTenantID &&
				   req.CreatedBy == suite.testUserID
		})).Return(board, nil)

		body, _ := json.Marshal(map[string]interface{}{
			"name":        "开发看板",
			"description": "主要开发任务看板",
			"type":        "kanban",
		})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/projects/%s/boards", suite.testProjectID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusCreated, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
		assert.NotNil(suite.T(), response.Data)
	})

	// 测试获取看板列表
	suite.Run("获取看板列表", func() {
		boards := []models.Board{
			{
				ID:          uuid.New(),
				ProjectID:   suite.testProjectID,
				Name:        "开发看板",
				Description: &[]string{"主要开发任务看板"}[0],
				Type:        "kanban",
			},
			{
				ID:          uuid.New(),
				ProjectID:   suite.testProjectID,
				Name:        "Scrum看板",
				Description: &[]string{"敏捷开发看板"}[0],
				Type:        "scrum",
			},
		}

		boardListResponse := &service.BoardListResponse{
			Boards:   boards,
			Total:    2,
			Page:     1,
			PageSize: 20,
		}

		suite.mockService.On("ListBoards", mock.Anything, suite.testProjectID, suite.testTenantID, mock.MatchedBy(func(req *service.ListBoardsRequest) bool {
			return req.Page == 1 && req.PageSize == 20
		})).Return(boardListResponse, nil)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/boards?page=1&page_size=20", suite.testProjectID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
		assert.NotNil(suite.T(), response.Data)
	})

	// 测试删除看板
	suite.Run("删除看板", func() {
		suite.mockService.On("DeleteBoard", mock.Anything, suite.testBoardID, suite.testTenantID).
			Return(nil)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/projects/%s/boards/%s", suite.testProjectID.String(), suite.testBoardID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
	})
}

// TestAnalyticsOperations 测试分析操作
func (suite *AgileHandlerTestSuite) TestAnalyticsOperations() {
	// 测试获取Sprint燃尽图
	suite.Run("获取Sprint燃尽图", func() {
		burndownChart := &service.BurndownChart{
			SprintID:    suite.testSprintID,
			StartDate:   time.Now().AddDate(0, 0, -14),
			EndDate:     time.Now(),
			TotalPoints: 100,
			DataPoints: []service.BurndownDataPoint{
				{
					Date:           time.Now().AddDate(0, 0, -14),
					RemainingWork:  100,
					IdealRemaining: 100,
				},
				{
					Date:           time.Now().AddDate(0, 0, -7),
					RemainingWork:  60,
					IdealRemaining: 50,
				},
				{
					Date:           time.Now(),
					RemainingWork:  10,
					IdealRemaining: 0,
				},
			},
		}

		suite.mockService.On("GetSprintBurndown", mock.Anything, suite.testSprintID, suite.testTenantID).
			Return(burndownChart, nil)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/sprints/%s/burndown", suite.testProjectID.String(), suite.testSprintID.String()), nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response response.Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.True(suite.T(), response.Success)
		assert.NotNil(suite.T(), response.Data)
	})

	// 测试获取速度图表
	suite.Run("获取速度图表", func() {
		velocityChart := &service.VelocityChart{
			ProjectID: suite.testProjectID,
			Period:    "last_6_sprints",
			DataPoints: []service.VelocityDataPoint{
				{
					SprintName:      "Sprint 1",
					PlannedVelocity: 50,
					ActualVelocity:  45,
					CompletedStories: 8,
				},
				{
					SprintName:      "Sprint 2",
					PlannedVelocity: 55,
					ActualVelocity:  60,
					CompletedStories: 10,
				},
				{
					SprintName:      "Sprint 3",
					PlannedVelocity: 60,
					ActualVelocity:  58,
					CompletedStories: 9,
				},
			},
			AverageVelocity: 54.3,
		}

		suite.mockService.On("GetVelocityChart", mock.Anything, suite.testProjectID, suite.testTenantID, mock.MatchedBy(func(req *service.VelocityChartRequest) bool {
			return req.Period == "last_6_sprints"
		})).Return(velocityChart, nil)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/analytics/velocity?period=last_6_sprints", suite.testProjectID.String()), nil)
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
func (suite *AgileHandlerTestSuite) TestErrorHandling() {
	testCases := []struct {
		name             string
		method           string
		url              string
		setupMocks       func()
		expectedStatus   int
		expectedContains string
	}{
		{
			name:   "Sprint不存在",
			method: http.MethodGet,
			url:    fmt.Sprintf("/api/v1/projects/%s/sprints/%s", suite.testProjectID.String(), uuid.New().String()),
			setupMocks: func() {
				suite.mockService.On("GetSprint", mock.Anything, mock.AnythingOfType("uuid.UUID"), suite.testTenantID).
					Return(nil, fmt.Errorf("sprint not found"))
			},
			expectedStatus:   http.StatusNotFound,
			expectedContains: "sprint not found",
		},
		{
			name:             "无效的Sprint ID",
			method:           http.MethodGet,
			url:              fmt.Sprintf("/api/v1/projects/%s/sprints/invalid-uuid", suite.testProjectID.String()),
			setupMocks:       func() {},
			expectedStatus:   http.StatusBadRequest,
			expectedContains: "无效的Sprint ID",
		},
		{
			name:   "Epic不存在",
			method: http.MethodGet,
			url:    fmt.Sprintf("/api/v1/projects/%s/epics/%s", suite.testProjectID.String(), uuid.New().String()),
			setupMocks: func() {
				suite.mockService.On("GetEpic", mock.Anything, mock.AnythingOfType("uuid.UUID"), suite.testTenantID).
					Return(nil, fmt.Errorf("epic not found"))
			},
			expectedStatus:   http.StatusNotFound,
			expectedContains: "epic not found",
		},
		{
			name:   "任务不存在",
			method: http.MethodGet,
			url:    fmt.Sprintf("/api/v1/projects/%s/tasks/%s", suite.testProjectID.String(), uuid.New().String()),
			setupMocks: func() {
				suite.mockService.On("GetTask", mock.Anything, mock.AnythingOfType("uuid.UUID"), suite.testTenantID).
					Return(nil, fmt.Errorf("task not found"))
			},
			expectedStatus:   http.StatusNotFound,
			expectedContains: "task not found",
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
func TestAgileHandlerSuite(t *testing.T) {
	suite.Run(t, new(AgileHandlerTestSuite))
}