package repository_test

import (
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-gin/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDBForAuditLog 创建审计日志测试数据库
func setupTestDBForAuditLog(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 迁移数据库
	err = db.AutoMigrate(&model.AuditLogModel{})
	require.NoError(t, err)

	return db
}

// TestAuditLogRepository_Save 测试保存审计日志
func TestAuditLogRepository_Save(t *testing.T) {
	db := setupTestDBForAuditLog(t)
	repo := repository.NewAuditLogRepository(db)

	auditLog := &model.AuditLogModel{
		ID:           "audit-001",
		UserID:       "user-001",
		Action:       "create",
		ResourceType: "template",
		ResourceID:   "tpl-001",
		RequestID:    "req-001",
		IP:           "127.0.0.1",
		UserAgent:    "test-agent",
		Details:      []byte(`{"name":"测试模板"}`),
		CreatedAt:    time.Now(),
	}

	err := repo.Save(auditLog)
	assert.NoError(t, err)

	// 验证审计日志已保存
	var saved model.AuditLogModel
	err = db.Where("id = ?", "audit-001").First(&saved).Error
	assert.NoError(t, err)
	assert.Equal(t, "audit-001", saved.ID)
	assert.Equal(t, "user-001", saved.UserID)
	assert.Equal(t, "create", saved.Action)
	assert.Equal(t, "template", saved.ResourceType)
}

// TestAuditLogRepository_FindByUserID 测试根据用户 ID 查找审计日志
func TestAuditLogRepository_FindByUserID(t *testing.T) {
	db := setupTestDBForAuditLog(t)
	repo := repository.NewAuditLogRepository(db)

	// 先保存多个审计日志
	userIDs := []string{"user-001", "user-001", "user-002"}
	for i, userID := range userIDs {
		auditLog := &model.AuditLogModel{
			ID:           "audit-00" + string(rune(i+'1')),
			UserID:       userID,
			Action:       "create",
			ResourceType: "template",
			ResourceID:   "tpl-00" + string(rune(i+'1')),
			RequestID:    "req-00" + string(rune(i+'1')),
			IP:           "127.0.0.1",
			UserAgent:    "test-agent",
			Details:      []byte(`{}`),
			CreatedAt:    time.Now(),
		}
		err := repo.Save(auditLog)
		require.NoError(t, err)
	}

	// 查找用户的所有审计日志
	logs, err := repo.FindByUserID("user-001")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(logs))
	for _, log := range logs {
		assert.Equal(t, "user-001", log.UserID)
	}
}

// TestAuditLogRepository_FindByResource 测试根据资源查找审计日志
func TestAuditLogRepository_FindByResource(t *testing.T) {
	db := setupTestDBForAuditLog(t)
	repo := repository.NewAuditLogRepository(db)

	// 先保存多个审计日志
	resources := []struct {
		resourceType string
		resourceID   string
	}{
		{"template", "tpl-001"},
		{"template", "tpl-001"},
		{"task", "task-001"},
	}

	for i, resource := range resources {
		auditLog := &model.AuditLogModel{
			ID:           "audit-00" + string(rune(i+'1')),
			UserID:       "user-001",
			Action:       "create",
			ResourceType: resource.resourceType,
			ResourceID:   resource.resourceID,
			RequestID:    "req-00" + string(rune(i+'1')),
			IP:           "127.0.0.1",
			UserAgent:    "test-agent",
			Details:      []byte(`{}`),
			CreatedAt:    time.Now(),
		}
		err := repo.Save(auditLog)
		require.NoError(t, err)
	}

	// 查找资源的所有审计日志
	logs, err := repo.FindByResource("template", "tpl-001")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(logs))
	for _, log := range logs {
		assert.Equal(t, "template", log.ResourceType)
		assert.Equal(t, "tpl-001", log.ResourceID)
	}
}


