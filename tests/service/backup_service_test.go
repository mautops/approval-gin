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

func TestBackupService_CreateBackup(t *testing.T) {
	// 创建测试数据库
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	// 创建备份服务
	backupService := service.NewBackupService(db, "/tmp/backups")

	// 创建备份
	backupPath, err := backupService.CreateBackup(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, backupPath)

	// 验证备份文件存在
	_, err = os.Stat(backupPath)
	assert.NoError(t, err)

	// 清理
	os.Remove(backupPath)
}

func TestBackupService_RestoreBackup(t *testing.T) {
	// 创建源数据库并插入测试数据
	sourceDB, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	// 创建备份服务
	backupService := service.NewBackupService(sourceDB, "/tmp/backups")

	// 创建备份
	backupPath, err := backupService.CreateBackup(context.Background())
	require.NoError(t, err)
	defer os.Remove(backupPath)

	// 创建目标数据库
	targetDB, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	// 创建恢复服务
	restoreService := service.NewBackupService(targetDB, "/tmp/backups")

	// 恢复备份
	err = restoreService.RestoreBackup(context.Background(), backupPath)
	require.NoError(t, err)

	// 验证数据已恢复（这里可以添加具体的数据验证逻辑）
}

func TestBackupService_ListBackups(t *testing.T) {
	// 创建测试数据库
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	// 使用临时目录
	tmpDir := t.TempDir()
	backupService := service.NewBackupService(db, tmpDir)

	// 创建多个备份（添加延迟确保时间戳不同）
	backup1, err := backupService.CreateBackup(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, backup1)

	// 等待一下，确保时间戳不同
	time.Sleep(1 * time.Second)

	backup2, err := backupService.CreateBackup(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, backup2)

	// 验证两个备份文件不同
	assert.NotEqual(t, backup1, backup2, "backup files should be different")

	// 等待一下，确保文件系统同步
	time.Sleep(100 * time.Millisecond)

	// 列出备份
	backups, err := backupService.ListBackups(context.Background())
	require.NoError(t, err)
	// 应该至少包含我们刚创建的两个备份
	assert.GreaterOrEqual(t, len(backups), 2, "should have at least 2 backups, got %d", len(backups))
}

func TestBackupService_DeleteBackup(t *testing.T) {
	// 创建测试数据库
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	// 创建备份服务
	backupService := service.NewBackupService(db, "/tmp/backups")

	// 创建备份
	backupPath, err := backupService.CreateBackup(context.Background())
	require.NoError(t, err)

	// 验证备份文件存在
	_, err = os.Stat(backupPath)
	assert.NoError(t, err)

	// 删除备份
	err = backupService.DeleteBackup(context.Background(), filepath.Base(backupPath))
	require.NoError(t, err)

	// 验证备份文件已删除
	_, err = os.Stat(backupPath)
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

