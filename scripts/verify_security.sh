#!/bin/bash

# 安全功能验证脚本
# 用于验证API限流、IP过滤、安全头部和CSRF防护是否正常工作

echo "🔐 开始安全功能验证..."

BASE_URL="http://localhost:8081/api/v1"
TEST_RESULTS=()

# 颜色代码
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 测试函数
test_security_headers() {
    echo -e "${BLUE}📋 测试1: 安全头部检查${NC}"
    
    response=$(curl -s -I "$BASE_URL/health" 2>/dev/null)
    
    # 检查各种安全头部
    if echo "$response" | grep -q "X-Content-Type-Options: nosniff"; then
        echo -e "  ✅ X-Content-Type-Options 头部存在"
        TEST_RESULTS+=("PASS: X-Content-Type-Options")
    else
        echo -e "  ❌ X-Content-Type-Options 头部缺失"
        TEST_RESULTS+=("FAIL: X-Content-Type-Options")
    fi
    
    if echo "$response" | grep -q "X-Frame-Options: DENY"; then
        echo -e "  ✅ X-Frame-Options 头部存在"
        TEST_RESULTS+=("PASS: X-Frame-Options")
    else
        echo -e "  ❌ X-Frame-Options 头部缺失"
        TEST_RESULTS+=("FAIL: X-Frame-Options")
    fi
    
    if echo "$response" | grep -q "Content-Security-Policy:"; then
        echo -e "  ✅ Content-Security-Policy 头部存在"
        TEST_RESULTS+=("PASS: Content-Security-Policy")
    else
        echo -e "  ❌ Content-Security-Policy 头部缺失"
        TEST_RESULTS+=("FAIL: Content-Security-Policy")
    fi
    
    if echo "$response" | grep -q "Strict-Transport-Security:"; then
        echo -e "  ✅ Strict-Transport-Security 头部存在"
        TEST_RESULTS+=("PASS: Strict-Transport-Security")
    else
        echo -e "  ❌ Strict-Transport-Security 头部缺失"
        TEST_RESULTS+=("FAIL: Strict-Transport-Security")
    fi
    
    echo ""
}

test_csrf_protection() {
    echo -e "${BLUE}🛡️  测试2: CSRF防护检查${NC}"
    
    # 测试GET请求是否返回CSRF令牌
    response=$(curl -s -I "$BASE_URL/health" 2>/dev/null)
    
    if echo "$response" | grep -q "X-CSRF-Token:"; then
        echo -e "  ✅ GET请求返回CSRF令牌"
        TEST_RESULTS+=("PASS: CSRF Token Generation")
        
        # 提取CSRF令牌进行后续测试
        csrf_token=$(echo "$response" | grep "X-CSRF-Token:" | sed 's/X-CSRF-Token: //g' | tr -d '\r\n')
        echo -e "  📝 CSRF Token: ${csrf_token:0:16}..."
    else
        echo -e "  ❌ GET请求未返回CSRF令牌"
        TEST_RESULTS+=("FAIL: CSRF Token Generation")
    fi
    
    # 注意：由于健康检查端点通常不需要CSRF保护，这里主要测试令牌生成
    echo ""
}

test_rate_limiting() {
    echo -e "${BLUE}⚡ 测试3: API限流检查${NC}"
    
    echo -e "  📊 发送10个连续请求测试限流..."
    
    success_count=0
    rate_limited_count=0
    
    for i in {1..10}; do
        response_code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/health" 2>/dev/null)
        
        if [ "$response_code" = "200" ]; then
            success_count=$((success_count + 1))
        elif [ "$response_code" = "429" ]; then
            rate_limited_count=$((rate_limited_count + 1))
            echo -e "  ⏳ 第$i个请求被限流 (HTTP 429)"
        fi
        
        # 小延迟避免过快请求
        sleep 0.1
    done
    
    echo -e "  📈 成功请求: $success_count, 被限流请求: $rate_limited_count"
    
    if [ $success_count -gt 0 ]; then
        echo -e "  ✅ 基础请求可以通过"
        TEST_RESULTS+=("PASS: Basic Rate Limiting")
    else
        echo -e "  ❌ 所有请求都被阻止"
        TEST_RESULTS+=("FAIL: Basic Rate Limiting")
    fi
    
    echo ""
}

test_ip_filtering() {
    echo -e "${BLUE}🌐 测试4: IP过滤检查${NC}"
    
    # 测试正常IP访问
    response_code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/health" 2>/dev/null)
    
    if [ "$response_code" = "200" ]; then
        echo -e "  ✅ 本地IP可以正常访问"
        TEST_RESULTS+=("PASS: IP Filtering - Normal Access")
    else
        echo -e "  ❌ 本地IP访问异常 (HTTP $response_code)"
        TEST_RESULTS+=("FAIL: IP Filtering - Normal Access")
    fi
    
    echo -e "  ℹ️  注意: IP黑名单功能需要管理员手动配置才能测试"
    echo ""
}

check_service_availability() {
    echo -e "${BLUE}🔍 测试0: 服务可用性检查${NC}"
    
    # 检查服务是否运行
    if curl -s "$BASE_URL/health" > /dev/null 2>&1; then
        echo -e "  ✅ 项目服务运行正常"
        return 0
    else
        echo -e "  ❌ 项目服务不可访问"
        echo -e "  💡 请先启动项目服务: go run ./cmd/project-service/"
        return 1
    fi
    echo ""
}

print_summary() {
    echo -e "${YELLOW}📊 测试结果汇总${NC}"
    echo "================================="
    
    pass_count=0
    fail_count=0
    
    for result in "${TEST_RESULTS[@]}"; do
        if [[ $result == PASS:* ]]; then
            echo -e "  ✅ ${result#PASS: }"
            pass_count=$((pass_count + 1))
        else
            echo -e "  ❌ ${result#FAIL: }"
            fail_count=$((fail_count + 1))
        fi
    done
    
    echo "================================="
    echo -e "  总计: ${GREEN}$pass_count 通过${NC}, ${RED}$fail_count 失败${NC}"
    
    if [ $fail_count -eq 0 ]; then
        echo -e "${GREEN}🎉 所有安全功能测试通过！${NC}"
        return 0
    else
        echo -e "${RED}⚠️  部分安全功能测试失败，请检查配置${NC}"
        return 1
    fi
}

# 主测试流程
main() {
    if ! check_service_availability; then
        exit 1
    fi
    
    test_security_headers
    test_csrf_protection
    test_rate_limiting  
    test_ip_filtering
    
    print_summary
    
    echo ""
    echo -e "${BLUE}💡 提示${NC}"
    echo "1. 要测试完整功能，请确保项目服务正在运行"
    echo "2. 可以使用管理API来测试IP黑名单功能"
    echo "3. 生产环境中建议启用更严格的限流和IP过滤"
    echo ""
}

# 如果是直接执行脚本
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi