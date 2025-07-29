package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	iamhandlers "github.com/cloud-platform/collaborative-dev/cmd/iam-service/handlers"
	iamservices "github.com/cloud-platform/collaborative-dev/cmd/iam-service/services"
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

// IAMServiceIntegrationTestSuite IAM服务集成测试套件
type IAMServiceIntegrationTestSuite struct {
	suite.Suite
	db             *gorm.DB
	logger         logger.Logger
	router         *gin.Engine
	server         *httptest.Server
	userService    *iamservices.UserService
	jwtService     *auth.JWTService
	authHandler    *iamhandlers.AuthHandler
	testTenantID   uuid.UUID
	testUsers      []uuid.UUID
	testTokens     []string
	testSessions   []uuid.UUID
	testRoles      []uuid.UUID
	testPermissions []uuid.UUID
}

// SetupSuite 初始化测试套件
func (suite *IAMServiceIntegrationTestSuite) SetupSuite() {
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
		"test-secret-key-for-integration-testing",
		time.Hour*1,
		time.Hour*24*7,
	)

	// 初始化用户服务
	userConfig := iamservices.UserServiceConfig{
		PasswordMinLength: 8,
		MaxLoginAttempts:  5,
		LockoutDuration:   time.Minute * 15,
	}
	suite.userService = iamservices.NewUserService(
		&database.PostgresDB{DB: suite.db},
		suite.jwtService,
		userConfig,
	)

	// 初始化认证处理器
	suite.authHandler = iamhandlers.NewAuthHandler(suite.userService, suite.logger)

	// 设置路由
	suite.setupRouter()

	// 启动测试服务器
	suite.server = httptest.NewServer(suite.router)

	// 设置测试数据
	suite.testTenantID = uuid.New()
	suite.testUsers = []uuid.UUID{}
	suite.testTokens = []string{}
	suite.testSessions = []uuid.UUID{}
	suite.testRoles = []uuid.UUID{}
	suite.testPermissions = []uuid.UUID{}

	// 创建测试基础数据
	suite.createTestData()

	suite.logger.Info("IAM服务集成测试套件初始化完成", 
		"tenant_id", suite.testTenantID,
		"server_url", suite.server.URL)
}

// setupTestDatabase 设置测试数据库
func (suite *IAMServiceIntegrationTestSuite) setupTestDatabase() {
	config := database.Config{
		Host:     "localhost",
		Port:     5432,
		Name:     "test_iam_service",
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
		&models.User{},
		&models.Role{},
		&models.Permission{},
		&models.UserRole{},
		&models.RolePermission{},
		&models.UserSession{},
	)
	if err != nil {
		suite.T().Fatalf("数据库迁移失败: %v", err)
	}
}

// setupRouter 设置路由
func (suite *IAMServiceIntegrationTestSuite) setupRouter() {
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
			}
		}
		c.Next()
	})

	// IAM API路由
	api := suite.router.Group("/api/v1")
	{
		// 认证相关
		auth := api.Group("/auth")
		{
			auth.POST("/login", suite.authHandler.Login)
			auth.POST("/register", suite.authHandler.Register)
			auth.POST("/refresh", suite.authHandler.RefreshToken)
			auth.POST("/logout", suite.authHandler.Logout)
			auth.GET("/validate", suite.authHandler.ValidateToken)
			auth.GET("/profile", suite.authHandler.GetProfile)
			auth.PUT("/profile", suite.authHandler.UpdateProfile)
			auth.POST("/change-password", suite.authHandler.ChangePassword)
		}
	}
}

