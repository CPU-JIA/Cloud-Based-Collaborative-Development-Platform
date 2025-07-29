#!/bin/bash

# Cloud-Based Collaborative Development Platform
# Test Database Initialization Script
# 专门用于测试环境的数据库初始化

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

# 设置测试环境默认值
setup_test_environment() {
    log_info "设置测试环境默认配置..."
    
    export DATABASE_HOST="${DATABASE_HOST:-localhost}"
    export DATABASE_PORT="${DATABASE_PORT:-5432}"
    export DATABASE_NAME="${DATABASE_NAME:-devcollab_test}"
    export DATABASE_USER="${DATABASE_USER:-devcollab_test}"
    export DATABASE_PASSWORD="${DATABASE_PASSWORD:-test_password_2025}"
    export POSTGRES_ADMIN_USER="${POSTGRES_ADMIN_USER:-postgres}"
    
    # 测试环境必须有管理员密码
    if [ -z "$POSTGRES_ADMIN_PASSWORD" ]; then
        log_warning "POSTGRES_ADMIN_PASSWORD 未设置，尝试使用默认值"
        export POSTGRES_ADMIN_PASSWORD="${POSTGRES_ADMIN_PASSWORD:-postgres}"
    fi
    
    log_info "测试数据库配置:"
    log_info "  HOST: $DATABASE_HOST"
    log_info "  PORT: $DATABASE_PORT"
    log_info "  DATABASE: $DATABASE_NAME"
    log_info "  USER: $DATABASE_USER"
}

# 检查并启动PostgreSQL (如果需要)
ensure_postgresql_running() {
    log_info "检查PostgreSQL服务状态..."
    
    # 尝试连接数据库
    export PGPASSWORD="$POSTGRES_ADMIN_PASSWORD"
    if psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$POSTGRES_ADMIN_USER" -d postgres -c "SELECT 1;" &> /dev/null; then
        log_success "PostgreSQL已运行"
        return 0
    fi
    
    log_warning "PostgreSQL未运行，尝试启动..."
    
    # 尝试启动PostgreSQL服务
    if command -v systemctl &> /dev/null; then
        sudo systemctl start postgresql || true
    elif command -v service &> /dev/null; then
        sudo service postgresql start || true
    elif command -v brew &> /dev/null; then
        brew services start postgresql || true
    fi
    
    # 等待服务启动
    sleep 3
    
    # 再次检查连接
    if psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$POSTGRES_ADMIN_USER" -d postgres -c "SELECT 1;" &> /dev/null; then
        log_success "PostgreSQL启动成功"
    else
        log_error "无法启动PostgreSQL，请手动启动后重试"
        log_error "或者使用Docker: docker run --name postgres-test -e POSTGRES_PASSWORD=postgres -p 5432:5432 -d postgres:15"
        exit 1
    fi
}

# 强制重创建测试数据库
recreate_test_database() {
    log_info "重新创建测试数据库..."
    
    export PGPASSWORD="$POSTGRES_ADMIN_PASSWORD"
    
    # 断开所有连接
    psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$POSTGRES_ADMIN_USER" -d postgres -c "
        SELECT pg_terminate_backend(pg_stat_activity.pid)
        FROM pg_stat_activity
        WHERE pg_stat_activity.datname = '$DATABASE_NAME'
        AND pid <> pg_backend_pid();
    " 2>/dev/null || true
    
    # 删除数据库（如果存在）
    psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$POSTGRES_ADMIN_USER" -d postgres -c "
        DROP DATABASE IF EXISTS \"$DATABASE_NAME\";
    " 2>/dev/null || true
    
    # 删除用户（如果存在）
    psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$POSTGRES_ADMIN_USER" -d postgres -c "
        DROP ROLE IF EXISTS \"$DATABASE_USER\";
    " 2>/dev/null || true
    
    # 创建用户
    psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$POSTGRES_ADMIN_USER" -d postgres -c "
        CREATE USER \"$DATABASE_USER\" WITH 
        PASSWORD '$DATABASE_PASSWORD'
        CREATEDB
        NOCREATEROLE
        NOSUPERUSER;
    "
    
    # 创建数据库
    psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$POSTGRES_ADMIN_USER" -d postgres -c "
        CREATE DATABASE \"$DATABASE_NAME\" 
        WITH OWNER \"$DATABASE_USER\"
        ENCODING 'UTF8'
        LC_COLLATE 'C'
        LC_CTYPE 'C'
        TEMPLATE template0;
    "
    
    # 授予权限
    psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$POSTGRES_ADMIN_USER" -d "$DATABASE_NAME" -c "
        GRANT ALL PRIVILEGES ON DATABASE \"$DATABASE_NAME\" TO \"$DATABASE_USER\";
        GRANT ALL PRIVILEGES ON SCHEMA public TO \"$DATABASE_USER\";
        ALTER SCHEMA public OWNER TO \"$DATABASE_USER\";
    "
    
    log_success "测试数据库重新创建完成"
}

