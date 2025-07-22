-- Git Gateway Service 数据库迁移
-- 创建Git网关服务相关表

-- 启用UUID生成函数
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 创建uuid_generate_v7函数（如果不存在）
CREATE OR REPLACE FUNCTION uuid_generate_v7() RETURNS UUID AS $$
BEGIN
    RETURN gen_random_uuid();
END;
$$ LANGUAGE plpgsql;

-- Git仓库表
CREATE TABLE IF NOT EXISTS repositories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    project_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    visibility VARCHAR(20) NOT NULL DEFAULT 'private' CHECK (visibility IN ('public', 'private', 'internal')),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived', 'deleted')),
    default_branch VARCHAR(255) NOT NULL DEFAULT 'main',
    
    -- Git配置
    git_path VARCHAR(512) NOT NULL,
    clone_url VARCHAR(512),
    ssh_url VARCHAR(512),
    
    -- 统计信息
    size BIGINT DEFAULT 0,
    commit_count BIGINT DEFAULT 0,
    branch_count INTEGER DEFAULT 0,
    tag_count INTEGER DEFAULT 0,
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    last_pushed_at TIMESTAMP WITH TIME ZONE,
    
    -- 约束
    UNIQUE(project_id, name),
    INDEX idx_repositories_project_id (project_id),
    INDEX idx_repositories_status (status),
    INDEX idx_repositories_visibility (visibility),
    INDEX idx_repositories_deleted_at (deleted_at),
    INDEX idx_repositories_created_at (created_at)
);

-- Git分支表
CREATE TABLE IF NOT EXISTS branches (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    repository_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    commit_sha VARCHAR(40) NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    is_protected BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- 约束和索引
    UNIQUE(repository_id, name),
    INDEX idx_branches_repository_id (repository_id),
    INDEX idx_branches_default (is_default),
    INDEX idx_branches_protected (is_protected),
    INDEX idx_branches_deleted_at (deleted_at),
    
    -- 外键约束
    FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
);

-- Git提交表
CREATE TABLE IF NOT EXISTS commits (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    repository_id UUID NOT NULL,
    sha VARCHAR(40) NOT NULL,
    message TEXT NOT NULL,
    author VARCHAR(255) NOT NULL,
    author_email VARCHAR(255) NOT NULL,
    committer VARCHAR(255) NOT NULL,
    committer_email VARCHAR(255) NOT NULL,
    
    -- Git信息
    parent_shas JSONB,
    tree_sha VARCHAR(40) NOT NULL,
    
    -- 统计信息
    added_lines INTEGER DEFAULT 0,
    deleted_lines INTEGER DEFAULT 0,
    changed_files INTEGER DEFAULT 0,
    
    -- 时间戳
    committed_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- 约束和索引
    UNIQUE(repository_id, sha),
    INDEX idx_commits_repository_id (repository_id),
    INDEX idx_commits_sha (sha),
    INDEX idx_commits_author (author),
    INDEX idx_commits_committed_at (committed_at),
    
    -- 外键约束
    FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
);

-- Git提交文件变更表
CREATE TABLE IF NOT EXISTS commit_files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    commit_id UUID NOT NULL,
    file_path VARCHAR(1024) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('added', 'modified', 'deleted', 'renamed')),
    old_path VARCHAR(1024),
    
    -- 变更统计
    added_lines INTEGER DEFAULT 0,
    deleted_lines INTEGER DEFAULT 0,
    
    -- 约束和索引
    INDEX idx_commit_files_commit_id (commit_id),
    INDEX idx_commit_files_path (file_path),
    INDEX idx_commit_files_status (status),
    
    -- 外键约束
    FOREIGN KEY (commit_id) REFERENCES commits(id) ON DELETE CASCADE
);

-- Git标签表
CREATE TABLE IF NOT EXISTS tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    repository_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    commit_sha VARCHAR(40) NOT NULL,
    message TEXT,
    tagger VARCHAR(255),
    tagger_email VARCHAR(255),
    
    -- 时间戳
    tagged_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- 约束和索引
    UNIQUE(repository_id, name),
    INDEX idx_tags_repository_id (repository_id),
    INDEX idx_tags_commit_sha (commit_sha),
    INDEX idx_tags_tagged_at (tagged_at),
    
    -- 外键约束
    FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
);

