package tests

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/integration"
	"github.com/mautops/approval-gin/internal/model"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestTemplateManagerCreate 测试模板创建
func TestTemplateManagerCreate(t *testing.T) {
	db := setupTestDB(t)
	mgr := integration.NewTemplateManager(db)
	
	// 创建测试模板数据
	tplData := map[string]interface{}{
		"id":          "tpl-001",
		"name":        "Test Template",
		"description": "Test Description",
		"version":     1,
		"created_at": time.Now(),
		"updated_at": time.Now(),
		"nodes":       make(map[string]interface{}),
		"edges":       []interface{}{},
		"config":      map[string]interface{}{},
	}
	tplJSON, _ := json.Marshal(tplData)
	
	// 先保存到数据库,然后通过 Manager 读取验证
	tm := &model.TemplateModel{
		ID:          "tpl-001",
		Name:        "Test Template",
		Description: "Test Description",
		Version:     1,
		Data:        tplJSON,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := db.Create(tm).Error
	assert.NoError(t, err)
	
	// 通过 Manager 读取
	retrieved, err := mgr.Get("tpl-001", 1)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "tpl-001", retrieved.ID)
	assert.Equal(t, "Test Template", retrieved.Name)
}

// TestTemplateManagerGet 测试模板查询
func TestTemplateManagerGet(t *testing.T) {
	db := setupTestDB(t)
	mgr := integration.NewTemplateManager(db)
	
	// 先创建模板数据
	tplData := createTestTemplateData("tpl-002", 1)
	tplJSON, _ := json.Marshal(tplData)
	tm := &model.TemplateModel{
		ID:          "tpl-002",
		Name:        "Test Template tpl-002",
		Description: "Test Description",
		Version:     1,
		Data:        tplJSON,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := db.Create(tm).Error
	assert.NoError(t, err)
	
	// 查询模板
	retrieved, err := mgr.Get("tpl-002", 1)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "tpl-002", retrieved.ID)
	assert.Equal(t, 1, retrieved.Version)
	
	// 查询最新版本
	retrievedLatest, err := mgr.Get("tpl-002", 0)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedLatest)
	assert.Equal(t, "tpl-002", retrievedLatest.ID)
}

// TestTemplateManagerUpdate 测试模板更新
func TestTemplateManagerUpdate(t *testing.T) {
	db := setupTestDB(t)
	mgr := integration.NewTemplateManager(db)
	
	// 先创建模板数据
	tplData1 := createTestTemplateData("tpl-003", 1)
	tplJSON1, _ := json.Marshal(tplData1)
	tm1 := &model.TemplateModel{
		ID:          "tpl-003",
		Name:        "Test Template tpl-003",
		Description: "Test Description",
		Version:     1,
		Data:        tplJSON1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := db.Create(tm1).Error
	assert.NoError(t, err)
	
	// 通过 Manager 更新(会创建新版本)
	// 注意: 这里需要先获取模板对象,但由于不能直接导入 template 包,
	// 我们通过 Manager 的 Get 方法获取,然后更新
	retrieved, err := mgr.Get("tpl-003", 1)
	assert.NoError(t, err)
	retrieved.Name = "Updated Template"
	err = mgr.Update("tpl-003", retrieved)
	assert.NoError(t, err)
	
	// 验证新版本已创建
	retrievedNew, err := mgr.Get("tpl-003", 2)
	assert.NoError(t, err)
	assert.Equal(t, 2, retrievedNew.Version)
	assert.Equal(t, "Updated Template", retrievedNew.Name)
	
	// 验证旧版本仍然存在
	oldVersion, err := mgr.Get("tpl-003", 1)
	assert.NoError(t, err)
	assert.Equal(t, 1, oldVersion.Version)
}

// TestTemplateManagerDelete 测试模板删除
func TestTemplateManagerDelete(t *testing.T) {
	db := setupTestDB(t)
	mgr := integration.NewTemplateManager(db)
	
	// 先创建模板数据
	tplData := createTestTemplateData("tpl-004", 1)
	tplJSON, _ := json.Marshal(tplData)
	tm := &model.TemplateModel{
		ID:          "tpl-004",
		Name:        "Test Template tpl-004",
		Description: "Test Description",
		Version:     1,
		Data:        tplJSON,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := db.Create(tm).Error
	assert.NoError(t, err)
	
	// 删除模板
	err = mgr.Delete("tpl-004")
	assert.NoError(t, err)
	
	// 验证模板已删除
	_, err = mgr.Get("tpl-004", 0)
	assert.Error(t, err)
}

// TestTemplateManagerListVersions 测试版本列表
func TestTemplateManagerListVersions(t *testing.T) {
	db := setupTestDB(t)
	mgr := integration.NewTemplateManager(db)
	
	// 创建多个版本的模板数据
	for i := 1; i <= 3; i++ {
		tplData := createTestTemplateData("tpl-005", i)
		tplJSON, _ := json.Marshal(tplData)
		tm := &model.TemplateModel{
			ID:          "tpl-005",
			Name:        "Test Template tpl-005",
			Description: "Test Description",
			Version:     i,
			Data:        tplJSON,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		err := db.Create(tm).Error
		assert.NoError(t, err)
	}
	
	// 列出所有版本
	versions, err := mgr.ListVersions("tpl-005")
	assert.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, versions)
}

// setupTestDB 设置测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	
	// 执行迁移
	err = db.AutoMigrate(
		&model.TemplateModel{},
		&model.TaskModel{},
		&model.ApprovalRecordModel{},
		&model.StateHistoryModel{},
		&model.EventModel{},
		&model.AuditLogModel{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	
	return db
}

// createTestTemplateData 创建测试模板数据(JSON 格式)
func createTestTemplateData(id string, version int) map[string]interface{} {
	return map[string]interface{}{
		"id":          id,
		"name":        "Test Template " + id,
		"description": "Test Description",
		"version":     version,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"nodes":       make(map[string]interface{}),
		"edges":       []interface{}{},
		"config":      map[string]interface{}{},
	}
}

