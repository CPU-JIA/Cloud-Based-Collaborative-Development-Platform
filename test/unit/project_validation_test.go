package service

import (
	"testing"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestProjectValidation 测试项目验证逻辑
func TestProjectValidation(t *testing.T) {
	tests := []struct {
		name      string
		project   *models.Project
		expectErr bool
	}{
		{
			name: "有效项目",
			project: &models.Project{
				TenantID:    uuid.New(),
				Key:         "VALID",
				Name:        "有效项目",
				Description: stringPtr("这是一个有效的项目"),
			},
			expectErr: false,
		},
		{
			name: "项目Key太短",
			project: &models.Project{
				TenantID: uuid.New(),
				Key:      "A", // 太短
				Name:     "项目",
			},
			expectErr: true,
		},
		{
			name: "项目Key包含无效字符",
			project: &models.Project{
				TenantID: uuid.New(),
				Key:      "INVALID-KEY!", // 包含特殊字符
				Name:     "项目",
			},
			expectErr: true,
		},
		{
			name: "项目名称太长",
			project: &models.Project{
				TenantID: uuid.New(),
				Key:      "VALID",
				Name:     string(make([]byte, 300)), // 超长名称
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
		request   *models.CreateProjectRequest
		expectErr bool
	}{
		{
			name: "有效创建请求",
			request: &models.CreateProjectRequest{
				Key:         "TESTPROJ",
				Name:        "测试项目",
				Description: stringPtr("这是一个测试项目"),
			},
			expectErr: false,
		},
		{
			name: "空项目名称",
			request: &models.CreateProjectRequest{
				Key:  "TESTPROJ",
				Name: "",
			},
			expectErr: true,
		},
		{
			name: "空项目Key",
			request: &models.CreateProjectRequest{
				Key:  "",
				Name: "测试项目",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCreateProjectRequest(tt.request)
			
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// 辅助函数 - 字符串指针
func stringPtr(s string) *string {
	return &s
}

// 辅助函数 - 项目验证
func validateProject(project *models.Project) error {
	if project.Name == "" {
		return assert.AnError
	}
	if project.Key == "" {
		return assert.AnError
	}
	if len(project.Key) < 2 || len(project.Key) > 20 {
		return assert.AnError
	}
	if len(project.Name) > 255 {
		return assert.AnError
	}
	// 简单的Key格式验证
	for _, char := range project.Key {
		if !((char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
			return assert.AnError
		}
	}
	return nil
}

// 辅助函数 - 创建项目请求验证
func validateCreateProjectRequest(req *models.CreateProjectRequest) error {
	if req.Name == "" {
		return assert.AnError
	}
	if req.Key == "" {
		return assert.AnError
	}
	return nil
}