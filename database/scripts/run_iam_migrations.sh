#!/bin/bash

# Cloud-Based Collaborative Development Platform
# IAM Service Database Migration Execution Script
# 执行IAM服务相关的数据库迁移

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

# 检查必要工具
check_requirements() {
    log_info "检查必要工具..."
    
    if ! command -v psql &> /dev/null; then
        log_error "psql command not found. Please install PostgreSQL client."
        exit 1
    fi
    
    log_success "所有必要工具已就绪"
}

# 读取数据库配置
load_db_config() {
    local config_file="../config/database.yml"
    
    if [[ -f "$config_file" ]]; then
        log_info "从配置文件加载数据库连接信息: $config_file"
        # 简单解析YAML配置文件（实际项目中应使用更强大的YAML解析器）
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
    DB_USER=${DB_USER:-${POSTGRES_USER:-"postgres"}}
    DB_PASSWORD=${DB_PASSWORD:-${POSTGRES_PASSWORD:-""}}
    
    # 构建连接字符串
    if [[ -n "$DB_PASSWORD" ]]; then
        export PGPASSWORD="$DB_PASSWORD"
    fi
    
    DB_CONNECTION_PARAMS="-h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME"
    
    log_info "数据库连接信息: $DB_USER@$DB_HOST:$DB_PORT/$DB_NAME"
}

# 测试数据库连接
test_db_connection() {
    log_info "测试数据库连接..."
    
    if psql $DB_CONNECTION_PARAMS -c "SELECT version();" > /dev/null 2>&1; then
        log_success "数据库连接成功"
    else
        log_error "无法连接到数据库，请检查连接参数"
        exit 1
    fi
}

# 检查UUID扩展
check_uuid_extension() {
    log_info "检查UUID扩展..."
    
    local has_uuid_ossp=$(psql $DB_CONNECTION_PARAMS -t -c "SELECT count(*) FROM pg_extension WHERE extname = 'uuid-ossp';" 2>/dev/null | tr -d ' ')
    
    if [[ "$has_uuid_ossp" == "0" ]]; then
        log_warning "UUID扩展未安装，正在安装..."
        psql $DB_CONNECTION_PARAMS -c "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";" || {
            log_error "无法安装UUID扩展，请检查数据库权限"
            exit 1
        }
        log_success "UUID扩展安装成功"
    else
        log_success "UUID扩展已安装"
    fi
}

# 检查是否存在必要的函数
check_required_functions() {
    log_info "检查必要的数据库函数..."
    
    # 检查update_updated_at_column函数
    local has_update_function=$(psql $DB_CONNECTION_PARAMS -t -c "SELECT count(*) FROM information_schema.routines WHERE routine_name = 'update_updated_at_column';" 2>/dev/null | tr -d ' ')
    
    if [[ "$has_update_function" == "0" ]]; then
        log_warning "update_updated_at_column函数不存在，正在创建..."
        psql $DB_CONNECTION_PARAMS -c "
        CREATE OR REPLACE FUNCTION update_updated_at_column()
        RETURNS TRIGGER AS \$\$
        BEGIN
            NEW.updated_at = NOW();
            RETURN NEW;
        END;
        \$\$ LANGUAGE plpgsql;
        " || {
            log_error "无法创建必要的数据库函数"
            exit 1
        }
        log_success "必要函数创建成功"
    else
        log_success "必要函数已存在"
    fi
}

# 执行迁移文件
run_migration() {
    local migration_file="$1"
    local migration_name=$(basename "$migration_file" .sql)
    
    log_info "执行迁移: $migration_name"
    
    # 开始事务并执行迁移
    if psql $DB_CONNECTION_PARAMS -f "$migration_file" > /tmp/migration_${migration_name}.log 2>&1; then
        log_success "迁移 $migration_name 执行成功"
        # 如果有重要输出，显示它
        if grep -q "NOTICE" /tmp/migration_${migration_name}.log; then
            grep "NOTICE" /tmp/migration_${migration_name}.log | while read line; do
                log_info "$line"
            done
        fi
    else
        log_error "迁移 $migration_name 执行失败"
        echo "错误详情:"
        cat /tmp/migration_${migration_name}.log
        exit 1
    fi
}

