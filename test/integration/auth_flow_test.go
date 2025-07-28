package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getJWTSecret 从环境变量获取JWT密钥
func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		// 仅用于测试环境的默认值
		secret = "test-jwt-secret-key-minimum-32-characters-long"
	}
	return []byte(secret)
}

// JWT密钥 (从环境变量获取)
var jwtSecretKey = getJWTSecret()

// Claims JWT声明
type Claims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// User 用户模型
type User struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// AuthService 认证服务接口
type AuthService interface {
	Register(email, username, password string) (*User, error)
	Login(email, password string) (string, *User, error)
	ValidateToken(tokenString string) (*Claims, error)
	RefreshToken(tokenString string) (string, error)
}

// MockAuthService 模拟认证服务
type MockAuthService struct {
	users  map[string]*User
	tokens map[string]*Claims
	nextID int
}

// NewMockAuthService 创建模拟认证服务
func NewMockAuthService() *MockAuthService {
	return &MockAuthService{
		users:  make(map[string]*User),
		tokens: make(map[string]*Claims),
		nextID: 1,
	}
}

// Register 注册用户
func (m *MockAuthService) Register(email, username, password string) (*User, error) {
	if _, exists := m.users[email]; exists {
		return nil, fmt.Errorf("用户已存在")
	}

	user := &User{
		ID:        m.nextID,
		Email:     email,
		Username:  username,
		Role:      "developer",
		IsActive:  true,
		CreatedAt: time.Now(),
	}

	m.users[email] = user
	m.nextID++
	return user, nil
}

// Login 用户登录
func (m *MockAuthService) Login(email, password string) (string, *User, error) {
	user, exists := m.users[email]
	if !exists {
		return "", nil, fmt.Errorf("用户不存在")
	}

	if !user.IsActive {
		return "", nil, fmt.Errorf("用户已被禁用")
	}

	// 生成JWT token
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "collaborative-dev-platform",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecretKey)
	if err != nil {
		return "", nil, fmt.Errorf("生成token失败: %v", err)
	}

	m.tokens[tokenString] = claims
	return tokenString, user, nil
}

// ValidateToken 验证token
func (m *MockAuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("token解析失败: %v", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token无效")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("token claims类型错误")
	}

	return claims, nil
}

// RefreshToken 刷新token
func (m *MockAuthService) RefreshToken(tokenString string) (string, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	// 生成新token
	newClaims := &Claims{
		UserID: claims.UserID,
		Email:  claims.Email,
		Role:   claims.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "collaborative-dev-platform",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	newTokenString, err := token.SignedString(jwtSecretKey)
	if err != nil {
		return "", fmt.Errorf("生成新token失败: %v", err)
	}

	delete(m.tokens, tokenString)
	m.tokens[newTokenString] = newClaims
	return newTokenString, nil
}

// setupRouter 设置测试路由
func setupRouter(authService AuthService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// 注册端点
	r.POST("/api/v1/auth/register", func(c *gin.Context) {
		var req struct {
			Email    string `json:"email" binding:"required,email"`
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required,min=8"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user, err := authService.Register(req.Email, req.Username, req.Password)
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "注册成功",
			"user":    user,
		})
	})

	// 登录端点
	r.POST("/api/v1/auth/login", func(c *gin.Context) {
		var req struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		token, user, err := authService.Login(req.Email, req.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token": token,
			"user":  user,
		})
	})

	// 刷新token端点
	r.POST("/api/v1/auth/refresh", func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少Authorization header"})
			return
		}

		tokenString := authHeader[7:] // 移除 "Bearer " 前缀
		newToken, err := authService.RefreshToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token": newToken,
		})
	})

	// 受保护的端点示例
	r.GET("/api/v1/profile", func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少Authorization header"})
			return
		}

		tokenString := authHeader[7:]
		claims, err := authService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user_id": claims.UserID,
			"email":   claims.Email,
			"role":    claims.Role,
		})
	})

	return r
}

