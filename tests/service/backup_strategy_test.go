package service_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestBackupStrategy_DefaultConfig 测试默认备份策略配置
func TestBackupStrategy_DefaultConfig(t *testing.T) {
	backupSvc := createTestBackupService(t)
	// 传入 nil 使用默认配置
	scheduler := service.NewBackupScheduler(backupSvc, nil)

	assert.NotNil(t, scheduler)
	assert.True(t, scheduler.Config().FullBackupEnabled)
	assert.Equal(t, "0 0 * * *", scheduler.Config().FullBackupSchedule)
	assert.Equal(t, 30, scheduler.Config().FullBackupRetentionDays)
}

// TestBackupStrategy_FullBackupSchedule 测试全量备份计划
func TestBackupStrategy_FullBackupSchedule(t *testing.T) {
	backupSvc := createTestBackupService(t)
	config := &service.BackupScheduleConfig{
		FullBackupEnabled:       true,
		FullBackupSchedule:      "0 2 * * *", // 每天凌晨 2 点
		FullBackupRetentionDays: 30,
		VerifyBackup:            true,
	}
	scheduler := service.NewBackupScheduler(backupSvc, config)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := scheduler.Start(ctx)
	require.NoError(t, err)
	defer scheduler.Stop()

	// 等待备份执行
	time.Sleep(2 * time.Second)

	// 验证备份文件存在
	backups, err := backupSvc.ListBackups(ctx)
	assert.NoError(t, err)
	assert.Greater(t, len(backups), 0, "should have at least one backup")
}

// TestBackupStrategy_IncrementalBackupSchedule 测试增量备份计划
func TestBackupStrategy_IncrementalBackupSchedule(t *testing.T) {
	backupSvc := createTestBackupService(t)
	config := &service.BackupScheduleConfig{
		FullBackupEnabled:              true,
		FullBackupSchedule:             "0 0 * * *",
		FullBackupRetentionDays:        30,
		IncrementalBackupEnabled:       true,
		IncrementalBackupInterval:      1 * time.Hour,
		IncrementalBackupRetentionDays: 7,
		VerifyBackup:                   true,
	}
	scheduler := service.NewBackupScheduler(backupSvc, config)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := scheduler.Start(ctx)
	require.NoError(t, err)
	defer scheduler.Stop()

	// 等待增量备份执行
	time.Sleep(2 * time.Second)

	// 验证备份文件存在
	backups, err := backupSvc.ListBackups(ctx)
	assert.NoError(t, err)
	assert.Greater(t, len(backups), 0, "should have at least one backup")
}

// TestBackupStrategy_BackupRetention 测试备份保留策略
func TestBackupStrategy_BackupRetention(t *testing.T) {
	backupSvc := createTestBackupService(t)
	config := &service.BackupScheduleConfig{
		FullBackupEnabled:       true,
		FullBackupSchedule:      "0 0 * * *",
		FullBackupRetentionDays: 7, // 只保留 7 天
		VerifyBackup:            true,
	}
	scheduler := service.NewBackupScheduler(backupSvc, config)

	ctx := context.Background()

	// 创建一些旧备份（8 天前）
	oldBackupPath := filepath.Join(backupSvc.BackupDir(), "backup_sqlite_20250101_000000.tar.gz")
	err := os.WriteFile(oldBackupPath, []byte("old backup"), 0644)
	require.NoError(t, err)

	// 设置文件修改时间为 8 天前
	oldTime := time.Now().Add(-8 * 24 * time.Hour)
	err = os.Chtimes(oldBackupPath, oldTime, oldTime)
	require.NoError(t, err)

	// 执行清理
	scheduler.CleanupOldBackups(ctx)

	// 验证旧备份被删除
	_, err = os.Stat(oldBackupPath)
	assert.True(t, os.IsNotExist(err), "old backup should be deleted")
}

// TestBackupStrategy_BackupVerification 测试备份验证
func TestBackupStrategy_BackupVerification(t *testing.T) {
	backupSvc := createTestBackupService(t)
	config := &service.BackupScheduleConfig{
		FullBackupEnabled:       true,
		FullBackupSchedule:      "0 0 * * *",
		FullBackupRetentionDays: 30,
		VerifyBackup:            true, // 启用备份验证
	}
	scheduler := service.NewBackupScheduler(backupSvc, config)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := scheduler.Start(ctx)
	require.NoError(t, err)
	defer scheduler.Stop()

	// 等待备份执行
	time.Sleep(2 * time.Second)

	// 验证备份文件存在且有效
	backups, err := backupSvc.ListBackups(ctx)
	assert.NoError(t, err)
	assert.Greater(t, len(backups), 0)

	// 验证备份文件大小大于 0
	for _, backup := range backups {
		backupPath := filepath.Join(backupSvc.BackupDir(), backup.Filename)
		info, err := os.Stat(backupPath)
		assert.NoError(t, err)
		assert.Greater(t, info.Size(), int64(0), "backup file should not be empty")
	}
}

// createTestBackupService 创建测试用的备份服务
func createTestBackupService(t *testing.T) *service.BackupService {
	// 使用 SQLite 内存数据库
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	
	backupDir := t.TempDir()
	return service.NewBackupService(db, backupDir)
}
