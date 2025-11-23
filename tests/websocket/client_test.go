package websocket_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gorillaWS "github.com/gorilla/websocket"
	"github.com/mautops/approval-gin/internal/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClient_ReadPump 测试客户端读取消息
func TestClient_ReadPump(t *testing.T) {
	hub := websocket.NewHub()
	go hub.Run()

	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := gorillaWS.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)

		client := websocket.NewClient("client-001", "user-001", hub, conn)
		hub.Register <- client

		// 启动 readPump 和 writePump
		go client.ReadPump()
		go client.WritePump()
	}))
	defer server.Close()

	// 连接到 WebSocket
	wsURL := "ws" + server.URL[4:]
	conn, _, err := gorillaWS.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// 等待连接建立
	time.Sleep(100 * time.Millisecond)

	// 验证客户端已注册
	assert.True(t, hub.HasClient("client-001"))
}

// TestClient_WritePump 测试客户端发送消息
func TestClient_WritePump(t *testing.T) {
	hub := websocket.NewHub()
	go hub.Run()

	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := gorillaWS.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)

		client := websocket.NewClient("client-001", "user-001", hub, conn)
		hub.Register <- client

		// 启动 readPump 和 writePump
		go client.ReadPump()
		go client.WritePump()

		// 发送消息到客户端
		message := []byte("test message")
		client.Send <- message
	}))
	defer server.Close()

	// 连接到 WebSocket
	wsURL := "ws" + server.URL[4:]
	conn, _, err := gorillaWS.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// 等待消息发送
	time.Sleep(200 * time.Millisecond)

	// 读取消息
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, []byte("test message"), message)
}

