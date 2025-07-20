package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/cloud-platform/collaborative-dev/shared/auth"
	"github.com/cloud-platform/collaborative-dev/shared/database"
	"github.com/cloud-platform/collaborative-dev/shared/models"
)

// MockJWTService JWT服务模拟 - 由于目前使用具体类型，我们创建一个包装器
type MockJWTService struct {
	mock.Mock
}

func (m *MockJWTService) GenerateTokenPair(userID, tenantID uuid.UUID, email, role string, permissions []string) (*auth.TokenPair, error) {
	args := m.Called(userID, tenantID, email, role, permissions)
	return args.Get(0).(*auth.TokenPair), args.Error(1)
}

func (m *MockJWTService) ValidateToken(tokenString string) (*auth.Claims, error) {
	args := m.Called(tokenString)
	return args.Get(0).(*auth.Claims), args.Error(1)
}

func (m *MockJWTService) RefreshToken(refreshTokenString string) (*auth.TokenPair, error) {
	args := m.Called(refreshTokenString)
	return args.Get(0).(*auth.TokenPair), args.Error(1)
}

func (m *MockJWTService) ExtractUserInfo(tokenString string) (userID, tenantID uuid.UUID, err error) {
	args := m.Called(tokenString)
	return args.Get(0).(uuid.UUID), args.Get(1).(uuid.UUID), args.Error(2)
}

// 添加其他JWTService方法
func (m *MockJWTService) HasPermission(tokenString, resource, action string) (bool, error) {
	args := m.Called(tokenString, resource, action)
	return args.Bool(0), args.Error(1)
}

func (m *MockJWTService) GetTokenExpiration(tokenString string) (time.Time, error) {
	args := m.Called(tokenString)
	return args.Get(0).(time.Time), args.Error(1)
}

func (m *MockJWTService) GeneratePasswordResetToken(userID, tenantID uuid.UUID, email string) (string, error) {
	args := m.Called(userID, tenantID, email)
	return args.String(0), args.Error(1)
}

func (m *MockJWTService) ValidatePasswordResetToken(tokenString string) (*auth.Claims, error) {
	args := m.Called(tokenString)
	return args.Get(0).(*auth.Claims), args.Error(1)
}

// TestUser 用于测试的简化用户模型，避免UUID兼容性问题
type TestUser struct {
	ID                string `gorm:"primary_key"`
	TenantID          string `gorm:"not null;index"`
	Email             string `gorm:"uniqueIndex;not null"`
	Username          string `gorm:"uniqueIndex;not null"`
	PasswordHash      string `gorm:"not null"`
	FirstName         string
	LastName          string
	Avatar            string
	Phone             string
	IsActive          bool `gorm:"default:true"`
	IsEmailVerified   bool `gorm:"default:false"`
	EmailVerifiedAt   *time.Time
	LastLoginAt       *time.Time
	FailedLoginCount  int `gorm:"default:0"`
	LockedUntil       *time.Time
	TwoFactorEnabled  bool `gorm:"default:false"`
	TwoFactorSecret   string
	PasswordResetAt   *time.Time
	PasswordChangedAt *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time `gorm:"index"`
}

// TableName 指定表名
func (TestUser) TableName() string {
	return "users"
}

// 设置测试数据库
func setupTestDB(t *testing.T) *database.PostgresDB {
	// 使用内存SQLite进行测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	// 迁移简化的表结构，适配SQLite
	err = db.AutoMigrate(
		&TestUser{},
		&models.Role{},
		&models.Permission{},
		&models.RefreshToken{},
	)
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)

	return &database.PostgresDB{
		DB:    db,
		SqlDB: sqlDB,
	}
}

// 创建测试用户服务
func createTestUserService(t *testing.T) (*UserService, *MockJWTService, *database.PostgresDB) {
	db := setupTestDB(t)
	mockJWT := &MockJWTService{}

	config := UserServiceConfig{
		PasswordMinLength: 8,
		MaxLoginAttempts:  5,
		LockoutDuration:   time.Minute * 15,
	}

	// 创建实际的JWT服务用于测试
	jwtService := auth.NewJWTService("test-secret-key-32-chars-long!!", time.Hour, time.Hour*24*7)
	service := NewUserService(db, jwtService, config)
	return service, mockJWT, db
}

