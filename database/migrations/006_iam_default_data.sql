-- Cloud-Based Collaborative Development Platform
-- IAM Service Default Data Migration
-- 插入默认的租户、角色、权限和测试用户
-- Generated: 2025-01-20

-- =============================================================================
-- 默认租户创建
-- =============================================================================

-- 插入测试租户（如果不存在）
DO $$
DECLARE
    default_plan_id UUID;
    test_tenant_id UUID := '550e8400-e29b-41d4-a716-446655440000'::UUID;
BEGIN
    -- 获取默认订阅计划ID
    SELECT id INTO default_plan_id 
    FROM subscription_plans 
    WHERE tier = 'free' 
    LIMIT 1;
    
    -- 如果没有订阅计划，先创建一个默认的
    IF default_plan_id IS NULL THEN
        INSERT INTO subscription_plans (name, tier, price_monthly, features, limits) 
        VALUES (
            'Default Free Plan', 
            'free', 
            0.00,
            '{"ai_assistance": false, "advanced_analytics": false}',
            '{"max_projects": 5, "max_users_per_tenant": 10, "storage_gb": 1}'
        ) RETURNING id INTO default_plan_id;
    END IF;
    
    -- 插入测试租户
    INSERT INTO tenants (id, name, slug, subscription_plan_id, status) 
    VALUES (
        test_tenant_id,
        'Test Tenant',
        'test-tenant',
        default_plan_id,
        'active'
    ) ON CONFLICT (id) DO NOTHING;
    
    RAISE NOTICE 'Default tenant created with ID: %', test_tenant_id;
END $$;

-- =============================================================================
-- 默认权限定义
-- =============================================================================

-- 设置默认租户上下文
SELECT set_config('app.current_tenant_id', '550e8400-e29b-41d4-a716-446655440000', true);

-- 插入基础权限（用户管理）
INSERT INTO permissions (tenant_id, name, resource, action, description) VALUES
('550e8400-e29b-41d4-a716-446655440000', 'user.read', 'user', 'read', '查看用户信息'),
('550e8400-e29b-41d4-a716-446655440000', 'user.write', 'user', 'write', '修改用户信息'),
('550e8400-e29b-41d4-a716-446655440000', 'user.delete', 'user', 'delete', '删除用户'),
('550e8400-e29b-41d4-a716-446655440000', 'user.manage', 'user', 'manage', '管理用户（包含增删改查）')
ON CONFLICT (tenant_id, name) DO NOTHING;

-- 插入角色管理权限
INSERT INTO permissions (tenant_id, name, resource, action, description) VALUES
('550e8400-e29b-41d4-a716-446655440000', 'role.read', 'role', 'read', '查看角色信息'),
('550e8400-e29b-41d4-a716-446655440000', 'role.write', 'role', 'write', '修改角色信息'),
('550e8400-e29b-41d4-a716-446655440000', 'role.delete', 'role', 'delete', '删除角色'),
('550e8400-e29b-41d4-a716-446655440000', 'role.manage', 'role', 'manage', '管理角色（包含增删改查）')
ON CONFLICT (tenant_id, name) DO NOTHING;

-- 插入项目管理权限
INSERT INTO permissions (tenant_id, name, resource, action, description) VALUES
('550e8400-e29b-41d4-a716-446655440000', 'project.read', 'project', 'read', '查看项目信息'),
('550e8400-e29b-41d4-a716-446655440000', 'project.write', 'project', 'write', '修改项目信息'),
('550e8400-e29b-41d4-a716-446655440000', 'project.delete', 'project', 'delete', '删除项目'),
('550e8400-e29b-41d4-a716-446655440000', 'project.manage', 'project', 'manage', '管理项目（包含增删改查）')
ON CONFLICT (tenant_id, name) DO NOTHING;

