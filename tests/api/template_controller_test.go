package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/mautops/approval-gin/internal/integration"
	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-gin/internal/service"
	"github.com/mautops/approval-kit/pkg/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDBForController 创建控制器测试数据库
func setupTestDBForController(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 迁移数据库
	err = db.AutoMigrate(
		&model.TemplateModel{},
		&model.TaskModel{},
	)
	require.NoError(t, err)

	return db
}

// TestTemplateController_Create 测试创建模板
func TestTemplateController_Create(t *testing.T) {
	db := setupTestDBForController(t)
	templateMgr := integration.NewTemplateManager(db)
	templateService := service.NewTemplateService(templateMgr, db, service.NewAuditLogService(nil))
	
	router := gin.New()
	controller := api.NewTemplateController(templateService)
	
	// 注册路由
	router.POST("/api/v1/templates", controller.Create)
	
	// 创建请求
	reqBody := service.CreateTemplateRequest{
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	}
	body, _ := json.Marshal(reqBody)
	
	req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response api.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 0, response.Code)
}

// TestTemplateController_Get 测试获取模板
func TestTemplateController_Get(t *testing.T) {
	db := setupTestDBForController(t)
	templateMgr := integration.NewTemplateManager(db)
	templateService := service.NewTemplateService(templateMgr, db, service.NewAuditLogService(nil))
	
	// 先创建模板
	template, err := templateService.Create(context.Background(), &service.CreateTemplateRequest{
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	})
	require.NoError(t, err)
	
	router := gin.New()
	controller := api.NewTemplateController(templateService)
	
	// 注册路由
	router.GET("/api/v1/templates/:id", controller.Get)
	
	req := httptest.NewRequest("GET", "/api/v1/templates/"+template.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response api.Response
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 0, response.Code)
}

// TestTemplateController_List 测试模板列表
func TestTemplateController_List(t *testing.T) {
	db := setupTestDBForController(t)
	templateMgr := integration.NewTemplateManager(db)
	templateService := service.NewTemplateService(templateMgr, db, service.NewAuditLogService(nil))
	
	// 创建几个模板
	for i := 0; i < 3; i++ {
		_, err := templateService.Create(context.Background(), &service.CreateTemplateRequest{
			Name:        "测试模板",
			Description: "这是一个测试模板",
			Nodes:       make(map[string]*template.Node),
			Edges:       []*template.Edge{},
			Config:      nil,
		})
		require.NoError(t, err)
	}
	
	router := gin.New()
	controller := api.NewTemplateController(templateService)
	
	// 注册路由
	router.GET("/api/v1/templates", controller.List)
	
	req := httptest.NewRequest("GET", "/api/v1/templates", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response api.PaginatedResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 0, response.Code)
}

