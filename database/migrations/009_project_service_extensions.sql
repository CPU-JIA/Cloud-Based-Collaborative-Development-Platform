-- Project Service Extensions
-- 项目服务扩展迁移
-- 为项目服务添加额外的约束、触发器和优化

-- =============================================================================
-- 项目表优化
-- =============================================================================

-- 创建项目任务序号生成函数
CREATE OR REPLACE FUNCTION get_next_task_number(project_uuid UUID)
RETURNS BIGINT AS $$
DECLARE
    next_num BIGINT;
BEGIN
    -- 获取项目中下一个任务序号
    SELECT COALESCE(MAX(task_number), 0) + 1 
    INTO next_num
    FROM tasks 
    WHERE project_id = project_uuid;
    
    RETURN next_num;
END;
$$ LANGUAGE plpgsql;

-- 创建项目key验证函数
CREATE OR REPLACE FUNCTION validate_project_key(key_value TEXT)
RETURNS BOOLEAN AS $$
BEGIN
    -- 验证项目key格式：2-20个字符，只能包含字母、数字和连字符，必须以字母开头
    RETURN key_value ~ '^[a-zA-Z][a-zA-Z0-9\-]{1,19}$' AND LENGTH(key_value) BETWEEN 2 AND 20;
END;
$$ LANGUAGE plpgsql;

-- 为projects表添加key验证约束
ALTER TABLE projects ADD CONSTRAINT chk_projects_key_format 
CHECK (validate_project_key(key));

-- 创建项目状态更新触发器函数
CREATE OR REPLACE FUNCTION update_project_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 创建项目更新时间触发器
DROP TRIGGER IF EXISTS trigger_update_projects_updated_at ON projects;
CREATE TRIGGER trigger_update_projects_updated_at
    BEFORE UPDATE ON projects
    FOR EACH ROW
    EXECUTE FUNCTION update_project_updated_at();

-- =============================================================================
-- 项目成员表优化
-- =============================================================================

-- 创建项目成员验证函数
CREATE OR REPLACE FUNCTION validate_project_member()
RETURNS TRIGGER AS $$
BEGIN
    -- 检查用户是否属于项目所在的租户
    IF NOT EXISTS (
        SELECT 1 FROM tenant_members tm
        JOIN projects p ON p.tenant_id = tm.tenant_id
        WHERE p.id = NEW.project_id AND tm.user_id = NEW.user_id
    ) THEN
        RAISE EXCEPTION '用户不属于项目所在的租户';
    END IF;
    
    -- 检查角色是否属于项目级别
    IF NOT EXISTS (
        SELECT 1 FROM roles r
        JOIN projects p ON p.tenant_id = r.tenant_id
        WHERE r.id = NEW.role_id AND p.id = NEW.project_id 
        AND r.scope = 'project'
    ) THEN
        RAISE EXCEPTION '角色不是有效的项目级角色';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 创建项目成员验证触发器
DROP TRIGGER IF EXISTS trigger_validate_project_member ON project_members;
CREATE TRIGGER trigger_validate_project_member
    BEFORE INSERT OR UPDATE ON project_members
    FOR EACH ROW
    EXECUTE FUNCTION validate_project_member();

-- =============================================================================
-- 项目权限辅助视图
-- =============================================================================

-- 创建用户项目权限视图
CREATE OR REPLACE VIEW user_project_permissions AS
SELECT 
    pm.user_id,
    pm.project_id,
    p.tenant_id,
    p.key as project_key,
    p.name as project_name,
    p.status as project_status,
    r.name as role_name,
    r.permissions as role_permissions,
    pm.added_at,
    -- 检查是否为项目管理员
    (p.manager_id = pm.user_id) as is_manager
FROM project_members pm
JOIN projects p ON p.id = pm.project_id
JOIN roles r ON r.id = pm.role_id
WHERE p.deleted_at IS NULL;

COMMENT ON VIEW user_project_permissions IS '用户项目权限视图 - 便于权限检查';

-- =============================================================================
-- 项目统计函数
-- =============================================================================

