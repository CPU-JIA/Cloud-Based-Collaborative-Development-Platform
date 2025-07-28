package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// RateLimiter 限流器接口
type RateLimiter interface {
	Allow(key string) bool
	Reset(key string)
	GetRemaining(key string) int
}

// TokenBucketLimiter 令牌桶限流器
type TokenBucketLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	cleanup  time.Duration
	lastSeen map[string]time.Time
}

// NewTokenBucketLimiter 创建令牌桶限流器
func NewTokenBucketLimiter(rps rate.Limit, burst int, cleanup time.Duration) *TokenBucketLimiter {
	limiter := &TokenBucketLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rps,
		burst:    burst,
		cleanup:  cleanup,
		lastSeen: make(map[string]time.Time),
	}

	// 启动清理协程
	go limiter.cleanupRoutine()

	return limiter
}

// Allow 检查是否允许请求
func (l *TokenBucketLimiter) Allow(key string) bool {
	l.mu.RLock()
	limiter, exists := l.limiters[key]
	l.mu.RUnlock()

	if !exists {
		l.mu.Lock()
		// 双重检查
		limiter, exists = l.limiters[key]
		if !exists {
			limiter = rate.NewLimiter(l.rate, l.burst)
			l.limiters[key] = limiter
		}
		l.lastSeen[key] = time.Now()
		l.mu.Unlock()
	} else {
		l.mu.Lock()
		l.lastSeen[key] = time.Now()
		l.mu.Unlock()
	}

	return limiter.Allow()
}

// Reset 重置限流器
func (l *TokenBucketLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if limiter, exists := l.limiters[key]; exists {
		// 创建新的限流器替换旧的
		l.limiters[key] = rate.NewLimiter(l.rate, l.burst)
		// 重置令牌桶
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
		defer cancel()
		limiter.WaitN(ctx, l.burst)
	}
}

// GetRemaining 获取剩余请求数
func (l *TokenBucketLimiter) GetRemaining(key string) int {
	l.mu.RLock()
	limiter, exists := l.limiters[key]
	l.mu.RUnlock()

	if !exists {
		return l.burst
	}

	// 估算剩余令牌数
	tokens := limiter.Tokens()
	if tokens > float64(l.burst) {
		return l.burst
	}
	return int(tokens)
}

// cleanupRoutine 定期清理过期的限流器
func (l *TokenBucketLimiter) cleanupRoutine() {
	ticker := time.NewTicker(l.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		for key, lastSeen := range l.lastSeen {
			if now.Sub(lastSeen) > l.cleanup {
				delete(l.limiters, key)
				delete(l.lastSeen, key)
			}
		}
		l.mu.Unlock()
	}
}

// 注意：IPBlacklist 和 BlacklistEntry 已移动到 ip_filter.go 文件中

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	// 全局限流
	GlobalRPS   rate.Limit `json:"global_rps" yaml:"global_rps"`
	GlobalBurst int        `json:"global_burst" yaml:"global_burst"`

	// 按IP限流
	PerIPRPS   rate.Limit `json:"per_ip_rps" yaml:"per_ip_rps"`
	PerIPBurst int        `json:"per_ip_burst" yaml:"per_ip_burst"`

	// 按端点限流
	EndpointLimits map[string]EndpointLimit `json:"endpoint_limits" yaml:"endpoint_limits"`

	// 清理配置
	CleanupInterval time.Duration `json:"cleanup_interval" yaml:"cleanup_interval"`

	// 自动封禁配置
	AutoBanEnabled   bool          `json:"auto_ban_enabled" yaml:"auto_ban_enabled"`
	AutoBanThreshold int           `json:"auto_ban_threshold" yaml:"auto_ban_threshold"` // 超出限制次数
	AutoBanWindow    time.Duration `json:"auto_ban_window" yaml:"auto_ban_window"`       // 统计窗口
	AutoBanDuration  time.Duration `json:"auto_ban_duration" yaml:"auto_ban_duration"`   // 封禁时长
}

