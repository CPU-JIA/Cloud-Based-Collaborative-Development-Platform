-- Cloud-Based Collaborative Development Platform
-- IAM Service Database Schema Migration
-- 支持JWT认证、RBAC权限模型和多租户隔离
-- Generated: 2025-01-20

-- =============================================================================
-- IAM CORE TABLES - IAM服务核心表
-- =============================================================================

-- 1. 用户表 (增强版，支持IAM服务特性)
-- 注意：如果users表已存在，此语句会根据需要添加缺失字段
DO $$
BEGIN
    -- 检查users表是否存在
    IF NOT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'users') THEN
        -- 创建全新的users表
        CREATE TABLE users (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
            tenant_id UUID NOT NULL,
            email VARCHAR(255) NOT NULL,
            username VARCHAR(50) NOT NULL,
            password_hash VARCHAR(255) NOT NULL,
            first_name VARCHAR(100) NOT NULL,
            last_name VARCHAR(100) NOT NULL,
            avatar VARCHAR(512),
            phone VARCHAR(20),
            is_active BOOLEAN NOT NULL DEFAULT true,
            two_factor_enabled BOOLEAN NOT NULL DEFAULT false,
            failed_login_count INTEGER NOT NULL DEFAULT 0,
            locked_until TIMESTAMPTZ,
            last_login_at TIMESTAMPTZ,
            password_reset_at TIMESTAMPTZ,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            deleted_at TIMESTAMPTZ
        );
    ELSE
        -- 为现有users表添加IAM服务需要的字段
        -- 添加tenant_id字段（如果不存在）
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'tenant_id') THEN
            ALTER TABLE users ADD COLUMN tenant_id UUID;
        END IF;
        
        -- 添加first_name字段（如果不存在）
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'first_name') THEN
            ALTER TABLE users ADD COLUMN first_name VARCHAR(100);
        END IF;
        
        -- 添加last_name字段（如果不存在）
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'last_name') THEN
            ALTER TABLE users ADD COLUMN last_name VARCHAR(100);
        END IF;
        
        -- 添加avatar字段（如果不存在）
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'avatar') THEN
            ALTER TABLE users ADD COLUMN avatar VARCHAR(512);
        END IF;
        
        -- 添加phone字段（如果不存在）
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'phone') THEN
            ALTER TABLE users ADD COLUMN phone VARCHAR(20);
        END IF;
        
        -- 添加is_active字段（如果不存在）
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'is_active') THEN
            ALTER TABLE users ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT true;
        END IF;
        
        -- 添加two_factor_enabled字段（如果不存在）
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'two_factor_enabled') THEN
            ALTER TABLE users ADD COLUMN two_factor_enabled BOOLEAN NOT NULL DEFAULT false;
        END IF;
        
        -- 添加failed_login_count字段（如果不存在）
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'failed_login_count') THEN
            ALTER TABLE users ADD COLUMN failed_login_count INTEGER NOT NULL DEFAULT 0;
        END IF;
        
        -- 添加locked_until字段（如果不存在）
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'locked_until') THEN
            ALTER TABLE users ADD COLUMN locked_until TIMESTAMPTZ;
        END IF;
        
        -- 添加password_reset_at字段（如果不存在）
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'password_reset_at') THEN
            ALTER TABLE users ADD COLUMN password_reset_at TIMESTAMPTZ;
        END IF;
    END IF;
END $$;

COMMENT ON TABLE users IS 'IAM用户表 - 支持多租户、安全认证和RBAC';
COMMENT ON COLUMN users.tenant_id IS '租户ID - 多租户隔离';
COMMENT ON COLUMN users.password_hash IS 'bcrypt加密后的密码哈希';
COMMENT ON COLUMN users.failed_login_count IS '连续失败登录次数';
COMMENT ON COLUMN users.locked_until IS '账户锁定截止时间';
COMMENT ON COLUMN users.two_factor_enabled IS '是否启用双因子认证';

