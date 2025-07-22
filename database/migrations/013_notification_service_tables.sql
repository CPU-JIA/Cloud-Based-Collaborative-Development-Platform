-- Notification Service Tables
-- 通知服务相关数据表

-- 通知记录表
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID,
    tenant_id UUID NOT NULL,
    project_id UUID,
    
    -- 通知基本信息
    type VARCHAR(100) NOT NULL,
    category VARCHAR(50) NOT NULL,
    priority VARCHAR(20) NOT NULL DEFAULT 'medium',
    title VARCHAR(500) NOT NULL,
    content TEXT,
    
    -- 发送渠道配置
    channels JSONB DEFAULT '{}',
    
    -- 状态管理
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    sent_at TIMESTAMP WITH TIME ZONE,
    failed_at TIMESTAMP WITH TIME ZONE,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    
    -- 元数据
    metadata JSONB,
    template_id UUID,
    event_data JSONB,
    
    -- 追踪信息
    source_event VARCHAR(200),
    correlation_id VARCHAR(200),
    
    -- 审计字段
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by UUID NOT NULL,

    -- 外键约束
    CONSTRAINT fk_notifications_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT fk_notifications_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    CONSTRAINT fk_notifications_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_notifications_creator FOREIGN KEY (created_by) REFERENCES users(id)
);

-- 通知模板表
CREATE TABLE IF NOT EXISTS notification_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    
    -- 模板基本信息
    name VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL,
    category VARCHAR(50) NOT NULL,
    description TEXT,
    
    -- 模板内容
    subject_template VARCHAR(500) NOT NULL,
    body_template TEXT NOT NULL,
    html_template TEXT,
    
    -- 模板配置
    variables JSONB,
    language VARCHAR(10) DEFAULT 'zh-CN',
    format VARCHAR(20) DEFAULT 'text',
    
    -- 默认渠道配置
    default_channels JSONB DEFAULT '{}',
    
    -- 状态管理
    is_active BOOLEAN DEFAULT TRUE,
    version INTEGER DEFAULT 1,
    
    -- 审计字段
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by UUID NOT NULL,
    updated_by UUID,

    -- 外键约束
    CONSTRAINT fk_notification_templates_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT fk_notification_templates_creator FOREIGN KEY (created_by) REFERENCES users(id),
    CONSTRAINT fk_notification_templates_updater FOREIGN KEY (updated_by) REFERENCES users(id)
);

-- 通知规则表
CREATE TABLE IF NOT EXISTS notification_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID,
    project_id UUID,
    
    -- 规则配置
    name VARCHAR(255) NOT NULL,
    description TEXT,
    event_types JSONB NOT NULL,
    conditions JSONB,
    
    -- 通知配置
    template_id UUID NOT NULL,
    channels JSONB DEFAULT '{}',
    priority VARCHAR(20) NOT NULL DEFAULT 'medium',
    
    -- 频率控制
    rate_limit JSONB,
    quiet_hours JSONB,
    
    -- 状态管理
    is_active BOOLEAN DEFAULT TRUE,
    
    -- 审计字段
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by UUID NOT NULL,
    updated_by UUID,

    -- 外键约束
    CONSTRAINT fk_notification_rules_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT fk_notification_rules_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    CONSTRAINT fk_notification_rules_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_notification_rules_template FOREIGN KEY (template_id) REFERENCES notification_templates(id),
    CONSTRAINT fk_notification_rules_creator FOREIGN KEY (created_by) REFERENCES users(id),
    CONSTRAINT fk_notification_rules_updater FOREIGN KEY (updated_by) REFERENCES users(id)
);

-- 发送日志表
CREATE TABLE IF NOT EXISTS delivery_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id UUID NOT NULL,
    
    -- 发送信息
    channel VARCHAR(50) NOT NULL,
    recipient VARCHAR(500) NOT NULL,
    
    -- 状态追踪
    status VARCHAR(20) NOT NULL,
    attempt_count INTEGER DEFAULT 1,
    
    -- 时间记录
    sent_at TIMESTAMP WITH TIME ZONE,
    delivered_at TIMESTAMP WITH TIME ZONE,
    failed_at TIMESTAMP WITH TIME ZONE,
    
    -- 错误信息
    error_message TEXT,
    error_code VARCHAR(100),
    
    -- 外部追踪
    external_id VARCHAR(500),
    
    -- 响应数据
    response JSONB,
    
    -- 审计字段
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- 外键约束
    CONSTRAINT fk_delivery_logs_notification FOREIGN KEY (notification_id) REFERENCES notifications(id) ON DELETE CASCADE
);

-- 用户通知偏好设置表
CREATE TABLE IF NOT EXISTS user_notification_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,
    tenant_id UUID NOT NULL,
    
    -- 全局设置
    globally_enabled BOOLEAN DEFAULT TRUE,
    
    -- 渠道偏好
    email_enabled BOOLEAN DEFAULT TRUE,
    in_app_enabled BOOLEAN DEFAULT TRUE,
    push_enabled BOOLEAN DEFAULT TRUE,
    
    -- 类别设置
    category_settings JSONB,
    
    -- 免打扰设置
    quiet_hours JSONB,
    
    -- 频率设置
    digest_mode BOOLEAN DEFAULT FALSE,
    digest_frequency VARCHAR(20) DEFAULT 'daily',
    
    -- 审计字段
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by UUID,

    -- 外键约束
    CONSTRAINT fk_user_notification_settings_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_notification_settings_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_notification_settings_updater FOREIGN KEY (updated_by) REFERENCES users(id)
);

-- 添加外键约束到notifications表（模板关联）
ALTER TABLE notifications 
ADD CONSTRAINT fk_notifications_template 
FOREIGN KEY (template_id) REFERENCES notification_templates(id);

-- 创建索引以提高查询性能

-- 通知表索引
CREATE INDEX IF NOT EXISTS idx_notifications_user_tenant ON notifications(user_id, tenant_id);
CREATE INDEX IF NOT EXISTS idx_notifications_project ON notifications(project_id);
CREATE INDEX IF NOT EXISTS idx_notifications_type ON notifications(type);
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);
CREATE INDEX IF NOT EXISTS idx_notifications_category ON notifications(category);
CREATE INDEX IF NOT EXISTS idx_notifications_priority ON notifications(priority);
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at);
CREATE INDEX IF NOT EXISTS idx_notifications_correlation_id ON notifications(correlation_id);
CREATE INDEX IF NOT EXISTS idx_notifications_source_event ON notifications(source_event);
CREATE INDEX IF NOT EXISTS idx_notifications_deleted_at ON notifications(deleted_at);

-- 模板表索引
CREATE INDEX IF NOT EXISTS idx_notification_templates_tenant ON notification_templates(tenant_id);
CREATE INDEX IF NOT EXISTS idx_notification_templates_type ON notification_templates(type);
CREATE INDEX IF NOT EXISTS idx_notification_templates_category ON notification_templates(category);
CREATE INDEX IF NOT EXISTS idx_notification_templates_active ON notification_templates(is_active);
CREATE INDEX IF NOT EXISTS idx_notification_templates_deleted_at ON notification_templates(deleted_at);

-- 规则表索引
CREATE INDEX IF NOT EXISTS idx_notification_rules_tenant ON notification_rules(tenant_id);
CREATE INDEX IF NOT EXISTS idx_notification_rules_user ON notification_rules(user_id);
CREATE INDEX IF NOT EXISTS idx_notification_rules_project ON notification_rules(project_id);
CREATE INDEX IF NOT EXISTS idx_notification_rules_active ON notification_rules(is_active);
CREATE INDEX IF NOT EXISTS idx_notification_rules_template ON notification_rules(template_id);
CREATE INDEX IF NOT EXISTS idx_notification_rules_deleted_at ON notification_rules(deleted_at);

-- 发送日志表索引
CREATE INDEX IF NOT EXISTS idx_delivery_logs_notification ON delivery_logs(notification_id);
CREATE INDEX IF NOT EXISTS idx_delivery_logs_channel ON delivery_logs(channel);
CREATE INDEX IF NOT EXISTS idx_delivery_logs_status ON delivery_logs(status);
CREATE INDEX IF NOT EXISTS idx_delivery_logs_recipient ON delivery_logs(recipient);
CREATE INDEX IF NOT EXISTS idx_delivery_logs_external_id ON delivery_logs(external_id);
CREATE INDEX IF NOT EXISTS idx_delivery_logs_created_at ON delivery_logs(created_at);

-- 用户设置表索引
CREATE INDEX IF NOT EXISTS idx_user_notification_settings_tenant ON user_notification_settings(tenant_id);

-- 创建RLS（行级安全）策略

-- 启用RLS
ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE delivery_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_notification_settings ENABLE ROW LEVEL SECURITY;

-- 通知表RLS策略
CREATE POLICY notifications_tenant_isolation ON notifications
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

CREATE POLICY notifications_user_access ON notifications
    FOR SELECT USING (
        tenant_id = current_setting('app.current_tenant_id')::uuid 
        AND (user_id = current_setting('app.current_user_id')::uuid 
             OR current_setting('app.user_role') = 'admin')
    );

-- 模板表RLS策略
CREATE POLICY notification_templates_tenant_isolation ON notification_templates
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- 规则表RLS策略
CREATE POLICY notification_rules_tenant_isolation ON notification_rules
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

CREATE POLICY notification_rules_user_access ON notification_rules
    FOR ALL USING (
        tenant_id = current_setting('app.current_tenant_id')::uuid 
        AND (user_id = current_setting('app.current_user_id')::uuid 
             OR user_id IS NULL
             OR current_setting('app.user_role') = 'admin')
    );

