package integration_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/integration"
	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-kit/pkg/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDBForTask 创建任务测试数据库
func setupTestDBForTask(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 迁移数据库
	err = db.AutoMigrate(
		&model.TemplateModel{},
		&model.TaskModel{},
		&model.StateHistoryModel{},
		&model.ApprovalRecordModel{},
	)
	require.NoError(t, err)

	return db
}

// TestTaskManager_Create 测试创建任务
func TestTaskManager_Create(t *testing.T) {
	db := setupTestDBForTask(t)
	
	// 先创建模板
	templateMgr := integration.NewTemplateManager(db)
	template := &template.Template{
		ID:          "tpl-001",
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	}
	err := templateMgr.Create(template)
	require.NoError(t, err)

	// 创建任务管理器
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)

	// 创建任务
	params := json.RawMessage(`{"amount": 1000}`)
	tsk, err := taskMgr.Create("tpl-001", "biz-001", params)
	assert.NoError(t, err)
	assert.NotNil(t, tsk)
	assert.Equal(t, "tpl-001", tsk.TemplateID)
	assert.Equal(t, "biz-001", tsk.BusinessID)
	assert.Equal(t, "pending", string(tsk.State))

	// 验证任务已保存
	var tm model.TaskModel
	err = db.Where("id = ?", tsk.ID).First(&tm).Error
	assert.NoError(t, err)
	assert.Equal(t, tsk.ID, tm.ID)
	assert.Equal(t, "tpl-001", tm.TemplateID)
}

// TestTaskManager_Get 测试获取任务
func TestTaskManager_Get(t *testing.T) {
	db := setupTestDBForTask(t)
	
	// 先创建模板
	templateMgr := integration.NewTemplateManager(db)
	template := &template.Template{
		ID:          "tpl-001",
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	}
	err := templateMgr.Create(template)
	require.NoError(t, err)

	// 创建任务管理器
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)

	// 先创建任务
	params := json.RawMessage(`{"amount": 1000}`)
	created, err := taskMgr.Create("tpl-001", "biz-001", params)
	require.NoError(t, err)

	// 获取任务
	got, err := taskMgr.Get(created.ID)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "tpl-001", got.TemplateID)
	assert.Equal(t, "biz-001", got.BusinessID)
}

