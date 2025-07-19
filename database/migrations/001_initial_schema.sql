-- Cloud-Based Collaborative Development Platform
-- PostgreSQL Multi-Tenant Database Schema V3.1
-- Initial Schema Migration
-- Generated: 2025-01-19

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =============================================================================
-- UUID v7 IMPLEMENTATION 自定义UUID v7实现
-- =============================================================================

-- 创建UUID v7函数（时间排序的UUID）
-- UUID v7格式: 48位时间戳 + 12位随机数 + 2位版本(7) + 2位变体 + 62位随机数
CREATE OR REPLACE FUNCTION uuid_generate_v7()
RETURNS UUID AS $$
DECLARE
    -- 获取当前时间戳（毫秒）
    timestamp_ms BIGINT;
    -- 随机数部分
    random_part BIGINT;
    random_part2 BIGINT;
    -- 最终UUID的各个部分
    time_high INTEGER;
    time_mid INTEGER;
    time_low INTEGER;
    clock_seq INTEGER;
    node_high INTEGER;
    node_low INTEGER;
    uuid_string TEXT;
BEGIN
    -- 获取Unix时间戳（毫秒）
    timestamp_ms := EXTRACT(EPOCH FROM NOW()) * 1000;
    
    -- 生成随机数
    random_part := (RANDOM() * 4294967295)::BIGINT;  -- 32位随机数
    random_part2 := (RANDOM() * 4294967295)::BIGINT; -- 32位随机数
    
    -- 构建UUID各部分
    -- 时间戳高32位
    time_high := (timestamp_ms >> 16)::INTEGER;
    -- 时间戳低16位
    time_mid := (timestamp_ms & 65535)::INTEGER;
    -- 版本7 + 时间戳扩展
    time_low := ((7 << 12) | ((random_part >> 20) & 4095))::INTEGER;
    -- 变体(10) + 14位随机数
    clock_seq := ((2 << 14) | ((random_part >> 6) & 16383))::INTEGER;
    -- 剩余48位随机数
    node_high := ((random_part & 63) << 10 | (random_part2 >> 22))::INTEGER;
    node_low := (random_part2 & 4194303)::INTEGER;
    
    -- 格式化为UUID字符串
    uuid_string := FORMAT('%08x-%04x-%04x-%04x-%04x%08x',
        time_high,
        time_mid,
        time_low,
        clock_seq,
        node_high,
        node_low
    );
    
    RETURN uuid_string::UUID;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- ENUM TYPES 枚举类型定义
-- =============================================================================

-- 订阅计划类型
CREATE TYPE subscription_plans_tier_enum AS ENUM (
    'free',
    'standard', 
    'premium',
    'enterprise'
);

-- 用户状态
CREATE TYPE users_status_enum AS ENUM (
    'active',
    'suspended',
    'deactivated'
);

-- 角色作用域
CREATE TYPE roles_scope_enum AS ENUM (
    'tenant',
    'project'
);

-- 任务优先级
CREATE TYPE tasks_priority_enum AS ENUM (
    'low',
    'medium',
    'high',
    'urgent'
);

-- 任务状态类别
CREATE TYPE task_statuses_category_enum AS ENUM (
    'todo',
    'in_progress',
    'done'
);

-- Pull Request状态
CREATE TYPE pull_requests_status_enum AS ENUM (
    'open',
    'draft',
    'merged',
    'closed'
);

-- 流水线执行状态
CREATE TYPE pipeline_run_status_enum AS ENUM (
    'pending',
    'running',
    'success',
    'failed',
    'cancelled'
);

-- 作业状态
CREATE TYPE job_status_enum AS ENUM (
    'pending',
    'running',
    'success',
    'failed',
    'cancelled'
);

-- Runner状态
CREATE TYPE runner_status_enum AS ENUM (
    'online',
    'offline',
    'idle',
    'busy'
);

-- =============================================================================
-- CORE TABLES 核心表定义
-- =============================================================================

-- 1. 订阅计划表
CREATE TABLE subscription_plans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name VARCHAR(50) NOT NULL UNIQUE,
    tier subscription_plans_tier_enum NOT NULL,
    price_monthly DECIMAL(10,2) NOT NULL DEFAULT 0.00,
    features JSONB NOT NULL DEFAULT '{}',
    limits JSONB NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE subscription_plans IS '订阅计划定义';
