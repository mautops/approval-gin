# 多阶段构建 Dockerfile
# 阶段 1: 构建阶段
FROM golang:1.25.4-alpine AS builder

# 设置工作目录
WORKDIR /build

# 安装构建依赖
RUN apk add --no-cache git ca-certificates tzdata

# 设置 Go 代理（加速依赖下载）
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn

# 复制 go.mod 和 go.sum（利用 Docker 缓存层）
# 注意：如果从父目录构建，需要指定正确的路径
COPY approval-gin/go.mod approval-gin/go.sum ./

# 如果 go.mod 中有 replace 指令指向 approval-kit，需要先复制该目录
# 创建父目录结构以匹配 replace 指令中的路径
COPY approval-kit/ ../approval-kit/

# 下载依赖（利用 Docker 缓存层）
RUN go mod download

# 复制源代码（如果从父目录构建，需要指定正确的路径）
COPY approval-gin/ ./

# 构建应用
# CGO_ENABLED=0 禁用 CGO，生成静态链接的二进制文件
# -ldflags '-w -s' 减小二进制文件大小
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags '-w -s' \
    -o approval-gin \
    -a -installsuffix cgo \
    ./main.go

# 阶段 2: 运行阶段
FROM alpine:latest

# 安装运行时依赖（包括 wget 用于健康检查）
RUN apk --no-cache add ca-certificates tzdata wget

# 创建非 root 用户
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/approval-gin /app/approval-gin

# 复制配置文件（如果存在，使用 RUN 命令处理可选文件）
# COPY 命令不支持可选文件，所以使用 RUN 命令
RUN mkdir -p /app && \
    if [ -f /build/config.yaml ]; then \
        cp /build/config.yaml /app/; \
    fi && \
    if [ -f /build/config.yaml.example ]; then \
        cp /build/config.yaml.example /app/; \
    fi || true

# 设置文件权限
RUN chown -R appuser:appuser /app

# 切换到非 root 用户
USER appuser

# 暴露端口（默认 8080，可通过环境变量配置）
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 设置时区
ENV TZ=Asia/Shanghai

# 启动应用
ENTRYPOINT ["/app/approval-gin"]
CMD ["server"]

