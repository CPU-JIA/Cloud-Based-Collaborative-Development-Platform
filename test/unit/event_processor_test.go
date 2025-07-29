package unit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/repository"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/service"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/webhook"
)

// MockProjectRepository Mock项目存储库
type MockProjectRepository struct {
	mock.Mock
}

func (m *MockProjectRepository) CreateProject(ctx context.Context, project *repository.ProjectCreateData) (*repository.ProjectData, error) {
	args := m.Called(ctx, project)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.ProjectData), args.Error(1)
}

func (m *MockProjectRepository) GetProject(ctx context.Context, projectID, tenantID uuid.UUID) (*repository.ProjectData, error) {
	args := m.Called(ctx, projectID, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.ProjectData), args.Error(1)
}

func (m *MockProjectRepository) UpdateProject(ctx context.Context, projectID, tenantID uuid.UUID, updates *repository.ProjectUpdateData) (*repository.ProjectData, error) {
	args := m.Called(ctx, projectID, tenantID, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.ProjectData), args.Error(1)
}

func (m *MockProjectRepository) DeleteProject(ctx context.Context, projectID, tenantID uuid.UUID) error {
	args := m.Called(ctx, projectID, tenantID)
	return args.Error(0)
}

func (m *MockProjectRepository) ListProjects(ctx context.Context, tenantID uuid.UUID, filters *repository.ProjectFilters) ([]*repository.ProjectData, int64, error) {
	args := m.Called(ctx, tenantID, filters)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*repository.ProjectData), args.Get(1).(int64), args.Error(2)
}

// MockProjectService Mock项目服务
type MockProjectService struct {
	mock.Mock
}

func (m *MockProjectService) CreateProject(ctx context.Context, req *service.CreateProjectRequest) (*service.ProjectResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.ProjectResponse), args.Error(1)
}

func (m *MockProjectService) GetProject(ctx context.Context, projectID, tenantID uuid.UUID) (*service.ProjectResponse, error) {
	args := m.Called(ctx, projectID, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.ProjectResponse), args.Error(1)
}

func (m *MockProjectService) UpdateProject(ctx context.Context, req *service.UpdateProjectRequest) (*service.ProjectResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.ProjectResponse), args.Error(1)
}

func (m *MockProjectService) DeleteProject(ctx context.Context, projectID, tenantID uuid.UUID) error {
	args := m.Called(ctx, projectID, tenantID)
	return args.Error(0)
}

func (m *MockProjectService) ListProjects(ctx context.Context, req *service.ListProjectsRequest) (*service.ProjectListResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.ProjectListResponse), args.Error(1)
}

// EventProcessorTestSuite 事件处理器测试套件
type EventProcessorTestSuite struct {
	suite.Suite
	processor       *webhook.DefaultEventProcessor
	mockProjectRepo *MockProjectRepository
	mockProjectSvc  *MockProjectService
	logger          *zap.Logger
	testProjectID   uuid.UUID
	testRepositoryID uuid.UUID
	testUserID      uuid.UUID
}

func (suite *EventProcessorTestSuite) SetupSuite() {
	suite.logger = zaptest.NewLogger(suite.T())
	
	suite.testProjectID = uuid.New()
	suite.testRepositoryID = uuid.New()
	suite.testUserID = uuid.New()
}

func (suite *EventProcessorTestSuite) SetupTest() {
	suite.mockProjectRepo = new(MockProjectRepository)
	suite.mockProjectSvc = new(MockProjectService)
	suite.processor = webhook.NewDefaultEventProcessor(
		suite.mockProjectRepo, 
		suite.mockProjectSvc, 
		suite.logger,
	)
}

func (suite *EventProcessorTestSuite) TearDownTest() {
	suite.mockProjectRepo.AssertExpectations(suite.T())
	suite.mockProjectSvc.AssertExpectations(suite.T())
}

