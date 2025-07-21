-- CI/CD服务数据库迁移
-- 创建流水线、执行记录、作业和执行器相关表

-- 创建流水线表
CREATE TABLE IF NOT EXISTS pipelines (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    repository_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    definition_file_path VARCHAR(512) NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- 创建流水线运行表
CREATE TABLE IF NOT EXISTS pipeline_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    pipeline_id UUID NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    trigger_type VARCHAR(50) NOT NULL DEFAULT 'manual',
    trigger_by UUID, -- 触发用户ID，可能为空（系统触发）
    commit_sha VARCHAR(40) NOT NULL,
    branch VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMP WITH TIME ZONE,
    finished_at TIMESTAMP WITH TIME ZONE,
    duration BIGINT, -- 持续时间（秒）
    variables JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 创建作业表
CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    pipeline_run_id UUID NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    stage VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    runner_id UUID, -- 执行器ID，可能为空（未分配）
    started_at TIMESTAMP WITH TIME ZONE,
    finished_at TIMESTAMP WITH TIME ZONE,
    duration BIGINT, -- 持续时间（秒）
    exit_code INTEGER,
    log_output TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 创建执行器表
CREATE TABLE IF NOT EXISTS runners (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    tags JSONB DEFAULT '[]',
    status VARCHAR(20) NOT NULL DEFAULT 'offline',
    version VARCHAR(50),
    os VARCHAR(50),
    architecture VARCHAR(50),
    last_contact_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 创建索引

-- 流水线表索引
CREATE INDEX IF NOT EXISTS idx_pipelines_repository_id ON pipelines(repository_id);
CREATE INDEX IF NOT EXISTS idx_pipelines_created_at ON pipelines(created_at);
CREATE INDEX IF NOT EXISTS idx_pipelines_deleted_at ON pipelines(deleted_at);
CREATE INDEX IF NOT EXISTS idx_pipelines_is_active ON pipelines(is_active);

-- 流水线运行表索引
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_pipeline_id ON pipeline_runs(pipeline_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_status ON pipeline_runs(status);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_trigger_by ON pipeline_runs(trigger_by);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_created_at ON pipeline_runs(created_at);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_commit_sha ON pipeline_runs(commit_sha);

-- 作业表索引
CREATE INDEX IF NOT EXISTS idx_jobs_pipeline_run_id ON jobs(pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_jobs_runner_id ON jobs(runner_id);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_stage ON jobs(stage);
CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs(created_at);

-- 执行器表索引
CREATE INDEX IF NOT EXISTS idx_runners_tenant_id ON runners(tenant_id);
CREATE INDEX IF NOT EXISTS idx_runners_status ON runners(status);
CREATE INDEX IF NOT EXISTS idx_runners_tags ON runners USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_runners_last_contact_at ON runners(last_contact_at);
CREATE INDEX IF NOT EXISTS idx_runners_created_at ON runners(created_at);

-- 添加约束

-- 流水线状态约束
ALTER TABLE pipeline_runs 
ADD CONSTRAINT IF NOT EXISTS chk_pipeline_run_status 
CHECK (status IN ('pending', 'running', 'success', 'failed', 'cancelled'));

-- 作业状态约束
ALTER TABLE jobs 
ADD CONSTRAINT IF NOT EXISTS chk_job_status 
CHECK (status IN ('pending', 'running', 'success', 'failed', 'cancelled'));

-- 执行器状态约束
ALTER TABLE runners 
ADD CONSTRAINT IF NOT EXISTS chk_runner_status 
CHECK (status IN ('online', 'offline', 'idle', 'busy'));

-- 触发类型约束
ALTER TABLE pipeline_runs 
ADD CONSTRAINT IF NOT EXISTS chk_trigger_type 
CHECK (trigger_type IN ('manual', 'push', 'pull_request', 'scheduled', 'webhook'));

-- 添加触发器

-- 流水线更新时间触发器
CREATE OR REPLACE FUNCTION update_pipeline_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_pipelines_updated_at
    BEFORE UPDATE ON pipelines
    FOR EACH ROW
    EXECUTE FUNCTION update_pipeline_updated_at();

-- 执行器更新时间触发器
CREATE OR REPLACE FUNCTION update_runner_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_runners_updated_at
    BEFORE UPDATE ON runners
    FOR EACH ROW
    EXECUTE FUNCTION update_runner_updated_at();

-- 流水线运行持续时间自动计算触发器
CREATE OR REPLACE FUNCTION calculate_pipeline_run_duration()
RETURNS TRIGGER AS $$
BEGIN
    -- 当状态变为完成状态且finished_at被设置时，自动计算持续时间
    IF NEW.status IN ('success', 'failed', 'cancelled') 
       AND NEW.finished_at IS NOT NULL 
       AND NEW.started_at IS NOT NULL THEN
        NEW.duration = EXTRACT(EPOCH FROM (NEW.finished_at - NEW.started_at));
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_pipeline_run_duration
    BEFORE UPDATE ON pipeline_runs
    FOR EACH ROW
    EXECUTE FUNCTION calculate_pipeline_run_duration();

-- 作业持续时间自动计算触发器
CREATE OR REPLACE FUNCTION calculate_job_duration()
RETURNS TRIGGER AS $$
BEGIN
    -- 当状态变为完成状态且finished_at被设置时，自动计算持续时间
    IF NEW.status IN ('success', 'failed', 'cancelled') 
       AND NEW.finished_at IS NOT NULL 
       AND NEW.started_at IS NOT NULL THEN
        NEW.duration = EXTRACT(EPOCH FROM (NEW.finished_at - NEW.started_at));
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_job_duration
    BEFORE UPDATE ON jobs
    FOR EACH ROW
    EXECUTE FUNCTION calculate_job_duration();

-- 创建视图

-- 流水线运行概览视图
CREATE OR REPLACE VIEW pipeline_run_overview AS
SELECT 
    pr.id,
    pr.pipeline_id,
    p.name as pipeline_name,
    p.repository_id,
    pr.status,
    pr.trigger_type,
    pr.commit_sha,
    pr.branch,
    pr.started_at,
    pr.finished_at,
    pr.duration,
    pr.created_at,
    COUNT(j.id) as total_jobs,
    COUNT(CASE WHEN j.status = 'success' THEN 1 END) as successful_jobs,
    COUNT(CASE WHEN j.status = 'failed' THEN 1 END) as failed_jobs,
    COUNT(CASE WHEN j.status = 'running' THEN 1 END) as running_jobs,
    COUNT(CASE WHEN j.status = 'pending' THEN 1 END) as pending_jobs
FROM pipeline_runs pr
JOIN pipelines p ON pr.pipeline_id = p.id
LEFT JOIN jobs j ON pr.id = j.pipeline_run_id
GROUP BY pr.id, p.id;

-- 执行器利用率视图
CREATE OR REPLACE VIEW runner_utilization AS
SELECT 
    r.id,
    r.name,
    r.status,
    r.tenant_id,
    r.last_contact_at,
    COUNT(j.id) as total_jobs_executed,
    COUNT(CASE WHEN j.status = 'success' THEN 1 END) as successful_jobs,
    COUNT(CASE WHEN j.status = 'failed' THEN 1 END) as failed_jobs,
    AVG(j.duration) as avg_job_duration,
    MAX(j.finished_at) as last_job_finished_at
FROM runners r
LEFT JOIN jobs j ON r.id = j.runner_id
GROUP BY r.id;

-- 流水线成功率统计视图
CREATE OR REPLACE VIEW pipeline_success_rate AS
SELECT 
    p.id as pipeline_id,
    p.name as pipeline_name,
    p.repository_id,
    COUNT(pr.id) as total_runs,
    COUNT(CASE WHEN pr.status = 'success' THEN 1 END) as successful_runs,
    COUNT(CASE WHEN pr.status = 'failed' THEN 1 END) as failed_runs,
    CASE 
        WHEN COUNT(pr.id) > 0 THEN 
            ROUND((COUNT(CASE WHEN pr.status = 'success' THEN 1 END) * 100.0 / COUNT(pr.id)), 2)
        ELSE 0 
    END as success_rate,
    AVG(pr.duration) as avg_duration,
    MAX(pr.created_at) as last_run_at
FROM pipelines p
LEFT JOIN pipeline_runs pr ON p.id = pr.pipeline_id
WHERE p.deleted_at IS NULL
GROUP BY p.id;

-- 添加注释
COMMENT ON TABLE pipelines IS 'CI/CD流水线定义表';
COMMENT ON TABLE pipeline_runs IS 'CI/CD流水线执行记录表';
COMMENT ON TABLE jobs IS 'CI/CD作业执行记录表';
COMMENT ON TABLE runners IS 'CI/CD执行器表';

COMMENT ON COLUMN pipelines.definition_file_path IS '流水线定义文件路径';
COMMENT ON COLUMN pipeline_runs.trigger_type IS '触发类型：manual, push, pull_request, scheduled, webhook';
COMMENT ON COLUMN pipeline_runs.variables IS '流水线执行变量，JSON格式';
COMMENT ON COLUMN jobs.stage IS '作业所属阶段';
COMMENT ON COLUMN jobs.exit_code IS '作业退出码';
COMMENT ON COLUMN runners.tags IS '执行器标签，用于作业匹配';
COMMENT ON COLUMN runners.last_contact_at IS '最后心跳时间';

-- 创建函数

-- 获取流水线统计信息函数
CREATE OR REPLACE FUNCTION get_pipeline_stats(
    p_pipeline_id UUID,
    p_days INTEGER DEFAULT 30
)
RETURNS TABLE (
    total_runs BIGINT,
    successful_runs BIGINT,
    failed_runs BIGINT,
    success_rate NUMERIC,
    avg_duration NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*) as total_runs,
        COUNT(CASE WHEN status = 'success' THEN 1 END) as successful_runs,
        COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_runs,
        CASE 
            WHEN COUNT(*) > 0 THEN 
                ROUND((COUNT(CASE WHEN status = 'success' THEN 1 END) * 100.0 / COUNT(*)), 2)
            ELSE 0 
        END as success_rate,
        AVG(duration) as avg_duration
    FROM pipeline_runs
    WHERE pipeline_id = p_pipeline_id
    AND created_at >= NOW() - INTERVAL '1 day' * p_days;
END;
$$ LANGUAGE plpgsql;

-- 获取可用执行器函数
CREATE OR REPLACE FUNCTION get_available_runners(
    p_tenant_id UUID,
    p_tags TEXT[] DEFAULT NULL
)
RETURNS TABLE (
    runner_id UUID,
    runner_name VARCHAR(255),
    runner_tags JSONB,
    status VARCHAR(20)
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        r.id as runner_id,
        r.name as runner_name,
        r.tags as runner_tags,
        r.status::VARCHAR(20)
    FROM runners r
    WHERE r.tenant_id = p_tenant_id
    AND r.status IN ('online', 'idle')
    AND (p_tags IS NULL OR r.tags ?& p_tags); -- 包含所有指定标签
END;
$$ LANGUAGE plpgsql;

-- 清理旧的流水线运行记录函数
CREATE OR REPLACE FUNCTION cleanup_old_pipeline_runs(
    p_retention_days INTEGER DEFAULT 30
)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    -- 删除超过保留期的完成状态的流水线运行记录
    DELETE FROM pipeline_runs
    WHERE created_at < NOW() - INTERVAL '1 day' * p_retention_days
    AND status IN ('success', 'failed', 'cancelled');
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- 数据完整性检查
DO $$
BEGIN
    -- 检查必要的扩展是否已安装
    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'uuid-ossp') THEN
        RAISE EXCEPTION 'uuid-ossp扩展未安装，请先安装该扩展';
    END IF;
    
    -- 检查uuid_generate_v7函数是否存在
    IF NOT EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'uuid_generate_v7') THEN
        RAISE NOTICE 'uuid_generate_v7函数不存在，将使用uuid_generate_v4作为备选';
    END IF;
    
    RAISE NOTICE 'CI/CD服务数据库迁移完成';
END
$$;