#!/bin/bash

# Cloud-Based Collaborative Development Platform - 测试运行脚本
# 该脚本单独运行每个测试文件以避免命名冲突

set -e

# 定义颜色
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "============================================"
echo "🚀 开始运行单元测试"
echo "============================================"

# 切换到项目目录
cd "$(dirname "$0")"

# 创建临时目录用于测试输出
TEMP_DIR=$(mktemp -d)
echo "📁 临时输出目录: $TEMP_DIR"

# 测试文件列表
TEST_FILES=(
    "test/unit/project_validation_test.go"
    "test/unit/git_gateway_test.go"
    "test/unit/tenant_service_test.go"
    "test/unit/iam_service_test.go"
    "test/unit/notification_service_test.go"
    "test/unit/cicd_service_test.go"
    "test/unit/file_service_test.go"
    "test/unit/team_service_test.go"
    "test/unit/knowledge_base_service_test.go"
)

# 统计数据
TOTAL=0
PASSED=0
FAILED=0

# 单独运行每个测试文件
for test_file in "${TEST_FILES[@]}"; do
    if [[ -f "$test_file" ]]; then
        echo -e "\n${YELLOW}📋 正在运行: $test_file${NC}"
        
        # 创建临时测试目录
        TEST_DIR="$TEMP_DIR/$(basename $test_file .go)"
        mkdir -p "$TEST_DIR"
        
        # 复制测试文件到临时目录
        cp "$test_file" "$TEST_DIR/"
        
        # 运行测试
        if go test -v -coverprofile="$TEST_DIR/coverage.out" "./$TEST_DIR" 2>&1 | tee "$TEST_DIR/test.log"; then
            echo -e "${GREEN}✅ 通过${NC}"
            ((PASSED++))
        else
            echo -e "${RED}❌ 失败${NC}"
            ((FAILED++))
        fi
        ((TOTAL++))
    else
        echo -e "${RED}❌ 文件不存在: $test_file${NC}"
    fi
done

# 运行集成测试
echo -e "\n${YELLOW}📋 正在运行集成测试${NC}"
if go test -v ./test/integration/... 2>&1 | tee "$TEMP_DIR/integration_test.log"; then
    echo -e "${GREEN}✅ 集成测试通过${NC}"
    ((PASSED++))
else
    echo -e "${RED}❌ 集成测试失败${NC}"
    ((FAILED++))
fi
((TOTAL++))

# 输出测试摘要
echo -e "\n============================================"
echo "📊 测试摘要"
echo "============================================"
echo -e "总测试文件数: ${TOTAL}"
echo -e "通过: ${GREEN}${PASSED}${NC}"
echo -e "失败: ${RED}${FAILED}${NC}"
echo -e "成功率: $(( PASSED * 100 / TOTAL ))%"

# 清理临时目录
rm -rf "$TEMP_DIR"

# 根据结果返回状态码
if [[ $FAILED -gt 0 ]]; then
    echo -e "\n${RED}❌ 测试失败！${NC}"
    exit 1
else
    echo -e "\n${GREEN}✅ 所有测试通过！${NC}"
    exit 0
fi