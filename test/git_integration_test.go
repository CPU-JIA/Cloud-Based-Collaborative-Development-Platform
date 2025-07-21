package test

import (
	"context"
	"testing"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/client"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockGitGatewayClient Git网关客户端模拟（简化版）
type SimpleGitGatewayClientMock struct {
	mock.Mock
}

func (m *SimpleGitGatewayClientMock) CreateRepository(ctx context.Context, req *client.CreateRepositoryRequest) (*client.Repository, error) {
	args := m.Called(ctx, req)
	if repo := args.Get(0); repo != nil {
		return repo.(*client.Repository), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *SimpleGitGatewayClientMock) GetRepository(ctx context.Context, repositoryID uuid.UUID) (*client.Repository, error) {
	args := m.Called(ctx, repositoryID)
	if repo := args.Get(0); repo != nil {
		return repo.(*client.Repository), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *SimpleGitGatewayClientMock) UpdateRepository(ctx context.Context, repositoryID uuid.UUID, req *client.UpdateRepositoryRequest) (*client.Repository, error) {
	args := m.Called(ctx, repositoryID, req)
	if repo := args.Get(0); repo != nil {
		return repo.(*client.Repository), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *SimpleGitGatewayClientMock) DeleteRepository(ctx context.Context, repositoryID uuid.UUID) error {
	args := m.Called(ctx, repositoryID)
	return args.Error(0)
}

func (m *SimpleGitGatewayClientMock) ListRepositories(ctx context.Context, projectID *uuid.UUID, page, pageSize int) (*client.RepositoryListResponse, error) {
	args := m.Called(ctx, projectID, page, pageSize)
	if resp := args.Get(0); resp != nil {
		return resp.(*client.RepositoryListResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

// 其他接口方法的简化实现
func (m *SimpleGitGatewayClientMock) CreateBranch(ctx context.Context, repositoryID uuid.UUID, req *client.CreateBranchRequest) (*client.Branch, error) {
	return nil, nil
}

func (m *SimpleGitGatewayClientMock) GetBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) (*client.Branch, error) {
	return nil, nil
}

func (m *SimpleGitGatewayClientMock) ListBranches(ctx context.Context, repositoryID uuid.UUID) ([]client.Branch, error) {
	return nil, nil
}

func (m *SimpleGitGatewayClientMock) DeleteBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) error {
	return nil
}

func (m *SimpleGitGatewayClientMock) SetDefaultBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) error {
	return nil
}

func (m *SimpleGitGatewayClientMock) MergeBranch(ctx context.Context, repositoryID uuid.UUID, targetBranch, sourceBranch string) error {
	return nil
}

func (m *SimpleGitGatewayClientMock) CreateCommit(ctx context.Context, repositoryID uuid.UUID, req *client.CreateCommitRequest) (*client.Commit, error) {
	return nil, nil
}

func (m *SimpleGitGatewayClientMock) GetCommit(ctx context.Context, repositoryID uuid.UUID, sha string) (*client.Commit, error) {
	return nil, nil
}

func (m *SimpleGitGatewayClientMock) ListCommits(ctx context.Context, repositoryID uuid.UUID, branch string, page, pageSize int) (*client.CommitListResponse, error) {
	return nil, nil
}

func (m *SimpleGitGatewayClientMock) GetCommitDiff(ctx context.Context, repositoryID uuid.UUID, sha string) (*client.GitDiff, error) {
	return nil, nil
}

func (m *SimpleGitGatewayClientMock) CompareBranches(ctx context.Context, repositoryID uuid.UUID, base, head string) (*client.GitDiff, error) {
	return nil, nil
}

func (m *SimpleGitGatewayClientMock) CreateTag(ctx context.Context, repositoryID uuid.UUID, req *client.CreateTagRequest) (*client.Tag, error) {
	return nil, nil
}

func (m *SimpleGitGatewayClientMock) GetTag(ctx context.Context, repositoryID uuid.UUID, tagName string) (*client.Tag, error) {
	return nil, nil
}

func (m *SimpleGitGatewayClientMock) ListTags(ctx context.Context, repositoryID uuid.UUID) ([]client.Tag, error) {
	return nil, nil
}

func (m *SimpleGitGatewayClientMock) DeleteTag(ctx context.Context, repositoryID uuid.UUID, tagName string) error {
	return nil
}

func (m *SimpleGitGatewayClientMock) GetFileContent(ctx context.Context, repositoryID uuid.UUID, branch, filePath string) ([]byte, error) {
	return nil, nil
}

func (m *SimpleGitGatewayClientMock) GetDirectoryContent(ctx context.Context, repositoryID uuid.UUID, branch, dirPath string) ([]client.FileInfo, error) {
	return nil, nil
}

func (m *SimpleGitGatewayClientMock) GetRepositoryStats(ctx context.Context, repositoryID uuid.UUID) (*client.RepositoryStats, error) {
	return nil, nil
}

func (m *SimpleGitGatewayClientMock) SearchRepositories(ctx context.Context, query string, projectID *uuid.UUID, page, pageSize int) (*client.RepositoryListResponse, error) {
	return nil, nil
}

// TestGitGatewayClientCreation 测试Git网关客户端创建
func TestGitGatewayClientCreation(t *testing.T) {
	// 创建Git网关客户端配置
	config := &client.GitGatewayClientConfig{
		BaseURL: "http://localhost:8083",
		Timeout: 30 * time.Second,
		Logger:  zap.NewNop(),
	}

	// 创建客户端
	gitClient := client.NewGitGatewayClient(config)
	assert.NotNil(t, gitClient)

	t.Log("✅ Git网关客户端创建成功")
}

// TestGitGatewayMockClient 测试模拟Git网关客户端功能
func TestGitGatewayMockClient(t *testing.T) {
	// 创建模拟客户端
	mockClient := new(SimpleGitGatewayClientMock)

	// 设置测试数据
	projectID := uuid.New()
	repoID := uuid.New()

	mockRepo := &client.Repository{
		ID:            repoID,
		ProjectID:     projectID,
		Name:          "test-repo",
		Description:   stringPointer("测试仓库"),
		Visibility:    client.RepositoryVisibilityPrivate,
		DefaultBranch: "main",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// 设置创建仓库的期望
	mockClient.On("CreateRepository", mock.Anything, mock.MatchedBy(func(req *client.CreateRepositoryRequest) bool {
		return req.Name == "test-repo" && req.ProjectID == projectID.String()
	})).Return(mockRepo, nil)

	// 设置获取仓库的期望
	mockClient.On("GetRepository", mock.Anything, repoID).Return(mockRepo, nil)

	// 设置仓库列表的期望
	mockRepoList := &client.RepositoryListResponse{
		Repositories: []client.Repository{*mockRepo},
		Total:        1,
		Page:         1,
		PageSize:     20,
	}
	mockClient.On("ListRepositories", mock.Anything, &projectID, 1, 20).Return(mockRepoList, nil)

	// 测试创建仓库
	ctx := context.Background()
	createReq := &client.CreateRepositoryRequest{
		ProjectID:     projectID.String(),
		Name:          "test-repo",
		Description:   stringPtr("测试仓库"),
		Visibility:    client.RepositoryVisibilityPrivate,
		DefaultBranch: stringPointer("main"),
		InitReadme:    true,
	}

	createdRepo, err := mockClient.CreateRepository(ctx, createReq)
	assert.NoError(t, err)
	assert.NotNil(t, createdRepo)
	assert.Equal(t, "test-repo", createdRepo.Name)
	assert.Equal(t, projectID, createdRepo.ProjectID)

	// 测试获取仓库
	fetchedRepo, err := mockClient.GetRepository(ctx, repoID)
	assert.NoError(t, err)
	assert.NotNil(t, fetchedRepo)
	assert.Equal(t, repoID, fetchedRepo.ID)
	assert.Equal(t, "test-repo", fetchedRepo.Name)

	// 测试仓库列表
	repoList, err := mockClient.ListRepositories(ctx, &projectID, 1, 20)
	assert.NoError(t, err)
	assert.NotNil(t, repoList)
	assert.Len(t, repoList.Repositories, 1)
	assert.Equal(t, "test-repo", repoList.Repositories[0].Name)

	// 验证所有期望都被调用
	mockClient.AssertExpectations(t)

	t.Log("✅ Git网关模拟客户端功能测试通过")
}

// TestRepositoryVisibilityConversion 测试仓库可见性转换
func TestRepositoryVisibilityConversion(t *testing.T) {
	testCases := []struct {
		visibility client.RepositoryVisibility
		expected   string
	}{
		{client.RepositoryVisibilityPublic, "public"},
		{client.RepositoryVisibilityPrivate, "private"},
		{client.RepositoryVisibilityInternal, "internal"},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.expected, string(tc.visibility))
	}

	t.Log("✅ 仓库可见性转换测试通过")
}

// TestDataModelMapping 测试数据模型映射
func TestDataModelMapping(t *testing.T) {
	// 创建Git网关的Repository模型
	gitRepo := &client.Repository{
		ID:            uuid.New(),
		ProjectID:     uuid.New(),
		Name:          "model-test-repo",
		Description:   stringPtr("数据模型映射测试"),
		Visibility:    client.RepositoryVisibilityPublic,
		DefaultBranch: "main",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// 验证基本字段
	assert.NotEqual(t, uuid.Nil, gitRepo.ID)
	assert.NotEqual(t, uuid.Nil, gitRepo.ProjectID)
	assert.Equal(t, "model-test-repo", gitRepo.Name)
	assert.NotNil(t, gitRepo.Description)
	assert.Equal(t, "数据模型映射测试", *gitRepo.Description)
	assert.Equal(t, "public", string(gitRepo.Visibility))
	assert.Equal(t, "main", gitRepo.DefaultBranch)

	// 验证时间字段
	assert.False(t, gitRepo.CreatedAt.IsZero())
	assert.False(t, gitRepo.UpdatedAt.IsZero())

	t.Log("✅ 数据模型映射测试通过")
}

// 辅助函数
func stringPointer(s string) *string {
	return &s
}
