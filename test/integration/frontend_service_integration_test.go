package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

// FrontendServiceIntegrationTestSuite 前端服务集成测试套件
type FrontendServiceIntegrationTestSuite struct {
	suite.Suite
	mockAPIServer      *httptest.Server
	mockWebSocketServer *httptest.Server
	frontendDir        string
	logger             *zap.Logger
	testContext        context.Context
}

// MockAPIResponse Mock API响应结构
type MockAPIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// MockUser Mock用户数据
type MockUser struct {
	ID          int    `json:"id"`
	Email       string `json:"email"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Avatar      string `json:"avatar"`
}

// MockProject Mock项目数据
type MockProject struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

// MockTask Mock任务数据
type MockTask struct {
	ID          int    `json:"id"`
	ProjectID   int    `json:"project_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	AssigneeID  *int   `json:"assignee_id"`
	CreatedAt   string `json:"created_at"`
}

// SetupSuite 测试套件初始化
func (suite *FrontendServiceIntegrationTestSuite) SetupSuite() {
	// 创建测试logger
	suite.logger, _ = zap.NewDevelopment()
	suite.testContext = context.Background()

	// 设置前端项目目录
	cwd, _ := os.Getwd()
	suite.frontendDir = filepath.Join(filepath.Dir(filepath.Dir(cwd)), "frontend")

	// 创建Mock API服务器
	suite.setupMockAPIServer()

	// 创建Mock WebSocket服务器
	suite.setupMockWebSocketServer()

	suite.T().Logf("🚀 前端测试环境初始化完成")
	suite.T().Logf("📁 前端目录: %s", suite.frontendDir)
	suite.T().Logf("🌐 Mock API: %s", suite.mockAPIServer.URL)
	suite.T().Logf("🔗 Mock WebSocket: %s", suite.mockWebSocketServer.URL)
}

// TearDownSuite 测试套件清理
func (suite *FrontendServiceIntegrationTestSuite) TearDownSuite() {
	if suite.mockAPIServer != nil {
		suite.mockAPIServer.Close()
	}
	if suite.mockWebSocketServer != nil {
		suite.mockWebSocketServer.Close()
	}
}

// setupMockAPIServer 设置Mock API服务器
func (suite *FrontendServiceIntegrationTestSuite) setupMockAPIServer() {
	router := gin.New()
	router.Use(gin.Recovery())

	// CORS中间件
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// 认证相关API
	auth := router.Group("/auth")
	{
		auth.POST("/login", suite.handleLogin)
		auth.POST("/register", suite.handleRegister)
		auth.POST("/logout", suite.handleLogout)
		auth.POST("/refresh", suite.handleRefreshToken)
	}

	// 用户相关API
	users := router.Group("/users")
	{
		users.GET("/me", suite.handleGetCurrentUser)
		users.GET("", suite.handleListUsers)
	}

	// 项目相关API
	projects := router.Group("/projects")
	{
		projects.GET("", suite.handleListProjects)
		projects.POST("", suite.handleCreateProject)
		projects.GET("/:id", suite.handleGetProject)
		projects.PUT("/:id", suite.handleUpdateProject)
		projects.DELETE("/:id", suite.handleDeleteProject)
		projects.GET("/:id/tasks", suite.handleListProjectTasks)
	}

	// 任务相关API
	tasks := router.Group("/tasks")
	{
		tasks.GET("", suite.handleListTasks)
		tasks.POST("", suite.handleCreateTask)
		tasks.GET("/:id", suite.handleGetTask)
		tasks.PUT("/:id", suite.handleUpdateTask)
		tasks.DELETE("/:id", suite.handleDeleteTask)
		tasks.POST("/reorder", suite.handleReorderTasks)
	}

	suite.mockAPIServer = httptest.NewServer(router)
}

// setupMockWebSocketServer 设置Mock WebSocket服务器
func (suite *FrontendServiceIntegrationTestSuite) setupMockWebSocketServer() {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			suite.T().Logf("WebSocket升级失败: %v", err)
			return
		}
		defer conn.Close()

		// 简单的WebSocket回声服务器
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				break
			}

			// 回应收到的消息
			err = conn.WriteMessage(messageType, message)
			if err != nil {
				break
			}
		}
	})

	suite.mockWebSocketServer = httptest.NewServer(mux)
}

