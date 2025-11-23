package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/metrics"
	"github.com/sirupsen/logrus"
)

// RequestLogMiddleware 请求日志中间件
func RequestLogMiddleware() gin.HandlerFunc {
	logger := GetLogger()
	
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		requestID := c.GetString("request_id")

		// 记录 Prometheus 指标
		metrics.RecordAPIRequest(method, path, status, latency.Seconds())

		// 使用结构化日志记录请求信息
		entry := logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"method":     method,
			"path":       path,
			"status":     status,
			"latency":    latency.String(),
			"ip":         c.ClientIP(),
		})

		// 根据状态码选择日志级别
		if status >= 500 {
			entry.Error("API request")
		} else if status >= 400 {
			entry.Warn("API request")
		} else {
			entry.Info("API request")
		}
	}
}