-- 2. 角色表 (IAM RBAC模型)
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'roles') THEN
        CREATE TABLE roles (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
            tenant_id UUID NOT NULL,
            name VARCHAR(50) NOT NULL,
            description TEXT,
            is_system BOOLEAN NOT NULL DEFAULT false,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            
            UNIQUE (tenant_id, name)
        );
    ELSE
        -- 确保现有roles表有必要的字段
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'roles' AND column_name = 'description') THEN
            ALTER TABLE roles ADD COLUMN description TEXT;
        END IF;
        
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'roles' AND column_name = 'is_system') THEN
            ALTER TABLE roles ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT false;
        END IF;
    END IF;
END $$;

COMMENT ON TABLE roles IS 'IAM角色表 - RBAC权限模型';
COMMENT ON COLUMN roles.is_system IS '是否为系统预定义角色';

-- 3. 权限表 (IAM RBAC模型)
CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    tenant_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    resource VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE (tenant_id, name),
    UNIQUE (tenant_id, resource, action)
);

COMMENT ON TABLE permissions IS 'IAM权限表 - 定义系统中的权限';
COMMENT ON COLUMN permissions.resource IS '权限作用的资源类型（如: project, user, repository）';
COMMENT ON COLUMN permissions.action IS '权限动作（如: read, write, delete, manage）';

-- 4. 用户角色关联表
CREATE TABLE IF NOT EXISTS user_roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL,
    role_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    assigned_by UUID,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE (user_id, role_id, tenant_id)
);

COMMENT ON TABLE user_roles IS 'IAM用户角色关联表 - 多对多关系';

-- 5. 角色权限关联表
CREATE TABLE IF NOT EXISTS role_permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    role_id UUID NOT NULL,
    permission_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE (role_id, permission_id)
);

COMMENT ON TABLE role_permissions IS 'IAM角色权限关联表 - 多对多关系';

-- 6. 用户会话表 (JWT令牌管理)
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    token_id VARCHAR(255) NOT NULL,
    refresh_token TEXT NOT NULL,
    user_agent TEXT,
    ip_address INET,
    expires_at TIMESTAMPTZ NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE user_sessions IS 'IAM用户会话表 - JWT令牌会话管理';
COMMENT ON COLUMN user_sessions.token_id IS '访问令牌唯一标识';
COMMENT ON COLUMN user_sessions.refresh_token IS '刷新令牌（用于获取新的访问令牌）';

-- =============================================================================
-- 外键约束
-- =============================================================================

-- users表外键
DO $$
BEGIN
    -- 检查tenants表是否存在以决定是否添加外键
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'tenants') THEN
        -- 如果外键不存在则添加
        IF NOT EXISTS (
            SELECT FROM information_schema.table_constraints 
            WHERE constraint_name = 'fk_users_tenant_id' AND table_name = 'users'
        ) THEN
            ALTER TABLE users ADD CONSTRAINT fk_users_tenant_id 
            FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
        END IF;
    END IF;
END $$;

-- roles表外键
DO $$
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'tenants') THEN
        IF NOT EXISTS (
            SELECT FROM information_schema.table_constraints 
            WHERE constraint_name = 'fk_roles_tenant_id' AND table_name = 'roles'
        ) THEN
            ALTER TABLE roles ADD CONSTRAINT fk_roles_tenant_id 
            FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
        END IF;
    END IF;
END $$;

-- permissions表外键
DO $$
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'tenants') THEN
        IF NOT EXISTS (
            SELECT FROM information_schema.table_constraints 
            WHERE constraint_name = 'fk_permissions_tenant_id' AND table_name = 'permissions'
        ) THEN
            ALTER TABLE permissions ADD CONSTRAINT fk_permissions_tenant_id 
            FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
        END IF;
    END IF;
END $$;

-- user_roles表外键
ALTER TABLE user_roles 
ADD CONSTRAINT IF NOT EXISTS fk_user_roles_user_id 
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE user_roles 
ADD CONSTRAINT IF NOT EXISTS fk_user_roles_role_id 
FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE;

