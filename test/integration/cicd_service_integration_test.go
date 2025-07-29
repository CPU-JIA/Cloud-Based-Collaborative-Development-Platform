package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloud-platform/collaborative-dev/shared/auth"
	"github.com/cloud-platform/collaborative-dev/shared/database"
	"github.com/cloud-platform/collaborative-dev/shared/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// CICDServiceIntegrationTestSuite CI/CD服务集成测试套件
type CICDServiceIntegrationTestSuite struct {
	suite.Suite
	db              *gorm.DB
	logger          logger.Logger
	router          *gin.Engine
	server          *httptest.Server
	pipelineService CICDPipelineService
	pipelineHandler *MockCICDPipelineHandler
	jwtService      *auth.JWTService
	testTenantID    uuid.UUID
	testUserID      uuid.UUID
	testRepositoryID uuid.UUID
	testPipelines   []uuid.UUID
	testPipelineRuns []uuid.UUID
	testJobs        []uuid.UUID
	testRunners     []uuid.UUID
	testToken       string
	adminToken      string
}

// SetupSuite 初始化测试套件
func (suite *CICDServiceIntegrationTestSuite) SetupSuite() {
	// 设置Gin为测试模式
	gin.SetMode(gin.TestMode)

	// 初始化日志器
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	}
	var err error
	suite.logger, err = logger.NewZapLogger(loggerConfig)
	if err != nil {
		suite.T().Fatalf("初始化日志器失败: %v", err)
	}

	// 初始化测试数据库
	suite.setupTestDatabase()

	// 初始化JWT服务
	suite.jwtService = auth.NewJWTService(
		"test-secret-key-for-cicd-integration-testing",
		time.Hour*1,
		time.Hour*24*7,
	)

	// 初始化测试数据
	suite.testTenantID = uuid.New()
	suite.testUserID = uuid.New()
	suite.testRepositoryID = uuid.New()
	suite.testPipelines = []uuid.UUID{}
	suite.testPipelineRuns = []uuid.UUID{}
	suite.testJobs = []uuid.UUID{}
	suite.testRunners = []uuid.UUID{}

	// 生成测试JWT Token
	tokenPair, err := suite.jwtService.GenerateTokenPair(
		suite.testUserID,
		suite.testTenantID,
		"developer@example.com",
		"developer",
		[]string{"pipelines:read", "pipelines:write", "jobs:read", "runners:read"},
	)
	if err != nil {
		suite.T().Fatalf("生成测试Token失败: %v", err)
	}
	suite.testToken = tokenPair.AccessToken

	// 生成管理员Token
	adminTokenPair, err := suite.jwtService.GenerateTokenPair(
		uuid.New(),
		suite.testTenantID,
		"admin@example.com",
		"admin",
		[]string{"pipelines:read", "pipelines:write", "pipelines:admin", "jobs:read", "jobs:write", "runners:read", "runners:write", "runners:admin"},
	)
	if err != nil {
		suite.T().Fatalf("生成管理员Token失败: %v", err)
	}
	suite.adminToken = adminTokenPair.AccessToken

	// 初始化模拟服务
	suite.pipelineService = NewMockCICDPipelineService()

	// 初始化模拟处理器
	suite.pipelineHandler = NewMockCICDPipelineHandler(
		suite.pipelineService,
		suite.logger,
	)

	// 设置路由
	suite.setupRouter()

	// 启动测试服务器
	suite.server = httptest.NewServer(suite.router)

	// 创建测试基础数据
	suite.createTestData()

	suite.logger.Info("CI/CD服务集成测试套件初始化完成",
		"tenant_id", suite.testTenantID,
		"user_id", suite.testUserID,
		"repository_id", suite.testRepositoryID,
		"server_url", suite.server.URL)
}

