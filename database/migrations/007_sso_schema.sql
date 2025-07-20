-- Migration: 007_sso_schema.sql
-- Description: Create SSO (Single Sign-On) related tables and indexes
-- Author: AI Assistant
-- Date: 2024-12-20

BEGIN;

-- Create SSO providers table
CREATE TABLE IF NOT EXISTS sso_providers (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('saml', 'oauth2', 'oidc')),
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'disabled', 'deleted')),
    configuration JSONB NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    modified_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL
);

-- Create SSO sessions table
CREATE TABLE IF NOT EXISTS sso_sessions (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    provider_id UUID NOT NULL REFERENCES sso_providers(id) ON DELETE CASCADE,
    external_user_id VARCHAR(255) NOT NULL,
    state VARCHAR(255) NOT NULL,
    nonce VARCHAR(255),
    code_challenge VARCHAR(255),
    redirect_uri VARCHAR(1000),
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'failed', 'expired')),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create SSO user mappings table
CREATE TABLE IF NOT EXISTS sso_user_mappings (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider_id UUID NOT NULL REFERENCES sso_providers(id) ON DELETE CASCADE,
    external_user_id VARCHAR(255) NOT NULL,
    external_email VARCHAR(255),
    external_name VARCHAR(255),
    attributes JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance optimization
CREATE INDEX IF NOT EXISTS idx_sso_providers_tenant_id ON sso_providers(tenant_id);
CREATE INDEX IF NOT EXISTS idx_sso_providers_tenant_type ON sso_providers(tenant_id, type);
CREATE INDEX IF NOT EXISTS idx_sso_providers_status ON sso_providers(status);

CREATE INDEX IF NOT EXISTS idx_sso_sessions_tenant_id ON sso_sessions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_sso_sessions_provider_id ON sso_sessions(provider_id);
CREATE INDEX IF NOT EXISTS idx_sso_sessions_user_id ON sso_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sso_sessions_state ON sso_sessions(state);
CREATE INDEX IF NOT EXISTS idx_sso_sessions_status ON sso_sessions(status);
CREATE INDEX IF NOT EXISTS idx_sso_sessions_expires_at ON sso_sessions(expires_at);

CREATE INDEX IF NOT EXISTS idx_sso_user_mappings_tenant_id ON sso_user_mappings(tenant_id);
CREATE INDEX IF NOT EXISTS idx_sso_user_mappings_user_id ON sso_user_mappings(user_id);
CREATE INDEX IF NOT EXISTS idx_sso_user_mappings_provider_id ON sso_user_mappings(provider_id);
CREATE INDEX IF NOT EXISTS idx_sso_user_mappings_external_user_id ON sso_user_mappings(external_user_id);
CREATE INDEX IF NOT EXISTS idx_sso_user_mappings_external_email ON sso_user_mappings(external_email);

-- Create unique constraints
CREATE UNIQUE INDEX IF NOT EXISTS idx_sso_providers_tenant_name 
    ON sso_providers(tenant_id, name) WHERE status != 'deleted';

CREATE UNIQUE INDEX IF NOT EXISTS idx_sso_sessions_state 
    ON sso_sessions(state);

CREATE UNIQUE INDEX IF NOT EXISTS idx_sso_user_mappings_tenant_provider_external 
    ON sso_user_mappings(tenant_id, provider_id, external_user_id);

-- Create composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_sso_sessions_tenant_provider_status 
    ON sso_sessions(tenant_id, provider_id, status);

CREATE INDEX IF NOT EXISTS idx_sso_user_mappings_tenant_user_provider 
    ON sso_user_mappings(tenant_id, user_id, provider_id);

-- Row Level Security (RLS) policies
ALTER TABLE sso_providers ENABLE ROW LEVEL SECURITY;
ALTER TABLE sso_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE sso_user_mappings ENABLE ROW LEVEL SECURITY;

-- RLS Policy for sso_providers table
CREATE POLICY sso_providers_tenant_isolation ON sso_providers
    FOR ALL
    TO authenticated_users
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

-- RLS Policy for sso_sessions table
CREATE POLICY sso_sessions_tenant_isolation ON sso_sessions
    FOR ALL
    TO authenticated_users
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

-- RLS Policy for sso_user_mappings table
CREATE POLICY sso_user_mappings_tenant_isolation ON sso_user_mappings
    FOR ALL
    TO authenticated_users
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

-- Create updated_at trigger functions for SSO tables
CREATE OR REPLACE FUNCTION update_sso_providers_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION update_sso_sessions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION update_sso_user_mappings_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for updated_at columns
CREATE TRIGGER trigger_sso_providers_updated_at
    BEFORE UPDATE ON sso_providers
    FOR EACH ROW
    EXECUTE FUNCTION update_sso_providers_updated_at();

CREATE TRIGGER trigger_sso_sessions_updated_at
    BEFORE UPDATE ON sso_sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_sso_sessions_updated_at();

CREATE TRIGGER trigger_sso_user_mappings_updated_at
    BEFORE UPDATE ON sso_user_mappings
    FOR EACH ROW
    EXECUTE FUNCTION update_sso_user_mappings_updated_at();

-- Create cleanup function for expired SSO sessions
CREATE OR REPLACE FUNCTION cleanup_expired_sso_sessions()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    -- Delete expired SSO sessions (older than 24 hours)
    DELETE FROM sso_sessions 
    WHERE expires_at < CURRENT_TIMESTAMP - INTERVAL '24 hours'
      AND status IN ('expired', 'failed', 'completed');
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Create function to expire pending SSO sessions
CREATE OR REPLACE FUNCTION expire_pending_sso_sessions()
RETURNS INTEGER AS $$
DECLARE
    expired_count INTEGER;
BEGIN
    -- Mark expired sessions as expired
    UPDATE sso_sessions 
    SET status = 'expired', updated_at = CURRENT_TIMESTAMP
    WHERE status = 'pending' 
      AND expires_at < CURRENT_TIMESTAMP;
    
    GET DIAGNOSTICS expired_count = ROW_COUNT;
    
    RETURN expired_count;
END;
$$ LANGUAGE plpgsql;

-- Add comments for documentation
COMMENT ON TABLE sso_providers IS 'SSO identity provider configurations for each tenant';
COMMENT ON TABLE sso_sessions IS 'Active SSO authentication sessions with state tracking';
COMMENT ON TABLE sso_user_mappings IS 'Mapping between external SSO users and internal users';

COMMENT ON COLUMN sso_providers.type IS 'SSO provider type: saml, oauth2, or oidc';
COMMENT ON COLUMN sso_providers.configuration IS 'Provider-specific configuration (certificates, URLs, etc.)';
COMMENT ON COLUMN sso_providers.metadata IS 'Additional provider metadata and settings';

COMMENT ON COLUMN sso_sessions.state IS 'OAuth2/OIDC state parameter for CSRF protection';
COMMENT ON COLUMN sso_sessions.nonce IS 'OIDC nonce parameter for replay attack protection';
COMMENT ON COLUMN sso_sessions.code_challenge IS 'PKCE code challenge for OAuth2 security';

COMMENT ON COLUMN sso_user_mappings.external_user_id IS 'User ID from the external SSO provider';
COMMENT ON COLUMN sso_user_mappings.attributes IS 'Additional user attributes from SSO provider';

-- Insert sample configurations for testing (only in development)
-- Note: These should be removed or made conditional for production deployments
INSERT INTO sso_providers (tenant_id, name, type, configuration, created_by_user_id) 
SELECT 
    t.id,
    'Google OAuth2',
    'oauth2',
    '{
        "client_id": "your-google-client-id",
        "client_secret": "your-google-client-secret",
        "authorization_url": "https://accounts.google.com/o/oauth2/v2/auth",
        "token_url": "https://oauth2.googleapis.com/token",
        "user_info_url": "https://www.googleapis.com/oauth2/v2/userinfo",
        "redirect_url": "https://your-domain.com/api/v1/sso/callback",
        "scopes": ["openid", "email", "profile"],
        "attribute_mapping": {
            "email": "email",
            "first_name": "given_name",
            "last_name": "family_name"
        },
        "pkce_enabled": true,
        "state_param_required": true
    }'::jsonb,
    u.id
