package tests

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/model"
	"github.com/stretchr/testify/assert"
)

// TestApprovalRecordModel 测试审批记录数据模型
func TestApprovalRecordModel(t *testing.T) {
	attachmentsJSON, _ := json.Marshal([]string{"file1.pdf", "file2.pdf"})
	
	arm := &model.ApprovalRecordModel{
		ID:          "record-001",
		TaskID:      "task-001",
		NodeID:      "node-001",
		Approver:    "user-001",
		Result:      "approve",
		Comment:     "Approved",
		Attachments: attachmentsJSON,
		CreatedAt:   time.Now(),
	}
	
	// 验证模型字段
	assert.Equal(t, "record-001", arm.ID)
	assert.Equal(t, "task-001", arm.TaskID)
	assert.Equal(t, "approve", arm.Result)
}

// TestApprovalRecordModelTableName 测试表名
func TestApprovalRecordModelTableName(t *testing.T) {
	arm := model.ApprovalRecordModel{}
	assert.Equal(t, "approval_records", arm.TableName())
}

// TestApprovalRecordModelValidation 测试模型验证
func TestApprovalRecordModelValidation(t *testing.T) {
	arm := &model.ApprovalRecordModel{
		ID:       "record-001",
		TaskID:   "task-001",
		NodeID:   "node-001",
		Approver: "user-001",
		Result:   "approve",
	}
	
	err := arm.Validate()
	assert.NoError(t, err)
	
	// 测试无效模型 - ID 为空
	armInvalidID := &model.ApprovalRecordModel{
		ID:       "",
		TaskID:   "task-001",
		NodeID:   "node-001",
		Approver: "user-001",
		Result:   "approve",
	}
	err = armInvalidID.Validate()
	assert.Error(t, err)
	
	// 测试无效模型 - TaskID 为空
	armInvalidTaskID := &model.ApprovalRecordModel{
		ID:       "record-002",
		TaskID:   "",
		NodeID:   "node-001",
		Approver: "user-001",
		Result:   "approve",
	}
	err = armInvalidTaskID.Validate()
	assert.Error(t, err)
	
	// 测试无效模型 - Result 为空
	armInvalidResult := &model.ApprovalRecordModel{
		ID:       "record-003",
		TaskID:   "task-001",
		NodeID:   "node-001",
		Approver: "user-001",
		Result:   "",
	}
	err = armInvalidResult.Validate()
	assert.Error(t, err)
}