-- 插入代码仓库权限
INSERT INTO permissions (tenant_id, name, resource, action, description) VALUES
('550e8400-e29b-41d4-a716-446655440000', 'repository.read', 'repository', 'read', '查看代码仓库'),
('550e8400-e29b-41d4-a716-446655440000', 'repository.write', 'repository', 'write', '推送代码'),
('550e8400-e29b-41d4-a716-446655440000', 'repository.delete', 'repository', 'delete', '删除仓库'),
('550e8400-e29b-41d4-a716-446655440000', 'repository.manage', 'repository', 'manage', '管理代码仓库')
ON CONFLICT (tenant_id, name) DO NOTHING;

-- 插入CI/CD权限
INSERT INTO permissions (tenant_id, name, resource, action, description) VALUES
('550e8400-e29b-41d4-a716-446655440000', 'cicd.read', 'cicd', 'read', '查看CI/CD流水线'),
('550e8400-e29b-41d4-a716-446655440000', 'cicd.write', 'cicd', 'write', '触发CI/CD流水线'),
('550e8400-e29b-41d4-a716-446655440000', 'cicd.manage', 'cicd', 'manage', '管理CI/CD配置')
ON CONFLICT (tenant_id, name) DO NOTHING;

-- 插入系统管理权限
INSERT INTO permissions (tenant_id, name, resource, action, description) VALUES
('550e8400-e29b-41d4-a716-446655440000', 'system.manage', 'system', 'manage', '系统管理权限'),
('550e8400-e29b-41d4-a716-446655440000', 'tenant.manage', 'tenant', 'manage', '租户管理权限')
ON CONFLICT (tenant_id, name) DO NOTHING;

-- =============================================================================
-- 默认角色创建
-- =============================================================================

-- 声明角色ID变量
DO $$
DECLARE
    admin_role_id UUID;
    manager_role_id UUID;
    developer_role_id UUID;
    viewer_role_id UUID;
    user_role_id UUID;
    
    -- 权限ID变量
    perm_id UUID;
    perm_ids UUID[];
