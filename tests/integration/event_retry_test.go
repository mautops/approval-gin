package integration_test

import (
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/integration"
	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-kit/pkg/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEventHandler_RetryMechanism 测试事件重试机制
func TestEventHandler_RetryMechanism(t *testing.T) {
	db := setupTestDBForEventHandler(t)
	handler := integration.NewEventHandler(db, 1)

	// 创建测试事件
	evt := &event.Event{
		ID:   "event-001",
		Type: event.EventTypeTaskCreated,
		Time: time.Now(),
		Task: &event.TaskInfo{
			ID:         "task-001",
			TemplateID: "tpl-001",
			BusinessID: "biz-001",
			State:      "pending",
		},
		Node: &event.NodeInfo{
			ID:   "node-001",
			Name: "Start Node",
			Type: "start",
		},
	}

	// 处理事件
	err := handler.Handle(evt)
	require.NoError(t, err)

	// 验证事件已保存，状态为 pending
	var eventModel model.EventModel
	err = db.Where("task_id = ?", "task-001").First(&eventModel).Error
	require.NoError(t, err)
	assert.Equal(t, "pending", eventModel.Status)
	assert.Equal(t, 0, eventModel.RetryCount)

	// 等待一段时间，确保异步处理完成
	time.Sleep(500 * time.Millisecond)
}

// TestEventHandler_RetryCount 测试重试计数
func TestEventHandler_RetryCount(t *testing.T) {
	db := setupTestDBForEventHandler(t)
	handler := integration.NewEventHandler(db, 1)

	// 创建测试事件
	evt := &event.Event{
		ID:   "event-001",
		Type: event.EventTypeTaskCreated,
		Time: time.Now(),
		Task: &event.TaskInfo{
			ID:         "task-001",
			TemplateID: "tpl-001",
			BusinessID: "biz-001",
			State:      "pending",
		},
		Node: &event.NodeInfo{
			ID:   "node-001",
			Name: "Start Node",
			Type: "start",
		},
	}

	// 处理事件
	err := handler.Handle(evt)
	require.NoError(t, err)

	// 验证事件已保存，重试计数为 0
	var eventModel model.EventModel
	err = db.Where("task_id = ?", "task-001").First(&eventModel).Error
	require.NoError(t, err)
	assert.Equal(t, 0, eventModel.RetryCount)

	// 等待一段时间，确保异步处理完成
	time.Sleep(500 * time.Millisecond)
}


