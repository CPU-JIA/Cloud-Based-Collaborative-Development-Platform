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

// TestGitGatewayClientBasic 测试Git网关客户端基本功能
func TestGitGatewayClientBasic(t *testing.T) {
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

// TestMockGitGatewayClient 测试模拟Git网关客户端
func TestMockGitGatewayClient(t *testing.T) {
	// 创建模拟客户端
	mockClient := new(MockGitGatewayClient)

	// 设置模拟行为
	projectID := uuid.New()
	repoID := uuid.New()
	
	mockRepo := &client.Repository{
		ID:          repoID,
		ProjectID:   projectID,
		Name:        "test-repo",
		Description: stringPtr("测试仓库"),
		Visibility:  client.RepositoryVisibilityPrivate,
		DefaultBranch: "main",
	}

	// 设置创建仓库的期望
	mockClient.On("CreateRepository", mock.Anything, mock.MatchedBy(func(req *client.CreateRepositoryRequest) bool {
		return req.Name == "test-repo" && req.ProjectID == projectID.String()
	})).Return(mockRepo, nil)

	// 设置获取仓库的期望
	mockClient.On("GetRepository", mock.Anything, repoID).Return(mockRepo, nil)

	// 测试创建仓库
	ctx := context.Background()
	createReq := &client.CreateRepositoryRequest{
		ProjectID:     projectID.String(),
		Name:          "test-repo",
		Description:   stringPtr("测试仓库"),
		Visibility:    client.RepositoryVisibilityPrivate,
		DefaultBranch: stringPtr("main"),
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

	// 验证所有期望都被调用
	mockClient.AssertExpectations(t)

	t.Log("✅ 模拟Git网关客户端测试通过")
}

// TestDataModelConversion 测试数据模型转换
func TestDataModelConversion(t *testing.T) {
	// Git网关的Repository模型
	gitRepo := &client.Repository{
		ID:          uuid.New(),
		ProjectID:   uuid.New(),
		Name:        "test-conversion-repo",
		Description: stringPtr("数据模型转换测试"),
		Visibility:  client.RepositoryVisibilityPublic,
		DefaultBranch: "main",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 验证枚举值转换
	assert.Equal(t, "public", string(gitRepo.Visibility))

	// 验证指针字段处理
	assert.NotNil(t, gitRepo.Description)
	assert.Equal(t, "数据模型转换测试", *gitRepo.Description)

	t.Log("✅ 数据模型转换测试通过")
}

// TestRepositoryVisibilityEnum 测试仓库可见性枚举
func TestRepositoryVisibilityEnum(t *testing.T) {
	// 测试所有可见性枚举值
	visibilities := []client.RepositoryVisibility{
		client.RepositoryVisibilityPublic,
		client.RepositoryVisibilityPrivate,
		client.RepositoryVisibilityInternal,
	}

	expectedStrings := []string{"public", "private", "internal"}

	for i, visibility := range visibilities {
		assert.Equal(t, expectedStrings[i], string(visibility))
	}

	t.Log("✅ 仓库可见性枚举测试通过")
}

// 辅助函数定义在 test_helpers.go 中