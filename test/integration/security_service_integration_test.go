package integration

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
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/cloud-platform/collaborative-dev/cmd/iam-service/services"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/handler"
	"github.com/cloud-platform/collaborative-dev/shared/database"
	"github.com/cloud-platform/collaborative-dev/shared/middleware"
	"github.com/cloud-platform/collaborative-dev/shared/models"
)

// SecurityServiceIntegrationTestSuite 安全服务集成测试套件
type SecurityServiceIntegrationTestSuite struct {
	suite.Suite
	db                      *database.PostgresDB
	securityAuditService    *services.SecurityAuditService
	rateLimitMiddleware     *middleware.RateLimitMiddleware
	ipFilterMiddleware      *middleware.IPFilterMiddleware
	securityHandler         *handler.SecurityHandler
	router                  *gin.Engine
	logger                  *zap.Logger
	testTenantID           uuid.UUID
	testUserID             uuid.UUID
	testContext            context.Context
}

// SetupSuite 测试套件初始化
func (suite *SecurityServiceIntegrationTestSuite) SetupSuite() {
	var err error

	// 创建测试数据库连接
	suite.db, err = database.NewPostgresDB("host=localhost port=5432 user=postgres password=123456 dbname=test_collaborative_dev sslmode=disable")
	require.NoError(suite.T(), err, "数据库连接失败")

	// 自动迁移测试表
	err = suite.db.DB.AutoMigrate(
		&models.SecurityAuditLog{},
		&models.LoginAttempt{},
		&models.MFAAttempt{},
		&models.User{},
	)
	require.NoError(suite.T(), err, "数据库迁移失败")

	// 创建测试logger
	suite.logger, _ = zap.NewDevelopment()

	// 初始化服务和中间件
	suite.securityAuditService = services.NewSecurityAuditService(suite.db)

	// 配置限流中间件
	rateLimitConfig := &middleware.RateLimitConfig{
		GlobalRPS:        1000,
		GlobalBurst:      100,
		PerIPRPS:         10,
		PerIPBurst:       5,
		CleanupInterval:  1 * time.Minute,
		AutoBanEnabled:   true,
		AutoBanThreshold: 3,
		AutoBanWindow:    1 * time.Minute,
		AutoBanDuration:  5 * time.Minute,
		EndpointLimits: map[string]middleware.EndpointLimit{
			"POST /api/v1/test/login": {
				RPS:   2,
				Burst: 1,
			},
		},
	}
	suite.rateLimitMiddleware = middleware.NewRateLimitMiddleware(rateLimitConfig, suite.logger)

	// 创建IP过滤中间件（假设存在）
	suite.ipFilterMiddleware = &middleware.IPFilterMiddleware{} // 简化实现

	// 创建安全处理器
	suite.securityHandler = handler.NewSecurityHandler(
		suite.rateLimitMiddleware,
		suite.ipFilterMiddleware,
		suite.logger,
	)

	// 设置测试上下文和ID
	suite.testContext = context.Background()
	suite.testTenantID = uuid.New()
	suite.testUserID = uuid.New()

	// 设置Gin为测试模式
	gin.SetMode(gin.TestMode)
	suite.setupRouter()
}

// TearDownSuite 测试套件清理
func (suite *SecurityServiceIntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		// 清理测试数据
		suite.db.DB.Exec("TRUNCATE TABLE security_audit_logs, login_attempts, mfa_attempts CASCADE")
		suite.db.Close()
	}
}

// SetupTest 每个测试前的设置
func (suite *SecurityServiceIntegrationTestSuite) SetupTest() {
	// 清理测试数据
	suite.db.DB.Exec("DELETE FROM security_audit_logs WHERE tenant_id = ?", suite.testTenantID)
	suite.db.DB.Exec("DELETE FROM login_attempts WHERE tenant_id = ?", suite.testTenantID)
}

