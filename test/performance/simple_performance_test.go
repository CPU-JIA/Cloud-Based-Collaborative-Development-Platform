package performance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/models"
	"github.com/cloud-platform/collaborative-dev/shared/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

// SimplePerformanceMetrics 简单的性能指标
type SimplePerformanceMetrics struct {
	mu                    sync.RWMutex
	RequestCount          int64                     `json:"request_count"`
	TotalResponseTime     time.Duration             `json:"total_response_time"`
	AverageResponseTime   time.Duration             `json:"average_response_time"`
	MinResponseTime       time.Duration             `json:"min_response_time"`
	MaxResponseTime       time.Duration             `json:"max_response_time"`
	ErrorCount            int64                     `json:"error_count"`
	SuccessRate           float64                   `json:"success_rate"`
	ResponseTimes         []time.Duration           `json:"-"`
	ErrorDistribution     map[int]int64             `json:"error_distribution"`
	EndpointMetrics       map[string]*SimpleEndpoint `json:"endpoint_metrics"`
}

// SimpleEndpoint 简单的端点性能指标
type SimpleEndpoint struct {
	RequestCount      int64         `json:"request_count"`
	TotalResponseTime time.Duration `json:"total_response_time"`
	AverageTime       time.Duration `json:"average_time"`
	MinTime           time.Duration `json:"min_time"`
	MaxTime           time.Duration `json:"max_time"`
	ErrorCount        int64         `json:"error_count"`
	SuccessRate       float64       `json:"success_rate"`
}

// InMemoryProjectStore 简单的内存项目存储
type InMemoryProjectStore struct {
	mu       sync.RWMutex
	projects map[uuid.UUID]*models.Project
	counter  int
}

func NewInMemoryProjectStore() *InMemoryProjectStore {
	return &InMemoryProjectStore{
		projects: make(map[uuid.UUID]*models.Project),
		counter:  0,
	}
}

func (s *InMemoryProjectStore) Create(project *models.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if project.ID == uuid.Nil {
		project.ID = uuid.New()
	}
	
	// 检查Key是否已存在
	for _, p := range s.projects {
		if p.Key == project.Key {
			return fmt.Errorf("project key already exists")
		}
	}
	
	project.CreatedAt = time.Now()
	project.UpdatedAt = time.Now()
	s.projects[project.ID] = project
	s.counter++
	
	return nil
}

func (s *InMemoryProjectStore) GetByID(id uuid.UUID) (*models.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	project, exists := s.projects[id]
	if !exists {
		return nil, fmt.Errorf("project not found")
	}
	
	return project, nil
}

func (s *InMemoryProjectStore) List(page, pageSize int) ([]*models.Project, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	total := int64(len(s.projects))
	projects := make([]*models.Project, 0, pageSize)
	
	counter := 0
	offset := (page - 1) * pageSize
	
	for _, project := range s.projects {
		if counter >= offset && len(projects) < pageSize {
			projects = append(projects, project)
		}
		counter++
		if len(projects) >= pageSize {
			break
		}
	}
	
	return projects, total, nil
}

func (s *InMemoryProjectStore) Update(id uuid.UUID, updates map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	project, exists := s.projects[id]
	if !exists {
		return fmt.Errorf("project not found")
	}
	
	// 简单的字段更新
	if name, ok := updates["name"].(string); ok {
		project.Name = name
	}
	if desc, ok := updates["description"].(*string); ok {
		project.Description = desc
	}
	if status, ok := updates["status"].(string); ok {
		project.Status = status
	}
	
	project.UpdatedAt = time.Now()
	
	return nil
}

func (s *InMemoryProjectStore) Delete(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.projects[id]; !exists {
		return fmt.Errorf("project not found")
	}
	
	delete(s.projects, id)
	
	return nil
}

// SimpleProjectServicePerformanceTestSuite 简单的项目服务性能测试套件
type SimpleProjectServicePerformanceTestSuite struct {
	suite.Suite
	store              *InMemoryProjectStore
	logger             *zap.Logger
	router             *gin.Engine
	server             *httptest.Server
	testUserID         uuid.UUID
	testTenantID       uuid.UUID
	authToken          string
	performanceMetrics *SimplePerformanceMetrics
}

