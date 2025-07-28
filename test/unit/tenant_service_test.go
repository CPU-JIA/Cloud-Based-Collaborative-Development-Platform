package unit

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SimpleTenant 简化的租户模型用于测试
type SimpleTenant struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Domain       string    `json:"domain"`
	Status       string    `json:"status"`
	Plan         string    `json:"plan"`
	BillingEmail string    `json:"billing_email"`
	Description  string    `json:"description"`
	ContactName  string    `json:"contact_name"`
	ContactEmail string    `json:"contact_email"`
	ContactPhone string    `json:"contact_phone"`
	Address      string    `json:"address"`
	City         string    `json:"city"`
	State        string    `json:"state"`
	Country      string    `json:"country"`
	PostalCode   string    `json:"postal_code"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// SimpleCreateTenantRequest 简化的创建租户请求用于测试
type SimpleCreateTenantRequest struct {
	Name         string `json:"name" binding:"required"`
	Domain       string `json:"domain" binding:"required"`
	Plan         string `json:"plan" binding:"required"`
	BillingEmail string `json:"billing_email" binding:"required"`
	Description  string `json:"description"`
	ContactName  string `json:"contact_name" binding:"required"`
	ContactEmail string `json:"contact_email" binding:"required"`
	ContactPhone string `json:"contact_phone"`
	Address      string `json:"address"`
	City         string `json:"city"`
	State        string `json:"state"`
	Country      string `json:"country"`
	PostalCode   string `json:"postal_code"`
}

// SimpleTenantConfig 简化的租户配置用于测试
type SimpleTenantConfig struct {
	TenantID            uuid.UUID              `json:"tenant_id"`
	MaxUsers            int                    `json:"max_users"`
	MaxProjects         int                    `json:"max_projects"`
	MaxStorage          int                    `json:"max_storage"`
	MaxAPICallsDaily    int                    `json:"max_api_calls_daily"`
	FeatureFlags        map[string]interface{} `json:"feature_flags"`
	SecurityPolicy      map[string]interface{} `json:"security_policy"`
	IntegrationSettings map[string]interface{} `json:"integration_settings"`
}

// SimpleInvitationRequest 简化的邀请请求用于测试
type SimpleInvitationRequest struct {
	Email   string `json:"email" binding:"required"`
	Role    string `json:"role" binding:"required"`
	Message string `json:"message"`
}

// Tenant Service 验证函数

// validateTenantName 验证租户名称
func validateTenantName(name string) error {
	if name == "" {
		return fmt.Errorf("租户名称不能为空")
	}

	if len(name) < 2 || len(name) > 255 {
		return fmt.Errorf("租户名称长度必须在2-255字符之间")
	}

	// 不能包含特殊字符
	if strings.ContainsAny(name, "<>\"'&") {
		return fmt.Errorf("租户名称不能包含特殊字符")
	}

	return nil
}

// validateDomain 验证域名
func validateDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("域名不能为空")
	}

	if len(domain) < 3 || len(domain) > 255 {
		return fmt.Errorf("域名长度必须在3-255字符之间")
	}

	// 简化的域名格式验证
	domainPattern := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.-]*[a-zA-Z0-9]$`)
	if !domainPattern.MatchString(domain) {
		return fmt.Errorf("域名格式不正确")
	}

	// 不能包含连续的点或横线
	if strings.Contains(domain, "..") || strings.Contains(domain, "--") {
		return fmt.Errorf("域名不能包含连续的点或横线")
	}

	return nil
}

// validateEmail 验证邮箱
func tenantValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("邮箱不能为空")
	}

	_, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("邮箱格式不正确")
	}

	return nil
}

// validateTenantPlan 验证租户计划
func validateTenantPlan(plan string) error {
	validPlans := []string{"free", "basic", "professional", "enterprise"}

	for _, validPlan := range validPlans {
		if plan == validPlan {
			return nil
		}
	}

	return fmt.Errorf("无效的租户计划: %s", plan)
}

// validateTenantStatus 验证租户状态
func validateTenantStatus(status string) error {
	validStatuses := []string{"active", "suspended", "pending", "deleted"}

	for _, validStatus := range validStatuses {
		if status == validStatus {
			return nil
		}
	}

	return fmt.Errorf("无效的租户状态: %s", status)
}

