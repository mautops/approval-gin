package tests

import (
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/model"
	"github.com/stretchr/testify/assert"
)

// TestTaskModel 测试任务数据模型
func TestTaskModel(t *testing.T) {
	tm := &model.TaskModel{
		ID:             "task-001",
		TemplateID:     "tpl-001",
		TemplateVersion: 1,
		BusinessID:     "biz-001",
		State:          "pending",
		CurrentNode:    "node-001",
		Data:           []byte(`{"id":"task-001"}`),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		CreatedBy:      "user-001",
	}
	
	// 验证模型字段
	assert.Equal(t, "task-001", tm.ID)
	assert.Equal(t, "tpl-001", tm.TemplateID)
	assert.Equal(t, "pending", tm.State)
	assert.NotEmpty(t, tm.Data)
}

// TestTaskModelTableName 测试表名
func TestTaskModelTableName(t *testing.T) {
	tm := model.TaskModel{}
	assert.Equal(t, "tasks", tm.TableName())
}

// TestTaskModelValidation 测试模型验证
func TestTaskModelValidation(t *testing.T) {
	tm := &model.TaskModel{
		ID:             "task-001",
		TemplateID:     "tpl-001",
		TemplateVersion: 1,
		State:          "pending",
		Data:           []byte(`{}`),
	}
	
	err := tm.Validate()
	assert.NoError(t, err)
	
	// 测试无效模型 - ID 为空
	tmInvalidID := &model.TaskModel{
		ID:             "",
		TemplateID:     "tpl-001",
		TemplateVersion: 1,
		State:          "pending",
		Data:           []byte(`{}`),
	}
	err = tmInvalidID.Validate()
	assert.Error(t, err)
	
	// 测试无效模型 - TemplateID 为空
	tmInvalidTemplateID := &model.TaskModel{
		ID:             "task-002",
		TemplateID:     "",
		TemplateVersion: 1,
		State:          "pending",
		Data:           []byte(`{}`),
	}
	err = tmInvalidTemplateID.Validate()
	assert.Error(t, err)
	
	// 测试无效模型 - State 为空
	tmInvalidState := &model.TaskModel{
		ID:             "task-003",
		TemplateID:     "tpl-001",
		TemplateVersion: 1,
		State:          "",
		Data:           []byte(`{}`),
	}
	err = tmInvalidState.Validate()
	assert.Error(t, err)
}

