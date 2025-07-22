-- 项目Dashboard和DORA指标数据库迁移
-- 创建仪表板、DORA指标、项目健康度等相关表

-- 创建项目仪表板表
CREATE TABLE IF NOT EXISTS project_dashboards (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    project_id UUID NOT NULL UNIQUE REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Dashboard配置
    layout JSONB DEFAULT '{}', -- 布局配置JSON
    widgets JSONB DEFAULT '[]', -- 启用的组件列表
    refresh_rate INTEGER DEFAULT 300, -- 刷新间隔(秒)
    
    -- 审计字段
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by UUID
);

-- 创建DORA指标表
CREATE TABLE IF NOT EXISTS dora_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    metric_date TIMESTAMP WITH TIME ZONE NOT NULL,
    
    -- 部署频率 (Deployment Frequency)
    deployment_count INTEGER DEFAULT 0,
    deployment_frequency DECIMAL(10,4), -- 部署/天
    
    -- 变更前置时间 (Lead Time for Changes)
    lead_time_hours DECIMAL(10,2), -- 平均前置时间(小时)
    lead_time_p50 DECIMAL(10,2), -- 中位数
    lead_time_p90 DECIMAL(10,2), -- 90分位数
    
    -- 变更失败率 (Change Failure Rate)
    total_changes INTEGER DEFAULT 0,
    failed_changes INTEGER DEFAULT 0,
    change_failure_rate DECIMAL(5,4), -- 失败率百分比
    
    -- 恢复时间 (Mean Time to Recovery)
    incident_count INTEGER DEFAULT 0,
    recovery_time_hours DECIMAL(10,2), -- 平均恢复时间(小时)
    mttr DECIMAL(10,2), -- 平均故障恢复时间
    
    -- 综合评分
    dora_level VARCHAR(20), -- Elite/High/Medium/Low
    overall_score DECIMAL(5,2), -- 0-100综合评分
    
    -- 审计字段
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 创建项目健康度指标表
CREATE TABLE IF NOT EXISTS project_health_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    metric_date TIMESTAMP WITH TIME ZONE NOT NULL,
    
    -- 代码质量指标
    code_coverage DECIMAL(5,2), -- 代码覆盖率
    technical_debt DECIMAL(10,2), -- 技术债务(小时)
    code_duplication DECIMAL(5,2), -- 代码重复率
    cyclomatic_complexity DECIMAL(10,2), -- 圈复杂度
    
    -- 团队协作指标
    active_developers INTEGER, -- 活跃开发者数
    commit_frequency DECIMAL(10,2), -- 提交频率
    code_review_coverage DECIMAL(5,2), -- 代码审查覆盖率
    pr_merge_time_hours DECIMAL(10,2), -- PR平均合并时间
    
    -- 交付效率指标
    velocity_points INTEGER, -- 团队速度(故事点)
    sprint_goal_success DECIMAL(5,2), -- Sprint目标达成率
    task_completion_rate DECIMAL(5,2), -- 任务完成率
    bug_rate DECIMAL(5,2), -- 缺陷率
    
    -- 质量指标
    test_pass_rate DECIMAL(5,2), -- 测试通过率
    build_success_rate DECIMAL(5,2), -- 构建成功率
    security_vulnerabilities INTEGER, -- 安全漏洞数
    
    -- 综合健康评分
    health_score DECIMAL(5,2), -- 0-100健康评分
    risk_level VARCHAR(20), -- Low/Medium/High/Critical
    
    -- 审计字段
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 创建指标趋势表
CREATE TABLE IF NOT EXISTS metric_trends (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    metric_type VARCHAR(100) NOT NULL, -- dora, health, velocity等
    metric_name VARCHAR(100) NOT NULL, -- 具体指标名称
    metric_value DECIMAL(15,4) NOT NULL,
    period VARCHAR(20) NOT NULL, -- daily, weekly, monthly
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    
    -- 趋势分析
    trend_direction VARCHAR(20), -- up, down, stable
    change_percent DECIMAL(7,2), -- 变化百分比
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 创建告警规则表
CREATE TABLE IF NOT EXISTS alert_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- 规则配置
    metric_type VARCHAR(100) NOT NULL, -- dora, health等
    metric_name VARCHAR(100) NOT NULL, -- 具体指标
    operator VARCHAR(20) NOT NULL, -- >, <, >=, <=, ==
    threshold DECIMAL(15,4) NOT NULL,
    severity VARCHAR(20) NOT NULL, -- low, medium, high, critical
    
    -- 状态管理
    is_enabled BOOLEAN DEFAULT TRUE,
    last_triggered TIMESTAMP WITH TIME ZONE,
    trigger_count INTEGER DEFAULT 0,
    
    -- 通知配置
    notification_channels JSONB DEFAULT '[]', -- email, slack, webhook
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by UUID
);

