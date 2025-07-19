-- Cloud-Based Collaborative Development Platform
-- Row Level Security (RLS) Policies
-- Multi-Tenant Data Isolation
-- Generated: 2025-01-19

-- =============================================================================
-- 启用RLS功能
-- =============================================================================

-- 启用所有包含tenant_id的表的行级安全
ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE projects ENABLE ROW LEVEL SECURITY;
ALTER TABLE project_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE task_statuses ENABLE ROW LEVEL SECURITY;
ALTER TABLE tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE repositories ENABLE ROW LEVEL SECURITY;
ALTER TABLE pull_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE pipelines ENABLE ROW LEVEL SECURITY;
ALTER TABLE pipeline_runs ENABLE ROW LEVEL SECURITY;
ALTER TABLE runners ENABLE ROW LEVEL SECURITY;
ALTER TABLE jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE comments ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE secrets ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

-- =============================================================================
-- RLS策略定义
-- =============================================================================

-- 1. 租户表策略
CREATE POLICY tenant_isolation_policy ON tenants
FOR ALL
USING (id = current_setting('app.current_tenant_id')::uuid);

-- 2. 租户成员策略
CREATE POLICY tenant_members_isolation_policy ON tenant_members
FOR ALL
USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- 3. 角色策略
CREATE POLICY roles_isolation_policy ON roles
FOR ALL
USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- 4. 项目策略
CREATE POLICY projects_isolation_policy ON projects
FOR ALL
USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- 5. 项目成员策略 (通过项目关联到租户)
CREATE POLICY project_members_isolation_policy ON project_members
FOR ALL
USING (
    project_id IN (
        SELECT id FROM projects 
        WHERE tenant_id = current_setting('app.current_tenant_id')::uuid
    )
);

-- 6. 任务状态策略
CREATE POLICY task_statuses_isolation_policy ON task_statuses
FOR ALL
USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- 7. 任务策略 (通过项目关联到租户)
CREATE POLICY tasks_isolation_policy ON tasks
FOR ALL
USING (
    project_id IN (
        SELECT id FROM projects 
        WHERE tenant_id = current_setting('app.current_tenant_id')::uuid
    )
);

-- 8. 代码仓库策略 (通过项目关联到租户)
CREATE POLICY repositories_isolation_policy ON repositories
FOR ALL
USING (
    project_id IN (
        SELECT id FROM projects 
        WHERE tenant_id = current_setting('app.current_tenant_id')::uuid
    )
);

-- 9. Pull Request策略 (通过仓库->项目关联到租户)
CREATE POLICY pull_requests_isolation_policy ON pull_requests
FOR ALL
USING (
    repository_id IN (
        SELECT r.id FROM repositories r
        JOIN projects p ON r.project_id = p.id
        WHERE p.tenant_id = current_setting('app.current_tenant_id')::uuid
    )
);

-- 10. 流水线策略 (通过仓库->项目关联到租户)
CREATE POLICY pipelines_isolation_policy ON pipelines
FOR ALL
USING (
    repository_id IN (
        SELECT r.id FROM repositories r
        JOIN projects p ON r.project_id = p.id
        WHERE p.tenant_id = current_setting('app.current_tenant_id')::uuid
    )
);

-- 11. 流水线执行策略 (优化：使用EXISTS提升性能)
CREATE POLICY pipeline_runs_isolation_policy ON pipeline_runs
FOR ALL
USING (
    EXISTS (
        SELECT 1 FROM pipelines pl
        JOIN repositories r ON pl.repository_id = r.id
        JOIN projects p ON r.project_id = p.id
        WHERE pl.id = pipeline_runs.pipeline_id
        AND p.tenant_id = current_setting('app.current_tenant_id')::uuid
    )
);

-- 12. Runner策略
CREATE POLICY runners_isolation_policy ON runners
FOR ALL
USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- 13. 作业策略 (优化：使用EXISTS减少子查询复杂度)
CREATE POLICY jobs_isolation_policy ON jobs
FOR ALL
USING (
    EXISTS (
        SELECT 1 FROM pipeline_runs pr
        JOIN pipelines pl ON pr.pipeline_id = pl.id
        JOIN repositories r ON pl.repository_id = r.id
        JOIN projects p ON r.project_id = p.id
        WHERE pr.id = jobs.pipeline_run_id
        AND p.tenant_id = current_setting('app.current_tenant_id')::uuid
    )
);

