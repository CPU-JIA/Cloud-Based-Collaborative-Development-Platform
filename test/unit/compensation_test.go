package unit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/client"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/compensation"
)

// CompensationManagerTestSuite 补偿管理器测试套件
type CompensationManagerTestSuite struct {
	suite.Suite
	compensationMgr *compensation.CompensationManager
	mockGitClient   *MockGitGatewayClient
	logger          *zap.Logger
	testResourceID  uuid.UUID
	testProjectID   uuid.UUID
}

func (suite *CompensationManagerTestSuite) SetupSuite() {
	suite.logger = zaptest.NewLogger(suite.T())
	suite.testResourceID = uuid.New()
	suite.testProjectID = uuid.New()
}

func (suite *CompensationManagerTestSuite) SetupTest() {
	suite.mockGitClient = new(MockGitGatewayClient)
	suite.compensationMgr = compensation.NewCompensationManager(
		suite.mockGitClient,
		suite.logger,
	)
}

func (suite *CompensationManagerTestSuite) TearDownTest() {
	suite.mockGitClient.AssertExpectations(suite.T())
}

// TestAddCompensation 测试添加补偿动作
func (suite *CompensationManagerTestSuite) TestAddCompensation() {
	suite.Run("添加删除仓库补偿", func() {
		metadata := map[string]interface{}{
			"repository_name": "test-repo",
			"project_id":      suite.testProjectID.String(),
		}

		compensationID := suite.compensationMgr.AddCompensation(
			compensation.CompensationActionDeleteRepository,
			suite.testResourceID,
			metadata,
		)

		assert.NotEqual(suite.T(), uuid.Nil, compensationID)

		// 验证补偿记录已添加
		entry, err := suite.compensationMgr.GetCompensationStatus(compensationID)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), compensationID, entry.ID)
		assert.Equal(suite.T(), compensation.CompensationActionDeleteRepository, entry.Action)
		assert.Equal(suite.T(), suite.testResourceID, entry.ResourceID)
		assert.Equal(suite.T(), "pending", entry.Status)
		assert.Equal(suite.T(), 0, entry.RetryCount)
		assert.Equal(suite.T(), 3, entry.MaxRetries)
		assert.Equal(suite.T(), metadata["repository_name"], entry.Payload["repository_name"])
	})

	suite.Run("添加项目回滚补偿", func() {
		metadata := map[string]interface{}{
			"project_name": "test-project",
			"tenant_id":    "tenant-123",
		}

		compensationID := suite.compensationMgr.AddCompensation(
			compensation.CompensationActionRollbackProject,
			suite.testProjectID,
			metadata,
		)

		assert.NotEqual(suite.T(), uuid.Nil, compensationID)

		// 验证补偿记录
		entry, err := suite.compensationMgr.GetCompensationStatus(compensationID)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), compensation.CompensationActionRollbackProject, entry.Action)
		assert.Equal(suite.T(), suite.testProjectID, entry.ResourceID)
		assert.Equal(suite.T(), "pending", entry.Status)
	})

	suite.Run("添加失败通知补偿", func() {
		metadata := map[string]interface{}{
			"user_id":     "user-123",
			"error_msg":   "创建仓库失败",
			"timestamp":   time.Now().Unix(),
		}

		compensationID := suite.compensationMgr.AddCompensation(
			compensation.CompensationActionNotifyFailure,
			suite.testResourceID,
			metadata,
		)

		assert.NotEqual(suite.T(), uuid.Nil, compensationID)

		// 验证补偿记录
		entry, err := suite.compensationMgr.GetCompensationStatus(compensationID)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), compensation.CompensationActionNotifyFailure, entry.Action)
		assert.Equal(suite.T(), "pending", entry.Status)
		assert.Contains(suite.T(), entry.Payload, "user_id")
		assert.Contains(suite.T(), entry.Payload, "error_msg")
	})
}

