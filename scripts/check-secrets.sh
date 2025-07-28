#!/bin/bash

# 密钥安全检查脚本
# 用于在提交前检查是否有硬编码的密钥

set -e

echo "🔍 检查硬编码密钥..."

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查的文件类型
FILE_PATTERNS=(
    "*.go"
    "*.yaml"
    "*.yml"
    "*.json"
    "*.env*"
    "docker-compose*.yml"
)

# 危险的模式
DANGEROUS_PATTERNS=(
    # 密码模式
    "password.*=.*['\"].*['\"]"
    "passwd.*=.*['\"].*['\"]"
    "pwd.*=.*['\"].*['\"]"
    
    # 密钥模式
    "secret.*=.*['\"].*['\"]"
    "key.*=.*['\"].*['\"]"
    "token.*=.*['\"].*['\"]"
    "api_key.*=.*['\"].*['\"]"
    
    # 特定的硬编码值
    "password.*:.*admin"
    "password.*:.*123"
    "password.*:.*test"
    "password.*:.*demo"
    "secret.*:.*development"
    "jwt.*:.*secret"
    
    # Base64 编码的密钥
    "^[A-Za-z0-9+/]{40,}={0,2}$"
)

# 白名单文件
WHITELIST_FILES=(
    "check-secrets.sh"
    "secret-management.md"
    "*.test.go"
    "*_test.go"
    "test-*.go"
)

# 函数：检查文件是否在白名单中
is_whitelisted() {
    local file=$1
    for pattern in "${WHITELIST_FILES[@]}"; do
        if [[ "$file" == $pattern ]] || [[ "$file" == *"/$pattern" ]]; then
            return 0
        fi
    done
    return 1
}

# 函数：检查单个文件
check_file() {
    local file=$1
    local found_issues=0
    
    # 跳过白名单文件
    if is_whitelisted "$file"; then
        return 0
    fi
    
    # 检查每个危险模式
    for pattern in "${DANGEROUS_PATTERNS[@]}"; do
        if grep -qiE "$pattern" "$file" 2>/dev/null; then
            echo -e "${RED}❌ 发现潜在的硬编码密钥在文件: $file${NC}"
            grep -niE "$pattern" "$file" | head -5
            found_issues=1
        fi
    done
    
    return $found_issues
}

# 主检查逻辑
total_issues=0

echo "检查的文件类型: ${FILE_PATTERNS[*]}"

for pattern in "${FILE_PATTERNS[@]}"; do
    while IFS= read -r -d '' file; do
        if check_file "$file"; then
            ((total_issues++))
        fi
    done < <(find . -name "$pattern" -type f -not -path "*/vendor/*" -not -path "*/.git/*" -not -path "*/node_modules/*" -print0)
done

# 检查 .env 文件是否被错误提交
if [ -f ".env" ]; then
    echo -e "${RED}❌ 警告: .env 文件存在于项目中！${NC}"
    echo "   .env 文件不应该被提交到版本控制"
    ((total_issues++))
fi

# 检查特定的配置文件
echo ""
echo "🔍 检查配置文件..."

# 检查 docker-compose.yml
if [ -f "docker-compose.yml" ]; then
    if grep -q "PASSWORD.*=.*[^$\{]" docker-compose.yml; then
        echo -e "${YELLOW}⚠️  docker-compose.yml 可能包含硬编码密码${NC}"
        echo "   建议使用环境变量: \${DATABASE_PASSWORD:-default_value}"
    fi
fi

# 结果汇总
echo ""
if [ $total_issues -eq 0 ]; then
    echo -e "${GREEN}✅ 没有发现硬编码的密钥！${NC}"
    exit 0
else
    echo -e "${RED}❌ 发现 $total_issues 个潜在的安全问题${NC}"
    echo ""
    echo "建议的修复方法:"
    echo "1. 使用环境变量存储敏感信息"
    echo "2. 使用密钥管理器 (secrets-cli)"
    echo "3. 确保 .env 文件在 .gitignore 中"
    echo "4. 对于测试文件，使用模拟值而不是真实密码"
    echo ""
    echo "运行以下命令初始化密钥管理:"
    echo "  ./bin/secrets-cli init"
    exit 1
fi