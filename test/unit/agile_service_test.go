package unit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/models"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/service"
)

// AgileServiceTestSuite 敏捷开发服务测试套件
type AgileServiceTestSuite struct {
	suite.Suite
	db              *gorm.DB
	agileService    service.AgileService
	ctx             context.Context
	testTenantID    uuid.UUID
	testProjectID   uuid.UUID
	testUserID      uuid.UUID
	testAdminID     uuid.UUID
	testMemberID    uuid.UUID
	testSprintID    uuid.UUID
	testEpicID      uuid.UUID
	testTaskID      uuid.UUID
	testBoardID     uuid.UUID
	testColumnID    uuid.UUID
	testWorkLogID   uuid.UUID
	testCommentID   uuid.UUID
}

func (suite *AgileServiceTestSuite) SetupSuite() {
	// 使用内存SQLite数据库进行测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	suite.Require().NoError(err)

	// 创建表结构
	err = db.AutoMigrate(
		&models.Sprint{},
		&models.AgileTask{},
		&models.Epic{},
		&models.TaskComment{},
		&models.TaskAttachment{},
		&models.WorkLog{},
		&models.Board{},
		&models.BoardColumn{},
		&MockUser{},
		&MockProject{},
	)
	suite.Require().NoError(err)

	suite.db = db
	suite.agileService = service.NewAgileService(db, zap.NewNop())
	suite.ctx = context.Background()

	// 初始化测试ID
	suite.testTenantID = uuid.New()
	suite.testProjectID = uuid.New()
	suite.testUserID = uuid.New()
	suite.testAdminID = uuid.New()
	suite.testMemberID = uuid.New()
	suite.testSprintID = uuid.New()
	suite.testEpicID = uuid.New()
	suite.testTaskID = uuid.New()
	suite.testBoardID = uuid.New()
	suite.testColumnID = uuid.New()
	suite.testWorkLogID = uuid.New()
	suite.testCommentID = uuid.New()
}

func (suite *AgileServiceTestSuite) SetupTest() {
	// 清理测试数据
	suite.db.Exec("DELETE FROM sprints")
	suite.db.Exec("DELETE FROM agile_tasks")
	suite.db.Exec("DELETE FROM epics")
	suite.db.Exec("DELETE FROM task_comments")
	suite.db.Exec("DELETE FROM task_attachments")
	suite.db.Exec("DELETE FROM work_logs")
	suite.db.Exec("DELETE FROM boards")
	suite.db.Exec("DELETE FROM board_columns")
	suite.db.Exec("DELETE FROM mock_users")
	suite.db.Exec("DELETE FROM mock_projects")

	// 创建测试基础数据
	suite.createTestData()
}

func (suite *AgileServiceTestSuite) createTestData() {
	// 创建测试用户
	users := []*MockUser{
		{ID: suite.testUserID, Name: "Owner User", Email: "owner@example.com"},
		{ID: suite.testAdminID, Name: "Admin User", Email: "admin@example.com"},
		{ID: suite.testMemberID, Name: "Member User", Email: "member@example.com"},
	}
	for _, user := range users {
		suite.db.Create(user)
	}

	// 创建测试项目
	project := &MockProject{
		ID:        suite.testProjectID,
		Name:      "Test Project",
		ManagerID: suite.testUserID,
	}
	suite.db.Create(project)

	// 添加项目成员（模拟）
	members := []*MockProjectMember{
		{ProjectID: suite.testProjectID, UserID: suite.testUserID, Role: "owner"},
		{ProjectID: suite.testProjectID, UserID: suite.testAdminID, Role: "admin"},
		{ProjectID: suite.testProjectID, UserID: suite.testMemberID, Role: "member"},
	}
	for _, member := range members {
		suite.db.Create(member)
	}
}

