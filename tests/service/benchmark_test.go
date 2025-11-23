package service_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/integration"
	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-gin/internal/repository"
	"github.com/mautops/approval-gin/internal/service"
	"github.com/mautops/approval-kit/pkg/template"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// BenchmarkTaskService_Create 基准测试: 任务创建性能
func BenchmarkTaskService_Create(b *testing.B) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}

	// 执行数据库迁移
	err = db.AutoMigrate(
		&model.TemplateModel{},
		&model.TaskModel{},
		&model.ApprovalRecordModel{},
		&model.StateHistoryModel{},
	)
	if err != nil {
		b.Fatalf("Failed to migrate: %v", err)
	}

	// 初始化服务
	templateMgr := integration.NewTemplateManager(db)
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)
	auditLogSvc := service.NewAuditLogService(repository.NewAuditLogRepository(db))
	taskSvc := service.NewTaskService(taskMgr, db, auditLogSvc, nil)

	// 创建测试模板（使用唯一的模板 ID）
	templateID := fmt.Sprintf("tpl-%s-%d", b.Name(), time.Now().UnixNano())
	tpl := &template.Template{
		ID:          templateID,
		Name:        "Benchmark Template",
		Description: "Template for benchmark",
		Version:     1,
		Nodes: map[string]*template.Node{
			"start": {
				ID:    "start",
				Name:  "开始",
				Type:  "start",
				Order: 1,
			},
		},
		Edges: []*template.Edge{},
		Config: nil,
	}
	err = templateMgr.Create(tpl)
	if err != nil {
		b.Fatalf("Failed to create template: %v", err)
	}

	ctx := context.Background()
	req := &service.CreateTaskRequest{
		TemplateID: templateID,
		BusinessID: "biz-bench-001",
		Params:     json.RawMessage(`{"amount": 1000}`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req.BusinessID = fmt.Sprintf("biz-bench-%d", i)
		_, err := taskSvc.Create(ctx, req)
		if err != nil {
			b.Fatalf("Failed to create task: %v", err)
		}
	}
}

// BenchmarkTaskService_Get 基准测试: 任务查询性能
func BenchmarkTaskService_Get(b *testing.B) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}

	// 执行数据库迁移
	err = db.AutoMigrate(
		&model.TemplateModel{},
		&model.TaskModel{},
		&model.ApprovalRecordModel{},
		&model.StateHistoryModel{},
	)
	if err != nil {
		b.Fatalf("Failed to migrate: %v", err)
	}

	// 初始化服务
	templateMgr := integration.NewTemplateManager(db)
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)
	auditLogSvc := service.NewAuditLogService(repository.NewAuditLogRepository(db))
	taskSvc := service.NewTaskService(taskMgr, db, auditLogSvc, nil)

	// 创建测试模板和任务（使用唯一的模板 ID）
	templateID := fmt.Sprintf("tpl-%s-%d", b.Name(), time.Now().UnixNano())
	tpl := &template.Template{
		ID:          templateID,
		Name:        "Benchmark Template",
		Description: "Template for benchmark",
		Version:     1,
		Nodes: map[string]*template.Node{
			"start": {
				ID:    "start",
				Name:  "开始",
				Type:  "start",
				Order: 1,
			},
		},
		Edges: []*template.Edge{},
		Config: nil,
	}
	err = templateMgr.Create(tpl)
	if err != nil {
		b.Fatalf("Failed to create template: %v", err)
	}

	task, err := taskMgr.Create(templateID, "biz-bench-002", json.RawMessage(`{"amount": 1000}`))
	if err != nil {
		b.Fatalf("Failed to create task: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := taskSvc.Get(task.ID)
		if err != nil {
			b.Fatalf("Failed to get task: %v", err)
		}
	}
}