# 验证IAM表结构
verify_iam_tables() {
    log_info "验证IAM表结构..."
    
    local required_tables=("users" "roles" "permissions" "user_roles" "role_permissions" "user_sessions")
    local missing_tables=()
    
    for table in "${required_tables[@]}"; do
        local exists=$(psql $DB_CONNECTION_PARAMS -t -c "SELECT count(*) FROM information_schema.tables WHERE table_name = '$table';" 2>/dev/null | tr -d ' ')
        if [[ "$exists" == "0" ]]; then
            missing_tables+=("$table")
        fi
    done
    
    if [[ ${#missing_tables[@]} -gt 0 ]]; then
        log_error "以下IAM表缺失: ${missing_tables[*]}"
        exit 1
    else
        log_success "所有IAM表都已创建"
    fi
}

# 验证默认数据
verify_default_data() {
    log_info "验证默认数据..."
    
    # 检查默认租户
    local tenant_count=$(psql $DB_CONNECTION_PARAMS -t -c "SELECT count(*) FROM tenants WHERE slug = 'test-tenant';" 2>/dev/null | tr -d ' ')
    
    # 检查默认角色
    local role_count=$(psql $DB_CONNECTION_PARAMS -t -c "SELECT count(*) FROM roles WHERE name IN ('admin', 'user', 'manager', 'developer', 'viewer');" 2>/dev/null | tr -d ' ')
    
    # 检查默认权限
    local permission_count=$(psql $DB_CONNECTION_PARAMS -t -c "SELECT count(*) FROM permissions;" 2>/dev/null | tr -d ' ')
    
    log_info "默认数据统计:"
    log_info "  - 测试租户: $tenant_count"
    log_info "  - 系统角色: $role_count"
    log_info "  - 系统权限: $permission_count"
    
    if [[ "$role_count" -ge "5" ]] && [[ "$permission_count" -ge "10" ]]; then
        log_success "默认数据验证通过"
    else
        log_warning "默认数据可能不完整，请检查迁移日志"
    fi
}

# 运行数据一致性检查
run_consistency_check() {
    log_info "运行数据一致性检查..."
    
    psql $DB_CONNECTION_PARAMS -c "SELECT * FROM check_iam_data_consistency();" 2>/dev/null || {
        log_warning "数据一致性检查函数不可用，跳过检查"
        return
    }
    
    log_success "数据一致性检查完成"
}

# 清理过期数据
cleanup_expired_data() {
    log_info "清理过期数据..."
    
    # 清理过期会话
    local cleaned_sessions=$(psql $DB_CONNECTION_PARAMS -t -c "SELECT cleanup_expired_sessions();" 2>/dev/null | tr -d ' ')
    
    # 解锁过期用户
    local unlocked_users=$(psql $DB_CONNECTION_PARAMS -t -c "SELECT unlock_expired_users();" 2>/dev/null | tr -d ' ')
    
    log_info "清理结果:"
    log_info "  - 清理过期会话: ${cleaned_sessions:-0} 个"
    log_info "  - 解锁过期用户: ${unlocked_users:-0} 个"
    
    log_success "过期数据清理完成"
}

# 主执行流程
main() {
    echo "=============================================="
    echo "IAM Service Database Migration"
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
    check_requirements
    load_db_config
    test_db_connection
    check_uuid_extension
    check_required_functions
    
    # 执行IAM迁移
    log_info "开始执行IAM服务迁移..."
    
    # 执行IAM模式迁移
    if [[ -f "$migration_dir/005_iam_service_schema.sql" ]]; then
        run_migration "$migration_dir/005_iam_service_schema.sql"
    else
        log_error "IAM模式迁移文件不存在: $migration_dir/005_iam_service_schema.sql"
        exit 1
    fi
    
    # 执行默认数据迁移
    if [[ -f "$migration_dir/006_iam_default_data.sql" ]]; then
        run_migration "$migration_dir/006_iam_default_data.sql"
    else
        log_warning "IAM默认数据迁移文件不存在: $migration_dir/006_iam_default_data.sql"
    fi
    
    # 验证结果
    verify_iam_tables
    verify_default_data
    run_consistency_check
    cleanup_expired_data
    
    echo "=============================================="
    log_success "IAM服务数据库迁移完成"
    log_info "您现在可以启动IAM服务进行测试"
    log_info "测试账户:"
    log_info "  管理员: admin@test.com / admin123"
    log_info "  普通用户: user@test.com / user123"
    echo "=============================================="
}

# 脚本入口
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi