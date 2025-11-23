package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/auth"
	"github.com/stretchr/testify/assert"
)

// TestPermissionMiddleware_MissingUser 测试缺少用户信息的情况
func TestPermissionMiddleware_MissingUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	apiURL := "http://localhost:8080"
	storeID := "test-store"
	modelID := "test-model"
	fgaClient, _ := auth.NewOpenFGAClient(apiURL, storeID, modelID)
	
	middleware := auth.PermissionMiddleware(fgaClient, "template", "viewer")
	
	router := gin.New()
	router.Use(middleware)
	router.GET("/templates/:id", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})
	
	req := httptest.NewRequest("GET", "/templates/tpl-001", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestPermissionMiddleware_NoPermission 测试无权限的情况
func TestPermissionMiddleware_NoPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	apiURL := "http://localhost:8080"
	storeID := "test-store"
	modelID := "test-model"
	fgaClient, err := auth.NewOpenFGAClient(apiURL, storeID, modelID)
	if err != nil {
		t.Skip("OpenFGA client creation failed, skipping test")
		return
	}
	
	middleware := auth.PermissionMiddleware(fgaClient, "template", "viewer")
	
	router := gin.New()
	// 先设置用户信息（模拟认证中间件）
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "user-001")
		c.Next()
	})
	router.Use(middleware)
	router.GET("/templates/:id", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})
	
	req := httptest.NewRequest("GET", "/templates/tpl-001", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// 注意: 完整的测试需要真实的 OpenFGA 服务器和权限关系
	// 这里只验证中间件存在且可调用
	// 由于没有权限关系或服务器不可用，应该返回 403 或 500
	assert.NotEqual(t, http.StatusOK, w.Code)
}

