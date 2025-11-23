package integration_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/integration"
	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-kit/pkg/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEventHandler_AsyncPush 测试事件异步推送
func TestEventHandler_AsyncPush(t *testing.T) {
	db := setupTestDBForEventHandler(t)
	
	// 创建测试模板（事件处理器需要模板来获取 Webhook 配置）
	templateData := map[string]interface{}{
		"id":   "tpl-001",
		"name": "Test Template",
		"nodes": map[string]interface{}{
			"start": map[string]interface{}{
				"id":   "start",
				"type": "start",
			},
		},
		"edges": []interface{}{},
	}
	data, _ := json.Marshal(templateData)
	template := &model.TemplateModel{
		ID:          "tpl-001",
		Name:        "Test Template",
		Description: "Test",
		Version:     1,
		Data:        data,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   "test",
	}
	err := db.Create(template).Error
	require.NoError(t, err)

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

	// 处理事件（应该立即返回，不阻塞）
	start := time.Now()
	err = handler.Handle(evt)
	duration := time.Since(start)

	require.NoError(t, err)
	// 验证处理是异步的（应该很快返回）
	assert.Less(t, duration, 100*time.Millisecond, "Handle should return quickly (async)")

	// 等待一段时间，确保异步处理完成
	time.Sleep(200 * time.Millisecond)
}

// TestEventHandler_AsyncPushMultiple 测试多个事件异步推送
func TestEventHandler_AsyncPushMultiple(t *testing.T) {
	db := setupTestDBForEventHandler(t)
	
	// setupTestDBForEventHandler 应该已经创建了所有必需的表，包括 events 表
	// 创建测试模板（事件处理器需要模板来获取 Webhook 配置）
	templateData := map[string]interface{}{
		"id":   "tpl-001",
		"name": "Test Template",
		"nodes": map[string]interface{}{
			"start": map[string]interface{}{
				"id":   "start",
				"type": "start",
			},
		},
		"edges": []interface{}{},
	}
	data, _ := json.Marshal(templateData)
	template := &model.TemplateModel{
		ID:          "tpl-001",
		Name:        "Test Template",
		Description: "Test",
		Version:     1,
		Data:        data,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   "test",
	}
	err := db.Create(template).Error
	require.NoError(t, err)

	handler := integration.NewEventHandler(db, 2) // 使用 2 个 worker

	// 创建多个测试事件
	events := make([]*event.Event, 5)
	for i := 0; i < 5; i++ {
		events[i] = &event.Event{
			ID:   fmt.Sprintf("event-%d", i),
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
	}

	// 处理所有事件（应该立即返回，不阻塞）
	start := time.Now()
	for _, evt := range events {
		err := handler.Handle(evt)
		require.NoError(t, err)
	}
	duration := time.Since(start)

	// 验证处理是异步的（应该很快返回）
	assert.Less(t, duration, 500*time.Millisecond, "Handle should return quickly (async)")

	// 等待一段时间，确保异步处理完成
	time.Sleep(500 * time.Millisecond)
}

