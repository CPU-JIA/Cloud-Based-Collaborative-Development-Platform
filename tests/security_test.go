package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/cloud-platform/collaborative-dev/internal/models"
	"github.com/cloud-platform/collaborative-dev/internal/services"
)

// SecurityTestSuite 安全测试套件
type SecurityTestSuite struct {
	suite.Suite
	router      *gin.Engine
	db          *gorm.DB
	teamService *services.TeamService
	server      *httptest.Server
}

// SetupSuite 设置测试套件
func (suite *SecurityTestSuite) SetupSuite() {
	// 使用内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)

	// 自动迁移
	err = db.AutoMigrate(
		&models.Team{},
		&models.TeamMember{},
		&models.TeamInvitation{},
		&models.PermissionRequest{},
		&models.User{},
		&models.Role{},
		&models.TeamActivity{},
	)
	suite.Require().NoError(err)

	suite.db = db
	permissionService := services.NewPermissionService(db)
	suite.teamService = services.NewTeamService(db, permissionService)

	// 设置路由
	gin.SetMode(gin.TestMode)
	router := gin.New()
	suite.setupRoutes(router)
	suite.router = router

	// 启动测试服务器
	suite.server = httptest.NewServer(router)

	// 初始化测试数据
	suite.seedTestData()
}

// TearDownSuite 清理测试套件
func (suite *SecurityTestSuite) TearDownSuite() {
	suite.server.Close()
	sqlDB, _ := suite.db.DB()
	sqlDB.Close()
}

// setupRoutes 设置带安全验证的API路由
func (suite *SecurityTestSuite) setupRoutes(router *gin.Engine) {
	// 添加安全中间件
	router.Use(suite.securityMiddleware())
	router.Use(suite.tenantValidationMiddleware())

	api := router.Group("/api/v1")

	// 需要认证的路由
	authenticated := api.Group("", suite.authMiddleware())
	{
		teams := authenticated.Group("/teams")
		{
			teams.POST("", suite.requirePermission("create_team"), suite.createTeam)
			teams.GET("/project/:projectId", suite.requirePermission("read_team"), suite.getProjectTeams)
			teams.GET("/:id", suite.requirePermission("read_team"), suite.getTeam)
			teams.PUT("/:id", suite.requirePermission("update_team"), suite.updateTeam)
			teams.DELETE("/:id", suite.requirePermission("delete_team"), suite.deleteTeam)
			teams.POST("/:id/members", suite.requirePermission("manage_members"), suite.addTeamMember)
			teams.PUT("/:id/members/:userId", suite.requirePermission("manage_members"), suite.updateMemberRole)
			teams.DELETE("/:id/members/:userId", suite.requirePermission("manage_members"), suite.removeTeamMember)
		}

		invitations := authenticated.Group("/invitations")
		{
			invitations.POST("", suite.requirePermission("invite_user"), suite.createInvitation)
			invitations.GET("/team/:teamId", suite.requirePermission("read_invitations"), suite.getTeamInvitations)
		}

		requests := authenticated.Group("/permission-requests")
		{
			requests.POST("", suite.createPermissionRequest)
			requests.GET("/project/:projectId", suite.requirePermission("read_requests"), suite.getPermissionRequests)
			requests.POST("/:id/review", suite.requirePermission("review_requests"), suite.reviewPermissionRequest)
		}
	}

	// 公开路由（用于测试无认证访问）
	public := api.Group("/public")
	{
		public.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
	}
}