// setupRouter 设置测试路由
func (suite *SecurityServiceIntegrationTestSuite) setupRouter() {
	suite.router = gin.New()
	
	// 添加限流中间件
	suite.router.Use(suite.rateLimitMiddleware.Handler())

	api := suite.router.Group("/api/v1")
	{
		// 安全管理路由
		security := api.Group("/security")
		{
			security.GET("/status", suite.securityHandler.GetSecurityStatus)
			security.POST("/blacklist", suite.securityHandler.AddToBlacklist)
			security.DELETE("/blacklist/:ip", suite.securityHandler.RemoveFromBlacklist)
			security.POST("/whitelist", suite.securityHandler.AddToWhitelist)
			security.DELETE("/whitelist/:ip", suite.securityHandler.RemoveFromWhitelist)
			security.GET("/violations", suite.securityHandler.GetViolationStats)
			security.POST("/ban", suite.securityHandler.BanIP)
			security.DELETE("/ban/:ip", suite.securityHandler.UnbanIP)
			security.GET("/metrics", suite.securityHandler.GetSecurityMetrics)
		}

		// 测试端点（用于限流测试）
		test := api.Group("/test")
		{
			test.POST("/login", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"success": true})
			})
			test.GET("/public", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "public endpoint"})
			})
		}
	}
}

// TestSecurityAuditLogging 测试安全审计日志记录
func (suite *SecurityServiceIntegrationTestSuite) TestSecurityAuditLogging() {
	suite.T().Run("记录成功登录审计事件", func(t *testing.T) {
		req := &services.AuditLogRequest{
			TenantID:  suite.testTenantID,
			UserID:    &suite.testUserID,
			IPAddress: "192.168.1.100",
			UserAgent: "TestAgent/1.0",
			Action:    "login_success",
			Resource:  "auth",
			Success:   true,
			Metadata: map[string]interface{}{
				"login_method": "password",
				"session_id":   "test-session-123",
			},
		}

		err := suite.securityAuditService.LogAuditEvent(suite.testContext, req)
		assert.NoError(t, err)

		// 验证审计日志是否正确记录
		var auditLog models.SecurityAuditLog
		err = suite.db.DB.Where("tenant_id = ? AND action = ?", suite.testTenantID, "login_success").First(&auditLog).Error
		assert.NoError(t, err)
		assert.Equal(t, req.IPAddress, auditLog.IPAddress)
		assert.Equal(t, req.Action, auditLog.Action)
		assert.True(t, auditLog.Success)
	})

	suite.T().Run("记录失败登录审计事件", func(t *testing.T) {
		req := &services.AuditLogRequest{
			TenantID:     suite.testTenantID,
			IPAddress:    "192.168.1.200",
			UserAgent:    "AttackerAgent/1.0",
			Action:       "login_failed",
			Resource:     "auth",
			Success:      false,
			ErrorCode:    "INVALID_CREDENTIALS",
			ErrorMessage: "用户名或密码错误",
			Metadata: map[string]interface{}{
				"attempted_email": "hacker@evil.com",
				"fail_count":      1,
			},
		}

		err := suite.securityAuditService.LogAuditEvent(suite.testContext, req)
		assert.NoError(t, err)

		// 验证失败审计日志
		var auditLog models.SecurityAuditLog
		err = suite.db.DB.Where("tenant_id = ? AND action = ? AND success = false", 
			suite.testTenantID, "login_failed").First(&auditLog).Error
		assert.NoError(t, err)
		assert.Equal(t, req.ErrorCode, auditLog.ErrorCode)
		assert.False(t, auditLog.Success)
	})

	suite.T().Run("批量记录审计事件", func(t *testing.T) {
		// 并发记录多个审计事件
		events := []services.AuditLogRequest{
			{
				TenantID:  suite.testTenantID,
				UserID:    &suite.testUserID,
				IPAddress: "192.168.1.101",
				Action:    "project_create",
				Resource:  "project",
				Success:   true,
			},
			{
				TenantID:  suite.testTenantID,
				UserID:    &suite.testUserID,
				IPAddress: "192.168.1.101",
				Action:    "file_upload",
				Resource:  "file",
				Success:   true,
			},
			{
				TenantID:  suite.testTenantID,
				IPAddress: "192.168.1.102",
				Action:    "unauthorized_access",
				Resource:  "admin",
				Success:   false,
			},
		}

		for i, event := range events {
			go func(e services.AuditLogRequest, index int) {
				e.Metadata = map[string]interface{}{"batch_index": index}
				suite.securityAuditService.LogAuditEvent(suite.testContext, &e)
			}(event, i)
		}

		// 等待所有事件记录完成
		time.Sleep(100 * time.Millisecond)

		// 验证所有事件都被记录
		var count int64
		suite.db.DB.Model(&models.SecurityAuditLog{}).Where("tenant_id = ?", suite.testTenantID).Count(&count)
		assert.GreaterOrEqual(t, count, int64(3))
	})
}