-- 创建仪表板组件表
CREATE TABLE IF NOT EXISTS dashboard_widgets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    dashboard_id UUID NOT NULL REFERENCES project_dashboards(id) ON DELETE CASCADE,
    widget_type VARCHAR(50) NOT NULL, -- chart, metric, table等
    title VARCHAR(255) NOT NULL,
    
    -- 位置和大小
    position_x INTEGER NOT NULL,
    position_y INTEGER NOT NULL,
    width INTEGER NOT NULL DEFAULT 4,
    height INTEGER NOT NULL DEFAULT 3,
    
    -- 配置
    configuration JSONB, -- 组件配置JSON
    data_source VARCHAR(100), -- 数据源
    refresh_rate INTEGER DEFAULT 300, -- 刷新间隔
    
    -- 状态
    is_visible BOOLEAN DEFAULT TRUE,
    "order" INTEGER DEFAULT 0,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 创建索引

-- 项目仪表板索引
CREATE INDEX IF NOT EXISTS idx_project_dashboards_project_id ON project_dashboards(project_id);
CREATE INDEX IF NOT EXISTS idx_project_dashboards_deleted_at ON project_dashboards(deleted_at);
CREATE INDEX IF NOT EXISTS idx_project_dashboards_created_by ON project_dashboards(created_by);

-- DORA指标索引
CREATE INDEX IF NOT EXISTS idx_dora_metrics_project_id ON dora_metrics(project_id);
CREATE INDEX IF NOT EXISTS idx_dora_metrics_metric_date ON dora_metrics(metric_date);
CREATE INDEX IF NOT EXISTS idx_dora_metrics_dora_level ON dora_metrics(dora_level);
CREATE INDEX IF NOT EXISTS idx_dora_metrics_project_date ON dora_metrics(project_id, metric_date DESC);

-- 项目健康度指标索引
CREATE INDEX IF NOT EXISTS idx_project_health_project_id ON project_health_metrics(project_id);
CREATE INDEX IF NOT EXISTS idx_project_health_metric_date ON project_health_metrics(metric_date);
CREATE INDEX IF NOT EXISTS idx_project_health_risk_level ON project_health_metrics(risk_level);
CREATE INDEX IF NOT EXISTS idx_project_health_project_date ON project_health_metrics(project_id, metric_date DESC);

-- 指标趋势索引
CREATE INDEX IF NOT EXISTS idx_metric_trends_project_id ON metric_trends(project_id);
CREATE INDEX IF NOT EXISTS idx_metric_trends_metric_type ON metric_trends(metric_type);
CREATE INDEX IF NOT EXISTS idx_metric_trends_period_start ON metric_trends(period_start);
CREATE INDEX IF NOT EXISTS idx_metric_trends_project_metric ON metric_trends(project_id, metric_type, metric_name);

-- 告警规则索引
CREATE INDEX IF NOT EXISTS idx_alert_rules_project_id ON alert_rules(project_id);
CREATE INDEX IF NOT EXISTS idx_alert_rules_metric_type ON alert_rules(metric_type);
CREATE INDEX IF NOT EXISTS idx_alert_rules_is_enabled ON alert_rules(is_enabled);
CREATE INDEX IF NOT EXISTS idx_alert_rules_deleted_at ON alert_rules(deleted_at);

