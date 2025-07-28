package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/cloud-platform/collaborative-dev/shared/cache"
	"github.com/cloud-platform/collaborative-dev/shared/config"
	"github.com/cloud-platform/collaborative-dev/shared/database"
)

func main() {
	// 初始化日志
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	// 加载配置
	cfg, err := loadConfig()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// 初始化数据库
	db, err := initDatabase(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}

	// 初始化Redis
	redisClient, err := cache.NewRedisClient(&cfg.Redis, logger)
	if err != nil {
		logger.Fatal("Failed to initialize Redis", zap.Error(err))
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Error("Failed to close Redis connection", zap.Error(err))
		}
	}()

	// 创建缓存管理器
	cacheManager := cache.NewCacheManager(redisClient, logger)

	// 执行性能优化
	if err := performOptimizations(db, cacheManager, logger); err != nil {
		logger.Fatal("Failed to perform optimizations", zap.Error(err))
	}

	logger.Info("Performance optimizations completed successfully")
}

func loadConfig() (*config.Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// 设置默认值
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)

	// 读取环境变量
	viper.SetEnvPrefix("APP")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg config.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func initDatabase(cfg *config.Config, logger *zap.Logger) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 配置连接池
	poolConfig := database.ProductionPoolConfig()
	if err := poolConfig.ApplyToGorm(db); err != nil {
		return nil, err
	}

	return db, nil
}

func performOptimizations(db *gorm.DB, cacheManager *cache.CacheManager, logger *zap.Logger) error {
	ctx := context.Background()

	// 1. 创建数据库索引
	logger.Info("Creating database indexes...")
	optimizer := database.NewPerformanceOptimizer(db, logger)
	if err := optimizer.CreateIndexes(); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	// 2. 预热缓存
	logger.Info("Warming up cache...")
	if err := warmupCache(ctx, db, cacheManager, logger); err != nil {
		return fmt.Errorf("failed to warm up cache: %w", err)
	}

	// 3. 分析慢查询
	logger.Info("Analyzing slow queries...")
	stats, err := optimizer.AnalyzeSlowQueries(ctx, 100*time.Millisecond)
	if err != nil {
		logger.Warn("Failed to analyze slow queries", zap.Error(err))
	} else {
		logger.Info("Slow query analysis completed",
			zap.Int("slow_queries", len(stats.SlowQueries)),
			zap.Int64("total_queries", stats.QueryCount))
	}

	// 4. 执行VACUUM ANALYZE
	logger.Info("Running VACUUM ANALYZE...")
	tables := []string{"users", "projects", "files", "tasks", "notifications"}
	if err := optimizer.VacuumAnalyze(tables...); err != nil {
		logger.Warn("Failed to vacuum analyze", zap.Error(err))
	}

	// 5. 创建性能基准
	logger.Info("Creating performance baseline...")
	if err := createPerformanceBaseline(ctx, db, logger); err != nil {
		logger.Warn("Failed to create performance baseline", zap.Error(err))
	}

	return nil
}

func warmupCache(ctx context.Context, db *gorm.DB, cacheManager *cache.CacheManager, logger *zap.Logger) error {
	// 预热常用数据
	return cacheManager.WarmupCache(ctx, func(ctx context.Context) error {
		// 1. 缓存活跃用户
		logger.Info("Caching active users...")
		var users []struct {
			ID       string
			Email    string
			Username string
		}

		if err := db.Table("users").
			Where("is_active = ? AND last_login_at > ?", true, time.Now().AddDate(0, -1, 0)).
			Select("id, email, username").
			Find(&users).Error; err != nil {
			return err
		}

		for _, user := range users {
			// 缓存用户基本信息
			userKey := fmt.Sprintf(cache.KeyUserByID, user.ID)
			var cachedUser interface{}
			cacheManager.GetOrSet(ctx, userKey, &cachedUser, cache.TTLUserCache, func() (interface{}, error) {
				return user, nil
			})
		}

		// 2. 缓存热门项目
		logger.Info("Caching hot projects...")
		var projects []struct {
			ID   string
			Name string
		}

		if err := db.Table("projects").
			Where("status = ? AND updated_at > ?", "active", time.Now().AddDate(0, 0, -7)).
			Select("id, name").
			Limit(100).
			Find(&projects).Error; err != nil {
			return err
		}

		for _, project := range projects {
			projectKey := fmt.Sprintf(cache.KeyProjectByID, project.ID)
			var cachedProject interface{}
			cacheManager.GetOrSet(ctx, projectKey, &cachedProject, cache.TTLProjectCache, func() (interface{}, error) {
				return project, nil
			})
		}

		logger.Info("Cache warmup completed",
			zap.Int("users_cached", len(users)),
			zap.Int("projects_cached", len(projects)))

		return nil
	})
}

