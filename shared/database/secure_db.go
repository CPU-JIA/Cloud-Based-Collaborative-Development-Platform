package database

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"github.com/cloud-platform/collaborative-dev/shared/logger"
	"gorm.io/gorm"
)

// SecureDB 安全的数据库包装器
type SecureDB struct {
	*gorm.DB
	logger logger.Logger
	config *SecureDBConfig
}

// SecureDBConfig 安全数据库配置
type SecureDBConfig struct {
	// EnableQueryValidation 启用查询验证
	EnableQueryValidation bool

	// EnableParameterValidation 启用参数验证
	EnableParameterValidation bool

	// EnableAuditLog 启用审计日志
	EnableAuditLog bool

	// MaxQueryLength 最大查询长度
	MaxQueryLength int

	// AllowedTables 允许访问的表列表（白名单）
	AllowedTables []string

	// BlockedTables 禁止访问的表列表（黑名单）
	BlockedTables []string

	// Logger 日志记录器
	Logger logger.Logger
}

// DefaultSecureDBConfig 默认安全数据库配置
var DefaultSecureDBConfig = SecureDBConfig{
	EnableQueryValidation:     true,
	EnableParameterValidation: true,
	EnableAuditLog:            true,
	MaxQueryLength:            10000,
	BlockedTables: []string{
		"pg_", "information_schema", "mysql.", "sys.",
	},
}

// NewSecureDB 创建安全的数据库包装器
func NewSecureDB(db *gorm.DB, config *SecureDBConfig) *SecureDB {
	if config == nil {
		cfg := DefaultSecureDBConfig
		config = &cfg
	}

	sdb := &SecureDB{
		DB:     db,
		logger: config.Logger,
		config: config,
	}

	// 注册安全回调
	sdb.registerSecurityCallbacks()

	return sdb
}

// registerSecurityCallbacks 注册安全回调
func (sdb *SecureDB) registerSecurityCallbacks() {
	// 在创建之前验证
	sdb.Callback().Create().Before("gorm:create").Register("security:validate_create", sdb.validateBeforeCreate)

	// 在更新之前验证
	sdb.Callback().Update().Before("gorm:update").Register("security:validate_update", sdb.validateBeforeUpdate)

	// 在删除之前验证
	sdb.Callback().Delete().Before("gorm:delete").Register("security:validate_delete", sdb.validateBeforeDelete)

	// 在查询之前验证
	sdb.Callback().Query().Before("gorm:query").Register("security:validate_query", sdb.validateBeforeQuery)
}

// validateBeforeCreate 创建前验证
func (sdb *SecureDB) validateBeforeCreate(db *gorm.DB) {
	if sdb.config.EnableParameterValidation {
		// 验证表名
		if err := sdb.validateTableAccess(db.Statement.Table); err != nil {
			db.AddError(err)
			return
		}

		// 验证字段值
		if err := sdb.validateFieldValues(db.Statement.Dest); err != nil {
			db.AddError(err)
			return
		}
	}

	if sdb.config.EnableAuditLog {
		sdb.logDatabaseOperation(db, "CREATE")
	}
}

// validateBeforeUpdate 更新前验证
func (sdb *SecureDB) validateBeforeUpdate(db *gorm.DB) {
	if sdb.config.EnableParameterValidation {
		// 验证表名
		if err := sdb.validateTableAccess(db.Statement.Table); err != nil {
			db.AddError(err)
			return
		}

		// 验证WHERE条件
		if err := sdb.validateWhereConditions(db); err != nil {
			db.AddError(err)
			return
		}
	}

	if sdb.config.EnableAuditLog {
		sdb.logDatabaseOperation(db, "UPDATE")
	}
}

// validateBeforeDelete 删除前验证
func (sdb *SecureDB) validateBeforeDelete(db *gorm.DB) {
	if sdb.config.EnableParameterValidation {
		// 验证表名
		if err := sdb.validateTableAccess(db.Statement.Table); err != nil {
			db.AddError(err)
			return
		}

		// 防止无条件删除
		if db.Statement.SQL.String() == "" && len(db.Statement.Clauses) == 0 {
			db.AddError(fmt.Errorf("禁止无条件删除操作"))
			return
		}
	}

	if sdb.config.EnableAuditLog {
		sdb.logDatabaseOperation(db, "DELETE")
	}
}

