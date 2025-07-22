-- Cloud-Based Collaborative Development Platform
-- Seed Data for Development and Testing
-- Sample Data for Multi-Tenant Environment
-- Generated: 2025-01-19

-- =============================================================================
-- 开发环境种子数据
-- =============================================================================

-- 创建示例租户
INSERT INTO tenants (id, name, slug, subscription_plan_id, status) VALUES
(uuid_generate_v7(), 'ACME Corporation', 'acme-corp', 
 (SELECT id FROM subscription_plans WHERE name = 'Enterprise Plan'), 'active'),
(uuid_generate_v7(), 'TechStart Inc', 'techstart', 
 (SELECT id FROM subscription_plans WHERE name = 'Premium Plan'), 'active'),
(uuid_generate_v7(), 'DevTeam Studio', 'devteam', 
 (SELECT id FROM subscription_plans WHERE name = 'Standard Plan'), 'active'),
(uuid_generate_v7(), 'Freelancer Hub', 'freelancer', 
 (SELECT id FROM subscription_plans WHERE name = 'Free Plan'), 'active');

-- 获取租户ID（用于后续数据插入）
DO $$
DECLARE
    acme_tenant_id UUID;
    techstart_tenant_id UUID;
    devteam_tenant_id UUID;
    freelancer_tenant_id UUID;
    
    -- 用户ID变量
    admin_user_id UUID;
    dev_user_id UUID;
    pm_user_id UUID;
    designer_user_id UUID;
    tester_user_id UUID;
    
    -- 角色ID变量
    tenant_admin_role_id UUID;
    tenant_member_role_id UUID;
    project_admin_role_id UUID;
    project_developer_role_id UUID;
    project_viewer_role_id UUID;
    
    -- 项目ID变量
    web_project_id UUID;
    mobile_project_id UUID;
    api_project_id UUID;
    
    -- 状态ID变量
    todo_status_id UUID;
    in_progress_status_id UUID;
    done_status_id UUID;
    
    -- 仓库ID变量
    frontend_repo_id UUID;
    backend_repo_id UUID;
    mobile_repo_id UUID;