// TestRepositoryEventProcessing 测试仓库事件处理
func (suite *EventProcessorTestSuite) TestRepositoryEventProcessing() {
	testCases := []struct {
		name        string
		action      string
		description string
	}{
		{
			name:        "处理仓库创建事件",
			action:      "created",
			description: "应该正确记录仓库创建活动",
		},
		{
			name:        "处理仓库更新事件",
			action:      "updated",
			description: "应该正确记录仓库更新活动",
		},
		{
			name:        "处理仓库删除事件",
			action:      "deleted",
			description: "应该正确记录仓库删除活动",
		},
		{
			name:        "处理仓库归档事件",
			action:      "archived",
			description: "应该正确记录仓库归档活动",
		},
		{
			name:        "处理仓库取消归档事件",
			action:      "unarchived",
			description: "应该正确记录仓库取消归档活动",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			ctx := context.Background()
			
			gitEvent := &webhook.GitEvent{
				EventType:    "repository",
				EventID:      uuid.New().String(),
				Timestamp:    time.Now(),
				ProjectID:    suite.testProjectID.String(),
				RepositoryID: suite.testRepositoryID.String(),
			}

			repoEvent := &webhook.RepositoryEvent{
				Action: tc.action,
				Repository: struct {
					ID            string `json:"id"`
					Name          string `json:"name"`
					ProjectID     string `json:"project_id"`
					Visibility    string `json:"visibility"`
					DefaultBranch string `json:"default_branch"`
				}{
					ID:            suite.testRepositoryID.String(),
					Name:          "test-repository",
					ProjectID:     suite.testProjectID.String(),
					Visibility:    "private",
					DefaultBranch: "main",
				},
			}

			err := suite.processor.ProcessRepositoryEvent(ctx, gitEvent, repoEvent)
			
			assert.NoError(suite.T(), err, tc.description)
		})
	}
}

// TestBranchEventProcessing 测试分支事件处理
func (suite *EventProcessorTestSuite) TestBranchEventProcessing() {
	testCases := []struct {
		name        string
		action      string
		branchName  string
		description string
	}{
		{
			name:        "处理分支创建事件",
			action:      "created",
			branchName:  "feature/new-feature",
			description: "应该正确记录分支创建活动",
		},
		{
			name:        "处理分支删除事件",
			action:      "deleted",
			branchName:  "feature/old-feature",
			description: "应该正确记录分支删除活动",
		},
		{
			name:        "处理默认分支变更事件",
			action:      "default_changed",
			branchName:  "develop",
			description: "应该正确记录默认分支变更活动",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			ctx := context.Background()
			
			gitEvent := &webhook.GitEvent{
				EventType:    "branch",
				EventID:      uuid.New().String(),
				Timestamp:    time.Now(),
				ProjectID:    suite.testProjectID.String(),
				RepositoryID: suite.testRepositoryID.String(),
			}

			branchEvent := &webhook.BranchEvent{
				Action: tc.action,
				Branch: struct {
					Name         string `json:"name"`
					RepositoryID string `json:"repository_id"`
					Commit       string `json:"commit"`
				}{
					Name:         tc.branchName,
					RepositoryID: suite.testRepositoryID.String(),
					Commit:       "abc123def456",
				},
			}

			err := suite.processor.ProcessBranchEvent(ctx, gitEvent, branchEvent)
			
			assert.NoError(suite.T(), err, tc.description)
		})
	}
}

