package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/notification-service/handlers"
	"github.com/cloud-platform/collaborative-dev/internal/notification-service/models"
	"github.com/cloud-platform/collaborative-dev/internal/notification-service/repository"
	"github.com/cloud-platform/collaborative-dev/internal/notification-service/services"
	"github.com/cloud-platform/collaborative-dev/shared/auth"
	"github.com/cloud-platform/collaborative-dev/shared/database"
	"github.com/cloud-platform/collaborative-dev/shared/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// NotificationServiceIntegrationTestSuite 通知服务集成测试套件
type NotificationServiceIntegrationTestSuite struct {
	suite.Suite
	db                     *gorm.DB
	logger                 logger.Logger
	router                 *gin.Engine
	server                 *httptest.Server
	notificationService    *services.NotificationService
	notificationHandler    *handlers.NotificationHandler
	jwtService             *auth.JWTService
	testTenantID           uuid.UUID
	testUserID             uuid.UUID
	testProjectID          uuid.UUID
	testToken              string
	testNotifications      []uuid.UUID
	testTemplates          []uuid.UUID
	testRules              []uuid.UUID
	mockEmailService       *services.MockEmailService
	mockWebhookService     *services.MockWebhookService
	mockInAppService       *services.MockInAppService
}

// SetupSuite 初始化测试套件
func (suite *NotificationServiceIntegrationTestSuite) SetupSuite() {
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
		"test-secret-key-for-notification-integration-testing",
		time.Hour*1,
		time.Hour*24*7,
	)

	// 初始化测试数据
	suite.testTenantID = uuid.New()
	suite.testUserID = uuid.New()
	suite.testProjectID = uuid.New()
	suite.testNotifications = []uuid.UUID{}
	suite.testTemplates = []uuid.UUID{}
	suite.testRules = []uuid.UUID{}

	// 生成测试JWT Token
	tokenPair, err := suite.jwtService.GenerateTokenPair(
		suite.testUserID,
		suite.testTenantID,
		"test@example.com",
		"user",
		[]string{"notifications:read", "notifications:write"},
	)
	if err != nil {
		suite.T().Fatalf("生成测试Token失败: %v", err)
	}
	suite.testToken = tokenPair.AccessToken

	// 初始化存储库
	notificationRepo := repository.NewNotificationRepository(suite.db)
	templateRepo := repository.NewTemplateRepository(suite.db)
	ruleRepo := repository.NewRuleRepository(suite.db)
	deliveryLogRepo := repository.NewDeliveryLogRepository(suite.db)

	// 初始化Mock服务
	suite.mockEmailService = &services.MockEmailService{}
	suite.mockWebhookService = &services.MockWebhookService{}
	suite.mockInAppService = &services.MockInAppService{}

	// 初始化模板引擎
	templateEngine := services.NewTemplateEngine()

	// 初始化投递服务
	deliveryService := services.NewDeliveryService(
		notificationRepo,
		deliveryLogRepo,
		suite.mockEmailService,
		suite.mockWebhookService,
		suite.mockInAppService,
		suite.logger,
	)

	// 初始化通知服务
	suite.notificationService = services.NewNotificationService(
		notificationRepo,
		templateRepo,
		ruleRepo,
		templateEngine,
		deliveryService,
		suite.logger,
	)

	// 初始化处理器
	suite.notificationHandler = handlers.NewNotificationHandler(
		suite.notificationService,
		suite.logger,
	)

	// 设置路由
	suite.setupRouter()

	// 启动测试服务器
	suite.server = httptest.NewServer(suite.router)

	// 创建测试基础数据
	suite.createTestData()

	suite.logger.Info("通知服务集成测试套件初始化完成",
		"tenant_id", suite.testTenantID,
		"user_id", suite.testUserID,
		"server_url", suite.server.URL)
}

// setupTestDatabase 设置测试数据库
func (suite *NotificationServiceIntegrationTestSuite) setupTestDatabase() {
	config := database.Config{
		Host:     "localhost",
		Port:     5432,
		Name:     "test_notification_service",
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
		&models.Notification{},
		&models.NotificationTemplate{},
		&models.NotificationRule{},
		&models.DeliveryLog{},
		&models.UserNotificationSettings{},
	)
	if err != nil {
		suite.T().Fatalf("数据库迁移失败: %v", err)
	}
}

// setupRouter 设置路由
func (suite *NotificationServiceIntegrationTestSuite) setupRouter() {
	suite.router = gin.New()

	// 添加认证中间件模拟
	suite.router.Use(func(c *gin.Context) {
		// 对于需要认证的路由，从Authorization头获取token
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token := authHeader[7:]
			claims, err := suite.jwtService.ValidateToken(token)
			if err == nil {
				c.Set("user_id", claims.UserID.String())
				c.Set("tenant_id", claims.TenantID.String())
			}
		}
		c.Next()
	})

	// 通知API路由
	api := suite.router.Group("/api/v1")
	{
		notifications := api.Group("/notifications")
		{
			notifications.GET("", suite.notificationHandler.GetNotifications)
			notifications.POST("", suite.notificationHandler.CreateNotification)
			notifications.GET("/unread/count", suite.notificationHandler.GetUnreadCount)
			notifications.POST("/:id/read", suite.notificationHandler.MarkAsRead)
			notifications.DELETE("/:id", suite.notificationHandler.DeleteNotification)
			notifications.POST("/:id/retry", suite.notificationHandler.RetryNotification)
			notifications.GET("/correlation/:correlation_id", suite.notificationHandler.GetNotificationsByCorrelationID)
		}
	}
}

// createTestData 创建测试基础数据
func (suite *NotificationServiceIntegrationTestSuite) createTestData() {
	ctx := context.Background()

	// 创建测试通知模板
	templates := []models.NotificationTemplate{
		{
			ID:              uuid.New(),
			TenantID:        suite.testTenantID,
			Name:            "任务分配通知",
			Type:            "task_assigned",
			Category:        "project",
			Description:     "任务分配给用户时的通知模板",
			SubjectTemplate: "新任务分配: {{.event.task_title}}",
			BodyTemplate:    "您有一个新任务: {{.event.task_title}}，截止时间: {{.event.due_date}}",
			Language:        "zh-CN",
			Format:          "text",
			DefaultChannels: models.Channels{
				InApp: &models.InAppChannel{
					Enabled: true,
					Badge:   true,
				},
				Email: &models.EmailChannel{
					Enabled: true,
				},
			},
			IsActive:  true,
			Version:   1,
			CreatedBy: suite.testUserID,
			UpdatedBy: suite.testUserID,
		},
		{
			ID:              uuid.New(),
			TenantID:        suite.testTenantID,
			Name:            "构建失败通知",
			Type:            "build_failed",
			Category:        "cicd",
			Description:     "构建失败时的通知模板",
			SubjectTemplate: "构建失败: {{.event.project_name}}",
			BodyTemplate:    "项目 {{.event.project_name}} 的构建失败，错误信息: {{.event.error_message}}",
			Language:        "zh-CN",
			Format:          "text",
			DefaultChannels: models.Channels{
				InApp: &models.InAppChannel{
					Enabled: true,
					Badge:   true,
				},
				Email: &models.EmailChannel{
					Enabled: true,
				},
			},
			IsActive:  true,
			Version:   1,
			CreatedBy: suite.testUserID,
			UpdatedBy: suite.testUserID,
		},
	}

	for _, template := range templates {
		suite.db.WithContext(ctx).Create(&template)
		suite.testTemplates = append(suite.testTemplates, template.ID)
	}

	// 创建测试通知规则
	rules := []models.NotificationRule{
		{
			ID:          uuid.New(),
			TenantID:    suite.testTenantID,
			UserID:      &suite.testUserID,
			ProjectID:   &suite.testProjectID,
			Name:        "任务分配规则",
			Description: "当任务分配给用户时发送通知",
			EventTypes:  []string{"task_assigned"},
			TemplateID:  templates[0].ID,
			Channels: models.Channels{
				InApp: &models.InAppChannel{
					Enabled: true,
					Badge:   true,
				},
			},
			Priority:  "medium",
			IsActive:  true,
			CreatedBy: suite.testUserID,
			UpdatedBy: suite.testUserID,
		},
		{
			ID:          uuid.New(),
			TenantID:    suite.testTenantID,
			UserID:      &suite.testUserID,
			ProjectID:   &suite.testProjectID,
			Name:        "构建失败规则",
			Description: "当构建失败时发送通知",
			EventTypes:  []string{"build_failed"},
			TemplateID:  templates[1].ID,
			Channels: models.Channels{
				InApp: &models.InAppChannel{
					Enabled: true,
					Badge:   true,
				},
				Email: &models.EmailChannel{
					Enabled: true,
				},
			},
			Priority: "high",
			RateLimit: &models.RateLimit{
				MaxCount:   5,
				TimeWindow: 60,
				Strategy:   "throttle",
			},
			IsActive:  true,
			CreatedBy: suite.testUserID,
			UpdatedBy: suite.testUserID,
		},
	}

	for _, rule := range rules {
		suite.db.WithContext(ctx).Create(&rule)
		suite.testRules = append(suite.testRules, rule.ID)
	}

	// 创建测试通知
	notifications := []models.Notification{
		{
			ID:        uuid.New(),
			UserID:    &suite.testUserID,
			TenantID:  suite.testTenantID,
			ProjectID: &suite.testProjectID,
			Type:      "task_assigned",
			Category:  "project",
			Priority:  "medium",
			Title:     "新任务分配: 开发登录功能",
			Content:   "您有一个新任务: 开发登录功能，截止时间: 2024-12-31",
			Status:    models.StatusSent,
			Channels: models.Channels{
				InApp: &models.InAppChannel{
					Enabled: true,
					Badge:   true,
				},
			},
			TemplateID:    &templates[0].ID,
			CorrelationID: "task-123",
			SourceEvent:   "task.assigned",
			CreatedBy:     suite.testUserID,
		},
		{
			ID:        uuid.New(),
			UserID:    &suite.testUserID,
			TenantID:  suite.testTenantID,
			ProjectID: &suite.testProjectID,
			Type:      "build_failed",
			Category:  "cicd",
			Priority:  "high",
			Title:     "构建失败: 协作开发平台",
			Content:   "项目 协作开发平台 的构建失败，错误信息: 编译错误",
			Status:    models.StatusSent,
			Channels: models.Channels{
				InApp: &models.InAppChannel{
					Enabled: true,
					Badge:   true,
				},
				Email: &models.EmailChannel{
					Enabled: true,
				},
			},
			TemplateID:    &templates[1].ID,
			CorrelationID: "build-456",
			SourceEvent:   "build.failed",
			CreatedBy:     suite.testUserID,
		},
	}

	for _, notification := range notifications {
		suite.db.WithContext(ctx).Create(&notification)
		suite.testNotifications = append(suite.testNotifications, notification.ID)
	}
}