// SetupSuite 设置测试套件
func (suite *SimpleProjectServicePerformanceTestSuite) SetupSuite() {
	// 初始化日志
	logger, _ := zap.NewDevelopment()
	suite.logger = logger

	// 初始化内存存储
	suite.store = NewInMemoryProjectStore()

	// 初始化性能指标
	suite.performanceMetrics = &SimplePerformanceMetrics{
		MinResponseTime:   time.Hour, // 初始化为一个大值
		ErrorDistribution: make(map[int]int64),
		EndpointMetrics:   make(map[string]*SimpleEndpoint),
	}

	// 初始化测试数据
	suite.testUserID = uuid.New()
	suite.testTenantID = uuid.New()
	suite.authToken = "simple-test-auth-token"

	// 设置路由
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()
	suite.setupSimpleRoutes()

	// 创建测试服务器
	suite.server = httptest.NewServer(suite.router)

	suite.logger.Info("简单性能测试套件初始化完成",
		zap.String("server_url", suite.server.URL))
}

// TearDownSuite 清理测试套件
func (suite *SimpleProjectServicePerformanceTestSuite) TearDownSuite() {
	if suite.server != nil {
		suite.server.Close()
	}

	// 生成性能测试报告
	suite.generateSimplePerformanceReport()

	suite.logger.Info("简单性能测试套件清理完成")
}

// setupSimpleRoutes 设置简单路由
func (suite *SimpleProjectServicePerformanceTestSuite) setupSimpleRoutes() {
	// 设置基本的REST API路由用于性能测试
	v1 := suite.router.Group("/api/v1")
	
	// 项目路由
	projects := v1.Group("/projects")
	{
		projects.POST("", suite.simpleCreateProject)
		projects.GET("", suite.simpleListProjects)
		projects.GET("/:id", suite.simpleGetProject)
		projects.PUT("/:id", suite.simpleUpdateProject)
		projects.DELETE("/:id", suite.simpleDeleteProject)
	}
	
	// 健康检查路由
	v1.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, response.Response{
			Code:    http.StatusOK,
			Message: "Service is healthy",
		})
	})
}

// Simple handlers for performance testing
func (suite *SimpleProjectServicePerformanceTestSuite) simpleCreateProject(c *gin.Context) {
	var req models.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request",
		})
		return
	}

	// 创建项目
	project := &models.Project{
		ID:          uuid.New(),
		TenantID:    suite.testTenantID,
		Key:         req.Key,
		Name:        req.Name,
		Description: req.Description,
		Status:      "active",
	}

	if err := suite.store.Create(project); err != nil {
		c.JSON(http.StatusInternalServerError, response.Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to create project",
		})
		return
	}

	c.JSON(http.StatusCreated, response.Response{
		Code:    http.StatusCreated,
		Message: "Project created successfully",
		Data:    project,
	})
}

func (suite *SimpleProjectServicePerformanceTestSuite) simpleListProjects(c *gin.Context) {
	// 分页参数
	page := 1
	pageSize := 20
	
	if c.Query("page") != "" {
		fmt.Sscanf(c.Query("page"), "%d", &page)
	}
	if c.Query("page_size") != "" {
		fmt.Sscanf(c.Query("page_size"), "%d", &pageSize)
	}
	
	projects, total, err := suite.store.List(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to list projects",
		})
		return
	}

	// 转换为响应格式
	projectList := make([]models.Project, len(projects))
	for i, p := range projects {
		projectList[i] = *p
	}

	c.JSON(http.StatusOK, response.Response{
		Code:    http.StatusOK,
		Message: "Projects listed successfully",
		Data: models.ProjectListResponse{
			Projects: projectList,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	})
}

func (suite *SimpleProjectServicePerformanceTestSuite) simpleGetProject(c *gin.Context) {
	projectIDStr := c.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid project ID",
		})
		return
	}
	
	project, err := suite.store.GetByID(projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, response.Response{
			Code:    http.StatusNotFound,
			Message: "Project not found",
		})
		return
	}

	c.JSON(http.StatusOK, response.Response{
		Code:    http.StatusOK,
		Message: "Project retrieved successfully",
		Data:    project,
	})
}

