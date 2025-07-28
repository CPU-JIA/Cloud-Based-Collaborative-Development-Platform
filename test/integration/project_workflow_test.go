package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Project 项目模型
type Project struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	OwnerID     int       `json:"owner_id"`
	TeamID      int       `json:"team_id"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Task 任务模型
type Task struct {
	ID          int        `json:"id"`
	ProjectID   int        `json:"project_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	AssigneeID  *int       `json:"assignee_id"`
	CreatorID   int        `json:"creator_id"`
	DueDate     *time.Time `json:"due_date"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Sprint Sprint模型
type Sprint struct {
	ID        int       `json:"id"`
	ProjectID int       `json:"project_id"`
	Name      string    `json:"name"`
	Goal      string    `json:"goal"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// WebSocketMessage WebSocket消息
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Action    string      `json:"action"`
	Data      interface{} `json:"data"`
	UserID    int         `json:"user_id"`
	Timestamp time.Time   `json:"timestamp"`
}

// ProjectService 项目服务接口
type ProjectService interface {
	CreateProject(name, description string, ownerID, teamID int) (*Project, error)
	GetProject(projectID, userID int) (*Project, error)
	UpdateProject(projectID int, updates map[string]interface{}, userID int) (*Project, error)
	DeleteProject(projectID, userID int) error

	CreateTask(projectID int, title, description, priority string, assigneeID *int, creatorID int) (*Task, error)
	UpdateTaskStatus(taskID int, status string, userID int) (*Task, error)
	MoveTask(taskID int, newStatus string, userID int) error

	CreateSprint(projectID int, name, goal string, startDate, endDate time.Time, userID int) (*Sprint, error)
	StartSprint(sprintID, userID int) error
	CompleteSprint(sprintID, userID int) error
}

// MockProjectService 模拟项目服务
type MockProjectService struct {
	projects      map[int]*Project
	tasks         map[int]*Task
	sprints       map[int]*Sprint
	nextProjectID int
	nextTaskID    int
	nextSprintID  int
	wsConnections map[int]*websocket.Conn
}

// NewMockProjectService 创建模拟项目服务
func NewMockProjectService() *MockProjectService {
	return &MockProjectService{
		projects:      make(map[int]*Project),
		tasks:         make(map[int]*Task),
		sprints:       make(map[int]*Sprint),
		nextProjectID: 1,
		nextTaskID:    1,
		nextSprintID:  1,
		wsConnections: make(map[int]*websocket.Conn),
	}
}