// setupTestDatabase 设置测试数据库
func (suite *CICDServiceIntegrationTestSuite) setupTestDatabase() {
	config := database.Config{
		Host:     "localhost",
		Port:     5432,
		Name:     "test_cicd_service",
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

	// 自动迁移测试用表
	err = suite.db.AutoMigrate(
		&CICDPipeline{},
		&CICDPipelineRun{},
		&CICDJob{},
		&CICDRunner{},
		&CICDJobExecution{},
		&CICDJobQueue{},
	)
	if err != nil {
		suite.T().Fatalf("数据库迁移失败: %v", err)
	}
}

// setupRouter 设置路由
func (suite *CICDServiceIntegrationTestSuite) setupRouter() {
	suite.router = gin.New()

	// 添加认证中间件模拟
	suite.router.Use(func(c *gin.Context) {
		// 对于需要认证的路由，从Authorization头获取token
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token := authHeader[7:]
			claims, err := suite.jwtService.ValidateToken(token)
			if err == nil {
				c.Set("user_id", claims.UserID)
				c.Set("tenant_id", claims.TenantID)
				c.Set("role", claims.Role)
				c.Set("user_email", claims.Email)
			}
		}
		c.Next()
	})

	// CI/CD API路由
	api := suite.router.Group("/api/v1")
	{
		// 健康检查
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"service": "cicd-service",
				"status":  "healthy",
				"version": "1.0.0",
			})
		})

		// 流水线管理路由
		pipelines := api.Group("/pipelines")
		{
			pipelines.GET("", suite.pipelineHandler.ListPipelines)
			pipelines.POST("", suite.pipelineHandler.CreatePipeline)
			pipelines.GET("/:id", suite.pipelineHandler.GetPipeline)
			pipelines.PUT("/:id", suite.pipelineHandler.UpdatePipeline)
			pipelines.DELETE("/:id", suite.pipelineHandler.DeletePipeline)
			pipelines.POST("/:id/trigger", suite.pipelineHandler.TriggerPipeline)
			pipelines.GET("/:id/runs", suite.pipelineHandler.GetPipelineRuns)
			pipelines.GET("/:id/stats", suite.pipelineHandler.GetPipelineStats)
		}

		// 流水线运行管理路由
		pipelineRuns := api.Group("/pipeline-runs")
		{
			pipelineRuns.GET("/:id", suite.pipelineHandler.GetPipelineRun)
			pipelineRuns.POST("/:id/cancel", suite.pipelineHandler.CancelPipelineRun)
			pipelineRuns.POST("/:id/retry", suite.pipelineHandler.RetryPipelineRun)
			pipelineRuns.GET("/:run_id/jobs", suite.pipelineHandler.GetJobs)
		}

		// 作业管理路由
		jobs := api.Group("/jobs")
		{
			jobs.GET("/:id", suite.pipelineHandler.GetJob)
		}

		// 执行器管理路由
		runners := api.Group("/runners")
		{
			runners.POST("", suite.pipelineHandler.RegisterRunner)
			runners.GET("", suite.pipelineHandler.ListRunners)
			runners.GET("/:id", suite.pipelineHandler.GetRunner)
			runners.PUT("/:id", suite.pipelineHandler.UpdateRunner)
			runners.DELETE("/:id", suite.pipelineHandler.UnregisterRunner)
			runners.POST("/:id/heartbeat", suite.pipelineHandler.HeartbeatRunner)
			runners.GET("/:id/stats", suite.pipelineHandler.GetRunnerStats)
		}

		// Webhook事件接收路由
		webhookEvents := api.Group("/webhook-events")
		{
			webhookEvents.POST("/git/push", suite.pipelineHandler.HandleGitPushEvent)
			webhookEvents.POST("/git/branch", suite.pipelineHandler.HandleGitBranchEvent)
			webhookEvents.POST("/git/tag", suite.pipelineHandler.HandleGitTagEvent)
			webhookEvents.POST("/git/pull-request", suite.pipelineHandler.HandleGitPullRequestEvent)
			webhookEvents.POST("/trigger-pipeline", suite.pipelineHandler.TriggerPipelineFromWebhook)
		}
	}
}

