package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/client"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/compensation"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/models"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/transaction"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockProjectRepository 模拟项目仓库
type MockProjectRepository struct {
	mock.Mock
}

func (m *MockProjectRepository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.Project, error) {
	args := m.Called(ctx, id, tenantID)
	if project := args.Get(0); project != nil {
		return project.(*models.Project), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectRepository) CheckUserAccess(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, projectID, userID)
	return args.Bool(0), args.Error(1)
}

// 其他必需的方法（简化实现）
func (m *MockProjectRepository) Create(ctx context.Context, project *models.Project) error {
	return nil
}

func (m *MockProjectRepository) Update(ctx context.Context, project *models.Project) error {
	return nil
}

func (m *MockProjectRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	return nil
}

func (m *MockProjectRepository) GetByKey(ctx context.Context, key string, tenantID uuid.UUID) (*models.Project, error) {
	return nil, nil
}

func (m *MockProjectRepository) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, filters map[string]interface{}) ([]models.Project, int64, error) {
	return nil, 0, nil
}

func (m *MockProjectRepository) AddMember(ctx context.Context, member *models.ProjectMember) error {
	return nil
}

func (m *MockProjectRepository) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	return nil
}

func (m *MockProjectRepository) GetMembers(ctx context.Context, projectID uuid.UUID) ([]models.ProjectMember, error) {
	return nil, nil
}

func (m *MockProjectRepository) GetMemberRole(ctx context.Context, projectID, userID uuid.UUID) (*models.Role, error) {
	return nil, nil
}

func (m *MockProjectRepository) GetUserProjects(ctx context.Context, userID, tenantID uuid.UUID) ([]models.Project, error) {
	return nil, nil
}


// TestCompensationManager 测试补偿管理器
func TestCompensationManager(t *testing.T) {
	// 创建模拟Git客户端
	mockGitClient := new(SimpleGitGatewayClientMock)
	logger := zap.NewNop()

	// 创建补偿管理器
	compensationMgr := compensation.NewCompensationManager(mockGitClient, logger)

	// 测试添加补偿动作
	repoID := uuid.New()
	compensationID := compensationMgr.AddCompensation(
		compensation.CompensationActionDeleteRepository,
		repoID,
		map[string]interface{}{
			"repository_name": "test-repo",
			"project_id":      uuid.New().String(),
		},
	)

	assert.NotEqual(t, uuid.Nil, compensationID)

	// 检查补偿状态
	entry, err := compensationMgr.GetCompensationStatus(compensationID)
	assert.NoError(t, err)
	assert.NotNil(t, entry)
	assert.Equal(t, "pending", entry.Status)
	assert.Equal(t, compensation.CompensationActionDeleteRepository, entry.Action)

	// 设置模拟期望
	mockGitClient.On("DeleteRepository", mock.Anything, repoID).Return(nil)

	// 执行补偿
	ctx := context.Background()
	err = compensationMgr.ExecuteCompensation(ctx, compensationID)
	assert.NoError(t, err)

	// 验证状态更新
	entry, err = compensationMgr.GetCompensationStatus(compensationID)
	assert.NoError(t, err)
	assert.Equal(t, "executed", entry.Status)
	assert.NotNil(t, entry.ExecutedAt)

	// 验证模拟调用
	mockGitClient.AssertExpectations(t)

	t.Log("✅ 补偿管理器测试通过")
}

