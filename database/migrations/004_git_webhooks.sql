-- Git Webhooks Migration
-- 创建Git钩子相关的数据库表

-- 创建webhook_events表
CREATE TABLE IF NOT EXISTS webhook_events (
    id UUID PRIMARY KEY,
    repository_id UUID NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB NOT NULL,
    source VARCHAR(50) NOT NULL DEFAULT 'git',
    signature TEXT,
    processed BOOLEAN NOT NULL DEFAULT FALSE,
    processed_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- 索引
    INDEX idx_webhook_events_repository_id (repository_id),
    INDEX idx_webhook_events_event_type (event_type),
    INDEX idx_webhook_events_source (source),
    INDEX idx_webhook_events_processed (processed),
    INDEX idx_webhook_events_created_at (created_at),
    
    -- 外键约束
    FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
);

-- 创建webhook_triggers表
CREATE TABLE IF NOT EXISTS webhook_triggers (
    id UUID PRIMARY KEY,
    repository_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    event_types JSONB NOT NULL, -- 存储事件类型数组
    conditions JSONB NOT NULL,  -- 存储触发条件
    actions JSONB NOT NULL,     -- 存储触发动作
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- 索引
    INDEX idx_webhook_triggers_repository_id (repository_id),
    INDEX idx_webhook_triggers_enabled (enabled),
    INDEX idx_webhook_triggers_name (name),
    
    -- 外键约束
    FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE,
    
    -- 唯一约束
    UNIQUE(repository_id, name)
);

-- 创建webhook_deliveries表
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id UUID PRIMARY KEY,
    webhook_id UUID NOT NULL,
    event_id UUID NOT NULL,
    url TEXT NOT NULL,
    method VARCHAR(10) NOT NULL DEFAULT 'POST',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    status_code INTEGER,
    request_body TEXT,
    response_body TEXT,
    duration BIGINT, -- 毫秒
    attempts INTEGER NOT NULL DEFAULT 1,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    next_retry_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- 索引
    INDEX idx_webhook_deliveries_webhook_id (webhook_id),
    INDEX idx_webhook_deliveries_event_id (event_id),
    INDEX idx_webhook_deliveries_status (status),
    INDEX idx_webhook_deliveries_created_at (created_at),
    INDEX idx_webhook_deliveries_next_retry (next_retry_at),
    
    -- 外键约束
    FOREIGN KEY (webhook_id) REFERENCES webhook_triggers(id) ON DELETE CASCADE,
    FOREIGN KEY (event_id) REFERENCES webhook_events(id) ON DELETE CASCADE
);

-- 创建pipeline_triggers表（连接流水线和Git事件）
CREATE TABLE IF NOT EXISTS pipeline_triggers (
    id UUID PRIMARY KEY,
    pipeline_id UUID NOT NULL,
    repository_id UUID NOT NULL,
    trigger_type VARCHAR(50) NOT NULL, -- webhook, schedule, manual
    trigger_config JSONB NOT NULL,     -- 触发器配置
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    last_triggered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- 索引
    INDEX idx_pipeline_triggers_pipeline_id (pipeline_id),
    INDEX idx_pipeline_triggers_repository_id (repository_id),
    INDEX idx_pipeline_triggers_type (trigger_type),
    INDEX idx_pipeline_triggers_enabled (enabled),
    
    -- 外键约束
    FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
    -- 注意: pipeline_id 外键约束需要等待pipeline表创建后再添加
);

-- 创建触发器来自动更新updated_at字段
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- webhook_events表的触发器
CREATE TRIGGER update_webhook_events_updated_at 
    BEFORE UPDATE ON webhook_events 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- webhook_triggers表的触发器
CREATE TRIGGER update_webhook_triggers_updated_at 
    BEFORE UPDATE ON webhook_triggers 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- webhook_deliveries表的触发器
CREATE TRIGGER update_webhook_deliveries_updated_at 
    BEFORE UPDATE ON webhook_deliveries 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- pipeline_triggers表的触发器
CREATE TRIGGER update_pipeline_triggers_updated_at 
    BEFORE UPDATE ON pipeline_triggers 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 插入示例数据

-- 示例Webhook触发器配置
INSERT INTO webhook_triggers (
    id, repository_id, name, event_types, conditions, actions, enabled
) VALUES 
(
    gen_random_uuid(),
    (SELECT id FROM repositories LIMIT 1),
    'CI/CD Pipeline Trigger',
    '["push", "pull_request"]'::jsonb,
    '{
        "branches": ["main", "develop"],
        "paths": ["src/**", "tests/**"],
        "file_changes": {
            "include": ["**.go", "**.py"],
            "exclude": ["**.md"]
        }
    }'::jsonb,
    '{
        "start_pipeline": {
            "pipeline_id": "' || gen_random_uuid() || '",
            "variables": {
                "TRIGGER_SOURCE": "webhook",
                "BUILD_TYPE": "ci"
            },
            "environment": "development"
        },
        "send_notification": {
            "type": "slack",
            "recipients": ["#dev-team"],
            "template": "build_started"
        }
    }'::jsonb,
    true
),
(
    gen_random_uuid(),
    (SELECT id FROM repositories LIMIT 1),
    'Release Pipeline Trigger',
    '["tag_push"]'::jsonb,
    '{
        "tags": ["v*.*.*", "release-*"],
        "authors": ["release-bot@company.com"]
    }'::jsonb,
    '{
        "start_pipeline": {
            "pipeline_id": "' || gen_random_uuid() || '",
            "variables": {
                "TRIGGER_SOURCE": "webhook",
                "BUILD_TYPE": "release"
            },
            "environment": "production"
        }
    }'::jsonb,
    true
);

