# 密钥管理最佳实践

## 概述

本文档描述了 Cloud-Based Collaborative Development Platform 的密钥管理系统和最佳实践。

## 目录

1. [架构设计](#架构设计)
2. [密钥管理系统](#密钥管理系统)
3. [环境配置](#环境配置)
4. [密钥轮换](#密钥轮换)
5. [安全最佳实践](#安全最佳实践)
6. [故障排除](#故障排除)

## 架构设计

### 密钥管理层次

```
┌─────────────────────────────────────────────────┐
│              应用层 (Application)                │
├─────────────────────────────────────────────────┤
│           密钥管理器 (Secret Manager)            │
├─────────────────────────────────────────────────┤
│            密钥提供者 (Providers)                │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐     │
│  │   文件    │  │ 环境变量  │  │  Vault   │     │
│  │ Provider │  │ Provider │  │ Provider │     │
│  └──────────┘  └──────────┘  └──────────┘     │
└─────────────────────────────────────────────────┘
```

### 核心组件

1. **Secret Manager**: 统一的密钥管理接口
2. **Providers**: 不同的密钥存储后端
3. **Encryption**: AES-256-GCM 加密
4. **Rotation**: 自动密钥轮换
5. **Validation**: 密钥强度验证

## 密钥管理系统

### 初始化密钥

```bash
# 开发环境
./bin/secrets-cli init --env development

# 生产环境
./bin/secrets-cli init --env production --provider vault
```

### 设置密钥

```bash
# 设置单个密钥
./bin/secrets-cli set database_password "your-secure-password"

# 从标准输入设置（更安全）
echo "your-secure-password" | ./bin/secrets-cli set database_password -
```

### 获取密钥

```bash
# 查看密钥（masked）
./bin/secrets-cli get database_password

# 显示完整值
./bin/secrets-cli get database_password --show
```

### 列出所有密钥

```bash
./bin/secrets-cli list
```

### 验证密钥配置

```bash
./bin/secrets-cli validate
```

## 环境配置

### 开发环境

1. 复制环境变量模板：
   ```bash
   cp configs/.env.example .env
   ```

2. 编辑 `.env` 文件，设置必要的值

3. 使用文件提供者存储密钥：
   ```bash
   export SECRETS_ENCRYPTION_KEY="your-development-encryption-key"
   ./bin/secrets-cli init --env development --provider file
   ```

### 测试环境

1. 使用环境变量提供者：
   ```bash
   export CLOUDPLATFORM_DATABASE_PASSWORD="test-db-password"
   export CLOUDPLATFORM_JWT_SECRET="test-jwt-secret-minimum-32-chars"
   export CLOUDPLATFORM_ENCRYPTION_KEY="test-encryption-key"
   ```

### 生产环境

1. **推荐使用 HashiCorp Vault**:
   ```bash
   export VAULT_ADDR="https://vault.example.com:8200"
   export VAULT_TOKEN="your-vault-token"
   ./bin/secrets-cli init --env production --provider vault
   ```

2. **备选方案 - 环境变量**:
   ```bash
   # 在 CI/CD 或 Kubernetes 中设置
   export CLOUDPLATFORM_DATABASE_PASSWORD="${DB_PASSWORD}"
   export CLOUDPLATFORM_JWT_SECRET="${JWT_SECRET}"
   ```

## 密钥轮换

### 自动轮换

系统支持自动密钥轮换，配置示例：

```go
rotationManager := secrets.NewRotationManager(secretManager)

// 设置轮换策略
rotationManager.SetPolicy("database_password", &secrets.RotationPolicy{
    Enabled:      true,
    Interval:     30 * 24 * time.Hour, // 30天
    NotifyBefore: 7 * 24 * time.Hour,  // 提前7天通知
})

// 启动轮换管理器
rotationManager.Start()
```

### 手动轮换

```bash
# 轮换特定密钥
./bin/secrets-cli rotate database_password

# 强制轮换（不需要确认）
./bin/secrets-cli rotate jwt_secret --force
```

### 轮换流程

1. 生成新密钥
2. 更新密钥存储
3. 通知相关服务
4. 记录轮换历史
5. 验证新密钥

## 安全最佳实践

### 1. 密钥强度要求

- **数据库密码**: 最少 12 字符
- **JWT 密钥**: 最少 32 字符
- **API 密钥**: 最少 32 字符
- **加密密钥**: 最少 32 字符

### 2. 存储安全

- **开发环境**: 使用加密的文件存储
- **生产环境**: 使用 Vault 或 云提供商的密钥管理服务
- **传输**: 始终使用 TLS/SSL
- **权限**: 最小权限原则

### 3. 代码安全

```go
// ❌ 错误：硬编码密码
password := "admin123"

// ✅ 正确：从密钥管理器获取
password, err := secretManager.GetSecret("database_password")
if err != nil {
    return err
}
```

### 4. Git 安全

确保以下文件在 `.gitignore` 中：

```
.env
*.secrets.yaml
configs/secrets/
credentials.json
*.pem
*.key
```

### 5. Docker 安全

使用环境变量而不是硬编码：

```yaml
# docker-compose.yml
services:
  app:
    environment:
      - DATABASE_PASSWORD=${DATABASE_PASSWORD}
      - JWT_SECRET=${JWT_SECRET}
```

### 6. CI/CD 安全

- 使用 CI/CD 平台的密钥管理功能
- 不要在日志中打印密钥
- 使用临时凭据
- 实施密钥扫描

### 7. 监控和审计

- 记录密钥访问日志
- 监控异常访问模式
- 定期审计密钥使用
- 实施告警机制

## 故障排除

### 常见问题

1. **密钥未找到错误**
   ```
   Error: secret database_password not found
   ```
   解决方案：运行 `secrets-cli init` 初始化密钥

2. **加密密钥错误**
   ```
   Error: SECRETS_ENCRYPTION_KEY is required
   ```
   解决方案：设置环境变量 `SECRETS_ENCRYPTION_KEY`

3. **Vault 连接失败**
   ```
   Error: failed to create vault client
   ```
   解决方案：检查 `VAULT_ADDR` 和 `VAULT_TOKEN`

4. **密钥验证失败**
   ```
   Error: JWT密钥长度必须至少32字符
   ```
   解决方案：生成符合要求的密钥

### 调试技巧

1. 启用调试日志：
   ```bash
   export LOG_LEVEL=debug
   ```

2. 检查配置加载：
   ```bash
   ./bin/secrets-cli validate
   ```

3. 测试密钥提供者：
   ```bash
   ./bin/secrets-cli list
   ```

## 迁移指南

### 从硬编码密码迁移

1. **识别硬编码密码**：
   ```bash
   grep -r "password\|secret\|key" --include="*.go" .
   ```

2. **创建密钥配置**：
   ```bash
   ./bin/secrets-cli init
   ```

3. **更新代码**：
   ```go
   // 旧代码
   cfg.Database.Password = "hardcoded_password"
   
   // 新代码
   cfg.Database.Password, _ = secretManager.GetSecret("database_password")
   ```

4. **更新部署配置**：
   - 更新 Docker Compose
   - 更新 Kubernetes 配置
   - 更新 CI/CD 流程

5. **验证迁移**：
   ```bash
   ./bin/secrets-cli validate
   ```

## 合规性

### 支持的合规标准

- **PCI DSS**: 密钥加密存储和传输
- **HIPAA**: 访问控制和审计日志
- **SOC 2**: 密钥轮换和访问管理
- **GDPR**: 数据加密和访问控制

### 审计要求

1. 所有密钥访问必须记录
2. 密钥轮换历史必须保留
3. 定期密钥强度审查
4. 访问权限定期审查

## 总结

通过实施这个密钥管理系统，我们确保：

1. ✅ 没有硬编码的密码
2. ✅ 密钥安全存储和传输
3. ✅ 支持多环境配置
4. ✅ 自动密钥轮换
5. ✅ 完整的审计跟踪
6. ✅ 符合安全最佳实践

记住：**永远不要在代码中硬编码密钥！**