func createPerformanceBaseline(ctx context.Context, db *gorm.DB, logger *zap.Logger) error {
	// 创建性能基准表
	type PerformanceBaseline struct {
		ID              uint   `gorm:"primaryKey"`
		MetricName      string `gorm:"index"`
		MetricValue     float64
		Unit            string
		MeasuredAt      time.Time
		EnvironmentInfo string
	}

	// 迁移表
	if err := db.AutoMigrate(&PerformanceBaseline{}); err != nil {
		return err
	}

	// 记录基准指标
	baselines := []PerformanceBaseline{
		{
			MetricName:  "avg_query_time",
			MetricValue: measureAvgQueryTime(db),
			Unit:        "ms",
			MeasuredAt:  time.Now(),
		},
		{
			MetricName:  "connection_pool_size",
			MetricValue: float64(getConnectionPoolSize(db)),
			Unit:        "connections",
			MeasuredAt:  time.Now(),
		},
		{
			MetricName:  "table_row_counts",
			MetricValue: float64(getTotalRowCount(db)),
			Unit:        "rows",
			MeasuredAt:  time.Now(),
		},
	}

	for _, baseline := range baselines {
		if err := db.Create(&baseline).Error; err != nil {
			logger.Warn("Failed to create baseline",
				zap.String("metric", baseline.MetricName),
				zap.Error(err))
		}
	}

	return nil
}

func measureAvgQueryTime(db *gorm.DB) float64 {
	// 简单的查询时间测量
	start := time.Now()
	var count int64
	db.Table("users").Count(&count)
	duration := time.Since(start)
	return float64(duration.Milliseconds())
}

func getConnectionPoolSize(db *gorm.DB) int {
	sqlDB, _ := db.DB()
	stats := sqlDB.Stats()
	return stats.OpenConnections
}

func getTotalRowCount(db *gorm.DB) int64 {
	var total int64
	tables := []string{"users", "projects", "files", "tasks"}

	for _, table := range tables {
		var count int64
		db.Table(table).Count(&count)
		total += count
	}

	return total
}

// 性能优化建议生成器
type OptimizationSuggestion struct {
	Category        string
	Issue           string
	Suggestion      string
	Priority        string
	EstimatedImpact string
}

func generateOptimizationSuggestions(db *gorm.DB, logger *zap.Logger) []OptimizationSuggestion {
	var suggestions []OptimizationSuggestion

	// 检查缺失的索引
	var missingIndexes []struct {
		TableName  string
		ColumnName string
	}

	query := `
		SELECT 
			schemaname || '.' || tablename as table_name,
			attname as column_name
		FROM pg_stats
		WHERE schemaname = 'public'
		AND n_distinct > 100
		AND correlation < 0.1
		AND tablename || '_' || attname NOT IN (
			SELECT indexname FROM pg_indexes WHERE schemaname = 'public'
		)
		LIMIT 10
	`

	if err := db.Raw(query).Scan(&missingIndexes).Error; err == nil {
		for _, idx := range missingIndexes {
			suggestions = append(suggestions, OptimizationSuggestion{
				Category: "Database",
				Issue:    fmt.Sprintf("Missing index on %s.%s", idx.TableName, idx.ColumnName),
				Suggestion: fmt.Sprintf("CREATE INDEX idx_%s_%s ON %s(%s)",
					idx.TableName, idx.ColumnName, idx.TableName, idx.ColumnName),
				Priority:        "High",
				EstimatedImpact: "20-50% query improvement",
			})
		}
	}

	// 检查表膨胀
	var bloatedTables []struct {
		TableName string
		BloatSize int64
	}

	bloatQuery := `
		SELECT 
			schemaname || '.' || tablename as table_name,
			pg_relation_size(schemaname||'.'||tablename) - pg_relation_size(schemaname||'.'||tablename, 'main') as bloat_size
		FROM pg_tables
		WHERE schemaname = 'public'
		AND pg_relation_size(schemaname||'.'||tablename) > 1048576
		ORDER BY bloat_size DESC
		LIMIT 5
	`

	if err := db.Raw(bloatQuery).Scan(&bloatedTables).Error; err == nil {
		for _, table := range bloatedTables {
			if table.BloatSize > 10*1024*1024 { // 10MB膨胀
				suggestions = append(suggestions, OptimizationSuggestion{
					Category:        "Database",
					Issue:           fmt.Sprintf("Table bloat detected in %s", table.TableName),
					Suggestion:      fmt.Sprintf("VACUUM FULL %s", table.TableName),
					Priority:        "Medium",
					EstimatedImpact: "10-30% storage reduction",
				})
			}
		}
	}

	return suggestions
}