DO $$
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'tenants') THEN
        IF NOT EXISTS (
            SELECT FROM information_schema.table_constraints 
            WHERE constraint_name = 'fk_user_roles_tenant_id' AND table_name = 'user_roles'
        ) THEN
            ALTER TABLE user_roles ADD CONSTRAINT fk_user_roles_tenant_id 
            FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
        END IF;
    END IF;
END $$;

ALTER TABLE user_roles 
ADD CONSTRAINT IF NOT EXISTS fk_user_roles_assigned_by 
FOREIGN KEY (assigned_by) REFERENCES users(id);

-- role_permissions表外键
ALTER TABLE role_permissions 
ADD CONSTRAINT IF NOT EXISTS fk_role_permissions_role_id 
FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE;

ALTER TABLE role_permissions 
ADD CONSTRAINT IF NOT EXISTS fk_role_permissions_permission_id 
FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE;

DO $$
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'tenants') THEN
        IF NOT EXISTS (
            SELECT FROM information_schema.table_constraints 
            WHERE constraint_name = 'fk_role_permissions_tenant_id' AND table_name = 'role_permissions'
        ) THEN
            ALTER TABLE role_permissions ADD CONSTRAINT fk_role_permissions_tenant_id 
            FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
        END IF;
    END IF;
END $$;

-- user_sessions表外键
ALTER TABLE user_sessions 
ADD CONSTRAINT IF NOT EXISTS fk_user_sessions_user_id 
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

DO $$
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'tenants') THEN
        IF NOT EXISTS (
            SELECT FROM information_schema.table_constraints 
            WHERE constraint_name = 'fk_user_sessions_tenant_id' AND table_name = 'user_sessions'
        ) THEN
            ALTER TABLE user_sessions ADD CONSTRAINT fk_user_sessions_tenant_id 
            FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
        END IF;
    END IF;
END $$;

-- =============================================================================
-- 索引优化 (IAM性能优化)
-- =============================================================================

-- 用户表索引
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_active 
ON users(email) WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_tenant 
ON users(username, tenant_id) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);
CREATE INDEX IF NOT EXISTS idx_users_last_login ON users(last_login_at);
CREATE INDEX IF NOT EXISTS idx_users_locked_until ON users(locked_until);

-- 角色表索引
CREATE INDEX IF NOT EXISTS idx_roles_tenant_id ON roles(tenant_id);
CREATE INDEX IF NOT EXISTS idx_roles_name_tenant ON roles(name, tenant_id);
CREATE INDEX IF NOT EXISTS idx_roles_is_system ON roles(is_system);

-- 权限表索引
CREATE INDEX IF NOT EXISTS idx_permissions_tenant_id ON permissions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_permissions_resource_action ON permissions(resource, action);
CREATE INDEX IF NOT EXISTS idx_permissions_name_tenant ON permissions(name, tenant_id);

