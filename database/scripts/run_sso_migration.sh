#!/bin/bash

# Cloud-Based Collaborative Development Platform
# SSO Database Migration Execution Script
# 执行SSO相关的数据库迁移

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

# 检查是否在Docker环境中运行
check_environment() {
    log_info "检查运行环境..."
    
    if command -v docker &> /dev/null && docker ps &> /dev/null; then
        log_info "检测到Docker环境"
        USE_DOCKER=true
    else
        log_info "使用本地环境"
        USE_DOCKER=false
    fi
}

# 读取数据库配置
load_db_config() {
    local config_file="../config/database.yml"
    
    if [[ -f "$config_file" ]]; then
        log_info "从配置文件加载数据库连接信息: $config_file"
        DB_HOST=$(grep -E "^\s*host:" "$config_file" | sed 's/.*host:\s*//' | tr -d '"' | head -1)
        DB_PORT=$(grep -E "^\s*port:" "$config_file" | sed 's/.*port:\s*//' | tr -d '"' | head -1)
        DB_NAME=$(grep -E "^\s*database:" "$config_file" | sed 's/.*database:\s*//' | tr -d '"' | head -1)
        DB_USER=$(grep -E "^\s*username:" "$config_file" | sed 's/.*username:\s*//' | tr -d '"' | head -1)
        DB_PASSWORD=$(grep -E "^\s*password:" "$config_file" | sed 's/.*password:\s*//' | tr -d '"' | head -1)
    else
        log_warning "配置文件不存在，使用环境变量或默认值"
    fi
    
    # 设置默认值或从环境变量读取
    DB_HOST=${DB_HOST:-${POSTGRES_HOST:-"localhost"}}
    DB_PORT=${DB_PORT:-${POSTGRES_PORT:-"5432"}}
    DB_NAME=${DB_NAME:-${POSTGRES_DB:-"collaborative_platform"}}
    DB_USER=${DB_USER:-${POSTGRES_USER:-"platform_user"}}
    DB_PASSWORD=${DB_PASSWORD:-${POSTGRES_PASSWORD:-"platform_password"}}
    
    # 构建连接字符串
    if [[ -n "$DB_PASSWORD" ]]; then
        export PGPASSWORD="$DB_PASSWORD"
    fi
    
    DB_CONNECTION_PARAMS="-h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME"
    
    log_info "数据库连接信息: $DB_USER@$DB_HOST:$DB_PORT/$DB_NAME"
}

# 执行SQL语句
execute_sql() {
    local sql_content="$1"
    local description="$2"
    
    if [[ "$USE_DOCKER" == "true" ]]; then
        # 使用Docker执行
        echo "$sql_content" | docker exec -i $(docker ps -q -f name=postgres) psql -U "$DB_USER" -d "$DB_NAME"
    else
        # 使用本地psql
        if command -v psql &> /dev/null; then
            echo "$sql_content" | psql $DB_CONNECTION_PARAMS
        else
            log_error "psql命令未找到，请安装PostgreSQL客户端或使用Docker"
            exit 1
        fi
    fi
}

# 执行迁移文件
run_migration() {
    local migration_file="$1"
    local migration_name=$(basename "$migration_file" .sql)
    
    log_info "执行迁移: $migration_name"
    
    if [[ ! -f "$migration_file" ]]; then
        log_error "迁移文件不存在: $migration_file"
        exit 1
    fi
    
    # 执行迁移
    if [[ "$USE_DOCKER" == "true" ]]; then
        # 使用Docker执行
        if docker exec -i $(docker ps -q -f name=postgres) psql -U "$DB_USER" -d "$DB_NAME" -f "/docker-entrypoint-initdb.d/$(basename "$migration_file")" > /tmp/migration_${migration_name}.log 2>&1; then
            log_success "迁移 $migration_name 执行成功"
        else
            # 如果文件不在容器中，尝试直接通过stdin传递
            if cat "$migration_file" | docker exec -i $(docker ps -q -f name=postgres) psql -U "$DB_USER" -d "$DB_NAME" > /tmp/migration_${migration_name}.log 2>&1; then
                log_success "迁移 $migration_name 执行成功"
            else
                log_error "迁移 $migration_name 执行失败"
                cat /tmp/migration_${migration_name}.log
                exit 1
            fi
        fi
    else
        # 使用本地psql
        if psql $DB_CONNECTION_PARAMS -f "$migration_file" > /tmp/migration_${migration_name}.log 2>&1; then
            log_success "迁移 $migration_name 执行成功"
        else
            log_error "迁移 $migration_name 执行失败"
            cat /tmp/migration_${migration_name}.log
            exit 1
        fi
    fi
    
    # 显示重要输出
    if [[ -f "/tmp/migration_${migration_name}.log" ]] && grep -q "NOTICE\|INSERT\|CREATE" /tmp/migration_${migration_name}.log; then
        grep "NOTICE\|INSERT\|CREATE" /tmp/migration_${migration_name}.log | head -10 | while read line; do
            log_info "$line"
        done
    fi
}

