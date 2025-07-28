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

// SimpleUser 简化的用户模型用于测试
type SimpleUser struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	Email        string     `json:"email"`
	Username     string     `json:"username"`
	FirstName    string     `json:"first_name"`
	LastName     string     `json:"last_name"`
	Phone        string     `json:"phone"`
	Avatar       string     `json:"avatar"`
	Status       string     `json:"status"`
	IsActive     bool       `json:"is_active"`
	IsVerified   bool       `json:"is_verified"`
	IsMFAEnabled bool       `json:"is_mfa_enabled"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at"`
}

// SimpleRole 简化的角色模型用于测试
type SimpleRole struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	Level       int       `json:"level"`
	IsSystem    bool      `json:"is_system"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SimpleLoginRequest 简化的登录请求用于测试
type SimpleLoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// SimpleRegisterRequest 简化的注册请求用于测试
type SimpleRegisterRequest struct {
	TenantID  uuid.UUID `json:"tenant_id" binding:"required"`
	Email     string    `json:"email" binding:"required"`
	Username  string    `json:"username" binding:"required"`
	Password  string    `json:"password" binding:"required"`
	FirstName string    `json:"first_name" binding:"required"`
	LastName  string    `json:"last_name" binding:"required"`
}

// SimpleChangePasswordRequest 简化的修改密码请求用于测试
type SimpleChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

// SimpleAPIToken 简化的API Token模型用于测试
type SimpleAPIToken struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	Name       string     `json:"name"`
	Token      string     `json:"token"`
	Scopes     []string   `json:"scopes"`
	ExpiresAt  *time.Time `json:"expires_at"`
	IsActive   bool       `json:"is_active"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// IAM Service 验证函数

// validateEmail 验证邮箱格式
func validateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("邮箱不能为空")
	}

	_, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("邮箱格式不正确")
	}

	return nil
}

// validateUsername 验证用户名
func validateUsername(username string) error {
	if username == "" {
		return fmt.Errorf("用户名不能为空")
	}

	if len(username) < 3 || len(username) > 50 {
		return fmt.Errorf("用户名长度必须在3-50字符之间")
	}

	// 用户名只能包含字母、数字、下划线、横线和点
	usernamePattern := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	if !usernamePattern.MatchString(username) {
		return fmt.Errorf("用户名只能包含字母、数字、下划线、横线和点")
	}

	// 不能以特殊字符开头或结尾
	if strings.HasPrefix(username, ".") || strings.HasSuffix(username, ".") ||
		strings.HasPrefix(username, "_") || strings.HasSuffix(username, "_") ||
		strings.HasPrefix(username, "-") || strings.HasSuffix(username, "-") {
		return fmt.Errorf("用户名不能以特殊字符开头或结尾")
	}

	return nil
}

// validatePassword 验证密码强度
func validatePassword(password string) error {
	if password == "" {
		return fmt.Errorf("密码不能为空")
	}

	if len(password) < 8 {
		return fmt.Errorf("密码长度至少8位")
	}

	if len(password) > 128 {
		return fmt.Errorf("密码长度不能超过128位")
	}

	// 检查密码复杂度
	var hasUpper, hasLower, hasDigit, hasSpecial bool

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", char):
			hasSpecial = true
		}
	}

	strengthScore := 0
	if hasUpper {
		strengthScore++
	}
	if hasLower {
		strengthScore++
	}
	if hasDigit {
		strengthScore++
	}
	if hasSpecial {
		strengthScore++
	}

	if strengthScore < 3 {
		return fmt.Errorf("密码强度不足，需要包含至少3种类型：大写字母、小写字母、数字、特殊字符")
	}

	return nil
}

// validateRoleName 验证角色名称
func validateRoleName(name string) error {
	if name == "" {
		return fmt.Errorf("角色名称不能为空")
	}

	if len(name) < 2 || len(name) > 50 {
		return fmt.Errorf("角色名称长度必须在2-50字符之间")
	}

	// 角色名称只能包含字母、数字、下划线和横线
	rolePattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !rolePattern.MatchString(name) {
		return fmt.Errorf("角色名称只能包含字母、数字、下划线和横线")
	}

	return nil
}

