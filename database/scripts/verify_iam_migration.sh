#!/bin/bash

# Cloud-Based Collaborative Development Platform
# IAM Service Migration Verification Script
# 验证IAM服务数据库迁移是否成功

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

# 读取数据库配置
load_db_config() {
    local config_file="../config/database.yml"
    
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
}

# 测试数据库连接
test_db_connection() {
    log_info "测试数据库连接..."
    
    if psql $DB_CONNECTION_PARAMS -c "SELECT version();" > /dev/null 2>&1; then
        log_success "数据库连接成功"
        return 0
    else
        log_error "无法连接到数据库"
        return 1
    fi
}

# 检查IAM表是否存在
verify_tables() {
    log_info "验证IAM表结构..."
    
    local required_tables=("users" "roles" "permissions" "user_roles" "role_permissions" "user_sessions")
    local verification_results=()
    local missing_tables=()
    
    for table in "${required_tables[@]}"; do
        local exists=$(psql $DB_CONNECTION_PARAMS -t -c "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = '$table');" 2>/dev/null | tr -d ' ')
        
        if [[ "$exists" == "t" ]]; then
            local row_count=$(psql $DB_CONNECTION_PARAMS -t -c "SELECT count(*) FROM $table;" 2>/dev/null | tr -d ' ')
            verification_results+=("✓ $table (行数: $row_count)")
        else
            missing_tables+=("$table")
            verification_results+=("✗ $table (缺失)")
        fi
    done
    
    echo
    log_info "表验证结果:"
    for result in "${verification_results[@]}"; do
        if [[ "$result" == *"✓"* ]]; then
            echo -e "  ${GREEN}$result${NC}"
        else
            echo -e "  ${RED}$result${NC}"
        fi
    done
    echo
    
    if [[ ${#missing_tables[@]} -gt 0 ]]; then
        log_error "以下IAM表缺失: ${missing_tables[*]}"
        return 1
    else
        log_success "所有IAM表都已创建"
        return 0
    fi
}

# 检查索引
verify_indexes() {
    log_info "验证索引创建..."
    
    local important_indexes=(
        "idx_users_email_active"
        "idx_users_tenant_id"
        "idx_roles_tenant_id"
        "idx_permissions_tenant_id"
        "idx_user_roles_user_id"
        "idx_user_sessions_user_id"
    )
    
    local missing_indexes=()
    
    for index in "${important_indexes[@]}"; do
        local exists=$(psql $DB_CONNECTION_PARAMS -t -c "SELECT EXISTS (SELECT FROM pg_indexes WHERE indexname = '$index');" 2>/dev/null | tr -d ' ')
        
        if [[ "$exists" != "t" ]]; then
            missing_indexes+=("$index")
        fi
    done
    
    if [[ ${#missing_indexes[@]} -gt 0 ]]; then
        log_warning "以下重要索引缺失: ${missing_indexes[*]}"
        return 1
    else
        log_success "重要索引都已创建"
        return 0
    fi
}

# 检查外键约束
verify_foreign_keys() {
    log_info "验证外键约束..."
    
    local important_fks=(
        "fk_user_roles_user_id"
        "fk_user_roles_role_id"
        "fk_role_permissions_role_id"
        "fk_role_permissions_permission_id"
        "fk_user_sessions_user_id"
    )
    
    local missing_fks=()
    
    for fk in "${important_fks[@]}"; do
        local exists=$(psql $DB_CONNECTION_PARAMS -t -c "SELECT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = '$fk' AND constraint_type = 'FOREIGN KEY');" 2>/dev/null | tr -d ' ')
        
        if [[ "$exists" != "t" ]]; then
            missing_fks+=("$fk")
        fi
    done
    
    if [[ ${#missing_fks[@]} -gt 0 ]]; then
        log_warning "以下外键约束缺失: ${missing_fks[*]}"
        return 1
    else
        log_success "外键约束都已创建"
        return 0
    fi
}

# 检查RLS策略
verify_rls_policies() {
    log_info "验证行级安全策略..."
    
    local tables_with_rls=("users" "roles" "permissions" "user_roles" "role_permissions" "user_sessions")
    local rls_issues=()
    
    for table in "${tables_with_rls[@]}"; do
        # 检查RLS是否启用
        local rls_enabled=$(psql $DB_CONNECTION_PARAMS -t -c "SELECT relrowsecurity FROM pg_class WHERE relname = '$table';" 2>/dev/null | tr -d ' ')
        
        if [[ "$rls_enabled" != "t" ]]; then
            rls_issues+=("$table: RLS未启用")
            continue
        fi
        
        # 检查是否有策略
        local policy_count=$(psql $DB_CONNECTION_PARAMS -t -c "SELECT count(*) FROM pg_policies WHERE tablename = '$table';" 2>/dev/null | tr -d ' ')
        
        if [[ "$policy_count" == "0" ]]; then
            rls_issues+=("$table: 缺少RLS策略")
        fi
    done
    
    if [[ ${#rls_issues[@]} -gt 0 ]]; then
        log_warning "RLS配置问题:"
        for issue in "${rls_issues[@]}"; do
            echo -e "  ${YELLOW}$issue${NC}"
        done
        return 1
    else
        log_success "行级安全策略配置正确"
        return 0
    fi
}

# 检查默认数据
verify_default_data() {
    log_info "验证默认数据..."
    
    # 检查默认租户
    local tenant_count=$(psql $DB_CONNECTION_PARAMS -t -c "SELECT count(*) FROM tenants WHERE slug = 'test-tenant';" 2>/dev/null | tr -d ' ')
    
    # 检查默认角色
    local role_count=$(psql $DB_CONNECTION_PARAMS -t -c "SELECT count(*) FROM roles WHERE name IN ('admin', 'user', 'manager', 'developer', 'viewer');" 2>/dev/null | tr -d ' ')
    
    # 检查默认权限
    local permission_count=$(psql $DB_CONNECTION_PARAMS -t -c "SELECT count(*) FROM permissions;" 2>/dev/null | tr -d ' ')
    
    # 检查默认用户
    local user_count=$(psql $DB_CONNECTION_PARAMS -t -c "SELECT count(*) FROM users WHERE email IN ('admin@test.com', 'user@test.com');" 2>/dev/null | tr -d ' ')
    
    echo
    log_info "默认数据统计:"
    echo -e "  测试租户: ${tenant_count:-0}"
    echo -e "  系统角色: ${role_count:-0}"
    echo -e "  系统权限: ${permission_count:-0}"
    echo -e "  测试用户: ${user_count:-0}"
    echo
    
    local data_issues=()
    
    if [[ "${tenant_count:-0}" == "0" ]]; then
        data_issues+=("缺少测试租户")
    fi
    
    if [[ "${role_count:-0}" -lt "5" ]]; then
        data_issues+=("系统角色不完整（期望5个，实际${role_count:-0}个）")
    fi
    
    if [[ "${permission_count:-0}" -lt "10" ]]; then
        data_issues+=("系统权限不完整（期望至少10个，实际${permission_count:-0}个）")
    fi
    
    if [[ "${user_count:-0}" -lt "2" ]]; then
        data_issues+=("测试用户不完整（期望2个，实际${user_count:-0}个）")
    fi
    
    if [[ ${#data_issues[@]} -gt 0 ]]; then
        log_warning "默认数据问题:"
        for issue in "${data_issues[@]}"; do
            echo -e "  ${YELLOW}$issue${NC}"
        done
        return 1
    else
        log_success "默认数据验证通过"
        return 0
    fi
}

# 检查必要函数
verify_functions() {
    log_info "验证数据库函数..."
    
    local required_functions=(
        "check_iam_data_consistency"
        "cleanup_expired_sessions"
        "unlock_expired_users"
        "update_updated_at_column"
    )
    
    local missing_functions=()
    
    for func in "${required_functions[@]}"; do
        local exists=$(psql $DB_CONNECTION_PARAMS -t -c "SELECT EXISTS (SELECT FROM information_schema.routines WHERE routine_name = '$func');" 2>/dev/null | tr -d ' ')
        
        if [[ "$exists" != "t" ]]; then
            missing_functions+=("$func")
        fi
    done
    
    if [[ ${#missing_functions[@]} -gt 0 ]]; then
        log_warning "以下必要函数缺失: ${missing_functions[*]}"
        return 1
    else
        log_success "数据库函数都已创建"
        return 0
    fi
}

# 运行数据一致性检查
run_consistency_check() {
    log_info "运行数据一致性检查..."
    
    local check_result=$(psql $DB_CONNECTION_PARAMS -t -c "SELECT check_name, status, message FROM check_iam_data_consistency();" 2>/dev/null)
    
    if [[ -z "$check_result" ]]; then
        log_warning "数据一致性检查函数不可用"
        return 1
    fi
    
    echo
    log_info "一致性检查结果:"
    echo "$check_result" | while read line; do
        if [[ "$line" == *"PASS"* ]]; then
            echo -e "  ${GREEN}$line${NC}"
        elif [[ "$line" == *"FAIL"* ]]; then
            echo -e "  ${RED}$line${NC}"
        else
            echo -e "  ${YELLOW}$line${NC}"
        fi
    done
    echo
    
    local fail_count=$(echo "$check_result" | grep -c "FAIL" || true)
    
    if [[ "$fail_count" -gt "0" ]]; then
        log_warning "发现 $fail_count 个数据一致性问题"
        return 1
    else
        log_success "数据一致性检查通过"
        return 0
    fi
}

# 测试基本IAM功能
test_basic_functionality() {
    log_info "测试基本IAM功能..."
    
    # 设置测试租户上下文
    psql $DB_CONNECTION_PARAMS -c "SELECT set_config('app.current_tenant_id', '550e8400-e29b-41d4-a716-446655440000', true);" > /dev/null 2>&1
    
    # 测试查询用户
    local user_query_result=$(psql $DB_CONNECTION_PARAMS -t -c "SELECT count(*) FROM users WHERE email = 'admin@test.com';" 2>/dev/null | tr -d ' ')
    
    # 测试查询角色权限
    local role_permission_result=$(psql $DB_CONNECTION_PARAMS -t -c "SELECT count(*) FROM role_permissions rp JOIN roles r ON rp.role_id = r.id WHERE r.name = 'admin';" 2>/dev/null | tr -d ' ')
    
    local test_issues=()
    
    if [[ "${user_query_result:-0}" == "0" ]]; then
        test_issues+=("无法查询到管理员用户")
    fi
    
    if [[ "${role_permission_result:-0}" == "0" ]]; then
        test_issues+=("管理员角色没有权限分配")
    fi
    
    if [[ ${#test_issues[@]} -gt 0 ]]; then
        log_warning "基本功能测试问题:"
        for issue in "${test_issues[@]}"; do
            echo -e "  ${YELLOW}$issue${NC}"
        done
        return 1
    else
        log_success "基本IAM功能测试通过"
        return 0
    fi
}

# 生成验证报告
generate_report() {
    local overall_status="$1"
    
    echo
    echo "=============================================="
    echo "IAM服务迁移验证报告"
    echo "=============================================="
    echo "验证时间: $(date)"
    echo "数据库: $DB_HOST:$DB_PORT/$DB_NAME"
    echo
    
    if [[ "$overall_status" == "success" ]]; then
        echo -e "总体状态: ${GREEN}✓ 验证通过${NC}"
        echo
        echo "您的IAM服务数据库迁移已成功完成！"
        echo
        echo "测试账户信息:"
        echo "  管理员: admin@test.com / admin123"
        echo "  普通用户: user@test.com / user123"
        echo
        echo "下一步:"
        echo "  1. 启动IAM服务: go run cmd/iam-service/main.go"
        echo "  2. 测试API端点: curl http://localhost:8080/api/v1/health"
        echo "  3. 登录测试: POST /api/v1/auth/login"
    else
        echo -e "总体状态: ${RED}✗ 验证失败${NC}"
        echo
        echo "请检查上述错误信息并重新运行迁移脚本："
        echo "  cd database && ./scripts/run_iam_migrations.sh"
    fi
    
    echo "=============================================="
}

# 主执行流程
main() {
    echo "=============================================="
    echo "IAM Service Migration Verification"
    echo "=============================================="
    
    local overall_status="success"
    local step_count=0
    local success_count=0
    
    # 加载配置并测试连接
    load_db_config
    
    if ! test_db_connection; then
        generate_report "failed"
        exit 1
    fi
    
    # 执行验证步骤
    local verification_steps=(
        "verify_tables:验证表结构"
        "verify_indexes:验证索引"
        "verify_foreign_keys:验证外键"
        "verify_rls_policies:验证RLS策略"
        "verify_functions:验证函数"
        "verify_default_data:验证默认数据"
        "run_consistency_check:数据一致性检查"
        "test_basic_functionality:基本功能测试"
    )
    
    for step_info in "${verification_steps[@]}"; do
        local func_name="${step_info%%:*}"
        local step_name="${step_info##*:}"
        
        ((step_count++))
        
        echo
        echo "[$step_count/${#verification_steps[@]}] $step_name"
        echo "----------------------------------------"
        
        if $func_name; then
            ((success_count++))
        else
            overall_status="failed"
        fi
    done
    
    # 生成报告
    generate_report "$overall_status"
    
    # 返回适当的退出码
    if [[ "$overall_status" == "success" ]]; then
        exit 0
    else
        exit 1
    fi
}

# 脚本入口
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi