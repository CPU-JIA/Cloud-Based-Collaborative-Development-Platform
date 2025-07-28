package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloud-platform/collaborative-dev/shared/config/secrets"
	"github.com/spf13/viper"
)

// ConfigLoader 配置加载器
type ConfigLoader struct {
	secretManager secrets.SecretManager
	environment   string
	configPath    string
}

// NewConfigLoader 创建配置加载器
func NewConfigLoader(environment string) (*ConfigLoader, error) {
	// 确定配置路径
	configPath := getConfigPath()

	// 创建密钥管理器
	secretManager, err := createSecretManager(environment)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret manager: %w", err)
	}

	return &ConfigLoader{
		secretManager: secretManager,
		environment:   environment,
		configPath:    configPath,
	}, nil
}

// LoadConfig 加载配置
func (cl *ConfigLoader) LoadConfig() (*Config, error) {
	var cfg Config

	// 设置默认值
	cl.setDefaults()

	// 加载基础配置文件
	if err := cl.loadConfigFile("config.yaml"); err != nil {
		fmt.Printf("Warning: failed to load base config: %v\n", err)
	}

	// 加载环境特定配置
	envConfigFile := fmt.Sprintf("config.%s.yaml", cl.environment)
	if err := cl.loadConfigFile(envConfigFile); err != nil {
		fmt.Printf("Info: no environment-specific config found: %s\n", envConfigFile)
	}

	// 绑定环境变量
	cl.bindEnvironmentVariables()

	// 解析配置
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 加载密钥
	if err := cl.loadSecrets(&cfg); err != nil {
		return nil, fmt.Errorf("failed to load secrets: %w", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// setDefaults 设置默认值
func (cl *ConfigLoader) setDefaults() {
	// 服务器默认值
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.environment", cl.environment)
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.idle_timeout", "120s")

	// 数据库默认值
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.name", "devcollab")
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", "300s")
	viper.SetDefault("database.conn_max_idle_time", "60s")

	// Redis默认值
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.pool_size", 10)
	viper.SetDefault("redis.dial_timeout", "5s")
	viper.SetDefault("redis.read_timeout", "3s")
	viper.SetDefault("redis.write_timeout", "3s")

	// Kafka默认值
	viper.SetDefault("kafka.brokers", []string{"localhost:9092"})
	viper.SetDefault("kafka.group_id", "collaborative-platform")

	// 认证默认值
	viper.SetDefault("auth.jwt_expiration", "24h")
	viper.SetDefault("auth.refresh_token_expiry", "168h")
	viper.SetDefault("auth.password_min_length", 8)
	viper.SetDefault("auth.max_login_attempts", 5)
	viper.SetDefault("auth.lockout_duration", "15m")
	viper.SetDefault("auth.session_timeout", "30m")
	viper.SetDefault("auth.two_factor_enabled", false)

	// 日志默认值
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")
	viper.SetDefault("log.output", "stdout")
	viper.SetDefault("log.max_size", 100)
	viper.SetDefault("log.max_backups", 3)
	viper.SetDefault("log.max_age", 28)
	viper.SetDefault("log.compress", true)

	// 监控默认值
	viper.SetDefault("monitor.enabled", true)
	viper.SetDefault("monitor.metrics_port", 9090)
	viper.SetDefault("monitor.tracing_enabled", true)
	viper.SetDefault("monitor.sampling_rate", 0.1)

	// 安全默认值
	viper.SetDefault("security.max_request_size", "10MB")
}

// loadConfigFile 加载配置文件
func (cl *ConfigLoader) loadConfigFile(filename string) error {
	configFile := filepath.Join(cl.configPath, filename)

	// 检查文件是否存在
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %s", configFile)
	}

	// 读取配置文件
	v := viper.New()
	v.SetConfigFile(configFile)

	if err := v.ReadInConfig(); err != nil {
		return err
	}

	// 合并到主配置
	for key, value := range v.AllSettings() {
		viper.Set(key, value)
	}

	return nil
}

// bindEnvironmentVariables 绑定环境变量
func (cl *ConfigLoader) bindEnvironmentVariables() {
	// 启用自动环境变量
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 绑定特定的环境变量
	envBindings := map[string][]string{
		"server.port":                  {"SERVER_PORT", "PORT"},
		"server.environment":           {"ENVIRONMENT", "ENV"},
		"database.host":                {"DATABASE_HOST", "POSTGRES_HOST", "DB_HOST"},
		"database.port":                {"DATABASE_PORT", "POSTGRES_PORT", "DB_PORT"},
		"database.name":                {"DATABASE_NAME", "POSTGRES_DB", "DB_NAME"},
		"database.user":                {"DATABASE_USER", "POSTGRES_USER", "DB_USER"},
		"database.password":            {"DATABASE_PASSWORD", "POSTGRES_PASSWORD", "DB_PASSWORD"},
		"redis.host":                   {"REDIS_HOST"},
		"redis.port":                   {"REDIS_PORT"},
		"redis.password":               {"REDIS_PASSWORD"},
		"kafka.brokers":                {"KAFKA_BROKERS"},
		"auth.jwt_secret":              {"JWT_SECRET", "JWT_SECRET_KEY"},
		"monitor.tracing_endpoint":     {"TRACING_ENDPOINT", "JAEGER_ENDPOINT"},
		"storage.s3.access_key_id":     {"AWS_ACCESS_KEY_ID", "S3_ACCESS_KEY"},
		"storage.s3.secret_access_key": {"AWS_SECRET_ACCESS_KEY", "S3_SECRET_KEY"},
		"storage.s3.region":            {"AWS_REGION", "S3_REGION"},
		"storage.s3.bucket":            {"S3_BUCKET"},
	}

	for configKey, envVars := range envBindings {
		for _, envVar := range envVars {
			viper.BindEnv(configKey, envVar)
		}
	}
}

// loadSecrets 加载密钥
func (cl *ConfigLoader) loadSecrets(cfg *Config) error {
	// 数据库密码
	if cfg.Database.Password == "" {
		password, err := cl.secretManager.GetSecret("database_password")
		if err != nil {
			// 开发环境使用默认密码
			if cl.environment == "development" {
				cfg.Database.Password = "dev_password_123"
				fmt.Println("Warning: Using default database password for development environment")
			} else {
				return fmt.Errorf("failed to get database password: %w", err)
			}
		} else {
			cfg.Database.Password = password
		}
	}

	// JWT密钥
	if cfg.Auth.JWTSecret == "" {
		jwtSecret, err := cl.secretManager.GetSecret("jwt_secret")
		if err != nil {
			// 开发环境生成默认密钥
			if cl.environment == "development" {
				cfg.Auth.JWTSecret = "development_jwt_secret_key_32_chars_minimum_here_safe"
				fmt.Println("Warning: Using default JWT secret for development")
			} else {
				return fmt.Errorf("failed to get JWT secret: %w", err)
			}
		} else {
			cfg.Auth.JWTSecret = jwtSecret
		}
	}

	// Redis密码
	if cfg.Redis.Password == "" {
		redisPassword, err := cl.secretManager.GetSecret("redis_password")
		if err != nil {
			// Redis密码是可选的，不记录具体错误信息
			fmt.Printf("Info: Redis password not configured, using default connection\n")
		} else {
			cfg.Redis.Password = redisPassword
		}
	}

	// S3凭据
	if cfg.Storage.Type == "s3" {
		if cfg.Storage.S3.AccessKeyID == "" {
			accessKey, err := cl.secretManager.GetSecret("s3_access_key_id")
			if err == nil {
				cfg.Storage.S3.AccessKeyID = accessKey
			}
		}

		if cfg.Storage.S3.SecretAccessKey == "" {
			secretKey, err := cl.secretManager.GetSecret("s3_secret_access_key")
			if err == nil {
				cfg.Storage.S3.SecretAccessKey = secretKey
			}
		}
	}

	return nil
}

// createSecretManager 创建密钥管理器
func createSecretManager(environment string) (secrets.SecretManager, error) {
	// 获取加密密钥
	encryptionKey := os.Getenv("SECRETS_ENCRYPTION_KEY")
	if encryptionKey == "" && environment == "production" {
		return nil, fmt.Errorf("SECRETS_ENCRYPTION_KEY is required in production")
	}

	// 根据环境选择密钥提供者
	var provider secrets.SecretProvider
	var err error

	switch environment {
	case "production":
		// 生产环境使用Vault
		vaultAddr := os.Getenv("VAULT_ADDR")
		if vaultAddr != "" {
			provider, err = createVaultProvider()
			if err != nil {
				return nil, fmt.Errorf("failed to create vault provider: %w", err)
			}
		} else {
			// 如果没有Vault，使用环境变量
			provider = secrets.NewEnvironmentProvider("CLOUDPLATFORM")
		}

	case "development", "test":
		// 开发和测试环境使用文件提供者
		secretsPath := filepath.Join(getConfigPath(), "secrets", fmt.Sprintf("%s.secrets.yaml", environment))
		provider, err = secrets.NewFileProvider(secretsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create file provider: %w", err)
		}

	default:
		// 默认使用环境变量
		provider = secrets.NewEnvironmentProvider("CLOUDPLATFORM")
	}

	// 创建密钥管理器
	if encryptionKey == "" {
		encryptionKey = "default_encryption_key_for_dev_only"
	}

	manager, err := secrets.NewManager(provider, encryptionKey)
	if err != nil {
		return nil, err
	}

	return manager, nil
}

// createVaultProvider 创建Vault提供者
func createVaultProvider() (secrets.SecretProvider, error) {
	config := secrets.VaultConfig{
		Address:    os.Getenv("VAULT_ADDR"),
		Token:      os.Getenv("VAULT_TOKEN"),
		Namespace:  os.Getenv("VAULT_NAMESPACE"),
		MountPath:  getEnvOrDefault("VAULT_MOUNT_PATH", "secret"),
		SecretPath: getEnvOrDefault("VAULT_SECRET_PATH", "data/cloudplatform"),
	}

	// TLS配置
	// TLS配置暂时注释，需要导入正确的类型
	// if certPath := os.Getenv("VAULT_CACERT"); certPath != "" {
	// 	config.TLSConfig = &vault.TLSConfig{
	// 		CACert: certPath,
	// 	}
	// }

	return secrets.NewVaultProvider(config)
}

// getConfigPath 获取配置路径
func getConfigPath() string {
	// 优先级：环境变量 > 当前目录 > 默认路径
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		return path
	}

	// 检查当前目录
	if _, err := os.Stat("./configs"); err == nil {
		return "./configs"
	}

	// 检查项目根目录
	if _, err := os.Stat("../../configs"); err == nil {
		return "../../configs"
	}

	// 默认路径
	return "/etc/collaborative-platform"
}

// getEnvOrDefault 获取环境变量或默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
