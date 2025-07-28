package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cloud-platform/collaborative-dev/shared/logger"
	"github.com/gin-gonic/gin"
)

// CSRFConfig CSRF保护配置
type CSRFConfig struct {
	// Secret 用于生成CSRF令牌的密钥
	Secret string

	// TokenLookup 令牌查找方式: "header:X-CSRF-Token,form:_csrf,query:_csrf"
	TokenLookup string

	// TokenLength 令牌长度
	TokenLength int

	// CookieName Cookie名称
	CookieName string

	// CookieDomain Cookie域名
	CookieDomain string

	// CookiePath Cookie路径
	CookiePath string

	// CookieMaxAge Cookie最大存活时间（秒）
	CookieMaxAge int

	// CookieHTTPOnly 是否设置HTTPOnly
	CookieHTTPOnly bool

	// CookieSecure 是否仅HTTPS
	CookieSecure bool

	// CookieSameSite SameSite策略
	CookieSameSite http.SameSite

	// ErrorHandler 错误处理函数
	ErrorHandler func(c *gin.Context, err error)

	// SkipperFunc 跳过检查的函数
	SkipperFunc func(c *gin.Context) bool

	// Mode CSRF模式: "double-submit" 或 "synchronizer-token"
	Mode string

	// SessionStore 会话存储（用于同步令牌模式）
	SessionStore SessionStore

	// Logger 日志记录器
	Logger logger.Logger
}

// SessionStore 会话存储接口
type SessionStore interface {
	Get(sessionID string) (string, error)
	Set(sessionID string, token string, expiry time.Duration) error
	Delete(sessionID string) error
}

// MemorySessionStore 内存会话存储
type MemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*sessionData
}

type sessionData struct {
	token  string
	expiry time.Time
}

// NewMemorySessionStore 创建内存会话存储
func NewMemorySessionStore() *MemorySessionStore {
	store := &MemorySessionStore{
		sessions: make(map[string]*sessionData),
	}
	// 启动清理过期会话的goroutine
	go store.cleanup()
	return store
}

func (s *MemorySessionStore) Get(sessionID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, exists := s.sessions[sessionID]
	if !exists || time.Now().After(data.expiry) {
		return "", fmt.Errorf("session not found or expired")
	}
	return data.token, nil
}

func (s *MemorySessionStore) Set(sessionID string, token string, expiry time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[sessionID] = &sessionData{
		token:  token,
		expiry: time.Now().Add(expiry),
	}
	return nil
}

func (s *MemorySessionStore) Delete(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
	return nil
}

func (s *MemorySessionStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for sessionID, data := range s.sessions {
			if now.After(data.expiry) {
				delete(s.sessions, sessionID)
			}
		}
		s.mu.Unlock()
	}
}

// DefaultCSRFConfig 默认CSRF配置
var DefaultCSRFConfig = CSRFConfig{
	TokenLookup:    "header:X-CSRF-Token,form:_csrf",
	TokenLength:    32,
	CookieName:     "_csrf",
	CookiePath:     "/",
	CookieMaxAge:   86400, // 24小时
	CookieHTTPOnly: false, // 设置为false以便JavaScript可以读取
	CookieSecure:   false, // 在生产环境应设置为true
	CookieSameSite: http.SameSiteLaxMode,
	Mode:           "double-submit",
}

// CSRFMiddleware CSRF中间件
type CSRFMiddleware struct {
	config *CSRFConfig
}

// NewCSRFMiddleware 创建CSRF中间件
func NewCSRFMiddleware(config *CSRFConfig) *CSRFMiddleware {
	if config == nil {
		cfg := DefaultCSRFConfig
		config = &cfg
	}

	if config.Secret == "" {
		// 返回一个错误处理函数而不是panic
		return func(c *gin.Context) {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "CSRF protection is not properly configured",
			})
			c.Abort()
		}
	}

	if config.TokenLength == 0 {
		config.TokenLength = DefaultCSRFConfig.TokenLength
	}

	if config.TokenLookup == "" {
		config.TokenLookup = DefaultCSRFConfig.TokenLookup
	}

	if config.CookieName == "" {
		config.CookieName = DefaultCSRFConfig.CookieName
	}

	if config.Mode == "" {
		config.Mode = DefaultCSRFConfig.Mode
	}

	if config.Mode == "synchronizer-token" && config.SessionStore == nil {
		config.SessionStore = NewMemorySessionStore()
	}

	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultCSRFErrorHandler
	}

	return &CSRFMiddleware{
		config: config,
	}
}

