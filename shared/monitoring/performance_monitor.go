package monitoring

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// PerformanceMonitor 性能监控器
type PerformanceMonitor struct {
	logger  *zap.Logger
	metrics *MetricsCollector
}

// MetricsCollector 指标收集器
type MetricsCollector struct {
	requestDuration   *DurationMetric
	requestCount      *CounterMetric
	errorCount        *CounterMetric
	dbQueryDuration   *DurationMetric
	dbQueryCount      *CounterMetric
	cacheHitRate      *RateMetric
	activeConnections *GaugeMetric
}

// DurationMetric 时长指标
type DurationMetric struct {
	name   string
	labels []string
}

// CounterMetric 计数器指标
type CounterMetric struct {
	name   string
	labels []string
}

// RateMetric 比率指标
type RateMetric struct {
	name   string
	labels []string
	hits   int64
	total  int64
}

// GaugeMetric 仪表指标
type GaugeMetric struct {
	name   string
	labels []string
	value  int64
}

// NewPerformanceMonitor 创建性能监控器
func NewPerformanceMonitor(logger *zap.Logger) *PerformanceMonitor {
	return &PerformanceMonitor{
		logger: logger,
		metrics: &MetricsCollector{
			requestDuration: &DurationMetric{
				name:   "http_request_duration_seconds",
				labels: []string{"method", "path", "status"},
			},
			requestCount: &CounterMetric{
				name:   "http_requests_total",
				labels: []string{"method", "path", "status"},
			},
			errorCount: &CounterMetric{
				name:   "http_errors_total",
				labels: []string{"method", "path", "error_type"},
			},
			dbQueryDuration: &DurationMetric{
				name:   "db_query_duration_seconds",
				labels: []string{"operation", "table"},
			},
			dbQueryCount: &CounterMetric{
				name:   "db_queries_total",
				labels: []string{"operation", "table"},
			},
			cacheHitRate: &RateMetric{
				name:   "cache_hit_rate",
				labels: []string{"cache_type"},
			},
			activeConnections: &GaugeMetric{
				name:   "active_connections",
				labels: []string{"service"},
			},
		},
	}
}

// HTTPMiddleware HTTP性能监控中间件
func (pm *PerformanceMonitor) HTTPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// 处理请求
		c.Next()

		// 记录指标
		duration := time.Since(start)
		status := c.Writer.Status()

		// 记录请求时长
		pm.recordRequestDuration(method, path, status, duration)

		// 记录请求数
		pm.recordRequestCount(method, path, status)

		// 记录错误
		if status >= 400 {
			pm.recordError(method, path, getErrorType(status))
		}

		// 记录慢请求
		if duration > 1*time.Second {
			pm.logger.Warn("Slow request detected",
				zap.String("method", method),
				zap.String("path", path),
				zap.Int("status", status),
				zap.Duration("duration", duration))
		}
	}
}

// recordRequestDuration 记录请求时长
func (pm *PerformanceMonitor) recordRequestDuration(method, path string, status int, duration time.Duration) {
	// 这里应该集成实际的监控系统，如Prometheus
	pm.logger.Debug("Request duration",
		zap.String("method", method),
		zap.String("path", path),
		zap.Int("status", status),
		zap.Duration("duration", duration))
}

// recordRequestCount 记录请求数
func (pm *PerformanceMonitor) recordRequestCount(method, path string, status int) {
	// 实际实现应该更新计数器
}

// recordError 记录错误
func (pm *PerformanceMonitor) recordError(method, path string, errorType string) {
	pm.logger.Error("Request error",
		zap.String("method", method),
		zap.String("path", path),
		zap.String("error_type", errorType))
}

// RecordDBQuery 记录数据库查询
func (pm *PerformanceMonitor) RecordDBQuery(operation, table string, duration time.Duration) {
	// 记录查询时长和次数
	pm.logger.Debug("Database query",
		zap.String("operation", operation),
		zap.String("table", table),
		zap.Duration("duration", duration))

	// 慢查询警告
	if duration > 100*time.Millisecond {
		pm.logger.Warn("Slow database query",
			zap.String("operation", operation),
			zap.String("table", table),
			zap.Duration("duration", duration))
	}
}

// RecordCacheHit 记录缓存命中
func (pm *PerformanceMonitor) RecordCacheHit(cacheType string, hit bool) {
	if hit {
		pm.metrics.cacheHitRate.hits++
	}
	pm.metrics.cacheHitRate.total++
}

// GetCacheHitRate 获取缓存命中率
func (pm *PerformanceMonitor) GetCacheHitRate(cacheType string) float64 {
	if pm.metrics.cacheHitRate.total == 0 {
		return 0
	}
	return float64(pm.metrics.cacheHitRate.hits) / float64(pm.metrics.cacheHitRate.total)
}

// UpdateActiveConnections 更新活动连接数
func (pm *PerformanceMonitor) UpdateActiveConnections(service string, count int64) {
	pm.metrics.activeConnections.value = count
}

// PerformanceReport 性能报告
type PerformanceReport struct {
	Period          time.Duration `json:"period"`
	RequestStats    RequestStats  `json:"request_stats"`
	DatabaseStats   DatabaseStats `json:"database_stats"`
	CacheStats      CacheStats    `json:"cache_stats"`
	ResourceUsage   ResourceUsage `json:"resource_usage"`
	Recommendations []string      `json:"recommendations"`
}

