package service_test

import (
	"testing"

	"github.com/mautops/approval-gin/internal/integration"
	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-gin/internal/service"
	"github.com/mautops/approval-gin/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDBForSQLInjection 创建 SQL 注入测试数据库
func setupTestDBForSQLInjection(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

// TestSQLInjection_SortField 测试排序字段 SQL 注入防护
func TestSQLInjection_SortField(t *testing.T) {
	// 测试 SQL 注入 payload
	sqlPayloads := []string{
		"created_at; DROP TABLE tasks; --",
		"created_at' OR '1'='1",
		"created_at UNION SELECT * FROM tasks",
		"created_at; DELETE FROM tasks; --",
	}

	for _, payload := range sqlPayloads {
		err := utils.ValidateSortField(payload)
		assert.Error(t, err, "Should reject SQL injection in sort field: %s", payload)
		assert.Contains(t, err.Error(), "invalid", "Error should indicate invalid field")
	}

	// 测试有效字段
	validFields := []string{
		"created_at",
		"updated_at",
		"name",
		"state",
		"tasks.created_at",
		"templates.name",
	}

	for _, field := range validFields {
		err := utils.ValidateSortField(field)
		assert.NoError(t, err, "Should accept valid sort field: %s", field)
	}
}

// TestSQLInjection_SortOrder 测试排序方向 SQL 注入防护
func TestSQLInjection_SortOrder(t *testing.T) {
	// 测试 SQL 注入 payload
	sqlPayloads := []string{
		"desc; DROP TABLE tasks; --",
		"asc' OR '1'='1",
		"desc UNION SELECT * FROM tasks",
	}

	for _, payload := range sqlPayloads {
		err := utils.ValidateSortOrder(payload)
		assert.Error(t, err, "Should reject SQL injection in sort order: %s", payload)
	}

	// 测试有效方向
	validOrders := []string{
		"asc",
		"ASC",
		"desc",
		"DESC",
		" asc ",
		" desc ",
	}

	for _, order := range validOrders {
		err := utils.ValidateSortOrder(order)
		assert.NoError(t, err, "Should accept valid sort order: %s", order)
	}
}

// TestSQLInjection_QueryService 测试查询服务的 SQL 注入防护
func TestSQLInjection_QueryService(t *testing.T) {
	db := setupTestDBForSQLInjection(t)
	// 迁移数据库（创建表）
	err := db.AutoMigrate(&model.TaskModel{})
	require.NoError(t, err)
	
	templateMgr := integration.NewTemplateManager(db)
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)
	querySvc := service.NewQueryService(db, taskMgr)

	// 测试 SQL 注入在排序字段中
	filter := &service.ListTasksFilter{
		Page:     1,
		PageSize: 20,
		SortBy:   "created_at; DROP TABLE tasks; --",
		Order:    "desc",
	}

	_, _, err = querySvc.ListTasks(filter)
	assert.Error(t, err, "Should reject SQL injection in sort field")
	assert.Contains(t, err.Error(), "invalid sort field", "Error should indicate invalid sort field")
}

// TestSQLInjection_TemplateService 测试模板服务的 SQL 注入防护
func TestSQLInjection_TemplateService(t *testing.T) {
	db := setupTestDBForSQLInjection(t)
	// 迁移数据库（创建表）
	err := db.AutoMigrate(&model.TemplateModel{})
	require.NoError(t, err)
	
	templateMgr := integration.NewTemplateManager(db)
	auditLogSvc := service.NewAuditLogService(nil)
	templateSvc := service.NewTemplateService(templateMgr, db, auditLogSvc)

	// 测试 SQL 注入在排序字段中
	filter := &service.TemplateListFilter{
		Page:     1,
		PageSize: 20,
		SortBy:   "created_at; DROP TABLE templates; --",
		Order:    "desc",
	}

	_, err = templateSvc.List(filter)
	assert.Error(t, err, "Should reject SQL injection in sort field")
	assert.Contains(t, err.Error(), "invalid sort field", "Error should indicate invalid sort field")
}

