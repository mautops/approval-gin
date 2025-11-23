# 备份策略指南

本文档说明如何配置和使用 Approval Gin 的自动备份功能。

## 概述

Approval Gin 提供自动备份功能，支持：

- **全量备份**: 定期创建完整数据库备份
- **增量备份**: 定期创建增量备份（可选）
- **备份验证**: 验证备份文件完整性
- **自动清理**: 自动删除过期备份

## 备份配置

### 配置文件

在 `config.yaml` 中配置备份策略：

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
  compression: true
```

### 环境变量

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

## 备份类型

### 全量备份

全量备份包含完整的数据库数据，适合：

- 定期完整备份
- 灾难恢复
- 数据迁移

**配置建议**:
- 频率: 每天一次（凌晨执行）
- 保留期: 30 天
- 压缩: 启用

### 增量备份

增量备份只包含自上次备份以来的变更，适合：

- 频繁备份
- 减少存储空间
- 快速恢复最近数据

**配置建议**:
- 频率: 每小时一次
- 保留期: 7 天
- 压缩: 启用

## 备份文件格式

备份文件命名格式：`backup_{database_type}_{timestamp}.tar.gz`

示例：
- `backup_postgres_20251122_000000.tar.gz`
- `backup_sqlite_20251122_120000.tar.gz`

## 备份验证

启用备份验证后，系统会：

1. 检查备份文件大小（不为空）
2. 验证文件完整性
3. 记录验证结果

## 备份保留策略

系统自动清理过期备份：

- **全量备份**: 超过保留期（默认 30 天）的备份会被删除
- **增量备份**: 超过保留期（默认 7 天）的备份会被删除
- **清理频率**: 每天执行一次

## 使用示例

### 启动备份调度器

```go
import (
    "context"
    "github.com/mautops/approval-gin/internal/service"
)

// 创建备份服务
backupSvc := service.NewBackupService(db, "/var/backups/approval-gin")

// 配置备份策略
config := &service.BackupScheduleConfig{
    FullBackupEnabled:       true,
    FullBackupSchedule:      "0 0 * * *",
    FullBackupRetentionDays: 30,
    IncrementalBackupEnabled: true,
    IncrementalBackupInterval: time.Hour,
    IncrementalBackupRetentionDays: 7,
    VerifyBackup: true,
}

// 创建并启动调度器
scheduler := service.NewBackupScheduler(backupSvc, config)
ctx := context.Background()
scheduler.Start(ctx)
defer scheduler.Stop()
```

### 手动创建备份

```go
backupPath, err := backupSvc.CreateBackup(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Backup created: %s\n", backupPath)
```

### 列出备份

```go
backups, err := backupSvc.ListBackups(ctx)
if err != nil {
    log.Fatal(err)
}

for _, backup := range backups {
    fmt.Printf("Backup: %s, Size: %d, Created: %s\n",
        backup.Filename, backup.Size, backup.CreatedAt)
}
```

### 恢复备份

```go
err := backupSvc.RestoreBackup(ctx, "/var/backups/approval-gin/backup_postgres_20251122_000000.tar.gz")
if err != nil {
    log.Fatal(err)
}
fmt.Println("Backup restored successfully")
```

## 生产环境建议

### 备份策略

1. **全量备份**: 每天凌晨执行，保留 30 天
2. **增量备份**: 每小时执行，保留 7 天
3. **备份验证**: 启用备份验证
4. **异地备份**: 定期将备份复制到异地存储

### 存储配置

1. **备份目录**: 使用独立的存储卷，避免影响应用数据
2. **磁盘空间**: 确保有足够的磁盘空间（建议至少 3 倍数据库大小）
3. **备份监控**: 监控备份目录磁盘使用率

### 安全配置

1. **文件权限**: 备份文件应设置适当的权限（建议 0600）
2. **加密**: 敏感数据备份应加密存储
3. **访问控制**: 限制备份目录的访问权限

## 故障排查

### 备份失败

1. 检查磁盘空间是否充足
2. 检查备份目录权限
3. 检查数据库连接是否正常
4. 查看应用日志获取详细错误信息

### 备份文件损坏

1. 启用备份验证
2. 定期测试备份恢复
3. 使用校验和验证文件完整性

### 备份清理失败

1. 检查备份目录权限
2. 检查文件是否被其他进程占用
3. 查看应用日志获取详细错误信息

## 最佳实践

1. **定期测试恢复**: 定期测试备份恢复流程，确保备份可用
2. **监控备份状态**: 监控备份执行状态，及时发现备份失败
3. **异地存储**: 将备份复制到异地存储，防止单点故障
4. **版本控制**: 保留多个版本的备份，便于回滚
5. **文档记录**: 记录备份和恢复流程，便于操作

## 更新记录

- **2025-11-22**: 初始备份策略配置完成

