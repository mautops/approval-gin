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

// TestEventHandler_EventPersistence 测试事件持久化
func TestEventHandler_EventPersistence(t *testing.T) {
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
		Business: &event.BusinessInfo{
			ID: "biz-001",
		},
	}

	// 处理事件
	err := handler.Handle(evt)
	require.NoError(t, err)

	// 验证事件已保存到数据库
	var eventModel model.EventModel
	err = db.Where("task_id = ?", "task-001").First(&eventModel).Error
	require.NoError(t, err)

	assert.Equal(t, "task-001", eventModel.TaskID)
	assert.Equal(t, string(event.EventTypeTaskCreated), eventModel.Type)
	assert.Equal(t, "pending", eventModel.Status)
	assert.Equal(t, 0, eventModel.RetryCount)
	assert.NotEmpty(t, eventModel.Data)
}

// TestEventHandler_EventPersistenceMultiple 测试多个事件持久化
func TestEventHandler_EventPersistenceMultiple(t *testing.T) {
	db := setupTestDBForEventHandler(t)
	handler := integration.NewEventHandler(db, 1)

	events := []*event.Event{
		{
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
		},
		{
			ID:   "event-002",
			Type: event.EventTypeTaskSubmitted,
			Time: time.Now(),
			Task: &event.TaskInfo{
				ID:         "task-001",
				TemplateID: "tpl-001",
				BusinessID: "biz-001",
				State:      "submitted",
			},
			Node: &event.NodeInfo{
				ID:   "node-001",
				Name: "Start Node",
				Type: "start",
			},
		},
	}

	// 处理所有事件
	for _, evt := range events {
		err := handler.Handle(evt)
		require.NoError(t, err)
	}

	// 等待一段时间，确保事件保存完成
	time.Sleep(100 * time.Millisecond)

	// 验证所有事件已保存
	var eventModels []model.EventModel
	err := db.Where("task_id = ?", "task-001").Order("created_at ASC").Find(&eventModels).Error
	require.NoError(t, err)
	assert.Len(t, eventModels, 2)

	// 验证第一个事件
	assert.Equal(t, string(event.EventTypeTaskCreated), eventModels[0].Type)
	// 事件状态可能是 "pending"（未推送）或 "success"（已成功推送）
	assert.Contains(t, []string{"pending", "success"}, eventModels[0].Status, "event status should be pending or success")

	// 验证第二个事件
	assert.Equal(t, string(event.EventTypeTaskSubmitted), eventModels[1].Type)
	// 事件状态可能是 "pending"（未推送）或 "success"（已成功推送）
	assert.Contains(t, []string{"pending", "success"}, eventModels[1].Status, "event status should be pending or success")
}


