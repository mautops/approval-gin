package integration_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mautops/approval-kit/pkg/task"
	"github.com/mautops/approval-kit/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTask_JSONSerialization 测试任务的 JSON 序列化
func TestTask_JSONSerialization(t *testing.T) {
	now := time.Now()
	submittedAt := now.Add(time.Hour)
	pausedAt := now.Add(2 * time.Hour)

	tsk := &task.Task{
		ID:              "task-001",
		TemplateID:      "tpl-001",
		TemplateVersion: 1,
		BusinessID:      "biz-001",
		Params:          json.RawMessage(`{"amount": 1000, "department": "IT"}`),
		State:           types.TaskStateApproving,
		CurrentNode:     "node-approval-1",
		PausedAt:        &pausedAt,
		PausedState:     types.TaskStateApproving,
		CreatedAt:       now,
		UpdatedAt:       now,
		SubmittedAt:     &submittedAt,
		NodeOutputs: map[string]json.RawMessage{
			"node-1": json.RawMessage(`{"result": "success"}`),
		},
		Approvers: map[string][]string{
			"node-approval-1": {"user-1", "user-2"},
		},
		Approvals: map[string]map[string]*task.Approval{
			"node-approval-1": {
				"user-1": {
					Result:    "approve",
					Comment:   "同意",
					CreatedAt: now,
				},
			},
		},
		CompletedNodes: []string{"node-1"},
		Records: []*task.Record{
			{
				ID:          "record-001",
				TaskID:      "task-001",
				NodeID:      "node-approval-1",
				Approver:    "user-1",
				Result:      "approve",
				Comment:     "同意",
				CreatedAt:   now,
				Attachments: []string{"file-1.pdf"},
			},
		},
		StateHistory: []*task.StateChange{
			{
				From:   types.TaskStatePending,
				To:     types.TaskStateSubmitted,
				Reason: "任务提交",
				Time:   now,
			},
			{
				From:   types.TaskStateSubmitted,
				To:     types.TaskStateApproving,
				Reason: "进入审批流程",
				Time:   now.Add(30 * time.Minute),
			},
		},
	}

	// 序列化
	data, err := json.Marshal(tsk)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// 反序列化
	var restored task.Task
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	// 验证基本信息
	assert.Equal(t, tsk.ID, restored.ID)
	assert.Equal(t, tsk.TemplateID, restored.TemplateID)
	assert.Equal(t, tsk.TemplateVersion, restored.TemplateVersion)
	assert.Equal(t, tsk.BusinessID, restored.BusinessID)
	assert.Equal(t, tsk.State, restored.State)
	assert.Equal(t, tsk.CurrentNode, restored.CurrentNode)

	// 验证时间字段
	assert.Equal(t, tsk.CreatedAt.Unix(), restored.CreatedAt.Unix())
	assert.Equal(t, tsk.UpdatedAt.Unix(), restored.UpdatedAt.Unix())
	assert.NotNil(t, restored.SubmittedAt)
	assert.Equal(t, tsk.SubmittedAt.Unix(), restored.SubmittedAt.Unix())
	assert.NotNil(t, restored.PausedAt)
	assert.Equal(t, tsk.PausedAt.Unix(), restored.PausedAt.Unix())
	assert.Equal(t, tsk.PausedState, restored.PausedState)

	// 验证参数(JSON 格式可能不同,比较解析后的内容)
	var originalParams, restoredParams map[string]interface{}
	json.Unmarshal(tsk.Params, &originalParams)
	json.Unmarshal(restored.Params, &restoredParams)
	assert.Equal(t, originalParams, restoredParams)

	// 验证节点输出(JSON 格式可能不同,比较解析后的内容)
	assert.Equal(t, len(tsk.NodeOutputs), len(restored.NodeOutputs))
	var originalOutput, restoredOutput map[string]interface{}
	json.Unmarshal(tsk.NodeOutputs["node-1"], &originalOutput)
	json.Unmarshal(restored.NodeOutputs["node-1"], &restoredOutput)
	assert.Equal(t, originalOutput, restoredOutput)

	// 验证审批人
	assert.Equal(t, len(tsk.Approvers), len(restored.Approvers))
	assert.Equal(t, tsk.Approvers["node-approval-1"], restored.Approvers["node-approval-1"])

	// 验证审批结果
	assert.Equal(t, len(tsk.Approvals), len(restored.Approvals))
	assert.NotNil(t, restored.Approvals["node-approval-1"])
	assert.NotNil(t, restored.Approvals["node-approval-1"]["user-1"])
	assert.Equal(t, tsk.Approvals["node-approval-1"]["user-1"].Result, restored.Approvals["node-approval-1"]["user-1"].Result)

	// 验证已完成节点
	assert.Equal(t, tsk.CompletedNodes, restored.CompletedNodes)

	// 验证审批记录
	assert.Equal(t, len(tsk.Records), len(restored.Records))
	assert.Equal(t, tsk.Records[0].ID, restored.Records[0].ID)
	assert.Equal(t, tsk.Records[0].Result, restored.Records[0].Result)

	// 验证状态历史
	assert.Equal(t, len(tsk.StateHistory), len(restored.StateHistory))
	assert.Equal(t, tsk.StateHistory[0].From, restored.StateHistory[0].From)
	assert.Equal(t, tsk.StateHistory[0].To, restored.StateHistory[0].To)
}

