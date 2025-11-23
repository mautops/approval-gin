package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/stretchr/testify/assert"
)

// TestOpenTelemetry_Integration 测试 OpenTelemetry 集成
func TestOpenTelemetry_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// 初始化追踪（使用内存 exporter，不连接真实的 Jaeger）
	// 注意：在实际使用中，应该连接到真实的 Jaeger 服务器
	// 这里我们只测试中间件是否存在且可调用
	router := gin.New()
	// 添加追踪中间件
	router.Use(api.TracingMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	// 验证请求处理成功（追踪中间件应该正常工作）
}

// TestTracingMiddleware_Exists 测试追踪中间件是否存在
func TestTracingMiddleware_Exists(t *testing.T) {
	// 验证 TracingMiddleware 函数存在且可调用
	middleware := api.TracingMiddleware()
	assert.NotNil(t, middleware)
}