// createTestData 创建测试基础数据
func (suite *IAMServiceIntegrationTestSuite) createTestData() {
	ctx := context.Background()

	// 创建测试权限
	permissions := []models.Permission{
		{
			ID:          uuid.New(),
			Name:        "read_users",
			Description: "读取用户信息",
			Resource:    "users",
			Action:      "read",
		},
		{
			ID:          uuid.New(),
			Name:        "write_users",
			Description: "写入用户信息",
			Resource:    "users",
			Action:      "write",
		},
		{
			ID:          uuid.New(),
			Name:        "admin_all",
			Description: "管理员权限",
			Resource:    "*",
			Action:      "*",
		},
	}

	for _, perm := range permissions {
		suite.db.WithContext(ctx).Create(&perm)
		suite.testPermissions = append(suite.testPermissions, perm.ID)
	}

	// 创建测试角色
	roles := []models.Role{
		{
			ID:          uuid.New(),
			TenantID:    suite.testTenantID,
			Name:        "user",
			Description: "普通用户",
			IsSystem:    true,
		},
		{
			ID:          uuid.New(),
			TenantID:    suite.testTenantID,
			Name:        "admin",
			Description: "管理员",
			IsSystem:    true,
		},
	}

	for i, role := range roles {
		suite.db.WithContext(ctx).Create(&role)
		suite.testRoles = append(suite.testRoles, role.ID)

		// 为角色分配权限
		if i == 0 { // user角色
			suite.db.WithContext(ctx).Create(&models.RolePermission{
				RoleID:       role.ID,
				PermissionID: permissions[0].ID, // read_users
			})
		} else { // admin角色
			for _, perm := range permissions {
				suite.db.WithContext(ctx).Create(&models.RolePermission{
					RoleID:       role.ID,
					PermissionID: perm.ID,
				})
			}
		}
	}
}

// TearDownSuite 清理测试套件
func (suite *IAMServiceIntegrationTestSuite) TearDownSuite() {
	if suite.server != nil {
		suite.server.Close()
	}
	suite.logger.Info("IAM服务集成测试套件清理完成")
}

// makeRequest 发送HTTP请求
func (suite *IAMServiceIntegrationTestSuite) makeRequest(method, path string, body interface{}, token string) (*http.Response, []byte) {
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

// 测试用户认证功能
func (suite *IAMServiceIntegrationTestSuite) TestAuthentication() {
	// 测试用户注册
	suite.Run("用户注册", func() {
		registerReq := iamservices.RegisterRequest{
			TenantID:  suite.testTenantID,
			Email:     "test@example.com",
			Username:  "testuser",
			Password:  "password123",
			FirstName: "Test",
			LastName:  "User",
		}

		resp, body := suite.makeRequest("POST", "/api/v1/auth/register", registerReq, "")
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(201), response["code"])
		assert.Contains(suite.T(), response, "data")

		// 保存用户ID
		if userData, ok := response["data"].(map[string]interface{}); ok {
			if userID, ok := userData["id"].(string); ok {
				if uid, err := uuid.Parse(userID); err == nil {
					suite.testUsers = append(suite.testUsers, uid)
				}
			}
		}
	})

	// 测试用户登录
	suite.Run("用户登录", func() {
		loginReq := iamservices.LoginRequest{
			Email:    "test@example.com",
			Password: "password123",
		}

		resp, body := suite.makeRequest("POST", "/api/v1/auth/login", loginReq, "")
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])

		// 提取token
		if data, ok := response["data"].(map[string]interface{}); ok {
			if tokens, ok := data["tokens"].(map[string]interface{}); ok {
				if accessToken, ok := tokens["access_token"].(string); ok {
					suite.testTokens = append(suite.testTokens, accessToken)
				}
			}
		}
	})

	// 测试错误的登录凭据
	suite.Run("错误的登录凭据", func() {
		loginReq := iamservices.LoginRequest{
			Email:    "test@example.com",
			Password: "wrongpassword",
		}

		resp, body := suite.makeRequest("POST", "/api/v1/auth/login", loginReq, "")
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(401), response["code"])
	})

	// 测试令牌验证
	suite.Run("令牌验证", func() {
		if len(suite.testTokens) == 0 {
			suite.T().Skip("没有可用的测试令牌")
			return
		}

		token := suite.testTokens[0]
		resp, body := suite.makeRequest("GET", "/api/v1/auth/validate", nil, token)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])
	})

	// 测试无效令牌
	suite.Run("无效令牌验证", func() {
		invalidToken := "invalid.jwt.token"
		resp, body := suite.makeRequest("GET", "/api/v1/auth/validate", nil, invalidToken)
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(401), response["code"])
	})
}

