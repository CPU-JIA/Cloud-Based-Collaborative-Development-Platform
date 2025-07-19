#!/bin/bash

# Cloud-Based Collaborative Development Platform
# Database Restore Script
# PostgreSQL Multi-Tenant Database Restore
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

# 默认配置
RESTORE_TYPE="${RESTORE_TYPE:-full}"
FORCE_RESTORE="${FORCE_RESTORE:-false}"
CREATE_DATABASE="${CREATE_DATABASE:-true}"
PARALLEL_JOBS="${PARALLEL_JOBS:-4}"

# 检查环境变量
check_environment() {
    log_info "检查环境变量..."
    
    if [ -z "$DATABASE_HOST" ]; then
        export DATABASE_HOST="localhost"
    fi
    
    if [ -z "$DATABASE_PORT" ]; then
        export DATABASE_PORT="5432"
    fi
    
    if [ -z "$DATABASE_NAME" ]; then
        log_error "DATABASE_NAME 必须设置"
        exit 1
    fi
    
    if [ -z "$DATABASE_USER" ]; then
        log_error "DATABASE_USER 必须设置"
        exit 1
    fi
    
    if [ -z "$DATABASE_PASSWORD" ]; then
        log_error "DATABASE_PASSWORD 必须设置"
        exit 1
    fi
    
    # 设置PGPASSWORD环境变量
    export PGPASSWORD="$DATABASE_PASSWORD"
}

# 检查备份文件
check_backup_file() {
    local backup_file="$1"
    
    if [ -z "$backup_file" ]; then
        log_error "必须指定备份文件"
        exit 1
    fi
    
    # 处理加密文件
    if [[ "$backup_file" == *.gpg ]]; then
        log_info "检测到加密备份文件"
        if [ -z "$ENCRYPTION_KEY" ]; then
            log_error "需要设置 ENCRYPTION_KEY 来解密备份文件"
            exit 1
        fi
        
        local decrypted_file="${backup_file%.gpg}"
        log_info "解密备份文件..."
        
        if command -v gpg &> /dev/null; then
            gpg --batch --yes --quiet --decrypt --passphrase "$ENCRYPTION_KEY" --output "$decrypted_file" "$backup_file"
            backup_file="$decrypted_file"
            log_success "备份文件解密完成"
        else
            log_error "gpg 命令未找到，无法解密"
            exit 1
        fi
    fi
    
    # 处理压缩文件
    if [[ "$backup_file" == *.gz ]]; then
        log_info "检测到压缩备份文件"
        local uncompressed_file="${backup_file%.gz}"
        
        log_info "解压备份文件..."
        gunzip -k "$backup_file"
        backup_file="$uncompressed_file"
        log_success "备份文件解压完成"
    fi
    
    if [ ! -f "$backup_file" ]; then
        log_error "备份文件不存在: $backup_file"
        exit 1
    fi
    
    # 验证备份文件
    log_info "验证备份文件: $backup_file"
    local file_size=$(stat -c%s "$backup_file")
    if [ "$file_size" -lt 1024 ]; then
        log_error "备份文件太小，可能已损坏"
        exit 1
    fi
    
    log_success "备份文件验证通过: $(numfmt --to=iec $file_size)"
    echo "$backup_file"
}

# 检查数据库连接
check_database_connection() {
    log_info "检查数据库连接..."
    
    if ! command -v psql &> /dev/null; then
        log_error "psql 命令未找到，请安装PostgreSQL客户端"
        exit 1
    fi
    
    if ! command -v pg_restore &> /dev/null; then
        log_error "pg_restore 命令未找到，请安装PostgreSQL客户端"
        exit 1
    fi
    
    # 检查服务器连接
    if ! psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d postgres -c "SELECT 1;" &> /dev/null; then
        log_error "无法连接到PostgreSQL服务器"
        exit 1
    fi
    
    log_success "数据库连接成功"
}

