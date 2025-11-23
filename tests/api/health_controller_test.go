package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestHealthController_Check 测试健康检查
func TestHealthController_Check(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// 创建测试数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	
	router := gin.New()
	controller := api.NewHealthController(db, nil) // 不配置 OpenFGA 客户端
	
	// 注册路由
	router.GET("/health", controller.Check)
	
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	// 验证响应包含健康状态
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	status, ok := response["status"].(string)
	assert.True(t, ok)
	assert.Equal(t, "healthy", status)
	
	// 验证包含检查项
	checks, ok := response["checks"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, checks, "database")
	assert.Contains(t, checks, "openfga")
}

// TestHealthController_Check_NoDB 测试无数据库的健康检查
func TestHealthController_Check_NoDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.New()
	controller := api.NewHealthController(nil, nil) // 不配置数据库和 OpenFGA
	
	// 注册路由
	router.GET("/health", controller.Check)
	
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	// 验证响应
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	status, ok := response["status"].(string)
	assert.True(t, ok)
	assert.Equal(t, "healthy", status)
}

