# 性能优化实施指南

## 概述

本文档详细说明了 Cloud-Based Collaborative Development Platform 的性能优化实施方案，包括缓存层、数据库优化、文件处理优化和性能监控。

## 1. Redis缓存层实现

### 1.1 缓存架构

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   应用层     │ ──> │  缓存层      │ ──> │  数据库层    │
│  (Services) │     │   (Redis)   │     │ (PostgreSQL)│
└─────────────┘     └─────────────┘     └─────────────┘
```

### 1.2 缓存策略

#### 缓存键设计
```go
// 用户相关
KeyUserByID       = "user:id:%s"
KeyUserByEmail    = "user:email:%s"
KeyUserPermissions = "user:permissions:%s"

// 项目相关
KeyProjectByID    = "project:id:%s"
KeyProjectList    = "project:list:tenant:%s:page:%d:size:%d"
KeyProjectMembers = "project:members:%s"

// 文件相关
KeyFileMetadata   = "file:metadata:%s"
KeyFileTree       = "file:tree:project:%s:path:%s"
```

#### TTL配置
```go
TTLUserCache       = 1 * time.Hour
TTLProjectCache    = 30 * time.Minute
TTLFileMetadata    = 1 * time.Hour
TTLListCache       = 5 * time.Minute
TTLPermissionCache = 2 * time.Hour
```

### 1.3 使用示例

```go
// GetOrSet 模式
err := cacheManager.GetOrSet(ctx, cacheKey, &project, TTLProjectCache, 
    func() (interface{}, error) {
        // 从数据库获取数据
        var p Project
        err := db.Where("id = ?", projectID).First(&p).Error
        return p, err
    })
```

## 2. 数据库优化

### 2.1 索引优化

已创建的关键索引：

```sql
-- 用户表索引
CREATE UNIQUE INDEX idx_users_tenant_email ON users(tenant_id, email);
CREATE INDEX idx_users_tenant_active ON users(tenant_id, is_active);
CREATE INDEX idx_users_last_login ON users(last_login_at);

-- 项目表索引
CREATE INDEX idx_projects_tenant_status ON projects(tenant_id, status);
CREATE UNIQUE INDEX idx_projects_tenant_key ON projects(tenant_id, key);
CREATE INDEX idx_projects_created_at ON projects(created_at);

-- 部分索引（PostgreSQL特性）
CREATE INDEX idx_users_active_verified ON users(tenant_id, is_email_verified) 
WHERE is_active = true AND deleted_at IS NULL;

CREATE INDEX idx_projects_active ON projects(tenant_id, created_at) 
WHERE status = 'active' AND deleted_at IS NULL;
```

### 2.2 查询优化

#### 预加载防止N+1查询
```go
// 优化前（N+1查询）
projects := []Project{}
db.Find(&projects)
for _, p := range projects {
    db.Where("project_id = ?", p.ID).Find(&p.Members)
}

// 优化后（预加载）
db.Preload("Members.User").
   Preload("Members.Role").
   Find(&projects)
```

#### 只选择需要的字段
```go
db.Select("id", "name", "status", "created_at").
   Where("tenant_id = ?", tenantID).
   Find(&projects)
