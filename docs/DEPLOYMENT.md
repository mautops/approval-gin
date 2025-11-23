# Approval Gin 部署文档

本文档介绍如何部署 Approval Gin API 服务器。

## 目录

- [系统要求](#系统要求)
- [快速开始](#快速开始)
- [配置说明](#配置说明)
- [部署方式](#部署方式)
  - [Docker 部署](#docker-部署)
  - [Docker Compose 部署](#docker-compose-部署)
  - [二进制文件部署](#二进制文件部署)
  - [源码部署](#源码部署)
- [数据库迁移](#数据库迁移)
- [环境变量](#环境变量)
- [健康检查](#健康检查)
- [故障排查](#故障排查)

## 系统要求

### 最低要求

- **操作系统**: Linux, macOS, Windows
- **Go 版本**: 1.25.4 或更高版本（仅源码部署需要）
- **内存**: 512MB 以上
- **磁盘**: 1GB 以上可用空间

### 依赖服务

- **PostgreSQL**: 12.0 或更高版本（推荐 16+）
- **OpenFGA**: 最新版本（可选，用于权限管理）
- **Keycloak**: 最新版本（可选，用于用户认证）

## 快速开始

### 使用 Docker Compose（推荐）

```bash
# 1. 克隆仓库
git clone https://github.com/mautops/approval-gin.git
cd approval-gin

# 2. 启动所有服务
docker-compose up -d

# 3. 运行数据库迁移
docker-compose exec approval-gin go run main.go migrate

# 4. 检查服务状态
docker-compose ps
```

服务将在以下地址可用：
- API 服务: http://localhost:8080
- Swagger UI: http://localhost:8080/swagger/index.html
- PostgreSQL: localhost:5432
- OpenFGA: http://localhost:8081

## 配置说明

### 配置文件

创建 `config.yaml` 文件（或使用环境变量）：

```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "postgres"
  dbname: "approval"
  sslmode: "disable"

openfga:
  api_url: "http://localhost:8081"
  store_id: ""
  model_id: ""

keycloak:
  issuer: "https://keycloak.example.com/realms/your-realm"
  jwks_url: "https://keycloak.example.com/realms/your-realm/protocol/openid-connect/certs"

cors:
  allowed_origins:
    - "*"
  allowed_methods:
    - "GET"
    - "POST"
    - "PUT"
    - "DELETE"
    - "PATCH"
    - "OPTIONS"
  allowed_headers:
    - "Content-Type"
    - "Authorization"
    - "X-Request-ID"
  max_age: 86400
```

### 环境变量

所有配置项都支持通过环境变量设置，使用 `APP_` 前缀：

```bash
export APP_SERVER_HOST=0.0.0.0
export APP_SERVER_PORT=8080
export APP_DATABASE_HOST=localhost
export APP_DATABASE_PORT=5432
export APP_DATABASE_USER=postgres
export APP_DATABASE_PASSWORD=postgres
export APP_DATABASE_DBNAME=approval
export APP_DATABASE_SSLMODE=disable
```

详细的环境变量列表请参考 `.env.example` 文件。

## 部署方式

### Docker 部署

#### 构建镜像

```bash
# 从父目录构建（包含 approval-kit）
cd ..
docker build -f approval-gin/Dockerfile -t approval-gin:latest .
```

#### 运行容器

```bash
docker run -d \
  --name approval-gin \
  -p 8080:8080 \
  -e APP_DATABASE_HOST=postgres \
  -e APP_DATABASE_USER=postgres \
  -e APP_DATABASE_PASSWORD=postgres \
  -e APP_DATABASE_DBNAME=approval \
  approval-gin:latest
```

### Docker Compose 部署

使用提供的 `docker-compose.yml` 文件：

```bash
# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f approval-gin

# 停止服务
docker-compose down

# 停止并删除数据卷
docker-compose down -v
```

### 二进制文件部署

#### 构建二进制文件

```bash
# 构建 Linux 版本
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o approval-gin ./main.go

# 构建 macOS 版本
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o approval-gin ./main.go

# 构建 Windows 版本
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o approval-gin.exe ./main.go
```

#### 运行二进制文件

```bash
# 运行数据库迁移
./approval-gin migrate

# 启动服务器
./approval-gin server
```

### 源码部署

#### 安装依赖

```bash
go mod download
```

#### 使用启动脚本

```bash
# 使用启动脚本
./scripts/start.sh

# 或手动运行
go run main.go server
```

## 数据库迁移

### 首次部署

在首次部署时，需要运行数据库迁移：

```bash
# 使用 migrate 命令
./approval-gin migrate

# 或使用 Docker
docker-compose exec approval-gin go run main.go migrate
```

### 迁移内容

迁移命令会：
- 创建所有必需的数据表
- 创建索引以优化查询性能
- 支持 PostgreSQL 和 SQLite 数据库

### 迁移验证

迁移完成后，可以验证表是否创建成功：

```bash
# PostgreSQL
psql -U postgres -d approval -c "\dt"

# 或使用 Docker
docker-compose exec postgres psql -U postgres -d approval -c "\dt"
```

## 环境变量

所有配置项都支持通过环境变量设置。环境变量优先级高于配置文件。

### 常用环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `APP_SERVER_HOST` | 服务器监听地址 | `0.0.0.0` |
| `APP_SERVER_PORT` | 服务器端口 | `8080` |
| `APP_DATABASE_HOST` | 数据库主机 | `localhost` |
| `APP_DATABASE_PORT` | 数据库端口 | `5432` |
| `APP_DATABASE_USER` | 数据库用户 | `postgres` |
| `APP_DATABASE_PASSWORD` | 数据库密码 | - |
| `APP_DATABASE_DBNAME` | 数据库名称 | `approval` |
| `APP_DATABASE_SSLMODE` | SSL 模式 | `disable` |
| `APP_OPENFGA_API_URL` | OpenFGA API 地址 | `http://localhost:8081` |
| `APP_KEYCLOAK_ISSUER` | Keycloak Issuer URL | - |

完整的环境变量列表请参考 `.env.example` 文件。

## 健康检查

### HTTP 健康检查

```bash
curl http://localhost:8080/health
```

### Docker 健康检查

Docker 容器包含内置健康检查：

```bash
# 查看健康状态
docker inspect --format='{{.State.Health.Status}}' approval-gin

# 查看健康检查日志
docker inspect --format='{{json .State.Health}}' approval-gin | jq
```

### 健康检查端点

- **健康检查**: `GET /health`
- **指标**: `GET /metrics` (如果启用 Prometheus)

## 故障排查

### 常见问题

#### 1. 数据库连接失败

**症状**: 启动时提示数据库连接失败

**解决方案**:
- 检查数据库服务是否运行
- 验证数据库连接配置（主机、端口、用户名、密码）
- 检查网络连接和防火墙设置
- 验证数据库用户权限

#### 2. 端口已被占用

**症状**: 启动时提示端口已被占用

**解决方案**:
- 更改 `APP_SERVER_PORT` 环境变量
- 或修改配置文件中的端口设置
- 检查是否有其他服务占用该端口

#### 3. 迁移失败

**症状**: 运行 `migrate` 命令时失败

**解决方案**:
- 检查数据库连接配置
- 验证数据库用户是否有创建表的权限
- 检查数据库版本是否兼容
- 查看详细错误日志

#### 4. OpenFGA 连接失败

**症状**: 权限检查失败

**解决方案**:
- 检查 OpenFGA 服务是否运行
- 验证 `APP_OPENFGA_API_URL` 配置
- 检查网络连接
- 验证 Store ID 和 Model ID 配置

### 日志查看

#### Docker 日志

```bash
# 查看所有服务日志
docker-compose logs

# 查看特定服务日志
docker-compose logs approval-gin

# 实时查看日志
docker-compose logs -f approval-gin
```

#### 系统日志

如果使用 systemd 管理服务，查看日志：

```bash
journalctl -u approval-gin -f
```

## 生产环境配置

### 环境变量设置

生产环境应设置 `APP_ENV=production`:

```bash
export APP_ENV=production
```

### 数据库连接池配置

生产环境默认连接池配置：

```yaml
database:
  max_idle_conns: 20      # 最大空闲连接数
  max_open_conns: 200     # 最大打开连接数
  conn_max_lifetime: 3600 # 连接最大生存时间（秒）
  conn_max_idle_time: 300 # 连接最大空闲时间（秒）
```

或使用环境变量：

```bash
export APP_DATABASE_MAX_IDLE_CONNS=20
export APP_DATABASE_MAX_OPEN_CONNS=200
export APP_DATABASE_CONN_MAX_LIFETIME=3600
export APP_DATABASE_CONN_MAX_IDLE_TIME=300
```

### 日志配置

生产环境日志配置：

```yaml
log:
  level: warn          # 生产环境使用 warn 级别
  format: json         # 使用 JSON 格式便于日志聚合
  output: both         # 同时输出到标准输出和文件
```

或使用环境变量：

```bash
export APP_LOG_LEVEL=warn
export APP_LOG_FORMAT=json
export APP_LOG_OUTPUT=both
```

## 监控和告警

### Prometheus 监控

应用暴露 Prometheus 指标端点：`/metrics`

#### 配置 Prometheus

编辑 `deploy/prometheus/prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'approval-gin'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

#### 告警规则

告警规则文件位于 `deploy/prometheus/alerts.yml`，包含：

- HighErrorRate: API 错误率过高
- HighLatency: API 延迟过高
- DatabaseConnectionFailure: 数据库连接失败
- HighTaskCreationRate: 任务创建率过高
- HighApprovalFailureRate: 审批失败率过高

详细说明请参考 [监控和告警指南](./MONITORING.md)

## 日志聚合

### ELK Stack 集成

应用支持结构化 JSON 日志输出，便于日志聚合。

#### 配置 Filebeat

编辑 `deploy/logging/filebeat.yml`:

```yaml
filebeat.inputs:
  - type: log
    paths:
      - /var/log/approval-gin/*.log
    json.keys_under_root: true
```

#### 配置 Logstash

编辑 `deploy/logging/logstash.conf`:

```conf
input {
  beats {
    port => 5044
  }
}

filter {
  if [fields][service] == "approval-gin" {
    date {
      match => [ "time", "ISO8601" ]
    }
  }
}

output {
  elasticsearch {
    hosts => ["http://elasticsearch:9200"]
    index => "approval-gin-%{+YYYY.MM.dd}"
  }
}
```

详细说明请参考 [日志聚合指南](./LOG_AGGREGATION.md)

## 备份策略

### 自动备份配置

配置自动备份策略：

```yaml
backup:
  full_backup:
    enabled: true
    schedule: "0 0 * * *"  # 每天凌晨执行
    retention_days: 30
  
  incremental_backup:
    enabled: true
    interval: "1h"          # 每小时执行一次
    retention_days: 7
  
  verify: true
  directory: "/var/backups/approval-gin"
```

或使用环境变量：

```bash
export APP_BACKUP_FULL_ENABLED=true
export APP_BACKUP_FULL_SCHEDULE="0 0 * * *"
export APP_BACKUP_FULL_RETENTION_DAYS=30
export APP_BACKUP_INCREMENTAL_ENABLED=true
export APP_BACKUP_INCREMENTAL_INTERVAL=1h
export APP_BACKUP_INCREMENTAL_RETENTION_DAYS=7
export APP_BACKUP_VERIFY=true
export APP_BACKUP_DIRECTORY=/var/backups/approval-gin
```

详细说明请参考 [备份策略指南](./BACKUP_STRATEGY.md)

## 生产环境建议

### 安全配置

1. **使用强密码**: 为数据库和 Keycloak 设置强密码
2. **启用 SSL/TLS**: 在生产环境中启用数据库 SSL 连接
3. **限制 CORS**: 配置 `APP_CORS_ALLOWED_ORIGINS` 限制允许的源
4. **使用 HTTPS**: 通过反向代理（如 Nginx）提供 HTTPS
5. **定期更新**: 保持依赖和系统更新
6. **敏感数据加密**: 敏感配置信息使用加密存储

### 性能优化

1. **数据库连接池**: 根据负载调整连接池大小（生产环境默认：MaxIdleConns=20, MaxOpenConns=200）
2. **缓存策略**: 启用模板和权限缓存
3. **负载均衡**: 使用多个实例和负载均衡器
4. **监控告警**: 配置 Prometheus 监控和告警
5. **索引优化**: 确保数据库索引正确创建

### 高可用性

1. **数据库主从**: 配置 PostgreSQL 主从复制
2. **多实例部署**: 部署多个 API 实例
3. **健康检查**: 配置健康检查和自动重启
4. **备份策略**: 配置自动备份策略（全量备份 + 增量备份）
5. **异地备份**: 定期将备份复制到异地存储

### 运维监控

1. **Prometheus 监控**: 配置 Prometheus 收集指标
2. **告警规则**: 配置告警规则，及时发现和响应问题
3. **日志聚合**: 配置 ELK Stack 或类似方案进行日志聚合
4. **性能监控**: 监控 API 响应时间、错误率、数据库连接数等指标
5. **容量规划**: 定期评估系统容量，提前扩容

## 支持

如有问题，请参考：
- [GitHub Issues](https://github.com/mautops/approval-gin/issues)
- [API 文档](http://localhost:8080/swagger/index.html)
- [README.md](../README.md)