-- 14. 文档策略 (通过项目关联到租户)
CREATE POLICY documents_isolation_policy ON documents
FOR ALL
USING (
    project_id IN (
        SELECT id FROM projects 
        WHERE tenant_id = current_setting('app.current_tenant_id')::uuid
    )
);

-- 15. 评论策略
CREATE POLICY comments_isolation_policy ON comments
FOR ALL
USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- 16. 通知策略
CREATE POLICY notifications_isolation_policy ON notifications
FOR ALL
USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- 17. 密钥策略 (基于所有者类型和ID)
CREATE POLICY secrets_isolation_policy ON secrets
FOR ALL
USING (
    CASE owner_type
        WHEN 'tenant' THEN owner_id = current_setting('app.current_tenant_id')::uuid
        WHEN 'project' THEN owner_id IN (
            SELECT id FROM projects 
            WHERE tenant_id = current_setting('app.current_tenant_id')::uuid
        )
        WHEN 'repository' THEN owner_id IN (
            SELECT r.id FROM repositories r
            JOIN projects p ON r.project_id = p.id
            WHERE p.tenant_id = current_setting('app.current_tenant_id')::uuid
        )
        ELSE false
    END
);

-- 18. 审计日志策略
CREATE POLICY audit_logs_isolation_policy ON audit_logs
FOR ALL
USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- =============================================================================
-- 用户表特殊策略 (全局表但需要基于租户成员关系过滤)
-- =============================================================================

-- 启用用户表RLS
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- 用户可以查看同租户内的其他用户
CREATE POLICY users_tenant_visibility_policy ON users
FOR SELECT
USING (
    id IN (
        SELECT tm.user_id FROM tenant_members tm
        WHERE tm.tenant_id = current_setting('app.current_tenant_id')::uuid
    )
    OR id = current_setting('app.current_user_id')::uuid
);

-- 用户只能更新自己的信息
CREATE POLICY users_self_update_policy ON users
FOR UPDATE
USING (id = current_setting('app.current_user_id')::uuid);

-- 平台管理员可以查看所有用户
CREATE POLICY users_admin_all_access_policy ON users
FOR ALL
USING (
    EXISTS (
        SELECT 1 FROM users u
        WHERE u.id = current_setting('app.current_user_id')::uuid
        AND u.is_platform_admin = true
    )
);

-- =============================================================================
-- 订阅计划表策略 (公开只读)
-- =============================================================================

-- 启用订阅计划表RLS
ALTER TABLE subscription_plans ENABLE ROW LEVEL SECURITY;

-- 所有人都可以查看激活的订阅计划
CREATE POLICY subscription_plans_public_read_policy ON subscription_plans
FOR SELECT
USING (is_active = true);

-- 只有平台管理员可以修改订阅计划
CREATE POLICY subscription_plans_admin_manage_policy ON subscription_plans
FOR ALL
USING (
    EXISTS (
        SELECT 1 FROM users u
        WHERE u.id = current_setting('app.current_user_id')::uuid
        AND u.is_platform_admin = true
    )
);

-- =============================================================================
-- 系统设置表策略
-- =============================================================================

-- 启用系统设置表RLS
ALTER TABLE system_settings ENABLE ROW LEVEL SECURITY;

-- 公开设置可以被所有人查看
CREATE POLICY system_settings_public_read_policy ON system_settings
FOR SELECT
USING (is_public = true);

-- 只有平台管理员可以查看和修改所有设置
CREATE POLICY system_settings_admin_manage_policy ON system_settings
FOR ALL
USING (
    EXISTS (
        SELECT 1 FROM users u
        WHERE u.id = current_setting('app.current_user_id')::uuid
        AND u.is_platform_admin = true
    )
);

