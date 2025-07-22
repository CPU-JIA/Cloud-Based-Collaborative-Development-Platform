package vault

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Manager Vault管理器
type Manager struct {
	client      VaultClient
	config      *Config
	logger      *zap.Logger
	
	// 缓存的秘钥
	jwtSecret   string
	jwtSecretMu sync.RWMutex
	
	// 后台任务控制
	stopCh   chan struct{}
	stopOnce sync.Once
}

// NewManager 创建Vault管理器
func NewManager(config *Config, logger *zap.Logger) (*Manager, error) {
	var client VaultClient
	var err error
	
	// 根据配置决定使用真实客户端还是模拟客户端
	if config == nil || config.Address == "" {
		logger.Warn("Vault未配置，使用模拟客户端")
		client = NewMockVaultClient(logger)
	} else {
		client, err = NewVaultClient(config, logger)
		if err != nil {
			logger.Error("创建Vault客户端失败，回退到模拟客户端", zap.Error(err))
			client = NewMockVaultClient(logger)
		}
	}
	
	manager := &Manager{
		client: client,
		config: config,
		logger: logger,
		stopCh: make(chan struct{}),
	}
	
	// 初始化JWT秘钥
	if err := manager.initializeJWTSecret(context.Background()); err != nil {
		logger.Error("初始化JWT秘钥失败", zap.Error(err))
		return nil, err
	}
	
	// 启动后台任务
	go manager.runBackgroundTasks()
	
	logger.Info("Vault管理器初始化成功")
	return manager, nil
}

// initializeJWTSecret 初始化JWT秘钥
func (m *Manager) initializeJWTSecret(ctx context.Context) error {
	m.logger.Info("初始化JWT秘钥")
	
	// 尝试从Vault获取现有的JWT秘钥
	jwtSecret, err := m.client.GetJWTSecret(ctx)
	if err != nil {
		m.logger.Warn("获取JWT秘钥失败，生成新秘钥", zap.Error(err))
		
		// 生成新的JWT秘钥
		newSecret := generateRandomSecret(64)
		
		// 保存到Vault
		secretPath := "app/jwt"
		if m.config != nil && m.config.SecretPaths.JWT != "" {
			secretPath = m.config.SecretPaths.JWT
		}
		
		data := map[string]interface{}{
			"secret":     newSecret,
			"created_at": time.Now().Unix(),
			"created_by": "vault-manager",
		}
		
		if err := m.client.PutSecret(ctx, secretPath, data); err != nil {
			return fmt.Errorf("保存JWT秘钥失败: %w", err)
		}
		
		jwtSecret = newSecret
	}
	
	// 缓存JWT秘钥
	m.jwtSecretMu.Lock()
	m.jwtSecret = jwtSecret
	m.jwtSecretMu.Unlock()
	
	m.logger.Info("JWT秘钥初始化完成")
	return nil
}

// GetJWTSecret 获取JWT秘钥（从缓存）
func (m *Manager) GetJWTSecret() string {
	m.jwtSecretMu.RLock()
	defer m.jwtSecretMu.RUnlock()
	return m.jwtSecret
}

// RotateJWTSecret 轮换JWT秘钥
func (m *Manager) RotateJWTSecret(ctx context.Context) (string, error) {
	m.logger.Info("开始轮换JWT秘钥")
	
	newSecret, err := m.client.RotateJWTSecret(ctx)
	if err != nil {
		return "", fmt.Errorf("轮换JWT秘钥失败: %w", err)
	}
	
	// 更新缓存
	m.jwtSecretMu.Lock()
	oldSecret := m.jwtSecret
	m.jwtSecret = newSecret
	m.jwtSecretMu.Unlock()
	
	m.logger.Info("JWT秘钥轮换完成",
		zap.String("old_secret_prefix", oldSecret[:8]+"..."),
		zap.String("new_secret_prefix", newSecret[:8]+"..."))
	
	return newSecret, nil
}

// GetDatabaseCredentials 获取数据库凭证
func (m *Manager) GetDatabaseCredentials(ctx context.Context, role string) (*DatabaseCredentials, error) {
	return m.client.GetDatabaseCredentials(ctx, role)
}

// EncryptData 加密敏感数据
func (m *Manager) EncryptData(ctx context.Context, keyName, data string) (string, error) {
	return m.client.Encrypt(ctx, keyName, data)
}

// DecryptData 解密敏感数据
func (m *Manager) DecryptData(ctx context.Context, keyName, encryptedData string) (string, error) {
	return m.client.Decrypt(ctx, keyName, encryptedData)
}

// GetSecret 获取应用秘钥
func (m *Manager) GetSecret(ctx context.Context, path string) (map[string]interface{}, error) {
	return m.client.GetSecret(ctx, path)
}

