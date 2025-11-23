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
)

// TestRoutes_WebSocket 测试 WebSocket 路由
func TestRoutes_WebSocket(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	hub := websocket.NewHub()
	validator := auth.NewKeycloakTokenValidator("http://localhost:8080/auth/realms/test")

	// 配置路由
	router = api.SetupRoutes(hub, validator, nil, nil)

	// 测试 WebSocket 路由
	req := httptest.NewRequest("GET", "/ws/tasks/task-001?token=test-token", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// WebSocket 升级请求应该返回 101 状态码或 401（如果 token 无效）
	// 由于没有真实的 WebSocket 连接，这里只验证路由存在
	assert.True(t, w.Code == http.StatusSwitchingProtocols || w.Code == http.StatusUnauthorized || w.Code == http.StatusBadRequest)
}

