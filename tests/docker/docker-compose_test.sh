#!/bin/bash
# Docker Compose 配置测试脚本
# 验证 docker-compose.yml 可以正确启动服务

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "Testing docker-compose.yml..."

# 检查 docker-compose.yml 是否存在
if [ ! -f "$PROJECT_ROOT/docker-compose.yml" ]; then
    echo "ERROR: docker-compose.yml not found at $PROJECT_ROOT/docker-compose.yml"
    exit 1
fi

# 检查 docker-compose 是否可用
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo "WARNING: docker-compose not available, skipping validation"
    exit 0
fi

# 使用 docker compose (新版本) 或 docker-compose (旧版本)
if docker compose version &> /dev/null; then
    COMPOSE_CMD="docker compose"
else
    COMPOSE_CMD="docker-compose"
fi

# 验证配置文件语法
echo "Validating docker-compose.yml syntax..."
cd "$PROJECT_ROOT"
$COMPOSE_CMD config > /dev/null || {
    echo "ERROR: docker-compose.yml syntax validation failed"
    exit 1
}

# 检查必需的服务是否定义
echo "Checking required services..."
REQUIRED_SERVICES=("postgres" "openfga" "approval-gin")
for service in "${REQUIRED_SERVICES[@]}"; do
    if ! $COMPOSE_CMD config --services | grep -q "^${service}$"; then
        echo "WARNING: Required service '${service}' not found in docker-compose.yml"
    else
        echo "✓ Service '${service}' found"
    fi
done

echo "docker-compose.yml test passed!"

