-- Cloud-Based Collaborative Development Platform
-- Partitioning Strategy for High-Volume Tables
-- Performance Optimization for audit_logs and notifications
-- Generated: 2025-01-19

-- =============================================================================
-- 审计日志表分区策略
-- =============================================================================

-- 首先删除现有的audit_logs表（如果有数据需要先备份）
-- DROP TABLE IF EXISTS audit_logs CASCADE;

-- 重新创建audit_logs作为分区表 (按租户ID和时间的复合分区)
CREATE TABLE audit_logs_partitioned (
    id BIGSERIAL,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    user_id UUID REFERENCES users(id),
    impersonator_id UUID REFERENCES users(id),
    action VARCHAR(100) NOT NULL,
    target_entity_type VARCHAR(50),
    target_entity_id UUID,
    details JSONB,
    client_ip INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
) PARTITION BY LIST (tenant_id);

-- 创建默认分区（用于新租户）
CREATE TABLE audit_logs_default PARTITION OF audit_logs_partitioned
DEFAULT;

-- 为默认分区创建按时间的子分区
ALTER TABLE audit_logs_default PARTITION BY RANGE (created_at);

-- 创建默认分区的时间子分区（当前月份及未来几个月）
CREATE TABLE audit_logs_default_202501 PARTITION OF audit_logs_default
FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

CREATE TABLE audit_logs_default_202502 PARTITION OF audit_logs_default
FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');

CREATE TABLE audit_logs_default_202503 PARTITION OF audit_logs_default
FOR VALUES FROM ('2025-03-01') TO ('2025-04-01');

CREATE TABLE audit_logs_default_202504 PARTITION OF audit_logs_default
FOR VALUES FROM ('2025-04-01') TO ('2025-05-01');

-- 为分区表添加注释
COMMENT ON TABLE audit_logs_partitioned IS '分区版本的审计日志表 - 按租户和时间分区';

-- 启用分区表的行级安全
ALTER TABLE audit_logs_partitioned ENABLE ROW LEVEL SECURITY;

-- 创建审计日志的RLS策略
CREATE POLICY audit_logs_partitioned_isolation_policy ON audit_logs_partitioned
FOR ALL
USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- 创建索引（在分区表上）
CREATE INDEX idx_audit_logs_partitioned_tenant_created ON audit_logs_partitioned(tenant_id, created_at);
CREATE INDEX idx_audit_logs_partitioned_user ON audit_logs_partitioned(user_id);
CREATE INDEX idx_audit_logs_partitioned_action ON audit_logs_partitioned(action);
CREATE INDEX idx_audit_logs_partitioned_target ON audit_logs_partitioned(target_entity_type, target_entity_id);
CREATE INDEX idx_audit_logs_partitioned_details ON audit_logs_partitioned USING GIN(details);

-- =============================================================================
-- 通知表分区策略
-- =============================================================================

