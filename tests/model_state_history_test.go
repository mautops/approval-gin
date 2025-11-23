package tests

import (
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/model"
	"github.com/stretchr/testify/assert"
)

// TestStateHistoryModel 测试状态历史数据模型
func TestStateHistoryModel(t *testing.T) {
	shm := &model.StateHistoryModel{
		ID:        "history-001",
		TaskID:    "task-001",
		FromState: "pending",
		ToState:   "submitted",
		Reason:    "Task submitted",
		Operator:  "user-001",
		CreatedAt: time.Now(),
	}
	
	// 验证模型字段
	assert.Equal(t, "history-001", shm.ID)
	assert.Equal(t, "task-001", shm.TaskID)
	assert.Equal(t, "pending", shm.FromState)
	assert.Equal(t, "submitted", shm.ToState)
}

// TestStateHistoryModelTableName 测试表名
func TestStateHistoryModelTableName(t *testing.T) {
	shm := model.StateHistoryModel{}
	assert.Equal(t, "state_history", shm.TableName())
}

// TestStateHistoryModelValidation 测试模型验证
func TestStateHistoryModelValidation(t *testing.T) {
	shm := &model.StateHistoryModel{
		ID:       "history-001",
		TaskID:   "task-001",
		ToState:  "submitted",
		Operator: "user-001",
	}
	
	err := shm.Validate()
	assert.NoError(t, err)
	
	// 测试无效模型 - ID 为空
	shmInvalidID := &model.StateHistoryModel{
		ID:       "",
		TaskID:   "task-001",
		ToState:  "submitted",
		Operator: "user-001",
	}
	err = shmInvalidID.Validate()
	assert.Error(t, err)
	
	// 测试无效模型 - TaskID 为空
	shmInvalidTaskID := &model.StateHistoryModel{
		ID:       "history-002",
		TaskID:   "",
		ToState:  "submitted",
		Operator: "user-001",
	}
	err = shmInvalidTaskID.Validate()
	assert.Error(t, err)
	
	// 测试无效模型 - ToState 为空
	shmInvalidToState := &model.StateHistoryModel{
		ID:       "history-003",
		TaskID:   "task-001",
		ToState:  "",
		Operator: "user-001",
	}
	err = shmInvalidToState.Validate()
	assert.Error(t, err)
}