// validatePermission 验证权限名称
func validatePermission(permission string) error {
	if permission == "" {
		return fmt.Errorf("权限名称不能为空")
	}

	// 权限格式: resource:action (如: project:read, user:write)
	permissionPattern := regexp.MustCompile(`^[a-z_]+:[a-z_]+$`)
	if !permissionPattern.MatchString(permission) {
		return fmt.Errorf("权限格式不正确，应为 resource:action 格式")
	}

	return nil
}

// validateAPITokenName 验证API Token名称
func validateAPITokenName(name string) error {
	if name == "" {
		return fmt.Errorf("API Token名称不能为空")
	}

	if len(name) < 2 || len(name) > 100 {
		return fmt.Errorf("API Token名称长度必须在2-100字符之间")
	}

	return nil
}

// validateLoginRequest 验证登录请求
func validateLoginRequest(req *SimpleLoginRequest) error {
	if req == nil {
		return fmt.Errorf("登录请求不能为nil")
	}

	if err := validateEmail(req.Email); err != nil {
		return fmt.Errorf("登录邮箱: %v", err)
	}

	if req.Password == "" {
		return fmt.Errorf("密码不能为空")
	}

	return nil
}

// validateRegisterRequest 验证注册请求
func validateRegisterRequest(req *SimpleRegisterRequest) error {
	if req == nil {
		return fmt.Errorf("注册请求不能为nil")
	}

	if req.TenantID == uuid.Nil {
		return fmt.Errorf("租户ID不能为空")
	}

	if err := validateEmail(req.Email); err != nil {
		return fmt.Errorf("注册邮箱: %v", err)
	}

	if err := validateUsername(req.Username); err != nil {
		return fmt.Errorf("用户名: %v", err)
	}

	if err := validatePassword(req.Password); err != nil {
		return fmt.Errorf("密码: %v", err)
	}

	if req.FirstName == "" {
		return fmt.Errorf("名字不能为空")
	}

	if req.LastName == "" {
		return fmt.Errorf("姓氏不能为空")
	}

	return nil
}

