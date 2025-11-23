package api_test

import (
	"testing"

	"github.com/mautops/approval-gin/internal/api"
	"github.com/stretchr/testify/assert"
)

func TestI18nResourceManager_LoadMessages(t *testing.T) {
	manager := api.NewI18nManager()

	// 加载英文消息
	manager.LoadMessages("en", map[string]string{
		"test.key1": "Test value 1",
		"test.key2": "Test value 2",
	})

	// 加载中文消息
	manager.LoadMessages("zh", map[string]string{
		"test.key1": "测试值 1",
		"test.key2": "测试值 2",
	})

	// 测试英文翻译
	assert.Equal(t, "Test value 1", manager.Translate("en", "test.key1"))
	assert.Equal(t, "Test value 2", manager.Translate("en", "test.key2"))

	// 测试中文翻译
	assert.Equal(t, "测试值 1", manager.Translate("zh", "test.key1"))
	assert.Equal(t, "测试值 2", manager.Translate("zh", "test.key2"))
}

func TestI18nResourceManager_FallbackToEnglish(t *testing.T) {
	manager := api.NewI18nManager()

	// 只加载英文消息
	manager.LoadMessages("en", map[string]string{
		"test.key": "Test value",
	})

	// 测试中文翻译（应该回退到英文）
	assert.Equal(t, "Test value", manager.Translate("zh", "test.key"))
}

func TestI18nResourceManager_MissingKey(t *testing.T) {
	manager := api.NewI18nManager()

	// 加载消息
	manager.LoadMessages("en", map[string]string{
		"test.key": "Test value",
	})

	// 测试不存在的 key（应该返回 key 本身）
	assert.Equal(t, "test.missing", manager.Translate("en", "test.missing"))
}

func TestI18nResourceManager_LoadFromFile(t *testing.T) {
	// 这个测试可以验证从文件加载语言资源
	// 实际实现中可以使用 YAML 或 JSON 文件
	manager := api.NewI18nManager()

	// 模拟从文件加载
	enMessages := map[string]string{
		"error.not_found": "Resource not found",
		"error.unauthorized": "Unauthorized",
	}
	manager.LoadMessages("en", enMessages)

	// 验证加载的消息
	assert.Equal(t, "Resource not found", manager.Translate("en", "error.not_found"))
	assert.Equal(t, "Unauthorized", manager.Translate("en", "error.unauthorized"))
}

