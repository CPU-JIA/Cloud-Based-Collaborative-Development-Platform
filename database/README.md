# 数据库迁移指南

Cloud-Based Collaborative Development Platform 数据库迁移文档

## 概述

本目录包含数据库迁移脚本和工具，支持IAM服务的完整数据库架构部署和管理。

## 目录结构

```
database/
├── README.md                           # 本文档
├── config/                            # 数据库配置
│   └── database.yml                   # 数据库连接配置
├── migrations/                        # 迁移文件
│   ├── 001_initial_schema.sql         # 初始数据库架构
│   ├── 002_row_level_security.sql     # 行级安全策略
│   ├── 003_partitioning.sql           # 分区表设置
│   ├── 004_seed_data.sql              # 基础数据
│   ├── 005_iam_service_schema.sql     # IAM服务架构 ⭐️
│   └── 006_iam_default_data.sql       # IAM默认数据 ⭐️
└── scripts/                           # 执行脚本
    ├── init_database.sh               # 数据库初始化
    ├── run_iam_migrations.sh          # IAM迁移执行 ⭐️
    ├── verify_iam_migration.sh        # IAM迁移验证 ⭐️
    ├── backup_database.sh             # 数据库备份
    └── restore_database.sh            # 数据库恢复
```

## IAM服务数据库架构

### 核心表结构

| 表名 | 说明 | 关键特性 |
|-----|------|---------|
| `users` | 用户表 | 支持多租户、密码安全、账户锁定 |
| `roles` | 角色表 | RBAC权限模型、租户隔离 |
| `permissions` | 权限表 | 资源-动作权限定义 |
| `user_roles` | 用户角色关联 | 多对多关系、租户隔离 |
| `role_permissions` | 角色权限关联 | 多对多关系、权限继承 |
| `user_sessions` | 用户会话 | JWT令牌管理、会话跟踪 |

### 安全特性

- **行级安全 (RLS)**: 多租户数据隔离
- **密码安全**: bcrypt加密、强度验证
- **账户保护**: 登录失败锁定、自动解锁
- **会话管理**: JWT令牌对、过期清理
- **审计追踪**: 操作日志、状态变更

## 快速开始

### 1. 环境准备

确保已安装 PostgreSQL 客户端：

```bash
# macOS
brew install postgresql

# Ubuntu/Debian
sudo apt-get install postgresql-client

# CentOS/RHEL
sudo yum install postgresql
```

### 2. 配置数据库连接

设置环境变量或修改配置文件：

```bash
# 环境变量方式
export POSTGRES_HOST="localhost"
export POSTGRES_PORT="5432"
export POSTGRES_DB="collaborative_platform"
export POSTGRES_USER="postgres"
export POSTGRES_PASSWORD="your_password"

# 或者修改 config/database.yml
```

### 3. 执行IAM迁移

```bash
# 使用 Makefile (推荐)
make db-migrate-iam

# 或直接执行脚本
cd database
./scripts/run_iam_migrations.sh
```

### 4. 验证迁移结果

```bash
# 验证迁移
make db-verify-iam

# 或直接执行验证脚本
cd database
./scripts/verify_iam_migration.sh
```

## 详细使用说明

### IAM迁移脚本

`run_iam_migrations.sh` 脚本会执行以下操作：

1. **环境检查**: 验证工具和数据库连接
2. **前置条件**: 确保UUID扩展和必要函数存在
3. **架构迁移**: 创建IAM服务所需的表结构
4. **数据迁移**: 插入默认的租户、角色、权限和测试用户
5. **验证检查**: 确认迁移成功完成

#### 命令行选项

```bash
./scripts/run_iam_migrations.sh [options]

选项:
  --migration-dir DIR    指定迁移文件目录 (默认: ../migrations)
  --force               强制执行，忽略警告
  --help                显示帮助信息
```

### 验证脚本

`verify_iam_migration.sh` 会执行全面的验证检查：

- ✅ 表结构验证
- ✅ 索引创建检查
- ✅ 外键约束验证
- ✅ RLS策略检查
- ✅ 默认数据验证
- ✅ 函数存在性检查
- ✅ 数据一致性检查
- ✅ 基本功能测试

## 默认数据

### 测试租户
- **ID**: `550e8400-e29b-41d4-a716-446655440000`
- **名称**: Test Tenant
- **标识**: test-tenant

### 系统角色

| 角色 | 说明 | 权限范围 |
|-----|------|---------|
| `admin` | 租户管理员 | 所有权限 |
| `manager` | 项目管理员 | 项目、用户、代码仓库、CI/CD管理 |
| `developer` | 开发者 | 代码读写、CI/CD操作、项目查看 |
| `viewer` | 查看者 | 所有资源的只读权限 |
| `user` | 普通用户 | 基础权限（查看自己信息和项目） |

