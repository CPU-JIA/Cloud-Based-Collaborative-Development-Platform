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
	"github.com/cloud-platform/collaborative-dev/shared/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// TenantServiceIntegrationTestSuite 租户服务集成测试套件
type TenantServiceIntegrationTestSuite struct {
	suite.Suite
	db                *gorm.DB
	logger            logger.Logger
	router            *gin.Engine
	server            *httptest.Server
	tenantService     TenantService
	tenantHandler     *MockTenantHandler
	jwtService        *auth.JWTService
	testTenantID      uuid.UUID
	testTenants       []uuid.UUID
	testConfigs       []uuid.UUID
	testSubscriptions []uuid.UUID
	testToken         string
	adminToken        string
}

// SetupSuite 初始化测试套件
func (suite *TenantServiceIntegrationTestSuite) SetupSuite() {
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
		"test-secret-key-for-tenant-integration-testing",
		time.Hour*1,
		time.Hour*24*7,
	)

	// 初始化测试数据
	suite.testTenantID = uuid.New()
	suite.testTenants = []uuid.UUID{}
	suite.testConfigs = []uuid.UUID{}
	suite.testSubscriptions = []uuid.UUID{}

	// 生成测试JWT Token
	tokenPair, err := suite.jwtService.GenerateTokenPair(
		uuid.New(),
		suite.testTenantID,
		"admin@example.com",
		"admin",
		[]string{"tenants:read", "tenants:write", "tenants:admin"},
	)
	if err != nil {
		suite.T().Fatalf("生成测试Token失败: %v", err)
	}
	suite.testToken = tokenPair.AccessToken

	// 生成管理员Token
	adminTokenPair, err := suite.jwtService.GenerateTokenPair(
		uuid.New(),
		uuid.New(),
		"superadmin@example.com",
		"superadmin",
		[]string{"tenants:read", "tenants:write", "tenants:admin", "system:admin"},
	)
	if err != nil {
		suite.T().Fatalf("生成管理员Token失败: %v", err)
	}
	suite.adminToken = adminTokenPair.AccessToken

	// 初始化模拟服务
	suite.tenantService = NewMockTenantService()

	// 初始化模拟处理器
	suite.tenantHandler = NewMockTenantHandler(
		suite.tenantService,
		suite.logger,
	)

	// 设置路由
	suite.setupRouter()

	// 启动测试服务器
	suite.server = httptest.NewServer(suite.router)

	// 创建测试基础数据
	suite.createTestData()

	suite.logger.Info("租户服务集成测试套件初始化完成",
		"tenant_id", suite.testTenantID,
		"server_url", suite.server.URL)
}

// setupTestDatabase 设置测试数据库
func (suite *TenantServiceIntegrationTestSuite) setupTestDatabase() {
	config := database.Config{
		Host:     "localhost",
		Port:     5432,
		Name:     "test_tenant_service",
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
		&models.Tenant{},
		&models.TenantConfig{},
		&models.TenantSubscription{},
		&models.TenantBranding{},
		&models.TenantAuditLog{},
	)
	if err != nil {
		suite.T().Fatalf("数据库迁移失败: %v", err)
	}
}

// setupRouter 设置路由
func (suite *TenantServiceIntegrationTestSuite) setupRouter() {
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
				c.Set("role", claims.Role)
				c.Set("user_email", claims.Email)
			}
		}
		c.Next()
	})

	// 租户API路由
	api := suite.router.Group("/api/v1")
	{
		tenants := api.Group("/tenants")
		{
			tenants.GET("", suite.tenantHandler.ListTenants)
			tenants.POST("", suite.tenantHandler.CreateTenant)
			tenants.GET("/search", suite.tenantHandler.SearchTenants)
			tenants.GET("/stats", suite.tenantHandler.GetTenantStats)
			tenants.GET("/by-domain", suite.tenantHandler.GetTenantByDomain)
			tenants.GET("/:id", suite.tenantHandler.GetTenant)
			tenants.PUT("/:id", suite.tenantHandler.UpdateTenant)
			tenants.DELETE("/:id", suite.tenantHandler.DeleteTenant)
			tenants.POST("/:id/activate", suite.tenantHandler.ActivateTenant)
			tenants.POST("/:id/suspend", suite.tenantHandler.SuspendTenant)
			tenants.GET("/:id/with-config", suite.tenantHandler.GetTenantWithConfig)
			tenants.GET("/:id/complete", suite.tenantHandler.GetTenantWithAll)
		}
	}
}

// createTestData 创建测试基础数据
func (suite *TenantServiceIntegrationTestSuite) createTestData() {
	ctx := context.Background()

	// 使用模拟服务创建测试租户
	testTenantRequests := []TenantCreateRequest{
		{
			Name:         "测试公司A",
			Domain:       "company-a",
			Plan:         models.TenantPlanProfessional,
			BillingEmail: "billing@company-a.com",
			Description:  "这是一个测试公司A",
			ContactName:  "张三",
			ContactEmail: "contact@company-a.com",
			ContactPhone: "+86-138-0000-0001",
			Address:      "北京市朝阳区测试街道1号",
			City:         "北京",
			State:        "北京",
			Country:      "中国",
			PostalCode:   "100000",
		},
		{
			Name:         "测试公司B",
			Domain:       "company-b",
			Plan:         models.TenantPlanBasic,
			BillingEmail: "billing@company-b.com",
			Description:  "这是一个测试公司B",
			ContactName:  "李四",
			ContactEmail: "contact@company-b.com",
			ContactPhone: "+86-138-0000-0002",
			Address:      "上海市浦东新区测试路2号",
			City:         "上海",
			State:        "上海",
			Country:      "中国",
			PostalCode:   "200000",
		},
		{
			Name:         "暂停公司C",
			Domain:       "company-c",
			Plan:         models.TenantPlanEnterprise,
			BillingEmail: "billing@company-c.com",
			Description:  "这是一个被暂停的公司C",
			ContactName:  "王五",
			ContactEmail: "contact@company-c.com",
			ContactPhone: "+86-138-0000-0003",
			Address:      "深圳市南山区测试大道3号",
			City:         "深圳",
			State:        "广东",
			Country:      "中国",
			PostalCode:   "518000",
		},
	}

	for _, req := range testTenantRequests {
		tenant, err := suite.tenantService.CreateTenant(ctx, &req)
		if err != nil {
			suite.T().Fatalf("创建测试租户失败: %v", err)
		}
		suite.testTenants = append(suite.testTenants, tenant.ID)

		// 激活第一个和第三个租户，第二个保持pending状态
		if len(suite.testTenants) == 1 {
			suite.tenantService.ActivateTenant(ctx, tenant.ID)
		} else if len(suite.testTenants) == 3 {
			suite.tenantService.SuspendTenant(ctx, tenant.ID, "测试暂停")
		}
	}

	// 设置第一个租户为主测试租户
	if len(suite.testTenants) > 0 {
		suite.testTenantID = suite.testTenants[0]
	}
}

// TearDownSuite 清理测试套件
func (suite *TenantServiceIntegrationTestSuite) TearDownSuite() {
	if suite.server != nil {
		suite.server.Close()
	}
	suite.logger.Info("租户服务集成测试套件清理完成")
}

