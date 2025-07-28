# CI/CD 指南

## 概述

本项目使用 GitHub Actions 实现完整的 CI/CD 流程，包括代码质量检查、安全扫描、自动化测试、构建和部署。

## CI/CD 架构

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   开发者    │────▶│   GitHub    │────▶│  GitHub     │
│   提交代码  │     │   仓库      │     │  Actions    │
└─────────────┘     └─────────────┘     └─────────────┘
                                               │
                    ┌──────────────────────────┼──────────────────────────┐
                    │                          │                          │
                    ▼                          ▼                          ▼
            ┌─────────────┐           ┌─────────────┐            ┌─────────────┐
            │  代码检查   │           │  安全扫描   │            │  自动测试   │
            │  Linting    │           │  Security   │            │   Tests     │
            └─────────────┘           └─────────────┘            └─────────────┘
                    │                          │                          │
                    └──────────────────────────┼──────────────────────────┘
                                               │
                                               ▼
                                        ┌─────────────┐
                                        │  构建镜像   │
                                        │   Docker    │
                                        └─────────────┘
                                               │
                           ┌───────────────────┼───────────────────┐
                           │                   │                   │
                           ▼                   ▼                   ▼
                    ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
                    │   开发环境  │     │  预发布环境 │     │  生产环境  │
                    │    (Dev)    │     │  (Staging)  │     │    (Prod)   │
                    └─────────────┘     └─────────────┘     └─────────────┘
```

## 工作流程

### 1. 持续集成 (CI)

#### 代码提交触发
- **触发条件**: Push 到 main、develop 或 feature/* 分支
- **Pull Request**: 向 main 或 develop 分支提交 PR

#### CI 流程步骤

1. **代码检查 (Linting)**
   ```yaml
   - Go 代码: golangci-lint
   - JavaScript/TypeScript: ESLint
   - Dockerfile: Hadolint
   ```

2. **安全扫描**
   ```yaml
   - 代码安全: Gosec, CodeQL
   - 依赖扫描: Trivy, Snyk
   - 密钥扫描: Gitleaks, TruffleHog
   - 容器扫描: Trivy, Grype
   ```

3. **自动化测试**
   ```yaml
   - 单元测试: go test
   - 集成测试: 带数据库和 Redis
   - 前端测试: Vitest
   - 覆盖率: Codecov
   ```

4. **构建验证**
   ```yaml
   - Go 二进制构建
   - Docker 镜像构建
   - 前端资源构建
   ```

### 2. 持续部署 (CD)

#### 部署策略

- **开发环境**: 自动部署 (develop 分支)
- **预发布环境**: 自动部署 (develop 分支)
- **生产环境**: 手动批准后部署 (main 分支)

#### 部署方式

1. **蓝绿部署**
   - 零停机时间
   - 快速回滚
   - 流量逐步切换

2. **金丝雀发布**
   - 10% → 50% → 100% 流量切换
   - 监控指标验证
   - 自动回滚机制

## 配置说明

### 1. GitHub Actions 配置

#### 必需的 Secrets

```yaml
# Docker Hub
DOCKER_USERNAME: Docker Hub 用户名
DOCKER_PASSWORD: Docker Hub 密码

# Kubernetes
STAGING_KUBECONFIG: 预发布环境 kubeconfig (base64)
PRODUCTION_KUBECONFIG: 生产环境 kubeconfig (base64)

# 代码质量
SONAR_TOKEN: SonarCloud token
SONAR_ORGANIZATION: SonarCloud 组织
SONAR_PROJECT_KEY: SonarCloud 项目 key

# 安全扫描
SNYK_TOKEN: Snyk API token

# AI 代码审查
OPENAI_API_KEY: OpenAI API key (可选)
```

#### 环境变量

```yaml
GO_VERSION: '1.23'
NODE_VERSION: '20'
POSTGRES_VERSION: '15'
REDIS_VERSION: '7'
```

### 2. 本地开发配置

#### Pre-commit Hooks

```bash
# 安装 pre-commit
pip install pre-commit