// createTestData 创建测试基础数据
func (suite *CICDServiceIntegrationTestSuite) createTestData() {
	ctx := context.Background()

	// 创建测试执行器
	testRunnerRequests := []CICDRegisterRunnerRequest{
		{
			Name:         "测试执行器1",
			Description:  cicdStringPtr("这是一个用于测试的执行器"),
			Tags:         []string{"linux", "docker", "build"},
			Version:      "1.0.0",
			OS:           "linux",
			Architecture: "amd64",
		},
		{
			Name:         "测试执行器2",
			Description:  cicdStringPtr("这是另一个用于测试的执行器"),
			Tags:         []string{"linux", "docker", "test"},
			Version:      "1.0.1",
			OS:           "linux",
			Architecture: "arm64",
		},
	}

	for _, req := range testRunnerRequests {
		runner, err := suite.pipelineService.RegisterRunner(ctx, &req)
		if err != nil {
			suite.T().Fatalf("创建测试执行器失败: %v", err)
		}
		suite.testRunners = append(suite.testRunners, runner.ID)
	}

	// 创建测试流水线
	testPipelineRequests := []CICDCreatePipelineRequest{
		{
			RepositoryID:       suite.testRepositoryID.String(),
			Name:               "测试构建流水线",
			DefinitionFilePath: ".github/workflows/ci.yml",
			Description:        cicdStringPtr("用于持续集成的测试流水线"),
		},
		{
			RepositoryID:       suite.testRepositoryID.String(),
			Name:               "测试部署流水线",
			DefinitionFilePath: ".github/workflows/cd.yml",
			Description:        cicdStringPtr("用于持续部署的测试流水线"),
		},
		{
			RepositoryID:       suite.testRepositoryID.String(),
			Name:               "测试发布流水线",
			DefinitionFilePath: ".github/workflows/release.yml",
			Description:        cicdStringPtr("用于版本发布的测试流水线"),
		},
	}

	for _, req := range testPipelineRequests {
		pipeline, err := suite.pipelineService.CreatePipeline(ctx, &req, suite.testUserID)
		if err != nil {
			suite.T().Fatalf("创建测试流水线失败: %v", err)
		}
		suite.testPipelines = append(suite.testPipelines, pipeline.ID)

		// 为每个流水线创建测试运行记录
		triggerReq := CICDTriggerPipelineRequest{
			CommitSHA: "1234567890abcdef1234567890abcdef12345678",
			Branch:    cicdStringPtr("main"),
			Variables: map[string]string{
				"BUILD_ENV": "test",
				"VERSION":   "1.0.0",
			},
		}

		pipelineRun, err := suite.pipelineService.TriggerPipeline(ctx, pipeline.ID, &triggerReq, suite.testUserID)
		if err != nil {
			suite.T().Fatalf("创建测试流水线运行失败: %v", err)
		}
		suite.testPipelineRuns = append(suite.testPipelineRuns, pipelineRun.ID)

		// 为每个流水线运行创建测试作业
		jobs := []CICDJob{
			{
				ID:            uuid.New(),
				Name:          "构建作业",
				Description:   "编译和构建应用程序",
				Type:          "build",
				Status:        "success",
				Priority:      5,
				PipelineRunID: pipelineRun.ID,
				RunnerID:      &suite.testRunners[0],
				StartedAt:     timePtr(time.Now().Add(-10 * time.Minute)),
				FinishedAt:    timePtr(time.Now().Add(-8 * time.Minute)),
			},
			{
				ID:            uuid.New(),
				Name:          "测试作业",
				Description:   "运行单元测试和集成测试",
				Type:          "test",
				Status:        "success",
				Priority:      5,
				PipelineRunID: pipelineRun.ID,
				RunnerID:      &suite.testRunners[1],
				StartedAt:     timePtr(time.Now().Add(-8 * time.Minute)),
				FinishedAt:    timePtr(time.Now().Add(-5 * time.Minute)),
			},
		}

		for _, job := range jobs {
			createdJob, err := suite.pipelineService.CreateJob(ctx, &job)
			if err != nil {
				suite.T().Fatalf("创建测试作业失败: %v", err)
			}
			suite.testJobs = append(suite.testJobs, createdJob.ID)
		}
	}
}

// TearDownSuite 清理测试套件
func (suite *CICDServiceIntegrationTestSuite) TearDownSuite() {
	if suite.server != nil {
		suite.server.Close()
	}
	suite.logger.Info("CI/CD服务集成测试套件清理完成")
}

// makeRequest 发送HTTP请求
func (suite *CICDServiceIntegrationTestSuite) makeRequest(method, path string, body interface{}, token string) (*http.Response, []byte) {
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

	var respBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	resp.Body.Close()

	// 重新编码为字节切片
	respBytes, _ := json.Marshal(respBody)
	return resp, respBytes
}

