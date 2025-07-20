-- Migration: 008_api_tokens_schema.sql
-- Description: Create API tokens and related tables for API access management
-- Author: AI Assistant
-- Date: 2024-12-20

BEGIN;

-- Create API scopes table
CREATE TABLE IF NOT EXISTS api_scopes (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description VARCHAR(500),
    category VARCHAR(50),
    resources JSONB DEFAULT '[]',
    actions JSONB DEFAULT '[]',
    is_system BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create API tokens table
CREATE TABLE IF NOT EXISTS api_tokens (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description VARCHAR(1000),
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    token_prefix VARCHAR(10) NOT NULL,
    scopes JSONB DEFAULT '[]',
    permissions JSONB DEFAULT '[]',
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'revoked', 'expired', 'suspended', 'deleted')),
    last_used_at TIMESTAMP WITH TIME ZONE,
    last_used_ip VARCHAR(45),
    use_count BIGINT DEFAULT 0,
    rate_limit_rps INTEGER DEFAULT 100 CHECK (rate_limit_rps > 0),
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoked_by UUID REFERENCES users(id) ON DELETE SET NULL
);

-- Create API token scopes relationship table
CREATE TABLE IF NOT EXISTS api_token_scopes (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    token_id UUID NOT NULL REFERENCES api_tokens(id) ON DELETE CASCADE,
    scope_id UUID NOT NULL REFERENCES api_scopes(id) ON DELETE CASCADE,
    granted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create API token usage tracking table
CREATE TABLE IF NOT EXISTS api_token_usage (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    token_id UUID NOT NULL REFERENCES api_tokens(id) ON DELETE CASCADE,
    endpoint VARCHAR(500) NOT NULL,
    method VARCHAR(10) NOT NULL,
    status_code INTEGER NOT NULL,
    response_time BIGINT NOT NULL, -- microseconds
    ip_address VARCHAR(45),
    user_agent VARCHAR(1000),
    request_size BIGINT DEFAULT 0,
    response_size BIGINT DEFAULT 0,
    error_message VARCHAR(1000),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create API rate limits table
CREATE TABLE IF NOT EXISTS api_rate_limits (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    token_id UUID NOT NULL REFERENCES api_tokens(id) ON DELETE CASCADE,
    window_start TIMESTAMP WITH TIME ZONE NOT NULL,
    window_size INTEGER NOT NULL, -- seconds
    request_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance optimization
CREATE INDEX IF NOT EXISTS idx_api_scopes_name ON api_scopes(name);
CREATE INDEX IF NOT EXISTS idx_api_scopes_category ON api_scopes(category);
CREATE INDEX IF NOT EXISTS idx_api_scopes_is_system ON api_scopes(is_system);

CREATE INDEX IF NOT EXISTS idx_api_tokens_tenant_id ON api_tokens(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_tokens_user_id ON api_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_api_tokens_token_hash ON api_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_api_tokens_status ON api_tokens(status);
CREATE INDEX IF NOT EXISTS idx_api_tokens_expires_at ON api_tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_api_tokens_last_used_at ON api_tokens(last_used_at);

CREATE INDEX IF NOT EXISTS idx_api_token_scopes_tenant_id ON api_token_scopes(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_token_scopes_token_id ON api_token_scopes(token_id);
CREATE INDEX IF NOT EXISTS idx_api_token_scopes_scope_id ON api_token_scopes(scope_id);

CREATE INDEX IF NOT EXISTS idx_api_token_usage_tenant_id ON api_token_usage(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_token_usage_token_id ON api_token_usage(token_id);
CREATE INDEX IF NOT EXISTS idx_api_token_usage_created_at ON api_token_usage(created_at);
CREATE INDEX IF NOT EXISTS idx_api_token_usage_endpoint ON api_token_usage(endpoint);
CREATE INDEX IF NOT EXISTS idx_api_token_usage_status_code ON api_token_usage(status_code);

CREATE INDEX IF NOT EXISTS idx_api_rate_limits_token_id ON api_rate_limits(token_id);
CREATE INDEX IF NOT EXISTS idx_api_rate_limits_window_start ON api_rate_limits(window_start);

-- Create unique constraints
CREATE UNIQUE INDEX IF NOT EXISTS idx_api_tokens_tenant_user_name 
    ON api_tokens(tenant_id, user_id, name) WHERE status != 'deleted';

CREATE UNIQUE INDEX IF NOT EXISTS idx_api_token_scopes_token_scope 
    ON api_token_scopes(token_id, scope_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_api_rate_limits_token_window 
    ON api_rate_limits(token_id, window_start);

-- Create composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_api_tokens_tenant_status_expires 
    ON api_tokens(tenant_id, status, expires_at);

CREATE INDEX IF NOT EXISTS idx_api_token_usage_token_date 
    ON api_token_usage(token_id, created_at);

CREATE INDEX IF NOT EXISTS idx_api_token_usage_tenant_date_status 
    ON api_token_usage(tenant_id, created_at, status_code);

-- Row Level Security (RLS) policies
ALTER TABLE api_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE api_token_scopes ENABLE ROW LEVEL SECURITY;
ALTER TABLE api_token_usage ENABLE ROW LEVEL SECURITY;
ALTER TABLE api_rate_limits ENABLE ROW LEVEL SECURITY;

-- RLS Policy for api_tokens table
CREATE POLICY api_tokens_tenant_isolation ON api_tokens
    FOR ALL
    TO authenticated_users
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

-- RLS Policy for api_token_scopes table
CREATE POLICY api_token_scopes_tenant_isolation ON api_token_scopes
    FOR ALL
    TO authenticated_users
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

-- RLS Policy for api_token_usage table
CREATE POLICY api_token_usage_tenant_isolation ON api_token_usage
    FOR ALL
    TO authenticated_users
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

-- RLS Policy for api_rate_limits table (related through token)
CREATE POLICY api_rate_limits_tenant_isolation ON api_rate_limits
    FOR ALL
    TO authenticated_users
    USING (
        token_id IN (
            SELECT id FROM api_tokens 
            WHERE tenant_id = current_setting('app.current_tenant_id')::UUID
        )
    );

-- Create updated_at trigger functions for API token tables
CREATE OR REPLACE FUNCTION update_api_tokens_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION update_api_scopes_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION update_api_rate_limits_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for updated_at columns
CREATE TRIGGER trigger_api_tokens_updated_at
    BEFORE UPDATE ON api_tokens
    FOR EACH ROW
    EXECUTE FUNCTION update_api_tokens_updated_at();

CREATE TRIGGER trigger_api_scopes_updated_at
    BEFORE UPDATE ON api_scopes
    FOR EACH ROW
    EXECUTE FUNCTION update_api_scopes_updated_at();

CREATE TRIGGER trigger_api_rate_limits_updated_at
    BEFORE UPDATE ON api_rate_limits
    FOR EACH ROW
    EXECUTE FUNCTION update_api_rate_limits_updated_at();

-- Create cleanup functions for API tokens
CREATE OR REPLACE FUNCTION cleanup_expired_api_tokens()
RETURNS INTEGER AS $$
DECLARE
    expired_count INTEGER;
BEGIN
    -- Mark expired tokens
    UPDATE api_tokens 
    SET status = 'expired', updated_at = CURRENT_TIMESTAMP
    WHERE expires_at < CURRENT_TIMESTAMP 
      AND status = 'active';
    
    GET DIAGNOSTICS expired_count = ROW_COUNT;
    
    -- Clean up old usage records (older than 90 days)
    DELETE FROM api_token_usage 
    WHERE created_at < CURRENT_TIMESTAMP - INTERVAL '90 days';
    
    -- Clean up old rate limit records (older than 1 day)
    DELETE FROM api_rate_limits 
    WHERE created_at < CURRENT_TIMESTAMP - INTERVAL '1 day';
    
    RETURN expired_count;
END;
$$ LANGUAGE plpgsql;

-- Create function to get token usage statistics
CREATE OR REPLACE FUNCTION get_api_token_usage_stats(
    p_tenant_id UUID,
    p_token_id UUID,
    p_days INTEGER DEFAULT 30
)
RETURNS TABLE (
    total_requests BIGINT,
    successful_requests BIGINT,
    error_requests BIGINT,
    avg_response_time BIGINT,
    top_endpoints JSONB
) AS $$
DECLARE
    start_date TIMESTAMP WITH TIME ZONE;
BEGIN
    start_date := CURRENT_TIMESTAMP - (p_days || ' days')::INTERVAL;
    
    RETURN QUERY
    WITH stats AS (
        SELECT 
            COUNT(*) as total_req,
            COUNT(CASE WHEN status_code < 400 THEN 1 END) as success_req,
            COUNT(CASE WHEN status_code >= 400 THEN 1 END) as error_req,
            COALESCE(AVG(response_time), 0)::BIGINT as avg_resp_time
        FROM api_token_usage
        WHERE tenant_id = p_tenant_id 
          AND token_id = p_token_id
          AND created_at >= start_date
    ),
    endpoints AS (
        SELECT jsonb_agg(
            jsonb_build_object(
                'endpoint', endpoint,
                'count', count
            ) ORDER BY count DESC
        ) as top_eps
        FROM (
            SELECT endpoint, COUNT(*) as count
            FROM api_token_usage
            WHERE tenant_id = p_tenant_id 
              AND token_id = p_token_id
              AND created_at >= start_date
            GROUP BY endpoint
            ORDER BY count DESC
            LIMIT 10
        ) t
    )
    SELECT 
        s.total_req,
        s.success_req,
        s.error_req,
        s.avg_resp_time,
        COALESCE(e.top_eps, '[]'::jsonb)
    FROM stats s
    CROSS JOIN endpoints e;
END;
$$ LANGUAGE plpgsql;

-- Insert default API scopes
INSERT INTO api_scopes (name, description, category, resources, actions, is_system) VALUES
    ('users:read', 'Read user information', 'users', '["users"]', '["read", "list"]', true),
    ('users:write', 'Create and update users', 'users', '["users"]', '["create", "update"]', true),
    ('users:delete', 'Delete users', 'users', '["users"]', '["delete"]', true),
    ('projects:read', 'Read project information', 'projects', '["projects", "tasks", "sprints"]', '["read", "list"]', true),
    ('projects:write', 'Create and update projects', 'projects', '["projects", "tasks", "sprints"]', '["create", "update"]', true),
    ('projects:admin', 'Full project administration', 'projects', '["projects", "tasks", "sprints", "members"]', '["create", "read", "update", "delete", "manage"]', true),
    ('repos:read', 'Read repository information', 'repositories', '["repositories", "commits", "branches"]', '["read", "list", "clone"]', true),
    ('repos:write', 'Push to repositories', 'repositories', '["repositories", "commits", "branches"]', '["push", "create_branch", "merge"]', true),
    ('repos:admin', 'Full repository administration', 'repositories', '["repositories", "commits", "branches", "hooks", "settings"]', '["create", "read", "update", "delete", "admin"]', true),
    ('cicd:read', 'Read CI/CD pipeline information', 'cicd', '["pipelines", "builds", "deployments"]', '["read", "list"]', true),
    ('cicd:trigger', 'Trigger CI/CD pipelines', 'cicd', '["pipelines", "builds"]', '["trigger", "cancel"]', true),
    ('cicd:admin', 'Full CI/CD administration', 'cicd', '["pipelines", "builds", "deployments", "runners"]', '["create", "read", "update", "delete", "manage"]', true),
    ('notifications:read', 'Read notifications', 'notifications', '["notifications"]', '["read", "list"]', true),
    ('notifications:write', 'Send notifications', 'notifications', '["notifications"]', '["create", "send"]', true),
    ('kb:read', 'Read knowledge base', 'knowledge', '["documents", "pages", "attachments"]', '["read", "search"]', true),
    ('kb:write', 'Create and edit knowledge base', 'knowledge', '["documents", "pages", "attachments"]', '["create", "update", "upload"]', true),
    ('admin:read', 'Read administrative information', 'admin', '["tenants", "settings", "logs", "metrics"]', '["read", "list"]', true),
    ('admin:write', 'Administrative actions', 'admin', '["tenants", "settings", "users", "permissions"]', '["create", "update", "manage"]', true)
ON CONFLICT (name) DO NOTHING;

-- Add comments for documentation
COMMENT ON TABLE api_scopes IS 'Available API scopes for token authorization';
COMMENT ON TABLE api_tokens IS 'Long-lived API access tokens for external integrations';
COMMENT ON TABLE api_token_scopes IS 'Relationship between API tokens and their granted scopes';
COMMENT ON TABLE api_token_usage IS 'Detailed usage tracking for API tokens';
COMMENT ON TABLE api_rate_limits IS 'Rate limiting tracking for API tokens';

COMMENT ON COLUMN api_tokens.token_hash IS 'SHA256 hash of the token for secure storage';
COMMENT ON COLUMN api_tokens.token_prefix IS 'First 10 characters of token for display purposes';
COMMENT ON COLUMN api_tokens.rate_limit_rps IS 'Requests per second limit for this token';
COMMENT ON COLUMN api_token_usage.response_time IS 'Response time in microseconds';
COMMENT ON COLUMN api_rate_limits.window_size IS 'Rate limiting window size in seconds';

COMMIT;