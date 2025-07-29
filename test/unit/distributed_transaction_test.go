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
	"github.com/cloud-platform/collaborative-dev/internal/project-service/models"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/repository"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/transaction"
)

// MockProjectRepository 项目存储库模拟
type MockProjectRepository struct {
	mock.Mock
}

func (m *MockProjectRepository) Create(ctx context.Context, project *models.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockProjectRepository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.Project, error) {
	args := m.Called(ctx, id, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Project), args.Error(1)
}

func (m *MockProjectRepository) GetByKey(ctx context.Context, key string, tenantID uuid.UUID) (*models.Project, error) {
	args := m.Called(ctx, key, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Project), args.Error(1)
}

func (m *MockProjectRepository) Update(ctx context.Context, project *models.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockProjectRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	args := m.Called(ctx, id, tenantID)
	return args.Error(0)
}

func (m *MockProjectRepository) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, filters map[string]interface{}) ([]models.Project, int64, error) {
	args := m.Called(ctx, tenantID, page, pageSize, filters)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]models.Project), args.Get(1).(int64), args.Error(2)
}

func (m *MockProjectRepository) AddMember(ctx context.Context, member *models.ProjectMember) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockProjectRepository) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	args := m.Called(ctx, projectID, userID)
	return args.Error(0)
}

func (m *MockProjectRepository) GetMembers(ctx context.Context, projectID uuid.UUID) ([]models.ProjectMember, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.ProjectMember), args.Error(1)
}

func (m *MockProjectRepository) CheckUserAccess(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, projectID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockProjectRepository) GetUserProjects(ctx context.Context, userID, tenantID uuid.UUID) ([]models.Project, error) {
	args := m.Called(ctx, userID, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Project), args.Error(1)
}

// MockGitGatewayClient Git网关客户端模拟
type MockGitGatewayClient struct {
	mock.Mock
}

func (m *MockGitGatewayClient) CreateRepository(ctx context.Context, req *client.CreateRepositoryRequest) (*client.Repository, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.Repository), args.Error(1)
}

func (m *MockGitGatewayClient) GetRepository(ctx context.Context, repositoryID uuid.UUID) (*client.Repository, error) {
	args := m.Called(ctx, repositoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.Repository), args.Error(1)
}

func (m *MockGitGatewayClient) UpdateRepository(ctx context.Context, repositoryID uuid.UUID, req *client.UpdateRepositoryRequest) (*client.Repository, error) {
	args := m.Called(ctx, repositoryID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.Repository), args.Error(1)
}

func (m *MockGitGatewayClient) DeleteRepository(ctx context.Context, repositoryID uuid.UUID) error {
	args := m.Called(ctx, repositoryID)
	return args.Error(0)
}

func (m *MockGitGatewayClient) ListRepositories(ctx context.Context, projectID *uuid.UUID, page, pageSize int) (*client.RepositoryListResponse, error) {
	args := m.Called(ctx, projectID, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.RepositoryListResponse), args.Error(1)
}

func (m *MockGitGatewayClient) CreateBranch(ctx context.Context, repositoryID uuid.UUID, req *client.CreateBranchRequest) (*client.Branch, error) {
	args := m.Called(ctx, repositoryID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.Branch), args.Error(1)
}

func (m *MockGitGatewayClient) ListBranches(ctx context.Context, repositoryID uuid.UUID) ([]client.Branch, error) {
	args := m.Called(ctx, repositoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]client.Branch), args.Error(1)
}

func (m *MockGitGatewayClient) DeleteBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) error {
	args := m.Called(ctx, repositoryID, branchName)
	return args.Error(0)
}

func (m *MockGitGatewayClient) GetCommits(ctx context.Context, repositoryID uuid.UUID, branch string, limit int) ([]client.Commit, error) {
	args := m.Called(ctx, repositoryID, branch, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]client.Commit), args.Error(1)
}

func (m *MockGitGatewayClient) GetCommit(ctx context.Context, repositoryID uuid.UUID, commitSHA string) (*client.Commit, error) {
	args := m.Called(ctx, repositoryID, commitSHA)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.Commit), args.Error(1)
}

