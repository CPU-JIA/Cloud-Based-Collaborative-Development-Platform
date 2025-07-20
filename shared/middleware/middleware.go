package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/golang-jwt/jwt/v5"
	"github.com/cloud-platform/collaborative-dev/shared/logger"
)

// CORS 跨域中间件
func CORS(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// 检查是否在允许列表中
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}
		
		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Tenant-ID, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "Content-Length, X-Request-ID")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RequestID 请求ID中间件
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取或生成请求ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		
		// 设置到上下文和响应头
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		
		c.Next()
	}
}

// Logger 日志中间件
func Logger(log logger.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// 结构化日志
		fields := map[string]interface{}{
			"timestamp":     param.TimeStamp.Format(time.RFC3339),
			"status_code":   param.StatusCode,
			"latency":       param.Latency.String(),
			"client_ip":     param.ClientIP,
			"method":        param.Method,
			"path":          param.Path,
			"error_message": param.ErrorMessage,
			"body_size":     param.BodySize,
			"user_agent":    param.Request.UserAgent(),
		}
		
		// 添加请求ID
		if requestID := param.Keys["request_id"]; requestID != nil {
			fields["request_id"] = requestID
		}
		
		// 根据状态码选择日志级别
		switch {
		case param.StatusCode >= 500:
			log.WithFields(fields).Error("HTTP请求")
		case param.StatusCode >= 400:
			log.WithFields(fields).Warn("HTTP请求")
		default:
			log.WithFields(fields).Info("HTTP请求")
		}
		
		return ""
	})
}

// Recovery 恢复中间件
func Recovery(log logger.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		requestID := c.GetString("request_id")
		
		log.WithFields(map[string]interface{}{
			"request_id": requestID,
			"panic":      recovered,
			"path":       c.Request.URL.Path,
			"method":     c.Request.Method,
		}).Error("服务器内部错误")
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "内部服务器错误",
			"request_id": requestID,
		})
	})
}

// RateLimit 限流中间件配置
type RateLimitConfig struct {
	RequestsPerMinute int
	BurstSize        int
}

// TenantMiddleware 租户中间件
func TenantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取租户ID
		tenantID := c.GetHeader("X-Tenant-ID")
		if tenantID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "缺少租户ID",
			})
			c.Abort()
			return
		}
		
		// 验证租户ID格式
		if _, err := uuid.Parse(tenantID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "无效的租户ID格式",
			})
			c.Abort()
			return
		}
		
		// 设置到上下文
		c.Set("tenant_id", tenantID)
		c.Next()
	}
}

// JWTClaims JWT声明
type JWTClaims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// JWTAuth JWT认证中间件
func JWTAuth(secretKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取Authorization头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "缺少认证信息",
			})
			c.Abort()
			return
		}
		
		// 验证Bearer前缀
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "无效的认证格式",
			})
			c.Abort()
			return
		}
		
		// 提取token
		tokenString := authHeader[len(bearerPrefix):]
		
		// 解析和验证token
		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		})
		
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "无效的认证令牌",
			})
			c.Abort()
			return
		}
		
		// 验证token有效性
		if !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "认证令牌已过期",
			})
			c.Abort()
			return
		}
		
		// 获取声明
		claims, ok := token.Claims.(*JWTClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "无效的令牌声明",
			})
			c.Abort()
			return
		}
		
		// 设置用户信息到上下文
		c.Set("user_id", claims.UserID)
		c.Set("tenant_id", claims.TenantID)
		c.Set("user_role", claims.Role)
		
		c.Next()
	}
}

// RequireRole 角色验证中间件
func RequireRole(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "缺少角色信息",
			})
			c.Abort()
			return
		}
		
		role, ok := userRole.(string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "无效的角色信息",
			})
			c.Abort()
			return
		}
		
		// 检查角色权限
		hasPermission := false
		for _, requiredRole := range requiredRoles {
			if role == requiredRole {
				hasPermission = true
				break
			}
		}
		
		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "权限不足",
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// SecurityHeaders 安全头中间件
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 安全头设置
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		
		c.Next()
	}
}

// Timeout 超时中间件
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置超时上下文
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		
		c.Request = c.Request.WithContext(ctx)
		
		// 监听超时
		done := make(chan struct{})
		go func() {
			c.Next()
			done <- struct{}{}
		}()
		
		select {
		case <-done:
			return
		case <-ctx.Done():
			c.JSON(http.StatusRequestTimeout, gin.H{
				"error": "请求超时",
			})
			c.Abort()
		}
	}
}

// HealthCheck 健康检查中间件
func HealthCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/healthz" {
			c.JSON(http.StatusOK, gin.H{
				"status":    "ok",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"service":   "collaborative-platform",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}