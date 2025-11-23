package api

import (
	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/metrics"
)

// MetricsHandler Prometheus 指标处理器
func MetricsHandler(c *gin.Context) {
	metrics.Handler().ServeHTTP(c.Writer, c.Request)
}
