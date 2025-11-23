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

// setupTestDBForRecord 创建审批记录测试数据库
func setupTestDBForRecord(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 迁移数据库
	err = db.AutoMigrate(&model.ApprovalRecordModel{})
	require.NoError(t, err)

	return db
}

// TestApprovalRecordRepository_Save 测试保存审批记录
func TestApprovalRecordRepository_Save(t *testing.T) {
	db := setupTestDBForRecord(t)
	repo := repository.NewApprovalRecordRepository(db)

	record := &model.ApprovalRecordModel{
		ID:          "record-001",
		TaskID:      "task-001",
		NodeID:      "node-001",
		Approver:    "user-001",
		Result:      "approve",
		Comment:     "同意",
		Attachments: []byte(`["file-1.pdf"]`), // JSON 格式
		CreatedAt:   time.Now(),
	}

	err := repo.Save(record)
	assert.NoError(t, err)

	// 验证记录已保存
	var saved model.ApprovalRecordModel
	err = db.Where("id = ?", "record-001").First(&saved).Error
	assert.NoError(t, err)
	assert.Equal(t, "record-001", saved.ID)
	assert.Equal(t, "task-001", saved.TaskID)
	assert.Equal(t, "approve", saved.Result)
}

// TestApprovalRecordRepository_FindByTaskID 测试根据任务 ID 查找审批记录
func TestApprovalRecordRepository_FindByTaskID(t *testing.T) {
	db := setupTestDBForRecord(t)
	repo := repository.NewApprovalRecordRepository(db)

	// 先保存多个记录
	for i := 1; i <= 3; i++ {
		record := &model.ApprovalRecordModel{
			ID:          "record-00" + string(rune(i+'0')),
			TaskID:      "task-001",
			NodeID:      "node-001",
			Approver:    "user-00" + string(rune(i+'0')),
			Result:      "approve",
			Comment:     "同意",
			Attachments: []byte(`[]`), // JSON 格式
			CreatedAt:   time.Now(),
		}
		err := repo.Save(record)
		require.NoError(t, err)
	}

	// 查找任务的所有记录
	records, err := repo.FindByTaskID("task-001")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(records))
	for _, record := range records {
		assert.Equal(t, "task-001", record.TaskID)
	}
}

// TestApprovalRecordRepository_FindByApprover 测试根据审批人查找审批记录
func TestApprovalRecordRepository_FindByApprover(t *testing.T) {
	db := setupTestDBForRecord(t)
	repo := repository.NewApprovalRecordRepository(db)

	// 先保存多个记录
	approvers := []string{"user-001", "user-001", "user-002"}
	for i, approver := range approvers {
		record := &model.ApprovalRecordModel{
			ID:          "record-00" + string(rune(i+'1')),
			TaskID:      "task-00" + string(rune(i+'1')),
			NodeID:      "node-001",
			Approver:    approver,
			Result:      "approve",
			Comment:     "同意",
			Attachments: []byte(`[]`), // JSON 格式
			CreatedAt:   time.Now(),
		}
		err := repo.Save(record)
		require.NoError(t, err)
	}

	// 查找审批人的所有记录
	records, err := repo.FindByApprover("user-001")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(records))
	for _, record := range records {
		assert.Equal(t, "user-001", record.Approver)
	}
}

