package repository_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-gin/internal/repository"
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
	err = db.AutoMigrate(&model.TaskModel{})
	require.NoError(t, err)

	return db
}

// TestTaskRepository_Save 测试保存任务
func TestTaskRepository_Save(t *testing.T) {
	db := setupTestDBForTask(t)
	repo := repository.NewTaskRepository(db)

	task := &model.TaskModel{
		ID:             "task-001",
		TemplateID:     "tpl-001",
		TemplateVersion: 1,
		BusinessID:     "biz-001",
		State:          "pending",
		CurrentNode:    "",
		Data:           []byte(`{"id":"task-001"}`),
		CreatedAt:     time.Now(),
		UpdatedAt:      time.Now(),
		SubmittedAt:    nil,
	}

	err := repo.Save(task)
	assert.NoError(t, err)

	// 验证任务已保存
	var saved model.TaskModel
	err = db.Where("id = ?", "task-001").First(&saved).Error
	assert.NoError(t, err)
	assert.Equal(t, "task-001", saved.ID)
	assert.Equal(t, "tpl-001", saved.TemplateID)
}

// TestTaskRepository_FindByID 测试根据 ID 查找任务
func TestTaskRepository_FindByID(t *testing.T) {
	db := setupTestDBForTask(t)
	repo := repository.NewTaskRepository(db)

	// 先保存任务
	task := &model.TaskModel{
		ID:             "task-001",
		TemplateID:     "tpl-001",
		TemplateVersion: 1,
		BusinessID:     "biz-001",
		State:          "pending",
		CurrentNode:    "",
		Data:           []byte(`{"id":"task-001"}`),
		CreatedAt:     time.Now(),
		UpdatedAt:      time.Now(),
	}
	err := repo.Save(task)
	require.NoError(t, err)

	// 查找任务
	found, err := repo.FindByID("task-001")
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "task-001", found.ID)
	assert.Equal(t, "tpl-001", found.TemplateID)
}

// TestTaskRepository_FindByID_NotFound 测试查找不存在的任务
func TestTaskRepository_FindByID_NotFound(t *testing.T) {
	db := setupTestDBForTask(t)
	repo := repository.NewTaskRepository(db)

	// 查找不存在的任务
	found, err := repo.FindByID("task-999")
	assert.Error(t, err)
	assert.Nil(t, found)
}

// TestTaskRepository_FindAll 测试查找所有任务
func TestTaskRepository_FindAll(t *testing.T) {
	db := setupTestDBForTask(t)
	repo := repository.NewTaskRepository(db)

	// 保存多个任务
	for i := 1; i <= 3; i++ {
		taskID := fmt.Sprintf("task-%03d", i)
		task := &model.TaskModel{
			ID:             taskID,
			TemplateID:     "tpl-001",
			TemplateVersion: 1,
			BusinessID:     "biz-001",
			State:          "pending",
			CurrentNode:    "",
			Data:           []byte(fmt.Sprintf(`{"id":"%s"}`, taskID)),
			CreatedAt:     time.Now(),
			UpdatedAt:      time.Now(),
		}
		err := repo.Save(task)
		require.NoError(t, err)
	}

	// 查找所有任务
	tasks, err := repo.FindAll()
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(tasks), 3)
}

// TestTaskRepository_FindByFilter 测试根据过滤器查找任务
func TestTaskRepository_FindByFilter(t *testing.T) {
	db := setupTestDBForTask(t)
	repo := repository.NewTaskRepository(db)

	// 保存多个不同状态的任务
	states := []string{"pending", "approving", "approved"}
	for i, state := range states {
		taskID := fmt.Sprintf("task-%03d", i+1)
		task := &model.TaskModel{
			ID:             taskID,
			TemplateID:     "tpl-001",
			TemplateVersion: 1,
			BusinessID:     "biz-001",
			State:          state,
			CurrentNode:    "",
			Data:           []byte(fmt.Sprintf(`{"id":"%s"}`, taskID)),
			CreatedAt:     time.Now(),
			UpdatedAt:      time.Now(),
		}
		err := repo.Save(task)
		require.NoError(t, err)
	}

	// 按状态过滤
	filter := &repository.TaskFilter{
		State: stringPtr("pending"),
	}
	tasks, err := repo.FindByFilter(filter)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(tasks), 1)
	for _, task := range tasks {
		assert.Equal(t, "pending", task.State)
	}

	// 按模板 ID 过滤
	filter = &repository.TaskFilter{
		TemplateID: stringPtr("tpl-001"),
	}
	tasks, err = repo.FindByFilter(filter)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(tasks), 3)

	// 按业务 ID 过滤
	filter = &repository.TaskFilter{
		BusinessID: stringPtr("biz-001"),
	}
	tasks, err = repo.FindByFilter(filter)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(tasks), 3)
}

// stringPtr 返回字符串指针
func stringPtr(s string) *string {
	return &s
}

