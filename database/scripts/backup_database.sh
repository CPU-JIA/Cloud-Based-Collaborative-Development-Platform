#!/bin/bash

# Cloud-Based Collaborative Development Platform
# Database Backup Script
# PostgreSQL Multi-Tenant Database Backup
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
BACKUP_DIR="${BACKUP_DIR:-/var/backups/devcollab}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"
COMPRESSION="${COMPRESSION:-true}"
ENCRYPTION="${ENCRYPTION:-false}"
ENCRYPTION_KEY="${ENCRYPTION_KEY}"
BACKUP_TYPE="${BACKUP_TYPE:-full}"
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

# 创建备份目录
create_backup_directory() {
    log_info "创建备份目录: $BACKUP_DIR"
    
    if [ ! -d "$BACKUP_DIR" ]; then
        mkdir -p "$BACKUP_DIR"
        chmod 750 "$BACKUP_DIR"
    fi
    
    # 创建日期子目录
    DATE_DIR="$BACKUP_DIR/$(date +%Y-%m-%d)"
    if [ ! -d "$DATE_DIR" ]; then
        mkdir -p "$DATE_DIR"
    fi
    
    export BACKUP_DATE_DIR="$DATE_DIR"
}

# 检查数据库连接
check_database_connection() {
    log_info "检查数据库连接..."
    
    if ! command -v pg_dump &> /dev/null; then
        log_error "pg_dump 命令未找到，请安装PostgreSQL客户端"
        exit 1
    fi
    
    if ! psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -c "SELECT 1;" &> /dev/null; then
        log_error "无法连接到数据库"
        exit 1
    fi
    
    log_success "数据库连接成功"
}

# 执行完整备份
perform_full_backup() {
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="$BACKUP_DATE_DIR/devcollab_full_${timestamp}.sql"
    
    log_info "开始完整数据库备份..."
    log_info "备份文件: $backup_file"
    
    # 执行pg_dump
    pg_dump \
        -h "$DATABASE_HOST" \
        -p "$DATABASE_PORT" \
        -U "$DATABASE_USER" \
        -d "$DATABASE_NAME" \
        --verbose \
        --no-password \
        --format=custom \
        --compress=9 \
        --jobs="$PARALLEL_JOBS" \
        --file="$backup_file.dump"
    
    # 同时创建SQL格式的备份
    pg_dump \
        -h "$DATABASE_HOST" \
        -p "$DATABASE_PORT" \
        -U "$DATABASE_USER" \
        -d "$DATABASE_NAME" \
        --verbose \
        --no-password \
        --format=plain \
        --clean \
        --if-exists \
        --create \
        > "$backup_file"
    
    log_success "完整备份完成"
    echo "$backup_file" > "$BACKUP_DATE_DIR/latest_full_backup.txt"
}

# 执行Schema备份
perform_schema_backup() {
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="$BACKUP_DATE_DIR/devcollab_schema_${timestamp}.sql"
    
    log_info "开始Schema备份..."
    log_info "备份文件: $backup_file"
    
    # 执行schema-only备份
    pg_dump \
        -h "$DATABASE_HOST" \
        -p "$DATABASE_PORT" \
        -U "$DATABASE_USER" \
        -d "$DATABASE_NAME" \
        --verbose \
        --no-password \
        --schema-only \
        --format=plain \
        --create \
        > "$backup_file"
    
    log_success "Schema备份完成"
}

# 执行数据备份
perform_data_backup() {
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="$BACKUP_DATE_DIR/devcollab_data_${timestamp}.sql"
    
    log_info "开始数据备份..."
    log_info "备份文件: $backup_file"
    
    # 执行data-only备份
    pg_dump \
        -h "$DATABASE_HOST" \
        -p "$DATABASE_PORT" \
        -U "$DATABASE_USER" \
        -d "$DATABASE_NAME" \
        --verbose \
        --no-password \
        --data-only \
        --format=plain \
        --disable-triggers \
        > "$backup_file"
    
    log_success "数据备份完成"
}

