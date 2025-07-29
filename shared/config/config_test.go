package config

import (
	"os"
	"testing"
)

func TestConfig_EnvironmentVariableBinding(t *testing.T) {
	// 设置测试环境变量
	testEnvVars := map[string]string{
		"DATABASE_HOST":     "test-host",
		"DATABASE_PORT":     "5432",
		"DATABASE_NAME":     "test-db",
		"DATABASE_USER":     "test-user",
		"DATABASE_PASSWORD": "strongtestpassword2024",
		"JWT_SECRET":        "test_jwt_secret_key_32_chars_minimum_here_safe",
		"SERVER_PORT":       "8082",
	}

	// 设置环境变量
	for key, value := range testEnvVars {
		os.Setenv(key, value)
	}
	defer func() {
		// 清理环境变量
		for key := range testEnvVars {
			os.Unsetenv(key)
		}
	}()

	// 加载配置
	cfg, err := Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证环境变量是否正确绑定
	if cfg.Database.Host != "test-host" {
		t.Errorf("期望数据库主机为 'test-host'，实际为 '%s'", cfg.Database.Host)
	}

	if cfg.Database.Password != "strongtestpassword2024" {
		t.Errorf("期望数据库密码为 'strongtestpassword2024'，实际为 '%s'", cfg.Database.Password)
	}

	if cfg.Auth.JWTSecret != "test_jwt_secret_key_32_chars_minimum_here_safe" {
		t.Errorf("期望JWT密钥为 'test_jwt_secret_key_32_chars_minimum_here_safe'，实际为 '%s'", cfg.Auth.JWTSecret)
	}

	if cfg.Server.Port != 8082 {
		t.Errorf("期望服务器端口为 8082，实际为 %d", cfg.Server.Port)
	}
}

func TestConfig_DevelopmentDefaults(t *testing.T) {
	// 清除所有相关环境变量
	envVarsToUnset := []string{
		"DATABASE_PASSWORD", "JWT_SECRET", "SERVER_PORT",
	}
	for _, key := range envVarsToUnset {
		os.Unsetenv(key)
	}
	defer func() {
		for _, key := range envVarsToUnset {
			os.Unsetenv(key)
		}
	}()

	// 加载配置
	cfg, err := Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证开发环境默认值
	if cfg.IsDevelopment() {
		if cfg.Database.Password != "dev_password_123" {
			t.Errorf("期望开发环境默认数据库密码为 'dev_password_123'，实际为 '%s'", cfg.Database.Password)
		}

		if cfg.Auth.JWTSecret != "development_jwt_secret_key_32_chars_minimum_here_safe" {
			t.Errorf("期望开发环境默认JWT密钥，实际为 '%s'", cfg.Auth.JWTSecret)
		}
	}
}

func TestConfig_ValidationLogic(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "生产环境缺少数据库密码",
			config: Config{
				Server:   ServerConfig{Environment: "production", Port: 8080},
				Database: DatabaseConfig{Password: ""},
				Auth:     AuthConfig{JWTSecret: "test_jwt_secret_key_32_chars_minimum_here_safe"},
			},
			expectError: true,
			errorMsg:    "生产环境数据库密码不能为空",
		},
		{
			name: "生产环境缺少JWT密钥",
			config: Config{
				Server:   ServerConfig{Environment: "production", Port: 8080},
				Database: DatabaseConfig{Password: "strongtestpassword2024"},
				Auth:     AuthConfig{JWTSecret: ""},
			},
			expectError: true,
			errorMsg:    "生产环境JWT密钥不能为空",
		},
		{
			name: "开发环境缺少密码应该通过验证",
			config: Config{
				Server:   ServerConfig{Environment: "development", Port: 8080},
				Database: DatabaseConfig{Password: ""},
				Auth:     AuthConfig{JWTSecret: "development_jwt_secret_key_32_chars_minimum_here_safe"},
				Security: SecurityConfig{CorsAllowedOrigins: []string{"http://localhost:3000"}},
			},
			expectError: false,
		},
		{
			name: "JWT密钥太短",
			config: Config{
				Server:   ServerConfig{Environment: "development", Port: 8080},
				Database: DatabaseConfig{Password: "strongtestpassword2024"},
				Auth:     AuthConfig{JWTSecret: "short"},
			},
			expectError: true,
			errorMsg:    "JWT密钥长度必须至少32字符，当前长度: 5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("期望验证失败，但通过了验证")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("期望错误消息包含 '%s'，实际为 '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("期望验证通过，但失败了: %v", err)
				}
			}
		})
	}
}

func TestConfig_PortConfiguration(t *testing.T) {
	// 设置服务器端口环境变量
	os.Setenv("SERVER_PORT", "8082")
	defer os.Unsetenv("SERVER_PORT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if cfg.Server.Port != 8082 {
		t.Errorf("期望服务器端口为 8082，实际为 %d", cfg.Server.Port)
	}
}
