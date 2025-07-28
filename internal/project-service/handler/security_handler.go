package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/cloud-platform/collaborative-dev/shared/middleware"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SecurityHandler 安全管理处理器
type SecurityHandler struct {
	rateLimitMiddleware *middleware.RateLimitMiddleware
	ipFilterMiddleware  *middleware.IPFilterMiddleware
	logger              *zap.Logger
}

// NewSecurityHandler 创建安全管理处理器
func NewSecurityHandler(
	rateLimitMiddleware *middleware.RateLimitMiddleware,
	ipFilterMiddleware *middleware.IPFilterMiddleware,
	logger *zap.Logger,
) *SecurityHandler {
	return &SecurityHandler{
		rateLimitMiddleware: rateLimitMiddleware,
		ipFilterMiddleware:  ipFilterMiddleware,
		logger:              logger,
	}
}

// GetSecurityStatus 获取安全状态
func (h *SecurityHandler) GetSecurityStatus(c *gin.Context) {
	// 获取黑名单统计
	blacklistEntries := h.rateLimitMiddleware.GetBlacklist().GetBlacklistEntries()

	// 获取白名单统计
	whitelistEntries := h.ipFilterMiddleware.GetWhitelist().GetWhitelistEntries()

	// 获取违规统计
	violationStats := h.rateLimitMiddleware.GetViolationStats()

	c.JSON(http.StatusOK, gin.H{
		"blacklist": gin.H{
			"total":   len(blacklistEntries),
			"entries": blacklistEntries,
		},
		"whitelist": gin.H{
			"total":   len(whitelistEntries),
			"entries": whitelistEntries,
		},
		"violations": gin.H{
			"active_ips": len(violationStats),
			"stats":      violationStats,
		},
		"status": "active",
	})
}

// AddToBlacklist 添加IP到黑名单
func (h *SecurityHandler) AddToBlacklist(c *gin.Context) {
	var req struct {
		IP       string `json:"ip" binding:"required"`
		Reason   string `json:"reason" binding:"required"`
		Duration string `json:"duration"` // 可选，格式如 "1h", "30m", "" 表示永久
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "无效的请求参数",
			"details": err.Error(),
		})
		return
	}

	// 解析持续时间
	var duration time.Duration
	var err error
	if req.Duration != "" {
		duration, err = time.ParseDuration(req.Duration)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "无效的持续时间格式",
				"details": "请使用如 '1h', '30m', '24h' 等格式",
			})
			return
		}
	}

	// 添加到黑名单
	h.rateLimitMiddleware.GetBlacklist().AddToBlacklist(req.IP, req.Reason, duration)

	h.logger.Info("管理员添加IP到黑名单",
		zap.String("ip", req.IP),
		zap.String("reason", req.Reason),
		zap.Duration("duration", duration),
		zap.String("admin_id", c.GetString("user_id")))

	c.JSON(http.StatusOK, gin.H{
		"message":  "IP已成功添加到黑名单",
		"ip":       req.IP,
		"reason":   req.Reason,
		"duration": req.Duration,
	})
}

// RemoveFromBlacklist 从黑名单移除IP
func (h *SecurityHandler) RemoveFromBlacklist(c *gin.Context) {
	ip := c.Param("ip")
	if ip == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "缺少IP参数",
		})
		return
	}

	h.rateLimitMiddleware.GetBlacklist().RemoveFromBlacklist(ip)

	h.logger.Info("管理员从黑名单移除IP",
		zap.String("ip", ip),
		zap.String("admin_id", c.GetString("user_id")))

	c.JSON(http.StatusOK, gin.H{
		"message": "IP已从黑名单移除",
		"ip":      ip,
	})
}