// TestCommitEventProcessing 测试提交事件处理
func (suite *EventProcessorTestSuite) TestCommitEventProcessing() {
	suite.Run("处理提交创建事件", func() {
		ctx := context.Background()
		
		gitEvent := &webhook.GitEvent{
			EventType:    "commit",
			EventID:      uuid.New().String(),
			Timestamp:    time.Now(),
			ProjectID:    suite.testProjectID.String(),
			RepositoryID: suite.testRepositoryID.String(),
		}

		commitEvent := &webhook.CommitEvent{
			Action: "created",
			Commit: struct {
				SHA          string    `json:"sha"`
				Message      string    `json:"message"`
				Author       string    `json:"author"`
				RepositoryID string    `json:"repository_id"`
				Branch       string    `json:"branch"`
				Timestamp    time.Time `json:"timestamp"`
			}{
				SHA:          "abc123def456789",
				Message:      "feat: implement new feature",
				Author:       "developer@example.com",
				RepositoryID: suite.testRepositoryID.String(),
				Branch:       "main",
				Timestamp:    time.Now(),
			},
		}

		err := suite.processor.ProcessCommitEvent(ctx, gitEvent, commitEvent)
		
		assert.NoError(suite.T(), err, "应该正确处理提交创建事件")
	})

	suite.Run("忽略未处理的提交事件", func() {
		ctx := context.Background()
		
		gitEvent := &webhook.GitEvent{
			EventType:    "commit",
			EventID:      uuid.New().String(),
			Timestamp:    time.Now(),
			ProjectID:    suite.testProjectID.String(),
			RepositoryID: suite.testRepositoryID.String(),
		}

		commitEvent := &webhook.CommitEvent{
			Action: "unknown_action",
		}

		err := suite.processor.ProcessCommitEvent(ctx, gitEvent, commitEvent)
		
		assert.NoError(suite.T(), err, "应该忽略未知的提交事件动作")
	})
}

// TestPushEventProcessing 测试推送事件处理
func (suite *EventProcessorTestSuite) TestPushEventProcessing() {
	suite.Run("处理推送事件", func() {
		ctx := context.Background()
		
		gitEvent := &webhook.GitEvent{
			EventType:    "push",
			EventID:      uuid.New().String(),
			Timestamp:    time.Now(),
			ProjectID:    suite.testProjectID.String(),
			RepositoryID: suite.testRepositoryID.String(),
		}

		pushEvent := &webhook.PushEvent{
			RepositoryID: suite.testRepositoryID.String(),
			Branch:       "main",
			Before:       "abc123",
			After:        "def456",
			Commits: []struct {
				SHA     string `json:"sha"`
				Message string `json:"message"`
				Author  string `json:"author"`
			}{
				{
					SHA:     "commit1sha",
					Message: "feat: add feature A",
					Author:  "developer@example.com",
				},
				{
					SHA:     "commit2sha",
					Message: "fix: fix bug in feature A",
					Author:  "developer@example.com",
				},
			},
			Pusher: "developer@example.com",
		}

		err := suite.processor.ProcessPushEvent(ctx, gitEvent, pushEvent)
		
		assert.NoError(suite.T(), err, "应该正确处理推送事件")
	})

	suite.Run("处理空提交推送事件", func() {
		ctx := context.Background()
		
		gitEvent := &webhook.GitEvent{
			EventType:    "push",
			EventID:      uuid.New().String(),
			Timestamp:    time.Now(),
			ProjectID:    suite.testProjectID.String(),
			RepositoryID: suite.testRepositoryID.String(),
		}

		pushEvent := &webhook.PushEvent{
			RepositoryID: suite.testRepositoryID.String(),
			Branch:       "main",
			Before:       "abc123",
			After:        "def456",
			Commits:      []struct {
				SHA     string `json:"sha"`
				Message string `json:"message"`
				Author  string `json:"author"`
			}{}, // 空提交列表
			Pusher: "developer@example.com",
		}

		err := suite.processor.ProcessPushEvent(ctx, gitEvent, pushEvent)
		
		assert.NoError(suite.T(), err, "应该正确处理空提交的推送事件")
	})
}