func TestUserService_Register(t *testing.T) {
	service, _, db := createTestUserService(t)
	ctx := context.Background()

	// 创建测试租户
	tenant := &models.Tenant{
		ID:   uuid.New(),
		Name: "Test Tenant",
	}
	err := db.DB.Create(tenant).Error
	require.NoError(t, err)

	// 创建测试角色
	role := &models.Role{
		ID:       uuid.New(),
		TenantID: tenant.ID,
		Name:     "user",
	}
	err = db.DB.Create(role).Error
	require.NoError(t, err)

	tests := []struct {
		name    string
		request *RegisterRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "成功注册",
			request: &RegisterRequest{
				TenantID:  tenant.ID,
				Email:     "test@example.com",
				Password:  "password123",
				FirstName: "Test",
				LastName:  "User",
			},
			wantErr: false,
		},
		{
			name: "密码太短",
			request: &RegisterRequest{
				TenantID:  tenant.ID,
				Email:     "test2@example.com",
				Password:  "123",
				FirstName: "Test",
				LastName:  "User",
			},
			wantErr: true,
			errMsg:  "密码长度至少需要8位",
		},
		{
			name: "邮箱格式无效",
			request: &RegisterRequest{
				TenantID:  tenant.ID,
				Email:     "invalid-email",
				Password:  "password123",
				FirstName: "Test",
				LastName:  "User",
			},
			wantErr: true,
			errMsg:  "邮箱格式无效",
		},
		{
			name: "重复邮箱",
			request: &RegisterRequest{
				TenantID:  tenant.ID,
				Email:     "test@example.com", // 重复使用第一个测试的邮箱
				Password:  "password123",
				FirstName: "Test",
				LastName:  "User",
			},
			wantErr: true,
			errMsg:  "邮箱已被注册",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := service.Register(ctx, tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.request.Email, user.Email)
				assert.Equal(t, tt.request.FirstName, user.FirstName)
				assert.Equal(t, tt.request.LastName, user.LastName)
				assert.Equal(t, tt.request.TenantID, user.TenantID)
				assert.NotEmpty(t, user.ID)
				assert.NotEmpty(t, user.CreatedAt)
				assert.False(t, user.TwoFactorEnabled)
				assert.True(t, user.IsActive)
			}
		})
	}
}

func TestUserService_Login(t *testing.T) {
	service, _, db := createTestUserService(t)
	ctx := context.Background()

	// 创建测试租户
	tenant := &models.Tenant{
		ID:   uuid.New(),
		Name: "Test Tenant",
	}
	err := db.DB.Create(tenant).Error
	require.NoError(t, err)

	// 创建测试用户
	registerReq := &RegisterRequest{
		TenantID:  tenant.ID,
		Email:     "login@example.com",
		Password:  "password123",
		FirstName: "Login",
		LastName:  "Test",
	}

	_, err = service.Register(ctx, registerReq)
	require.NoError(t, err)

	tests := []struct {
		name      string
		request   *LoginRequest
		wantErr   bool
		errMsg    string
		expectMFA bool
	}{
		{
			name: "成功登录",
			request: &LoginRequest{
				Email:    "login@example.com",
				Password: "password123",
			},
			wantErr:   false,
			expectMFA: false,
		},
		{
			name: "错误密码",
			request: &LoginRequest{
				Email:    "login@example.com",
				Password: "wrongpassword",
			},
			wantErr: true,
			errMsg:  "邮箱或密码错误",
		},
		{
			name: "用户不存在",
			request: &LoginRequest{
				Email:    "nonexistent@example.com",
				Password: "password123",
			},
			wantErr: true,
			errMsg:  "邮箱或密码错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := service.Login(ctx, tt.request, "127.0.0.1", "test-agent")

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Equal(t, tt.expectMFA, response.RequiresMFA)

				if !tt.expectMFA {
					assert.NotNil(t, response.TokenPair)
					assert.NotEmpty(t, response.TokenPair.AccessToken)
					assert.NotEmpty(t, response.TokenPair.RefreshToken)
				}
			}
		})
	}
}

