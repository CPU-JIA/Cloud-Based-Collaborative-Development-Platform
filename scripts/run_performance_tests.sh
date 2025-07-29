#!/bin/bash

# Project Service Performance Test Runner
# 项目服务性能测试运行脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# 配置文件路径
PERFORMANCE_CONFIG="$PROJECT_ROOT/test/performance/performance_config.yaml"
TEST_REPORT_DIR="$PROJECT_ROOT/test-report"

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

# 显示帮助信息
show_help() {
    cat << EOF
Project Service Performance Test Runner

用法: $0 [选项]

选项:
    -h, --help              显示此帮助信息
    -c, --config FILE       指定配置文件 (默认: test/performance/performance_config.yaml)
    -o, --output DIR        指定输出目录 (默认: test-report)
    -t, --test NAME         运行特定测试 (crud|concurrent|list|mixed|stress|all)
    -u, --users NUM         并发用户数 (覆盖配置文件设置)
    -d, --duration TIME     测试持续时间 (覆盖配置文件设置)
    -v, --verbose           详细输出
    --cleanup               测试后清理数据
    --no-setup              跳过环境设置
    --profile               启用性能分析
    --benchmark             运行基准测试

示例:
    $0                              # 运行所有性能测试
    $0 -t crud                      # 只运行CRUD性能测试
    $0 -t concurrent -u 100 -d 60s  # 运行并发测试，100用户，60秒
    $0 --profile --benchmark        # 运行基准测试并启用性能分析

EOF
}

# 检查依赖
check_dependencies() {
    log_info "检查依赖..."
    
    # 检查Go
    if ! command -v go &> /dev/null; then
        log_error "Go未安装或不在PATH中"
        exit 1
    fi
    
    # 检查PostgreSQL客户端
    if ! command -v psql &> /dev/null; then
        log_warning "PostgreSQL客户端未安装，无法验证数据库连接"
    fi
    
    # 检查必要的Go包
    cd "$PROJECT_ROOT"
    if ! go mod verify &> /dev/null; then
        log_error "Go模块验证失败"
        exit 1
    fi
    
    log_success "依赖检查完成"
}

# 设置测试环境
setup_test_environment() {
    log_info "设置测试环境..."
    
    # 创建报告目录
    mkdir -p "$TEST_REPORT_DIR"
    
    # 设置环境变量
    export TEST_DB_HOST="${TEST_DB_HOST:-localhost}"
    export TEST_DB_PORT="${TEST_DB_PORT:-5432}"
    export TEST_DB_USER="${TEST_DB_USER:-test_user}"
    export TEST_DB_PASSWORD="${TEST_DB_PASSWORD:-test_password}"
    export TEST_DB_NAME="${TEST_DB_NAME:-test_db}"
    export GIN_MODE="test"
    export LOG_LEVEL="info"
    
    # 验证数据库连接
    if command -v psql &> /dev/null; then
        log_info "验证数据库连接..."
        if ! PGPASSWORD="$TEST_DB_PASSWORD" psql -h "$TEST_DB_HOST" -p "$TEST_DB_PORT" -U "$TEST_DB_USER" -d "$TEST_DB_NAME" -c "SELECT 1;" &> /dev/null; then
            log_warning "无法连接到测试数据库，请确保数据库正在运行"
        else
            log_success "数据库连接验证成功"
        fi
    fi
    
    log_success "测试环境设置完成"
}

# 运行特定性能测试
run_performance_test() {
    local test_name="$1"
    local additional_args="$2"
    
    log_info "运行性能测试: $test_name"
    
    cd "$PROJECT_ROOT"
    
    local test_cmd="go test -v -timeout=30m ./test/performance/"
    
    # 添加测试过滤器
    case "$test_name" in
        "crud")
            test_cmd+=" -run TestProjectCRUDPerformance"
            ;;
        "concurrent")
            test_cmd+=" -run TestConcurrentProjectCreation"
            ;;
        "list")
            test_cmd+=" -run TestProjectListingPerformance"
            ;;
        "mixed")
            test_cmd+=" -run TestMixedWorkloadPerformance"
            ;;
        "all"|"")
            test_cmd+=" -run TestProjectServicePerformance"
            ;;
        *)
            log_error "未知的测试类型: $test_name"
            return 1
            ;;
    esac
    
    # 添加额外参数
    if [ -n "$additional_args" ]; then
        test_cmd+=" $additional_args"
    fi
    
    # 运行测试
    log_info "执行命令: $test_cmd"
    
    if eval "$test_cmd"; then
        log_success "性能测试 '$test_name' 完成"
        return 0
    else
        log_error "性能测试 '$test_name' 失败"
        return 1
    fi
}

