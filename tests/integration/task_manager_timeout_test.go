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

// setupTestDBForTimeout 创建超时测试数据库
func setupTestDBForTimeout(t *testing.T) *gorm.DB {
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

// TestTaskManager_HandleTimeout_NoConfig 测试节点配置为 nil 时的超时处理
func TestTaskManager_HandleTimeout_NoConfig(t *testing.T) {
	db := setupTestDBForTimeout(t)

	// 创建模板，节点配置为 nil
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
			"approval": {
				ID:    "approval",
				Name:  "审批节点",
				Type:  template.NodeTypeApproval,
				Order: 2,
				Config: nil, // 配置为 nil，应该不会处理超时
			},
		},
		Edges: []*template.Edge{
			{From: "start", To: "approval"},
		},
		Config: nil,
	}
	err := templateMgr.Create(tpl)
	require.NoError(t, err)

	// 创建任务管理器
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)

	// 创建并提交任务
	task, err := taskMgr.Create("tpl-001", "biz-001", json.RawMessage(`{}`))
	require.NoError(t, err)
	
	// 提交任务
	err = taskMgr.Submit(task.ID)
	require.NoError(t, err)

	// 验证任务状态为 submitted 或 approving
	task, err = taskMgr.Get(task.ID)
	require.NoError(t, err)
	originalState := task.GetState()
	assert.True(t, originalState == types.TaskStateSubmitted || originalState == types.TaskStateApproving,
		"task state should be submitted or approving after submit")

	// 处理超时（由于节点配置为 nil，应该直接返回，不处理超时）
	err = taskMgr.HandleTimeout(task.ID)
	require.NoError(t, err)

	// 验证任务状态未改变（因为节点配置为 nil，不会处理超时）
	task, err = taskMgr.Get(task.ID)
	require.NoError(t, err)
	assert.Equal(t, originalState, task.GetState(), "task state should not change when node config is nil")
}

// TestTaskManager_HandleTimeout_NotInTimeoutState 测试不在需要检查超时状态的任务
func TestTaskManager_HandleTimeout_NotInTimeoutState(t *testing.T) {
	db := setupTestDBForTimeout(t)

	// 创建模板
	templateMgr := integration.NewTemplateManager(db)
	tpl := &template.Template{
		ID:          "tpl-002",
		Name:        "测试模板2",
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

	// 创建任务（不提交，保持 pending 状态）
	task, err := taskMgr.Create("tpl-002", "biz-002", json.RawMessage(`{}`))
	require.NoError(t, err)

	// 验证任务状态为 pending
	task, err = taskMgr.Get(task.ID)
	require.NoError(t, err)
	assert.Equal(t, types.TaskStatePending, task.GetState(), "task state should be pending")

	// 处理超时（pending 状态不需要检查超时，应该直接返回）
	err = taskMgr.HandleTimeout(task.ID)
	require.NoError(t, err)

	// 验证任务状态未改变
	task, err = taskMgr.Get(task.ID)
	require.NoError(t, err)
	assert.Equal(t, types.TaskStatePending, task.GetState(), "task state should remain pending")
}

