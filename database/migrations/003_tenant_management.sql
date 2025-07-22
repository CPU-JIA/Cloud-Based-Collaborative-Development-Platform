-- 003_tenant_management.sql
-- 租户管理系统数据表创建

-- 创建租户主表
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255) NOT NULL UNIQUE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    plan VARCHAR(50) NOT NULL DEFAULT 'free',
    billing_email VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- 联系信息
    contact_name VARCHAR(255),
    contact_email VARCHAR(255),
    contact_phone VARCHAR(50),
    
    -- 地址信息  
    address TEXT,
    city VARCHAR(100),
    state VARCHAR(100),
    country VARCHAR(100),
    postal_code VARCHAR(20),
    
    -- 系统字段
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- 创建租户配置表
CREATE TABLE IF NOT EXISTS tenant_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- 资源限制
    max_users INTEGER DEFAULT 10,
    max_projects INTEGER DEFAULT 5,
    max_storage INTEGER DEFAULT 1024,  -- MB
    max_api_calls_daily INTEGER DEFAULT 10000,
    
    -- JSON配置字段
    feature_flags JSONB DEFAULT '{}',
    security_policy JSONB DEFAULT '{}',
    integration_settings JSONB DEFAULT '{}',
    
    -- 系统字段
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    UNIQUE(tenant_id)
);

-- 创建租户订阅表
CREATE TABLE IF NOT EXISTS tenant_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- 订阅信息
    plan_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'trialing',
    billing_cycle VARCHAR(20) NOT NULL DEFAULT 'monthly',
    
    -- 时间信息
    trial_ends_at TIMESTAMP WITH TIME ZONE,
    current_period_start TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    current_period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE,
    
    -- 计费信息
    amount DECIMAL(10,2) DEFAULT 0,
    currency VARCHAR(3) DEFAULT 'USD',
    payment_method_id VARCHAR(255),
    
    -- 外部系统ID
    stripe_customer_id VARCHAR(255),
    stripe_subscription_id VARCHAR(255) UNIQUE,
    
    -- 使用量统计
    usage_metrics JSONB DEFAULT '{}',
    
    -- 系统字段
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    UNIQUE(tenant_id)
);

-- 创建租户品牌定制表
CREATE TABLE IF NOT EXISTS tenant_brandings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Logo和图标
    logo_url VARCHAR(500),
    favicon_url VARCHAR(500),
    
    -- 主题配色 (HEX颜色值)
    primary_color VARCHAR(7) DEFAULT '#1890ff',
    secondary_color VARCHAR(7) DEFAULT '#52c41a',
    accent_color VARCHAR(7) DEFAULT '#fa8c16',
    
    -- 自定义域名
    custom_domain VARCHAR(255) UNIQUE,
    custom_domain_ssl BOOLEAN DEFAULT FALSE,
    
    -- 自定义CSS
    custom_css TEXT,
    
    -- 邮件品牌
    email_from_name VARCHAR(255),
    email_reply_to VARCHAR(255),
    
    -- 其他品牌设置
    branding_settings JSONB DEFAULT '{}',
    
    -- 系统字段
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    UNIQUE(tenant_id)
);

-- 创建租户审计日志表
CREATE TABLE IF NOT EXISTS tenant_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- 操作信息
    action VARCHAR(100) NOT NULL,    -- create, update, delete, activate, suspend, etc.
    entity VARCHAR(100) NOT NULL,    -- tenant, config, subscription, branding
    entity_id VARCHAR(255),
    
    -- 用户信息
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    user_email VARCHAR(255),
    
    -- 变更内容
    old_values JSONB,
    new_values JSONB,
    
    -- 请求信息  
    ip_address VARCHAR(45),
    user_agent TEXT,
    
    -- 系统字段
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引以提升查询性能

-- 租户表索引
CREATE INDEX IF NOT EXISTS idx_tenants_domain ON tenants(domain);
CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status);
CREATE INDEX IF NOT EXISTS idx_tenants_plan ON tenants(plan);
CREATE INDEX IF NOT EXISTS idx_tenants_created_at ON tenants(created_at);
CREATE INDEX IF NOT EXISTS idx_tenants_deleted_at ON tenants(deleted_at);

-- 租户配置表索引
CREATE INDEX IF NOT EXISTS idx_tenant_configs_tenant_id ON tenant_configs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_configs_deleted_at ON tenant_configs(deleted_at);