func (suite *SimpleProjectServicePerformanceTestSuite) simpleUpdateProject(c *gin.Context) {
	projectIDStr := c.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid project ID",
		})
		return
	}
	
	var req models.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request",
		})
		return
	}

	// 准备更新数据
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = req.Description
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if err := suite.store.Update(projectID, updates); err != nil {
		c.JSON(http.StatusNotFound, response.Response{
			Code:    http.StatusNotFound,
			Message: "Project not found",
		})
		return
	}

	// 获取更新后的项目
	project, _ := suite.store.GetByID(projectID)

	c.JSON(http.StatusOK, response.Response{
		Code:    http.StatusOK,
		Message: "Project updated successfully",
		Data:    project,
	})
}

func (suite *SimpleProjectServicePerformanceTestSuite) simpleDeleteProject(c *gin.Context) {
	projectIDStr := c.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid project ID",
		})
		return
	}
	
	if err := suite.store.Delete(projectID); err != nil {
		c.JSON(http.StatusNotFound, response.Response{
			Code:    http.StatusNotFound,
			Message: "Project not found",
		})
		return
	}

	c.JSON(http.StatusOK, response.Response{
		Code:    http.StatusOK,
		Message: "Project deleted successfully",
	})
}

// makeSimpleRequest 发送HTTP请求并记录性能指标
func (suite *SimpleProjectServicePerformanceTestSuite) makeSimpleRequest(method, endpoint string, body interface{}, authToken string) (*http.Response, []byte, time.Duration) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(bodyBytes)
	}

	url := suite.server.URL + endpoint
	req, _ := http.NewRequest(method, url, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}

	startTime := time.Now()
	resp, err := http.DefaultClient.Do(req)
	responseTime := time.Since(startTime)

	var respBody []byte
	if err == nil && resp != nil {
		respBody, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
	}

	// 记录性能指标
	suite.recordSimpleMetrics(endpoint, resp.StatusCode, responseTime, err)

	return resp, respBody, responseTime
}

// recordSimpleMetrics 记录简单的性能指标
func (suite *SimpleProjectServicePerformanceTestSuite) recordSimpleMetrics(endpoint string, statusCode int, responseTime time.Duration, err error) {
	suite.performanceMetrics.mu.Lock()
	defer suite.performanceMetrics.mu.Unlock()

	// 全局指标
	suite.performanceMetrics.RequestCount++
	suite.performanceMetrics.TotalResponseTime += responseTime
	suite.performanceMetrics.ResponseTimes = append(suite.performanceMetrics.ResponseTimes, responseTime)

	if responseTime < suite.performanceMetrics.MinResponseTime {
		suite.performanceMetrics.MinResponseTime = responseTime
	}
	if responseTime > suite.performanceMetrics.MaxResponseTime {
		suite.performanceMetrics.MaxResponseTime = responseTime
	}

	if err != nil || statusCode >= 400 {
		suite.performanceMetrics.ErrorCount++
		suite.performanceMetrics.ErrorDistribution[statusCode]++
	}

	// 端点指标
	if suite.performanceMetrics.EndpointMetrics[endpoint] == nil {
		suite.performanceMetrics.EndpointMetrics[endpoint] = &SimpleEndpoint{
			MinTime: time.Hour,
		}
	}

	endpointMetric := suite.performanceMetrics.EndpointMetrics[endpoint]
	endpointMetric.RequestCount++
	endpointMetric.TotalResponseTime += responseTime

	if responseTime < endpointMetric.MinTime {
		endpointMetric.MinTime = responseTime
	}
	if responseTime > endpointMetric.MaxTime {
		endpointMetric.MaxTime = responseTime
	}

	if err != nil || statusCode >= 400 {
		endpointMetric.ErrorCount++
	}
}

