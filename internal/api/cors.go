package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware CORS 中间件
func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// 检查是否允许所有源
		allowAll := false
		for _, allowedOrigin := range allowedOrigins {
			if allowedOrigin == "*" {
				allowAll = true
				break
			}
		}

		// 检查 origin 是否在允许列表中
		allowed := false
		if allowAll {
			allowed = true
		} else {
			for _, allowedOrigin := range allowedOrigins {
				if allowedOrigin == origin {
					allowed = true
					break
				}
			}
		}

		if allowed {
			if allowAll {
				// 允许所有源时,不能设置 credentials
				c.Header("Access-Control-Allow-Origin", "*")
			} else {
				// 指定具体源时,可以设置 credentials
				if origin != "" {
					c.Header("Access-Control-Allow-Origin", origin)
					c.Header("Access-Control-Allow-Credentials", "true")
				} else {
					// 如果没有 Origin 头(同源请求),不设置 CORS 头
				}
			}
		}

		// 设置其他 CORS 头
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400") // 24 小时

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}