// TestLoginAttemptTracking 测试登录尝试追踪
func (suite *SecurityServiceIntegrationTestSuite) TestLoginAttemptTracking() {
	suite.T().Run("记录成功登录尝试", func(t *testing.T) {
		req := &services.LoginAttemptRequest{
			TenantID:    suite.testTenantID,
			UserID:      &suite.testUserID,
			Email:       "user@example.com",
			IPAddress:   "192.168.1.100",
			UserAgent:   "Mozilla/5.0...",
			Success:     true,
			MFARequired: true,
			MFASuccess:  &[]bool{true}[0],
		}

		err := suite.securityAuditService.LogLoginAttempt(suite.testContext, req)
		assert.NoError(t, err)

		// 验证登录尝试记录
		var attempt models.LoginAttempt
		err = suite.db.DB.Where("tenant_id = ? AND email = ? AND success = true", 
			suite.testTenantID, req.Email).First(&attempt).Error
		assert.NoError(t, err)
		assert.True(t, attempt.MFARequired)
		assert.True(t, *attempt.MFASuccess)
	})

	suite.T().Run("记录失败登录尝试序列", func(t *testing.T) {
		attackerIP := "192.168.1.200"
		targetEmail := "victim@example.com"

		// 模拟暴力破解攻击
		for i := 0; i < 5; i++ {
			req := &services.LoginAttemptRequest{
				TenantID:   suite.testTenantID,
				Email:      targetEmail,
				IPAddress:  attackerIP,
				UserAgent:  "AttackerBot/1.0",
				Success:    false,
				FailReason: fmt.Sprintf("密码错误 - 尝试 %d", i+1),
			}

			err := suite.securityAuditService.LogLoginAttempt(suite.testContext, req)
			assert.NoError(t, err)
		}

		// 验证失败尝试次数
		var failedCount int64
		suite.db.DB.Model(&models.LoginAttempt{}).
			Where("tenant_id = ? AND ip_address = ? AND success = false", 
				suite.testTenantID, attackerIP).Count(&failedCount)
		assert.Equal(t, int64(5), failedCount)
	})

	suite.T().Run("MFA失败场景", func(t *testing.T) {
		req := &services.LoginAttemptRequest{
			TenantID:    suite.testTenantID,
			UserID:      &suite.testUserID,
			Email:       "user@example.com",
			IPAddress:   "192.168.1.100",
			Success:     false,
			FailReason:  "MFA验证失败",
			MFARequired: true,
			MFASuccess:  &[]bool{false}[0],
		}

		err := suite.securityAuditService.LogLoginAttempt(suite.testContext, req)
		assert.NoError(t, err)

		// 验证MFA失败记录
		var attempt models.LoginAttempt
		err = suite.db.DB.Where("tenant_id = ? AND fail_reason LIKE ?", 
			suite.testTenantID, "%MFA%").First(&attempt).Error
		assert.NoError(t, err)
		assert.False(t, *attempt.MFASuccess)
	})
}

