package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/stretchr/testify/assert"
)

// TestCORSMiddleware_AllowedOrigin 测试允许的源
func TestCORSMiddleware_AllowedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	allowedOrigins := []string{"http://localhost:3000", "https://example.com"}
	router := gin.New()
	router.Use(api.CORSMiddleware(allowedOrigins))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
}

// TestCORSMiddleware_OptionsRequest 测试 OPTIONS 请求
func TestCORSMiddleware_OptionsRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	allowedOrigins := []string{"http://localhost:3000"}
	router := gin.New()
	router.Use(api.CORSMiddleware(allowedOrigins))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})
	
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
}

// TestCORSMiddleware_DisallowedOrigin 测试不允许的源
func TestCORSMiddleware_DisallowedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	allowedOrigins := []string{"http://localhost:3000"}
	router := gin.New()
	router.Use(api.CORSMiddleware(allowedOrigins))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://malicious.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// 不允许的源不应该设置 Access-Control-Allow-Origin
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}


