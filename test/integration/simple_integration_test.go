package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cloud-platform/collaborative-dev/shared/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// TestSimpleDatabaseConnection 测试数据库连接
func TestSimpleDatabaseConnection(t *testing.T) {
	// 跳过如果没有数据库环境变量
	if os.Getenv("TEST_DB_HOST") == "" {
		t.Skip("跳过数据库测试 - 未设置 TEST_DB_HOST 环境变量")
	}
	
	// 创建数据库配置
	dbConfig := database.Config{
		Host:            getSimpleEnvOrDefault("TEST_DB_HOST", "localhost"),
		Port:            5432,
		Name:            getSimpleEnvOrDefault("TEST_DB_NAME", "collaborative_dev_test"),
		User:            getSimpleEnvOrDefault("TEST_DB_USER", "postgres"),
		Password:        getSimpleEnvOrDefault("TEST_DB_PASSWORD", "postgres"),
		SSLMode:         getSimpleEnvOrDefault("TEST_DB_SSLMODE", "disable"),
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: time.Minute * 5,
		ConnMaxIdleTime: time.Minute,
		LogLevel:        1, // Silent
	}
	
	// 创建数据库连接
	pgDB, err := database.NewPostgresDB(dbConfig)
	require.NoError(t, err, "创建数据库连接失败")
	defer pgDB.Close()
	
	// 测试健康检查
	err = pgDB.HealthCheck(context.Background())
	assert.NoError(t, err, "数据库健康检查失败")
	
	// 测试简单查询
	var result int
	err = pgDB.DB.Raw("SELECT 1").Scan(&result).Error
	assert.NoError(t, err, "执行查询失败")
	assert.Equal(t, 1, result, "查询结果不正确")
}

