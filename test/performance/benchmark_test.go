package performance

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/cloud-platform/collaborative-dev/internal/iam-service"
	"github.com/cloud-platform/collaborative-dev/internal/project-service"
	"github.com/cloud-platform/collaborative-dev/shared/cache"
	"github.com/cloud-platform/collaborative-dev/shared/config"
	"github.com/cloud-platform/collaborative-dev/shared/database"
)

// BenchmarkSuite 性能测试套件
type BenchmarkSuite struct {
	db             *gorm.DB
	cache          *cache.CacheManager
	redis          *cache.RedisClient
	logger         *zap.Logger
	projectService *projectservice.OptimizedProjectService
	authService    *iamservice.OptimizedAuthService
	testTenantID   uuid.UUID
	testUsers      []uuid.UUID
	testProjects   []uuid.UUID
}

// SetupBenchmark 设置性能测试环境
func SetupBenchmark(t *testing.T) *BenchmarkSuite {
	// 初始化日志
	logger, _ := zap.NewDevelopment()

	// 加载测试配置
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Name:     "devcollab_test",
			User:     "postgres",
			Password: "postgres",
			SSLMode:  "disable",
		},
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			DB:       1,
			PoolSize: 10,
		},
		Auth: config.AuthConfig{
			JWTSecret:          "test-secret-key-for-benchmarking-only",
			JWTExpiration:      24 * time.Hour,
			RefreshTokenExpiry: 7 * 24 * time.Hour,
			MaxLoginAttempts:   5,
			LockoutDuration:    15 * time.Minute,
		},
	}

	// 初始化数据库
	dsn := cfg.GetDatabaseDSN()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	// 配置连接池
	poolConfig := database.ProductionPoolConfig()
	err = poolConfig.ApplyToGorm(db)
	require.NoError(t, err)

	// 初始化Redis
	redis, err := cache.NewRedisClient(&cfg.Redis, logger)
	require.NoError(t, err)

	// 创建缓存管理器
	cacheManager := cache.NewCacheManager(redis, logger)

	// 创建服务
	projectService := projectservice.NewOptimizedProjectService(db, cacheManager, logger)
	authService := iamservice.NewOptimizedAuthService(db, cacheManager, logger, &cfg.Auth)

	// 创建测试数据
	suite := &BenchmarkSuite{
		db:             db,
		cache:          cacheManager,
		redis:          redis,
		logger:         logger,
		projectService: projectService,
		authService:    authService,
		testTenantID:   uuid.New(),
	}

	suite.setupTestData(t)

	return suite
}

// setupTestData 创建测试数据
func (s *BenchmarkSuite) setupTestData(t *testing.T) {
	// 创建测试用户
	for i := 0; i < 100; i++ {
		userID := uuid.New()
		s.testUsers = append(s.testUsers, userID)

		// 这里应该创建实际的用户数据
		// 简化处理
	}

	// 创建测试项目
	for i := 0; i < 50; i++ {
		projectID := uuid.New()
		s.testProjects = append(s.testProjects, projectID)

		// 这里应该创建实际的项目数据
		// 简化处理
	}
}

// Cleanup 清理测试环境
func (s *BenchmarkSuite) Cleanup() {
	s.redis.FlushDB(context.Background())
	// 清理数据库测试数据
}

