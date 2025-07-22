package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// IPWhitelist IP白名单管理器
type IPWhitelist struct {
	whitelist map[string]*WhitelistEntry
	mu        sync.RWMutex
	logger    *zap.Logger
}

// WhitelistEntry 白名单条目
type WhitelistEntry struct {
	IP          string    `json:"ip"`
	Description string    `json:"description"`
	AddedAt     time.Time `json:"added_at"`
	AddedBy     string    `json:"added_by"`
	IsNetwork   bool      `json:"is_network"`  // 是否为网络段
	Network     *net.IPNet `json:"-"`          // 网络段解析结果
}

// NewIPWhitelist 创建IP白名单管理器
func NewIPWhitelist(logger *zap.Logger) *IPWhitelist {
	return &IPWhitelist{
		whitelist: make(map[string]*WhitelistEntry),
		logger:    logger,
	}
}

// IsWhitelisted 检查IP是否在白名单中
func (wl *IPWhitelist) IsWhitelisted(ip string) bool {
	wl.mu.RLock()
	defer wl.mu.RUnlock()

	clientIP := net.ParseIP(ip)
	if clientIP == nil {
		wl.logger.Warn("无效IP地址", zap.String("ip", ip))
		return false
	}

	for _, entry := range wl.whitelist {
		if entry.IsNetwork {
			// 网络段匹配
			if entry.Network != nil && entry.Network.Contains(clientIP) {
				return true
			}
		} else {
			// 单个IP匹配
			if entry.IP == ip {
				return true
			}
		}
	}

	return false
}

// AddToWhitelist 添加IP到白名单
func (wl *IPWhitelist) AddToWhitelist(ip, description, addedBy string) error {
	wl.mu.Lock()
	defer wl.mu.Unlock()

	entry := &WhitelistEntry{
		IP:          ip,
		Description: description,
		AddedAt:     time.Now(),
		AddedBy:     addedBy,
	}

	// 检查是否为网络段
	if strings.Contains(ip, "/") {
		_, network, err := net.ParseCIDR(ip)
		if err != nil {
			wl.logger.Error("解析网络段失败", 
				zap.String("ip", ip), 
				zap.Error(err))
			return err
		}
		entry.IsNetwork = true
		entry.Network = network
	} else {
		// 验证单个IP地址
		if net.ParseIP(ip) == nil {
			wl.logger.Error("无效IP地址", zap.String("ip", ip))
			return &IPFilterError{Message: "无效的IP地址格式"}
		}
	}

	wl.whitelist[ip] = entry

	wl.logger.Info("IP已添加到白名单",
		zap.String("ip", ip),
		zap.String("description", description),
		zap.String("added_by", addedBy),
		zap.Bool("is_network", entry.IsNetwork))

	return nil
}

// RemoveFromWhitelist 从白名单移除IP
func (wl *IPWhitelist) RemoveFromWhitelist(ip string) {
	wl.mu.Lock()
	defer wl.mu.Unlock()

	if _, exists := wl.whitelist[ip]; exists {
		delete(wl.whitelist, ip)
		wl.logger.Info("IP已从白名单移除", zap.String("ip", ip))
	}
}

// GetWhitelistEntries 获取所有白名单条目
func (wl *IPWhitelist) GetWhitelistEntries() map[string]*WhitelistEntry {
	wl.mu.RLock()
	defer wl.mu.RUnlock()

	entries := make(map[string]*WhitelistEntry)
	for ip, entry := range wl.whitelist {
		entries[ip] = &WhitelistEntry{
			IP:          entry.IP,
			Description: entry.Description,
			AddedAt:     entry.AddedAt,
			AddedBy:     entry.AddedBy,
			IsNetwork:   entry.IsNetwork,
		}
	}

	return entries
}

// IPFilterConfig IP过滤配置
type IPFilterConfig struct {
	WhitelistEnabled bool     `json:"whitelist_enabled" yaml:"whitelist_enabled"`
	BlacklistEnabled bool     `json:"blacklist_enabled" yaml:"blacklist_enabled"`
	DefaultWhitelist []string `json:"default_whitelist" yaml:"default_whitelist"`
	TrustedProxies   []string `json:"trusted_proxies" yaml:"trusted_proxies"`
	LogBlocked       bool     `json:"log_blocked" yaml:"log_blocked"`
}

// IPFilterMiddleware IP过滤中间件
type IPFilterMiddleware struct {
	config    *IPFilterConfig
	whitelist *IPWhitelist
	blacklist *IPBlacklist
	logger    *zap.Logger
}

// NewIPFilterMiddleware 创建IP过滤中间件
func NewIPFilterMiddleware(config *IPFilterConfig, logger *zap.Logger) *IPFilterMiddleware {
	if config == nil {
		config = getDefaultIPFilterConfig()
	}

	middleware := &IPFilterMiddleware{
		config:    config,
		whitelist: NewIPWhitelist(logger),
		blacklist: NewIPBlacklist(logger),
		logger:    logger,
	}

	// 添加默认白名单
	for _, ip := range config.DefaultWhitelist {
		middleware.whitelist.AddToWhitelist(ip, "默认白名单", "system")
	}

	return middleware
}

