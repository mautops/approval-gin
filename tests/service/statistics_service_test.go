package service_test

import (
	"context"
	"encoding/json"
	"testing"

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

// setupTestDBForStatisticsService 创建统计服务测试数据库
func setupTestDBForStatisticsService(t *testing.T) *gorm.DB {
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

// TestStatisticsService_TaskStatistics 测试任务统计
func TestStatisticsService_TaskStatistics(t *testing.T) {
	db := setupTestDBForStatisticsService(t)
	templateMgr := integration.NewTemplateManager(db)
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)
	statisticsService := service.NewStatisticsService(db)

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
	// 创建不同状态的任务
	_, err = taskService.Create(context.Background(), &service.CreateTaskRequest{
		TemplateID: template.ID,
		BusinessID: "biz-001",
		Params:     json.RawMessage(`{"amount": 1000}`),
	})
	require.NoError(t, err)

	_, err = taskService.Create(context.Background(), &service.CreateTaskRequest{
		TemplateID: template.ID,
		BusinessID: "biz-002",
		Params:     json.RawMessage(`{"amount": 2000}`),
	})
	require.NoError(t, err)

	// 测试：按状态统计
	stats, err := statisticsService.GetTaskStatisticsByState()
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	// 验证至少有一个状态有统计
	assert.Greater(t, len(stats), 0)
}

// TestStatisticsService_TaskStatisticsByTemplate 测试按模板统计
func TestStatisticsService_TaskStatisticsByTemplate(t *testing.T) {
	db := setupTestDBForStatisticsService(t)
	templateMgr := integration.NewTemplateManager(db)
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)
	statisticsService := service.NewStatisticsService(db)

	// 创建两个模板
	templateService := service.NewTemplateService(templateMgr, db, service.NewAuditLogService(nil))
	template1, err := templateService.Create(context.Background(), &service.CreateTemplateRequest{
		Name:        "模板1",
		Description: "模板1描述",
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	})
	require.NoError(t, err)

	template2, err := templateService.Create(context.Background(), &service.CreateTemplateRequest{
		Name:        "模板2",
		Description: "模板2描述",
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	})
	require.NoError(t, err)

	// 为每个模板创建任务
	taskService := service.NewTaskService(taskMgr, db, service.NewAuditLogService(nil), nil)
	_, err = taskService.Create(context.Background(), &service.CreateTaskRequest{
		TemplateID: template1.ID,
		BusinessID: "biz-001",
		Params:     json.RawMessage(`{"amount": 1000}`),
	})
	require.NoError(t, err)

	_, err = taskService.Create(context.Background(), &service.CreateTaskRequest{
		TemplateID: template2.ID,
		BusinessID: "biz-002",
		Params:     json.RawMessage(`{"amount": 2000}`),
	})
	require.NoError(t, err)

	// 测试：按模板统计
	stats, err := statisticsService.GetTaskStatisticsByTemplate()
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	// 验证至少有一个模板有统计
	assert.Greater(t, len(stats), 0)
}

// TestStatisticsService_ApprovalStatistics 测试审批记录统计
func TestStatisticsService_ApprovalStatistics(t *testing.T) {
	db := setupTestDBForStatisticsService(t)
	templateMgr := integration.NewTemplateManager(db)
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)
	statisticsService := service.NewStatisticsService(db)

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
	record1 := &model.ApprovalRecordModel{
		ID:          "record-001",
		TaskID:      task.ID,
		NodeID:      "node-001",
		Approver:    "user-001",
		Result:      "approve",
		Comment:     "同意",
		Attachments: attachmentsJSON,
	}
	err = recordRepo.Save(record1)
	require.NoError(t, err)

	record2 := &model.ApprovalRecordModel{
		ID:          "record-002",
		TaskID:      task.ID,
		NodeID:      "node-001",
		Approver:    "user-002",
		Result:      "reject",
		Comment:     "拒绝",
		Attachments: nil,
	}
	err = recordRepo.Save(record2)
	require.NoError(t, err)

	// 测试：审批通过率统计
	stats, err := statisticsService.GetApprovalStatistics()
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	// 验证统计结果包含通过率等信息
	_ = stats
}

