-- 敏捷任务管理系统数据库迁移
-- 创建Sprint、敏捷任务、史诗、看板等相关表

-- 创建史诗表
CREATE TABLE IF NOT EXISTS epics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'open',
    color VARCHAR(7), -- 十六进制颜色代码
    
    -- 时间计划
    start_date TIMESTAMP WITH TIME ZONE,
    end_date TIMESTAMP WITH TIME ZONE,
    
    -- 目标和指标
    goal TEXT,
    success_criteria TEXT,
    
    -- 审计字段
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by UUID
);

-- 创建Sprint表
CREATE TABLE IF NOT EXISTS sprints (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    goal TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'planned',
    
    -- 时间规划
    start_date TIMESTAMP WITH TIME ZONE NOT NULL,
    end_date TIMESTAMP WITH TIME ZONE NOT NULL,
    
    -- 容量规划
    capacity INTEGER DEFAULT 0, -- 故事点容量
    
    -- 审计字段
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by UUID
);

-- 创建敏捷任务表
CREATE TABLE IF NOT EXISTS agile_tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    sprint_id UUID REFERENCES sprints(id) ON DELETE SET NULL,
    epic_id UUID REFERENCES epics(id) ON DELETE SET NULL,
    parent_id UUID REFERENCES agile_tasks(id) ON DELETE SET NULL,
    
    -- 任务基本信息
    task_number BIGINT NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL DEFAULT 'story',
    status VARCHAR(50) NOT NULL DEFAULT 'todo',
    priority VARCHAR(20) NOT NULL DEFAULT 'medium',
    
    -- 敏捷估算
    story_points INTEGER,
    original_estimate DECIMAL(10,2), -- 原始估算（小时）
    remaining_time DECIMAL(10,2),    -- 剩余时间（小时）
    logged_time DECIMAL(10,2) DEFAULT 0, -- 已记录时间（小时）
    
    -- 人员分配
    assignee_id UUID,
    reporter_id UUID NOT NULL,
    
    -- 标签和分类
    labels JSONB DEFAULT '[]',
    components JSONB DEFAULT '[]',
    
    -- 排序权重（用于看板拖拽排序）
    rank TEXT,
    
    -- 验收标准
    acceptance_criteria JSONB DEFAULT '[]',
    
    -- 审计字段
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- 创建任务评论表
CREATE TABLE IF NOT EXISTS task_comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    task_id UUID NOT NULL REFERENCES agile_tasks(id) ON DELETE CASCADE,
    author_id UUID NOT NULL,
    content TEXT NOT NULL,
    is_internal BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- 创建任务附件表