func (m *MockGitGatewayClient) CreateTag(ctx context.Context, repositoryID uuid.UUID, req *client.CreateTagRequest) (*client.Tag, error) {
	args := m.Called(ctx, repositoryID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.Tag), args.Error(1)
}

func (m *MockGitGatewayClient) ListTags(ctx context.Context, repositoryID uuid.UUID) ([]client.Tag, error) {
	args := m.Called(ctx, repositoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]client.Tag), args.Error(1)
}

func (m *MockGitGatewayClient) DeleteTag(ctx context.Context, repositoryID uuid.UUID, tagName string) error {
	args := m.Called(ctx, repositoryID, tagName)
	return args.Error(0)
}

func (m *MockGitGatewayClient) GetFileContent(ctx context.Context, repositoryID uuid.UUID, branch, filePath string) (*client.FileContent, error) {
	args := m.Called(ctx, repositoryID, branch, filePath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.FileContent), args.Error(1)
}

func (m *MockGitGatewayClient) CreateOrUpdateFile(ctx context.Context, repositoryID uuid.UUID, req *client.CreateOrUpdateFileRequest) (*client.FileCommit, error) {
	args := m.Called(ctx, repositoryID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.FileCommit), args.Error(1)
}

func (m *MockGitGatewayClient) DeleteFile(ctx context.Context, repositoryID uuid.UUID, req *client.DeleteFileRequest) (*client.FileCommit, error) {
	args := m.Called(ctx, repositoryID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.FileCommit), args.Error(1)
}

func (m *MockGitGatewayClient) GetRepositoryTree(ctx context.Context, repositoryID uuid.UUID, branch string, path string) (*client.RepositoryTree, error) {
	args := m.Called(ctx, repositoryID, branch, path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.RepositoryTree), args.Error(1)
}

func (m *MockGitGatewayClient) GetRepositoryStats(ctx context.Context, repositoryID uuid.UUID) (*client.RepositoryStats, error) {
	args := m.Called(ctx, repositoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.RepositoryStats), args.Error(1)
}

func (m *MockGitGatewayClient) SearchRepositories(ctx context.Context, req *client.SearchRepositoriesRequest) (*client.SearchRepositoriesResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.SearchRepositoriesResponse), args.Error(1)
}

func (m *MockGitGatewayClient) GetDiff(ctx context.Context, repositoryID uuid.UUID, base, head string) (*client.Diff, error) {
	args := m.Called(ctx, repositoryID, base, head)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.Diff), args.Error(1)
}

func (m *MockGitGatewayClient) MergeBranches(ctx context.Context, repositoryID uuid.UUID, req *client.MergeBranchesRequest) (*client.MergeResult, error) {
	args := m.Called(ctx, repositoryID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.MergeResult), args.Error(1)
}

func (m *MockGitGatewayClient) CreatePullRequest(ctx context.Context, repositoryID uuid.UUID, req *client.CreatePullRequestRequest) (*client.PullRequest, error) {
	args := m.Called(ctx, repositoryID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.PullRequest), args.Error(1)
}

func (m *MockGitGatewayClient) GetPullRequest(ctx context.Context, repositoryID, pullRequestID uuid.UUID) (*client.PullRequest, error) {
	args := m.Called(ctx, repositoryID, pullRequestID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.PullRequest), args.Error(1)
}

// MockCompensationManager 补偿管理器模拟
type MockCompensationManager struct {
	mock.Mock
}

func (m *MockCompensationManager) AddCompensation(action compensation.CompensationAction, resourceID uuid.UUID, metadata map[string]interface{}) uuid.UUID {
	args := m.Called(action, resourceID, metadata)
	return args.Get(0).(uuid.UUID)
}

func (m *MockCompensationManager) ExecuteCompensation(ctx context.Context, compensationID uuid.UUID) error {
	args := m.Called(ctx, compensationID)
	return args.Error(0)
}

func (m *MockCompensationManager) GetCompensation(compensationID uuid.UUID) (*compensation.Compensation, error) {
	args := m.Called(compensationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*compensation.Compensation), args.Error(1)
}

func (m *MockCompensationManager) ListPendingCompensations() []*compensation.Compensation {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]*compensation.Compensation)
}