// TestTask_JSONSerialization_EmptyFields 测试空字段的序列化
func TestTask_JSONSerialization_EmptyFields(t *testing.T) {
	tsk := &task.Task{
		ID:              "task-002",
		TemplateID:      "tpl-001",
		TemplateVersion: 1,
		BusinessID:      "biz-001",
		Params:          nil,
		State:           types.TaskStatePending,
		CurrentNode:     "",
		PausedAt:        nil,
		PausedState:     "",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		SubmittedAt:     nil,
		NodeOutputs:     nil,
		Approvers:       nil,
		Approvals:       nil,
		CompletedNodes:  nil,
		Records:         nil,
		StateHistory:    nil,
	}

	// 序列化
	data, err := json.Marshal(tsk)
	require.NoError(t, err)

	// 反序列化
	var restored task.Task
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	// 验证空字段
	assert.Equal(t, tsk.ID, restored.ID)
	// json.RawMessage 为 nil 时序列化后会变成 "null",反序列化后不为 nil
	// 但内容应该是空的或 null
	if tsk.Params == nil {
		assert.True(t, len(restored.Params) == 0 || string(restored.Params) == "null")
	}
	assert.Equal(t, "", restored.CurrentNode)
	assert.Nil(t, restored.SubmittedAt)
	assert.Nil(t, restored.PausedAt)
	assert.Nil(t, restored.NodeOutputs)
	assert.Nil(t, restored.Approvers)
	assert.Nil(t, restored.Approvals)
	assert.Nil(t, restored.CompletedNodes)
	assert.Nil(t, restored.Records)
	assert.Nil(t, restored.StateHistory)
}

// TestTask_JSONSerialization_RoundTrip 测试往返序列化
func TestTask_JSONSerialization_RoundTrip(t *testing.T) {
	original := &task.Task{
		ID:              "task-003",
		TemplateID:      "tpl-001",
		TemplateVersion: 1,
		BusinessID:      "biz-001",
		Params:          json.RawMessage(`{"key": "value"}`),
		State:           types.TaskStateApproved,
		CurrentNode:     "end",
		CreatedAt:       time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
		NodeOutputs: map[string]json.RawMessage{
			"node-1": json.RawMessage(`{"output": "data"}`),
		},
		Approvers: map[string][]string{
			"node-1": {"user-1"},
		},
		CompletedNodes: []string{"node-1", "node-2"},
	}

	// 多次往返序列化
	var current = original
	for i := 0; i < 3; i++ {
		data, err := json.Marshal(current)
		require.NoError(t, err)

		var restored task.Task
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)

		// 验证关键字段
		assert.Equal(t, original.ID, restored.ID)
		assert.Equal(t, original.TemplateID, restored.TemplateID)
		assert.Equal(t, original.State, restored.State)
		assert.Equal(t, len(original.NodeOutputs), len(restored.NodeOutputs))
		assert.Equal(t, len(original.Approvers), len(restored.Approvers))
		assert.Equal(t, len(original.CompletedNodes), len(restored.CompletedNodes))

		current = &restored
	}
}