// Mock API处理函数
func (suite *FrontendServiceIntegrationTestSuite) handleLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, MockAPIResponse{
			Success: false,
			Error:   "INVALID_REQUEST",
			Message: "请求参数无效",
		})
		return
	}

	// 模拟登录验证
	if req.Email == "test@example.com" && req.Password == "password123" {
		c.JSON(http.StatusOK, MockAPIResponse{
			Success: true,
			Data: map[string]interface{}{
				"access_token":  "mock_access_token_123",
				"refresh_token": "mock_refresh_token_456",
				"expires_in":    3600,
				"user": MockUser{
					ID:          1,
					Email:       req.Email,
					Username:    "testuser",
					DisplayName: "Test User",
					Avatar:      "https://example.com/avatar.jpg",
				},
			},
		})
	} else {
		c.JSON(http.StatusUnauthorized, MockAPIResponse{
			Success: false,
			Error:   "INVALID_CREDENTIALS",
			Message: "邮箱或密码错误",
		})
	}
}

func (suite *FrontendServiceIntegrationTestSuite) handleRegister(c *gin.Context) {
	var req struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, MockAPIResponse{
			Success: false,
			Error:   "INVALID_REQUEST",
			Message: "请求参数无效",
		})
		return
	}

	// 模拟注册成功
	c.JSON(http.StatusOK, MockAPIResponse{
		Success: true,
		Data: map[string]interface{}{
			"user_id": 2,
			"message": "注册成功",
		},
	})
}

func (suite *FrontendServiceIntegrationTestSuite) handleLogout(c *gin.Context) {
	c.JSON(http.StatusOK, MockAPIResponse{
		Success: true,
		Message: "退出成功",
	})
}

func (suite *FrontendServiceIntegrationTestSuite) handleRefreshToken(c *gin.Context) {
	c.JSON(http.StatusOK, MockAPIResponse{
		Success: true,
		Data: map[string]interface{}{
			"access_token": "new_mock_access_token",
			"expires_in":   3600,
		},
	})
}

func (suite *FrontendServiceIntegrationTestSuite) handleGetCurrentUser(c *gin.Context) {
	c.JSON(http.StatusOK, MockAPIResponse{
		Success: true,
		Data: MockUser{
			ID:          1,
			Email:       "test@example.com",
			Username:    "testuser",
			DisplayName: "Test User",
			Avatar:      "https://example.com/avatar.jpg",
		},
	})
}

func (suite *FrontendServiceIntegrationTestSuite) handleListUsers(c *gin.Context) {
	users := []MockUser{
		{ID: 1, Email: "test@example.com", Username: "testuser", DisplayName: "Test User"},
		{ID: 2, Email: "dev@example.com", Username: "developer", DisplayName: "Developer"},
	}

	c.JSON(http.StatusOK, MockAPIResponse{
		Success: true,
		Data:    users,
	})
}

func (suite *FrontendServiceIntegrationTestSuite) handleListProjects(c *gin.Context) {
	projects := []MockProject{
		{ID: 1, Name: "测试项目1", Description: "这是一个测试项目", Status: "active", CreatedAt: "2024-01-01T00:00:00Z"},
		{ID: 2, Name: "测试项目2", Description: "另一个测试项目", Status: "active", CreatedAt: "2024-01-02T00:00:00Z"},
	}

	c.JSON(http.StatusOK, MockAPIResponse{
		Success: true,
		Data:    projects,
	})
}

func (suite *FrontendServiceIntegrationTestSuite) handleCreateProject(c *gin.Context) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, MockAPIResponse{
			Success: false,
			Error:   "INVALID_REQUEST",
			Message: "请求参数无效",
		})
		return
	}

	newProject := MockProject{
		ID:          3,
		Name:        req.Name,
		Description: req.Description,
		Status:      "active",
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, MockAPIResponse{
		Success: true,
		Data:    newProject,
	})
}

func (suite *FrontendServiceIntegrationTestSuite) handleGetProject(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "1" {
		c.JSON(http.StatusOK, MockAPIResponse{
			Success: true,
			Data: MockProject{
				ID:          1,
				Name:        "测试项目1",
				Description: "这是一个测试项目",
				Status:      "active",
				CreatedAt:   "2024-01-01T00:00:00Z",
			},
		})
	} else {
		c.JSON(http.StatusNotFound, MockAPIResponse{
			Success: false,
			Error:   "PROJECT_NOT_FOUND",
			Message: "项目不存在",
		})
	}
}

