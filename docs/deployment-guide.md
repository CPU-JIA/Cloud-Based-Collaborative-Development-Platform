# 部署指南

## 概述

本文档提供了云协作开发平台的完整部署指南，包括本地开发、测试环境、生产环境的配置和部署流程。

## 系统要求

### 硬件要求

**最低配置：**
- CPU: 2核
- 内存: 4GB RAM
- 存储: 20GB 可用磁盘空间
- 网络: 宽带互联网连接

**推荐配置：**
- CPU: 4核或更高
- 内存: 8GB RAM 或更高
- 存储: 50GB+ SSD
- 网络: 稳定的互联网连接

### 软件要求

- **操作系统**: Linux (Ubuntu 20.04+/CentOS 8+), macOS 10.15+, Windows 10+
- **Go**: 1.23+
- **Node.js**: 18+
- **Docker**: 20.10+
- **Docker Compose**: 2.0+
- **Kubernetes**: 1.25+ (生产环境)
- **Helm**: 3.0+ (Kubernetes部署)

## 快速开始

### 1. 克隆项目

```bash
git clone https://github.com/your-org/cloud-collaborative-platform.git
cd cloud-collaborative-platform
```

### 2. 环境配置

```bash
# 复制环境配置模板
cp .env.example .env

# 编辑配置文件
nano .env
```

### 3. 本地开发环境

```bash
# 安装依赖
make install

# 启动开发环境
make dev
```

## 详细部署指南

### 本地开发环境

#### 1. 环境准备

```bash
# 安装Go依赖
go mod download

# 安装前端依赖
cd frontend && npm install && cd ..

# 设置环境变量
export DATABASE_URL="postgres://user:password@localhost:5432/platform_dev"
export REDIS_URL="redis://localhost:6379"
export JWT_SECRET="your-jwt-secret-key"
```

#### 2. 数据库设置

```bash
# 启动PostgreSQL (使用Docker)
docker run -d \
  --name postgres-dev \
  -e POSTGRES_DB=platform_dev \
  -e POSTGRES_USER=platform \
  -e POSTGRES_PASSWORD=password123 \
  -p 5432:5432 \
  postgres:15

# 运行数据库迁移
make migrate-up
```

#### 3. 启动服务

```bash
# 方式1: 使用Makefile
make run-all

# 方式2: 使用Docker Compose
docker-compose -f docker-compose.yml up -d

# 方式3: 手动启动各服务
make run-project-service &
make run-iam-service &
make run-git-gateway &
# ... 其他服务
```

### 测试环境部署

#### 1. 使用Docker Compose

```bash
# 启动完整测试环境
docker-compose -f docker-compose.test.yml up -d

# 运行测试套件
make test-all

# 查看服务状态
docker-compose ps
```

#### 2. 环境验证

```bash
# 健康检查
./scripts/health-checks.sh test

# 冒烟测试
./scripts/smoke-tests.sh test

# API测试
curl http://localhost:8080/api/health
curl http://localhost:8081/api/v1/health
```

### 生产环境部署

#### 选项1: Docker Compose部署

```bash
# 1. 准备生产配置
cp .env.example .env.production
# 编辑生产环境配置

# 2. 构建生产镜像
make build-prod

# 3. 启动生产环境
docker-compose -f docker-compose.prod.yml up -d

# 4. 验证部署
make verify-prod
```

#### 选项2: Kubernetes部署

```bash
# 1. 准备Kubernetes集群
kubectl cluster-info

# 2. 安装Helm Chart
helm repo add bitnami https://charts.bitnami.com/bitnami
helm dependency update deployments/helm/cloud-platform

# 3. 部署到Kubernetes
helm install cloud-platform ./deployments/helm/cloud-platform \
  --namespace cloud-platform \
  --create-namespace \
  --values deployments/helm/cloud-platform/values-production.yaml

# 4. 验证部署
kubectl get pods -n cloud-platform
kubectl get services -n cloud-platform
```

## 配置管理

### 环境变量配置

#### 核心配置

