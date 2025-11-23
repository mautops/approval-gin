package websocket

import (
	"sync"
)

// Hub 管理所有 WebSocket 连接
type Hub struct {
	// 已注册的客户端
	clients map[*Client]bool

	// 广播消息到所有客户端
	Broadcast chan []byte

	// 注册新客户端
	Register chan *Client

	// 注销客户端
	Unregister chan *Client

	// 互斥锁，保护 clients map
	mu sync.RWMutex
}

// NewHub 创建新的 Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

// Run 运行 Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()

		case message := <-h.Broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastToUser 向特定用户广播消息
func (h *Hub) BroadcastToUser(userID string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client.UserID == userID {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(h.clients, client)
			}
		}
	}
}

// HasClient 检查客户端是否存在
func (h *Hub) HasClient(clientID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client.ID == clientID {
			return true
		}
	}
	return false
}

// GetClientCount 获取客户端数量
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.clients)
}

