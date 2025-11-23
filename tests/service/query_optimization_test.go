package service_test

import (
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/integration"
	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-gin/internal/service"
	"github.com/mautops/approval-kit/pkg/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDBForQueryOptimization 创建查询优化测试数据库
func setupTestDBForQueryOptimization(t *testing.T) *gorm.DB {
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

// TestQueryService_ListTasks_NPlusOneProblem 测试 N+1 查询问题
func TestQueryService_ListTasks_NPlusOneProblem(t *testing.T) {
	db := setupTestDBForQueryOptimization(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()

	// 创建模板
	templateMgr := integration.NewTemplateManager(db)
	template := &template.Template{
		ID:          "tpl-001",
		Name:        "Test Template",
		Description: "Test",
		Version:     1,
		Nodes:       map[string]*template.Node{},
		Edges:       []*template.Edge{},
		Config:      &template.TemplateConfig{},
	}
	err = templateMgr.Create(template)
	require.NoError(t, err)

	// 创建测试数据：10 个任务
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)
	for i := 0; i < 10; i++ {
		// 创建任务
		_, err := taskMgr.Create("tpl-001", "biz-001", nil)
		require.NoError(t, err)
	}

	// 记录查询次数
	queryCount := 0
	db.Callback().Query().Before("gorm:query").Register("count_queries", func(d *gorm.DB) {
		queryCount++
	})

	// 创建查询服务
	querySvc := service.NewQueryService(db, taskMgr)

	// 执行查询
	filter := &service.ListTasksFilter{
		Page:     1,
		PageSize: 10,
	}
	tasks, total, err := querySvc.ListTasks(filter)
	require.NoError(t, err)
	assert.Equal(t, int64(10), total)
	assert.Len(t, tasks, 10)

	// 验证查询次数：优化后应该只有 2 次查询（Count + Find），而不是 12 次（Count + Find + 10*Get）
	t.Logf("Query count: %d (expected: 2 after optimization)", queryCount)
	// 优化后应该只有 2 次查询（1 次 Count, 1 次 Find），不再有 N+1 问题
	assert.Equal(t, 2, queryCount, "Should have only 2 queries after optimization (Count + Find)")
}

// TestQueryService_ListTasks_Performance 测试查询性能
func TestQueryService_ListTasks_Performance(t *testing.T) {
	db := setupTestDBForQueryOptimization(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()

	// 创建模板
	templateMgr := integration.NewTemplateManager(db)
	template := &template.Template{
		ID:          "tpl-001",
		Name:        "Test Template",
		Description: "Test",
		Version:     1,
		Nodes:       map[string]*template.Node{},
		Edges:       []*template.Edge{},
		Config:      &template.TemplateConfig{},
	}
	err = templateMgr.Create(template)
	require.NoError(t, err)

	// 创建大量测试数据：100 个任务
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)

	for i := 0; i < 100; i++ {
		_, err = taskMgr.Create("tpl-001", "biz-001", nil)
		require.NoError(t, err)
	}

	// 创建查询服务
	querySvc := service.NewQueryService(db, taskMgr)

	// 测试查询性能
	start := time.Now()
	filter := &service.ListTasksFilter{
		Page:     1,
		PageSize: 20,
	}
	tasks, total, err := querySvc.ListTasks(filter)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, int64(100), total)
	assert.Len(t, tasks, 20)

	// 验证查询时间应该在合理范围内（优化后应该更快）
	t.Logf("Query duration: %v", duration)
	assert.Less(t, duration, 1*time.Second, "Query should complete within 1 second")
}

// TestTemplateService_List_NPlusOneProblem 测试模板列表查询的 N+1 问题
func TestTemplateService_List_NPlusOneProblem(t *testing.T) {
	db := setupTestDBForQueryOptimization(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()

	// 创建测试数据：10 个模板
	templateMgr := integration.NewTemplateManager(db)
	for i := 0; i < 10; i++ {
		template := &template.Template{
			ID:          "tpl-001",
			Name:        "Test Template",
			Description: "Test",
			Version:     i + 1,
			Nodes:       map[string]*template.Node{},
			Edges:       []*template.Edge{},
			Config:      &template.TemplateConfig{},
		}
		err := templateMgr.Create(template)
		require.NoError(t, err)
	}

	// 创建模板服务
	templateSvc := service.NewTemplateService(templateMgr, db, nil)

	// 记录查询次数
	queryCount := 0
	db.Callback().Query().Before("gorm:query").Register("count_queries", func(db *gorm.DB) {
		queryCount++
	})

	// 执行查询
	filter := &service.TemplateListFilter{
		Page:     1,
		PageSize: 10,
	}
	response, err := templateSvc.List(filter)
	require.NoError(t, err)
	assert.Equal(t, int64(10), response.Pagination.Total)
	assert.Len(t, response.Data, 10)

	// 验证查询次数：优化后应该只有 2 次查询（Count + Find），而不是 12 次（Count + Find + 10*Get）
	t.Logf("Query count: %d (expected: 2 after optimization)", queryCount)
	// 优化后应该只有 2 次查询（1 次 Count, 1 次 Find），不再有 N+1 问题
	assert.Equal(t, 2, queryCount, "Should have only 2 queries after optimization (Count + Find)")
}

