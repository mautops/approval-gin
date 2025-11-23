package websocket

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	gorillaWS "github.com/gorilla/websocket"
	"github.com/mautops/approval-gin/internal/auth"
)

var upgrader = gorillaWS.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// 在生产环境中应该检查 Origin
		return true
	},
}

// WebSocketHandler WebSocket 处理器
// 支持 token 认证和用户关联
func WebSocketHandler(hub *Hub, validator *auth.KeycloakTokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从 query 参数获取 token
		token := c.Query("token")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}

		// 2. 验证 token
		claims, err := validator.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		// 3. 升级连接
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upgrade connection"})
			return
		}

		// 4. 创建客户端
		client := NewClient(
			uuid.New().String(),
			claims.Sub,
			hub,
			conn,
		)

		// 5. 注册客户端
		hub.Register <- client

		// 6. 启动 readPump 和 writePump
		go client.ReadPump()
		go client.WritePump()
	}
}


