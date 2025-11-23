package config_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigWatcher_WatchConfigFile(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// 写入初始配置
	initialConfig := `
server:
  host: "0.0.0.0"
  port: 8080
database:
  host: "localhost"
  port: 5432
`
	err := os.WriteFile(configPath, []byte(initialConfig), 0644)
	require.NoError(t, err)

	// 加载配置
	cfg, err := config.Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, 8080, cfg.Server.Port)

	// 创建配置监听器
	watcher := config.NewConfigWatcher(cfg, configPath)
	var mu sync.Mutex
	callbackCalled := false
	var newConfig *config.Config

	// 注册回调
	watcher.OnConfigChange(func(cfg *config.Config) {
		mu.Lock()
		defer mu.Unlock()
		callbackCalled = true
		newConfig = cfg
	})

	// 启动监听
	err = watcher.Start()
	require.NoError(t, err)
	defer watcher.Stop()

	// 等待一下，确保监听器启动
	time.Sleep(100 * time.Millisecond)

	// 修改配置文件
	updatedConfig := `
server:
  host: "0.0.0.0"
  port: 9090
database:
  host: "localhost"
  port: 5432
`
	err = os.WriteFile(configPath, []byte(updatedConfig), 0644)
	require.NoError(t, err)

	// 等待配置变更被检测到
	time.Sleep(500 * time.Millisecond)

	// 验证回调被调用（需要加锁读取）
	mu.Lock()
	wasCalled := callbackCalled
	newCfg := newConfig
	mu.Unlock()
	
	assert.True(t, wasCalled, "config change callback should be called")
	assert.NotNil(t, newCfg)
	assert.Equal(t, 9090, newCfg.Server.Port)
}

func TestConfigWatcher_MultipleCallbacks(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// 写入初始配置
	initialConfig := `
server:
  host: "0.0.0.0"
  port: 8080
`
	err := os.WriteFile(configPath, []byte(initialConfig), 0644)
	require.NoError(t, err)

	// 加载配置
	cfg, err := config.Load(configPath)
	require.NoError(t, err)

	// 创建配置监听器
	watcher := config.NewConfigWatcher(cfg, configPath)
	var mu sync.Mutex
	callback1Called := false
	callback2Called := false

	// 注册多个回调
	watcher.OnConfigChange(func(cfg *config.Config) {
		mu.Lock()
		defer mu.Unlock()
		callback1Called = true
	})
	watcher.OnConfigChange(func(cfg *config.Config) {
		mu.Lock()
		defer mu.Unlock()
		callback2Called = true
	})

	// 启动监听
	err = watcher.Start()
	require.NoError(t, err)
	defer watcher.Stop()

	// 等待一下，确保监听器启动
	time.Sleep(100 * time.Millisecond)

	// 修改配置文件
	updatedConfig := `
server:
  host: "0.0.0.0"
  port: 9090
`
	err = os.WriteFile(configPath, []byte(updatedConfig), 0644)
	require.NoError(t, err)

	// 等待配置变更被检测到
	time.Sleep(500 * time.Millisecond)

	// 验证所有回调都被调用（需要加锁读取）
	mu.Lock()
	wasCalled1 := callback1Called
	wasCalled2 := callback2Called
	mu.Unlock()
	
	assert.True(t, wasCalled1, "first callback should be called")
	assert.True(t, wasCalled2, "second callback should be called")
}

func TestConfigWatcher_Stop(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// 写入初始配置
	initialConfig := `
server:
  host: "0.0.0.0"
  port: 8080
`
	err := os.WriteFile(configPath, []byte(initialConfig), 0644)
	require.NoError(t, err)

	// 加载配置
	cfg, err := config.Load(configPath)
	require.NoError(t, err)

	// 创建配置监听器
	watcher := config.NewConfigWatcher(cfg, configPath)
	var mu sync.Mutex
	callbackCalled := false

	// 注册回调
	watcher.OnConfigChange(func(cfg *config.Config) {
		mu.Lock()
		defer mu.Unlock()
		callbackCalled = true
	})

	// 启动监听
	err = watcher.Start()
	require.NoError(t, err)

	// 停止监听
	watcher.Stop()

	// 等待一下
	time.Sleep(100 * time.Millisecond)

	// 修改配置文件
	updatedConfig := `
server:
  host: "0.0.0.0"
  port: 9090
`
	err = os.WriteFile(configPath, []byte(updatedConfig), 0644)
	require.NoError(t, err)

	// 等待一下
	time.Sleep(500 * time.Millisecond)

	// 验证回调未被调用（因为已停止）（需要加锁读取）
	mu.Lock()
	wasCalled := callbackCalled
	mu.Unlock()
	
	assert.False(t, wasCalled, "callback should not be called after stop")
}