# 运行基准测试
run_benchmark_test() {
    log_info "运行基准测试..."
    
    cd "$PROJECT_ROOT"
    
    local benchmark_cmd="go test -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof ./test/performance/"
    
    log_info "执行基准测试命令: $benchmark_cmd"
    
    if eval "$benchmark_cmd"; then
        log_success "基准测试完成"
        
        # 移动性能分析文件到报告目录
        if [ -f "cpu.prof" ]; then
            mv cpu.prof "$TEST_REPORT_DIR/"
            log_info "CPU性能分析文件已保存到 $TEST_REPORT_DIR/cpu.prof"
        fi
        
        if [ -f "mem.prof" ]; then
            mv mem.prof "$TEST_REPORT_DIR/"
            log_info "内存性能分析文件已保存到 $TEST_REPORT_DIR/mem.prof"
        fi
        
        return 0
    else
        log_error "基准测试失败"
        return 1
    fi
}

# 生成性能报告
generate_performance_html_report() {
    log_info "生成HTML性能报告..."
    
    local json_report="$TEST_REPORT_DIR/project_service_performance_report.json"
    local html_report="$TEST_REPORT_DIR/project_service_performance_report.html"
    
    if [ ! -f "$json_report" ]; then
        log_warning "JSON报告文件不存在，跳过HTML报告生成"
        return 0
    fi
    
    # 生成HTML报告
    cat > "$html_report" << 'EOF'
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Project Service Performance Test Report</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h1, h2, h3 {
            color: #333;
            border-bottom: 2px solid #007cba;
            padding-bottom: 10px;
        }
        .metrics-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin: 20px 0;
        }
        .metric-card {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 6px;
            border-left: 4px solid #007cba;
        }
        .metric-value {
            font-size: 2em;
            font-weight: bold;
            color: #007cba;
        }
        .metric-label {
            color: #666;
            text-transform: uppercase;
            font-size: 0.9em;
        }
        .chart-container {
            width: 100%;
            height: 400px;
            margin: 20px 0;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
        }
        th, td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #ddd;
        }
        th {
            background-color: #007cba;
            color: white;
        }
        tr:hover {
            background-color: #f5f5f5;
        }
        .success {
            color: #28a745;
        }
        .warning {
            color: #ffc107;
        }
        .error {
            color: #dc3545;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>📊 Project Service Performance Test Report</h1>
        <p>Generated at: <span id="timestamp"></span></p>
        
        <h2>🎯 Overall Performance Metrics</h2>
        <div class="metrics-grid" id="overall-metrics">
            <!-- 动态生成 -->
        </div>
        
        <h2>📈 Response Time Distribution</h2>
        <div class="chart-container">
            <canvas id="responseTimeChart"></canvas>
        </div>
        
        <h2>🔗 Endpoint Performance</h2>
        <table id="endpoint-table">
            <thead>
                <tr>
                    <th>Endpoint</th>
                    <th>Requests</th>
                    <th>Avg Time</th>
                    <th>Min Time</th>
                    <th>Max Time</th>
                    <th>Success Rate</th>
                    <th>Status</th>
                </tr>
            </thead>
            <tbody id="endpoint-tbody">
                <!-- 动态生成 -->
            </tbody>
        </table>
        
        <h2>⚠️ Error Distribution</h2>
        <table id="error-table">
            <thead>
                <tr>
                    <th>Status Code</th>
                    <th>Count</th>
                    <th>Percentage</th>
                </tr>
            </thead>
            <tbody id="error-tbody">
                <!-- 动态生成 -->
            </tbody>
        </table>
    </div>

    <script>
        // 设置时间戳
        document.getElementById('timestamp').textContent = new Date().toLocaleString();
        
        // 加载性能数据
        fetch('./project_service_performance_report.json')
            .then(response => response.json())
            .then(data => {
                renderOverallMetrics(data);
                renderEndpointTable(data);
                renderErrorTable(data);
                renderResponseTimeChart(data);
            })
            .catch(error => {
                console.error('Error loading performance data:', error);
                document.querySelector('.container').innerHTML += 
                    '<div class="error">Error loading performance data. Please ensure the JSON report exists.</div>';
            });
        
        function renderOverallMetrics(data) {
            const container = document.getElementById('overall-metrics');
            const metrics = [
                { label: 'Total Requests', value: data.request_count, suffix: '' },
                { label: 'Success Rate', value: data.success_rate?.toFixed(1) || '0.0', suffix: '%' },
                { label: 'Avg Response Time', value: formatDuration(data.average_response_time), suffix: '' },
                { label: 'P95 Response Time', value: formatDuration(data.p95_response_time), suffix: '' },
                { label: 'P99 Response Time', value: formatDuration(data.p99_response_time), suffix: '' },
                { label: 'Throughput', value: data.throughput_rps?.toFixed(1) || '0.0', suffix: ' RPS' }
            ];
            
            container.innerHTML = metrics.map(metric => `
                <div class="metric-card">
                    <div class="metric-value">${metric.value}${metric.suffix}</div>
                    <div class="metric-label">${metric.label}</div>
                </div>
            `).join('');
        }
        
        function renderEndpointTable(data) {
            const tbody = document.getElementById('endpoint-tbody');
            if (!data.endpoint_metrics) return;
            
            const rows = Object.entries(data.endpoint_metrics).map(([endpoint, metrics]) => {
                const statusClass = metrics.success_rate >= 95 ? 'success' : 
                                  metrics.success_rate >= 90 ? 'warning' : 'error';
                const status = metrics.success_rate >= 95 ? '✅' : 
                              metrics.success_rate >= 90 ? '⚠️' : '❌';
                
                return `
                    <tr>
                        <td>${endpoint}</td>
                        <td>${metrics.request_count}</td>
                        <td>${formatDuration(metrics.average_time)}</td>
                        <td>${formatDuration(metrics.min_time)}</td>
                        <td>${formatDuration(metrics.max_time)}</td>
                        <td class="${statusClass}">${metrics.success_rate?.toFixed(1) || '0.0'}%</td>
                        <td>${status}</td>
                    </tr>
                `;
            }).join('');
            
            tbody.innerHTML = rows;
        }
        
        function renderErrorTable(data) {
            const tbody = document.getElementById('error-tbody');
            if (!data.error_distribution) return;
            
            const totalErrors = Object.values(data.error_distribution).reduce((sum, count) => sum + count, 0);
            
            const rows = Object.entries(data.error_distribution).map(([statusCode, count]) => {
                const percentage = ((count / totalErrors) * 100).toFixed(1);
                return `
                    <tr>
                        <td>${statusCode}</td>
                        <td>${count}</td>
                        <td>${percentage}%</td>
                    </tr>
                `;
            }).join('');
            
            tbody.innerHTML = rows || '<tr><td colspan="3">No errors detected</td></tr>';
        }
        
        function renderResponseTimeChart(data) {
            // 这里可以添加更复杂的图表逻辑
            // 由于简化，暂时显示基本信息
            const ctx = document.getElementById('responseTimeChart').getContext('2d');
            
            // 模拟响应时间分布数据
            new Chart(ctx, {
                type: 'line',
                data: {
                    labels: ['0-100ms', '100-200ms', '200-500ms', '500ms-1s', '1s+'],
                    datasets: [{
                        label: 'Response Time Distribution',
                        data: [40, 30, 20, 8, 2], // 示例数据
                        borderColor: '#007cba',
                        backgroundColor: 'rgba(0, 124, 186, 0.1)',
                        tension: 0.4
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    scales: {
                        y: {
                            beginAtZero: true,
                            title: {
                                display: true,
                                text: 'Request Count'
                            }
                        },
                        x: {
                            title: {
                                display: true,
                                text: 'Response Time Range'
                            }
                        }
                    }
                }
            });
        }
        
        function formatDuration(nanoseconds) {
            if (!nanoseconds) return '0ms';
            const ms = nanoseconds / 1000000;
            if (ms < 1000) {
                return `${ms.toFixed(1)}ms`;
            } else {
                return `${(ms / 1000).toFixed(2)}s`;
            }
        }
    </script>
</body>
</html>
EOF

    log_success "HTML报告已生成: $html_report"
}

# 清理测试数据
cleanup_test_data() {
    log_info "清理测试数据..."
    
    if command -v psql &> /dev/null; then
        # 清理数据库中的测试数据
        PGPASSWORD="$TEST_DB_PASSWORD" psql -h "$TEST_DB_HOST" -p "$TEST_DB_PORT" -U "$TEST_DB_USER" -d "$TEST_DB_NAME" << EOF
DELETE FROM projects WHERE key LIKE 'perf-test-%' OR key LIKE 'load-test-%' OR key LIKE 'list-perf-%' OR key LIKE 'mixed-%';
DELETE FROM project_members WHERE project_id IN (SELECT id FROM projects WHERE key LIKE '%test%');
DELETE FROM repositories WHERE project_id IN (SELECT id FROM projects WHERE key LIKE '%test%');
EOF
    fi
    
    log_success "测试数据清理完成"
}

# 主函数
main() {
    local config_file="$PERFORMANCE_CONFIG"
    local output_dir="$TEST_REPORT_DIR"
    local test_name="all"
    local concurrent_users=""
    local duration=""
    local verbose=false
    local cleanup=false
    local no_setup=false
    local profile=false
    local benchmark=false
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -c|--config)
                config_file="$2"
                shift 2
                ;;
            -o|--output)
                output_dir="$2"
                shift 2
                ;;
            -t|--test)
                test_name="$2"
                shift 2
                ;;
            -u|--users)
                concurrent_users="$2"
                shift 2
                ;;
            -d|--duration)
                duration="$2"
                shift 2
                ;;
            -v|--verbose)
                verbose=true
                shift
                ;;
            --cleanup)
                cleanup=true
                shift
                ;;
            --no-setup)
                no_setup=true
                shift
                ;;
            --profile)
                profile=true
                shift
                ;;
            --benchmark)
                benchmark=true
                shift
                ;;
            *)
                log_error "未知选项: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    # 设置详细输出
    if [ "$verbose" = true ]; then
        set -x
    fi
    
    log_info "项目服务性能测试开始"
    log_info "配置文件: $config_file"
    log_info "输出目录: $output_dir"
    log_info "测试类型: $test_name"
    
    # 检查依赖
    check_dependencies
    
    # 设置环境
    if [ "$no_setup" = false ]; then
        setup_test_environment
    fi
    
    # 构建额外参数
    local additional_args=""
    if [ -n "$concurrent_users" ]; then
        additional_args+=" -args -users=$concurrent_users"
    fi
    if [ -n "$duration" ]; then
        additional_args+=" -duration=$duration"
    fi
    
    # 运行测试
    local test_failed=false
    
    if [ "$benchmark" = true ]; then
        if ! run_benchmark_test; then
            test_failed=true
        fi
    else
        if ! run_performance_test "$test_name" "$additional_args"; then
            test_failed=true
        fi
    fi
    
    # 生成报告
    generate_performance_html_report
    
    # 清理
    if [ "$cleanup" = true ]; then
        cleanup_test_data
    fi
    
    # 输出结果
    if [ "$test_failed" = true ]; then
        log_error "性能测试执行失败"
        exit 1
    else
        log_success "性能测试执行完成"
        log_info "查看报告: $output_dir/project_service_performance_report.html"
        exit 0
    fi
}

# 执行主函数
main "$@"