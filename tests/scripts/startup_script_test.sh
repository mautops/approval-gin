#!/bin/bash
# 启动脚本测试
# 验证启动脚本可以正确执行

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "Testing startup scripts..."

# 检查启动脚本是否存在
START_SCRIPT="$PROJECT_ROOT/scripts/start.sh"
if [ ! -f "$START_SCRIPT" ]; then
    echo "ERROR: start.sh not found at $START_SCRIPT"
    exit 1
fi

# 检查脚本是否可执行
if [ ! -x "$START_SCRIPT" ]; then
    echo "ERROR: start.sh is not executable"
    exit 1
fi

# 检查脚本语法
echo "Validating start.sh syntax..."
bash -n "$START_SCRIPT" || {
    echo "ERROR: start.sh has syntax errors"
    exit 1
}

# 检查部署文档是否存在
DEPLOY_DOC="$PROJECT_ROOT/docs/DEPLOYMENT.md"
if [ ! -f "$DEPLOY_DOC" ]; then
    echo "WARNING: DEPLOYMENT.md not found at $DEPLOY_DOC"
else
    echo "✓ DEPLOYMENT.md found"
fi

echo "Startup script test passed!"