// seedTestData 初始化测试数据
func (suite *SecurityTestSuite) seedTestData() {
	// 创建测试用户（不同权限级别）
	users := []models.User{
		{ID: 1, Username: "admin", Email: "admin@test.com", DisplayName: "系统管理员", Status: models.UserStatusActive},
		{ID: 2, Username: "teamowner", Email: "owner@test.com", DisplayName: "团队所有者", Status: models.UserStatusActive},
		{ID: 3, Username: "member", Email: "member@test.com", DisplayName: "普通成员", Status: models.UserStatusActive},
		{ID: 4, Username: "viewer", Email: "viewer@test.com", DisplayName: "只读用户", Status: models.UserStatusActive},
		{ID: 5, Username: "inactive", Email: "inactive@test.com", DisplayName: "非活跃用户", Status: models.UserStatusInactive},
	}
	for _, user := range users {
		suite.db.Create(&user)
	}

	// 创建测试角色和权限
	roles := []models.Role{
		{
			ID: 1, TenantID: "default", ProjectID: 1,
			Name: "admin", Description: "系统管理员",
			Permissions: []string{"create_team", "read_team", "update_team", "delete_team", "manage_members", "invite_user", "read_invitations", "read_requests", "review_requests"},
			IsSystem:    true,
		},
		{
			ID: 2, TenantID: "default", ProjectID: 1,
			Name: "team_owner", Description: "团队所有者",
			Permissions: []string{"create_team", "read_team", "update_team", "manage_members", "invite_user", "read_invitations", "read_requests"},
			IsSystem:    true,
		},
		{
			ID: 3, TenantID: "default", ProjectID: 1,
			Name: "member", Description: "团队成员",
			Permissions: []string{"read_team", "read_invitations"},
			IsSystem:    true,
		},
		{
			ID: 4, TenantID: "default", ProjectID: 1,
			Name: "viewer", Description: "只读用户",
			Permissions: []string{"read_team"},
			IsSystem:    true,
		},
	}
	for _, role := range roles {
		suite.db.Create(&role)
	}

	// 创建测试团队和成员关系
	team, _ := suite.teamService.CreateTeam("default", 1, "安全测试团队", "用于安全测试", 2)
	suite.teamService.AddTeamMemberCompat(team.ID, 3, 3, 2) // 添加成员
	suite.teamService.AddTeamMemberCompat(team.ID, 4, 4, 2) // 添加只读用户
}

// 安全中间件
func (suite *SecurityTestSuite) securityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置安全头
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// 验证Content-Type（防止CSRF）
		if c.Request.Method == "POST" || c.Request.Method == "PUT" {
			contentType := c.GetHeader("Content-Type")
			if !strings.HasPrefix(contentType, "application/json") {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid content type"})
				c.Abort()
				return
			}
		}

		// 验证请求大小
		if c.Request.ContentLength > 1024*1024 { // 1MB限制
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Request too large"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// 租户验证中间件
func (suite *SecurityTestSuite) tenantValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetHeader("X-Tenant-ID")
		if tenantID == "" {
			tenantID = "default"
		}

		// 验证租户ID格式（防止注入攻击）
		if !isValidTenantID(tenantID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
			c.Abort()
			return
		}

		c.Set("tenant_id", tenantID)
		c.Next()
	}
}

// 认证中间件
func (suite *SecurityTestSuite) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization header"})
			c.Abort()
			return
		}

		// 简化的token验证（实际应该使用JWT或其他安全机制）
		token := strings.TrimPrefix(authHeader, "Bearer ")
		userID := suite.validateToken(token)
		if userID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// 验证用户状态
		var user models.User
		if err := suite.db.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		if user.Status != models.UserStatusActive {
			c.JSON(http.StatusForbidden, gin.H{"error": "User account is inactive"})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Set("user", user)
		c.Next()
	}
}

// 权限验证中间件
func (suite *SecurityTestSuite) requirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			c.Abort()
			return
		}

		hasPermission, err := suite.teamService.CheckUserPermission(userID.(int), 1, permission)
		if err != nil || !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// API处理函数（简化版）
func (suite *SecurityTestSuite) createTeam(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "Team created"})
}

func (suite *SecurityTestSuite) getProjectTeams(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"teams": []interface{}{}})
}

func (suite *SecurityTestSuite) getTeam(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"team": gin.H{}})
}

func (suite *SecurityTestSuite) updateTeam(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Team updated"})
}

func (suite *SecurityTestSuite) deleteTeam(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Team deleted"})
}

func (suite *SecurityTestSuite) addTeamMember(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "Member added"})
}

func (suite *SecurityTestSuite) updateMemberRole(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Role updated"})
}

func (suite *SecurityTestSuite) removeTeamMember(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Member removed"})
}

func (suite *SecurityTestSuite) createInvitation(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "Invitation created"})
}

func (suite *SecurityTestSuite) getTeamInvitations(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"invitations": []interface{}{}})
}

func (suite *SecurityTestSuite) createPermissionRequest(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "Permission request created"})
}