// TestSimpleProjectCRUDPerformance 测试简单项目CRUD操作性能
func (suite *SimpleProjectServicePerformanceTestSuite) TestSimpleProjectCRUDPerformance() {
	suite.Run("简单CRUD性能测试", func() {
		const numOperations = 30
		projects := make([]uuid.UUID, 0, numOperations)

		suite.logger.Info("开始简单CRUD性能测试", zap.Int("operations", numOperations))

		startTime := time.Now()

		// 创建项目
		for i := 0; i < numOperations; i++ {
			createReq := models.CreateProjectRequest{
				Key:         fmt.Sprintf("simple-test-%d", i),
				Name:        fmt.Sprintf("简单性能测试项目%d", i),
				Description: simpleStringPtr(fmt.Sprintf("简单性能测试项目描述%d", i)),
			}

			resp, body, responseTime := suite.makeSimpleRequest("POST", "/api/v1/projects", createReq, suite.authToken)
			
			suite.logger.Debug("项目创建请求",
				zap.Int("index", i),
				zap.Int("status_code", resp.StatusCode),
				zap.Duration("response_time", responseTime))
			
			assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

			var createResponse response.Response
			err := json.Unmarshal(body, &createResponse)
			assert.NoError(suite.T(), err)

			if createResponse.Data != nil {
				projectData, _ := json.Marshal(createResponse.Data)
				var project models.Project
				err = json.Unmarshal(projectData, &project)
				if err == nil {
					projects = append(projects, project.ID)
				}
			}
		}

		suite.logger.Info("项目创建完成", zap.Int("created_projects", len(projects)))

		// 读取项目
		for i, projectID := range projects {
			resp, _, responseTime := suite.makeSimpleRequest("GET", fmt.Sprintf("/api/v1/projects/%s", projectID), nil, suite.authToken)
			
			suite.logger.Debug("项目读取请求",
				zap.Int("index", i),
				zap.String("project_id", projectID.String()),
				zap.Int("status_code", resp.StatusCode),
				zap.Duration("response_time", responseTime))
			
			assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		}

		// 更新项目
		for i, projectID := range projects {
			updateReq := models.UpdateProjectRequest{
				Name: simpleStringPtr(fmt.Sprintf("更新的简单性能测试项目%d", i)),
			}

			resp, _, responseTime := suite.makeSimpleRequest("PUT", fmt.Sprintf("/api/v1/projects/%s", projectID), updateReq, suite.authToken)
			
			suite.logger.Debug("项目更新请求",
				zap.Int("index", i),
				zap.String("project_id", projectID.String()),
				zap.Int("status_code", resp.StatusCode),
				zap.Duration("response_time", responseTime))
			
			assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		}

		// 删除项目
		for i, projectID := range projects {
			resp, _, responseTime := suite.makeSimpleRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s", projectID), nil, suite.authToken)
			
			suite.logger.Debug("项目删除请求",
				zap.Int("index", i),
				zap.String("project_id", projectID.String()),
				zap.Int("status_code", resp.StatusCode),
				zap.Duration("response_time", responseTime))
			
			assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		}

		totalTime := time.Since(startTime)
		totalOperations := numOperations * 4 // 每个项目4个操作
		suite.logger.Info("简单CRUD性能测试完成",
			zap.Int("operations", totalOperations),
			zap.Duration("total_time", totalTime),
			zap.Float64("ops_per_second", float64(totalOperations)/totalTime.Seconds()))
	})
}

// TestSimpleLoadTest 测试简单负载
func (suite *SimpleProjectServicePerformanceTestSuite) TestSimpleLoadTest() {
	suite.Run("简单负载测试", func() {
		const concurrentUsers = 8
		const operationsPerUser = 5
		
		var wg sync.WaitGroup
		startTime := time.Now()
		
		suite.logger.Info("开始简单负载测试",
			zap.Int("concurrent_users", concurrentUsers),
			zap.Int("operations_per_user", operationsPerUser))
		
		for user := 0; user < concurrentUsers; user++ {
			wg.Add(1)
			go func(userID int) {
				defer wg.Done()
				
				suite.logger.Debug("用户开始操作", zap.Int("user_id", userID))
				
				for op := 0; op < operationsPerUser; op++ {
					createReq := models.CreateProjectRequest{
						Key:         fmt.Sprintf("simple-load-test-%d-%d", userID, op),
						Name:        fmt.Sprintf("简单负载测试项目%d-%d", userID, op),
						Description: simpleStringPtr("简单负载测试项目"),
					}
					
					resp, _, responseTime := suite.makeSimpleRequest("POST", "/api/v1/projects", createReq, suite.authToken)
					
					suite.logger.Debug("负载测试操作",
						zap.Int("user_id", userID),
						zap.Int("operation", op),
						zap.Int("status_code", resp.StatusCode),
						zap.Duration("response_time", responseTime))
					
					assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
				}
				
				suite.logger.Debug("用户完成操作", zap.Int("user_id", userID))
			}(user)
		}
		
		wg.Wait()
		totalTime := time.Since(startTime)
		totalOperations := concurrentUsers * operationsPerUser
		
		suite.logger.Info("简单负载测试完成",
			zap.Int("concurrent_users", concurrentUsers),
			zap.Int("operations_per_user", operationsPerUser),
			zap.Int("total_operations", totalOperations),
			zap.Duration("total_time", totalTime),
			zap.Float64("ops_per_second", float64(totalOperations)/totalTime.Seconds()))
	})
}

