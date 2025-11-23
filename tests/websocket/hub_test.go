package websocket_test

import (
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/websocket"
	"github.com/stretchr/testify/assert"
)

// TestHub_Register 测试 Hub 注册客户端
func TestHub_Register(t *testing.T) {
	hub := websocket.NewHub()

	// 在后台运行 Hub
	go hub.Run()

	// 创建测试客户端
	client := &websocket.Client{
		ID:     "client-001",
		UserID: "user-001",
		Hub:    hub,
		Send:   make(chan []byte, 256),
	}

	// 注册客户端
	hub.Register <- client

	// 等待注册完成
	time.Sleep(100 * time.Millisecond)

	// 验证客户端已注册
	assert.True(t, hub.HasClient(client.ID))
}

// TestHub_Unregister 测试 Hub 注销客户端
func TestHub_Unregister(t *testing.T) {
	hub := websocket.NewHub()

	// 在后台运行 Hub
	go hub.Run()

	// 创建测试客户端
	client := &websocket.Client{
		ID:     "client-001",
		UserID: "user-001",
		Hub:    hub,
		Send:   make(chan []byte, 256),
	}

	// 注册客户端
	hub.Register <- client
	time.Sleep(100 * time.Millisecond)

	// 注销客户端
	hub.Unregister <- client
	time.Sleep(100 * time.Millisecond)

	// 验证客户端已注销
	assert.False(t, hub.HasClient(client.ID))
}

// TestHub_Broadcast 测试 Hub 广播消息
func TestHub_Broadcast(t *testing.T) {
	hub := websocket.NewHub()

	// 在后台运行 Hub
	go hub.Run()

	// 创建测试客户端
	client1 := &websocket.Client{
		ID:     "client-001",
		UserID: "user-001",
		Hub:    hub,
		Send:   make(chan []byte, 256),
	}

	client2 := &websocket.Client{
		ID:     "client-002",
		UserID: "user-002",
		Hub:    hub,
		Send:   make(chan []byte, 256),
	}

	// 注册客户端
	hub.Register <- client1
	hub.Register <- client2
	time.Sleep(100 * time.Millisecond)

	// 广播消息
	message := []byte("test message")
	hub.Broadcast <- message

	// 等待消息发送
	time.Sleep(100 * time.Millisecond)

	// 验证两个客户端都收到了消息
	select {
	case msg := <-client1.Send:
		assert.Equal(t, message, msg)
	case <-time.After(1 * time.Second):
		t.Error("client1 did not receive message")
	}

	select {
	case msg := <-client2.Send:
		assert.Equal(t, message, msg)
	case <-time.After(1 * time.Second):
		t.Error("client2 did not receive message")
	}
}

// TestHub_BroadcastToUser 测试 Hub 向特定用户广播消息
func TestHub_BroadcastToUser(t *testing.T) {
	hub := websocket.NewHub()

	// 在后台运行 Hub
	go hub.Run()

	// 创建测试客户端
	client1 := &websocket.Client{
		ID:     "client-001",
		UserID: "user-001",
		Hub:    hub,
		Send:   make(chan []byte, 256),
	}

	client2 := &websocket.Client{
		ID:     "client-002",
		UserID: "user-002",
		Hub:    hub,
		Send:   make(chan []byte, 256),
	}

	// 注册客户端
	hub.Register <- client1
	hub.Register <- client2
	time.Sleep(100 * time.Millisecond)

	// 向 user-001 广播消息
	message := []byte("test message")
	hub.BroadcastToUser("user-001", message)

	// 等待消息发送
	time.Sleep(100 * time.Millisecond)

	// 验证只有 client1 收到了消息
	select {
	case msg := <-client1.Send:
		assert.Equal(t, message, msg)
	case <-time.After(1 * time.Second):
		t.Error("client1 did not receive message")
	}

	// 验证 client2 没有收到消息
	select {
	case <-client2.Send:
		t.Error("client2 should not receive message")
	case <-time.After(100 * time.Millisecond):
		// 正确，client2 没有收到消息
	}
}