COMMENT ON COLUMN subscription_plans.features IS '功能开关配置';
COMMENT ON COLUMN subscription_plans.limits IS '使用限制配置';

-- 2. 租户表
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(50) NOT NULL UNIQUE,
    subscription_plan_id UUID NOT NULL REFERENCES subscription_plans(id),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    CONSTRAINT chk_tenants_status CHECK (status IN ('active', 'suspended', 'deactivated'))
);

COMMENT ON TABLE tenants IS '租户表 - 多租户架构的核心';

-- 3. 用户表
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    username VARCHAR(50) NOT NULL,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    display_name VARCHAR(100),
    avatar_url VARCHAR(1024),
    status users_status_enum NOT NULL DEFAULT 'active',
    is_platform_admin BOOLEAN NOT NULL DEFAULT false,
    email_verified BOOLEAN NOT NULL DEFAULT false,
    mfa_enabled BOOLEAN NOT NULL DEFAULT false,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

COMMENT ON TABLE users IS '用户表 - 支持多租户的全局用户';

-- 4. 租户成员关系表
CREATE TABLE tenant_members (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL, -- 稍后通过外键关联到roles表
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    added_by UUID REFERENCES users(id),
    
    PRIMARY KEY (tenant_id, user_id)
);

COMMENT ON TABLE tenant_members IS '租户成员关系 - 定义用户在租户中的角色';

-- 5. 角色表
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    scope roles_scope_enum NOT NULL,
    permissions JSONB NOT NULL DEFAULT '[]',
    is_system BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE (tenant_id, name, scope)
);

COMMENT ON TABLE roles IS '角色定义表 - 支持租户级和项目级角色';
COMMENT ON COLUMN roles.permissions IS '权限数组，如["project.read", "project.write"]';

-- 添加外键约束到tenant_members
ALTER TABLE tenant_members 
ADD CONSTRAINT fk_tenant_members_role_id 
FOREIGN KEY (role_id) REFERENCES roles(id);

-- 6. 项目表
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    key VARCHAR(20) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    manager_id UUID REFERENCES users(id),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    CONSTRAINT chk_projects_status CHECK (status IN ('active', 'archived'))
);

COMMENT ON TABLE projects IS '项目表 - 协作开发的基本单位';

-- 7. 项目成员表
CREATE TABLE project_members (
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id),
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    added_by UUID REFERENCES users(id),
    
    PRIMARY KEY (project_id, user_id)
);

COMMENT ON TABLE project_members IS '项目成员关系 - 定义用户在项目中的角色';

-- =============================================================================
-- PROJECT RELATED TABLES 项目相关表
-- =============================================================================

-- 8. 任务状态表
CREATE TABLE task_statuses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    category task_statuses_category_enum NOT NULL,
    display_order INT NOT NULL,
    
    UNIQUE (tenant_id, name)
);

COMMENT ON TABLE task_statuses IS '自定义任务状态定义';

-- 9. 任务表
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    task_number BIGINT NOT NULL,
    title VARCHAR(512) NOT NULL,
    description TEXT,
    status_id UUID REFERENCES task_statuses(id),
    assignee_id UUID REFERENCES users(id),
    creator_id UUID NOT NULL REFERENCES users(id),
    parent_task_id UUID REFERENCES tasks(id),
    due_date DATE,
    priority tasks_priority_enum NOT NULL DEFAULT 'medium',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE (project_id, task_number)
);

COMMENT ON TABLE tasks IS '任务表 - 项目管理的核心';
COMMENT ON COLUMN tasks.task_number IS '项目内任务序号，通过序列生成';

-- =============================================================================
-- CODE & CI/CD TABLES 代码与CI/CD表
-- =============================================================================

-- 10. 代码仓库表
CREATE TABLE repositories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    visibility VARCHAR(20) NOT NULL DEFAULT 'private',
    default_branch VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE (project_id, name),
    CONSTRAINT chk_repositories_visibility CHECK (visibility IN ('private', 'internal'))
);

COMMENT ON TABLE repositories IS '代码仓库元数据';

