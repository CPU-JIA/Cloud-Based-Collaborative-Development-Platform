package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cloud-platform/collaborative-dev/shared/config"
	"github.com/cloud-platform/collaborative-dev/shared/database"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// TestConfig 集成测试配置
type TestConfig struct {
	Database struct {
		Host     string
		Port     string
		Name     string
		User     string
		Password string
		SSLMode  string
	}
	LogLevel string
	Timeout  time.Duration
}

// LoadTestConfig 加载测试配置
func LoadTestConfig() (*TestConfig, error) {
	cfg := &TestConfig{}
	
	// 数据库配置
	cfg.Database.Host = getEnvOrDefault("TEST_DB_HOST", "localhost")
	cfg.Database.Port = getEnvOrDefault("TEST_DB_PORT", "5432")
	cfg.Database.Name = getEnvOrDefault("TEST_DB_NAME", "collaborative_dev_test")
	cfg.Database.User = getEnvOrDefault("TEST_DB_USER", "postgres")
	cfg.Database.Password = getEnvOrDefault("TEST_DB_PASSWORD", "postgres")
	cfg.Database.SSLMode = getEnvOrDefault("TEST_DB_SSLMODE", "disable")
	
	// 其他配置
	cfg.LogLevel = getEnvOrDefault("TEST_LOG_LEVEL", "debug")
	
	timeout := getEnvOrDefault("TEST_TIMEOUT", "30s")
	duration, err := time.ParseDuration(timeout)
	if err != nil {
		return nil, fmt.Errorf("解析测试超时配置失败: %w", err)
	}
	cfg.Timeout = duration
	
	return cfg, nil
}

// getEnvOrDefault 获取环境变量或返回默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// LoadConfig 简化的配置加载函数
func LoadConfig() (*config.Config, error) {
	return &config.Config{
		Database: config.DatabaseConfig{
			Host:     getEnvOrDefault("DB_HOST", "localhost"),
			Port:     5432,
			Name:     getEnvOrDefault("DB_NAME", "collaborative_dev_test"),
			User:     getEnvOrDefault("DB_USER", "postgres"),
			Password: getEnvOrDefault("DB_PASSWORD", "postgres"),
			SSLMode:  getEnvOrDefault("DB_SSLMODE", "disable"),
		},
	}, nil
}

// SetupTestDatabase 设置测试数据库
func SetupTestDatabase(t *testing.T) (*gorm.DB, func()) {
	cfg, err := LoadTestConfig()
	require.NoError(t, err, "加载测试配置失败")
	
	// 创建数据库连接字符串
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)
	
	// 尝试连接数据库
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err, "连接测试数据库失败")
	
	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	err = db.PingContext(ctx)
	require.NoError(t, err, "测试数据库连接失败")
	
	// 如果数据库不存在，创建它
	_, err = db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", cfg.Database.Name))
	if err != nil && !isDBAlreadyExistsError(err) {
		require.NoError(t, err, "创建测试数据库失败")
	}
	
	db.Close()
	
	// 创建GORM连接  
	dbConfig := database.Config{
		Host:            cfg.Database.Host,
		Port:            5432, // Convert string to int
		Name:            cfg.Database.Name,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: time.Minute * 5,
		ConnMaxIdleTime: time.Minute,
		LogLevel:        1, // Silent
	}
	
	pgDB, err := database.NewPostgresDB(dbConfig)
	require.NoError(t, err, "创建GORM连接失败")
	
	// 返回清理函数
	cleanup := func() {
		if err := pgDB.Close(); err != nil {
			// Log error but don't fail test
		}
	}
	
	return pgDB.DB, cleanup
}

// isDBAlreadyExistsError 检查是否为数据库已存在错误
func isDBAlreadyExistsError(err error) bool {
	return err != nil && (
		err.Error() == "pq: database \"collaborative_dev_test\" already exists" ||
		err.Error() == "database \"collaborative_dev_test\" already exists")
}