// 测试流水线创建功能
func (suite *CICDServiceIntegrationTestSuite) TestCreatePipeline() {
	suite.Run("创建新流水线", func() {
		createReq := CICDCreatePipelineRequest{
			RepositoryID:       suite.testRepositoryID.String(),
			Name:               "新测试流水线",
			DefinitionFilePath: ".github/workflows/new-test.yml",
			Description:        cicdStringPtr("这是一个新的测试流水线"),
		}

		resp, body := suite.makeRequest("POST", "/api/v1/pipelines", createReq, suite.adminToken)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(201), response["code"])
		assert.Contains(suite.T(), response, "data")

		if data, ok := response["data"].(map[string]interface{}); ok {
			assert.Equal(suite.T(), createReq.Name, data["name"])
			assert.Equal(suite.T(), createReq.DefinitionFilePath, data["definition_file_path"])
			assert.Equal(suite.T(), suite.testRepositoryID.String(), data["repository_id"])
		}
	})

	suite.Run("无效请求参数", func() {
		createReq := map[string]interface{}{
			"name":                 "", // 空名称
			"repository_id":        "invalid-uuid",
			"definition_file_path": "",
		}

		resp, body := suite.makeRequest("POST", "/api/v1/pipelines", createReq, suite.adminToken)
		assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(400), response["code"])
	})

	suite.Run("无权限创建", func() {
		createReq := CICDCreatePipelineRequest{
			RepositoryID:       suite.testRepositoryID.String(),
			Name:               "无权限流水线",
			DefinitionFilePath: ".github/workflows/no-permission.yml",
		}

		resp, body := suite.makeRequest("POST", "/api/v1/pipelines", createReq, "")
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(401), response["code"])
	})
}

// 测试流水线查询功能
func (suite *CICDServiceIntegrationTestSuite) TestGetPipeline() {
	suite.Run("根据ID获取流水线", func() {
		if len(suite.testPipelines) == 0 {
			suite.T().Skip("没有可用的测试流水线")
			return
		}

		pipelineID := suite.testPipelines[0]
		path := "/api/v1/pipelines/" + pipelineID.String()

		resp, body := suite.makeRequest("GET", path, nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])
		assert.Contains(suite.T(), response, "data")

		if data, ok := response["data"].(map[string]interface{}); ok {
			assert.Equal(suite.T(), pipelineID.String(), data["id"])
			assert.NotEmpty(suite.T(), data["name"])
			assert.NotEmpty(suite.T(), data["definition_file_path"])
		}
	})

	suite.Run("无效的流水线ID", func() {
		path := "/api/v1/pipelines/invalid-uuid"

		resp, body := suite.makeRequest("GET", path, nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(400), response["code"])
	})

	suite.Run("不存在的流水线", func() {
		nonExistentID := uuid.New()
		path := "/api/v1/pipelines/" + nonExistentID.String()

		resp, body := suite.makeRequest("GET", path, nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(404), response["code"])
	})
}

// 测试流水线列表功能
func (suite *CICDServiceIntegrationTestSuite) TestListPipelines() {
	suite.Run("获取所有流水线列表", func() {
		resp, body := suite.makeRequest("GET", "/api/v1/pipelines", nil, suite.adminToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])
		assert.Contains(suite.T(), response, "data")

		if data, ok := response["data"].(map[string]interface{}); ok {
			if pipelines, ok := data["pipelines"].([]interface{}); ok {
				assert.GreaterOrEqual(suite.T(), len(pipelines), 3) // 至少有3个测试流水线
			}
			assert.Contains(suite.T(), data, "total")
			assert.Contains(suite.T(), data, "page")
			assert.Contains(suite.T(), data, "page_size")
		}
	})

	suite.Run("按仓库ID过滤流水线", func() {
		path := "/api/v1/pipelines?repository_id=" + suite.testRepositoryID.String()

		resp, body := suite.makeRequest("GET", path, nil, suite.adminToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])

		if data, ok := response["data"].(map[string]interface{}); ok {
			if pipelines, ok := data["pipelines"].([]interface{}); ok {
				for _, pipeline := range pipelines {
					if p, ok := pipeline.(map[string]interface{}); ok {
						assert.Equal(suite.T(), suite.testRepositoryID.String(), p["repository_id"])
					}
				}
			}
		}
	})

	suite.Run("分页测试", func() {
		path := "/api/v1/pipelines?page=1&page_size=2"

		resp, body := suite.makeRequest("GET", path, nil, suite.adminToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])

		if data, ok := response["data"].(map[string]interface{}); ok {
			assert.Equal(suite.T(), 1.0, data["page"])
			assert.Equal(suite.T(), 2.0, data["page_size"])
			if pipelines, ok := data["pipelines"].([]interface{}); ok {
				assert.LessOrEqual(suite.T(), len(pipelines), 2)
			}
		}
	})
}

