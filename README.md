# Cloud-Based Collaborative Development Platform

## 项目概述

基于企业级需求的智能开发协作平台，采用微服务架构、多租户SaaS模式，集成项目管理、代码仓库、CI/CD管道、知识库等核心功能。

## 技术架构

- **后端**: Go微服务 + Gin框架
- **数据库**: PostgreSQL 14+ (多租户RLS) + Redis
- **消息队列**: Apache Kafka
- **容器编排**: Kubernetes + Docker
- **前端**: React + TypeScript + Ant Design Pro
- **CI/CD**: Tekton Pipelines
- **监控**: Prometheus + Grafana + Jaeger

## 项目结构

```
.
├── cmd/                     # 主要应用程序入口点
├── internal/                # 私有库代码
├── pkg/                     # 公共库代码
├── services/                # 微服务实现
├── shared/                  # 跨服务共享代码
├── web/                     # 前端应用
├── deployments/             # 部署配置
├── infrastructure/          # 基础设施代码
├── docs/                    # 项目文档
├── scripts/                 # 构建和部署脚本
├── tests/                   # 集成测试和端到端测试
├── tools/                   # 工具和实用程序
└── configs/                 # 配置文件
```

## 7个核心微服务

1. **iam-service** - 身份认证中心
2. **tenant-service** - 租户管理与RBAC
3. **project-service** - 项目协作管理
4. **git-gateway-service** - Git协议代理
5. **cicd-service** - CI/CD管道
6. **notification-service** - 实时通知
7. **kb-service** - 知识库管理

## 快速开始

### 环境要求

- Go 1.21+
- PostgreSQL 14+
- Redis 6+
- Docker & Docker Compose
- Kubernetes (可选)

### 本地开发

```bash
# 克隆项目
git clone <repository-url>
cd cloud-collaborative-platform

# 启动基础设施
docker-compose up -d postgres redis kafka

# 安装依赖
go mod download

# 运行数据库迁移
make db-migrate

# 启动服务
make dev
```

## 开发规范

- 遵循 [企业内部软件开发规范](./企业内部软件开发规范.md)
- 使用 Git Flow 分支策略
- 代码必须通过静态检查和测试
- 所有API必须有OpenAPI文档

## 安全策略

本项目遵循零信任安全架构，详见 [公司信息安全政策](./公司信息安全政策.md)

## 许可证

内部项目，仅供学习使用。

## 联系方式

如有问题，请联系开发团队。