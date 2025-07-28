#!/bin/bash

# 密钥初始化脚本
# 用于快速设置开发环境的密钥

set -e

echo "🔐 Cloud Platform 密钥设置向导"
echo "================================"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 检查环境
ENVIRONMENT="${ENVIRONMENT:-development}"
echo -e "${BLUE}环境: $ENVIRONMENT${NC}"

# 创建必要的目录
mkdir -p configs/secrets

# 检查 secrets-cli 是否存在
if [ ! -f "./bin/secrets-cli" ]; then
    echo -e "${YELLOW}正在构建 secrets-cli...${NC}"
    go build -o bin/secrets-cli cmd/secrets-cli/main.go
fi

# 函数：读取密码（隐藏输入）
read_password() {
    local prompt=$1
    local var_name=$2
    
    echo -n "$prompt: "
    read -s password
    echo
    
    if [ -z "$password" ]; then
        echo -e "${YELLOW}跳过（使用默认值）${NC}"
        return 1
    fi
    
    eval "$var_name='$password'"
    return 0
}

# 函数：生成随机密钥
generate_secret() {
    openssl rand -base64 32 | tr -d "=+/" | cut -c1-32
}

echo ""
echo "1. 设置加密密钥"
echo "----------------"

if [ -z "$SECRETS_ENCRYPTION_KEY" ]; then
    echo -e "${YELLOW}未检测到加密密钥${NC}"
    echo "是否自动生成加密密钥？(Y/n): "
    read -r response
    
    if [[ "$response" =~ ^([nN][oO]|[nN])$ ]]; then
        read_password "请输入加密密钥（最少32字符）" SECRETS_ENCRYPTION_KEY
    else
        SECRETS_ENCRYPTION_KEY=$(generate_secret)
        echo -e "${GREEN}✅ 已生成加密密钥${NC}"
    fi
    
    export SECRETS_ENCRYPTION_KEY
    echo "export SECRETS_ENCRYPTION_KEY='$SECRETS_ENCRYPTION_KEY'" >> ~/.bashrc
    echo -e "${GREEN}✅ 加密密钥已设置${NC}"
else
    echo -e "${GREEN}✅ 加密密钥已存在${NC}"
fi

echo ""
echo "2. 初始化密钥存储"
echo "------------------"

./bin/secrets-cli init --env "$ENVIRONMENT"

echo ""
echo "3. 设置必需的密钥"
echo "------------------"

# 数据库密码
echo -e "${BLUE}数据库密码${NC}"
if ! ./bin/secrets-cli get database_password &>/dev/null; then
    if [ "$ENVIRONMENT" == "development" ]; then
        echo "使用开发默认密码？(Y/n): "
        read -r response
        if [[ ! "$response" =~ ^([nN][oO]|[nN])$ ]]; then
            ./bin/secrets-cli set database_password "dev_password_123"
        else
            read_password "请输入数据库密码" db_password
            ./bin/secrets-cli set database_password "$db_password"
        fi
    else
        read_password "请输入数据库密码" db_password
        ./bin/secrets-cli set database_password "$db_password"
    fi
else
    echo -e "${GREEN}✅ 数据库密码已配置${NC}"
fi

# JWT密钥
echo -e "${BLUE}JWT密钥${NC}"
if ! ./bin/secrets-cli get jwt_secret &>/dev/null; then
    echo "是否自动生成JWT密钥？(Y/n): "
    read -r response
    
    if [[ "$response" =~ ^([nN][oO]|[nN])$ ]]; then
        read_password "请输入JWT密钥（最少32字符）" jwt_secret
        ./bin/secrets-cli set jwt_secret "$jwt_secret"
    else
        jwt_secret=$(generate_secret)$(generate_secret)  # 64字符
        ./bin/secrets-cli set jwt_secret "$jwt_secret"
        echo -e "${GREEN}✅ 已生成JWT密钥${NC}"
    fi
else
    echo -e "${GREEN}✅ JWT密钥已配置${NC}"
fi

# Redis密码（可选）
echo -e "${BLUE}Redis密码（可选）${NC}"
if ! ./bin/secrets-cli get redis_password &>/dev/null; then
    echo "是否设置Redis密码？(y/N): "
    read -r response
    
    if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
        read_password "请输入Redis密码" redis_password
        ./bin/secrets-cli set redis_password "$redis_password"
    else
        echo -e "${YELLOW}跳过Redis密码${NC}"
    fi
else
    echo -e "${GREEN}✅ Redis密码已配置${NC}"
fi

echo ""
echo "4. 验证配置"
echo "------------"

./bin/secrets-cli validate

echo ""
echo "5. 生成环境变量文件"
echo "--------------------"

if [ ! -f ".env" ]; then
    echo "是否生成 .env 文件？(Y/n): "
    read -r response
    
    if [[ ! "$response" =~ ^([nN][oO]|[nN])$ ]]; then
        cp configs/.env.example .env
        echo -e "${GREEN}✅ 已创建 .env 文件${NC}"
        echo -e "${YELLOW}请编辑 .env 文件并填入实际的配置值${NC}"
    fi
else
    echo -e "${GREEN}✅ .env 文件已存在${NC}"
fi

echo ""
echo "================================"
echo -e "${GREEN}✅ 密钥设置完成！${NC}"
echo ""
echo "后续步骤："
echo "1. 编辑 .env 文件（如果需要）"
echo "2. 运行 'source ~/.bashrc' 以应用环境变量"
echo "3. 使用 './bin/secrets-cli list' 查看所有密钥"
echo "4. 使用 './bin/secrets-cli rotate <key>' 轮换密钥"
echo ""
echo -e "${YELLOW}注意：请确保 configs/secrets/ 目录不被提交到版本控制${NC}"