// TearDownSuite 清理测试套件
func (suite *NotificationServiceIntegrationTestSuite) TearDownSuite() {
	if suite.server != nil {
		suite.server.Close()
	}
	suite.logger.Info("通知服务集成测试套件清理完成")
}

// makeRequest 发送HTTP请求
func (suite *NotificationServiceIntegrationTestSuite) makeRequest(method, path string, body interface{}, token string) (*http.Response, []byte) {
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

// 测试通知获取功能
func (suite *NotificationServiceIntegrationTestSuite) TestGetNotifications() {
	suite.Run("获取通知列表", func() {
		resp, body := suite.makeRequest("GET", "/api/v1/notifications", nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])
		assert.Contains(suite.T(), response, "data")

		if data, ok := response["data"].(map[string]interface{}); ok {
			if notifications, ok := data["notifications"].([]interface{}); ok {
				assert.GreaterOrEqual(suite.T(), len(notifications), 1)
			}
		}
	})

	suite.Run("按项目过滤通知", func() {
		path := "/api/v1/notifications?project_id=" + suite.testProjectID.String()
		resp, body := suite.makeRequest("GET", path, nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])
	})

	suite.Run("按类别过滤通知", func() {
		path := "/api/v1/notifications?category=project"
		resp, body := suite.makeRequest("GET", path, nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])
	})

	suite.Run("分页获取通知", func() {
		path := "/api/v1/notifications?limit=10&offset=0"
		resp, body := suite.makeRequest("GET", path, nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])
	})

	suite.Run("无令牌访问", func() {
		resp, body := suite.makeRequest("GET", "/api/v1/notifications", nil, "")
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(401), response["code"])
	})
}

// 测试未读通知数量
func (suite *NotificationServiceIntegrationTestSuite) TestGetUnreadCount() {
	suite.Run("获取未读通知数量", func() {
		resp, body := suite.makeRequest("GET", "/api/v1/notifications/unread/count", nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])
		assert.Contains(suite.T(), response, "data")

		if data, ok := response["data"].(map[string]interface{}); ok {
			assert.Contains(suite.T(), data, "unread_count")
		}
	})

	suite.Run("无令牌访问", func() {
		resp, body := suite.makeRequest("GET", "/api/v1/notifications/unread/count", nil, "")
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(401), response["code"])
	})
}