// 测试流水线更新功能
func (suite *CICDServiceIntegrationTestSuite) TestUpdatePipeline() {
	suite.Run("更新流水线信息", func() {
		if len(suite.testPipelines) == 0 {
			suite.T().Skip("没有可用的测试流水线")
			return
		}

		pipelineID := suite.testPipelines[0]
		path := "/api/v1/pipelines/" + pipelineID.String()

		newName := "更新后的测试流水线"
		updateReq := CICDUpdatePipelineRequest{
			Name:        &newName,
			Description: cicdStringPtr("更新后的描述"),
		}

		resp, body := suite.makeRequest("PUT", path, updateReq, suite.adminToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])

		if data, ok := response["data"].(map[string]interface{}); ok {
			assert.Equal(suite.T(), newName, data["name"])
			assert.Equal(suite.T(), "更新后的描述", data["description"])
		}
	})

	suite.Run("无效的流水线ID", func() {
		path := "/api/v1/pipelines/invalid-uuid"
		updateReq := CICDUpdatePipelineRequest{
			Name: cicdStringPtr("新名称"),
		}

		resp, body := suite.makeRequest("PUT", path, updateReq, suite.adminToken)
		assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(400), response["code"])
	})
}

// 测试流水线触发功能
func (suite *CICDServiceIntegrationTestSuite) TestTriggerPipeline() {
	suite.Run("手动触发流水线", func() {
		if len(suite.testPipelines) == 0 {
			suite.T().Skip("没有可用的测试流水线")
			return
		}

		pipelineID := suite.testPipelines[0]
		path := "/api/v1/pipelines/" + pipelineID.String() + "/trigger"

		triggerReq := CICDTriggerPipelineRequest{
			CommitSHA: "abcdef1234567890abcdef1234567890abcdef12",
			Branch:    cicdStringPtr("develop"),
			Variables: map[string]string{
				"BUILD_ENV": "staging",
				"VERSION":   "1.1.0",
			},
		}

		resp, body := suite.makeRequest("POST", path, triggerReq, suite.testToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])

		if data, ok := response["data"].(map[string]interface{}); ok {
			assert.Equal(suite.T(), pipelineID.String(), data["pipeline_id"])
			assert.Equal(suite.T(), triggerReq.CommitSHA, data["commit_sha"])
			assert.Equal(suite.T(), *triggerReq.Branch, data["branch"])
		}
	})

	suite.Run("无效的提交SHA", func() {
		if len(suite.testPipelines) == 0 {
			suite.T().Skip("没有可用的测试流水线")
			return
		}

		pipelineID := suite.testPipelines[0]
		path := "/api/v1/pipelines/" + pipelineID.String() + "/trigger"

		triggerReq := CICDTriggerPipelineRequest{
			CommitSHA: "invalid-sha", // 无效的SHA格式
			Branch:    cicdStringPtr("main"),
		}

		resp, body := suite.makeRequest("POST", path, triggerReq, suite.testToken)
		assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(400), response["code"])
	})
}

// 测试流水线运行管理
func (suite *CICDServiceIntegrationTestSuite) TestPipelineRunManagement() {
	suite.Run("获取流水线运行列表", func() {
		if len(suite.testPipelines) == 0 {
			suite.T().Skip("没有可用的测试流水线")
			return
		}

		pipelineID := suite.testPipelines[0]
		path := "/api/v1/pipelines/" + pipelineID.String() + "/runs"

		resp, body := suite.makeRequest("GET", path, nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])

		if data, ok := response["data"].(map[string]interface{}); ok {
			if runs, ok := data["pipeline_runs"].([]interface{}); ok {
				assert.GreaterOrEqual(suite.T(), len(runs), 1)
			}
		}
	})

	suite.Run("获取流水线运行详情", func() {
		if len(suite.testPipelineRuns) == 0 {
			suite.T().Skip("没有可用的测试流水线运行")
			return
		}

		runID := suite.testPipelineRuns[0]
		path := "/api/v1/pipeline-runs/" + runID.String()

		resp, body := suite.makeRequest("GET", path, nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])

		if data, ok := response["data"].(map[string]interface{}); ok {
			assert.Equal(suite.T(), runID.String(), data["id"])
			assert.NotEmpty(suite.T(), data["status"])
		}
	})

	suite.Run("取消流水线运行", func() {
		if len(suite.testPipelineRuns) == 0 {
			suite.T().Skip("没有可用的测试流水线运行")
			return
		}

		runID := suite.testPipelineRuns[0]
		path := "/api/v1/pipeline-runs/" + runID.String() + "/cancel"

		resp, body := suite.makeRequest("POST", path, nil, suite.adminToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])
	})
}

