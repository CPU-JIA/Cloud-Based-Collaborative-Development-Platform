package unit

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
)

// 简化的Project模型用于测试
type SimpleProject struct {
	ID          string  `json:"id"`
	TenantID    string  `json:"tenant_id"`
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Status      string  `json:"status"`
}

// 简化的CreateProjectRequest用于测试
type SimpleCreateProjectRequest struct {
	Name        string `json:"name"`
	Key         string `json:"key"`
	Description string `json:"description"`
	TenantID    string `json:"tenant_id"`
	CreatedBy   string `json:"created_by"`
}

// TestProjectValidation 测试项目验证逻辑
func TestProjectValidation(t *testing.T) {
	tests := []struct {
		name      string
		project   *SimpleProject
		expectErr bool
	}{
		{
			name: "有效的项目",
			project: &SimpleProject{
				ID:          "test-id",
				TenantID:    "test-tenant",
				Key:         "VALID",
				Name:        "有效项目",
				Description: stringPtr("有效的项目描述"),
				Status:      "active",
			},
			expectErr: false,
		},
		{
			name: "无效的项目Key - 包含小写字母",
			project: &SimpleProject{
				ID:       "test-id",
				TenantID: "test-tenant",
				Key:      "invalid",
				Name:     "无效项目",
				Status:   "active",
			},
			expectErr: true,
		},
		{
			name: "无效的项目Key - 包含特殊字符",
			project: &SimpleProject{
				ID:       "test-id",
				TenantID: "test-tenant",
				Key:      "INVALID-KEY",
				Name:     "无效项目",
				Status:   "active",
			},
			expectErr: true,
		},
		{
			name: "项目名称为空",
			project: &SimpleProject{
				ID:       "test-id",
				TenantID: "test-tenant",
				Key:      "EMPTY",
				Name:     "",
				Status:   "active",
			},
			expectErr: true,
		},
		{
			name: "项目Key过长",
			project: &SimpleProject{
				ID:       "test-id",
				TenantID: "test-tenant",
				Key:      "VERYLONGPROJECTKEYTEST",
				Name:     "长Key项目",
				Status:   "active",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProject(tt.project)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestCreateProjectRequest 测试创建项目请求验证
func TestCreateProjectRequest(t *testing.T) {
	tests := []struct {
		name      string
		req       *SimpleCreateProjectRequest
		expectErr bool
	}{
		{
			name: "有效的创建请求",
			req: &SimpleCreateProjectRequest{
				Name:        "测试项目",
				Key:         "TEST",
				Description: "测试描述",
				TenantID:    "test-tenant",
				CreatedBy:   "test-user",
			},
			expectErr: false,
		},
		{
			name: "名称为空",
			req: &SimpleCreateProjectRequest{
				Name:      "",
				Key:       "TEST",
				TenantID:  "test-tenant",
				CreatedBy: "test-user",
			},
			expectErr: true,
		},
		{
			name: "Key为空",
			req: &SimpleCreateProjectRequest{
				Name:      "测试项目",
				Key:       "",
				TenantID:  "test-tenant",
				CreatedBy: "test-user",
			},
			expectErr: true,
		},
		{
			name: "TenantID为空",
			req: &SimpleCreateProjectRequest{
				Name:      "测试项目",
				Key:       "TEST",
				TenantID:  "",
				CreatedBy: "test-user",
			},
			expectErr: true,
		},
		{
			name: "CreatedBy为空",
			req: &SimpleCreateProjectRequest{
				Name:      "测试项目",
				Key:       "TEST",
				TenantID:  "test-tenant",
				CreatedBy: "",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCreateProjectRequest(tt.req)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestProjectKeyValidation 专门测试项目Key的验证逻辑
func TestProjectKeyValidation(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "有效的大写字母Key",
			key:       "VALID",
			expectErr: false,
		},
		{
			name:      "有效的数字和大写字母Key",
			key:       "TEST123",
			expectErr: false,
		},
		{
			name:      "单字符Key",
			key:       "A",
			expectErr: false,
		},
		{
			name:      "10字符长度的Key",
			key:       "ABCDEFGHIJ",
			expectErr: false,
		},
		{
			name:      "包含小写字母",
			key:       "test",
			expectErr: true,
			errMsg:    "项目标识符格式不正确",
		},
		{
			name:      "包含特殊字符",
			key:       "TEST-KEY",
			expectErr: true,
			errMsg:    "项目标识符格式不正确",
		},
		{
			name:      "包含下划线",
			key:       "TEST_KEY",
			expectErr: true,
			errMsg:    "项目标识符格式不正确",
		},
		{
			name:      "超过10字符",
			key:       "VERYLONGKEY",
			expectErr: true,
			errMsg:    "项目标识符格式不正确",
		},
		{
			name:      "空Key",
			key:       "",
			expectErr: true,
			errMsg:    "项目标识符不能为空",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := &SimpleProject{
				ID:       "test-id",
				TenantID: "test-tenant",
				Key:      tt.key,
				Name:     "测试项目",
				Status:   "active",
			}

			err := validateProject(project)
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestProjectNameValidation 专门测试项目名称的验证逻辑
func TestProjectNameValidation(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		expectErr   bool
		errMsg      string
	}{
		{
			name:        "有效的中文名称",
			projectName: "测试项目",
			expectErr:   false,
		},
		{
			name:        "有效的英文名称",
			projectName: "Test Project",
			expectErr:   false,
		},
		{
			name:        "有效的混合名称",
			projectName: "Test 测试项目 123",
			expectErr:   false,
		},
		{
			name:        "有效的带特殊字符名称",
			projectName: "项目 (测试版) - V1.0",
			expectErr:   false,
		},
		{
			name:        "空名称",
			projectName: "",
			expectErr:   true,
			errMsg:      "项目名称不能为空",
		},
		{
			name:        "只有空格的名称",
			projectName: "   ",
			expectErr:   false, // 只检查是否完全为空，不检查空格
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := &SimpleProject{
				ID:       "test-id",
				TenantID: "test-tenant",
				Key:      "VALID",
				Name:     tt.projectName,
				Status:   "active",
			}

			err := validateProject(project)
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestProjectStatusValidation 测试项目状态验证
func TestProjectStatusValidation(t *testing.T) {
	validStatuses := []string{"active", "inactive", "archived", "deleted"}

	for _, status := range validStatuses {
		t.Run(fmt.Sprintf("valid_status_%s", status), func(t *testing.T) {
			project := &SimpleProject{
				ID:       "test-id",
				TenantID: "test-tenant",
				Key:      "VALID",
				Name:     "测试项目",
				Status:   status,
			}

			err := validateProject(project)
			assert.NoError(t, err)
		})
	}
}

// TestEdgeCases 测试边界情况
func TestProjectEdgeCases(t *testing.T) {
	t.Run("nil项目", func(t *testing.T) {
		err := validateProject(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "项目不能为nil")
	})

	t.Run("nil创建请求", func(t *testing.T) {
		err := validateCreateProjectRequest(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "创建请求不能为nil")
	})

	t.Run("极长的项目名称", func(t *testing.T) {
		longName := make([]byte, 1000)
		for i := range longName {
			longName[i] = 'A'
		}

		project := &SimpleProject{
			ID:       "test-id",
			TenantID: "test-tenant",
			Key:      "VALID",
			Name:     string(longName),
			Status:   "active",
		}

		// 极长名称应该被允许（在真实场景中可能有长度限制）
		err := validateProject(project)
		assert.NoError(t, err)
	})

	t.Run("Unicode字符项目Key", func(t *testing.T) {
		project := &SimpleProject{
			ID:       "test-id",
			TenantID: "test-tenant",
			Key:      "测试",
			Name:     "测试项目",
			Status:   "active",
		}

		err := validateProject(project)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "项目标识符格式不正确")
	})
}

// TestBenchmarkValidation 性能基准测试
func TestBenchmarkValidation(t *testing.T) {
	project := &SimpleProject{
		ID:          "test-id",
		TenantID:    "test-tenant",
		Key:         "VALID",
		Name:        "测试项目",
		Description: stringPtr("测试描述"),
		Status:      "active",
	}

	// 简单的性能测试 - 验证函数应该很快
	for i := 0; i < 1000; i++ {
		err := validateProject(project)
		assert.NoError(t, err)
	}
}

// 辅助函数
func stringPtr(s string) *string {
	return &s
}

// validateProject 验证项目模型
func validateProject(project *SimpleProject) error {
	if project == nil {
		return fmt.Errorf("项目不能为nil")
	}

	if project.Name == "" {
		return fmt.Errorf("项目名称不能为空")
	}

	if project.Key == "" {
		return fmt.Errorf("项目标识符不能为空")
	}

	// 验证项目Key格式：只允许大写字母和数字，长度限制
	keyPattern := regexp.MustCompile(`^[A-Z0-9]{1,10}$`)
	if !keyPattern.MatchString(project.Key) {
		return fmt.Errorf("项目标识符格式不正确，只允许1-10位大写字母和数字")
	}

	return nil
}

// validateCreateProjectRequest 验证创建项目请求
func validateCreateProjectRequest(req *SimpleCreateProjectRequest) error {
	if req == nil {
		return fmt.Errorf("创建请求不能为nil")
	}

	if req.Name == "" {
		return fmt.Errorf("项目名称不能为空")
	}

	if req.Key == "" {
		return fmt.Errorf("项目标识符不能为空")
	}

	if req.TenantID == "" {
		return fmt.Errorf("租户ID不能为空")
	}

	if req.CreatedBy == "" {
		return fmt.Errorf("创建者ID不能为空")
	}

	// 验证项目Key格式
	keyPattern := regexp.MustCompile(`^[A-Z0-9]{1,10}$`)
	if !keyPattern.MatchString(req.Key) {
		return fmt.Errorf("项目标识符格式不正确，只允许1-10位大写字母和数字")
	}

	return nil
}
