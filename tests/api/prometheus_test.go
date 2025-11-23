package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/stretchr/testify/assert"
)

func TestPrometheusMetrics_Endpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 注册 Prometheus 指标端点
	router.GET("/metrics", api.MetricsHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/metrics", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	// 验证返回 Prometheus 格式的指标
	body := w.Body.String()
	assert.Contains(t, body, "# HELP")
	assert.Contains(t, body, "# TYPE")
}

func TestPrometheusMetrics_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/metrics", api.MetricsHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/metrics", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	body := w.Body.String()

	// 验证 Prometheus 格式
	lines := strings.Split(body, "\n")
	hasHelp := false
	hasType := false
	for _, line := range lines {
		if strings.HasPrefix(line, "# HELP") {
			hasHelp = true
		}
		if strings.HasPrefix(line, "# TYPE") {
			hasType = true
		}
	}
	assert.True(t, hasHelp, "should have HELP comments")
	assert.True(t, hasType, "should have TYPE comments")
}

func TestPrometheusMetrics_ContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/metrics", api.MetricsHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/metrics", nil)
	router.ServeHTTP(w, req)

	// Prometheus 指标应该返回 text/plain 或 text/plain; version=0.0.4
	contentType := w.Header().Get("Content-Type")
	assert.True(t, strings.Contains(contentType, "text/plain"), "should return text/plain content type")
}

