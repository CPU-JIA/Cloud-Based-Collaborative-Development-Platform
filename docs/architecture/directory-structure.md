# Monorepo 目录结构说明

## 概述

本项目采用标准的Go语言Monorepo结构，支持多个微服务的统一管理和部署。

## 目录结构

```
Cloud-Based Collaborative Development Platform/
├── README.md                          # 项目主文档
├── Makefile                           # 构建和部署脚本
├── go.mod                             # Go模块定义
├── go.sum                             # Go模块依赖锁定
├── .gitignore                         # Git忽略文件
├── .env.template                      # 环境变量模板
│
├── cmd/                               # 应用程序入口点
│   ├── iam-service/                   # 身份认证服务
│   │   └── main.go
│   ├── tenant-service/                # 租户管理服务
│   │   └── main.go
│   ├── project-service/               # 项目管理服务
│   │   └── main.go
│   ├── git-gateway-service/           # Git代理服务
│   │   └── main.go
│   ├── cicd-service/                  # CI/CD服务
│   │   └── main.go
│   ├── notification-service/          # 通知服务
│   │   └── main.go
│   └── kb-service/                    # 知识库服务
│       └── main.go
│
├── internal/                          # 私有库代码 (不可被外部导入)
│   ├── iam/                          # IAM服务内部逻辑
│   │   ├── handler/                  # HTTP处理器
│   │   ├── service/                  # 业务逻辑层
│   │   ├── repository/               # 数据访问层
│   │   └── model/                    # 数据模型
│   ├── tenant/                       # 租户服务内部逻辑
│   ├── project/                      # 项目服务内部逻辑
│   ├── git-gateway/                  # Git网关内部逻辑
│   ├── cicd/                         # CI/CD服务内部逻辑
│   ├── notification/                 # 通知服务内部逻辑
│   └── kb/                           # 知识库服务内部逻辑
│
├── pkg/                              # 公共库代码 (可被外部导入)
│   ├── auth/                         # 认证相关工具
│   ├── errors/                       # 错误处理
│   ├── utils/                        # 通用工具函数
│   ├── validators/                   # 数据验证
│   └── constants/                    # 常量定义
│
├── shared/                           # 跨服务共享代码
│   ├── config/                       # 配置管理
│   │   └── config.go
│   ├── logger/                       # 日志管理
│   │   └── logger.go
│   ├── database/                     # 数据库连接管理
│   │   ├── postgres.go
│   │   └── redis.go
│   ├── middleware/                   # HTTP中间件
│   │   └── middleware.go
│   ├── auth/                         # 认证相关共享代码
│   ├── types/                        # 共享数据类型
│   ├── utils/                        # 共享工具函数
│   ├── grpc/                         # gRPC相关代码
│   └── events/                       # 事件处理
│
├── api/                              # API定义和文档
│   ├── iam/                          # IAM服务API
│   │   ├── v1/                       # API版本1
│   │   │   ├── iam.proto            # Protobuf定义
│   │   │   └── iam.pb.go            # 生成的Go代码
│   │   └── docs/                     # API文档
│   ├── tenant/                       # 租户管理API
│   ├── project/                      # 项目管理API
│   ├── git-gateway/                  # Git网关API
│   ├── cicd/                         # CI/CD API
│   ├── notification/                 # 通知API
│   └── kb/                           # 知识库API
│
├── web/                              # 前端应用
│   ├── src/                          # 源代码
│   ├── public/                       # 静态资源
│   ├── package.json                  # 依赖定义
│   └── README.md                     # 前端文档
│
├── configs/                          # 配置文件
│   ├── config.yaml                   # 主配置文件
│   ├── development.yaml              # 开发环境配置
│   ├── staging.yaml                  # 预发布环境配置
│   └── production.yaml               # 生产环境配置
│
├── deployments/                      # 部署配置
│   ├── docker/                       # Docker相关
│   │   ├── Dockerfile.iam-service   # IAM服务Docker文件
│   │   ├── docker-compose.dev.yml   # 开发环境Docker Compose
│   │   └── docker-compose.prod.yml  # 生产环境Docker Compose
│   ├── kubernetes/                   # Kubernetes配置
│   │   ├── namespace.yaml
│   │   ├── configmap.yaml
│   │   ├── secret.yaml
│   │   ├── services/                 # 各服务的K8s配置
│   │   └── ingress.yaml
│   ├── terraform/                    # Terraform基础设施代码
│   └── helm/                         # Helm Charts
│
├── infrastructure/                   # 基础设施代码
│   ├── database/                     # 数据库相关
│   │   ├── migrations/               # 数据库迁移文件
│   │   └── seeds/                    # 种子数据
│   ├── docker/                       # Docker配置
│   ├── kubernetes/                   # K8s配置
│   └── monitoring/                   # 监控配置
│       ├── prometheus/
│       ├── grafana/
│       └── jaeger/
│
├── scripts/                          # 脚本文件
│   ├── build.sh                      # 构建脚本
│   ├── deploy.sh                     # 部署脚本
│   ├── test.sh                       # 测试脚本
│   └── migrate.sh                    # 数据库迁移脚本
│
├── tests/                            # 测试代码
│   ├── unit/                         # 单元测试
│   ├── integration/                  # 集成测试
│   ├── e2e/                          # 端到端测试
│   ├── performance/                  # 性能测试
│   ├── security/                     # 安全测试
│   └── mocks/                        # Mock文件
│
├── tools/                            # 工具和实用程序
│   ├── codegen/                      # 代码生成工具
│   ├── migration/                    # 数据迁移工具
│   ├── deployment/                   # 部署工具
│   └── testing/                      # 测试工具
│
├── docs/                             # 项目文档
│   ├── architecture/                 # 架构文档
│   ├── api/                          # API文档
│   ├── deployment/                   # 部署文档
│   ├── development/                  # 开发文档
│   └── user/                         # 用户文档
│
└── database/                         # 数据库相关文件 (已存在)
    ├── migrations/                   # 迁移文件
    ├── scripts/                      # 数据库脚本
    └── config/                       # 数据库配置
```

