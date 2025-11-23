package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/stretchr/testify/assert"
)

// TestCSRFProtection_MissingToken 测试缺少 CSRF Token 的请求
func TestCSRFProtection_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	// 使用 CSRF 保护中间件
	router.Use(api.CSRFMiddleware(nil))
	router.POST("/api/v1/templates", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	body, _ := json.Marshal(map[string]string{"name": "test"})
	req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	// 不设置 X-CSRF-Token 头
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 应该被 CSRF 保护拒绝
	// assert.Equal(t, http.StatusForbidden, w.Code, "Should reject request without CSRF token")
	// 当前没有实现，所以先让测试失败
	assert.Equal(t, http.StatusForbidden, w.Code, "Should reject request without CSRF token")
}

// TestCSRFProtection_InvalidToken 测试无效的 CSRF Token
func TestCSRFProtection_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(api.CSRFMiddleware(nil))
	router.POST("/api/v1/templates", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	body, _ := json.Marshal(map[string]string{"name": "test"})
	req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", "invalid-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 应该被 CSRF 保护拒绝
	// assert.Equal(t, http.StatusForbidden, w.Code, "Should reject request with invalid CSRF token")
	// 当前没有实现，所以先让测试失败
	assert.Equal(t, http.StatusForbidden, w.Code, "Should reject request with invalid CSRF token")
}

// TestCSRFProtection_ValidToken 测试有效的 CSRF Token
func TestCSRFProtection_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(api.CSRFMiddleware(nil))
	router.GET("/api/v1/csrf-token", func(c *gin.Context) {
		// 返回 CSRF token
		token, err := api.GetCSRFToken(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": token})
	})
	router.POST("/api/v1/templates", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 1. 先获取 CSRF token
	tokenReq := httptest.NewRequest("GET", "/api/v1/csrf-token", nil)
	tokenW := httptest.NewRecorder()
	router.ServeHTTP(tokenW, tokenReq)
	assert.Equal(t, http.StatusOK, tokenW.Code, "Should return CSRF token")

	// 解析响应获取 token
	var tokenResp map[string]string
	json.Unmarshal(tokenW.Body.Bytes(), &tokenResp)
	token := tokenResp["token"]
	assert.NotEmpty(t, token, "Token should not be empty")

	// 2. 使用有效的 token 发送请求
	body, _ := json.Marshal(map[string]string{"name": "test"})
	req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", token)
	// 同时设置 Cookie（从第一次请求的响应中获取）
	for _, cookie := range tokenW.Result().Cookies() {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 应该成功
	assert.Equal(t, http.StatusOK, w.Code, "Should accept request with valid CSRF token")
}

// TestCSRFProtection_GETRequest 测试 GET 请求不需要 CSRF Token
func TestCSRFProtection_GETRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(api.CSRFMiddleware(nil))
	router.GET("/api/v1/templates", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"templates": []string{}})
	})

	req := httptest.NewRequest("GET", "/api/v1/templates", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// GET 请求应该不需要 CSRF token
	assert.Equal(t, http.StatusOK, w.Code, "GET request should not require CSRF token")
}