// TestSprintManagement 测试Sprint管理
func (suite *AgileServiceTestSuite) TestSprintManagement() {
	suite.Run("创建Sprint", func() {
		req := &service.CreateSprintRequest{
			ProjectID:   suite.testProjectID,
			Name:        "Sprint 1",
			Description: stringPtr("First sprint"),
			Goal:        stringPtr("Complete initial features"),
			StartDate:   time.Now(),
			EndDate:     time.Now().Add(14 * 24 * time.Hour),
			Capacity:    20,
		}

		sprint, err := suite.agileService.CreateSprint(suite.ctx, req, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), sprint)
		assert.Equal(suite.T(), req.ProjectID, sprint.ProjectID)
		assert.Equal(suite.T(), req.Name, sprint.Name)
		assert.Equal(suite.T(), req.Description, sprint.Description)
		assert.Equal(suite.T(), req.Goal, sprint.Goal)
		assert.Equal(suite.T(), req.Capacity, sprint.Capacity)
		assert.Equal(suite.T(), models.SprintStatusPlanned, sprint.Status)
		assert.Equal(suite.T(), &suite.testUserID, sprint.CreatedBy)
		assert.WithinDuration(suite.T(), req.StartDate, sprint.StartDate, time.Second)
		assert.WithinDuration(suite.T(), req.EndDate, sprint.EndDate, time.Second)
	})

	suite.Run("获取Sprint", func() {
		// 先创建一个Sprint
		createReq := &service.CreateSprintRequest{
			ProjectID: suite.testProjectID,
			Name:      "Test Sprint",
			StartDate: time.Now(),
			EndDate:   time.Now().Add(7 * 24 * time.Hour),
			Capacity:  15,
		}
		createdSprint, err := suite.agileService.CreateSprint(suite.ctx, createReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 获取Sprint
		retrievedSprint, err := suite.agileService.GetSprint(suite.ctx, createdSprint.ID, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), retrievedSprint)
		assert.Equal(suite.T(), createdSprint.ID, retrievedSprint.ID)
		assert.Equal(suite.T(), createdSprint.Name, retrievedSprint.Name)
		assert.Equal(suite.T(), createdSprint.ProjectID, retrievedSprint.ProjectID)
	})

	suite.Run("列出Sprint", func() {
		// 创建多个Sprint
		for i := 0; i < 3; i++ {
			req := &service.CreateSprintRequest{
				ProjectID: suite.testProjectID,
				Name:      fmt.Sprintf("Sprint %d", i+1),
				StartDate: time.Now().Add(time.Duration(i) * 24 * time.Hour),
				EndDate:   time.Now().Add(time.Duration(i+7) * 24 * time.Hour),
				Capacity:  10,
			}
			_, err := suite.agileService.CreateSprint(suite.ctx, req, suite.testUserID, suite.testTenantID)
			suite.Require().NoError(err)
		}

		// 列出Sprint
		response, err := suite.agileService.ListSprints(suite.ctx, suite.testProjectID, 1, 10, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), response)
		assert.Len(suite.T(), response.Sprints, 3)
		assert.Equal(suite.T(), int64(3), response.Total)
		assert.Equal(suite.T(), 1, response.Page)
		assert.Equal(suite.T(), 10, response.PageSize)
	})

	suite.Run("更新Sprint", func() {
		// 创建Sprint
		createReq := &service.CreateSprintRequest{
			ProjectID: suite.testProjectID,
			Name:      "Original Sprint",
			StartDate: time.Now(),
			EndDate:   time.Now().Add(7 * 24 * time.Hour),
			Capacity:  10,
		}
		sprint, err := suite.agileService.CreateSprint(suite.ctx, createReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 更新Sprint
		updateReq := &service.UpdateSprintRequest{
			Name:        stringPtr("Updated Sprint"),
			Description: stringPtr("Updated description"),
			Goal:        stringPtr("Updated goal"),
			Capacity:    intPtr(25),
		}
		updatedSprint, err := suite.agileService.UpdateSprint(suite.ctx, sprint.ID, updateReq, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), updatedSprint)
		assert.Equal(suite.T(), *updateReq.Name, updatedSprint.Name)
		assert.Equal(suite.T(), updateReq.Description, updatedSprint.Description)
		assert.Equal(suite.T(), updateReq.Goal, updatedSprint.Goal)
		assert.Equal(suite.T(), *updateReq.Capacity, updatedSprint.Capacity)
	})

	suite.Run("删除Sprint", func() {
		// 创建Sprint
		createReq := &service.CreateSprintRequest{
			ProjectID: suite.testProjectID,
			Name:      "To Delete Sprint",
			StartDate: time.Now(),
			EndDate:   time.Now().Add(7 * 24 * time.Hour),
			Capacity:  10,
		}
		sprint, err := suite.agileService.CreateSprint(suite.ctx, createReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 删除Sprint
		err = suite.agileService.DeleteSprint(suite.ctx, sprint.ID, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)

		// 验证Sprint已删除
		_, err = suite.agileService.GetSprint(suite.ctx, sprint.ID, suite.testUserID, suite.testTenantID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "not found")
	})

	suite.Run("启动Sprint", func() {
		// 创建计划中的Sprint
		createReq := &service.CreateSprintRequest{
			ProjectID: suite.testProjectID,
			Name:      "To Start Sprint",
			StartDate: time.Now(),
			EndDate:   time.Now().Add(7 * 24 * time.Hour),
			Capacity:  10,
		}
		sprint, err := suite.agileService.CreateSprint(suite.ctx, createReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 启动Sprint
		err = suite.agileService.StartSprint(suite.ctx, sprint.ID, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)

		// 验证Sprint状态
		updatedSprint, err := suite.agileService.GetSprint(suite.ctx, sprint.ID, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), models.SprintStatusActive, updatedSprint.Status)
	})

	suite.Run("完成Sprint", func() {
		// 创建并启动Sprint
		createReq := &service.CreateSprintRequest{
			ProjectID: suite.testProjectID,
			Name:      "To Complete Sprint",
			StartDate: time.Now(),
			EndDate:   time.Now().Add(7 * 24 * time.Hour),
			Capacity:  10,
		}
		sprint, err := suite.agileService.CreateSprint(suite.ctx, createReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		err = suite.agileService.StartSprint(suite.ctx, sprint.ID, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 完成Sprint
		err = suite.agileService.CompleteSprint(suite.ctx, sprint.ID, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)

		// 验证Sprint状态
		updatedSprint, err := suite.agileService.GetSprint(suite.ctx, sprint.ID, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), models.SprintStatusClosed, updatedSprint.Status)
	})

	suite.Run("Sprint状态转换验证", func() {
		// 创建Sprint
		createReq := &service.CreateSprintRequest{
			ProjectID: suite.testProjectID,
			Name:      "Status Test Sprint",
			StartDate: time.Now(),
			EndDate:   time.Now().Add(7 * 24 * time.Hour),
			Capacity:  10,
		}
		sprint, err := suite.agileService.CreateSprint(suite.ctx, createReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 尝试完成计划中的Sprint（应该失败）
		err = suite.agileService.CompleteSprint(suite.ctx, sprint.ID, suite.testUserID, suite.testTenantID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "can only complete active sprint")

		// 尝试重复启动Sprint（应该失败）
		err = suite.agileService.StartSprint(suite.ctx, sprint.ID, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err) // 第一次启动成功

		err = suite.agileService.StartSprint(suite.ctx, sprint.ID, suite.testUserID, suite.testTenantID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "can only start planned sprint")
	})
}

// TestTaskManagement 测试任务管理
func (suite *AgileServiceTestSuite) TestTaskManagement() {
	suite.Run("创建任务", func() {
		req := &service.CreateTaskRequest{
			ProjectID:          suite.testProjectID,
			Title:              "Test Task",
			Description:        stringPtr("Task description"),
			Type:               models.TaskTypeStory,
			Priority:           models.PriorityHigh,
			StoryPoints:        intPtr(5),
			OriginalEstimate:   float64Ptr(8.0),
			AssigneeID:         &suite.testMemberID,
			Labels:             []string{"frontend", "urgent"},
			Components:         []string{"web", "api"},
			AcceptanceCriteria: []models.AcceptanceCriteria{
				{ID: "1", Description: "User can login", Completed: false},
				{ID: "2", Description: "User can logout", Completed: false},
			},
		}

		task, err := suite.agileService.CreateTask(suite.ctx, req, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), task)
		assert.Equal(suite.T(), req.ProjectID, task.ProjectID)
		assert.Equal(suite.T(), req.Title, task.Title)
		assert.Equal(suite.T(), req.Description, task.Description)
		assert.Equal(suite.T(), req.Type, task.Type)
		assert.Equal(suite.T(), req.Priority, task.Priority)
		assert.Equal(suite.T(), req.StoryPoints, task.StoryPoints)
		assert.Equal(suite.T(), req.OriginalEstimate, task.OriginalEstimate)
		assert.Equal(suite.T(), req.AssigneeID, task.AssigneeID)
		assert.Equal(suite.T(), suite.testUserID, task.ReporterID)
		assert.Equal(suite.T(), models.TaskStatusTodo, task.Status)
		assert.Equal(suite.T(), req.Labels, task.Labels)
		assert.Equal(suite.T(), req.Components, task.Components)
		assert.Equal(suite.T(), req.AcceptanceCriteria, task.AcceptanceCriteria)
		assert.NotEmpty(suite.T(), task.Rank)
	})

	suite.Run("获取任务", func() {
		// 创建任务
		createReq := &service.CreateTaskRequest{
			ProjectID: suite.testProjectID,
			Title:     "Test Get Task",
			Type:      models.TaskTypeTask,
			Priority:  models.PriorityMedium,
		}
		task, err := suite.agileService.CreateTask(suite.ctx, createReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 获取任务
		retrievedTask, err := suite.agileService.GetTask(suite.ctx, task.ID, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), retrievedTask)
		assert.Equal(suite.T(), task.ID, retrievedTask.ID)
		assert.Equal(suite.T(), task.Title, retrievedTask.Title)
		assert.Equal(suite.T(), task.ProjectID, retrievedTask.ProjectID)
	})

	suite.Run("列出任务", func() {
		// 创建多个任务
		for i := 0; i < 5; i++ {
			req := &service.CreateTaskRequest{
				ProjectID: suite.testProjectID,
				Title:     fmt.Sprintf("Task %d", i+1),
				Type:      models.TaskTypeStory,
				Priority:  models.PriorityMedium,
			}
			_, err := suite.agileService.CreateTask(suite.ctx, req, suite.testUserID, suite.testTenantID)
			suite.Require().NoError(err)
		}

		// 列出任务
		filter := &service.TaskFilter{
			ProjectID: suite.testProjectID,
		}
		response, err := suite.agileService.ListTasks(suite.ctx, filter, 1, 10, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), response)
		assert.Len(suite.T(), response.Tasks, 5)
		assert.Equal(suite.T(), int64(5), response.Total)
		assert.Equal(suite.T(), 1, response.Page)
		assert.Equal(suite.T(), 10, response.PageSize)
	})

	suite.Run("更新任务", func() {
		// 创建任务
		createReq := &service.CreateTaskRequest{
			ProjectID: suite.testProjectID,
			Title:     "Original Task",
			Type:      models.TaskTypeStory,
			Priority:  models.PriorityLow,
		}
		task, err := suite.agileService.CreateTask(suite.ctx, createReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 更新任务
		updateReq := &service.UpdateTaskRequest{
			Title:            stringPtr("Updated Task"),
			Description:      stringPtr("Updated description"),
			Status:           stringPtr(models.TaskStatusInProgress),
			Priority:         stringPtr(models.PriorityHigh),
			StoryPoints:      intPtr(8),
			OriginalEstimate: float64Ptr(12.0),
			RemainingTime:    float64Ptr(10.0),
			AssigneeID:       &suite.testAdminID,
		}
		updatedTask, err := suite.agileService.UpdateTask(suite.ctx, task.ID, updateReq, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), updatedTask)
		assert.Equal(suite.T(), *updateReq.Title, updatedTask.Title)
		assert.Equal(suite.T(), updateReq.Description, updatedTask.Description)
		assert.Equal(suite.T(), *updateReq.Status, updatedTask.Status)
		assert.Equal(suite.T(), *updateReq.Priority, updatedTask.Priority)
		assert.Equal(suite.T(), updateReq.StoryPoints, updatedTask.StoryPoints)
		assert.Equal(suite.T(), updateReq.OriginalEstimate, updatedTask.OriginalEstimate)
		assert.Equal(suite.T(), updateReq.RemainingTime, updatedTask.RemainingTime)
		assert.Equal(suite.T(), updateReq.AssigneeID, updatedTask.AssigneeID)
	})

	suite.Run("删除任务", func() {
		// 创建任务
		createReq := &service.CreateTaskRequest{
			ProjectID: suite.testProjectID,
			Title:     "To Delete Task",
			Type:      models.TaskTypeTask,
			Priority:  models.PriorityMedium,
		}
		task, err := suite.agileService.CreateTask(suite.ctx, createReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 删除任务
		err = suite.agileService.DeleteTask(suite.ctx, task.ID, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)

		// 验证任务已删除
		_, err = suite.agileService.GetTask(suite.ctx, task.ID, suite.testUserID, suite.testTenantID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "not found")
	})

	suite.Run("任务状态转换", func() {
		// 创建任务
		createReq := &service.CreateTaskRequest{
			ProjectID: suite.testProjectID,
			Title:     "Status Test Task",
			Type:      models.TaskTypeStory,
			Priority:  models.PriorityMedium,
		}
		task, err := suite.agileService.CreateTask(suite.ctx, createReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 转换到进行中
		err = suite.agileService.TransitionTask(suite.ctx, task.ID, models.TaskStatusInProgress, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)

		// 验证状态更新
		updatedTask, err := suite.agileService.GetTask(suite.ctx, task.ID, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), models.TaskStatusInProgress, updatedTask.Status)

		// 尝试非法状态转换
		err = suite.agileService.TransitionTask(suite.ctx, task.ID, models.TaskStatusCancelled, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err) // 从in_progress到cancelled是合法的

		// 从cancelled转换到done（非法）
		err = suite.agileService.TransitionTask(suite.ctx, task.ID, models.TaskStatusDone, suite.testUserID, suite.testTenantID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "invalid status transition")
	})

	suite.Run("更新任务状态", func() {
		// 创建任务
		createReq := &service.CreateTaskRequest{
			ProjectID: suite.testProjectID,
			Title:     "Status Update Task",
			Type:      models.TaskTypeTask,
			Priority:  models.PriorityMedium,
		}
		task, err := suite.agileService.CreateTask(suite.ctx, createReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 更新任务状态
		err = suite.agileService.UpdateTaskStatus(suite.ctx, task.ID, models.TaskStatusInProgress, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)

		// 验证状态更新
		updatedTask, err := suite.agileService.GetTask(suite.ctx, task.ID, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), models.TaskStatusInProgress, updatedTask.Status)
	})

	suite.Run("分配任务", func() {
		// 创建任务
		createReq := &service.CreateTaskRequest{
			ProjectID: suite.testProjectID,
			Title:     "Assignment Task",
			Type:      models.TaskTypeTask,
			Priority:  models.PriorityMedium,
		}
		task, err := suite.agileService.CreateTask(suite.ctx, createReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 分配任务
		err = suite.agileService.AssignTask(suite.ctx, task.ID, &suite.testMemberID, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)

		// 验证分配
		updatedTask, err := suite.agileService.GetTask(suite.ctx, task.ID, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), &suite.testMemberID, updatedTask.AssigneeID)

		// 取消分配
		err = suite.agileService.AssignTask(suite.ctx, task.ID, nil, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)

		updatedTask, err = suite.agileService.GetTask(suite.ctx, task.ID, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)
		assert.Nil(suite.T(), updatedTask.AssigneeID)
	})
}

// TestEpicManagement 测试史诗管理
func (suite *AgileServiceTestSuite) TestEpicManagement() {
	suite.Run("创建史诗", func() {
		startDate := time.Now()
		endDate := time.Now().Add(30 * 24 * time.Hour)
		
		req := &service.CreateEpicRequest{
			ProjectID:       suite.testProjectID,
			Name:            "User Authentication Epic",
			Description:     stringPtr("Implement complete user authentication system"),
			Color:           stringPtr("#FF5722"),
			StartDate:       &startDate,
			EndDate:         &endDate,
			Goal:            stringPtr("Secure user access management"),
			SuccessCriteria: stringPtr("Users can login, logout, and manage profiles securely"),
		}

		epic, err := suite.agileService.CreateEpic(suite.ctx, req, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), epic)
		assert.Equal(suite.T(), req.ProjectID, epic.ProjectID)
		assert.Equal(suite.T(), req.Name, epic.Name)
		assert.Equal(suite.T(), req.Description, epic.Description)
		assert.Equal(suite.T(), req.Color, epic.Color)
		assert.Equal(suite.T(), req.Goal, epic.Goal)
		assert.Equal(suite.T(), req.SuccessCriteria, epic.SuccessCriteria)
		assert.Equal(suite.T(), models.EpicStatusOpen, epic.Status)
		assert.Equal(suite.T(), &suite.testUserID, epic.CreatedBy)
		assert.WithinDuration(suite.T(), startDate, *epic.StartDate, time.Second)
		assert.WithinDuration(suite.T(), endDate, *epic.EndDate, time.Second)
	})

	suite.Run("获取史诗", func() {
		// 创建史诗
		createReq := &service.CreateEpicRequest{
			ProjectID: suite.testProjectID,
			Name:      "Test Epic",
		}
		epic, err := suite.agileService.CreateEpic(suite.ctx, createReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 获取史诗
		retrievedEpic, err := suite.agileService.GetEpic(suite.ctx, epic.ID, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), retrievedEpic)
		assert.Equal(suite.T(), epic.ID, retrievedEpic.ID)
		assert.Equal(suite.T(), epic.Name, retrievedEpic.Name)
		assert.Equal(suite.T(), epic.ProjectID, retrievedEpic.ProjectID)
	})

	suite.Run("列出史诗", func() {
		// 创建多个史诗
		for i := 0; i < 4; i++ {
			req := &service.CreateEpicRequest{
				ProjectID: suite.testProjectID,
				Name:      fmt.Sprintf("Epic %d", i+1),
			}
			_, err := suite.agileService.CreateEpic(suite.ctx, req, suite.testUserID, suite.testTenantID)
			suite.Require().NoError(err)
		}

		// 列出史诗
		response, err := suite.agileService.ListEpics(suite.ctx, suite.testProjectID, 1, 10, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), response)
		assert.Len(suite.T(), response.Epics, 4)
		assert.Equal(suite.T(), int64(4), response.Total)
		assert.Equal(suite.T(), 1, response.Page)
		assert.Equal(suite.T(), 10, response.PageSize)
	})

	suite.Run("更新史诗", func() {
		// 创建史诗
		createReq := &service.CreateEpicRequest{
			ProjectID: suite.testProjectID,
			Name:      "Original Epic",
		}
		epic, err := suite.agileService.CreateEpic(suite.ctx, createReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 更新史诗
		endDate := time.Now().Add(60 * 24 * time.Hour)
		updateReq := &service.UpdateEpicRequest{
			Name:            stringPtr("Updated Epic"),
			Description:     stringPtr("Updated epic description"),
			Status:          stringPtr(models.EpicStatusInProgress),
			Color:           stringPtr("#2196F3"),
			EndDate:         &endDate,
			Goal:            stringPtr("Updated goal"),
			SuccessCriteria: stringPtr("Updated success criteria"),
		}
		updatedEpic, err := suite.agileService.UpdateEpic(suite.ctx, epic.ID, updateReq, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), updatedEpic)
		assert.Equal(suite.T(), *updateReq.Name, updatedEpic.Name)
		assert.Equal(suite.T(), updateReq.Description, updatedEpic.Description)
		assert.Equal(suite.T(), *updateReq.Status, updatedEpic.Status)
		assert.Equal(suite.T(), updateReq.Color, updatedEpic.Color)
		assert.Equal(suite.T(), updateReq.Goal, updatedEpic.Goal)
		assert.Equal(suite.T(), updateReq.SuccessCriteria, updatedEpic.SuccessCriteria)
		assert.WithinDuration(suite.T(), endDate, *updatedEpic.EndDate, time.Second)
	})

	suite.Run("删除史诗", func() {
		// 创建史诗
		createReq := &service.CreateEpicRequest{
			ProjectID: suite.testProjectID,
			Name:      "To Delete Epic",
		}
		epic, err := suite.agileService.CreateEpic(suite.ctx, createReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 删除史诗
		err = suite.agileService.DeleteEpic(suite.ctx, epic.ID, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)

		// 验证史诗已删除
		_, err = suite.agileService.GetEpic(suite.ctx, epic.ID, suite.testUserID, suite.testTenantID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "not found")
	})
}

