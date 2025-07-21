#!/bin/bash

# Git Gateway Service 基本功能测试脚本

set -e

echo "🚀 开始Git Gateway Service基本功能测试"

# 服务配置
SERVICE_URL="http://localhost:8083"
API_BASE="$SERVICE_URL/api/v1"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查服务是否运行
check_service() {
    log_info "检查Git Gateway Service是否运行..."
    
    if curl -s "$API_BASE/health" > /dev/null; then
        log_info "✅ Git Gateway Service运行正常"
        
        # 获取服务信息
        health_response=$(curl -s "$API_BASE/health")
        echo "服务信息: $health_response"
    else
        log_error "❌ Git Gateway Service未运行"
        log_info "请先启动服务: ./build/git-gateway-service"
        exit 1
    fi
}

# 生成JWT Token（简化版，实际需要从IAM服务获取）
generate_jwt_token() {
    # 这里使用一个简单的JWT Token进行测试
    # 实际应用中需要从IAM服务获取有效的JWT Token
    JWT_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiZTNmYTg1ZGItZTRiYy00ZGY3LWIzYWEtOTVkYjE2ZWJkMzQ5IiwiZXhwIjo5OTk5OTk5OTk5fQ.mock_signature"
    log_info "使用测试JWT Token: ${JWT_TOKEN:0:50}..."
}

# 测试仓库管理功能
test_repository_management() {
    log_info "🧪 测试仓库管理功能..."
    
    # 模拟项目ID
    PROJECT_ID="e3fa85db-e4bc-4df7-b3aa-95db16ebd349"
    
    # 1. 创建仓库
    log_info "1️⃣ 测试创建仓库..."
    
    create_payload='{
        "project_id": "'$PROJECT_ID'",
        "name": "test-repo-'$(date +%s)'",
        "description": "测试仓库",
        "visibility": "private",
        "default_branch": "main",
        "init_readme": true
    }'
    
    create_response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -d "$create_payload" \
        "$API_BASE/repositories")
    
    if echo "$create_response" | grep -q "Repository created successfully"; then
        log_info "✅ 仓库创建成功"
        
        # 提取仓库ID
        REPO_ID=$(echo "$create_response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
        log_info "仓库ID: $REPO_ID"
    else
        log_error "❌ 仓库创建失败"
        echo "响应: $create_response"
        return 1
    fi
    
    # 2. 获取仓库详情
    log_info "2️⃣ 测试获取仓库详情..."
    
    get_response=$(curl -s -H "Authorization: Bearer $JWT_TOKEN" \
        "$API_BASE/repositories/$REPO_ID")
    
    if echo "$get_response" | grep -q "Repository retrieved successfully"; then
        log_info "✅ 仓库详情获取成功"
    else
        log_error "❌ 仓库详情获取失败"
        echo "响应: $get_response"
    fi
    
    # 3. 获取仓库列表
    log_info "3️⃣ 测试获取仓库列表..."
    
    list_response=$(curl -s -H "Authorization: Bearer $JWT_TOKEN" \
        "$API_BASE/repositories?project_id=$PROJECT_ID&page=1&page_size=10")
    
    if echo "$list_response" | grep -q "Repositories retrieved successfully"; then
        log_info "✅ 仓库列表获取成功"
    else
        log_error "❌ 仓库列表获取失败"
        echo "响应: $list_response"
    fi
    
    # 4. 更新仓库
    log_info "4️⃣ 测试更新仓库..."
    
    update_payload='{
        "description": "更新后的测试仓库描述",
        "visibility": "public"
    }'
    
    update_response=$(curl -s -X PUT \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -d "$update_payload" \
        "$API_BASE/repositories/$REPO_ID")
    
    if echo "$update_response" | grep -q "Repository updated successfully"; then
        log_info "✅ 仓库更新成功"
    else
        log_error "❌ 仓库更新失败"
        echo "响应: $update_response"
    fi
    
    # 5. 删除仓库
    log_info "5️⃣ 测试删除仓库..."
    
    delete_response=$(curl -s -X DELETE \
        -H "Authorization: Bearer $JWT_TOKEN" \
        "$API_BASE/repositories/$REPO_ID")
    
    if echo "$delete_response" | grep -q "Repository deleted successfully"; then
        log_info "✅ 仓库删除成功"
    else
        log_error "❌ 仓库删除失败"
        echo "响应: $delete_response"
    fi
}