// Handler 返回Gin中间件
func (m *IPFilterMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := m.getRealClientIP(c)

		// 如果启用白名单模式
		if m.config.WhitelistEnabled {
			if !m.whitelist.IsWhitelisted(clientIP) {
				m.handleBlocked(c, clientIP, "IP不在白名单中")
				return
			}
		}

		// 如果启用黑名单模式
		if m.config.BlacklistEnabled {
			if m.blacklist.IsBlacklisted(clientIP) {
				m.handleBlocked(c, clientIP, "IP在黑名单中")
				return
			}
		}

		c.Next()
	}
}

// getRealClientIP 获取真实客户端IP
func (m *IPFilterMiddleware) getRealClientIP(c *gin.Context) string {
	// 检查X-Forwarded-For头部（来自可信代理）
	xff := c.GetHeader("X-Forwarded-For")
	if xff != "" && m.isTrustedProxy(c.ClientIP()) {
		// 获取第一个IP（原始客户端IP）
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	// 检查X-Real-IP头部
	xri := c.GetHeader("X-Real-IP")
	if xri != "" && m.isTrustedProxy(c.ClientIP()) {
		if net.ParseIP(xri) != nil {
			return xri
		}
	}

	// 使用直接连接的IP
	return c.ClientIP()
}

// isTrustedProxy 检查是否为可信代理
func (m *IPFilterMiddleware) isTrustedProxy(ip string) bool {
	for _, trustedIP := range m.config.TrustedProxies {
		if strings.Contains(trustedIP, "/") {
			// 网络段匹配
			_, network, err := net.ParseCIDR(trustedIP)
			if err != nil {
				continue
			}
			clientIP := net.ParseIP(ip)
			if clientIP != nil && network.Contains(clientIP) {
				return true
			}
		} else {
			// 单个IP匹配
			if ip == trustedIP {
				return true
			}
		}
	}
	return false
}

// handleBlocked 处理被阻止的请求
func (m *IPFilterMiddleware) handleBlocked(c *gin.Context, clientIP, reason string) {
	if m.config.LogBlocked {
		m.logger.Warn("IP访问被阻止",
			zap.String("ip", clientIP),
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.String("reason", reason))
	}

	c.JSON(http.StatusForbidden, gin.H{
		"error":   "访问被拒绝",
		"code":    "IP_BLOCKED",
		"message": "您的IP地址无权访问此资源",
	})
	c.Abort()
}

// GetWhitelist 获取白名单管理器
func (m *IPFilterMiddleware) GetWhitelist() *IPWhitelist {
	return m.whitelist
}

// GetBlacklist 获取黑名单管理器
func (m *IPFilterMiddleware) GetBlacklist() *IPBlacklist {
	return m.blacklist
}

// IPFilterError IP过滤错误
type IPFilterError struct {
	Message string
}

func (e *IPFilterError) Error() string {
	return e.Message
}

// getDefaultIPFilterConfig 获取默认IP过滤配置
func getDefaultIPFilterConfig() *IPFilterConfig {
	return &IPFilterConfig{
		WhitelistEnabled: false,
		BlacklistEnabled: true,
		DefaultWhitelist: []string{
			"127.0.0.1",      // 本地回环
			"::1",            // IPv6本地回环
			"10.0.0.0/8",     // 私有网络A类
			"172.16.0.0/12",  // 私有网络B类
			"192.168.0.0/16", // 私有网络C类
		},
		TrustedProxies: []string{
			"127.0.0.1",
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
		},
		LogBlocked: true,
	}
}

// 高级IP过滤功能

// GeoIPFilter 地理位置IP过滤器（简化实现）
type GeoIPFilter struct {
	allowedCountries []string
	blockedCountries []string
	logger           *zap.Logger
}

// NewGeoIPFilter 创建地理位置IP过滤器
func NewGeoIPFilter(allowedCountries, blockedCountries []string, logger *zap.Logger) *GeoIPFilter {
	return &GeoIPFilter{
		allowedCountries: allowedCountries,
		blockedCountries: blockedCountries,
		logger:           logger,
	}
}

// IsAllowed 检查IP的地理位置是否被允许
func (gf *GeoIPFilter) IsAllowed(ip string) bool {
	// 这里需要集成GeoIP数据库（如MaxMind GeoIP2）
	// 简化实现，实际项目中需要使用真实的GeoIP库
	
	country := gf.getCountryByIP(ip)
	if country == "" {
		return true // 无法确定地理位置时允许访问
	}

	// 检查阻止列表
	for _, blocked := range gf.blockedCountries {
		if country == blocked {
			gf.logger.Warn("地理位置被阻止",
				zap.String("ip", ip),
				zap.String("country", country))
			return false
		}
	}

	// 如果有允许列表，检查是否在列表中
	if len(gf.allowedCountries) > 0 {
		for _, allowed := range gf.allowedCountries {
			if country == allowed {
				return true
			}
		}
		return false // 不在允许列表中
	}

	return true // 默认允许
}

// getCountryByIP 根据IP获取国家代码（简化实现）
func (gf *GeoIPFilter) getCountryByIP(ip string) string {
	// 实际实现需要集成GeoIP数据库
	// 这里返回空字符串作为占位符
	return ""
}

// ASNFilter ASN（自治系统号）过滤器
type ASNFilter struct {
	blockedASNs []string
	logger      *zap.Logger
}

// NewASNFilter 创建ASN过滤器
func NewASNFilter(blockedASNs []string, logger *zap.Logger) *ASNFilter {
	return &ASNFilter{
		blockedASNs: blockedASNs,
		logger:      logger,
	}
}

// IsAllowed 检查IP的ASN是否被允许
func (af *ASNFilter) IsAllowed(ip string) bool {
	asn := af.getASNByIP(ip)
	if asn == "" {
		return true // 无法确定ASN时允许访问
	}

	for _, blocked := range af.blockedASNs {
		if asn == blocked {
			af.logger.Warn("ASN被阻止",
				zap.String("ip", ip),
				zap.String("asn", asn))
			return false
		}
	}

	return true
}

// getASNByIP 根据IP获取ASN（简化实现）
func (af *ASNFilter) getASNByIP(ip string) string {
	// 实际实现需要集成ASN数据库
	return ""
}

// IPBlacklist IP黑名单管理器
type IPBlacklist struct {
	blacklist map[string]*BlacklistEntry
	mu        sync.RWMutex
	logger    *zap.Logger
}

// BlacklistEntry 黑名单条目
type BlacklistEntry struct {
	IP        string    `json:"ip"`
	Reason    string    `json:"reason"`
	ExpiresAt time.Time `json:"expires_at"`
	AddedAt   time.Time `json:"added_at"`
	Permanent bool      `json:"permanent"`
}

// NewIPBlacklist 创建IP黑名单管理器
func NewIPBlacklist(logger *zap.Logger) *IPBlacklist {
	bl := &IPBlacklist{
		blacklist: make(map[string]*BlacklistEntry),
		logger:    logger,
	}

	// 启动清理协程
	go bl.cleanupExpiredEntries()

	return bl
}

// IsBlacklisted 检查IP是否在黑名单中
func (bl *IPBlacklist) IsBlacklisted(ip string) bool {
	bl.mu.RLock()
	defer bl.mu.RUnlock()

	entry, exists := bl.blacklist[ip]
	if !exists {
		return false
	}

	// 检查是否过期
	if !entry.Permanent && time.Now().After(entry.ExpiresAt) {
		bl.mu.RUnlock()
		bl.mu.Lock()
		delete(bl.blacklist, ip)
		bl.mu.Unlock()
		bl.mu.RLock()
		return false
	}

	return true
}

// AddToBlacklist 添加IP到黑名单
func (bl *IPBlacklist) AddToBlacklist(ip, reason string, duration time.Duration) {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	entry := &BlacklistEntry{
		IP:        ip,
		Reason:    reason,
		AddedAt:   time.Now(),
		Permanent: duration <= 0,
	}

	if duration > 0 {
		entry.ExpiresAt = time.Now().Add(duration)
	}

	bl.blacklist[ip] = entry

	bl.logger.Warn("IP已添加到黑名单",
		zap.String("ip", ip),
		zap.String("reason", reason),
		zap.Bool("permanent", entry.Permanent),
		zap.Time("expires_at", entry.ExpiresAt))
}

// RemoveFromBlacklist 从黑名单移除IP
func (bl *IPBlacklist) RemoveFromBlacklist(ip string) {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	if _, exists := bl.blacklist[ip]; exists {
		delete(bl.blacklist, ip)
		bl.logger.Info("IP已从黑名单移除", zap.String("ip", ip))
	}
}

// GetBlacklistEntries 获取所有黑名单条目
func (bl *IPBlacklist) GetBlacklistEntries() map[string]*BlacklistEntry {
	bl.mu.RLock()
	defer bl.mu.RUnlock()

	entries := make(map[string]*BlacklistEntry)
	for ip, entry := range bl.blacklist {
		entries[ip] = &BlacklistEntry{
			IP:        entry.IP,
			Reason:    entry.Reason,
			ExpiresAt: entry.ExpiresAt,
			AddedAt:   entry.AddedAt,
			Permanent: entry.Permanent,
		}
	}

	return entries
}

// cleanupExpiredEntries 清理过期的黑名单条目
func (bl *IPBlacklist) cleanupExpiredEntries() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		bl.mu.Lock()
		now := time.Now()
		for ip, entry := range bl.blacklist {
			if !entry.Permanent && now.After(entry.ExpiresAt) {
				delete(bl.blacklist, ip)
				bl.logger.Info("黑名单条目已过期并移除", zap.String("ip", ip))
			}
		}
		bl.mu.Unlock()
	}
}