package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config 应用程序配置结构
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Kafka    KafkaConfig    `mapstructure:"kafka"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Log      LogConfig      `mapstructure:"log"`
	Monitor  MonitorConfig  `mapstructure:"monitor"`
	Security SecurityConfig `mapstructure:"security"`
	Storage  StorageConfig  `mapstructure:"storage"`
	CICD     CICDConfig     `mapstructure:"cicd"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port         int           `mapstructure:"port" default:"8080"`
	Host         string        `mapstructure:"host" default:"0.0.0.0"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout" default:"30s"`
	WriteTimeout time.Duration `mapstructure:"write_timeout" default:"30s"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout" default:"120s"`
	Environment  string        `mapstructure:"environment" default:"development"`
}

// Address 返回服务器监听地址
func (s *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host            string        `mapstructure:"host" default:"localhost"`
	Port            int           `mapstructure:"port" default:"5432"`
	Name            string        `mapstructure:"name" default:"devcollab"`
	User            string        `mapstructure:"user" default:"postgres"`
	Password        string        `mapstructure:"password"`
	SSLMode         string        `mapstructure:"ssl_mode" default:"disable"`
	MaxOpenConns    int           `mapstructure:"max_open_conns" default:"25"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns" default:"5"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime" default:"300s"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time" default:"60s"`
}

// ToDBConfig 转换为database.Config
func (d *DatabaseConfig) ToDBConfig() interface{} {
	return struct {
		Host            string
		Port            int
		Name            string
		User            string
		Password        string
		SSLMode         string
		MaxOpenConns    int
		MaxIdleConns    int
		ConnMaxLifetime time.Duration
		ConnMaxIdleTime time.Duration
		LogLevel        interface{}
	}{
		Host:            d.Host,
		Port:            d.Port,
		Name:            d.Name,
		User:            d.User,
		Password:        d.Password,
		SSLMode:         d.SSLMode,
		MaxOpenConns:    d.MaxOpenConns,
		MaxIdleConns:    d.MaxIdleConns,
		ConnMaxLifetime: d.ConnMaxLifetime,
		ConnMaxIdleTime: d.ConnMaxIdleTime,
		LogLevel:        "info",
	}
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host         string        `mapstructure:"host" default:"localhost"`
	Port         int           `mapstructure:"port" default:"6379"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db" default:"0"`
	PoolSize     int           `mapstructure:"pool_size" default:"10"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout" default:"5s"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout" default:"3s"`
	WriteTimeout time.Duration `mapstructure:"write_timeout" default:"3s"`
}

// KafkaConfig Kafka配置
type KafkaConfig struct {
	Brokers []string `mapstructure:"brokers" default:"localhost:9092"`
	GroupID string   `mapstructure:"group_id" default:"collaborative-platform"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	JWTSecret          string        `mapstructure:"jwt_secret"`
	JWTExpiration      time.Duration `mapstructure:"jwt_expiration" default:"24h"`
	RefreshTokenExpiry time.Duration `mapstructure:"refresh_token_expiry" default:"168h"` // 7天
	PasswordMinLength  int           `mapstructure:"password_min_length" default:"8"`
	MaxLoginAttempts   int           `mapstructure:"max_login_attempts" default:"5"`
	LockoutDuration    time.Duration `mapstructure:"lockout_duration" default:"15m"`
	SessionTimeout     time.Duration `mapstructure:"session_timeout" default:"30m"`
	TwoFactorEnabled   bool          `mapstructure:"two_factor_enabled" default:"false"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level" default:"info"`
	Format     string `mapstructure:"format" default:"json"`
	Output     string `mapstructure:"output" default:"stdout"`
	MaxSize    int    `mapstructure:"max_size" default:"100"` // MB
	MaxBackups int    `mapstructure:"max_backups" default:"3"`
	MaxAge     int    `mapstructure:"max_age" default:"28"` // 天
	Compress   bool   `mapstructure:"compress" default:"true"`
}