// 测试创建通知功能
func (suite *NotificationServiceIntegrationTestSuite) TestCreateNotification() {
	suite.Run("创建新通知", func() {
		createReq := services.CreateNotificationRequest{
			TenantID:  suite.testTenantID,
			UserID:    &suite.testUserID,
			ProjectID: &suite.testProjectID,
			Type:      "task_assigned",
			Category:  "project",
			Priority:  "medium",
			Title:     "测试通知",
			Content:   "这是一个测试通知",
		}

		resp, body := suite.makeRequest("POST", "/api/v1/notifications", createReq, suite.testToken)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(201), response["code"])
	})

	suite.Run("缺少必填字段", func() {
		createReq := map[string]interface{}{
			"tenant_id": suite.testTenantID,
			// 缺少type字段
			"category": "project",
			"priority": "medium",
		}

		resp, body := suite.makeRequest("POST", "/api/v1/notifications", createReq, suite.testToken)
		assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(400), response["code"])
	})

	suite.Run("无效的JSON请求", func() {
		req, err := http.NewRequest("POST", suite.server.URL+"/api/v1/notifications",
			bytes.NewBuffer([]byte("invalid json")))
		suite.NoError(err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+suite.testToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		suite.NoError(err)

		assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)
		resp.Body.Close()
	})
}

// 测试通知状态管理
func (suite *NotificationServiceIntegrationTestSuite) TestNotificationStatusManagement() {
	// 标记通知为已读
	suite.Run("标记通知为已读", func() {
		if len(suite.testNotifications) == 0 {
			suite.T().Skip("没有可用的测试通知")
			return
		}

		notificationID := suite.testNotifications[0]
		path := "/api/v1/notifications/" + notificationID.String() + "/read"
		
		resp, body := suite.makeRequest("POST", path, nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])
	})

	// 删除通知
	suite.Run("删除通知", func() {
		if len(suite.testNotifications) < 2 {
			suite.T().Skip("没有足够的测试通知")
			return
		}

		notificationID := suite.testNotifications[1]
		path := "/api/v1/notifications/" + notificationID.String()
		
		resp, body := suite.makeRequest("DELETE", path, nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])
	})

	// 无效的通知ID
	suite.Run("无效的通知ID", func() {
		path := "/api/v1/notifications/invalid-uuid/read"
		
		resp, body := suite.makeRequest("POST", path, nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(400), response["code"])
	})
}

// 测试关联ID查询
func (suite *NotificationServiceIntegrationTestSuite) TestGetNotificationsByCorrelationID() {
	suite.Run("根据关联ID获取通知", func() {
		correlationID := "task-123"
		path := "/api/v1/notifications/correlation/" + correlationID
		
		resp, body := suite.makeRequest("GET", path, nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])
		assert.Contains(suite.T(), response, "data")

		if data, ok := response["data"].(map[string]interface{}); ok {
			assert.Equal(suite.T(), correlationID, data["correlation_id"])
			assert.Contains(suite.T(), data, "notifications")
			assert.Contains(suite.T(), data, "total")
		}
	})

	suite.Run("空的关联ID", func() {
		path := "/api/v1/notifications/correlation/"
		
		resp, body := suite.makeRequest("GET", path, nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(404), response["code"])
	})
}

// 测试边界条件和错误处理
func (suite *NotificationServiceIntegrationTestSuite) TestEdgeCasesAndErrorHandling() {
	suite.Run("无效的用户ID格式", func() {
		// 创建无效的token
		invalidToken := "invalid.jwt.token"
		
		resp, body := suite.makeRequest("GET", "/api/v1/notifications", nil, invalidToken)
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(401), response["code"])
	})

	suite.Run("超大分页参数", func() {
		path := "/api/v1/notifications?limit=1000&offset=99999"
		
		resp, body := suite.makeRequest("GET", path, nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])

		// 检查是否应用了默认限制
		if data, ok := response["data"].(map[string]interface{}); ok {
			if limit, ok := data["limit"].(float64); ok {
				assert.LessOrEqual(suite.T(), int(limit), 100) // 应该被限制在合理范围内
			}
		}
	})

	suite.Run("无效的查询参数", func() {
		path := "/api/v1/notifications?limit=invalid&offset=invalid"
		
		resp, body := suite.makeRequest("GET", path, nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])

		// 应该使用默认值
		if data, ok := response["data"].(map[string]interface{}); ok {
			if limit, ok := data["limit"].(float64); ok {
				assert.Equal(suite.T(), 20.0, limit) // 默认限制
			}
		}
	})
}

// TestNotificationServiceIntegration 运行所有通知服务集成测试
func TestNotificationServiceIntegration(t *testing.T) {
	suite.Run(t, new(NotificationServiceIntegrationTestSuite))
}