// AddToWhitelist 添加IP到白名单
func (h *SecurityHandler) AddToWhitelist(c *gin.Context) {
	var req struct {
		IP          string `json:"ip" binding:"required"`
		Description string `json:"description" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "无效的请求参数",
			"details": err.Error(),
		})
		return
	}

	adminID := c.GetString("user_id")
	if adminID == "" {
		adminID = "system"
	}

	// 添加到白名单
	err := h.ipFilterMiddleware.GetWhitelist().AddToWhitelist(req.IP, req.Description, adminID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "添加到白名单失败",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info("管理员添加IP到白名单",
		zap.String("ip", req.IP),
		zap.String("description", req.Description),
		zap.String("admin_id", adminID))

	c.JSON(http.StatusOK, gin.H{
		"message":     "IP已成功添加到白名单",
		"ip":          req.IP,
		"description": req.Description,
	})
}

// RemoveFromWhitelist 从白名单移除IP
func (h *SecurityHandler) RemoveFromWhitelist(c *gin.Context) {
	ip := c.Param("ip")
	if ip == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "缺少IP参数",
		})
		return
	}

	h.ipFilterMiddleware.GetWhitelist().RemoveFromWhitelist(ip)

	h.logger.Info("管理员从白名单移除IP",
		zap.String("ip", ip),
		zap.String("admin_id", c.GetString("user_id")))

	c.JSON(http.StatusOK, gin.H{
		"message": "IP已从白名单移除",
		"ip":      ip,
	})
}

// GetViolationStats 获取违规统计
func (h *SecurityHandler) GetViolationStats(c *gin.Context) {
	// 获取查询参数
	limit := 100 // 默认限制
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	stats := h.rateLimitMiddleware.GetViolationStats()

	// 转换为数组并限制数量
	result := make([]gin.H, 0, limit)
	count := 0

	for ip, violation := range stats {
		if count >= limit {
			break
		}

		result = append(result, gin.H{
			"ip":         ip,
			"count":      violation.Count,
			"first_seen": violation.FirstSeen,
			"last_seen":  violation.LastSeen,
		})
		count++
	}

	c.JSON(http.StatusOK, gin.H{
		"violations": result,
		"total":      len(stats),
		"limit":      limit,
	})
}

// BanIP 立即封禁IP
func (h *SecurityHandler) BanIP(c *gin.Context) {
	var req struct {
		IP       string `json:"ip" binding:"required"`
		Reason   string `json:"reason" binding:"required"`
		Duration string `json:"duration"` // 可选，默认1小时
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "无效的请求参数",
			"details": err.Error(),
		})
		return
	}

	// 解析持续时间，默认1小时
	duration := 1 * time.Hour
	if req.Duration != "" {
		var err error
		duration, err = time.ParseDuration(req.Duration)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "无效的持续时间格式",
				"details": "请使用如 '1h', '30m', '24h' 等格式",
			})
			return
		}
	}

	// 立即封禁
	h.rateLimitMiddleware.GetBlacklist().AddToBlacklist(req.IP,
		"管理员手动封禁: "+req.Reason, duration)

	h.logger.Warn("管理员手动封禁IP",
		zap.String("ip", req.IP),
		zap.String("reason", req.Reason),
		zap.Duration("duration", duration),
		zap.String("admin_id", c.GetString("user_id")))

	c.JSON(http.StatusOK, gin.H{
		"message":  "IP已被立即封禁",
		"ip":       req.IP,
		"reason":   req.Reason,
		"duration": req.Duration,
	})
}

// UnbanIP 解除IP封禁
func (h *SecurityHandler) UnbanIP(c *gin.Context) {
	ip := c.Param("ip")
	if ip == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "缺少IP参数",
		})
		return
	}

	h.rateLimitMiddleware.GetBlacklist().RemoveFromBlacklist(ip)

	h.logger.Info("管理员解除IP封禁",
		zap.String("ip", ip),
		zap.String("admin_id", c.GetString("user_id")))

	c.JSON(http.StatusOK, gin.H{
		"message": "IP封禁已解除",
		"ip":      ip,
	})
}

// GetSecurityMetrics 获取安全指标
func (h *SecurityHandler) GetSecurityMetrics(c *gin.Context) {
	blacklistEntries := h.rateLimitMiddleware.GetBlacklist().GetBlacklistEntries()
	whitelistEntries := h.ipFilterMiddleware.GetWhitelist().GetWhitelistEntries()
	violationStats := h.rateLimitMiddleware.GetViolationStats()

	// 统计不同类型的封禁
	permanentBans := 0
	temporaryBans := 0
	now := time.Now()

	for _, entry := range blacklistEntries {
		if entry.Permanent {
			permanentBans++
		} else if now.Before(entry.ExpiresAt) {
			temporaryBans++
		}
	}

	// 统计活跃违规
	activeViolations := 0
	for _, violation := range violationStats {
		if now.Sub(violation.LastSeen) < 10*time.Minute {
			activeViolations++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"metrics": gin.H{
			"blacklist": gin.H{
				"total":     len(blacklistEntries),
				"permanent": permanentBans,
				"temporary": temporaryBans,
			},
			"whitelist": gin.H{
				"total": len(whitelistEntries),
			},
			"violations": gin.H{
				"total":  len(violationStats),
				"active": activeViolations,
			},
		},
		"timestamp": now,
	})
}
