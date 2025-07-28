# Cloud-Based Collaborative Development Platform

[![Build Status](https://github.com/your-org/cloud-platform/workflows/CI/badge.svg)](https://github.com/your-org/cloud-platform/actions)
[![Test Coverage](https://codecov.io/gh/your-org/cloud-platform/branch/main/graph/badge.svg)](https://codecov.io/gh/your-org/cloud-platform)
[![Go Version](https://img.shields.io/badge/Go-1.23+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## 🚀 项目概述

基于企业级需求的智能开发协作平台，采用微服务架构、多租户SaaS模式，集成项目管理、代码仓库、CI/CD管道、知识库等核心功能。

### ✨ 核心特性

- 🏗️ **微服务架构**: 9个独立微服务，支持水平扩展
- 🔐 **企业级安全**: JWT认证、RBAC权限、数据加密
- 📊 **实时协作**: WebSocket实时通信、在线编辑
- 🔄 **CI/CD集成**: GitHub Actions、自动化部署
- 📈 **监控告警**: Prometheus + Grafana监控体系
- 🧪 **测试覆盖**: 80%+测试覆盖率，620+测试用例

## 🛠️ 技术架构

### 后端技术栈
- **语言**: Go 1.23+
- **框架**: Gin Web框架
- **数据库**: PostgreSQL 15 + Redis 7
- **消息队列**: NATS
- **认证**: JWT + OAuth2
- **ORM**: GORM v2

### 前端技术栈
- **框架**: React 18 + TypeScript
- **状态管理**: Redux Toolkit + RTK Query
- **UI组件**: Material-UI (MUI)
- **构建工具**: Vite
- **通信**: Socket.IO + Axios

### DevOps技术栈
- **容器化**: Docker + Docker Compose
- **编排**: Kubernetes + Helm
- **CI/CD**: GitHub Actions
- **监控**: Prometheus + Grafana
- **日志**: ELK Stack

## 📁 项目结构

```
cloud-collaborative-platform/
├── 📂 cmd/                          # 服务入口点
│   ├── project-service/             # 项目管理服务
│   ├── iam-service/                 # 身份认证服务
│   ├── git-gateway-service/         # Git网关服务
│   └── ...                          # 其他7个微服务
├── 📂 internal/                     # 内部业务逻辑
│   ├── project/                     # 项目管理模块
│   ├── auth/                        # 认证模块
│   └── ...                          # 其他业务模块
├── 📂 shared/                       # 共享组件
│   ├── auth/                        # 认证中间件
│   ├── config/                      # 配置管理
│   ├── database/                    # 数据库连接
│   └── middleware/                  # HTTP中间件
├── 📂 frontend/                     # React前端应用
│   ├── src/components/              # UI组件
│   ├── src/pages/                   # 页面组件
│   ├── src/store/                   # Redux状态管理
│   └── src/api/                     # API客户端
├── 📂 test/                         # 测试套件
│   ├── unit/                        # 单元测试 (620个测试用例)
│   ├── integration/                 # 集成测试
│   └── common/                      # 测试工具
├── 📂 docs/                         # 项目文档
│   ├── api-documentation.md         # API文档
│   ├── deployment-guide.md          # 部署指南
│   ├── developer-guide.md           # 开发者指南
│   └── ci-cd-guide.md              # CI/CD指南
├── 📂 deployments/                  # 部署配置
│   ├── helm/                        # Helm Charts
│   └── k8s/                         # Kubernetes配置
├── 📂 scripts/                      # 构建脚本
├── 📂 .github/workflows/            # GitHub Actions
└── 📄 Makefile                      # 构建命令
```

## 🎯 核心微服务

| 服务 | 端口 | 功能描述 | 测试用例 |
|------|------|----------|----------|
| **Project Service** | 8080 | 项目管理、文件操作 | 15个 |
| **IAM Service** | 8081 | 用户认证、权限管理 | 94个 |
| **Git Gateway** | 8082 | Git操作代理 | 96个 |
| **Tenant Service** | 8083 | 多租户管理 | 28个 |
| **Notification Service** | 8084 | 实时通知 | 36个 |
| **CI/CD Service** | 8085 | 持续集成部署 | 68个 |
| **File Service** | 8086 | 文件存储管理 | 96个 |
| **Team Service** | 8087 | 团队协作 | 92个 |
| **Knowledge Base** | 8088 | 知识库管理 | 95个 |

## ⚡ 快速开始

### 📋 环境要求

- **Go**: 1.23+
- **Node.js**: 18+
- **PostgreSQL**: 15+
- **Redis**: 7+
- **Docker**: 20.10+
- **Docker Compose**: 2.0+

### 🚀 本地开发

```bash
# 1. 克隆项目
git clone https://github.com/your-org/cloud-collaborative-platform.git
cd cloud-collaborative-platform

# 2. 环境配置
cp .env.example .env
# 编辑 .env 文件配置数据库连接等

# 3. 启动基础设施
docker-compose up -d postgres redis

# 4. 安装依赖
go mod download
cd frontend && npm install && cd ..

# 5. 数据库迁移
make migrate-up

# 6. 启动所有服务
make dev
```

### 🌐 访问地址

- **前端应用**: http://localhost:3000
- **API网关**: http://localhost:8080
- **API文档**: http://localhost:8080/swagger/index.html
- **监控面板**: http://localhost:9090 (Prometheus)

## 📚 文档导航

| 文档 | 描述 |
|------|------|
| [🚀 部署指南](docs/deployment-guide.md) | 完整的部署指南，支持Docker、Kubernetes |
| [📖 API文档](docs/api-documentation.md) | RESTful API接口文档 |
| [👨‍💻 开发者指南](docs/developer-guide.md) | 开发环境搭建、编码规范 |
| [🔄 CI/CD指南](docs/ci-cd-guide.md) | GitHub Actions工作流程 |
| [🔐 安全文档](docs/security/) | 安全配置和最佳实践 |

## 🧪 测试与质量

### 测试覆盖率

```bash
# 运行所有测试
make test

# 生成覆盖率报告
make test-coverage

# 运行特定服务测试
make test-service SERVICE=project-service
```

**当前测试状态:**
- ✅ **总测试用例**: 620个
- ✅ **测试覆盖率**: 80%+
- ✅ **服务覆盖**: 9/9 (100%)
- ✅ **测试执行时间**: 4.67秒

### 质量检查

```bash
# 代码静态检查
make lint

# 安全扫描
make security-scan

# 性能测试
make benchmark
```

## 🚀 部署方式

### Docker Compose (开发/测试)

```bash
# 开发环境
docker-compose up -d

# 生产环境
docker-compose -f docker-compose.prod.yml up -d
```

### Kubernetes (生产)

```bash
# 使用Helm部署
helm install cloud-platform ./deployments/helm/cloud-platform \
  --namespace cloud-platform \
  --create-namespace \
  --values deployments/helm/cloud-platform/values-production.yaml
```

## 📊 监控与日志

### 监控指标

- **应用指标**: QPS、响应时间、错误率
- **系统指标**: CPU、内存、磁盘、网络
- **业务指标**: 用户活跃度、项目数量
- **自定义指标**: 通过 `/metrics` 端点暴露

### 日志管理

```bash
# 查看服务日志
docker-compose logs -f project-service

# Kubernetes环境
kubectl logs -f deployment/project-service -n cloud-platform
```

## 🛡️ 安全特性

- ✅ **JWT认证**: 基于Token的无状态认证
- ✅ **RBAC权限**: 细粒度角色权限控制
- ✅ **数据加密**: 敏感数据AES-256加密
- ✅ **CSRF防护**: 跨站请求伪造防护
- ✅ **SQL注入防护**: 参数化查询+输入验证
- ✅ **XSS防护**: 输入清理+CSP策略
- ✅ **API限流**: 防止API滥用
- ✅ **安全headers**: 完整的HTTP安全头配置

## 🤝 贡献指南

### 提交代码

1. Fork项目到个人仓库
2. 创建特性分支: `git checkout -b feature/amazing-feature`
3. 提交更改: `git commit -m 'feat: add amazing feature'`
4. 推送分支: `git push origin feature/amazing-feature`
5. 创建Pull Request

### 提交信息规范

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**类型说明:**
- `feat`: 新功能
- `fix`: 错误修复
- `docs`: 文档更新
- `style`: 代码格式
- `refactor`: 代码重构
- `test`: 测试相关
- `chore`: 构建工具

### 代码审查

- ✅ 所有测试必须通过
- ✅ 代码覆盖率不得降低
- ✅ 通过静态代码检查
- ✅ 至少一名团队成员审查

## 📈 项目状态

### 开发进度

- ✅ **Phase 1**: 项目架构设计 (100%)
- ✅ **Phase 2A**: 核心服务开发 (100%)
- ✅ **Phase 2B**: 基础设施服务 (100%)
- ✅ **Phase 2C**: 应用服务开发 (100%)
- ✅ **Phase 3**: 测试覆盖完善 (100%)
- ✅ **Phase 4**: CI/CD流程建立 (100%)
- 🔄 **Phase 5**: 性能优化 (进行中)

### 发布计划

- **v1.0.0**: 核心功能发布 ✅
- **v1.1.0**: 性能优化版本 (2025-Q2)
- **v1.2.0**: 高级协作功能 (2025-Q3)
- **v2.0.0**: 企业版功能 (2025-Q4)

## 📄 许可证

本项目采用 [MIT License](LICENSE) 开源协议。

## 📞 联系我们

- **项目主页**: https://github.com/your-org/cloud-platform
- **问题反馈**: https://github.com/your-org/cloud-platform/issues
- **技术支持**: support@yourplatform.com
- **文档站点**: https://docs.yourplatform.com

## 🙏 致谢

感谢所有贡献者和开源社区的支持！

---

<div align="center">
  <img src="https://img.shields.io/badge/Made%20with-💖-red.svg" alt="Made with Love">
  <img src="https://img.shields.io/badge/Built%20with-Go%20&%20React-blue.svg" alt="Built with Go & React">
</div>

*最后更新: 2025-07-27 | 版本: v1.0.0*