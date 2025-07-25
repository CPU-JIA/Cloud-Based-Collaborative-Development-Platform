package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// 数据结构定义
type User struct {
	ID          int    `json:"id"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Username    string `json:"username"`
	Avatar      string `json:"avatar"`
	CreatedAt   string `json:"created_at"`
}

type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"` 
	Username    string `json:"username"`
}

type Project struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Key         string `json:"key"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	TeamSize    int    `json:"team_size"`
	OwnerID     int    `json:"owner_id"`
	TasksCount  int    `json:"tasks_count"`
}

type CreateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Key         string `json:"key"`
}

type UpdateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type Task struct {
	ID           int    `json:"id"`
	ProjectID    int    `json:"project_id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	TaskNumber   string `json:"task_number"`
	StatusID     string `json:"status_id"`
	Priority     string `json:"priority"`
	AssigneeID   string `json:"assignee_id"`
	DueDate      string `json:"due_date"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type CreateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	StatusID    string `json:"status_id"`
	AssigneeID  string `json:"assignee_id"`
	DueDate     string `json:"due_date"`
}

type UpdateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	StatusID    string `json:"status_id"`
	AssigneeID  string `json:"assignee_id"`
	DueDate     string `json:"due_date"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Success     bool   `json:"success"`
	AccessToken string `json:"access_token"`
	User        User   `json:"user"`
	Message     string `json:"message"`
}

type ApiResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message"`
	Error   string      `json:"error,omitempty"`
}

// 全局变量用于存储数据
var nextProjectID = 4
var nextTaskID = 6
var nextUserID = 2

var mockUser = User{
	ID:          1,
	Email:       "demo@clouddev.com",
	Name:        "演示用户",
	DisplayName: "演示用户",
	Username:    "demo",
	Avatar:      "https://api.dicebear.com/7.x/avataaars/svg?seed=demo",
	CreatedAt:   "2024-01-01T00:00:00Z",
}

var mockUsers = []User{
	mockUser,
}

var mockProjects = []Project{
	{
		ID:          1,
		Name:        "企业协作开发平台",
		Key:         "CLOUD-DEV",
		Description: "基于云端的现代化企业级协作开发平台",
		Status:      "active",
		CreatedAt:   "2024-01-15T10:00:00Z",
		UpdatedAt:   "2024-07-24T19:00:00Z",
		TeamSize:    8,
		OwnerID:     1,
		TasksCount:  5,
	},
	{
		ID:          2,
		Name:        "AI智能助手系统",
		Key:         "AI-ASSIST",
		Description: "集成GPT的智能客服和文档助手系统",
		Status:      "active",
		CreatedAt:   "2024-02-01T09:30:00Z",
		UpdatedAt:   "2024-07-20T15:30:00Z",
		TeamSize:    5,
		OwnerID:     1,
		TasksCount:  3,
	},
	{
		ID:          3,
		Name:        "微服务监控平台",
		Key:         "MONITOR",
		Description: "分布式系统的实时监控和告警平台",
		Status:      "planning",
		CreatedAt:   "2024-03-10T14:20:00Z",
		UpdatedAt:   "2024-07-22T11:45:00Z",
		TeamSize:    3,
		OwnerID:     1,
		TasksCount:  2,
	},
}