// TestAuthenticationFlow 测试完整的认证流程
func TestAuthenticationFlow(t *testing.T) {
	authService := NewMockAuthService()
	router := setupRouter(authService)

	t.Run("完整认证流程", func(t *testing.T) {
		// 1. 注册新用户
		registerReq := map[string]string{
			"email":    "test@example.com",
			"username": "testuser",
			"password": "password123",
		}
		body, _ := json.Marshal(registerReq)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var registerResp struct {
			Message string `json:"message"`
			User    *User  `json:"user"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &registerResp)
		require.NoError(t, err)
		assert.Equal(t, "注册成功", registerResp.Message)
		assert.Equal(t, "test@example.com", registerResp.User.Email)

		// 2. 登录获取token
		loginReq := map[string]string{
			"email":    "test@example.com",
			"password": "password123",
		}
		body, _ = json.Marshal(loginReq)

		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var loginResp struct {
			Token string `json:"token"`
			User  *User  `json:"user"`
		}
		err = json.Unmarshal(w.Body.Bytes(), &loginResp)
		require.NoError(t, err)
		assert.NotEmpty(t, loginResp.Token)

		// 3. 使用token访问受保护资源
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/api/v1/profile", nil)
		req.Header.Set("Authorization", "Bearer "+loginResp.Token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var profileResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &profileResp)
		require.NoError(t, err)
		assert.Equal(t, "test@example.com", profileResp["email"])

		// 4. 刷新token
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/api/v1/auth/refresh", nil)
		req.Header.Set("Authorization", "Bearer "+loginResp.Token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var refreshResp struct {
			Token string `json:"token"`
		}
		err = json.Unmarshal(w.Body.Bytes(), &refreshResp)
		require.NoError(t, err)
		assert.NotEmpty(t, refreshResp.Token)
		assert.NotEqual(t, loginResp.Token, refreshResp.Token)
	})

	t.Run("无效凭据登录", func(t *testing.T) {
		loginReq := map[string]string{
			"email":    "nonexistent@example.com",
			"password": "wrongpassword",
		}
		body, _ := json.Marshal(loginReq)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("无token访问受保护资源", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/profile", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("重复注册", func(t *testing.T) {
		// 先注册一个用户
		authService.Register("duplicate@example.com", "dupuser", "password123")

		// 尝试重复注册
		registerReq := map[string]string{
			"email":    "duplicate@example.com",
			"username": "dupuser2",
			"password": "password456",
		}
		body, _ := json.Marshal(registerReq)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

// TestTokenValidation 测试token验证逻辑
func TestTokenValidation(t *testing.T) {
	authService := NewMockAuthService()

	t.Run("有效token验证", func(t *testing.T) {
		// 创建用户并登录
		user, _ := authService.Register("valid@example.com", "validuser", "password123")
		token, _, err := authService.Login("valid@example.com", "password123")
		require.NoError(t, err)

		// 验证token
		claims, err := authService.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, user.ID, claims.UserID)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.Role, claims.Role)
	})

	t.Run("无效token验证", func(t *testing.T) {
		invalidToken := "invalid.token.string"
		_, err := authService.ValidateToken(invalidToken)
		assert.Error(t, err)
	})

	t.Run("过期token验证", func(t *testing.T) {
		// 创建一个过期的token
		claims := &Claims{
			UserID: 1,
			Email:  "expired@example.com",
			Role:   "developer",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // 已过期
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
				NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
				Issuer:    "collaborative-dev-platform",
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString(jwtSecretKey)
		require.NoError(t, err)

		_, err = authService.ValidateToken(tokenString)
		assert.Error(t, err)
	})
}

// TestAuthServiceConcurrency 测试并发场景
func TestAuthServiceConcurrency(t *testing.T) {
	authService := NewMockAuthService()
	router := setupRouter(authService)

	t.Run("并发注册", func(t *testing.T) {
		const numGoroutines = 10
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer func() { done <- true }()

				registerReq := map[string]string{
					"email":    fmt.Sprintf("concurrent%d@example.com", id),
					"username": fmt.Sprintf("user%d", id),
					"password": "password123",
				}
				body, _ := json.Marshal(registerReq)

				w := httptest.NewRecorder()
				req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusCreated, w.Code)
			}(i)
		}

		// 等待所有goroutine完成
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// 验证所有用户都被成功创建
		assert.Equal(t, numGoroutines, len(authService.users))
	})

	t.Run("并发登录", func(t *testing.T) {
		// 先创建一个用户
		authService.Register("concurrent@example.com", "concurrentuser", "password123")

		const numGoroutines = 10
		tokens := make([]string, numGoroutines)
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer func() { done <- true }()

				loginReq := map[string]string{
					"email":    "concurrent@example.com",
					"password": "password123",
				}
				body, _ := json.Marshal(loginReq)

				w := httptest.NewRecorder()
				req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)

				var loginResp struct {
					Token string `json:"token"`
				}
				json.Unmarshal(w.Body.Bytes(), &loginResp)
				tokens[id] = loginResp.Token
			}(i)
		}

		// 等待所有goroutine完成
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// 验证所有生成的token都是有效的
		for _, token := range tokens {
			assert.NotEmpty(t, token)
			_, err := authService.ValidateToken(token)
			assert.NoError(t, err)
		}
	})
}

// BenchmarkAuthFlow 性能基准测试
func BenchmarkAuthFlow(b *testing.B) {
	authService := NewMockAuthService()
	router := setupRouter(authService)

	// 预先创建用户
	authService.Register("bench@example.com", "benchuser", "password123")

	b.Run("登录性能", func(b *testing.B) {
		loginReq := map[string]string{
			"email":    "bench@example.com",
			"password": "password123",
		}
		body, _ := json.Marshal(loginReq)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
		}
	})

	b.Run("Token验证性能", func(b *testing.B) {
		token, _, _ := authService.Login("bench@example.com", "password123")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			authService.ValidateToken(token)
		}
	})
}
