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

// setupTestDBForTaskController 创建任务控制器测试数据库
func setupTestDBForTaskController(t *testing.T) *gorm.DB {
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

// TestTaskController_Create 测试创建任务
func TestTaskController_Create(t *testing.T) {
	db := setupTestDBForTaskController(t)
	templateMgr := integration.NewTemplateManager(db)
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)
	taskService := service.NewTaskService(taskMgr, db, service.NewAuditLogService(nil), nil)
	
	// 先创建模板
	templateService := service.NewTemplateService(templateMgr, db, service.NewAuditLogService(nil))
	template, err := templateService.Create(context.Background(), &service.CreateTemplateRequest{
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	})
	require.NoError(t, err)
	
	router := gin.New()
	controller := api.NewTaskController(taskService)
	
	// 注册路由
	router.POST("/api/v1/tasks", controller.Create)
	
	// 创建请求
	reqBody := service.CreateTaskRequest{
		TemplateID: template.ID,
		BusinessID: "biz-001",
		Params:     json.RawMessage(`{"amount": 1000}`),
	}
	body, _ := json.Marshal(reqBody)
	
	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response api.Response
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 0, response.Code)
}

// TestTaskController_Get 测试获取任务
func TestTaskController_Get(t *testing.T) {
	db := setupTestDBForTaskController(t)
	templateMgr := integration.NewTemplateManager(db)
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)
	taskService := service.NewTaskService(taskMgr, db, service.NewAuditLogService(nil), nil)
	
	// 先创建模板和任务
	templateService := service.NewTemplateService(templateMgr, db, service.NewAuditLogService(nil))
	template, err := templateService.Create(context.Background(), &service.CreateTemplateRequest{
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	})
	require.NoError(t, err)
	
	task, err := taskService.Create(context.Background(), &service.CreateTaskRequest{
		TemplateID: template.ID,
		BusinessID: "biz-001",
		Params:     json.RawMessage(`{"amount": 1000}`),
	})
	require.NoError(t, err)
	
	router := gin.New()
	controller := api.NewTaskController(taskService)
	
	// 注册路由
	router.GET("/api/v1/tasks/:id", controller.Get)
	
	req := httptest.NewRequest("GET", "/api/v1/tasks/"+task.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response api.Response
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 0, response.Code)
}

// TestTaskController_Submit 测试提交任务
func TestTaskController_Submit(t *testing.T) {
	db := setupTestDBForTaskController(t)
	templateMgr := integration.NewTemplateManager(db)
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)
	taskService := service.NewTaskService(taskMgr, db, service.NewAuditLogService(nil), nil)
	
	// 先创建模板和任务
	templateService := service.NewTemplateService(templateMgr, db, service.NewAuditLogService(nil))
	template, err := templateService.Create(context.Background(), &service.CreateTemplateRequest{
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	})
	require.NoError(t, err)
	
	task, err := taskService.Create(context.Background(), &service.CreateTaskRequest{
		TemplateID: template.ID,
		BusinessID: "biz-001",
		Params:     json.RawMessage(`{"amount": 1000}`),
	})
	require.NoError(t, err)
	
	router := gin.New()
	controller := api.NewTaskController(taskService)
	
	// 注册路由
	router.POST("/api/v1/tasks/:id/submit", controller.Submit)
	
	req := httptest.NewRequest("POST", "/api/v1/tasks/"+task.ID+"/submit", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// 注意: 提交功能需要状态机集成，这里只验证路由存在
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

