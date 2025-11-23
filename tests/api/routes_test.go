package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/stretchr/testify/assert"
)

// TestRoutes_HealthCheck 测试健康检查路由
func TestRoutes_HealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := api.SetupRoutes(nil, nil, nil, nil)
	
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestRoutes_TemplatesRoutes 测试模板路由
func TestRoutes_TemplatesRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := api.SetupRoutes(nil, nil, nil, nil)
	
	// 测试模板列表路由（需要认证，这里只验证路由存在）
	req := httptest.NewRequest("GET", "/api/v1/templates", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// 由于没有认证，应该返回 401 或 404（取决于路由配置）
	// 这里只验证路由被注册
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

// TestRoutes_TasksRoutes 测试任务路由
func TestRoutes_TasksRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := api.SetupRoutes(nil, nil, nil, nil)
	
	// 测试任务列表路由（需要认证，这里只验证路由存在）
	req := httptest.NewRequest("GET", "/api/v1/tasks", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// 由于没有认证，应该返回 401 或 404（取决于路由配置）
	// 这里只验证路由被注册
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