-- 重新创建notifications作为分区表 (按时间分区)
CREATE TABLE notifications_partitioned (
    id UUID DEFAULT uuid_generate_v7(),
    recipient_id UUID NOT NULL REFERENCES users(id),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    message TEXT NOT NULL,
    link VARCHAR(1024),
    is_read BOOLEAN NOT NULL DEFAULT false,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (created_at);

-- 创建通知表的时间分区（当前月份及未来几个月）
CREATE TABLE notifications_202501 PARTITION OF notifications_partitioned
FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

CREATE TABLE notifications_202502 PARTITION OF notifications_partitioned
FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');

CREATE TABLE notifications_202503 PARTITION OF notifications_partitioned
FOR VALUES FROM ('2025-03-01') TO ('2025-04-01');

CREATE TABLE notifications_202504 PARTITION OF notifications_partitioned
FOR VALUES FROM ('2025-04-01') TO ('2025-05-01');

-- 为分区表添加注释
COMMENT ON TABLE notifications_partitioned IS '分区版本的通知表 - 按时间分区';

-- 启用分区表的行级安全
ALTER TABLE notifications_partitioned ENABLE ROW LEVEL SECURITY;

-- 创建通知的RLS策略
CREATE POLICY notifications_partitioned_isolation_policy ON notifications_partitioned
FOR ALL
USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- 创建索引（在分区表上）
CREATE INDEX idx_notifications_partitioned_recipient ON notifications_partitioned(recipient_id);
CREATE INDEX idx_notifications_partitioned_tenant ON notifications_partitioned(tenant_id);
CREATE INDEX idx_notifications_partitioned_unread ON notifications_partitioned(recipient_id, is_read) WHERE is_read = false;
CREATE INDEX idx_notifications_partitioned_created ON notifications_partitioned(created_at);

-- =============================================================================
-- 自动分区管理函数
-- =============================================================================

-- 创建自动创建下个月分区的函数
CREATE OR REPLACE FUNCTION create_next_month_partitions()
RETURNS VOID AS $$
DECLARE
    next_month_start DATE;
    next_month_end DATE;
    partition_name_audit TEXT;
    partition_name_notifications TEXT;
    sql_audit TEXT;
    sql_notifications TEXT;
BEGIN
    -- 计算下个月的开始和结束日期
    next_month_start := date_trunc('month', CURRENT_DATE + INTERVAL '1 month');
    next_month_end := next_month_start + INTERVAL '1 month';
    
    -- 生成分区名称（格式：YYYYMM）
    partition_name_audit := 'audit_logs_default_' || to_char(next_month_start, 'YYYYMM');
    partition_name_notifications := 'notifications_' || to_char(next_month_start, 'YYYYMM');
    
    -- 检查分区是否已存在，如果不存在则创建
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_name = partition_name_audit
    ) THEN
        sql_audit := format(
            'CREATE TABLE %I PARTITION OF audit_logs_default FOR VALUES FROM (%L) TO (%L)',
            partition_name_audit,
            next_month_start,
            next_month_end
        );
        EXECUTE sql_audit;
        
        RAISE NOTICE 'Created audit_logs partition: %', partition_name_audit;
    END IF;
    
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_name = partition_name_notifications
    ) THEN
        sql_notifications := format(
            'CREATE TABLE %I PARTITION OF notifications_partitioned FOR VALUES FROM (%L) TO (%L)',
            partition_name_notifications,
            next_month_start,
            next_month_end
        );
        EXECUTE sql_notifications;
        
        RAISE NOTICE 'Created notifications partition: %', partition_name_notifications;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- 创建清理旧分区的函数
CREATE OR REPLACE FUNCTION cleanup_old_partitions(months_to_keep INTEGER DEFAULT 12)
RETURNS VOID AS $$
DECLARE
    cutoff_date DATE;
    partition_record RECORD;
    sql_command TEXT;
BEGIN
    -- 计算保留期限的截止日期
    cutoff_date := date_trunc('month', CURRENT_DATE - (months_to_keep * INTERVAL '1 month'));
    
    -- 查找需要清理的审计日志分区
    FOR partition_record IN
        SELECT schemaname, tablename
        FROM pg_tables
        WHERE tablename LIKE 'audit_logs_default_%'
        AND tablename ~ '^audit_logs_default_\d{6}$'
        AND to_date(substring(tablename from 'audit_logs_default_(\d{6})'), 'YYYYMM') < cutoff_date
    LOOP
        sql_command := format('DROP TABLE %I.%I', partition_record.schemaname, partition_record.tablename);
        EXECUTE sql_command;
        RAISE NOTICE 'Dropped old audit_logs partition: %', partition_record.tablename;
    END LOOP;
    
    -- 查找需要清理的通知分区
    FOR partition_record IN
        SELECT schemaname, tablename
        FROM pg_tables
        WHERE tablename LIKE 'notifications_%'
        AND tablename ~ '^notifications_\d{6}$'
        AND to_date(substring(tablename from 'notifications_(\d{6})'), 'YYYYMM') < cutoff_date
    LOOP
        sql_command := format('DROP TABLE %I.%I', partition_record.schemaname, partition_record.tablename);
        EXECUTE sql_command;
        RAISE NOTICE 'Dropped old notifications partition: %', partition_record.tablename;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- 创建租户专用审计日志分区的函数
CREATE OR REPLACE FUNCTION create_tenant_audit_partition(tenant_uuid UUID)
RETURNS VOID AS $$
DECLARE
    partition_name TEXT;
    sql_command TEXT;
    current_month_start DATE;
    current_month_end DATE;
