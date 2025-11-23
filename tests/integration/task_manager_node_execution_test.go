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

// setupTestDBForNodeExecution 创建节点执行测试数据库
func setupTestDBForNodeExecution(t *testing.T) *gorm.DB {
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

// TestTaskManager_NodeExecution_SubmitWithStartNode 测试提交任务时执行开始节点
func TestTaskManager_NodeExecution_SubmitWithStartNode(t *testing.T) {
	db := setupTestDBForNodeExecution(t)

	// 创建模板,包含开始节点和审批节点
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
				Config: nil,
			},
		},
		Edges: []*template.Edge{
			{
				From: "start",
				To:   "approval",
			},
		},
		Config: nil,
	}
	err := templateMgr.Create(tpl)
	require.NoError(t, err)

	// 创建任务管理器
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)

	// 创建任务
	task, err := taskMgr.Create("tpl-001", "biz-001", json.RawMessage(`{"amount": 1000}`))
	require.NoError(t, err)
	assert.Equal(t, "start", task.CurrentNode)
	assert.Equal(t, types.TaskStatePending, task.State)

	// 提交任务,应该执行开始节点,找到下一个节点
	err = taskMgr.Submit(task.ID)
	require.NoError(t, err)

	// 验证任务状态和当前节点
	submittedTask, err := taskMgr.Get(task.ID)
	require.NoError(t, err)
	assert.Equal(t, types.TaskStateSubmitted, submittedTask.State)
	// 提交后,当前节点应该从 start 移动到下一个节点(approval)
	assert.Equal(t, "approval", submittedTask.CurrentNode)
}

// TestTaskManager_NodeExecution_SubmitWithNoNextNode 测试提交任务时如果没有下一个节点
func TestTaskManager_NodeExecution_SubmitWithNoNextNode(t *testing.T) {
	db := setupTestDBForNodeExecution(t)

	// 创建模板,只包含开始节点,没有下一个节点
	templateMgr := integration.NewTemplateManager(db)
	tpl := &template.Template{
		ID:          "tpl-002",
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
	task, err := taskMgr.Create("tpl-002", "biz-002", json.RawMessage(`{}`))
	require.NoError(t, err)

	// 提交任务
	err = taskMgr.Submit(task.ID)
	require.NoError(t, err)

	// 验证任务状态
	submittedTask, err := taskMgr.Get(task.ID)
	require.NoError(t, err)
	assert.Equal(t, types.TaskStateSubmitted, submittedTask.State)
	// 如果没有下一个节点,当前节点应该保持为 start
	assert.Equal(t, "start", submittedTask.CurrentNode)
}