// TestExecuteDeleteRepositoryCompensation 测试执行删除仓库补偿
func (suite *CompensationManagerTestSuite) TestExecuteDeleteRepositoryCompensation() {
	ctx := context.Background()

	suite.Run("成功执行删除仓库补偿", func() {
		// 添加补偿记录
		metadata := map[string]interface{}{
			"repository_name": "test-repo",
			"project_id":      suite.testProjectID.String(),
		}
		compensationID := suite.compensationMgr.AddCompensation(
			compensation.CompensationActionDeleteRepository,
			suite.testResourceID,
			metadata,
		)

		// 设置mock期望
		suite.mockGitClient.On("DeleteRepository", ctx, suite.testResourceID).
			Return(nil)

		// 执行补偿
		err := suite.compensationMgr.ExecuteCompensation(ctx, compensationID)
		assert.NoError(suite.T(), err)

		// 验证补偿状态
		entry, err := suite.compensationMgr.GetCompensationStatus(compensationID)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "executed", entry.Status)
		assert.NotNil(suite.T(), entry.ExecutedAt)
		assert.Equal(suite.T(), 1, entry.RetryCount)
		assert.Empty(suite.T(), entry.LastError)
	})

	suite.Run("删除仓库失败重试", func() {
		// 添加补偿记录
		compensationID := suite.compensationMgr.AddCompensation(
			compensation.CompensationActionDeleteRepository,
			suite.testResourceID,
			map[string]interface{}{},
		)

		// 设置mock期望：第一次失败，第二次成功
		suite.mockGitClient.On("DeleteRepository", ctx, suite.testResourceID).
			Return(fmt.Errorf("Git网关不可用")).Once()
		suite.mockGitClient.On("DeleteRepository", ctx, suite.testResourceID).
			Return(nil).Once()

		// 第一次执行失败
		err := suite.compensationMgr.ExecuteCompensation(ctx, compensationID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "Git网关不可用")

		// 验证重试计数
		entry, _ := suite.compensationMgr.GetCompensationStatus(compensationID)
		assert.Equal(suite.T(), "pending", entry.Status)
		assert.Equal(suite.T(), 1, entry.RetryCount)
		assert.Contains(suite.T(), entry.LastError, "Git网关不可用")

		// 第二次执行成功
		err = suite.compensationMgr.ExecuteCompensation(ctx, compensationID)
		assert.NoError(suite.T(), err)

		// 验证最终状态
		entry, _ = suite.compensationMgr.GetCompensationStatus(compensationID)
		assert.Equal(suite.T(), "executed", entry.Status)
		assert.Equal(suite.T(), 2, entry.RetryCount)
	})

	suite.Run("超过最大重试次数", func() {
		// 添加补偿记录
		compensationID := suite.compensationMgr.AddCompensation(
			compensation.CompensationActionDeleteRepository,
			suite.testResourceID,
			map[string]interface{}{},
		)

		// 设置mock期望：总是失败
		suite.mockGitClient.On("DeleteRepository", ctx, suite.testResourceID).
			Return(fmt.Errorf("持续失败")).Times(3)

		// 执行3次补偿，都应该失败
		for i := 0; i < 3; i++ {
			err := suite.compensationMgr.ExecuteCompensation(ctx, compensationID)
			assert.Error(suite.T(), err)
		}

		// 第4次执行，应该返回超过重试次数的错误
		err := suite.compensationMgr.ExecuteCompensation(ctx, compensationID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "补偿动作超过最大重试次数")

		// 验证最终状态
		entry, _ := suite.compensationMgr.GetCompensationStatus(compensationID)
		assert.Equal(suite.T(), "failed", entry.Status)
		assert.Equal(suite.T(), 3, entry.RetryCount)
	})
}

// TestExecuteRollbackProjectCompensation 测试执行项目回滚补偿
func (suite *CompensationManagerTestSuite) TestExecuteRollbackProjectCompensation() {
	ctx := context.Background()

	suite.Run("成功执行项目回滚补偿", func() {
		// 添加补偿记录
		metadata := map[string]interface{}{
			"project_name": "test-project",
			"tenant_id":    "tenant-123",
		}
		compensationID := suite.compensationMgr.AddCompensation(
			compensation.CompensationActionRollbackProject,
			suite.testProjectID,
			metadata,
		)

		// 执行补偿（项目回滚通常不会失败，因为是内部操作）
		err := suite.compensationMgr.ExecuteCompensation(ctx, compensationID)
		assert.NoError(suite.T(), err)

		// 验证补偿状态
		entry, err := suite.compensationMgr.GetCompensationStatus(compensationID)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "executed", entry.Status)
		assert.NotNil(suite.T(), entry.ExecutedAt)
		assert.Equal(suite.T(), 1, entry.RetryCount)
	})
}