BEGIN
    -- 获取租户ID
    SELECT id INTO acme_tenant_id FROM tenants WHERE slug = 'acme-corp';
    SELECT id INTO techstart_tenant_id FROM tenants WHERE slug = 'techstart';
    SELECT id INTO devteam_tenant_id FROM tenants WHERE slug = 'devteam';
    SELECT id INTO freelancer_tenant_id FROM tenants WHERE slug = 'freelancer';

    -- =============================================================================
    -- 创建示例用户
    -- =============================================================================
    
    -- 创建平台管理员
    admin_user_id := uuid_generate_v7();
    INSERT INTO users (id, username, email, password_hash, display_name, status, is_platform_admin, email_verified) VALUES
    (admin_user_id, 'admin', 'admin@devcollab.com', '$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/jl.7bHcGZl1JQ8YQm', '平台管理员', 'active', true, true);
    
    -- 创建普通用户
    dev_user_id := uuid_generate_v7();
    INSERT INTO users (id, username, email, password_hash, display_name, status, email_verified) VALUES
    (dev_user_id, 'john_dev', 'john@acme.com', '$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/jl.7bHcGZl1JQ8YQm', 'John Smith (Developer)', 'active', true);
    
    pm_user_id := uuid_generate_v7();
    INSERT INTO users (id, username, email, password_hash, display_name, status, email_verified) VALUES
    (pm_user_id, 'sarah_pm', 'sarah@acme.com', '$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/jl.7bHcGZl1JQ8YQm', 'Sarah Johnson (PM)', 'active', true);
    
    designer_user_id := uuid_generate_v7();
    INSERT INTO users (id, username, email, password_hash, display_name, status, email_verified) VALUES
    (designer_user_id, 'mike_designer', 'mike@acme.com', '$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/jl.7bHcGZl1JQ8YQm', 'Mike Chen (Designer)', 'active', true);
    
    tester_user_id := uuid_generate_v7();
    INSERT INTO users (id, username, email, password_hash, display_name, status, email_verified) VALUES
    (tester_user_id, 'anna_tester', 'anna@acme.com', '$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/jl.7bHcGZl1JQ8YQm', 'Anna Wang (Tester)', 'active', true);

    -- =============================================================================
    -- 为ACME公司创建角色
    -- =============================================================================
    
    -- 租户级角色
    tenant_admin_role_id := uuid_generate_v7();
    INSERT INTO roles (id, tenant_id, name, scope, permissions, is_system) VALUES
    (tenant_admin_role_id, acme_tenant_id, 'Tenant Admin', 'tenant', 
     '["tenant.manage", "project.create", "user.invite", "billing.manage"]', true);
    
    tenant_member_role_id := uuid_generate_v7();
    INSERT INTO roles (id, tenant_id, name, scope, permissions, is_system) VALUES
    (tenant_member_role_id, acme_tenant_id, 'Tenant Member', 'tenant', 
     '["project.view", "project.join"]', true);
    
    -- 项目级角色
    project_admin_role_id := uuid_generate_v7();
    INSERT INTO roles (id, tenant_id, name, scope, permissions, is_system) VALUES
    (project_admin_role_id, acme_tenant_id, 'Project Admin', 'project', 
     '["project.manage", "member.invite", "task.create", "task.assign", "repository.create"]', true);
    
    project_developer_role_id := uuid_generate_v7();
    INSERT INTO roles (id, tenant_id, name, scope, permissions, is_system) VALUES
    (project_developer_role_id, acme_tenant_id, 'Developer', 'project', 
     '["task.view", "task.update", "repository.read", "repository.write", "pr.create"]', true);
    
    project_viewer_role_id := uuid_generate_v7();
    INSERT INTO roles (id, tenant_id, name, scope, permissions, is_system) VALUES
    (project_viewer_role_id, acme_tenant_id, 'Viewer', 'project', 
     '["task.view", "repository.read"]', true);

    -- =============================================================================
    -- 添加用户到租户
    -- =============================================================================
    
    INSERT INTO tenant_members (tenant_id, user_id, role_id, added_by) VALUES
    (acme_tenant_id, pm_user_id, tenant_admin_role_id, admin_user_id),
    (acme_tenant_id, dev_user_id, tenant_member_role_id, pm_user_id),
    (acme_tenant_id, designer_user_id, tenant_member_role_id, pm_user_id),
    (acme_tenant_id, tester_user_id, tenant_member_role_id, pm_user_id);

    -- =============================================================================
    -- 创建示例项目
    -- =============================================================================
    
    web_project_id := uuid_generate_v7();
    INSERT INTO projects (id, tenant_id, key, name, description, manager_id, status) VALUES
    (web_project_id, acme_tenant_id, 'WEB', 'E-commerce Website', 
     '企业级电商网站开发项目，包含前端界面、后端API和管理后台', pm_user_id, 'active');
    
    mobile_project_id := uuid_generate_v7();
    INSERT INTO projects (id, tenant_id, key, name, description, manager_id, status) VALUES
    (mobile_project_id, acme_tenant_id, 'MOBILE', 'Mobile App', 
     '配套移动应用开发，支持iOS和Android平台', pm_user_id, 'active');
    
    api_project_id := uuid_generate_v7();
    INSERT INTO projects (id, tenant_id, key, name, description, manager_id, status) VALUES
    (api_project_id, acme_tenant_id, 'API', 'Backend API Services', 
     '微服务架构的后端API开发，包含用户管理、订单处理、支付集成等', dev_user_id, 'active');

    -- =============================================================================
    -- 添加项目成员
    -- =============================================================================
    
    -- Web项目成员
    INSERT INTO project_members (project_id, user_id, role_id, added_by) VALUES
    (web_project_id, pm_user_id, project_admin_role_id, pm_user_id),
    (web_project_id, dev_user_id, project_developer_role_id, pm_user_id),
    (web_project_id, designer_user_id, project_developer_role_id, pm_user_id),
    (web_project_id, tester_user_id, project_viewer_role_id, pm_user_id);
    
    -- Mobile项目成员
    INSERT INTO project_members (project_id, user_id, role_id, added_by) VALUES
    (mobile_project_id, pm_user_id, project_admin_role_id, pm_user_id),
    (mobile_project_id, dev_user_id, project_developer_role_id, pm_user_id),
    (mobile_project_id, designer_user_id, project_developer_role_id, pm_user_id);
    
    -- API项目成员
    INSERT INTO project_members (project_id, user_id, role_id, added_by) VALUES
    (api_project_id, dev_user_id, project_admin_role_id, dev_user_id),
    (api_project_id, pm_user_id, project_viewer_role_id, dev_user_id);

    -- =============================================================================
    -- 创建任务状态
    -- =============================================================================
    
    todo_status_id := uuid_generate_v7();
    INSERT INTO task_statuses (id, tenant_id, name, category, display_order) VALUES
    (todo_status_id, acme_tenant_id, 'To Do', 'todo', 1);
    
    in_progress_status_id := uuid_generate_v7();
    INSERT INTO task_statuses (id, tenant_id, name, category, display_order) VALUES
    (in_progress_status_id, acme_tenant_id, 'In Progress', 'in_progress', 2);
    
    INSERT INTO task_statuses (id, tenant_id, name, category, display_order) VALUES
    (uuid_generate_v7(), acme_tenant_id, 'Code Review', 'in_progress', 3);
    
    done_status_id := uuid_generate_v7();
    INSERT INTO task_statuses (id, tenant_id, name, category, display_order) VALUES
    (done_status_id, acme_tenant_id, 'Done', 'done', 4);

    -- =============================================================================
    -- 创建示例任务
    -- =============================================================================
    
    -- Web项目任务
    INSERT INTO tasks (project_id, task_number, title, description, status_id, assignee_id, creator_id, priority) VALUES
    (web_project_id, 1, '设计首页界面', '设计电商网站的首页界面，包含产品展示、导航菜单、搜索功能等', todo_status_id, designer_user_id, pm_user_id, 'high'),
    (web_project_id, 2, '实现用户注册功能', '开发用户注册和登录功能，包含邮箱验证和密码加密', in_progress_status_id, dev_user_id, pm_user_id, 'high'),
    (web_project_id, 3, '商品列表页面开发', '实现商品分类浏览和筛选功能', todo_status_id, dev_user_id, pm_user_id, 'medium'),
    (web_project_id, 4, '购物车功能实现', '开发购物车添加、删除、修改数量等功能', todo_status_id, dev_user_id, pm_user_id, 'medium'),
    (web_project_id, 5, '支付集成测试', '集成第三方支付平台并进行功能测试', todo_status_id, tester_user_id, pm_user_id, 'high');
    
    -- Mobile项目任务
    INSERT INTO tasks (project_id, task_number, title, description, status_id, assignee_id, creator_id, priority) VALUES
    (mobile_project_id, 1, 'App架构设计', '设计移动应用的整体架构和技术栈选择', done_status_id, dev_user_id, pm_user_id, 'high'),
    (mobile_project_id, 2, '用户界面设计', '设计移动端的用户界面和交互流程', in_progress_status_id, designer_user_id, pm_user_id, 'high'),
    (mobile_project_id, 3, '原生功能开发', '实现相机、地理位置等原生功能', todo_status_id, dev_user_id, pm_user_id, 'medium');
    
    -- API项目任务
    INSERT INTO tasks (project_id, task_number, title, description, status_id, assignee_id, creator_id, priority) VALUES
    (api_project_id, 1, '用户管理API', '开发用户注册、登录、资料管理等API接口', in_progress_status_id, dev_user_id, dev_user_id, 'high'),
    (api_project_id, 2, '订单处理API', '实现订单创建、查询、状态更新等API', todo_status_id, dev_user_id, dev_user_id, 'high'),
    (api_project_id, 3, 'API文档编写', '编写完整的API接口文档和使用说明', todo_status_id, dev_user_id, dev_user_id, 'medium');

    -- =============================================================================
    -- 创建代码仓库
    -- =============================================================================
    
    frontend_repo_id := uuid_generate_v7();
    INSERT INTO repositories (id, project_id, name, description, visibility, default_branch) VALUES
    (frontend_repo_id, web_project_id, 'ecommerce-frontend', 'React前端应用代码仓库', 'private', 'main');
    
    backend_repo_id := uuid_generate_v7();
    INSERT INTO repositories (id, project_id, name, description, visibility, default_branch) VALUES
    (backend_repo_id, api_project_id, 'ecommerce-backend', 'Go语言后端API服务代码仓库', 'private', 'main');
    
    mobile_repo_id := uuid_generate_v7();
    INSERT INTO repositories (id, project_id, name, description, visibility, default_branch) VALUES
    (mobile_repo_id, mobile_project_id, 'ecommerce-mobile', 'React Native移动应用代码仓库', 'private', 'main');

    -- =============================================================================
    -- 创建Pull Request示例
    -- =============================================================================
    
    INSERT INTO pull_requests (repository_id, pr_number, title, source_branch, target_branch, status, creator_id) VALUES
    (frontend_repo_id, 1, 'feat: 添加用户注册页面', 'feature/user-registration', 'develop', 'open', dev_user_id),
    (frontend_repo_id, 2, 'fix: 修复购物车数量计算错误', 'bugfix/cart-quantity', 'main', 'merged', dev_user_id),
    (backend_repo_id, 1, 'feat: 实现JWT认证中间件', 'feature/jwt-auth', 'develop', 'open', dev_user_id);

    -- =============================================================================
    -- 创建CI/CD流水线
    -- =============================================================================
    
    INSERT INTO pipelines (repository_id, name, definition_file_path) VALUES
    (frontend_repo_id, 'Frontend CI/CD', '.github/workflows/frontend.yml'),
    (backend_repo_id, 'Backend CI/CD', '.github/workflows/backend.yml'),
    (mobile_repo_id, 'Mobile CI/CD', '.github/workflows/mobile.yml');

    -- =============================================================================
    -- 创建Runner
    -- =============================================================================
    
    INSERT INTO runners (tenant_id, name, tags, status) VALUES
    (acme_tenant_id, 'runner-web-01', '{"os": "linux", "arch": "amd64", "environment": "production"}', 'online'),
    (acme_tenant_id, 'runner-mobile-01', '{"os": "macos", "arch": "arm64", "environment": "mobile"}', 'online'),
    (acme_tenant_id, 'runner-test-01', '{"os": "linux", "arch": "amd64", "environment": "testing"}', 'idle');

    -- =============================================================================
    -- 创建文档
    -- =============================================================================
    
    INSERT INTO documents (project_id, title, content, creator_id) VALUES
    (web_project_id, '项目开发规范', 
     '# 项目开发规范\n\n## 代码风格\n- 使用TypeScript\n- 遵循ESLint规则\n- 使用Prettier格式化\n\n## Git工作流\n- 使用Git Flow\n- PR必须经过代码审查\n- 确保CI测试通过', 
     pm_user_id),
    (mobile_project_id, 'UI设计指南', 
     '# UI设计指南\n\n## 设计原则\n- 简洁直观\n- 一致性\n- 可访问性\n\n## 组件库\n使用Ant Design Mobile组件库', 
     designer_user_id),
    (api_project_id, 'API接口文档', 
     '# API接口文档\n\n## 认证\n使用JWT Token认证\n\n## 接口规范\n- RESTful设计\n- 统一错误码\n- JSON格式响应', 
     dev_user_id);

    -- =============================================================================
    -- 创建评论示例
    -- =============================================================================
    
    INSERT INTO comments (tenant_id, author_id, content, parent_entity_type, parent_entity_id) VALUES
    (acme_tenant_id, pm_user_id, '请确保界面符合品牌设计规范，特别是颜色和字体的使用。', 'task', 
     (SELECT id FROM tasks WHERE title = '设计首页界面')),
    (acme_tenant_id, dev_user_id, '用户注册功能已经完成基本实现，正在进行邮箱验证功能的测试。', 'task', 
     (SELECT id FROM tasks WHERE title = '实现用户注册功能')),
    (acme_tenant_id, tester_user_id, '发现一个bug：购物车数量为0时，仍然可以点击结算按钮。', 'task', 
     (SELECT id FROM tasks WHERE title = '购物车功能实现'));

    -- =============================================================================
    -- 创建通知示例
    -- =============================================================================
    
    INSERT INTO notifications (recipient_id, tenant_id, message, link) VALUES
    (dev_user_id, acme_tenant_id, '您被分配了新任务：商品列表页面开发', '/projects/WEB/tasks/3'),
    (designer_user_id, acme_tenant_id, '您的任务"用户界面设计"有新评论', '/projects/MOBILE/tasks/2'),
    (pm_user_id, acme_tenant_id, 'Pull Request #1 已创建，等待您的审查', '/projects/WEB/repositories/ecommerce-frontend/pull/1'),
    (tester_user_id, acme_tenant_id, '新的测试版本已发布，请进行功能验证', '/projects/WEB/releases/v1.2.0');

    -- =============================================================================
    -- 创建审计日志示例
    -- =============================================================================
    
    INSERT INTO audit_logs_partitioned (tenant_id, user_id, action, target_entity_type, target_entity_id, details, client_ip) VALUES
    (acme_tenant_id, pm_user_id, 'project.create', 'project', web_project_id, 
     '{"project_name": "E-commerce Website", "project_key": "WEB"}', '192.168.1.100'),
    (acme_tenant_id, dev_user_id, 'task.update', 'task', 
     (SELECT id FROM tasks WHERE title = '实现用户注册功能'),
     '{"field": "status", "old_value": "todo", "new_value": "in_progress"}', '192.168.1.101'),
    (acme_tenant_id, designer_user_id, 'task.assign', 'task', 
     (SELECT id FROM tasks WHERE title = '设计首页界面'),
     '{"assignee": "mike_designer"}', '192.168.1.102');

    RAISE NOTICE 'Seed data for ACME Corporation tenant created successfully';
    RAISE NOTICE 'Tenant ID: %', acme_tenant_id;
    RAISE NOTICE 'Sample users created: admin, john_dev, sarah_pm, mike_designer, anna_tester';
    RAISE NOTICE 'Sample projects: WEB (E-commerce Website), MOBILE (Mobile App), API (Backend API Services)';
END $$;