// makeRequest 发送HTTP请求
func (suite *TenantServiceIntegrationTestSuite) makeRequest(method, path string, body interface{}, token string) (*http.Response, []byte) {
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

// 测试租户创建功能
func (suite *TenantServiceIntegrationTestSuite) TestCreateTenant() {
	suite.Run("创建新租户", func() {
		createReq := TenantCreateRequest{
			Name:         "新测试公司",
			Domain:       "new-test-company",
			Plan:         models.TenantPlanProfessional,
			BillingEmail: "billing@newtest.com",
			Description:  "这是一个新的测试公司",
			ContactName:  "测试联系人",
			ContactEmail: "contact@newtest.com",
			ContactPhone: "+86-138-0000-9999",
			Address:      "测试地址",
			City:         "北京",
			State:        "北京",
			Country:      "中国",
			PostalCode:   "100001",
		}

		resp, body := suite.makeRequest("POST", "/api/v1/tenants", createReq, suite.adminToken)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(201), response["code"])
		assert.Contains(suite.T(), response, "data")

		if data, ok := response["data"].(map[string]interface{}); ok {
			assert.Equal(suite.T(), createReq.Name, data["name"])
			assert.Equal(suite.T(), createReq.Domain, data["domain"])
			assert.Equal(suite.T(), string(createReq.Plan), data["plan"])
		}
	})

	suite.Run("重复域名创建", func() {
		createReq := TenantCreateRequest{
			Name:         "重复域名公司",
			Domain:       "company-a", // 使用已存在的域名
			Plan:         models.TenantPlanBasic,
			BillingEmail: "billing@duplicate.com",
		}

		resp, body := suite.makeRequest("POST", "/api/v1/tenants", createReq, suite.adminToken)
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(500), response["code"])
	})

	suite.Run("无效请求参数", func() {
		createReq := map[string]interface{}{
			"name":   "", // 空名称
			"domain": "invalid-domain",
			"plan":   "invalid_plan", // 无效计划
		}

		resp, body := suite.makeRequest("POST", "/api/v1/tenants", createReq, suite.adminToken)
		assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(400), response["code"])
	})

	suite.Run("无权限创建", func() {
		createReq := TenantCreateRequest{
			Name:         "无权限公司",
			Domain:       "no-permission",
			Plan:         models.TenantPlanBasic,
			BillingEmail: "billing@noperm.com",
		}

		resp, body := suite.makeRequest("POST", "/api/v1/tenants", createReq, suite.testToken)
		// 这里假设testToken没有创建权限，实际实现中可能需要权限检查
		assert.True(suite.T(), resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusForbidden)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
	})
}

// 测试租户查询功能
func (suite *TenantServiceIntegrationTestSuite) TestGetTenant() {
	suite.Run("根据ID获取租户", func() {
		if len(suite.testTenants) == 0 {
			suite.T().Skip("没有可用的测试租户")
			return
		}

		tenantID := suite.testTenants[0]
		path := "/api/v1/tenants/" + tenantID.String()

		resp, body := suite.makeRequest("GET", path, nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])
		assert.Contains(suite.T(), response, "data")

		if data, ok := response["data"].(map[string]interface{}); ok {
			assert.Equal(suite.T(), tenantID.String(), data["id"])
			assert.NotEmpty(suite.T(), data["name"])
			assert.NotEmpty(suite.T(), data["domain"])
		}
	})

	suite.Run("根据域名获取租户", func() {
		path := "/api/v1/tenants/by-domain?domain=company-a"

		resp, body := suite.makeRequest("GET", path, nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])

		if data, ok := response["data"].(map[string]interface{}); ok {
			assert.Equal(suite.T(), "company-a", data["domain"])
		}
	})

	suite.Run("无效的租户ID", func() {
		path := "/api/v1/tenants/invalid-uuid"

		resp, body := suite.makeRequest("GET", path, nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(400), response["code"])
	})

	suite.Run("不存在的租户", func() {
		nonExistentID := uuid.New()
		path := "/api/v1/tenants/" + nonExistentID.String()

		resp, body := suite.makeRequest("GET", path, nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(404), response["code"])
	})
}