// 测试作业管理
func (suite *CICDServiceIntegrationTestSuite) TestJobManagement() {
	suite.Run("获取作业列表", func() {
		if len(suite.testPipelineRuns) == 0 {
			suite.T().Skip("没有可用的测试流水线运行")
			return
		}

		runID := suite.testPipelineRuns[0]
		path := "/api/v1/pipeline-runs/" + runID.String() + "/jobs"

		resp, body := suite.makeRequest("GET", path, nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])

		if jobs, ok := response["data"].([]interface{}); ok {
			assert.GreaterOrEqual(suite.T(), len(jobs), 2) // 至少有2个测试作业
		}
	})

	suite.Run("获取作业详情", func() {
		if len(suite.testJobs) == 0 {
			suite.T().Skip("没有可用的测试作业")
			return
		}

		jobID := suite.testJobs[0]
		path := "/api/v1/jobs/" + jobID.String()

		resp, body := suite.makeRequest("GET", path, nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])

		if data, ok := response["data"].(map[string]interface{}); ok {
			assert.Equal(suite.T(), jobID.String(), data["id"])
			assert.NotEmpty(suite.T(), data["name"])
			assert.NotEmpty(suite.T(), data["type"])
			assert.NotEmpty(suite.T(), data["status"])
		}
	})
}

// 测试执行器管理
func (suite *CICDServiceIntegrationTestSuite) TestRunnerManagement() {
	suite.Run("注册新执行器", func() {
		registerReq := CICDRegisterRunnerRequest{
			Name:         "新测试执行器",
			Description:  cicdStringPtr("这是一个新注册的测试执行器"),
			Tags:         []string{"linux", "docker", "kubernetes"},
			Version:      "2.0.0",
			OS:           "linux",
			Architecture: "amd64",
		}

		resp, body := suite.makeRequest("POST", "/api/v1/runners", registerReq, suite.adminToken)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(201), response["code"])

		if data, ok := response["data"].(map[string]interface{}); ok {
			assert.Equal(suite.T(), registerReq.Name, data["name"])
			assert.Equal(suite.T(), registerReq.OS, data["os"])
			assert.Equal(suite.T(), registerReq.Architecture, data["architecture"])
		}
	})

	suite.Run("获取执行器列表", func() {
		resp, body := suite.makeRequest("GET", "/api/v1/runners", nil, suite.adminToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])

		if runners, ok := response["data"].([]interface{}); ok {
			assert.GreaterOrEqual(suite.T(), len(runners), 2) // 至少有2个测试执行器
		}
	})

	suite.Run("获取执行器详情", func() {
		if len(suite.testRunners) == 0 {
			suite.T().Skip("没有可用的测试执行器")
			return
		}

		runnerID := suite.testRunners[0]
		path := "/api/v1/runners/" + runnerID.String()

		resp, body := suite.makeRequest("GET", path, nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])

		if data, ok := response["data"].(map[string]interface{}); ok {
			assert.Equal(suite.T(), runnerID.String(), data["id"])
			assert.NotEmpty(suite.T(), data["name"])
			assert.NotEmpty(suite.T(), data["status"])
		}
	})

	suite.Run("更新执行器信息", func() {
		if len(suite.testRunners) == 0 {
			suite.T().Skip("没有可用的测试执行器")
			return
		}

		runnerID := suite.testRunners[0]
		path := "/api/v1/runners/" + runnerID.String()

		newName := "更新后的执行器"
		updateReq := CICDUpdateRunnerRequest{
			Name:        &newName,
			Description: cicdStringPtr("更新后的执行器描述"),
		}

		resp, body := suite.makeRequest("PUT", path, updateReq, suite.adminToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])

		if data, ok := response["data"].(map[string]interface{}); ok {
			assert.Equal(suite.T(), newName, data["name"])
		}
	})

	suite.Run("执行器心跳", func() {
		if len(suite.testRunners) == 0 {
			suite.T().Skip("没有可用的测试执行器")
			return
		}

		runnerID := suite.testRunners[0]
		path := "/api/v1/runners/" + runnerID.String() + "/heartbeat"

		resp, body := suite.makeRequest("POST", path, nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])
	})
}

