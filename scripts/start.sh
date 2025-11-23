#!/bin/bash
# Approval Gin 启动脚本
# 用于启动 Approval Gin API 服务器

set -e

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# 切换到项目根目录
cd "$PROJECT_ROOT"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 打印信息
info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查 Go 是否安装
if ! command -v go &> /dev/null; then
    error "Go is not installed. Please install Go 1.25.4 or higher."
    exit 1
fi

# 检查 Go 版本
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED_VERSION="1.25.4"
if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
    error "Go version $GO_VERSION is too old. Required: $REQUIRED_VERSION or higher."
    exit 1
fi

info "Go version: $GO_VERSION"

# 检查配置文件
CONFIG_FILE="${CONFIG_FILE:-config.yaml}"
if [ ! -f "$CONFIG_FILE" ] && [ ! -f "./config/$CONFIG_FILE" ]; then
    warn "Config file not found. Using environment variables or defaults."
fi

# 检查数据库连接（可选）
if [ -n "$APP_DATABASE_HOST" ]; then
    info "Database host: $APP_DATABASE_HOST:$APP_DATABASE_PORT"
fi

# 运行数据库迁移（如果设置了环境变量）
if [ "${RUN_MIGRATIONS:-false}" = "true" ]; then
    info "Running database migrations..."
    go run main.go migrate --config "$CONFIG_FILE" || {
        error "Database migration failed"
        exit 1
    }
fi

# 启动服务器
info "Starting Approval Gin API server..."
info "Server will listen on ${APP_SERVER_HOST:-0.0.0.0}:${APP_SERVER_PORT:-8080}"

# 执行服务器命令
exec go run main.go server --config "$CONFIG_FILE"