var mockTasks = []Task{
	{
		ID:           1,
		ProjectID:    1,
		Title:        "用户界面设计",
		Description:  "设计现代化的用户界面，提升用户体验",
		TaskNumber:   "CLOUD-DEV-1",
		StatusID:     "2",
		Priority:     "high",
		AssigneeID:   "1",
		DueDate:      "2024-08-15T23:59:59Z",
		CreatedAt:    "2024-07-24T08:00:00Z",
		UpdatedAt:    "2024-07-24T19:00:00Z",
	},
	{
		ID:           2,
		ProjectID:    1,
		Title:        "JWT认证系统",
		Description:  "实现安全的用户认证和权限管理",
		TaskNumber:   "CLOUD-DEV-2",
		StatusID:     "3",
		Priority:     "high",
		AssigneeID:   "1",
		DueDate:      "2024-07-30T23:59:59Z",
		CreatedAt:    "2024-07-24T08:30:00Z",
		UpdatedAt:    "2024-07-24T18:30:00Z",
	},
	{
		ID:           3,
		ProjectID:    1,
		Title:        "项目看板优化",
		Description:  "增强看板功能和用户交互体验",
		TaskNumber:   "CLOUD-DEV-3",
		StatusID:     "2",
		Priority:     "medium",
		AssigneeID:   "1",
		DueDate:      "2024-08-10T23:59:59Z",
		CreatedAt:    "2024-07-24T09:00:00Z",
		UpdatedAt:    "2024-07-24T19:00:00Z",
	},
	{
		ID:           4,
		ProjectID:    1,
		Title:        "性能监控集成",
		Description:  "集成应用性能监控和错误追踪",
		TaskNumber:   "CLOUD-DEV-4",
		StatusID:     "1",
		Priority:     "medium",
		AssigneeID:   "",
		DueDate:      "2024-09-01T23:59:59Z",
		CreatedAt:    "2024-07-24T09:30:00Z",
		UpdatedAt:    "2024-07-24T09:30:00Z",
	},
	{
		ID:           5,
		ProjectID:    1,
		Title:        "移动端适配",
		Description:  "优化移动设备上的用户体验",
		TaskNumber:   "CLOUD-DEV-5",
		StatusID:     "1",
		Priority:     "low",
		AssigneeID:   "",
		DueDate:      "2024-09-15T23:59:59Z",
		CreatedAt:    "2024-07-24T10:00:00Z",
		UpdatedAt:    "2024-07-24T10:00:00Z",
	},
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// CORS中间件
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3001", "http://localhost:3002", "http://localhost:3003", "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"service":   "mock-api",
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// 认证路由
	auth := r.Group("/auth")
	{
		auth.POST("/login", handleLogin)
		auth.POST("/register", handleRegister)
		auth.POST("/logout", handleLogout)
	}

	// 项目路由
	projects := r.Group("/projects")
	{
		projects.GET("", handleGetProjects)
		projects.POST("", handleCreateProject)
		projects.GET("/:id", handleGetProject)
		projects.PUT("/:id", handleUpdateProject)
		projects.DELETE("/:id", handleDeleteProject)
		projects.GET("/:id/tasks", handleGetProjectTasks)
	}

	// 任务路由
	tasks := r.Group("/tasks")
	{
		tasks.POST("", handleCreateTask)
		tasks.GET("/:id", handleGetTask)
		tasks.PUT("/:id", handleUpdateTask)
		tasks.DELETE("/:id", handleDeleteTask)
	}

	// 用户路由
	users := r.Group("/users")
	{
		users.GET("/me", handleGetCurrentUser)
		users.GET("", handleGetUsers)
	}

	fmt.Println("🚀 Mock API服务启动成功！")
	fmt.Println("📡 监听地址: http://localhost:8082")
	fmt.Println("🔍 健康检查: http://localhost:8082/health")
	fmt.Println("📚 API文档:")
	fmt.Println("   POST /auth/login - 用户登录")
	fmt.Println("   POST /auth/register - 用户注册")
	fmt.Println("   GET /projects - 获取项目列表")
	fmt.Println("   POST /projects - 创建项目")
	fmt.Println("   GET /projects/:id - 获取项目详情")
	fmt.Println("   PUT /projects/:id - 更新项目")
	fmt.Println("   DELETE /projects/:id - 删除项目")
	fmt.Println("   GET /projects/:id/tasks - 获取项目任务")
	fmt.Println("   POST /tasks - 创建任务")
	fmt.Println("   PUT /tasks/:id - 更新任务")
	fmt.Println("   DELETE /tasks/:id - 删除任务")

	r.Run(":8082")
}

// 认证相关处理器
func handleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, ApiResponse{
			Success: false,
			Error:   "请求数据格式错误",
		})
		return
	}

	fmt.Printf("🔐 登录请求: %s\n", req.Email)

	if req.Email == "demo@clouddev.com" && req.Password == "demo123" {
		c.JSON(200, LoginResponse{
			Success:     true,
			AccessToken: "mock-jwt-token-" + fmt.Sprintf("%d", time.Now().Unix()),
			User:        mockUser,
			Message:     "登录成功",
		})
		fmt.Printf("✅ 登录成功: %s\n", req.Email)
	} else {
		c.JSON(401, ApiResponse{
			Success: false,
			Error:   "邮箱或密码错误",
		})
		fmt.Printf("❌ 登录失败: %s\n", req.Email)
	}
}

func handleRegister(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, ApiResponse{
			Success: false,
			Error:   "请求数据格式错误",
		})
		return
	}

	fmt.Printf("📝 注册请求: %s\n", req.Email)

	// 检查邮箱是否已存在
	for _, user := range mockUsers {
		if user.Email == req.Email {
			c.JSON(409, ApiResponse{
				Success: false,
				Error:   "邮箱已被注册",
			})
			return
		}
	}

	// 创建新用户
	newUser := User{
		ID:          nextUserID,
		Email:       req.Email,
		Name:        req.DisplayName,
		DisplayName: req.DisplayName,
		Username:    req.Username,
		Avatar:      fmt.Sprintf("https://api.dicebear.com/7.x/avataaars/svg?seed=%s", req.Username),
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	mockUsers = append(mockUsers, newUser)
	nextUserID++

	c.JSON(201, ApiResponse{
		Success: true,
		Data:    newUser,
		Message: "注册成功",
	})
	fmt.Printf("✅ 注册成功: %s\n", req.Email)
}