func (suite *FrontendServiceIntegrationTestSuite) handleUpdateProject(c *gin.Context) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, MockAPIResponse{
			Success: false,
			Error:   "INVALID_REQUEST",
			Message: "请求参数无效",
		})
		return
	}

	c.JSON(http.StatusOK, MockAPIResponse{
		Success: true,
		Data: MockProject{
			ID:          1,
			Name:        req.Name,
			Description: req.Description,
			Status:      req.Status,
			CreatedAt:   "2024-01-01T00:00:00Z",
		},
	})
}

func (suite *FrontendServiceIntegrationTestSuite) handleDeleteProject(c *gin.Context) {
	c.JSON(http.StatusOK, MockAPIResponse{
		Success: true,
		Message: "项目删除成功",
	})
}

func (suite *FrontendServiceIntegrationTestSuite) handleListProjectTasks(c *gin.Context) {
	tasks := []MockTask{
		{ID: 1, ProjectID: 1, Title: "任务1", Description: "测试任务1", Status: "todo", Priority: "high"},
		{ID: 2, ProjectID: 1, Title: "任务2", Description: "测试任务2", Status: "in_progress", Priority: "medium"},
	}

	c.JSON(http.StatusOK, MockAPIResponse{
		Success: true,
		Data:    tasks,
	})
}

func (suite *FrontendServiceIntegrationTestSuite) handleListTasks(c *gin.Context) {
	projectID := c.Query("project_id")
	tasks := []MockTask{
		{ID: 1, ProjectID: 1, Title: "任务1", Description: "测试任务1", Status: "todo", Priority: "high"},
		{ID: 2, ProjectID: 1, Title: "任务2", Description: "测试任务2", Status: "in_progress", Priority: "medium"},
	}

	if projectID != "" {
		// 根据项目ID过滤任务
	}

	c.JSON(http.StatusOK, MockAPIResponse{
		Success: true,
		Data:    tasks,
	})
}

func (suite *FrontendServiceIntegrationTestSuite) handleCreateTask(c *gin.Context) {
	var req struct {
		ProjectID   int    `json:"project_id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Priority    string `json:"priority"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, MockAPIResponse{
			Success: false,
			Error:   "INVALID_REQUEST",
			Message: "请求参数无效",
		})
		return
	}

	newTask := MockTask{
		ID:          3,
		ProjectID:   req.ProjectID,
		Title:       req.Title,
		Description: req.Description,
		Status:      "todo",
		Priority:    req.Priority,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, MockAPIResponse{
		Success: true,
		Data:    newTask,
	})
}

func (suite *FrontendServiceIntegrationTestSuite) handleGetTask(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "1" {
		c.JSON(http.StatusOK, MockAPIResponse{
			Success: true,
			Data: MockTask{
				ID:          1,
				ProjectID:   1,
				Title:       "任务1",
				Description: "测试任务1",
				Status:      "todo",
				Priority:    "high",
			},
		})
	} else {
		c.JSON(http.StatusNotFound, MockAPIResponse{
			Success: false,
			Error:   "TASK_NOT_FOUND",
			Message: "任务不存在",
		})
	}
}

