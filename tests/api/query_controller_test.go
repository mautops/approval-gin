package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/mautops/approval-gin/internal/integration"
	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-gin/internal/repository"
	"github.com/mautops/approval-gin/internal/service"
	"github.com/mautops/approval-kit/pkg/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDBForQueryController 创建查询控制器测试数据库
func setupTestDBForQueryController(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 迁移数据库
	err = db.AutoMigrate(
		&model.TemplateModel{},
		&model.TaskModel{},
		&model.ApprovalRecordModel{},
		&model.StateHistoryModel{},
	)
	require.NoError(t, err)

	return db
}

// TestQueryController_ListTasks 测试任务列表查询
func TestQueryController_ListTasks(t *testing.T) {
	db := setupTestDBForQueryController(t)
	templateMgr := integration.NewTemplateManager(db)
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)
	
	// 创建模板和任务
	templateService := service.NewTemplateService(templateMgr, db, service.NewAuditLogService(nil))
	template, err := templateService.Create(context.Background(), &service.CreateTemplateRequest{
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	})
	require.NoError(t, err)
	
	taskService := service.NewTaskService(taskMgr, db, service.NewAuditLogService(nil), nil)
	_, err = taskService.Create(context.Background(), &service.CreateTaskRequest{
		TemplateID: template.ID,
		BusinessID: "biz-001",
		Params:     json.RawMessage(`{"amount": 1000}`),
	})
	require.NoError(t, err)
	
	queryService := service.NewQueryService(db, taskMgr)
	router := gin.New()
	controller := api.NewQueryController(queryService)
	
	// 注册路由
	router.GET("/api/v1/tasks", controller.ListTasks)
	
	req := httptest.NewRequest("GET", "/api/v1/tasks", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response api.PaginatedResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 0, response.Code)
}

// TestQueryController_GetRecords 测试获取审批记录
func TestQueryController_GetRecords(t *testing.T) {
	db := setupTestDBForQueryController(t)
	templateMgr := integration.NewTemplateManager(db)
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)
	
	// 创建模板和任务
	templateService := service.NewTemplateService(templateMgr, db, service.NewAuditLogService(nil))
	template, err := templateService.Create(context.Background(), &service.CreateTemplateRequest{
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	})
	require.NoError(t, err)
	
	taskService := service.NewTaskService(taskMgr, db, service.NewAuditLogService(nil), nil)
	task, err := taskService.Create(context.Background(), &service.CreateTaskRequest{
		TemplateID: template.ID,
		BusinessID: "biz-001",
		Params:     json.RawMessage(`{"amount": 1000}`),
	})
	require.NoError(t, err)
	
	// 创建审批记录
	recordRepo := repository.NewApprovalRecordRepository(db)
	attachmentsJSON, _ := json.Marshal([]string{"file1.pdf"})
	record := &model.ApprovalRecordModel{
		ID:          "record-001",
		TaskID:      task.ID,
		NodeID:      "node-001",
		Approver:    "user-001",
		Result:      "approve",
		Comment:     "同意",
		Attachments: attachmentsJSON,
	}
	err = recordRepo.Save(record)
	require.NoError(t, err)
	
	queryService := service.NewQueryService(db, taskMgr)
	router := gin.New()
	controller := api.NewQueryController(queryService)
	
	// 注册路由
	router.GET("/api/v1/tasks/:id/records", controller.GetRecords)
	
	req := httptest.NewRequest("GET", "/api/v1/tasks/"+task.ID+"/records", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response api.Response
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 0, response.Code)
}

// TestQueryController_GetHistory 测试获取状态历史
func TestQueryController_GetHistory(t *testing.T) {
	db := setupTestDBForQueryController(t)
	templateMgr := integration.NewTemplateManager(db)
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)
	
	// 创建模板和任务
	templateService := service.NewTemplateService(templateMgr, db, service.NewAuditLogService(nil))
	template, err := templateService.Create(context.Background(), &service.CreateTemplateRequest{
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	})
	require.NoError(t, err)
	
	taskService := service.NewTaskService(taskMgr, db, service.NewAuditLogService(nil), nil)
	task, err := taskService.Create(context.Background(), &service.CreateTaskRequest{
		TemplateID: template.ID,
		BusinessID: "biz-001",
		Params:     json.RawMessage(`{"amount": 1000}`),
	})
	require.NoError(t, err)
	
	// 创建状态历史
	historyRepo := repository.NewStateHistoryRepository(db)
	history := &model.StateHistoryModel{
		ID:        "history-001",
		TaskID:    task.ID,
		FromState: "pending",
		ToState:   "submitted",
		Reason:    "任务提交",
		Operator:  "user-001",
	}
	err = historyRepo.Save(history)
	require.NoError(t, err)
	
	queryService := service.NewQueryService(db, taskMgr)
	router := gin.New()
	controller := api.NewQueryController(queryService)
	
	// 注册路由
	router.GET("/api/v1/tasks/:id/history", controller.GetHistory)
	
	req := httptest.NewRequest("GET", "/api/v1/tasks/"+task.ID+"/history", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response api.Response
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 0, response.Code)
}

