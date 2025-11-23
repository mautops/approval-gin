package integration_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/integration"
	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-kit/pkg/task"
	"github.com/mautops/approval-kit/pkg/template"
	"github.com/mautops/approval-kit/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDBForQuery 创建查询测试数据库
func setupTestDBForQuery(t *testing.T) *gorm.DB {
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

// TestTaskManager_Query_ByState 测试按状态查询任务
func TestTaskManager_Query_ByState(t *testing.T) {
	db := setupTestDBForQuery(t)

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

	// 创建多个任务
	task1, err := taskMgr.Create("tpl-001", "biz-001", json.RawMessage(`{}`))
	require.NoError(t, err)

	task2, err := taskMgr.Create("tpl-001", "biz-002", json.RawMessage(`{}`))
	require.NoError(t, err)

	// 提交 task1
	err = taskMgr.Submit(task1.ID)
	require.NoError(t, err)

	// 按状态查询
	filter := &task.TaskFilter{
		State: types.TaskStatePending,
	}
	tasks, err := taskMgr.Query(filter)
	require.NoError(t, err)
	
	// 应该只返回 task2 (pending 状态)
	assert.Len(t, tasks, 1)
	assert.Equal(t, task2.ID, tasks[0].ID)
	assert.Equal(t, types.TaskStatePending, tasks[0].State)

	// 查询 submitted 状态的任务
	filter.State = types.TaskStateSubmitted
	tasks, err = taskMgr.Query(filter)
	require.NoError(t, err)
	
	// 应该只返回 task1 (submitted 状态)
	assert.Len(t, tasks, 1)
	assert.Equal(t, task1.ID, tasks[0].ID)
	assert.Equal(t, types.TaskStateSubmitted, tasks[0].State)
}

// TestTaskManager_Query_ByTemplateID 测试按模板 ID 查询任务
func TestTaskManager_Query_ByTemplateID(t *testing.T) {
	db := setupTestDBForQuery(t)

	// 创建多个模板
	templateMgr := integration.NewTemplateManager(db)
	tpl1 := &template.Template{
		ID:          "tpl-001",
		Name:        "模板1",
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
	err := templateMgr.Create(tpl1)
	require.NoError(t, err)

	tpl2 := &template.Template{
		ID:          "tpl-002",
		Name:        "模板2",
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
	err = templateMgr.Create(tpl2)
	require.NoError(t, err)

	// 创建任务管理器
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)

	// 创建多个任务
	task1, err := taskMgr.Create("tpl-001", "biz-001", json.RawMessage(`{}`))
	require.NoError(t, err)

	_, err = taskMgr.Create("tpl-002", "biz-002", json.RawMessage(`{}`))
	require.NoError(t, err)

	// 按模板 ID 查询
	filter := &task.TaskFilter{
		TemplateID: "tpl-001",
	}
	tasks, err := taskMgr.Query(filter)
	require.NoError(t, err)
	
	// 应该只返回 task1
	assert.Len(t, tasks, 1)
	assert.Equal(t, task1.ID, tasks[0].ID)
	assert.Equal(t, "tpl-001", tasks[0].TemplateID)
}

// TestTaskManager_Query_ByBusinessID 测试按业务 ID 查询任务
func TestTaskManager_Query_ByBusinessID(t *testing.T) {
	db := setupTestDBForQuery(t)

	// 创建模板
	templateMgr := integration.NewTemplateManager(db)
	tpl := &template.Template{
		ID:          "tpl-001",
		Name:        "测试模板",
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

	// 创建多个任务
	task1, err := taskMgr.Create("tpl-001", "biz-001", json.RawMessage(`{}`))
	require.NoError(t, err)

	_, err = taskMgr.Create("tpl-001", "biz-002", json.RawMessage(`{}`))
	require.NoError(t, err)

	// 按业务 ID 查询
	filter := &task.TaskFilter{
		BusinessID: "biz-001",
	}
	tasks, err := taskMgr.Query(filter)
	require.NoError(t, err)
	
	// 应该只返回 task1
	assert.Len(t, tasks, 1)
	assert.Equal(t, task1.ID, tasks[0].ID)
	assert.Equal(t, "biz-001", tasks[0].BusinessID)
}

// TestTaskManager_Query_ByApprover 测试按审批人查询任务
func TestTaskManager_Query_ByApprover(t *testing.T) {
	db := setupTestDBForQuery(t)

	// 创建模板
	templateMgr := integration.NewTemplateManager(db)
	tpl := &template.Template{
		ID:          "tpl-001",
		Name:        "测试模板",
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
			{From: "start", To: "approval"},
		},
		Config: nil,
	}
	err := templateMgr.Create(tpl)
	require.NoError(t, err)

	// 创建任务管理器
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, nil)

	// 创建并提交任务
	task1, err := taskMgr.Create("tpl-001", "biz-001", json.RawMessage(`{}`))
	require.NoError(t, err)
	err = taskMgr.Submit(task1.ID)
	require.NoError(t, err)

	// 为 task1 添加审批人
	err = taskMgr.AddApprover(task1.ID, "approval", "user-001", "设置审批人")
	require.NoError(t, err)

	// 创建另一个任务
	task2, err := taskMgr.Create("tpl-001", "biz-002", json.RawMessage(`{}`))
	require.NoError(t, err)
	err = taskMgr.Submit(task2.ID)
	require.NoError(t, err)

	// 为 task2 添加不同的审批人
	err = taskMgr.AddApprover(task2.ID, "approval", "user-002", "设置审批人")
	require.NoError(t, err)
	
	// 按审批人查询
	filter := &task.TaskFilter{
		Approver: "user-001",
	}
	tasks, err := taskMgr.Query(filter)
	require.NoError(t, err)
	
	// 应该只返回 task1
	assert.Len(t, tasks, 1)
	assert.Equal(t, task1.ID, tasks[0].ID)
	// 验证 task2 不在结果中(查询 user-001 时)
	// task2 的审批人是 user-002,所以不应该出现在结果中
	for _, taskItem := range tasks {
		assert.NotEqual(t, task2.ID, taskItem.ID)
	}
}

// TestTaskManager_Query_Combined 测试组合条件查询
func TestTaskManager_Query_Combined(t *testing.T) {
	db := setupTestDBForQuery(t)

	// 创建模板
	templateMgr := integration.NewTemplateManager(db)
	tpl := &template.Template{
		ID:          "tpl-001",
		Name:        "测试模板",
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
	task1, err := taskMgr.Create("tpl-001", "biz-001", json.RawMessage(`{}`))
	require.NoError(t, err)

	// 提交任务
	err = taskMgr.Submit(task1.ID)
	require.NoError(t, err)

	// 组合条件查询: 模板 ID + 状态
	filter := &task.TaskFilter{
		TemplateID: "tpl-001",
		State:      types.TaskStateSubmitted,
	}
	tasks, err := taskMgr.Query(filter)
	require.NoError(t, err)
	
	// 应该返回 task1
	assert.Len(t, tasks, 1)
	assert.Equal(t, task1.ID, tasks[0].ID)
	assert.Equal(t, "tpl-001", tasks[0].TemplateID)
	assert.Equal(t, types.TaskStateSubmitted, tasks[0].State)
}

