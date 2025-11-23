# 日志聚合指南

本文档说明如何配置和使用日志聚合功能，将 Approval Gin 的日志收集、处理和存储到集中式日志系统（如 ELK Stack）。

## 概述

Approval Gin 支持结构化日志输出，便于日志聚合和分析：

- **JSON 格式**: 默认使用 JSON 格式输出日志，便于解析和索引
- **结构化字段**: 包含服务名称、请求 ID、用户 ID、操作类型等结构化字段
- **ISO 8601 时间戳**: 使用标准时间戳格式，便于时间序列分析
- **多输出支持**: 支持同时输出到标准输出和文件

## 日志格式

### JSON 格式示例

```json
{
  "time": "2025-11-22T18:57:07.123Z",
  "level": "info",
  "msg": "API request",
  "service": "approval-gin",
  "request_id": "req-123",
  "method": "POST",
  "path": "/api/v1/tasks",
  "status": 200,
  "latency": "45.2ms",
  "ip": "192.168.1.100",
  "user_id": "user-456",
  "action": "create_task"
}
```

### 日志字段说明

- `time`: 时间戳（ISO 8601 格式）
- `level`: 日志级别（debug, info, warn, error）
- `msg`: 日志消息
- `service`: 服务名称（固定为 "approval-gin"）
- `request_id`: 请求 ID（用于追踪请求）
- `method`: HTTP 方法
- `path`: 请求路径
- `status`: HTTP 状态码
- `latency`: 请求延迟
- `ip`: 客户端 IP
- `user_id`: 用户 ID（如适用）
- `action`: 操作类型（如适用）
- `resource`: 资源类型（如适用）
- `resource_id`: 资源 ID（如适用）

## 日志配置

### 配置文件

在 `config.yaml` 中配置日志：

```yaml
log:
  level: info          # 日志级别: debug, info, warn, error
  format: json         # 日志格式: json, text
  output: both         # 输出位置: stdout, file, both
```

### 环境变量

```bash
export APP_LOG_LEVEL=info
export APP_LOG_FORMAT=json
export APP_LOG_OUTPUT=both
```

### 生产环境配置

生产环境建议配置：

```yaml
log:
  level: warn          # 生产环境使用 warn 级别
  format: json         # 使用 JSON 格式便于日志聚合
  output: both         # 同时输出到标准输出和文件
```

## ELK Stack 集成

### 1. 安装 ELK Stack

使用 Docker Compose 快速启动：

```yaml
version: '3.8'
services:
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.11.0
    environment:
      - discovery.type=single-node
    ports:
      - "9200:9200"
  
  logstash:
    image: docker.elastic.co/logstash/logstash:8.11.0
    volumes:
      - ./deploy/logging/logstash.conf:/usr/share/logstash/pipeline/logstash.conf
    ports:
      - "5044:5044"
    depends_on:
      - elasticsearch
  
  kibana:
    image: docker.elastic.co/kibana/kibana:8.11.0
    ports:
      - "5601:5601"
    depends_on:
      - elasticsearch
  
  filebeat:
    image: docker.elastic.co/beats/filebeat:8.11.0
    volumes:
      - ./deploy/logging/filebeat.yml:/usr/share/filebeat/filebeat.yml
      - ./logs:/var/log/approval-gin:ro
    depends_on:
      - logstash
```

### 2. 配置 Filebeat

编辑 `deploy/logging/filebeat.yml`：

```yaml
filebeat.inputs:
  - type: log
    enabled: true
    paths:
      - /var/log/approval-gin/*.log
    fields:
      service: approval-gin
      environment: production
    json.keys_under_root: true
    json.message_key: msg

output.logstash:
  hosts: ["logstash:5044"]
```

### 3. 配置 Logstash

编辑 `deploy/logging/logstash.conf`：

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
      target => "@timestamp"
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

### 4. 创建 Elasticsearch 索引模板

```bash
curl -X PUT "http://localhost:9200/_template/approval-gin" \
  -H 'Content-Type: application/json' \
  -d @deploy/logging/elasticsearch-template.json
```

### 5. 启动服务

```bash
docker-compose up -d
```

## Kibana 仪表板

### 创建索引模式

1. 访问 http://localhost:5601
2. 进入 Management > Index Patterns
3. 创建索引模式：`approval-gin-*`
4. 选择时间字段：`@timestamp`

### 常用查询

**错误日志查询**:
```
level:error
```

**特定用户的日志**:
```
user_id:user-123
```

**特定操作的日志**:
```
action:create_task
```

**高延迟请求**:
```
latency:>1000ms
```

**时间范围查询**:
```
@timestamp:[2025-11-22T00:00:00 TO 2025-11-22T23:59:59]
```

### 可视化图表

可以创建以下可视化图表：

1. **错误率趋势**: 按时间统计错误日志数量
2. **API 请求分布**: 按路径统计请求数量
3. **用户活动**: 按用户统计操作数量
4. **响应时间分布**: 统计响应时间分布
5. **操作类型分布**: 统计不同操作类型的数量

## 其他日志聚合方案

### Loki + Grafana

如果使用 Loki 作为日志聚合系统：

```yaml
# promtail 配置
server:
  http_listen_port: 9080
  grpc_listen_port: 0

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: approval-gin
    static_configs:
      - targets:
          - localhost
        labels:
          job: approval-gin
          __path__: /var/log/approval-gin/*.log
```

### Fluentd

如果使用 Fluentd：

```xml
<source>
  @type tail
  path /var/log/approval-gin/*.log
  pos_file /var/log/fluentd/approval-gin.log.pos
  tag approval-gin
  format json
  time_key time
  time_format %Y-%m-%dT%H:%M:%S.%NZ
</source>

<match approval-gin>
  @type elasticsearch
  host localhost
  port 9200
  index_name approval-gin
  type_name _doc
</match>
```

## 日志轮转

建议配置日志轮转，避免日志文件过大：

### 使用 logrotate

创建 `/etc/logrotate.d/approval-gin`:

```
/var/log/approval-gin/*.log {
    daily
    rotate 7
    compress
    delaycompress
    notifempty
    create 0644 root root
    sharedscripts
    postrotate
        /bin/kill -HUP `cat /var/run/approval-gin.pid 2> /dev/null` 2> /dev/null || true
    endscript
}
```

## 最佳实践

1. **结构化日志**: 始终使用结构化日志，包含必要的上下文信息
2. **日志级别**: 合理使用日志级别，避免过多 debug 日志
3. **敏感信息**: 不要在日志中记录敏感信息（密码、token 等）
4. **日志轮转**: 配置日志轮转，避免磁盘空间耗尽
5. **集中存储**: 使用集中式日志系统，便于查询和分析
6. **监控告警**: 基于日志设置告警规则，及时发现问题

## 故障排查

### 日志文件未生成

1. 检查日志目录权限
2. 检查日志配置是否正确
3. 检查应用是否有写入权限

### 日志格式不正确

1. 检查日志格式配置（json/text）
2. 验证 JSON 格式是否有效
3. 检查时间戳格式

### 日志聚合失败

1. 检查 Filebeat/Logstash 配置
2. 检查网络连接
3. 检查 Elasticsearch 是否正常运行
4. 查看 Filebeat/Logstash 日志

## 更新记录

- **2025-11-22**: 初始日志聚合配置完成