// TestBoardManagement 测试看板管理
func (suite *AgileServiceTestSuite) TestBoardManagement() {
	suite.Run("创建看板", func() {
		req := &service.CreateBoardRequest{
			ProjectID:   suite.testProjectID,
			Name:        "Development Board",
			Description: stringPtr("Main development kanban board"),
			Type:        "kanban",
		}

		board, err := suite.agileService.CreateBoard(suite.ctx, req, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), board)
		assert.Equal(suite.T(), req.ProjectID, board.ProjectID)
		assert.Equal(suite.T(), req.Name, board.Name)
		assert.Equal(suite.T(), req.Description, board.Description)
		assert.Equal(suite.T(), req.Type, board.Type)
		assert.Equal(suite.T(), &suite.testUserID, board.CreatedBy)
	})

	suite.Run("获取看板", func() {
		// 创建看板
		createReq := &service.CreateBoardRequest{
			ProjectID: suite.testProjectID,
			Name:      "Test Board",
			Type:      "scrum",
		}
		board, err := suite.agileService.CreateBoard(suite.ctx, createReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 获取看板
		retrievedBoard, err := suite.agileService.GetBoard(suite.ctx, board.ID, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), retrievedBoard)
		assert.Equal(suite.T(), board.ID, retrievedBoard.ID)
		assert.Equal(suite.T(), board.Name, retrievedBoard.Name)
		assert.Equal(suite.T(), board.ProjectID, retrievedBoard.ProjectID)
	})

	suite.Run("列出看板", func() {
		// 创建多个看板
		for i := 0; i < 3; i++ {
			req := &service.CreateBoardRequest{
				ProjectID: suite.testProjectID,
				Name:      fmt.Sprintf("Board %d", i+1),
				Type:      "kanban",
			}
			_, err := suite.agileService.CreateBoard(suite.ctx, req, suite.testUserID, suite.testTenantID)
			suite.Require().NoError(err)
		}

		// 列出看板
		response, err := suite.agileService.ListBoards(suite.ctx, suite.testProjectID, 1, 10, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), response)
		assert.Len(suite.T(), response.Boards, 3)
		assert.Equal(suite.T(), int64(3), response.Total)
		assert.Equal(suite.T(), 1, response.Page)
		assert.Equal(suite.T(), 10, response.PageSize)
	})

	suite.Run("更新看板", func() {
		// 创建看板
		createReq := &service.CreateBoardRequest{
			ProjectID: suite.testProjectID,
			Name:      "Original Board",
			Type:      "kanban",
		}
		board, err := suite.agileService.CreateBoard(suite.ctx, createReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 更新看板
		updateReq := &service.UpdateBoardRequest{
			Name:        stringPtr("Updated Board"),
			Description: stringPtr("Updated board description"),
		}
		updatedBoard, err := suite.agileService.UpdateBoard(suite.ctx, board.ID, updateReq, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), updatedBoard)
		assert.Equal(suite.T(), *updateReq.Name, updatedBoard.Name)
		assert.Equal(suite.T(), updateReq.Description, updatedBoard.Description)
	})

	suite.Run("删除看板", func() {
		// 创建看板
		createReq := &service.CreateBoardRequest{
			ProjectID: suite.testProjectID,
			Name:      "To Delete Board",
			Type:      "kanban",
		}
		board, err := suite.agileService.CreateBoard(suite.ctx, createReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 删除看板
		err = suite.agileService.DeleteBoard(suite.ctx, board.ID, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)

		// 验证看板已删除
		_, err = suite.agileService.GetBoard(suite.ctx, board.ID, suite.testUserID, suite.testTenantID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "not found")
	})
}

