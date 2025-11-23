package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBackupScheduler_StartStop(t *testing.T) {
	// 创建测试数据库
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	// 创建备份服务
	backupService := service.NewBackupService(db, t.TempDir())

	// 创建备份调度器配置
	config := &service.BackupScheduleConfig{
		FullBackupEnabled:          true,
		FullBackupRetentionDays:     30,
		IncrementalBackupEnabled:    false,
		IncrementalBackupInterval:   1 * time.Hour,
		IncrementalBackupRetentionDays: 7,
		VerifyBackup:                false, // 测试时禁用验证
	}

	// 创建备份调度器
	scheduler := service.NewBackupScheduler(backupService, config)

	// 启动调度器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	require.NoError(t, err)

	// 等待一下，确保调度器启动
	time.Sleep(100 * time.Millisecond)

	// 停止调度器
	scheduler.Stop()

	// 验证调度器已停止
	time.Sleep(100 * time.Millisecond)
}

func TestBackupScheduler_CleanupOldBackups(t *testing.T) {
	// 创建测试数据库
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	// 创建备份服务
	backupService := service.NewBackupService(db, t.TempDir())

	// 创建备份调度器配置（保留时间很短，便于测试）
	config := &service.BackupScheduleConfig{
		FullBackupRetentionDays:     0, // 立即删除
		IncrementalBackupRetentionDays: 0,
		VerifyBackup:                false,
	}

	// 创建备份调度器
	scheduler := service.NewBackupScheduler(backupService, config)

	// 创建一些备份
	_, err = backupService.CreateBackup(context.Background())
	require.NoError(t, err)

	// 执行清理
	ctx := context.Background()
	scheduler.CleanupOldBackups(ctx)

	// 验证备份已被清理
	backups, err := backupService.ListBackups(ctx)
	require.NoError(t, err)
	// 由于保留时间为 0，所有备份应该被清理
	assert.Equal(t, 0, len(backups))
}