// SetupTestLogger 设置测试日志
func SetupTestLogger(t *testing.T) *zap.Logger {
	cfg, err := LoadTestConfig()
	require.NoError(t, err, "加载测试配置失败")
	
	var logger *zap.Logger
	
	switch cfg.LogLevel {
	case "debug":
		logger, err = zap.NewDevelopment()
	case "info":
		logger, err = zap.NewProduction()
	default:
		logger, err = zap.NewDevelopment()
	}
	
	require.NoError(t, err, "创建测试日志失败")
	
	return logger
}

// TestDatabaseMigration 测试数据库迁移
func TestDatabaseMigration(t *testing.T) {
	db, cleanup := SetupTestDatabase(t)
	defer cleanup()
	
	logger := SetupTestLogger(t)
	
	// 测试创建项目表
	err := db.Exec(`
		CREATE TABLE IF NOT EXISTS projects (
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
	require.NoError(t, err, "创建项目表失败")
	
	// 测试创建项目成员表
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS project_members (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			project_id UUID NOT NULL,
			user_id UUID NOT NULL,
			role_id UUID NOT NULL,
			added_by UUID,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(project_id, user_id),
			FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
		)
	`).Error
	require.NoError(t, err, "创建项目成员表失败")
	
	// 测试创建索引
	err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_projects_tenant_id ON projects(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_projects_key ON projects(key);
		CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
		CREATE INDEX IF NOT EXISTS idx_project_members_project_id ON project_members(project_id);
		CREATE INDEX IF NOT EXISTS idx_project_members_user_id ON project_members(user_id);
	`).Error
	require.NoError(t, err, "创建索引失败")
	
	logger.Info("数据库迁移测试完成")
}

// SetupBenchmarkDatabase 为基准测试设置数据库连接
func SetupBenchmarkDatabase(b *testing.B) (*gorm.DB, func()) {
	cfg, err := LoadTestConfig()
	if err != nil {
		b.Fatalf("加载测试配置失败: %v", err)
	}
	
	// 创建GORM连接  
	dbConfig := database.Config{
		Host:            cfg.Database.Host,
		Port:            5432,
		Name:            cfg.Database.Name,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: time.Minute * 5,
		ConnMaxIdleTime: time.Minute,
		LogLevel:        1, // Silent
	}
	
	pgDB, err := database.NewPostgresDB(dbConfig)
	if err != nil {
		b.Fatalf("创建GORM连接失败: %v", err)
	}
	
	// 返回清理函数
	cleanup := func() {
		if err := pgDB.Close(); err != nil {
			// Log error but don't fail benchmark
		}
	}
	
	return pgDB.DB, cleanup
}

// BenchmarkDatabaseOperations 数据库操作基准测试
func BenchmarkDatabaseOperations(b *testing.B) {
	db, cleanup := SetupBenchmarkDatabase(b)
	defer cleanup()
	
	// 确保表存在
	err := db.Exec(`
		CREATE TABLE IF NOT EXISTS projects (
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
	require.NoError(b, err)
	
	b.ResetTimer()
	
	b.Run("项目创建性能", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := db.Exec(`
				INSERT INTO projects (tenant_id, key, name, description)
				VALUES (gen_random_uuid(), $1, $2, $3)
			`, fmt.Sprintf("bench-key-%d", i), fmt.Sprintf("基准测试项目%d", i), "基准测试描述").Error
			
			if err != nil {
				b.Fatalf("创建项目失败: %v", err)
			}
		}
	})
	
	// 清理数据
	db.Exec("DELETE FROM projects WHERE key LIKE 'bench-key-%'")
	
	b.Run("项目查询性能", func(b *testing.B) {
		// 先创建一些测试数据
		for i := 0; i < 100; i++ {
			db.Exec(`
				INSERT INTO projects (tenant_id, key, name, description)
				VALUES (gen_random_uuid(), $1, $2, $3)
			`, fmt.Sprintf("query-key-%d", i), fmt.Sprintf("查询测试项目%d", i), "查询测试描述")
		}
		
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			var count int64
			err := db.Raw("SELECT COUNT(*) FROM projects WHERE key LIKE 'query-key-%'").Scan(&count).Error
			if err != nil {
				b.Fatalf("查询项目失败: %v", err)
			}
		}
	})
	
	// 清理数据
	db.Exec("DELETE FROM projects WHERE key LIKE 'query-key-%'")
}

