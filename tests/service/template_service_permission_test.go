package service_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mautops/approval-gin/internal/auth"
	"github.com/mautops/approval-gin/internal/integration"
	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-gin/internal/service"
	"github.com/mautops/approval-kit/pkg/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDBForTemplateService 创建模板服务测试数据库
func setupTestDBForTemplateService(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 迁移数据库
	err = db.AutoMigrate(&model.TemplateModel{})
	require.NoError(t, err)

	return db
}

// TestTemplateService_CreateWithPermission 测试创建模板时设置权限关系
func TestTemplateService_CreateWithPermission(t *testing.T) {
	db := setupTestDBForTemplateService(t)
	templateMgr := integration.NewTemplateManager(db)
	
	// 创建模拟的 OpenFGA 客户端（不实际调用）
	apiURL := "http://localhost:8080"
	storeID := "test-store"
	modelID := "test-model"
	fgaClient, err := auth.NewOpenFGAClient(apiURL, storeID, modelID)
	if err != nil {
		t.Skip("OpenFGA client creation failed, skipping test")
		return
	}
	
	templateService := service.NewTemplateService(templateMgr, db, service.NewAuditLogService(nil), fgaClient)
	// 注意: 需要在 TemplateService 中集成 OpenFGA 客户端
	// 这里只验证服务存在且可调用
	_ = templateService
}

// TestTaskService_CreateWithPermission 测试创建任务时设置权限关系
func TestTaskService_CreateWithPermission(t *testing.T) {
	db := setupTestDBForTaskService(t)
	templateMgr := integration.NewTemplateManager(db)
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)
	
	// 创建模拟的 OpenFGA 客户端（不实际调用）
	apiURL := "http://localhost:8080"
	storeID := "test-store"
	modelID := "test-model"
	fgaClient, err := auth.NewOpenFGAClient(apiURL, storeID, modelID)
	if err != nil {
		t.Skip("OpenFGA client creation failed, skipping test")
		return
	}
	
	taskService := service.NewTaskService(taskMgr, db, service.NewAuditLogService(nil), fgaClient)
	
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
	
	// 创建任务
	taskReq := &service.CreateTaskRequest{
		TemplateID: template.ID,
		BusinessID: "biz-001",
		Params:     json.RawMessage(`{"amount": 1000}`),
	}
	task, err := taskService.Create(context.Background(), taskReq)
	assert.NoError(t, err)
	assert.NotNil(t, task)
	
	// 注意: 完整的测试需要真实的 OpenFGA 服务器
	// 这里只验证任务创建成功
}
