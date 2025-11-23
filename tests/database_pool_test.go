package tests

import (
	"testing"

	"github.com/mautops/approval-gin/internal/database"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestDatabaseConnectionPoolConfig 测试数据库连接池配置
func TestDatabaseConnectionPoolConfig(t *testing.T) {
	// 使用 SQLite 内存数据库进行测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	
	// 获取连接池配置
	poolConfig := database.GetPoolConfig()
	assert.NotNil(t, poolConfig)
	assert.Greater(t, poolConfig.MaxIdleConns, 0)
	assert.Greater(t, poolConfig.MaxOpenConns, 0)
	assert.Greater(t, poolConfig.ConnMaxLifetime, 0)
	assert.Greater(t, poolConfig.ConnMaxIdleTime, 0)
	
	// 获取底层 sql.DB
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	
	// 应用连接池配置
	sqlDB.SetMaxIdleConns(poolConfig.MaxIdleConns)
	sqlDB.SetMaxOpenConns(poolConfig.MaxOpenConns)
	
	// 验证配置可以设置(通过检查没有错误)
	stats := sqlDB.Stats()
	assert.NotNil(t, stats)
	// 注意: 这里只是验证配置可以设置,实际配置值无法直接获取
}

// TestDatabaseConnectionPoolDefaults 测试连接池配置参数
func TestDatabaseConnectionPoolDefaults(t *testing.T) {
	poolConfig := database.GetPoolConfig()
	
	// 验证默认配置值
	assert.Equal(t, 10, poolConfig.MaxIdleConns)
	assert.Equal(t, 100, poolConfig.MaxOpenConns)
	assert.Equal(t, 3600, poolConfig.ConnMaxLifetime) // 1 小时
	assert.Equal(t, 600, poolConfig.ConnMaxIdleTime)  // 10 分钟
}