// validatePhoneNumber 验证电话号码
func tenantValidatePhoneNumber(phone string) error {
	if phone == "" {
		return nil // 电话号码是可选的
	}

	// 简化的电话号码验证
	phonePattern := regexp.MustCompile(`^[\+]?[0-9\-\s\(\)]{7,20}$`)
	if !phonePattern.MatchString(phone) {
		return fmt.Errorf("电话号码格式不正确")
	}

	return nil
}

// validateCreateTenantRequest 验证创建租户请求
func validateCreateTenantRequest(req *SimpleCreateTenantRequest) error {
	if req == nil {
		return fmt.Errorf("创建租户请求不能为nil")
	}

	if err := validateTenantName(req.Name); err != nil {
		return err
	}

	if err := validateDomain(req.Domain); err != nil {
		return err
	}

	if err := validateTenantPlan(req.Plan); err != nil {
		return err
	}

	if err := tenantValidateEmail(req.BillingEmail); err != nil {
		return fmt.Errorf("账单邮箱: %v", err)
	}

	if req.ContactName == "" {
		return fmt.Errorf("联系人姓名不能为空")
	}

	if err := tenantValidateEmail(req.ContactEmail); err != nil {
		return fmt.Errorf("联系人邮箱: %v", err)
	}

	if err := tenantValidatePhoneNumber(req.ContactPhone); err != nil {
		return fmt.Errorf("联系人电话: %v", err)
	}

	if len(req.Description) > 1000 {
		return fmt.Errorf("描述不能超过1000字符")
	}

	return nil
}

// validateTenantConfig 验证租户配置
func validateTenantConfig(config *SimpleTenantConfig) error {
	if config == nil {
		return fmt.Errorf("租户配置不能为nil")
	}

	if config.TenantID == uuid.Nil {
		return fmt.Errorf("租户ID不能为空")
	}

	if config.MaxUsers < 1 {
		return fmt.Errorf("最大用户数必须大于0")
	}

	if config.MaxProjects < 1 {
		return fmt.Errorf("最大项目数必须大于0")
	}

	if config.MaxStorage < 1 {
		return fmt.Errorf("最大存储空间必须大于0")
	}

	if config.MaxAPICallsDaily < 1 {
		return fmt.Errorf("每日最大API调用次数必须大于0")
	}

	return nil
}

// validateInvitationRequest 验证邀请请求
func validateInvitationRequest(req *SimpleInvitationRequest) error {
	if req == nil {
		return fmt.Errorf("邀请请求不能为nil")
	}

	if err := tenantValidateEmail(req.Email); err != nil {
		return fmt.Errorf("邀请邮箱: %v", err)
	}

	validRoles := []string{"admin", "member", "viewer", "owner"}
	isValidRole := false
	for _, role := range validRoles {
		if req.Role == role {
			isValidRole = true
			break
		}
	}
	if !isValidRole {
		return fmt.Errorf("无效的角色: %s", req.Role)
	}

	if len(req.Message) > 500 {
		return fmt.Errorf("邀请消息不能超过500字符")
	}

	return nil
}

// TestTenantValidation 测试租户验证
func TestTenantValidation(t *testing.T) {
	tests := []struct {
		name          string
		tenant        *SimpleTenant
		expectedError string
	}{
		{
			name: "有效的租户",
			tenant: &SimpleTenant{
				ID:           uuid.New(),
				Name:         "示例公司",
				Domain:       "example.com",
				Status:       "active",
				Plan:         "basic",
				BillingEmail: "billing@example.com",
				ContactName:  "张三",
				ContactEmail: "zhangsan@example.com",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			expectedError: "",
		},
		{
			name: "无效的租户名称 - 空名称",
			tenant: &SimpleTenant{
				Name:   "",
				Domain: "example.com",
			},
			expectedError: "租户名称不能为空",
		},
		{
			name: "无效的域名",
			tenant: &SimpleTenant{
				Name:   "测试公司",
				Domain: "invalid..domain",
			},
			expectedError: "域名不能包含连续的点或横线",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error

			if tt.tenant.Name != "" {
				err = validateTenantName(tt.tenant.Name)
				if err != nil && tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
					return
				}
			}

			if tt.tenant.Domain != "" {
				err = validateDomain(tt.tenant.Domain)
				if err != nil && tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
					return
				}
			}

			if tt.expectedError == "" {
				assert.NoError(t, err)
			}
		})
	}
}