```bash
# 应用配置
APP_ENV=production
APP_NAME="Cloud Collaborative Platform"
APP_VERSION=1.0.0
LOG_LEVEL=info

# 数据库配置
DATABASE_URL=postgres://user:pass@host:5432/dbname
DATABASE_MAX_CONNECTIONS=100
DATABASE_CONNECTION_TIMEOUT=30s

# Redis配置
REDIS_URL=redis://localhost:6379
REDIS_PASSWORD=your-redis-password
REDIS_MAX_CONNECTIONS=50

# 安全配置
JWT_SECRET=your-super-secret-jwt-key
ENCRYPTION_KEY=your-32-char-encryption-key
CSRF_SECRET=your-csrf-secret-key

# 服务端口
PROJECT_SERVICE_PORT=8080
IAM_SERVICE_PORT=8081
GIT_GATEWAY_PORT=8082
NOTIFICATION_SERVICE_PORT=8083
# ... 其他服务端口
```

#### 外部服务配置

```bash
# Git集成
GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret
GITLAB_CLIENT_ID=your-gitlab-client-id
GITLAB_CLIENT_SECRET=your-gitlab-client-secret

# 邮件服务
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USERNAME=noreply@example.com
SMTP_PASSWORD=your-smtp-password

# 对象存储
S3_BUCKET=your-s3-bucket
S3_REGION=us-west-2
S3_ACCESS_KEY=your-access-key
S3_SECRET_KEY=your-secret-key

# 监控和日志
PROMETHEUS_ENDPOINT=http://prometheus:9090
GRAFANA_URL=http://grafana:3000
ELASTICSEARCH_URL=http://elasticsearch:9200
```

### 密钥管理

#### 开发环境

```bash
# 使用.env文件
echo "JWT_SECRET=$(openssl rand -hex 32)" >> .env
echo "ENCRYPTION_KEY=$(openssl rand -hex 32)" >> .env
```

#### 生产环境

```bash
# 使用Kubernetes Secrets
kubectl create secret generic app-secrets \
  --from-literal=jwt-secret=$(openssl rand -hex 32) \
  --from-literal=encryption-key=$(openssl rand -hex 32) \
  --from-literal=database-password="your-db-password" \
  -n cloud-platform

# 使用HashiCorp Vault (推荐)
vault kv put secret/cloud-platform \
  jwt_secret="$(openssl rand -hex 32)" \
  encryption_key="$(openssl rand -hex 32)" \
  database_password="your-secure-password"
```

## 监控和日志

### 应用监控

```bash
# Prometheus指标
curl http://localhost:9090/metrics

# 应用健康检查
curl http://localhost:8080/health
curl http://localhost:8080/metrics
```

### 日志管理

```bash
# 查看服务日志
docker-compose logs -f project-service
kubectl logs -f deployment/project-service -n cloud-platform

# 日志聚合 (ELK Stack)
# 配置在 docker-compose.yml 中
```

## 性能优化

### 数据库优化

```sql
-- 创建必要的索引
CREATE INDEX IF NOT EXISTS idx_projects_owner_id ON projects(owner_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_files_project_id ON files(project_id);

-- 连接池配置
-- 在配置文件中设置
max_connections: 100
connection_timeout: 30s
idle_timeout: 5m
```

### 缓存配置

```yaml
# Redis缓存配置
redis:
  host: redis
  port: 6379
  password: your-password
  db: 0
  max_connections: 50
  timeout: 5s
  
# 应用级缓存
cache:
  ttl: 300s # 5分钟
  max_memory: 100MB
```

## 安全配置

### SSL/TLS配置

```nginx
# Nginx配置
server {
    listen 443 ssl http2;
    server_name your-domain.com;
    
    ssl_certificate /path/to/certificate.crt;
    ssl_certificate_key /path/to/private.key;
    
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512;
    
    location / {
        proxy_pass http://frontend:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
    
    location /api/ {
        proxy_pass http://api-gateway:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### 防火墙配置

```bash
# UFW配置 (Ubuntu)
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw enable

# iptables配置
iptables -A INPUT -p tcp --dport 22 -j ACCEPT
iptables -A INPUT -p tcp --dport 80 -j ACCEPT
iptables -A INPUT -p tcp --dport 443 -j ACCEPT
```

## 备份和恢复

### 数据库备份

```bash
# 自动备份脚本
#!/bin/bash
BACKUP_DIR="/backups"
DATE=$(date +%Y%m%d_%H%M%S)

# PostgreSQL备份
pg_dump $DATABASE_URL > "$BACKUP_DIR/postgres_$DATE.sql"