# 测试数据库连接
test_db_connection() {
    log_info "测试数据库连接..."
    
    local test_sql="SELECT version();"
    
    if execute_sql "$test_sql" "测试连接" > /dev/null 2>&1; then
        log_success "数据库连接成功"
    else
        log_error "无法连接到数据库，请检查连接参数"
        exit 1
    fi
}

# 验证SSO表结构
verify_sso_tables() {
    log_info "验证SSO表结构..."
    
    local required_tables=("sso_providers" "sso_sessions" "sso_user_mappings")
    local missing_tables=()
    
    for table in "${required_tables[@]}"; do
        local check_sql="SELECT count(*) FROM information_schema.tables WHERE table_name = '$table';"
        local exists=$(execute_sql "$check_sql" "检查表 $table" 2>/dev/null | tail -1 | tr -d ' ')
        
        if [[ "$exists" != "1" ]]; then
            missing_tables+=("$table")
        fi
    done
    
    if [[ ${#missing_tables[@]} -gt 0 ]]; then
        log_error "以下SSO表缺失: ${missing_tables[*]}"
        exit 1
    else
        log_success "所有SSO表都已创建"
    fi
}

# 验证SSO示例数据
verify_sso_data() {
    log_info "验证SSO示例数据..."
    
    local provider_sql="SELECT count(*) FROM sso_providers;"
    local provider_count=$(execute_sql "$provider_sql" "检查SSO提供商" 2>/dev/null | tail -1 | tr -d ' ')
    
    log_info "SSO数据统计:"
    log_info "  - SSO提供商: ${provider_count:-0}"
    
    if [[ "${provider_count:-0}" -ge "1" ]]; then
        log_success "SSO示例数据验证通过"
    else
        log_warning "SSO示例数据可能不完整"
    fi
}

# 清理过期SSO会话
cleanup_expired_sso_sessions() {
    log_info "清理过期SSO会话..."
    
    local cleanup_sql="SELECT cleanup_expired_sso_sessions();"
    local cleaned_sessions=$(execute_sql "$cleanup_sql" "清理过期会话" 2>/dev/null | tail -1 | tr -d ' ')
    
    log_info "清理结果:"
    log_info "  - 清理过期SSO会话: ${cleaned_sessions:-0} 个"
    
    log_success "过期SSO会话清理完成"
}

# 主执行流程
main() {
    echo "=============================================="
    echo "SSO Database Migration"
    echo "=============================================="
    
    # 检查命令行参数
    local migration_dir="../migrations"
    local force_mode=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --migration-dir)
                migration_dir="$2"
                shift 2
                ;;
            --force)
                force_mode=true
                shift
                ;;
            --help)
                echo "Usage: $0 [options]"
                echo "Options:"
                echo "  --migration-dir DIR    指定迁移文件目录 (默认: ../migrations)"
                echo "  --force               强制执行，忽略警告"
                echo "  --help                显示帮助信息"
                exit 0
                ;;
            *)
                log_error "未知参数: $1"
                exit 1
                ;;
        esac
    done
    
    # 执行步骤
    check_environment
    load_db_config
    test_db_connection
    
    # 执行SSO迁移
    log_info "开始执行SSO数据库迁移..."
    
    # 执行SSO模式迁移
    if [[ -f "$migration_dir/007_sso_schema.sql" ]]; then
        run_migration "$migration_dir/007_sso_schema.sql"
    else
        log_error "SSO模式迁移文件不存在: $migration_dir/007_sso_schema.sql"
        exit 1
    fi
    
    # 验证结果
    verify_sso_tables
    verify_sso_data
    cleanup_expired_sso_sessions
    
    echo "=============================================="
    log_success "SSO数据库迁移完成"
    log_info "SSO功能现在可以使用，支持以下协议:"
    log_info "  - SAML 2.0"
    log_info "  - OAuth 2.0"
    log_info "  - OpenID Connect (OIDC)"
    log_info ""
    log_info "示例SSO提供商已创建，请在生产环境中更新配置"
    echo "=============================================="
}

# 脚本入口
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi