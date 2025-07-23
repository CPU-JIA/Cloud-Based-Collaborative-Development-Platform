#\!/bin/bash
# 智能流量切换器 - 无需重启容器

set -euo pipefail

GREEN_CONFIG="/home/jia/Cloud-Based Collaborative Development Platform/nginx/green.conf"
BLUE_CONFIG="/home/jia/Cloud-Based Collaborative Development Platform/nginx/simple.conf"
TARGET_ENV=${1:-green}

echo "🔄 切换流量到 $TARGET_ENV 环境..."

# 根据目标环境选择配置
if [[ "$TARGET_ENV" == "green" ]]; then
    CONFIG_FILE="$GREEN_CONFIG"
    TARGET_PORT=8083
else
    CONFIG_FILE="$BLUE_CONFIG"
    TARGET_PORT=8082
fi

# 验证目标环境健康状态
echo "🏥 验证目标环境健康状态..."
if \! curl -sf "http://localhost:$TARGET_PORT/api/v1/health" >/dev/null; then
    echo "❌ 目标环境不健康，终止切换"
    exit 1
fi

# 创建临时容器进行配置更新
echo "📋 更新Nginx配置..."
docker run --rm \
  --network container:devcollab-postgres \
  -v "$CONFIG_FILE:/tmp/new.conf:ro" \
  -v "/home/jia/Cloud-Based Collaborative Development Platform/nginx:/host-nginx" \
  busybox sh -c "cp /tmp/new.conf /host-nginx/current.conf"

# 启动新的Nginx容器
echo "🚀 启动新Nginx容器..."
docker stop devcollab-nginx 2>/dev/null || true
docker rm devcollab-nginx 2>/dev/null || true

# 使用host网络模式避免网络配置复杂性
docker run -d \
  --name devcollab-nginx \
  --network host \
  -v "/home/jia/Cloud-Based Collaborative Development Platform/nginx/current.conf:/etc/nginx/conf.d/default.conf:ro" \
  --restart unless-stopped \
  nginx:latest

# 等待Nginx启动
sleep 3

# 验证切换结果
echo "✅ 验证流量切换结果..."
if curl -sf "http://localhost:8080/deployment-status"  < /dev/null |  grep -q "$TARGET_ENV"; then
    echo "🎉 流量切换成功到 $TARGET_ENV 环境"
    echo "📊 当前状态:"
    curl -s "http://localhost:8080/deployment-status"
    exit 0
else
    echo "❌ 流量切换验证失败"
    exit 1
fi
