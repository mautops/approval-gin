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

// setupTestDBForHistory 创建状态历史测试数据库
func setupTestDBForHistory(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 迁移数据库
	err = db.AutoMigrate(&model.StateHistoryModel{})
	require.NoError(t, err)

	return db
}

// TestStateHistoryRepository_Save 测试保存状态历史
func TestStateHistoryRepository_Save(t *testing.T) {
	db := setupTestDBForHistory(t)
	repo := repository.NewStateHistoryRepository(db)

	history := &model.StateHistoryModel{
		ID:        "history-001",
		TaskID:    "task-001",
		FromState: "pending",
		ToState:   "submitted",
		Reason:    "任务提交",
		Operator:  "user-001",
		CreatedAt: time.Now(),
	}

	err := repo.Save(history)
	assert.NoError(t, err)

	// 验证历史已保存
	var saved model.StateHistoryModel
	err = db.Where("id = ?", "history-001").First(&saved).Error
	assert.NoError(t, err)
	assert.Equal(t, "history-001", saved.ID)
	assert.Equal(t, "task-001", saved.TaskID)
	assert.Equal(t, "pending", saved.FromState)
	assert.Equal(t, "submitted", saved.ToState)
}

// TestStateHistoryRepository_FindByTaskID 测试根据任务 ID 查找状态历史
func TestStateHistoryRepository_FindByTaskID(t *testing.T) {
	db := setupTestDBForHistory(t)
	repo := repository.NewStateHistoryRepository(db)

	// 先保存多个历史记录
	states := []struct {
		from string
		to   string
	}{
		{"pending", "submitted"},
		{"submitted", "approving"},
		{"approving", "approved"},
	}

	for i, state := range states {
		history := &model.StateHistoryModel{
			ID:        "history-00" + string(rune(i+'1')),
			TaskID:    "task-001",
			FromState: state.from,
			ToState:   state.to,
			Reason:    "状态转换",
			Operator:  "user-001",
			CreatedAt: time.Now().Add(time.Duration(i) * time.Minute),
		}
		err := repo.Save(history)
		require.NoError(t, err)
	}

	// 查找任务的所有历史记录
	histories, err := repo.FindByTaskID("task-001")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(histories))
	
	// 验证历史记录按时间排序
	for i, history := range histories {
		assert.Equal(t, "task-001", history.TaskID)
		if i > 0 {
			assert.True(t, history.CreatedAt.After(histories[i-1].CreatedAt) || history.CreatedAt.Equal(histories[i-1].CreatedAt))
		}
	}
}