# 测试分支管理功能
test_branch_management() {
    log_info "🌿 测试分支管理功能..."
    
    # 模拟项目ID
    PROJECT_ID="e3fa85db-e4bc-4df7-b3aa-95db16ebd349"
    
    # 1. 创建一个测试仓库用于分支测试
    log_info "1️⃣ 创建测试仓库用于分支测试..."
    
    create_payload='{ 
        "project_id": "'$PROJECT_ID'",
        "name": "branch-test-repo-'$(date +%s)'",
        "description": "分支测试仓库",
        "visibility": "private",
        "default_branch": "main",
        "init_readme": true
    }'
    
    create_response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -d "$create_payload" \
        "$API_BASE/repositories")
    
    if echo "$create_response" | grep -q "Repository created successfully"; then
        log_info "✅ 测试仓库创建成功"
        
        # 提取仓库ID
        REPO_ID=$(echo "$create_response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
        log_info "仓库ID: $REPO_ID"
        
        # 2. 测试创建分支
        log_info "2️⃣ 测试创建分支..."
        
        branch_payload='{
            "name": "feature-test",
            "from_sha": "main"
        }'
        
        branch_response=$(curl -s -X POST \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $JWT_TOKEN" \
            -d "$branch_payload" \
            "$API_BASE/repositories/$REPO_ID/branches")
        
        if echo "$branch_response" | grep -q "Branch created successfully"; then
            log_info "✅ 分支创建成功"
        else
            log_error "❌ 分支创建失败"
            echo "响应: $branch_response"
        fi
        
        # 3. 测试获取分支列表
        log_info "3️⃣ 测试获取分支列表..."
        
        list_response=$(curl -s -H "Authorization: Bearer $JWT_TOKEN" \
            "$API_BASE/repositories/$REPO_ID/branches")
        
        if echo "$list_response" | grep -q "Branches retrieved successfully"; then
            log_info "✅ 分支列表获取成功"
        else
            log_error "❌ 分支列表获取失败"
            echo "响应: $list_response"
        fi
        
        # 4. 测试分支合并
        log_info "4️⃣ 测试分支合并..."
        
        merge_payload='{
            "target_branch": "main",
            "source_branch": "feature-test"
        }'
        
        merge_response=$(curl -s -X POST \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $JWT_TOKEN" \
            -d "$merge_payload" \
            "$API_BASE/repositories/$REPO_ID/merge")
        
        if echo "$merge_response" | grep -q "Branch merged successfully"; then
            log_info "✅ 分支合并成功"
        else
            log_error "❌ 分支合并失败"
            echo "响应: $merge_response"
        fi
        
        # 5. 清理：删除测试仓库
        log_info "5️⃣ 清理测试仓库..."
        
        delete_response=$(curl -s -X DELETE \
            -H "Authorization: Bearer $JWT_TOKEN" \
            "$API_BASE/repositories/$REPO_ID")
        
        if echo "$delete_response" | grep -q "Repository deleted successfully"; then
            log_info "✅ 测试仓库清理成功"
        else
            log_warn "⚠️ 测试仓库清理失败"
            echo "响应: $delete_response"
        fi
        
    else
        log_error "❌ 测试仓库创建失败"
        echo "响应: $create_response"
        log_info "将使用假仓库ID测试API端点..."
        
        # 使用假的仓库ID测试API端点
        fake_repo_id="00000000-0000-0000-0000-000000000000"
        
        branch_response=$(curl -s -H "Authorization: Bearer $JWT_TOKEN" \
            "$API_BASE/repositories/$fake_repo_id/branches")
        
        if echo "$branch_response" | grep -q "Repository not found\|Branches retrieved successfully"; then
            log_info "✅ 分支管理API端点响应正常"
        else
            log_error "❌ 分支管理API端点异常"
            echo "响应: $branch_response"
        fi
    fi
}