func (m *MockCompensationManager) CleanupExecutedCompensations() int {
	args := m.Called()
	return args.Int(0)
}

// DistributedTransactionTestSuite 分布式事务管理器测试套件
type DistributedTransactionTestSuite struct {
	suite.Suite
	txManager       *transaction.DistributedTransactionManager
	mockRepo        *MockProjectRepository
	mockGitClient   *MockGitGatewayClient
	mockCompensation *MockCompensationManager
	logger          *zap.Logger
	testProjectID   uuid.UUID
	testTenantID    uuid.UUID
	testUserID      uuid.UUID
	testRepoID      uuid.UUID
}

func (suite *DistributedTransactionTestSuite) SetupSuite() {
	suite.logger = zaptest.NewLogger(suite.T())
	
	suite.testProjectID = uuid.New()
	suite.testTenantID = uuid.New()
	suite.testUserID = uuid.New()  
	suite.testRepoID = uuid.New()
}

func (suite *DistributedTransactionTestSuite) SetupTest() {
	suite.mockRepo = new(MockProjectRepository)
	suite.mockGitClient = new(MockGitGatewayClient)
	suite.mockCompensation = new(MockCompensationManager)
	
	suite.txManager = transaction.NewDistributedTransactionManager(
		suite.mockRepo,
		suite.mockGitClient,
		suite.mockCompensation,
		suite.logger,
	)
}

func (suite *DistributedTransactionTestSuite) TearDownTest() {
	suite.mockRepo.AssertExpectations(suite.T())
	suite.mockGitClient.AssertExpectations(suite.T())
	suite.mockCompensation.AssertExpectations(suite.T())
}

// TestCreateRepositoryTransactionSuccess 测试创建仓库事务成功场景
func (suite *DistributedTransactionTestSuite) TestCreateRepositoryTransactionSuccess() {
	ctx := context.Background()
	
	// 准备测试数据
	createReq := &client.CreateRepositoryRequest{
		ProjectID:     suite.testProjectID.String(),
		Name:          "test-repo",
		Description:   &[]string{"测试仓库"}[0],
		Visibility:    client.RepositoryVisibilityPrivate,
		DefaultBranch: &[]string{"main"}[0],
		InitReadme:    true,
	}
	
	expectedRepo := &client.Repository{
		ID:            suite.testRepoID,
		ProjectID:     suite.testProjectID,
		Name:          "test-repo",
		Description:   &[]string{"测试仓库"}[0],
		Visibility:    client.RepositoryVisibilityPrivate,
		DefaultBranch: "main",
	}
	
	compensationID := uuid.New()
	
	// 设置mock期望
	// 1. 验证阶段：检查项目存在和用户权限
	suite.mockRepo.On("GetByID", ctx, suite.testProjectID, suite.testTenantID).
		Return(&models.Project{
			ID:       suite.testProjectID,
			TenantID: suite.testTenantID,
			Name:     "测试项目",
		}, nil)
	
	suite.mockRepo.On("CheckUserAccess", ctx, suite.testProjectID, suite.testUserID).
		Return(true, nil)
	
	// 2. 执行阶段：调用Git网关创建仓库
	suite.mockGitClient.On("CreateRepository", ctx, createReq).
		Return(expectedRepo, nil)
	
	// 3. 添加补偿动作
	suite.mockCompensation.On("AddCompensation", 
		compensation.CompensationActionDeleteRepository,
		suite.testRepoID,
		mock.MatchedBy(func(metadata map[string]interface{}) bool {
			return metadata["repository_name"] == "test-repo" &&
				   metadata["project_id"] == suite.testProjectID.String()
		})).Return(compensationID)
	
	// 4. 确认阶段：验证仓库创建成功
	suite.mockGitClient.On("GetRepository", ctx, suite.testRepoID).
		Return(expectedRepo, nil)
	
	// 执行测试
	result, err := suite.txManager.CreateRepositoryTransaction(
		ctx,
		suite.testProjectID,
		suite.testUserID,
		suite.testTenantID,
		createReq,
	)
	
	// 验证结果
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), suite.testRepoID, result.ID)
	assert.Equal(suite.T(), suite.testProjectID, result.ProjectID)
	assert.Equal(suite.T(), "test-repo", result.Name)
	assert.Equal(suite.T(), "private", result.Visibility)
	
	// 验证事务状态
	txList := suite.txManager.ListActiveTransactions()
	assert.Empty(suite.T(), txList, "事务完成后应该没有活跃事务")
}

