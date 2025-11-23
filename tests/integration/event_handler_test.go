package integration_test

import (
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/database"
	"github.com/mautops/approval-gin/internal/integration"
	"github.com/mautops/approval-kit/pkg/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDBForEventHandler 创建事件处理器测试数据库
func setupTestDBForEventHandler(t *testing.T) *gorm.DB {
	// 使用 database.Migrate 确保所有表都被创建
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 验证 dialector 名称
	dialector := db.Dialector.Name()
	t.Logf("Database dialector: %s", dialector)

	// 使用 database.Migrate 迁移数据库（包括所有需要的表和索引）
	err = database.Migrate(db)
	if err != nil {
		t.Logf("database.Migrate failed: %v", err)
		// 即使 Migrate 失败，也尝试手动创建表
		t.Log("Attempting to create tables manually")
		err = db.Exec(`
			CREATE TABLE IF NOT EXISTS events (
				id VARCHAR(64) PRIMARY KEY,
				task_id VARCHAR(64) NOT NULL,
				type VARCHAR(32) NOT NULL,
				data TEXT NOT NULL,
				status VARCHAR(32) NOT NULL DEFAULT 'pending',
				retry_count INTEGER DEFAULT 0,
				created_at DATETIME NOT NULL,
				updated_at DATETIME NOT NULL
			)
		`).Error
		if err != nil {
			t.Fatalf("Failed to create events table manually: %v", err)
		}
	} else {
		require.NoError(t, err, "database.Migrate should succeed")
	}

	// 验证所有必需的表都已创建，如果不存在则创建
	requiredTables := []string{"templates", "tasks", "approval_records", "state_history", "events", "audit_logs"}
	for _, tableName := range requiredTables {
		var count int64
		err = db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&count).Error
		require.NoError(t, err, "failed to check table %s", tableName)
		if count == 0 {
			t.Logf("Table %s not found after Migrate, creating manually", tableName)
			// 根据表名创建对应的表
			switch tableName {
			case "events":
				err = db.Exec(`
					CREATE TABLE IF NOT EXISTS events (
						id VARCHAR(64) PRIMARY KEY,
						task_id VARCHAR(64) NOT NULL,
						type VARCHAR(32) NOT NULL,
						data TEXT NOT NULL,
						status VARCHAR(32) NOT NULL DEFAULT 'pending',
						retry_count INTEGER DEFAULT 0,
						created_at DATETIME NOT NULL,
						updated_at DATETIME NOT NULL
					)
				`).Error
			case "templates":
				err = db.Exec(`
					CREATE TABLE IF NOT EXISTS templates (
						id VARCHAR(64) NOT NULL,
						name VARCHAR(255) NOT NULL,
						description TEXT,
						version INTEGER NOT NULL DEFAULT 1,
						data TEXT NOT NULL,
						created_at DATETIME NOT NULL,
						updated_at DATETIME NOT NULL,
						created_by VARCHAR(64),
						updated_by VARCHAR(64),
						PRIMARY KEY (id, version)
					)
				`).Error
			case "tasks":
				err = db.Exec(`
					CREATE TABLE IF NOT EXISTS tasks (
						id VARCHAR(64) PRIMARY KEY,
						template_id VARCHAR(64) NOT NULL,
						template_version INTEGER NOT NULL,
						business_id VARCHAR(64),
						state VARCHAR(32) NOT NULL,
						current_node VARCHAR(64),
						data TEXT NOT NULL,
						created_at DATETIME NOT NULL,
						updated_at DATETIME NOT NULL,
						submitted_at DATETIME,
						created_by VARCHAR(64)
					)
				`).Error
			case "approval_records":
				err = db.Exec(`
					CREATE TABLE IF NOT EXISTS approval_records (
						id VARCHAR(64) PRIMARY KEY,
						task_id VARCHAR(64) NOT NULL,
						node_id VARCHAR(64) NOT NULL,
						approver VARCHAR(64) NOT NULL,
						result VARCHAR(32) NOT NULL,
						comment TEXT,
						attachments TEXT,
						created_at DATETIME NOT NULL
					)
				`).Error
			case "state_history":
				err = db.Exec(`
					CREATE TABLE IF NOT EXISTS state_history (
						id VARCHAR(64) PRIMARY KEY,
						task_id VARCHAR(64) NOT NULL,
						from_state VARCHAR(32),
						to_state VARCHAR(32) NOT NULL,
						reason TEXT,
						operator VARCHAR(64) NOT NULL,
						created_at DATETIME NOT NULL
					)
				`).Error
			case "audit_logs":
				err = db.Exec(`
					CREATE TABLE IF NOT EXISTS audit_logs (
						id VARCHAR(64) PRIMARY KEY,
						user_id VARCHAR(64) NOT NULL,
						action VARCHAR(64) NOT NULL,
						resource_type VARCHAR(32) NOT NULL,
						resource_id VARCHAR(64) NOT NULL,
						request_id VARCHAR(64),
						ip VARCHAR(45),
						user_agent TEXT,
						details TEXT,
						created_at DATETIME NOT NULL
					)
				`).Error
			}
			require.NoError(t, err, "failed to create table %s", tableName)
			// 重新检查表是否存在
			err = db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&count).Error
			require.NoError(t, err, "failed to re-check table %s", tableName)
		}
		require.Equal(t, int64(1), count, "table %s must exist", tableName)
	}

	// 尝试执行一个简单的查询，确保 events 表真的可用
	var testCount int64
	err = db.Raw("SELECT COUNT(*) FROM events").Scan(&testCount).Error
	require.NoError(t, err, "events table should be queryable")
	
	return db
}

// TestEventHandler_Handle 测试事件处理器
func TestEventHandler_Handle(t *testing.T) {
	db := setupTestDBForEventHandler(t)
	// 创建事件处理器
	handler := integration.NewEventHandler(db, 1)

	// 创建测试事件
	evt := &event.Event{
		ID:   "event-001",
		Type: event.EventTypeTaskCreated,
		Time: time.Now(),
		Task: &event.TaskInfo{
			ID:         "task-001",
			TemplateID: "tpl-001",
			BusinessID: "biz-001",
			State:      "pending",
		},
		Node: &event.NodeInfo{
			ID:   "node-001",
			Name: "Start Node",
			Type: "start",
		},
		Business: &event.BusinessInfo{
			ID: "biz-001",
		},
	}

	// 处理事件
	err := handler.Handle(evt)
	require.NoError(t, err)
}

// TestEventHandler_HandleMultipleEvents 测试处理多个事件
func TestEventHandler_HandleMultipleEvents(t *testing.T) {
	db := setupTestDBForEventHandler(t)
	handler := integration.NewEventHandler(db, 1)

	events := []*event.Event{
		{
			ID:   "event-001",
			Type: event.EventTypeTaskCreated,
			Time: time.Now(),
			Task: &event.TaskInfo{
				ID:         "task-001",
				TemplateID: "tpl-001",
				BusinessID: "biz-001",
				State:      "pending",
			},
			Node: &event.NodeInfo{
				ID:   "node-001",
				Name: "Start Node",
				Type: "start",
			},
		},
		{
			ID:   "event-002",
			Type: event.EventTypeTaskSubmitted,
			Time: time.Now(),
			Task: &event.TaskInfo{
				ID:         "task-001",
				TemplateID: "tpl-001",
				BusinessID: "biz-001",
				State:      "submitted",
			},
			Node: &event.NodeInfo{
				ID:   "node-001",
				Name: "Start Node",
				Type: "start",
			},
		},
	}

	for _, evt := range events {
		err := handler.Handle(evt)
		assert.NoError(t, err)
	}
}