-- 租户订阅表索引
CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_tenant_id ON tenant_subscriptions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_status ON tenant_subscriptions(status);
CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_expires_at ON tenant_subscriptions(expires_at);
CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_current_period_end ON tenant_subscriptions(current_period_end);
CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_stripe_customer ON tenant_subscriptions(stripe_customer_id);

-- 租户品牌表索引
CREATE INDEX IF NOT EXISTS idx_tenant_brandings_tenant_id ON tenant_brandings(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_brandings_custom_domain ON tenant_brandings(custom_domain);

-- 审计日志表索引
CREATE INDEX IF NOT EXISTS idx_tenant_audit_logs_tenant_id ON tenant_audit_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_audit_logs_action ON tenant_audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_tenant_audit_logs_user_id ON tenant_audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_tenant_audit_logs_created_at ON tenant_audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_tenant_audit_logs_ip_address ON tenant_audit_logs(ip_address);

-- 创建更新时间触发器
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为需要的表添加更新时间触发器
CREATE TRIGGER update_tenants_updated_at 
    BEFORE UPDATE ON tenants 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_tenant_configs_updated_at 
    BEFORE UPDATE ON tenant_configs 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_tenant_subscriptions_updated_at 
    BEFORE UPDATE ON tenant_subscriptions 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_tenant_brandings_updated_at 
    BEFORE UPDATE ON tenant_brandings 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 检查约束（业务规则）
-- 确保状态值有效
ALTER TABLE tenants ADD CONSTRAINT chk_tenant_status 
    CHECK (status IN ('active', 'suspended', 'pending', 'deleted'));

-- 确保计划类型有效  
ALTER TABLE tenants ADD CONSTRAINT chk_tenant_plan 
    CHECK (plan IN ('free', 'basic', 'professional', 'enterprise'));

-- 确保订阅状态有效
ALTER TABLE tenant_subscriptions ADD CONSTRAINT chk_subscription_status 
    CHECK (status IN ('active', 'canceled', 'expired', 'trialing'));

-- 确保计费周期有效
ALTER TABLE tenant_subscriptions ADD CONSTRAINT chk_billing_cycle 
    CHECK (billing_cycle IN ('monthly', 'yearly'));

-- 确保金额非负
ALTER TABLE tenant_subscriptions ADD CONSTRAINT chk_amount_positive 
    CHECK (amount >= 0);

-- 确保资源限制为正数
ALTER TABLE tenant_configs ADD CONSTRAINT chk_max_users_positive 
    CHECK (max_users > 0);
ALTER TABLE tenant_configs ADD CONSTRAINT chk_max_projects_positive 
    CHECK (max_projects > 0);
ALTER TABLE tenant_configs ADD CONSTRAINT chk_max_storage_positive 
    CHECK (max_storage > 0);
ALTER TABLE tenant_configs ADD CONSTRAINT chk_max_api_calls_positive 
    CHECK (max_api_calls_daily > 0);

-- 创建RLS (行级安全策略)
-- 为多租户架构提供数据隔离
-- (注意：这里我们先创建策略结构，具体的RLS策略根据应用需求配置)

-- 添加注释
COMMENT ON TABLE tenants IS '租户主表：存储租户基本信息';
COMMENT ON TABLE tenant_configs IS '租户配置表：存储租户的资源限制和功能配置';
COMMENT ON TABLE tenant_subscriptions IS '租户订阅表：存储租户的订阅和计费信息';
COMMENT ON TABLE tenant_brandings IS '租户品牌表：存储租户的品牌定制信息';
COMMENT ON TABLE tenant_audit_logs IS '租户审计日志表：记录租户相关的所有操作';

-- 插入默认数据（可选）
-- 插入系统默认租户（系统管理租户）
INSERT INTO tenants (id, name, domain, status, plan, billing_email, description) 
VALUES (
    '00000000-0000-0000-0000-000000000000'::UUID,
    'System Admin',
    'system',
    'active',
    'enterprise',
    'admin@system.local',
    'System administration tenant'
) ON CONFLICT (id) DO NOTHING;

-- 为系统租户创建默认配置
INSERT INTO tenant_configs (tenant_id, max_users, max_projects, max_storage, max_api_calls_daily, feature_flags)
VALUES (
    '00000000-0000-0000-0000-000000000000'::UUID,
    999999,  -- 无限制用户
    999999,  -- 无限制项目
    999999,  -- 无限制存储
    999999,  -- 无限制API调用
    '{"all_features": true}'::JSONB
) ON CONFLICT (tenant_id) DO NOTHING;