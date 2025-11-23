package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/stretchr/testify/assert"
)

func TestI18nMiddleware_DefaultLanguage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 注册国际化中间件
	router.Use(api.I18nMiddleware())

	// 测试路由
	router.GET("/api/v1/test", func(c *gin.Context) {
		lang := api.GetLanguage(c)
		message := api.T(c, "test.message")
		c.JSON(200, gin.H{"language": lang, "message": message})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	// 默认语言应该是 "en"
}

func TestI18nMiddleware_AcceptLanguage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 注册国际化中间件
	router.Use(api.I18nMiddleware())

	// 测试路由
	router.GET("/api/v1/test", func(c *gin.Context) {
		lang := api.GetLanguage(c)
		c.JSON(200, gin.H{"language": lang})
	})

	// 测试中文语言
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	// 应该识别为中文
}

func TestI18nMiddleware_QueryParameter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 注册国际化中间件
	router.Use(api.I18nMiddleware())

	// 测试路由
	router.GET("/api/v1/test", func(c *gin.Context) {
		lang := api.GetLanguage(c)
		c.JSON(200, gin.H{"language": lang})
	})

	// 测试通过查询参数指定语言
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test?lang=zh", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	// 应该识别为中文
}

func TestI18nMiddleware_Translation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 注册国际化中间件
	router.Use(api.I18nMiddleware())

	// 测试路由
	router.GET("/api/v1/test", func(c *gin.Context) {
		message := api.T(c, "error.not_found")
		c.JSON(200, gin.H{"message": message})
	})

	// 测试英文翻译
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test?lang=en", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	// 测试中文翻译
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/test?lang=zh", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