-- 创建视图用于统计
CREATE OR REPLACE VIEW webhook_event_statistics AS
SELECT 
    DATE_TRUNC('day', created_at) as event_date,
    event_type,
    source,
    COUNT(*) as event_count,
    COUNT(CASE WHEN processed = true THEN 1 END) as processed_count,
    COUNT(CASE WHEN processed = false THEN 1 END) as pending_count,
    COUNT(CASE WHEN error_message IS NOT NULL THEN 1 END) as error_count
FROM webhook_events
WHERE created_at >= NOW() - INTERVAL '30 days'
GROUP BY DATE_TRUNC('day', created_at), event_type, source
ORDER BY event_date DESC, event_type;

-- 创建视图用于触发器统计
CREATE OR REPLACE VIEW webhook_trigger_statistics AS
SELECT 
    wt.repository_id,
    wt.name as trigger_name,
    wt.enabled,
    COUNT(we.id) as total_events,
    COUNT(CASE WHEN we.processed = true THEN 1 END) as processed_events,
    MAX(we.created_at) as last_event_time,
    wt.created_at as trigger_created_at
FROM webhook_triggers wt
LEFT JOIN webhook_events we ON wt.repository_id = we.repository_id
GROUP BY wt.id, wt.repository_id, wt.name, wt.enabled, wt.created_at
ORDER BY wt.created_at DESC;

-- 创建视图用于投递统计
CREATE OR REPLACE VIEW webhook_delivery_statistics AS
SELECT 
    DATE_TRUNC('hour', created_at) as delivery_hour,
    status,
    COUNT(*) as delivery_count,
    AVG(duration) as avg_duration,
    AVG(attempts) as avg_attempts
FROM webhook_deliveries
WHERE created_at >= NOW() - INTERVAL '24 hours'
GROUP BY DATE_TRUNC('hour', created_at), status
ORDER BY delivery_hour DESC, status;

-- 添加注释
COMMENT ON TABLE webhook_events IS 'Git钩子事件表，存储所有Git相关的Webhook事件';
COMMENT ON TABLE webhook_triggers IS '钩子触发器表，定义了什么条件下触发什么动作';
COMMENT ON TABLE webhook_deliveries IS 'Webhook投递记录表，记录外部Webhook的调用结果';
COMMENT ON TABLE pipeline_triggers IS '流水线触发器表，连接Git事件和CI/CD流水线';

COMMENT ON COLUMN webhook_events.event_type IS '事件类型：push, pull_request, branch_create等';
COMMENT ON COLUMN webhook_events.event_data IS '事件数据，JSON格式存储具体的事件内容';
COMMENT ON COLUMN webhook_events.source IS '事件来源：git, github, gitlab等';
COMMENT ON COLUMN webhook_events.signature IS '事件签名，用于验证事件的合法性';

COMMENT ON COLUMN webhook_triggers.event_types IS '监听的事件类型数组';
COMMENT ON COLUMN webhook_triggers.conditions IS '触发条件，包括分支、路径、作者等';
COMMENT ON COLUMN webhook_triggers.actions IS '触发动作，如启动流水线、发送通知等';

COMMENT ON COLUMN webhook_deliveries.duration IS '投递耗时，单位毫秒';
COMMENT ON COLUMN webhook_deliveries.attempts IS '投递尝试次数';
COMMENT ON COLUMN webhook_deliveries.next_retry_at IS '下次重试时间';

-- 创建用于清理过期数据的存储过程
CREATE OR REPLACE FUNCTION cleanup_old_webhook_data()
RETURNS INTEGER AS $$
DECLARE
    deleted_events INTEGER := 0;
    deleted_deliveries INTEGER := 0;
BEGIN
    -- 清理30天前的已处理事件（保留失败的事件用于调试）
    DELETE FROM webhook_events 
    WHERE created_at < NOW() - INTERVAL '30 days' 
      AND processed = true 
      AND error_message IS NULL;
    GET DIAGNOSTICS deleted_events = ROW_COUNT;
    
    -- 清理7天前的投递记录（保留失败的投递记录）
    DELETE FROM webhook_deliveries 
    WHERE created_at < NOW() - INTERVAL '7 days' 
      AND status = 'success';
    GET DIAGNOSTICS deleted_deliveries = ROW_COUNT;
    
    -- 记录清理结果
    INSERT INTO system_logs (level, message, details, created_at) VALUES (
        'INFO',
        'Webhook data cleanup completed',
        jsonb_build_object(
            'deleted_events', deleted_events,
            'deleted_deliveries', deleted_deliveries
        ),
        NOW()
    );
    
    RETURN deleted_events + deleted_deliveries;
END;
$$ LANGUAGE plpgsql;

-- 创建system_logs表（如果不存在）
CREATE TABLE IF NOT EXISTS system_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    level VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    details JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    INDEX idx_system_logs_level (level),
    INDEX idx_system_logs_created_at (created_at)
);

-- 创建索引优化查询性能
CREATE INDEX IF NOT EXISTS idx_webhook_events_repository_processed ON webhook_events(repository_id, processed);
CREATE INDEX IF NOT EXISTS idx_webhook_events_type_source ON webhook_events(event_type, source);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_status_retry ON webhook_deliveries(status, next_retry_at) WHERE next_retry_at IS NOT NULL;