// 测试用户授权功能
func (suite *IAMServiceIntegrationTestSuite) TestAuthorization() {
	suite.Run("获取用户资料", func() {
		if len(suite.testTokens) == 0 {
			suite.T().Skip("没有可用的测试令牌")
			return
		}

		token := suite.testTokens[0]
		resp, body := suite.makeRequest("GET", "/api/v1/auth/profile", nil, token)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])
		assert.Contains(suite.T(), response, "data")
	})

	suite.Run("无令牌访问受保护资源", func() {
		resp, body := suite.makeRequest("GET", "/api/v1/auth/profile", nil, "")
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(401), response["code"])
	})
}

// 测试用户管理功能
func (suite *IAMServiceIntegrationTestSuite) TestUserManagement() {
	// 测试更新用户资料
	suite.Run("更新用户资料", func() {
		if len(suite.testTokens) == 0 {
			suite.T().Skip("没有可用的测试令牌")
			return
		}

		updateReq := iamservices.UpdateUserRequest{
			FirstName: "Updated",
			LastName:  "Name",
			Phone:     "+1234567890",
		}

		token := suite.testTokens[0]
		resp, body := suite.makeRequest("PUT", "/api/v1/auth/profile", updateReq, token)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])

		// 验证更新结果
		if data, ok := response["data"].(map[string]interface{}); ok {
			assert.Equal(suite.T(), "Updated", data["first_name"])
			assert.Equal(suite.T(), "Name", data["last_name"])
		}
	})

	// 测试修改密码
	suite.Run("修改密码", func() {
		if len(suite.testTokens) == 0 {
			suite.T().Skip("没有可用的测试令牌")
			return
		}

		changePasswordReq := iamservices.ChangePasswordRequest{
			CurrentPassword: "password123",
			NewPassword:     "newpassword456",
		}

		token := suite.testTokens[0]
		resp, body := suite.makeRequest("POST", "/api/v1/auth/change-password", changePasswordReq, token)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])
	})

	// 测试用错误的当前密码修改密码
	suite.Run("错误的当前密码", func() {
		// 创建新用户用于此测试
		registerReq := iamservices.RegisterRequest{
			TenantID:  suite.testTenantID,
			Email:     "test2@example.com",
			Username:  "testuser2",
			Password:  "password123",
			FirstName: "Test2",
			LastName:  "User2",
		}

		// 注册用户
		resp, _ := suite.makeRequest("POST", "/api/v1/auth/register", registerReq, "")
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

		// 登录获取令牌
		loginReq := iamservices.LoginRequest{
			Email:    "test2@example.com",
			Password: "password123",
		}

		resp, body := suite.makeRequest("POST", "/api/v1/auth/login", loginReq, "")
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var loginResponse map[string]interface{}
		err := json.Unmarshal(body, &loginResponse)
		assert.NoError(suite.T(), err)

		var token string
		if data, ok := loginResponse["data"].(map[string]interface{}); ok {
			if tokens, ok := data["tokens"].(map[string]interface{}); ok {
				if accessToken, ok := tokens["access_token"].(string); ok {
					token = accessToken
				}
			}
		}

		// 尝试用错误的当前密码修改密码
		changePasswordReq := iamservices.ChangePasswordRequest{
			CurrentPassword: "wrongpassword",
			NewPassword:     "newpassword456",
		}

		resp, body = suite.makeRequest("POST", "/api/v1/auth/change-password", changePasswordReq, token)
		assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(400), response["code"])
	})
}