### 测试用户

| 用户 | 邮箱 | 密码 | 角色 |
|-----|------|------|-----|
| Admin User | admin@test.com | admin123 | admin |
| Test User | user@test.com | user123 | user |

### 权限体系

权限采用 `资源.动作` 的命名格式：

- **用户管理**: `user.read`, `user.write`, `user.delete`, `user.manage`
- **角色管理**: `role.read`, `role.write`, `role.delete`, `role.manage`
- **项目管理**: `project.read`, `project.write`, `project.delete`, `project.manage`
- **代码仓库**: `repository.read`, `repository.write`, `repository.delete`, `repository.manage`
- **CI/CD**: `cicd.read`, `cicd.write`, `cicd.manage`
- **系统管理**: `system.manage`, `tenant.manage`

## 故障排除

### 常见问题

#### 1. 数据库连接失败

```bash
ERROR: 无法连接到数据库，请检查连接参数
```

**解决方案**:
- 检查PostgreSQL服务是否运行
- 验证连接参数（主机、端口、用户名、密码）
- 确认数据库存在
- 检查网络连接和防火墙设置

#### 2. UUID扩展安装失败

```bash
ERROR: 无法安装UUID扩展，请检查数据库权限
```

**解决方案**:
- 确保用户有SUPERUSER权限，或者
- 预先以超级用户身份安装扩展：
```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
```

#### 3. 表已存在冲突

```bash
ERROR: relation "users" already exists
```

**解决方案**:
- 迁移脚本会智能处理现有表，仅添加缺失字段
- 如需完全重建，先备份数据再删除表
- 或使用 `--force` 选项跳过警告

#### 4. RLS策略创建失败

```bash
ERROR: policy "users_tenant_isolation" already exists
```

**解决方案**:
- 脚本会先删除现有策略再重新创建
- 如果手动创建过策略，可能需要手动清理

### 调试方法

#### 1. 启用详细日志

```bash
# 设置详细日志级别
export PGCLIENTENCODING=UTF8
export PGOPTIONS="--client-min-messages=debug1"
```

#### 2. 检查迁移日志

```bash
# 查看迁移日志
ls -la /tmp/migration_*.log
cat /tmp/migration_iam_service_schema.log
```

#### 3. 手动验证表结构

```sql
-- 检查表是否存在
SELECT table_name FROM information_schema.tables 
WHERE table_name IN ('users', 'roles', 'permissions', 'user_roles', 'role_permissions', 'user_sessions');

-- 检查表结构
\d users
\d roles
\d permissions

-- 检查索引
SELECT indexname FROM pg_indexes WHERE tablename = 'users';

-- 检查RLS状态
SELECT schemaname, tablename, rowsecurity 
FROM pg_tables 
WHERE tablename LIKE '%users%' OR tablename LIKE '%roles%';
```

#### 4. 数据一致性检查

```sql
-- 运行一致性检查函数
SELECT * FROM check_iam_data_consistency();

-- 检查孤立记录
SELECT COUNT(*) FROM user_roles ur 
LEFT JOIN users u ON ur.user_id = u.id 
WHERE u.id IS NULL;
```

## 维护操作

### 定期清理

```bash
# 清理过期会话和解锁用户
psql -c "SELECT cleanup_expired_sessions();"
psql -c "SELECT unlock_expired_users();"
```

### 数据备份

```bash
# 使用提供的备份脚本
./scripts/backup_database.sh

# 或手动备份
pg_dump -h localhost -U postgres collaborative_platform > backup.sql
```

### 性能监控

```sql
-- 检查表大小
SELECT schemaname, tablename, pg_total_relation_size(schemaname||'.'||tablename) as size
FROM pg_tables 
WHERE schemaname = 'public' 
ORDER BY size DESC;

-- 检查索引使用情况
SELECT schemaname, tablename, attname, n_distinct, correlation
FROM pg_stats 
WHERE tablename IN ('users', 'roles', 'permissions');
```

## 参考资料

- [PostgreSQL Row Level Security](https://www.postgresql.org/docs/current/ddl-rowsecurity.html)
- [RBAC权限模型最佳实践](https://en.wikipedia.org/wiki/Role-based_access_control)
- [多租户数据库设计模式](https://docs.microsoft.com/en-us/azure/sql-database/saas-tenancy-app-design-patterns)
- [JWT令牌安全最佳实践](https://tools.ietf.org/html/rfc7519)

## 贡献

如有问题或改进建议，请：

1. 查看现有的Issue和PR
2. 创建详细的Bug报告或功能请求
3. 提交代码前运行所有测试
4. 遵循现有的代码风格和约定

---

📝 **注意**: 这是企业级IAM服务的核心数据库架构，请确保在生产环境部署前进行充分测试。