BEGIN
    -- 生成分区名称
    partition_name := 'audit_logs_tenant_' || replace(tenant_uuid::text, '-', '_');
    
    -- 检查分区是否已存在
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_name = partition_name
    ) THEN
        -- 创建租户专用分区
        sql_command := format(
            'CREATE TABLE %I PARTITION OF audit_logs_partitioned FOR VALUES IN (%L)',
            partition_name,
            tenant_uuid
        );
        EXECUTE sql_command;
        
        -- 将租户分区设置为按时间的子分区
        sql_command := format('ALTER TABLE %I PARTITION BY RANGE (created_at)', partition_name);
        EXECUTE sql_command;
        
        -- 为当前月份创建子分区
        current_month_start := date_trunc('month', CURRENT_DATE);
        current_month_end := current_month_start + INTERVAL '1 month';
        
        sql_command := format(
            'CREATE TABLE %I PARTITION OF %I FOR VALUES FROM (%L) TO (%L)',
            partition_name || '_' || to_char(current_month_start, 'YYYYMM'),
            partition_name,
            current_month_start,
            current_month_end
        );
        EXECUTE sql_command;
        
        RAISE NOTICE 'Created tenant audit_logs partition: % with current month subpartition', partition_name;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- 分区维护计划任务
-- =============================================================================

-- 注意：以下命令需要在安装了pg_cron扩展的情况下执行
-- 如果没有pg_cron，需要通过外部调度程序（如crontab）来执行这些函数

-- 创建每月自动创建下个月分区的计划任务（每月1日凌晨2点执行）
-- SELECT cron.schedule('create-monthly-partitions', '0 2 1 * *', 'SELECT create_next_month_partitions();');

-- 创建每月清理旧分区的计划任务（每月1日凌晨3点执行，保留12个月数据）
-- SELECT cron.schedule('cleanup-old-partitions', '0 3 1 * *', 'SELECT cleanup_old_partitions(12);');

-- =============================================================================
-- 数据迁移辅助函数
-- =============================================================================

-- 从原始表迁移数据到分区表的函数
CREATE OR REPLACE FUNCTION migrate_to_partitioned_tables()
RETURNS VOID AS $$
BEGIN
    -- 如果存在原始的audit_logs表，将数据迁移到分区表
    IF EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_name = 'audit_logs' 
        AND table_name != 'audit_logs_partitioned'
    ) THEN
        INSERT INTO audit_logs_partitioned 
        SELECT * FROM audit_logs;
        
        RAISE NOTICE 'Migrated data from audit_logs to audit_logs_partitioned';
    END IF;
    
    -- 如果存在原始的notifications表，将数据迁移到分区表
    IF EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_name = 'notifications' 
        AND table_name != 'notifications_partitioned'
    ) THEN
        INSERT INTO notifications_partitioned 
        SELECT * FROM notifications;
        
        RAISE NOTICE 'Migrated data from notifications to notifications_partitioned';
    END IF;
END;
$$ LANGUAGE plpgsql;

-- 执行数据迁移（如果需要）
-- SELECT migrate_to_partitioned_tables();

-- =============================================================================
-- 性能监控视图
-- =============================================================================

-- 创建分区信息监控视图
CREATE OR REPLACE VIEW partition_info AS
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size,
    (SELECT count(*) FROM pg_stat_user_tables WHERE relname = tablename) as row_count_estimate
FROM pg_tables 
WHERE tablename LIKE 'audit_logs_%' 
   OR tablename LIKE 'notifications_%'
ORDER BY tablename;

COMMENT ON VIEW partition_info IS '分区表信息监控视图';

-- 创建分区性能统计视图
CREATE OR REPLACE VIEW partition_performance AS
SELECT 
    schemaname,
    tablename,
    seq_scan,
    seq_tup_read,
    idx_scan,
    idx_tup_fetch,
    n_tup_ins,
    n_tup_upd,
    n_tup_del,
    n_live_tup,
    n_dead_tup,
    last_vacuum,
    last_autovacuum,
    last_analyze,
    last_autoanalyze
FROM pg_stat_user_tables 
WHERE relname LIKE 'audit_logs_%' 
   OR relname LIKE 'notifications_%'
ORDER BY relname;

COMMENT ON VIEW partition_performance IS '分区表性能统计视图';