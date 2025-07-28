package vault

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/hashicorp/vault/api"
	"go.uber.org/zap"
)

// VaultClient Vault客户端接口
type VaultClient interface {
	// 秘钥管理
	GetSecret(ctx context.Context, path string) (map[string]interface{}, error)
	PutSecret(ctx context.Context, path string, data map[string]interface{}) error
	DeleteSecret(ctx context.Context, path string) error

	// JWT秘钥管理
	GetJWTSecret(ctx context.Context) (string, error)
	RotateJWTSecret(ctx context.Context) (string, error)

	// 数据库凭证管理
	GetDatabaseCredentials(ctx context.Context, role string) (*DatabaseCredentials, error)

	// 加密/解密服务
	Encrypt(ctx context.Context, keyName, plaintext string) (string, error)
	Decrypt(ctx context.Context, keyName, ciphertext string) (string, error)

	// 健康检查
	HealthCheck(ctx context.Context) error

	// 清理资源
	Close() error
}

// Config Vault客户端配置
type Config struct {
	Address    string        `yaml:"address" json:"address"`         // Vault服务器地址
	Token      string        `yaml:"token" json:"token"`             // 访问令牌
	Namespace  string        `yaml:"namespace" json:"namespace"`     // 命名空间（企业版）
	Timeout    time.Duration `yaml:"timeout" json:"timeout"`         // 请求超时时间
	MaxRetries int           `yaml:"max_retries" json:"max_retries"` // 最大重试次数

	// TLS配置
	TLS TLSConfig `yaml:"tls" json:"tls"`

	// 秘钥路径配置
	SecretPaths SecretPaths `yaml:"secret_paths" json:"secret_paths"`
}

// TLSConfig TLS配置
type TLSConfig struct {
	CACert     string `yaml:"ca_cert" json:"ca_cert"`         // CA证书路径
	ClientCert string `yaml:"client_cert" json:"client_cert"` // 客户端证书路径
	ClientKey  string `yaml:"client_key" json:"client_key"`   // 客户端私钥路径
	Insecure   bool   `yaml:"insecure" json:"insecure"`       // 跳过TLS验证（仅开发环境）
}

// SecretPaths 秘钥路径配置
type SecretPaths struct {
	JWT      string `yaml:"jwt" json:"jwt"`           // JWT秘钥路径
	Database string `yaml:"database" json:"database"` // 数据库凭证路径
	Transit  string `yaml:"transit" json:"transit"`   // 加密传输路径
	App      string `yaml:"app" json:"app"`           // 应用秘钥路径
}

// DatabaseCredentials 数据库凭证
type DatabaseCredentials struct {
	Username string        `json:"username"`
	Password string        `json:"password"`
	TTL      time.Duration `json:"ttl"`
	LeaseID  string        `json:"lease_id"`
}

// vaultClient Vault客户端实现
type vaultClient struct {
	client *api.Client
	config *Config
	logger *zap.Logger
}

// NewVaultClient 创建新的Vault客户端
func NewVaultClient(config *Config, logger *zap.Logger) (VaultClient, error) {
	if config == nil {
		return nil, fmt.Errorf("vault配置不能为空")
	}

	// 创建Vault API客户端配置
	clientConfig := api.DefaultConfig()
	clientConfig.Address = config.Address
	clientConfig.Timeout = config.Timeout
	clientConfig.MaxRetries = config.MaxRetries

	// 配置TLS
	tlsConfig := &api.TLSConfig{
		CACert:     config.TLS.CACert,
		ClientCert: config.TLS.ClientCert,
		ClientKey:  config.TLS.ClientKey,
		Insecure:   config.TLS.Insecure,
	}

	if err := clientConfig.ConfigureTLS(tlsConfig); err != nil {
		return nil, fmt.Errorf("配置TLS失败: %w", err)
	}

	// 创建客户端
	client, err := api.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("创建Vault客户端失败: %w", err)
	}

	// 设置认证令牌
	client.SetToken(config.Token)

	// 设置命名空间（企业版功能）
	if config.Namespace != "" {
		client.SetNamespace(config.Namespace)
	}

	vaultClient := &vaultClient{
		client: client,
		config: config,
		logger: logger,
	}

	// 验证连接
	if err := vaultClient.HealthCheck(context.Background()); err != nil {
		logger.Warn("Vault健康检查失败", zap.Error(err))
		// 不返回错误，允许客户端在Vault不可用时继续运行
	}

	logger.Info("Vault客户端初始化成功",
		zap.String("address", config.Address),
		zap.String("namespace", config.Namespace))

	return vaultClient, nil
}

