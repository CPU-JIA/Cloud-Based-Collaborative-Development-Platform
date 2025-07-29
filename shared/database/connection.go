package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
)

// NewConnection 创建数据库连接（支持PostgreSQL和SQLite）
func NewConnection(config Config) (*gorm.DB, error) {
	var dialector gorm.Dialector
	
	// 根据配置决定使用哪种数据库
	if config.Host != "" && config.Port > 0 {
		// PostgreSQL配置
		dsn := fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			config.Host, config.Port, config.User, config.Password, config.Name, config.SSLMode,
		)
		dialector = postgres.Open(dsn)
	} else if config.Name != "" {
		// SQLite配置（用于测试）
		dialector = sqlite.Open(config.Name)
	} else {
		// 内存SQLite（用于单元测试）
		dialector = sqlite.Open(":memory:")
	}

	// 配置GORM日志
	gormLogger := logger.Default.LogMode(config.LogLevel)

	// 打开数据库连接
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		DisableForeignKeyConstraintWhenMigrating: true, // 兼容SQLite
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 仅对PostgreSQL配置连接池
	if config.Host != "" && config.Port > 0 {
		sqlDB, err := db.DB()
		if err != nil {
			return nil, fmt.Errorf("获取sql.DB失败: %w", err)
		}

		// 配置连接池
		sqlDB.SetMaxOpenConns(config.MaxOpenConns)
		sqlDB.SetMaxIdleConns(config.MaxIdleConns)
		sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
		sqlDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)

		// 测试连接
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := sqlDB.PingContext(ctx); err != nil {
			return nil, fmt.Errorf("数据库连接测试失败: %w", err)
		}
	}

	return db, nil
}

// TestConfig 创建测试专用的数据库配置
func TestConfig() Config {
	return Config{
		Name:            ":memory:", // SQLite内存数据库
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute,
		LogLevel:        logger.Silent, // 测试时静默
	}
}

// IsPostgreSQL 检查是否为PostgreSQL连接
func IsPostgreSQL(db *gorm.DB) bool {
	return db.Dialector.Name() == "postgres"
}

// IsSQLite 检查是否为SQLite连接
func IsSQLite(db *gorm.DB) bool {
	return db.Dialector.Name() == "sqlite"
}

// SetTenantContext 设置租户上下文（兼容不同数据库）
func SetTenantContext(db *gorm.DB, tenantID string) *gorm.DB {
	if IsPostgreSQL(db) {
		// PostgreSQL使用set_config
		var result string
		if err := db.Raw("SELECT set_config('app.current_tenant_id', ?, true)", tenantID).Scan(&result).Error; err != nil {
			// 如果set_config失败，静默处理
			return db
		}
	}
	// SQLite不需要特殊处理，直接返回
	return db
}

// GetTenantContext 获取租户上下文（兼容不同数据库）
func GetTenantContext(db *gorm.DB) string {
	if IsPostgreSQL(db) {
		// PostgreSQL使用current_setting
		var tenantID string
		if err := db.Raw("SELECT current_setting('app.current_tenant_id', true)").Scan(&tenantID).Error; err != nil {
			return "default" // 默认租户
		}
		if tenantID == "" {
			return "default"
		}
		return tenantID
	}
	// SQLite返回默认值
	return "default"
}