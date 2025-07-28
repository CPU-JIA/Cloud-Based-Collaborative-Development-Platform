package database

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PerformanceOptimizer 数据库性能优化器
type PerformanceOptimizer struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewPerformanceOptimizer 创建性能优化器
func NewPerformanceOptimizer(db *gorm.DB, logger *zap.Logger) *PerformanceOptimizer {
	return &PerformanceOptimizer{
		db:     db,
		logger: logger,
	}
}

// CreateIndexes 创建必要的索引
func (po *PerformanceOptimizer) CreateIndexes() error {
	indexes := []struct {
		Table   string
		Name    string
		Columns []string
		Unique  bool
		Where   string
	}{
		// 用户相关索引
		{Table: "users", Name: "idx_users_tenant_email", Columns: []string{"tenant_id", "email"}, Unique: true},
		{Table: "users", Name: "idx_users_tenant_username", Columns: []string{"tenant_id", "username"}, Unique: true},
		{Table: "users", Name: "idx_users_tenant_active", Columns: []string{"tenant_id", "is_active"}},
		{Table: "users", Name: "idx_users_last_login", Columns: []string{"last_login_at"}},

		// 项目相关索引
		{Table: "projects", Name: "idx_projects_tenant_status", Columns: []string{"tenant_id", "status"}},
		{Table: "projects", Name: "idx_projects_tenant_key", Columns: []string{"tenant_id", "key"}, Unique: true},
		{Table: "projects", Name: "idx_projects_manager", Columns: []string{"manager_id"}},
		{Table: "projects", Name: "idx_projects_created_at", Columns: []string{"created_at"}},

		// 项目成员索引
		{Table: "project_members", Name: "idx_project_members_user", Columns: []string{"user_id"}},
		{Table: "project_members", Name: "idx_project_members_project_user", Columns: []string{"project_id", "user_id"}, Unique: true},

		// 文件相关索引
		{Table: "files", Name: "idx_files_project_path", Columns: []string{"project_id", "path"}},
		{Table: "files", Name: "idx_files_project_parent", Columns: []string{"project_id", "parent_id"}},
		{Table: "files", Name: "idx_files_updated_at", Columns: []string{"updated_at"}},

		// 会话相关索引
		{Table: "user_sessions", Name: "idx_sessions_user_active", Columns: []string{"user_id", "is_active"}},
		{Table: "user_sessions", Name: "idx_sessions_expires", Columns: []string{"expires_at"}},
		{Table: "user_sessions", Name: "idx_sessions_token", Columns: []string{"session_token"}, Unique: true},

		// 权限相关索引
		{Table: "user_roles", Name: "idx_user_roles_user_tenant", Columns: []string{"user_id", "tenant_id"}},
		{Table: "role_permissions", Name: "idx_role_permissions_role", Columns: []string{"role_id"}},

		// 通知相关索引
		{Table: "notifications", Name: "idx_notifications_user_read", Columns: []string{"user_id", "is_read"}},
		{Table: "notifications", Name: "idx_notifications_created", Columns: []string{"created_at"}},

		// 任务相关索引
		{Table: "tasks", Name: "idx_tasks_project_status", Columns: []string{"project_id", "status_id"}},
		{Table: "tasks", Name: "idx_tasks_assignee", Columns: []string{"assignee_id"}},
		{Table: "tasks", Name: "idx_tasks_number", Columns: []string{"project_id", "task_number"}, Unique: true},

		// CI/CD相关索引
		{Table: "pipelines", Name: "idx_pipelines_project_status", Columns: []string{"project_id", "status"}},
		{Table: "pipelines", Name: "idx_pipelines_created", Columns: []string{"created_at"}},
		{Table: "jobs", Name: "idx_jobs_pipeline_status", Columns: []string{"pipeline_id", "status"}},

		// 部分索引示例（PostgreSQL特有）
		{Table: "users", Name: "idx_users_active_verified", Columns: []string{"tenant_id", "is_email_verified"},
			Where: "is_active = true AND deleted_at IS NULL"},
		{Table: "projects", Name: "idx_projects_active", Columns: []string{"tenant_id", "created_at"},
			Where: "status = 'active' AND deleted_at IS NULL"},
	}

	for _, idx := range indexes {
		if err := po.createIndex(idx.Table, idx.Name, idx.Columns, idx.Unique, idx.Where); err != nil {
			po.logger.Error("Failed to create index",
				zap.String("table", idx.Table),
				zap.String("index", idx.Name),
				zap.Error(err))
			// 继续创建其他索引
		}
	}

	return nil
}

// createIndex 创建单个索引
func (po *PerformanceOptimizer) createIndex(table, name string, columns []string, unique bool, where string) error {
	sql := fmt.Sprintf("CREATE ")
	if unique {
		sql += "UNIQUE "
	}
	sql += fmt.Sprintf("INDEX IF NOT EXISTS %s ON %s (", name, table)

	for i, col := range columns {
		if i > 0 {
			sql += ", "
		}
		sql += col
	}
	sql += ")"

	if where != "" {
		sql += " WHERE " + where
	}

	return po.db.Exec(sql).Error
}