-- 创建项目统计函数
CREATE OR REPLACE FUNCTION get_project_stats(project_uuid UUID)
RETURNS TABLE(
    total_tasks BIGINT,
    completed_tasks BIGINT,
    active_members BIGINT,
    total_repositories BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        (SELECT COUNT(*) FROM tasks WHERE project_id = project_uuid)::BIGINT,
        (SELECT COUNT(*) FROM tasks t 
         JOIN task_statuses ts ON t.status_id = ts.id 
         WHERE t.project_id = project_uuid AND ts.category = 'done')::BIGINT,
        (SELECT COUNT(*) FROM project_members WHERE project_id = project_uuid)::BIGINT,
        (SELECT COUNT(*) FROM repositories WHERE project_id = project_uuid)::BIGINT;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- 项目搜索优化
-- =============================================================================

-- 创建项目全文搜索索引
CREATE INDEX IF NOT EXISTS idx_projects_search 
ON projects USING GIN(to_tsvector('english', name || ' ' || COALESCE(description, '')));

-- 创建项目搜索函数
CREATE OR REPLACE FUNCTION search_projects(
    tenant_uuid UUID,
    search_term TEXT,
    page_limit INTEGER DEFAULT 20,
    page_offset INTEGER DEFAULT 0
)
RETURNS TABLE(
    id UUID,
    key VARCHAR,
    name VARCHAR,
    description TEXT,
    status VARCHAR,
    manager_id UUID,
    created_at TIMESTAMPTZ,
    search_rank REAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        p.id,
        p.key,
        p.name,
        p.description,
        p.status,
        p.manager_id,
        p.created_at,
        ts_rank(to_tsvector('english', p.name || ' ' || COALESCE(p.description, '')), 
                plainto_tsquery('english', search_term)) as search_rank
    FROM projects p
    WHERE p.tenant_id = tenant_uuid 
    AND p.deleted_at IS NULL
    AND to_tsvector('english', p.name || ' ' || COALESCE(p.description, '')) 
        @@ plainto_tsquery('english', search_term)
    ORDER BY search_rank DESC, p.created_at DESC
    LIMIT page_limit OFFSET page_offset;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- 项目清理功能
-- =============================================================================

-- 创建项目软删除清理函数
CREATE OR REPLACE FUNCTION cleanup_deleted_projects(days_old INTEGER DEFAULT 30)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    -- 永久删除超过指定天数的软删除项目
    WITH deleted_projects AS (
        DELETE FROM projects 
        WHERE deleted_at IS NOT NULL 
        AND deleted_at < NOW() - INTERVAL '1 day' * days_old
        RETURNING id
    )
    SELECT COUNT(*) INTO deleted_count FROM deleted_projects;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- 性能优化索引
-- =============================================================================

-- 项目成员查询优化索引
CREATE INDEX IF NOT EXISTS idx_project_members_user_project 
ON project_members(user_id, project_id);

-- 项目状态查询优化索引
CREATE INDEX IF NOT EXISTS idx_projects_tenant_status_created 
ON projects(tenant_id, status, created_at DESC) 
WHERE deleted_at IS NULL;

-- 任务项目关联优化索引
CREATE INDEX IF NOT EXISTS idx_tasks_project_status_created 
ON tasks(project_id, status_id, created_at DESC);

-- =============================================================================
-- 数据完整性约束
-- =============================================================================

-- 确保项目管理员是项目成员
CREATE OR REPLACE FUNCTION ensure_manager_is_member()
RETURNS TRIGGER AS $$
BEGIN
    -- 当设置项目管理员时，确保管理员是项目成员
    IF NEW.manager_id IS NOT NULL THEN
        -- 如果管理员不是项目成员，自动添加为成员
        INSERT INTO project_members (project_id, user_id, role_id, added_by)
        SELECT NEW.id, NEW.manager_id, 
               (SELECT id FROM roles 
                WHERE tenant_id = NEW.tenant_id 
                AND scope = 'project' 
                AND name = 'Project Manager' 
                LIMIT 1),
               NEW.manager_id
        WHERE NOT EXISTS (
            SELECT 1 FROM project_members 
            WHERE project_id = NEW.id AND user_id = NEW.manager_id
        );
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 创建管理员成员检查触发器
DROP TRIGGER IF EXISTS trigger_ensure_manager_is_member ON projects;
CREATE TRIGGER trigger_ensure_manager_is_member
    AFTER INSERT OR UPDATE OF manager_id ON projects
    FOR EACH ROW
    WHEN (NEW.manager_id IS NOT NULL)
    EXECUTE FUNCTION ensure_manager_is_member();

-- =============================================================================
-- 添加注释
-- =============================================================================

COMMENT ON FUNCTION get_next_task_number(UUID) IS '获取项目中下一个任务序号';
COMMENT ON FUNCTION validate_project_key(TEXT) IS '验证项目key格式';
COMMENT ON FUNCTION get_project_stats(UUID) IS '获取项目统计信息';
COMMENT ON FUNCTION search_projects(UUID, TEXT, INTEGER, INTEGER) IS '项目全文搜索';
COMMENT ON FUNCTION cleanup_deleted_projects(INTEGER) IS '清理软删除的项目';
COMMENT ON FUNCTION ensure_manager_is_member() IS '确保项目管理员是项目成员';
COMMENT ON FUNCTION validate_project_member() IS '验证项目成员的有效性';

-- 验证迁移
DO $$
BEGIN
    RAISE NOTICE '项目服务扩展迁移完成';
    RAISE NOTICE '- 添加了项目key格式验证';
    RAISE NOTICE '- 创建了项目成员验证规则';
    RAISE NOTICE '- 添加了项目权限视图';
    RAISE NOTICE '- 优化了搜索和统计功能';
    RAISE NOTICE '- 添加了性能优化索引';
END $$;