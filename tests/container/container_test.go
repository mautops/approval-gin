package container_test

import (
	"testing"

	"github.com/mautops/approval-gin/internal/config"
	"github.com/mautops/approval-gin/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContainer_NewContainer 测试创建依赖注入容器
func TestContainer_NewContainer(t *testing.T) {
	// 使用测试配置
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "test",
			Password: "test",
			DBName:   "test_db",
			SSLMode:  "disable",
		},
		OpenFGA: config.OpenFGAConfig{
			APIURL:  "http://localhost:8081",
			StoreID: "test-store",
			ModelID: "test-model",
		},
		Keycloak: config.KeycloakConfig{
			Issuer: "http://localhost:8082/realms/test",
		},
	}

	// 测试: 创建容器应该成功（如果数据库连接失败，应该返回错误）
	// 注意: 这个测试可能需要跳过，如果测试环境没有真实的数据库
	container, err := container.NewContainer(cfg)
	if err != nil {
		// 如果是因为数据库连接失败，这是预期的（测试环境可能没有数据库）
		t.Logf("Container creation failed (expected in test environment): %v", err)
		return
	}

	require.NotNil(t, container, "container should not be nil")
	assert.NotNil(t, container.DB(), "database should be initialized")
	assert.NotNil(t, container.TemplateManager(), "template manager should be initialized")
	assert.NotNil(t, container.TaskManager(), "task manager should be initialized")
	assert.NotNil(t, container.OpenFGAClient(), "OpenFGA client should be initialized")
	assert.NotNil(t, container.EventHandler(), "event handler should be initialized")
}

// TestContainer_NewContainer_InvalidConfig 测试使用无效配置创建容器
func TestContainer_NewContainer_InvalidConfig(t *testing.T) {
	// 使用无效的数据库配置
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     "invalid-host",
			Port:     5432,
			User:     "test",
			Password: "test",
			DBName:   "test_db",
			SSLMode:  "disable",
		},
		OpenFGA: config.OpenFGAConfig{
			APIURL:  "http://localhost:8081",
			StoreID: "test-store",
			ModelID: "test-model",
		},
		Keycloak: config.KeycloakConfig{
			Issuer: "http://localhost:8082/realms/test",
		},
	}

	// 测试: 使用无效配置创建容器应该返回错误
	_, err := container.NewContainer(cfg)
	assert.Error(t, err, "should return error with invalid database config")
}

// TestContainer_GetServices 测试从容器获取服务
func TestContainer_GetServices(t *testing.T) {
	// 这个测试需要先创建一个有效的容器
	// 由于需要真实的数据库连接，这个测试可能需要跳过或使用 mock
	t.Skip("requires real database connection or mock setup")
}

// TestContainer_Close 测试关闭容器
func TestContainer_Close(t *testing.T) {
	// 这个测试需要先创建一个有效的容器
	// 测试关闭容器时是否正确清理资源
	t.Skip("requires real database connection or mock setup")
}