# 压缩备份
gzip "$BACKUP_DIR/postgres_$DATE.sql"

# 清理旧备份 (保留30天)
find $BACKUP_DIR -name "postgres_*.sql.gz" -mtime +30 -delete
```

### 配置备份

```bash
# 备份配置文件
tar -czf config_backup_$(date +%Y%m%d).tar.gz \
  .env.production \
  configs/ \
  deployments/helm/cloud-platform/values-production.yaml

# 备份到云存储
aws s3 cp config_backup_$(date +%Y%m%d).tar.gz \
  s3://your-backup-bucket/configs/
```

## 故障排除

### 常见问题

#### 1. 服务启动失败

```bash
# 检查日志
docker-compose logs service-name

# 检查端口占用
netstat -tulpn | grep :8080

# 检查环境变量
env | grep -E "(DATABASE|REDIS|JWT)"
```

#### 2. 数据库连接问题

```bash
# 测试数据库连接
psql $DATABASE_URL -c "SELECT version();"

# 检查数据库状态
docker-compose exec postgres pg_isready

# 查看连接数
psql $DATABASE_URL -c "SELECT count(*) FROM pg_stat_activity;"
```

#### 3. 内存和CPU问题

```bash
# 监控资源使用
docker stats

# 系统资源监控
top
htop
free -h
df -h
```

### 紧急恢复流程

#### 1. 服务恢复

```bash
# 快速重启所有服务
docker-compose down && docker-compose up -d

# 回滚到上一个版本
helm rollback cloud-platform
```

#### 2. 数据恢复

```bash
# 从备份恢复数据库
gunzip latest_backup.sql.gz
psql $DATABASE_URL < latest_backup.sql

# 从Kubernetes备份恢复
kubectl apply -f backup-restore-job.yaml
```

## 升级和维护

### 应用升级

```bash
# 1. 准备升级
git fetch --tags
git checkout v1.1.0

# 2. 数据库迁移
make migrate-up

# 3. 构建新版本
make build-prod

# 4. 滚动升级 (Kubernetes)
helm upgrade cloud-platform ./deployments/helm/cloud-platform \
  --values deployments/helm/cloud-platform/values-production.yaml

# 5. 验证升级
./scripts/smoke-tests.sh production
```

### 定期维护

```bash
# 每日维护脚本
#!/bin/bash
# 清理日志
find /var/log -name "*.log" -size +100M -delete

# 清理Docker资源
docker system prune -f

# 数据库维护
psql $DATABASE_URL -c "VACUUM ANALYZE;"

# 备份检查
./scripts/verify-backups.sh
```

## 支持和联系

- **文档**: [项目Wiki](https://github.com/your-org/cloud-platform/wiki)
- **问题反馈**: [GitHub Issues](https://github.com/your-org/cloud-platform/issues)
- **技术支持**: support@your-domain.com
- **紧急联系**: +1-555-0123 (24/7)

## 附录

### A. 端口映射

| 服务 | 内部端口 | 外部端口 | 协议 |
|------|----------|----------|------|
| Frontend | 3000 | 80/443 | HTTP/HTTPS |
| API Gateway | 8080 | 8080 | HTTP |
| Project Service | 8080 | - | HTTP |
| IAM Service | 8081 | - | HTTP |
| Git Gateway | 8082 | - | HTTP |
| Notification | 8083 | - | HTTP |
| PostgreSQL | 5432 | - | TCP |
| Redis | 6379 | - | TCP |

### B. 环境对比

| 配置项 | 开发环境 | 测试环境 | 生产环境 |
|--------|----------|----------|----------|
| 日志级别 | debug | info | warn |
| 数据库连接数 | 10 | 50 | 100 |
| Redis连接数 | 10 | 25 | 50 |
| CPU限制 | 无限制 | 2核 | 4核 |
| 内存限制 | 无限制 | 4GB | 8GB |
| 副本数 | 1 | 2 | 3 |

### C. 监控指标

- **应用指标**: QPS, 响应时间, 错误率
- **系统指标**: CPU, 内存, 磁盘, 网络
- **业务指标**: 用户数, 项目数, 活跃度
- **安全指标**: 登录失败次数, 异常访问

---

*最后更新: 2025-07-27*
*版本: 1.0.0*