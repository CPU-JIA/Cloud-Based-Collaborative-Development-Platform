package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cloud-platform/collaborative-dev/shared/config"
	"github.com/cloud-platform/collaborative-dev/shared/database"
	"github.com/cloud-platform/collaborative-dev/shared/logger"
	gormlogger "gorm.io/gorm/logger"
)

// TestConfig 集成测试专用配置
type TestConfig struct {
	*config.Config
}

// LoadTestConfig 加载集成测试配置
func LoadTestConfig() (*TestConfig, error) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:         8080,
			Host:         "127.0.0.1",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
			Environment:  "test",
		},
		Database: config.DatabaseConfig{
			Host:            "localhost",
			Port:            5432,
			Name:            "devcollab_test",
			User:            "postgres",
			Password:        getTestDBPassword(),
			SSLMode:         "disable",
			MaxOpenConns:    5,
			MaxIdleConns:    2,
			ConnMaxLifetime: 300 * time.Second,
			ConnMaxIdleTime: 60 * time.Second,
		},
		Redis: config.RedisConfig{
			Host:         "localhost",
			Port:         6379,
			Password:     "",
			DB:           1, // 使用测试专用DB
			PoolSize:     5,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		},
		Auth: config.AuthConfig{
			JWTSecret:          getTestJWTSecret(),
			JWTExpiration:      24 * time.Hour,
			RefreshTokenExpiry: 168 * time.Hour,
			PasswordMinLength:  6, // 测试环境放宽要求
			MaxLoginAttempts:   5,
			LockoutDuration:    15 * time.Minute,
			SessionTimeout:     30 * time.Minute,
			TwoFactorEnabled:   false,
		},
		Log: config.LogConfig{
			Level:      "debug",
			Format:     "text",
			Output:     "stdout",
			MaxSize:    10,
			MaxBackups: 2,
			MaxAge:     7,
			Compress:   false,
		},
		Monitor: config.MonitorConfig{
			Enabled:         false, // 测试时禁用监控
			MetricsPort:     9090,
			TracingEnabled:  false,
			TracingEndpoint: "",
			SamplingRate:    1.0,
		},
		Security: config.SecurityConfig{
			CorsAllowedOrigins: []string{"http://localhost:3000", "http://127.0.0.1:3000"},
			TrustedProxies:     []string{"127.0.0.1"},
			MaxRequestSize:     "10MB",
		},
	}

	return &TestConfig{Config: cfg}, nil
}

// getTestDBPassword 获取测试数据库密码
func getTestDBPassword() string {
	// 优先使用环境变量
	if pwd := os.Getenv("TEST_DB_PASSWORD"); pwd != "" {
		return pwd
	}

	// 检查是否在CI环境中
	if os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true" {
		return "postgres" // CI环境默认密码
	}

	// Docker Compose环境
	if os.Getenv("DOCKER_ENV") == "true" {
		return "testpassword123"
	}

	// 本地开发环境 - 尝试读取密码文件
	homeDir, _ := os.UserHomeDir()
	passwordFile := filepath.Join(homeDir, ".devcollab_test_db_password")
	if data, err := os.ReadFile(passwordFile); err == nil {
		return string(data)
	}

	// 默认测试密码
	return "strongtestpassword2024"
}

// getTestJWTSecret 获取测试JWT密钥
func getTestJWTSecret() string {
	// 优先使用环境变量
	if secret := os.Getenv("TEST_JWT_SECRET"); secret != "" {
		return secret
	}

	// 测试专用32字符以上的JWT密钥
	return "test_jwt_secret_for_integration_testing_2024_cloud_platform"
}

// ToDBConfig 转换为database.Config (测试专用)
func (tc *TestConfig) ToDBConfig() database.Config {
	return database.Config{
		Host:            tc.Database.Host,
		Port:            tc.Database.Port,
		Name:            tc.Database.Name,
		User:            tc.Database.User,
		Password:        tc.Database.Password,
		SSLMode:         tc.Database.SSLMode,
		MaxOpenConns:    tc.Database.MaxOpenConns,
		MaxIdleConns:    tc.Database.MaxIdleConns,
		ConnMaxLifetime: tc.Database.ConnMaxLifetime,
		ConnMaxIdleTime: tc.Database.ConnMaxIdleTime,
		LogLevel:        gormlogger.Warn, // 测试时减少日志输出
	}
}

// ToLoggerConfig 转换为logger.Config (测试专用)
func (tc *TestConfig) ToLoggerConfig() logger.Config {
	return logger.Config{
		Level:      tc.Log.Level,
		Format:     tc.Log.Format,
		Output:     tc.Log.Output,
		FilePath:   "",
		MaxSize:    tc.Log.MaxSize,
		MaxBackups: tc.Log.MaxBackups,
		MaxAge:     tc.Log.MaxAge,
		Compress:   tc.Log.Compress,
	}
}

// Validate 测试配置验证（比生产环境宽松）
func (tc *TestConfig) Validate() error {
	// 基本验证
	if tc.Server.Port <= 0 || tc.Server.Port > 65535 {
		return fmt.Errorf("服务器端口无效: %d", tc.Server.Port)
	}

	// 测试环境允许较短的JWT密钥，但不能为空
	if tc.Auth.JWTSecret == "" {
		return fmt.Errorf("测试环境JWT密钥不能为空")
	}

	if len(tc.Auth.JWTSecret) < 16 {
		return fmt.Errorf("测试环境JWT密钥长度必须至少16字符，当前长度: %d", len(tc.Auth.JWTSecret))
	}

	return nil
}

// IsTest 是否为测试环境
func (tc *TestConfig) IsTest() bool {
	return tc.Server.Environment == "test"
}

// SetupTestDatabase 设置测试数据库
func (tc *TestConfig) SetupTestDatabase() error {
	fmt.Println("🔧 设置集成测试数据库...")

	// 设置测试环境变量
	os.Setenv("ENVIRONMENT", "test")
	os.Setenv("TEST_DB_PASSWORD", tc.Database.Password)
	os.Setenv("TEST_JWT_SECRET", tc.Auth.JWTSecret)

	// 初始化数据库连接
	dbConfig := tc.ToDBConfig()

	// 尝试连接数据库
	db, err := database.NewConnection(dbConfig)
	if err != nil {
		return fmt.Errorf("连接测试数据库失败: %w", err)
	}
	defer func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	}()

	fmt.Println("✅ 集成测试数据库连接成功")
	return nil
}

// PrintTestConfig 打印测试配置信息
func (tc *TestConfig) PrintTestConfig() {
	fmt.Println("📋 集成测试环境配置:")
	fmt.Printf("  服务器: %s\n", tc.Server.Address())
	fmt.Printf("  数据库: %s@%s:%d/%s\n", tc.Database.User, tc.Database.Host, tc.Database.Port, tc.Database.Name)
	fmt.Printf("  Redis: %s\n", tc.GetRedisAddr())
	fmt.Printf("  日志级别: %s\n", tc.Log.Level)
	fmt.Printf("  环境: %s\n", tc.Server.Environment)
}