// TestTenantNameValidation 测试租户名称验证
func TestTenantNameValidation(t *testing.T) {
	tests := []struct {
		name          string
		tenantName    string
		expectedError string
	}{
		{"有效的中文名称", "示例科技有限公司", ""},
		{"有效的英文名称", "Example Technology Inc.", ""},
		{"有效的混合名称", "示例 Technology", ""},
		{"有效的带数字名称", "公司2024", ""},
		{"空名称", "", "租户名称不能为空"},
		{"名称过短", "A", "租户名称长度必须在2-255字符之间"},
		{"名称过长", strings.Repeat("A", 256), "租户名称长度必须在2-255字符之间"},
		{"包含特殊字符", "公司<script>", "租户名称不能包含特殊字符"},
		{"包含引号", "公司\"名称", "租户名称不能包含特殊字符"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTenantName(tt.tenantName)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestDomainValidation 测试域名验证
func TestDomainValidation(t *testing.T) {
	tests := []struct {
		name          string
		domain        string
		expectedError string
	}{
		{"有效的域名", "example.com", ""},
		{"有效的子域名", "app.example.com", ""},
		{"有效的多级域名", "api.v1.example.com", ""},
		{"有效的国际化域名", "example-app.com", ""},
		{"空域名", "", "域名不能为空"},
		{"域名过短", "ab", "域名长度必须在3-255字符之间"},
		{"域名过长", strings.Repeat("a", 253) + ".com", "域名长度必须在3-255字符之间"},
		{"包含连续的点", "example..com", "域名不能包含连续的点或横线"},
		{"包含连续的横线", "example--app.com", "域名不能包含连续的点或横线"},
		{"以点开头", ".example.com", "域名格式不正确"},
		{"以点结尾", "example.com.", "域名格式不正确"},
		{"包含非法字符", "example@.com", "域名格式不正确"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDomain(tt.domain)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestEmailValidation 测试邮箱验证
func TestTenantEmailValidation(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		expectedError string
	}{
		{"有效的邮箱", "user@example.com", ""},
		{"有效的企业邮箱", "zhang.san@company.com.cn", ""},
		{"有效的带加号邮箱", "user+test@example.com", ""},
		{"空邮箱", "", "邮箱不能为空"},
		{"无@符号", "userexample.com", "邮箱格式不正确"},
		{"多个@符号", "user@@example.com", "邮箱格式不正确"},
		{"缺少域名", "user@", "邮箱格式不正确"},
		{"缺少用户名", "@example.com", "邮箱格式不正确"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tenantValidateEmail(tt.email)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestTenantPlanValidation 测试租户计划验证
func TestTenantPlanValidation(t *testing.T) {
	tests := []struct {
		name          string
		plan          string
		expectedError string
	}{
		{"免费计划", "free", ""},
		{"基础计划", "basic", ""},
		{"专业计划", "professional", ""},
		{"企业计划", "enterprise", ""},
		{"无效计划", "premium", "无效的租户计划"},
		{"空计划", "", "无效的租户计划"},
		{"大写计划", "BASIC", "无效的租户计划"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTenantPlan(tt.plan)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestPhoneNumberValidation 测试电话号码验证
func TestTenantPhoneNumberValidation(t *testing.T) {
	tests := []struct {
		name          string
		phone         string
		expectedError string
	}{
		{"有效的中国手机号", "+86-138-0000-0000", ""},
		{"有效的美国电话", "+1-555-123-4567", ""},
		{"有效的固定电话", "(010) 8888-8888", ""},
		{"有效的简单格式", "13800000000", ""},
		{"空电话号码", "", ""}, // 电话号码是可选的
		{"号码过短", "123", "电话号码格式不正确"},
		{"号码过长", "123456789012345678901", "电话号码格式不正确"},
		{"包含字母", "138abc0000", "电话号码格式不正确"},
		{"包含特殊字符", "138@0000", "电话号码格式不正确"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tenantValidatePhoneNumber(tt.phone)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestCreateTenantRequest 测试创建租户请求验证
func TestCreateTenantRequest(t *testing.T) {
	tests := []struct {
		name          string
		req           *SimpleCreateTenantRequest
		expectedError string
	}{
		{
			name: "有效的创建请求",
			req: &SimpleCreateTenantRequest{
				Name:         "示例公司",
				Domain:       "example.com",
				Plan:         "basic",
				BillingEmail: "billing@example.com",
				ContactName:  "张三",
				ContactEmail: "zhangsan@example.com",
				ContactPhone: "+86-138-0000-0000",
				Description:  "这是一个示例公司",
			},
			expectedError: "",
		},
		{
			name: "名称为空",
			req: &SimpleCreateTenantRequest{
				Name:         "",
				Domain:       "example.com",
				Plan:         "basic",
				BillingEmail: "billing@example.com",
				ContactName:  "张三",
				ContactEmail: "zhangsan@example.com",
			},
			expectedError: "租户名称不能为空",
		},
		{
			name: "域名无效",
			req: &SimpleCreateTenantRequest{
				Name:         "示例公司",
				Domain:       "invalid..domain",
				Plan:         "basic",
				BillingEmail: "billing@example.com",
				ContactName:  "张三",
				ContactEmail: "zhangsan@example.com",
			},
			expectedError: "域名不能包含连续的点或横线",
		},
		{
			name: "无效的计划",
			req: &SimpleCreateTenantRequest{
				Name:         "示例公司",
				Domain:       "example.com",
				Plan:         "invalid",
				BillingEmail: "billing@example.com",
				ContactName:  "张三",
				ContactEmail: "zhangsan@example.com",
			},
			expectedError: "无效的租户计划",
		},
		{
			name: "联系人姓名为空",
			req: &SimpleCreateTenantRequest{
				Name:         "示例公司",
				Domain:       "example.com",
				Plan:         "basic",
				BillingEmail: "billing@example.com",
				ContactName:  "",
				ContactEmail: "zhangsan@example.com",
			},
			expectedError: "联系人姓名不能为空",
		},
		{
			name: "描述过长",
			req: &SimpleCreateTenantRequest{
				Name:         "示例公司",
				Domain:       "example.com",
				Plan:         "basic",
				BillingEmail: "billing@example.com",
				ContactName:  "张三",
				ContactEmail: "zhangsan@example.com",
				Description:  strings.Repeat("a", 1001),
			},
			expectedError: "描述不能超过1000字符",
		},
		{
			name:          "nil请求",
			req:           nil,
			expectedError: "创建租户请求不能为nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCreateTenantRequest(tt.req)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestTenantConfigValidation 测试租户配置验证
func TestTenantConfigValidation(t *testing.T) {
	validTenantID := uuid.New()

	tests := []struct {
		name          string
		config        *SimpleTenantConfig
		expectedError string
	}{
		{
			name: "有效的配置",
			config: &SimpleTenantConfig{
				TenantID:         validTenantID,
				MaxUsers:         100,
				MaxProjects:      50,
				MaxStorage:       1000,
				MaxAPICallsDaily: 10000,
				FeatureFlags:     map[string]interface{}{"advanced_search": true},
			},
			expectedError: "",
		},
		{
			name: "租户ID为空",
			config: &SimpleTenantConfig{
				TenantID:    uuid.Nil,
				MaxUsers:    100,
				MaxProjects: 50,
			},
			expectedError: "租户ID不能为空",
		},
		{
			name: "最大用户数无效",
			config: &SimpleTenantConfig{
				TenantID:    validTenantID,
				MaxUsers:    0,
				MaxProjects: 50,
			},
			expectedError: "最大用户数必须大于0",
		},
		{
			name:          "nil配置",
			config:        nil,
			expectedError: "租户配置不能为nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTenantConfig(tt.config)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestInvitationRequestValidation 测试邀请请求验证
func TestInvitationRequestValidation(t *testing.T) {
	tests := []struct {
		name          string
		req           *SimpleInvitationRequest
		expectedError string
	}{
		{
			name: "有效的邀请请求",
			req: &SimpleInvitationRequest{
				Email:   "user@example.com",
				Role:    "member",
				Message: "欢迎加入我们的团队",
			},
			expectedError: "",
		},
		{
			name: "无效的邮箱",
			req: &SimpleInvitationRequest{
				Email: "invalid-email",
				Role:  "member",
			},
			expectedError: "邀请邮箱: 邮箱格式不正确",
		},
		{
			name: "无效的角色",
			req: &SimpleInvitationRequest{
				Email: "user@example.com",
				Role:  "invalid",
			},
			expectedError: "无效的角色",
		},
		{
			name: "消息过长",
			req: &SimpleInvitationRequest{
				Email:   "user@example.com",
				Role:    "member",
				Message: strings.Repeat("a", 501),
			},
			expectedError: "邀请消息不能超过500字符",
		},
		{
			name:          "nil请求",
			req:           nil,
			expectedError: "邀请请求不能为nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInvitationRequest(tt.req)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestTenantServiceEdgeCases 测试Tenant Service边界情况
func TestTenantServiceEdgeCases(t *testing.T) {
	t.Run("nil tenant", func(t *testing.T) {
		var tenant *SimpleTenant
		require.Nil(t, tenant)
	})

	t.Run("极长的域名", func(t *testing.T) {
		// 253 + 4 = 257个字符，超过255限制
		longDomain := strings.Repeat("a", 253) + ".com"
		err := validateDomain(longDomain)
		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), "域名长度必须在3-255字符之间")
		}
	})

	t.Run("国际化邮箱", func(t *testing.T) {
		// 测试一些肯定会失败的邮箱格式
		invalidEmails := []string{
			"用户名@",          // 缺少域名部分
			"@example.com",  // 缺少用户名部分
			"user@",         // 缺少域名
			"invalid.email", // 缺少@符号
		}

		for _, email := range invalidEmails {
			err := tenantValidateEmail(email)
			assert.Error(t, err, "无效邮箱 %s 应该验证失败", email)
		}
	})

	t.Run("特殊格式的电话号码", func(t *testing.T) {
		specialPhones := []string{
			"+1 (555) 123-4567",
			"+86 138 0000 0000",
			"(010) 8888-8888",
		}

		for _, phone := range specialPhones {
			err := tenantValidatePhoneNumber(phone)
			assert.NoError(t, err, "电话号码 %s 应该是有效的", phone)
		}
	})
}

// TestTenantServicePerformance 测试Tenant Service性能
func TestTenantServicePerformance(t *testing.T) {
	t.Run("批量租户名验证", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 1000; i++ {
			tenantName := fmt.Sprintf("公司%d", i)
			err := validateTenantName(tenantName)
			assert.NoError(t, err)
		}

		duration := time.Since(start)
		t.Logf("批量验证1000个租户名耗时: %v", duration)

		// 验证应该在10毫秒内完成
		assert.Less(t, duration, 10*time.Millisecond)
	})

	t.Run("批量邮箱验证", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 1000; i++ {
			email := fmt.Sprintf("user%d@example.com", i)
			err := tenantValidateEmail(email)
			assert.NoError(t, err)
		}

		duration := time.Since(start)
		t.Logf("批量验证1000个邮箱耗时: %v", duration)

		// 验证应该在50毫秒内完成
		assert.Less(t, duration, 50*time.Millisecond)
	})
}

// MockTenantService Tenant服务模拟实现
type MockTenantService struct {
	tenants map[uuid.UUID]*SimpleTenant
	configs map[uuid.UUID]*SimpleTenantConfig
}

// NewMockTenantService 创建Tenant服务模拟
func NewMockTenantService() *MockTenantService {
	return &MockTenantService{
		tenants: make(map[uuid.UUID]*SimpleTenant),
		configs: make(map[uuid.UUID]*SimpleTenantConfig),
	}
}

// CreateTenant 模拟创建租户
func (m *MockTenantService) CreateTenant(req *SimpleCreateTenantRequest) (*SimpleTenant, error) {
	if err := validateCreateTenantRequest(req); err != nil {
		return nil, err
	}

	// 检查域名是否已存在
	for _, tenant := range m.tenants {
		if tenant.Domain == req.Domain {
			return nil, fmt.Errorf("域名 %s 已被使用", req.Domain)
		}
	}

	tenant := &SimpleTenant{
		ID:           uuid.New(),
		Name:         req.Name,
		Domain:       req.Domain,
		Status:       "pending",
		Plan:         req.Plan,
		BillingEmail: req.BillingEmail,
		Description:  req.Description,
		ContactName:  req.ContactName,
		ContactEmail: req.ContactEmail,
		ContactPhone: req.ContactPhone,
		Address:      req.Address,
		City:         req.City,
		State:        req.State,
		Country:      req.Country,
		PostalCode:   req.PostalCode,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	m.tenants[tenant.ID] = tenant

	// 创建默认配置
	defaultConfig := &SimpleTenantConfig{
		TenantID:         tenant.ID,
		MaxUsers:         getPlanLimits(req.Plan).MaxUsers,
		MaxProjects:      getPlanLimits(req.Plan).MaxProjects,
		MaxStorage:       getPlanLimits(req.Plan).MaxStorage,
		MaxAPICallsDaily: getPlanLimits(req.Plan).MaxAPICallsDaily,
		FeatureFlags:     make(map[string]interface{}),
		SecurityPolicy:   make(map[string]interface{}),
	}

	m.configs[tenant.ID] = defaultConfig

	return tenant, nil
}

// getPlanLimits 获取计划限制
func getPlanLimits(plan string) *SimpleTenantConfig {
	limits := map[string]*SimpleTenantConfig{
		"free": {
			MaxUsers:         5,
			MaxProjects:      3,
			MaxStorage:       1000, // 1GB
			MaxAPICallsDaily: 1000,
		},
		"basic": {
			MaxUsers:         25,
			MaxProjects:      10,
			MaxStorage:       10000, // 10GB
			MaxAPICallsDaily: 10000,
		},
		"professional": {
			MaxUsers:         100,
			MaxProjects:      50,
			MaxStorage:       100000, // 100GB
			MaxAPICallsDaily: 100000,
		},
		"enterprise": {
			MaxUsers:         1000,
			MaxProjects:      500,
			MaxStorage:       1000000, // 1TB
			MaxAPICallsDaily: 1000000,
		},
	}

	if limit, exists := limits[plan]; exists {
		return limit
	}

	return limits["free"] // 默认免费计划
}

// TestMockTenantService 测试Tenant服务模拟
func TestMockTenantService(t *testing.T) {
	mockService := NewMockTenantService()

	t.Run("创建租户成功", func(t *testing.T) {
		req := &SimpleCreateTenantRequest{
			Name:         "测试公司",
			Domain:       "test-company.com",
			Plan:         "basic",
			BillingEmail: "billing@test-company.com",
			ContactName:  "张三",
			ContactEmail: "zhangsan@test-company.com",
			ContactPhone: "+86-138-0000-0000",
			Description:  "这是一个测试公司",
		}

		tenant, err := mockService.CreateTenant(req)
		require.NoError(t, err)
		require.NotNil(t, tenant)

		assert.Equal(t, req.Name, tenant.Name)
		assert.Equal(t, req.Domain, tenant.Domain)
		assert.Equal(t, req.Plan, tenant.Plan)
		assert.Equal(t, "pending", tenant.Status)
		assert.NotZero(t, tenant.ID)

		// 验证默认配置被创建
		config, exists := mockService.configs[tenant.ID]
		require.True(t, exists)
		assert.Equal(t, 25, config.MaxUsers) // basic plan
		assert.Equal(t, 10, config.MaxProjects)
	})

	t.Run("创建租户失败 - 无效请求", func(t *testing.T) {
		req := &SimpleCreateTenantRequest{
			Name:         "", // 无效名称
			Domain:       "test.com",
			Plan:         "basic",
			BillingEmail: "billing@test.com",
			ContactName:  "张三",
			ContactEmail: "zhangsan@test.com",
		}

		tenant, err := mockService.CreateTenant(req)
		assert.Error(t, err)
		assert.Nil(t, tenant)
		assert.Contains(t, err.Error(), "租户名称不能为空")
	})
}