// validateBeforeQuery 查询前验证
func (sdb *SecureDB) validateBeforeQuery(db *gorm.DB) {
	if sdb.config.EnableQueryValidation {
		// 验证查询长度
		query := db.Statement.SQL.String()
		if len(query) > sdb.config.MaxQueryLength {
			db.AddError(fmt.Errorf("查询长度超过限制：%d", len(query)))
			return
		}

		// 验证危险查询模式
		if err := sdb.validateQuerySafety(query); err != nil {
			db.AddError(err)
			return
		}
	}

	if sdb.config.EnableAuditLog {
		sdb.logDatabaseOperation(db, "QUERY")
	}
}

// validateTableAccess 验证表访问权限
func (sdb *SecureDB) validateTableAccess(tableName string) error {
	if tableName == "" {
		return nil
	}

	tableLower := strings.ToLower(tableName)

	// 检查黑名单
	for _, blocked := range sdb.config.BlockedTables {
		if strings.HasPrefix(tableLower, strings.ToLower(blocked)) {
			return fmt.Errorf("禁止访问系统表：%s", tableName)
		}
	}

	// 如果配置了白名单，检查是否在白名单中
	if len(sdb.config.AllowedTables) > 0 {
		allowed := false
		for _, table := range sdb.config.AllowedTables {
			if strings.EqualFold(table, tableName) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("表不在访问白名单中：%s", tableName)
		}
	}

	return nil
}

// validateFieldValues 验证字段值
func (sdb *SecureDB) validateFieldValues(model interface{}) error {
	// 这里可以添加具体的字段验证逻辑
	// 例如：检查SQL注入、XSS等
	return nil
}

// validateWhereConditions 验证WHERE条件
func (sdb *SecureDB) validateWhereConditions(db *gorm.DB) error {
	// 检查是否有危险的WHERE条件
	whereStr := fmt.Sprintf("%v", db.Statement.Clauses["WHERE"])
	if strings.Contains(whereStr, "1=1") || strings.Contains(whereStr, "1 = 1") {
		return fmt.Errorf("检测到危险的WHERE条件")
	}
	return nil
}

