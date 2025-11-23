package websocket_test

import (
	"net/http/httptest"
	"testing"

	gorillaWS "github.com/gorilla/websocket"
	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/auth"
	"github.com/mautops/approval-gin/internal/websocket"
	"github.com/stretchr/testify/assert"
)

// TestWebSocketHandler_WithToken 测试带 token 的 WebSocket 连接
func TestWebSocketHandler_WithToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建测试 Hub
	hub := websocket.NewHub()
	go hub.Run()

	// 创建测试 Keycloak 验证器
	validator := auth.NewKeycloakTokenValidator("http://localhost:8080/auth/realms/test")

	// 创建测试服务器
	router := gin.New()
	router.GET("/ws/tasks/:id", websocket.WebSocketHandler(hub, validator))

	server := httptest.NewServer(router)
	defer server.Close()

	// 连接到 WebSocket（不带 token，应该失败）
	wsURL := "ws" + server.URL[4:] + "/ws/tasks/task-001"
	_, resp, err := gorillaWS.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		resp.Body.Close()
	}
	// 没有 token 时应该失败
	assert.Error(t, err)
}

// TestWebSocketHandler_InvalidToken 测试无效 token 的 WebSocket 连接
func TestWebSocketHandler_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建测试 Hub
	hub := websocket.NewHub()
	go hub.Run()

	// 创建测试 Keycloak 验证器
	validator := auth.NewKeycloakTokenValidator("http://localhost:8080/auth/realms/test")

	// 创建测试服务器
	router := gin.New()
	router.GET("/ws/tasks/:id", websocket.WebSocketHandler(hub, validator))

	server := httptest.NewServer(router)
	defer server.Close()

	// 连接到 WebSocket（带无效 token，应该失败）
	wsURL := "ws" + server.URL[4:] + "/ws/tasks/task-001?token=invalid-token"
	_, resp, err := gorillaWS.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		resp.Body.Close()
	}
	// 无效 token 时应该失败
	assert.Error(t, err)
}