// ToLoggerConfig 转换为logger.Config
func (l *LogConfig) ToLoggerConfig() interface{} {
	return struct {
		Level      string `json:"level" yaml:"level"`
		Format     string `json:"format" yaml:"format"`
		Output     string `json:"output" yaml:"output"`
		FilePath   string `json:"file_path" yaml:"file_path"`
		MaxSize    int    `json:"max_size" yaml:"max_size"`
		MaxBackups int    `json:"max_backups" yaml:"max_backups"`
		MaxAge     int    `json:"max_age" yaml:"max_age"`
		Compress   bool   `json:"compress" yaml:"compress"`
	}{
		Level:      l.Level,
		Format:     l.Format,
		Output:     l.Output,
		FilePath:   "",
		MaxSize:    l.MaxSize,
		MaxBackups: l.MaxBackups,
		MaxAge:     l.MaxAge,
		Compress:   l.Compress,
	}
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	Enabled         bool    `mapstructure:"enabled" default:"true"`
	MetricsPort     int     `mapstructure:"metrics_port" default:"9090"`
	TracingEnabled  bool    `mapstructure:"tracing_enabled" default:"true"`
	TracingEndpoint string  `mapstructure:"tracing_endpoint"`
	SamplingRate    float64 `mapstructure:"sampling_rate" default:"0.1"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	CorsAllowedOrigins []string `mapstructure:"cors_allowed_origins"`
	TrustedProxies     []string `mapstructure:"trusted_proxies"`
	MaxRequestSize     string   `mapstructure:"max_request_size" default:"10MB"`
}

// Load 加载配置
func Load() (*Config, error) {
	var cfg Config

	// 设置配置文件路径
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("/etc/collaborative-platform")
	viper.AddConfigPath("$HOME/.collaborative-platform")

	// 设置环境变量前缀
	viper.SetEnvPrefix("COLLAB")
	viper.AutomaticEnv()

	// 设置环境变量映射
	viper.BindEnv("database.host", "DATABASE_HOST")
	viper.BindEnv("database.port", "DATABASE_PORT")
	viper.BindEnv("database.name", "DATABASE_NAME")
	viper.BindEnv("database.user", "DATABASE_USER")
	viper.BindEnv("database.password", "DATABASE_PASSWORD")
	viper.BindEnv("redis.host", "REDIS_HOST")
	viper.BindEnv("redis.port", "REDIS_PORT")
	viper.BindEnv("redis.password", "REDIS_PASSWORD")
	viper.BindEnv("kafka.brokers", "KAFKA_BROKERS")
	viper.BindEnv("auth.jwt_secret", "JWT_SECRET")

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 配置文件未找到，使用默认值
			fmt.Println("配置文件未找到，使用默认配置")
		} else {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	// 解析配置
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// 验证必要的配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return &cfg, nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Database.Password == "" {
		return fmt.Errorf("数据库密码不能为空")
	}

	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("JWT密钥不能为空")
	}

	// 增强JWT密钥验证 - 要求至少32字符长度
	if len(c.Auth.JWTSecret) < 32 {
		return fmt.Errorf("JWT密钥长度必须至少32字符，当前长度: %d", len(c.Auth.JWTSecret))
	}

	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("服务器端口无效: %d", c.Server.Port)
	}

	// 验证CORS配置
	if len(c.Security.CorsAllowedOrigins) == 0 && c.IsProduction() {
		return fmt.Errorf("生产环境必须配置CORS允许的域名")
	}

	return nil
}

// GetDatabaseDSN 获取数据库连接字符串
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

// GetRedisAddr 获取Redis地址
func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}

// IsDevelopment 是否为开发环境
func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

// IsProduction 是否为生产环境
func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

// StorageConfig 存储配置
type StorageConfig struct {
	Type       string              `mapstructure:"type" default:"local"`
	Local      LocalStorageConfig  `mapstructure:"local"`
	S3         S3StorageConfig     `mapstructure:"s3"`
	Cache      CacheStorageConfig  `mapstructure:"cache"`
	Artifact   ArtifactConfig      `mapstructure:"artifact"`
}

// LocalStorageConfig 本地存储配置
type LocalStorageConfig struct {
	BasePath    string   `mapstructure:"base_path" default:"/var/lib/cicd/storage"`
	MaxFileSize int64    `mapstructure:"max_file_size" default:"104857600"` // 100MB
	AllowedExts []string `mapstructure:"allowed_exts"`
}

// S3StorageConfig S3存储配置
type S3StorageConfig struct {
	Region          string `mapstructure:"region"`
	Bucket          string `mapstructure:"bucket"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	Endpoint        string `mapstructure:"endpoint"`
	UseSSL          bool   `mapstructure:"use_ssl" default:"true"`
}

// CacheStorageConfig 缓存存储配置
type CacheStorageConfig struct {
	Type        string        `mapstructure:"type" default:"memory"`
	TTL         time.Duration `mapstructure:"ttl" default:"30m"`
	MaxSize     int64         `mapstructure:"max_size" default:"52428800"` // 50MB
	MaxEntries  int           `mapstructure:"max_entries" default:"1000"`
}

// ArtifactConfig 构建产物配置
type ArtifactConfig struct {
	RetentionDays   int    `mapstructure:"retention_days" default:"30"`
	MaxSizePerJob   int64  `mapstructure:"max_size_per_job" default:"524288000"`   // 500MB
	MaxTotalSize    int64  `mapstructure:"max_total_size" default:"10737418240"`   // 10GB
	CompressionType string `mapstructure:"compression_type" default:"gzip"`
}

// CICDConfig CI/CD服务配置
type CICDConfig struct {
	Scheduler SchedulerConfig `mapstructure:"scheduler"`
	Runner    RunnerConfig    `mapstructure:"runner"`
}

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	WorkerCount       int           `mapstructure:"worker_count" default:"5"`
	QueueSize         int           `mapstructure:"queue_size" default:"1000"`
	PollInterval      time.Duration `mapstructure:"poll_interval" default:"10s"`
	JobTimeout        time.Duration `mapstructure:"job_timeout" default:"30m"`
	MaxRetries        int           `mapstructure:"max_retries" default:"3"`
	EnablePriority    bool          `mapstructure:"enable_priority" default:"true"`
	EnableLoadBalance bool          `mapstructure:"enable_load_balance" default:"true"`
}

// RunnerConfig 执行器配置
type RunnerConfig struct {
	PoolSize          int           `mapstructure:"pool_size" default:"10"`
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval" default:"30s"`
	MaxIdleTime       time.Duration `mapstructure:"max_idle_time" default:"5m"`
	EnableAutoScale   bool          `mapstructure:"enable_auto_scale" default:"false"`
}

// ToStorageConfig 转换为存储配置
func (s *StorageConfig) ToStorageConfig() interface{} {
	return map[string]interface{}{
		"type":     s.Type,
		"local":    s.Local,
		"s3":       s.S3,
		"cache":    s.Cache,
		"artifact": s.Artifact,
	}
}

// ToSchedulerConfig 转换为调度器配置
func (c *CICDConfig) ToSchedulerConfig() interface{} {
	return map[string]interface{}{
		"worker_count":        c.Scheduler.WorkerCount,
		"queue_size":          c.Scheduler.QueueSize,
		"poll_interval":       c.Scheduler.PollInterval,
		"job_timeout":         c.Scheduler.JobTimeout,
		"max_retries":         c.Scheduler.MaxRetries,
		"enable_priority":     c.Scheduler.EnablePriority,
		"enable_load_balance": c.Scheduler.EnableLoadBalance,
	}
}
