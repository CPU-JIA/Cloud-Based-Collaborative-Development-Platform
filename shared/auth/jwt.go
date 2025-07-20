package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTService JWT认证服务
type JWTService struct {
	secretKey         []byte
	tokenExpiration   time.Duration
	refreshExpiration time.Duration
}

// Claims JWT声明结构
type Claims struct {
	UserID      uuid.UUID `json:"user_id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
	Permissions []string  `json:"permissions"`
	TokenType   string    `json:"token_type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// TokenPair 令牌对
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// NewJWTService 创建JWT服务实例
func NewJWTService(secretKey string, tokenExpiration, refreshExpiration time.Duration) *JWTService {
	return &JWTService{
		secretKey:         []byte(secretKey),
		tokenExpiration:   tokenExpiration,
		refreshExpiration: refreshExpiration,
	}
}

// GenerateTokenPair 生成访问令牌和刷新令牌对
func (j *JWTService) GenerateTokenPair(userID, tenantID uuid.UUID, email, role string, permissions []string) (*TokenPair, error) {
	now := time.Now()

	// 生成访问令牌
	accessClaims := &Claims{
		UserID:      userID,
		TenantID:    tenantID,
		Email:       email,
		Role:        role,
		Permissions: permissions,
		TokenType:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.tokenExpiration)),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "collaborative-platform",
			Audience:  []string{"collaborative-platform-api"},
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(j.secretKey)
	if err != nil {
		return nil, fmt.Errorf("生成访问令牌失败: %w", err)
	}

	// 生成刷新令牌
	refreshClaims := &Claims{
		UserID:    userID,
		TenantID:  tenantID,
		Email:     email,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.refreshExpiration)),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "collaborative-platform",
			Audience:  []string{"collaborative-platform-api"},
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(j.secretKey)
	if err != nil {
		return nil, fmt.Errorf("生成刷新令牌失败: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    now.Add(j.tokenExpiration),
		TokenType:    "Bearer",
	}, nil
}

// ValidateToken 验证令牌
func (j *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("无效的签名方法: %v", token.Header["alg"])
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("令牌解析失败: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("无效的令牌")
	}

	// 额外验证
	if err := j.validateClaims(claims); err != nil {
		return nil, fmt.Errorf("令牌声明验证失败: %w", err)
	}

	return claims, nil
}

// RefreshToken 刷新访问令牌
func (j *JWTService) RefreshToken(refreshTokenString string) (*TokenPair, error) {
	// 验证刷新令牌
	claims, err := j.ValidateToken(refreshTokenString)
	if err != nil {
		return nil, fmt.Errorf("刷新令牌验证失败: %w", err)
	}

	// 确保是刷新令牌
	if claims.TokenType != "refresh" {
		return nil, fmt.Errorf("提供的不是刷新令牌")
	}

	// 生成新的令牌对
	return j.GenerateTokenPair(
		claims.UserID,
		claims.TenantID,
		claims.Email,
		claims.Role,
		claims.Permissions,
	)
}

// ExtractUserInfo 从令牌中提取用户信息
func (j *JWTService) ExtractUserInfo(tokenString string) (userID, tenantID uuid.UUID, err error) {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	// 确保是访问令牌
	if claims.TokenType != "access" {
		return uuid.Nil, uuid.Nil, fmt.Errorf("提供的不是访问令牌")
	}

	return claims.UserID, claims.TenantID, nil
}

// HasPermission 检查用户是否具有特定权限
func (j *JWTService) HasPermission(tokenString, permission string) (bool, error) {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return false, err
	}

	// 检查权限列表
	for _, perm := range claims.Permissions {
		if perm == permission || perm == "*" { // * 表示所有权限
			return true, nil
		}
	}

	return false, nil
}

// GetTokenExpiration 获取令牌剩余有效时间
func (j *JWTService) GetTokenExpiration(tokenString string) (time.Duration, error) {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return 0, err
	}

	if claims.ExpiresAt == nil {
		return 0, fmt.Errorf("令牌没有过期时间")
	}

	return time.Until(claims.ExpiresAt.Time), nil
}

// validateClaims 验证声明内容
func (j *JWTService) validateClaims(claims *Claims) error {
	// 验证用户ID
	if claims.UserID == uuid.Nil {
		return fmt.Errorf("无效的用户ID")
	}

	// 验证租户ID（某些情况下可能为空，比如系统管理员）
	// if claims.TenantID == uuid.Nil {
	//     return fmt.Errorf("无效的租户ID")
	// }

	// 验证邮箱格式
	if claims.Email == "" {
		return fmt.Errorf("邮箱不能为空")
	}

	// 验证令牌类型
	if claims.TokenType != "access" && claims.TokenType != "refresh" {
		return fmt.Errorf("无效的令牌类型: %s", claims.TokenType)
	}

	// 验证发行者
	if claims.Issuer != "collaborative-platform" {
		return fmt.Errorf("无效的令牌发行者: %s", claims.Issuer)
	}

	return nil
}

// GeneratePasswordResetToken 生成密码重置令牌
func (j *JWTService) GeneratePasswordResetToken(userID uuid.UUID, email string) (string, error) {
	now := time.Now()

	claims := &Claims{
		UserID:    userID,
		Email:     email,
		TokenType: "password_reset",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Hour)), // 1小时有效期
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "collaborative-platform",
			Audience:  []string{"collaborative-platform-password-reset"},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(j.secretKey)
	if err != nil {
		return "", fmt.Errorf("生成密码重置令牌失败: %w", err)
	}

	return tokenString, nil
}

// ValidatePasswordResetToken 验证密码重置令牌
func (j *JWTService) ValidatePasswordResetToken(tokenString string) (uuid.UUID, string, error) {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return uuid.Nil, "", err
	}

	if claims.TokenType != "password_reset" {
		return uuid.Nil, "", fmt.Errorf("提供的不是密码重置令牌")
	}

	return claims.UserID, claims.Email, nil
}