// EndpointLimit 端点限流配置
type EndpointLimit struct {
	RPS   rate.Limit `json:"rps" yaml:"rps"`
	Burst int        `json:"burst" yaml:"burst"`
}

// RateLimitMiddleware 综合限流中间件
type RateLimitMiddleware struct {
	config           *RateLimitConfig
	globalLimiter    *TokenBucketLimiter
	ipLimiter        *TokenBucketLimiter
	endpointLimiters map[string]*TokenBucketLimiter
	blacklist        *IPBlacklist
	violations       map[string]*ViolationTracker
	violationMu      sync.RWMutex
	logger           *zap.Logger
}

// ViolationTracker 违规追踪器
type ViolationTracker struct {
	Count     int       `json:"count"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

// NewRateLimitMiddleware 创建限流中间件
func NewRateLimitMiddleware(config *RateLimitConfig, logger *zap.Logger) *RateLimitMiddleware {
	if config == nil {
		config = getDefaultRateLimitConfig()
	}

	middleware := &RateLimitMiddleware{
		config:           config,
		globalLimiter:    NewTokenBucketLimiter(config.GlobalRPS, config.GlobalBurst, config.CleanupInterval),
		ipLimiter:        NewTokenBucketLimiter(config.PerIPRPS, config.PerIPBurst, config.CleanupInterval),
		endpointLimiters: make(map[string]*TokenBucketLimiter),
		blacklist:        NewIPBlacklist(logger),
		violations:       make(map[string]*ViolationTracker),
		logger:           logger,
	}

	// 初始化端点限流器
	for endpoint, limit := range config.EndpointLimits {
		middleware.endpointLimiters[endpoint] = NewTokenBucketLimiter(
			limit.RPS, limit.Burst, config.CleanupInterval)
	}

	// 启动违规清理协程
	if config.AutoBanEnabled {
		go middleware.cleanupViolations()
	}

	return middleware
}

// Handler 返回Gin中间件
func (m *RateLimitMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		path := c.Request.URL.Path
		method := c.Request.Method
		endpoint := fmt.Sprintf("%s %s", method, path)

		// 1. 检查IP黑名单
		if m.blacklist.IsBlacklisted(clientIP) {
			m.logger.Warn("阻止黑名单IP访问",
				zap.String("ip", clientIP),
				zap.String("path", path))

			c.JSON(http.StatusForbidden, gin.H{
				"error":   "访问被拒绝",
				"code":    "IP_BLACKLISTED",
				"message": "您的IP地址已被加入黑名单",
			})
			c.Abort()
			return
		}

		// 2. 全局限流检查
		if !m.globalLimiter.Allow("global") {
			m.handleRateLimit(c, clientIP, "global", "全局请求频率过高")
			return
		}

		// 3. 按IP限流检查
		if !m.ipLimiter.Allow(clientIP) {
			m.handleRateLimit(c, clientIP, "ip", "IP请求频率过高")
			return
		}

		// 4. 端点限流检查
		if limiter, exists := m.endpointLimiters[endpoint]; exists {
			if !limiter.Allow(clientIP) {
				m.handleRateLimit(c, clientIP, "endpoint", fmt.Sprintf("端点 %s 请求频率过高", endpoint))
				return
			}
		}

		// 添加限流头部信息
		m.addRateLimitHeaders(c, clientIP)

		c.Next()
	}
}

// handleRateLimit 处理限流情况
func (m *RateLimitMiddleware) handleRateLimit(c *gin.Context, clientIP, limitType, message string) {
	m.logger.Warn("请求被限流",
		zap.String("ip", clientIP),
		zap.String("path", c.Request.URL.Path),
		zap.String("limit_type", limitType),
		zap.String("user_agent", c.Request.UserAgent()))

	// 记录违规行为
	if m.config.AutoBanEnabled {
		m.recordViolation(clientIP)
	}

	c.JSON(http.StatusTooManyRequests, gin.H{
		"error":       "请求过于频繁",
		"code":        "RATE_LIMITED",
		"message":     message,
		"retry_after": "60",
	})
	c.Abort()
}

// recordViolation 记录违规行为
func (m *RateLimitMiddleware) recordViolation(ip string) {
	m.violationMu.Lock()
	defer m.violationMu.Unlock()

	now := time.Now()
	violation, exists := m.violations[ip]

	if !exists {
		m.violations[ip] = &ViolationTracker{
			Count:     1,
			FirstSeen: now,
			LastSeen:  now,
		}
		return
	}

	// 检查是否在时间窗口内
	if now.Sub(violation.FirstSeen) > m.config.AutoBanWindow {
		// 重置计数器
		violation.Count = 1
		violation.FirstSeen = now
	} else {
		violation.Count++
	}
	violation.LastSeen = now

	// 检查是否达到自动封禁阈值
	if violation.Count >= m.config.AutoBanThreshold {
		m.blacklist.AddToBlacklist(ip,
			fmt.Sprintf("自动封禁: %d次违规", violation.Count),
			m.config.AutoBanDuration)

		// 清除违规记录
		delete(m.violations, ip)

		m.logger.Warn("IP被自动封禁",
			zap.String("ip", ip),
			zap.Int("violations", violation.Count),
			zap.Duration("ban_duration", m.config.AutoBanDuration))
	}
}

// addRateLimitHeaders 添加限流头部信息
func (m *RateLimitMiddleware) addRateLimitHeaders(c *gin.Context, clientIP string) {
	// 添加剩余请求数信息
	remaining := m.ipLimiter.GetRemaining(clientIP)
	c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
	c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", m.config.PerIPBurst))

	// 重置时间（简化实现，固定60秒）
	resetTime := time.Now().Add(time.Minute).Unix()
	c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime))
}

// cleanupViolations 清理过期违规记录
func (m *RateLimitMiddleware) cleanupViolations() {
	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		m.violationMu.Lock()
		now := time.Now()
		for ip, violation := range m.violations {
			if now.Sub(violation.LastSeen) > m.config.AutoBanWindow*2 {
				delete(m.violations, ip)
			}
		}
		m.violationMu.Unlock()
	}
}

// GetBlacklist 获取黑名单管理器（用于外部管理）
func (m *RateLimitMiddleware) GetBlacklist() *IPBlacklist {
	return m.blacklist
}

// GetViolationStats 获取违规统计
func (m *RateLimitMiddleware) GetViolationStats() map[string]*ViolationTracker {
	m.violationMu.RLock()
	defer m.violationMu.RUnlock()

	stats := make(map[string]*ViolationTracker)
	for ip, violation := range m.violations {
		stats[ip] = &ViolationTracker{
			Count:     violation.Count,
			FirstSeen: violation.FirstSeen,
			LastSeen:  violation.LastSeen,
		}
	}

	return stats
}

// getDefaultRateLimitConfig 获取默认限流配置
func getDefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		GlobalRPS:   1000, // 每秒1000个请求
		GlobalBurst: 100,  // 突发100个请求

		PerIPRPS:   100, // 每个IP每秒100个请求
		PerIPBurst: 10,  // 每个IP突发10个请求

		EndpointLimits: map[string]EndpointLimit{
			"POST /api/v1/auth/login": {
				RPS:   10, // 登录接口每秒10次
				Burst: 2,  // 突发2次
			},
			"POST /api/v1/projects": {
				RPS:   50, // 创建项目每秒50次
				Burst: 5,  // 突发5次
			},
			"POST /api/v1/tasks": {
				RPS:   100, // 创建任务每秒100次
				Burst: 10,  // 突发10次
			},
		},

		CleanupInterval: 10 * time.Minute,

		AutoBanEnabled:   true,
		AutoBanThreshold: 5,               // 5次违规
		AutoBanWindow:    5 * time.Minute, // 5分钟窗口
		AutoBanDuration:  1 * time.Hour,   // 封禁1小时
	}
}