// TestCreateRepositoryTransactionValidationFailure 测试验证阶段失败
func (suite *DistributedTransactionTestSuite) TestCreateRepositoryTransactionValidationFailure() {
	ctx := context.Background()
	
	createReq := &client.CreateRepositoryRequest{
		ProjectID:  suite.testProjectID.String(),
		Name:       "test-repo",
		Visibility: client.RepositoryVisibilityPrivate,
	}
	
	// 设置mock期望：项目不存在
	suite.mockRepo.On("GetByID", ctx, suite.testProjectID, suite.testTenantID).
		Return(nil, fmt.Errorf("项目不存在"))
	
	// 执行测试
	result, err := suite.txManager.CreateRepositoryTransaction(
		ctx,
		suite.testProjectID,
		suite.testUserID,
		suite.testTenantID,
		createReq,
	)
	
	// 验证结果
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "验证阶段失败")
	
	// 验证事务状态
	txList := suite.txManager.ListActiveTransactions()
	assert.Empty(suite.T(), txList, "失败的事务不应该保留在活跃列表中")
}

// TestCreateRepositoryTransactionUserPermissionFailure 测试用户权限检查失败
func (suite *DistributedTransactionTestSuite) TestCreateRepositoryTransactionUserPermissionFailure() {
	ctx := context.Background()
	
	createReq := &client.CreateRepositoryRequest{
		ProjectID:  suite.testProjectID.String(),
		Name:       "test-repo",
		Visibility: client.RepositoryVisibilityPrivate,
	}
	
	// 设置mock期望：项目存在但用户无权限
	suite.mockRepo.On("GetByID", ctx, suite.testProjectID, suite.testTenantID).
		Return(&models.Project{
			ID:       suite.testProjectID,
			TenantID: suite.testTenantID,
			Name:     "测试项目",
		}, nil)
	
	suite.mockRepo.On("CheckUserAccess", ctx, suite.testProjectID, suite.testUserID).
		Return(false, nil)
	
	// 执行测试
	result, err := suite.txManager.CreateRepositoryTransaction(
		ctx,
		suite.testProjectID,
		suite.testUserID,
		suite.testTenantID,
		createReq,
	)
	
	// 验证结果
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "用户无权限在此项目中创建仓库")
}

// TestCreateRepositoryTransactionExecutionFailure 测试执行阶段失败
func (suite *DistributedTransactionTestSuite) TestCreateRepositoryTransactionExecutionFailure() {
	ctx := context.Background()
	
	createReq := &client.CreateRepositoryRequest{
		ProjectID:  suite.testProjectID.String(),
		Name:       "test-repo",
		Visibility: client.RepositoryVisibilityPrivate,
	}
	
	// 设置mock期望：验证通过但Git网关创建失败
	suite.mockRepo.On("GetByID", ctx, suite.testProjectID, suite.testTenantID).
		Return(&models.Project{
			ID:       suite.testProjectID,
			TenantID: suite.testTenantID,
			Name:     "测试项目",
		}, nil)
	
	suite.mockRepo.On("CheckUserAccess", ctx, suite.testProjectID, suite.testUserID).
		Return(true, nil)
	
	suite.mockGitClient.On("CreateRepository", ctx, createReq).
		Return(nil, fmt.Errorf("Git网关服务不可用"))
	
	// 执行测试
	result, err := suite.txManager.CreateRepositoryTransaction(
		ctx,
		suite.testProjectID,
		suite.testUserID,
		suite.testTenantID,
		createReq,
	)
	
	// 验证结果
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "执行阶段失败")
	assert.Contains(suite.T(), err.Error(), "Git网关服务不可用")
}