// TestBoardColumnManagement 测试看板列管理
func (suite *AgileServiceTestSuite) TestBoardColumnManagement() {
	var testBoard *models.Board

	suite.Run("创建看板列", func() {
		// 先创建看板
		createBoardReq := &service.CreateBoardRequest{
			ProjectID: suite.testProjectID,
			Name:      "Column Test Board",
			Type:      "kanban",
		}
		board, err := suite.agileService.CreateBoard(suite.ctx, createBoardReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)
		testBoard = board

		// 创建看板列
		req := &service.CreateBoardColumnRequest{
			BoardID:  board.ID,
			Name:     "To Do",
			Position: 0,
			WIPLimit: intPtr(5),
			Status:   models.TaskStatusTodo,
			Color:    stringPtr("#FF9800"),
		}

		column, err := suite.agileService.CreateBoardColumn(suite.ctx, req, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), column)
		assert.Equal(suite.T(), req.BoardID, column.BoardID)
		assert.Equal(suite.T(), req.Name, column.Name)
		assert.Equal(suite.T(), req.Position, column.Position)
		assert.Equal(suite.T(), req.WIPLimit, column.WIPLimit)
		assert.Equal(suite.T(), req.Status, column.Status)
		assert.Equal(suite.T(), req.Color, column.Color)
	})

	suite.Run("更新看板列", func() {
		// 创建看板列
		createReq := &service.CreateBoardColumnRequest{
			BoardID:  testBoard.ID,
			Name:     "In Progress",
			Position: 1,
			Status:   models.TaskStatusInProgress,
		}
		column, err := suite.agileService.CreateBoardColumn(suite.ctx, createReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 更新看板列
		updateReq := &service.UpdateBoardColumnRequest{
			Name:     stringPtr("In Development"),
			Position: intPtr(2),
			WIPLimit: intPtr(3),
			Color:    stringPtr("#2196F3"),
		}
		updatedColumn, err := suite.agileService.UpdateBoardColumn(suite.ctx, column.ID, updateReq, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), updatedColumn)
		assert.Equal(suite.T(), *updateReq.Name, updatedColumn.Name)
		assert.Equal(suite.T(), *updateReq.Position, updatedColumn.Position)
		assert.Equal(suite.T(), updateReq.WIPLimit, updatedColumn.WIPLimit)
		assert.Equal(suite.T(), updateReq.Color, updatedColumn.Color)
	})

	suite.Run("删除看板列", func() {
		// 创建看板列
		createReq := &service.CreateBoardColumnRequest{
			BoardID:  testBoard.ID,
			Name:     "Done",
			Position: 3,
			Status:   models.TaskStatusDone,
		}
		column, err := suite.agileService.CreateBoardColumn(suite.ctx, createReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 删除看板列
		err = suite.agileService.DeleteBoardColumn(suite.ctx, column.ID, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)
	})

	suite.Run("重新排序看板列", func() {
		// 创建多个看板列
		var columnIDs []uuid.UUID
		for i := 0; i < 3; i++ {
			createReq := &service.CreateBoardColumnRequest{
				BoardID:  testBoard.ID,
				Name:     fmt.Sprintf("Column %d", i+1),
				Position: i,
				Status:   fmt.Sprintf("status_%d", i+1),
			}
			column, err := suite.agileService.CreateBoardColumn(suite.ctx, createReq, suite.testUserID, suite.testTenantID)
			suite.Require().NoError(err)
			columnIDs = append(columnIDs, column.ID)
		}

		// 重新排序
		reorderReq := &service.ReorderColumnsRequest{
			BoardID:   testBoard.ID,
			ColumnIDs: []uuid.UUID{columnIDs[2], columnIDs[0], columnIDs[1]}, // 反转顺序
		}
		err := suite.agileService.ReorderBoardColumns(suite.ctx, reorderReq, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)
	})
}