```

### 2.3 连接池配置

```go
// 生产环境配置
MaxOpenConns:    100,  // 最大开放连接数
MaxIdleConns:    25,   // 最大空闲连接数
ConnMaxLifetime: 10 * time.Minute,
ConnMaxIdleTime: 2 * time.Minute,
```

## 3. 文件上传/下载优化

### 3.1 流式处理

```go
// 使用流式上传，避免内存占用
func (h *OptimizedFileHandler) UploadFileStream(c *gin.Context) {
    reader, _ := c.Request.MultipartReader()
    
    // 使用缓冲池
    buffer := h.bufferPool.Get().([]byte)
    defer h.bufferPool.Put(buffer)
    
    // 流式处理每个文件
    for {
        part, err := reader.NextPart()
        // 处理文件...
    }
}
```

### 3.2 断点续传支持

```go
// 支持Range请求
rangeHeader := c.GetHeader("Range")
if rangeHeader != "" {
    h.handleRangeRequest(c, file, rangeHeader)
}
```

### 3.3 文件去重

基于文件哈希的去重机制：
```go
// 检查重复文件
var existingFile File
if err := db.Where("hash = ?", file.Hash).First(&existingFile).Error; err == nil {
    // 创建硬链接而不是重复存储
    file.StoragePath = existingFile.StoragePath
}
```

## 4. 性能监控

### 4.1 监控指标

```go
type MetricsCollector struct {
    requestDuration   *DurationMetric    // HTTP请求时长
    requestCount      *CounterMetric     // 请求计数
    dbQueryDuration   *DurationMetric    // 数据库查询时长
    cacheHitRate      *RateMetric        // 缓存命中率
    activeConnections *GaugeMetric       // 活动连接数
}
```

### 4.2 性能中间件

```go
// HTTP性能监控中间件
func (pm *PerformanceMonitor) HTTPMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        
        duration := time.Since(start)
        // 记录慢请求
        if duration > 1*time.Second {
            logger.Warn("Slow request detected", 
                zap.Duration("duration", duration))
        }
    }
}
```

## 5. 初始化和配置

### 5.1 运行性能优化脚本

```bash
# 编译性能初始化工具
go build -o init_performance ./scripts/init_performance.go

# 运行性能优化
./init_performance
```

### 5.2 配置文件示例

```yaml
# config.yaml
redis:
  host: localhost
  port: 6379
  pool_size: 10
  dial_timeout: 5s
  read_timeout: 3s
  write_timeout: 3s

database:
  max_open_conns: 100
  max_idle_conns: 25
  conn_max_lifetime: 10m
  conn_max_idle_time: 2m

cache:
  default_ttl: 5m
  max_cache_size: 1048576
  allowed_methods: ["GET"]
  excluded_paths: ["/health", "/metrics", "/ws"]
```

## 6. 性能测试

### 6.1 运行基准测试

```bash
# 运行所有性能测试
go test -bench=. ./test/performance/

# 运行特定测试
go test -bench=BenchmarkProjectList ./test/performance/

# 运行负载测试
go test -run TestLoadTest -v ./test/performance/
```

### 6.2 预期性能指标

- **缓存命中率**: > 80%
- **平均响应时间**: < 100ms
- **P95响应时间**: < 200ms
- **P99响应时间**: < 500ms
- **成功率**: > 99%

## 7. 最佳实践

### 7.1 缓存使用

1. **缓存粒度**: 使用细粒度缓存，避免缓存整个大对象
2. **缓存失效**: 在数据更新时主动失效相关缓存
3. **缓存预热**: 系统启动时预热热点数据

### 7.2 数据库优化

1. **批量操作**: 使用批量插入/更新减少数据库交互
2. **索引维护**: 定期运行 VACUUM ANALYZE
3. **查询优化**: 使用 EXPLAIN ANALYZE 分析慢查询

### 7.3 并发控制

1. **连接池**: 合理配置数据库和Redis连接池大小
2. **限流**: 实现API级别的限流保护
3. **异步处理**: 将耗时操作异步化

## 8. 监控和告警

### 8.1 关键指标监控

- 响应时间（平均值、P95、P99）
- 错误率
- 缓存命中率
- 数据库连接池使用率
- 内存和CPU使用率

### 8.2 告警阈值

- 响应时间 > 1秒
- 错误率 > 1%
- 缓存命中率 < 60%
- 数据库连接池使用率 > 80%

## 9. 故障排查

### 9.1 性能问题排查步骤

1. 检查监控指标，定位性能瓶颈
2. 分析慢查询日志
3. 检查缓存命中率
4. 分析数据库执行计划
5. 检查资源使用情况

### 9.2 常见问题

1. **缓存雪崩**: 使用随机TTL避免大量缓存同时失效
2. **缓存击穿**: 使用分布式锁防止并发查询数据库
3. **慢查询**: 添加必要索引，优化查询语句

## 10. 扩展计划

### 10.1 短期优化

- [ ] 实现查询结果的二级缓存
- [ ] 添加数据库读写分离
- [ ] 实现自动化的慢查询优化

### 10.2 长期规划

- [ ] 引入 CDN 加速静态资源
- [ ] 实现分布式缓存集群
- [ ] 数据库分片和分区
- [ ] 引入消息队列处理异步任务