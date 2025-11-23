package tests

import (
	"os"
	"testing"

	"github.com/mautops/approval-gin/internal/config"
)

// TestLoadConfigFromFile 测试从配置文件加载配置
func TestLoadConfigFromFile(t *testing.T) {
	// 创建临时配置文件
	configContent := `
server:
  host: "0.0.0.0"
  port: 8080
database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "password"
  dbname: "approval"
  sslmode: "disable"
`
	
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	
	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()
	
	// 加载配置
	cfg, err := config.Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	// 验证配置值
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Expected server host '0.0.0.0', got '%s'", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected server port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("Expected database host 'localhost', got '%s'", cfg.Database.Host)
	}
}

// TestLoadConfigFromEnv 测试从环境变量加载配置
func TestLoadConfigFromEnv(t *testing.T) {
	// 设置环境变量
	os.Setenv("APP_SERVER_HOST", "127.0.0.1")
	os.Setenv("APP_SERVER_PORT", "9090")
	os.Setenv("APP_DATABASE_HOST", "db.example.com")
	defer func() {
		os.Unsetenv("APP_SERVER_HOST")
		os.Unsetenv("APP_SERVER_PORT")
		os.Unsetenv("APP_DATABASE_HOST")
	}()
	
	// 加载配置(使用环境变量)
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Failed to load config from env: %v", err)
	}
	
	// 验证环境变量值
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Expected server host '127.0.0.1' from env, got '%s'", cfg.Server.Host)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Expected server port 9090 from env, got %d", cfg.Server.Port)
	}
}

// TestConfigDefaults 测试配置默认值
func TestConfigDefaults(t *testing.T) {
	cfg := config.Default()
	
	if cfg.Server.Host == "" {
		t.Error("Server host should have a default value")
	}
	if cfg.Server.Port == 0 {
		t.Error("Server port should have a default value")
	}
}

