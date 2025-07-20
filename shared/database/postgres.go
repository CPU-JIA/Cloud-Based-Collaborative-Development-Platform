package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config 数据库配置
type Config struct {
	Host            string
	Port            int
	Name            string
	User            string
	Password        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	LogLevel        logger.LogLevel
}

// PostgresDB PostgreSQL数据库封装
type PostgresDB struct {
	DB     *gorm.DB
	SqlDB  *sql.DB
	config Config
}

// TenantContext 租户上下文
type TenantContext struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
}

// NewPostgresDB 创建PostgreSQL数据库连接
func NewPostgresDB(config Config) (*PostgresDB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.Name, config.SSLMode,
	)

	// 配置GORM日志级别
	var logLevel logger.LogLevel
	switch config.LogLevel {
	case 1:
		logLevel = logger.Silent
	case 2:
		logLevel = logger.Error
	case 3:
		logLevel = logger.Warn
	case 4:
		logLevel = logger.Info
	default:
		logLevel = logger.Info
	}

	// 打开数据库连接
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 获取底层sql.DB
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

	return &PostgresDB{
		DB:     db,
		SqlDB:  sqlDB,
		config: config,
	}, nil
}

// WithTenant 设置租户上下文
func (p *PostgresDB) WithTenant(ctx context.Context, tenantID uuid.UUID) *gorm.DB {
	return p.DB.WithContext(ctx).Exec("SELECT set_config('app.current_tenant_id', ?, true)", tenantID.String())
}

// WithUser 设置用户上下文
func (p *PostgresDB) WithUser(ctx context.Context, userID uuid.UUID) *gorm.DB {
	return p.DB.WithContext(ctx).Exec("SELECT set_config('app.current_user_id', ?, true)", userID.String())
}

// WithContext 设置完整上下文（租户+用户）- 优化版本，减少SQL调用
func (p *PostgresDB) WithContext(ctx context.Context, tenantCtx TenantContext) *gorm.DB {
	db := p.DB.WithContext(ctx)
	
	// 批量设置配置参数，减少SQL调用次数
	if tenantCtx.TenantID != uuid.Nil && tenantCtx.UserID != uuid.Nil {
		// 一次性设置两个参数
		db = db.Exec(`
			SELECT 
				set_config('app.current_tenant_id', ?, true),
				set_config('app.current_user_id', ?, true)
		`, tenantCtx.TenantID.String(), tenantCtx.UserID.String())
	} else if tenantCtx.TenantID != uuid.Nil {
		// 只设置租户ID
		db = db.Exec("SELECT set_config('app.current_tenant_id', ?, true)", tenantCtx.TenantID.String())
	} else if tenantCtx.UserID != uuid.Nil {
		// 只设置用户ID
		db = db.Exec("SELECT set_config('app.current_user_id', ?, true)", tenantCtx.UserID.String())
	}
	
	return db
}

// Transaction 执行事务
func (p *PostgresDB) Transaction(ctx context.Context, tenantCtx TenantContext, fn func(*gorm.DB) error) error {
	return p.WithContext(ctx, tenantCtx).Transaction(fn)
}

// Ping 测试数据库连接
func (p *PostgresDB) Ping(ctx context.Context) error {
	return p.SqlDB.PingContext(ctx)
}

// Close 关闭数据库连接
func (p *PostgresDB) Close() error {
	return p.SqlDB.Close()
}

// Stats 获取连接池统计信息
func (p *PostgresDB) Stats() sql.DBStats {
	return p.SqlDB.Stats()
}

// HealthCheck 健康检查
func (p *PostgresDB) HealthCheck(ctx context.Context) error {
	// 测试连接
	if err := p.Ping(ctx); err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	// 检查连接池状态
	stats := p.Stats()
	if stats.OpenConnections >= p.config.MaxOpenConns {
		return fmt.Errorf("连接池已满: %d/%d", stats.OpenConnections, p.config.MaxOpenConns)
	}

	// 执行简单查询
	var result int
	if err := p.DB.WithContext(ctx).Raw("SELECT 1").Scan(&result).Error; err != nil {
		return fmt.Errorf("数据库查询失败: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("数据库查询结果异常: %d", result)
	}

	return nil
}

// Migration 数据库迁移接口
type Migration interface {
	Up(db *gorm.DB) error
	Down(db *gorm.DB) error
	Version() string
}

// MigrationManager 迁移管理器
type MigrationManager struct {
	db         *PostgresDB
	migrations []Migration
}

// NewMigrationManager 创建迁移管理器
func NewMigrationManager(db *PostgresDB) *MigrationManager {
	return &MigrationManager{
		db:         db,
		migrations: make([]Migration, 0),
	}
}

// AddMigration 添加迁移
func (m *MigrationManager) AddMigration(migration Migration) {
	m.migrations = append(m.migrations, migration)
}

// Migrate 执行迁移
func (m *MigrationManager) Migrate(ctx context.Context) error {
	// 创建迁移记录表
	if err := m.createMigrationTable(ctx); err != nil {
		return err
	}

	for _, migration := range m.migrations {
		// 检查是否已执行
		if executed, err := m.isMigrationExecuted(ctx, migration.Version()); err != nil {
			return err
		} else if executed {
			continue
		}

		// 执行迁移
		if err := m.executeMigration(ctx, migration); err != nil {
			return fmt.Errorf("执行迁移 %s 失败: %w", migration.Version(), err)
		}
	}

	return nil
}

// createMigrationTable 创建迁移记录表
func (m *MigrationManager) createMigrationTable(ctx context.Context) error {
	sql := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			executed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`
	return m.db.DB.WithContext(ctx).Exec(sql).Error
}

// isMigrationExecuted 检查迁移是否已执行
func (m *MigrationManager) isMigrationExecuted(ctx context.Context, version string) (bool, error) {
	var count int64
	err := m.db.DB.WithContext(ctx).
		Table("schema_migrations").
		Where("version = ?", version).
		Count(&count).Error
	return count > 0, err
}

// executeMigration 执行单个迁移
func (m *MigrationManager) executeMigration(ctx context.Context, migration Migration) error {
	return m.db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 执行迁移
		if err := migration.Up(tx); err != nil {
			return err
		}

		// 记录迁移
		return tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", migration.Version()).Error
	})
}

// GetTenantContextFromGorm 从GORM上下文中获取租户信息
func GetTenantContextFromGorm(db *gorm.DB) (TenantContext, error) {
	var tenantCtx TenantContext

	// 获取租户ID
	var tenantIDStr string
	if err := db.Raw("SELECT current_setting('app.current_tenant_id', true)").Scan(&tenantIDStr).Error; err == nil && tenantIDStr != "" {
		if tenantID, err := uuid.Parse(tenantIDStr); err == nil {
			tenantCtx.TenantID = tenantID
		}
	}

	// 获取用户ID
	var userIDStr string
	if err := db.Raw("SELECT current_setting('app.current_user_id', true)").Scan(&userIDStr).Error; err == nil && userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			tenantCtx.UserID = userID
		}
	}

	return tenantCtx, nil
}