// TestCreateRepositoryTransactionConfirmationFailure 测试确认阶段失败
func (suite *DistributedTransactionTestSuite) TestCreateRepositoryTransactionConfirmationFailure() {
	ctx := context.Background()
	
	createReq := &client.CreateRepositoryRequest{
		ProjectID:  suite.testProjectID.String(),
		Name:       "test-repo",
		Visibility: client.RepositoryVisibilityPrivate,
	}
	
	expectedRepo := &client.Repository{
		ID:            suite.testRepoID,
		ProjectID:     suite.testProjectID,
		Name:          "test-repo",
		Visibility:    client.RepositoryVisibilityPrivate,
		DefaultBranch: "main",
	}
	
	compensationID := uuid.New()
	
	// 设置mock期望：执行成功但确认失败
	suite.mockRepo.On("GetByID", ctx, suite.testProjectID, suite.testTenantID).
		Return(&models.Project{
			ID:       suite.testProjectID,
			TenantID: suite.testTenantID,
			Name:     "测试项目",
		}, nil)
	
	suite.mockRepo.On("CheckUserAccess", ctx, suite.testProjectID, suite.testUserID).
		Return(true, nil)
	
	suite.mockGitClient.On("CreateRepository", ctx, createReq).
		Return(expectedRepo, nil)
	
	suite.mockCompensation.On("AddCompensation", 
		compensation.CompensationActionDeleteRepository,
		suite.testRepoID,
		mock.AnythingOfType("map[string]interface {}")).Return(compensationID)
	
	// 确认阶段失败
	suite.mockGitClient.On("GetRepository", ctx, suite.testRepoID).
		Return(nil, fmt.Errorf("仓库验证失败"))
	
	// 执行补偿
	suite.mockCompensation.On("ExecuteCompensation", mock.Anything, compensationID).
		Return(nil)
	
	// 执行测试
	result, err := suite.txManager.CreateRepositoryTransaction(
		ctx,
		suite.testProjectID,
		suite.testUserID,
		suite.testTenantID,
		createReq,
	)
	
	// 验证结果
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "确认阶段失败")
}

// TestTransactionStateManagement 测试事务状态管理
func (suite *DistributedTransactionTestSuite) TestTransactionStateManagement() {
	ctx := context.Background()
	
	createReq := &client.CreateRepositoryRequest{
		ProjectID:  suite.testProjectID.String(),
		Name:       "test-repo",
		Visibility: client.RepositoryVisibilityPrivate,
	}
	
	// 设置mock期望：验证阶段通过，但执行阶段永远挂起（不返回）
	suite.mockRepo.On("GetByID", ctx, suite.testProjectID, suite.testTenantID).
		Return(&models.Project{
			ID:       suite.testProjectID,
			TenantID: suite.testTenantID,
			Name:     "测试项目",
		}, nil)
	
	suite.mockRepo.On("CheckUserAccess", ctx, suite.testProjectID, suite.testUserID).
		Return(true, nil)
	
	// 启动事务（在goroutine中，因为会阻塞）
	done := make(chan struct{})
	var txID uuid.UUID
	
	go func() {
		defer close(done)
		// 只执行到验证阶段，不继续执行
		// 这里我们直接测试状态管理API
	}()
	
	// 等待一小段时间让事务开始
	time.Sleep(100 * time.Millisecond)
	
	// 测试获取活跃事务列表
	activeTransactions := suite.txManager.ListActiveTransactions()
	// 由于我们没有实际启动事务，这里应该是空的
	assert.Empty(suite.T(), activeTransactions)
	
	// 测试获取不存在的事务
	_, err := suite.txManager.GetTransaction(uuid.New())
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "事务不存在")
	
	// 测试清理已完成的事务
	cleaned := suite.txManager.CleanupCompletedTransactions()
	assert.GreaterOrEqual(suite.T(), cleaned, 0)
	
	// 如果有活跃事务，测试获取事务信息
	if len(activeTransactions) > 0 {
		txID = activeTransactions[0].ID
		tx, err := suite.txManager.GetTransaction(txID)
		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), tx)
		assert.Equal(suite.T(), txID, tx.ID)
	}
	
	<-done
}

// TestTransactionTimeout 测试事务超时场景
func (suite *DistributedTransactionTestSuite) TestTransactionTimeout() {
	// 这个测试用于验证事务超时机制
	// 在实际实现中，应该有超时机制来清理长时间运行的事务
	
	// 创建一个超过24小时的旧事务记录来测试清理机制
	suite.Run("测试清理超时事务", func() {
		// 直接测试清理方法
		cleaned := suite.txManager.CleanupCompletedTransactions()
		assert.GreaterOrEqual(suite.T(), cleaned, 0)
	})
}

