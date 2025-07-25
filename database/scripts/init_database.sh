#!/bin/bash

# Cloud-Based Collaborative Development Platform
# Database Initialization Script
# PostgreSQL Multi-Tenant Database Setup
# Generated: 2025-01-19

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查必要的环境变量
check_environment() {
    log_info "检查环境变量..."
    
    if [ -z "$DATABASE_HOST" ]; then
        export DATABASE_HOST="localhost"
        log_warning "DATABASE_HOST 未设置，使用默认值: localhost"
    fi
    
    if [ -z "$DATABASE_PORT" ]; then
        export DATABASE_PORT="5432"
        log_warning "DATABASE_PORT 未设置，使用默认值: 5432"
    fi
    
    if [ -z "$DATABASE_NAME" ]; then
        export DATABASE_NAME="devcollab_development"
        log_warning "DATABASE_NAME 未设置，使用默认值: devcollab_development"
    fi
    
    if [ -z "$DATABASE_USER" ]; then
        export DATABASE_USER="devcollab_dev"
        log_warning "DATABASE_USER 未设置，使用默认值: devcollab_dev"
    fi
    
    if [ -z "$DATABASE_PASSWORD" ]; then
        export DATABASE_PASSWORD="dev_password_2025"
        log_warning "DATABASE_PASSWORD 未设置，使用默认值"
    fi
    
    if [ -z "$POSTGRES_ADMIN_USER" ]; then
        export POSTGRES_ADMIN_USER="postgres"
        log_warning "POSTGRES_ADMIN_USER 未设置，使用默认值: postgres"
    fi
    
    if [ -z "$POSTGRES_ADMIN_PASSWORD" ]; then
        log_error "POSTGRES_ADMIN_PASSWORD 必须设置"
        exit 1
    fi
}

# 检查PostgreSQL连接
check_postgresql() {
    log_info "检查PostgreSQL连接..."
    
    if ! command -v psql &> /dev/null; then
        log_error "psql 命令未找到，请安装PostgreSQL客户端"
        exit 1
    fi
    
    export PGPASSWORD="$POSTGRES_ADMIN_PASSWORD"
    if ! psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$POSTGRES_ADMIN_USER" -d postgres -c "SELECT 1;" &> /dev/null; then
        log_error "无法连接到PostgreSQL服务器"
        log_error "请检查: HOST=$DATABASE_HOST, PORT=$DATABASE_PORT, USER=$POSTGRES_ADMIN_USER"
        exit 1
    fi
    
    log_success "PostgreSQL连接成功"
}

