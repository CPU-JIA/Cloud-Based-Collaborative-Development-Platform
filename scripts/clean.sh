#!/bin/bash

# 项目清理脚本 - 清理临时文件和构建产物
echo "🧹 启动项目清理脚本..."

DRY_RUN=false
if [[ "$1" == "--dry-run" ]]; then
    DRY_RUN=true
    echo "🔍 预览模式 - 将显示要删除的文件但不实际删除"
fi

# 清理日志文件
echo "📜 清理日志文件..."
if [ "$DRY_RUN" = true ]; then
    find . -name "*.log" -type f
else
    find . -name "*.log" -type f -delete
    echo "   ✅ 已删除所有 .log 文件"
fi

# 清理进程ID文件
echo "🔧 清理进程ID文件..."
if [ "$DRY_RUN" = true ]; then
    find . -name "*.pid" -type f
else
    find . -name "*.pid" -type f -delete
    echo "   ✅ 已删除所有 .pid 文件"
fi

# 清理编译后的可执行文件
echo "⚙️  清理可执行文件..."
EXECUTABLES=(
    "project-service"
    "web-server"
    "web-server-3001"
    "web/web-server"
    "tools/api-test/api-test"
    "tools/db-test/db-test" 
    "tools/docker-test/docker-test"
)

for executable in "${EXECUTABLES[@]}"; do
    if [ -f "$executable" ]; then
        if [ "$DRY_RUN" = true ]; then
            echo "   将删除: $executable"
        else
            rm -f "$executable"
            echo "   ✅ 已删除: $executable"
        fi
    fi
done

# 清理构建产物目录
echo "📁 清理构建产物..."
BUILD_DIRS=("dist" "build" "coverage" "temp")

for dir in "${BUILD_DIRS[@]}"; do
    if [ -d "$dir" ]; then
        if [ "$DRY_RUN" = true ]; then
            echo "   将删除目录: $dir/"
        else
            rm -rf "$dir"
            echo "   ✅ 已删除目录: $dir/"
        fi
    fi
done

# 清理测试覆盖率文件
echo "📊 清理测试覆盖率文件..."
if [ "$DRY_RUN" = true ]; then
    find . -name "coverage.out" -type f
else
    find . -name "coverage.out" -type f -delete
    echo "   ✅ 已删除测试覆盖率文件"
fi

echo ""
if [ "$DRY_RUN" = false ]; then
    echo "🎉 清理完成！"
else
    echo "🔍 预览完成！要实际清理请运行: ./scripts/clean.sh"
fi