// TestRateLimitingFunctionality 测试限流功能
func (suite *SecurityServiceIntegrationTestSuite) TestRateLimitingFunctionality() {
	suite.T().Run("全局限流测试", func(t *testing.T) {
		// 正常请求应该通过
		for i := 0; i < 5; i++ {
			req, _ := http.NewRequest("GET", "/api/v1/test/public", nil)
			req.RemoteAddr = fmt.Sprintf("192.168.1.%d:12345", 100+i)
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}
	})

	suite.T().Run("IP级别限流测试", func(t *testing.T) {
		attackerIP := "192.168.1.250"
		
		// 快速发送大量请求，应该触发限流
		successCount := 0
		rateLimitedCount := 0

		for i := 0; i < 20; i++ {
			req, _ := http.NewRequest("GET", "/api/v1/test/public", nil)
			req.RemoteAddr = attackerIP + ":12345"
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				successCount++
			} else if w.Code == http.StatusTooManyRequests {
				rateLimitedCount++
			}

			// 稍微间隔避免所有请求被阻止
			time.Sleep(10 * time.Millisecond)
		}

		assert.Greater(t, rateLimitedCount, 0, "应该有请求被限流")
		assert.Greater(t, successCount, 0, "应该有请求通过")
	})

	suite.T().Run("端点特定限流测试", func(t *testing.T) {
		loginIP := "192.168.1.300"
		
		// 对登录端点进行快速请求
		successCount := 0
		rateLimitedCount := 0

		for i := 0; i < 10; i++ {
			reqBody := `{"email":"test@example.com","password":"password"}`
			req, _ := http.NewRequest("POST", "/api/v1/test/login", bytes.NewBufferString(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.RemoteAddr = loginIP + ":12345"
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				successCount++
			} else if w.Code == http.StatusTooManyRequests {
				rateLimitedCount++
			}

			time.Sleep(100 * time.Millisecond) // 登录端点限制更严格
		}

		assert.Greater(t, rateLimitedCount, 0, "登录端点应该有请求被限流")
	})

	suite.T().Run("自动封禁测试", func(t *testing.T) {
		banTestIP := "192.168.1.400"
		
		// 快速触发多次限流，应该导致自动封禁
		for i := 0; i < 10; i++ {
			req, _ := http.NewRequest("GET", "/api/v1/test/public", nil)
			req.RemoteAddr = banTestIP + ":12345"
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)
			// 不等待，快速发送请求触发自动封禁
		}

		// 等待自动封禁逻辑处理
		time.Sleep(200 * time.Millisecond)

		// 再次请求，应该被封禁
		req, _ := http.NewRequest("GET", "/api/v1/test/public", nil)
		req.RemoteAddr = banTestIP + ":12345"
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		// 检查是否被封禁（应该返回403或429）
		assert.Contains(t, []int{http.StatusForbidden, http.StatusTooManyRequests}, w.Code)
	})
}

// TestSecurityHandlerAPIs 测试安全处理器API
func (suite *SecurityServiceIntegrationTestSuite) TestSecurityHandlerAPIs() {
	suite.T().Run("获取安全状态", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/security/status", nil)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "blacklist")
		assert.Contains(t, response, "whitelist")
		assert.Contains(t, response, "violations")
	})

	suite.T().Run("手动添加IP到黑名单", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"ip":       "192.168.1.500",
			"reason":   "手动测试封禁",
			"duration": "1h",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/v1/security/blacklist", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// 验证IP确实被封禁
		testReq, _ := http.NewRequest("GET", "/api/v1/test/public", nil)
		testReq.RemoteAddr = "192.168.1.500:12345"
		testW := httptest.NewRecorder()
		suite.router.ServeHTTP(testW, testReq)
		assert.Equal(t, http.StatusForbidden, testW.Code)
	})

	suite.T().Run("从黑名单移除IP", func(t *testing.T) {
		// 先添加到黑名单
		reqBody := map[string]interface{}{
			"ip":     "192.168.1.600",
			"reason": "测试解封",
		}
		body, _ := json.Marshal(reqBody)
		addReq, _ := http.NewRequest("POST", "/api/v1/security/blacklist", bytes.NewBuffer(body))
		addReq.Header.Set("Content-Type", "application/json")
		addW := httptest.NewRecorder()
		suite.router.ServeHTTP(addW, addReq)
		assert.Equal(t, http.StatusOK, addW.Code)

		// 然后移除
		removeReq, _ := http.NewRequest("DELETE", "/api/v1/security/blacklist/192.168.1.600", nil)
		removeW := httptest.NewRecorder()
		suite.router.ServeHTTP(removeW, removeReq)
		assert.Equal(t, http.StatusOK, removeW.Code)

		// 验证IP可以正常访问
		testReq, _ := http.NewRequest("GET", "/api/v1/test/public", nil)
		testReq.RemoteAddr = "192.168.1.600:12345"
		testW := httptest.NewRecorder()
		suite.router.ServeHTTP(testW, testReq)
		assert.Equal(t, http.StatusOK, testW.Code)
	})

	suite.T().Run("获取违规统计", func(t *testing.T) {
		// 先触发一些违规
		for i := 0; i < 3; i++ {
			req, _ := http.NewRequest("GET", "/api/v1/test/public", nil)
			req.RemoteAddr = "192.168.1.700:12345"
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)
		}

		// 获取违规统计
		req, _ := http.NewRequest("GET", "/api/v1/security/violations", nil)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "violations")
		assert.Contains(t, response, "total")
	})

	suite.T().Run("获取安全指标", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/security/metrics", nil)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "metrics")
		
		metrics := response["metrics"].(map[string]interface{})
		assert.Contains(t, metrics, "blacklist")
		assert.Contains(t, metrics, "violations")
	})
}