// BenchmarkTemplateService_Get 基准测试: 模板查询性能(带缓存)
func BenchmarkTemplateService_Get(b *testing.B) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}

	// 执行数据库迁移
	err = db.AutoMigrate(&model.TemplateModel{})
	if err != nil {
		b.Fatalf("Failed to migrate: %v", err)
	}

	// 初始化服务
	templateMgr := integration.NewTemplateManager(db)
	auditLogSvc := service.NewAuditLogService(repository.NewAuditLogRepository(db))
	templateSvc := service.NewTemplateService(templateMgr, db, auditLogSvc, nil)

	// 创建测试模板（使用唯一的模板 ID）
	templateID := fmt.Sprintf("tpl-%s-%d", b.Name(), time.Now().UnixNano())
	tpl := &template.Template{
		ID:          templateID,
		Name:        "Benchmark Template",
		Description: "Template for benchmark",
		Version:     1,
		Nodes: map[string]*template.Node{
			"start": {
				ID:    "start",
				Name:  "开始",
				Type:  "start",
				Order: 1,
			},
		},
		Edges: []*template.Edge{},
		Config: nil,
	}
	err = templateMgr.Create(tpl)
	if err != nil {
		b.Fatalf("Failed to create template: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := templateSvc.Get(templateID, 0)
		if err != nil {
			b.Fatalf("Failed to get template: %v", err)
		}
	}
}

// BenchmarkQueryService_ListTasks 基准测试: 任务列表查询性能
func BenchmarkQueryService_ListTasks(b *testing.B) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}

	// 执行数据库迁移
	err = db.AutoMigrate(
		&model.TemplateModel{},
		&model.TaskModel{},
		&model.ApprovalRecordModel{},
		&model.StateHistoryModel{},
	)
	if err != nil {
		b.Fatalf("Failed to migrate: %v", err)
	}

	// 初始化服务
	templateMgr := integration.NewTemplateManager(db)
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)
	querySvc := service.NewQueryService(db, taskMgr)

	// 创建测试模板（使用唯一的模板 ID）
	templateID := fmt.Sprintf("tpl-%s-%d", b.Name(), time.Now().UnixNano())
	tpl := &template.Template{
		ID:          templateID,
		Name:        "Benchmark Template",
		Description: "Template for benchmark",
		Version:     1,
		Nodes: map[string]*template.Node{
			"start": {
				ID:    "start",
				Name:  "开始",
				Type:  "start",
				Order: 1,
			},
		},
		Edges: []*template.Edge{},
		Config: nil,
	}
	err = templateMgr.Create(tpl)
	if err != nil {
		b.Fatalf("Failed to create template: %v", err)
	}

	// 创建测试数据
	for i := 0; i < 100; i++ {
		_, err = taskMgr.Create(templateID, "biz-bench-004", json.RawMessage(`{"amount": 1000}`))
		if err != nil {
			b.Fatalf("Failed to create task: %v", err)
		}
	}

	filter := &service.ListTasksFilter{
		Page:     1,
		PageSize: 20,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := querySvc.ListTasks(filter)
		if err != nil {
			b.Fatalf("Failed to list tasks: %v", err)
		}
	}
}

// BenchmarkTaskManager_Submit 基准测试: 任务提交性能
func BenchmarkTaskManager_Submit(b *testing.B) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}

	// 执行数据库迁移
	err = db.AutoMigrate(
		&model.TemplateModel{},
		&model.TaskModel{},
		&model.ApprovalRecordModel{},
		&model.StateHistoryModel{},
	)
	if err != nil {
		b.Fatalf("Failed to migrate: %v", err)
	}

	// 初始化服务
	templateMgr := integration.NewTemplateManager(db)
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)

	// 创建测试模板（使用唯一的模板 ID）
	templateID := fmt.Sprintf("tpl-%s-%d", b.Name(), time.Now().UnixNano())
	tpl := &template.Template{
		ID:          templateID,
		Name:        "Benchmark Template",
		Description: "Template for benchmark",
		Version:     1,
		Nodes: map[string]*template.Node{
			"start": {
				ID:    "start",
				Name:  "开始",
				Type:  "start",
				Order: 1,
			},
		},
		Edges: []*template.Edge{},
		Config: nil,
	}
	err = templateMgr.Create(tpl)
	if err != nil {
		b.Fatalf("Failed to create template: %v", err)
	}

	// 预先创建任务
	tasks := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		task, err := taskMgr.Create(templateID, "biz-bench-005", json.RawMessage(`{"amount": 1000}`))
		if err != nil {
			b.Fatalf("Failed to create task: %v", err)
		}
		tasks[i] = task.ID
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := taskMgr.Submit(tasks[i])
		if err != nil {
			b.Fatalf("Failed to submit task: %v", err)
		}
	}
}