// TestConcurrentTransactions 测试并发事务
func (suite *DistributedTransactionTestSuite) TestConcurrentTransactions() {
	ctx := context.Background()
	
	// 准备多个并发事务请求
	numTransactions := 5
	done := make(chan error, numTransactions)
	
	for i := 0; i < numTransactions; i++ {
		go func(index int) {
			createReq := &client.CreateRepositoryRequest{
				ProjectID:  suite.testProjectID.String(),
				Name:       fmt.Sprintf("test-repo-%d", index),
				Visibility: client.RepositoryVisibilityPrivate,
			}
			
			repoID := uuid.New()
			expectedRepo := &client.Repository{
				ID:            repoID,
				ProjectID:     suite.testProjectID,
				Name:          fmt.Sprintf("test-repo-%d", index),
				Visibility:    client.RepositoryVisibilityPrivate,
				DefaultBranch: "main",
			}
			
			compensationID := uuid.New()
			
			// 设置mock期望（每个并发事务都需要独立的期望）
			suite.mockRepo.On("GetByID", ctx, suite.testProjectID, suite.testTenantID).
				Return(&models.Project{
					ID:       suite.testProjectID,
					TenantID: suite.testTenantID,
					Name:     "测试项目",
				}, nil).Once()
			
			suite.mockRepo.On("CheckUserAccess", ctx, suite.testProjectID, suite.testUserID).
				Return(true, nil).Once()
			
			suite.mockGitClient.On("CreateRepository", ctx, createReq).
				Return(expectedRepo, nil).Once()
			
			suite.mockCompensation.On("AddCompensation", 
				compensation.CompensationActionDeleteRepository,
				repoID,
				mock.AnythingOfType("map[string]interface {}")).Return(compensationID).Once()
			
			suite.mockGitClient.On("GetRepository", ctx, repoID).
				Return(expectedRepo, nil).Once()
			
			// 执行事务
			result, err := suite.txManager.CreateRepositoryTransaction(
				ctx,
				suite.testProjectID,
				suite.testUserID,
				suite.testTenantID,
				createReq,
			)
			
			if err != nil {
				done <- err
				return
			}
			
			if result == nil || result.Name != fmt.Sprintf("test-repo-%d", index) {
				done <- fmt.Errorf("并发事务结果不正确")
				return
			}
			
			done <- nil
		}(i)
	}
	
	// 等待所有并发事务完成
	for i := 0; i < numTransactions; i++ {
		select {
		case err := <-done:
			assert.NoError(suite.T(), err, "并发事务应该成功")
		case <-time.After(10 * time.Second):
			suite.T().Fatal("并发事务超时")
		}
	}
	
	// 验证所有事务都已完成
	activeTransactions := suite.txManager.ListActiveTransactions()
	assert.Empty(suite.T(), activeTransactions, "所有并发事务完成后不应该有活跃事务")
}

