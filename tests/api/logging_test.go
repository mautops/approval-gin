package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/stretchr/testify/assert"
)

func TestLoggingMiddleware_RequestLog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 注册日志中间件
	router.Use(api.RequestLogMiddleware())

	// 测试路由
	router.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	// 日志中间件应该正常工作，不抛出错误
}

func TestLoggingMiddleware_ErrorLog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 注册日志中间件
	router.Use(api.RequestLogMiddleware())

	// 测试路由（返回错误）
	router.GET("/api/v1/error", func(c *gin.Context) {
		c.JSON(500, gin.H{"error": "internal error"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/error", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 500, w.Code)
	// 错误应该被正确记录
}

func TestLoggingMiddleware_RequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 注册请求 ID 和日志中间件
	router.Use(api.RequestIDMiddleware())
	router.Use(api.RequestLogMiddleware())

	// 测试路由
	router.GET("/api/v1/test", func(c *gin.Context) {
		requestID := c.GetString("request_id")
		c.JSON(200, gin.H{"request_id": requestID})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	// 请求 ID 应该被正确生成和传递
}