-- 发送日志表RLS策略
CREATE POLICY delivery_logs_notification_access ON delivery_logs
    USING (
        EXISTS (
            SELECT 1 FROM notifications 
            WHERE notifications.id = delivery_logs.notification_id 
            AND notifications.tenant_id = current_setting('app.current_tenant_id')::uuid
        )
    );

-- 用户设置表RLS策略
CREATE POLICY user_notification_settings_tenant_isolation ON user_notification_settings
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

CREATE POLICY user_notification_settings_user_access ON user_notification_settings
    FOR ALL USING (
        tenant_id = current_setting('app.current_tenant_id')::uuid 
        AND (user_id = current_setting('app.current_user_id')::uuid 
             OR current_setting('app.user_role') = 'admin')
    );

-- 创建触发器以自动更新updated_at字段

-- 更新时间戳触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为各表添加更新时间戳触发器
CREATE TRIGGER update_notifications_updated_at 
    BEFORE UPDATE ON notifications 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_notification_templates_updated_at 
    BEFORE UPDATE ON notification_templates 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_notification_rules_updated_at 
    BEFORE UPDATE ON notification_rules 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_delivery_logs_updated_at 
    BEFORE UPDATE ON delivery_logs 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_notification_settings_updated_at 
    BEFORE UPDATE ON user_notification_settings 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 创建一些默认的通知模板
INSERT INTO notification_templates (
    tenant_id, name, type, category, subject_template, body_template, 
    language, format, created_by
) VALUES 
-- 任务分配模板
(
    '00000000-0000-0000-0000-000000000000', -- 需要替换为实际的默认tenant_id
    'Task Assignment Template',
    'project.task.assigned',
    'project',
    '新任务分配：{{.event.task_title}}',
    '您有一个新的任务被分配：

任务标题：{{.event.task_title}}
项目：{{.event.project_name}}
分配者：{{.event.assigner_name}}
优先级：{{.event.priority}}
{{if .event.due_date}}截止日期：{{.event.due_date}}{{end}}

{{if .event.description}}任务描述：
{{.event.description}}{{end}}

请及时处理。',
    'zh-CN',
    'text',
    '00000000-0000-0000-0000-000000000000' -- 需要替换为实际的系统用户ID
),
-- Sprint开始模板
(
    '00000000-0000-0000-0000-000000000000',
    'Sprint Started Template', 
    'project.sprint.started',
    'project',
    'Sprint开始：{{.event.sprint_name}}',
    'Sprint已经开始：

Sprint：{{.event.sprint_name}}
项目：{{.event.project_name}}
开始日期：{{.event.start_date}}
结束日期：{{.event.end_date}}
任务数量：{{.event.task_count}}
故事点：{{.event.story_points}}

让我们开始这个精彩的Sprint吧！',
    'zh-CN',
    'text',
    '00000000-0000-0000-0000-000000000000'
),
-- 系统告警模板
(
    '00000000-0000-0000-0000-000000000000',
    'System Alert Template',
    'system.alert',
    'system',
    '[{{.event.severity}}] 系统告警：{{.event.title}}',
    '系统告警通知：

告警类型：{{.event.alert_type}}
严重程度：{{.event.severity}}
服务：{{.event.service}}
环境：{{.event.environment}}

描述：{{.event.description}}

触发时间：{{.event.triggered_at}}
{{if .event.action_required}}需要立即处理！{{end}}',
    'zh-CN',
    'text',
    '00000000-0000-0000-0000-000000000000'
) ON CONFLICT DO NOTHING;

-- 注释说明
COMMENT ON TABLE notifications IS '通知记录表，存储所有通知信息';
COMMENT ON TABLE notification_templates IS '通知模板表，定义通知内容格式';
COMMENT ON TABLE notification_rules IS '通知规则表，定义何时发送通知';
COMMENT ON TABLE delivery_logs IS '发送日志表，记录通知发送状态';
COMMENT ON TABLE user_notification_settings IS '用户通知偏好设置表';

COMMENT ON COLUMN notifications.channels IS '发送渠道配置，JSON格式存储邮件、webhook、应用内等渠道设置';
COMMENT ON COLUMN notifications.event_data IS '原始事件数据，用于模板渲染';
COMMENT ON COLUMN notifications.correlation_id IS '关联ID，用于追踪相关通知';
COMMENT ON COLUMN notification_templates.variables IS '模板支持的变量列表';
COMMENT ON COLUMN notification_rules.event_types IS '监听的事件类型列表';
COMMENT ON COLUMN notification_rules.conditions IS '触发条件，JSON格式的复杂条件表达式';
COMMENT ON COLUMN notification_rules.rate_limit IS '频率限制配置';
COMMENT ON COLUMN notification_rules.quiet_hours IS '免打扰时间配置';
COMMENT ON COLUMN user_notification_settings.category_settings IS '按类别的个性化设置';