package unit

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SimpleRepository 简化的仓库模型用于测试
type SimpleRepository struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ProjectID   uuid.UUID `json:"project_id"`
	IsPrivate   bool      `json:"is_private"`
	Status      string    `json:"status"`
	CreatedBy   uuid.UUID `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SimpleBranch 简化的分支模型用于测试
type SimpleBranch struct {
	Name        string    `json:"name"`
	SHA         string    `json:"sha"`
	IsDefault   bool      `json:"is_default"`
	IsProtected bool      `json:"is_protected"`
	CreatedAt   time.Time `json:"created_at"`
}

// SimpleCommit 简化的提交模型用于测试
type SimpleCommit struct {
	SHA         string    `json:"sha"`
	Message     string    `json:"message"`
	Author      string    `json:"author"`
	AuthorEmail string    `json:"author_email"`
	Branch      string    `json:"branch"`
	CreatedAt   time.Time `json:"created_at"`
}

// SimpleTag 简化的标签模型用于测试
type SimpleTag struct {
	Name      string    `json:"name"`
	SHA       string    `json:"sha"`
	Message   string    `json:"message"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// SimpleCreateRepositoryRequest 简化的创建仓库请求用于测试
type SimpleCreateRepositoryRequest struct {
	Name        string    `json:"name" binding:"required"`
	Description string    `json:"description"`
	ProjectID   uuid.UUID `json:"project_id" binding:"required"`
	IsPrivate   bool      `json:"is_private"`
	Template    string    `json:"template,omitempty"`
}

// Git Gateway 验证函数

// validateRepositoryName 验证仓库名称
func validateRepositoryName(name string) error {
	if name == "" {
		return fmt.Errorf("仓库名称不能为空")
	}

	if len(name) < 1 || len(name) > 100 {
		return fmt.Errorf("仓库名称长度必须在1-100字符之间")
	}

	// 检查是否包含非法字符（简化版规则）
	namePattern := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	if !namePattern.MatchString(name) {
		return fmt.Errorf("仓库名称只能包含字母、数字、点、下划线和横线")
	}

	// 不能以点开头或结尾
	if strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".") {
		return fmt.Errorf("仓库名称不能以点开头或结尾")
	}

	return nil
}

// validateBranchName 验证分支名称
func validateBranchName(name string) error {
	if name == "" {
		return fmt.Errorf("分支名称不能为空")
	}

	if len(name) > 250 {
		return fmt.Errorf("分支名称不能超过250字符")
	}

	// Git分支名称规则
	branchPattern := regexp.MustCompile(`^[a-zA-Z0-9._/-]+$`)
	if !branchPattern.MatchString(name) {
		return fmt.Errorf("分支名称包含非法字符")
	}

	// 不能以/开头或结尾
	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") {
		return fmt.Errorf("分支名称不能以/开头或结尾")
	}

	// 不能包含连续的/
	if strings.Contains(name, "//") {
		return fmt.Errorf("分支名称不能包含连续的/")
	}

	return nil
}

// validateCommitSHA 验证提交SHA
func validateCommitSHA(sha string) error {
	if sha == "" {
		return fmt.Errorf("提交SHA不能为空")
	}

	// SHA通常是40位十六进制字符
	shaPattern := regexp.MustCompile(`^[a-f0-9]{7,40}$`)
	if !shaPattern.MatchString(sha) {
		return fmt.Errorf("无效的提交SHA格式")
	}

	return nil
}

// validateTagName 验证标签名称
func validateTagName(name string) error {
	if name == "" {
		return fmt.Errorf("标签名称不能为空")
	}

	if len(name) > 250 {
		return fmt.Errorf("标签名称不能超过250字符")
	}

	// 标签名称规则
	tagPattern := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	if !tagPattern.MatchString(name) {
		return fmt.Errorf("标签名称包含非法字符")
	}

	return nil
}