// TestHealthChecks 健康检查测试
func TestHealthChecks(t *testing.T) {
	t.Run("数据库健康检查", func(t *testing.T) {
		db, cleanup := SetupTestDatabase(t)
		defer cleanup()
		
		// 测试数据库连接
		sqlDB, err := db.DB()
		require.NoError(t, err, "获取SQL数据库实例失败")
		
		err = sqlDB.Ping()
		require.NoError(t, err, "数据库健康检查失败")
		
		// 测试数据库查询
		var result int
		err = db.Raw("SELECT 1").Scan(&result).Error
		require.NoError(t, err, "数据库查询测试失败")
		require.Equal(t, 1, result, "数据库查询结果不正确")
	})
	
	t.Run("日志系统健康检查", func(t *testing.T) {
		logger := SetupTestLogger(t)
		
		// 测试不同级别的日志输出
		logger.Debug("调试日志测试")
		logger.Info("信息日志测试")
		logger.Warn("警告日志测试")
		logger.Error("错误日志测试")
		
		// 测试结构化日志
		logger.Info("结构化日志测试",
			zap.String("test_type", "health_check"),
			zap.Int("test_count", 1),
			zap.Duration("test_duration", time.Millisecond*100))
	})
}

// TestConcurrentDatabaseAccess 并发数据库访问测试
func TestConcurrentDatabaseAccess(t *testing.T) {
	db, cleanup := SetupTestDatabase(t)
	defer cleanup()
	
	// 确保表存在
	err := db.Exec(`
		CREATE TABLE IF NOT EXISTS projects (
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
	require.NoError(t, err)
	
	const numGoroutines = 10
	const opsPerGoroutine = 50
	
	// 并发写入测试
	t.Run("并发写入", func(t *testing.T) {
		results := make(chan error, numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				var lastErr error
				for j := 0; j < opsPerGoroutine; j++ {
					err := db.Exec(`
						INSERT INTO projects (tenant_id, key, name, description)
						VALUES (gen_random_uuid(), $1, $2, $3)
					`,
						fmt.Sprintf("concurrent-key-%d-%d", goroutineID, j),
						fmt.Sprintf("并发测试项目%d-%d", goroutineID, j),
						"并发测试描述").Error
					
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
			require.NoError(t, err, "并发写入失败")
		}
		
		// 验证写入的数据数量
		var count int64
		err = db.Raw("SELECT COUNT(*) FROM projects WHERE key LIKE 'concurrent-key-%'").Scan(&count).Error
		require.NoError(t, err, "统计并发写入数据失败")
		require.Equal(t, int64(numGoroutines*opsPerGoroutine), count, "并发写入数据数量不正确")
	})
	
	// 并发读取测试
	t.Run("并发读取", func(t *testing.T) {
		results := make(chan error, numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func() {
				var lastErr error
				for j := 0; j < opsPerGoroutine; j++ {
					var count int64
					err := db.Raw("SELECT COUNT(*) FROM projects WHERE key LIKE 'concurrent-key-%'").Scan(&count).Error
					if err != nil {
						lastErr = err
						break
					}
				}
				results <- lastErr
			}()
		}
		
		// 收集结果
		for i := 0; i < numGoroutines; i++ {
			err := <-results
			require.NoError(t, err, "并发读取失败")
		}
	})
	
	// 清理测试数据
	db.Exec("DELETE FROM projects WHERE key LIKE 'concurrent-key-%'")
}

// TestDatabaseConstraints 数据库约束测试
func TestDatabaseConstraints(t *testing.T) {
	db, cleanup := SetupTestDatabase(t)
	defer cleanup()
	
	// 确保表存在
	err := db.Exec(`
		CREATE TABLE IF NOT EXISTS projects (
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
	require.NoError(t, err)
	
	t.Run("唯一约束测试", func(t *testing.T) {
		// 创建第一个项目
		err := db.Exec(`
			INSERT INTO projects (tenant_id, key, name, description)
			VALUES (gen_random_uuid(), 'unique-test-key', '唯一约束测试项目1', '描述1')
		`).Error
		require.NoError(t, err, "创建第一个项目失败")
		
		// 尝试创建具有相同key的第二个项目（应该失败）
		err = db.Exec(`
			INSERT INTO projects (tenant_id, key, name, description)
			VALUES (gen_random_uuid(), 'unique-test-key', '唯一约束测试项目2', '描述2')
		`).Error
		require.Error(t, err, "唯一约束应该阻止创建重复key的项目")
		
		// 清理
		db.Exec("DELETE FROM projects WHERE key = 'unique-test-key'")
	})
	
	t.Run("非空约束测试", func(t *testing.T) {
		// 尝试创建没有必填字段的项目
		err := db.Exec(`
			INSERT INTO projects (key, description)
			VALUES ('null-test-key', '非空约束测试')
		`).Error
		require.Error(t, err, "非空约束应该阻止创建缺少必填字段的项目")
	})
	
	t.Run("长度约束测试", func(t *testing.T) {
		// 尝试创建key过长的项目
		longKey := "this-is-a-very-long-key-that-exceeds-the-20-character-limit"
		err := db.Exec(`
			INSERT INTO projects (tenant_id, key, name, description)
			VALUES (gen_random_uuid(), $1, '长度约束测试项目', '描述')
		`, longKey).Error
		require.Error(t, err, "长度约束应该阻止创建key过长的项目")
	})
}

// TestDatabaseIndexes 数据库索引测试
func TestDatabaseIndexes(t *testing.T) {
	db, cleanup := SetupTestDatabase(t)
	defer cleanup()
	
	// 确保表和索引存在
	err := db.Exec(`
		CREATE TABLE IF NOT EXISTS projects (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL,
			key VARCHAR(20) UNIQUE NOT NULL,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			status VARCHAR(20) DEFAULT 'active',
			manager_id UUID,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE INDEX IF NOT EXISTS idx_projects_tenant_id ON projects(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_projects_key ON projects(key);
		CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
	`).Error
	require.NoError(t, err)
	
	// 插入测试数据
	tenantID := "550e8400-e29b-41d4-a716-446655440000"
	for i := 0; i < 1000; i++ {
		status := "active"
		if i%3 == 0 {
			status = "archived"
		}
		
		err = db.Exec(`
			INSERT INTO projects (tenant_id, key, name, description, status)
			VALUES ($1, $2, $3, $4, $5)
		`,
			tenantID,
			fmt.Sprintf("index-test-key-%d", i),
			fmt.Sprintf("索引测试项目%d", i),
			"索引测试描述",
			status).Error
		require.NoError(t, err)
	}
	
	t.Run("tenant_id索引性能", func(t *testing.T) {
		start := time.Now()
		
		var count int64
		err := db.Raw("SELECT COUNT(*) FROM projects WHERE tenant_id = $1", tenantID).Scan(&count).Error
		require.NoError(t, err)
		
		duration := time.Since(start)
		require.Less(t, duration, 100*time.Millisecond, "tenant_id索引查询耗时过长")
		require.Equal(t, int64(1000), count, "查询结果数量不正确")
	})
	
	t.Run("key索引性能", func(t *testing.T) {
		start := time.Now()
		
		var name string
		err := db.Raw("SELECT name FROM projects WHERE key = $1", "index-test-key-500").Scan(&name).Error
		require.NoError(t, err)
		
		duration := time.Since(start)
		require.Less(t, duration, 50*time.Millisecond, "key索引查询耗时过长")
		require.Equal(t, "索引测试项目500", name, "查询结果不正确")
	})
	
	t.Run("status索引性能", func(t *testing.T) {
		start := time.Now()
		
		var count int64
		err := db.Raw("SELECT COUNT(*) FROM projects WHERE status = $1", "active").Scan(&count).Error
		require.NoError(t, err)
		
		duration := time.Since(start)
		require.Less(t, duration, 100*time.Millisecond, "status索引查询耗时过长")
		require.Greater(t, count, int64(600), "active状态项目数量应该大于600")
	})
	
	// 清理测试数据
	db.Exec("DELETE FROM projects WHERE key LIKE 'index-test-key-%'")
}