// TestTagEventProcessing 测试标签事件处理
func (suite *EventProcessorTestSuite) TestTagEventProcessing() {
	testCases := []struct {
		name        string
		action      string
		tagName     string
		description string
	}{
		{
			name:        "处理标签创建事件",
			action:      "created",
			tagName:     "v1.0.0",
			description: "应该正确记录标签创建活动",
		},
		{
			name:        "处理标签删除事件",
			action:      "deleted",
			tagName:     "v0.9.0",
			description: "应该正确记录标签删除活动",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			ctx := context.Background()
			
			gitEvent := &webhook.GitEvent{
				EventType:    "tag",
				EventID:      uuid.New().String(),
				Timestamp:    time.Now(),
				ProjectID:    suite.testProjectID.String(),
				RepositoryID: suite.testRepositoryID.String(),
			}

			tagEvent := &webhook.TagEvent{
				Action: tc.action,
				Tag: struct {
					Name         string `json:"name"`
					RepositoryID string `json:"repository_id"`
					Target       string `json:"target"`
					Message      string `json:"message,omitempty"`
				}{
					Name:         tc.tagName,
					RepositoryID: suite.testRepositoryID.String(),
					Target:       "commit123",
					Message:      "Release " + tc.tagName,
				},
			}

			err := suite.processor.ProcessTagEvent(ctx, gitEvent, tagEvent)
			
			assert.NoError(suite.T(), err, tc.description)
		})
	}

	suite.Run("忽略未处理的标签事件", func() {
		ctx := context.Background()
		
		gitEvent := &webhook.GitEvent{
			EventType:    "tag",
			EventID:      uuid.New().String(),
			Timestamp:    time.Now(),
			ProjectID:    suite.testProjectID.String(),
			RepositoryID: suite.testRepositoryID.String(),
		}

		tagEvent := &webhook.TagEvent{
			Action: "unknown_action",
		}

		err := suite.processor.ProcessTagEvent(ctx, gitEvent, tagEvent)
		
		assert.NoError(suite.T(), err, "应该忽略未知的标签事件动作")
	})
}

// TestErrorHandling 测试错误处理
func (suite *EventProcessorTestSuite) TestErrorHandling() {
	suite.Run("处理无效的项目ID", func() {
		ctx := context.Background()
		
		gitEvent := &webhook.GitEvent{
			EventType:    "repository",
			EventID:      uuid.New().String(),
			Timestamp:    time.Now(),
			ProjectID:    "invalid-uuid",
			RepositoryID: suite.testRepositoryID.String(),
		}

		repoEvent := &webhook.RepositoryEvent{
			Action: "created",
			Repository: struct {
				ID            string `json:"id"`
				Name          string `json:"name"`
				ProjectID     string `json:"project_id"`
				Visibility    string `json:"visibility"`
				DefaultBranch string `json:"default_branch"`
			}{
				ID:        suite.testRepositoryID.String(),
				Name:      "test-repository",
				ProjectID: "invalid-uuid",
			},
		}

		err := suite.processor.ProcessRepositoryEvent(ctx, gitEvent, repoEvent)
		
		assert.Error(suite.T(), err, "应该返回UUID解析错误")
		assert.Contains(suite.T(), err.Error(), "无效的项目ID")
	})

	suite.Run("处理无效的仓库ID", func() {
		ctx := context.Background()
		
		gitEvent := &webhook.GitEvent{
			EventType:    "branch",
			EventID:      uuid.New().String(),
			Timestamp:    time.Now(),
			ProjectID:    suite.testProjectID.String(),
			RepositoryID: "invalid-uuid",
		}

		branchEvent := &webhook.BranchEvent{
			Action: "created",
			Branch: struct {
				Name         string `json:"name"`
				RepositoryID string `json:"repository_id"`
				Commit       string `json:"commit"`
			}{
				Name:         "feature/test",
				RepositoryID: "invalid-uuid",
				Commit:       "abc123",
			},
		}

		err := suite.processor.ProcessBranchEvent(ctx, gitEvent, branchEvent)
		
		assert.Error(suite.T(), err, "应该返回UUID解析错误")
		assert.Contains(suite.T(), err.Error(), "无效的仓库ID")
	})
}

