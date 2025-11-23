# 运维手册

本文档提供 Approval Gin 的运维操作指南，包括日常维护、故障处理、性能优化等内容。

## 目录

- [日常维护](#日常维护)
- [监控和告警](#监控和告警)
- [日志管理](#日志管理)
- [备份和恢复](#备份和恢复)
- [性能优化](#性能优化)
- [故障处理](#故障处理)
- [升级和迁移](#升级和迁移)

## 日常维护

### 健康检查

定期检查服务健康状态：

```bash
# HTTP 健康检查
curl http://localhost:8080/health

# 检查服务状态
docker-compose ps

# 查看服务日志
docker-compose logs -f approval-gin
```

### 数据库维护

#### 检查数据库连接

```bash
# PostgreSQL
psql -U postgres -d approval -c "SELECT version();"

# 检查连接数
psql -U postgres -d approval -c "SELECT count(*) FROM pg_stat_activity;"
```

#### 数据库优化

```sql
-- 更新表统计信息
ANALYZE;

-- 重建索引
REINDEX DATABASE approval;

-- 检查表大小
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

### 清理旧数据

定期清理过期的审计日志和事件记录：

```sql
-- 清理 90 天前的审计日志
DELETE FROM audit_logs WHERE created_at < NOW() - INTERVAL '90 days';

-- 清理已处理的事件
DELETE FROM events WHERE status = 'success' AND created_at < NOW() - INTERVAL '30 days';
```

## 监控和告警

### Prometheus 指标

访问 Prometheus 指标端点：

```bash
curl http://localhost:8080/metrics
```

### 关键指标

需要重点关注的指标：

1. **API 请求速率**: `rate(api_requests_total[5m])`
2. **API 错误率**: `rate(api_requests_total{status=~"5.."}[5m]) / rate(api_requests_total[5m])`
3. **API 延迟**: `histogram_quantile(0.95, rate(api_request_duration_seconds_bucket[5m]))`
4. **数据库连接数**: `database_connections_active / database_connections_max`
5. **任务创建速率**: `rate(tasks_created_total[5m])`

### 告警处理

当收到告警时：

1. **HighErrorRate**: 检查应用日志，查找错误原因
2. **HighLatency**: 检查数据库性能，优化慢查询
3. **DatabaseConnectionFailure**: 检查数据库服务状态和网络连接
4. **HighTaskCreationRate**: 评估是否需要扩容

详细说明请参考 [监控和告警指南](./MONITORING.md)

## 日志管理

### 日志位置

- **标准输出**: Docker 容器日志
- **日志文件**: `/var/log/approval-gin/approval-gin.log`（如果配置了文件输出）

### 日志级别

- **debug**: 详细调试信息（开发环境）
- **info**: 一般信息（默认）
- **warn**: 警告信息（生产环境推荐）
- **error**: 错误信息

### 日志查询

#### 使用 grep

```bash
# 查找错误日志
grep '"level":"error"' /var/log/approval-gin/approval-gin.log

# 查找特定用户的日志
grep '"user_id":"user-123"' /var/log/approval-gin/approval-gin.log

# 查找特定操作的日志
grep '"action":"create_task"' /var/log/approval-gin/approval-gin.log
```

#### 使用 ELK Stack

在 Kibana 中查询：

```
# 错误日志
level:error

# 特定用户
user_id:user-123

# 时间范围
@timestamp:[2025-11-22T00:00:00 TO 2025-11-22T23:59:59]
```

详细说明请参考 [日志聚合指南](./LOG_AGGREGATION.md)

## 备份和恢复

### 备份检查

定期检查备份状态：

```bash
# 列出所有备份
ls -lh /var/backups/approval-gin/

# 检查备份文件大小
du -sh /var/backups/approval-gin/*
```

### 手动备份

```bash
# 使用备份服务 API（如果实现了）
curl -X POST http://localhost:8080/api/v1/backup

# 或使用数据库工具
pg_dump -U postgres -d approval > backup_$(date +%Y%m%d_%H%M%S).sql
```

### 备份恢复

#### 从备份恢复

```bash
# 使用备份服务 API（如果实现了）
curl -X POST http://localhost:8080/api/v1/backup/restore \
  -d '{"backup_path": "/var/backups/approval-gin/backup_postgres_20251122_000000.tar.gz"}'

# 或使用数据库工具
psql -U postgres -d approval < backup_20251122_000000.sql
```

#### 恢复前准备

1. 停止应用服务
2. 备份当前数据库（以防恢复失败）
3. 清空目标数据库（可选）
4. 执行恢复操作
5. 验证数据完整性
6. 重启应用服务

详细说明请参考 [备份策略指南](./BACKUP_STRATEGY.md)

## 性能优化

### 数据库优化

#### 索引优化

检查缺失的索引：

```sql
-- 查找慢查询
SELECT 
    query,
    calls,
    total_time,
    mean_time
FROM pg_stat_statements
ORDER BY mean_time DESC
LIMIT 10;
```

#### 连接池优化

根据实际负载调整连接池大小：

```yaml
database:
  max_idle_conns: 20      # 根据并发连接数调整
  max_open_conns: 200     # 根据最大并发数调整
  conn_max_lifetime: 3600  # 1 小时
  conn_max_idle_time: 300 # 5 分钟
```

### 缓存优化

#### 模板缓存

模板缓存默认 TTL 为 1 小时，可以根据实际情况调整。

#### 权限缓存

权限缓存默认 TTL 为 5 分钟，可以根据实际情况调整。

### API 性能

#### 响应时间优化

1. 优化数据库查询（使用索引、避免 N+1 查询）
2. 启用缓存（模板缓存、权限缓存）
3. 使用连接池
4. 优化 JSON 序列化

#### 并发优化

1. 调整 Goroutine 数量
2. 使用连接池
3. 优化锁的使用

## 故障处理

### 服务无法启动

#### 检查步骤

1. 检查日志：`docker-compose logs approval-gin`
2. 检查数据库连接：`psql -U postgres -d approval -c "SELECT 1;"`
3. 检查端口占用：`netstat -tuln | grep 8080`
4. 检查配置文件：验证配置是否正确

#### 常见原因

- 数据库连接失败
- 端口已被占用
- 配置文件错误
- 依赖服务未启动

### 数据库连接问题

#### 症状

- 启动时提示数据库连接失败
- API 请求返回 500 错误
- 健康检查失败

#### 解决方案

1. 检查数据库服务状态
2. 验证连接配置（主机、端口、用户名、密码）
3. 检查网络连接和防火墙
4. 验证数据库用户权限
5. 检查连接池配置

### 性能问题

#### 症状

- API 响应时间过长
- 数据库连接数过高
- CPU 或内存使用率过高

#### 解决方案

1. 检查慢查询日志
2. 优化数据库索引
3. 调整连接池大小
4. 启用缓存
5. 扩容服务实例

### 数据不一致

#### 症状

- 任务状态不正确
- 审批记录缺失
- 数据不同步

#### 解决方案

1. 检查应用日志，查找错误信息
2. 验证数据库事务是否正确提交
3. 检查并发操作是否导致竞态条件
4. 必要时从备份恢复

## 升级和迁移

### 版本升级

#### 升级步骤

1. **备份数据库**: 执行完整备份
2. **停止服务**: 停止当前运行的服务
3. **更新代码**: 拉取新版本代码
4. **运行迁移**: 执行数据库迁移（如有）
5. **启动服务**: 启动新版本服务
6. **验证功能**: 验证关键功能是否正常

#### 回滚步骤

如果升级失败，执行回滚：

1. 停止新版本服务
2. 恢复数据库备份
3. 启动旧版本服务
4. 验证服务正常

### 数据迁移

#### 迁移到新服务器

1. 在新服务器上部署应用
2. 导出旧服务器数据
3. 导入到新服务器
4. 验证数据完整性
5. 切换流量到新服务器

#### 数据库迁移

```bash
# 导出数据
pg_dump -U postgres -d approval > migration_backup.sql

# 在新数据库导入
psql -U postgres -d approval_new < migration_backup.sql
```

## 安全维护

### 定期更新

1. **依赖更新**: 定期更新 Go 依赖包
2. **系统更新**: 保持操作系统更新
3. **安全补丁**: 及时应用安全补丁

### 访问控制

1. **API 认证**: 确保所有 API 都需要认证
2. **权限控制**: 验证权限检查是否生效
3. **审计日志**: 定期审查审计日志

### 敏感数据

1. **密码管理**: 使用强密码，定期更换
2. **密钥管理**: 妥善保管 API 密钥和证书
3. **数据加密**: 敏感数据加密存储

## 容量规划

### 资源监控

定期监控以下资源：

1. **CPU 使用率**: 应保持在 80% 以下
2. **内存使用率**: 应保持在 90% 以下
3. **磁盘使用率**: 应保持在 80% 以下
4. **数据库连接数**: 应保持在最大连接数的 80% 以下

### 扩容指标

当以下指标达到阈值时，考虑扩容：

1. **API 响应时间**: 95 分位延迟 > 1 秒
2. **错误率**: 错误率 > 1%
3. **数据库连接数**: 连接数 > 最大连接数的 80%
4. **CPU 使用率**: 持续 > 80%
5. **内存使用率**: 持续 > 90%

## 更新记录

- **2025-11-22**: 初始运维手册完成