-- Git Webhook表
CREATE TABLE IF NOT EXISTS webhooks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    repository_id UUID NOT NULL,
    url VARCHAR(512) NOT NULL,
    secret VARCHAR(255),
    events JSONB NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- 约束和索引
    INDEX idx_webhooks_repository_id (repository_id),
    INDEX idx_webhooks_active (is_active),
    
    -- 外键约束
    FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
);

-- 创建更新时间戳的触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 为需要自动更新时间戳的表创建触发器
CREATE TRIGGER update_repositories_updated_at
    BEFORE UPDATE ON repositories
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_branches_updated_at
    BEFORE UPDATE ON branches
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_webhooks_updated_at
    BEFORE UPDATE ON webhooks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 为仓库表创建全文搜索索引
CREATE INDEX IF NOT EXISTS idx_repositories_search 
ON repositories USING gin(to_tsvector('english', name || ' ' || COALESCE(description, '')));

-- 创建一些有用的视图

-- 仓库统计视图
CREATE OR REPLACE VIEW repository_stats_view AS
SELECT 
    r.id,
    r.name,
    r.project_id,
    r.visibility,
    r.status,
    r.size,
    COUNT(DISTINCT b.id) FILTER (WHERE b.deleted_at IS NULL) as branch_count,
    COUNT(DISTINCT c.id) as commit_count,
    COUNT(DISTINCT t.id) as tag_count,
    r.last_pushed_at,
    r.created_at,
    r.updated_at
FROM repositories r
LEFT JOIN branches b ON r.id = b.repository_id
LEFT JOIN commits c ON r.id = c.repository_id
LEFT JOIN tags t ON r.id = t.repository_id
WHERE r.deleted_at IS NULL
GROUP BY r.id, r.name, r.project_id, r.visibility, r.status, r.size, r.last_pushed_at, r.created_at, r.updated_at;

-- 分支提交统计视图
CREATE OR REPLACE VIEW branch_commit_stats_view AS
SELECT 
    b.id as branch_id,
    b.repository_id,
    b.name as branch_name,
    b.is_default,
    b.is_protected,
    COUNT(c.id) as commit_count,
    MAX(c.committed_at) as last_commit_at,
    b.created_at,
    b.updated_at
FROM branches b
LEFT JOIN commits c ON b.commit_sha = c.sha AND b.repository_id = c.repository_id
WHERE b.deleted_at IS NULL
GROUP BY b.id, b.repository_id, b.name, b.is_default, b.is_protected, b.created_at, b.updated_at;

-- 提交统计视图
CREATE OR REPLACE VIEW commit_stats_view AS
SELECT 
    c.id,
    c.repository_id,
    c.sha,
    c.message,
    c.author,
    c.author_email,
    c.committed_at,
    c.added_lines,
    c.deleted_lines,
    c.changed_files,
    COUNT(cf.id) as file_changes_count
FROM commits c
LEFT JOIN commit_files cf ON c.id = cf.commit_id
GROUP BY c.id, c.repository_id, c.sha, c.message, c.author, c.author_email, 
         c.committed_at, c.added_lines, c.deleted_lines, c.changed_files;

-- 插入一些示例数据（可选）
-- 这里可以添加一些测试数据，但在生产环境中应该删除

COMMENT ON TABLE repositories IS 'Git仓库信息表';
COMMENT ON TABLE branches IS 'Git分支信息表';
COMMENT ON TABLE commits IS 'Git提交信息表';
COMMENT ON TABLE commit_files IS 'Git提交文件变更表';
COMMENT ON TABLE tags IS 'Git标签信息表';
COMMENT ON TABLE webhooks IS 'Git Webhook配置表';

COMMENT ON COLUMN repositories.git_path IS '物理Git仓库路径';
COMMENT ON COLUMN repositories.clone_url IS 'HTTPS克隆URL';
COMMENT ON COLUMN repositories.ssh_url IS 'SSH克隆URL';
COMMENT ON COLUMN branches.commit_sha IS '分支指向的提交SHA';
COMMENT ON COLUMN commits.parent_shas IS '父提交SHA列表，JSON格式';
COMMENT ON COLUMN commits.tree_sha IS 'Git树对象SHA';
COMMENT ON COLUMN webhooks.events IS '监听的Git事件列表，JSON格式';