// validateCreateRepositoryRequest 验证创建仓库请求
func validateCreateRepositoryRequest(req *SimpleCreateRepositoryRequest) error {
	if req == nil {
		return fmt.Errorf("创建仓库请求不能为nil")
	}

	if err := validateRepositoryName(req.Name); err != nil {
		return err
	}

	if req.ProjectID == uuid.Nil {
		return fmt.Errorf("项目ID不能为空")
	}

	if len(req.Description) > 500 {
		return fmt.Errorf("仓库描述不能超过500字符")
	}

	return nil
}

// TestRepositoryValidation 测试仓库验证
func TestRepositoryValidation(t *testing.T) {
	tests := []struct {
		name          string
		repo          *SimpleRepository
		expectedError string
	}{
		{
			name: "有效的仓库",
			repo: &SimpleRepository{
				ID:          uuid.New(),
				Name:        "my-repo",
				Description: "测试仓库",
				ProjectID:   uuid.New(),
				IsPrivate:   false,
				Status:      "active",
				CreatedBy:   uuid.New(),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			expectedError: "",
		},
		{
			name: "无效的仓库名称 - 空名称",
			repo: &SimpleRepository{
				ID:        uuid.New(),
				Name:      "",
				ProjectID: uuid.New(),
			},
			expectedError: "仓库名称不能为空",
		},
		{
			name: "无效的仓库名称 - 包含非法字符",
			repo: &SimpleRepository{
				ID:        uuid.New(),
				Name:      "my@repo!",
				ProjectID: uuid.New(),
			},
			expectedError: "仓库名称只能包含字母、数字、点、下划线和横线",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRepositoryName(tt.repo.Name)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestRepositoryNameValidation 测试仓库名称验证
func TestRepositoryNameValidation(t *testing.T) {
	tests := []struct {
		name          string
		repoName      string
		expectedError string
	}{
		{"有效的简单名称", "my-repo", ""},
		{"有效的复杂名称", "frontend-ui-components", ""},
		{"有效的带数字名称", "service-v2.1", ""},
		{"有效的带下划线名称", "data_processor", ""},
		{"空名称", "", "仓库名称不能为空"},
		{"包含@符号", "my@repo", "仓库名称只能包含字母、数字、点、下划线和横线"},
		{"包含空格", "my repo", "仓库名称只能包含字母、数字、点、下划线和横线"},
		{"包含特殊字符", "repo#1", "仓库名称只能包含字母、数字、点、下划线和横线"},
		{"以点开头", ".gitignore-repo", "仓库名称不能以点开头或结尾"},
		{"以点结尾", "my-repo.", "仓库名称不能以点开头或结尾"},
		{"超长名称", strings.Repeat("a", 101), "仓库名称长度必须在1-100字符之间"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRepositoryName(tt.repoName)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestBranchNameValidation 测试分支名称验证
func TestBranchNameValidation(t *testing.T) {
	tests := []struct {
		name          string
		branchName    string
		expectedError string
	}{
		{"有效的主分支", "main", ""},
		{"有效的开发分支", "develop", ""},
		{"有效的功能分支", "feature/user-auth", ""},
		{"有效的修复分支", "hotfix/critical-bug", ""},
		{"有效的版本分支", "release/v1.2.0", ""},
		{"空分支名", "", "分支名称不能为空"},
		{"以/开头", "/feature", "分支名称不能以/开头或结尾"},
		{"以/结尾", "feature/", "分支名称不能以/开头或结尾"},
		{"连续的/", "feature//auth", "分支名称不能包含连续的/"},
		{"包含非法字符", "feature@auth", "分支名称包含非法字符"},
		{"超长分支名", strings.Repeat("a", 251), "分支名称不能超过250字符"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBranchName(tt.branchName)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestCommitSHAValidation 测试提交SHA验证
func TestCommitSHAValidation(t *testing.T) {
	tests := []struct {
		name          string
		sha           string
		expectedError string
	}{
		{"有效的完整SHA", "a1b2c3d4e5f6789012345678901234567890abcd", ""},
		{"有效的短SHA", "a1b2c3d", ""},
		{"有效的中等长度SHA", "a1b2c3d4e5f67890", ""},
		{"空SHA", "", "提交SHA不能为空"},
		{"包含大写字母", "A1B2C3D4E5F6789012345678901234567890ABCD", "无效的提交SHA格式"},
		{"包含非法字符", "g1h2i3j4k5l6789012345678901234567890mnop", "无效的提交SHA格式"},
		{"太短的SHA", "a1b2c", "无效的提交SHA格式"},
		{"包含特殊字符", "a1b2c3d-e5f6789012345678901234567890abcd", "无效的提交SHA格式"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCommitSHA(tt.sha)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestTagNameValidation 测试标签名称验证
func TestTagNameValidation(t *testing.T) {
	tests := []struct {
		name          string
		tagName       string
		expectedError string
	}{
		{"有效的版本标签", "v1.0.0", ""},
		{"有效的简单标签", "release", ""},
		{"有效的带横线标签", "beta-1", ""},
		{"有效的带下划线标签", "stable_build", ""},
		{"空标签名", "", "标签名称不能为空"},
		{"包含非法字符", "v1.0@beta", "标签名称包含非法字符"},
		{"包含空格", "v1 0 0", "标签名称包含非法字符"},
		{"包含斜杠", "release/v1.0", "标签名称包含非法字符"},
		{"超长标签名", strings.Repeat("a", 251), "标签名称不能超过250字符"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTagName(tt.tagName)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestCreateRepositoryRequest 测试创建仓库请求验证
func TestCreateRepositoryRequest(t *testing.T) {
	validProjectID := uuid.New()

	tests := []struct {
		name          string
		req           *SimpleCreateRepositoryRequest
		expectedError string
	}{
		{
			name: "有效的创建请求",
			req: &SimpleCreateRepositoryRequest{
				Name:        "my-repo",
				Description: "测试仓库",
				ProjectID:   validProjectID,
				IsPrivate:   false,
			},
			expectedError: "",
		},
		{
			name: "名称为空",
			req: &SimpleCreateRepositoryRequest{
				Name:      "",
				ProjectID: validProjectID,
			},
			expectedError: "仓库名称不能为空",
		},
		{
			name: "项目ID为空",
			req: &SimpleCreateRepositoryRequest{
				Name:      "my-repo",
				ProjectID: uuid.Nil,
			},
			expectedError: "项目ID不能为空",
		},
		{
			name: "描述过长",
			req: &SimpleCreateRepositoryRequest{
				Name:        "my-repo",
				Description: strings.Repeat("a", 501),
				ProjectID:   validProjectID,
			},
			expectedError: "仓库描述不能超过500字符",
		},
		{
			name:          "nil请求",
			req:           nil,
			expectedError: "创建仓库请求不能为nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCreateRepositoryRequest(tt.req)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestGitGatewayEdgeCases 测试Git Gateway边界情况
func TestGitGatewayEdgeCases(t *testing.T) {
	t.Run("nil repository", func(t *testing.T) {
		var repo *SimpleRepository
		require.Nil(t, repo)
		// nil repository 应该在更高层被处理
	})

	t.Run("nil branch", func(t *testing.T) {
		var branch *SimpleBranch
		require.Nil(t, branch)
		// nil branch 应该在更高层被处理
	})

	t.Run("极长的仓库描述", func(t *testing.T) {
		description := strings.Repeat("这是一个很长的描述", 100)
		req := &SimpleCreateRepositoryRequest{
			Name:        "test-repo",
			Description: description,
			ProjectID:   uuid.New(),
		}
		err := validateCreateRepositoryRequest(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "仓库描述不能超过500字符")
	})

	t.Run("Unicode字符仓库名", func(t *testing.T) {
		err := validateRepositoryName("测试仓库")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "仓库名称只能包含字母、数字、点、下划线和横线")
	})

	t.Run("特殊格式的分支名", func(t *testing.T) {
		specialBranches := []string{
			"feature/PROJ-123-add-new-feature",
			"bugfix/fix-memory-leak.v2",
			"release/2024.01.15",
		}

		for _, branch := range specialBranches {
			err := validateBranchName(branch)
			assert.NoError(t, err, "分支名 %s 应该是有效的", branch)
		}
	})
}

// TestGitGatewayPerformance 测试Git Gateway性能
func TestGitGatewayPerformance(t *testing.T) {
	t.Run("批量仓库名验证", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 1000; i++ {
			repoName := fmt.Sprintf("repo-%d", i)
			err := validateRepositoryName(repoName)
			assert.NoError(t, err)
		}

		duration := time.Since(start)
		t.Logf("批量验证1000个仓库名耗时: %v", duration)

		// 验证应该在10毫秒内完成
		assert.Less(t, duration, 10*time.Millisecond)
	})

	t.Run("批量分支名验证", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 1000; i++ {
			branchName := fmt.Sprintf("feature/branch-%d", i)
			err := validateBranchName(branchName)
			assert.NoError(t, err)
		}

		duration := time.Since(start)
		t.Logf("批量验证1000个分支名耗时: %v", duration)

		// 验证应该在10毫秒内完成
		assert.Less(t, duration, 10*time.Millisecond)
	})
}

// MockGitService Git服务模拟实现
type MockGitService struct {
	repositories map[uuid.UUID]*SimpleRepository
	branches     map[uuid.UUID][]SimpleBranch
	commits      map[uuid.UUID][]SimpleCommit
	tags         map[uuid.UUID][]SimpleTag
}

// NewMockGitService 创建Git服务模拟
func NewMockGitService() *MockGitService {
	return &MockGitService{
		repositories: make(map[uuid.UUID]*SimpleRepository),
		branches:     make(map[uuid.UUID][]SimpleBranch),
		commits:      make(map[uuid.UUID][]SimpleCommit),
		tags:         make(map[uuid.UUID][]SimpleTag),
	}
}

// CreateRepository 模拟创建仓库
func (m *MockGitService) CreateRepository(req *SimpleCreateRepositoryRequest) (*SimpleRepository, error) {
	if err := validateCreateRepositoryRequest(req); err != nil {
		return nil, err
	}

	repo := &SimpleRepository{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		ProjectID:   req.ProjectID,
		IsPrivate:   req.IsPrivate,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	m.repositories[repo.ID] = repo

	// 初始化默认分支
	m.branches[repo.ID] = []SimpleBranch{
		{
			Name:      "main",
			SHA:       "a1b2c3d4e5f6789012345678901234567890abcd",
			IsDefault: true,
			CreatedAt: time.Now(),
		},
	}

	return repo, nil
}

// TestMockGitService 测试Git服务模拟
func TestMockGitService(t *testing.T) {
	mockService := NewMockGitService()

	t.Run("创建仓库成功", func(t *testing.T) {
		req := &SimpleCreateRepositoryRequest{
			Name:        "test-repo",
			Description: "测试仓库",
			ProjectID:   uuid.New(),
			IsPrivate:   false,
		}

		repo, err := mockService.CreateRepository(req)
		require.NoError(t, err)
		require.NotNil(t, repo)

		assert.Equal(t, req.Name, repo.Name)
		assert.Equal(t, req.Description, repo.Description)
		assert.Equal(t, req.ProjectID, repo.ProjectID)
		assert.Equal(t, req.IsPrivate, repo.IsPrivate)
		assert.Equal(t, "active", repo.Status)
		assert.NotZero(t, repo.ID)

		// 验证默认分支被创建
		branches, exists := mockService.branches[repo.ID]
		require.True(t, exists)
		require.Len(t, branches, 1)
		assert.Equal(t, "main", branches[0].Name)
		assert.True(t, branches[0].IsDefault)
	})

	t.Run("创建仓库失败 - 无效名称", func(t *testing.T) {
		req := &SimpleCreateRepositoryRequest{
			Name:      "", // 无效名称
			ProjectID: uuid.New(),
		}

		repo, err := mockService.CreateRepository(req)
		assert.Error(t, err)
		assert.Nil(t, repo)
		assert.Contains(t, err.Error(), "仓库名称不能为空")
	})
}