-- 仪表板组件索引
CREATE INDEX IF NOT EXISTS idx_dashboard_widgets_dashboard_id ON dashboard_widgets(dashboard_id);
CREATE INDEX IF NOT EXISTS idx_dashboard_widgets_widget_type ON dashboard_widgets(widget_type);
CREATE INDEX IF NOT EXISTS idx_dashboard_widgets_is_visible ON dashboard_widgets(is_visible);
CREATE INDEX IF NOT EXISTS idx_dashboard_widgets_order ON dashboard_widgets("order");

-- 添加约束

-- DORA等级约束
ALTER TABLE dora_metrics 
ADD CONSTRAINT IF NOT EXISTS chk_dora_level 
CHECK (dora_level IN ('elite', 'high', 'medium', 'low'));

-- DORA评分约束
ALTER TABLE dora_metrics 
ADD CONSTRAINT IF NOT EXISTS chk_dora_overall_score 
CHECK (overall_score >= 0 AND overall_score <= 100);

-- 失败率约束
ALTER TABLE dora_metrics 
ADD CONSTRAINT IF NOT EXISTS chk_change_failure_rate 
CHECK (change_failure_rate >= 0 AND change_failure_rate <= 1);

-- 健康评分约束
ALTER TABLE project_health_metrics 
ADD CONSTRAINT IF NOT EXISTS chk_health_score 
CHECK (health_score >= 0 AND health_score <= 100);

-- 风险等级约束
ALTER TABLE project_health_metrics 
ADD CONSTRAINT IF NOT EXISTS chk_risk_level 
CHECK (risk_level IN ('low', 'medium', 'high', 'critical'));

-- 百分比字段约束
ALTER TABLE project_health_metrics 
ADD CONSTRAINT IF NOT EXISTS chk_percentage_fields 
CHECK (
    code_coverage >= 0 AND code_coverage <= 100 AND
    code_duplication >= 0 AND code_duplication <= 100 AND
    code_review_coverage >= 0 AND code_review_coverage <= 100 AND
    sprint_goal_success >= 0 AND sprint_goal_success <= 100 AND
    task_completion_rate >= 0 AND task_completion_rate <= 100 AND
    bug_rate >= 0 AND bug_rate <= 100 AND
    test_pass_rate >= 0 AND test_pass_rate <= 100 AND
    build_success_rate >= 0 AND build_success_rate <= 100
);

-- 告警级别约束
ALTER TABLE alert_rules 
ADD CONSTRAINT IF NOT EXISTS chk_alert_severity 
CHECK (severity IN ('low', 'medium', 'high', 'critical'));

-- 告警操作符约束
ALTER TABLE alert_rules 
ADD CONSTRAINT IF NOT EXISTS chk_alert_operator 
CHECK (operator IN ('>', '<', '>=', '<=', '==', '!='));

-- 趋势方向约束
ALTER TABLE metric_trends 
ADD CONSTRAINT IF NOT EXISTS chk_trend_direction 
CHECK (trend_direction IN ('up', 'down', 'stable'));

-- 周期约束
ALTER TABLE metric_trends 
ADD CONSTRAINT IF NOT EXISTS chk_period 
CHECK (period IN ('daily', 'weekly', 'monthly', 'quarterly', 'yearly'));

-- 组件类型约束
ALTER TABLE dashboard_widgets 
ADD CONSTRAINT IF NOT EXISTS chk_widget_type 
CHECK (widget_type IN ('chart', 'metric', 'table', 'progress', 'gauge', 'heatmap'));

-- 位置约束
ALTER TABLE dashboard_widgets 
ADD CONSTRAINT IF NOT EXISTS chk_widget_position 
CHECK (position_x >= 0 AND position_y >= 0);

-- 尺寸约束
ALTER TABLE dashboard_widgets 
ADD CONSTRAINT IF NOT EXISTS chk_widget_size 
CHECK (width > 0 AND height > 0);