// TestWorkLogManagement 测试工作日志管理
func (suite *AgileServiceTestSuite) TestWorkLogManagement() {
	var testTask *models.AgileTask

	suite.Run("记录工作日志", func() {
		// 先创建任务
		createTaskReq := &service.CreateTaskRequest{
			ProjectID: suite.testProjectID,
			Title:     "Work Log Test Task",
			Type:      models.TaskTypeTask,
			Priority:  models.PriorityMedium,
		}
		task, err := suite.agileService.CreateTask(suite.ctx, createTaskReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)
		testTask = task

		// 记录工作日志
		workDate := time.Now().Truncate(24 * time.Hour)
		req := &service.LogWorkRequest{
			TaskID:      task.ID,
			TimeSpent:   4.5,
			Description: stringPtr("Implemented login functionality"),
			WorkDate:    workDate,
		}

		workLog, err := suite.agileService.LogWork(suite.ctx, req, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), workLog)
		assert.Equal(suite.T(), req.TaskID, workLog.TaskID)
		assert.Equal(suite.T(), suite.testUserID, workLog.UserID)
		assert.Equal(suite.T(), req.TimeSpent, workLog.TimeSpent)
		assert.Equal(suite.T(), req.Description, workLog.Description)
		assert.WithinDuration(suite.T(), workDate, workLog.WorkDate, time.Second)
	})

	suite.Run("获取工作日志", func() {
		// 创建多个工作日志
		for i := 0; i < 3; i++ {
			req := &service.LogWorkRequest{
				TaskID:      testTask.ID,
				TimeSpent:   float64(i + 1),
				Description: stringPtr(fmt.Sprintf("Work log %d", i+1)),
				WorkDate:    time.Now().AddDate(0, 0, -i),
			}
			_, err := suite.agileService.LogWork(suite.ctx, req, suite.testUserID, suite.testTenantID)
			suite.Require().NoError(err)
		}

		// 获取工作日志
		workLogs, err := suite.agileService.GetWorkLogs(suite.ctx, testTask.ID, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.Len(suite.T(), workLogs, 3)
		// 验证按日期倒序排列
		for i := 0; i < len(workLogs)-1; i++ {
			assert.True(suite.T(), workLogs[i].WorkDate.After(workLogs[i+1].WorkDate) || workLogs[i].WorkDate.Equal(workLogs[i+1].WorkDate))
		}
	})

	suite.Run("删除工作日志", func() {
		// 创建工作日志
		req := &service.LogWorkRequest{
			TaskID:    testTask.ID,
			TimeSpent: 2.0,
			WorkDate:  time.Now(),
		}
		workLog, err := suite.agileService.LogWork(suite.ctx, req, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 删除工作日志
		err = suite.agileService.DeleteWorkLog(suite.ctx, workLog.ID, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)
	})
}

// TestTaskCommentManagement 测试任务评论管理
func (suite *AgileServiceTestSuite) TestTaskCommentManagement() {
	var testTask *models.AgileTask

	suite.Run("添加任务评论", func() {
		// 先创建任务
		createTaskReq := &service.CreateTaskRequest{
			ProjectID: suite.testProjectID,
			Title:     "Comment Test Task",
			Type:      models.TaskTypeTask,
			Priority:  models.PriorityMedium,
		}
		task, err := suite.agileService.CreateTask(suite.ctx, createTaskReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)
		testTask = task

		// 添加评论
		req := &service.AddCommentRequest{
			TaskID:     task.ID,
			Content:    "This task looks good to me!",
			IsInternal: false,
		}

		comment, err := suite.agileService.AddComment(suite.ctx, req, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), comment)
		assert.Equal(suite.T(), req.TaskID, comment.TaskID)
		assert.Equal(suite.T(), suite.testUserID, comment.AuthorID)
		assert.Equal(suite.T(), req.Content, comment.Content)
		assert.Equal(suite.T(), req.IsInternal, comment.IsInternal)
	})

	suite.Run("获取任务评论", func() {
		// 创建多个评论
		for i := 0; i < 4; i++ {
			req := &service.AddCommentRequest{
				TaskID:     testTask.ID,
				Content:    fmt.Sprintf("Comment %d", i+1),
				IsInternal: i%2 == 0, // 交替设置内部/外部评论
			}
			_, err := suite.agileService.AddComment(suite.ctx, req, suite.testUserID, suite.testTenantID)
			suite.Require().NoError(err)
		}

		// 获取评论
		comments, err := suite.agileService.GetComments(suite.ctx, testTask.ID, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.Len(suite.T(), comments, 4)
		// 验证按创建时间正序排列
		for i := 0; i < len(comments)-1; i++ {
			assert.True(suite.T(), comments[i].CreatedAt.Before(comments[i+1].CreatedAt) || comments[i].CreatedAt.Equal(comments[i+1].CreatedAt))
		}
	})

	suite.Run("更新评论", func() {
		// 创建评论
		createReq := &service.AddCommentRequest{
			TaskID:  testTask.ID,
			Content: "Original comment",
		}
		comment, err := suite.agileService.AddComment(suite.ctx, createReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 更新评论
		newContent := "Updated comment content"
		updatedComment, err := suite.agileService.UpdateComment(suite.ctx, comment.ID, newContent, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), updatedComment)
		assert.Equal(suite.T(), newContent, updatedComment.Content)
	})

	suite.Run("删除评论", func() {
		// 创建评论
		createReq := &service.AddCommentRequest{
			TaskID:  testTask.ID,
			Content: "Comment to delete",
		}
		comment, err := suite.agileService.AddComment(suite.ctx, createReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 删除评论
		err = suite.agileService.DeleteComment(suite.ctx, comment.ID, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)
	})

	suite.Run("权限验证 - 只有作者能编辑评论", func() {
		// 创建评论
		createReq := &service.AddCommentRequest{
			TaskID:  testTask.ID,
			Content: "Author only comment",
		}
		comment, err := suite.agileService.AddComment(suite.ctx, createReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 尝试用其他用户更新评论
		_, err = suite.agileService.UpdateComment(suite.ctx, comment.ID, "Hacked content", suite.testAdminID, suite.testTenantID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "permission denied")

		// 尝试用其他用户删除评论
		err = suite.agileService.DeleteComment(suite.ctx, comment.ID, suite.testAdminID, suite.testTenantID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "permission denied")
	})
}

// TestTaskOrdering 测试任务排序
func (suite *AgileServiceTestSuite) TestTaskOrdering() {
	var taskIDs []uuid.UUID

	suite.Run("重新排序任务", func() {
		// 创建多个任务
		for i := 0; i < 5; i++ {
			req := &service.CreateTaskRequest{
				ProjectID: suite.testProjectID,
				Title:     fmt.Sprintf("Order Task %d", i+1),
				Type:      models.TaskTypeTask,
				Priority:  models.PriorityMedium,
			}
			task, err := suite.agileService.CreateTask(suite.ctx, req, suite.testUserID, suite.testTenantID)
			suite.Require().NoError(err)
			taskIDs = append(taskIDs, task.ID)
		}

		// 重新排序任务
		reorderReq := &service.ReorderTasksRequest{
			ProjectID: suite.testProjectID,
			Tasks: []service.TaskRankUpdate{
				{TaskID: taskIDs[0], Rank: "rank_1"},
				{TaskID: taskIDs[1], Rank: "rank_2"},
				{TaskID: taskIDs[2], Rank: "rank_3"},
				{TaskID: taskIDs[3], Rank: "rank_4"},
				{TaskID: taskIDs[4], Rank: "rank_5"},
			},
		}

		err := suite.agileService.ReorderTasks(suite.ctx, reorderReq, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)
	})

	suite.Run("移动任务", func() {
		if len(taskIDs) < 3 {
			suite.T().Skip("需要至少3个任务")
		}

		// 移动任务
		moveReq := &service.TaskMoveRequest{
			TaskID:       taskIDs[0],
			PrevTaskID:   &taskIDs[1],
			NextTaskID:   &taskIDs[2],
			TargetStatus: stringPtr(models.TaskStatusInProgress),
		}

		err := suite.agileService.MoveTask(suite.ctx, moveReq, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)

		// 验证任务状态更新
		movedTask, err := suite.agileService.GetTask(suite.ctx, taskIDs[0], suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), models.TaskStatusInProgress, movedTask.Status)
	})

	suite.Run("批量重新排序任务", func() {
		// 批量重新排序
		batchReorderReq := &service.BatchReorderRequest{
			ProjectID: suite.testProjectID,
			TaskIDs:   taskIDs,
		}

		err := suite.agileService.BatchReorderTasks(suite.ctx, batchReorderReq, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)
	})

	suite.Run("重新平衡任务rank", func() {
		err := suite.agileService.RebalanceTaskRanks(suite.ctx, suite.testProjectID, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)
	})

	suite.Run("验证任务排序", func() {
		err := suite.agileService.ValidateTaskOrder(suite.ctx, suite.testProjectID, suite.testUserID, suite.testTenantID)
		assert.NoError(suite.T(), err)
	})
}

// TestStatisticsAndReports 测试统计和报表
func (suite *AgileServiceTestSuite) TestStatisticsAndReports() {
	suite.Run("获取用户工作负载", func() {
		// 为用户创建一些任务
		for i := 0; i < 3; i++ {
			req := &service.CreateTaskRequest{
				ProjectID:        suite.testProjectID,
				Title:            fmt.Sprintf("Workload Task %d", i+1),
				Type:             models.TaskTypeTask,
				Priority:         models.PriorityMedium,
				AssigneeID:       &suite.testMemberID,
				OriginalEstimate: float64Ptr(float64((i + 1) * 2)),
			}
			_, err := suite.agileService.CreateTask(suite.ctx, req, suite.testUserID, suite.testTenantID)
			suite.Require().NoError(err)
		}

		// 获取用户工作负载
		workload, err := suite.agileService.GetUserWorkload(suite.ctx, suite.testMemberID, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), workload)
		assert.Equal(suite.T(), suite.testMemberID, workload.UserID)
		assert.Equal(suite.T(), 3, workload.TotalTasks)
		assert.Equal(suite.T(), 12, workload.TotalEstimatedHours) // 2+4+6=12
	})

	suite.Run("获取任务统计", func() {
		// 创建不同状态的任务
		statuses := []string{
			models.TaskStatusTodo,
			models.TaskStatusInProgress,
			models.TaskStatusDone,
			models.TaskStatusDone,
		}

		for i, status := range statuses {
			req := &service.CreateTaskRequest{
				ProjectID:   suite.testProjectID,
				Title:       fmt.Sprintf("Stats Task %d", i+1),
				Type:        models.TaskTypeStory,
				Priority:    models.PriorityMedium,
				StoryPoints: intPtr(3),
			}
			task, err := suite.agileService.CreateTask(suite.ctx, req, suite.testUserID, suite.testTenantID)
			suite.Require().NoError(err)

			// 更新任务状态
			if status != models.TaskStatusTodo {
				err = suite.agileService.UpdateTaskStatus(suite.ctx, task.ID, status, suite.testUserID, suite.testTenantID)
				suite.Require().NoError(err)
			}
		}

		// 获取任务统计
		stats, err := suite.agileService.GetTaskStatistics(suite.ctx, suite.testProjectID, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), stats)
		assert.Equal(suite.T(), suite.testProjectID, stats.ProjectID)
		assert.True(suite.T(), stats.TotalTasks >= 4)
		assert.True(suite.T(), stats.CompletedTasks >= 2)
		assert.True(suite.T(), stats.InProgressTasks >= 1)
		assert.True(suite.T(), stats.TotalStoryPoints >= 12)
		assert.True(suite.T(), stats.CompletedStoryPoints >= 6)
	})

	suite.Run("获取项目统计", func() {
		// 创建Sprint
		sprintReq := &service.CreateSprintRequest{
			ProjectID: suite.testProjectID,
			Name:      "Stats Sprint",
			StartDate: time.Now(),
			EndDate:   time.Now().Add(14 * 24 * time.Hour),
			Capacity:  20,
		}
		sprint, err := suite.agileService.CreateSprint(suite.ctx, sprintReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 启动Sprint
		err = suite.agileService.StartSprint(suite.ctx, sprint.ID, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 获取项目统计
		stats, err := suite.agileService.GetProjectStatistics(suite.ctx, suite.testProjectID, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), stats)
		assert.Equal(suite.T(), suite.testProjectID, stats.ProjectID)
		assert.True(suite.T(), stats.TotalSprints >= 1)
		assert.True(suite.T(), stats.ActiveSprints >= 1)
		assert.True(suite.T(), stats.TotalTasks >= 0)
		assert.True(suite.T(), stats.CompletionRate >= 0 && stats.CompletionRate <= 100)
	})

	suite.Run("获取Sprint燃尽图数据", func() {
		// 创建Sprint和任务
		sprintReq := &service.CreateSprintRequest{
			ProjectID: suite.testProjectID,
			Name:      "Burndown Sprint",
			StartDate: time.Now().Add(-7 * 24 * time.Hour),
			EndDate:   time.Now().Add(7 * 24 * time.Hour),
			Capacity:  30,
		}
		sprint, err := suite.agileService.CreateSprint(suite.ctx, sprintReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 为Sprint创建任务
		for i := 0; i < 5; i++ {
			taskReq := &service.CreateTaskRequest{
				ProjectID:   suite.testProjectID,
				SprintID:    &sprint.ID,
				Title:       fmt.Sprintf("Burndown Task %d", i+1),
				Type:        models.TaskTypeStory,
				Priority:    models.PriorityMedium,
				StoryPoints: intPtr(5),
			}
			_, err := suite.agileService.CreateTask(suite.ctx, taskReq, suite.testUserID, suite.testTenantID)
			suite.Require().NoError(err)
		}

		// 获取燃尽图数据
		burndownData, err := suite.agileService.GetSprintBurndown(suite.ctx, sprint.ID, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), burndownData)
		assert.Equal(suite.T(), sprint.ID, burndownData.SprintID)
		assert.Equal(suite.T(), sprint.Name, burndownData.SprintName)
		assert.NotEmpty(suite.T(), burndownData.DataPoints)
	})

	suite.Run("获取速度图数据", func() {
		// 创建已完成的Sprint
		sprintReq := &service.CreateSprintRequest{
			ProjectID: suite.testProjectID,
			Name:      "Velocity Sprint",
			StartDate: time.Now().Add(-21 * 24 * time.Hour),
			EndDate:   time.Now().Add(-7 * 24 * time.Hour),
			Capacity:  20,
		}
		sprint, err := suite.agileService.CreateSprint(suite.ctx, sprintReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 启动并完成Sprint
		err = suite.agileService.StartSprint(suite.ctx, sprint.ID, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		err = suite.agileService.CompleteSprint(suite.ctx, sprint.ID, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 为Sprint创建已完成的任务
		for i := 0; i < 3; i++ {
			taskReq := &service.CreateTaskRequest{
				ProjectID:   suite.testProjectID,
				SprintID:    &sprint.ID,
				Title:       fmt.Sprintf("Velocity Task %d", i+1),
				Type:        models.TaskTypeStory,
				Priority:    models.PriorityMedium,
				StoryPoints: intPtr(3),
			}
			task, err := suite.agileService.CreateTask(suite.ctx, taskReq, suite.testUserID, suite.testTenantID)
			suite.Require().NoError(err)

			// 标记为完成
			err = suite.agileService.UpdateTaskStatus(suite.ctx, task.ID, models.TaskStatusDone, suite.testUserID, suite.testTenantID)
			suite.Require().NoError(err)
		}

		// 获取速度图数据
		velocityData, err := suite.agileService.GetVelocityChart(suite.ctx, suite.testProjectID, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), velocityData)
		assert.Equal(suite.T(), suite.testProjectID, velocityData.ProjectID)
		assert.NotEmpty(suite.T(), velocityData.Sprints)
		assert.True(suite.T(), velocityData.Average >= 0)
	})
}

// TestBoardStatistics 测试看板统计
func (suite *AgileServiceTestSuite) TestBoardStatistics() {
	suite.Run("获取看板统计", func() {
		// 创建看板
		boardReq := &service.CreateBoardRequest{
			ProjectID: suite.testProjectID,
			Name:      "Statistics Board",
			Type:      "kanban",
		}
		board, err := suite.agileService.CreateBoard(suite.ctx, boardReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 创建看板列
		columnReq := &service.CreateBoardColumnRequest{
			BoardID:  board.ID,
			Name:     "Todo Column",
			Position: 0,
			WIPLimit: intPtr(3),
			Status:   models.TaskStatusTodo,
		}
		column, err := suite.agileService.CreateBoardColumn(suite.ctx, columnReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 创建任务
		for i := 0; i < 5; i++ {
			taskReq := &service.CreateTaskRequest{
				ProjectID: suite.testProjectID,
				Title:     fmt.Sprintf("Board Stats Task %d", i+1),
				Type:      models.TaskTypeTask,
				Priority:  models.PriorityMedium,
			}
			task, err := suite.agileService.CreateTask(suite.ctx, taskReq, suite.testUserID, suite.testTenantID)
			suite.Require().NoError(err)

			// 部分任务标记为完成
			if i >= 3 {
				err = suite.agileService.UpdateTaskStatus(suite.ctx, task.ID, models.TaskStatusDone, suite.testUserID, suite.testTenantID)
				suite.Require().NoError(err)
			}
		}

		// 获取看板统计
		stats, err := suite.agileService.GetBoardStatistics(suite.ctx, board.ID, suite.testUserID, suite.testTenantID)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), stats)
		assert.Equal(suite.T(), board.ID, stats.BoardID)
		assert.True(suite.T(), stats.TotalTasks >= 5)
		assert.True(suite.T(), stats.CompletedTasks >= 2)
		assert.NotEmpty(suite.T(), stats.ColumnStats)

		// 检查列统计
		for _, colStat := range stats.ColumnStats {
			if colStat.ColumnID == column.ID {
				assert.Equal(suite.T(), column.Name, colStat.ColumnName)
				assert.Equal(suite.T(), column.WIPLimit, colStat.WIPLimit)
				// 验证是否超出WIP限制
				if colStat.WIPLimit != nil && colStat.TaskCount > int64(*colStat.WIPLimit) {
					assert.True(suite.T(), colStat.IsOverLimit)
				}
			}
		}
	})
}

// TestAccessControl 测试访问控制
func (suite *AgileServiceTestSuite) TestAccessControl() {
	otherProjectID := uuid.New()
	
	// 创建其他项目（用户无权限）
	otherProject := &MockProject{
		ID:        otherProjectID,
		Name:      "Other Project",
		ManagerID: uuid.New(), // 不同的管理员
	}
	suite.db.Create(otherProject)

	suite.Run("无权限访问其他项目的Sprint", func() {
		// 尝试为无权限的项目创建Sprint
		req := &service.CreateSprintRequest{
			ProjectID: otherProjectID,
			Name:      "Unauthorized Sprint",
			StartDate: time.Now(),
			EndDate:   time.Now().Add(7 * 24 * time.Hour),
			Capacity:  10,
		}

		_, err := suite.agileService.CreateSprint(suite.ctx, req, suite.testUserID, suite.testTenantID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "no access to project")
	})

	suite.Run("无权限访问其他项目的任务", func() {
		// 尝试为无权限的项目创建任务
		req := &service.CreateTaskRequest{
			ProjectID: otherProjectID,
			Title:     "Unauthorized Task",
			Type:      models.TaskTypeTask,
			Priority:  models.PriorityMedium,
		}

		_, err := suite.agileService.CreateTask(suite.ctx, req, suite.testUserID, suite.testTenantID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "no access to project")
	})

	suite.Run("无权限访问其他项目的史诗", func() {
		// 尝试为无权限的项目创建史诗
		req := &service.CreateEpicRequest{
			ProjectID: otherProjectID,
			Name:      "Unauthorized Epic",
		}

		_, err := suite.agileService.CreateEpic(suite.ctx, req, suite.testUserID, suite.testTenantID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "no access to project")
	})
}

// TestErrorHandling 测试错误处理
func (suite *AgileServiceTestSuite) TestErrorHandling() {
	suite.Run("Sprint日期验证", func() {
		// 结束日期早于开始日期
		req := &service.CreateSprintRequest{
			ProjectID: suite.testProjectID,
			Name:      "Invalid Date Sprint",
			StartDate: time.Now().Add(7 * 24 * time.Hour),
			EndDate:   time.Now(), // 结束日期早于开始日期
			Capacity:  10,
		}

		_, err := suite.agileService.CreateSprint(suite.ctx, req, suite.testUserID, suite.testTenantID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "end date must be after start date")
	})

	suite.Run("获取不存在的资源", func() {
		nonExistentID := uuid.New()

		// 获取不存在的Sprint
		_, err := suite.agileService.GetSprint(suite.ctx, nonExistentID, suite.testUserID, suite.testTenantID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "not found")

		// 获取不存在的任务
		_, err = suite.agileService.GetTask(suite.ctx, nonExistentID, suite.testUserID, suite.testTenantID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "not found")

		// 获取不存在的史诗
		_, err = suite.agileService.GetEpic(suite.ctx, nonExistentID, suite.testUserID, suite.testTenantID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "not found")
	})

	suite.Run("无效的任务状态转换", func() {
		// 创建任务
		createReq := &service.CreateTaskRequest{
			ProjectID: suite.testProjectID,
			Title:     "Transition Test Task",
			Type:      models.TaskTypeTask,
			Priority:  models.PriorityMedium,
		}
		task, err := suite.agileService.CreateTask(suite.ctx, createReq, suite.testUserID, suite.testTenantID)
		suite.Require().NoError(err)

		// 尝试无效的状态转换（从todo直接到done）
		updateReq := &service.UpdateTaskRequest{
			Status: stringPtr(models.TaskStatusDone),
		}
		_, err = suite.agileService.UpdateTask(suite.ctx, task.ID, updateReq, suite.testUserID, suite.testTenantID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "invalid status transition")
	})
}

// 辅助函数和Mock类型
type MockUser struct {
	ID    uuid.UUID `gorm:"type:uuid;primary_key"`
	Name  string    `gorm:"size:255"`
	Email string    `gorm:"size:255"`
}

func (MockUser) TableName() string {
	return "mock_users"
}

type MockProject struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key"`
	Name      string    `gorm:"size:255"`
	ManagerID uuid.UUID `gorm:"type:uuid"`
}

func (MockProject) TableName() string {
	return "mock_projects"
}

type MockProjectMember struct {
	ProjectID uuid.UUID `gorm:"type:uuid;primary_key"`
	UserID    uuid.UUID `gorm:"type:uuid;primary_key"`
	Role      string    `gorm:"size:50"`
}

func (MockProjectMember) TableName() string {
	return "mock_project_members"
}

// 辅助函数
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

// 运行测试套件
func TestAgileServiceSuite(t *testing.T) {
	suite.Run(t, new(AgileServiceTestSuite))
}