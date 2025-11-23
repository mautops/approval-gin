package config_test

import (
	"os"
	"testing"

	"github.com/mautops/approval-gin/internal/config"
	"github.com/stretchr/testify/assert"
)

// TestProductionConfig_LoadFromEnv 测试从环境变量加载生产环境配置
func TestProductionConfig_LoadFromEnv(t *testing.T) {
	// 设置环境变量
	os.Setenv("APP_ENV", "production")
	os.Setenv("APP_DATABASE_MAX_IDLE_CONNS", "20")
	os.Setenv("APP_DATABASE_MAX_OPEN_CONNS", "200")
	os.Setenv("APP_DATABASE_CONN_MAX_LIFETIME", "3600")
	os.Setenv("APP_DATABASE_CONN_MAX_IDLE_TIME", "300")
	os.Setenv("APP_LOG_LEVEL", "warn")
	defer func() {
		os.Unsetenv("APP_ENV")
		os.Unsetenv("APP_DATABASE_MAX_IDLE_CONNS")
		os.Unsetenv("APP_DATABASE_MAX_OPEN_CONNS")
		os.Unsetenv("APP_DATABASE_CONN_MAX_LIFETIME")
		os.Unsetenv("APP_DATABASE_CONN_MAX_IDLE_TIME")
		os.Unsetenv("APP_LOG_LEVEL")
	}()

	cfg, err := config.Load("")
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// 验证数据库连接池配置
	assert.Equal(t, 20, cfg.Database.MaxIdleConns)
	assert.Equal(t, 200, cfg.Database.MaxOpenConns)
	assert.Equal(t, 3600, cfg.Database.ConnMaxLifetime)
	assert.Equal(t, 300, cfg.Database.ConnMaxIdleTime)

	// 验证日志级别
	assert.Equal(t, "warn", cfg.Log.Level)
}

// TestProductionConfig_DefaultValues 测试生产环境默认值
func TestProductionConfig_DefaultValues(t *testing.T) {
	// 设置生产环境
	os.Setenv("APP_ENV", "production")
	defer os.Unsetenv("APP_ENV")

	cfg, err := config.Load("")
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// 验证生产环境默认值
	assert.Equal(t, 20, cfg.Database.MaxIdleConns)
	assert.Equal(t, 200, cfg.Database.MaxOpenConns)
	assert.Equal(t, 3600, cfg.Database.ConnMaxLifetime)
	assert.Equal(t, 300, cfg.Database.ConnMaxIdleTime)
	assert.Equal(t, "warn", cfg.Log.Level)
}

// TestProductionConfig_IsProduction 测试判断是否为生产环境
func TestProductionConfig_IsProduction(t *testing.T) {
	os.Setenv("APP_ENV", "production")
	defer os.Unsetenv("APP_ENV")

	cfg, err := config.Load("")
	assert.NoError(t, err)
	assert.True(t, config.IsProduction(cfg))
}

// TestProductionConfig_IsNotProduction 测试非生产环境
func TestProductionConfig_IsNotProduction(t *testing.T) {
	os.Setenv("APP_ENV", "development")
	defer os.Unsetenv("APP_ENV")

	cfg, err := config.Load("")
	assert.NoError(t, err)
	assert.False(t, config.IsProduction(cfg))
}