-- 添加触发器

-- 项目仪表板更新时间触发器
CREATE OR REPLACE FUNCTION update_dashboard_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_project_dashboards_updated_at
    BEFORE UPDATE ON project_dashboards
    FOR EACH ROW
    EXECUTE FUNCTION update_dashboard_updated_at();

-- DORA指标更新时间触发器
CREATE OR REPLACE FUNCTION update_dora_metrics_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    -- 自动计算DORA等级和综合评分
    NEW.dora_level = calculate_dora_level(
        NEW.deployment_frequency,
        NEW.lead_time_hours,
        NEW.change_failure_rate,
        NEW.mttr
    );
    NEW.overall_score = calculate_dora_score(
        NEW.deployment_frequency,
        NEW.lead_time_hours,
        NEW.change_failure_rate,
        NEW.mttr
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_dora_metrics_updated_at
    BEFORE UPDATE ON dora_metrics
    FOR EACH ROW
    EXECUTE FUNCTION update_dora_metrics_updated_at();

-- 项目健康度指标更新时间触发器
CREATE OR REPLACE FUNCTION update_health_metrics_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    -- 自动计算健康评分和风险等级
    NEW.health_score = calculate_health_score(
        NEW.code_coverage,
        NEW.code_duplication,
        NEW.test_pass_rate,
        NEW.build_success_rate,
        NEW.code_review_coverage,
        NEW.sprint_goal_success,
        NEW.task_completion_rate,
        NEW.bug_rate,
        NEW.technical_debt,
        NEW.security_vulnerabilities
    );
    NEW.risk_level = calculate_risk_level(NEW.health_score);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_health_metrics_updated_at
    BEFORE UPDATE ON project_health_metrics
    FOR EACH ROW
    EXECUTE FUNCTION update_health_metrics_updated_at();

-- 告警规则更新时间触发器
CREATE OR REPLACE FUNCTION update_alert_rules_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_alert_rules_updated_at
    BEFORE UPDATE ON alert_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_alert_rules_updated_at();

-- 创建函数

-- 计算DORA等级
CREATE OR REPLACE FUNCTION calculate_dora_level(
    p_deployment_frequency DECIMAL,
    p_lead_time_hours DECIMAL,
    p_change_failure_rate DECIMAL,
    p_mttr DECIMAL
)
RETURNS VARCHAR(20) AS $$
BEGIN
    -- Elite级别标准
    IF p_deployment_frequency >= 1.0 AND 
       p_lead_time_hours <= 24 AND 
       p_change_failure_rate <= 0.15 AND 
       p_mttr <= 1 THEN
        RETURN 'elite';
    END IF;
    
    -- High级别标准
    IF p_deployment_frequency >= 0.14 AND 
       p_lead_time_hours <= 168 AND 
       p_change_failure_rate <= 0.2 AND 
       p_mttr <= 24 THEN
        RETURN 'high';
    END IF;
    
    -- Medium级别标准
    IF p_deployment_frequency >= 0.033 AND 
       p_lead_time_hours <= 720 AND 
       p_change_failure_rate <= 0.3 AND 
       p_mttr <= 168 THEN
        RETURN 'medium';
    END IF;
    
    RETURN 'low';
END;
$$ LANGUAGE plpgsql;

-- 计算DORA综合评分
CREATE OR REPLACE FUNCTION calculate_dora_score(
    p_deployment_frequency DECIMAL,
    p_lead_time_hours DECIMAL,
    p_change_failure_rate DECIMAL,
    p_mttr DECIMAL
)
RETURNS DECIMAL(5,2) AS $$
DECLARE
    score DECIMAL(5,2) := 0;
BEGIN
    -- 部署频率评分 (0-25分)
    IF p_deployment_frequency >= 1.0 THEN
        score := score + 25;
    ELSIF p_deployment_frequency >= 0.14 THEN
        score := score + 20;
    ELSIF p_deployment_frequency >= 0.033 THEN
        score := score + 15;
    ELSE
        score := score + 10;
    END IF;
    
    -- 前置时间评分 (0-25分)
    IF p_lead_time_hours <= 24 THEN
        score := score + 25;
    ELSIF p_lead_time_hours <= 168 THEN
        score := score + 20;
    ELSIF p_lead_time_hours <= 720 THEN
        score := score + 15;
    ELSE
        score := score + 10;
    END IF;
    
    -- 失败率评分 (0-25分)
    IF p_change_failure_rate <= 0.15 THEN
        score := score + 25;
    ELSIF p_change_failure_rate <= 0.2 THEN
        score := score + 20;
    ELSIF p_change_failure_rate <= 0.3 THEN
        score := score + 15;
    ELSE
        score := score + 10;
    END IF;
    
    -- 恢复时间评分 (0-25分)
    IF p_mttr <= 1 THEN
        score := score + 25;
    ELSIF p_mttr <= 24 THEN
        score := score + 20;
    ELSIF p_mttr <= 168 THEN
        score := score + 15;
    ELSE
        score := score + 10;
    END IF;
    
    RETURN score;
END;
$$ LANGUAGE plpgsql;

-- 计算健康评分
CREATE OR REPLACE FUNCTION calculate_health_score(
    p_code_coverage DECIMAL,
    p_code_duplication DECIMAL,
    p_test_pass_rate DECIMAL,
    p_build_success_rate DECIMAL,
    p_code_review_coverage DECIMAL,
    p_sprint_goal_success DECIMAL,
    p_task_completion_rate DECIMAL,
    p_bug_rate DECIMAL,
    p_technical_debt DECIMAL,
    p_security_vulnerabilities INTEGER
)
RETURNS DECIMAL(5,2) AS $$
DECLARE
    score DECIMAL(5,2) := 0;
    quality_score DECIMAL(5,2);
    collaboration_score DECIMAL(5,2);
    efficiency_score DECIMAL(5,2);
    debt_security_score DECIMAL(5,2);
BEGIN
    -- 代码质量 (30%)
    quality_score := (COALESCE(p_code_coverage, 0) + 
                     (100 - COALESCE(p_code_duplication, 0)) + 
                     COALESCE(p_test_pass_rate, 0) +
                     COALESCE(p_build_success_rate, 0)) / 4;
    score := score + (quality_score * 0.3);
    
    -- 团队协作 (25%)
    collaboration_score := (COALESCE(p_code_review_coverage, 0) + 
                           COALESCE(p_sprint_goal_success, 0) + 
                           COALESCE(p_task_completion_rate, 0)) / 3;
    score := score + (collaboration_score * 0.25);
    
    -- 交付效率 (25%)
    efficiency_score := (COALESCE(p_sprint_goal_success, 0) + 
                        COALESCE(p_task_completion_rate, 0) + 
                        (100 - COALESCE(p_bug_rate, 0))) / 3;
    score := score + (efficiency_score * 0.25);
    
    -- 技术债务和安全 (20%)
    debt_security_score := 100 - (COALESCE(p_technical_debt, 0)/10 + COALESCE(p_security_vulnerabilities, 0)*5);
    IF debt_security_score < 0 THEN
        debt_security_score := 0;
    END IF;
    score := score + (debt_security_score * 0.2);
    
    IF score > 100 THEN
        score := 100;
    END IF;
    
    RETURN score;
END;
$$ LANGUAGE plpgsql;

-- 计算风险等级
CREATE OR REPLACE FUNCTION calculate_risk_level(
    p_health_score DECIMAL
)
RETURNS VARCHAR(20) AS $$
BEGIN
    IF p_health_score >= 80 THEN
        RETURN 'low';
    ELSIF p_health_score >= 60 THEN
        RETURN 'medium';
    ELSIF p_health_score >= 40 THEN
        RETURN 'high';
    END IF;
    RETURN 'critical';
END;
$$ LANGUAGE plpgsql;

-- 创建视图

-- DORA指标概览视图
CREATE OR REPLACE VIEW dora_overview AS
SELECT 
    dm.project_id,
    p.name as project_name,
    dm.metric_date,
    dm.deployment_frequency,
    dm.lead_time_hours,
    dm.change_failure_rate * 100 as change_failure_percentage,
    dm.mttr,
    dm.dora_level,
    dm.overall_score,
    -- 趋势计算（与上一期比较）
    LAG(dm.overall_score, 1) OVER (
        PARTITION BY dm.project_id 
        ORDER BY dm.metric_date
    ) as previous_score,
    dm.overall_score - LAG(dm.overall_score, 1) OVER (
        PARTITION BY dm.project_id 
        ORDER BY dm.metric_date
    ) as score_change
FROM dora_metrics dm
JOIN projects p ON dm.project_id = p.id
WHERE p.deleted_at IS NULL
ORDER BY dm.project_id, dm.metric_date DESC;

-- 项目健康度概览视图
CREATE OR REPLACE VIEW health_overview AS
SELECT 
    phm.project_id,
    p.name as project_name,
    phm.metric_date,
    phm.health_score,
    phm.risk_level,
    phm.code_coverage,
    phm.test_pass_rate,
    phm.build_success_rate,
    phm.task_completion_rate,
    phm.bug_rate,
    phm.security_vulnerabilities,
    -- 趋势计算
    LAG(phm.health_score, 1) OVER (
        PARTITION BY phm.project_id 
        ORDER BY phm.metric_date
    ) as previous_health_score,
    phm.health_score - LAG(phm.health_score, 1) OVER (
        PARTITION BY phm.project_id 
        ORDER BY phm.metric_date
    ) as health_score_change
FROM project_health_metrics phm
JOIN projects p ON phm.project_id = p.id
WHERE p.deleted_at IS NULL
ORDER BY phm.project_id, phm.metric_date DESC;

-- Dashboard组件数据视图
CREATE OR REPLACE VIEW dashboard_widget_data AS
SELECT 
    dw.id as widget_id,
    dw.dashboard_id,
    pd.project_id,
    dw.widget_type,
    dw.title,
    dw.position_x,
    dw.position_y,
    dw.width,
    dw.height,
    dw.configuration,
    dw.data_source,
    dw.is_visible,
    dw.order,
    pd.name as dashboard_name,
    p.name as project_name
FROM dashboard_widgets dw
JOIN project_dashboards pd ON dw.dashboard_id = pd.id
JOIN projects p ON pd.project_id = p.id
WHERE pd.deleted_at IS NULL AND dw.is_visible = TRUE
ORDER BY pd.project_id, dw.order, dw.position_y, dw.position_x;

-- 添加注释
COMMENT ON TABLE project_dashboards IS '项目仪表板配置表';
COMMENT ON TABLE dora_metrics IS 'DORA四大指标数据表';
COMMENT ON TABLE project_health_metrics IS '项目健康度指标表';
COMMENT ON TABLE metric_trends IS '指标趋势数据表';
COMMENT ON TABLE alert_rules IS '告警规则配置表';
COMMENT ON TABLE dashboard_widgets IS '仪表板组件表';

COMMENT ON COLUMN dora_metrics.deployment_frequency IS '部署频率(次/天)';
COMMENT ON COLUMN dora_metrics.lead_time_hours IS '变更前置时间(小时)';
COMMENT ON COLUMN dora_metrics.change_failure_rate IS '变更失败率(0-1)';
COMMENT ON COLUMN dora_metrics.mttr IS '平均故障恢复时间(小时)';
COMMENT ON COLUMN project_health_metrics.health_score IS '项目健康综合评分(0-100)';
COMMENT ON COLUMN dashboard_widgets.configuration IS '组件配置JSON，包含图表类型、数据源参数等';

-- 数据完整性检查
DO $$
BEGIN
    RAISE NOTICE '项目Dashboard和DORA指标数据库迁移完成';
    RAISE NOTICE '已创建6个表、26个索引、4个触发器、5个函数和3个视图';
END
$$;