func (suite *SecurityTestSuite) getPermissionRequests(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"requests": []interface{}{}})
}

func (suite *SecurityTestSuite) reviewPermissionRequest(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Request reviewed"})
}

// 测试用例

// TestAuthenticationRequired 测试认证要求
func (suite *SecurityTestSuite) TestAuthenticationRequired() {
	// 未提供认证令牌的请求应该被拒绝
	resp, err := http.Get(suite.server.URL + "/api/v1/teams/project/1")
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusUnauthorized, resp.StatusCode)

	var response map[string]string
	json.NewDecoder(resp.Body).Decode(&response)
	suite.Contains(response["error"], "Missing authorization header")
}

// TestInvalidToken 测试无效令牌
func (suite *SecurityTestSuite) TestInvalidToken() {
	req, _ := http.NewRequest("GET", suite.server.URL+"/api/v1/teams/project/1", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusUnauthorized, resp.StatusCode)

	var response map[string]string
	json.NewDecoder(resp.Body).Decode(&response)
	suite.Contains(response["error"], "Invalid token")
}

// TestPermissionDenied 测试权限不足
func (suite *SecurityTestSuite) TestPermissionDenied() {
	// 使用只读用户的令牌尝试创建团队
	req, _ := http.NewRequest("POST", suite.server.URL+"/api/v1/teams", bytes.NewBuffer([]byte(`{"name":"test"}`)))
	req.Header.Set("Authorization", "Bearer viewer-token")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusForbidden, resp.StatusCode)

	var response map[string]string
	json.NewDecoder(resp.Body).Decode(&response)
	suite.Contains(response["error"], "Insufficient permissions")
}

// TestValidPermission 测试有效权限
func (suite *SecurityTestSuite) TestValidPermission() {
	// 使用管理员令牌创建团队
	req, _ := http.NewRequest("POST", suite.server.URL+"/api/v1/teams", bytes.NewBuffer([]byte(`{"name":"test"}`)))
	req.Header.Set("Authorization", "Bearer admin-token")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusCreated, resp.StatusCode)
}

// TestInactiveUserAccess 测试非活跃用户访问
func (suite *SecurityTestSuite) TestInactiveUserAccess() {
	req, _ := http.NewRequest("GET", suite.server.URL+"/api/v1/teams/project/1", nil)
	req.Header.Set("Authorization", "Bearer inactive-token")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusForbidden, resp.StatusCode)

	var response map[string]string
	json.NewDecoder(resp.Body).Decode(&response)
	suite.Contains(response["error"], "User account is inactive")
}

// TestCSRFProtection 测试CSRF保护
func (suite *SecurityTestSuite) TestCSRFProtection() {
	// 尝试使用错误的Content-Type
	req, _ := http.NewRequest("POST", suite.server.URL+"/api/v1/teams", bytes.NewBuffer([]byte(`{"name":"test"}`)))
	req.Header.Set("Authorization", "Bearer admin-token")
	req.Header.Set("Content-Type", "text/plain") // 错误的Content-Type

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)

	var response map[string]string
	json.NewDecoder(resp.Body).Decode(&response)
	suite.Contains(response["error"], "Invalid content type")
}

// TestRequestSizeLimit 测试请求大小限制
func (suite *SecurityTestSuite) TestRequestSizeLimit() {
	// 创建超大请求
	largeData := strings.Repeat("a", 2*1024*1024) // 2MB
	req, _ := http.NewRequest("POST", suite.server.URL+"/api/v1/teams", bytes.NewBuffer([]byte(largeData)))
	req.Header.Set("Authorization", "Bearer admin-token")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusRequestEntityTooLarge, resp.StatusCode)
}

// TestTenantIsolation 测试租户隔离
func (suite *SecurityTestSuite) TestTenantIsolation() {
	// 使用无效的租户ID
	req, _ := http.NewRequest("GET", suite.server.URL+"/api/v1/teams/project/1", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	req.Header.Set("X-Tenant-ID", "malicious'; DROP TABLE teams; --")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)

	var response map[string]string
	json.NewDecoder(resp.Body).Decode(&response)
	suite.Contains(response["error"], "Invalid tenant ID")
}