CREATE TABLE IF NOT EXISTS task_attachments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    task_id UUID NOT NULL REFERENCES agile_tasks(id) ON DELETE CASCADE,
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    uploaded_by UUID NOT NULL,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- 创建工作日志表
CREATE TABLE IF NOT EXISTS work_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    task_id UUID NOT NULL REFERENCES agile_tasks(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    
    time_spent DECIMAL(10,2) NOT NULL, -- 工作时长（小时）
    description TEXT,
    work_date TIMESTAMP WITH TIME ZONE NOT NULL,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- 创建看板表
CREATE TABLE IF NOT EXISTS boards (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL DEFAULT 'kanban', -- kanban, scrum
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by UUID
);

-- 创建看板列表
CREATE TABLE IF NOT EXISTS board_columns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    board_id UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    position INTEGER NOT NULL,
    wip_limit INTEGER, -- Work In Progress 限制
    status VARCHAR(50) NOT NULL, -- 对应的任务状态
    color VARCHAR(7),
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 创建索引

-- 史诗表索引
CREATE INDEX IF NOT EXISTS idx_epics_project_id ON epics(project_id);
CREATE INDEX IF NOT EXISTS idx_epics_status ON epics(status);
CREATE INDEX IF NOT EXISTS idx_epics_deleted_at ON epics(deleted_at);
CREATE INDEX IF NOT EXISTS idx_epics_created_by ON epics(created_by);

-- Sprint表索引
CREATE INDEX IF NOT EXISTS idx_sprints_project_id ON sprints(project_id);
CREATE INDEX IF NOT EXISTS idx_sprints_status ON sprints(status);
CREATE INDEX IF NOT EXISTS idx_sprints_start_date ON sprints(start_date);
CREATE INDEX IF NOT EXISTS idx_sprints_end_date ON sprints(end_date);
CREATE INDEX IF NOT EXISTS idx_sprints_deleted_at ON sprints(deleted_at);
CREATE INDEX IF NOT EXISTS idx_sprints_created_by ON sprints(created_by);

-- 敏捷任务表索引
CREATE INDEX IF NOT EXISTS idx_agile_tasks_project_id ON agile_tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_agile_tasks_sprint_id ON agile_tasks(sprint_id);
CREATE INDEX IF NOT EXISTS idx_agile_tasks_epic_id ON agile_tasks(epic_id);
CREATE INDEX IF NOT EXISTS idx_agile_tasks_parent_id ON agile_tasks(parent_id);
CREATE INDEX IF NOT EXISTS idx_agile_tasks_task_number ON agile_tasks(task_number);
CREATE INDEX IF NOT EXISTS idx_agile_tasks_type ON agile_tasks(type);
CREATE INDEX IF NOT EXISTS idx_agile_tasks_status ON agile_tasks(status);
CREATE INDEX IF NOT EXISTS idx_agile_tasks_priority ON agile_tasks(priority);
CREATE INDEX IF NOT EXISTS idx_agile_tasks_assignee_id ON agile_tasks(assignee_id);
CREATE INDEX IF NOT EXISTS idx_agile_tasks_reporter_id ON agile_tasks(reporter_id);
CREATE INDEX IF NOT EXISTS idx_agile_tasks_rank ON agile_tasks(rank);
CREATE INDEX IF NOT EXISTS idx_agile_tasks_deleted_at ON agile_tasks(deleted_at);
CREATE INDEX IF NOT EXISTS idx_agile_tasks_created_at ON agile_tasks(created_at);

-- 复合索引用于常见查询
CREATE INDEX IF NOT EXISTS idx_agile_tasks_project_sprint ON agile_tasks(project_id, sprint_id);
CREATE INDEX IF NOT EXISTS idx_agile_tasks_assignee_status ON agile_tasks(assignee_id, status);

-- 任务评论表索引
CREATE INDEX IF NOT EXISTS idx_task_comments_task_id ON task_comments(task_id);
CREATE INDEX IF NOT EXISTS idx_task_comments_author_id ON task_comments(author_id);
CREATE INDEX IF NOT EXISTS idx_task_comments_created_at ON task_comments(created_at);
CREATE INDEX IF NOT EXISTS idx_task_comments_deleted_at ON task_comments(deleted_at);

-- 任务附件表索引
CREATE INDEX IF NOT EXISTS idx_task_attachments_task_id ON task_attachments(task_id);
CREATE INDEX IF NOT EXISTS idx_task_attachments_uploaded_by ON task_attachments(uploaded_by);
CREATE INDEX IF NOT EXISTS idx_task_attachments_deleted_at ON task_attachments(deleted_at);

-- 工作日志表索引
CREATE INDEX IF NOT EXISTS idx_work_logs_task_id ON work_logs(task_id);
CREATE INDEX IF NOT EXISTS idx_work_logs_user_id ON work_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_work_logs_work_date ON work_logs(work_date);
CREATE INDEX IF NOT EXISTS idx_work_logs_deleted_at ON work_logs(deleted_at);

-- 看板表索引
CREATE INDEX IF NOT EXISTS idx_boards_project_id ON boards(project_id);
CREATE INDEX IF NOT EXISTS idx_boards_type ON boards(type);
CREATE INDEX IF NOT EXISTS idx_boards_deleted_at ON boards(deleted_at);
CREATE INDEX IF NOT EXISTS idx_boards_created_by ON boards(created_by);

-- 看板列索引
CREATE INDEX IF NOT EXISTS idx_board_columns_board_id ON board_columns(board_id);
CREATE INDEX IF NOT EXISTS idx_board_columns_position ON board_columns(position);
CREATE INDEX IF NOT EXISTS idx_board_columns_status ON board_columns(status);

-- 添加约束

-- 史诗状态约束
ALTER TABLE epics 
ADD CONSTRAINT IF NOT EXISTS chk_epic_status 
CHECK (status IN ('open', 'in_progress', 'done', 'cancelled'));

-- Sprint状态约束
ALTER TABLE sprints 
ADD CONSTRAINT IF NOT EXISTS chk_sprint_status 
CHECK (status IN ('planned', 'active', 'closed'));

-- Sprint时间约束
ALTER TABLE sprints 
ADD CONSTRAINT IF NOT EXISTS chk_sprint_dates 
CHECK (end_date > start_date);

-- 敏捷任务类型约束
ALTER TABLE agile_tasks 
ADD CONSTRAINT IF NOT EXISTS chk_task_type 
CHECK (type IN ('story', 'task', 'bug', 'epic', 'subtask'));

-- 敏捷任务状态约束
ALTER TABLE agile_tasks 
ADD CONSTRAINT IF NOT EXISTS chk_task_status 
CHECK (status IN ('todo', 'in_progress', 'in_review', 'testing', 'done', 'cancelled'));

-- 优先级约束
ALTER TABLE agile_tasks 
ADD CONSTRAINT IF NOT EXISTS chk_task_priority 
CHECK (priority IN ('lowest', 'low', 'medium', 'high', 'highest'));

-- 故事点约束（正数）
ALTER TABLE agile_tasks 
ADD CONSTRAINT IF NOT EXISTS chk_story_points 
CHECK (story_points IS NULL OR story_points > 0);

-- 时间约束（非负数）
ALTER TABLE agile_tasks 
ADD CONSTRAINT IF NOT EXISTS chk_time_estimates 
CHECK (original_estimate IS NULL OR original_estimate >= 0);

ALTER TABLE agile_tasks 
ADD CONSTRAINT IF NOT EXISTS chk_remaining_time 
CHECK (remaining_time IS NULL OR remaining_time >= 0);

ALTER TABLE agile_tasks 
ADD CONSTRAINT IF NOT EXISTS chk_logged_time 
CHECK (logged_time >= 0);

-- 工作日志时间约束（正数）
ALTER TABLE work_logs 
ADD CONSTRAINT IF NOT EXISTS chk_work_log_time_spent 
CHECK (time_spent > 0);

-- 看板类型约束
ALTER TABLE boards 
ADD CONSTRAINT IF NOT EXISTS chk_board_type 
CHECK (type IN ('kanban', 'scrum'));

-- 看板列位置约束（非负数）
ALTER TABLE board_columns 
ADD CONSTRAINT IF NOT EXISTS chk_column_position 
CHECK (position >= 0);

-- 看板列WIP限制约束（正数）
ALTER TABLE board_columns 
ADD CONSTRAINT IF NOT EXISTS chk_wip_limit 
CHECK (wip_limit IS NULL OR wip_limit > 0);

-- 添加触发器

-- 更新时间触发器 - Sprint
CREATE OR REPLACE FUNCTION update_sprint_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_sprints_updated_at
    BEFORE UPDATE ON sprints
    FOR EACH ROW
    EXECUTE FUNCTION update_sprint_updated_at();

-- 更新时间触发器 - 史诗
CREATE OR REPLACE FUNCTION update_epic_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_epics_updated_at
    BEFORE UPDATE ON epics
    FOR EACH ROW
    EXECUTE FUNCTION update_epic_updated_at();

-- 更新时间触发器 - 敏捷任务
CREATE OR REPLACE FUNCTION update_agile_task_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_agile_tasks_updated_at
    BEFORE UPDATE ON agile_tasks
    FOR EACH ROW
    EXECUTE FUNCTION update_agile_task_updated_at();

-- 任务编号自动生成触发器
CREATE OR REPLACE FUNCTION generate_task_number()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.task_number IS NULL OR NEW.task_number = 0 THEN
        SELECT COALESCE(MAX(task_number), 0) + 1
        INTO NEW.task_number
        FROM agile_tasks
        WHERE project_id = NEW.project_id
        AND deleted_at IS NULL;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_generate_task_number
    BEFORE INSERT ON agile_tasks
    FOR EACH ROW
    EXECUTE FUNCTION generate_task_number();

-- 工作日志更新任务已记录时间触发器
CREATE OR REPLACE FUNCTION update_task_logged_time()
RETURNS TRIGGER AS $$
BEGIN
    -- 插入或更新工作日志时，重新计算任务的已记录时间
    IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
        UPDATE agile_tasks
        SET logged_time = (
            SELECT COALESCE(SUM(time_spent), 0)
            FROM work_logs
            WHERE task_id = NEW.task_id
            AND deleted_at IS NULL
        )
        WHERE id = NEW.task_id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE agile_tasks
        SET logged_time = (
            SELECT COALESCE(SUM(time_spent), 0)
            FROM work_logs
            WHERE task_id = OLD.task_id
            AND deleted_at IS NULL
        )
        WHERE id = OLD.task_id;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_task_logged_time
    AFTER INSERT OR UPDATE OR DELETE ON work_logs
    FOR EACH ROW
    EXECUTE FUNCTION update_task_logged_time();

-- 创建视图

-- Sprint概览视图
CREATE OR REPLACE VIEW sprint_overview AS
SELECT 
    s.id,
    s.name,
    s.status,
    s.start_date,
    s.end_date,
    s.capacity,
    COUNT(at.id) as total_tasks,
    COUNT(CASE WHEN at.status = 'done' THEN 1 END) as completed_tasks,
    COUNT(CASE WHEN at.status = 'in_progress' THEN 1 END) as in_progress_tasks,
    COALESCE(SUM(at.story_points), 0) as total_story_points,
    COALESCE(SUM(CASE WHEN at.status = 'done' THEN at.story_points ELSE 0 END), 0) as completed_story_points,
    COALESCE(SUM(at.logged_time), 0) as total_logged_time,
    CASE 
        WHEN COUNT(at.id) > 0 THEN 
            ROUND((COUNT(CASE WHEN at.status = 'done' THEN 1 END) * 100.0 / COUNT(at.id)), 2)
        ELSE 0 
    END as completion_percentage
FROM sprints s
LEFT JOIN agile_tasks at ON s.id = at.sprint_id AND at.deleted_at IS NULL
WHERE s.deleted_at IS NULL
GROUP BY s.id;

-- 史诗概览视图
CREATE OR REPLACE VIEW epic_overview AS
SELECT 
    e.id,
    e.name,
    e.status,
    e.start_date,
    e.end_date,
    COUNT(at.id) as total_tasks,
    COUNT(CASE WHEN at.status = 'done' THEN 1 END) as completed_tasks,
    COALESCE(SUM(at.story_points), 0) as total_story_points,
    COALESCE(SUM(CASE WHEN at.status = 'done' THEN at.story_points ELSE 0 END), 0) as completed_story_points,
    CASE 
        WHEN COUNT(at.id) > 0 THEN 
            ROUND((COUNT(CASE WHEN at.status = 'done' THEN 1 END) * 100.0 / COUNT(at.id)), 2)
        ELSE 0 
    END as completion_percentage
FROM epics e
LEFT JOIN agile_tasks at ON e.id = at.epic_id AND at.deleted_at IS NULL
WHERE e.deleted_at IS NULL
GROUP BY e.id;

-- 任务统计视图
CREATE OR REPLACE VIEW task_statistics AS
SELECT 
    at.project_id,
    at.assignee_id,
    COUNT(*) as total_tasks,
    COUNT(CASE WHEN at.status = 'done' THEN 1 END) as completed_tasks,
    COUNT(CASE WHEN at.status = 'in_progress' THEN 1 END) as in_progress_tasks,
    COUNT(CASE WHEN at.type = 'bug' THEN 1 END) as bug_count,
    COALESCE(SUM(at.story_points), 0) as total_story_points,
    COALESCE(SUM(at.logged_time), 0) as total_logged_time,
    AVG(at.story_points) as avg_story_points
FROM agile_tasks at
WHERE at.deleted_at IS NULL
GROUP BY at.project_id, at.assignee_id;

-- 添加注释
COMMENT ON TABLE epics IS '史诗表，用于管理大型功能集合';
COMMENT ON TABLE sprints IS 'Sprint冲刺表，用于敏捷迭代管理';
COMMENT ON TABLE agile_tasks IS '敏捷任务表，用于管理用户故事、任务、缺陷等';
COMMENT ON TABLE task_comments IS '任务评论表';
COMMENT ON TABLE task_attachments IS '任务附件表';
COMMENT ON TABLE work_logs IS '工作日志表，用于时间追踪';
COMMENT ON TABLE boards IS '看板表，用于可视化任务管理';
COMMENT ON TABLE board_columns IS '看板列表，定义看板的工作流状态';

COMMENT ON COLUMN agile_tasks.rank IS 'Lexorank排序字段，用于拖拽排序';
COMMENT ON COLUMN agile_tasks.story_points IS '故事点，用于敏捷估算';
COMMENT ON COLUMN agile_tasks.acceptance_criteria IS '验收标准，JSON格式';
COMMENT ON COLUMN work_logs.time_spent IS '工作时长，单位：小时';
COMMENT ON COLUMN board_columns.wip_limit IS '在制品限制，看板列的最大任务数';

-- 创建函数

-- 获取Sprint燃尽图数据
CREATE OR REPLACE FUNCTION get_sprint_burndown_data(
    p_sprint_id UUID
)
RETURNS TABLE (
    date DATE,
    remaining_story_points INTEGER,
    ideal_remaining INTEGER
) AS $$
DECLARE
    sprint_record RECORD;
    total_points INTEGER;
    sprint_days INTEGER;
BEGIN
    -- 获取Sprint信息
    SELECT s.start_date, s.end_date
    INTO sprint_record
    FROM sprints s
    WHERE s.id = p_sprint_id;
    
    IF NOT FOUND THEN
        RETURN;
    END IF;
    
    -- 计算总故事点数
    SELECT COALESCE(SUM(story_points), 0)
    INTO total_points
    FROM agile_tasks
    WHERE sprint_id = p_sprint_id
    AND deleted_at IS NULL;
    
    -- 计算Sprint工作日数
    sprint_days := EXTRACT(DAY FROM sprint_record.end_date - sprint_record.start_date);
    
    -- 生成燃尽图数据（简化版，实际应该基于历史数据）
    FOR i IN 0..sprint_days LOOP
        RETURN QUERY
        SELECT 
            (sprint_record.start_date + i)::DATE,
            -- 这里应该查询历史数据，简化为线性递减
            GREATEST(0, total_points - (total_points * i / sprint_days))::INTEGER,
            GREATEST(0, total_points - (total_points * i / sprint_days))::INTEGER;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- 获取任务层次结构
CREATE OR REPLACE FUNCTION get_task_hierarchy(
    p_task_id UUID,
    p_depth INTEGER DEFAULT 10
)
RETURNS TABLE (
    id UUID,
    parent_id UUID,
    title TEXT,
    level INTEGER
) AS $$
WITH RECURSIVE task_tree AS (
    -- 根节点
    SELECT 
        t.id,
        t.parent_id,
        t.title,
        0 as level
    FROM agile_tasks t
    WHERE t.id = p_task_id
    AND t.deleted_at IS NULL
    
    UNION ALL
    
    -- 递归查找子任务
    SELECT 
        t.id,
        t.parent_id,
        t.title,
        tt.level + 1
    FROM agile_tasks t
    INNER JOIN task_tree tt ON t.parent_id = tt.id
    WHERE tt.level < p_depth
    AND t.deleted_at IS NULL
)
SELECT * FROM task_tree ORDER BY level, title;
$$ LANGUAGE sql;

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
    
    RAISE NOTICE '敏捷任务管理系统数据库迁移完成';
END
$$;