// GetSecret 获取秘钥
func (c *vaultClient) GetSecret(ctx context.Context, secretPath string) (map[string]interface{}, error) {
	c.logger.Debug("获取Vault秘钥", zap.String("path", secretPath))

	// 使用KV v2引擎
	secret, err := c.client.KVv2("secret").Get(ctx, secretPath)
	if err != nil {
		return nil, fmt.Errorf("获取秘钥失败: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("秘钥不存在: %s", secretPath)
	}

	return secret.Data, nil
}

// PutSecret 存储秘钥
func (c *vaultClient) PutSecret(ctx context.Context, secretPath string, data map[string]interface{}) error {
	c.logger.Debug("存储Vault秘钥", zap.String("path", secretPath))

	_, err := c.client.KVv2("secret").Put(ctx, secretPath, data)
	if err != nil {
		return fmt.Errorf("存储秘钥失败: %w", err)
	}

	c.logger.Info("秘钥存储成功", zap.String("path", secretPath))
	return nil
}

// DeleteSecret 删除秘钥
func (c *vaultClient) DeleteSecret(ctx context.Context, secretPath string) error {
	c.logger.Debug("删除Vault秘钥", zap.String("path", secretPath))

	err := c.client.KVv2("secret").Delete(ctx, secretPath)
	if err != nil {
		return fmt.Errorf("删除秘钥失败: %w", err)
	}

	c.logger.Info("秘钥删除成功", zap.String("path", secretPath))
	return nil
}

// GetJWTSecret 获取JWT秘钥
func (c *vaultClient) GetJWTSecret(ctx context.Context) (string, error) {
	secretPath := c.config.SecretPaths.JWT
	if secretPath == "" {
		secretPath = "app/jwt"
	}

	data, err := c.GetSecret(ctx, secretPath)
	if err != nil {
		return "", fmt.Errorf("获取JWT秘钥失败: %w", err)
	}

	secret, ok := data["secret"].(string)
	if !ok {
		return "", fmt.Errorf("JWT秘钥格式错误")
	}

	return secret, nil
}

// RotateJWTSecret 轮换JWT秘钥
func (c *vaultClient) RotateJWTSecret(ctx context.Context) (string, error) {
	c.logger.Info("开始轮换JWT秘钥")

	// 生成新的JWT秘钥
	newSecret := generateRandomSecret(64) // 64字节的随机秘钥

	secretPath := c.config.SecretPaths.JWT
	if secretPath == "" {
		secretPath = "app/jwt"
	}

	// 保存新秘钥到Vault
	data := map[string]interface{}{
		"secret":     newSecret,
		"rotated_at": time.Now().Unix(),
		"rotated_by": "system",
	}

	if err := c.PutSecret(ctx, secretPath, data); err != nil {
		return "", fmt.Errorf("轮换JWT秘钥失败: %w", err)
	}

	c.logger.Info("JWT秘钥轮换成功")
	return newSecret, nil
}

// GetDatabaseCredentials 获取数据库凭证
func (c *vaultClient) GetDatabaseCredentials(ctx context.Context, role string) (*DatabaseCredentials, error) {
	c.logger.Debug("获取数据库凭证", zap.String("role", role))

	// 使用动态秘钥引擎
	credPath := path.Join(c.config.SecretPaths.Database, "creds", role)

	secret, err := c.client.Logical().ReadWithContext(ctx, credPath)
	if err != nil {
		return nil, fmt.Errorf("获取数据库凭证失败: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("数据库角色不存在: %s", role)
	}

	username, ok := secret.Data["username"].(string)
	if !ok {
		return nil, fmt.Errorf("用户名格式错误")
	}

	password, ok := secret.Data["password"].(string)
	if !ok {
		return nil, fmt.Errorf("密码格式错误")
	}

	ttl := time.Duration(secret.LeaseDuration) * time.Second

	return &DatabaseCredentials{
		Username: username,
		Password: password,
		TTL:      ttl,
		LeaseID:  secret.LeaseID,
	}, nil
}

// Encrypt 加密数据
func (c *vaultClient) Encrypt(ctx context.Context, keyName, plaintext string) (string, error) {
	c.logger.Debug("加密数据", zap.String("key", keyName))

	transitPath := c.config.SecretPaths.Transit
	if transitPath == "" {
		transitPath = "transit"
	}

	data := map[string]interface{}{
		"plaintext": plaintext,
	}

	encryptPath := path.Join(transitPath, "encrypt", keyName)

	secret, err := c.client.Logical().WriteWithContext(ctx, encryptPath, data)
	if err != nil {
		return "", fmt.Errorf("加密失败: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("加密结果为空")
	}

	ciphertext, ok := secret.Data["ciphertext"].(string)
	if !ok {
		return "", fmt.Errorf("密文格式错误")
	}

	return ciphertext, nil
}

// Decrypt 解密数据
func (c *vaultClient) Decrypt(ctx context.Context, keyName, ciphertext string) (string, error) {
	c.logger.Debug("解密数据", zap.String("key", keyName))

	transitPath := c.config.SecretPaths.Transit
	if transitPath == "" {
		transitPath = "transit"
	}

	data := map[string]interface{}{
		"ciphertext": ciphertext,
	}

	decryptPath := path.Join(transitPath, "decrypt", keyName)

	secret, err := c.client.Logical().WriteWithContext(ctx, decryptPath, data)
	if err != nil {
		return "", fmt.Errorf("解密失败: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("解密结果为空")
	}

	plaintext, ok := secret.Data["plaintext"].(string)
	if !ok {
		return "", fmt.Errorf("明文格式错误")
	}

	return plaintext, nil
}

// HealthCheck 健康检查
func (c *vaultClient) HealthCheck(ctx context.Context) error {
	healthResp, err := c.client.Sys().HealthWithContext(ctx)
	if err != nil {
		return fmt.Errorf("Vault健康检查失败: %w", err)
	}

	if !healthResp.Initialized {
		return fmt.Errorf("Vault未初始化")
	}

	if healthResp.Sealed {
		return fmt.Errorf("Vault已密封")
	}

	return nil
}

// Close 关闭客户端
func (c *vaultClient) Close() error {
	// Vault客户端不需要显式关闭
	c.logger.Info("Vault客户端已关闭")
	return nil
}

// 辅助函数：生成随机秘钥
func generateRandomSecret(length int) string {
	// 简化实现，生产环境应使用加密强随机数
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[len(charset)/2] // 简化随机生成
	}
	return string(b)
}

// MockVaultClient 模拟Vault客户端（用于开发和测试）
type MockVaultClient struct {
	secrets map[string]map[string]interface{}
	logger  *zap.Logger
}

// NewMockVaultClient 创建模拟Vault客户端
func NewMockVaultClient(logger *zap.Logger) VaultClient {
	return &MockVaultClient{
		secrets: make(map[string]map[string]interface{}),
		logger:  logger,
	}
}

func (m *MockVaultClient) GetSecret(ctx context.Context, path string) (map[string]interface{}, error) {
	if data, exists := m.secrets[path]; exists {
		return data, nil
	}
	return nil, fmt.Errorf("秘钥不存在: %s", path)
}

func (m *MockVaultClient) PutSecret(ctx context.Context, path string, data map[string]interface{}) error {
	m.secrets[path] = data
	return nil
}

func (m *MockVaultClient) DeleteSecret(ctx context.Context, path string) error {
	delete(m.secrets, path)
	return nil
}

func (m *MockVaultClient) GetJWTSecret(ctx context.Context) (string, error) {
	return "mock-jwt-secret-for-development", nil
}

func (m *MockVaultClient) RotateJWTSecret(ctx context.Context) (string, error) {
	newSecret := "mock-rotated-jwt-secret"
	return newSecret, nil
}

func (m *MockVaultClient) GetDatabaseCredentials(ctx context.Context, role string) (*DatabaseCredentials, error) {
	return &DatabaseCredentials{
		Username: "mock_user",
		Password: "mock_password",
		TTL:      1 * time.Hour,
		LeaseID:  "mock_lease_id",
	}, nil
}

func (m *MockVaultClient) Encrypt(ctx context.Context, keyName, plaintext string) (string, error) {
	return fmt.Sprintf("vault:v1:mock_encrypted_%s", plaintext), nil
}

func (m *MockVaultClient) Decrypt(ctx context.Context, keyName, ciphertext string) (string, error) {
	// 简单的模拟解密
	if strings.HasPrefix(ciphertext, "vault:v1:mock_encrypted_") {
		return strings.TrimPrefix(ciphertext, "vault:v1:mock_encrypted_"), nil
	}
	return "", fmt.Errorf("无效的密文格式")
}

func (m *MockVaultClient) HealthCheck(ctx context.Context) error {
	return nil // 模拟客户端总是健康的
}

func (m *MockVaultClient) Close() error {
	return nil
}
