package api

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeadersMiddleware 安全头中间件
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// X-Content-Type-Options: 防止 MIME 类型嗅探
		c.Header("X-Content-Type-Options", "nosniff")

		// X-Frame-Options: 防止点击劫持
		c.Header("X-Frame-Options", "DENY")

		// X-XSS-Protection: XSS 保护（虽然现代浏览器已内置，但保留以兼容旧浏览器）
		c.Header("X-XSS-Protection", "1; mode=block")

		// Strict-Transport-Security: 强制 HTTPS（HSTS）
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Referrer-Policy: 控制 Referer 头的发送
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content-Security-Policy: 内容安全策略（可选，可根据需要配置）
		// c.Header("Content-Security-Policy", "default-src 'self'")

		c.Next()
	}
}

