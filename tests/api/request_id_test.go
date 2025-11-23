package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/stretchr/testify/assert"
)

// TestRequestIDMiddleware_GenerateID 测试生成请求 ID
func TestRequestIDMiddleware_GenerateID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.New()
	router.Use(api.RequestIDMiddleware())
	router.GET("/test", func(c *gin.Context) {
		requestID := c.GetString("request_id")
		c.JSON(200, gin.H{"request_id": requestID})
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	// 验证响应头包含请求 ID
	requestID := w.Header().Get("X-Request-ID")
	assert.NotEmpty(t, requestID)
}

// TestRequestIDMiddleware_UseExistingID 测试使用已有的请求 ID
func TestRequestIDMiddleware_UseExistingID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.New()
	router.Use(api.RequestIDMiddleware())
	router.GET("/test", func(c *gin.Context) {
		requestID := c.GetString("request_id")
		c.JSON(200, gin.H{"request_id": requestID})
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "custom-request-id")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	// 验证使用已有的请求 ID
	requestID := w.Header().Get("X-Request-ID")
	assert.Equal(t, "custom-request-id", requestID)
}