# 测试提交管理功能
test_commit_management() {
    log_info "📝 测试提交管理功能..."
    
    # 模拟项目ID
    PROJECT_ID="e3fa85db-e4bc-4df7-b3aa-95db16ebd349"
    
    # 1. 创建一个测试仓库用于提交测试
    log_info "1️⃣ 创建测试仓库用于提交测试..."
    
    create_payload='{ 
        "project_id": "'$PROJECT_ID'",
        "name": "commit-test-repo-'$(date +%s)'",
        "description": "提交测试仓库",
        "visibility": "private",
        "default_branch": "main",
        "init_readme": true
    }'
    
    create_response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -d "$create_payload" \
        "$API_BASE/repositories")
    
    if echo "$create_response" | grep -q "Repository created successfully"; then
        log_info "✅ 测试仓库创建成功"
        
        # 提取仓库ID
        REPO_ID=$(echo "$create_response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
        log_info "仓库ID: $REPO_ID"
        
        # 2. 测试创建提交
        log_info "2️⃣ 测试创建提交..."
        
        commit_payload='{
            "branch": "main",
            "message": "feat: 添加新功能文件",
            "author": {
                "name": "Test User",
                "email": "test@example.com"
            },
            "files": [
                {
                    "path": "src/main.go",
                    "content": "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}\n"
                },
                {
                    "path": "README.md",
                    "content": "# 测试项目\n\n这是一个测试项目。\n"
                }
            ]
        }'
        
        commit_response=$(curl -s -X POST \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $JWT_TOKEN" \
            -d "$commit_payload" \
            "$API_BASE/repositories/$REPO_ID/commits")
        
        if echo "$commit_response" | grep -q "Commit created successfully"; then
            log_info "✅ 提交创建成功"
            
            # 提取提交SHA
            COMMIT_SHA=$(echo "$commit_response" | grep -o '"sha":"[^"]*"' | cut -d'"' -f4)
            log_info "提交SHA: $COMMIT_SHA"
            
            # 3. 测试获取提交详情
            log_info "3️⃣ 测试获取提交详情..."
            
            get_commit_response=$(curl -s -H "Authorization: Bearer $JWT_TOKEN" \
                "$API_BASE/repositories/$REPO_ID/commits/$COMMIT_SHA")
            
            if echo "$get_commit_response" | grep -q "Commit retrieved successfully"; then
                log_info "✅ 提交详情获取成功"
            else
                log_error "❌ 提交详情获取失败"
                echo "响应: $get_commit_response"
            fi
            
            # 4. 测试获取提交列表
            log_info "4️⃣ 测试获取提交列表..."
            
            list_commits_response=$(curl -s -H "Authorization: Bearer $JWT_TOKEN" \
                "$API_BASE/repositories/$REPO_ID/commits?branch=main&page=1&page_size=10")
            
            if echo "$list_commits_response" | grep -q "Commits retrieved successfully"; then
                log_info "✅ 提交列表获取成功"
            else
                log_error "❌ 提交列表获取失败"
                echo "响应: $list_commits_response"
            fi
            
            # 5. 测试获取提交差异
            log_info "5️⃣ 测试获取提交差异..."
            
            diff_response=$(curl -s -H "Authorization: Bearer $JWT_TOKEN" \
                "$API_BASE/repositories/$REPO_ID/commits/$COMMIT_SHA/diff")
            
            if echo "$diff_response" | grep -q "Commit diff retrieved successfully"; then
                log_info "✅ 提交差异获取成功"
            else
                log_error "❌ 提交差异获取失败"
                echo "响应: $diff_response"
            fi
            
        else
            log_error "❌ 提交创建失败"
            echo "响应: $commit_response"
        fi
        
        # 6. 清理：删除测试仓库
        log_info "6️⃣ 清理测试仓库..."
        
        delete_response=$(curl -s -X DELETE \
            -H "Authorization: Bearer $JWT_TOKEN" \
            "$API_BASE/repositories/$REPO_ID")
        
        if echo "$delete_response" | grep -q "Repository deleted successfully"; then
            log_info "✅ 测试仓库清理成功"
        else
            log_warn "⚠️ 测试仓库清理失败"
            echo "响应: $delete_response"
        fi
        
    else
        log_error "❌ 测试仓库创建失败"
        echo "响应: $create_response"
        log_info "将使用假仓库ID测试API端点..."
        
        # 使用假的仓库ID和提交SHA测试API端点
        fake_repo_id="00000000-0000-0000-0000-000000000000"
        fake_commit_sha="1234567890123456789012345678901234567890"
        
        commits_response=$(curl -s -H "Authorization: Bearer $JWT_TOKEN" \
            "$API_BASE/repositories/$fake_repo_id/commits")
        
        if echo "$commits_response" | grep -q "Repository not found\|Commits retrieved successfully"; then
            log_info "✅ 提交管理API端点响应正常"
        else
            log_error "❌ 提交管理API端点异常"
            echo "响应: $commits_response"
        fi
    fi
}