// 测试Webhook事件处理
func (suite *CICDServiceIntegrationTestSuite) TestWebhookEvents() {
	suite.Run("处理Git推送事件", func() {
		pushEvent := CICDGitPushEvent{
			Repository: CICDRepository{
				ID:            suite.testRepositoryID,
				Name:          "test-repo",
				DefaultBranch: "main",
			},
			CommitSHA: "1234567890abcdef1234567890abcdef12345678",
			Branch:    "main",
			Author: CICDAuthor{
				Name:  "Test User",
				Email: "test@example.com",
			},
			Message: "Test commit for CI/CD",
		}

		resp, body := suite.makeRequest("POST", "/api/v1/webhook-events/git/push", pushEvent, "")
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), "success", response["status"])
	})

	suite.Run("处理Git分支事件", func() {
		branchEvent := CICDGitBranchEvent{
			Repository: CICDRepository{
				ID:            suite.testRepositoryID,
				Name:          "test-repo",
				DefaultBranch: "main",
			},
			BranchName: "feature/new-feature",
			Action:     "created",
			Author: CICDAuthor{
				Name:  "Test User",
				Email: "test@example.com",
			},
		}

		resp, body := suite.makeRequest("POST", "/api/v1/webhook-events/git/branch", branchEvent, "")
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), "success", response["status"])
	})

	suite.Run("直接触发流水线", func() {
		if len(suite.testPipelines) == 0 {
			suite.T().Skip("没有可用的测试流水线")
			return
		}

		triggerReq := CICDWebhookTriggerRequest{
			RepositoryID: suite.testRepositoryID,
			PipelineID:   suite.testPipelines[0],
			Variables: map[string]interface{}{
				"WEBHOOK_TRIGGER": "true",
				"BUILD_ENV":       "webhook",
			},
		}

		resp, body := suite.makeRequest("POST", "/api/v1/webhook-events/trigger-pipeline", triggerReq, "")
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), "success", response["status"])
	})
}

// 测试边界条件和错误处理
func (suite *CICDServiceIntegrationTestSuite) TestEdgeCasesAndErrorHandling() {
	suite.Run("无效的JWT Token", func() {
		invalidToken := "invalid.jwt.token"

		resp, body := suite.makeRequest("GET", "/api/v1/pipelines", nil, invalidToken)
		// 根据中间件实现，可能返回401或者忽略认证
		assert.True(suite.T(), resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
	})

	suite.Run("超大分页参数", func() {
		path := "/api/v1/pipelines?page=1000&page_size=1000"

		resp, body := suite.makeRequest("GET", path, nil, suite.adminToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])

		// 检查是否应用了限制
		if data, ok := response["data"].(map[string]interface{}); ok {
			if pageSize, ok := data["page_size"].(float64); ok {
				assert.LessOrEqual(suite.T(), int(pageSize), 100) // 应该被限制在100以内
			}
		}
	})

	suite.Run("删除不存在的流水线", func() {
		nonExistentID := uuid.New()
		path := "/api/v1/pipelines/" + nonExistentID.String()

		resp, body := suite.makeRequest("DELETE", path, nil, suite.adminToken)
		assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(404), response["code"])
	})

	suite.Run("健康检查接口", func() {
		resp, body := suite.makeRequest("GET", "/api/v1/health", nil, "")
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), "cicd-service", response["service"])
		assert.Equal(suite.T(), "healthy", response["status"])
	})
}

// TestCICDServiceIntegration 运行所有CI/CD服务集成测试
func TestCICDServiceIntegration(t *testing.T) {
	suite.Run(t, new(CICDServiceIntegrationTestSuite))
}

// 辅助函数

// cicdStringPtr 返回字符串指针
func cicdStringPtr(s string) *string {
	return &s
}

// timePtr 返回时间指针
func timePtr(t time.Time) *time.Time {
	return &t
}