package auth

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"
)

// MFAService MFA多因子认证服务
type MFAService struct {
	issuer    string
	keySize   uint
	digits    otp.Digits
	algorithm otp.Algorithm
	period    uint
	skew      uint
}

// MFAConfig MFA配置
type MFAConfig struct {
	Issuer    string        `json:"issuer" yaml:"issuer"`
	KeySize   uint          `json:"key_size" yaml:"key_size"`
	Digits    otp.Digits    `json:"digits" yaml:"digits"`
	Algorithm otp.Algorithm `json:"algorithm" yaml:"algorithm"`
	Period    uint          `json:"period" yaml:"period"`
	Skew      uint          `json:"skew" yaml:"skew"`
}

// QRCodeInfo QR码信息
type QRCodeInfo struct {
	Secret    string `json:"secret"`
	QRCodeURL string `json:"qr_code_url"`
	QRCodeB64 string `json:"qr_code_base64"`
}

// NewMFAService 创建MFA服务实例
func NewMFAService(config MFAConfig) *MFAService {
	// 设置默认值
	if config.KeySize == 0 {
		config.KeySize = 32
	}
	if config.Digits == 0 {
		config.Digits = otp.DigitsSix
	}
	if config.Algorithm == 0 {
		config.Algorithm = otp.AlgorithmSHA1
	}
	if config.Period == 0 {
		config.Period = 30
	}
	if config.Skew == 0 {
		config.Skew = 1
	}
	if config.Issuer == "" {
		config.Issuer = "Collaborative Platform"
	}

	return &MFAService{
		issuer:    config.Issuer,
		keySize:   config.KeySize,
		digits:    config.Digits,
		algorithm: config.Algorithm,
		period:    config.Period,
		skew:      config.Skew,
	}
}

// GenerateSecret 为用户生成TOTP密钥
func (m *MFAService) GenerateSecret(userEmail string) (*QRCodeInfo, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      m.issuer,
		AccountName: userEmail,
		SecretSize:  m.keySize,
		Digits:      m.digits,
		Algorithm:   m.algorithm,
		Period:      m.period,
	})
	if err != nil {
		return nil, fmt.Errorf("生成TOTP密钥失败: %w", err)
	}

	// 生成QR码
	qrCode, err := qrcode.Encode(key.URL(), qrcode.Medium, 256)
	if err != nil {
		return nil, fmt.Errorf("生成QR码失败: %w", err)
	}

	// 转换为Base64
	qrCodeB64 := base64.StdEncoding.EncodeToString(qrCode)

	return &QRCodeInfo{
		Secret:    key.Secret(),
		QRCodeURL: key.URL(),
		QRCodeB64: qrCodeB64,
	}, nil
}

// ValidateCode 验证TOTP代码
func (m *MFAService) ValidateCode(secret, code string) bool {
	valid, _ := totp.ValidateCustom(code, secret, time.Now(), totp.ValidateOpts{
		Period:    m.period,
		Skew:      m.skew,
		Digits:    m.digits,
		Algorithm: m.algorithm,
	})
	return valid
}

// GenerateBackupCodes 生成备用验证码
func (m *MFAService) GenerateBackupCodes(count int) ([]string, error) {
	if count <= 0 || count > 20 {
		count = 10 // 默认生成10个备用码
	}

	backupCodes := make([]string, count)
	for i := 0; i < count; i++ {
		code, err := m.generateRandomCode()
		if err != nil {
			return nil, fmt.Errorf("生成备用码失败: %w", err)
		}
		backupCodes[i] = code
	}

	return backupCodes, nil
}

// generateRandomCode 生成随机备用验证码
func (m *MFAService) generateRandomCode() (string, error) {
	// 生成8位随机数字代码
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "backup",
		AccountName: "temp",
		SecretSize:  10,
	})
	if err != nil {
		return "", err
	}

	// 使用当前时间生成代码并格式化为8位
	code, err := totp.GenerateCodeCustom(key.Secret(), time.Now(), totp.ValidateOpts{
		Period:    30,
		Digits:    otp.DigitsEight,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		return "", err
	}

	return code, nil
}

// GetCurrentCode 获取当前TOTP代码（用于测试）
func (m *MFAService) GetCurrentCode(secret string) (string, error) {
	return totp.GenerateCodeCustom(secret, time.Now(), totp.ValidateOpts{
		Period:    m.period,
		Digits:    m.digits,
		Algorithm: m.algorithm,
	})
}

// IsValidBackupCode 验证备用码格式
func (m *MFAService) IsValidBackupCode(code string) bool {
	// 备用码应该是8位数字
	if len(code) != 8 {
		return false
	}

	for _, char := range code {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}

// GetTimeWindow 获取当前时间窗口信息
func (m *MFAService) GetTimeWindow() map[string]interface{} {
	now := time.Now()
	currentWindow := now.Unix() / int64(m.period)

	return map[string]interface{}{
		"current_time":   now.Unix(),
		"current_window": currentWindow,
		"period":         m.period,
		"skew":           m.skew,
		"next_window":    (currentWindow + 1) * int64(m.period),
		"prev_window":    (currentWindow - 1) * int64(m.period),
	}
}