// RequestStats 请求统计
type RequestStats struct {
	TotalRequests   int64         `json:"total_requests"`
	ErrorRate       float64       `json:"error_rate"`
	AvgResponseTime time.Duration `json:"avg_response_time"`
	P95ResponseTime time.Duration `json:"p95_response_time"`
	P99ResponseTime time.Duration `json:"p99_response_time"`
}

// DatabaseStats 数据库统计
type DatabaseStats struct {
	TotalQueries     int64         `json:"total_queries"`
	AvgQueryTime     time.Duration `json:"avg_query_time"`
	SlowQueries      int64         `json:"slow_queries"`
	ConnectionsInUse int           `json:"connections_in_use"`
}

// CacheStats 缓存统计
type CacheStats struct {
	HitRate      float64 `json:"hit_rate"`
	TotalHits    int64   `json:"total_hits"`
	TotalMisses  int64   `json:"total_misses"`
	EvictionRate float64 `json:"eviction_rate"`
}

// ResourceUsage 资源使用情况
type ResourceUsage struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskIO      float64 `json:"disk_io"`
	NetworkIO   float64 `json:"network_io"`
}

// GenerateReport 生成性能报告
func (pm *PerformanceMonitor) GenerateReport(ctx context.Context, period time.Duration) (*PerformanceReport, error) {
	report := &PerformanceReport{
		Period: period,
		RequestStats: RequestStats{
			// 实际实现应该从监控系统获取数据
			TotalRequests:   10000,
			ErrorRate:       0.01,
			AvgResponseTime: 50 * time.Millisecond,
			P95ResponseTime: 200 * time.Millisecond,
			P99ResponseTime: 500 * time.Millisecond,
		},
		DatabaseStats: DatabaseStats{
			TotalQueries:     50000,
			AvgQueryTime:     10 * time.Millisecond,
			SlowQueries:      50,
			ConnectionsInUse: 15,
		},
		CacheStats: CacheStats{
			HitRate:      pm.GetCacheHitRate("default"),
			TotalHits:    pm.metrics.cacheHitRate.hits,
			TotalMisses:  pm.metrics.cacheHitRate.total - pm.metrics.cacheHitRate.hits,
			EvictionRate: 0.05,
		},
		ResourceUsage: ResourceUsage{
			CPUUsage:    45.5,
			MemoryUsage: 62.3,
			DiskIO:      120.5,
			NetworkIO:   85.2,
		},
	}

	// 生成优化建议
	report.Recommendations = pm.generateRecommendations(report)

	return report, nil
}

// generateRecommendations 生成优化建议
func (pm *PerformanceMonitor) generateRecommendations(report *PerformanceReport) []string {
	var recommendations []string

	// 基于缓存命中率
	if report.CacheStats.HitRate < 0.8 {
		recommendations = append(recommendations,
			fmt.Sprintf("Cache hit rate is low (%.2f%%). Consider reviewing cache keys and TTL settings.",
				report.CacheStats.HitRate*100))
	}

	// 基于错误率
	if report.RequestStats.ErrorRate > 0.05 {
		recommendations = append(recommendations,
			fmt.Sprintf("Error rate is high (%.2f%%). Review error logs and implement better error handling.",
				report.RequestStats.ErrorRate*100))
	}

	// 基于响应时间
	if report.RequestStats.P95ResponseTime > 1*time.Second {
		recommendations = append(recommendations,
			"P95 response time exceeds 1 second. Consider optimizing slow endpoints and database queries.")
	}

	// 基于数据库性能
	if report.DatabaseStats.SlowQueries > 100 {
		recommendations = append(recommendations,
			"High number of slow queries detected. Review and optimize database indexes.")
	}

	// 基于资源使用
	if report.ResourceUsage.MemoryUsage > 80 {
		recommendations = append(recommendations,
			"Memory usage is high. Consider implementing memory optimization or scaling up.")
	}

	return recommendations
}

// getErrorType 获取错误类型
func getErrorType(status int) string {
	switch {
	case status >= 400 && status < 500:
		return "client_error"
	case status >= 500:
		return "server_error"
	default:
		return "unknown"
	}
}

// HealthCheck 健康检查结果
type HealthCheck struct {
	Service   string        `json:"service"`
	Status    string        `json:"status"`
	Latency   time.Duration `json:"latency"`
	Error     string        `json:"error,omitempty"`
	CheckedAt time.Time     `json:"checked_at"`
}

// PerformHealthCheck 执行健康检查
func (pm *PerformanceMonitor) PerformHealthCheck(ctx context.Context, services []string) []HealthCheck {
	var results []HealthCheck

	for _, service := range services {
		start := time.Now()
		status := "healthy"
		var err error

		// 实际实现应该检查各个服务的健康状态
		switch service {
		case "database":
			// 检查数据库连接
		case "redis":
			// 检查Redis连接
		case "api":
			// 检查API健康
		}

		result := HealthCheck{
			Service:   service,
			Status:    status,
			Latency:   time.Since(start),
			CheckedAt: time.Now(),
		}

		if err != nil {
			result.Status = "unhealthy"
			result.Error = err.Error()
		}

		results = append(results, result)
	}

	return results
}