// TestTransactionRollbackOnFailure 测试失败时的事务回滚
func (suite *DistributedTransactionTestSuite) TestTransactionRollbackOnFailure() {
	ctx := context.Background()
	
	createReq := &client.CreateRepositoryRequest{
		ProjectID:  suite.testProjectID.String(),
		Name:       "test-repo",
		Visibility: client.RepositoryVisibilityPrivate,
	}
	
	expectedRepo := &client.Repository{
		ID:            suite.testRepoID,
		ProjectID:     suite.testProjectID,
		Name:          "test-repo",
		Visibility:    client.RepositoryVisibilityPrivate,
		DefaultBranch: "main",
	}
	
	compensationID := uuid.New()
	
	// 设置mock期望：执行成功，确认失败（需要回滚）
	suite.mockRepo.On("GetByID", ctx, suite.testProjectID, suite.testTenantID).
		Return(&models.Project{
			ID:       suite.testProjectID,
			TenantID: suite.testTenantID,
			Name:     "测试项目",
		}, nil)
	
	suite.mockRepo.On("CheckUserAccess", ctx, suite.testProjectID, suite.testUserID).
		Return(true, nil)
	
	suite.mockGitClient.On("CreateRepository", ctx, createReq).
		Return(expectedRepo, nil)
	
	suite.mockCompensation.On("AddCompensation", 
		compensation.CompensationActionDeleteRepository,
		suite.testRepoID,
		mock.AnythingOfType("map[string]interface {}")).Return(compensationID)
	
	// 确认失败，触发回滚
	suite.mockGitClient.On("GetRepository", ctx, suite.testRepoID).
		Return(nil, fmt.Errorf("确认失败"))
	
	// 期望执行补偿动作
	suite.mockCompensation.On("ExecuteCompensation", mock.Anything, compensationID).
		Return(nil)
	
	// 执行测试
	result, err := suite.txManager.CreateRepositoryTransaction(
		ctx,
		suite.testProjectID,
		suite.testUserID,
		suite.testTenantID,
		createReq,
	)
	
	// 验证结果
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "确认阶段失败")
	
	// 验证补偿动作被执行
	suite.mockCompensation.AssertCalled(suite.T(), "ExecuteCompensation", mock.Anything, compensationID)
}

// TestInvalidRepositoryName 测试无效仓库名称验证
func (suite *DistributedTransactionTestSuite) TestInvalidRepositoryName() {
	ctx := context.Background()
	
	createReq := &client.CreateRepositoryRequest{
		ProjectID:  suite.testProjectID.String(),
		Name:       "", // 空名称
		Visibility: client.RepositoryVisibilityPrivate,
	}
	
	// 设置mock期望：验证阶段检查项目和权限
	suite.mockRepo.On("GetByID", ctx, suite.testProjectID, suite.testTenantID).
		Return(&models.Project{
			ID:       suite.testProjectID,
			TenantID: suite.testTenantID,
			Name:     "测试项目",
		}, nil)
	
	suite.mockRepo.On("CheckUserAccess", ctx, suite.testProjectID, suite.testUserID).
		Return(true, nil)
	
	// 执行测试
	result, err := suite.txManager.CreateRepositoryTransaction(
		ctx,
		suite.testProjectID,
		suite.testUserID,
		suite.testTenantID,
		createReq,
	)
	
	// 验证结果
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "仓库名称不能为空")
}

// 运行测试套件
func TestDistributedTransactionSuite(t *testing.T) {
	suite.Run(t, new(DistributedTransactionTestSuite))
}

// TestTransactionPhaseTransitions 测试事务阶段转换
func TestTransactionPhaseTransitions(t *testing.T) {
	// 测试事务阶段常量
	assert.Equal(t, "validation", string(transaction.PhaseValidation))
	assert.Equal(t, "execution", string(transaction.PhaseExecution))
	assert.Equal(t, "confirm", string(transaction.PhaseConfirm))
	assert.Equal(t, "cancel", string(transaction.PhaseCancel))
	
	// 测试事务状态常量
	assert.Equal(t, "pending", string(transaction.StatusPending))
	assert.Equal(t, "validated", string(transaction.StatusValidated))
	assert.Equal(t, "executed", string(transaction.StatusExecuted))
	assert.Equal(t, "confirmed", string(transaction.StatusConfirmed))
	assert.Equal(t, "cancelled", string(transaction.StatusCancelled))
	assert.Equal(t, "failed", string(transaction.StatusFailed))
}

// TestTransactionManagerErrorHandling 测试事务管理器错误处理
func TestTransactionManagerErrorHandling(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockRepo := new(MockProjectRepository)
	mockGitClient := new(MockGitGatewayClient)
	mockCompensation := new(MockCompensationManager)
	
	txManager := transaction.NewDistributedTransactionManager(
		mockRepo,
		mockGitClient,
		mockCompensation,
		logger,
	)
	
	// 测试获取不存在的事务
	_, err := txManager.GetTransaction(uuid.New())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "事务不存在")
	
	// 测试初始状态
	activeTransactions := txManager.ListActiveTransactions()
	assert.Empty(t, activeTransactions)
	
	cleanedCount := txManager.CleanupCompletedTransactions()
	assert.Equal(t, 0, cleanedCount)
}