# 测试统计功能
test_statistics() {
    log_info "📊 测试统计功能..."
    
    fake_repo_id="00000000-0000-0000-0000-000000000000"
    
    stats_response=$(curl -s -H "Authorization: Bearer $JWT_TOKEN" \
        "$API_BASE/repositories/$fake_repo_id/stats")
    
    if echo "$stats_response" | grep -q "Repository stats retrieved successfully\|Repository not found"; then
        log_info "✅ 统计功能API端点响应正常"
    else
        log_error "❌ 统计功能API端点异常"
        echo "响应: $stats_response"
    fi
}

# 测试搜索功能
test_search() {
    log_info "🔍 测试搜索功能..."
    
    search_response=$(curl -s -H "Authorization: Bearer $JWT_TOKEN" \
        "$API_BASE/repositories/search?q=test&page=1&page_size=10")
    
    if echo "$search_response" | grep -q "Repository search completed successfully"; then
        log_info "✅ 搜索功能正常"
    else
        log_error "❌ 搜索功能异常"
        echo "响应: $search_response"
    fi
}

# 性能测试
performance_test() {
    log_info "⚡ 基本性能测试..."
    
    start_time=$(date +%s%N)
    
    for i in {1..5}; do
        curl -s "$API_BASE/health" > /dev/null
    done
    
    end_time=$(date +%s%N)
    duration=$((($end_time - $start_time) / 1000000))
    
    avg_response_time=$((duration / 5))
    
    log_info "5次健康检查平均响应时间: ${avg_response_time}ms"
    
    if [ $avg_response_time -lt 100 ]; then
        log_info "✅ 响应时间良好"
    elif [ $avg_response_time -lt 500 ]; then
        log_warn "⚠️ 响应时间一般"
    else
        log_error "❌ 响应时间较慢"
    fi
}

# 主测试流程
main() {
    echo "======================================"
    echo "Git Gateway Service 功能测试"
    echo "======================================"
    echo ""
    
    # 检查必要的工具
    if ! command -v curl &> /dev/null; then
        log_error "curl命令未找到，请安装curl"
        exit 1
    fi
    
    # 生成JWT Token
    generate_jwt_token
    
    # 运行测试
    check_service
    echo ""
    
    test_repository_management
    echo ""
    
    test_branch_management
    echo ""
    
    test_commit_management
    echo ""
    
    test_statistics
    echo ""
    
    test_search
    echo ""
    
    performance_test
    echo ""
    
    echo "======================================"
    log_info "🎉 Git Gateway Service测试完成！"
    echo "======================================"
}

# 运行主测试
main "$@"