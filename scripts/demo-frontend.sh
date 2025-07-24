#!/bin/bash

# React前端演示脚本
# 🚀 Cloud-Based Collaborative Development Platform - Frontend Demo

echo "🚀 启动React前端演示"
echo "=================================="

# 检查Node.js
if ! command -v node &> /dev/null; then
    echo "❌ Node.js未安装，请先安装Node.js"
    exit 1
fi

echo "✅ Node.js环境检查完成"

# 进入前端目录
cd "/home/jia/Cloud-Based Collaborative Development Platform/frontend"

echo ""
echo "📦 安装依赖包..."
npm install --silent

echo ""
echo "🔧 启动开发服务器..."
echo "🌐 前端应用将在 http://localhost:5173 运行"
echo ""
echo "📋 演示账户（UI展示用）："
echo "   邮箱: demo@example.com"
echo "   密码: demo123"
echo ""
echo "⚠️  注意：这是前端UI演示，后端API暂时不可用"
echo "   可以查看登录界面、项目看板等UI组件"
echo ""
echo "⏹️  停止演示：按 Ctrl+C"
echo ""

# 启动Vite开发服务器
npm run dev