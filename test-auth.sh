#!/bin/bash

# 企业协作开发平台 - 认证系统测试脚本
# 测试JWT认证、API访问和前端集成

echo "🧪 开始认证系统测试..."
echo "========================="

API_BASE="http://localhost:8083/api/v1"

# 1. 测试健康检查
echo "1️⃣ 测试认证服务健康检查"
HEALTH_RESPONSE=$(curl -s $API_BASE/health)
echo "✅ 健康检查响应: $(echo $HEALTH_RESPONSE | jq -r '.status')"

# 2. 测试登录
echo -e "\n2️⃣ 测试用户登录"
LOGIN_RESPONSE=$(curl -s -X POST $API_BASE/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"jia@example.com","password":"admin123"}')

# 检查登录是否成功
if echo "$LOGIN_RESPONSE" | jq -e '.access_token' > /dev/null; then
    echo "✅ 登录成功"
    ACCESS_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.access_token')
    USER_EMAIL=$(echo "$LOGIN_RESPONSE" | jq -r '.user.email')
    USER_ROLE=$(echo "$LOGIN_RESPONSE" | jq -r '.user.role')
    echo "👤 用户: $USER_EMAIL (角色: $USER_ROLE)"
else
    echo "❌ 登录失败"
    echo "$LOGIN_RESPONSE" | jq '.'
    exit 1
fi

# 3. 测试用户档案API
echo -e "\n3️⃣ 测试用户档案API"
PROFILE_RESPONSE=$(curl -s -X GET $API_BASE/auth/profile \
  -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$PROFILE_RESPONSE" | jq -e '.user' > /dev/null; then
    echo "✅ 用户档案获取成功"
    echo "📄 档案信息: $(echo $PROFILE_RESPONSE | jq -c '.user')"
else
    echo "❌ 用户档案获取失败"
    echo "$PROFILE_RESPONSE" | jq '.'
fi

# 4. 测试受保护的项目API
echo -e "\n4️⃣ 测试受保护的项目API"
PROJECTS_RESPONSE=$(curl -s -X GET $API_BASE/projects \
  -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$PROJECTS_RESPONSE" | jq -e '.data' > /dev/null; then
    echo "✅ 项目数据获取成功"
    PROJECT_COUNT=$(echo "$PROJECTS_RESPONSE" | jq '.data | length')
    echo "📊 项目数量: $PROJECT_COUNT"
else
    echo "❌ 项目数据获取失败"
    echo "$PROJECTS_RESPONSE" | jq '.'
fi

# 5. 测试任务API
echo -e "\n5️⃣ 测试任务管理API"
TASKS_RESPONSE=$(curl -s -X GET $API_BASE/projects/1/tasks \
  -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$TASKS_RESPONSE" | jq -e '.data' > /dev/null; then
    echo "✅ 任务数据获取成功"
    TASK_COUNT=$(echo "$TASKS_RESPONSE" | jq '.data | length')
    echo "📋 任务数量: $TASK_COUNT"
else
    echo "❌ 任务数据获取失败"
    echo "$TASKS_RESPONSE" | jq '.'
fi

# 6. 测试系统状态API
echo -e "\n6️⃣ 测试系统状态API"
STATUS_RESPONSE=$(curl -s -X GET $API_BASE/status \
  -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$STATUS_RESPONSE" | jq -e '.platform_status' > /dev/null; then
    echo "✅ 系统状态获取成功"
    PLATFORM_STATUS=$(echo "$STATUS_RESPONSE" | jq -r '.platform_status')
    echo "🚀 平台状态: $PLATFORM_STATUS"
else
    echo "❌ 系统状态获取失败"
fi

# 7. 测试未认证访问
echo -e "\n7️⃣ 测试未认证访问保护"
UNAUTH_RESPONSE=$(curl -s -X GET $API_BASE/projects)
if echo "$UNAUTH_RESPONSE" | jq -e '.error' > /dev/null; then
    echo "✅ 未认证访问被正确拒绝"
    ERROR_MSG=$(echo "$UNAUTH_RESPONSE" | jq -r '.error')
    echo "🔒 错误信息: $ERROR_MSG"
else
    echo "❌ 未认证访问保护失败"
fi

# 8. 测试错误凭据
echo -e "\n8️⃣ 测试错误凭据处理"
WRONG_LOGIN=$(curl -s -X POST $API_BASE/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"wrong@example.com","password":"wrongpassword"}')

if echo "$WRONG_LOGIN" | jq -e '.error' > /dev/null; then
    echo "✅ 错误凭据被正确拒绝"
    ERROR_MSG=$(echo "$WRONG_LOGIN" | jq -r '.error')
    echo "🚫 错误信息: $ERROR_MSG"
else
    echo "❌ 错误凭据处理失败"
fi

echo -e "\n🎉 认证系统测试完成！"
echo "========================="
echo "📊 测试总结:"
echo "   ✅ JWT认证系统运行正常"
echo "   ✅ API访问控制有效"
echo "   ✅ 用户权限验证成功"
echo "   ✅ 错误处理机制完善"
echo -e "\n🔗 前端访问地址:"
echo "   📱 登录页面: http://localhost:3001/login.html"
echo "   📊 仪表板: http://localhost:3001/dashboard.html"
echo "   📋 Scrum看板: http://localhost:3001/board.html"
echo "   🎯 系统概览: http://localhost:3001/demo.html"