-- 用户角色关联表索引
CREATE INDEX IF NOT EXISTS idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles(role_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_tenant_id ON user_roles(tenant_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_user_tenant ON user_roles(user_id, tenant_id);

-- 角色权限关联表索引
CREATE INDEX IF NOT EXISTS idx_role_permissions_role_id ON role_permissions(role_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_permission_id ON role_permissions(permission_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_tenant_id ON role_permissions(tenant_id);

-- 用户会话表索引
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_tenant_id ON user_sessions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_token_id ON user_sessions(token_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_refresh_token ON user_sessions(refresh_token);
CREATE INDEX IF NOT EXISTS idx_user_sessions_active ON user_sessions(is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);

-- =============================================================================
-- 行级安全策略 (RLS) - IAM多租户隔离
-- =============================================================================

-- 启用RLS
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE permissions ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE role_permissions ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_sessions ENABLE ROW LEVEL SECURITY;

-- 用户表RLS策略
DROP POLICY IF EXISTS users_tenant_isolation ON users;
CREATE POLICY users_tenant_isolation ON users
FOR ALL
USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- 角色表RLS策略
DROP POLICY IF EXISTS roles_tenant_isolation ON roles;
CREATE POLICY roles_tenant_isolation ON roles
FOR ALL
USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- 权限表RLS策略
DROP POLICY IF EXISTS permissions_tenant_isolation ON permissions;
CREATE POLICY permissions_tenant_isolation ON permissions
FOR ALL
USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- 用户角色关联表RLS策略
DROP POLICY IF EXISTS user_roles_tenant_isolation ON user_roles;
CREATE POLICY user_roles_tenant_isolation ON user_roles
FOR ALL
USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- 角色权限关联表RLS策略
DROP POLICY IF EXISTS role_permissions_tenant_isolation ON role_permissions;
CREATE POLICY role_permissions_tenant_isolation ON role_permissions
FOR ALL
USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- 用户会话表RLS策略
DROP POLICY IF EXISTS user_sessions_tenant_isolation ON user_sessions;
CREATE POLICY user_sessions_tenant_isolation ON user_sessions
FOR ALL
USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- 触发器 - 自动更新时间戳
-- =============================================================================

-- 为IAM表创建自动更新updated_at触发器
DO $$
BEGIN
    -- 用户表触发器
    IF NOT EXISTS (
        SELECT FROM information_schema.triggers 
        WHERE trigger_name = 'trigger_update_users_updated_at'
    ) THEN
        CREATE TRIGGER trigger_update_users_updated_at
            BEFORE UPDATE ON users
            FOR EACH ROW
            EXECUTE FUNCTION update_updated_at_column();
    END IF;
    
    -- 角色表触发器
    IF NOT EXISTS (
        SELECT FROM information_schema.triggers 
        WHERE trigger_name = 'trigger_update_roles_updated_at'
    ) THEN
        CREATE TRIGGER trigger_update_roles_updated_at
            BEFORE UPDATE ON roles
            FOR EACH ROW
            EXECUTE FUNCTION update_updated_at_column();
    END IF;
    
    -- 用户会话表触发器
    IF NOT EXISTS (
        SELECT FROM information_schema.triggers 
        WHERE trigger_name = 'trigger_update_user_sessions_updated_at'
    ) THEN
        CREATE TRIGGER trigger_update_user_sessions_updated_at
            BEFORE UPDATE ON user_sessions
            FOR EACH ROW
            EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;

-- =============================================================================
-- 默认数据插入 - IAM基础权限和角色
-- =============================================================================

-- 插入系统基础权限（如果不存在tenant）
-- 注意：实际部署时应该在有具体tenant后通过应用层插入

-- 系统权限定义（注释形式，供参考）
/*
基础权限矩阵：
- user.read: 查看用户信息
- user.write: 修改用户信息  
- user.delete: 删除用户
- user.manage: 管理用户（包含增删改查）

- role.read: 查看角色
- role.write: 修改角色
- role.delete: 删除角色
- role.manage: 管理角色

- project.read: 查看项目
- project.write: 修改项目
- project.delete: 删除项目
- project.manage: 管理项目

- repository.read: 查看代码仓库
- repository.write: 推送代码
- repository.delete: 删除仓库
- repository.manage: 管理仓库

基础角色定义：
- admin: 租户管理员（所有权限）
- manager: 项目管理员（项目相关权限）
- developer: 开发者（读写代码权限）
- viewer: 只读用户（查看权限）
- user: 普通用户（基础权限）
*/

-- =============================================================================
-- 数据一致性检查函数
-- =============================================================================

-- 检查IAM数据一致性的函数
CREATE OR REPLACE FUNCTION check_iam_data_consistency()
RETURNS TABLE(
    check_name TEXT,
    status TEXT,
    message TEXT
) AS $$
BEGIN
    -- 检查孤立的用户角色关联
    RETURN QUERY
    SELECT 
        'orphaned_user_roles'::TEXT,
        CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END::TEXT,
        FORMAT('Found %s orphaned user role assignments', COUNT(*))::TEXT
    FROM user_roles ur
    LEFT JOIN users u ON ur.user_id = u.id
    LEFT JOIN roles r ON ur.role_id = r.id
    WHERE u.id IS NULL OR r.id IS NULL;

    -- 检查孤立的角色权限关联
    RETURN QUERY
    SELECT 
        'orphaned_role_permissions'::TEXT,
        CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END::TEXT,
        FORMAT('Found %s orphaned role permission assignments', COUNT(*))::TEXT
    FROM role_permissions rp
    LEFT JOIN roles r ON rp.role_id = r.id
    LEFT JOIN permissions p ON rp.permission_id = p.id
    WHERE r.id IS NULL OR p.id IS NULL;

    -- 检查过期的用户会话
    RETURN QUERY
    SELECT 
        'expired_sessions'::TEXT,
        'INFO'::TEXT,
        FORMAT('Found %s expired but still active sessions', COUNT(*))::TEXT
    FROM user_sessions
    WHERE is_active = true AND expires_at < NOW();

    -- 检查锁定状态已过期但仍标记为锁定的用户
    RETURN QUERY
    SELECT 
        'unlockable_users'::TEXT,
        'INFO'::TEXT,
        FORMAT('Found %s users that can be unlocked', COUNT(*))::TEXT
    FROM users
    WHERE locked_until IS NOT NULL AND locked_until < NOW();

END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION check_iam_data_consistency() IS 'IAM数据一致性检查函数';

-- =============================================================================
-- 会话清理函数
-- =============================================================================

-- 清理过期会话的函数
CREATE OR REPLACE FUNCTION cleanup_expired_sessions()
RETURNS INTEGER AS $$
DECLARE
    cleaned_count INTEGER;
BEGIN
    -- 将过期的活跃会话标记为非活跃
    UPDATE user_sessions 
    SET 
        is_active = false,
        revoked_at = NOW()
    WHERE 
        is_active = true 
        AND expires_at < NOW();
    
    GET DIAGNOSTICS cleaned_count = ROW_COUNT;
    
    RETURN cleaned_count;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION cleanup_expired_sessions() IS '清理过期的用户会话';

-- =============================================================================
-- 用户解锁函数
-- =============================================================================

-- 解锁过期锁定用户的函数
CREATE OR REPLACE FUNCTION unlock_expired_users()
RETURNS INTEGER AS $$
DECLARE
    unlocked_count INTEGER;
BEGIN
    -- 解锁已过锁定期的用户
    UPDATE users 
    SET 
        locked_until = NULL,
        failed_login_count = 0
    WHERE 
        locked_until IS NOT NULL 
        AND locked_until < NOW();
    
    GET DIAGNOSTICS unlocked_count = ROW_COUNT;
    
    RETURN unlocked_count;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION unlock_expired_users() IS '解锁过期锁定的用户账户';

-- =============================================================================
-- 迁移完成标记
-- =============================================================================

-- 在system_settings中记录IAM迁移完成标记
DO $$
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'system_settings') THEN
        INSERT INTO system_settings (key, value, description, is_public) 
        VALUES (
            'migration.iam_service.completed', 
            'true', 
            'IAM Service Schema Migration Completed',
            false
        ) ON CONFLICT (key) DO UPDATE SET 
            value = EXCLUDED.value,
            updated_at = NOW();
    END IF;
END $$;

-- 输出迁移完成信息
DO $$
BEGIN
    RAISE NOTICE '=== IAM Service Schema Migration Completed Successfully ===';
    RAISE NOTICE 'Tables created/updated: users, roles, permissions, user_roles, role_permissions, user_sessions';
    RAISE NOTICE 'RLS policies enabled for multi-tenant isolation';
    RAISE NOTICE 'Performance indexes created';
    RAISE NOTICE 'Data consistency functions available';
    RAISE NOTICE 'Migration timestamp: %', NOW();
END $$;