#!/bin/bash

# 产品演示启动脚本
# 🚀 Cloud-Based Collaborative Development Platform Demo

echo "🚀 启动企业协作开发平台演示环境"
echo "=============================================="

# 检查依赖
echo "📋 检查依赖环境..."

# 检查Docker
if ! command -v docker &> /dev/null; then
    echo "❌ Docker未安装，请先安装Docker"
    exit 1
fi

# 检查Node.js
if ! command -v node &> /dev/null; then
    echo "❌ Node.js未安装，请先安装Node.js"
    exit 1
fi

# 检查Go
if ! command -v go &> /dev/null; then
    echo "❌ Go未安装，请先安装Go"
    exit 1
fi

echo "✅ 依赖环境检查完成"

# 启动基础设施
echo ""
echo "🏗️  启动基础设施服务..."
docker compose up -d postgres redis

# 等待数据库启动
echo "⏳ 等待数据库启动..."
sleep 10

# 构建前端
echo ""
echo "⚛️  构建React前端应用..."
cd frontend
npm install --silent
npm run build
cd ..

# 构建Go服务
echo ""
echo "🏗️  构建Go微服务..."
go build -o ./bin/frontend-service ./cmd/frontend-service
go build -o ./bin/project-service ./cmd/project-service

# 启动服务
echo ""
echo "🚀 启动应用服务..."

# 启动项目服务
echo "启动项目服务 (端口8082)..."
./bin/project-service &
PROJECT_PID=$!
sleep 3

# 启动前端服务
echo "启动前端服务 (端口3001)..."
./bin/frontend-service &
FRONTEND_PID=$!
sleep 3

echo ""
echo "🎉 演示环境启动完成！"
echo "=============================================="
echo ""
echo "🌐 访问地址："
echo "   React前端应用: http://localhost:3001"
echo "   项目管理API:   http://localhost:8082"
echo ""
echo "📋 演示账户："
echo "   邮箱: demo@example.com"
echo "   密码: demo123"
echo ""
echo "🔧 服务状态："
echo "   前端服务 PID: $FRONTEND_PID"
echo "   项目服务 PID: $PROJECT_PID"
echo ""
echo "⏹️  停止演示环境："
echo "   按 Ctrl+C 或运行: ./scripts/stop-demo.sh"
echo ""

# 创建PID文件
echo $FRONTEND_PID > .frontend.pid
echo $PROJECT_PID > .project.pid

# 等待用户输入
echo "按 Enter 键停止演示环境..."
read

# 清理
echo ""
echo "🧹 清理演示环境..."
kill $FRONTEND_PID 2>/dev/null
kill $PROJECT_PID 2>/dev/null
docker compose down

echo "✅ 演示环境已停止"