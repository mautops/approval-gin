package repository_test

import (
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-gin/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDBForEvent 创建事件测试数据库
func setupTestDBForEvent(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 迁移数据库
	err = db.AutoMigrate(&model.EventModel{})
	require.NoError(t, err)

	return db
}

// TestEventRepository_Save 测试保存事件
func TestEventRepository_Save(t *testing.T) {
	db := setupTestDBForEvent(t)
	repo := repository.NewEventRepository(db)

	event := &model.EventModel{
		ID:        "event-001",
		TaskID:    "task-001",
		Type:      "task_created",
		Data:      []byte(`{"task_id":"task-001"}`),
		Status:    "pending",
		RetryCount: 0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Save(event)
	assert.NoError(t, err)

	// 验证事件已保存
	var saved model.EventModel
	err = db.Where("id = ?", "event-001").First(&saved).Error
	assert.NoError(t, err)
	assert.Equal(t, "event-001", saved.ID)
	assert.Equal(t, "task-001", saved.TaskID)
	assert.Equal(t, "task_created", saved.Type)
	assert.Equal(t, "pending", saved.Status)
}

// TestEventRepository_FindByTaskID 测试根据任务 ID 查找事件
func TestEventRepository_FindByTaskID(t *testing.T) {
	db := setupTestDBForEvent(t)
	repo := repository.NewEventRepository(db)

	// 先保存多个事件
	eventTypes := []string{"task_created", "task_submitted", "node_activated"}
	for i, eventType := range eventTypes {
		event := &model.EventModel{
			ID:         "event-00" + string(rune(i+'1')),
			TaskID:     "task-001",
			Type:       eventType,
			Data:       []byte(`{"task_id":"task-001"}`),
			Status:     "pending",
			RetryCount: 0,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		err := repo.Save(event)
		require.NoError(t, err)
	}

	// 查找任务的所有事件
	events, err := repo.FindByTaskID("task-001")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(events))
	for _, event := range events {
		assert.Equal(t, "task-001", event.TaskID)
	}
}

// TestEventRepository_FindPending 测试查找待处理的事件
func TestEventRepository_FindPending(t *testing.T) {
	db := setupTestDBForEvent(t)
	repo := repository.NewEventRepository(db)

	// 保存多个不同状态的事件
	statuses := []string{"pending", "pending", "success", "failed"}
	for i, status := range statuses {
		event := &model.EventModel{
			ID:         "event-00" + string(rune(i+'1')),
			TaskID:     "task-001",
			Type:       "task_created",
			Data:       []byte(`{"task_id":"task-001"}`),
			Status:     status,
			RetryCount: 0,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		err := repo.Save(event)
		require.NoError(t, err)
	}

	// 查找待处理的事件
	events, err := repo.FindPending()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(events))
	for _, event := range events {
		assert.Equal(t, "pending", event.Status)
	}
}


