package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/mautops/approval-gin/internal/auth"
	"github.com/mautops/approval-gin/internal/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRoutes_SSE 测试 SSE 路由配置
func TestRoutes_SSE(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hub := websocket.NewHub()
	validator := auth.NewKeycloakTokenValidator("http://localhost:8080/auth/realms/test")

	router := api.SetupRoutes(hub, validator, nil, nil)

	// 测试 SSE 路由存在
	req, err := http.NewRequest("GET", "/sse/tasks/task-001", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 没有 token 时应该返回 401
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestRoutes_SSE_WithToken 测试带 token 的 SSE 路由
func TestRoutes_SSE_WithToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hub := websocket.NewHub()
	validator := auth.NewKeycloakTokenValidator("http://localhost:8080/auth/realms/test")

	router := api.SetupRoutes(hub, validator, nil, nil)

	// 测试 SSE 路由（带无效 token）
	req, err := http.NewRequest("GET", "/sse/tasks/task-001?token=invalid-token", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 无效 token 时应该返回 401
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

