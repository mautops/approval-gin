package tests

import (
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/model"
	"github.com/stretchr/testify/assert"
)

// TestTemplateModel 测试模板数据模型
func TestTemplateModel(t *testing.T) {
	tm := &model.TemplateModel{
		ID:          "tpl-001",
		Name:        "Test Template",
		Description: "Test Description",
		Version:     1,
		Data:        []byte(`{"id":"tpl-001","name":"Test Template"}`),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   "user-001",
		UpdatedBy:   "user-001",
	}
	
	// 验证模型字段
	assert.Equal(t, "tpl-001", tm.ID)
	assert.Equal(t, "Test Template", tm.Name)
	assert.Equal(t, 1, tm.Version)
	assert.NotEmpty(t, tm.Data)
}

// TestTemplateModelTableName 测试表名
func TestTemplateModelTableName(t *testing.T) {
	tm := model.TemplateModel{}
	assert.Equal(t, "templates", tm.TableName())
}

// TestTemplateModelValidation 测试模型验证
func TestTemplateModelValidation(t *testing.T) {
	tm := &model.TemplateModel{
		ID:   "tpl-001",
		Name: "Test Template",
		Data: []byte(`{}`),
	}
	
	err := tm.Validate()
	assert.NoError(t, err)
	
	// 测试无效模型 - ID 为空
	tmInvalidID := &model.TemplateModel{
		ID:   "",
		Name: "Test Template",
		Data: []byte(`{}`),
	}
	err = tmInvalidID.Validate()
	assert.Error(t, err)
	
	// 测试无效模型 - Name 为空
	tmInvalidName := &model.TemplateModel{
		ID:   "tpl-002",
		Name: "",
		Data: []byte(`{}`),
	}
	err = tmInvalidName.Validate()
	assert.Error(t, err)
	
	// 测试无效模型 - Data 为空
	tmInvalidData := &model.TemplateModel{
		ID:   "tpl-003",
		Name: "Test Template",
		Data: nil,
	}
	err = tmInvalidData.Validate()
	assert.Error(t, err)
}

