# 监控和告警指南

本文档说明如何配置和使用 Prometheus 监控和告警功能。

## 概述

Approval Gin 集成了 Prometheus 监控系统，提供以下功能：

- **指标收集**: 收集 API 请求、任务创建、审批操作等业务指标
- **系统指标**: 收集数据库连接数、内存使用、CPU 使用等系统指标
- **告警规则**: 配置告警规则，及时发现和响应问题

## 指标端点

应用暴露 Prometheus 指标端点：`/metrics`

### 访问指标

```bash
curl http://localhost:8080/metrics
```

## 可用指标

### API 指标

- `api_requests_total`: API 请求总数（按方法、路径、状态码分组）
- `api_request_duration_seconds`: API 请求响应时间（直方图）

### 业务指标

- `tasks_created_total`: 任务创建总数
- `approvals_total`: 审批操作总数（按操作类型分组：approve, reject 等）
- `tasks_by_state`: 任务状态分布（按状态分组）

### 系统指标

- `database_connections_active`: 活跃数据库连接数
- `database_connections_idle`: 空闲数据库连接数
- `database_connections_max`: 最大数据库连接数
- `process_resident_memory_bytes`: 进程内存使用
- `process_cpu_seconds_total`: 进程 CPU 使用时间

### Go 运行时指标

- `go_goroutines`: Goroutine 数量
- `go_memstats_*`: 内存统计信息

## Prometheus 配置

### 1. 安装 Prometheus

```bash
# 使用 Docker
docker run -d \
  --name prometheus \
  -p 9090:9090 \
  -v $(pwd)/deploy/prometheus:/etc/prometheus \
  prom/prometheus
```

### 2. 配置文件

配置文件位于 `deploy/prometheus/prometheus.yml`：

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - 'alerts.yml'

scrape_configs:
  - job_name: 'approval-gin'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

### 3. 启动 Prometheus

```bash
prometheus --config.file=deploy/prometheus/prometheus.yml
```

访问 Prometheus UI: http://localhost:9090

## 告警规则

告警规则文件位于 `deploy/prometheus/alerts.yml`。

### 已配置的告警

1. **HighErrorRate**: API 错误率过高（5xx 错误率 > 10%）
2. **HighLatency**: API 延迟过高（95 分位延迟 > 1 秒）
3. **DatabaseConnectionFailure**: 数据库连接失败
4. **HighTaskCreationRate**: 任务创建率过高（> 100 tasks/sec）
5. **HighApprovalFailureRate**: 审批失败率过高（拒绝率 > 50%）
6. **DatabaseConnectionPoolExhausted**: 数据库连接池耗尽（使用率 > 90%）
7. **HighMemoryUsage**: 内存使用率过高（> 90%）
8. **HighCPUUsage**: CPU 使用率过高（> 80%）
9. **UnusualRequestRate**: 请求速率异常（变化 > 50%）

### 告警规则示例

```yaml
- alert: HighErrorRate
  expr: |
    rate(api_requests_total{status=~"5.."}[5m]) > 0.1
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "High API error rate detected"
    description: "Error rate is {{ $value | humanizePercentage }}"
```

## Alertmanager 集成

### 1. 安装 Alertmanager

```bash
docker run -d \
  --name alertmanager \
  -p 9093:9093 \
  prom/alertmanager
```

### 2. 配置告警通知

编辑 `deploy/prometheus/alertmanager.yml`：

```yaml
route:
  receiver: 'default-receiver'
  group_by: ['alertname']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h

receivers:
  - name: 'default-receiver'
    email_configs:
      - to: 'admin@example.com'
        from: 'alerts@example.com'
        smarthost: 'smtp.example.com:587'
        auth_username: 'alerts@example.com'
        auth_password: 'password'
```

### 3. 更新 Prometheus 配置

在 `prometheus.yml` 中添加：

```yaml
alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - 'alertmanager:9093'
```

## Grafana 集成

### 1. 安装 Grafana

```bash
docker run -d \
  --name grafana \
  -p 3000:3000 \
  grafana/grafana
```

### 2. 配置数据源

1. 访问 http://localhost:3000
2. 登录（默认用户名/密码：admin/admin）
3. 添加 Prometheus 数据源：http://prometheus:9090

### 3. 导入仪表板

可以使用以下查询创建仪表板：

**API 请求速率**:
```
rate(api_requests_total[5m])
```

**API 错误率**:
```
rate(api_requests_total{status=~"5.."}[5m]) / rate(api_requests_total[5m])
```

**任务创建速率**:
```
rate(tasks_created_total[5m])
```

**数据库连接使用率**:
```
(database_connections_active / database_connections_max) * 100
```

## 指标收集器

应用内置了指标收集器，定期更新数据库连接数等指标。

### 启动收集器

```go
collector := metrics.NewCollector(db, 30*time.Second)
collector.Start()
defer collector.Stop()
```

## 最佳实践

1. **监控关键指标**: 重点关注错误率、延迟、数据库连接数
2. **设置合理的阈值**: 根据实际业务情况调整告警阈值
3. **告警分组**: 使用 Alertmanager 对告警进行分组，避免告警风暴
4. **定期审查**: 定期审查告警规则，优化误报和漏报
5. **性能影响**: 监控指标收集对应用性能的影响，必要时调整收集频率

## 故障排查

### 指标端点无响应

1. 检查应用是否正常运行
2. 检查 `/metrics` 端点是否可访问
3. 检查防火墙和网络配置

### 告警未触发

1. 检查 Prometheus 是否正常抓取指标
2. 检查告警规则表达式是否正确
3. 检查告警阈值是否合理
4. 检查 Alertmanager 配置

### 指标数据缺失

1. 检查应用日志，查看是否有错误
2. 检查指标收集器是否正常运行
3. 检查数据库连接是否正常

## 更新记录

- **2025-11-22**: 初始监控和告警配置完成