# 检查目标数据库
check_target_database() {
    log_info "检查目标数据库: $DATABASE_NAME"
    
    # 检查数据库是否存在
    local db_exists=$(psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$DATABASE_NAME'")
    
    if [ "$db_exists" = "1" ]; then
        log_warning "目标数据库已存在: $DATABASE_NAME"
        
        if [ "$FORCE_RESTORE" = "false" ]; then
            read -p "是否要覆盖现有数据库? (y/N): " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                log_info "恢复操作已取消"
                exit 0
            fi
        fi
        
        log_warning "将覆盖现有数据库"
    else
        if [ "$CREATE_DATABASE" = "true" ]; then
            log_info "将创建新数据库: $DATABASE_NAME"
        else
            log_error "目标数据库不存在且未启用自动创建"
            exit 1
        fi
    fi
}

# 创建数据库备份（恢复前）
create_pre_restore_backup() {
    local backup_dir="/tmp/devcollab_pre_restore_$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$backup_dir"
    
    log_info "创建恢复前备份: $backup_dir"
    
    # 检查数据库是否存在
    local db_exists=$(psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$DATABASE_NAME'")
    
    if [ "$db_exists" = "1" ]; then
        pg_dump \
            -h "$DATABASE_HOST" \
            -p "$DATABASE_PORT" \
            -U "$DATABASE_USER" \
            -d "$DATABASE_NAME" \
            --format=custom \
            --compress=9 \
            --file="$backup_dir/pre_restore_backup.dump"
        
        log_success "恢复前备份已创建: $backup_dir/pre_restore_backup.dump"
        echo "$backup_dir/pre_restore_backup.dump"
    else
        log_info "目标数据库不存在，跳过恢复前备份"
        echo ""
    fi
}

# 停止应用连接（可选）
stop_application_connections() {
    log_info "断开应用程序连接..."
    
    # 终止除当前用户外的所有连接
    psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d postgres -c "
        SELECT pg_terminate_backend(pid)
        FROM pg_stat_activity
        WHERE datname = '$DATABASE_NAME'
        AND pid <> pg_backend_pid()
        AND usename <> '$DATABASE_USER';
    " &> /dev/null || true
    
    log_success "应用程序连接已断开"
}

# 执行完整恢复
perform_full_restore() {
    local backup_file="$1"
    
    log_info "开始完整数据库恢复..."
    log_info "备份文件: $backup_file"
    
    # 检查备份文件格式
    if [[ "$backup_file" == *.dump ]]; then
        # 自定义格式备份
        log_info "检测到自定义格式备份，使用 pg_restore"
        
        # 删除现有数据库（如果存在）
        psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d postgres -c "DROP DATABASE IF EXISTS \"$DATABASE_NAME\";" &> /dev/null || true
        
        # 创建新数据库
        psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d postgres -c "CREATE DATABASE \"$DATABASE_NAME\" WITH ENCODING 'UTF8';"
        
        # 恢复数据
        pg_restore \
            -h "$DATABASE_HOST" \
            -p "$DATABASE_PORT" \
            -U "$DATABASE_USER" \
            -d "$DATABASE_NAME" \
            --verbose \
            --clean \
            --if-exists \
            --jobs="$PARALLEL_JOBS" \
            "$backup_file"
    else
        # SQL格式备份
        log_info "检测到SQL格式备份，使用 psql"
        
        # 直接执行SQL文件（假设文件包含DROP/CREATE DATABASE语句）
        psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d postgres -f "$backup_file"
    fi
    
    log_success "完整恢复完成"
}

# 执行Schema恢复
perform_schema_restore() {
    local backup_file="$1"
    
    log_info "开始Schema恢复..."
    log_info "备份文件: $backup_file"
    
    # 确保目标数据库存在
    psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d postgres -c "CREATE DATABASE IF NOT EXISTS \"$DATABASE_NAME\" WITH ENCODING 'UTF8';"
    
    # 执行Schema恢复
    psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -f "$backup_file"
    
    log_success "Schema恢复完成"
}

# 执行数据恢复
perform_data_restore() {
    local backup_file="$1"
    
    log_info "开始数据恢复..."
    log_info "备份文件: $backup_file"
    
    # 确保目标数据库存在
    local db_exists=$(psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$DATABASE_NAME'")
    
    if [ "$db_exists" != "1" ]; then
        log_error "目标数据库不存在，无法进行数据恢复"
        exit 1
    fi
    
    # 禁用触发器以提高性能
    psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -c "SET session_replication_role = replica;"
    
    # 执行数据恢复
    psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -f "$backup_file"
    
    # 重新启用触发器
    psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -c "SET session_replication_role = DEFAULT;"
    
    log_success "数据恢复完成"
}

# 执行租户恢复
perform_tenant_restore() {
    local backup_file="$1"
    local target_tenant_id="$2"
    
    log_info "开始租户恢复..."
    log_info "备份文件: $backup_file"
    log_info "目标租户ID: $target_tenant_id"
    
    # 解压租户备份文件
    if [[ "$backup_file" == *.tar.gz ]]; then
        local temp_dir="/tmp/tenant_restore_$(date +%Y%m%d_%H%M%S)"
        mkdir -p "$temp_dir"
        
        tar -xzf "$backup_file" -C "$temp_dir"
        
        # 设置当前租户
        psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -c "SELECT set_current_tenant('$target_tenant_id');"
        
        # 恢复各个表的数据
        for csv_file in "$temp_dir"/*.csv; do
            if [ -f "$csv_file" ]; then
                local table_name=$(basename "$csv_file" .csv)
                table_name=${table_name#*_tenant_*_*_}  # 移除前缀
                
                log_info "恢复表: $table_name"
                
                # 使用COPY命令导入数据
                psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -c "\\copy $table_name FROM '$csv_file' WITH CSV HEADER;"
            fi
        done
        
        # 清理临时目录
        rm -rf "$temp_dir"
    else
        log_error "租户备份文件格式不正确"
        exit 1
    fi
    
    log_success "租户恢复完成"
}

# 验证恢复结果
verify_restore() {
    log_info "验证恢复结果..."
    
    # 检查数据库是否可访问
    if ! psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -c "SELECT 1;" &> /dev/null; then
        log_error "恢复后无法连接到数据库"
        return 1
    fi
    
    # 检查关键表是否存在
    local tables=("tenants" "users" "projects" "tasks")
    for table in "${tables[@]}"; do
        local table_exists=$(psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -tAc "SELECT 1 FROM information_schema.tables WHERE table_name='$table'")
        
        if [ "$table_exists" = "1" ]; then
            log_success "表 $table 存在"
        else
            log_warning "表 $table 不存在"
        fi
    done
    
    # 检查数据完整性
    local record_count=$(psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -tAc "SELECT COUNT(*) FROM tenants WHERE deleted_at IS NULL" 2>/dev/null || echo "0")
    log_info "活跃租户数量: $record_count"
    
    # 检查RLS是否正确恢复
    local rls_enabled=$(psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -tAc "SELECT COUNT(*) FROM pg_class c JOIN pg_namespace n ON c.relnamespace = n.oid WHERE c.relrowsecurity = true AND n.nspname = 'public'" 2>/dev/null || echo "0")
    
    if [ "$rls_enabled" -gt "0" ]; then
        log_success "行级安全策略已启用"
    else
        log_warning "行级安全策略可能未正确恢复"
    fi
    
    log_success "恢复验证完成"
}

# 重建索引和统计信息
rebuild_indexes_and_stats() {
    log_info "重建索引和更新统计信息..."
    
    # 重建所有索引
    psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -c "REINDEX DATABASE \"$DATABASE_NAME\";" &> /dev/null || log_warning "索引重建可能失败"
    
    # 更新表统计信息
    psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -c "ANALYZE;" &> /dev/null || log_warning "统计信息更新可能失败"
    
    log_success "索引和统计信息更新完成"
}

# 显示使用说明
show_usage() {
    cat << EOF
Cloud-Based Collaborative Development Platform - Database Restore

用法: $0 [选项] <备份文件>

选项:
    --help, -h              显示此帮助信息
    --type TYPE             恢复类型: full, schema, data, tenant (默认: full)
    --tenant-id ID          目标租户ID (仅在type=tenant时需要)
    --force                 强制恢复，不进行确认
    --no-create-db          不自动创建数据库
    --no-pre-backup         跳过恢复前备份
    --no-verify             跳过恢复验证
    --parallel-jobs N       并行作业数 (默认: 4)

环境变量:
    DATABASE_HOST           数据库主机
    DATABASE_PORT           数据库端口
    DATABASE_NAME           数据库名称
    DATABASE_USER           数据库用户
    DATABASE_PASSWORD       数据库密码
    ENCRYPTION_KEY          解密密钥 (如果备份已加密)

示例:
    # 完整恢复
    $0 --type full /path/to/backup.sql
    
    # Schema恢复
    $0 --type schema /path/to/schema_backup.sql
    
    # 租户恢复
    $0 --type tenant --tenant-id "123e4567-e89b-12d3-a456-426614174000" /path/to/tenant_backup.tar.gz
    
    # 强制恢复（不确认）
    $0 --type full --force /path/to/backup.sql
    
    # 恢复加密备份
    ENCRYPTION_KEY="your-secret-key" $0 --type full /path/to/backup.sql.gz.gpg

EOF
}

# 主函数
main() {
    local restore_type="full"
    local tenant_id=""
    local backup_file=""
    local create_pre_backup=true
    local verify_restore_flag=true
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            --help|-h)
                show_usage
                exit 0
                ;;
            --type)
                restore_type="$2"
                shift 2
                ;;
            --tenant-id)
                tenant_id="$2"
                shift 2
                ;;
            --force)
                FORCE_RESTORE="true"
                shift
                ;;
            --no-create-db)
                CREATE_DATABASE="false"
                shift
                ;;
            --no-pre-backup)
                create_pre_backup=false
                shift
                ;;
            --no-verify)
                verify_restore_flag=false
                shift
                ;;
            --parallel-jobs)
                PARALLEL_JOBS="$2"
                shift 2
                ;;
            -*)
                log_error "未知选项: $1"
                show_usage
                exit 1
                ;;
            *)
                backup_file="$1"
                shift
                ;;
        esac
    done
    
    log_info "开始数据库恢复操作"
    log_info "==============================="
    
    check_environment
    backup_file=$(check_backup_file "$backup_file")
    check_database_connection
    check_target_database
    
    # 创建恢复前备份
    if [ "$create_pre_backup" = true ]; then
        local pre_backup_file=$(create_pre_restore_backup)
        if [ -n "$pre_backup_file" ]; then
            log_info "恢复前备份: $pre_backup_file"
        fi
    fi
    
    stop_application_connections
    
    # 根据恢复类型执行相应操作
    case "$restore_type" in
        "full")
            perform_full_restore "$backup_file"
            ;;
        "schema")
            perform_schema_restore "$backup_file"
            ;;
        "data")
            perform_data_restore "$backup_file"
            ;;
        "tenant")
            if [ -z "$tenant_id" ]; then
                log_error "租户恢复需要指定 --tenant-id"
                exit 1
            fi
            perform_tenant_restore "$backup_file" "$tenant_id"
            ;;
        *)
            log_error "不支持的恢复类型: $restore_type"
            exit 1
            ;;
    esac
    
    # 恢复后处理
    if [ "$verify_restore_flag" = true ]; then
        verify_restore
    fi
    
    rebuild_indexes_and_stats
    
    log_success "==============================="
    log_success "数据库恢复完成！"
    log_success "恢复类型: $restore_type"
    log_success "备份文件: $backup_file"
    log_success "目标数据库: $DATABASE_NAME"
    log_info "请重新启动应用程序以建立新的数据库连接"
}

# 执行主函数
main "$@"