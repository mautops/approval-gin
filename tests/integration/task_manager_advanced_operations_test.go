package integration_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/integration"
	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-kit/pkg/template"
	"github.com/mautops/approval-kit/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDBForAdvancedOps 创建高级操作测试数据库
func setupTestDBForAdvancedOps(t *testing.T) *gorm.DB {
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

// TestTaskManager_Pause 测试暂停任务
func TestTaskManager_Pause(t *testing.T) {
	db := setupTestDBForAdvancedOps(t)

	// 创建模板
	templateMgr := integration.NewTemplateManager(db)
	tpl := &template.Template{
		ID:          "tpl-001",
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Nodes: map[string]*template.Node{
			"start": {
				ID:    "start",
				Name:  "开始",
				Type:  template.NodeTypeStart,
				Order: 1,
				Config: nil,
			},
		},
		Edges: []*template.Edge{},
		Config: nil,
	}
	err := templateMgr.Create(tpl)
	require.NoError(t, err)

	// 创建任务管理器
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)

	// 创建任务
	task, err := taskMgr.Create("tpl-001", "biz-001", json.RawMessage(`{}`))
	require.NoError(t, err)
	assert.Equal(t, types.TaskStatePending, task.State)

	// 暂停任务
	err = taskMgr.Pause(task.ID, "测试暂停")
	require.NoError(t, err)

	// 验证任务状态
	pausedTask, err := taskMgr.Get(task.ID)
	require.NoError(t, err)
	assert.Equal(t, types.TaskStatePaused, pausedTask.State)
	assert.NotNil(t, pausedTask.PausedAt)
	assert.Equal(t, types.TaskStatePending, pausedTask.PausedState)
}

// TestTaskManager_Resume 测试恢复任务
func TestTaskManager_Resume(t *testing.T) {
	db := setupTestDBForAdvancedOps(t)

	// 创建模板
	templateMgr := integration.NewTemplateManager(db)
	tpl := &template.Template{
		ID:          "tpl-001",
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Nodes: map[string]*template.Node{
			"start": {
				ID:    "start",
				Name:  "开始",
				Type:  template.NodeTypeStart,
				Order: 1,
				Config: nil,
			},
		},
		Edges: []*template.Edge{},
		Config: nil,
	}
	err := templateMgr.Create(tpl)
	require.NoError(t, err)

	// 创建任务管理器
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)

	// 创建任务
	task, err := taskMgr.Create("tpl-001", "biz-001", json.RawMessage(`{}`))
	require.NoError(t, err)

	// 暂停任务
	err = taskMgr.Pause(task.ID, "测试暂停")
	require.NoError(t, err)

	// 恢复任务
	err = taskMgr.Resume(task.ID, "测试恢复")
	require.NoError(t, err)

	// 验证任务状态
	resumedTask, err := taskMgr.Get(task.ID)
	require.NoError(t, err)
	assert.Equal(t, types.TaskStatePending, resumedTask.State)
	assert.Nil(t, resumedTask.PausedAt)
	assert.Equal(t, types.TaskState(""), resumedTask.PausedState) // 恢复后应该清空暂停前状态
}

// TestTaskManager_PauseFromSubmitted 测试从 submitted 状态暂停
func TestTaskManager_PauseFromSubmitted(t *testing.T) {
	db := setupTestDBForAdvancedOps(t)

	// 创建模板
	templateMgr := integration.NewTemplateManager(db)
	tpl := &template.Template{
		ID:          "tpl-001",
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Nodes: map[string]*template.Node{
			"start": {
				ID:    "start",
				Name:  "开始",
				Type:  template.NodeTypeStart,
				Order: 1,
				Config: nil,
			},
		},
		Edges: []*template.Edge{},
		Config: nil,
	}
	err := templateMgr.Create(tpl)
	require.NoError(t, err)

	// 创建任务管理器
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)

	// 创建并提交任务
	task, err := taskMgr.Create("tpl-001", "biz-001", json.RawMessage(`{}`))
	require.NoError(t, err)
	err = taskMgr.Submit(task.ID)
	require.NoError(t, err)

	// 暂停任务
	err = taskMgr.Pause(task.ID, "测试暂停")
	require.NoError(t, err)

	// 验证任务状态
	pausedTask, err := taskMgr.Get(task.ID)
	require.NoError(t, err)
	assert.Equal(t, types.TaskStatePaused, pausedTask.State)
	assert.Equal(t, types.TaskStateSubmitted, pausedTask.PausedState)

	// 恢复任务
	err = taskMgr.Resume(task.ID, "测试恢复")
	require.NoError(t, err)

	// 验证任务恢复到 submitted 状态
	resumedTask, err := taskMgr.Get(task.ID)
	require.NoError(t, err)
	assert.Equal(t, types.TaskStateSubmitted, resumedTask.State)
}

// TestTaskManager_Withdraw 测试撤回任务
func TestTaskManager_Withdraw(t *testing.T) {
	db := setupTestDBForAdvancedOps(t)

	// 创建模板
	templateMgr := integration.NewTemplateManager(db)
	tpl := &template.Template{
		ID:          "tpl-001",
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Nodes: map[string]*template.Node{
			"start": {
				ID:    "start",
				Name:  "开始",
				Type:  template.NodeTypeStart,
				Order: 1,
				Config: nil,
			},
		},
		Edges: []*template.Edge{},
		Config: nil,
	}
	err := templateMgr.Create(tpl)
	require.NoError(t, err)

	// 创建任务管理器
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)

	// 创建并提交任务
	task, err := taskMgr.Create("tpl-001", "biz-001", json.RawMessage(`{}`))
	require.NoError(t, err)
	err = taskMgr.Submit(task.ID)
	require.NoError(t, err)

	// 撤回任务
	err = taskMgr.Withdraw(task.ID, "测试撤回")
	require.NoError(t, err)

	// 验证任务状态
	withdrawnTask, err := taskMgr.Get(task.ID)
	require.NoError(t, err)
	assert.Equal(t, types.TaskStatePending, withdrawnTask.State)
	assert.Nil(t, withdrawnTask.SubmittedAt)
}

// TestTaskManager_WithdrawWithRecords 测试有审批记录时不允许撤回
func TestTaskManager_WithdrawWithRecords(t *testing.T) {
	db := setupTestDBForAdvancedOps(t)

	// 创建模板
	templateMgr := integration.NewTemplateManager(db)
	tpl := &template.Template{
		ID:          "tpl-001",
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Nodes: map[string]*template.Node{
			"start": {
				ID:    "start",
				Name:  "开始",
				Type:  template.NodeTypeStart,
				Order: 1,
				Config: nil,
			},
		},
		Edges: []*template.Edge{},
		Config: nil,
	}
	err := templateMgr.Create(tpl)
	require.NoError(t, err)

	// 创建任务管理器
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)

	// 创建并提交任务
	task, err := taskMgr.Create("tpl-001", "biz-001", json.RawMessage(`{}`))
	require.NoError(t, err)
	err = taskMgr.Submit(task.ID)
	require.NoError(t, err)

	// 注意: 由于当前没有实现 Approve 方法,无法创建审批记录
	// 这个测试暂时跳过,等实现 Approve 后再测试
	t.Skip("Withdraw with records test requires Approve method implementation")
}

