package tests

import (
	"context"
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/config"
	"github.com/mautops/approval-gin/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDatabaseConnectionRetry(t *testing.T) {
	// 测试数据库连接重试机制
	// 使用无效的数据库配置测试重试逻辑
	cfg := config.DatabaseConfig{
		Host:     "invalid-host",
		Port:     5432,
		User:     "invalid-user",
		Password: "invalid-password",
		DBName:   "invalid-db",
		SSLMode:  "disable",
	}

	// 测试连接失败时返回错误（不阻塞）
	_, err := database.ConnectWithRetry(cfg, 3, time.Second)
	assert.Error(t, err, "should return error for invalid connection")
}

func TestDatabaseConnectionRecovery(t *testing.T) {
	// 测试数据库连接恢复机制
	// 使用 SQLite 内存数据库模拟正常连接
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 测试连接健康检查
	sqlDB, err := db.DB()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = sqlDB.PingContext(ctx)
	assert.NoError(t, err, "database should be healthy")

	// 测试连接恢复（关闭后重新打开）
	sqlDB.Close()

	// 重新连接
	db2, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB2, err := db2.DB()
	require.NoError(t, err)

	err = sqlDB2.PingContext(ctx)
	assert.NoError(t, err, "recovered database should be healthy")
}

func TestDatabaseHealthCheck(t *testing.T) {
	// 测试数据库健康检查
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 测试健康检查函数
	healthy := database.CheckHealth(db)
	assert.True(t, healthy, "database should be healthy")

	// 关闭连接后检查
	sqlDB, _ := db.DB()
	sqlDB.Close()

	healthy = database.CheckHealth(db)
	assert.False(t, healthy, "closed database should be unhealthy")
}