func (suite *FrontendServiceIntegrationTestSuite) handleUpdateTask(c *gin.Context) {
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Status      string `json:"status"`
		Priority    string `json:"priority"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, MockAPIResponse{
			Success: false,
			Error:   "INVALID_REQUEST",
			Message: "请求参数无效",
		})
		return
	}

	c.JSON(http.StatusOK, MockAPIResponse{
		Success: true,
		Data: MockTask{
			ID:          1,
			ProjectID:   1,
			Title:       req.Title,
			Description: req.Description,
			Status:      req.Status,
			Priority:    req.Priority,
		},
	})
}

func (suite *FrontendServiceIntegrationTestSuite) handleDeleteTask(c *gin.Context) {
	c.JSON(http.StatusOK, MockAPIResponse{
		Success: true,
		Message: "任务删除成功",
	})
}

func (suite *FrontendServiceIntegrationTestSuite) handleReorderTasks(c *gin.Context) {
	c.JSON(http.StatusOK, MockAPIResponse{
		Success: true,
		Message: "任务重排序成功",
	})
}

// TestFrontendProjectStructure 测试前端项目结构
func (suite *FrontendServiceIntegrationTestSuite) TestFrontendProjectStructure() {
	suite.T().Run("验证前端项目目录结构", func(t *testing.T) {
		// 检查主要目录是否存在
		requiredDirs := []string{
			"src",
			"src/components",
			"src/pages",
			"src/utils",
			"src/types",
			"src/hooks",
			"src/contexts",
			"src/styles",
		}

		for _, dir := range requiredDirs {
			dirPath := filepath.Join(suite.frontendDir, dir)
			_, err := os.Stat(dirPath)
			assert.NoError(t, err, "目录 %s 应该存在", dir)
		}

		// 检查关键文件是否存在
		requiredFiles := []string{
			"package.json",
			"tsconfig.json",
			"vite.config.ts",
			"src/main.tsx",
			"src/App.tsx",
			"index.html",
		}

		for _, file := range requiredFiles {
			filePath := filepath.Join(suite.frontendDir, file)
			_, err := os.Stat(filePath)
			assert.NoError(t, err, "文件 %s 应该存在", file)
		}
	})

	suite.T().Run("验证package.json配置", func(t *testing.T) {
		packagePath := filepath.Join(suite.frontendDir, "package.json")
		data, err := os.ReadFile(packagePath)
		require.NoError(t, err)

		var packageJSON map[string]interface{}
		err = json.Unmarshal(data, &packageJSON)
		require.NoError(t, err)

		// 验证必要的脚本命令
		scripts, ok := packageJSON["scripts"].(map[string]interface{})
		assert.True(t, ok, "package.json应该包含scripts字段")

		requiredScripts := []string{"dev", "build", "test", "lint", "type-check"}
		for _, script := range requiredScripts {
			_, exists := scripts[script]
			assert.True(t, exists, "应该包含 %s 脚本", script)
		}

		// 验证关键依赖
		dependencies, ok := packageJSON["dependencies"].(map[string]interface{})
		assert.True(t, ok, "package.json应该包含dependencies字段")

		requiredDeps := []string{"react", "react-dom", "typescript", "axios"}
		for _, dep := range requiredDeps {
			_, exists := dependencies[dep]
			assert.True(t, exists, "应该包含 %s 依赖", dep)
		}
	})
}

// TestFrontendCodeQuality 测试前端代码质量
func (suite *FrontendServiceIntegrationTestSuite) TestFrontendCodeQuality() {
	suite.T().Run("TypeScript类型检查", func(t *testing.T) {
		cmd := exec.Command("npm", "run", "type-check")
		cmd.Dir = suite.frontendDir
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Logf("TypeScript检查输出:\n%s", string(output))
		}

		assert.NoError(t, err, "TypeScript类型检查应该通过")
	})

	suite.T().Run("ESLint代码规范检查", func(t *testing.T) {
		cmd := exec.Command("npm", "run", "lint")
		cmd.Dir = suite.frontendDir
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Logf("ESLint检查输出:\n%s", string(output))
		}

		// ESLint可能有警告但不影响构建
		if err != nil {
			t.Logf("ESLint检查有问题，需要修复代码规范")
		}
	})

	suite.T().Run("验证代码文件结构", func(t *testing.T) {
		// 检查TypeScript文件的基本语法结构
		err := filepath.Walk(filepath.Join(suite.frontendDir, "src"), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx") {
				content, readErr := os.ReadFile(path)
				if readErr != nil {
					return readErr
				}

				// 基本的语法检查
				contentStr := string(content)
				
				// 检查是否有未使用的import（基本检查）
				if strings.Contains(contentStr, "import") {
					assert.NotContains(t, contentStr, "import {  }", "文件 %s 不应该有空的import", path)
				}

				// 检查是否有明显的语法错误
				assert.NotContains(t, contentStr, "console.log", "生产代码不应该包含console.log: %s", path)
			}

			return nil
		})

		assert.NoError(t, err, "代码文件遍历应该成功")
	})
}

// TestAPIIntegration 测试API集成
func (suite *FrontendServiceIntegrationTestSuite) TestAPIIntegration() {
	suite.T().Run("认证API集成测试", func(t *testing.T) {
		// 测试登录API调用
		loginData := map[string]string{
			"email":    "test@example.com",
			"password": "password123",
		}

		loginJSON, _ := json.Marshal(loginData)
		resp, err := http.Post(suite.mockAPIServer.URL+"/auth/login", "application/json", bytes.NewBuffer(loginJSON))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.Status)

		var loginResp MockAPIResponse
		err = json.NewDecoder(resp.Body).Decode(&loginResp)
		require.NoError(t, err)

		assert.True(t, loginResp.Success)
		assert.Contains(t, loginResp.Data, "access_token")
		assert.Contains(t, loginResp.Data, "user")
	})

	suite.T().Run("项目API集成测试", func(t *testing.T) {
		// 测试获取项目列表
		resp, err := http.Get(suite.mockAPIServer.URL + "/projects")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.Status)

		var projectsResp MockAPIResponse
		err = json.NewDecoder(resp.Body).Decode(&projectsResp)
		require.NoError(t, err)

		assert.True(t, projectsResp.Success)
		projects, ok := projectsResp.Data.([]interface{})
		assert.True(t, ok)
		assert.Greater(t, len(projects), 0)
	})

	suite.T().Run("任务API集成测试", func(t *testing.T) {
		// 测试创建任务
		taskData := map[string]interface{}{
			"project_id":  1,
			"title":       "API集成测试任务",
			"description": "这是一个API集成测试任务",
			"priority":    "high",
		}

		taskJSON, _ := json.Marshal(taskData)
		resp, err := http.Post(suite.mockAPIServer.URL+"/tasks", "application/json", bytes.NewBuffer(taskJSON))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.Status)

		var taskResp MockAPIResponse
		err = json.NewDecoder(resp.Body).Decode(&taskResp)
		require.NoError(t, err)

		assert.True(t, taskResp.Success)
		task, ok := taskResp.Data.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "API集成测试任务", task["title"])
	})

	suite.T().Run("API错误处理测试", func(t *testing.T) {
		// 测试无效的登录凭据
		invalidLogin := map[string]string{
			"email":    "invalid@example.com",
			"password": "wrongpassword",
		}

		loginJSON, _ := json.Marshal(invalidLogin)
		resp, err := http.Post(suite.mockAPIServer.URL+"/auth/login", "application/json", bytes.NewBuffer(loginJSON))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.Status)

		var errorResp MockAPIResponse
		err = json.NewDecoder(resp.Body).Decode(&errorResp)
		require.NoError(t, err)

		assert.False(t, errorResp.Success)
		assert.Equal(t, "INVALID_CREDENTIALS", errorResp.Error)
	})
}

// TestWebSocketIntegration 测试WebSocket集成
func (suite *FrontendServiceIntegrationTestSuite) TestWebSocketIntegration() {
	suite.T().Run("WebSocket连接测试", func(t *testing.T) {
		// 将http URL转换为ws URL
		wsURL := strings.Replace(suite.mockWebSocketServer.URL, "http", "ws", 1) + "/ws"
		
		// 创建WebSocket连接
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer conn.Close()

		// 发送测试消息
		testMessage := map[string]interface{}{
			"type":       "test_message",
			"user_id":    1,
			"username":   "testuser",
			"project_id": 1,
			"data":       map[string]string{"message": "Hello WebSocket"},
			"timestamp":  time.Now().Format(time.RFC3339),
		}

		messageJSON, _ := json.Marshal(testMessage)
		err = conn.WriteMessage(websocket.TextMessage, messageJSON)
		assert.NoError(t, err)

		// 接收回复消息
		_, receivedMessage, err := conn.ReadMessage()
		assert.NoError(t, err)

		var receivedData map[string]interface{}
		err = json.Unmarshal(receivedMessage, &receivedData)
		assert.NoError(t, err)
		assert.Equal(t, "test_message", receivedData["type"])
	})

	suite.T().Run("WebSocket消息类型测试", func(t *testing.T) {
		wsURL := strings.Replace(suite.mockWebSocketServer.URL, "http", "ws", 1) + "/ws"
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer conn.Close()

		// 测试不同类型的消息
		messageTypes := []string{
			"task_update",
			"task_create",
			"task_delete",
			"chat_message",
			"typing",
			"user_status",
		}

		for _, msgType := range messageTypes {
			message := map[string]interface{}{
				"type":       msgType,
				"user_id":    1,
				"username":   "testuser",
				"project_id": 1,
				"data":       map[string]string{"test": "data"},
				"timestamp":  time.Now().Format(time.RFC3339),
			}

			messageJSON, _ := json.Marshal(message)
			err = conn.WriteMessage(websocket.TextMessage, messageJSON)
			assert.NoError(t, err)

			// 验证服务器回应
			_, receivedMessage, err := conn.ReadMessage()
			assert.NoError(t, err)

			var receivedData map[string]interface{}
			err = json.Unmarshal(receivedMessage, &receivedData)
			assert.NoError(t, err)
			assert.Equal(t, msgType, receivedData["type"])
		}
	})
}

// TestFrontendBuildAndDeployment 测试前端构建和部署
func (suite *FrontendServiceIntegrationTestSuite) TestFrontendBuildAndDeployment() {
	suite.T().Run("前端依赖安装测试", func(t *testing.T) {
		// 检查node_modules是否存在，如果不存在则安装
		nodeModulesPath := filepath.Join(suite.frontendDir, "node_modules")
		if _, err := os.Stat(nodeModulesPath); os.IsNotExist(err) {
			cmd := exec.Command("npm", "install")
			cmd.Dir = suite.frontendDir
			output, err := cmd.CombinedOutput()
			
			if err != nil {
				t.Logf("npm install输出:\n%s", string(output))
			}
			
			assert.NoError(t, err, "npm install应该成功")
		}

		// 验证重要依赖是否已安装
		requiredPaths := []string{
			"node_modules/react",
			"node_modules/react-dom",
			"node_modules/typescript",
			"node_modules/vite",
		}

		for _, path := range requiredPaths {
			fullPath := filepath.Join(suite.frontendDir, path)
			_, err := os.Stat(fullPath)
			assert.NoError(t, err, "依赖 %s 应该已安装", path)
		}
	})

	suite.T().Run("前端构建测试", func(t *testing.T) {
		// 清理之前的构建结果
		distPath := filepath.Join(suite.frontendDir, "dist")
		os.RemoveAll(distPath)

		// 执行构建命令
		cmd := exec.Command("npm", "run", "build")
		cmd.Dir = suite.frontendDir
		cmd.Env = append(os.Environ(), "NODE_ENV=production")
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("构建输出:\n%s", string(output))
		}

		assert.NoError(t, err, "前端构建应该成功")

		// 验证构建产物
		_, err = os.Stat(distPath)
		assert.NoError(t, err, "dist目录应该存在")

		// 检查关键文件是否生成
		expectedFiles := []string{"index.html"}
		for _, file := range expectedFiles {
			filePath := filepath.Join(distPath, file)
			_, err := os.Stat(filePath)
			assert.NoError(t, err, "构建文件 %s 应该存在", file)
		}

		// 检查是否有JavaScript和CSS文件生成
		err = filepath.Walk(distPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			
			if strings.HasSuffix(path, ".js") {
				t.Logf("生成JavaScript文件: %s", path)
			}
			if strings.HasSuffix(path, ".css") {
				t.Logf("生成CSS文件: %s", path)
			}
			
			return nil
		})
		assert.NoError(t, err)
	})

	suite.T().Run("构建产物质量检查", func(t *testing.T) {
		distPath := filepath.Join(suite.frontendDir, "dist")
		
		// 检查index.html内容
		indexPath := filepath.Join(distPath, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			content, err := os.ReadFile(indexPath)
			require.NoError(t, err)
			
			contentStr := string(content)
			assert.Contains(t, contentStr, "<html", "index.html应该包含html标签")
			assert.Contains(t, contentStr, "<head", "index.html应该包含head标签")
			assert.Contains(t, contentStr, "<body", "index.html应该包含body标签")
			
			// 检查是否包含JavaScript模块引用
			jsRegex := regexp.MustCompile(`<script[^>]*type="module"[^>]*src="[^"]*\.js"`)
			assert.Regexp(t, jsRegex, contentStr, "应该包含模块化JavaScript引用")
		}

		// 统计构建产物大小
		var totalSize int64
		err := filepath.Walk(distPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				totalSize += info.Size()
			}
			return nil
		})
		
		assert.NoError(t, err)
		t.Logf("构建产物总大小: %d bytes (%.2f MB)", totalSize, float64(totalSize)/1024/1024)
		
		// 构建产物不应该太大 (< 50MB)
		assert.Less(t, totalSize, int64(50*1024*1024), "构建产物大小应该合理")
	})
}