// BenchmarkProjectList 测试项目列表查询性能
func BenchmarkProjectList(b *testing.B) {
	suite := SetupBenchmark(&testing.T{})
	defer suite.Cleanup()

	ctx := context.Background()

	b.Run("WithoutCache", func(b *testing.B) {
		// 清空缓存
		suite.redis.FlushDB(ctx)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := suite.projectService.ListProjects(ctx, suite.testTenantID, 1, 20, nil)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("WithCache", func(b *testing.B) {
		// 预热缓存
		suite.projectService.ListProjects(ctx, suite.testTenantID, 1, 20, nil)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := suite.projectService.ListProjects(ctx, suite.testTenantID, 1, 20, nil)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkGetProject 测试获取单个项目性能
func BenchmarkGetProject(b *testing.B) {
	suite := SetupBenchmark(&testing.T{})
	defer suite.Cleanup()

	ctx := context.Background()
	projectID := suite.testProjects[0]

	b.Run("WithoutCache", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// 清除特定缓存
			cacheKey := fmt.Sprintf(cache.KeyProjectByID, projectID.String())
			suite.redis.Delete(ctx, cacheKey)

			_, err := suite.projectService.GetProject(ctx, projectID)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("WithCache", func(b *testing.B) {
		// 预热缓存
		suite.projectService.GetProject(ctx, projectID)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := suite.projectService.GetProject(ctx, projectID)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkConcurrentAccess 测试并发访问性能
func BenchmarkConcurrentAccess(b *testing.B) {
	suite := SetupBenchmark(&testing.T{})
	defer suite.Cleanup()

	ctx := context.Background()

	b.Run("10Concurrent", func(b *testing.B) {
		benchmarkConcurrent(b, suite, ctx, 10)
	})

	b.Run("50Concurrent", func(b *testing.B) {
		benchmarkConcurrent(b, suite, ctx, 50)
	})

	b.Run("100Concurrent", func(b *testing.B) {
		benchmarkConcurrent(b, suite, ctx, 100)
	})
}

func benchmarkConcurrent(b *testing.B, suite *BenchmarkSuite, ctx context.Context, concurrency int) {
	b.ResetTimer()

	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	for i := 0; i < b.N; i++ {
		wg.Add(1)
		sem <- struct{}{}

		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			// 随机选择操作
			switch rand.Intn(3) {
			case 0:
				// 获取项目列表
				suite.projectService.ListProjects(ctx, suite.testTenantID, 1, 20, nil)
			case 1:
				// 获取单个项目
				projectID := suite.testProjects[rand.Intn(len(suite.testProjects))]
				suite.projectService.GetProject(ctx, projectID)
			case 2:
				// 获取用户权限
				userID := suite.testUsers[rand.Intn(len(suite.testUsers))]
				suite.authService.GetUserPermissions(ctx, userID)
			}
		}()
	}

	wg.Wait()
}

// TestCacheHitRate 测试缓存命中率
func TestCacheHitRate(t *testing.T) {
	suite := SetupBenchmark(t)
	defer suite.Cleanup()

	ctx := context.Background()
	projectID := suite.testProjects[0]

	// 清空缓存
	suite.redis.FlushDB(ctx)

	// 记录初始状态
	initialStats, _ := suite.cache.GetStats(ctx)

	// 执行100次查询
	for i := 0; i < 100; i++ {
		_, err := suite.projectService.GetProject(ctx, projectID)
		require.NoError(t, err)
	}

	// 获取最终统计
	finalStats, _ := suite.cache.GetStats(ctx)

	// 计算命中率
	hits := finalStats.Hits - initialStats.Hits
	total := hits + (finalStats.Misses - initialStats.Misses)
	hitRate := float64(hits) / float64(total)

	t.Logf("Cache hit rate: %.2f%% (%d hits, %d total)", hitRate*100, hits, total)

	// 期望命中率应该很高（因为重复查询相同的项目）
	assert.Greater(t, hitRate, 0.95)
}

// TestQueryOptimization 测试查询优化效果
func TestQueryOptimization(t *testing.T) {
	suite := SetupBenchmark(t)
	defer suite.Cleanup()

	ctx := context.Background()

	// 创建性能优化器
	optimizer := database.NewPerformanceOptimizer(suite.db, suite.logger)

	// 分析慢查询（设置较低的阈值以捕获更多查询）
	stats, err := optimizer.AnalyzeSlowQueries(ctx, 10*time.Millisecond)
	if err != nil {
		t.Logf("Slow query analysis not available: %v", err)
		return
	}

	t.Logf("Slow queries found: %d", len(stats.SlowQueries))
	t.Logf("Total queries: %d", stats.QueryCount)
	t.Logf("Average query time: %v", stats.AvgQueryTime)

	// 显示前5个慢查询
	for i, sq := range stats.SlowQueries {
		if i >= 5 {
			break
		}
		t.Logf("Slow query %d: %s (duration: %v)", i+1, sq.SQL, sq.Duration)
	}
}

// BenchmarkMemoryUsage 测试内存使用情况
func BenchmarkMemoryUsage(b *testing.B) {
	suite := SetupBenchmark(&testing.T{})
	defer suite.Cleanup()

	ctx := context.Background()

	b.Run("LargeDataSet", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// 模拟大量数据查询
			var projects []interface{}
			for j := 0; j < 1000; j++ {
				project, _ := suite.projectService.GetProject(ctx, suite.testProjects[j%len(suite.testProjects)])
				projects = append(projects, project)
			}

			// 强制GC以获得更准确的内存使用情况
			if i%10 == 0 {
				runtime.GC()
			}
		}
	})
}

// LoadTestResult 负载测试结果
type LoadTestResult struct {
	TotalRequests   int
	SuccessRequests int
	FailedRequests  int
	TotalDuration   time.Duration
	AvgDuration     time.Duration
	MinDuration     time.Duration
	MaxDuration     time.Duration
	RequestsPerSec  float64
}

// RunLoadTest 运行负载测试
func RunLoadTest(t *testing.T, duration time.Duration, concurrency int) *LoadTestResult {
	suite := SetupBenchmark(t)
	defer suite.Cleanup()

	ctx := context.Background()
	result := &LoadTestResult{
		MinDuration: time.Hour, // 初始化为一个很大的值
	}

	// 使用channel控制并发
	sem := make(chan struct{}, concurrency)
	done := make(chan struct{})

	var mu sync.Mutex
	var wg sync.WaitGroup

	// 启动负载生成器
	start := time.Now()
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				wg.Add(1)
				sem <- struct{}{}

				go func() {
					defer wg.Done()
					defer func() { <-sem }()

					reqStart := time.Now()
					err := performRandomOperation(ctx, suite)
					reqDuration := time.Since(reqStart)

					mu.Lock()
					result.TotalRequests++
					if err == nil {
						result.SuccessRequests++
					} else {
						result.FailedRequests++
					}

					result.TotalDuration += reqDuration
					if reqDuration < result.MinDuration {
						result.MinDuration = reqDuration
					}
					if reqDuration > result.MaxDuration {
						result.MaxDuration = reqDuration
					}
					mu.Unlock()
				}()
			}
		}
	}()

	// 运行指定时长
	time.Sleep(duration)
	close(done)
	wg.Wait()

	// 计算统计数据
	elapsed := time.Since(start)
	result.AvgDuration = result.TotalDuration / time.Duration(result.TotalRequests)
	result.RequestsPerSec = float64(result.TotalRequests) / elapsed.Seconds()

	return result
}

// performRandomOperation 执行随机操作
func performRandomOperation(ctx context.Context, suite *BenchmarkSuite) error {
	switch rand.Intn(5) {
	case 0:
		// 获取项目列表
		_, err := suite.projectService.ListProjects(ctx, suite.testTenantID, 1, 20, nil)
		return err
	case 1:
		// 获取单个项目
		projectID := suite.testProjects[rand.Intn(len(suite.testProjects))]
		_, err := suite.projectService.GetProject(ctx, projectID)
		return err
	case 2:
		// 获取项目成员
		projectID := suite.testProjects[rand.Intn(len(suite.testProjects))]
		_, err := suite.projectService.GetProjectMembers(ctx, projectID)
		return err
	case 3:
		// 获取用户权限
		userID := suite.testUsers[rand.Intn(len(suite.testUsers))]
		_, err := suite.authService.GetUserPermissions(ctx, userID)
		return err
	case 4:
		// 获取项目统计
		projectID := suite.testProjects[rand.Intn(len(suite.testProjects))]
		_, err := suite.projectService.GetProjectStats(ctx, projectID)
		return err
	}
	return nil
}

// TestLoadTest 执行负载测试
func TestLoadTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	scenarios := []struct {
		name        string
		duration    time.Duration
		concurrency int
	}{
		{"Light Load", 30 * time.Second, 10},
		{"Medium Load", 30 * time.Second, 50},
		{"Heavy Load", 30 * time.Second, 100},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			result := RunLoadTest(t, scenario.duration, scenario.concurrency)

			t.Logf("=== %s Results ===", scenario.name)
			t.Logf("Total Requests: %d", result.TotalRequests)
			t.Logf("Success Rate: %.2f%%", float64(result.SuccessRequests)/float64(result.TotalRequests)*100)
			t.Logf("Requests/sec: %.2f", result.RequestsPerSec)
			t.Logf("Avg Response Time: %v", result.AvgDuration)
			t.Logf("Min Response Time: %v", result.MinDuration)
			t.Logf("Max Response Time: %v", result.MaxDuration)

			// 验证性能指标
			assert.Greater(t, float64(result.SuccessRequests)/float64(result.TotalRequests), 0.99, "Success rate should be > 99%")
			assert.Less(t, result.AvgDuration, 100*time.Millisecond, "Average response time should be < 100ms")
		})
	}
}