# 按租户备份
perform_tenant_backup() {
    local tenant_id="$1"
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="$BACKUP_DATE_DIR/devcollab_tenant_${tenant_id}_${timestamp}.sql"
    
    log_info "开始租户备份: $tenant_id"
    log_info "备份文件: $backup_file"
    
    # 创建临时SQL文件
    local temp_sql="/tmp/tenant_backup_${tenant_id}.sql"
    
    cat > "$temp_sql" << EOF
-- 设置当前租户
SELECT set_current_tenant('$tenant_id');

-- 备份租户相关数据
\copy (SELECT * FROM tenants WHERE id = '$tenant_id') TO '$backup_file.tenants.csv' WITH CSV HEADER;
\copy (SELECT * FROM tenant_members WHERE tenant_id = '$tenant_id') TO '$backup_file.tenant_members.csv' WITH CSV HEADER;
\copy (SELECT * FROM roles WHERE tenant_id = '$tenant_id') TO '$backup_file.roles.csv' WITH CSV HEADER;
\copy (SELECT * FROM projects WHERE tenant_id = '$tenant_id') TO '$backup_file.projects.csv' WITH CSV HEADER;
\copy (SELECT t.* FROM tasks t JOIN projects p ON t.project_id = p.id WHERE p.tenant_id = '$tenant_id') TO '$backup_file.tasks.csv' WITH CSV HEADER;
\copy (SELECT r.* FROM repositories r JOIN projects p ON r.project_id = p.id WHERE p.tenant_id = '$tenant_id') TO '$backup_file.repositories.csv' WITH CSV HEADER;
\copy (SELECT * FROM notifications WHERE tenant_id = '$tenant_id') TO '$backup_file.notifications.csv' WITH CSV HEADER;
\copy (SELECT * FROM audit_logs_partitioned WHERE tenant_id = '$tenant_id') TO '$backup_file.audit_logs.csv' WITH CSV HEADER;
EOF
    
    # 执行备份
    psql -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -f "$temp_sql"
    
    # 压缩CSV文件
    tar -czf "$backup_file.tar.gz" -C "$BACKUP_DATE_DIR" $(basename "$backup_file").*.csv
    
    # 清理临时文件
    rm -f "$temp_sql" "$backup_file".*.csv
    
    log_success "租户备份完成: $tenant_id"
}

# 压缩备份文件
compress_backup() {
    local backup_file="$1"
    
    if [ "$COMPRESSION" = "true" ] && [ -f "$backup_file" ]; then
        log_info "压缩备份文件: $backup_file"
        
        gzip -9 "$backup_file"
        log_success "备份文件已压缩: ${backup_file}.gz"
    fi
}

# 加密备份文件
encrypt_backup() {
    local backup_file="$1"
    
    if [ "$ENCRYPTION" = "true" ] && [ -n "$ENCRYPTION_KEY" ] && [ -f "$backup_file" ]; then
        log_info "加密备份文件: $backup_file"
        
        if command -v gpg &> /dev/null; then
            gpg --batch --yes --symmetric --cipher-algo AES256 --passphrase "$ENCRYPTION_KEY" "$backup_file"
            rm -f "$backup_file"
            log_success "备份文件已加密: ${backup_file}.gpg"
        else
            log_warning "gpg 命令未找到，跳过加密"
        fi
    fi
}

# 清理旧备份
cleanup_old_backups() {
    log_info "清理超过 $RETENTION_DAYS 天的旧备份..."
    
    if [ -d "$BACKUP_DIR" ]; then
        find "$BACKUP_DIR" -type f -name "*.sql*" -mtime +$RETENTION_DAYS -delete
        find "$BACKUP_DIR" -type f -name "*.dump*" -mtime +$RETENTION_DAYS -delete
        find "$BACKUP_DIR" -type f -name "*.tar.gz*" -mtime +$RETENTION_DAYS -delete
        find "$BACKUP_DIR" -type d -empty -delete
        
        log_success "旧备份清理完成"
    fi
}

# 验证备份完整性
verify_backup() {
    local backup_file="$1"
    
    if [ -f "$backup_file" ]; then
        log_info "验证备份完整性: $backup_file"
        
        # 检查文件大小
        local file_size=$(stat -c%s "$backup_file")
        if [ "$file_size" -gt 1024 ]; then  # 大于1KB
            log_success "备份文件大小正常: $(numfmt --to=iec $file_size)"
        else
            log_error "备份文件太小，可能备份失败"
            return 1
        fi
        
        # 对于SQL文件，检查是否包含基本结构
        if [[ "$backup_file" == *.sql ]]; then
            if grep -q "CREATE TABLE" "$backup_file"; then
                log_success "备份文件包含表结构"
            else
                log_warning "备份文件可能不包含表结构"
            fi
        fi
    else
        log_error "备份文件不存在: $backup_file"
        return 1
    fi
}

# 发送备份报告
send_backup_report() {
    local status="$1"
    local backup_file="$2"
    
    if [ -n "$BACKUP_NOTIFICATION_EMAIL" ]; then
        local subject="Database Backup Report - $status"
        local body="Backup completed at $(date)\nBackup file: $backup_file\nStatus: $status"
        
        echo -e "$body" | mail -s "$subject" "$BACKUP_NOTIFICATION_EMAIL" 2>/dev/null || true
    fi
    
    # 记录到系统日志
    logger -t "devcollab-backup" "Database backup $status: $backup_file"
}

