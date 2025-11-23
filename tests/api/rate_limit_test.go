package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/stretchr/testify/assert"
)

// TestRateLimitMiddleware_AllowRequest 测试允许请求
func TestRateLimitMiddleware_AllowRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.New()
	router.Use(api.RateLimitMiddleware(100, 10)) // 100 req/s, burst 10
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestRateLimitMiddleware_TooManyRequests 测试限流
func TestRateLimitMiddleware_TooManyRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.New()
	router.Use(api.RateLimitMiddleware(1, 1)) // 1 req/s, burst 1
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})
	
	// 第一个请求应该成功
	req1 := httptest.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)
	
	// 立即发送第二个请求应该被限流
	req2 := httptest.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusTooManyRequests, w2.Code)
	
	// 等待一秒后应该可以再次请求
	time.Sleep(1100 * time.Millisecond)
	req3 := httptest.NewRequest("GET", "/test", nil)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)
}