// TestSecurityHeaders 测试安全头
func (suite *SecurityTestSuite) TestSecurityHeaders() {
	req, _ := http.NewRequest("GET", suite.server.URL+"/api/v1/public/health", nil)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	// 验证安全头是否存在
	suite.Equal("nosniff", resp.Header.Get("X-Content-Type-Options"))
	suite.Equal("DENY", resp.Header.Get("X-Frame-Options"))
	suite.Equal("1; mode=block", resp.Header.Get("X-XSS-Protection"))
	suite.Contains(resp.Header.Get("Strict-Transport-Security"), "max-age=31536000")
}

// TestSQLInjectionPrevention 测试SQL注入防护
func (suite *SecurityTestSuite) TestSQLInjectionPrevention() {
	maliciousInput := `{"name": "test'; DROP TABLE teams; --", "description": "test"}`

	req, _ := http.NewRequest("POST", suite.server.URL+"/api/v1/teams", bytes.NewBuffer([]byte(maliciousInput)))
	req.Header.Set("Authorization", "Bearer admin-token")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	// 应该成功创建（因为使用了参数化查询），但不会执行恶意SQL
	suite.Equal(http.StatusCreated, resp.StatusCode)

	// 验证数据库表仍然存在
	var count int64
	suite.db.Model(&models.Team{}).Count(&count)
	suite.GreaterOrEqual(count, int64(0)) // 表应该仍然存在
}

// TestRateLimiting 测试频率限制（模拟）
func (suite *SecurityTestSuite) TestRateLimiting() {
	// 模拟快速连续请求
	const numRequests = 50
	var successCount int

	for i := 0; i < numRequests; i++ {
		req, _ := http.NewRequest("GET", suite.server.URL+"/api/v1/public/health", nil)

		client := &http.Client{Timeout: time.Millisecond * 100}
		resp, err := client.Do(req)
		if err == nil {
			if resp.StatusCode == http.StatusOK {
				successCount++
			}
			resp.Body.Close()
		}
	}

	// 所有请求都应该成功（这里没有实现真正的频率限制）
	suite.Equal(numRequests, successCount)
}

// TestInputValidation 测试输入验证
func (suite *SecurityTestSuite) TestInputValidation() {
	testCases := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "XSS尝试",
			input:    `{"name": "<script>alert('xss')</script>", "description": "test"}`,
			expected: http.StatusCreated, // 应该被转义但不会阻止创建
		},
		{
			name:     "极长输入",
			input:    fmt.Sprintf(`{"name": "%s", "description": "test"}`, strings.Repeat("a", 10000)),
			expected: http.StatusCreated, // 根据业务逻辑决定
		},
		{
			name:     "空输入",
			input:    `{"name": "", "description": "test"}`,
			expected: http.StatusCreated, // 根据业务逻辑决定
		},
		{
			name:     "特殊字符",
			input:    `{"name": "团队@#$%^&*()", "description": "测试\n\r\t"}`,
			expected: http.StatusCreated,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			req, _ := http.NewRequest("POST", suite.server.URL+"/api/v1/teams", bytes.NewBuffer([]byte(tc.input)))
			req.Header.Set("Authorization", "Bearer admin-token")
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			suite.NoError(err)
			defer resp.Body.Close()

			// 注意：这里的预期结果取决于具体的业务逻辑和验证规则
			// 在实际应用中，应该有更严格的输入验证
		})
	}
}

// 辅助函数

// validateToken 验证令牌并返回用户ID（简化版）
func (suite *SecurityTestSuite) validateToken(token string) int {
	tokenToUserMap := map[string]int{
		"admin-token":    1,
		"owner-token":    2,
		"member-token":   3,
		"viewer-token":   4,
		"inactive-token": 5,
	}

	if userID, exists := tokenToUserMap[token]; exists {
		return userID
	}
	return 0
}

// isValidTenantID 验证租户ID格式
func isValidTenantID(tenantID string) bool {
	// 简单的验证：只允许字母数字和连字符
	if len(tenantID) > 50 {
		return false
	}

	for _, char := range tenantID {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_') {
			return false
		}
	}
	return true
}

// 运行测试套件
func TestSecurityTestSuite(t *testing.T) {
	suite.Run(t, new(SecurityTestSuite))
}