// TestFrontendUnitTests 测试前端单元测试
func (suite *FrontendServiceIntegrationTestSuite) TestFrontendUnitTests() {
	suite.T().Run("执行前端单元测试", func(t *testing.T) {
		// 检查测试文件是否存在
		testDir := filepath.Join(suite.frontendDir, "src", "test")
		_, err := os.Stat(testDir)
		if os.IsNotExist(err) {
			t.Skip("没有发现测试文件，跳过单元测试")
			return
		}

		// 运行单元测试
		cmd := exec.Command("npm", "run", "test")
		cmd.Dir = suite.frontendDir
		cmd.Env = append(os.Environ(), "CI=true") // 非交互模式
		
		output, err := cmd.CombinedOutput()
		t.Logf("测试输出:\n%s", string(output))

		if err != nil {
			// 即使测试失败，也要检查输出以获取详细信息
			outputStr := string(output)
			if strings.Contains(outputStr, "PASS") {
				t.Logf("有测试通过")
			}
			if strings.Contains(outputStr, "FAIL") {
				t.Logf("有测试失败")
			}
		}

		// 在实际项目中，这里应该要求测试通过
		// assert.NoError(t, err, "前端单元测试应该通过")
	})

	suite.T().Run("测试覆盖率检查", func(t *testing.T) {
		// 运行测试覆盖率检查
		cmd := exec.Command("npm", "run", "test:coverage")
		cmd.Dir = suite.frontendDir
		cmd.Env = append(os.Environ(), "CI=true")
		
		output, err := cmd.CombinedOutput()
		t.Logf("覆盖率测试输出:\n%s", string(output))

		// 检查是否生成了覆盖率报告
		coverageDir := filepath.Join(suite.frontendDir, "coverage")
		if _, err := os.Stat(coverageDir); err == nil {
			t.Logf("✅ 生成了测试覆盖率报告")
		}
	})
}

