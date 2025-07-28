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

// SimplePipeline 简化的流水线模型用于测试
type SimplePipeline struct {
	ID          uuid.UUID              `json:"id"`
	TenantID    uuid.UUID              `json:"tenant_id"`
	ProjectID   uuid.UUID              `json:"project_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
	Status      string                 `json:"status"`
	Trigger     string                 `json:"trigger"`
	Branch      string                 `json:"branch"`
	IsActive    bool                   `json:"is_active"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// SimpleBuild 简化的构建模型用于测试
type SimpleBuild struct {
	ID         uuid.UUID              `json:"id"`
	PipelineID uuid.UUID              `json:"pipeline_id"`
	ProjectID  uuid.UUID              `json:"project_id"`
	Number     int                    `json:"number"`
	Status     string                 `json:"status"`
	Branch     string                 `json:"branch"`
	CommitSHA  string                 `json:"commit_sha"`
	CommitMsg  string                 `json:"commit_message"`
	Config     map[string]interface{} `json:"config"`
	StartedAt  *time.Time             `json:"started_at"`
	FinishedAt *time.Time             `json:"finished_at"`
	Duration   *time.Duration         `json:"duration"`
	CreatedBy  uuid.UUID              `json:"created_by"`
}

// SimpleStage 简化的阶段模型用于测试
type SimpleStage struct {
	ID       uuid.UUID              `json:"id"`
	BuildID  uuid.UUID              `json:"build_id"`
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	Status   string                 `json:"status"`
	Config   map[string]interface{} `json:"config"`
	Order    int                    `json:"order"`
	Commands []string               `json:"commands"`
	Logs     string                 `json:"logs"`
	Duration *time.Duration         `json:"duration"`
}

// SimpleDeployment 简化的部署模型用于测试
type SimpleDeployment struct {
	ID            uuid.UUID              `json:"id"`
	PipelineID    uuid.UUID              `json:"pipeline_id"`
	BuildID       uuid.UUID              `json:"build_id"`
	Environment   string                 `json:"environment"`
	Strategy      string                 `json:"strategy"`
	Status        string                 `json:"status"`
	Config        map[string]interface{} `json:"config"`
	Version       string                 `json:"version"`
	RollbackCount int                    `json:"rollback_count"`
	CreatedAt     time.Time              `json:"created_at"`
	DeployedAt    *time.Time             `json:"deployed_at"`
}

// SimpleEnvironment 简化的环境模型用于测试
type SimpleEnvironment struct {
	ID          uuid.UUID              `json:"id"`
	TenantID    uuid.UUID              `json:"tenant_id"`
	ProjectID   uuid.UUID              `json:"project_id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"`
	Config      map[string]interface{} `json:"config"`
	Variables   map[string]string      `json:"variables"`
	IsProtected bool                   `json:"is_protected"`
}

// SimpleCreatePipelineRequest 简化的创建流水线请求用于测试
type SimpleCreatePipelineRequest struct {
	TenantID    uuid.UUID              `json:"tenant_id"`
	ProjectID   uuid.UUID              `json:"project_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
	Trigger     string                 `json:"trigger"`
	Branch      string                 `json:"branch"`
	CreatedBy   uuid.UUID              `json:"created_by"`
}

// CICD Service 验证函数

// validatePipelineName 验证流水线名称
func validatePipelineName(name string) error {
	if name == "" {
		return fmt.Errorf("流水线名称不能为空")
	}

	if len(name) < 2 || len(name) > 100 {
		return fmt.Errorf("流水线名称长度必须在2-100字符之间")
	}

	// 流水线名称只能包含字母、数字、下划线、横线、空格和点号
	namePattern := regexp.MustCompile(`^[a-zA-Z0-9_\-\s\.]+$`)
	if !namePattern.MatchString(name) {
		return fmt.Errorf("流水线名称只能包含字母、数字、下划线、横线和空格")
	}

	return nil
}

// validatePipelineStatus 验证流水线状态
func validatePipelineStatus(status string) error {
	if status == "" {
		return fmt.Errorf("流水线状态不能为空")
	}

	validStatuses := []string{"active", "inactive", "disabled", "archived"}
	for _, validStatus := range validStatuses {
		if status == validStatus {
			return nil
		}
	}

	return fmt.Errorf("无效的流水线状态: %s", status)
}

// validateTriggerType 验证触发类型
func validateTriggerType(trigger string) error {
	if trigger == "" {
		return fmt.Errorf("触发类型不能为空")
	}

	validTriggers := []string{"manual", "push", "pull_request", "schedule", "webhook", "tag"}
	for _, validTrigger := range validTriggers {
		if trigger == validTrigger {
			return nil
		}
	}

	return fmt.Errorf("无效的触发类型: %s", trigger)
}

// validateBranchName 验证分支名称
func cicdValidateBranchName(branch string) error {
	if branch == "" {
		return fmt.Errorf("分支名称不能为空")
	}

	if len(branch) > 255 {
		return fmt.Errorf("分支名称长度不能超过255字符")
	}

	// Git分支名称规则：允许字母、数字、斜杠、横线、下划线、点号
	// 不能以点开头或结尾，不能以斜杠结尾
	if strings.HasPrefix(branch, ".") || strings.HasSuffix(branch, ".") || strings.HasSuffix(branch, "/") {
		return fmt.Errorf("分支名称格式不正确")
	}

	// 检查不允许的字符
	invalidChars := []string{" ", "~", "^", ":", "?", "*", "[", "\\", "@{", ".."}
	for _, char := range invalidChars {
		if strings.Contains(branch, char) {
			return fmt.Errorf("分支名称不能包含字符: %s", char)
		}
	}

	// 基本字符检查
	branchPattern := regexp.MustCompile(`^[a-zA-Z0-9/_\-\.]+$`)
	if !branchPattern.MatchString(branch) {
		return fmt.Errorf("分支名称格式不正确")
	}

	return nil
}

// validateBuildStatus 验证构建状态
func validateBuildStatus(status string) error {
	if status == "" {
		return fmt.Errorf("构建状态不能为空")
	}

	validStatuses := []string{"pending", "running", "success", "failure", "cancelled", "timeout"}
	for _, validStatus := range validStatuses {
		if status == validStatus {
			return nil
		}
	}

	return fmt.Errorf("无效的构建状态: %s", status)
}

// validateStageType 验证阶段类型
func validateStageType(stageType string) error {
	if stageType == "" {
		return fmt.Errorf("阶段类型不能为空")
	}

	validTypes := []string{"build", "test", "security_scan", "deploy", "approval", "notification"}
	for _, validType := range validTypes {
		if stageType == validType {
			return nil
		}
	}

	return fmt.Errorf("无效的阶段类型: %s", stageType)
}

// validateEnvironmentType 验证环境类型
func validateEnvironmentType(envType string) error {
	if envType == "" {
		return fmt.Errorf("环境类型不能为空")
	}

	validTypes := []string{"development", "testing", "staging", "production"}
	for _, validType := range validTypes {
		if envType == validType {
			return nil
		}
	}

	return fmt.Errorf("无效的环境类型: %s", envType)
}

// validateDeploymentStrategy 验证部署策略
func validateDeploymentStrategy(strategy string) error {
	if strategy == "" {
		return fmt.Errorf("部署策略不能为空")
	}

	validStrategies := []string{"rolling", "blue_green", "canary", "recreate"}
	for _, validStrategy := range validStrategies {
		if strategy == validStrategy {
			return nil
		}
	}

	return fmt.Errorf("无效的部署策略: %s", strategy)
}

// validateCommitSHA 验证提交SHA
func cicdValidateCommitSHA(sha string) error {
	if sha == "" {
		return fmt.Errorf("提交SHA不能为空")
	}

	// Git SHA可以是短SHA(7-40位)或完整SHA(40位)
	if len(sha) < 7 || len(sha) > 40 {
		return fmt.Errorf("提交SHA长度必须在7-40字符之间")
	}

	// SHA只能包含十六进制字符
	shaPattern := regexp.MustCompile(`^[a-fA-F0-9]+$`)
	if !shaPattern.MatchString(sha) {
		return fmt.Errorf("提交SHA只能包含十六进制字符")
	}

	return nil
}

// validateCreatePipelineRequest 验证创建流水线请求
func validateCreatePipelineRequest(req *SimpleCreatePipelineRequest) error {
	if req == nil {
		return fmt.Errorf("创建流水线请求不能为nil")
	}

	if req.TenantID == uuid.Nil {
		return fmt.Errorf("租户ID不能为空")
	}

	if req.ProjectID == uuid.Nil {
		return fmt.Errorf("项目ID不能为空")
	}

	if req.CreatedBy == uuid.Nil {
		return fmt.Errorf("创建者ID不能为空")
	}

	if err := validatePipelineName(req.Name); err != nil {
		return fmt.Errorf("流水线名称: %v", err)
	}

	if err := validateTriggerType(req.Trigger); err != nil {
		return fmt.Errorf("触发类型: %v", err)
	}

	if err := cicdValidateBranchName(req.Branch); err != nil {
		return fmt.Errorf("分支名称: %v", err)
	}

	return nil
}

// TestPipelineNameValidation 测试流水线名称验证
func TestPipelineNameValidation(t *testing.T) {
	tests := []struct {
		name          string
		pipelineName  string
		expectedError string
	}{
		{"有效的英文名称", "Build Pipeline", ""},
		{"有效的带下划线名称", "build_pipeline", ""},
		{"有效的带横线名称", "build-pipeline", ""},
		{"有效的数字名称", "pipeline123", ""},
		{"有效的混合名称", "Build Pipeline v1.0", ""},
		{"有效的带点号名称", "build.pipeline", ""},
		{"空名称", "", "流水线名称不能为空"},
		{"名称过短", "a", "流水线名称长度必须在2-100字符之间"},
		{"名称过长", strings.Repeat("a", 101), "流水线名称长度必须在2-100字符之间"},
		{"包含特殊字符", "build@pipeline", "流水线名称只能包含字母、数字、下划线、横线和空格"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePipelineName(tt.pipelineName)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestPipelineStatusValidation 测试流水线状态验证
func TestPipelineStatusValidation(t *testing.T) {
	tests := []struct {
		name          string
		status        string
		expectedError string
	}{
		{"活跃状态", "active", ""},
		{"非活跃状态", "inactive", ""},
		{"禁用状态", "disabled", ""},
		{"归档状态", "archived", ""},
		{"空状态", "", "流水线状态不能为空"},
		{"无效状态", "unknown", "无效的流水线状态"},
		{"大写状态", "ACTIVE", "无效的流水线状态"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePipelineStatus(tt.status)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestTriggerTypeValidation 测试触发类型验证
func TestTriggerTypeValidation(t *testing.T) {
	tests := []struct {
		name          string
		trigger       string
		expectedError string
	}{
		{"手动触发", "manual", ""},
		{"推送触发", "push", ""},
		{"拉取请求触发", "pull_request", ""},
		{"计划触发", "schedule", ""},
		{"Webhook触发", "webhook", ""},
		{"标签触发", "tag", ""},
		{"空触发类型", "", "触发类型不能为空"},
		{"无效触发类型", "invalid", "无效的触发类型"},
		{"大写触发类型", "PUSH", "无效的触发类型"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTriggerType(tt.trigger)

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
func TestCICDBranchNameValidation(t *testing.T) {
	tests := []struct {
		name          string
		branch        string
		expectedError string
	}{
		{"有效的主分支", "main", ""},
		{"有效的开发分支", "develop", ""},
		{"有效的功能分支", "feature/user-auth", ""},
		{"有效的修复分支", "hotfix/bug-123", ""},
		{"有效的发布分支", "release/v1.0.0", ""},
		{"空分支名", "", "分支名称不能为空"},
		{"分支名过长", strings.Repeat("a", 256), "分支名称长度不能超过255字符"},
		{"以点开头", ".hidden", "分支名称格式不正确"},
		{"以斜杠结尾", "feature/", "分支名称格式不正确"},
		{"包含空格", "feature branch", "分支名称不能包含字符"},
		{"包含波浪号", "feature~1", "分支名称不能包含字符"},
		{"包含冒号", "feature:test", "分支名称不能包含字符"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cicdValidateBranchName(tt.branch)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestBuildStatusValidation 测试构建状态验证
func TestBuildStatusValidation(t *testing.T) {
	tests := []struct {
		name          string
		status        string
		expectedError string
	}{
		{"等待状态", "pending", ""},
		{"运行状态", "running", ""},
		{"成功状态", "success", ""},
		{"失败状态", "failure", ""},
		{"取消状态", "cancelled", ""},
		{"超时状态", "timeout", ""},
		{"空状态", "", "构建状态不能为空"},
		{"无效状态", "unknown", "无效的构建状态"},
		{"大写状态", "SUCCESS", "无效的构建状态"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBuildStatus(tt.status)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestStageTypeValidation 测试阶段类型验证
func TestStageTypeValidation(t *testing.T) {
	tests := []struct {
		name          string
		stageType     string
		expectedError string
	}{
		{"构建阶段", "build", ""},
		{"测试阶段", "test", ""},
		{"安全扫描阶段", "security_scan", ""},
		{"部署阶段", "deploy", ""},
		{"审批阶段", "approval", ""},
		{"通知阶段", "notification", ""},
		{"空阶段类型", "", "阶段类型不能为空"},
		{"无效阶段类型", "invalid", "无效的阶段类型"},
		{"大写阶段类型", "BUILD", "无效的阶段类型"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStageType(tt.stageType)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestEnvironmentTypeValidation 测试环境类型验证
func TestEnvironmentTypeValidation(t *testing.T) {
	tests := []struct {
		name          string
		envType       string
		expectedError string
	}{
		{"开发环境", "development", ""},
		{"测试环境", "testing", ""},
		{"预发布环境", "staging", ""},
		{"生产环境", "production", ""},
		{"空环境类型", "", "环境类型不能为空"},
		{"无效环境类型", "invalid", "无效的环境类型"},
		{"大写环境类型", "PRODUCTION", "无效的环境类型"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEnvironmentType(tt.envType)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestDeploymentStrategyValidation 测试部署策略验证
func TestDeploymentStrategyValidation(t *testing.T) {
	tests := []struct {
		name          string
		strategy      string
		expectedError string
	}{
		{"滚动部署", "rolling", ""},
		{"蓝绿部署", "blue_green", ""},
		{"金丝雀部署", "canary", ""},
		{"重建部署", "recreate", ""},
		{"空部署策略", "", "部署策略不能为空"},
		{"无效部署策略", "invalid", "无效的部署策略"},
		{"大写部署策略", "ROLLING", "无效的部署策略"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDeploymentStrategy(tt.strategy)

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
func TestCICDCommitSHAValidation(t *testing.T) {
	tests := []struct {
		name          string
		sha           string
		expectedError string
	}{
		{"有效的短SHA", "abc1234", ""},
		{"有效的完整SHA", "1234567890abcdef1234567890abcdef12345678", ""},
		{"有效的中等长度SHA", "1234567890abcdef", ""},
		{"空SHA", "", "提交SHA不能为空"},
		{"SHA过短", "abc12", "提交SHA长度必须在7-40字符之间"},
		{"SHA过长", strings.Repeat("a", 41), "提交SHA长度必须在7-40字符之间"},
		{"包含非十六进制字符", "abc123g", "提交SHA只能包含十六进制字符"},
		{"包含大写字母", "ABC1234", ""},
		{"包含特殊字符", "abc123!", "提交SHA只能包含十六进制字符"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cicdValidateCommitSHA(tt.sha)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestCreatePipelineRequestValidation 测试创建流水线请求验证
func TestCreatePipelineRequestValidation(t *testing.T) {
	validTenantID := uuid.New()
	validProjectID := uuid.New()
	validCreatedBy := uuid.New()

	tests := []struct {
		name          string
		req           *SimpleCreatePipelineRequest
		expectedError string
	}{
		{
			name: "有效的创建请求",
			req: &SimpleCreatePipelineRequest{
				TenantID:    validTenantID,
				ProjectID:   validProjectID,
				Name:        "Build Pipeline",
				Description: "主要构建流水线",
				Trigger:     "push",
				Branch:      "main",
				CreatedBy:   validCreatedBy,
			},
			expectedError: "",
		},
		{
			name: "租户ID为空",
			req: &SimpleCreatePipelineRequest{
				TenantID:  uuid.Nil,
				ProjectID: validProjectID,
				Name:      "Test Pipeline",
				Trigger:   "push",
				Branch:    "main",
				CreatedBy: validCreatedBy,
			},
			expectedError: "租户ID不能为空",
		},
		{
			name: "项目ID为空",
			req: &SimpleCreatePipelineRequest{
				TenantID:  validTenantID,
				ProjectID: uuid.Nil,
				Name:      "Test Pipeline",
				Trigger:   "push",
				Branch:    "main",
				CreatedBy: validCreatedBy,
			},
			expectedError: "项目ID不能为空",
		},
		{
			name: "创建者ID为空",
			req: &SimpleCreatePipelineRequest{
				TenantID:  validTenantID,
				ProjectID: validProjectID,
				Name:      "Test Pipeline",
				Trigger:   "push",
				Branch:    "main",
				CreatedBy: uuid.Nil,
			},
			expectedError: "创建者ID不能为空",
		},
		{
			name: "流水线名称无效",
			req: &SimpleCreatePipelineRequest{
				TenantID:  validTenantID,
				ProjectID: validProjectID,
				Name:      "pipe@line",
				Trigger:   "push",
				Branch:    "main",
				CreatedBy: validCreatedBy,
			},
			expectedError: "流水线名称: 流水线名称只能包含字母、数字、下划线、横线和空格",
		},
		{
			name: "触发类型无效",
			req: &SimpleCreatePipelineRequest{
				TenantID:  validTenantID,
				ProjectID: validProjectID,
				Name:      "Test Pipeline",
				Trigger:   "invalid",
				Branch:    "main",
				CreatedBy: validCreatedBy,
			},
			expectedError: "触发类型: 无效的触发类型",
		},
		{
			name: "分支名称无效",
			req: &SimpleCreatePipelineRequest{
				TenantID:  validTenantID,
				ProjectID: validProjectID,
				Name:      "Test Pipeline",
				Trigger:   "push",
				Branch:    "feature/",
				CreatedBy: validCreatedBy,
			},
			expectedError: "分支名称: 分支名称格式不正确",
		},
		{
			name:          "nil请求",
			req:           nil,
			expectedError: "创建流水线请求不能为nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCreatePipelineRequest(tt.req)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestCICDEdgeCases 测试CICD边界情况
func TestCICDEdgeCases(t *testing.T) {
	t.Run("极长的流水线描述", func(t *testing.T) {
		longDescription := strings.Repeat("很长的描述内容", 1000)
		// 流水线描述通常没有严格限制，但应该在合理范围内
		assert.Greater(t, len(longDescription), 1000)
	})

	t.Run("Unicode字符分支名", func(t *testing.T) {
		err := cicdValidateBranchName("feature/测试")
		// Git分支名不应包含非ASCII字符
		assert.Error(t, err)
	})

	t.Run("边界长度分支名", func(t *testing.T) {
		// 测试恰好255字符的分支名
		exactly255Branch := strings.Repeat("a", 255)
		err := cicdValidateBranchName(exactly255Branch)
		assert.NoError(t, err)

		// 测试256字符的分支名
		exactly256Branch := strings.Repeat("a", 256)
		err = cicdValidateBranchName(exactly256Branch)
		assert.Error(t, err)
	})

	t.Run("边界长度流水线名", func(t *testing.T) {
		// 测试恰好2字符的流水线名
		exactly2Name := "ab"
		err := validatePipelineName(exactly2Name)
		assert.NoError(t, err)

		// 测试恰好100字符的流水线名
		exactly100Name := strings.Repeat("a", 100)
		err = validatePipelineName(exactly100Name)
		assert.NoError(t, err)
	})
}

// TestCICDPerformance 测试CICD性能
func TestCICDPerformance(t *testing.T) {
	t.Run("批量流水线名验证", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 1000; i++ {
			name := fmt.Sprintf("pipeline_%d", i)
			err := validatePipelineName(name)
			assert.NoError(t, err)
		}

		duration := time.Since(start)
		t.Logf("批量验证1000个流水线名耗时: %v", duration)

		// 验证应该在10毫秒内完成
		assert.Less(t, duration, 10*time.Millisecond)
	})

	t.Run("批量分支名验证", func(t *testing.T) {
		start := time.Now()

		branches := []string{"main", "develop", "feature/auth", "hotfix/bug-123", "release/v1.0"}
		for i := 0; i < 1000; i++ {
			branch := branches[i%len(branches)]
			err := cicdValidateBranchName(branch)
			assert.NoError(t, err)
		}

		duration := time.Since(start)
		t.Logf("批量验证1000个分支名耗时: %v", duration)

		// 验证应该在15毫秒内完成
		assert.Less(t, duration, 15*time.Millisecond)
	})

	t.Run("批量提交SHA验证", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 1000; i++ {
			sha := fmt.Sprintf("abc123%d", i)
			if len(sha) < 7 {
				sha = sha + strings.Repeat("0", 7-len(sha))
			}
			err := cicdValidateCommitSHA(sha)
			assert.NoError(t, err)
		}

		duration := time.Since(start)
		t.Logf("批量验证1000个提交SHA耗时: %v", duration)

		// 验证应该在5毫秒内完成
		assert.Less(t, duration, 5*time.Millisecond)
	})
}

// MockCICDService CICD服务模拟实现
type MockCICDService struct {
	pipelines    map[uuid.UUID]*SimplePipeline
	builds       map[uuid.UUID]*SimpleBuild
	stages       map[uuid.UUID]*SimpleStage
	deployments  map[uuid.UUID]*SimpleDeployment
	environments map[uuid.UUID]*SimpleEnvironment
	buildCounter int
}

// NewMockCICDService 创建CICD服务模拟
func NewMockCICDService() *MockCICDService {
	return &MockCICDService{
		pipelines:    make(map[uuid.UUID]*SimplePipeline),
		builds:       make(map[uuid.UUID]*SimpleBuild),
		stages:       make(map[uuid.UUID]*SimpleStage),
		deployments:  make(map[uuid.UUID]*SimpleDeployment),
		environments: make(map[uuid.UUID]*SimpleEnvironment),
		buildCounter: 1,
	}
}

// CreatePipeline 模拟创建流水线
func (m *MockCICDService) CreatePipeline(req *SimpleCreatePipelineRequest) (*SimplePipeline, error) {
	if err := validateCreatePipelineRequest(req); err != nil {
		return nil, err
	}

	// 检查流水线名称是否已存在
	for _, pipeline := range m.pipelines {
		if pipeline.Name == req.Name && pipeline.ProjectID == req.ProjectID {
			return nil, fmt.Errorf("流水线名称 %s 在项目中已存在", req.Name)
		}
	}

	pipeline := &SimplePipeline{
		ID:          uuid.New(),
		TenantID:    req.TenantID,
		ProjectID:   req.ProjectID,
		Name:        req.Name,
		Description: req.Description,
		Config:      req.Config,
		Status:      "active",
		Trigger:     req.Trigger,
		Branch:      req.Branch,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	m.pipelines[pipeline.ID] = pipeline
	return pipeline, nil
}

// CreateBuild 模拟创建构建
func (m *MockCICDService) CreateBuild(pipelineID, projectID, createdBy uuid.UUID, branch, commitSHA, commitMsg string) (*SimpleBuild, error) {
	if pipelineID == uuid.Nil {
		return nil, fmt.Errorf("流水线ID不能为空")
	}

	if err := cicdValidateBranchName(branch); err != nil {
		return nil, fmt.Errorf("分支名称: %v", err)
	}

	if err := cicdValidateCommitSHA(commitSHA); err != nil {
		return nil, fmt.Errorf("提交SHA: %v", err)
	}

	// 检查流水线是否存在
	pipeline, exists := m.pipelines[pipelineID]
	if !exists {
		return nil, fmt.Errorf("流水线不存在")
	}

	if !pipeline.IsActive {
		return nil, fmt.Errorf("流水线未激活")
	}

	build := &SimpleBuild{
		ID:         uuid.New(),
		PipelineID: pipelineID,
		ProjectID:  projectID,
		Number:     m.buildCounter,
		Status:     "pending",
		Branch:     branch,
		CommitSHA:  commitSHA,
		CommitMsg:  commitMsg,
		Config:     make(map[string]interface{}),
		CreatedBy:  createdBy,
	}

	m.buildCounter++
	m.builds[build.ID] = build
	return build, nil
}

// CreateEnvironment 模拟创建环境
func (m *MockCICDService) CreateEnvironment(tenantID, projectID uuid.UUID, name, envType string, config map[string]interface{}) (*SimpleEnvironment, error) {
	if name == "" {
		return nil, fmt.Errorf("环境名称不能为空")
	}

	if err := validateEnvironmentType(envType); err != nil {
		return nil, err
	}

	// 检查环境名称是否已存在
	for _, env := range m.environments {
		if env.Name == name && env.ProjectID == projectID {
			return nil, fmt.Errorf("环境名称 %s 在项目中已存在", name)
		}
	}

	environment := &SimpleEnvironment{
		ID:          uuid.New(),
		TenantID:    tenantID,
		ProjectID:   projectID,
		Name:        name,
		Type:        envType,
		Status:      "active",
		Config:      config,
		Variables:   make(map[string]string),
		IsProtected: envType == "production", // 生产环境默认受保护
	}

	m.environments[environment.ID] = environment
	return environment, nil
}

// CreateDeployment 模拟创建部署
func (m *MockCICDService) CreateDeployment(pipelineID, buildID uuid.UUID, environment, strategy, version string) (*SimpleDeployment, error) {
	if environment == "" {
		return nil, fmt.Errorf("环境名称不能为空")
	}

	if err := validateDeploymentStrategy(strategy); err != nil {
		return nil, err
	}

	if version == "" {
		return nil, fmt.Errorf("版本号不能为空")
	}

	// 检查构建是否存在且成功
	build, exists := m.builds[buildID]
	if !exists {
		return nil, fmt.Errorf("构建不存在")
	}

	if build.Status != "success" {
		return nil, fmt.Errorf("只能部署成功的构建")
	}

	deployment := &SimpleDeployment{
		ID:            uuid.New(),
		PipelineID:    pipelineID,
		BuildID:       buildID,
		Environment:   environment,
		Strategy:      strategy,
		Status:        "pending",
		Config:        make(map[string]interface{}),
		Version:       version,
		RollbackCount: 0,
		CreatedAt:     time.Now(),
	}

	m.deployments[deployment.ID] = deployment
	return deployment, nil
}

// TestMockCICDService 测试CICD服务模拟
func TestMockCICDService(t *testing.T) {
	mockService := NewMockCICDService()

	t.Run("创建流水线成功", func(t *testing.T) {
		req := &SimpleCreatePipelineRequest{
			TenantID:    uuid.New(),
			ProjectID:   uuid.New(),
			Name:        "Build Pipeline",
			Description: "主要构建流水线",
			Config:      map[string]interface{}{"timeout": 300},
			Trigger:     "push",
			Branch:      "main",
			CreatedBy:   uuid.New(),
		}

		pipeline, err := mockService.CreatePipeline(req)
		require.NoError(t, err)
		require.NotNil(t, pipeline)

		assert.Equal(t, req.TenantID, pipeline.TenantID)
		assert.Equal(t, req.ProjectID, pipeline.ProjectID)
		assert.Equal(t, req.Name, pipeline.Name)
		assert.Equal(t, req.Description, pipeline.Description)
		assert.Equal(t, "active", pipeline.Status)
		assert.Equal(t, req.Trigger, pipeline.Trigger)
		assert.Equal(t, req.Branch, pipeline.Branch)
		assert.True(t, pipeline.IsActive)
	})

	t.Run("创建构建成功", func(t *testing.T) {
		// 先创建流水线
		pipelineReq := &SimpleCreatePipelineRequest{
			TenantID:  uuid.New(),
			ProjectID: uuid.New(),
			Name:      "Test Pipeline",
			Trigger:   "push",
			Branch:    "main",
			CreatedBy: uuid.New(),
		}
		pipeline, err := mockService.CreatePipeline(pipelineReq)
		require.NoError(t, err)

		// 创建构建
		build, err := mockService.CreateBuild(
			pipeline.ID,
			pipeline.ProjectID,
			uuid.New(),
			"feature/test",
			"abc1234567",
			"Fix bug in authentication",
		)

		require.NoError(t, err)
		require.NotNil(t, build)

		assert.Equal(t, pipeline.ID, build.PipelineID)
		assert.Equal(t, pipeline.ProjectID, build.ProjectID)
		assert.Equal(t, 1, build.Number)
		assert.Equal(t, "pending", build.Status)
		assert.Equal(t, "feature/test", build.Branch)
		assert.Equal(t, "abc1234567", build.CommitSHA)
	})

	t.Run("创建环境成功", func(t *testing.T) {
		environment, err := mockService.CreateEnvironment(
			uuid.New(),
			uuid.New(),
			"staging",
			"staging",
			map[string]interface{}{"replicas": 2},
		)

		require.NoError(t, err)
		require.NotNil(t, environment)

		assert.Equal(t, "staging", environment.Name)
		assert.Equal(t, "staging", environment.Type)
		assert.Equal(t, "active", environment.Status)
		assert.False(t, environment.IsProtected)
	})

	t.Run("创建生产环境 - 受保护", func(t *testing.T) {
		environment, err := mockService.CreateEnvironment(
			uuid.New(),
			uuid.New(),
			"production",
			"production",
			map[string]interface{}{"replicas": 5},
		)

		require.NoError(t, err)
		require.NotNil(t, environment)

		assert.Equal(t, "production", environment.Name)
		assert.Equal(t, "production", environment.Type)
		assert.True(t, environment.IsProtected) // 生产环境应该受保护
	})

	t.Run("创建流水线失败 - 名称重复", func(t *testing.T) {
		projectID := uuid.New()

		// 第一次创建成功
		req1 := &SimpleCreatePipelineRequest{
			TenantID:  uuid.New(),
			ProjectID: projectID,
			Name:      "Duplicate Pipeline",
			Trigger:   "push",
			Branch:    "main",
			CreatedBy: uuid.New(),
		}
		_, err := mockService.CreatePipeline(req1)
		require.NoError(t, err)

		// 第二次创建失败
		req2 := &SimpleCreatePipelineRequest{
			TenantID:  uuid.New(),
			ProjectID: projectID,            // 同一项目
			Name:      "Duplicate Pipeline", // 同名
			Trigger:   "push",
			Branch:    "develop",
			CreatedBy: uuid.New(),
		}
		pipeline, err := mockService.CreatePipeline(req2)

		assert.Error(t, err)
		assert.Nil(t, pipeline)
		assert.Contains(t, err.Error(), "流水线名称")
		assert.Contains(t, err.Error(), "已存在")
	})

	t.Run("创建构建失败 - 流水线不存在", func(t *testing.T) {
		build, err := mockService.CreateBuild(
			uuid.New(), // 不存在的流水线ID
			uuid.New(),
			uuid.New(),
			"main",
			"abc1234567",
			"Test commit",
		)

		assert.Error(t, err)
		assert.Nil(t, build)
		assert.Contains(t, err.Error(), "流水线不存在")
	})

	t.Run("创建部署失败 - 构建不成功", func(t *testing.T) {
		// 创建流水线和构建
		pipelineReq := &SimpleCreatePipelineRequest{
			TenantID:  uuid.New(),
			ProjectID: uuid.New(),
			Name:      "Deploy Pipeline",
			Trigger:   "push",
			Branch:    "main",
			CreatedBy: uuid.New(),
		}
		pipeline, err := mockService.CreatePipeline(pipelineReq)
		require.NoError(t, err)

		build, err := mockService.CreateBuild(
			pipeline.ID,
			pipeline.ProjectID,
			uuid.New(),
			"main",
			"abc1234567",
			"Test commit",
		)
		require.NoError(t, err)

		// 构建状态为pending，尝试部署
		deployment, err := mockService.CreateDeployment(
			pipeline.ID,
			build.ID,
			"production",
			"rolling",
			"v1.0.0",
		)

		assert.Error(t, err)
		assert.Nil(t, deployment)
		assert.Contains(t, err.Error(), "只能部署成功的构建")
	})
}