-- =============================================================================
-- 辅助函数
-- =============================================================================

-- 创建设置当前租户的函数
CREATE OR REPLACE FUNCTION set_current_tenant(tenant_uuid UUID)
RETURNS VOID AS $$
BEGIN
    PERFORM set_config('app.current_tenant_id', tenant_uuid::text, true);
END;
$$ LANGUAGE plpgsql;

-- 创建设置当前用户的函数
CREATE OR REPLACE FUNCTION set_current_user(user_uuid UUID)
RETURNS VOID AS $$
BEGIN
    PERFORM set_config('app.current_user_id', user_uuid::text, true);
END;
$$ LANGUAGE plpgsql;

-- 创建获取当前租户的函数
CREATE OR REPLACE FUNCTION get_current_tenant()
RETURNS UUID AS $$
BEGIN
    RETURN current_setting('app.current_tenant_id', true)::uuid;
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- 创建获取当前用户的函数
CREATE OR REPLACE FUNCTION get_current_user()
RETURNS UUID AS $$
BEGIN
    RETURN current_setting('app.current_user_id', true)::uuid;
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- 数据完整性触发器
-- =============================================================================

-- 验证项目成员角色必须是project作用域的触发器函数
CREATE OR REPLACE FUNCTION validate_project_member_role()
RETURNS TRIGGER AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM roles r
        WHERE r.id = NEW.role_id
        AND r.scope = 'project'
    ) THEN
        RAISE EXCEPTION 'Role for project member must have project scope';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 创建项目成员角色验证触发器
CREATE TRIGGER trigger_validate_project_member_role
    BEFORE INSERT OR UPDATE ON project_members
    FOR EACH ROW
    EXECUTE FUNCTION validate_project_member_role();

-- 验证租户成员角色必须是tenant作用域的触发器函数
CREATE OR REPLACE FUNCTION validate_tenant_member_role()
RETURNS TRIGGER AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM roles r
        WHERE r.id = NEW.role_id
        AND r.scope = 'tenant'
    ) THEN
        RAISE EXCEPTION 'Role for tenant member must have tenant scope';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 创建租户成员角色验证触发器
CREATE TRIGGER trigger_validate_tenant_member_role
    BEFORE INSERT OR UPDATE ON tenant_members
    FOR EACH ROW
    EXECUTE FUNCTION validate_tenant_member_role();

-- 自动更新updated_at字段的触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 为所有有updated_at字段的表创建自动更新触发器
CREATE TRIGGER trigger_update_subscription_plans_updated_at
    BEFORE UPDATE ON subscription_plans
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_update_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_update_roles_updated_at
    BEFORE UPDATE ON roles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_update_projects_updated_at
    BEFORE UPDATE ON projects
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_update_tasks_updated_at
    BEFORE UPDATE ON tasks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_update_repositories_updated_at
    BEFORE UPDATE ON repositories
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_update_pull_requests_updated_at
    BEFORE UPDATE ON pull_requests
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_update_pipelines_updated_at
    BEFORE UPDATE ON pipelines
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_update_documents_updated_at
    BEFORE UPDATE ON documents
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_update_comments_updated_at
    BEFORE UPDATE ON comments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_update_system_settings_updated_at
    BEFORE UPDATE ON system_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- RLS 性能优化索引
-- =============================================================================

-- 为RLS策略优化的复合索引
CREATE INDEX IF NOT EXISTS idx_pipelines_repository_tenant_opt 
ON pipelines(repository_id) 
INCLUDE (id);

CREATE INDEX IF NOT EXISTS idx_repositories_project_tenant_opt 
ON repositories(project_id) 
INCLUDE (id);

CREATE INDEX IF NOT EXISTS idx_projects_tenant_lookup 
ON projects(tenant_id) 
INCLUDE (id);

-- 流水线执行的优化索引
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_pipeline_lookup 
ON pipeline_runs(pipeline_id) 
INCLUDE (id);

-- 作业的优化索引
CREATE INDEX IF NOT EXISTS idx_jobs_pipeline_run_lookup 
ON jobs(pipeline_run_id) 
INCLUDE (id);