# 显示使用说明
show_usage() {
    cat << EOF
Cloud-Based Collaborative Development Platform - Database Backup

用法: $0 [选项]

选项:
    --help, -h              显示此帮助信息
    --type TYPE             备份类型: full, schema, data, tenant (默认: full)
    --tenant-id ID          租户ID (仅在type=tenant时需要)
    --backup-dir DIR        备份目录 (默认: /var/backups/devcollab)
    --retention DAYS        备份保留天数 (默认: 30)
    --compress              启用压缩 (默认: true)
    --encrypt               启用加密 (需要设置ENCRYPTION_KEY)
    --parallel-jobs N       并行作业数 (默认: 4)
    --verify                验证备份完整性
    --cleanup-only          仅清理旧备份

环境变量:
    DATABASE_HOST           数据库主机
    DATABASE_PORT           数据库端口
    DATABASE_NAME           数据库名称
    DATABASE_USER           数据库用户
    DATABASE_PASSWORD       数据库密码
    BACKUP_DIR              备份目录
    RETENTION_DAYS          保留天数
    ENCRYPTION_KEY          加密密钥
    BACKUP_NOTIFICATION_EMAIL  通知邮箱

示例:
    # 完整备份
    $0 --type full
    
    # Schema备份
    $0 --type schema
    
    # 租户备份
    $0 --type tenant --tenant-id "123e4567-e89b-12d3-a456-426614174000"
    
    # 带压缩和加密的备份
    ENCRYPTION_KEY="your-secret-key" $0 --type full --compress --encrypt
    
    # 仅清理旧备份
    $0 --cleanup-only

EOF
}

# 主函数
main() {
    local backup_type="full"
    local tenant_id=""
    local verify_backup_flag=false
    local cleanup_only=false
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            --help|-h)
                show_usage
                exit 0
                ;;
            --type)
                backup_type="$2"
                shift 2
                ;;
            --tenant-id)
                tenant_id="$2"
                shift 2
                ;;
            --backup-dir)
                BACKUP_DIR="$2"
                shift 2
                ;;
            --retention)
                RETENTION_DAYS="$2"
                shift 2
                ;;
            --compress)
                COMPRESSION="true"
                shift
                ;;
            --encrypt)
                ENCRYPTION="true"
                shift
                ;;
            --parallel-jobs)
                PARALLEL_JOBS="$2"
                shift 2
                ;;
            --verify)
                verify_backup_flag=true
                shift
                ;;
            --cleanup-only)
                cleanup_only=true
                shift
                ;;
            *)
                log_error "未知选项: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    log_info "开始数据库备份操作"
    log_info "==============================="
    
    check_environment
    create_backup_directory
    
    if [ "$cleanup_only" = true ]; then
        cleanup_old_backups
        exit 0
    fi
    
    check_database_connection
    
    local backup_file=""
    local backup_status="SUCCESS"
    
    # 根据备份类型执行相应操作
    case "$backup_type" in
        "full")
            perform_full_backup
            backup_file="$BACKUP_DATE_DIR/devcollab_full_$(date +%Y%m%d_%H%M%S).sql"
            ;;
        "schema")
            perform_schema_backup
            backup_file="$BACKUP_DATE_DIR/devcollab_schema_$(date +%Y%m%d_%H%M%S).sql"
            ;;
        "data")
            perform_data_backup
            backup_file="$BACKUP_DATE_DIR/devcollab_data_$(date +%Y%m%d_%H%M%S).sql"
            ;;
        "tenant")
            if [ -z "$tenant_id" ]; then
                log_error "租户备份需要指定 --tenant-id"
                exit 1
            fi
            perform_tenant_backup "$tenant_id"
            backup_file="$BACKUP_DATE_DIR/devcollab_tenant_${tenant_id}_$(date +%Y%m%d_%H%M%S).tar.gz"
            ;;
        *)
            log_error "不支持的备份类型: $backup_type"
            exit 1
            ;;
    esac
    
    # 处理备份文件
    if [ -n "$backup_file" ] && [ -f "$backup_file" ]; then
        if [ "$verify_backup_flag" = true ]; then
            if ! verify_backup "$backup_file"; then
                backup_status="FAILED"
            fi
        fi
        
        compress_backup "$backup_file"
        encrypt_backup "$backup_file"
    fi
    
    cleanup_old_backups
    send_backup_report "$backup_status" "$backup_file"
    
    log_success "==============================="
    log_success "备份操作完成！"
    log_success "备份类型: $backup_type"
    log_success "备份目录: $BACKUP_DATE_DIR"
    log_success "状态: $backup_status"
}

# 执行主函数
main "$@"