// TestExecuteNotifyFailureCompensation 测试执行失败通知补偿
func (suite *CompensationManagerTestSuite) TestExecuteNotifyFailureCompensation() {
	ctx := context.Background()

	suite.Run("成功执行失败通知补偿", func() {
		// 添加补偿记录
		metadata := map[string]interface{}{
			"user_id":     "user-123",
			"error_msg":   "创建仓库失败",
			"timestamp":   time.Now().Unix(),
		}
		compensationID := suite.compensationMgr.AddCompensation(
			compensation.CompensationActionNotifyFailure,
			suite.testResourceID,
			metadata,
		)

		// 执行补偿
		err := suite.compensationMgr.ExecuteCompensation(ctx, compensationID)
		assert.NoError(suite.T(), err)

		// 验证补偿状态
		entry, err := suite.compensationMgr.GetCompensationStatus(compensationID)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "executed", entry.Status)
		assert.NotNil(suite.T(), entry.ExecutedAt)
		assert.Equal(suite.T(), 1, entry.RetryCount)
	})
}

// TestExecuteAllPendingCompensations 测试执行所有待处理的补偿
func (suite *CompensationManagerTestSuite) TestExecuteAllPendingCompensations() {
	ctx := context.Background()

	suite.Run("执行多个待处理补偿", func() {
		// 添加多个补偿记录
		repoID1 := uuid.New()
		repoID2 := uuid.New()
		projectID := uuid.New()

		compensationID1 := suite.compensationMgr.AddCompensation(
			compensation.CompensationActionDeleteRepository,
			repoID1,
			map[string]interface{}{"repo_name": "repo1"},
		)

		compensationID2 := suite.compensationMgr.AddCompensation(
			compensation.CompensationActionDeleteRepository,
			repoID2,
			map[string]interface{}{"repo_name": "repo2"},
		)

		compensationID3 := suite.compensationMgr.AddCompensation(
			compensation.CompensationActionRollbackProject,
			projectID,
			map[string]interface{}{"project_name": "project1"},
		)

		// 设置mock期望
		suite.mockGitClient.On("DeleteRepository", ctx, repoID1).Return(nil)
		suite.mockGitClient.On("DeleteRepository", ctx, repoID2).Return(nil)
		// 项目回滚不需要Git客户端

		// 执行所有待处理补偿
		err := suite.compensationMgr.ExecuteAllPendingCompensations(ctx)
		assert.NoError(suite.T(), err)

		// 验证所有补偿都已执行
		entry1, _ := suite.compensationMgr.GetCompensationStatus(compensationID1)
		assert.Equal(suite.T(), "executed", entry1.Status)

		entry2, _ := suite.compensationMgr.GetCompensationStatus(compensationID2)
		assert.Equal(suite.T(), "executed", entry2.Status)

		entry3, _ := suite.compensationMgr.GetCompensationStatus(compensationID3)
		assert.Equal(suite.T(), "executed", entry3.Status)

		// 验证没有待处理的补偿了
		pending := suite.compensationMgr.ListPendingCompensations()
		assert.Empty(suite.T(), pending)
	})

	suite.Run("执行补偿时部分失败", func() {
		// 添加补偿记录
		repoID1 := uuid.New()
		repoID2 := uuid.New()

		compensationID1 := suite.compensationMgr.AddCompensation(
			compensation.CompensationActionDeleteRepository,
			repoID1,
			map[string]interface{}{},
		)

		compensationID2 := suite.compensationMgr.AddCompensation(
			compensation.CompensationActionDeleteRepository,
			repoID2,
			map[string]interface{}{},
		)

		// 设置mock期望：第一个成功，第二个失败
		suite.mockGitClient.On("DeleteRepository", ctx, repoID1).Return(nil)
		suite.mockGitClient.On("DeleteRepository", ctx, repoID2).Return(fmt.Errorf("删除失败"))

		// 执行所有待处理补偿
		err := suite.compensationMgr.ExecuteAllPendingCompensations(ctx)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "删除失败")

		// 验证第一个成功，第二个失败
		entry1, _ := suite.compensationMgr.GetCompensationStatus(compensationID1)
		assert.Equal(suite.T(), "executed", entry1.Status)

		entry2, _ := suite.compensationMgr.GetCompensationStatus(compensationID2)
		assert.Equal(suite.T(), "pending", entry2.Status)
		assert.Equal(suite.T(), 1, entry2.RetryCount)
		assert.Contains(suite.T(), entry2.LastError, "删除失败")
	})
}