// OptimizeQueries 优化查询的辅助方法
type QueryOptimizer struct {
	db *gorm.DB
}

// NewQueryOptimizer 创建查询优化器
func NewQueryOptimizer(db *gorm.DB) *QueryOptimizer {
	return &QueryOptimizer{db: db}
}

// PreloadAssociations 预加载关联数据，防止N+1查询
func (qo *QueryOptimizer) PreloadAssociations(query *gorm.DB, associations ...string) *gorm.DB {
	for _, assoc := range associations {
		query = query.Preload(assoc)
	}
	return query
}

// SelectFields 只选择需要的字段
func (qo *QueryOptimizer) SelectFields(query *gorm.DB, fields ...string) *gorm.DB {
	return query.Select(fields)
}

// AddPagination 添加分页
func (qo *QueryOptimizer) AddPagination(query *gorm.DB, page, pageSize int) *gorm.DB {
	offset := (page - 1) * pageSize
	return query.Limit(pageSize).Offset(offset)
}

// UseIndex 使用特定索引
func (qo *QueryOptimizer) UseIndex(query *gorm.DB, indexName string) *gorm.DB {
	// 在 GORM v1 中，可以通过 Raw SQL 来实现索引提示
	return query
}

// BatchInsert 批量插入优化
func (qo *QueryOptimizer) BatchInsert(db *gorm.DB, records interface{}, batchSize int) error {
	return db.CreateInBatches(records, batchSize).Error
}

// ExplainQuery 分析查询计划
func (qo *QueryOptimizer) ExplainQuery(query *gorm.DB) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	// 获取SQL
	sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx
	})

	// 执行EXPLAIN
	err := qo.db.Raw("EXPLAIN ANALYZE " + sql).Scan(&results).Error
	return results, err
}

// ConnectionPoolOptimizer 连接池优化配置
type ConnectionPoolOptimizer struct {
	MaxOpenConns    int           // 最大开放连接数
	MaxIdleConns    int           // 最大空闲连接数
	ConnMaxLifetime time.Duration // 连接最大生命周期
	ConnMaxIdleTime time.Duration // 连接最大空闲时间
}

// DefaultPoolConfig 默认连接池配置
func DefaultPoolConfig() *ConnectionPoolOptimizer {
	return &ConnectionPoolOptimizer{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
	}
}

// ProductionPoolConfig 生产环境连接池配置
func ProductionPoolConfig() *ConnectionPoolOptimizer {
	return &ConnectionPoolOptimizer{
		MaxOpenConns:    100,
		MaxIdleConns:    25,
		ConnMaxLifetime: 10 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
	}
}

// ApplyToGorm 应用到GORM数据库连接
func (cpo *ConnectionPoolOptimizer) ApplyToGorm(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	sqlDB.SetMaxOpenConns(cpo.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cpo.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cpo.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cpo.ConnMaxIdleTime)

	return nil
}

// QueryStats 查询统计
type QueryStats struct {
	SlowQueries  []SlowQuery   `json:"slow_queries"`
	QueryCount   int64         `json:"query_count"`
	AvgQueryTime time.Duration `json:"avg_query_time"`
}

// SlowQuery 慢查询记录
type SlowQuery struct {
	SQL      string        `json:"sql"`
	Duration time.Duration `json:"duration"`
	Time     time.Time     `json:"time"`
}

// AnalyzeSlowQueries 分析慢查询
func (po *PerformanceOptimizer) AnalyzeSlowQueries(ctx context.Context, threshold time.Duration) (*QueryStats, error) {
	stats := &QueryStats{}

	// PostgreSQL慢查询分析
	query := `
		SELECT query, total_time, calls, mean_time
		FROM pg_stat_statements
		WHERE mean_time > $1
		ORDER BY mean_time DESC
		LIMIT 20
	`

	var results []struct {
		Query     string  `json:"query"`
		TotalTime float64 `json:"total_time"`
		Calls     int64   `json:"calls"`
		MeanTime  float64 `json:"mean_time"`
	}

	err := po.db.Raw(query, threshold.Milliseconds()).Scan(&results).Error
	if err != nil {
		// 如果pg_stat_statements扩展未启用，返回空结果
		po.logger.Warn("pg_stat_statements not available", zap.Error(err))
		return stats, nil
	}

	for _, r := range results {
		stats.SlowQueries = append(stats.SlowQueries, SlowQuery{
			SQL:      r.Query,
			Duration: time.Duration(r.MeanTime) * time.Millisecond,
			Time:     time.Now(),
		})
		stats.QueryCount += r.Calls
	}

	return stats, nil
}

// VacuumAnalyze 执行VACUUM ANALYZE优化表
func (po *PerformanceOptimizer) VacuumAnalyze(tables ...string) error {
	for _, table := range tables {
		if err := po.db.Exec(fmt.Sprintf("VACUUM ANALYZE %s", table)).Error; err != nil {
			po.logger.Error("Failed to vacuum analyze table",
				zap.String("table", table),
				zap.Error(err))
			return err
		}
	}
	return nil
}