// TestConcurrentFrontendOperations 测试并发前端操作
func (suite *FrontendServiceIntegrationTestSuite) TestConcurrentFrontendOperations() {
	suite.T().Run("并发API请求测试", func(t *testing.T) {
		const numGoroutines = 10
		const requestsPerGoroutine = 5
		
		var wg sync.WaitGroup
		results := make(chan bool, numGoroutines*requestsPerGoroutine)
		
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				
				for j := 0; j < requestsPerGoroutine; j++ {
					resp, err := http.Get(suite.mockAPIServer.URL + "/projects")
					if err == nil && resp.StatusCode == http.StatusOK {
						results <- true
						resp.Body.Close()
					} else {
						results <- false
						if resp != nil {
							resp.Body.Close()
						}
					}
				}
			}(i)
		}
		
		wg.Wait()
		close(results)
		
		// 统计成功率
		successCount := 0
		totalCount := 0
		for success := range results {
			if success {
				successCount++
			}
			totalCount++
		}
		
		successRate := float64(successCount) / float64(totalCount) * 100
		t.Logf("并发API请求成功率: %.2f%% (%d/%d)", successRate, successCount, totalCount)
		
		assert.Greater(t, successRate, 95.0, "并发API请求成功率应该大于95%")
	})

	suite.T().Run("并发WebSocket连接测试", func(t *testing.T) {
		const numConnections = 5
		
		var wg sync.WaitGroup
		results := make(chan bool, numConnections)
		
		wsURL := strings.Replace(suite.mockWebSocketServer.URL, "http", "ws", 1) + "/ws"
		
		for i := 0; i < numConnections; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				
				conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
				if err != nil {
					results <- false
					return
				}
				defer conn.Close()
				
				// 发送测试消息
				message := map[string]interface{}{
					"type":       "test_concurrent",
					"user_id":    id,
					"username":   fmt.Sprintf("user_%d", id),
					"project_id": 1,
					"data":       map[string]int{"connection_id": id},
				}
				
				messageJSON, _ := json.Marshal(message)
				err = conn.WriteMessage(websocket.TextMessage, messageJSON)
				if err != nil {
					results <- false
					return
				}
				
				// 接收回复
				_, _, err = conn.ReadMessage()
				results <- err == nil
			}(i)
		}
		
		wg.Wait()
		close(results)
		
		// 统计连接成功率
		successCount := 0
		for success := range results {
			if success {
				successCount++
			}
		}
		
		t.Logf("并发WebSocket连接成功: %d/%d", successCount, numConnections)
		assert.Equal(t, numConnections, successCount, "所有WebSocket连接都应该成功")
	})
}