# 创建数据库用户
create_database_user() {
    log_info "创建数据库用户: $DATABASE_USER"
    
    export PGPASSWORD="$POSTGRES_ADMIN_PASSWORD"
    
    # 检查用户是否已存在
    USER_EXISTS=$(psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$POSTGRES_ADMIN_USER" -d postgres -tAc "SELECT 1 FROM pg_roles WHERE rolname='$DATABASE_USER'")
    
    if [ "$USER_EXISTS" = "1" ]; then
        log_warning "用户 $DATABASE_USER 已存在，跳过创建"
    else
        psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$POSTGRES_ADMIN_USER" -d postgres -c "
            CREATE USER \"$DATABASE_USER\" WITH 
            PASSWORD '$DATABASE_PASSWORD'
            CREATEDB
            NOCREATEROLE
            NOSUPERUSER;
        "
        log_success "用户 $DATABASE_USER 创建成功"
    fi
}

# 创建数据库
create_database() {
    log_info "创建数据库: $DATABASE_NAME"
    
    export PGPASSWORD="$POSTGRES_ADMIN_PASSWORD"
    
    # 检查数据库是否已存在
    DB_EXISTS=$(psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$POSTGRES_ADMIN_USER" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$DATABASE_NAME'")
    
    if [ "$DB_EXISTS" = "1" ]; then
        log_warning "数据库 $DATABASE_NAME 已存在"
        read -p "是否删除并重新创建? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            log_info "删除现有数据库..."
            psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$POSTGRES_ADMIN_USER" -d postgres -c "DROP DATABASE \"$DATABASE_NAME\";"
            log_success "数据库已删除"
        else
            log_info "使用现有数据库"
            return 0
        fi
    fi
    
    # 创建数据库
    psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$POSTGRES_ADMIN_USER" -d postgres -c "
        CREATE DATABASE \"$DATABASE_NAME\" 
        WITH OWNER \"$DATABASE_USER\"
        ENCODING 'UTF8'
        LC_COLLATE 'en_US.UTF-8'
        LC_CTYPE 'en_US.UTF-8'
        TEMPLATE template0;
    "
    
    # 授予权限
    psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$POSTGRES_ADMIN_USER" -d "$DATABASE_NAME" -c "
        GRANT ALL PRIVILEGES ON DATABASE \"$DATABASE_NAME\" TO \"$DATABASE_USER\";
        GRANT ALL PRIVILEGES ON SCHEMA public TO \"$DATABASE_USER\";
    "
    
    log_success "数据库 $DATABASE_NAME 创建成功"
}

# 安装必要的扩展
install_extensions() {
    log_info "安装PostgreSQL扩展..."
    
    export PGPASSWORD="$POSTGRES_ADMIN_PASSWORD"
    
    # 安装uuid-ossp扩展
    psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$POSTGRES_ADMIN_USER" -d "$DATABASE_NAME" -c "
        CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";
        CREATE EXTENSION IF NOT EXISTS \"pgcrypto\";
        CREATE EXTENSION IF NOT EXISTS \"citext\";
    " 2>/dev/null || {
        log_warning "某些扩展安装失败，这可能不会影响基本功能"
    }
    
    log_success "扩展安装完成"
}

