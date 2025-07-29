#!/bin/bash

# Project Service Integration Test Runner
# 项目服务集成测试执行脚本

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查依赖
check_dependencies() {
    log_info "检查依赖环境..."
    
    # 检查Go版本
    if ! command -v go &> /dev/null; then
        log_error "Go未安装"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    log_info "Go版本: $GO_VERSION"
    
    # 检查PostgreSQL
    if ! command -v psql &> /dev/null; then
        log_warning "PostgreSQL客户端未安装，跳过数据库连接测试"
    else
        log_info "PostgreSQL客户端已安装"
    fi
    
    # 检查Docker（可选）
    if command -v docker &> /dev/null; then
        log_info "Docker已安装"
    else
        log_warning "Docker未安装，跳过容器化测试"
    fi
}

# 设置测试环境
setup_test_environment() {
    log_info "设置测试环境..."
    
    # 设置环境变量
    export ENVIRONMENT=test
    export TEST_DB_HOST=${TEST_DB_HOST:-localhost}
    export TEST_DB_PORT=${TEST_DB_PORT:-5432}
    export TEST_DB_NAME=${TEST_DB_NAME:-collaborative_dev_test}
    export TEST_DB_USER=${TEST_DB_USER:-postgres}
    export TEST_DB_PASSWORD=${TEST_DB_PASSWORD:-postgres}
    export TEST_DB_SSLMODE=${TEST_DB_SSLMODE:-disable}
    export TEST_LOG_LEVEL=${TEST_LOG_LEVEL:-debug}
    export TEST_TIMEOUT=${TEST_TIMEOUT:-60s}
    
    log_info "测试数据库: $TEST_DB_HOST:$TEST_DB_PORT/$TEST_DB_NAME"
    log_info "测试超时: $TEST_TIMEOUT"
}

# 检查数据库连接
check_database_connection() {
    log_info "检查数据库连接..."
    
    if command -v psql &> /dev/null; then
        # 使用psql测试连接
        PGPASSWORD=$TEST_DB_PASSWORD psql -h $TEST_DB_HOST -p $TEST_DB_PORT -U $TEST_DB_USER -d postgres -c "SELECT 1;" > /dev/null 2>&1
        
        if [ $? -eq 0 ]; then
            log_success "数据库连接正常"
        else
            log_error "数据库连接失败"
            log_info "请确保PostgreSQL服务正在运行，并且连接参数正确"
            exit 1
        fi
    else
        log_warning "跳过数据库连接检查（psql未安装）"
    fi
}

# 创建测试数据库
create_test_database() {
    log_info "创建测试数据库..."
    
    if command -v psql &> /dev/null; then
        # 检查数据库是否存在
        PGPASSWORD=$TEST_DB_PASSWORD psql -h $TEST_DB_HOST -p $TEST_DB_PORT -U $TEST_DB_USER -d postgres -tc "SELECT 1 FROM pg_database WHERE datname = '$TEST_DB_NAME';" | grep -q 1
        
        if [ $? -eq 0 ]; then
            log_info "测试数据库已存在，删除并重新创建..."
            PGPASSWORD=$TEST_DB_PASSWORD psql -h $TEST_DB_HOST -p $TEST_DB_PORT -U $TEST_DB_USER -d postgres -c "DROP DATABASE IF EXISTS $TEST_DB_NAME;"
        fi
        
        # 创建测试数据库
        PGPASSWORD=$TEST_DB_PASSWORD psql -h $TEST_DB_HOST -p $TEST_DB_PORT -U $TEST_DB_USER -d postgres -c "CREATE DATABASE $TEST_DB_NAME;"
        
        if [ $? -eq 0 ]; then
            log_success "测试数据库创建成功"
        else
            log_error "测试数据库创建失败"
            exit 1
        fi
    else
        log_warning "跳过数据库创建（psql未安装）"
    fi
}

# 运行单元测试
run_unit_tests() {
    log_info "运行单元测试..."
    
    # 进入项目根目录
    cd "$(dirname "$0")/.."
    
    # 运行单元测试
    go test -v -race -timeout=$TEST_TIMEOUT ./test/unit/... 2>&1 | tee test_results_unit.log
    
    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        log_success "单元测试通过"
    else
        log_error "单元测试失败"
        return 1
    fi
}

# 运行集成测试
run_integration_tests() {
    log_info "运行集成测试..."
    
    # 进入项目根目录
    cd "$(dirname "$0")/.."
    
    # 运行集成测试（带覆盖率）
    go test -v -race -timeout=$TEST_TIMEOUT -coverprofile=integration_coverage.out ./test/integration/... 2>&1 | tee test_results_integration.log
    
    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        log_success "集成测试通过"
        
        # 生成覆盖率报告
        if [ -f integration_coverage.out ]; then
            go tool cover -html=integration_coverage.out -o integration_coverage.html
            log_info "集成测试覆盖率报告已生成: integration_coverage.html"
            
            # 显示覆盖率统计
            COVERAGE=$(go tool cover -func=integration_coverage.out | grep total | awk '{print $3}')
            log_info "集成测试覆盖率: $COVERAGE"
        fi
    else
        log_error "集成测试失败"
        return 1
    fi
}

# 运行性能测试
run_performance_tests() {
    log_info "运行性能测试..."
    
    # 进入项目根目录
    cd "$(dirname "$0")/.."
    
    # 运行基准测试
    go test -v -bench=. -benchmem -timeout=$TEST_TIMEOUT ./test/integration/... 2>&1 | tee test_results_benchmark.log
    
    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        log_success "性能测试完成"
    else
        log_warning "性能测试过程中有问题"
        return 1
    fi
}