// TestHealthCheckPerformance 测试健康检查性能
func (suite *SimpleProjectServicePerformanceTestSuite) TestHealthCheckPerformance() {
	suite.Run("健康检查性能测试", func() {
		const numRequests = 100
		
		suite.logger.Info("开始健康检查性能测试", zap.Int("requests", numRequests))
		
		startTime := time.Now()
		
		for i := 0; i < numRequests; i++ {
			resp, _, responseTime := suite.makeSimpleRequest("GET", "/api/v1/health", nil, "")
			
			suite.logger.Debug("健康检查请求",
				zap.Int("index", i),
				zap.Int("status_code", resp.StatusCode),
				zap.Duration("response_time", responseTime))
			
			assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		}
		
		totalTime := time.Since(startTime)
		
		suite.logger.Info("健康检查性能测试完成",
			zap.Int("requests", numRequests),
			zap.Duration("total_time", totalTime),
			zap.Float64("requests_per_second", float64(numRequests)/totalTime.Seconds()))
	})
}

// generateSimplePerformanceReport 生成简单的性能测试报告
func (suite *SimpleProjectServicePerformanceTestSuite) generateSimplePerformanceReport() {
	suite.performanceMetrics.mu.Lock()
	defer suite.performanceMetrics.mu.Unlock()

	if suite.performanceMetrics.RequestCount == 0 {
		suite.logger.Info("无性能数据，跳过报告生成")
		return
	}

	// 计算全局指标
	suite.performanceMetrics.AverageResponseTime = suite.performanceMetrics.TotalResponseTime / time.Duration(suite.performanceMetrics.RequestCount)
	suite.performanceMetrics.SuccessRate = float64(suite.performanceMetrics.RequestCount-suite.performanceMetrics.ErrorCount) / float64(suite.performanceMetrics.RequestCount) * 100

	// 计算端点指标
	for _, metric := range suite.performanceMetrics.EndpointMetrics {
		if metric.RequestCount > 0 {
			metric.AverageTime = metric.TotalResponseTime / time.Duration(metric.RequestCount)
			metric.SuccessRate = float64(metric.RequestCount-metric.ErrorCount) / float64(metric.RequestCount) * 100
		}
	}

	// 生成JSON报告
	reportData, err := json.MarshalIndent(suite.performanceMetrics, "", "  ")
	if err != nil {
		suite.logger.Error("生成性能报告失败", zap.Error(err))
		return
	}

	// 写入报告文件
	reportFile := "test-report/simple_project_service_performance_report.json"
	err = os.MkdirAll("test-report", 0755)
	if err != nil {
		suite.logger.Error("创建报告目录失败", zap.Error(err))
		return
	}

	err = os.WriteFile(reportFile, reportData, 0644)
	if err != nil {
		suite.logger.Error("写入性能报告失败", zap.Error(err))
	} else {
		suite.logger.Info("简单的性能测试报告已生成", zap.String("file", reportFile))
	}

	// 输出汇总信息
	suite.logger.Info("简单的性能测试汇总报告",
		zap.Int64("total_requests", suite.performanceMetrics.RequestCount),
		zap.Int64("total_errors", suite.performanceMetrics.ErrorCount),
		zap.Float64("success_rate", suite.performanceMetrics.SuccessRate),
		zap.Duration("avg_response_time", suite.performanceMetrics.AverageResponseTime),
		zap.Duration("min_response_time", suite.performanceMetrics.MinResponseTime),
		zap.Duration("max_response_time", suite.performanceMetrics.MaxResponseTime),
		zap.Int("endpoints_tested", len(suite.performanceMetrics.EndpointMetrics)))
}

// 辅助函数

func simpleStringPtr(s string) *string {
	return &s
}

// TestSimpleProjectServicePerformance 运行简单的项目服务性能测试
func TestSimpleProjectServicePerformance(t *testing.T) {
	suite.Run(t, new(SimpleProjectServicePerformanceTestSuite))
}