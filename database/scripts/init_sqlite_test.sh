#!/bin/bash

# Cloud-Based Collaborative Development Platform
# SQLite Test Database Initialization Script
# 使用SQLite作为测试数据库，避免依赖外部服务

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[TEST-DB]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[TEST-DB]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[TEST-DB]${NC} $1"
}

log_error() {
    echo -e "${RED}[TEST-DB]${NC} $1"
}

# 设置测试环境
setup_test_environment() {
    log_info "设置SQLite测试环境..."
    
    # 获取项目根目录
    PROJECT_ROOT="$(dirname "$(dirname "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)")")"
    TEST_DB_FILE="$PROJECT_ROOT/test_database.sqlite"
    
    # 如果存在旧的测试数据库，删除它
    if [ -f "$TEST_DB_FILE" ]; then
        rm "$TEST_DB_FILE"
        log_info "已删除旧的测试数据库"
    fi
    
    export TEST_DB_FILE="$TEST_DB_FILE"
    
    log_info "测试数据库文件: $TEST_DB_FILE"
}

# 创建SQLite测试数据库结构
create_sqlite_schema() {
    log_info "创建SQLite测试数据库结构..."
    
    sqlite3 "$TEST_DB_FILE" << 'EOF'
-- 启用外键约束
PRAGMA foreign_keys = ON;

-- 创建租户表
CREATE TABLE tenants (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 创建用户表
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

-- 创建角色表
CREATE TABLE roles (
    id INTEGER PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    project_id INTEGER,
    name TEXT NOT NULL,
    description TEXT,
    permissions TEXT, -- JSON格式
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

-- 创建项目表
CREATE TABLE projects (
    id INTEGER PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    owner_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    FOREIGN KEY (owner_id) REFERENCES users(id)
);

-- 创建团队表
CREATE TABLE teams (
    id INTEGER PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    project_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT 1,
    created_by INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    FOREIGN KEY (project_id) REFERENCES projects(id),
    FOREIGN KEY (created_by) REFERENCES users(id)
);

-- 创建团队成员表
CREATE TABLE team_members (
    id INTEGER PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    team_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    role TEXT DEFAULT 'member',
    joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    FOREIGN KEY (team_id) REFERENCES teams(id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE(team_id, user_id)
);

-- 创建任务表
CREATE TABLE tasks (
    id INTEGER PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    project_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    priority TEXT NOT NULL DEFAULT 'medium',
    assignee_id INTEGER,
    created_by INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    FOREIGN KEY (project_id) REFERENCES projects(id),
    FOREIGN KEY (assignee_id) REFERENCES users(id),
    FOREIGN KEY (created_by) REFERENCES users(id)
);

-- 创建仓库表
CREATE TABLE repositories (
    id INTEGER PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    project_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    url TEXT,
    branch TEXT DEFAULT 'main',
    status TEXT NOT NULL DEFAULT 'active',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    FOREIGN KEY (project_id) REFERENCES projects(id)
);

-- 创建权限请求表
CREATE TABLE permission_requests (
    id INTEGER PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    project_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    requested_role TEXT NOT NULL,
    reason TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    reviewed_by INTEGER,
    reviewed_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    FOREIGN KEY (project_id) REFERENCES projects(id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (reviewed_by) REFERENCES users(id)
);

-- 创建邀请表
CREATE TABLE invitations (
    id INTEGER PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    project_id INTEGER NOT NULL,
    email TEXT NOT NULL,
    role TEXT NOT NULL,
    token TEXT UNIQUE NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    invited_by INTEGER NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    FOREIGN KEY (project_id) REFERENCES projects(id),
    FOREIGN KEY (invited_by) REFERENCES users(id)
);

-- 创建迁移历史表
CREATE TABLE schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

EOF

    log_success "SQLite数据库结构创建完成"
}

# 插入测试数据
insert_test_data() {
    log_info "插入测试数据..."
    
    sqlite3 "$TEST_DB_FILE" << 'EOF'
-- 插入默认租户
INSERT INTO tenants (id, name, slug, status) VALUES 
    (1, 'Default Tenant', 'default', 'active');

-- 插入测试用户
INSERT INTO users (id, tenant_id, username, email, password_hash, status) VALUES 
    (1, 1, 'testuser1', 'test1@example.com', '$2a$10$dummy.hash.for.testing', 'active'),
    (2, 1, 'testuser2', 'test2@example.com', '$2a$10$dummy.hash.for.testing', 'active'),
    (3, 1, 'testuser3', 'test3@example.com', '$2a$10$dummy.hash.for.testing', 'active');

-- 插入测试项目
INSERT INTO projects (id, tenant_id, name, slug, description, status, owner_id) VALUES 
    (1, 1, 'Test Project 1', 'test-project-1', 'Test project for integration tests', 'active', 1),
    (2, 1, 'Test Project 2', 'test-project-2', 'Another test project', 'active', 2);

-- 插入默认角色
INSERT INTO roles (id, tenant_id, project_id, name, description, permissions) VALUES 
    (1, 1, NULL, 'admin', 'System Administrator', '["*"]'),
    (2, 1, NULL, 'manager', 'Project Manager', '["project.read", "project.write", "user.read"]'),
    (3, 1, NULL, 'viewer', 'Viewer', '["project.read"]'),
    (4, 1, 1, 'viewer', 'Project Viewer', '["project.read"]');

-- 插入测试团队
INSERT INTO teams (id, tenant_id, project_id, name, description, is_active, created_by) VALUES 
    (1, 1, 1, '测试团队', '这是一个测试团队', 1, 1),
    (2, 1, 1, 'Development Team', 'Main development team', 1, 1);

-- 插入一些测试任务
INSERT INTO tasks (id, tenant_id, project_id, title, description, status, priority, assignee_id, created_by) VALUES 
    (1, 1, 1, 'Test Task 1', 'First test task', 'pending', 'medium', 1, 1),
    (2, 1, 1, 'Test Task 2', 'Second test task', 'in_progress', 'high', 2, 1);

-- 插入测试仓库
INSERT INTO repositories (id, tenant_id, project_id, name, description, url, branch, status) VALUES 
    (1, 1, 1, 'test-repo-1', 'Test repository 1', 'https://github.com/test/repo1', 'main', 'active'),
    (2, 1, 2, 'test-repo-2', 'Test repository 2', 'https://github.com/test/repo2', 'main', 'active');

-- 记录迁移
INSERT INTO schema_migrations (version) VALUES ('sqlite_test_schema');

EOF

    log_success "测试数据插入完成"
}

# 验证测试数据库
verify_test_database() {
    log_info "验证测试数据库..."
    
    # 检查表和数据
    TABLES=("tenants" "users" "projects" "tasks" "repositories" "roles" "teams")
    
    for table in "${TABLES[@]}"; do
        COUNT=$(sqlite3 "$TEST_DB_FILE" "SELECT COUNT(*) FROM $table;")
        log_success "表 $table 存在 (记录数: $COUNT)"
    done
    
    log_success "测试数据库验证完成"
}

# 生成Go测试配置
generate_go_test_config() {
    log_info "生成Go测试配置..."
    
    PROJECT_ROOT="$(dirname "$(dirname "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)")")"
    
    # 创建测试配置文件
    cat > "$PROJECT_ROOT/test_config.go" << EOF
package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	_ "github.com/mattn/go-sqlite3"
)

// GetTestDatabasePath 获取测试数据库路径
func GetTestDatabasePath() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	return filepath.Join(dir, "test_database.sqlite")
}

// GetTestDB 获取测试数据库连接
func GetTestDB() (*sql.DB, error) {
	dbPath := GetTestDatabasePath()
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, err
	}
	return sql.Open("sqlite3", dbPath)
}

// SetupTestEnvironment 设置测试环境变量
func SetupTestEnvironment() {
	os.Setenv("DATABASE_DRIVER", "sqlite3")
	os.Setenv("DATABASE_URL", GetTestDatabasePath())
	os.Setenv("ENVIRONMENT", "test")
	os.Setenv("JWT_SECRET", "test_jwt_secret_key_for_testing_only_minimum_32_characters_long")
}
EOF

    # 创建.env.test文件
    cat > "$PROJECT_ROOT/.env.test" << EOF
# SQLite Test Environment Configuration
# Generated: $(date)

# Database Configuration
DATABASE_DRIVER=sqlite3
DATABASE_URL=$TEST_DB_FILE
DATABASE_PATH=$TEST_DB_FILE

# Environment
ENVIRONMENT=test

# JWT Configuration (test only)
JWT_SECRET=test_jwt_secret_key_for_testing_only_minimum_32_characters_long

# Server Configuration
SERVER_PORT=8080
SERVER_HOST=localhost

# Disable external services in tests
TWO_FACTOR_ENABLED=false
FEATURE_SSO_ENABLED=false

# Test database file
TEST_DATABASE_PATH=$TEST_DB_FILE
EOF
    
    log_success "Go测试配置已生成"
}

# 主函数
main() {
    log_info "初始化SQLite测试环境"
    log_info "========================"
    
    setup_test_environment
    create_sqlite_schema
    insert_test_data
    verify_test_database
    generate_go_test_config
    
    log_success "========================"
    log_success "SQLite测试数据库初始化完成！"
    log_success "数据库文件: $TEST_DB_FILE"
    log_info "现在可以运行测试了: go test ./tests/..."
    log_info "或运行集成测试: go test ./tests/api_integration_test.go"
}

# 显示使用说明
show_usage() {
    cat << EOF
SQLite测试数据库初始化脚本

用法: $0 [选项]

选项:
    --help, -h              显示此帮助信息

此脚本将创建一个SQLite测试数据库，无需外部PostgreSQL服务。

EOF
}

# 解析命令行参数
if [[ $# -gt 0 ]]; then
    case $1 in
        --help|-h)
            show_usage
            exit 0
            ;;
        *)
            log_error "未知选项: $1"
            show_usage
            exit 1
            ;;
    esac
fi

# 检查sqlite3命令是否可用
if ! command -v sqlite3 &> /dev/null; then
    log_error "sqlite3 命令未找到，请安装SQLite3"
    exit 1
fi

# 执行主函数
main "$@"