// PutSecret 存储应用秘钥
func (m *Manager) PutSecret(ctx context.Context, path string, data map[string]interface{}) error {
	return m.client.PutSecret(ctx, path, data)
}

// DeleteSecret 删除应用秘钥
func (m *Manager) DeleteSecret(ctx context.Context, path string) error {
	return m.client.DeleteSecret(ctx, path)
}

// HealthCheck 健康检查
func (m *Manager) HealthCheck(ctx context.Context) error {
	return m.client.HealthCheck(ctx)
}

// runBackgroundTasks 运行后台任务
func (m *Manager) runBackgroundTasks() {
	ticker := time.NewTicker(1 * time.Hour) // 每小时检查一次
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			m.performPeriodicTasks()
		case <-m.stopCh:
			m.logger.Info("Vault管理器后台任务已停止")
			return
		}
	}
}

// performPeriodicTasks 执行周期性任务
func (m *Manager) performPeriodicTasks() {
	ctx := context.Background()
	
	// 健康检查
	if err := m.client.HealthCheck(ctx); err != nil {
		m.logger.Error("Vault健康检查失败", zap.Error(err))
	} else {
		m.logger.Debug("Vault健康检查通过")
	}
	
	// 可以在这里添加其他周期性任务，如：
	// - 检查秘钥过期时间
	// - 自动轮换过期的秘钥
	// - 清理过期的数据库凭证
}

// Close 关闭管理器
func (m *Manager) Close() error {
	m.stopOnce.Do(func() {
		close(m.stopCh)
	})
	
	if err := m.client.Close(); err != nil {
		m.logger.Error("关闭Vault客户端失败", zap.Error(err))
		return err
	}
	
	m.logger.Info("Vault管理器已关闭")
	return nil
}

// SecretManager 秘钥管理器接口（简化版）
type SecretManager interface {
	GetJWTSecret() string
	GetSecret(ctx context.Context, path string) (map[string]interface{}, error)
	EncryptData(ctx context.Context, keyName, data string) (string, error)
	DecryptData(ctx context.Context, keyName, encryptedData string) (string, error)
}

// NewSecretManager 创建秘钥管理器
func NewSecretManager(config *Config, logger *zap.Logger) (SecretManager, error) {
	return NewManager(config, logger)
}

// GetDefaultVaultConfig 获取默认Vault配置
func GetDefaultVaultConfig() *Config {
	return &Config{
		Address:    "http://localhost:8200",
		Token:      "",
		Namespace:  "",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		TLS: TLSConfig{
			Insecure: true, // 开发环境默认不验证TLS
		},
		SecretPaths: SecretPaths{
			JWT:      "app/jwt",
			Database: "database",
			Transit:  "transit",
			App:      "app/secrets",
		},
	}
}

// VaultInitializer Vault初始化器
type VaultInitializer struct {
	manager *Manager
	logger  *zap.Logger
}

// NewVaultInitializer 创建Vault初始化器
func NewVaultInitializer(config *Config, logger *zap.Logger) (*VaultInitializer, error) {
	manager, err := NewManager(config, logger)
	if err != nil {
		return nil, err
	}
	
	return &VaultInitializer{
		manager: manager,
		logger:  logger,
	}, nil
}

// InitializeSecrets 初始化应用需要的秘钥
func (v *VaultInitializer) InitializeSecrets(ctx context.Context) error {
	v.logger.Info("开始初始化应用秘钥")
	
	// 初始化应用默认秘钥
	defaultSecrets := map[string]map[string]interface{}{
		"app/database": {
			"host":     "localhost",
			"port":     5432,
			"name":     "collaborative_dev",
			"ssl_mode": "disable",
		},
		"app/redis": {
			"host": "localhost",
			"port": 6379,
			"db":   0,
		},
		"app/csrf": {
			"secret": generateRandomSecret(32),
		},
	}
	
	for path, data := range defaultSecrets {
		// 检查秘钥是否已存在
		_, err := v.manager.GetSecret(ctx, path)
		if err != nil {
			// 秘钥不存在，创建默认秘钥
			v.logger.Info("创建默认秘钥", zap.String("path", path))
			if err := v.manager.PutSecret(ctx, path, data); err != nil {
				v.logger.Error("创建默认秘钥失败", 
					zap.String("path", path), zap.Error(err))
				return err
			}
		} else {
			v.logger.Debug("秘钥已存在，跳过创建", zap.String("path", path))
		}
	}
	
	v.logger.Info("应用秘钥初始化完成")
	return nil
}

// GetManager 获取管理器实例
func (v *VaultInitializer) GetManager() *Manager {
	return v.manager
}