#!/bin/bash
# 企业协作开发平台 - 蓝绿部署脚本
# Blue-Green Deployment Strategy Implementation

set -euo pipefail

# 配置变量
BLUE_PORT=8082
GREEN_PORT=8083
LB_PORT=8080
PROJECT_NAME="devcollab"
SERVICE_NAME="project-service"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 日志函数
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
}

# 获取当前活跃环境
get_active_environment() {
    # 检查当前Nginx配置指向哪个端口
    local current_upstream=$(docker exec devcollab-nginx cat /etc/nginx/conf.d/default.conf | grep "server host.docker.internal:" | grep -o ":[0-9]*" | head -1 | sed 's/://')
    
    if [[ "$current_upstream" == "$BLUE_PORT" ]]; then
        echo "blue"
    elif [[ "$current_upstream" == "$GREEN_PORT" ]]; then
        echo "green"
    else
        echo "unknown"
    fi
}

# 获取目标环境
get_target_environment() {
    local active=$(get_active_environment)
    if [[ "$active" == "blue" ]]; then
        echo "green"
    else
        echo "blue"
    fi
}

# 获取环境端口
get_environment_port() {
    local env=$1
    if [[ "$env" == "blue" ]]; then
        echo $BLUE_PORT
    else
        echo $GREEN_PORT
    fi
}

# 健康检查
health_check() {
    local port=$1
    local max_attempts=30
    local attempt=1
    
    log "执行健康检查 - 端口 $port"
    
    while [ $attempt -le $max_attempts ]; do
        if curl -sf "http://localhost:$port/api/v1/health" >/dev/null 2>&1; then
            log "✅ 健康检查通过 - 端口 $port (尝试 $attempt/$max_attempts)"
            return 0
        fi
        
        warn "健康检查失败 - 端口 $port (尝试 $attempt/$max_attempts)"
        sleep 2
        ((attempt++))
    done
    
    error "❌ 健康检查失败 - 端口 $port"
    return 1
}

# 构建新版本
build_new_version() {
    local target_env=$1
    local target_port=$(get_environment_port $target_env)
    
    log "🔨 构建新版本到 $target_env 环境 (端口: $target_port)"
    
    # 重新编译最新代码
    cd "/home/jia/Cloud-Based Collaborative Development Platform"
    log "编译最新静态二进制文件..."
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o "bin/${SERVICE_NAME}-${target_env}" ./cmd/project-service/
    
    # 验证二进制文件
    if [[ ! -f "bin/${SERVICE_NAME}-${target_env}" ]]; then
        error "二进制文件构建失败"
        return 1
    fi
    
    log "✅ 新版本构建完成 - $target_env 环境"
    return 0
}

# 部署到目标环境
deploy_to_environment() {
    local target_env=$1
    local target_port=$(get_environment_port $target_env)
    
    log "🚀 部署到 $target_env 环境 (端口: $target_port)"
    
    # 停止旧进程
    pkill -f "${SERVICE_NAME}-${target_env}" 2>/dev/null || true
    sleep 2
    
    # 启动新服务
    cd "/home/jia/Cloud-Based Collaborative Development Platform"
    
    POSTGRES_HOST=localhost \
    POSTGRES_PORT=15432 \
    POSTGRES_DB=devcollab_production \
    POSTGRES_USER=devcollab_prod_user \
    POSTGRES_PASSWORD=devcollab_prod_pass \
    REDIS_HOST=localhost \
    REDIS_PORT=16379 \
    REDIS_PASSWORD=redis_prod_pass_123 \
    JWT_SECRET_KEY=super-secure-jwt-key-for-production-environment-32characters \
    GIN_MODE=release \
    LOG_LEVEL=INFO \
    SERVER_PORT=$target_port \
    nohup "./bin/${SERVICE_NAME}-${target_env}" > "${target_env}_service.log" 2>&1 &
    
    local pid=$!
    echo $pid > "${target_env}_service.pid"
    
    log "✅ $target_env 环境服务已启动 (PID: $pid, 端口: $target_port)"
    
    # 等待服务启动
    sleep 5
    
    # 健康检查
    if health_check $target_port; then
        log "✅ $target_env 环境部署成功"
        return 0
    else
        error "❌ $target_env 环境部署失败"
        return 1
    fi
}