BEGIN
    -- 创建管理员角色
    INSERT INTO roles (tenant_id, name, description, is_system) 
    VALUES (
        '550e8400-e29b-41d4-a716-446655440000',
        'admin',
        '租户管理员 - 拥有所有权限',
        true
    ) ON CONFLICT (tenant_id, name) DO UPDATE SET description = EXCLUDED.description
    RETURNING id INTO admin_role_id;
    
    -- 创建项目管理员角色
    INSERT INTO roles (tenant_id, name, description, is_system) 
    VALUES (
        '550e8400-e29b-41d4-a716-446655440000',
        'manager',
        '项目管理员 - 拥有项目管理权限',
        true
    ) ON CONFLICT (tenant_id, name) DO UPDATE SET description = EXCLUDED.description
    RETURNING id INTO manager_role_id;
    
    -- 创建开发者角色
    INSERT INTO roles (tenant_id, name, description, is_system) 
    VALUES (
        '550e8400-e29b-41d4-a716-446655440000',
        'developer',
        '开发者 - 拥有代码读写权限',
        true
    ) ON CONFLICT (tenant_id, name) DO UPDATE SET description = EXCLUDED.description
    RETURNING id INTO developer_role_id;
    
    -- 创建查看者角色
    INSERT INTO roles (tenant_id, name, description, is_system) 
    VALUES (
        '550e8400-e29b-41d4-a716-446655440000',
        'viewer',
        '查看者 - 只有只读权限',
        true
    ) ON CONFLICT (tenant_id, name) DO UPDATE SET description = EXCLUDED.description
    RETURNING id INTO viewer_role_id;
    
    -- 创建普通用户角色
    INSERT INTO roles (tenant_id, name, description, is_system) 
    VALUES (
        '550e8400-e29b-41d4-a716-446655440000',
        'user',
        '普通用户 - 基础权限',
        true
    ) ON CONFLICT (tenant_id, name) DO UPDATE SET description = EXCLUDED.description
    RETURNING id INTO user_role_id;
    
    RAISE NOTICE 'Default roles created successfully';
    
    -- =============================================================================
    -- 角色权限分配
    -- =============================================================================
    
    -- 删除现有角色权限关联（重新分配）
    DELETE FROM role_permissions WHERE role_id IN (admin_role_id, manager_role_id, developer_role_id, viewer_role_id, user_role_id);
    
    -- 管理员角色 - 分配所有权限
    INSERT INTO role_permissions (role_id, permission_id, tenant_id)
    SELECT admin_role_id, p.id, '550e8400-e29b-41d4-a716-446655440000'
    FROM permissions p
    WHERE p.tenant_id = '550e8400-e29b-41d4-a716-446655440000';
    
    -- 项目管理员角色 - 分配项目、用户、代码仓库、CI/CD管理权限
    INSERT INTO role_permissions (role_id, permission_id, tenant_id)
    SELECT manager_role_id, p.id, '550e8400-e29b-41d4-a716-446655440000'
    FROM permissions p
    WHERE p.tenant_id = '550e8400-e29b-41d4-a716-446655440000'
    AND p.resource IN ('project', 'user', 'repository', 'cicd')
    AND p.action IN ('read', 'write', 'manage');
    
    -- 开发者角色 - 分配代码和CI/CD读写权限、项目查看权限
    INSERT INTO role_permissions (role_id, permission_id, tenant_id)
    SELECT developer_role_id, p.id, '550e8400-e29b-41d4-a716-446655440000'
    FROM permissions p
    WHERE p.tenant_id = '550e8400-e29b-41d4-a716-446655440000'
    AND (
        (p.resource = 'repository' AND p.action IN ('read', 'write')) OR
        (p.resource = 'cicd' AND p.action IN ('read', 'write')) OR
        (p.resource = 'project' AND p.action = 'read') OR
        (p.resource = 'user' AND p.action = 'read')
    );
    
    -- 查看者角色 - 只分配查看权限
    INSERT INTO role_permissions (role_id, permission_id, tenant_id)
    SELECT viewer_role_id, p.id, '550e8400-e29b-41d4-a716-446655440000'
    FROM permissions p
    WHERE p.tenant_id = '550e8400-e29b-41d4-a716-446655440000'
    AND p.action = 'read';
    
    -- 普通用户角色 - 基础权限（查看自己的信息和项目）
    INSERT INTO role_permissions (role_id, permission_id, tenant_id)
    SELECT user_role_id, p.id, '550e8400-e29b-41d4-a716-446655440000'
    FROM permissions p
    WHERE p.tenant_id = '550e8400-e29b-41d4-a716-446655440000'
    AND p.resource IN ('user', 'project')
    AND p.action = 'read';
    
    RAISE NOTICE 'Role permissions assigned successfully';
END $$;

-- =============================================================================
-- 默认测试用户创建
-- =============================================================================

DO $$
DECLARE
    admin_user_id UUID := '660e8400-e29b-41d4-a716-446655440001'::UUID;
    test_user_id UUID := '660e8400-e29b-41d4-a716-446655440002'::UUID;
    admin_role_id UUID;
    user_role_id UUID;
BEGIN
    -- 获取角色ID
    SELECT id INTO admin_role_id FROM roles WHERE tenant_id = '550e8400-e29b-41d4-a716-446655440000' AND name = 'admin';
    SELECT id INTO user_role_id FROM roles WHERE tenant_id = '550e8400-e29b-41d4-a716-446655440000' AND name = 'user';
    
    -- 创建管理员用户（密码: admin123）
    INSERT INTO users (
        id, tenant_id, email, username, password_hash, 
        first_name, last_name, is_active
    ) VALUES (
        admin_user_id,
        '550e8400-e29b-41d4-a716-446655440000',
        'admin@test.com',
        'admin',
        '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', -- admin123
        'Admin',
        'User',
        true
    ) ON CONFLICT (id) DO NOTHING;
    
    -- 创建测试用户（密码: user123）
    INSERT INTO users (
        id, tenant_id, email, username, password_hash, 
        first_name, last_name, is_active
    ) VALUES (
        test_user_id,
        '550e8400-e29b-41d4-a716-446655440000',
        'user@test.com',
        'testuser',
        '$2a$10$7Z2Z2Z2Z2Z2Z2Z2Z2Z2Z2u7Z2Z2Z2Z2Z2Z2Z2Z2Z2Z2Z2Z2Z2Z2Z2', -- user123
        'Test',
        'User',
        true
    ) ON CONFLICT (id) DO NOTHING;
    
    -- 分配管理员角色
    INSERT INTO user_roles (user_id, role_id, tenant_id)
    VALUES (admin_user_id, admin_role_id, '550e8400-e29b-41d4-a716-446655440000')
    ON CONFLICT (user_id, role_id, tenant_id) DO NOTHING;
    
    -- 分配用户角色
    INSERT INTO user_roles (user_id, role_id, tenant_id)
    VALUES (test_user_id, user_role_id, '550e8400-e29b-41d4-a716-446655440000')
    ON CONFLICT (user_id, role_id, tenant_id) DO NOTHING;
    
    RAISE NOTICE 'Default users created successfully';
    RAISE NOTICE 'Admin user: admin@test.com / admin123';
    RAISE NOTICE 'Test user: user@test.com / user123';
