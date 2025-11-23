package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/auth"
)

// SSEHandler SSE 处理器
// 支持 token 认证和任务状态实时推送
func SSEHandler(validator *auth.KeycloakTokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从 query 参数获取 token
		token := c.Query("token")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			c.Abort()
			return
		}

		// 2. 验证 token
		claims, err := validator.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// 3. 获取任务 ID
		taskID := c.Param("id")
		if taskID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "task id required"})
			c.Abort()
			return
		}

		// 4. 设置 SSE 响应头
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no") // 禁用 Nginx 缓冲

		// 5. 获取 Flusher（用于刷新响应缓冲区）
		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
			c.Abort()
			return
		}

		// 6. 创建 SSE 客户端通道
		messageChan := make(chan []byte, 256)
		defer close(messageChan)

		// 7. 启动 goroutine 发送心跳和监听任务更新
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-c.Request.Context().Done():
					return
				case <-ticker.C:
					// 发送心跳保持连接
					heartbeat := map[string]interface{}{
						"type":    "heartbeat",
						"task_id": taskID,
						"time":    time.Now().Unix(),
					}
					data, _ := json.Marshal(heartbeat)
					select {
					case messageChan <- data:
					default:
						return
					}
				}
			}
		}()

		// 8. 发送初始连接消息
		initialMsg := map[string]interface{}{
			"type":    "connected",
			"task_id": taskID,
			"user_id": claims.Sub,
			"time":    time.Now().Unix(),
		}
		initialData, _ := json.Marshal(initialMsg)
		if err := sendSSEMessage(c.Writer, initialData); err != nil {
			return
		}
		flusher.Flush()

		// 9. 持续发送消息
		for {
			select {
			case <-c.Request.Context().Done():
				return
			case message, ok := <-messageChan:
				if !ok {
					return
				}
				if err := sendSSEMessage(c.Writer, message); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	}
}

// sendSSEMessage 发送 SSE 消息
func sendSSEMessage(w io.Writer, data []byte) error {
	// SSE 格式: data: <json>\n\n
	_, err := fmt.Fprintf(w, "data: %s\n\n", string(data))
	return err
}