# 创建Nginx配置
create_nginx_config() {
    local target_port=$1
    local config_name=$2
    
    cat > "/home/jia/Cloud-Based Collaborative Development Platform/nginx/${config_name}.conf" << EOF
# 企业协作开发平台 - 蓝绿部署Nginx配置
upstream project_service {
    least_conn;
    server host.docker.internal:${target_port} max_fails=3 fail_timeout=30s;
    keepalive 32;
}

# 限流配置
limit_req_zone \$binary_remote_addr zone=api_limit:10m rate=100r/m;

# Gzip压缩
gzip on;
gzip_vary on;
gzip_min_length 1024;
gzip_proxied any;
gzip_comp_level 6;
gzip_types
    text/plain
    text/css
    text/xml
    text/javascript
    application/json
    application/javascript
    application/xml+rss
    application/atom+xml
    image/svg+xml;

server {
    listen 80 default_server;
    listen [::]:80 default_server;
    server_name api.devcollab.cloud localhost _;
    
    # 安全头设置
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;
    
    # 健康检查端点
    location /nginx-health {
        access_log off;
        return 200 "healthy\n";
        add_header Content-Type text/plain;
    }
    
    # 蓝绿部署状态端点
    location /deployment-status {
        access_log off;
        return 200 "Environment: ${config_name}, Port: ${target_port}\n";
        add_header Content-Type text/plain;
    }
    
    # 项目服务路由
    location /api/v1/ {
        limit_req zone=api_limit burst=20 nodelay;
        
        proxy_pass http://project_service;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_cache_bypass \$http_upgrade;
        
        proxy_connect_timeout 5s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
    
    # 监控指标端点
    location /metrics {
        allow 127.0.0.1;
        allow 10.0.0.0/8;
        allow 172.16.0.0/12;
        allow 192.168.0.0/16;
        deny all;
        
        proxy_pass http://project_service/api/v1/metrics;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
    
    # 默认响应
    location / {
        return 200 '{"service": "devcollab-api-gateway", "status": "healthy", "environment": "${config_name}", "port": ${target_port}}';
        add_header Content-Type application/json;
    }
}
EOF
}

# 切换流量
switch_traffic() {
    local target_env=$1
    local target_port=$(get_environment_port $target_env)
    
    log "🔄 切换流量到 $target_env 环境 (端口: $target_port)"
    
    # 创建新的Nginx配置
    create_nginx_config $target_port $target_env
    
    # 更新Nginx配置
    docker cp "/home/jia/Cloud-Based Collaborative Development Platform/nginx/${target_env}.conf" devcollab-nginx:/etc/nginx/conf.d/default.conf
    
    # 重新加载Nginx配置
    docker exec devcollab-nginx nginx -s reload
    
    # 验证切换
    sleep 2
    if curl -sf "http://localhost:$LB_PORT/deployment-status" | grep -q "$target_env"; then
        log "✅ 流量切换成功 - 当前环境: $target_env"
        return 0
    else
        error "❌ 流量切换失败"
        return 1
    fi
}

# 清理旧环境
cleanup_old_environment() {
    local old_env=$1
    
    warn "🧹 清理旧环境: $old_env"
    
    # 停止旧服务
    if [[ -f "${old_env}_service.pid" ]]; then
        local old_pid=$(cat "${old_env}_service.pid")
        if kill -0 "$old_pid" 2>/dev/null; then
            kill "$old_pid"
            log "✅ 已停止旧服务 (PID: $old_pid)"
        fi
        rm -f "${old_env}_service.pid"
    fi
    
    # 清理旧日志
    if [[ -f "${old_env}_service.log" ]]; then
        mv "${old_env}_service.log" "${old_env}_service.log.$(date +%Y%m%d_%H%M%S)"
    fi
    
    log "✅ $old_env 环境清理完成"
}

# 回滚函数
rollback() {
    local current_env=$(get_active_environment)
    local rollback_env
    
    if [[ "$current_env" == "blue" ]]; then
        rollback_env="green"
    else
        rollback_env="blue"
    fi
    
    error "🔄 执行回滚到 $rollback_env 环境"
    
    # 检查回滚目标是否健康
    local rollback_port=$(get_environment_port $rollback_env)
    if health_check $rollback_port; then
        switch_traffic $rollback_env
        log "✅ 回滚成功到 $rollback_env 环境"
    else
        error "❌ 回滚失败 - $rollback_env 环境不健康"
        return 1
    fi
}

# 主部署函数
deploy() {
    log "🚀 开始蓝绿部署流程"
    
    # 获取当前环境信息
    local active_env=$(get_active_environment)
    local target_env=$(get_target_environment)
    
    log "当前活跃环境: $active_env"
    log "目标部署环境: $target_env"
    
    # 构建新版本
    if ! build_new_version $target_env; then
        error "构建失败"
        return 1
    fi
    
    # 部署到目标环境
    if ! deploy_to_environment $target_env; then
        error "部署失败"
        return 1
    fi
    
    # 进行冒烟测试
    log "🧪 执行冒烟测试..."
    local target_port=$(get_environment_port $target_env)
    
    # 基本功能测试
    if ! curl -sf "http://localhost:$target_port/api/v1/health" >/dev/null; then
        error "冒烟测试失败 - 健康检查"
        return 1
    fi
    
    # API认证测试
    local auth_response=$(curl -s "http://localhost:$target_port/api/v1/projects" | grep -o "MISSING_AUTH_HEADER" || true)
    if [[ "$auth_response" != "MISSING_AUTH_HEADER" ]]; then
        error "冒烟测试失败 - 认证机制"
        return 1
    fi
    
    log "✅ 冒烟测试通过"
    
    # 切换流量
    if ! switch_traffic $target_env; then
        error "流量切换失败，执行回滚"
        rollback
        return 1
    fi
    
    # 验证新环境
    sleep 5
    if ! health_check $LB_PORT; then
        error "新环境验证失败，执行回滚"
        rollback
        return 1
    fi
    
    # 清理旧环境
    cleanup_old_environment $active_env
    
    log "🎉 蓝绿部署完成！"
    log "当前活跃环境: $target_env (端口: $target_port)"
    
    return 0
}

# 状态检查函数
status() {
    log "📊 蓝绿部署状态检查"
    
    local active_env=$(get_active_environment)
    local blue_port=$BLUE_PORT
    local green_port=$GREEN_PORT
    
    echo
    echo "🔵 蓝环境 (端口: $blue_port):"
    if curl -sf "http://localhost:$blue_port/api/v1/health" >/dev/null 2>&1; then
        echo "  状态: ✅ 健康"
        echo "  活跃: $([ "$active_env" == "blue" ] && echo "是" || echo "否")"
    else
        echo "  状态: ❌ 不健康或未运行"
    fi
    
    echo
    echo "🟢 绿环境 (端口: $green_port):"
    if curl -sf "http://localhost:$green_port/api/v1/health" >/dev/null 2>&1; then
        echo "  状态: ✅ 健康"
        echo "  活跃: $([ "$active_env" == "green" ] && echo "是" || echo "否")"
    else
        echo "  状态: ❌ 不健康或未运行"
    fi
    
    echo
    echo "🌐 负载均衡器 (端口: $LB_PORT):"
    if curl -sf "http://localhost:$LB_PORT/nginx-health" >/dev/null 2>&1; then
        echo "  状态: ✅ 健康"
        echo "  当前环境: $active_env"
        curl -s "http://localhost:$LB_PORT/deployment-status" | sed 's/^/  /'
    else
        echo "  状态: ❌ 不健康"
    fi
}

# 主程序
case "${1:-}" in
    "deploy")
        deploy
        ;;
    "rollback")
        rollback
        ;;
    "status")
        status
        ;;
    *)
        echo "Usage: $0 {deploy|rollback|status}"
        echo
        echo "Commands:"
        echo "  deploy   - 执行蓝绿部署"
        echo "  rollback - 回滚到前一个环境"
        echo "  status   - 查看当前部署状态"
        exit 1
        ;;
esac