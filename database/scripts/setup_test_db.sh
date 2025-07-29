#!/bin/bash

# 测试数据库设置脚本
set -e

echo "🔧 设置测试数据库环境..."

# 配置变量
DB_HOST=${TEST_DB_HOST:-localhost}
DB_PORT=${TEST_DB_PORT:-5432}
DB_USER=${TEST_DB_USER:-postgres}
DB_PASSWORD=${TEST_DB_PASSWORD:-strongtestpassword2024}
DB_NAME=${TEST_DB_NAME:-devcollab_test}

export PGPASSWORD=$DB_PASSWORD

echo "📋 数据库配置:"
echo "  主机: $DB_HOST:$DB_PORT"
echo "  用户: $DB_USER"
echo "  数据库: $DB_NAME"

# 检查PostgreSQL连接
echo "🔍 检查PostgreSQL连接..."
if ! pg_isready -h $DB_HOST -p $DB_PORT -U $DB_USER > /dev/null 2>&1; then
    echo "❌ PostgreSQL服务不可用，请确保PostgreSQL已启动"
    
    # 尝试启动本地PostgreSQL服务
    if command -v systemctl >/dev/null 2>&1; then
        echo "🚀 尝试启动PostgreSQL服务..."
        sudo systemctl start postgresql || true
        sleep 3
    elif command -v brew >/dev/null 2>&1; then
        echo "🚀 尝试启动PostgreSQL服务 (macOS)..."
        brew services start postgresql || true
        sleep 3
    elif command -v docker >/dev/null 2>&1; then
        echo "🐳 尝试启动PostgreSQL Docker容器..."
        docker run -d \
            --name test-postgres \
            -e POSTGRES_DB=$DB_NAME \
            -e POSTGRES_USER=$DB_USER \
            -e POSTGRES_PASSWORD=$DB_PASSWORD \
            -p $DB_PORT:5432 \
            postgres:13 || true
        
        echo "⏳ 等待PostgreSQL容器启动..."
        sleep 10
    fi
    
    # 再次检查连接
    if ! pg_isready -h $DB_HOST -p $DB_PORT -U $DB_USER > /dev/null 2>&1; then
        echo "❌ 无法连接到PostgreSQL，请手动启动服务"
        echo "💡 建议："
        echo "  1. 安装PostgreSQL: sudo apt-get install postgresql postgresql-contrib"
        echo "  2. 启动服务: sudo systemctl start postgresql"
        echo "  3. 创建用户: sudo -u postgres createuser --superuser $DB_USER"
        echo "  4. 设置密码: sudo -u postgres psql -c \"ALTER USER $DB_USER PASSWORD '$DB_PASSWORD';\""
        exit 1
    fi
fi

echo "✅ PostgreSQL连接成功"

# 删除现有测试数据库（如果存在）
echo "🗑️  清理现有测试数据库..."
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "DROP DATABASE IF EXISTS $DB_NAME;" 2>/dev/null || true

# 创建测试数据库
echo "🏗️  创建测试数据库..."
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "CREATE DATABASE $DB_NAME;"

# 执行初始化脚本
echo "📜 执行数据库初始化脚本..."
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f "$SCRIPT_DIR/init_test_database.sql"

# 验证数据库设置
echo "🔍 验证数据库设置..."
TABLE_COUNT=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';")
echo "  创建的表数量: $TABLE_COUNT"

USER_COUNT=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT COUNT(*) FROM users;")
echo "  测试用户数量: $USER_COUNT"

# 设置测试环境变量
echo "🔧 设置测试环境变量..."
export ENVIRONMENT=test
export TEST_DB_HOST=$DB_HOST
export TEST_DB_PORT=$DB_PORT
export TEST_DB_USER=$DB_USER
export TEST_DB_PASSWORD=$DB_PASSWORD
export TEST_DB_NAME=$DB_NAME
export TEST_JWT_SECRET="test_jwt_secret_for_integration_testing_2024_cloud_platform"

# 保存环境变量到文件
cat > .env.test << EOF
# 测试环境配置
ENVIRONMENT=test
TEST_DB_HOST=$DB_HOST
TEST_DB_PORT=$DB_PORT
TEST_DB_USER=$DB_USER
TEST_DB_PASSWORD=$DB_PASSWORD
TEST_DB_NAME=$DB_NAME
TEST_JWT_SECRET=test_jwt_secret_for_integration_testing_2024_cloud_platform
EOF

echo "✅ 测试数据库设置完成!"
echo ""
echo "🎯 使用以下命令运行测试:"
echo "  source .env.test"
echo "  go test ./... -v"
echo ""
echo "📊 检查测试覆盖率:"
echo "  go test -coverprofile=coverage.out ./..."
echo "  go tool cover -html=coverage.out"