// CreateProject 创建项目
func (m *MockProjectService) CreateProject(name, description string, ownerID, teamID int) (*Project, error) {
	if name == "" {
		return nil, fmt.Errorf("项目名称不能为空")
	}

	project := &Project{
		ID:          m.nextProjectID,
		Name:        name,
		Description: description,
		OwnerID:     ownerID,
		TeamID:      teamID,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	m.projects[project.ID] = project
	m.nextProjectID++

	// 广播项目创建事件
	m.broadcastEvent("project", "created", project)

	return project, nil
}

// GetProject 获取项目
func (m *MockProjectService) GetProject(projectID, userID int) (*Project, error) {
	project, exists := m.projects[projectID]
	if !exists {
		return nil, fmt.Errorf("项目不存在")
	}

	// 简单的权限检查
	if project.OwnerID != userID && project.TeamID != userID {
		return nil, fmt.Errorf("无权限访问此项目")
	}

	return project, nil
}

// UpdateProject 更新项目
func (m *MockProjectService) UpdateProject(projectID int, updates map[string]interface{}, userID int) (*Project, error) {
	project, err := m.GetProject(projectID, userID)
	if err != nil {
		return nil, err
	}

	// 应用更新
	if name, ok := updates["name"].(string); ok {
		project.Name = name
	}
	if description, ok := updates["description"].(string); ok {
		project.Description = description
	}
	if status, ok := updates["status"].(string); ok {
		project.Status = status
	}

	project.UpdatedAt = time.Now()

	// 广播项目更新事件
	m.broadcastEvent("project", "updated", project)

	return project, nil
}

// DeleteProject 删除项目
func (m *MockProjectService) DeleteProject(projectID, userID int) error {
	project, err := m.GetProject(projectID, userID)
	if err != nil {
		return err
	}

	if project.OwnerID != userID {
		return fmt.Errorf("只有项目所有者可以删除项目")
	}

	delete(m.projects, projectID)

	// 删除相关任务
	for id, task := range m.tasks {
		if task.ProjectID == projectID {
			delete(m.tasks, id)
		}
	}

	// 删除相关Sprint
	for id, sprint := range m.sprints {
		if sprint.ProjectID == projectID {
			delete(m.sprints, id)
		}
	}

	// 广播项目删除事件
	m.broadcastEvent("project", "deleted", map[string]int{"project_id": projectID})

	return nil
}

// CreateTask 创建任务
func (m *MockProjectService) CreateTask(projectID int, title, description, priority string, assigneeID *int, creatorID int) (*Task, error) {
	// 检查项目是否存在
	if _, exists := m.projects[projectID]; !exists {
		return nil, fmt.Errorf("项目不存在")
	}

	task := &Task{
		ID:          m.nextTaskID,
		ProjectID:   projectID,
		Title:       title,
		Description: description,
		Status:      "todo",
		Priority:    priority,
		AssigneeID:  assigneeID,
		CreatorID:   creatorID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	m.tasks[task.ID] = task
	m.nextTaskID++

	// 广播任务创建事件
	m.broadcastEvent("task", "created", task)

	return task, nil
}

// UpdateTaskStatus 更新任务状态
func (m *MockProjectService) UpdateTaskStatus(taskID int, status string, userID int) (*Task, error) {
	task, exists := m.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("任务不存在")
	}

	// 验证状态值
	validStatuses := []string{"todo", "in_progress", "review", "done"}
	isValid := false
	for _, s := range validStatuses {
		if s == status {
			isValid = true
			break
		}
	}
	if !isValid {
		return nil, fmt.Errorf("无效的任务状态")
	}

	task.Status = status
	task.UpdatedAt = time.Now()

	// 广播任务更新事件
	m.broadcastEvent("task", "status_changed", map[string]interface{}{
		"task_id":    taskID,
		"old_status": task.Status,
		"new_status": status,
		"user_id":    userID,
	})

	return task, nil
}

// MoveTask 移动任务(拖拽)
func (m *MockProjectService) MoveTask(taskID int, newStatus string, userID int) error {
	_, err := m.UpdateTaskStatus(taskID, newStatus, userID)
	return err
}

// CreateSprint 创建Sprint
func (m *MockProjectService) CreateSprint(projectID int, name, goal string, startDate, endDate time.Time, userID int) (*Sprint, error) {
	// 检查项目是否存在
	if _, exists := m.projects[projectID]; !exists {
		return nil, fmt.Errorf("项目不存在")
	}

	// 检查日期有效性
	if endDate.Before(startDate) {
		return nil, fmt.Errorf("结束日期不能早于开始日期")
	}

	sprint := &Sprint{
		ID:        m.nextSprintID,
		ProjectID: projectID,
		Name:      name,
		Goal:      goal,
		StartDate: startDate,
		EndDate:   endDate,
		Status:    "planning",
		CreatedAt: time.Now(),
	}

	m.sprints[sprint.ID] = sprint
	m.nextSprintID++

	// 广播Sprint创建事件
	m.broadcastEvent("sprint", "created", sprint)

	return sprint, nil
}

// StartSprint 开始Sprint
func (m *MockProjectService) StartSprint(sprintID, userID int) error {
	sprint, exists := m.sprints[sprintID]
	if !exists {
		return fmt.Errorf("Sprint不存在")
	}

	if sprint.Status != "planning" {
		return fmt.Errorf("只能启动处于计划中的Sprint")
	}

	sprint.Status = "active"

	// 广播Sprint开始事件
	m.broadcastEvent("sprint", "started", sprint)

	return nil
}

// CompleteSprint 完成Sprint
func (m *MockProjectService) CompleteSprint(sprintID, userID int) error {
	sprint, exists := m.sprints[sprintID]
	if !exists {
		return fmt.Errorf("Sprint不存在")
	}

	if sprint.Status != "active" {
		return fmt.Errorf("只能完成活跃的Sprint")
	}

	sprint.Status = "completed"

	// 广播Sprint完成事件
	m.broadcastEvent("sprint", "completed", sprint)

	return nil
}

// broadcastEvent 广播事件
func (m *MockProjectService) broadcastEvent(entityType, action string, data interface{}) {
	message := WebSocketMessage{
		Type:      "event",
		Action:    fmt.Sprintf("%s_%s", entityType, action),
		Data:      data,
		Timestamp: time.Now(),
	}

	// 向所有连接的客户端广播
	for _, conn := range m.wsConnections {
		conn.WriteJSON(message)
	}
}

// setupProjectRouter 设置项目路由
func setupProjectRouter(projectService *MockProjectService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// 项目管理端点
	api := r.Group("/api/v1")

	// 创建项目
	api.POST("/projects", func(c *gin.Context) {
		var req struct {
			Name        string `json:"name" binding:"required"`
			Description string `json:"description"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 从上下文获取用户信息(这里简化处理)
		userID := 1
		teamID := 1

		project, err := projectService.CreateProject(req.Name, req.Description, userID, teamID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, project)
	})

	// 获取项目详情
	api.GET("/projects/:id", func(c *gin.Context) {
		projectID := 1 // 简化处理
		userID := 1

		project, err := projectService.GetProject(projectID, userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, project)
	})

	// 创建任务
	api.POST("/projects/:id/tasks", func(c *gin.Context) {
		var req struct {
			Title       string `json:"title" binding:"required"`
			Description string `json:"description"`
			Priority    string `json:"priority"`
			AssigneeID  *int   `json:"assignee_id"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		projectID := 1
		userID := 1

		task, err := projectService.CreateTask(projectID, req.Title, req.Description, req.Priority, req.AssigneeID, userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, task)
	})

	// 更新任务状态
	api.PUT("/tasks/:id/status", func(c *gin.Context) {
		var req struct {
			Status string `json:"status" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		taskID := 1
		userID := 1

		task, err := projectService.UpdateTaskStatus(taskID, req.Status, userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, task)
	})

	// WebSocket端点
	r.GET("/ws/projects/:id", func(c *gin.Context) {
		// 这里简化了WebSocket处理
		c.JSON(http.StatusOK, gin.H{"message": "WebSocket endpoint"})
	})

	return r
}

// TestProjectWorkflow 测试项目工作流
func TestProjectWorkflow(t *testing.T) {
	projectService := NewMockProjectService()
	router := setupProjectRouter(projectService)

	t.Run("完整项目工作流", func(t *testing.T) {
		// 1. 创建项目
		createReq := map[string]string{
			"name":        "测试项目",
			"description": "这是一个测试项目",
		}
		body, _ := json.Marshal(createReq)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/projects", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var project Project
		err := json.Unmarshal(w.Body.Bytes(), &project)
		require.NoError(t, err)
		assert.Equal(t, "测试项目", project.Name)
		assert.Equal(t, "active", project.Status)

		// 2. 创建Sprint
		startDate := time.Now()
		endDate := startDate.AddDate(0, 0, 14) // 2周后

		sprint, err := projectService.CreateSprint(project.ID, "Sprint 1", "完成基础功能", startDate, endDate, 1)
		require.NoError(t, err)
		assert.Equal(t, "planning", sprint.Status)

		// 3. 创建任务
		taskReq := map[string]interface{}{
			"title":       "实现用户认证",
			"description": "实现JWT认证机制",
			"priority":    "high",
		}
		body, _ = json.Marshal(taskReq)

		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%d/tasks", project.ID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var task Task
		err = json.Unmarshal(w.Body.Bytes(), &task)
		require.NoError(t, err)
		assert.Equal(t, "todo", task.Status)

		// 4. 启动Sprint
		err = projectService.StartSprint(sprint.ID, 1)
		require.NoError(t, err)

		// 5. 更新任务状态
		statusReq := map[string]string{
			"status": "in_progress",
		}
		body, _ = json.Marshal(statusReq)

		w = httptest.NewRecorder()
		req, _ = http.NewRequest("PUT", fmt.Sprintf("/api/v1/tasks/%d/status", task.ID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// 6. 完成任务
		updatedTask, err := projectService.UpdateTaskStatus(task.ID, "done", 1)
		task = *updatedTask
		require.NoError(t, err)
		assert.Equal(t, "done", task.Status)

		// 7. 完成Sprint
		err = projectService.CompleteSprint(sprint.ID, 1)
		require.NoError(t, err)
	})

	t.Run("任务状态流转", func(t *testing.T) {
		// 创建项目和任务
		project, _ := projectService.CreateProject("状态测试项目", "", 1, 1)
		task, _ := projectService.CreateTask(project.ID, "测试任务", "", "medium", nil, 1)

		// 测试状态流转
		states := []string{"in_progress", "review", "done"}
		for _, state := range states {
			task, err := projectService.UpdateTaskStatus(task.ID, state, 1)
			require.NoError(t, err)
			assert.Equal(t, state, task.Status)
		}
	})

	t.Run("项目权限控制", func(t *testing.T) {
		// 创建项目
		project, _ := projectService.CreateProject("私有项目", "", 1, 1)

		// 其他用户尝试访问
		_, err := projectService.GetProject(project.ID, 999)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "无权限")

		// 所有者可以访问
		retrieved, err := projectService.GetProject(project.ID, 1)
		require.NoError(t, err)
		assert.Equal(t, project.ID, retrieved.ID)
	})

	t.Run("Sprint管理", func(t *testing.T) {
		project, _ := projectService.CreateProject("Sprint测试项目", "", 1, 1)

		// 创建重叠的Sprint应该失败
		startDate := time.Now()
		endDate := startDate.AddDate(0, 0, 14)

		sprint1, err := projectService.CreateSprint(project.ID, "Sprint 1", "", startDate, endDate, 1)
		require.NoError(t, err)

		// 日期验证
		invalidEndDate := startDate.AddDate(0, 0, -1) // 结束日期早于开始日期
		_, err = projectService.CreateSprint(project.ID, "Invalid Sprint", "", startDate, invalidEndDate, 1)
		assert.Error(t, err)

		// Sprint状态管理
		err = projectService.CompleteSprint(sprint1.ID, 1) // 未开始就完成
		assert.Error(t, err)

		err = projectService.StartSprint(sprint1.ID, 1)
		require.NoError(t, err)

		err = projectService.StartSprint(sprint1.ID, 1) // 重复启动
		assert.Error(t, err)
	})

	t.Run("项目删除级联", func(t *testing.T) {
		// 创建项目及相关资源
		project, _ := projectService.CreateProject("待删除项目", "", 1, 1)
		task1, _ := projectService.CreateTask(project.ID, "任务1", "", "high", nil, 1)
		task2, _ := projectService.CreateTask(project.ID, "任务2", "", "low", nil, 1)
		sprint, _ := projectService.CreateSprint(project.ID, "Sprint", "", time.Now(), time.Now().AddDate(0, 0, 7), 1)

		// 删除项目
		err := projectService.DeleteProject(project.ID, 1)
		require.NoError(t, err)

		// 验证级联删除
		_, exists := projectService.projects[project.ID]
		assert.False(t, exists)

		_, exists = projectService.tasks[task1.ID]
		assert.False(t, exists)

		_, exists = projectService.tasks[task2.ID]
		assert.False(t, exists)

		_, exists = projectService.sprints[sprint.ID]
		assert.False(t, exists)
	})
}

// TestTaskManagement 测试任务管理功能
func TestTaskManagement(t *testing.T) {
	projectService := NewMockProjectService()

	// 创建测试项目
	project, _ := projectService.CreateProject("任务测试项目", "", 1, 1)

	t.Run("任务优先级管理", func(t *testing.T) {
		priorities := []string{"low", "medium", "high", "critical"}

		for _, priority := range priorities {
			task, err := projectService.CreateTask(project.ID, fmt.Sprintf("%s优先级任务", priority), "", priority, nil, 1)
			require.NoError(t, err)
			assert.Equal(t, priority, task.Priority)
		}
	})

	t.Run("任务分配", func(t *testing.T) {
		assigneeID := 2
		task, err := projectService.CreateTask(project.ID, "分配任务", "", "medium", &assigneeID, 1)
		require.NoError(t, err)
		assert.Equal(t, &assigneeID, task.AssigneeID)

		// 创建未分配任务
		unassignedTask, err := projectService.CreateTask(project.ID, "未分配任务", "", "low", nil, 1)
		require.NoError(t, err)
		assert.Nil(t, unassignedTask.AssigneeID)
	})

	t.Run("批量任务操作", func(t *testing.T) {
		// 创建多个任务
		var taskIDs []int
		for i := 0; i < 5; i++ {
			task, _ := projectService.CreateTask(project.ID, fmt.Sprintf("批量任务%d", i), "", "medium", nil, 1)
			taskIDs = append(taskIDs, task.ID)
		}

		// 批量更新状态
		for _, id := range taskIDs {
			_, err := projectService.UpdateTaskStatus(id, "in_progress", 1)
			require.NoError(t, err)
		}

		// 验证更新
		for _, id := range taskIDs {
			task := projectService.tasks[id]
			assert.Equal(t, "in_progress", task.Status)
		}
	})
}

// BenchmarkProjectOperations 性能基准测试
func BenchmarkProjectOperations(b *testing.B) {
	projectService := NewMockProjectService()

	b.Run("创建项目", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			projectService.CreateProject(fmt.Sprintf("项目%d", i), "描述", 1, 1)
		}
	})

	b.Run("创建任务", func(b *testing.B) {
		project, _ := projectService.CreateProject("基准测试项目", "", 1, 1)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			projectService.CreateTask(project.ID, fmt.Sprintf("任务%d", i), "", "medium", nil, 1)
		}
	})

	b.Run("更新任务状态", func(b *testing.B) {
		project, _ := projectService.CreateProject("基准测试项目", "", 1, 1)
		task, _ := projectService.CreateTask(project.ID, "测试任务", "", "high", nil, 1)

		statuses := []string{"todo", "in_progress", "review", "done"}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			status := statuses[i%len(statuses)]
			projectService.UpdateTaskStatus(task.ID, status, 1)
		}
	})
}