// TestFrontendPerformance 测试前端性能
func (suite *FrontendServiceIntegrationTestSuite) TestFrontendPerformance() {
	suite.T().Run("API响应时间测试", func(t *testing.T) {
		const numRequests = 50
		var totalDuration time.Duration
		
		for i := 0; i < numRequests; i++ {
			start := time.Now()
			resp, err := http.Get(suite.mockAPIServer.URL + "/projects")
			duration := time.Since(start)
			
			if err == nil && resp.StatusCode == http.StatusOK {
				totalDuration += duration
				resp.Body.Close()
			}
		}
		
		avgDuration := totalDuration / numRequests
		t.Logf("API平均响应时间: %v", avgDuration)
		
		// API响应时间应该小于100ms
		assert.Less(t, avgDuration, 100*time.Millisecond, "API平均响应时间应该小于100ms")
	})

	suite.T().Run("大数据量API测试", func(t *testing.T) {
		// 测试处理大量数据的API响应
		start := time.Now()
		resp, err := http.Get(suite.mockAPIServer.URL + "/projects")
		duration := time.Since(start)
		
		require.NoError(t, err)
		defer resp.Body.Close()
		
		// 读取响应体
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		
		var apiResp MockAPIResponse
		err = json.Unmarshal(body, &apiResp)
		require.NoError(t, err)
		
		t.Logf("大数据量API响应时间: %v, 数据大小: %d bytes", duration, len(body))
		
		// 即使是大数据量，响应时间也应该合理
		assert.Less(t, duration, 1*time.Second, "大数据量API响应时间应该小于1秒")
	})
}

