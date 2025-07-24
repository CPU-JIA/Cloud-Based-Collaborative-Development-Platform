#!/bin/bash

# 企业协作开发平台 - 完整系统测试脚本
# 测试所有核心功能和页面访问

echo "🚀 企业协作开发平台 - 完整系统测试"
echo "========================================"

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试结果统计
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 测试函数
test_url() {
    local url=$1
    local description=$2
    local expected_code=${3:-200}
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    echo -n "  测试 $description ... "
    
    local response_code=$(curl -s -o /dev/null -w "%{http_code}" "$url")
    
    if [ "$response_code" -eq "$expected_code" ]; then
        echo -e "${GREEN}✅ 通过${NC} ($response_code)"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}❌ 失败${NC} (期望: $expected_code, 实际: $response_code)"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

test_api() {
    local url=$1
    local description=$2
    local method=${3:-GET}
    local data=$4
    local auth_header=$5
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    echo -n "  测试 $description ... "
    
    local curl_cmd="curl -s -X $method"
    
    if [ -n "$auth_header" ]; then
        curl_cmd="$curl_cmd -H \"Authorization: Bearer $auth_header\""
    fi
    
    if [ -n "$data" ]; then
        curl_cmd="$curl_cmd -H \"Content-Type: application/json\" -d '$data'"
    fi
    
    curl_cmd="$curl_cmd -w \"%{http_code}\" \"$url\""
    
    local response=$(eval $curl_cmd)
    local response_code=$(echo "$response" | tail -c 4)
    
    if [ "$response_code" -eq "200" ] || [ "$response_code" -eq "201" ]; then
        echo -e "${GREEN}✅ 通过${NC} ($response_code)"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        
        # 如果是登录API，提取token
        if [[ "$url" == *"/auth/login" ]]; then
            AUTH_TOKEN=$(echo "$response" | jq -r '.access_token' 2>/dev/null || echo "")
        fi
    else
        echo -e "${RED}❌ 失败${NC} ($response_code)"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

# 开始测试
echo -e "${BLUE}🌐 1. 前端页面访问测试${NC}"
test_url "http://localhost:3001/" "主页访问"
test_url "http://localhost:3001/index.html" "首页访问"
test_url "http://localhost:3001/product-demo.html" "产品演示页面"
test_url "http://localhost:3001/login.html" "登录页面"
test_url "http://localhost:3001/dashboard.html" "仪表板页面"
test_url "http://localhost:3001/board.html" "Scrum看板页面"
test_url "http://localhost:3001/knowledge.html" "知识库页面"
test_url "http://localhost:3001/demo.html" "系统概览页面"

echo ""
echo -e "${BLUE}🔐 2. 认证服务测试 (端口 8083)${NC}"

# 检查认证服务是否运行
if curl -s "http://localhost:8083/api/v1/health" > /dev/null; then
    test_api "http://localhost:8083/api/v1/health" "认证服务健康检查"
    
    # 登录测试
    test_api "http://localhost:8083/api/v1/auth/login" "用户登录" "POST" '{"email":"jia@example.com","password":"admin123"}'
    
    # 如果登录成功，测试受保护的API
    if [ -n "$AUTH_TOKEN" ] && [ "$AUTH_TOKEN" != "null" ]; then
        echo "  获取到认证token: ${AUTH_TOKEN:0:20}..."
        test_api "http://localhost:8083/api/v1/auth/profile" "用户档案获取" "GET" "" "$AUTH_TOKEN"
        test_api "http://localhost:8083/api/v1/projects" "项目列表获取" "GET" "" "$AUTH_TOKEN"
    else
        echo -e "  ${YELLOW}⚠️ 无法获取认证token，跳过受保护API测试${NC}"
    fi
else
    echo -e "  ${RED}❌ 认证服务未运行 (端口 8083)${NC}"
    TOTAL_TESTS=$((TOTAL_TESTS + 3))
    FAILED_TESTS=$((FAILED_TESTS + 3))
fi

echo ""
echo -e "${BLUE}📚 3. 知识库服务测试 (端口 8084)${NC}"

# 检查知识库服务是否运行
if curl -s "http://localhost:8084/api/v1/health" > /dev/null; then
    test_api "http://localhost:8084/api/v1/health" "知识库服务健康检查"
    
    # 如果有认证token，测试知识库API
    if [ -n "$AUTH_TOKEN" ] && [ "$AUTH_TOKEN" != "null" ]; then
        test_api "http://localhost:8084/api/v1/documents" "文档列表获取" "GET" "" "$AUTH_TOKEN"
        test_api "http://localhost:8084/api/v1/documents/1" "文档详情获取" "GET" "" "$AUTH_TOKEN"
    else
        echo -e "  ${YELLOW}⚠️ 无认证token，跳过知识库API测试${NC}"
    fi
else
    echo -e "  ${RED}❌ 知识库服务未运行 (端口 8084)${NC}"
    TOTAL_TESTS=$((TOTAL_TESTS + 2))
    FAILED_TESTS=$((FAILED_TESTS + 2))
fi

echo ""
echo -e "${BLUE}🎯 4. 核心功能完整性检查${NC}"

# 检查关键文件是否存在
check_file() {
    local file_path=$1
    local description=$2
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    echo -n "  检查 $description ... "
    
    if [ -f "$file_path" ]; then
        echo -e "${GREEN}✅ 存在${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}❌ 缺失${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

BASE_PATH="/home/jia/Cloud-Based Collaborative Development Platform"

check_file "$BASE_PATH/cmd/project-service/main-auth.go" "认证服务源码"
check_file "$BASE_PATH/cmd/project-service/main-knowledge.go" "知识库服务源码"
check_file "$BASE_PATH/project-service-auth" "认证服务可执行文件"
check_file "$BASE_PATH/project-service-websocket" "知识库服务可执行文件"
check_file "$BASE_PATH/web/product-demo.html" "产品演示页面"
check_file "$BASE_PATH/plan.md" "项目计划文档"

echo ""
echo -e "${BLUE}📊 5. 系统资源使用情况${NC}"

# 检查进程状态
echo "  🔍 运行中的服务进程:"
ps aux | grep -E "(project-service|web-server)" | grep -v grep | while read line; do
    echo "    📈 $line"
done

# 检查端口占用
echo ""
echo "  🔍 端口占用情况:"
for port in 3001 8083 8084; do
    if netstat -tlnp 2>/dev/null | grep ":$port " > /dev/null; then
        echo -e "    ✅ 端口 $port: ${GREEN}已占用${NC}"
    else
        echo -e "    ❌ 端口 $port: ${RED}未占用${NC}"
    fi
done

echo ""
echo "========================================"
echo -e "${BLUE}📋 测试结果汇总${NC}"
echo "========================================"
echo -e "📊 总测试数量: ${BLUE}$TOTAL_TESTS${NC}"
echo -e "✅ 通过测试: ${GREEN}$PASSED_TESTS${NC}"
echo -e "❌ 失败测试: ${RED}$FAILED_TESTS${NC}"

# 计算成功率
if [ $TOTAL_TESTS -gt 0 ]; then
    SUCCESS_RATE=$(( (PASSED_TESTS * 100) / TOTAL_TESTS ))
    echo -e "📈 成功率: ${BLUE}$SUCCESS_RATE%${NC}"
    
    if [ $SUCCESS_RATE -ge 90 ]; then
        echo -e "\n🎉 ${GREEN}系统状态: 优秀! 平台已准备好生产部署${NC}"
    elif [ $SUCCESS_RATE -ge 75 ]; then
        echo -e "\n⚠️  ${YELLOW}系统状态: 良好，建议修复部分问题${NC}"
    else
        echo -e "\n🚨 ${RED}系统状态: 需要修复，存在重要问题${NC}"
    fi
fi

echo ""
echo "🔗 快速访问链接:"
echo "  🏠 主页: http://localhost:3001/"
echo "  🎯 产品演示: http://localhost:3001/product-demo.html"
echo "  🔐 用户登录: http://localhost:3001/login.html"
echo "  📊 项目仪表板: http://localhost:3001/dashboard.html"
echo "  📋 敏捷看板: http://localhost:3001/board.html"
echo "  📚 知识库: http://localhost:3001/knowledge.html"
echo "  🎯 系统概览: http://localhost:3001/demo.html"

echo ""
echo "🛠️  API服务端点:"
echo "  🔐 认证服务: http://localhost:8083/api/v1/health"
echo "  📚 知识库服务: http://localhost:8084/api/v1/health"

echo ""
echo -e "${GREEN}🚀 企业协作开发平台系统测试完成！${NC}"