// TestSecurityMetricsAndAnalytics 测试安全指标和分析
func (suite *SecurityServiceIntegrationTestSuite) TestSecurityMetricsAndAnalytics() {
	suite.T().Run("生成安全指标报告", func(t *testing.T) {
		// 创建测试数据：多种类型的登录尝试
		testData := []services.LoginAttemptRequest{
			{
				TenantID:   suite.testTenantID,
				UserID:     &suite.testUserID,
				Email:      "user1@example.com",
				IPAddress:  "192.168.1.10",
				Success:    true,
				MFARequired: true,
				MFASuccess: &[]bool{true}[0],
			},
			{
				TenantID:   suite.testTenantID,
				Email:      "user2@example.com",
				IPAddress:  "192.168.1.20",
				Success:    false,
				FailReason: "密码错误",
			},
			{
				TenantID:   suite.testTenantID,
				Email:      "user3@example.com",
				IPAddress:  "192.168.1.30",
				Success:    false,
				FailReason: "账户锁定",
			},
		}

		for _, data := range testData {
			err := suite.securityAuditService.LogLoginAttempt(suite.testContext, &data)
			assert.NoError(t, err)
		}

		// 获取安全指标
		startTime := time.Now().Add(-1 * time.Hour)
		endTime := time.Now().Add(1 * time.Hour)
		
		metrics, err := suite.securityAuditService.GetSecurityMetrics(
			suite.testContext, suite.testTenantID, startTime, endTime)
		assert.NoError(t, err)
		assert.NotNil(t, metrics)

		// 验证指标数据
		assert.Equal(t, int64(3), metrics.TotalLoginAttempts)
		assert.Equal(t, int64(1), metrics.SuccessfulLogins)
		assert.Equal(t, int64(2), metrics.FailedLogins)
		assert.InDelta(t, 33.33, metrics.SuccessRate, 0.1)
		assert.Equal(t, int64(1), metrics.MFAUsage)
	})

	suite.T().Run("可疑活动检测", func(t *testing.T) {
		// 模拟暴力破解攻击
		attackerIP := "192.168.1.666"
		for i := 0; i < 12; i++ {
			req := &services.LoginAttemptRequest{
				TenantID:   suite.testTenantID,
				Email:      "victim@example.com",
				IPAddress:  attackerIP,
				Success:    false,
				FailReason: "暴力破解尝试",
			}
			suite.securityAuditService.LogLoginAttempt(suite.testContext, req)
		}

		// 模拟夜间异常登录
		nightTime := time.Now().Add(-2 * time.Hour) // 假设现在是深夜
		for i := 0; i < 60; i++ {
			req := &services.LoginAttemptRequest{
				TenantID:  suite.testTenantID,
				Email:     fmt.Sprintf("night_user_%d@example.com", i),
				IPAddress: fmt.Sprintf("192.168.2.%d", i),
				Success:   true,
			}
			// 手动设置创建时间为夜间
			suite.securityAuditService.LogLoginAttempt(suite.testContext, req)
			
			// 直接更新数据库记录时间
			suite.db.DB.Model(&models.LoginAttempt{}).
				Where("email = ?", req.Email).
				Update("created_at", nightTime.Add(time.Duration(i)*time.Minute))
		}

		// 获取安全指标，包含可疑活动
		startTime := time.Now().Add(-24 * time.Hour)
		endTime := time.Now().Add(1 * time.Hour)
		
		metrics, err := suite.securityAuditService.GetSecurityMetrics(
			suite.testContext, suite.testTenantID, startTime, endTime)
		assert.NoError(t, err)
		
		// 验证可疑活动检测
		assert.Greater(t, len(metrics.SuspiciousActivities), 0, "应该检测到可疑活动")
		
		// 查找暴力破解活动
		foundBruteForce := false
		for _, activity := range metrics.SuspiciousActivities {
			if activity.Type == "brute_force" && activity.IPAddress == attackerIP {
				foundBruteForce = true
				assert.GreaterOrEqual(t, activity.Count, int64(10))
				break
			}
		}
		assert.True(t, foundBruteForce, "应该检测到暴力破解攻击")
	})
}

