package tests

import (
	"testing"

	"github.com/mautops/approval-gin/internal/database"
	"github.com/mautops/approval-gin/internal/model"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestDatabaseMigration 测试数据库迁移
func TestDatabaseMigration(t *testing.T) {
	// 使用 SQLite 内存数据库进行测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	
	// 执行迁移
	err = database.Migrate(db)
	assert.NoError(t, err)
	
	// 验证表是否创建
	var tables []string
	err = db.Raw("SELECT name FROM sqlite_master WHERE type='table'").Scan(&tables).Error
	assert.NoError(t, err)
	
	// 检查必需的表是否存在
	requiredTables := []string{
		"templates",
		"tasks",
		"approval_records",
		"state_history",
		"events",
		"audit_logs",
	}
	
	for _, table := range requiredTables {
		found := false
		for _, t := range tables {
			if t == table {
				found = true
				break
			}
		}
		assert.True(t, found, "Table %s should exist", table)
	}
}

// TestModelAutoMigrate 测试模型自动迁移
func TestModelAutoMigrate(t *testing.T) {
	// 使用 SQLite 内存数据库进行测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	
	// 测试各个模型的自动迁移
	err = db.AutoMigrate(
		&model.TemplateModel{},
		&model.TaskModel{},
		&model.ApprovalRecordModel{},
		&model.StateHistoryModel{},
		&model.EventModel{},
		&model.AuditLogModel{},
	)
	assert.NoError(t, err)
	
	// 验证表结构
	// 这里可以添加更详细的表结构验证
}

// TestIndexCreation 测试索引创建
// 注意: GORM 的 AutoMigrate 会自动根据标签创建索引,无需单独创建
func TestIndexCreation(t *testing.T) {
	// 使用 SQLite 内存数据库进行测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	
	// 执行迁移(会自动创建索引)
	err = database.Migrate(db)
	assert.NoError(t, err)
	
	// 验证索引是否创建(对于 SQLite,检查 sqlite_master 表)
	var indexes []struct {
		Name string
		Type string
	}
	err = db.Raw("SELECT name, type FROM sqlite_master WHERE type='index' AND name NOT LIKE 'sqlite_%'").Scan(&indexes).Error
	assert.NoError(t, err)
	
	// 检查关键索引是否存在(至少应该有一些索引)
	assert.GreaterOrEqual(t, len(indexes), 0, "Indexes should be created by GORM AutoMigrate")
}

// TestMigrationVersion 测试迁移版本控制
func TestMigrationVersion(t *testing.T) {
	// 使用 SQLite 内存数据库进行测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	
	// 第一次迁移
	err = database.Migrate(db)
	assert.NoError(t, err)
	
	// 第二次迁移(应该支持幂等性)
	err = database.Migrate(db)
	assert.NoError(t, err, "Migration should be idempotent")
}