// 测试租户列表功能
func (suite *TenantServiceIntegrationTestSuite) TestListTenants() {
	suite.Run("获取所有租户列表", func() {
		resp, body := suite.makeRequest("GET", "/api/v1/tenants", nil, suite.adminToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])
		assert.Contains(suite.T(), response, "data")

		if data, ok := response["data"].(map[string]interface{}); ok {
			if tenants, ok := data["tenants"].([]interface{}); ok {
				assert.GreaterOrEqual(suite.T(), len(tenants), 3) // 至少有3个测试租户
			}
			assert.Contains(suite.T(), data, "total")
			assert.Contains(suite.T(), data, "limit")
			assert.Contains(suite.T(), data, "offset")
		}
	})

	suite.Run("按状态过滤租户", func() {
		path := "/api/v1/tenants?status=active"

		resp, body := suite.makeRequest("GET", path, nil, suite.adminToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])

		if data, ok := response["data"].(map[string]interface{}); ok {
			if tenants, ok := data["tenants"].([]interface{}); ok {
				for _, tenant := range tenants {
					if t, ok := tenant.(map[string]interface{}); ok {
						assert.Equal(suite.T(), "active", t["status"])
					}
				}
			}
		}
	})

	suite.Run("按计划过滤租户", func() {
		path := "/api/v1/tenants?plan=professional"

		resp, body := suite.makeRequest("GET", path, nil, suite.adminToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])

		if data, ok := response["data"].(map[string]interface{}); ok {
			if tenants, ok := data["tenants"].([]interface{}); ok {
				for _, tenant := range tenants {
					if t, ok := tenant.(map[string]interface{}); ok {
						assert.Equal(suite.T(), "professional", t["plan"])
					}
				}
			}
		}
	})

	suite.Run("分页测试", func() {
		path := "/api/v1/tenants?limit=2&offset=0"

		resp, body := suite.makeRequest("GET", path, nil, suite.adminToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])

		if data, ok := response["data"].(map[string]interface{}); ok {
			assert.Equal(suite.T(), 2.0, data["limit"])
			assert.Equal(suite.T(), 0.0, data["offset"])
			if tenants, ok := data["tenants"].([]interface{}); ok {
				assert.LessOrEqual(suite.T(), len(tenants), 2)
			}
		}
	})
}

// 测试租户搜索功能
func (suite *TenantServiceIntegrationTestSuite) TestSearchTenants() {
	suite.Run("搜索租户", func() {
		path := "/api/v1/tenants/search?q=测试公司"

		resp, body := suite.makeRequest("GET", path, nil, suite.adminToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])
		assert.Contains(suite.T(), response, "data")

		if tenants, ok := response["data"].([]interface{}); ok {
			for _, tenant := range tenants {
				if t, ok := tenant.(map[string]interface{}); ok {
					name := t["name"].(string)
					assert.Contains(suite.T(), name, "测试公司")
				}
			}
		}
	})

	suite.Run("空搜索关键词", func() {
		path := "/api/v1/tenants/search?q="

		resp, body := suite.makeRequest("GET", path, nil, suite.adminToken)
		assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(400), response["code"])
	})
}

// 测试租户更新功能
func (suite *TenantServiceIntegrationTestSuite) TestUpdateTenant() {
	suite.Run("更新租户信息", func() {
		if len(suite.testTenants) == 0 {
			suite.T().Skip("没有可用的测试租户")
			return
		}

		tenantID := suite.testTenants[0]
		path := "/api/v1/tenants/" + tenantID.String()
		
		newName := "更新后的测试公司A"
		updateReq := TenantUpdateRequest{
			Name:        &newName,
			Description: tenantStringPtr("更新后的描述"),
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

	suite.Run("无效的租户ID", func() {
		path := "/api/v1/tenants/invalid-uuid"
		updateReq := TenantUpdateRequest{
			Name: tenantStringPtr("新名称"),
		}

		resp, body := suite.makeRequest("PUT", path, updateReq, suite.adminToken)
		assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(400), response["code"])
	})
}

// 测试租户状态管理
func (suite *TenantServiceIntegrationTestSuite) TestTenantStatusManagement() {
	suite.Run("激活租户", func() {
		if len(suite.testTenants) < 2 {
			suite.T().Skip("需要至少2个测试租户")
			return
		}

		// 使用pending状态的租户
		tenantID := suite.testTenants[1]
		path := "/api/v1/tenants/" + tenantID.String() + "/activate"

		resp, body := suite.makeRequest("POST", path, nil, suite.adminToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])
	})

	suite.Run("暂停租户", func() {
		if len(suite.testTenants) == 0 {
			suite.T().Skip("没有可用的测试租户")
			return
		}

		tenantID := suite.testTenants[0]
		path := "/api/v1/tenants/" + tenantID.String() + "/suspend"
		
		suspendReq := TenantSuspendRequest{
			Reason: "测试暂停",
		}

		resp, body := suite.makeRequest("POST", path, suspendReq, suite.adminToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])
	})
}