// TestConcurrentSecurityOperations 测试并发安全操作
func (suite *SecurityServiceIntegrationTestSuite) TestConcurrentSecurityOperations() {
	suite.T().Run("并发限流处理", func(t *testing.T) {
		const numGoroutines = 50
		const requestsPerGoroutine = 10
		
		results := make(chan int, numGoroutines*requestsPerGoroutine)
		
		// 启动多个goroutine同时发送请求
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < requestsPerGoroutine; j++ {
					req, _ := http.NewRequest("GET", "/api/v1/test/public", nil)
					req.RemoteAddr = fmt.Sprintf("192.168.100.%d:12345", id)
					w := httptest.NewRecorder()
					suite.router.ServeHTTP(w, req)
					results <- w.Code
					
					time.Sleep(10 * time.Millisecond)
				}
			}(i)
		}

		// 收集结果
		successCount := 0
		rateLimitedCount := 0
		for i := 0; i < numGoroutines*requestsPerGoroutine; i++ {
			code := <-results
			if code == http.StatusOK {
				successCount++
			} else if code == http.StatusTooManyRequests {
				rateLimitedCount++
			}
		}

		assert.Greater(t, successCount, 0, "应该有成功的请求")
		assert.Greater(t, rateLimitedCount, 0, "应该有被限流的请求")
		assert.Equal(t, numGoroutines*requestsPerGoroutine, successCount+rateLimitedCount)
	})

	suite.T().Run("并发审计日志记录", func(t *testing.T) {
		const numGoroutines = 20
		const eventsPerGoroutine = 5
		
		// 并发记录审计事件
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < eventsPerGoroutine; j++ {
					req := &services.AuditLogRequest{
						TenantID:  suite.testTenantID,
						UserID:    &suite.testUserID,
						IPAddress: fmt.Sprintf("192.168.200.%d", id),
						Action:    fmt.Sprintf("concurrent_action_%d_%d", id, j),
						Resource:  "test",
						Success:   true,
						Metadata: map[string]interface{}{
							"goroutine_id": id,
							"event_id":     j,
						},
					}
					suite.securityAuditService.LogAuditEvent(suite.testContext, req)
				}
			}(i)
		}

		// 等待所有goroutine完成
		time.Sleep(500 * time.Millisecond)

		// 验证所有事件都被记录
		var count int64
		suite.db.DB.Model(&models.SecurityAuditLog{}).
			Where("tenant_id = ? AND action LIKE ?", suite.testTenantID, "concurrent_action_%").
			Count(&count)
		assert.Equal(t, int64(numGoroutines*eventsPerGoroutine), count)
	})
}

