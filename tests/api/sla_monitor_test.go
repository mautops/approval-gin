package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/stretchr/testify/assert"
)

func TestSLAMonitorMiddleware_WithinSLA(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 配置 SLA（任务创建最大响应时间 1 秒）
	slaConfig := &api.SLAConfig{
		TaskCreationMaxTime: 1 * time.Second,
		TaskApprovalMaxTime: 2 * time.Second,
		TemplateQueryMaxTime: 500 * time.Millisecond,
		TaskQueryMaxTime:     500 * time.Millisecond,
	}

	// 注册 SLA 监控中间件
	router.Use(api.SLAMonitorMiddleware(slaConfig))

	// 测试路由（快速响应，在 SLA 内）
	router.GET("/api/v1/tasks", func(c *gin.Context) {
		time.Sleep(100 * time.Millisecond) // 模拟快速响应
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tasks", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	// 应该没有 SLA 违反警告头
	assert.Empty(t, w.Header().Get("X-SLA-Violation"))
}

func TestSLAMonitorMiddleware_ExceedsSLA(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 配置 SLA（任务创建最大响应时间 100 毫秒）
	slaConfig := &api.SLAConfig{
		TaskCreationMaxTime: 100 * time.Millisecond,
		TaskApprovalMaxTime: 200 * time.Millisecond,
		TemplateQueryMaxTime: 50 * time.Millisecond,
		TaskQueryMaxTime:     50 * time.Millisecond,
	}

	// 注册 SLA 监控中间件
	router.Use(api.SLAMonitorMiddleware(slaConfig))

	// 测试路由（慢速响应，超过 SLA）
	router.GET("/api/v1/tasks", func(c *gin.Context) {
		time.Sleep(200 * time.Millisecond) // 模拟慢速响应，超过 SLA
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tasks", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	// 应该有 SLA 违反警告头
	assert.Equal(t, "true", w.Header().Get("X-SLA-Violation"))
}

func TestSLAMonitorMiddleware_DifferentOperations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 配置 SLA
	slaConfig := &api.SLAConfig{
		TaskCreationMaxTime: 1 * time.Second,
		TaskApprovalMaxTime: 2 * time.Second,
		TemplateQueryMaxTime: 500 * time.Millisecond,
		TaskQueryMaxTime:     500 * time.Millisecond,
	}

	// 注册 SLA 监控中间件
	router.Use(api.SLAMonitorMiddleware(slaConfig))

	// 任务创建路由
	router.POST("/api/v1/tasks", func(c *gin.Context) {
		time.Sleep(100 * time.Millisecond)
		c.JSON(200, gin.H{"status": "created"})
	})

	// 任务审批路由
	router.POST("/api/v1/tasks/:id/approve", func(c *gin.Context) {
		time.Sleep(100 * time.Millisecond)
		c.JSON(200, gin.H{"status": "approved"})
	})

	// 测试任务创建（在 SLA 内）
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/tasks", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Empty(t, w.Header().Get("X-SLA-Violation"))

	// 测试任务审批（在 SLA 内）
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/tasks/123/approve", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Empty(t, w.Header().Get("X-SLA-Violation"))
}

