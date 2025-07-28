-- 创建安全扫描结果表
CREATE TABLE IF NOT EXISTS security_scan_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL,
    scan_type VARCHAR(50) NOT NULL,
    project_path TEXT NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    duration BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL,
    vulnerabilities JSONB,
    summary JSONB,
    metadata JSONB,
    error TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX idx_security_scan_results_project_id ON security_scan_results(project_id);
CREATE INDEX idx_security_scan_results_scan_type ON security_scan_results(scan_type);
CREATE INDEX idx_security_scan_results_status ON security_scan_results(status);
CREATE INDEX idx_security_scan_results_created_at ON security_scan_results(created_at);

-- 创建更新时间触发器
CREATE OR REPLACE FUNCTION update_security_scan_results_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_security_scan_results_updated_at
    BEFORE UPDATE ON security_scan_results
    FOR EACH ROW
    EXECUTE FUNCTION update_security_scan_results_updated_at();

-- 添加注释
COMMENT ON TABLE security_scan_results IS '安全扫描结果表';
COMMENT ON COLUMN security_scan_results.id IS '扫描ID';
COMMENT ON COLUMN security_scan_results.project_id IS '项目ID';
COMMENT ON COLUMN security_scan_results.scan_type IS '扫描类型';
COMMENT ON COLUMN security_scan_results.project_path IS '项目路径';
COMMENT ON COLUMN security_scan_results.start_time IS '开始时间';
COMMENT ON COLUMN security_scan_results.end_time IS '结束时间';
COMMENT ON COLUMN security_scan_results.duration IS '持续时间（纳秒）';
COMMENT ON COLUMN security_scan_results.status IS '状态';
COMMENT ON COLUMN security_scan_results.vulnerabilities IS '漏洞列表';
COMMENT ON COLUMN security_scan_results.summary IS '扫描摘要';
COMMENT ON COLUMN security_scan_results.metadata IS '元数据';
COMMENT ON COLUMN security_scan_results.error IS '错误信息';
COMMENT ON COLUMN security_scan_results.created_at IS '创建时间';
COMMENT ON COLUMN security_scan_results.updated_at IS '更新时间';