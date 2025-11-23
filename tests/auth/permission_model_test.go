package auth_test

import (
	"testing"

	"github.com/mautops/approval-gin/internal/auth"
	"github.com/stretchr/testify/assert"
)

// TestPermissionModel_Definition 测试权限模型定义
func TestPermissionModel_Definition(t *testing.T) {
	// 验证权限模型定义是否正确
	model := auth.GetPermissionModel()
	
	assert.NotNil(t, model)
	assert.Contains(t, model, "type user")
	assert.Contains(t, model, "type template")
	assert.Contains(t, model, "type task")
	
	// 验证模板权限关系
	assert.Contains(t, model, "define owner")
	assert.Contains(t, model, "define viewer")
	assert.Contains(t, model, "define editor")
	
	// 验证任务权限关系
	assert.Contains(t, model, "define creator")
	assert.Contains(t, model, "define approver")
	assert.Contains(t, model, "define viewer")
}


