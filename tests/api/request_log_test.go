package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/stretchr/testify/assert"
)

// TestRequestLogMiddleware_LogRequest 测试请求日志记录
func TestRequestLogMiddleware_LogRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.New()
	router.Use(api.RequestLogMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	// 验证请求被处理（日志记录在中间件中完成）
}

// TestRequestLogMiddleware_LogError 测试错误请求日志
func TestRequestLogMiddleware_LogError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.New()
	router.Use(api.RequestLogMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(404, gin.H{"error": "not found"})
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusNotFound, w.Code)
	// 验证错误请求被记录
}


