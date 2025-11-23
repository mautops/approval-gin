package service_test

import (
	"context"
	"testing"

	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-gin/internal/repository"
	"github.com/mautops/approval-gin/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDBForAuditLog 创建测试数据库
func setupTestDBForAuditLog(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 迁移数据库
	err = db.AutoMigrate(&model.AuditLogModel{})
	require.NoError(t, err)

	return db
}

// TestAuditLog_RecordTemplateCreation 测试记录模板创建审计日志
func TestAuditLog_RecordTemplateCreation(t *testing.T) {
	db := setupTestDBForAuditLog(t)
	auditRepo := repository.NewAuditLogRepository(db)

	// 创建审计日志服务
	auditService := service.NewAuditLogService(auditRepo)

	ctx := context.Background()
	userID := "user-001"

	// 记录模板创建
	err := auditService.RecordAction(ctx, userID, "create", "template", "tpl-001", "created template")
	require.NoError(t, err)

	// 验证审计日志已保存
	logs, err := auditRepo.FindByUserID(userID)
	require.NoError(t, err)
	assert.Len(t, logs, 1, "should have one audit log")
	assert.Equal(t, "create", logs[0].Action)
	assert.Equal(t, "template", logs[0].ResourceType)
	assert.Equal(t, "tpl-001", logs[0].ResourceID)
}

// TestAuditLog_RecordTaskApproval 测试记录任务审批审计日志
func TestAuditLog_RecordTaskApproval(t *testing.T) {
	db := setupTestDBForAuditLog(t)
	auditRepo := repository.NewAuditLogRepository(db)

	// 创建审计日志服务
	auditService := service.NewAuditLogService(auditRepo)

	ctx := context.Background()
	userID := "user-001"

	// 记录任务审批
	err := auditService.RecordAction(ctx, userID, "approve", "task", "task-001", "approved task")
	require.NoError(t, err)

	// 验证审计日志已保存
	logs, err := auditRepo.FindByResource("task", "task-001")
	require.NoError(t, err)
	assert.Len(t, logs, 1, "should have one audit log")
	assert.Equal(t, "approve", logs[0].Action)
}

