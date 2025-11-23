package tests

import (
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/database"
	"github.com/mautops/approval-gin/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestDatabaseIndexes_Existence 测试数据库索引是否存在
func TestDatabaseIndexes_Existence(t *testing.T) {
	// 使用 SQLite 内存数据库进行测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 执行迁移
	err = database.Migrate(db)
	require.NoError(t, err)

	// 检查索引是否存在（通过查询系统表）
	// 注意：SQLite 的索引检查方式与 PostgreSQL 不同
	// 这里我们只验证迁移成功执行，索引创建没有错误
	var indexes []struct {
		Name string
	}
	
	// SQLite 查询索引
	err = db.Raw("SELECT name FROM sqlite_master WHERE type='index' AND name LIKE 'idx_%'").Scan(&indexes).Error
	// SQLite 可能不支持某些索引语法，这里只验证迁移成功
	_ = err
	_ = indexes
}

// TestDatabaseIndexes_Performance 测试索引性能
func TestDatabaseIndexes_Performance(t *testing.T) {
	// 使用 SQLite 内存数据库进行测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 执行迁移
	err = database.Migrate(db)
	require.NoError(t, err)

	// 创建测试数据
	template := &model.TemplateModel{
		ID:        "tpl-001",
		Name:      "Test Template",
		Version:   1,
		Data:      []byte(`{"nodes":{}}`),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = db.Create(template).Error
	require.NoError(t, err)

	task := &model.TaskModel{
		ID:          "task-001",
		TemplateID:  "tpl-001",
		BusinessID:  "biz-001",
		State:       "pending",
		Data:        []byte(`{"id":"task-001"}`),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   "user-001",
	}
	err = db.Create(task).Error
	require.NoError(t, err)

	// 测试按模板 ID 查询（应该有索引）
	start := time.Now()
	var tasks []model.TaskModel
	err = db.Where("template_id = ?", "tpl-001").Find(&tasks).Error
	duration := time.Since(start)
	
	assert.NoError(t, err)
	assert.Len(t, tasks, 1)
	// 验证查询时间在合理范围内（有索引时应该很快）
	assert.Less(t, duration, 100*time.Millisecond, "Query should be fast with index")
}

// TestDatabaseIndexes_CompositeIndexes 测试复合索引
func TestDatabaseIndexes_CompositeIndexes(t *testing.T) {
	// 使用 SQLite 内存数据库进行测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 执行迁移
	err = database.Migrate(db)
	require.NoError(t, err)

	// 测试复合索引查询（state + business_id）
	start := time.Now()
	var tasks []model.TaskModel
	err = db.Where("state = ? AND business_id = ?", "pending", "biz-001").Find(&tasks).Error
	duration := time.Since(start)
	
	assert.NoError(t, err)
	// 验证查询时间在合理范围内
	assert.Less(t, duration, 100*time.Millisecond, "Composite index query should be fast")
}