// TestContextHandling 测试上下文处理
func (suite *EventProcessorTestSuite) TestContextHandling() {
	suite.Run("处理取消的上下文", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // 立即取消上下文
		
		gitEvent := &webhook.GitEvent{
			EventType:    "repository",
			EventID:      uuid.New().String(),
			Timestamp:    time.Now(),
			ProjectID:    suite.testProjectID.String(),
			RepositoryID: suite.testRepositoryID.String(),
		}

		repoEvent := &webhook.RepositoryEvent{
			Action: "created",
			Repository: struct {
				ID            string `json:"id"`
				Name          string `json:"name"`
				ProjectID     string `json:"project_id"`
				Visibility    string `json:"visibility"`
				DefaultBranch string `json:"default_branch"`
			}{
				ID:        suite.testRepositoryID.String(),
				Name:      "test-repository",
				ProjectID: suite.testProjectID.String(),
			},
		}

		// 即使上下文被取消，当前实现仍应正常工作
		// 因为它不依赖于上下文的取消状态
		err := suite.processor.ProcessRepositoryEvent(ctx, gitEvent, repoEvent)
		
		assert.NoError(suite.T(), err, "当前实现应该能处理取消的上下文")
	})

	suite.Run("处理超时上下文", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		
		// 等待上下文超时
		time.Sleep(1 * time.Millisecond)
		
		gitEvent := &webhook.GitEvent{
			EventType:    "repository",
			EventID:      uuid.New().String(),
			Timestamp:    time.Now(),
			ProjectID:    suite.testProjectID.String(),
			RepositoryID: suite.testRepositoryID.String(),
		}

		repoEvent := &webhook.RepositoryEvent{
			Action: "created",
		}

		// 当前实现不检查上下文超时，所以仍应正常工作
		err := suite.processor.ProcessRepositoryEvent(ctx, gitEvent, repoEvent)
		
		assert.NoError(suite.T(), err, "当前实现应该能处理超时的上下文")
	})
}

// TestPerformanceMetrics 测试性能指标
func (suite *EventProcessorTestSuite) TestPerformanceMetrics() {
	suite.Run("测试事件处理性能", func() {
		ctx := context.Background()
		
		gitEvent := &webhook.GitEvent{
			EventType:    "repository",
			EventID:      uuid.New().String(),
			Timestamp:    time.Now(),
			ProjectID:    suite.testProjectID.String(),
			RepositoryID: suite.testRepositoryID.String(),
		}

		repoEvent := &webhook.RepositoryEvent{
			Action: "created",
			Repository: struct {
				ID            string `json:"id"`
				Name          string `json:"name"`
				ProjectID     string `json:"project_id"`
				Visibility    string `json:"visibility"`
				DefaultBranch string `json:"default_branch"`
			}{
				ID:        suite.testRepositoryID.String(),
				Name:      "test-repository",
				ProjectID: suite.testProjectID.String(),
			},
		}

		start := time.Now()
		err := suite.processor.ProcessRepositoryEvent(ctx, gitEvent, repoEvent)
		duration := time.Since(start)

		assert.NoError(suite.T(), err)
		assert.Less(suite.T(), duration, 100*time.Millisecond, "事件处理应该在100ms内完成")
	})

	suite.Run("测试批量事件处理性能", func() {
		ctx := context.Background()
		eventCount := 100
		
		start := time.Now()
		
		for i := 0; i < eventCount; i++ {
			gitEvent := &webhook.GitEvent{
				EventType:    "repository",
				EventID:      fmt.Sprintf("event-%d", i),
				Timestamp:    time.Now(),
				ProjectID:    suite.testProjectID.String(),
				RepositoryID: suite.testRepositoryID.String(),
			}

			repoEvent := &webhook.RepositoryEvent{
				Action: "created",
				Repository: struct {
					ID            string `json:"id"`
					Name          string `json:"name"`
					ProjectID     string `json:"project_id"`
					Visibility    string `json:"visibility"`
					DefaultBranch string `json:"default_branch"`
				}{
					ID:        suite.testRepositoryID.String(),
					Name:      fmt.Sprintf("test-repository-%d", i),
					ProjectID: suite.testProjectID.String(),
				},
			}

			err := suite.processor.ProcessRepositoryEvent(ctx, gitEvent, repoEvent)
			assert.NoError(suite.T(), err)
		}
		
		duration := time.Since(start)
		avgDuration := duration / time.Duration(eventCount)
		
		assert.Less(suite.T(), avgDuration, 10*time.Millisecond, "平均事件处理时间应该在10ms内")
	})
}

// 运行测试套件
func TestEventProcessorSuite(t *testing.T) {
	suite.Run(t, new(EventProcessorTestSuite))
}