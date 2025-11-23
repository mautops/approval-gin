package config

import (
	"fmt"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// ConfigWatcher 配置监听器
type ConfigWatcher struct {
	config     *Config
	configPath string
	viper      *viper.Viper
	callbacks  []func(*Config)
	mu         sync.RWMutex
	stopped    bool
	stopMu     sync.RWMutex
}

// NewConfigWatcher 创建配置监听器
func NewConfigWatcher(cfg *Config, configPath string) *ConfigWatcher {
	v := viper.New()
	v.SetConfigFile(configPath)

	return &ConfigWatcher{
		config:     cfg,
		configPath: configPath,
		viper:      v,
		callbacks:  make([]func(*Config), 0),
		stopped:    false,
	}
}

// OnConfigChange 注册配置变更回调
func (w *ConfigWatcher) OnConfigChange(callback func(*Config)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.callbacks = append(w.callbacks, callback)
}

// Start 启动配置监听
func (w *ConfigWatcher) Start() error {
	// 读取配置文件
	if err := w.viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// 设置配置变更监听
	w.viper.WatchConfig()
	w.viper.OnConfigChange(func(e fsnotify.Event) {
		// 检查是否已停止
		w.stopMu.RLock()
		stopped := w.stopped
		w.stopMu.RUnlock()

		if stopped {
			return
		}

		// 重新加载配置
		var newCfg Config
		if err := w.viper.Unmarshal(&newCfg); err != nil {
			fmt.Printf("Failed to unmarshal config: %v\n", err)
			return
		}

		// 获取回调列表（需要加锁保护）
		w.mu.RLock()
		callbacks := make([]func(*Config), len(w.callbacks))
		copy(callbacks, w.callbacks)
		w.mu.RUnlock()

		// 调用所有回调（在锁外执行，避免死锁）
		for _, callback := range callbacks {
			callback(&newCfg)
		}

		// 更新当前配置（需要加锁保护）
		w.mu.Lock()
		w.config = &newCfg
		w.mu.Unlock()
	})

	return nil
}

// Stop 停止配置监听
func (w *ConfigWatcher) Stop() {
	w.stopMu.Lock()
	defer w.stopMu.Unlock()
	w.stopped = true
}

// GetConfig 获取当前配置
func (w *ConfigWatcher) GetConfig() *Config {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.config
}