# 安装必要的扩展
install_test_extensions() {
    log_info "安装PostgreSQL扩展..."
    
    export PGPASSWORD="$POSTGRES_ADMIN_PASSWORD"
    
    psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$POSTGRES_ADMIN_USER" -d "$DATABASE_NAME" -c "
        CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";
        CREATE EXTENSION IF NOT EXISTS \"pgcrypto\";
        CREATE EXTENSION IF NOT EXISTS \"citext\";
    " || {
        log_warning "某些扩展安装失败，继续执行..."
    }
    
    log_success "扩展安装完成"
}

# 执行所有迁移
run_test_migrations() {
    log_info "执行测试环境数据库迁移..."
    
    export PGPASSWORD="$DATABASE_PASSWORD"
    
    # 获取脚本目录
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    MIGRATIONS_DIR="$(dirname "$SCRIPT_DIR")/migrations"
    
    if [ ! -d "$MIGRATIONS_DIR" ]; then
        log_error "迁移目录不存在: $MIGRATIONS_DIR"
        exit 1
    fi
    
    # 创建迁移历史表
    psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -c "
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version VARCHAR(255) PRIMARY KEY,
            applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        );
    "
    
    # 执行所有迁移文件
    for migration_file in "$MIGRATIONS_DIR"/*.sql; do
        if [ -f "$migration_file" ]; then
            migration_name=$(basename "$migration_file" .sql)
            
            log_info "执行迁移: $migration_name"
            
            if psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -f "$migration_file" -q; then
                # 记录迁移历史
                psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -c "
                    INSERT INTO schema_migrations (version) VALUES ('$migration_name') 
                    ON CONFLICT (version) DO NOTHING;
                " -q
                log_success "迁移 $migration_name 执行成功"
            else
                log_error "迁移 $migration_name 执行失败"
                exit 1
            fi
        fi
    done
    
    log_success "所有迁移执行完成"
}

# 插入测试数据
insert_test_data() {
    log_info "插入测试数据..."
    
    export PGPASSWORD="$DATABASE_PASSWORD"
    
    # 创建测试用例所需的基础数据
    psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -c "
        -- 创建默认租户
        INSERT INTO tenants (id, name, slug, status, created_at, updated_at)
        VALUES 
            (1, 'Default Tenant', 'default', 'active', NOW(), NOW())
        ON CONFLICT (id) DO NOTHING;
        
        -- 创建测试用户
        INSERT INTO users (id, tenant_id, username, email, password_hash, status, created_at, updated_at)
        VALUES 
            (1, 1, 'testuser1', 'test1@example.com', '\$2a\$10\$dummy.hash.for.testing', 'active', NOW(), NOW()),
            (2, 1, 'testuser2', 'test2@example.com', '\$2a\$10\$dummy.hash.for.testing', 'active', NOW(), NOW()),
            (3, 1, 'testuser3', 'test3@example.com', '\$2a\$10\$dummy.hash.for.testing', 'active', NOW(), NOW())
        ON CONFLICT (id) DO NOTHING;
        
        -- 创建测试项目
        INSERT INTO projects (id, tenant_id, name, slug, description, status, owner_id, created_at, updated_at)
        VALUES 
            (1, 1, 'Test Project 1', 'test-project-1', 'Test project for integration tests', 'active', 1, NOW(), NOW()),
            (2, 1, 'Test Project 2', 'test-project-2', 'Another test project', 'active', 2, NOW(), NOW())
        ON CONFLICT (id) DO NOTHING;
        
        -- 创建默认角色
        INSERT INTO roles (id, tenant_id, project_id, name, description, permissions, created_at, updated_at)
        VALUES 
            (1, 1, NULL, 'admin', 'System Administrator', '[\"*\"]'::jsonb, NOW(), NOW()),
            (2, 1, NULL, 'manager', 'Project Manager', '[\"project.read\", \"project.write\", \"user.read\"]'::jsonb, NOW(), NOW()),
            (3, 1, NULL, 'viewer', 'Viewer', '[\"project.read\"]'::jsonb, NOW(), NOW()),
            (4, 1, 1, 'viewer', 'Project Viewer', '[\"project.read\"]'::jsonb, NOW(), NOW())
        ON CONFLICT (id) DO NOTHING;
        
        -- 创建测试团队
        INSERT INTO teams (id, tenant_id, project_id, name, description, is_active, created_by, created_at, updated_at)
        VALUES 
            (1, 1, 1, '测试团队', '这是一个测试团队', true, 1, NOW(), NOW()),
            (2, 1, 1, 'Development Team', 'Main development team', true, 1, NOW(), NOW())
        ON CONFLICT (id) DO NOTHING;
        
        -- 重置序列
        SELECT setval('tenants_id_seq', (SELECT MAX(id) FROM tenants));
        SELECT setval('users_id_seq', (SELECT MAX(id) FROM users));
        SELECT setval('projects_id_seq', (SELECT MAX(id) FROM projects));
        SELECT setval('roles_id_seq', (SELECT MAX(id) FROM roles));
        SELECT setval('teams_id_seq', (SELECT MAX(id) FROM teams));
        
    " -q
    
    log_success "测试数据插入完成"
}

# 验证测试数据库
verify_test_database() {
    log_info "验证测试数据库..."
    
    export PGPASSWORD="$DATABASE_PASSWORD"
    
    # 检查关键表
    TABLES=("tenants" "users" "projects" "tasks" "repositories" "roles" "teams")
    
    for table in "${TABLES[@]}"; do
        TABLE_EXISTS=$(psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -tAc "
            SELECT 1 FROM information_schema.tables WHERE table_name='$table';
        ")
        
        if [ "$TABLE_EXISTS" = "1" ]; then
            # 获取表的记录数
            COUNT=$(psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -tAc "SELECT COUNT(*) FROM $table;")
            log_success "表 $table 存在 (记录数: $COUNT)"
        else
            log_error "表 $table 不存在"
            exit 1
        fi
    done
    
    log_success "测试数据库验证完成"
}

# 生成测试环境配置
generate_test_env() {
    log_info "生成测试环境配置..."
    
    # 获取项目根目录
    PROJECT_ROOT="$(dirname "$(dirname "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)")")"
    TEST_ENV_FILE="$PROJECT_ROOT/.env.test"
    
    cat > "$TEST_ENV_FILE" << EOF
# Test Environment Configuration
# Generated: $(date)

# Database Configuration
DATABASE_HOST=$DATABASE_HOST
DATABASE_PORT=$DATABASE_PORT
DATABASE_NAME=$DATABASE_NAME
DATABASE_USER=$DATABASE_USER
DATABASE_PASSWORD=$DATABASE_PASSWORD
DATABASE_URL=postgresql://$DATABASE_USER:$DATABASE_PASSWORD@$DATABASE_HOST:$DATABASE_PORT/$DATABASE_NAME

# Environment
ENVIRONMENT=test

# JWT Configuration (test only)
JWT_SECRET=test_jwt_secret_key_for_testing_only_minimum_32_characters_long

# Server Configuration
SERVER_PORT=8080
SERVER_HOST=localhost

# Redis Configuration (optional for tests)
REDIS_HOST=localhost
REDIS_PORT=6379

# Disable external services in tests
TWO_FACTOR_ENABLED=false
FEATURE_SSO_ENABLED=false

# Test specific settings
TEST_DATABASE_URL=postgresql://$DATABASE_USER:$DATABASE_PASSWORD@$DATABASE_HOST:$DATABASE_PORT/$DATABASE_NAME
EOF
    
    log_success "测试环境配置已保存到: $TEST_ENV_FILE"
}

# 主函数
main() {
    log_info "初始化测试环境数据库"
    log_info "================================"
    
    setup_test_environment
    ensure_postgresql_running
    recreate_test_database
    install_test_extensions
    run_test_migrations
    insert_test_data
    verify_test_database
    generate_test_env
    
    log_success "================================"
    log_success "测试数据库初始化完成！"
    log_success "数据库: $DATABASE_NAME"
    log_success "用户: $DATABASE_USER"
    log_success "主机: $DATABASE_HOST:$DATABASE_PORT"
    log_info "现在可以运行测试了: go test ./tests/..."
}

# 显示使用说明
show_usage() {
    cat << EOF
测试数据库初始化脚本

用法: $0 [选项]

选项:
    --help, -h              显示此帮助信息

环境变量:
    DATABASE_HOST          数据库主机 (默认: localhost)
    DATABASE_PORT          数据库端口 (默认: 5432)
    DATABASE_NAME          测试数据库名称 (默认: devcollab_test)
    DATABASE_USER          测试数据库用户 (默认: devcollab_test)
    DATABASE_PASSWORD      测试数据库密码 (默认: test_password_2025)
    POSTGRES_ADMIN_USER    PostgreSQL管理员用户 (默认: postgres)
    POSTGRES_ADMIN_PASSWORD PostgreSQL管理员密码 (默认: postgres)

示例:
    # 基本用法
    $0
    
    # 自定义管理员密码
    POSTGRES_ADMIN_PASSWORD=mypassword $0

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

# 执行主函数
main "$@"