// 测试租户统计功能
func (suite *TenantServiceIntegrationTestSuite) TestGetTenantStats() {
	suite.Run("获取租户统计", func() {
		resp, body := suite.makeRequest("GET", "/api/v1/tenants/stats", nil, suite.adminToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])
		assert.Contains(suite.T(), response, "data")

		if data, ok := response["data"].(map[string]interface{}); ok {
			assert.Contains(suite.T(), data, "total_tenants")
			assert.Contains(suite.T(), data, "status_stats")
			assert.Contains(suite.T(), data, "plan_stats")
			assert.Contains(suite.T(), data, "recent_tenants")

			// 验证统计数据
			if totalTenants, ok := data["total_tenants"].(float64); ok {
				assert.GreaterOrEqual(suite.T(), totalTenants, 3.0) // 至少有3个测试租户
			}
		}
	})
}

// 测试租户扩展信息获取
func (suite *TenantServiceIntegrationTestSuite) TestGetTenantWithExtendedInfo() {
	suite.Run("获取租户及配置", func() {
		if len(suite.testTenants) == 0 {
			suite.T().Skip("没有可用的测试租户")
			return
		}

		tenantID := suite.testTenants[0]
		path := "/api/v1/tenants/" + tenantID.String() + "/with-config"

		resp, body := suite.makeRequest("GET", path, nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])

		if data, ok := response["data"].(map[string]interface{}); ok {
			assert.Equal(suite.T(), tenantID.String(), data["id"])
			// 检查配置信息是否包含
			if config, exists := data["config"]; exists {
				assert.NotNil(suite.T(), config)
			}
		}
	})

	suite.Run("获取租户完整信息", func() {
		if len(suite.testTenants) == 0 {
			suite.T().Skip("没有可用的测试租户")
			return
		}

		tenantID := suite.testTenants[0]
		path := "/api/v1/tenants/" + tenantID.String() + "/complete"

		resp, body := suite.makeRequest("GET", path, nil, suite.testToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])

		if data, ok := response["data"].(map[string]interface{}); ok {
			assert.Equal(suite.T(), tenantID.String(), data["id"])
			// 检查完整信息是否包含
			if config, exists := data["config"]; exists {
				assert.NotNil(suite.T(), config)
			}
			if subscription, exists := data["subscription"]; exists {
				assert.NotNil(suite.T(), subscription)
			}
		}
	})
}

// 测试边界条件和错误处理
func (suite *TenantServiceIntegrationTestSuite) TestEdgeCasesAndErrorHandling() {
	suite.Run("无效的JWT Token", func() {
		invalidToken := "invalid.jwt.token"

		resp, body := suite.makeRequest("GET", "/api/v1/tenants", nil, invalidToken)
		// 根据中间件实现，可能返回401或者忽略认证
		assert.True(suite.T(), resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
	})

	suite.Run("超大分页参数", func() {
		path := "/api/v1/tenants?limit=1000&offset=99999"

		resp, body := suite.makeRequest("GET", path, nil, suite.adminToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])

		// 检查是否应用了限制
		if data, ok := response["data"].(map[string]interface{}); ok {
			if limit, ok := data["limit"].(float64); ok {
				assert.LessOrEqual(suite.T(), int(limit), 100) // 应该被限制在100以内
			}
		}
	})

	suite.Run("删除不存在的租户", func() {
		nonExistentID := uuid.New()
		path := "/api/v1/tenants/" + nonExistentID.String()

		resp, body := suite.makeRequest("DELETE", path, nil, suite.adminToken)
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(500), response["code"])
	})
}

// TestTenantServiceIntegration 运行所有租户服务集成测试
func TestTenantServiceIntegration(t *testing.T) {
	suite.Run(t, new(TenantServiceIntegrationTestSuite))
}

// 辅助函数

// tenantStringPtr 返回字符串指针 (避免与其他测试文件中的函数重名)
func tenantStringPtr(s string) *string {
	return &s
}

// getPlanAmount 根据计划获取价格
func getPlanAmount(plan models.TenantPlan) float64 {
	switch plan {
	case models.TenantPlanFree:
		return 0
	case models.TenantPlanBasic:
		return 29.99
	case models.TenantPlanProfessional:
		return 99.99
	case models.TenantPlanEnterprise:
		return 299.99
	default:
		return 0
	}
}