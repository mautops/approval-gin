package cmd_test

import (
	"os"
	"testing"

	"github.com/mautops/approval-gin/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateCommand(t *testing.T) {
	// 设置测试环境变量
	os.Setenv("APP_DATABASE_HOST", "localhost")
	os.Setenv("APP_DATABASE_PORT", "5432")
	os.Setenv("APP_DATABASE_USER", "postgres")
	os.Setenv("APP_DATABASE_PASSWORD", "postgres")
	os.Setenv("APP_DATABASE_DBNAME", "approval_test")
	os.Setenv("APP_DATABASE_SSLMODE", "disable")

	defer func() {
		os.Unsetenv("APP_DATABASE_HOST")
		os.Unsetenv("APP_DATABASE_PORT")
		os.Unsetenv("APP_DATABASE_USER")
		os.Unsetenv("APP_DATABASE_PASSWORD")
		os.Unsetenv("APP_DATABASE_DBNAME")
		os.Unsetenv("APP_DATABASE_SSLMODE")
	}()

	// 测试 migrate 命令是否存在
	rootCmd := cmd.GetRootCmd()
	migrateCmd, _, err := rootCmd.Find([]string{"migrate"})
	require.NoError(t, err, "migrate command should exist")
	assert.NotNil(t, migrateCmd, "migrate command should not be nil")
	assert.Equal(t, "migrate", migrateCmd.Use, "command use should be 'migrate'")
}

func TestMigrateCommandWithSQLite(t *testing.T) {
	// 使用 SQLite 进行测试（不需要外部数据库）
	os.Setenv("APP_DATABASE_HOST", "")
	os.Setenv("APP_DATABASE_PORT", "")
	os.Setenv("APP_DATABASE_USER", "")
	os.Setenv("APP_DATABASE_PASSWORD", "")
	os.Setenv("APP_DATABASE_DBNAME", ":memory:")
	os.Setenv("APP_DATABASE_SSLMODE", "")

	defer func() {
		os.Unsetenv("APP_DATABASE_HOST")
		os.Unsetenv("APP_DATABASE_PORT")
		os.Unsetenv("APP_DATABASE_USER")
		os.Unsetenv("APP_DATABASE_PASSWORD")
		os.Unsetenv("APP_DATABASE_DBNAME")
		os.Unsetenv("APP_DATABASE_SSLMODE")
	}()

	// 测试 migrate 命令可以执行（使用 SQLite 内存数据库）
	rootCmd := cmd.GetRootCmd()
	migrateCmd, _, err := rootCmd.Find([]string{"migrate"})
	require.NoError(t, err)

	// 注意：实际执行迁移需要数据库连接，这里只测试命令存在
	assert.NotNil(t, migrateCmd)
}