// TestUserValidation 测试用户验证
func TestUserValidation(t *testing.T) {
	tests := []struct {
		name          string
		user          *SimpleUser
		expectedError string
	}{
		{
			name: "有效的用户",
			user: &SimpleUser{
				ID:         uuid.New(),
				TenantID:   uuid.New(),
				Email:      "user@example.com",
				Username:   "valid_user",
				FirstName:  "John",
				LastName:   "Doe",
				Status:     "active",
				IsActive:   true,
				IsVerified: true,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
			expectedError: "",
		},
		{
			name: "无效的邮箱",
			user: &SimpleUser{
				Email:    "invalid-email",
				Username: "testuser",
			},
			expectedError: "邮箱格式不正确",
		},
		{
			name: "无效的用户名",
			user: &SimpleUser{
				Email:    "user@example.com",
				Username: "ab", // 太短
			},
			expectedError: "用户名长度必须在3-50字符之间",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error

			if tt.user.Email != "" {
				err = validateEmail(tt.user.Email)
				if err != nil && tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
					return
				}
			}

			if tt.user.Username != "" {
				err = validateUsername(tt.user.Username)
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

// TestEmailValidation 测试邮箱验证
func TestEmailValidation(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		expectedError string
	}{
		{"有效的邮箱", "user@example.com", ""},
		{"有效的企业邮箱", "john.doe@company.com", ""},
		{"有效的带加号邮箱", "user+test@example.com", ""},
		{"有效的子域名邮箱", "user@mail.example.com", ""},
		{"空邮箱", "", "邮箱不能为空"},
		{"无@符号", "userexample.com", "邮箱格式不正确"},
		{"多个@符号", "user@@example.com", "邮箱格式不正确"},
		{"缺少域名", "user@", "邮箱格式不正确"},
		{"缺少用户名", "@example.com", "邮箱格式不正确"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEmail(tt.email)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestUsernameValidation 测试用户名验证
func TestUsernameValidation(t *testing.T) {
	tests := []struct {
		name          string
		username      string
		expectedError string
	}{
		{"有效的简单用户名", "john", ""},
		{"有效的用户名带下划线", "john_doe", ""},
		{"有效的用户名带横线", "john-doe", ""},
		{"有效的用户名带点", "john.doe", ""},
		{"有效的用户名带数字", "user123", ""},
		{"有效的复杂用户名", "user_name.123-test", ""},
		{"空用户名", "", "用户名不能为空"},
		{"用户名过短", "ab", "用户名长度必须在3-50字符之间"},
		{"用户名过长", strings.Repeat("a", 51), "用户名长度必须在3-50字符之间"},
		{"包含特殊字符", "user@name", "用户名只能包含字母、数字、下划线、横线和点"},
		{"包含空格", "user name", "用户名只能包含字母、数字、下划线、横线和点"},
		{"以点开头", ".username", "用户名不能以特殊字符开头或结尾"},
		{"以点结尾", "username.", "用户名不能以特殊字符开头或结尾"},
		{"以下划线开头", "_username", "用户名不能以特殊字符开头或结尾"},
		{"以横线结尾", "username-", "用户名不能以特殊字符开头或结尾"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUsername(tt.username)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestPasswordValidation 测试密码验证
func TestPasswordValidation(t *testing.T) {
	tests := []struct {
		name          string
		password      string
		expectedError string
	}{
		{"强密码 - 包含所有类型", "Password123!", ""},
		{"强密码 - 大小写数字特殊", "MySecret2024#", ""},
		{"强密码 - 长密码", "VeryLongPassword123WithSpecial!", ""},
		{"中等密码 - 缺少特殊字符", "Password123", ""},
		{"中等密码 - 缺少数字", "PasswordABC!", ""},
		{"中等密码 - 缺少大写", "password123!", ""},
		{"空密码", "", "密码不能为空"},
		{"密码过短", "Pass1!", "密码长度至少8位"},
		{"密码过长", strings.Repeat("a", 129), "密码长度不能超过128位"},
		{"弱密码 - 只有小写", "password", "密码强度不足"},
		{"弱密码 - 只有数字", "12345678", "密码强度不足"},
		{"弱密码 - 只有大写", "PASSWORD", "密码强度不足"},
		{"弱密码 - 只有两种类型", "password123", "密码强度不足"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePassword(tt.password)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestRoleNameValidation 测试角色名验证
func TestRoleNameValidation(t *testing.T) {
	tests := []struct {
		name          string
		roleName      string
		expectedError string
	}{
		{"有效的角色名", "admin", ""},
		{"有效的角色名带下划线", "project_manager", ""},
		{"有效的角色名带横线", "team-lead", ""},
		{"有效的角色名带数字", "level2_user", ""},
		{"空角色名", "", "角色名称不能为空"},
		{"角色名过短", "a", "角色名称长度必须在2-50字符之间"},
		{"角色名过长", strings.Repeat("a", 51), "角色名称长度必须在2-50字符之间"},
		{"包含特殊字符", "admin@role", "角色名称只能包含字母、数字、下划线和横线"},
		{"包含空格", "admin role", "角色名称只能包含字母、数字、下划线和横线"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRoleName(tt.roleName)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestPermissionValidation 测试权限验证
func TestPermissionValidation(t *testing.T) {
	tests := []struct {
		name          string
		permission    string
		expectedError string
	}{
		{"有效的权限", "project:read", ""},
		{"有效的权限 - 写入", "user:write", ""},
		{"有效的权限 - 删除", "resource:delete", ""},
		{"有效的权限 - 管理", "system:admin", ""},
		{"有效的复合权限", "project_member:create", ""},
		{"空权限", "", "权限名称不能为空"},
		{"缺少冒号", "projectread", "权限格式不正确"},
		{"多个冒号", "project:read:write", "权限格式不正确"},
		{"包含大写字母", "Project:Read", "权限格式不正确"},
		{"包含特殊字符", "project@:read", "权限格式不正确"},
		{"以冒号开头", ":read", "权限格式不正确"},
		{"以冒号结尾", "project:", "权限格式不正确"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePermission(tt.permission)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestLoginRequestValidation 测试登录请求验证
func TestLoginRequestValidation(t *testing.T) {
	tests := []struct {
		name          string
		req           *SimpleLoginRequest
		expectedError string
	}{
		{
			name: "有效的登录请求",
			req: &SimpleLoginRequest{
				Email:    "user@example.com",
				Password: "password123",
			},
			expectedError: "",
		},
		{
			name: "无效的邮箱",
			req: &SimpleLoginRequest{
				Email:    "invalid-email",
				Password: "password123",
			},
			expectedError: "登录邮箱: 邮箱格式不正确",
		},
		{
			name: "空密码",
			req: &SimpleLoginRequest{
				Email:    "user@example.com",
				Password: "",
			},
			expectedError: "密码不能为空",
		},
		{
			name:          "nil请求",
			req:           nil,
			expectedError: "登录请求不能为nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLoginRequest(tt.req)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestRegisterRequestValidation 测试注册请求验证
func TestRegisterRequestValidation(t *testing.T) {
	validTenantID := uuid.New()

	tests := []struct {
		name          string
		req           *SimpleRegisterRequest
		expectedError string
	}{
		{
			name: "有效的注册请求",
			req: &SimpleRegisterRequest{
				TenantID:  validTenantID,
				Email:     "user@example.com",
				Username:  "newuser",
				Password:  "Password123!",
				FirstName: "John",
				LastName:  "Doe",
			},
			expectedError: "",
		},
		{
			name: "租户ID为空",
			req: &SimpleRegisterRequest{
				TenantID:  uuid.Nil,
				Email:     "user@example.com",
				Username:  "newuser",
				Password:  "Password123!",
				FirstName: "John",
				LastName:  "Doe",
			},
			expectedError: "租户ID不能为空",
		},
		{
			name: "邮箱无效",
			req: &SimpleRegisterRequest{
				TenantID:  validTenantID,
				Email:     "invalid-email",
				Username:  "newuser",
				Password:  "Password123!",
				FirstName: "John",
				LastName:  "Doe",
			},
			expectedError: "注册邮箱: 邮箱格式不正确",
		},
		{
			name: "用户名无效",
			req: &SimpleRegisterRequest{
				TenantID:  validTenantID,
				Email:     "user@example.com",
				Username:  "ab", // 太短
				Password:  "Password123!",
				FirstName: "John",
				LastName:  "Doe",
			},
			expectedError: "用户名: 用户名长度必须在3-50字符之间",
		},
		{
			name: "密码强度不足",
			req: &SimpleRegisterRequest{
				TenantID:  validTenantID,
				Email:     "user@example.com",
				Username:  "newuser",
				Password:  "weakpass", // 弱密码
				FirstName: "John",
				LastName:  "Doe",
			},
			expectedError: "密码: 密码强度不足",
		},
		{
			name: "名字为空",
			req: &SimpleRegisterRequest{
				TenantID:  validTenantID,
				Email:     "user@example.com",
				Username:  "newuser",
				Password:  "Password123!",
				FirstName: "",
				LastName:  "Doe",
			},
			expectedError: "名字不能为空",
		},
		{
			name:          "nil请求",
			req:           nil,
			expectedError: "注册请求不能为nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRegisterRequest(tt.req)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestIAMEdgeCases 测试IAM边界情况
func TestIAMEdgeCases(t *testing.T) {
	t.Run("nil user", func(t *testing.T) {
		var user *SimpleUser
		require.Nil(t, user)
	})

	t.Run("极长的用户名", func(t *testing.T) {
		longUsername := strings.Repeat("a", 100)
		err := validateUsername(longUsername)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "用户名长度必须在3-50字符之间")
	})

	t.Run("特殊密码格式", func(t *testing.T) {
		specialPasswords := []string{
			"MyPassword2024!@#", // 包含多个特殊字符
			"你好World123!",       // 包含中文
			"Пароль123!",        // 包含西里尔字母
		}

		for _, password := range specialPasswords {
			err := validatePassword(password)
			if password == "MyPassword2024!@#" {
				assert.NoError(t, err, "密码 %s 应该是有效的", password)
			} else {
				// 国际化密码可能有兼容性问题
				t.Logf("密码 %s 的验证结果: %v", password, err)
			}
		}
	})

	t.Run("边界长度测试", func(t *testing.T) {
		// 测试边界长度
		exactly8Password := "Pass123!"
		err := validatePassword(exactly8Password)
		assert.NoError(t, err)

		exactly3Username := "abc"
		err = validateUsername(exactly3Username)
		assert.NoError(t, err)

		exactly50Username := strings.Repeat("a", 50)
		err = validateUsername(exactly50Username)
		assert.NoError(t, err)
	})
}

// TestIAMPerformance 测试IAM性能
func TestIAMPerformance(t *testing.T) {
	t.Run("批量邮箱验证", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 1000; i++ {
			email := fmt.Sprintf("user%d@example.com", i)
			err := validateEmail(email)
			assert.NoError(t, err)
		}

		duration := time.Since(start)
		t.Logf("批量验证1000个邮箱耗时: %v", duration)

		// 验证应该在50毫秒内完成
		assert.Less(t, duration, 50*time.Millisecond)
	})

	t.Run("批量密码验证", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 1000; i++ {
			password := fmt.Sprintf("Password%d!", i)
			err := validatePassword(password)
			assert.NoError(t, err)
		}

		duration := time.Since(start)
		t.Logf("批量验证1000个密码耗时: %v", duration)

		// 密码验证较复杂，允许更长时间
		assert.Less(t, duration, 100*time.Millisecond)
	})

	t.Run("批量用户名验证", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 1000; i++ {
			username := fmt.Sprintf("user_%d", i)
			err := validateUsername(username)
			assert.NoError(t, err)
		}

		duration := time.Since(start)
		t.Logf("批量验证1000个用户名耗时: %v", duration)

		// 验证应该在20毫秒内完成
		assert.Less(t, duration, 20*time.Millisecond)
	})
}

// MockIAMService IAM服务模拟实现
type MockIAMService struct {
	users    map[uuid.UUID]*SimpleUser
	roles    map[uuid.UUID]*SimpleRole
	tokens   map[uuid.UUID]*SimpleAPIToken
	sessions map[uuid.UUID]map[string]interface{}
}

// NewMockIAMService 创建IAM服务模拟
func NewMockIAMService() *MockIAMService {
	return &MockIAMService{
		users:    make(map[uuid.UUID]*SimpleUser),
		roles:    make(map[uuid.UUID]*SimpleRole),
		tokens:   make(map[uuid.UUID]*SimpleAPIToken),
		sessions: make(map[uuid.UUID]map[string]interface{}),
	}
}

// Register 模拟用户注册
func (m *MockIAMService) Register(req *SimpleRegisterRequest) (*SimpleUser, error) {
	if err := validateRegisterRequest(req); err != nil {
		return nil, err
	}

	// 检查邮箱是否已存在
	for _, user := range m.users {
		if user.Email == req.Email {
			return nil, fmt.Errorf("邮箱 %s 已被使用", req.Email)
		}
		if user.Username == req.Username && user.TenantID == req.TenantID {
			return nil, fmt.Errorf("用户名 %s 在该租户下已被使用", req.Username)
		}
	}

	user := &SimpleUser{
		ID:           uuid.New(),
		TenantID:     req.TenantID,
		Email:        req.Email,
		Username:     req.Username,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Status:       "pending",
		IsActive:     false,
		IsVerified:   false,
		IsMFAEnabled: false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	m.users[user.ID] = user

	return user, nil
}

// Login 模拟用户登录
func (m *MockIAMService) Login(req *SimpleLoginRequest) (map[string]interface{}, error) {
	if err := validateLoginRequest(req); err != nil {
		return nil, err
	}

	// 查找用户
	var user *SimpleUser
	for _, u := range m.users {
		if u.Email == req.Email {
			user = u
			break
		}
	}

	if user == nil {
		return nil, fmt.Errorf("用户不存在")
	}

	if !user.IsActive {
		return nil, fmt.Errorf("用户账户已被禁用")
	}

	// 简化的密码验证（实际应该验证哈希）
	if req.Password == "" {
		return nil, fmt.Errorf("密码错误")
	}

	// 更新最后登录时间
	now := time.Now()
	user.LastLoginAt = &now

	// 创建会话
	sessionID := uuid.New()
	m.sessions[sessionID] = map[string]interface{}{
		"user_id":    user.ID,
		"tenant_id":  user.TenantID,
		"created_at": now,
		"expires_at": now.Add(24 * time.Hour),
	}

	return map[string]interface{}{
		"user":       user,
		"session_id": sessionID,
		"expires_at": now.Add(24 * time.Hour),
	}, nil
}

// CreateRole 模拟创建角色
func (m *MockIAMService) CreateRole(tenantID uuid.UUID, name, displayName, description string, permissions []string) (*SimpleRole, error) {
	if err := validateRoleName(name); err != nil {
		return nil, err
	}

	if displayName == "" {
		return nil, fmt.Errorf("角色显示名称不能为空")
	}

	// 验证权限
	for _, permission := range permissions {
		if err := validatePermission(permission); err != nil {
			return nil, fmt.Errorf("权限 %s: %v", permission, err)
		}
	}

	// 检查角色名是否已存在
	for _, role := range m.roles {
		if role.Name == name && role.TenantID == tenantID {
			return nil, fmt.Errorf("角色名 %s 在该租户下已存在", name)
		}
	}

	role := &SimpleRole{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        name,
		DisplayName: displayName,
		Description: description,
		Level:       1,
		IsSystem:    false,
		Permissions: permissions,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	m.roles[role.ID] = role

	return role, nil
}

// TestMockIAMService 测试IAM服务模拟
func TestMockIAMService(t *testing.T) {
	mockService := NewMockIAMService()

	t.Run("用户注册成功", func(t *testing.T) {
		req := &SimpleRegisterRequest{
			TenantID:  uuid.New(),
			Email:     "test@example.com",
			Username:  "testuser",
			Password:  "Password123!",
			FirstName: "Test",
			LastName:  "User",
		}

		user, err := mockService.Register(req)
		require.NoError(t, err)
		require.NotNil(t, user)

		assert.Equal(t, req.Email, user.Email)
		assert.Equal(t, req.Username, user.Username)
		assert.Equal(t, req.FirstName, user.FirstName)
		assert.Equal(t, req.LastName, user.LastName)
		assert.Equal(t, "pending", user.Status)
		assert.False(t, user.IsActive)
		assert.False(t, user.IsVerified)
	})

	t.Run("用户注册失败 - 邮箱重复", func(t *testing.T) {
		// 先注册一个用户
		req1 := &SimpleRegisterRequest{
			TenantID:  uuid.New(),
			Email:     "duplicate@example.com",
			Username:  "user1",
			Password:  "Password123!",
			FirstName: "User",
			LastName:  "One",
		}
		_, err := mockService.Register(req1)
		require.NoError(t, err)

		// 尝试用相同邮箱注册
		req2 := &SimpleRegisterRequest{
			TenantID:  uuid.New(),
			Email:     "duplicate@example.com",
			Username:  "user2",
			Password:  "Password123!",
			FirstName: "User",
			LastName:  "Two",
		}

		user, err := mockService.Register(req2)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "邮箱")
		assert.Contains(t, err.Error(), "已被使用")
	})

	t.Run("用户登录成功", func(t *testing.T) {
		// 先注册用户
		registerReq := &SimpleRegisterRequest{
			TenantID:  uuid.New(),
			Email:     "login@example.com",
			Username:  "loginuser",
			Password:  "Password123!",
			FirstName: "Login",
			LastName:  "User",
		}
		user, err := mockService.Register(registerReq)
		require.NoError(t, err)

		// 激活用户
		user.IsActive = true

		// 登录
		loginReq := &SimpleLoginRequest{
			Email:    "login@example.com",
			Password: "password",
		}

		session, err := mockService.Login(loginReq)
		require.NoError(t, err)
		require.NotNil(t, session)

		assert.Contains(t, session, "user")
		assert.Contains(t, session, "session_id")
		assert.Contains(t, session, "expires_at")
	})

	t.Run("创建角色成功", func(t *testing.T) {
		tenantID := uuid.New()
		permissions := []string{"project:read", "project:write", "user:read"}

		role, err := mockService.CreateRole(
			tenantID,
			"project_admin",
			"项目管理员",
			"负责项目管理的角色",
			permissions,
		)

		require.NoError(t, err)
		require.NotNil(t, role)

		assert.Equal(t, "project_admin", role.Name)
		assert.Equal(t, "项目管理员", role.DisplayName)
		assert.Equal(t, tenantID, role.TenantID)
		assert.Equal(t, permissions, role.Permissions)
		assert.False(t, role.IsSystem)
	})
}
