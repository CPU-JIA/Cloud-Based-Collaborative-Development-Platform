package middleware

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cloud-platform/collaborative-dev/shared/cache"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CacheMiddleware HTTP缓存中间件
type CacheMiddleware struct {
	cache  *cache.RedisClient
	logger *zap.Logger
	config CacheConfig
}

// CacheConfig 缓存配置
type CacheConfig struct {
	DefaultTTL     time.Duration
	MaxCacheSize   int64
	SkipAuth       bool // 是否跳过认证请求的缓存
	AllowedMethods []string
	ExcludedPaths  []string
	CacheKeyPrefix string
}

// DefaultCacheConfig 默认缓存配置
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		DefaultTTL:     5 * time.Minute,
		MaxCacheSize:   1024 * 1024, // 1MB
		SkipAuth:       true,
		AllowedMethods: []string{"GET"},
		ExcludedPaths: []string{
			"/health",
			"/metrics",
			"/ws",
			"/download",
			"/upload",
		},
		CacheKeyPrefix: "http:",
	}
}

// NewCacheMiddleware 创建缓存中间件
func NewCacheMiddleware(cache *cache.RedisClient, logger *zap.Logger, config CacheConfig) *CacheMiddleware {
	return &CacheMiddleware{
		cache:  cache,
		logger: logger,
		config: config,
	}
}

// responseWriter 自定义响应写入器，用于捕获响应
type responseWriter struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// CachedResponse 缓存的响应
type CachedResponse struct {
	Status  int                 `json:"status"`
	Headers map[string][]string `json:"headers"`
	Body    []byte              `json:"body"`
}

// Handle 处理请求缓存
func (cm *CacheMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否应该缓存
		if !cm.shouldCache(c) {
			c.Next()
			return
		}

		// 生成缓存键
		cacheKey := cm.generateCacheKey(c)

		// 尝试从缓存获取
		var cached CachedResponse
		ctx := c.Request.Context()

		err := cm.cache.Get(ctx, cacheKey, &cached)
		if err == nil {
			// 缓存命中
			cm.logger.Debug("Cache hit", zap.String("key", cacheKey))
			cm.serveFromCache(c, &cached)
			return
		}

		// 缓存未命中，继续处理请求
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBuffer(nil),
			status:         http.StatusOK,
		}
		c.Writer = writer

		c.Next()

		// 只缓存成功的响应
		if writer.status >= 200 && writer.status < 300 && writer.body.Len() > 0 {
			// 检查响应大小
			if int64(writer.body.Len()) <= cm.config.MaxCacheSize {
				cached := CachedResponse{
					Status:  writer.status,
					Headers: writer.Header(),
					Body:    writer.body.Bytes(),
				}

				// 异步保存到缓存
				go func() {
					if err := cm.cache.Set(context.Background(), cacheKey, cached, cm.config.DefaultTTL); err != nil {
						cm.logger.Warn("Failed to cache response",
							zap.String("key", cacheKey),
							zap.Error(err))
					}
				}()
			}
		}
	}
}

// shouldCache 判断是否应该缓存
func (cm *CacheMiddleware) shouldCache(c *gin.Context) bool {
	// 检查方法
	methodAllowed := false
	for _, method := range cm.config.AllowedMethods {
		if c.Request.Method == method {
			methodAllowed = true
			break
		}
	}
	if !methodAllowed {
		return false
	}

	// 检查排除路径
	path := c.Request.URL.Path
	for _, excluded := range cm.config.ExcludedPaths {
		if strings.HasPrefix(path, excluded) {
			return false
		}
	}

	// 检查认证
	if cm.config.SkipAuth && c.GetHeader("Authorization") != "" {
		return false
	}

	// 检查缓存控制头
	cacheControl := c.GetHeader("Cache-Control")
	if strings.Contains(cacheControl, "no-cache") || strings.Contains(cacheControl, "no-store") {
		return false
	}

	return true
}

