package api

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// I18nManager 国际化管理器
type I18nManager struct {
	messages map[string]map[string]string // lang -> key -> message
}

var defaultI18nManager *I18nManager

func init() {
	defaultI18nManager = NewI18nManager()
	// 加载默认语言资源
	defaultI18nManager.LoadMessages("en", map[string]string{
		"error.not_found":        "Resource not found",
		"error.unauthorized":     "Unauthorized",
		"error.forbidden":        "Forbidden",
		"error.bad_request":      "Bad request",
		"error.internal_error":   "Internal server error",
		"success.created":        "Created successfully",
		"success.updated":        "Updated successfully",
		"success.deleted":        "Deleted successfully",
		"test.message":           "Test message",
	})
	// 加载中文语言资源
	defaultI18nManager.LoadMessages("zh", map[string]string{
		"error.not_found":        "资源未找到",
		"error.unauthorized":     "未授权",
		"error.forbidden":        "禁止访问",
		"error.bad_request":      "请求错误",
		"error.internal_error":   "服务器内部错误",
		"success.created":        "创建成功",
		"success.updated":        "更新成功",
		"success.deleted":        "删除成功",
		"test.message":           "测试消息",
	})
}

// NewI18nManager 创建国际化管理器
func NewI18nManager() *I18nManager {
	return &I18nManager{
		messages: make(map[string]map[string]string),
	}
}

// LoadMessages 加载语言消息
func (m *I18nManager) LoadMessages(lang string, messages map[string]string) {
	m.messages[lang] = messages
}

// Translate 翻译消息
func (m *I18nManager) Translate(lang, key string) string {
	if messages, ok := m.messages[lang]; ok {
		if message, ok := messages[key]; ok {
			return message
		}
	}
	// 如果找不到翻译，尝试使用英文
	if lang != "en" {
		if messages, ok := m.messages["en"]; ok {
			if message, ok := messages[key]; ok {
				return message
			}
		}
	}
	// 如果还是找不到，返回 key
	return key
}

// I18nMiddleware 国际化中间件
func I18nMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := "en" // 默认语言

		// 方式 1: 从查询参数获取语言
		if queryLang := c.Query("lang"); queryLang != "" {
			lang = normalizeLanguage(queryLang)
		} else if headerLang := c.GetHeader("Accept-Language"); headerLang != "" {
			// 方式 2: 从 Accept-Language 头获取语言
			lang = parseAcceptLanguage(headerLang)
		}

		// 将语言信息存储到上下文
		c.Set("language", lang)

		c.Next()
	}
}

// GetLanguage 从上下文获取语言
func GetLanguage(c *gin.Context) string {
	if lang, exists := c.Get("language"); exists {
		if l, ok := lang.(string); ok {
			return l
		}
	}
	return "en" // 默认语言
}

// T 翻译消息（使用默认管理器）
func T(c *gin.Context, key string) string {
	lang := GetLanguage(c)
	return defaultI18nManager.Translate(lang, key)
}

// normalizeLanguage 规范化语言代码
func normalizeLanguage(lang string) string {
	lang = strings.ToLower(lang)
	// 支持的语言代码映射
	langMap := map[string]string{
		"zh-cn": "zh",
		"zh-tw": "zh",
		"zh-hk": "zh",
		"en-us": "en",
		"en-gb": "en",
	}
	if normalized, ok := langMap[lang]; ok {
		return normalized
	}
	// 如果语言代码以 zh 开头，返回 zh
	if strings.HasPrefix(lang, "zh") {
		return "zh"
	}
	// 如果语言代码以 en 开头，返回 en
	if strings.HasPrefix(lang, "en") {
		return "en"
	}
	return lang
}

// parseAcceptLanguage 解析 Accept-Language 头
func parseAcceptLanguage(header string) string {
	// 解析 Accept-Language: zh-CN,zh;q=0.9,en;q=0.8
	parts := strings.Split(header, ",")
	if len(parts) > 0 {
		// 取第一个语言代码
		lang := strings.TrimSpace(parts[0])
		// 移除质量值（如果有）
		if idx := strings.Index(lang, ";"); idx != -1 {
			lang = lang[:idx]
		}
		return normalizeLanguage(lang)
	}
	return "en"
}

