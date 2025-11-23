#!/bin/bash
# Dockerfile 构建测试脚本
# 验证 Dockerfile 可以成功构建镜像

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "Testing Dockerfile build..."

# 检查 Dockerfile 是否存在
if [ ! -f "$PROJECT_ROOT/Dockerfile" ]; then
    echo "ERROR: Dockerfile not found at $PROJECT_ROOT/Dockerfile"
    exit 1
fi

# 检查 .dockerignore 是否存在
if [ ! -f "$PROJECT_ROOT/.dockerignore" ]; then
    echo "WARNING: .dockerignore not found, but continuing..."
fi

# 实际构建测试（如果 Docker 可用）
if command -v docker &> /dev/null; then
    echo "Building Docker image..."
    # 从父目录构建，以便包含 approval-kit 目录（如果 go.mod 中有 replace 指令）
    PARENT_DIR="$(cd "$PROJECT_ROOT/.." && pwd)"
    docker build -f "$PROJECT_ROOT/Dockerfile" -t approval-gin:test "$PARENT_DIR" || {
        echo "ERROR: Docker image build failed"
        echo "Note: If go.mod has replace directive, ensure approval-kit is available in parent directory"
        exit 1
    }
    
    # 验证镜像是否创建成功
    if docker image inspect approval-gin:test &> /dev/null; then
        echo "SUCCESS: Docker image built successfully"
        # 清理测试镜像
        docker rmi approval-gin:test &> /dev/null || true
    else
        echo "ERROR: Docker image was not created"
        exit 1
    fi
else
    echo "WARNING: Docker not available, skipping actual build test"
fi

echo "Dockerfile test passed!"