// Handler 返回Gin中间件处理函数
func (m *CSRFMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否跳过
		if m.config.SkipperFunc != nil && m.config.SkipperFunc(c) {
			c.Next()
			return
		}

		// 获取请求方法
		method := c.Request.Method

		// 安全方法不需要CSRF保护
		if isSafeMethod(method) {
			// 生成新令牌
			token := m.generateToken(c)
			m.setTokenInContext(c, token)
			m.setTokenInResponse(c, token)
			c.Next()
			return
		}

		// 验证CSRF令牌
		if err := m.validateToken(c); err != nil {
			if m.config.Logger != nil {
				m.config.Logger.WithFields(map[string]interface{}{
					"error":      err.Error(),
					"path":       c.Request.URL.Path,
					"method":     method,
					"ip":         c.ClientIP(),
					"user_agent": c.Request.UserAgent(),
				}).Warn("CSRF令牌验证失败")
			}
			m.config.ErrorHandler(c, err)
			c.Abort()
			return
		}

		c.Next()
	}
}

// generateToken 生成CSRF令牌
func (m *CSRFMiddleware) generateToken(c *gin.Context) string {
	// 生成随机字节
	tokenBytes := make([]byte, m.config.TokenLength)
	for i := 0; i < m.config.TokenLength; i++ {
		tokenBytes[i] = byte(time.Now().UnixNano() & 0xff)
	}

	// 使用HMAC签名
	mac := hmac.New(sha256.New, []byte(m.config.Secret))
	mac.Write(tokenBytes)
	signature := mac.Sum(nil)

	// 组合令牌和签名
	fullToken := append(tokenBytes, signature...)

	// Base64编码
	return base64.URLEncoding.EncodeToString(fullToken)
}

// validateToken 验证CSRF令牌
func (m *CSRFMiddleware) validateToken(c *gin.Context) error {
	// 获取令牌
	token := m.extractToken(c)
	if token == "" {
		return fmt.Errorf("CSRF令牌缺失")
	}

	// 根据模式验证
	switch m.config.Mode {
	case "double-submit":
		return m.validateDoubleSubmitToken(c, token)
	case "synchronizer-token":
		return m.validateSynchronizerToken(c, token)
	default:
		return fmt.Errorf("未知的CSRF模式: %s", m.config.Mode)
	}
}

// validateDoubleSubmitToken 验证双重提交Cookie模式
func (m *CSRFMiddleware) validateDoubleSubmitToken(c *gin.Context, token string) error {
	// 获取Cookie中的令牌
	cookie, err := c.Cookie(m.config.CookieName)
	if err != nil {
		return fmt.Errorf("CSRF Cookie缺失")
	}

	// 比较令牌
	if subtle.ConstantTimeCompare([]byte(token), []byte(cookie)) != 1 {
		return fmt.Errorf("CSRF令牌不匹配")
	}

	// 验证令牌签名
	return m.verifyTokenSignature(token)
}

// validateSynchronizerToken 验证同步令牌模式
func (m *CSRFMiddleware) validateSynchronizerToken(c *gin.Context, token string) error {
	// 获取会话ID
	sessionID := c.GetHeader("X-Session-ID")
	if sessionID == "" {
		cookie, err := c.Cookie("session_id")
		if err != nil {
			return fmt.Errorf("会话ID缺失")
		}
		sessionID = cookie
	}

	// 从存储中获取令牌
	storedToken, err := m.config.SessionStore.Get(sessionID)
	if err != nil {
		return fmt.Errorf("无效的会话")
	}

	// 比较令牌
	if subtle.ConstantTimeCompare([]byte(token), []byte(storedToken)) != 1 {
		return fmt.Errorf("CSRF令牌不匹配")
	}

	return nil
}