// TestSecurityDataRetention 测试安全数据保留策略
func (suite *SecurityServiceIntegrationTestSuite) TestSecurityDataRetention() {
	suite.T().Run("清理过期安全日志", func(t *testing.T) {
		// 创建一些旧的审计日志
		oldTime := time.Now().AddDate(0, 0, -100) // 100天前
		recentTime := time.Now().AddDate(0, 0, -10) // 10天前

		// 插入旧日志
		oldAuditLog := models.SecurityAuditLog{
			TenantID:  suite.testTenantID,
			UserID:    &suite.testUserID,
			IPAddress: "192.168.1.1",
			Action:    "old_action",
			Resource:  "test",
			Success:   true,
			CreatedAt: oldTime,
		}
		suite.db.DB.Create(&oldAuditLog)

		// 插入最近日志
		recentAuditLog := models.SecurityAuditLog{
			TenantID:  suite.testTenantID,
			UserID:    &suite.testUserID,
			IPAddress: "192.168.1.2",
			Action:    "recent_action",
			Resource:  "test",
			Success:   true,
			CreatedAt: recentTime,
		}
		suite.db.DB.Create(&recentAuditLog)

		// 执行清理（保留30天）
		err := suite.securityAuditService.CleanupOldLogs(suite.testContext, 30)
		assert.NoError(t, err)

		// 验证旧日志被删除，最近日志保留
		var oldCount, recentCount int64
		suite.db.DB.Model(&models.SecurityAuditLog{}).
			Where("tenant_id = ? AND action = ?", suite.testTenantID, "old_action").
			Count(&oldCount)
		suite.db.DB.Model(&models.SecurityAuditLog{}).
			Where("tenant_id = ? AND action = ?", suite.testTenantID, "recent_action").
			Count(&recentCount)

		assert.Equal(t, int64(0), oldCount, "旧日志应该被清理")
		assert.Equal(t, int64(1), recentCount, "最近日志应该保留")
	})
}

// TestSecurityPerformance 测试安全功能性能
func (suite *SecurityServiceIntegrationTestSuite) TestSecurityPerformance() {
	suite.T().Run("限流性能测试", func(t *testing.T) {
		const numRequests = 1000
		
		start := time.Now()
		
		for i := 0; i < numRequests; i++ {
			req, _ := http.NewRequest("GET", "/api/v1/test/public", nil)
			req.RemoteAddr = fmt.Sprintf("192.168.250.%d:12345", i%50) // 50个不同IP
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)
		}
		
		duration := time.Since(start)
		
		// 平均每个请求处理时间应该小于10ms
		avgTime := duration / numRequests
		assert.Less(t, avgTime, 10*time.Millisecond, "限流处理性能应该满足要求")
		
		suite.T().Logf("处理 %d 个请求耗时: %v, 平均每请求: %v", 
			numRequests, duration, avgTime)
	})

	suite.T().Run("审计日志记录性能", func(t *testing.T) {
		const numLogs = 500
		
		start := time.Now()
		
		for i := 0; i < numLogs; i++ {
			req := &services.AuditLogRequest{
				TenantID:  suite.testTenantID,
				IPAddress: fmt.Sprintf("192.168.251.%d", i%100),
				Action:    "performance_test",
				Resource:  "test",
				Success:   true,
				Metadata: map[string]interface{}{
					"test_id": i,
				},
			}
			suite.securityAuditService.LogAuditEvent(suite.testContext, req)
		}
		
		duration := time.Since(start)
		
		// 平均每个日志记录时间应该小于5ms
		avgTime := duration / numLogs
		assert.Less(t, avgTime, 5*time.Millisecond, "审计日志记录性能应该满足要求")
		
		suite.T().Logf("记录 %d 条审计日志耗时: %v, 平均每条: %v", 
			numLogs, duration, avgTime)
	})
}

// 运行安全服务集成测试套件
func TestSecurityServiceIntegrationSuite(t *testing.T) {
	suite.Run(t, new(SecurityServiceIntegrationTestSuite))
}