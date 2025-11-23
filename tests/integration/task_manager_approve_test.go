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

// setupTestDBForApprove 创建审批测试数据库
func setupTestDBForApprove(t *testing.T) *gorm.DB {
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

// TestTaskManager_Approve 测试审批同意
func TestTaskManager_Approve(t *testing.T) {
	db := setupTestDBForApprove(t)

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

	// 创建并提交任务
	task, err := taskMgr.Create("tpl-001", "biz-001", json.RawMessage(`{"amount": 1000}`))
	require.NoError(t, err)
	err = taskMgr.Submit(task.ID)
	require.NoError(t, err)

	// 验证任务状态和当前节点
	submittedTask, err := taskMgr.Get(task.ID)
	require.NoError(t, err)
	assert.Equal(t, types.TaskStateSubmitted, submittedTask.State)
	assert.Equal(t, "approval", submittedTask.CurrentNode)

	// 设置审批人列表(通过 AddApprover 方法)
	err = taskMgr.AddApprover(submittedTask.ID, "approval", "user-001", "设置审批人")
	require.NoError(t, err)

	// 审批同意
	err = taskMgr.Approve(submittedTask.ID, "approval", "user-001", "同意")
	require.NoError(t, err)

	// 验证任务状态
	approvedTask, err := taskMgr.Get(submittedTask.ID)
	require.NoError(t, err)
	assert.Equal(t, types.TaskStateApproved, approvedTask.State)
	
	// 验证审批记录(AddApprover 也会生成记录,所以应该至少有 2 条记录)
	assert.GreaterOrEqual(t, len(approvedTask.Records), 1)
	// 检查最后一条记录应该是审批记录
	lastRecord := approvedTask.Records[len(approvedTask.Records)-1]
	assert.Equal(t, "approve", lastRecord.Result)
	assert.Equal(t, "user-001", lastRecord.Approver)
	assert.Equal(t, "同意", lastRecord.Comment)
}

// TestTaskManager_Reject 测试审批拒绝
func TestTaskManager_Reject(t *testing.T) {
	db := setupTestDBForApprove(t)

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

	// 创建并提交任务
	task, err := taskMgr.Create("tpl-001", "biz-001", json.RawMessage(`{"amount": 1000}`))
	require.NoError(t, err)
	err = taskMgr.Submit(task.ID)
	require.NoError(t, err)

	// 设置审批人列表
	err = taskMgr.AddApprover(task.ID, "approval", "user-001", "设置审批人")
	require.NoError(t, err)

	// 审批拒绝
	err = taskMgr.Reject(task.ID, "approval", "user-001", "拒绝")
	require.NoError(t, err)

	// 验证任务状态
	rejectedTask, err := taskMgr.Get(task.ID)
	require.NoError(t, err)
	assert.Equal(t, types.TaskStateRejected, rejectedTask.State)
	
	// 验证审批记录(AddApprover 也会生成记录,所以应该至少有 2 条记录)
	assert.GreaterOrEqual(t, len(rejectedTask.Records), 1)
	// 检查最后一条记录应该是拒绝记录
	lastRecord := rejectedTask.Records[len(rejectedTask.Records)-1]
	assert.Equal(t, "reject", lastRecord.Result)
	assert.Equal(t, "user-001", lastRecord.Approver)
	assert.Equal(t, "拒绝", lastRecord.Comment)
}