// 测试会话管理功能
func (suite *IAMServiceIntegrationTestSuite) TestSessionManagement() {
	var refreshToken string

	// 先登录获取refresh token
	suite.Run("登录获取会话", func() {
		loginReq := iamservices.LoginRequest{
			Email:    "test@example.com",
			Password: "newpassword456", // 使用修改后的密码
		}

		resp, body := suite.makeRequest("POST", "/api/v1/auth/login", loginReq, "")
		
		// 如果使用新密码失败，尝试原密码
		if resp.StatusCode != http.StatusOK {
			loginReq.Password = "password123"
			resp, body = suite.makeRequest("POST", "/api/v1/auth/login", loginReq, "")
		}

		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)

		if data, ok := response["data"].(map[string]interface{}); ok {
			if tokens, ok := data["tokens"].(map[string]interface{}); ok {
				if rt, ok := tokens["refresh_token"].(string); ok {
					refreshToken = rt
				}
			}
		}
	})

	// 测试令牌刷新
	suite.Run("刷新令牌", func() {
		if refreshToken == "" {
			suite.T().Skip("没有可用的刷新令牌")
			return
		}

		refreshReq := map[string]string{
			"refresh_token": refreshToken,
		}

		resp, body := suite.makeRequest("POST", "/api/v1/auth/refresh", refreshReq, "")
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])
	})

	// 测试用户登出
	suite.Run("用户登出", func() {
		if refreshToken == "" {
			suite.T().Skip("没有可用的刷新令牌")
			return
		}

		logoutReq := map[string]string{
			"refresh_token": refreshToken,
		}

		resp, body := suite.makeRequest("POST", "/api/v1/auth/logout", logoutReq, "")
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(200), response["code"])
	})

	// 测试使用已登出的refresh token刷新令牌
	suite.Run("使用已登出的令牌刷新", func() {
		if refreshToken == "" {
			suite.T().Skip("没有可用的刷新令牌")
			return
		}

		refreshReq := map[string]string{
			"refresh_token": refreshToken,
		}

		resp, body := suite.makeRequest("POST", "/api/v1/auth/refresh", refreshReq, "")
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(401), response["code"])
	})
}

// 测试边界条件和错误处理
func (suite *IAMServiceIntegrationTestSuite) TestEdgeCasesAndErrorHandling() {
	// 测试重复注册
	suite.Run("重复注册相同邮箱", func() {
		registerReq := iamservices.RegisterRequest{
			TenantID:  suite.testTenantID,
			Email:     "duplicate@example.com",
			Username:  "duplicate1",
			Password:  "password123",
			FirstName: "Duplicate",
			LastName:  "User",
		}

		// 第一次注册
		resp, _ := suite.makeRequest("POST", "/api/v1/auth/register", registerReq, "")
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

		// 第二次注册相同邮箱
		registerReq.Username = "duplicate2" // 改变用户名但保持邮箱相同
		resp, body := suite.makeRequest("POST", "/api/v1/auth/register", registerReq, "")
		assert.Equal(suite.T(), http.StatusConflict, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(409), response["code"])
	})

	// 测试弱密码
	suite.Run("弱密码注册", func() {
		registerReq := iamservices.RegisterRequest{
			TenantID:  suite.testTenantID,
			Email:     "weak@example.com",
			Username:  "weakuser",
			Password:  "123", // 太短的密码
			FirstName: "Weak",
			LastName:  "User",
		}

		resp, body := suite.makeRequest("POST", "/api/v1/auth/register", registerReq, "")
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(500), response["code"])
	})

	// 测试无效的请求参数
	suite.Run("无效的JSON请求", func() {
		req, err := http.NewRequest("POST", suite.server.URL+"/api/v1/auth/login", 
			bytes.NewBuffer([]byte("invalid json")))
		suite.NoError(err)

		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		suite.NoError(err)

		assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)
		resp.Body.Close()
	})

	// 测试缺少必需字段
	suite.Run("缺少必需字段", func() {
		incompleteReq := map[string]string{
			"email": "incomplete@example.com",
			// 缺少password字段
		}

		resp, body := suite.makeRequest("POST", "/api/v1/auth/login", incompleteReq, "")
		assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(400), response["code"])
	})
}

// TestIAMServiceIntegration 运行所有IAM服务集成测试
func TestIAMServiceIntegration(t *testing.T) {
	suite.Run(t, new(IAMServiceIntegrationTestSuite))
}

