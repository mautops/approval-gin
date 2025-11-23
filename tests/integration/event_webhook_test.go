package integration_test

import (
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/integration"
	"github.com/mautops/approval-kit/pkg/event"
	"github.com/mautops/approval-kit/pkg/template"
	"github.com/stretchr/testify/require"
)

// TestEventHandler_WebhookConfig 测试 Webhook 配置
func TestEventHandler_WebhookConfig(t *testing.T) {
	db := setupTestDBForEventHandler(t)
	handler := integration.NewEventHandler(db, 1)

	// 创建带 Webhook 配置的模板
	template := &template.Template{
		ID:          "tpl-001",
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config: &template.TemplateConfig{
			Webhooks: []*template.WebhookConfig{
				{
					URL:    "http://example.com/webhook",
					Method: "POST",
					Headers: map[string]string{
						"X-Custom-Header": "value",
					},
					Auth: &template.AuthConfig{
						Type:  "bearer",
						Token: "test-token",
					},
				},
			},
		},
	}

	// 保存模板
	templateMgr := integration.NewTemplateManager(db)
	err := templateMgr.Create(template)
	require.NoError(t, err)

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
	err = handler.Handle(evt)
	require.NoError(t, err)

	// 等待一段时间，确保异步处理完成
	time.Sleep(500 * time.Millisecond)
}

// TestEventHandler_NoWebhookConfig 测试没有 Webhook 配置的情况
func TestEventHandler_NoWebhookConfig(t *testing.T) {
	db := setupTestDBForEventHandler(t)
	handler := integration.NewEventHandler(db, 1)

	// 创建不带 Webhook 配置的模板
	template := &template.Template{
		ID:          "tpl-002",
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	}

	// 保存模板
	templateMgr := integration.NewTemplateManager(db)
	err := templateMgr.Create(template)
	require.NoError(t, err)

	// 创建测试事件
	evt := &event.Event{
		ID:   "event-002",
		Type: event.EventTypeTaskCreated,
		Time: time.Now(),
		Task: &event.TaskInfo{
			ID:         "task-002",
			TemplateID: "tpl-002",
			BusinessID: "biz-002",
			State:      "pending",
		},
		Node: &event.NodeInfo{
			ID:   "node-001",
			Name: "Start Node",
			Type: "start",
		},
	}

	// 处理事件（应该成功，因为没有 Webhook 配置）
	err = handler.Handle(evt)
	require.NoError(t, err)

	// 等待一段时间，确保异步处理完成
	time.Sleep(500 * time.Millisecond)
}