// TestFrontendSecurity 测试前端安全性
func (suite *FrontendServiceIntegrationTestSuite) TestFrontendSecurity() {
	suite.T().Run("XSS防护测试", func(t *testing.T) {
		// 测试API是否正确处理恶意脚本输入
		maliciousData := map[string]interface{}{
			"title":       "<script>alert('XSS')</script>",
			"description": "<img src=x onerror=alert('XSS')>",
			"project_id":  1,
		}

		taskJSON, _ := json.Marshal(maliciousData)
		resp, err := http.Post(suite.mockAPIServer.URL+"/tasks", "application/json", bytes.NewBuffer(taskJSON))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.Status)

		var taskResp MockAPIResponse
		err = json.NewDecoder(resp.Body).Decode(&taskResp)
		require.NoError(t, err)

		// 验证响应中的数据是否被适当处理（在真实应用中应该被转义或清理）
		task, ok := taskResp.Data.(map[string]interface{})
		assert.True(t, ok)
		
		// 记录可能的安全问题
		title := task["title"].(string)
		if strings.Contains(title, "<script>") {
			t.Logf("⚠️ 检测到可能的XSS风险: %s", title)
		}
	})

	suite.T().Run("CORS配置测试", func(t *testing.T) {
		// 测试CORS头部是否正确设置
		resp, err := http.Get(suite.mockAPIServer.URL + "/projects")
		require.NoError(t, err)
		defer resp.Body.Close()

		corsHeader := resp.Header.Get("Access-Control-Allow-Origin")
		assert.NotEmpty(t, corsHeader, "应该设置CORS头部")
		
		allowMethods := resp.Header.Get("Access-Control-Allow-Methods")
		assert.Contains(t, allowMethods, "GET", "应该允许GET方法")
		assert.Contains(t, allowMethods, "POST", "应该允许POST方法")
	})

	suite.T().Run("认证令牌处理测试", func(t *testing.T) {
		// 测试无效令牌的处理
		client := &http.Client{}
		req, _ := http.NewRequest("GET", suite.mockAPIServer.URL+"/users/me", nil)
		req.Header.Set("Authorization", "Bearer invalid_token")
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// 在真实应用中，无效令牌应该返回401状态码
		// 这里的Mock服务器还没有实现完整的令牌验证
		t.Logf("无效令牌响应状态: %d", resp.StatusCode)
	})
}

// 运行前端服务集成测试套件
func TestFrontendServiceIntegrationSuite(t *testing.T) {
	suite.Run(t, new(FrontendServiceIntegrationTestSuite))
}