-- 11. Pull Request表
CREATE TABLE pull_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    pr_number BIGINT NOT NULL,
    title VARCHAR(512) NOT NULL,
    source_branch VARCHAR(255) NOT NULL,
    target_branch VARCHAR(255) NOT NULL,
    status pull_requests_status_enum NOT NULL DEFAULT 'open',
    creator_id UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    merged_at TIMESTAMPTZ,
    
    UNIQUE (repository_id, pr_number)
);

COMMENT ON TABLE pull_requests IS 'Pull Request管理';

-- 12. CI/CD流水线表
CREATE TABLE pipelines (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    definition_file_path VARCHAR(512) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE pipelines IS 'CI/CD流水线定义';

-- 13. 流水线执行记录表
CREATE TABLE pipeline_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    pipeline_id UUID NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    trigger_type VARCHAR(50) NOT NULL,
    trigger_by UUID REFERENCES users(id),
    commit_sha VARCHAR(40) NOT NULL,
    branch VARCHAR(255),
    status pipeline_run_status_enum NOT NULL DEFAULT 'pending',
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE pipeline_runs IS '流水线执行记录';

-- 14. Runner表
CREATE TABLE runners (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    tags JSONB,
    status runner_status_enum NOT NULL,
    last_contact_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE runners IS 'CI/CD执行器';

-- 15. 作业表
CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    pipeline_run_id UUID NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    status job_status_enum NOT NULL DEFAULT 'pending',
    runner_id UUID REFERENCES runners(id),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE jobs IS '流水线作业记录';

-- =============================================================================
-- KNOWLEDGE & COLLABORATION TABLES 知识与协作表
-- =============================================================================

-- 16. 文档表
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title VARCHAR(512) NOT NULL,
    content TEXT,
    creator_id UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE documents IS '项目文档管理';

-- 17. 评论表（多态关联）
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users(id),
    content TEXT NOT NULL,
    parent_entity_type VARCHAR(50) NOT NULL,
    parent_entity_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_comments_entity_type CHECK (parent_entity_type IN ('task', 'pull_request', 'document'))
);

COMMENT ON TABLE comments IS '通用评论表 - 支持多种实体类型';

-- =============================================================================
-- SYSTEM & AUDITING TABLES 系统与审计表
-- =============================================================================

-- 18. 系统设置表
CREATE TABLE system_settings (
    key VARCHAR(255) PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT,
    is_public BOOLEAN NOT NULL DEFAULT false,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE system_settings IS '系统配置表';

-- 19. 通知表
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    recipient_id UUID NOT NULL REFERENCES users(id),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    message TEXT NOT NULL,
    link VARCHAR(1024),
    is_read BOOLEAN NOT NULL DEFAULT false,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE notifications IS '用户通知表';

-- 20. 密钥表
CREATE TABLE secrets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    owner_type VARCHAR(50) NOT NULL,
    owner_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    value_encrypted BYTEA NOT NULL,
    kek_ref VARCHAR(512) NOT NULL,
    dek_encrypted BYTEA NOT NULL,
    
    UNIQUE (owner_type, owner_id, name),
    CONSTRAINT chk_secrets_owner_type CHECK (owner_type IN ('tenant', 'project', 'repository'))
);

COMMENT ON TABLE secrets IS '加密密钥存储表';

-- 21. 审计日志表
-- 注意：audit_logs表定义已移至003_partitioning.sql中，采用分区表结构以提升性能
-- 这里仅保留注释说明，实际表结构在分区迁移文件中创建

COMMENT ON SCHEMA public IS '审计日志表采用分区结构，详见003_partitioning.sql';

-- =============================================================================
-- INDEXES 索引创建
-- =============================================================================

-- 租户相关索引
CREATE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_tenants_status ON tenants(status) WHERE deleted_at IS NULL;

-- 用户相关索引
CREATE UNIQUE INDEX idx_unique_active_username ON users(username) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_unique_active_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_last_login ON users(last_login_at);

-- 项目相关索引
CREATE UNIQUE INDEX idx_unique_active_project_key ON projects(tenant_id, key) WHERE deleted_at IS NULL;
CREATE INDEX idx_projects_tenant_status ON projects(tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_projects_manager ON projects(manager_id);

-- 任务相关索引
CREATE INDEX idx_tasks_project_status ON tasks(project_id, status_id);
CREATE INDEX idx_tasks_assignee ON tasks(assignee_id);
CREATE INDEX idx_tasks_creator ON tasks(creator_id);
CREATE INDEX idx_tasks_priority ON tasks(priority);
CREATE INDEX idx_tasks_due_date ON tasks(due_date);
CREATE INDEX idx_tasks_parent ON tasks(parent_task_id);

-- 代码仓库相关索引
CREATE INDEX idx_repositories_project ON repositories(project_id);
CREATE INDEX idx_pull_requests_repository ON pull_requests(repository_id);
CREATE INDEX idx_pull_requests_creator ON pull_requests(creator_id);
CREATE INDEX idx_pull_requests_status ON pull_requests(status);

-- CI/CD相关索引
CREATE INDEX idx_pipeline_runs_pipeline ON pipeline_runs(pipeline_id);
CREATE INDEX idx_pipeline_runs_status ON pipeline_runs(status);
CREATE INDEX idx_pipeline_runs_created ON pipeline_runs(created_at);
CREATE INDEX idx_jobs_pipeline_run ON jobs(pipeline_run_id);
CREATE INDEX idx_jobs_runner ON jobs(runner_id);
CREATE INDEX idx_jobs_status ON jobs(status);

-- 评论相关索引
CREATE INDEX idx_comments_parent_entity ON comments(parent_entity_type, parent_entity_id);
CREATE INDEX idx_comments_author ON comments(author_id);
CREATE INDEX idx_comments_tenant ON comments(tenant_id);

-- 通知相关索引
CREATE INDEX idx_notifications_recipient ON notifications(recipient_id);
CREATE INDEX idx_notifications_tenant ON notifications(tenant_id);
CREATE INDEX idx_notifications_unread ON notifications(recipient_id, is_read) WHERE is_read = false;
CREATE INDEX idx_notifications_created ON notifications(created_at);

-- 审计日志相关索引
-- 注意：audit_logs索引已移至003_partitioning.sql中

-- JSONB字段GIN索引
CREATE INDEX idx_subscription_plans_features ON subscription_plans USING GIN(features);
CREATE INDEX idx_subscription_plans_limits ON subscription_plans USING GIN(limits);
CREATE INDEX idx_roles_permissions ON roles USING GIN(permissions);
CREATE INDEX idx_runners_tags ON runners USING GIN(tags);
-- audit_logs的JSONB索引已移至003_partitioning.sql中

-- =============================================================================
-- 插入默认数据
-- =============================================================================

-- 插入默认订阅计划
INSERT INTO subscription_plans (name, tier, price_monthly, features, limits) VALUES
('Free Plan', 'free', 0.00, 
 '{"ai_assistance": false, "advanced_analytics": false, "priority_support": false}',
 '{"max_projects": 3, "max_users_per_tenant": 5, "storage_gb": 1}'),
('Standard Plan', 'standard', 29.99,
 '{"ai_assistance": true, "advanced_analytics": false, "priority_support": false}', 
 '{"max_projects": 20, "max_users_per_tenant": 50, "storage_gb": 50}'),
('Premium Plan', 'premium', 99.99,
 '{"ai_assistance": true, "advanced_analytics": true, "priority_support": true}',
 '{"max_projects": 100, "max_users_per_tenant": 200, "storage_gb": 500}'),
('Enterprise Plan', 'enterprise', 499.99,
 '{"ai_assistance": true, "advanced_analytics": true, "priority_support": true, "custom_integrations": true}',
 '{"max_projects": -1, "max_users_per_tenant": -1, "storage_gb": -1}');

-- 插入系统设置
INSERT INTO system_settings (key, value, description, is_public) VALUES
('platform.name', '"DevCollab Platform"', '平台名称', true),
('platform.version', '"1.0.0"', '平台版本', true),
('feature.ai.enabled', 'true', 'AI功能开关', false),
('defaults.user.role_id', 'null', '默认用户角色ID', false),
('security.password.min_length', '8', '密码最小长度', false),
('security.mfa.required', 'false', '是否强制MFA', false);