func handleLogout(c *gin.Context) {
	c.JSON(200, ApiResponse{
		Success: true,
		Message: "退出成功",
	})
}

// 项目相关处理器
func handleGetProjects(c *gin.Context) {
	fmt.Printf("📋 获取项目列表请求\n")
	c.JSON(200, ApiResponse{
		Success: true,
		Data:    mockProjects,
		Message: "获取项目列表成功",
	})
}

func handleCreateProject(c *gin.Context) {
	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, ApiResponse{
			Success: false,
			Error:   "请求数据格式错误",
		})
		return
	}

	fmt.Printf("🏗️ 创建项目请求: %s\n", req.Name)

	// 生成项目Key
	key := req.Key
	if key == "" {
		key = strings.ToUpper(strings.ReplaceAll(req.Name, " ", "-"))
	}

	newProject := Project{
		ID:          nextProjectID,
		Name:        req.Name,
		Key:         key,
		Description: req.Description,
		Status:      "active",
		CreatedAt:   time.Now().Format(time.RFC3339),
		UpdatedAt:   time.Now().Format(time.RFC3339),
		TeamSize:    1,
		OwnerID:     1,
		TasksCount:  0,
	}

	mockProjects = append(mockProjects, newProject)
	nextProjectID++

	c.JSON(201, ApiResponse{
		Success: true,
		Data:    newProject,
		Message: "项目创建成功",
	})
	fmt.Printf("✅ 项目创建成功: %s (ID: %d)\n", req.Name, newProject.ID)
}

func handleGetProject(c *gin.Context) {
	id := c.Param("id")
	fmt.Printf("📄 获取项目详情请求: %s\n", id)
	
	projectID, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(400, ApiResponse{
			Success: false,
			Error:   "项目ID格式错误",
		})
		return
	}

	for _, project := range mockProjects {
		if project.ID == projectID {
			c.JSON(200, ApiResponse{
				Success: true,
				Data:    project,
				Message: "获取项目详情成功",
			})
			return
		}
	}

	c.JSON(404, ApiResponse{
		Success: false,
		Error:   "项目不存在",
	})
}

func handleUpdateProject(c *gin.Context) {
	id := c.Param("id")
	projectID, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(400, ApiResponse{
			Success: false,
			Error:   "项目ID格式错误",
		})
		return
	}

	var req UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, ApiResponse{
			Success: false,
			Error:   "请求数据格式错误",
		})
		return
	}

	fmt.Printf("✏️ 更新项目请求: %d\n", projectID)

	for i, project := range mockProjects {
		if project.ID == projectID {
			mockProjects[i].Name = req.Name
			mockProjects[i].Description = req.Description
			if req.Status != "" {
				mockProjects[i].Status = req.Status
			}
			mockProjects[i].UpdatedAt = time.Now().Format(time.RFC3339)

			c.JSON(200, ApiResponse{
				Success: true,
				Data:    mockProjects[i],
				Message: "项目更新成功",
			})
			fmt.Printf("✅ 项目更新成功: %d\n", projectID)
			return
		}
	}

	c.JSON(404, ApiResponse{
		Success: false,
		Error:   "项目不存在",
	})
}

func handleDeleteProject(c *gin.Context) {
	id := c.Param("id")
	projectID, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(400, ApiResponse{
			Success: false,
			Error:   "项目ID格式错误",
		})
		return
	}

	fmt.Printf("🗑️ 删除项目请求: %d\n", projectID)

	for i, project := range mockProjects {
		if project.ID == projectID {
			// 删除项目相关的任务
			var remainingTasks []Task
			for _, task := range mockTasks {
				if task.ProjectID != projectID {
					remainingTasks = append(remainingTasks, task)
				}
			}
			mockTasks = remainingTasks

			// 删除项目
			mockProjects = append(mockProjects[:i], mockProjects[i+1:]...)

			c.JSON(200, ApiResponse{
				Success: true,
				Message: "项目删除成功",
			})
			fmt.Printf("✅ 项目删除成功: %d\n", projectID)
			return
		}
	}

	c.JSON(404, ApiResponse{
		Success: false,
		Error:   "项目不存在",
	})
}

func handleGetProjectTasks(c *gin.Context) {
	id := c.Param("id")
	fmt.Printf("📋 获取项目任务请求: %s\n", id)
	
	projectID, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(400, ApiResponse{
			Success: false,
			Error:   "项目ID格式错误",
		})
		return
	}

	var projectTasks []Task
	for _, task := range mockTasks {
		if task.ProjectID == projectID {
			projectTasks = append(projectTasks, task)
		}
	}

	c.JSON(200, ApiResponse{
		Success: true,
		Data:    projectTasks,
		Message: "获取项目任务成功",
	})
}

