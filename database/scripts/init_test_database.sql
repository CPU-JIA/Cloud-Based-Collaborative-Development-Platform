-- 测试数据库初始化脚本
-- 创建测试数据库和必要的扩展

-- 创建测试数据库（如果不存在）
CREATE DATABASE IF NOT EXISTS devcollab_test;

-- 连接到测试数据库
\c devcollab_test;

-- 启用必要的PostgreSQL扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 创建测试专用的配置函数（如果不存在默认的set_config）
-- 这些函数模拟PostgreSQL的set_config行为
CREATE OR REPLACE FUNCTION test_set_config(
    setting_name TEXT,
    new_value TEXT,
    is_local BOOLEAN DEFAULT FALSE
) RETURNS TEXT AS $$
BEGIN
    -- 在测试环境中，我们简单地返回设置的值
    -- 实际的PostgreSQL set_config函数会在session中设置参数
    RETURN new_value;
END;
$$ LANGUAGE plpgsql;

-- 创建测试专用的配置获取函数
CREATE OR REPLACE FUNCTION test_current_setting(
    setting_name TEXT,
    missing_ok BOOLEAN DEFAULT FALSE
) RETURNS TEXT AS $$
BEGIN
    -- 在测试环境中返回默认值
    CASE setting_name
        WHEN 'app.current_tenant_id' THEN RETURN 'default';
        WHEN 'app.current_user_id' THEN RETURN '1';
        ELSE 
            IF missing_ok THEN
                RETURN '';
            ELSE
                RAISE EXCEPTION 'unrecognized configuration parameter "%"', setting_name;
            END IF;
    END CASE;
END;
$$ LANGUAGE plpgsql;

-- 创建基础数据结构
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    subdomain VARCHAR(100) UNIQUE NOT NULL,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    username VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS teams (
    id SERIAL PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_by INTEGER REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS team_members (
    id SERIAL PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    team_id INTEGER REFERENCES teams(id),
    user_id INTEGER REFERENCES users(id),
    role VARCHAR(50) DEFAULT 'member',
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, team_id, user_id)
);

CREATE TABLE IF NOT EXISTS team_invitations (
    id SERIAL PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    team_id INTEGER REFERENCES teams(id),
    email VARCHAR(255) NOT NULL,
    token VARCHAR(255) UNIQUE NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    invited_by INTEGER REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() + INTERVAL '7 days'
);

CREATE TABLE IF NOT EXISTS permission_requests (
    id SERIAL PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    team_id INTEGER REFERENCES teams(id),
    user_id INTEGER REFERENCES users(id),
    requested_permission VARCHAR(100) NOT NULL,
    reason TEXT NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    reviewed_by INTEGER REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    reviewed_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS roles (
    id SERIAL PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    project_id INTEGER,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    permissions JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 插入测试数据
INSERT INTO tenants (id, name, subdomain, status) VALUES 
('cbd2db00-1ea3-412c-90dd-08b99b1258d0', 'Test Tenant', 'test', 'active')
ON CONFLICT (subdomain) DO NOTHING;

INSERT INTO users (id, tenant_id, username, email, password_hash, status) VALUES 
(1, 'cbd2db00-1ea3-412c-90dd-08b99b1258d0', 'testuser', 'test@example.com', '$2a$10$example.hash', 'active'),
(2, 'cbd2db00-1ea3-412c-90dd-08b99b1258d0', 'testuser2', 'test2@example.com', '$2a$10$example.hash', 'active'),
(3, 'cbd2db00-1ea3-412c-90dd-08b99b1258d0', 'testuser3', 'test3@example.com', '$2a$10$example.hash', 'active'),
(4, 'cbd2db00-1ea3-412c-90dd-08b99b1258d0', 'testuser4', 'test4@example.com', '$2a$10$example.hash', 'active'),
(5, 'cbd2db00-1ea3-412c-90dd-08b99b1258d0', 'inactive_user', 'inactive@example.com', '$2a$10$example.hash', 'inactive'),
(6, 'cbd2db00-1ea3-412c-90dd-08b99b1258d0', 'testuser6', 'test6@example.com', '$2a$10$example.hash', 'active')
ON CONFLICT (email) DO NOTHING;

-- 重置序列
SELECT setval('users_id_seq', COALESCE((SELECT MAX(id) FROM users), 1));

INSERT INTO roles (id, tenant_id, name, description) VALUES 
(1, 'cbd2db00-1ea3-412c-90dd-08b99b1258d0', 'owner', 'Team Owner'),
(2, 'cbd2db00-1ea3-412c-90dd-08b99b1258d0', 'admin', 'Team Admin'),
(3, 'cbd2db00-1ea3-412c-90dd-08b99b1258d0', 'member', 'Team Member'),
(4, 'cbd2db00-1ea3-412c-90dd-08b99b1258d0', 'viewer', 'Team Viewer')
ON CONFLICT DO NOTHING;

-- 重置序列
SELECT setval('roles_id_seq', COALESCE((SELECT MAX(id) FROM roles), 1));

-- 创建索引优化查询性能
CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_teams_tenant_id ON teams(tenant_id);
CREATE INDEX IF NOT EXISTS idx_team_members_tenant_team_user ON team_members(tenant_id, team_id, user_id);
CREATE INDEX IF NOT EXISTS idx_team_invitations_tenant_team ON team_invitations(tenant_id, team_id);
CREATE INDEX IF NOT EXISTS idx_team_invitations_token ON team_invitations(token);
CREATE INDEX IF NOT EXISTS idx_permission_requests_tenant_team ON permission_requests(tenant_id, team_id);
CREATE INDEX IF NOT EXISTS idx_roles_tenant_id ON roles(tenant_id);

-- 授予测试用户权限
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO postgres;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO postgres;

COMMIT;