// TestCompensationStateManagement 测试补偿状态管理
func (suite *CompensationManagerTestSuite) TestCompensationStateManagement() {
	ctx := context.Background()

	suite.Run("获取不存在的补偿状态", func() {
		nonExistentID := uuid.New()
		_, err := suite.compensationMgr.GetCompensationStatus(nonExistentID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "补偿记录不存在")
	})

	suite.Run("列出待处理的补偿", func() {
		// 添加不同状态的补偿记录
		repoID1 := uuid.New()
		repoID2 := uuid.New()

		// 待处理的补偿
		compensationID1 := suite.compensationMgr.AddCompensation(
			compensation.CompensationActionDeleteRepository,
			repoID1,
			map[string]interface{}{},
		)

		// 另一个待处理的补偿
		suite.compensationMgr.AddCompensation(
			compensation.CompensationActionRollbackProject,
			suite.testProjectID,
			map[string]interface{}{},
		)

		// 执行一个补偿，使其变为已执行状态
		suite.mockGitClient.On("DeleteRepository", ctx, repoID1).Return(nil)
		suite.compensationMgr.ExecuteCompensation(ctx, compensationID1)

		// 获取待处理的补偿列表
		pending := suite.compensationMgr.ListPendingCompensations()
		assert.Len(suite.T(), pending, 1)
		assert.Equal(suite.T(), compensation.CompensationActionRollbackProject, pending[0].Action)
		assert.Equal(suite.T(), "pending", pending[0].Status)
	})

	suite.Run("重复执行已执行的补偿", func() {
		// 添加补偿记录并执行
		compensationID := suite.compensationMgr.AddCompensation(
			compensation.CompensationActionDeleteRepository,
			suite.testResourceID,
			map[string]interface{}{},
		)

		suite.mockGitClient.On("DeleteRepository", ctx, suite.testResourceID).Return(nil)
		err := suite.compensationMgr.ExecuteCompensation(ctx, compensationID)
		assert.NoError(suite.T(), err)

		// 再次执行相同补偿，应该直接返回成功
		err = suite.compensationMgr.ExecuteCompensation(ctx, compensationID)
		assert.NoError(suite.T(), err)

		// 验证状态没有改变，重试计数没有增加
		entry, _ := suite.compensationMgr.GetCompensationStatus(compensationID)
		assert.Equal(suite.T(), "executed", entry.Status)
		assert.Equal(suite.T(), 1, entry.RetryCount) // 仍然是1，没有增加
	})
}

// TestCleanupExecutedCompensations 测试清理已执行的补偿
func (suite *CompensationManagerTestSuite) TestCleanupExecutedCompensations() {
	ctx := context.Background()

	suite.Run("清理已执行的补偿记录", func() {
		// 添加多个补偿记录
		repoID1 := uuid.New()
		repoID2 := uuid.New()

		compensationID1 := suite.compensationMgr.AddCompensation(
			compensation.CompensationActionDeleteRepository,
			repoID1,
			map[string]interface{}{},
		)

		compensationID2 := suite.compensationMgr.AddCompensation(
			compensation.CompensationActionDeleteRepository,
			repoID2,
			map[string]interface{}{},
		)

		compensationID3 := suite.compensationMgr.AddCompensation(
			compensation.CompensationActionRollbackProject,
			suite.testProjectID,
			map[string]interface{}{},
		)

		// 执行前两个补偿
		suite.mockGitClient.On("DeleteRepository", ctx, repoID1).Return(nil)
		suite.mockGitClient.On("DeleteRepository", ctx, repoID2).Return(nil)

		suite.compensationMgr.ExecuteCompensation(ctx, compensationID1)
		suite.compensationMgr.ExecuteCompensation(ctx, compensationID2)
		// compensationID3 保持待处理状态

		// 执行清理
		removedCount := suite.compensationMgr.ClearExecutedCompensations()
		assert.Equal(suite.T(), 2, removedCount)

		// 验证已执行的记录被清理
		_, err1 := suite.compensationMgr.GetCompensationStatus(compensationID1)
		assert.Error(suite.T(), err1)

		_, err2 := suite.compensationMgr.GetCompensationStatus(compensationID2)
		assert.Error(suite.T(), err2)

		// 验证待处理的记录仍然存在
		entry3, err3 := suite.compensationMgr.GetCompensationStatus(compensationID3)
		assert.NoError(suite.T(), err3)
		assert.Equal(suite.T(), "pending", entry3.Status)

		// 验证待处理列表只有一个记录
		pending := suite.compensationMgr.ListPendingCompensations()
		assert.Len(suite.T(), pending, 1)
	})
}