// 任务相关处理器
func handleCreateTask(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, ApiResponse{
			Success: false,
			Error:   "请求数据格式错误",
		})
		return
	}

	projectIDStr := c.Query("project_id")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		c.JSON(400, ApiResponse{
			Success: false,
			Error:   "项目ID格式错误",
		})
		return
	}

	fmt.Printf("📝 创建任务请求: %s (项目: %d)\n", req.Title, projectID)

	// 查找项目获取Key
	var projectKey string
	for _, project := range mockProjects {
		if project.ID == projectID {
			projectKey = project.Key
			break
		}
	}

	if projectKey == "" {
		c.JSON(404, ApiResponse{
			Success: false,
			Error:   "项目不存在",
		})
		return
	}

	newTask := Task{
		ID:           nextTaskID,
		ProjectID:    projectID,
		Title:        req.Title,
		Description:  req.Description,
		TaskNumber:   fmt.Sprintf("%s-%d", projectKey, nextTaskID),
		StatusID:     req.StatusID,
		Priority:     req.Priority,
		AssigneeID:   req.AssigneeID,
		DueDate:      req.DueDate,
		CreatedAt:    time.Now().Format(time.RFC3339),
		UpdatedAt:    time.Now().Format(time.RFC3339),
	}

	mockTasks = append(mockTasks, newTask)
	nextTaskID++

	// 更新项目任务计数
	for i, project := range mockProjects {
		if project.ID == projectID {
			mockProjects[i].TasksCount++
			break
		}
	}

	c.JSON(201, ApiResponse{
		Success: true,
		Data:    newTask,
		Message: "任务创建成功",
	})
	fmt.Printf("✅ 任务创建成功: %s (ID: %d)\n", req.Title, newTask.ID)
}

func handleGetTask(c *gin.Context) {
	id := c.Param("id")
	taskID, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(400, ApiResponse{
			Success: false,
			Error:   "任务ID格式错误",
		})
		return
	}

	fmt.Printf("📄 获取任务详情请求: %d\n", taskID)

	for _, task := range mockTasks {
		if task.ID == taskID {
			c.JSON(200, ApiResponse{
				Success: true,
				Data:    task,
				Message: "获取任务详情成功",
			})
			return
		}
	}

	c.JSON(404, ApiResponse{
		Success: false,
		Error:   "任务不存在",
	})
}

func handleUpdateTask(c *gin.Context) {
	id := c.Param("id")
	taskID, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(400, ApiResponse{
			Success: false,
			Error:   "任务ID格式错误",
		})
		return
	}

	var req UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, ApiResponse{
			Success: false,
			Error:   "请求数据格式错误",
		})
		return
	}

	fmt.Printf("✏️ 更新任务请求: %d\n", taskID)

	for i, task := range mockTasks {
		if task.ID == taskID {
			mockTasks[i].Title = req.Title
			mockTasks[i].Description = req.Description
			mockTasks[i].Priority = req.Priority
			mockTasks[i].StatusID = req.StatusID
			mockTasks[i].AssigneeID = req.AssigneeID
			mockTasks[i].DueDate = req.DueDate
			mockTasks[i].UpdatedAt = time.Now().Format(time.RFC3339)

			c.JSON(200, ApiResponse{
				Success: true,
				Data:    mockTasks[i],
				Message: "任务更新成功",
			})
			fmt.Printf("✅ 任务更新成功: %d\n", taskID)
			return
		}
	}

	c.JSON(404, ApiResponse{
		Success: false,
		Error:   "任务不存在",
	})
}

func handleDeleteTask(c *gin.Context) {
	id := c.Param("id")
	taskID, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(400, ApiResponse{
			Success: false,
			Error:   "任务ID格式错误",
		})
		return
	}

	fmt.Printf("🗑️ 删除任务请求: %d\n", taskID)

	for i, task := range mockTasks {
		if task.ID == taskID {
			projectID := task.ProjectID
			mockTasks = append(mockTasks[:i], mockTasks[i+1:]...)

			// 更新项目任务计数
			for j, project := range mockProjects {
				if project.ID == projectID {
					mockProjects[j].TasksCount--
					break
				}
			}

			c.JSON(200, ApiResponse{
				Success: true,
				Message: "任务删除成功",
			})
			fmt.Printf("✅ 任务删除成功: %d\n", taskID)
			return
		}
	}

	c.JSON(404, ApiResponse{
		Success: false,
		Error:   "任务不存在",
	})
}

// 用户相关处理器
func handleGetCurrentUser(c *gin.Context) {
	c.JSON(200, ApiResponse{
		Success: true,
		Data:    mockUser,
		Message: "获取当前用户成功",
	})
}

func handleGetUsers(c *gin.Context) {
	c.JSON(200, ApiResponse{
		Success: true,
		Data:    mockUsers,
		Message: "获取用户列表成功",
	})
}