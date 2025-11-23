package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// HTTPSRedirectMiddleware HTTPS 重定向中间件（生产环境强制 HTTPS）
func HTTPSRedirectMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否已经是 HTTPS
		if IsHTTPS(c) {
			c.Next()
			return
		}

		// 如果不是 HTTPS，重定向到 HTTPS
		host := c.Request.Host
		path := c.Request.RequestURI

		// 如果 host 为空，使用默认值
		if host == "" {
			host = "localhost"
		}

		// 构建完整的 HTTPS URL
		httpsURL := "https://" + host + path

		// 永久重定向（301）
		c.Redirect(http.StatusMovedPermanently, httpsURL)
		c.Abort()
	}
}

// HTTPSRedirectMiddlewareWithConfig 带配置的 HTTPS 重定向中间件
func HTTPSRedirectMiddlewareWithConfig(enabled bool) gin.HandlerFunc {
	if !enabled {
		// 如果未启用，返回空中间件
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return HTTPSRedirectMiddleware()
}

// IsHTTPS 检查请求是否通过 HTTPS
func IsHTTPS(c *gin.Context) bool {
	// 优先检查 X-Forwarded-Proto 头
	proto := strings.ToLower(c.GetHeader("X-Forwarded-Proto"))
	if proto == "https" {
		return true
	}

	// 检查 X-Forwarded-SSL 头（某些代理使用）
	if c.GetHeader("X-Forwarded-SSL") == "on" {
		return true
	}

	// 检查请求的 Scheme
	if c.Request.URL.Scheme == "https" {
		return true
	}

	// 检查 TLS 连接（直接连接时）
	if c.Request.TLS != nil {
		return true
	}

	return false
}