# 设置 hooks
pre-commit install

# 手动运行
pre-commit run --all-files
```

#### 本地 CI 验证

```bash
# 运行 linter
golangci-lint run

# 运行测试
go test -v ./...

# 构建 Docker 镜像
docker-compose build

# 运行安全扫描
./scripts/check-secrets.sh
```

## 部署流程

### 1. 开发环境部署

```bash
# 自动触发: git push origin develop
# 手动部署:
kubectl apply -f k8s/ -n development
```

### 2. 预发布环境部署

```bash
# 自动触发: git push origin develop
# 手动部署:
./scripts/deploy-staging.sh
```

### 3. 生产环境部署

```bash
# 创建发布标签
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0

# 蓝绿部署
./scripts/blue-green-deploy.sh production v1.0.0

# 健康检查
./scripts/health-checks.sh production

# 切换流量
./nginx/traffic-switch.sh production
```

## 监控和告警

### 1. 部署监控

- **GitHub Actions**: 实时查看工作流状态
- **Kubernetes Dashboard**: 查看部署状态
- **Grafana**: 应用性能监控

### 2. 告警配置

```yaml
# 部署失败
- Slack 通知
- 邮件通知
- PagerDuty (P0 问题)

# 健康检查失败
- 自动回滚
- 告警通知
- 事件记录
```

## 回滚策略

### 1. 自动回滚

触发条件：
- 健康检查失败
- 错误率超过阈值
- 响应时间超过阈值

### 2. 手动回滚

```bash
# Kubernetes 回滚
kubectl rollout undo deployment/app-deployment -n production

# 蓝绿切换回滚
./nginx/traffic-switch.sh production rollback

# Git 回滚
git revert <commit-hash>
git push origin main
```

## 最佳实践

### 1. 分支策略

```
main (生产)
  └── develop (开发)
       ├── feature/xxx (功能)
       ├── bugfix/xxx (修复)
       └── hotfix/xxx (紧急修复)
```

### 2. 提交规范

```
feat: 新功能
fix: 修复问题
docs: 文档更新
style: 代码格式
refactor: 重构
test: 测试
chore: 构建/工具
```

### 3. 版本管理

- 使用语义化版本: v1.2.3
- 主版本: 不兼容的 API 修改
- 次版本: 向下兼容的功能性新增
- 修订号: 向下兼容的问题修正

## 故障排查

### 1. CI 失败

```bash
# 查看日志
gh run view <run-id>

# 本地复现
act -j <job-name>

# 调试模式
ACTIONS_STEP_DEBUG=true
```

### 2. 部署失败

```bash
# 查看部署状态
kubectl rollout status deployment/<name> -n <namespace>

# 查看日志
kubectl logs -f deployment/<name> -n <namespace>

# 描述资源
kubectl describe deployment/<name> -n <namespace>
```

### 3. 常见问题

- **镜像拉取失败**: 检查 Docker Hub 凭据
- **健康检查超时**: 增加启动探测时间
- **资源不足**: 调整资源限制
- **权限问题**: 检查 RBAC 配置

## 性能优化

### 1. CI 优化

- 并行执行任务
- 缓存依赖项
- 增量构建
- 矩阵构建策略

### 2. 部署优化

- 镜像层缓存
- 多阶段构建
- 资源预分配
- 自动扩缩容

## 安全考虑

### 1. CI/CD 安全

- 最小权限原则
- Secrets 加密存储
- 审计日志
- 签名验证

### 2. 部署安全

- 网络策略
- Pod 安全策略
- RBAC 权限控制
- 镜像扫描

## 总结

本 CI/CD 流程提供了完整的自动化构建、测试和部署能力，确保代码质量和部署安全。通过合理的配置和使用，可以大大提高开发效率和软件质量。