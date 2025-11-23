package config_test

import (
	"os"
	"testing"

	"github.com/mautops/approval-gin/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_EnvironmentVariables(t *testing.T) {
	// 设置环境变量
	os.Setenv("APP_SERVER_HOST", "0.0.0.0")
	os.Setenv("APP_SERVER_PORT", "9090")
	os.Setenv("APP_DATABASE_HOST", "db.example.com")
	os.Setenv("APP_DATABASE_PORT", "5433")
	os.Setenv("APP_DATABASE_USER", "testuser")
	os.Setenv("APP_DATABASE_PASSWORD", "testpass")
	os.Setenv("APP_DATABASE_DBNAME", "testdb")
	os.Setenv("APP_DATABASE_SSLMODE", "require")
	os.Setenv("APP_OPENFGA_API_URL", "http://openfga.example.com:8081")
	os.Setenv("APP_OPENFGA_STORE_ID", "test-store")
	os.Setenv("APP_OPENFGA_MODEL_ID", "test-model")
	os.Setenv("APP_KEYCLOAK_ISSUER", "https://keycloak.example.com/realms/test")
	os.Setenv("APP_KEYCLOAK_JWKS_URL", "https://keycloak.example.com/realms/test/protocol/openid-connect/certs")
	os.Setenv("APP_CORS_ALLOWED_ORIGINS", "https://example.com,https://app.example.com")
	os.Setenv("APP_CORS_ALLOWED_METHODS", "GET,POST")
	os.Setenv("APP_CORS_ALLOWED_HEADERS", "Content-Type,Authorization")
	os.Setenv("APP_CORS_MAX_AGE", "3600")

	defer func() {
		// 清理环境变量
		os.Unsetenv("APP_SERVER_HOST")
		os.Unsetenv("APP_SERVER_PORT")
		os.Unsetenv("APP_DATABASE_HOST")
		os.Unsetenv("APP_DATABASE_PORT")
		os.Unsetenv("APP_DATABASE_USER")
		os.Unsetenv("APP_DATABASE_PASSWORD")
		os.Unsetenv("APP_DATABASE_DBNAME")
		os.Unsetenv("APP_DATABASE_SSLMODE")
		os.Unsetenv("APP_OPENFGA_API_URL")
		os.Unsetenv("APP_OPENFGA_STORE_ID")
		os.Unsetenv("APP_OPENFGA_MODEL_ID")
		os.Unsetenv("APP_KEYCLOAK_ISSUER")
		os.Unsetenv("APP_KEYCLOAK_JWKS_URL")
		os.Unsetenv("APP_CORS_ALLOWED_ORIGINS")
		os.Unsetenv("APP_CORS_ALLOWED_METHODS")
		os.Unsetenv("APP_CORS_ALLOWED_HEADERS")
		os.Unsetenv("APP_CORS_MAX_AGE")
	}()

	// 加载配置（不提供配置文件路径，仅使用环境变量）
	cfg, err := config.Load("")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// 验证服务器配置
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 9090, cfg.Server.Port)

	// 验证数据库配置
	assert.Equal(t, "db.example.com", cfg.Database.Host)
	assert.Equal(t, 5433, cfg.Database.Port)
	assert.Equal(t, "testuser", cfg.Database.User)
	assert.Equal(t, "testpass", cfg.Database.Password)
	assert.Equal(t, "testdb", cfg.Database.DBName)
	assert.Equal(t, "require", cfg.Database.SSLMode)

	// 验证 OpenFGA 配置
	assert.Equal(t, "http://openfga.example.com:8081", cfg.OpenFGA.APIURL)
	assert.Equal(t, "test-store", cfg.OpenFGA.StoreID)
	assert.Equal(t, "test-model", cfg.OpenFGA.ModelID)

	// 验证 Keycloak 配置
	assert.Equal(t, "https://keycloak.example.com/realms/test", cfg.Keycloak.Issuer)
	assert.Equal(t, "https://keycloak.example.com/realms/test/protocol/openid-connect/certs", cfg.Keycloak.JWKSURL)

	// 验证 CORS 配置
	assert.Contains(t, cfg.CORS.AllowedOrigins, "https://example.com")
	assert.Contains(t, cfg.CORS.AllowedOrigins, "https://app.example.com")
	assert.Contains(t, cfg.CORS.AllowedMethods, "GET")
	assert.Contains(t, cfg.CORS.AllowedMethods, "POST")
	assert.Contains(t, cfg.CORS.AllowedHeaders, "Content-Type")
	assert.Contains(t, cfg.CORS.AllowedHeaders, "Authorization")
	assert.Equal(t, 3600, cfg.CORS.MaxAge)
}

func TestConfig_EnvironmentVariablesWithDefaults(t *testing.T) {
	// 不设置环境变量，使用默认值
	cfg, err := config.Load("")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// 验证默认值
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "postgres", cfg.Database.User)
	assert.Equal(t, "approval", cfg.Database.DBName)
	assert.Equal(t, "disable", cfg.Database.SSLMode)
}

func TestConfig_EnvironmentVariablesOverrideConfigFile(t *testing.T) {
	// 设置环境变量
	os.Setenv("APP_SERVER_PORT", "9999")
	defer os.Unsetenv("APP_SERVER_PORT")

	// 加载配置（环境变量应该覆盖配置文件）
	cfg, err := config.Load("")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// 验证环境变量覆盖了默认值
	assert.Equal(t, 9999, cfg.Server.Port)
}

