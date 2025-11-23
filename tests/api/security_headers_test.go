package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/stretchr/testify/assert"
)

// TestSecurityHeaders_AllHeaders 测试所有安全头是否设置
func TestSecurityHeaders_AllHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(api.SecurityHeadersMiddleware())
	router.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证所有安全头都已设置
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"), "Should set X-Content-Type-Options")
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"), "Should set X-Frame-Options")
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"), "Should set X-XSS-Protection")
	assert.Contains(t, w.Header().Get("Strict-Transport-Security"), "max-age=", "Should set Strict-Transport-Security")
}

// TestSecurityHeaders_XContentTypeOptions 测试 X-Content-Type-Options 头
func TestSecurityHeaders_XContentTypeOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(api.SecurityHeadersMiddleware())
	router.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"), "Should set X-Content-Type-Options to nosniff")
}

// TestSecurityHeaders_XFrameOptions 测试 X-Frame-Options 头
func TestSecurityHeaders_XFrameOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(api.SecurityHeadersMiddleware())
	router.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"), "Should set X-Frame-Options to DENY")
}

// TestSecurityHeaders_XXSSProtection 测试 X-XSS-Protection 头
func TestSecurityHeaders_XXSSProtection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(api.SecurityHeadersMiddleware())
	router.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"), "Should set X-XSS-Protection to 1; mode=block")
}

// TestSecurityHeaders_StrictTransportSecurity 测试 Strict-Transport-Security 头
func TestSecurityHeaders_StrictTransportSecurity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(api.SecurityHeadersMiddleware())
	router.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	hsts := w.Header().Get("Strict-Transport-Security")
	assert.Contains(t, hsts, "max-age=", "Should set Strict-Transport-Security with max-age")
	assert.Contains(t, hsts, "includeSubDomains", "Should include includeSubDomains in HSTS")
}


