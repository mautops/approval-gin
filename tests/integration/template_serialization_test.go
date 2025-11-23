package integration_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mautops/approval-kit/pkg/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTemplate_JSONSerialization 测试模板的 JSON 序列化
func TestTemplate_JSONSerialization(t *testing.T) {
	tpl := &template.Template{
		ID:          "tpl-001",
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Version:     1,
		CreatedAt:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
		Nodes: map[string]*template.Node{
			"node-1": {
				ID:     "node-1",
				Name:   "开始节点",
				Type:   "start",
				Order:  1,
				Config: nil,
			},
			"node-2": {
				ID:     "node-2",
				Name:   "审批节点",
				Type:   "approval",
				Order:  2,
				Config: nil, // NodeConfig 是接口,测试中暂时使用 nil
			},
		},
		Edges: []*template.Edge{
			{
				From:      "node-1",
				To:        "node-2",
				Condition: "",
			},
		},
		Config: &template.TemplateConfig{
			Webhooks: []*template.WebhookConfig{
				{
					URL:    "https://example.com/webhook",
					Method: "POST",
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
					Auth: &template.AuthConfig{
						Type:  "token",
						Token: "secret-token",
						Key:   "",
					},
				},
			},
		},
	}

	// 序列化
	data, err := json.Marshal(tpl)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// 反序列化
	var restored template.Template
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	// 验证数据完整性
	assert.Equal(t, tpl.ID, restored.ID)
	assert.Equal(t, tpl.Name, restored.Name)
	assert.Equal(t, tpl.Description, restored.Description)
	assert.Equal(t, tpl.Version, restored.Version)
	assert.Equal(t, tpl.CreatedAt, restored.CreatedAt)
	assert.Equal(t, tpl.UpdatedAt, restored.UpdatedAt)

	// 验证节点
	assert.Equal(t, len(tpl.Nodes), len(restored.Nodes))
	assert.Equal(t, tpl.Nodes["node-1"].ID, restored.Nodes["node-1"].ID)
	assert.Equal(t, tpl.Nodes["node-1"].Name, restored.Nodes["node-1"].Name)
	assert.Equal(t, tpl.Nodes["node-2"].ID, restored.Nodes["node-2"].ID)

	// 验证边
	assert.Equal(t, len(tpl.Edges), len(restored.Edges))
	assert.Equal(t, tpl.Edges[0].From, restored.Edges[0].From)
	assert.Equal(t, tpl.Edges[0].To, restored.Edges[0].To)

	// 验证配置
	assert.NotNil(t, restored.Config)
	assert.Equal(t, len(tpl.Config.Webhooks), len(restored.Config.Webhooks))
	assert.Equal(t, tpl.Config.Webhooks[0].URL, restored.Config.Webhooks[0].URL)
	assert.Equal(t, tpl.Config.Webhooks[0].Method, restored.Config.Webhooks[0].Method)
	assert.NotNil(t, restored.Config.Webhooks[0].Auth)
	assert.Equal(t, tpl.Config.Webhooks[0].Auth.Type, restored.Config.Webhooks[0].Auth.Type)
}

// TestTemplate_JSONSerialization_EmptyFields 测试空字段的序列化
func TestTemplate_JSONSerialization_EmptyFields(t *testing.T) {
	tpl := &template.Template{
		ID:          "tpl-002",
		Name:        "空字段模板",
		Description: "",
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Nodes:       nil,
		Edges:       nil,
		Config:      nil,
	}

	// 序列化
	data, err := json.Marshal(tpl)
	require.NoError(t, err)

	// 反序列化
	var restored template.Template
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	// 验证空字段
	assert.Equal(t, tpl.ID, restored.ID)
	assert.Equal(t, tpl.Name, restored.Name)
	assert.Equal(t, "", restored.Description)
	assert.Nil(t, restored.Nodes)
	assert.Nil(t, restored.Edges)
	assert.Nil(t, restored.Config)
}

// TestTemplate_JSONSerialization_ComplexConfig 测试复杂配置的序列化
func TestTemplate_JSONSerialization_ComplexConfig(t *testing.T) {
	tpl := &template.Template{
		ID:          "tpl-003",
		Name:        "复杂配置模板",
		Description: "包含复杂配置的模板",
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Nodes: map[string]*template.Node{
			"node-1": {
				ID:     "node-1",
				Name:   "条件节点",
				Type:   "condition",
				Order:  1,
				Config: nil, // NodeConfig 是接口,测试中暂时使用 nil
			},
		},
		Edges: []*template.Edge{
			{
				From:      "node-1",
				To:        "node-2",
				Condition: "amount > 1000",
			},
		},
		Config: nil,
	}

	// 序列化
	data, err := json.Marshal(tpl)
	require.NoError(t, err)

	// 反序列化
	var restored template.Template
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	// 验证复杂配置
	assert.NotNil(t, restored.Nodes["node-1"])
	// NodeConfig 是接口,序列化后可能为 nil 或具体实现
	// 这里只验证节点存在即可
}

// TestTemplate_JSONSerialization_RoundTrip 测试往返序列化
func TestTemplate_JSONSerialization_RoundTrip(t *testing.T) {
	original := &template.Template{
		ID:          "tpl-004",
		Name:        "往返测试模板",
		Description: "测试多次序列化和反序列化",
		Version:     2,
		CreatedAt:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC),
		Nodes: map[string]*template.Node{
			"start": {
				ID:     "start",
				Name:   "开始",
				Type:   "start",
				Order:  0,
				Config: nil,
			},
			"end": {
				ID:     "end",
				Name:   "结束",
				Type:   "end",
				Order:  100,
				Config: nil,
			},
		},
		Edges: []*template.Edge{
			{From: "start", To: "end", Condition: ""},
		},
		Config: &template.TemplateConfig{
			Webhooks: []*template.WebhookConfig{
				{
					URL:     "https://example.com/webhook",
					Method:  "POST",
					Headers: map[string]string{"X-Custom": "value"},
					Auth:    nil,
				},
			},
		},
	}

	// 多次往返序列化
	var current = original
	for i := 0; i < 3; i++ {
		data, err := json.Marshal(current)
		require.NoError(t, err)

		var restored template.Template
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)

		// 验证关键字段
		assert.Equal(t, original.ID, restored.ID)
		assert.Equal(t, original.Name, restored.Name)
		assert.Equal(t, original.Version, restored.Version)
		assert.Equal(t, len(original.Nodes), len(restored.Nodes))
		assert.Equal(t, len(original.Edges), len(restored.Edges))

		current = &restored
	}
}

