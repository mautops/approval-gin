package api_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/stretchr/testify/assert"
)

// TestErrorHandlerMiddleware_HandleError 测试错误处理中间件
func TestErrorHandlerMiddleware_HandleError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.New()
	router.Use(api.ErrorHandlerMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.Error(errors.New("test error"))
		c.Next()
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// 验证错误被处理
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TestErrorHandlerMiddleware_NoError 测试无错误的情况
func TestErrorHandlerMiddleware_NoError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.New()
	router.Use(api.ErrorHandlerMiddleware())
	router.GET("/test", func(c *gin.Context) {
		api.Success(c, gin.H{"message": "success"})
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
}