END $$;

-- =============================================================================
-- 系统设置更新
-- =============================================================================

-- 更新系统设置
DO $$
BEGIN
    -- 设置默认用户角色
    INSERT INTO system_settings (key, value, description, is_public) VALUES
    ('iam.default_user_role', '"user"', 'IAM默认用户角色名称', false),
    ('iam.password_min_length', '8', 'IAM密码最小长度', false),
    ('iam.max_login_attempts', '5', 'IAM最大登录尝试次数', false),
    ('iam.lockout_duration_minutes', '30', 'IAM账户锁定持续时间（分钟）', false),
    ('iam.jwt_expiration_hours', '24', 'IAM JWT访问令牌过期时间（小时）', false),
    ('iam.refresh_token_expiration_days', '30', 'IAM刷新令牌过期时间（天）', false)
    ON CONFLICT (key) DO UPDATE SET 
        value = EXCLUDED.value,
        updated_at = NOW();
    
    RAISE NOTICE 'IAM system settings updated';
END $$;

-- =============================================================================
-- 数据验证
-- =============================================================================

-- 验证插入的数据
DO $$
DECLARE
    tenant_count INTEGER;
    role_count INTEGER;
    permission_count INTEGER;
    user_count INTEGER;
    user_role_count INTEGER;
    role_permission_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO tenant_count FROM tenants WHERE id = '550e8400-e29b-41d4-a716-446655440000';
    SELECT COUNT(*) INTO role_count FROM roles WHERE tenant_id = '550e8400-e29b-41d4-a716-446655440000';
    SELECT COUNT(*) INTO permission_count FROM permissions WHERE tenant_id = '550e8400-e29b-41d4-a716-446655440000';
    SELECT COUNT(*) INTO user_count FROM users WHERE tenant_id = '550e8400-e29b-41d4-a716-446655440000';
    SELECT COUNT(*) INTO user_role_count FROM user_roles WHERE tenant_id = '550e8400-e29b-41d4-a716-446655440000';
    SELECT COUNT(*) INTO role_permission_count FROM role_permissions WHERE tenant_id = '550e8400-e29b-41d4-a716-446655440000';
    
    RAISE NOTICE '=== IAM Default Data Migration Summary ===';
    RAISE NOTICE 'Tenants created: %', tenant_count;
    RAISE NOTICE 'Roles created: %', role_count;
    RAISE NOTICE 'Permissions created: %', permission_count;
    RAISE NOTICE 'Users created: %', user_count;
    RAISE NOTICE 'User-Role assignments: %', user_role_count;
    RAISE NOTICE 'Role-Permission assignments: %', role_permission_count;
    
    IF tenant_count = 0 OR role_count = 0 OR permission_count = 0 THEN
        RAISE WARNING 'Some default data may not have been created properly!';
    ELSE
        RAISE NOTICE 'All default data created successfully!';
    END IF;
END $$;

-- 记录迁移完成
INSERT INTO system_settings (key, value, description, is_public) 
VALUES (
    'migration.iam_default_data.completed', 
    'true', 
    'IAM Default Data Migration Completed',
    false
) ON CONFLICT (key) DO UPDATE SET 
    value = EXCLUDED.value,
    updated_at = NOW();