## 目录说明

### 核心目录

- **cmd/**: 应用程序的入口点，每个微服务都有自己的main.go文件
- **internal/**: 私有库代码，不能被外部包导入，包含各服务的核心业务逻辑
- **pkg/**: 公共库代码，可以被外部包导入和使用
- **shared/**: 跨服务共享的代码和组件

### API和接口

- **api/**: API定义文件，包括Protobuf、OpenAPI规范和生成的代码
- **web/**: 前端应用代码

### 配置和部署

- **configs/**: 应用程序配置文件
- **deployments/**: 部署相关的配置文件（Docker、Kubernetes等）
- **infrastructure/**: 基础设施即代码

### 开发和维护

- **scripts/**: 构建、部署、测试等脚本
- **tests/**: 各种类型的测试代码
- **tools/**: 开发和部署工具
- **docs/**: 项目文档

## 命名约定

### 文件命名
- 使用小写字母和连字符：`user-service.go`
- 测试文件：`user_service_test.go`
- 接口文件：`repository.go`
- 实现文件：`postgres_repository.go`

### 包命名
- 使用小写字母，避免下划线
- 包名应该简短且有意义
- 避免使用通用名称如`util`、`common`

### 目录命名
- 使用小写字母和连字符
- 服务目录：`iam-service`
- 功能目录：`user-management`

## 依赖管理

### 内部依赖
- `cmd/` 可以导入 `internal/`、`pkg/`、`shared/`
- `internal/` 只能导入 `pkg/`、`shared/`
- `pkg/` 只能导入 `shared/`
- `shared/` 不应导入其他内部包

### 外部依赖
- 所有外部依赖都在根目录的 `go.mod` 中管理
- 使用 `go mod tidy` 保持依赖清洁

## 构建和部署

### 本地开发
```bash
# 启动开发环境
make dev

# 运行测试
make test

# 构建所有服务
make build
```

### Docker部署
```bash
# 构建Docker镜像
make docker-build

# 启动Docker环境
make docker-dev
```

### Kubernetes部署
```bash
# 部署到Kubernetes
make k8s-deploy
```

## 最佳实践

1. **依赖方向**: 高层模块不应依赖低层模块，都应依赖抽象
2. **接口分离**: 定义小而专注的接口
3. **单一职责**: 每个包应该有明确的职责
4. **避免循环依赖**: 使用依赖注入和接口来避免循环依赖
5. **错误处理**: 统一的错误处理机制
6. **日志记录**: 结构化日志，包含足够的上下文信息
7. **测试**: 每个包都应该有对应的测试
8. **文档**: 重要的包和函数都应该有文档注释