// verifyTokenSignature 验证令牌签名
func (m *CSRFMiddleware) verifyTokenSignature(token string) error {
	// Base64解码
	decoded, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return fmt.Errorf("无效的令牌格式")
	}

	// 检查长度
	if len(decoded) < m.config.TokenLength+32 { // 32是SHA256的长度
		return fmt.Errorf("令牌长度无效")
	}

	// 分离令牌和签名
	tokenBytes := decoded[:m.config.TokenLength]
	signature := decoded[m.config.TokenLength:]

	// 重新计算签名
	mac := hmac.New(sha256.New, []byte(m.config.Secret))
	mac.Write(tokenBytes)
	expectedSignature := mac.Sum(nil)

	// 恒定时间比较
	if subtle.ConstantTimeCompare(signature, expectedSignature) != 1 {
		return fmt.Errorf("令牌签名无效")
	}

	return nil
}

// extractToken 从请求中提取令牌
func (m *CSRFMiddleware) extractToken(c *gin.Context) string {
	// 解析查找配置
	lookups := strings.Split(m.config.TokenLookup, ",")

	for _, lookup := range lookups {
		parts := strings.Split(lookup, ":")
		if len(parts) != 2 {
			continue
		}

		source := strings.TrimSpace(parts[0])
		key := strings.TrimSpace(parts[1])

		switch source {
		case "header":
			if token := c.GetHeader(key); token != "" {
				return token
			}
		case "form":
			if token := c.PostForm(key); token != "" {
				return token
			}
		case "query":
			if token := c.Query(key); token != "" {
				return token
			}
		}
	}

	return ""
}

// setTokenInContext 在上下文中设置令牌
func (m *CSRFMiddleware) setTokenInContext(c *gin.Context, token string) {
	c.Set("csrf_token", token)
}

// setTokenInResponse 在响应中设置令牌
func (m *CSRFMiddleware) setTokenInResponse(c *gin.Context, token string) {
	// 设置Cookie
	if m.config.Mode == "double-submit" {
		c.SetSameSite(m.config.CookieSameSite)
		c.SetCookie(
			m.config.CookieName,
			token,
			m.config.CookieMaxAge,
			m.config.CookiePath,
			m.config.CookieDomain,
			m.config.CookieSecure,
			m.config.CookieHTTPOnly,
		)
	}

	// 设置响应头（供JavaScript读取）
	c.Header("X-CSRF-Token", token)
}

// isSafeMethod 检查是否为安全方法
func isSafeMethod(method string) bool {
	return method == http.MethodGet ||
		method == http.MethodHead ||
		method == http.MethodOptions ||
		method == http.MethodTrace
}

// defaultCSRFErrorHandler 默认错误处理
func defaultCSRFErrorHandler(c *gin.Context, err error) {
	c.JSON(http.StatusForbidden, gin.H{
		"error":   "CSRF令牌验证失败",
		"code":    "CSRF_VALIDATION_FAILED",
		"details": err.Error(),
	})
}

// GetCSRFToken 获取当前请求的CSRF令牌
func GetCSRFToken(c *gin.Context) string {
	token, _ := c.Get("csrf_token")
	if tokenStr, ok := token.(string); ok {
		return tokenStr
	}
	return ""
}

// CSRFProtection 便捷函数，使用默认配置创建CSRF中间件
func CSRFProtection(secret string) gin.HandlerFunc {
	config := DefaultCSRFConfig
	config.Secret = secret
	middleware := NewCSRFMiddleware(&config)
	return middleware.Handler()
}

// CSRFProtectionWithConfig 使用自定义配置创建CSRF中间件
func CSRFProtectionWithConfig(config CSRFConfig) gin.HandlerFunc {
	middleware := NewCSRFMiddleware(&config)
	return middleware.Handler()
}