# 执行数据库迁移
run_migrations() {
    log_info "执行数据库迁移..."
    
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
    
    # 按顺序执行迁移文件
    for migration_file in "$MIGRATIONS_DIR"/*.sql; do
        if [ -f "$migration_file" ]; then
            migration_name=$(basename "$migration_file" .sql)
            
            # 检查迁移是否已执行
            MIGRATION_EXISTS=$(psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -tAc "SELECT 1 FROM schema_migrations WHERE version='$migration_name'")
            
            if [ "$MIGRATION_EXISTS" = "1" ]; then
                log_info "迁移 $migration_name 已执行，跳过"
            else
                log_info "执行迁移: $migration_name"
                
                if psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -f "$migration_file"; then
                    # 记录迁移历史
                    psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -c "
                        INSERT INTO schema_migrations (version) VALUES ('$migration_name');
                    "
                    log_success "迁移 $migration_name 执行成功"
                else
                    log_error "迁移 $migration_name 执行失败"
                    exit 1
                fi
            fi
        fi
    done
    
    log_success "所有迁移执行完成"
}

# 验证数据库结构
verify_database() {
    log_info "验证数据库结构..."
    
    export PGPASSWORD="$DATABASE_PASSWORD"
    
    # 检查关键表是否存在
    TABLES=("tenants" "users" "projects" "tasks" "repositories" "roles")
    
    for table in "${TABLES[@]}"; do
        TABLE_EXISTS=$(psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -tAc "SELECT 1 FROM information_schema.tables WHERE table_name='$table'")
        
        if [ "$TABLE_EXISTS" = "1" ]; then
            log_success "表 $table 存在"
        else
            log_error "表 $table 不存在"
            exit 1
        fi
    done
    
    # 检查RLS是否启用
    RLS_TABLES=$(psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -tAc "
        SELECT count(*) FROM pg_class c 
        JOIN pg_namespace n ON c.relnamespace = n.oid 
        WHERE c.relrowsecurity = true 
        AND n.nspname = 'public'
        AND c.relname IN ('tenants', 'projects', 'tasks');
    ")
    
    if [ "$RLS_TABLES" -ge "3" ]; then
        log_success "行级安全策略已启用"
    else
        log_warning "行级安全策略可能未正确启用"
    fi
    
    log_success "数据库结构验证完成"
}

# 设置数据库连接信息
setup_connection_info() {
    log_info "生成数据库连接信息..."
    
    CONNECTION_FILE="$(dirname "$(dirname "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)")")/database_connection.env"
    
    cat > "$CONNECTION_FILE" << EOF
# Database Connection Information
# Generated: $(date)

DATABASE_HOST=$DATABASE_HOST
DATABASE_PORT=$DATABASE_PORT
DATABASE_NAME=$DATABASE_NAME
DATABASE_USER=$DATABASE_USER
DATABASE_PASSWORD=$DATABASE_PASSWORD

# Connection URL
DATABASE_URL="postgresql://$DATABASE_USER:$DATABASE_PASSWORD@$DATABASE_HOST:$DATABASE_PORT/$DATABASE_NAME"

# For Go applications
export DATABASE_HOST="$DATABASE_HOST"
export DATABASE_PORT="$DATABASE_PORT"
export DATABASE_NAME="$DATABASE_NAME"
export DATABASE_USER="$DATABASE_USER"
export DATABASE_PASSWORD="$DATABASE_PASSWORD"
export DATABASE_URL="postgresql://$DATABASE_USER:$DATABASE_PASSWORD@$DATABASE_HOST:$DATABASE_PORT/$DATABASE_NAME"
EOF
    
    log_success "连接信息已保存到: $CONNECTION_FILE"
}

# 显示使用说明
show_usage() {
    cat << EOF
Cloud-Based Collaborative Development Platform - Database Initialization

用法: $0 [选项]

选项:
    --help, -h              显示此帮助信息
    --check-only           仅检查连接，不执行初始化
    --no-migrations        跳过数据库迁移
    --no-seed-data         跳过种子数据插入
    --force                强制重新创建数据库

环境变量:
    DATABASE_HOST          数据库主机 (默认: localhost)
    DATABASE_PORT          数据库端口 (默认: 5432)
    DATABASE_NAME          数据库名称 (默认: devcollab_development)
    DATABASE_USER          数据库用户 (默认: devcollab_dev)
    DATABASE_PASSWORD      数据库密码 (默认: dev_password_2025)
    POSTGRES_ADMIN_USER    PostgreSQL管理员用户 (默认: postgres)
    POSTGRES_ADMIN_PASSWORD PostgreSQL管理员密码 (必需)

示例:
    # 基本初始化
    POSTGRES_ADMIN_PASSWORD=admin123 $0
    
    # 自定义数据库配置
    DATABASE_NAME=my_devcollab \\
    DATABASE_USER=my_user \\
    DATABASE_PASSWORD=my_pass \\
    POSTGRES_ADMIN_PASSWORD=admin123 \\
    $0
    
    # 仅检查连接
    POSTGRES_ADMIN_PASSWORD=admin123 $0 --check-only

EOF
}

# 主函数
main() {
    local check_only=false
    local no_migrations=false
    local no_seed_data=false
    local force_recreate=false
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            --help|-h)
                show_usage
                exit 0
                ;;
            --check-only)
                check_only=true
                shift
                ;;
            --no-migrations)
                no_migrations=true
                shift
                ;;
            --no-seed-data)
                no_seed_data=true
                shift
                ;;
            --force)
                force_recreate=true
                shift
                ;;
            *)
                log_error "未知选项: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    log_info "开始初始化 Cloud-Based Collaborative Development Platform 数据库"
    log_info "=========================================================="
    
    # 执行初始化步骤
    check_environment
    check_postgresql
    
    if [ "$check_only" = true ]; then
        log_success "连接检查完成"
        exit 0
    fi
    
    create_database_user
    create_database
    install_extensions
    
    if [ "$no_migrations" = false ]; then
        run_migrations
    else
        log_info "跳过数据库迁移"
    fi
    
    verify_database
    setup_connection_info
    
    log_success "=========================================================="
    log_success "数据库初始化完成！"
    log_success "数据库: $DATABASE_NAME"
    log_success "用户: $DATABASE_USER"
    log_success "主机: $DATABASE_HOST:$DATABASE_PORT"
    log_info "连接信息已保存到 database_connection.env 文件"
    log_info "现在可以启动应用程序了！"
}

# 执行主函数
main "$@"