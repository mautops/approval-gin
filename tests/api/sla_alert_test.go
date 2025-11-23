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

func TestSLAAlert_RecordViolation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 创建 SLA 告警器
	alertManager := api.NewSLAAlertManager()

	// 配置 SLA（任务创建最大响应时间 100 毫秒）
	slaConfig := &api.SLAConfig{
		TaskCreationMaxTime: 100 * time.Millisecond,
		TaskApprovalMaxTime: 200 * time.Millisecond,
		TemplateQueryMaxTime: 50 * time.Millisecond,
		TaskQueryMaxTime:     50 * time.Millisecond,
	}

	// 注册 SLA 监控中间件（带告警器）
	router.Use(api.SLAMonitorMiddlewareWithAlert(slaConfig, alertManager))

	// 测试路由（慢速响应，超过 SLA）
	router.GET("/api/v1/tasks", func(c *gin.Context) {
		time.Sleep(200 * time.Millisecond) // 模拟慢速响应，超过 SLA
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tasks", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	// 验证 SLA 违反被记录
	violations := alertManager.GetViolations("task_query")
	assert.Greater(t, len(violations), 0)
}

func TestSLAAlert_AlertThreshold(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 创建 SLA 告警器（设置告警阈值）
	alertManager := api.NewSLAAlertManager()
	alertManager.SetAlertThreshold("task_query", 3) // 3 次违反后告警

	// 配置 SLA
	slaConfig := &api.SLAConfig{
		TaskQueryMaxTime: 50 * time.Millisecond,
	}

	// 注册 SLA 监控中间件（带告警器）
	router.Use(api.SLAMonitorMiddlewareWithAlert(slaConfig, alertManager))

	// 测试路由（慢速响应）
	router.GET("/api/v1/tasks", func(c *gin.Context) {
		time.Sleep(100 * time.Millisecond) // 超过 SLA
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 触发多次违反
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/tasks", nil)
		router.ServeHTTP(w, req)
		time.Sleep(10 * time.Millisecond)
	}

	// 验证告警被触发
	violations := alertManager.GetViolations("task_query")
	assert.GreaterOrEqual(t, len(violations), 3)
}

func TestSLAAlert_AlertCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 创建 SLA 告警器
	alertManager := api.NewSLAAlertManager()
	alertCalled := false

	// 注册告警回调
	alertManager.OnAlert(func(operation string, violations []api.SLAViolation) {
		alertCalled = true
		assert.Equal(t, "task_query", operation)
		assert.Greater(t, len(violations), 0)
	})

	// 配置 SLA
	slaConfig := &api.SLAConfig{
		TaskQueryMaxTime: 50 * time.Millisecond,
	}

	// 注册 SLA 监控中间件（带告警器）
	router.Use(api.SLAMonitorMiddlewareWithAlert(slaConfig, alertManager))

	// 测试路由（慢速响应）
	router.GET("/api/v1/tasks", func(c *gin.Context) {
		time.Sleep(100 * time.Millisecond) // 超过 SLA
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 触发多次违反（超过阈值）
	alertManager.SetAlertThreshold("task_query", 2)
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/tasks", nil)
		router.ServeHTTP(w, req)
		time.Sleep(10 * time.Millisecond)
	}

	// 验证告警回调被调用
	assert.True(t, alertCalled, "alert callback should be called")
}

