package service_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/integration"
	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-gin/internal/repository"
	"github.com/mautops/approval-gin/internal/service"
	"github.com/mautops/approval-kit/pkg/template"
	"github.com/mautops/approval-kit/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDBForQueryService 创建查询服务测试数据库
func setupTestDBForQueryService(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 迁移数据库
	err = db.AutoMigrate(&model.TemplateModel{}, &model.TaskModel{}, &model.ApprovalRecordModel{}, &model.StateHistoryModel{})
	require.NoError(t, err)

	return db
}

// TestQueryService_ListTasks 测试任务列表查询
func TestQueryService_ListTasks(t *testing.T) {
	db := setupTestDBForQueryService(t)
	templateMgr := integration.NewTemplateManager(db)
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)
	queryService := service.NewQueryService(db, taskMgr)

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

	taskService := service.NewTaskService(taskMgr, db, service.NewAuditLogService(nil), nil)
	task1, err := taskService.Create(context.Background(), &service.CreateTaskRequest{
		TemplateID: template.ID,
		BusinessID: "biz-001",
		Params:     json.RawMessage(`{"amount": 1000}`),
	})
	require.NoError(t, err)

	task2, err := taskService.Create(context.Background(), &service.CreateTaskRequest{
		TemplateID: template.ID,
		BusinessID: "biz-002",
		Params:     json.RawMessage(`{"amount": 2000}`),
	})
	require.NoError(t, err)

	// 查询所有任务
	filter := &service.ListTasksFilter{}
	tasks, total, err := queryService.ListTasks(filter)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(2))
	assert.GreaterOrEqual(t, len(tasks), 2)

	// 验证返回的任务包含创建的任务
	taskIDs := make(map[string]bool)
	for _, task := range tasks {
		taskIDs[task.ID] = true
	}
	assert.True(t, taskIDs[task1.ID])
	assert.True(t, taskIDs[task2.ID])
}

// TestQueryService_ListTasks_WithFilter 测试带过滤条件的任务查询
func TestQueryService_ListTasks_WithFilter(t *testing.T) {
	db := setupTestDBForQueryService(t)
	templateMgr := integration.NewTemplateManager(db)
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)
	queryService := service.NewQueryService(db, taskMgr)

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

	taskService := service.NewTaskService(taskMgr, db, service.NewAuditLogService(nil), nil)
	_, err = taskService.Create(context.Background(), &service.CreateTaskRequest{
		TemplateID: template.ID,
		BusinessID: "biz-001",
		Params:     json.RawMessage(`{"amount": 1000}`),
	})
	require.NoError(t, err)

	// 按模板ID查询
	state := types.TaskStatePending
	filter := &service.ListTasksFilter{
		TemplateID: &template.ID,
		State:      &state,
	}
	tasks, total, err := queryService.ListTasks(filter)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(1))
	assert.GreaterOrEqual(t, len(tasks), 1)
	if len(tasks) > 0 {
		assert.Equal(t, template.ID, tasks[0].TemplateID)
		assert.Equal(t, types.TaskStatePending, tasks[0].State)
	}

	// 按业务ID查询
	businessID := "biz-001"
	filter2 := &service.ListTasksFilter{
		BusinessID: &businessID,
	}
	tasks2, total2, err := queryService.ListTasks(filter2)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, total2, int64(1))
	assert.GreaterOrEqual(t, len(tasks2), 1)
	if len(tasks2) > 0 {
		assert.Equal(t, "biz-001", tasks2[0].BusinessID)
	}
}

// TestQueryService_GetRecords 测试获取审批记录
func TestQueryService_GetRecords(t *testing.T) {
	db := setupTestDBForQueryService(t)
	templateMgr := integration.NewTemplateManager(db)
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)
	queryService := service.NewQueryService(db, taskMgr)

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

	taskService := service.NewTaskService(taskMgr, db, service.NewAuditLogService(nil), nil)
	createdTask, err := taskService.Create(context.Background(), &service.CreateTaskRequest{
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
		TaskID:      createdTask.ID,
		NodeID:      "node-001",
		Approver:    "user-001",
		Result:      "approve",
		Comment:     "同意",
		Attachments: attachmentsJSON,
		CreatedAt:   time.Now(),
	}
	err = recordRepo.Save(record)
	require.NoError(t, err)

	// 获取审批记录
	records, err := queryService.GetRecords(createdTask.ID)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(records), 1)
	if len(records) > 0 {
		assert.Equal(t, createdTask.ID, records[0].TaskID)
		assert.Equal(t, "node-001", records[0].NodeID)
		assert.Equal(t, "user-001", records[0].Approver)
	}
}

// TestQueryService_GetHistory 测试获取状态历史
func TestQueryService_GetHistory(t *testing.T) {
	db := setupTestDBForQueryService(t)
	templateMgr := integration.NewTemplateManager(db)
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)
	queryService := service.NewQueryService(db, taskMgr)

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

	taskService := service.NewTaskService(taskMgr, db, service.NewAuditLogService(nil), nil)
	createdTask, err := taskService.Create(context.Background(), &service.CreateTaskRequest{
		TemplateID: template.ID,
		BusinessID: "biz-001",
		Params:     json.RawMessage(`{"amount": 1000}`),
	})
	require.NoError(t, err)

	// 创建状态历史
	historyRepo := repository.NewStateHistoryRepository(db)
	history := &model.StateHistoryModel{
		ID:        "history-001",
		TaskID:    createdTask.ID,
		FromState: "",
		ToState:   "pending",
		Reason:    "任务创建",
		Operator:  "system",
		CreatedAt: time.Now(),
	}
	err = historyRepo.Save(history)
	require.NoError(t, err)

	// 获取状态历史
	histories, err := queryService.GetHistory(createdTask.ID)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(histories), 1)
	if len(histories) > 0 {
		assert.Equal(t, createdTask.ID, histories[0].TaskID)
		assert.Equal(t, "pending", histories[0].ToState)
	}
}