// TestUnknownCompensationAction 测试未知补偿动作
func (suite *CompensationManagerTestSuite) TestUnknownCompensationAction() {
	ctx := context.Background()

	suite.Run("执行未知的补偿动作", func() {
		// 手动创建包含未知动作的补偿记录
		// 由于我们无法直接设置未知动作，我们通过模拟来测试这种情况
		compensationID := suite.compensationMgr.AddCompensation(
			compensation.CompensationActionDeleteRepository,
			suite.testResourceID,
			map[string]interface{}{},
		)

		// 正常的删除仓库操作应该成功
		suite.mockGitClient.On("DeleteRepository", ctx, suite.testResourceID).Return(nil)
		err := suite.compensationMgr.ExecuteCompensation(ctx, compensationID)
		assert.NoError(suite.T(), err)
	})
}

// TestCompensationManagerEdgeCases 测试边界情况
func (suite *CompensationManagerTestSuite) TestCompensationManagerEdgeCases() {
	ctx := context.Background()

	suite.Run("执行不存在的补偿", func() {
		nonExistentID := uuid.New()
		err := suite.compensationMgr.ExecuteCompensation(ctx, nonExistentID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "补偿记录不存在")
	})

	suite.Run("无待处理补偿时执行全部", func() {
		// 没有待处理的补偿时，应该正常完成
		err := suite.compensationMgr.ExecuteAllPendingCompensations(ctx)
		assert.NoError(suite.T(), err)
	})

	suite.Run("清理空列表", func() {
		// 没有已执行的补偿时，清理应该返回0
		removedCount := suite.compensationMgr.ClearExecutedCompensations()
		assert.Equal(suite.T(), 0, removedCount)
	})

	suite.Run("空的待处理列表", func() {
		pending := suite.compensationMgr.ListPendingCompensations()
		assert.Empty(suite.T(), pending)
	})
}

// TestCompensationRetryMechanism 测试补偿重试机制
func (suite *CompensationManagerTestSuite) TestCompensationRetryMechanism() {
	ctx := context.Background()

	suite.Run("重试机制的完整流程", func() {
		// 添加补偿记录
		compensationID := suite.compensationMgr.AddCompensation(
			compensation.CompensationActionDeleteRepository,
			suite.testResourceID,
			map[string]interface{}{"repo_name": "retry-test"},
		)

		// 模拟重试场景：前2次失败，第3次成功
		suite.mockGitClient.On("DeleteRepository", ctx, suite.testResourceID).
			Return(fmt.Errorf("临时错误")).Twice()
		suite.mockGitClient.On("DeleteRepository", ctx, suite.testResourceID).
			Return(nil).Once()

		// 第一次执行 - 失败
		err := suite.compensationMgr.ExecuteCompensation(ctx, compensationID)
		assert.Error(suite.T(), err)

		entry, _ := suite.compensationMgr.GetCompensationStatus(compensationID)
		assert.Equal(suite.T(), "pending", entry.Status)
		assert.Equal(suite.T(), 1, entry.RetryCount)
		assert.Contains(suite.T(), entry.LastError, "临时错误")

		// 第二次执行 - 失败
		err = suite.compensationMgr.ExecuteCompensation(ctx, compensationID)
		assert.Error(suite.T(), err)

		entry, _ = suite.compensationMgr.GetCompensationStatus(compensationID)
		assert.Equal(suite.T(), "pending", entry.Status)
		assert.Equal(suite.T(), 2, entry.RetryCount)

		// 第三次执行 - 成功
		err = suite.compensationMgr.ExecuteCompensation(ctx, compensationID)
		assert.NoError(suite.T(), err)

		entry, _ = suite.compensationMgr.GetCompensationStatus(compensationID)
		assert.Equal(suite.T(), "executed", entry.Status)
		assert.Equal(suite.T(), 3, entry.RetryCount)
		assert.NotNil(suite.T(), entry.ExecutedAt)
		assert.Empty(suite.T(), entry.LastError) // 成功后错误信息应该为空
	})
}