// TestDatabaseTableCreation 测试数据表创建
func TestDatabaseTableCreation(t *testing.T) {
	// 跳过如果没有数据库环境变量
	if os.Getenv("TEST_DB_HOST") == "" {
		t.Skip("跳过数据库测试 - 未设置 TEST_DB_HOST 环境变量")
	}
	
	// 创建数据库配置
	dbConfig := database.Config{
		Host:            getSimpleEnvOrDefault("TEST_DB_HOST", "localhost"),
		Port:            5432,
		Name:            getSimpleEnvOrDefault("TEST_DB_NAME", "collaborative_dev_test"),
		User:            getSimpleEnvOrDefault("TEST_DB_USER", "postgres"),
		Password:        getSimpleEnvOrDefault("TEST_DB_PASSWORD", "postgres"),
		SSLMode:         getSimpleEnvOrDefault("TEST_DB_SSLMODE", "disable"),
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: time.Minute * 5,
		ConnMaxIdleTime: time.Minute,
		LogLevel:        1, // Silent
	}
	
	// 创建数据库连接
	pgDB, err := database.NewPostgresDB(dbConfig)
	require.NoError(t, err, "创建数据库连接失败")
	defer pgDB.Close()
	
	// 创建测试表
	err = pgDB.DB.Exec(`
		CREATE TABLE IF NOT EXISTS test_projects (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL,
			key VARCHAR(20) UNIQUE NOT NULL,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			status VARCHAR(20) DEFAULT 'active',
			manager_id UUID,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	require.NoError(t, err, "创建测试表失败")
	
	// 插入测试数据
	err = pgDB.DB.Exec(`
		INSERT INTO test_projects (tenant_id, key, name, description)
		VALUES (gen_random_uuid(), 'test-key', '测试项目', '测试描述')
		ON CONFLICT (key) DO NOTHING
	`).Error
	require.NoError(t, err, "插入测试数据失败")
	
	// 查询测试数据
	var count int64
	err = pgDB.DB.Raw("SELECT COUNT(*) FROM test_projects WHERE key = 'test-key'").Scan(&count).Error
	require.NoError(t, err, "查询测试数据失败")
	assert.Greater(t, count, int64(0), "未找到测试数据")
	
	// 清理测试数据
	err = pgDB.DB.Exec("DELETE FROM test_projects WHERE key = 'test-key'").Error
	require.NoError(t, err, "清理测试数据失败")
	
	// 删除测试表
	err = pgDB.DB.Exec("DROP TABLE IF EXISTS test_projects").Error
	require.NoError(t, err, "删除测试表失败")
}

// TestSimpleConcurrentDatabaseAccess 测试并发数据库访问
func TestSimpleConcurrentDatabaseAccess(t *testing.T) {
	// 跳过如果没有数据库环境变量
	if os.Getenv("TEST_DB_HOST") == "" {
		t.Skip("跳过数据库测试 - 未设置 TEST_DB_HOST 环境变量")
	}
	
	// 创建数据库配置
	dbConfig := database.Config{
		Host:            getSimpleEnvOrDefault("TEST_DB_HOST", "localhost"),
		Port:            5432,
		Name:            getSimpleEnvOrDefault("TEST_DB_NAME", "collaborative_dev_test"),
		User:            getSimpleEnvOrDefault("TEST_DB_USER", "postgres"),
		Password:        getSimpleEnvOrDefault("TEST_DB_PASSWORD", "postgres"),
		SSLMode:         getSimpleEnvOrDefault("TEST_DB_SSLMODE", "disable"),
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: time.Minute * 5,
		ConnMaxIdleTime: time.Minute,
		LogLevel:        1, // Silent
	}
	
	// 创建数据库连接
	pgDB, err := database.NewPostgresDB(dbConfig)
	require.NoError(t, err, "创建数据库连接失败")
	defer pgDB.Close()
	
	// 创建测试表
	err = pgDB.DB.Exec(`
		CREATE TABLE IF NOT EXISTS concurrent_test (
			id SERIAL PRIMARY KEY,
			value VARCHAR(100) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	require.NoError(t, err, "创建测试表失败")
	
	const numGoroutines = 5
	const opsPerGoroutine = 10
	
	// 并发写入测试
	results := make(chan error, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			var lastErr error
			for j := 0; j < opsPerGoroutine; j++ {
				err := pgDB.DB.Exec(`
					INSERT INTO concurrent_test (value)
					VALUES ($1)
				`, fmt.Sprintf("goroutine-%d-op-%d", goroutineID, j)).Error
				
				if err != nil {
					lastErr = err
					break
				}
			}
			results <- lastErr
		}(i)
	}
	
	// 收集结果
	for i := 0; i < numGoroutines; i++ {
		err := <-results
		assert.NoError(t, err, "并发写入失败")
	}
	
	// 验证写入的数据数量
	var count int64
	err = pgDB.DB.Raw("SELECT COUNT(*) FROM concurrent_test").Scan(&count).Error
	require.NoError(t, err, "统计数据失败")
	assert.Equal(t, int64(numGoroutines*opsPerGoroutine), count, "并发写入数据数量不正确")
	
	// 清理测试数据
	err = pgDB.DB.Exec("DROP TABLE IF EXISTS concurrent_test").Error
	require.NoError(t, err, "清理测试表失败")
}

// TestLoggerSetup 测试日志设置
func TestLoggerSetup(t *testing.T) {
	// 测试开发模式日志
	logger, err := zap.NewDevelopment()
	require.NoError(t, err, "创建开发日志失败")
	
	// 测试日志输出
	logger.Debug("调试日志测试")
	logger.Info("信息日志测试")
	logger.Warn("警告日志测试")
	
	// 测试结构化日志
	logger.Info("结构化日志测试",
		zap.String("test_type", "logger_setup"),
		zap.Int("test_count", 1),
		zap.Duration("test_duration", time.Millisecond*100))
	
	// 测试生产模式日志
	prodLogger, err := zap.NewProduction()
	require.NoError(t, err, "创建生产日志失败")
	
	prodLogger.Info("生产模式日志测试",
		zap.String("environment", "test"),
		zap.Bool("test_mode", true))
}

// BenchmarkDatabaseQuery 数据库查询基准测试
func BenchmarkDatabaseQuery(b *testing.B) {
	// 跳过如果没有数据库环境变量
	if os.Getenv("TEST_DB_HOST") == "" {
		b.Skip("跳过数据库基准测试 - 未设置 TEST_DB_HOST 环境变量")
	}
	
	// 创建数据库配置
	dbConfig := database.Config{
		Host:            getSimpleEnvOrDefault("TEST_DB_HOST", "localhost"),
		Port:            5432,
		Name:            getSimpleEnvOrDefault("TEST_DB_NAME", "collaborative_dev_test"),
		User:            getSimpleEnvOrDefault("TEST_DB_USER", "postgres"),
		Password:        getSimpleEnvOrDefault("TEST_DB_PASSWORD", "postgres"),
		SSLMode:         getSimpleEnvOrDefault("TEST_DB_SSLMODE", "disable"),
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: time.Minute * 5,
		ConnMaxIdleTime: time.Minute,
		LogLevel:        1, // Silent
	}
	
	// 创建数据库连接
	pgDB, err := database.NewPostgresDB(dbConfig)
	if err != nil {
		b.Fatalf("创建数据库连接失败: %v", err)
	}
	defer pgDB.Close()
	
	b.ResetTimer()
	
	b.Run("简单查询性能", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var result int
			err := pgDB.DB.Raw("SELECT 1").Scan(&result).Error
			if err != nil {
				b.Fatalf("查询失败: %v", err)
			}
		}
	})
	
	b.Run("连接池性能", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var result int
				err := pgDB.DB.Raw("SELECT 1").Scan(&result).Error
				if err != nil {
					b.Fatalf("并发查询失败: %v", err)
				}
			}
		})
	})
}

// getSimpleEnvOrDefault 获取环境变量或返回默认值
func getSimpleEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}