// TestDistributedTransactionManager 测试分布式事务管理器
func TestDistributedTransactionManager(t *testing.T) {
	// 创建模拟依赖
	mockProjectRepo := new(MockProjectRepository)
	mockGitClient := new(SimpleGitGatewayClientMock)
	logger := zap.NewNop()

	// 创建管理器
	compensationMgr := compensation.NewCompensationManager(mockGitClient, logger)
	transactionMgr := transaction.NewDistributedTransactionManager(
		mockProjectRepo,
		mockGitClient,
		compensationMgr,
		logger,
	)

	// 测试数据
	projectID := uuid.New()
	userID := uuid.New()
	tenantID := uuid.New()
	repoID := uuid.New()

	// 设置模拟期望
	mockProject := &models.Project{
		ID:       projectID,
		TenantID: tenantID,
		Name:     "测试项目",
		Key:      "test-project",
	}

	mockRepo := &client.Repository{
		ID:            repoID,
		ProjectID:     projectID,
		Name:          "test-repo",
		Description:   stringPtr("测试仓库"),
		Visibility:    client.RepositoryVisibilityPrivate,
		DefaultBranch: "main",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// 项目仓库模拟
	mockProjectRepo.On("GetByID", mock.Anything, projectID, tenantID).Return(mockProject, nil)
	mockProjectRepo.On("CheckUserAccess", mock.Anything, projectID, userID).Return(true, nil)

	// Git客户端模拟
	mockGitClient.On("CreateRepository", mock.Anything, mock.MatchedBy(func(req *client.CreateRepositoryRequest) bool {
		return req.Name == "test-repo" && req.ProjectID == projectID.String()
	})).Return(mockRepo, nil)

	mockGitClient.On("GetRepository", mock.Anything, repoID).Return(mockRepo, nil)

	// 创建仓库请求
	createReq := &client.CreateRepositoryRequest{
		ProjectID:     projectID.String(),
		Name:          "test-repo",
		Description:   stringPtr("测试仓库"),
		Visibility:    client.RepositoryVisibilityPrivate,
		DefaultBranch: stringPtr("main"),
		InitReadme:    true,
	}

	// 执行分布式事务
	ctx := context.Background()
	repository, err := transactionMgr.CreateRepositoryTransaction(
		ctx,
		projectID,
		userID,
		tenantID,
		createReq,
	)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, repository)
	assert.Equal(t, "test-repo", repository.Name)
	assert.Equal(t, projectID, repository.ProjectID)

	// 验证模拟调用
	mockProjectRepo.AssertExpectations(t)
	mockGitClient.AssertExpectations(t)

	t.Log("✅ 分布式事务管理器测试通过")
}

// TestCompensationWithFailure 测试失败场景的补偿机制
func TestCompensationWithFailure(t *testing.T) {
	// 创建模拟依赖
	mockProjectRepo := new(MockProjectRepository)
	mockGitClient := new(SimpleGitGatewayClientMock)
	logger := zap.NewNop()

	// 创建管理器
	compensationMgr := compensation.NewCompensationManager(mockGitClient, logger)
	transactionMgr := transaction.NewDistributedTransactionManager(
		mockProjectRepo,
		mockGitClient,
		compensationMgr,
		logger,
	)

	// 测试数据
	projectID := uuid.New()
	userID := uuid.New()
	tenantID := uuid.New()
	repoID := uuid.New()

	// 设置模拟期望 - 项目验证成功
	mockProject := &models.Project{
		ID:       projectID,
		TenantID: tenantID,
		Name:     "测试项目",
		Key:      "test-project",
	}

	mockRepo := &client.Repository{
		ID:            repoID,
		ProjectID:     projectID,
		Name:          "test-repo",
		Description:   stringPtr("测试仓库"),
		Visibility:    client.RepositoryVisibilityPrivate,
		DefaultBranch: "main",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// 项目仓库模拟 - 成功
	mockProjectRepo.On("GetByID", mock.Anything, projectID, tenantID).Return(mockProject, nil)
	mockProjectRepo.On("CheckUserAccess", mock.Anything, projectID, userID).Return(true, nil)

	// Git客户端模拟 - 创建成功，但验证失败
	mockGitClient.On("CreateRepository", mock.Anything, mock.MatchedBy(func(req *client.CreateRepositoryRequest) bool {
		return req.Name == "test-repo" && req.ProjectID == projectID.String()
	})).Return(mockRepo, nil)

	// 模拟验证时获取仓库失败
	mockGitClient.On("GetRepository", mock.Anything, repoID).Return(nil, fmt.Errorf("仓库验证失败"))

	// 模拟补偿删除操作
	mockGitClient.On("DeleteRepository", mock.Anything, repoID).Return(nil)

	// 创建仓库请求
	createReq := &client.CreateRepositoryRequest{
		ProjectID:     projectID.String(),
		Name:          "test-repo",
		Description:   stringPtr("测试仓库"),
		Visibility:    client.RepositoryVisibilityPrivate,
		DefaultBranch: stringPtr("main"),
		InitReadme:    true,
	}

	// 执行分布式事务（预期失败）
	ctx := context.Background()
	repository, err := transactionMgr.CreateRepositoryTransaction(
		ctx,
		projectID,
		userID,
		tenantID,
		createReq,
	)

	// 验证失败结果
	assert.Error(t, err)
	assert.Nil(t, repository)
	assert.Contains(t, err.Error(), "确认阶段失败")

	// 验证模拟调用（包括补偿删除操作）
	mockProjectRepo.AssertExpectations(t)
	mockGitClient.AssertExpectations(t)

	t.Log("✅ 失败场景补偿机制测试通过")
}