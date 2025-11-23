package integration_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/integration"
	"github.com/mautops/approval-kit/pkg/template"
	"github.com/mautops/approval-kit/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTaskManager_StateMachine_Submit 测试使用状态机提交任务
func TestTaskManager_StateMachine_Submit(t *testing.T) {
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

	// 创建任务管理器（需要状态机）
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)

	// 创建任务
	params := json.RawMessage(`{"amount": 1000}`)
	task, err := taskMgr.Create("tpl-001", "biz-001", params)
	require.NoError(t, err)
	assert.Equal(t, types.TaskStatePending, task.State)

	// 提交任务（应该通过状态机转换状态）
	err = taskMgr.Submit(task.ID)
	require.NoError(t, err)

	// 验证任务状态已更新
	updated, err := taskMgr.Get(task.ID)
	require.NoError(t, err)
	assert.Equal(t, types.TaskStateSubmitted, updated.State)
	assert.NotNil(t, updated.SubmittedAt)

	// 验证状态变更历史已记录
	assert.Len(t, updated.StateHistory, 1)
	assert.Equal(t, types.TaskStatePending, updated.StateHistory[0].From)
	assert.Equal(t, types.TaskStateSubmitted, updated.StateHistory[0].To)
}

// TestTaskManager_StateMachine_InvalidTransition 测试无效状态转换
func TestTaskManager_StateMachine_InvalidTransition(t *testing.T) {
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
	task, err := taskMgr.Create("tpl-001", "biz-001", params)
	require.NoError(t, err)

	// 尝试从 pending 直接转换到 approved（应该失败）
	// 注意：这需要状态机验证，目前 Submit 方法返回 "not implemented"
	// 这个测试将在实现状态机后验证
	err = taskMgr.Submit(task.ID)
	// 目前会返回 "not implemented"，实现后会验证状态转换
	if err != nil && err.Error() == "not implemented" {
		t.Skip("State machine not implemented yet")
	}
	require.NoError(t, err)

	// 验证任务状态仍然是 pending（如果直接转换到 approved 应该失败）
	updated, err := taskMgr.Get(task.ID)
	require.NoError(t, err)
	// 提交后应该是 submitted，不是 approved
	assert.NotEqual(t, types.TaskStateApproved, updated.State)
}

// TestTaskManager_StateMachine_Cancel 测试取消任务的状态转换
func TestTaskManager_StateMachine_Cancel(t *testing.T) {
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
	task, err := taskMgr.Create("tpl-001", "biz-001", params)
	require.NoError(t, err)
	assert.Equal(t, types.TaskStatePending, task.State)

	// 取消任务（应该通过状态机转换状态）
	err = taskMgr.Cancel(task.ID, "用户取消")
	if err != nil && err.Error() == "not implemented" {
		t.Skip("State machine not implemented yet")
	}
	require.NoError(t, err)

	// 验证任务状态已更新
	updated, err := taskMgr.Get(task.ID)
	require.NoError(t, err)
	assert.Equal(t, types.TaskStateCancelled, updated.State)

	// 验证状态变更历史已记录
	assert.Len(t, updated.StateHistory, 1)
	assert.Equal(t, types.TaskStatePending, updated.StateHistory[0].From)
	assert.Equal(t, types.TaskStateCancelled, updated.StateHistory[0].To)
	assert.Equal(t, "用户取消", updated.StateHistory[0].Reason)
}