// 危险SQL模式
var dangerousSQLPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(exec|execute)\s+(sp_|xp_)`),
	regexp.MustCompile(`(?i)(\bunion\b.*\bselect\b)`),
	regexp.MustCompile(`(?i)(into\s+(outfile|dumpfile))`),
	regexp.MustCompile(`(?i)(load_file\s*\()`),
	regexp.MustCompile(`(?i)(benchmark\s*\()`),
	regexp.MustCompile(`(?i)(sleep\s*\()`),
}

// validateQuerySafety 验证查询安全性
func (sdb *SecureDB) validateQuerySafety(query string) error {
	for _, pattern := range dangerousSQLPatterns {
		if pattern.MatchString(query) {
			return fmt.Errorf("检测到危险的SQL模式")
		}
	}
	return nil
}

// logDatabaseOperation 记录数据库操作
func (sdb *SecureDB) logDatabaseOperation(db *gorm.DB, operation string) {
	if sdb.logger == nil {
		return
	}

	ctx := db.Statement.Context
	userID := ""
	if ctx != nil {
		if uid, ok := ctx.Value("user_id").(string); ok {
			userID = uid
		}
	}

	sdb.logger.WithFields(map[string]interface{}{
		"operation": operation,
		"table":     db.Statement.Table,
		"user_id":   userID,
		"query":     db.Statement.SQL.String(),
	}).Info("数据库操作审计")
}

// SafeQuery 安全查询（使用参数化查询）
func (sdb *SecureDB) SafeQuery(dest interface{}, query string, args ...interface{}) error {
	// 验证查询
	if err := sdb.validateQuerySafety(query); err != nil {
		return err
	}

	// 使用参数化查询防止SQL注入
	return sdb.Raw(query, args...).Scan(dest).Error
}

// SafeExec 安全执行（使用参数化查询）
func (sdb *SecureDB) SafeExec(query string, args ...interface{}) error {
	// 验证查询
	if err := sdb.validateQuerySafety(query); err != nil {
		return err
	}

	// 使用参数化查询防止SQL注入
	return sdb.Exec(query, args...).Error
}

// Transaction 安全事务
func (sdb *SecureDB) Transaction(fc func(tx *SecureDB) error, opts ...*sql.TxOptions) error {
	return sdb.DB.Transaction(func(tx *gorm.DB) error {
		// 创建事务中的安全DB
		stx := &SecureDB{
			DB:     tx,
			logger: sdb.logger,
			config: sdb.config,
		}
		return fc(stx)
	}, opts...)
}

// WithContext 带上下文的安全DB
func (sdb *SecureDB) WithContext(ctx context.Context) *SecureDB {
	return &SecureDB{
		DB:     sdb.DB.WithContext(ctx),
		logger: sdb.logger,
		config: sdb.config,
	}
}

// SanitizeInput 清理输入，移除潜在的SQL注入字符
func SanitizeInput(input string) string {
	// 移除单引号
	input = strings.ReplaceAll(input, "'", "''")

	// 移除注释符号
	input = strings.ReplaceAll(input, "--", "")
	input = strings.ReplaceAll(input, "/*", "")
	input = strings.ReplaceAll(input, "*/", "")

	// 移除分号（防止多语句执行）
	input = strings.ReplaceAll(input, ";", "")

	return input
}

// EscapeLike 转义LIKE查询中的特殊字符
func EscapeLike(input string) string {
	input = strings.ReplaceAll(input, "\\", "\\\\")
	input = strings.ReplaceAll(input, "%", "\\%")
	input = strings.ReplaceAll(input, "_", "\\_")
	return input
}

// BuildSafeWhereClause 构建安全的WHERE子句
func BuildSafeWhereClause(field string, operator string, value interface{}) (string, []interface{}) {
	// 验证字段名（只允许字母、数字和下划线）
	if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(field) {
		return "", nil
	}

	// 验证操作符
	allowedOperators := []string{"=", "!=", ">", "<", ">=", "<=", "LIKE", "IN", "NOT IN"}
	operatorValid := false
	for _, op := range allowedOperators {
		if strings.EqualFold(operator, op) {
			operatorValid = true
			break
		}
	}
	if !operatorValid {
		return "", nil
	}

	// 构建参数化查询
	return fmt.Sprintf("%s %s ?", field, operator), []interface{}{value}
}

// SecureQueryBuilder 安全查询构建器
type SecureQueryBuilder struct {
	whereClauses []string
	args         []interface{}
	orderBy      string
	limit        int
	offset       int
}

// NewSecureQueryBuilder 创建安全查询构建器
func NewSecureQueryBuilder() *SecureQueryBuilder {
	return &SecureQueryBuilder{
		whereClauses: []string{},
		args:         []interface{}{},
	}
}

// Where 添加WHERE条件
func (qb *SecureQueryBuilder) Where(field string, operator string, value interface{}) *SecureQueryBuilder {
	clause, args := BuildSafeWhereClause(field, operator, value)
	if clause != "" {
		qb.whereClauses = append(qb.whereClauses, clause)
		qb.args = append(qb.args, args...)
	}
	return qb
}

// OrderBy 设置排序
func (qb *SecureQueryBuilder) OrderBy(field string, direction string) *SecureQueryBuilder {
	// 验证字段名
	if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(field) {
		return qb
	}

	// 验证排序方向
	if strings.ToUpper(direction) != "ASC" && strings.ToUpper(direction) != "DESC" {
		return qb
	}

	qb.orderBy = fmt.Sprintf("%s %s", field, strings.ToUpper(direction))
	return qb
}

// Limit 设置限制
func (qb *SecureQueryBuilder) Limit(limit int) *SecureQueryBuilder {
	if limit > 0 && limit <= 1000 { // 最大限制1000条
		qb.limit = limit
	}
	return qb
}

// Offset 设置偏移
func (qb *SecureQueryBuilder) Offset(offset int) *SecureQueryBuilder {
	if offset >= 0 {
		qb.offset = offset
	}
	return qb
}

// Build 构建查询
func (qb *SecureQueryBuilder) Build(baseQuery string) (string, []interface{}) {
	query := baseQuery

	if len(qb.whereClauses) > 0 {
		query += " WHERE " + strings.Join(qb.whereClauses, " AND ")
	}

	if qb.orderBy != "" {
		query += " ORDER BY " + qb.orderBy
	}

	if qb.limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", qb.limit)
	}

	if qb.offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", qb.offset)
	}

	return query, qb.args
}