FROM tenants t
CROSS JOIN users u
WHERE t.name = 'Default Tenant' 
  AND u.username = 'admin@platform.local'
  AND NOT EXISTS (
      SELECT 1 FROM sso_providers sp 
      WHERE sp.tenant_id = t.id AND sp.name = 'Google OAuth2'
  )
LIMIT 1;

INSERT INTO sso_providers (tenant_id, name, type, configuration, created_by_user_id)
SELECT 
    t.id,
    'Azure AD OIDC',
    'oidc',
    '{
        "client_id": "your-azure-client-id",
        "client_secret": "your-azure-client-secret",
        "issuer_url": "https://login.microsoftonline.com/your-tenant-id/v2.0",
        "redirect_url": "https://your-domain.com/api/v1/sso/callback",
        "scopes": ["openid", "email", "profile"],
        "attribute_mapping": {
            "email": "email",
            "first_name": "given_name",
            "last_name": "family_name"
        },
        "pkce_enabled": true,
        "nonce_required": true,
        "validate_id_token": true,
        "allowed_issuers": ["https://login.microsoftonline.com/your-tenant-id/v2.0"]
    }'::jsonb,
    u.id
FROM tenants t
CROSS JOIN users u
WHERE t.name = 'Default Tenant' 
  AND u.username = 'admin@platform.local'
  AND NOT EXISTS (
      SELECT 1 FROM sso_providers sp 
      WHERE sp.tenant_id = t.id AND sp.name = 'Azure AD OIDC'
  )
LIMIT 1;

COMMIT;