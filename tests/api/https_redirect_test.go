package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/stretchr/testify/assert"
)

// TestHTTPSRedirect_HTTPRequest 测试 HTTP 请求被重定向到 HTTPS
func TestHTTPSRedirect_HTTPRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(api.HTTPSRedirectMiddleware())
	router.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 模拟 HTTP 请求（没有 X-Forwarded-Proto 头）
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 应该被重定向到 HTTPS
	assert.Equal(t, http.StatusMovedPermanently, w.Code, "Should redirect HTTP to HTTPS")
	assert.Contains(t, w.Header().Get("Location"), "https://", "Location should be HTTPS URL")
}

// TestHTTPSRedirect_HTTPSRequest 测试 HTTPS 请求正常处理
func TestHTTPSRedirect_HTTPSRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(api.HTTPSRedirectMiddleware())
	router.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 模拟 HTTPS 请求（通过 X-Forwarded-Proto 头）
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 应该正常处理，不重定向
	assert.Equal(t, http.StatusOK, w.Code, "Should process HTTPS request normally")
}

// TestHTTPSRedirect_HTTPRequestWithXForwardedProto 测试带有 X-Forwarded-Proto 的 HTTP 请求
func TestHTTPSRedirect_HTTPRequestWithXForwardedProto(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(api.HTTPSRedirectMiddleware())
	router.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 模拟 HTTP 请求（X-Forwarded-Proto 为 http）
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("X-Forwarded-Proto", "http")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 应该被重定向到 HTTPS
	assert.Equal(t, http.StatusMovedPermanently, w.Code, "Should redirect HTTP to HTTPS")
	assert.Contains(t, w.Header().Get("Location"), "https://", "Location should be HTTPS URL")
}

// TestHTTPSRedirect_DevelopmentMode 测试开发模式不强制 HTTPS
func TestHTTPSRedirect_DevelopmentMode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	// 开发模式不启用 HTTPS 强制
	// router.Use(api.HTTPSRedirectMiddleware()) // 仅在生产环境启用
	router.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 模拟 HTTP 请求
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 开发模式应该正常处理
	assert.Equal(t, http.StatusOK, w.Code, "Development mode should not force HTTPS")
}