// generateCacheKey 生成缓存键
func (cm *CacheMiddleware) generateCacheKey(c *gin.Context) string {
	// 基础键：方法 + 路径 + 查询参数
	key := fmt.Sprintf("%s:%s:%s:%s",
		cm.config.CacheKeyPrefix,
		c.Request.Method,
		c.Request.URL.Path,
		c.Request.URL.RawQuery)

	// 添加租户ID（如果存在）
	if tenantID, exists := c.Get("tenant_id"); exists {
		key += fmt.Sprintf(":tenant:%v", tenantID)
	}

	// 添加用户ID（如果不跳过认证）
	if !cm.config.SkipAuth {
		if userID, exists := c.Get("user_id"); exists {
			key += fmt.Sprintf(":user:%v", userID)
		}
	}

	// 添加Accept头
	accept := c.GetHeader("Accept")
	if accept != "" {
		key += fmt.Sprintf(":accept:%s", accept)
	}

	// 生成MD5哈希以缩短键长度
	hash := md5.Sum([]byte(key))
	return cm.config.CacheKeyPrefix + hex.EncodeToString(hash[:])
}

// serveFromCache 从缓存提供响应
func (cm *CacheMiddleware) serveFromCache(c *gin.Context, cached *CachedResponse) {
	// 设置响应头
	for key, values := range cached.Headers {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	// 添加缓存相关头
	c.Header("X-Cache", "HIT")
	c.Header("X-Cache-Key", cm.generateCacheKey(c))

	// 写入状态码和响应体
	c.Status(cached.Status)
	c.Writer.Write(cached.Body)
	c.Abort()
}

// InvalidateCache 使缓存失效
func (cm *CacheMiddleware) InvalidateCache(patterns ...string) error {
	_ = context.Background() // 预留给未来使用
	for _, pattern := range patterns {
		fullPattern := cm.config.CacheKeyPrefix + pattern
		// 这里应该实现基于模式的删除
		// 由于Redis的KEYS命令在生产环境不推荐使用，
		// 可以考虑使用SCAN或维护一个缓存键索引
		cm.logger.Info("Cache invalidation requested", zap.String("pattern", fullPattern))
	}
	return nil
}

// CacheInvalidator 缓存失效器接口
type CacheInvalidator interface {
	InvalidateUserCache(ctx context.Context, userID string) error
	InvalidateProjectCache(ctx context.Context, projectID string) error
	InvalidateTenantCache(ctx context.Context, tenantID string) error
}

// StandardCacheInvalidator 标准缓存失效器实现
type StandardCacheInvalidator struct {
	cache  *cache.CacheManager
	logger *zap.Logger
}

// NewStandardCacheInvalidator 创建标准缓存失效器
func NewStandardCacheInvalidator(cache *cache.CacheManager, logger *zap.Logger) *StandardCacheInvalidator {
	return &StandardCacheInvalidator{
		cache:  cache,
		logger: logger,
	}
}

// InvalidateUserCache 使用户相关缓存失效
func (sci *StandardCacheInvalidator) InvalidateUserCache(ctx context.Context, userID string) error {
	patterns := []string{
		fmt.Sprintf("*:user:%s:*", userID),
		fmt.Sprintf("*user:id:%s*", userID),
	}

	for _, pattern := range patterns {
		sci.logger.Debug("Invalidating user cache", zap.String("pattern", pattern))
	}

	return nil
}

// InvalidateProjectCache 使项目相关缓存失效
func (sci *StandardCacheInvalidator) InvalidateProjectCache(ctx context.Context, projectID string) error {
	patterns := []string{
		fmt.Sprintf("*:project:%s:*", projectID),
		fmt.Sprintf("*project:id:%s*", projectID),
	}

	for _, pattern := range patterns {
		sci.logger.Debug("Invalidating project cache", zap.String("pattern", pattern))
	}

	return nil
}

// InvalidateTenantCache 使租户相关缓存失效
func (sci *StandardCacheInvalidator) InvalidateTenantCache(ctx context.Context, tenantID string) error {
	patterns := []string{
		fmt.Sprintf("*:tenant:%s:*", tenantID),
		fmt.Sprintf("*tenant:id:%s*", tenantID),
	}

	for _, pattern := range patterns {
		sci.logger.Debug("Invalidating tenant cache", zap.String("pattern", pattern))
	}

	return nil
}