func TestUserService_MFARequired(t *testing.T) {
	service, _, db := createTestUserService(t)
	ctx := context.Background()

	// 创建测试租户
	tenant := &models.Tenant{
		ID:   uuid.New(),
		Name: "Test Tenant",
	}
	err := db.DB.Create(tenant).Error
	require.NoError(t, err)

	// 创建启用MFA的测试用户
	registerReq := &RegisterRequest{
		TenantID:  tenant.ID,
		Email:     "mfa@example.com",
		Password:  "password123",
		FirstName: "MFA",
		LastName:  "Test",
	}

	user, err := service.Register(ctx, registerReq)
	require.NoError(t, err)

	// 启用MFA
	err = db.DB.Model(user).Update("two_factor_enabled", true).Error
	require.NoError(t, err)

	t.Run("MFA用户登录应返回RequiresMFA", func(t *testing.T) {
		request := &LoginRequest{
			Email:    "mfa@example.com",
			Password: "password123",
		}

		response, err := service.Login(ctx, request, "127.0.0.1", "test-agent")

		// 这里会显示MFA验证逻辑缺失的问题
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.True(t, response.RequiresMFA, "应该要求MFA验证")
		assert.Nil(t, response.TokenPair, "MFA未完成时不应返回令牌")
	})
}

func TestUserService_ValidateToken(t *testing.T) {
	service, mockJWT, _ := createTestUserService(t)
	ctx := context.Background()

	userID := uuid.New()
	tenantID := uuid.New()

	mockClaims := &auth.Claims{
		UserID:   userID,
		TenantID: tenantID,
		Email:    "test@example.com",
		Role:     "user",
	}

	tests := []struct {
		name      string
		token     string
		setupMock func()
		wantErr   bool
	}{
		{
			name:  "有效令牌",
			token: "valid-token",
			setupMock: func() {
				mockJWT.On("ValidateToken", "valid-token").Return(mockClaims, nil).Once()
			},
			wantErr: false,
		},
		{
			name:  "无效令牌",
			token: "invalid-token",
			setupMock: func() {
				mockJWT.On("ValidateToken", "invalid-token").Return((*auth.Claims)(nil), assert.AnError).Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			user, err := service.ValidateToken(ctx, tt.token)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
			}

			mockJWT.AssertExpectations(t)
		})
	}
}

func TestUserService_ChangePassword(t *testing.T) {
	service, _, db := createTestUserService(t)
	ctx := context.Background()

	// 创建测试租户
	tenant := &models.Tenant{
		ID:   uuid.New(),
		Name: "Test Tenant",
	}
	err := db.DB.Create(tenant).Error
	require.NoError(t, err)

	// 创建测试用户
	registerReq := &RegisterRequest{
		TenantID:  tenant.ID,
		Email:     "changepw@example.com",
		Password:  "oldpassword123",
		FirstName: "Change",
		LastName:  "Password",
	}

	user, err := service.Register(ctx, registerReq)
	require.NoError(t, err)

	tests := []struct {
		name    string
		request *ChangePasswordRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "成功修改密码",
			request: &ChangePasswordRequest{
				CurrentPassword: "oldpassword123",
				NewPassword:     "newpassword123",
			},
			wantErr: false,
		},
		{
			name: "当前密码错误",
			request: &ChangePasswordRequest{
				CurrentPassword: "wrongpassword",
				NewPassword:     "newpassword123",
			},
			wantErr: true,
			errMsg:  "当前密码错误",
		},
		{
			name: "新密码太短",
			request: &ChangePasswordRequest{
				CurrentPassword: "oldpassword123",
				NewPassword:     "123",
			},
			wantErr: true,
			errMsg:  "新密码长度至少需要8位",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ChangePassword(ctx, user.ID, user.TenantID, tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)

				// 验证密码已更改
				var updatedUser models.User
				err = db.DB.Where("id = ?", user.ID).First(&updatedUser).Error
				require.NoError(t, err)

				// 新密码应该能够验证成功
				loginReq := &LoginRequest{
					Email:    user.Email,
					Password: tt.request.NewPassword,
				}
				_, err = service.Login(ctx, loginReq, "127.0.0.1", "test-agent")
				assert.NoError(t, err)
			}
		})
	}
}