# 生成测试报告
generate_test_report() {
    log_info "生成测试报告..."
    
    cd "$(dirname "$0")/.."
    
    # 创建测试报告目录
    mkdir -p test-report
    
    # 生成综合测试报告
    cat > test-report/integration_test_report.md << EOF
# Project Service Integration Test Report
# 项目服务集成测试报告

## 测试概述

- **测试时间**: $(date)
- **测试环境**: $ENVIRONMENT
- **数据库**: $TEST_DB_HOST:$TEST_DB_PORT/$TEST_DB_NAME
- **Go版本**: $(go version)

## 测试结果

### 单元测试
$(if [ -f test_results_unit.log ]; then
    if grep -q "PASS" test_results_unit.log; then
        echo "✅ **通过**"
    else
        echo "❌ **失败**"
    fi
else
    echo "⚠️ **未运行**"  
fi)

### 集成测试
$(if [ -f test_results_integration.log ]; then
    if grep -q "PASS" test_results_integration.log; then
        echo "✅ **通过**"
    else
        echo "❌ **失败**"
    fi
else
    echo "⚠️ **未运行**"
fi)

### 性能测试
$(if [ -f test_results_benchmark.log ]; then
    echo "✅ **完成**"
else
    echo "⚠️ **未运行**"
fi)

## 测试覆盖率

$(if [ -f integration_coverage.out ]; then
    go tool cover -func=integration_coverage.out | tail -n 20
else
    echo "覆盖率报告未生成"
fi)

## 详细日志

### 单元测试日志
$(if [ -f test_results_unit.log ]; then
    echo "\`\`\`"
    tail -n 50 test_results_unit.log
    echo "\`\`\`"
else
    echo "无单元测试日志"
fi)

### 集成测试日志
$(if [ -f test_results_integration.log ]; then
    echo "\`\`\`"
    tail -n 50 test_results_integration.log
    echo "\`\`\`"
else
    echo "无集成测试日志"
fi)

### 性能测试结果
$(if [ -f test_results_benchmark.log ]; then
    echo "\`\`\`"
    grep -E "(Benchmark|ns/op|B/op|allocs/op)" test_results_benchmark.log | tail -n 20
    echo "\`\`\`"
else
    echo "无性能测试结果"
fi)

---
*自动生成于 $(date)*
EOF

    log_success "测试报告已生成: test-report/integration_test_report.md"
}

# 清理测试环境
cleanup_test_environment() {
    log_info "清理测试环境..."
    
    cd "$(dirname "$0")/.."
    
    # 移动日志文件到报告目录
    if [ -d test-report ]; then
        mv test_results_*.log test-report/ 2>/dev/null || true
        mv *_coverage.out test-report/ 2>/dev/null || true
        mv *_coverage.html test-report/ 2>/dev/null || true
    fi
    
    # 可选：删除测试数据库
    if [ "$CLEANUP_TEST_DB" = "true" ] && command -v psql &> /dev/null; then
        log_info "删除测试数据库..."
        PGPASSWORD=$TEST_DB_PASSWORD psql -h $TEST_DB_HOST -p $TEST_DB_PORT -U $TEST_DB_USER -d postgres -c "DROP DATABASE IF EXISTS $TEST_DB_NAME;" > /dev/null 2>&1
        log_success "测试数据库已删除"
    fi
}

# 主函数
main() {
    log_info "启动项目服务集成测试..."
    
    # 解析命令行参数
    SKIP_UNIT=false
    SKIP_INTEGRATION=false
    SKIP_PERFORMANCE=false
    CLEANUP_TEST_DB=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --skip-unit)
                SKIP_UNIT=true
                shift
                ;;
            --skip-integration)
                SKIP_INTEGRATION=true
                shift
                ;;
            --skip-performance)
                SKIP_PERFORMANCE=true
                shift
                ;;
            --cleanup-db)
                CLEANUP_TEST_DB=true
                shift
                ;;
            --help)
                echo "用法: $0 [选项]"
                echo "选项:"
                echo "  --skip-unit         跳过单元测试"
                echo "  --skip-integration  跳过集成测试"
                echo "  --skip-performance  跳过性能测试"
                echo "  --cleanup-db        测试后删除测试数据库"
                echo "  --help              显示帮助信息"
                exit 0
                ;;
            *)
                log_error "未知选项: $1"
                exit 1
                ;;
        esac
    done
    
    # 执行测试流程
    check_dependencies
    setup_test_environment
    check_database_connection
    create_test_database
    
    local exit_code=0
    
    # 运行测试
    if [ "$SKIP_UNIT" = "false" ]; then
        run_unit_tests || exit_code=1
    fi
    
    if [ "$SKIP_INTEGRATION" = "false" ]; then
        run_integration_tests || exit_code=1
    fi
    
    if [ "$SKIP_PERFORMANCE" = "false" ]; then
        run_performance_tests || true  # 性能测试失败不影响整体结果
    fi
    
    # 生成报告和清理
    generate_test_report
    cleanup_test_environment
    
    # 最终结果
    if [ $exit_code -eq 0 ]; then
        log_success "所有测试通过！"
        echo
        log_info "测试报告位置: test-report/"
        log_info "覆盖率报告: test-report/integration_coverage.html"
    else
        log_error "部分测试失败！"
        echo
        log_info "请查看测试报告了解详情: test-report/integration_test_report.md"
        exit 1
    fi
}

# 捕获中断信号
trap cleanup_test_environment EXIT

# 执行主函数
main "$@"