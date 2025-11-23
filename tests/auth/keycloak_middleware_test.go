package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/auth"
	"github.com/stretchr/testify/assert"
)

// TestKeycloakAuthMiddleware_MissingToken 测试缺少 Token 的情况
func TestKeycloakAuthMiddleware_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	issuer := "https://keycloak.example.com/realms/test"
	validator := auth.NewKeycloakTokenValidator(issuer)
	middleware := auth.KeycloakAuthMiddleware(validator)
	
	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestKeycloakAuthMiddleware_InvalidToken 测试无效 Token 的情况
func TestKeycloakAuthMiddleware_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	issuer := "https://keycloak.example.com/realms/test"
	validator := auth.NewKeycloakTokenValidator(issuer)
	middleware := auth.KeycloakAuthMiddleware(validator)
	
	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestKeycloakAuthMiddleware_ValidToken 测试有效 Token 的情况
func TestKeycloakAuthMiddleware_ValidToken(t *testing.T) {
	// 注意: 完整的测试需要真实的 Keycloak 服务器和有效的 token
	// 这里只验证中间件存在且可调用
	gin.SetMode(gin.TestMode)
	
	issuer := "https://keycloak.example.com/realms/test"
	validator := auth.NewKeycloakTokenValidator(issuer)
	middleware := auth.KeycloakAuthMiddleware(validator)
	
	assert.NotNil(t, middleware)
}