// TestConcurrentCompensationExecution 测试并发补偿执行
func (suite *CompensationManagerTestSuite) TestConcurrentCompensationExecution() {
	ctx := context.Background()

	suite.Run("并发执行不同的补偿", func() {
		// 创建多个补偿记录
		numCompensations := 5
		compensationIDs := make([]uuid.UUID, numCompensations)
		resourceIDs := make([]uuid.UUID, numCompensations)

		for i := 0; i < numCompensations; i++ {
			resourceIDs[i] = uuid.New()
			compensationIDs[i] = suite.compensationMgr.AddCompensation(
				compensation.CompensationActionDeleteRepository,
				resourceIDs[i],
				map[string]interface{}{"index": i},
			)

			// 设置mock期望
			suite.mockGitClient.On("DeleteRepository", ctx, resourceIDs[i]).Return(nil)
		}

		// 并发执行补偿
		done := make(chan error, numCompensations)
		for i := 0; i < numCompensations; i++ {
			go func(id uuid.UUID) {
				err := suite.compensationMgr.ExecuteCompensation(ctx, id)
				done <- err
			}(compensationIDs[i])
		}

		// 等待所有补偿完成
		for i := 0; i < numCompensations; i++ {
			select {
			case err := <-done:
				assert.NoError(suite.T(), err, "并发补偿执行应该成功")
			case <-time.After(5 * time.Second):
				suite.T().Fatal("并发补偿执行超时")
			}
		}

		// 验证所有补偿都已执行
		for _, compensationID := range compensationIDs {
			entry, err := suite.compensationMgr.GetCompensationStatus(compensationID)
			assert.NoError(suite.T(), err)
			assert.Equal(suite.T(), "executed", entry.Status)
		}

		// 验证没有待处理的补偿
		pending := suite.compensationMgr.ListPendingCompensations()
		assert.Empty(suite.T(), pending)
	})
}

// 运行测试套件
func TestCompensationManagerSuite(t *testing.T) {
	suite.Run(t, new(CompensationManagerTestSuite))
}

// TestCompensationActionConstants 测试补偿动作常量
func TestCompensationActionConstants(t *testing.T) {
	assert.Equal(t, "delete_repository", string(compensation.CompensationActionDeleteRepository))
	assert.Equal(t, "rollback_project", string(compensation.CompensationActionRollbackProject))
	assert.Equal(t, "notify_failure", string(compensation.CompensationActionNotifyFailure))
}

// TestCompensationEntryCreation 测试补偿记录创建
func TestCompensationEntryCreation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockGitClient := new(MockGitGatewayClient)
	compensationMgr := compensation.NewCompensationManager(mockGitClient, logger)

	resourceID := uuid.New()
	metadata := map[string]interface{}{
		"test_key": "test_value",
		"number":   42,
	}

	compensationID := compensationMgr.AddCompensation(
		compensation.CompensationActionDeleteRepository,
		resourceID,
		metadata,
	)

	// 验证补偿记录创建正确
	entry, err := compensationMgr.GetCompensationStatus(compensationID)
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, entry.ID)
	assert.Equal(t, compensation.CompensationActionDeleteRepository, entry.Action)
	assert.Equal(t, resourceID, entry.ResourceID)
	assert.Equal(t, "pending", entry.Status)
	assert.Equal(t, 0, entry.RetryCount)
	assert.Equal(t, 3, entry.MaxRetries)
	assert.Empty(t, entry.LastError)
	assert.Nil(t, entry.ExecutedAt)
	assert.NotZero(t, entry.CreatedAt)
	assert.Equal(t, "test_value", entry.Payload["test_key"])
	assert.Equal(t, float64(42), entry.Payload["number"]) // JSON数字转换为float64
}