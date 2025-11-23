package tests

import (
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/model"
	"github.com/stretchr/testify/assert"
)

// TestEventModel 测试事件数据模型
func TestEventModel(t *testing.T) {
	em := &model.EventModel{
		ID:        "event-001",
		TaskID:    "task-001",
		Type:      "task_created",
		Data:      []byte(`{"type":"task_created"}`),
		Status:    "pending",
		RetryCount: 0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	// 验证模型字段
	assert.Equal(t, "event-001", em.ID)
	assert.Equal(t, "task-001", em.TaskID)
	assert.Equal(t, "task_created", em.Type)
	assert.Equal(t, "pending", em.Status)
}

// TestEventModelTableName 测试表名
func TestEventModelTableName(t *testing.T) {
	em := model.EventModel{}
	assert.Equal(t, "events", em.TableName())
}

// TestEventModelValidation 测试模型验证
func TestEventModelValidation(t *testing.T) {
	em := &model.EventModel{
		ID:     "event-001",
		TaskID: "task-001",
		Type:   "task_created",
		Data:   []byte(`{}`),
		Status: "pending",
	}
	
	err := em.Validate()
	assert.NoError(t, err)
	
	// 测试无效模型 - ID 为空
	emInvalidID := &model.EventModel{
		ID:     "",
		TaskID: "task-001",
		Type:   "task_created",
		Data:   []byte(`{}`),
		Status: "pending",
	}
	err = emInvalidID.Validate()
	assert.Error(t, err)
	
	// 测试无效模型 - TaskID 为空
	emInvalidTaskID := &model.EventModel{
		ID:     "event-002",
		TaskID: "",
		Type:   "task_created",
		Data:   []byte(`{}`),
		Status: "pending",
	}
	err = emInvalidTaskID.Validate()
	assert.Error(t, err)
	
	// 测试无效模型 - Type 为空
	emInvalidType := &model.EventModel{
		ID:     "event-003",
		TaskID: "task-001",
		Type:   "",
		Data:   []byte(`{}`),
		Status: "pending",
	}
	err = emInvalidType.Validate()
	assert.Error(t, err)
}

