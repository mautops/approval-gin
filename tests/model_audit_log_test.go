package tests

import (
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/model"
	"github.com/stretchr/testify/assert"
)

// TestAuditLogModel 测试审计日志数据模型
func TestAuditLogModel(t *testing.T) {
	alm := &model.AuditLogModel{
		ID:           "audit-001",
		UserID:       "user-001",
		Action:       "create",
		ResourceType: "template",
		ResourceID:   "tpl-001",
		RequestID:    "req-001",
		IP:           "127.0.0.1",
		UserAgent:    "test-agent",
		Details:      []byte(`{"action":"create"}`),
		CreatedAt:    time.Now(),
	}
	
	// 验证模型字段
	assert.Equal(t, "audit-001", alm.ID)
	assert.Equal(t, "user-001", alm.UserID)
	assert.Equal(t, "create", alm.Action)
	assert.Equal(t, "template", alm.ResourceType)
}

// TestAuditLogModelTableName 测试表名
func TestAuditLogModelTableName(t *testing.T) {
	alm := model.AuditLogModel{}
	assert.Equal(t, "audit_logs", alm.TableName())
}

// TestAuditLogModelValidation 测试模型验证
func TestAuditLogModelValidation(t *testing.T) {
	alm := &model.AuditLogModel{
		ID:           "audit-001",
		UserID:       "user-001",
		Action:       "create",
		ResourceType: "template",
		ResourceID:   "tpl-001",
	}
	
	err := alm.Validate()
	assert.NoError(t, err)
	
	// 测试无效模型 - ID 为空
	almInvalidID := &model.AuditLogModel{
		ID:           "",
		UserID:       "user-001",
		Action:       "create",
		ResourceType: "template",
		ResourceID:   "tpl-001",
	}
	err = almInvalidID.Validate()
	assert.Error(t, err)
	
	// 测试无效模型 - UserID 为空
	almInvalidUserID := &model.AuditLogModel{
		ID:           "audit-002",
		UserID:       "",
		Action:       "create",
		ResourceType: "template",
		ResourceID:   "tpl-001",
	}
	err = almInvalidUserID.Validate()
	assert.Error(t, err)
	
	// 测试无效模型 - Action 为空
	almInvalidAction := &model.AuditLogModel{
		ID:           "audit-003",
		UserID:       "user-001",
		Action:       "",
		ResourceType: "template",
		ResourceID:   "tpl-001",
	}
	err = almInvalidAction.Validate()
	assert.Error(t, err)
}