// TestTaskRepository_Save_Update 测试更新已存在的任务
func TestTaskRepository_Save_Update(t *testing.T) {
	db := setupTestDBForTask(t)
	repo := repository.NewTaskRepository(db)

	// 先保存任务
	task := &model.TaskModel{
		ID:             "task-001",
		TemplateID:     "tpl-001",
		TemplateVersion: 1,
		BusinessID:     "biz-001",
		State:          "pending",
		CurrentNode:    "",
		Data:           []byte(`{"id":"task-001"}`),
		CreatedAt:     time.Now(),
		UpdatedAt:      time.Now(),
	}
	err := repo.Save(task)
	require.NoError(t, err)

	// 更新任务状态
	task.State = "approving"
	task.CurrentNode = "node-001"
	err = repo.Save(task)
	assert.NoError(t, err)

	// 验证任务已更新
	found, err := repo.FindByID("task-001")
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "approving", found.State)
	assert.Equal(t, "node-001", found.CurrentNode)
}

// TestTaskRepository_FindByFilter_Combined 测试组合过滤条件
func TestTaskRepository_FindByFilter_Combined(t *testing.T) {
	db := setupTestDBForTask(t)
	repo := repository.NewTaskRepository(db)

	// 保存多个任务
	tasks := []struct {
		id         string
		templateID string
		businessID string
		state      string
	}{
		{"task-001", "tpl-001", "biz-001", "pending"},
		{"task-002", "tpl-001", "biz-002", "pending"},
		{"task-003", "tpl-002", "biz-001", "approving"},
		{"task-004", "tpl-001", "biz-001", "approved"},
	}

	for _, tsk := range tasks {
		task := &model.TaskModel{
			ID:             tsk.id,
			TemplateID:     tsk.templateID,
			TemplateVersion: 1,
			BusinessID:     tsk.businessID,
			State:          tsk.state,
			CurrentNode:    "",
			Data:           []byte(fmt.Sprintf(`{"id":"%s"}`, tsk.id)),
			CreatedAt:     time.Now(),
			UpdatedAt:      time.Now(),
		}
		err := repo.Save(task)
		require.NoError(t, err)
	}

	// 组合过滤: 模板 ID + 状态
	filter := &repository.TaskFilter{
		TemplateID: stringPtr("tpl-001"),
		State:      stringPtr("pending"),
	}
	found, err := repo.FindByFilter(filter)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(found), 2)
	for _, task := range found {
		assert.Equal(t, "tpl-001", task.TemplateID)
		assert.Equal(t, "pending", task.State)
	}

	// 组合过滤: 业务 ID + 状态
	filter = &repository.TaskFilter{
		BusinessID: stringPtr("biz-001"),
		State:      stringPtr("pending"),
	}
	found, err = repo.FindByFilter(filter)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(found), 1)
	for _, task := range found {
		assert.Equal(t, "biz-001", task.BusinessID)
		assert.Equal(t, "pending", task.State)
	}
}

// TestTaskRepository_FindByFilter_Empty 测试空过滤器
func TestTaskRepository_FindByFilter_Empty(t *testing.T) {
	db := setupTestDBForTask(t)
	repo := repository.NewTaskRepository(db)

	// 保存一些任务
	for i := 1; i <= 3; i++ {
		taskID := fmt.Sprintf("task-%03d", i)
		task := &model.TaskModel{
			ID:             taskID,
			TemplateID:     "tpl-001",
			TemplateVersion: 1,
			BusinessID:     "biz-001",
			State:          "pending",
			CurrentNode:    "",
			Data:           []byte(fmt.Sprintf(`{"id":"%s"}`, taskID)),
			CreatedAt:     time.Now(),
			UpdatedAt:      time.Now(),
		}
		err := repo.Save(task)
		require.NoError(t, err)
	}

	// 空过滤器应该返回所有任务
	filter := &repository.TaskFilter{}
	tasks, err := repo.FindByFilter(filter)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(tasks), 3)
}

// TestTaskRepository_FindAll_Empty 测试查找空列表
func TestTaskRepository_FindAll_Empty(t *testing.T) {
	db := setupTestDBForTask(t)
	repo := repository.NewTaskRepository(db)

	// 查找所有任务(应该返回空列表)
	tasks, err := repo.FindAll()
	assert.NoError(t, err)
	assert.Empty(t, tasks)
}

// TestTaskRepository_Save_WithSubmittedAt 测试保存带提交时间的任务
func TestTaskRepository_Save_WithSubmittedAt(t *testing.T) {
	db := setupTestDBForTask(t)
	repo := repository.NewTaskRepository(db)

	now := time.Now()
	task := &model.TaskModel{
		ID:             "task-001",
		TemplateID:     "tpl-001",
		TemplateVersion: 1,
		BusinessID:     "biz-001",
		State:          "submitted",
		CurrentNode:    "",
		Data:           []byte(`{"id":"task-001"}`),
		CreatedAt:     now,
		UpdatedAt:      now,
		SubmittedAt:    &now,
	}

	err := repo.Save(task)
	assert.NoError(t, err)

	// 验证任务已保存
	found, err := repo.FindByID("task-001")
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.NotNil(t, found.SubmittedAt)
}

