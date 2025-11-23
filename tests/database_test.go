package tests

import (
	"strings"
	"testing"

	"github.com/mautops/approval-gin/internal/config"
	"github.com/mautops/approval-gin/internal/database"
)

// TestDatabaseConnection 测试数据库连接配置
func TestDatabaseConnection(t *testing.T) {
	// 使用测试配置
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "test",
			DBName:   "test_approval",
			SSLMode:  "disable",
		},
	}
	
	// 测试连接配置生成
	dsn := database.BuildDSN(cfg.Database)
	if dsn == "" {
		t.Error("DSN should not be empty")
	}
	
	// 验证 DSN 包含必要的组件
	if !strings.Contains(dsn, "host=localhost") {
		t.Error("DSN should contain host")
	}
	if !strings.Contains(dsn, "user=postgres") {
		t.Error("DSN should contain user")
	}
	if !strings.Contains(dsn, "dbname=test_approval") {
		t.Error("DSN should contain dbname")
	}
}

// TestDatabaseConnectionPool 测试连接池配置
func TestDatabaseConnectionPool(t *testing.T) {
	// 测试连接池配置
	poolConfig := database.GetPoolConfig()
	if poolConfig == nil {
		t.Error("Pool config should not be nil")
	}
	
	// 验证连接池参数
	if poolConfig.MaxIdleConns <= 0 {
		t.Error("MaxIdleConns should be greater than 0")
	}
	if poolConfig.MaxOpenConns <= 0 {
		t.Error("MaxOpenConns should be greater than 0")
	}
}

