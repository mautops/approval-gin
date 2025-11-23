package integration_test

import (
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/integration"
	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-kit/pkg/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 迁移数据库
	err = db.AutoMigrate(&model.TemplateModel{})
	require.NoError(t, err)

	return db
}

// TestTemplateManager_Create 测试创建模板
func TestTemplateManager_Create(t *testing.T) {
	db := setupTestDB(t)
	mgr := integration.NewTemplateManager(db)

	tpl := &template.Template{
		ID:          "tpl-001",
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	}

	err := mgr.Create(tpl)
	assert.NoError(t, err)

	// 验证模板已保存
	var tm model.TemplateModel
	err = db.Where("id = ?", "tpl-001").First(&tm).Error
	assert.NoError(t, err)
	assert.Equal(t, "tpl-001", tm.ID)
	assert.Equal(t, "测试模板", tm.Name)
}

// TestTemplateManager_Get 测试获取模板
func TestTemplateManager_Get(t *testing.T) {
	db := setupTestDB(t)
	mgr := integration.NewTemplateManager(db)

	// 先创建模板
	tpl := &template.Template{
		ID:          "tpl-001",
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	}
	err := mgr.Create(tpl)
	require.NoError(t, err)

	// 获取模板
	got, err := mgr.Get("tpl-001", 0)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, "tpl-001", got.ID)
	assert.Equal(t, "测试模板", got.Name)
}

// TestTemplateManager_Update 测试更新模板
func TestTemplateManager_Update(t *testing.T) {
	db := setupTestDB(t)
	mgr := integration.NewTemplateManager(db)

	// 先创建模板
	tpl := &template.Template{
		ID:          "tpl-001",
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	}
	err := mgr.Create(tpl)
	require.NoError(t, err)

	// 更新模板
	updated := &template.Template{
		ID:          "tpl-001",
		Name:        "更新后的模板",
		Description: "这是更新后的模板",
		Version:     2,
		CreatedAt:   tpl.CreatedAt,
		UpdatedAt:   time.Now(),
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	}
	err = mgr.Update("tpl-001", updated)
	assert.NoError(t, err)

	// 验证新版本已创建
	got, err := mgr.Get("tpl-001", 2)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, 2, got.Version)
	assert.Equal(t, "更新后的模板", got.Name)
}

// TestTemplateManager_Delete 测试删除模板
func TestTemplateManager_Delete(t *testing.T) {
	db := setupTestDB(t)
	mgr := integration.NewTemplateManager(db)

	// 先创建模板
	tpl := &template.Template{
		ID:          "tpl-001",
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	}
	err := mgr.Create(tpl)
	require.NoError(t, err)

	// 删除模板
	err = mgr.Delete("tpl-001")
	assert.NoError(t, err)

	// 验证模板已删除
	_, err = mgr.Get("tpl-001", 0)
	assert.Error(t, err)
}

// TestTemplateManager_ListVersions 测试列出模板版本
func TestTemplateManager_ListVersions(t *testing.T) {
	db := setupTestDB(t)
	mgr := integration.NewTemplateManager(db)

	// 创建多个版本的模板
	for i := 1; i <= 3; i++ {
		tpl := &template.Template{
			ID:          "tpl-001",
			Name:        "测试模板",
			Description: "这是一个测试模板",
			Version:     i,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Nodes:       make(map[string]*template.Node),
			Edges:       []*template.Edge{},
			Config:      nil,
		}
		err := mgr.Create(tpl)
		require.NoError(t, err)
	}

	// 列出版本
	versions, err := mgr.ListVersions("tpl-001")
	assert.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, versions)
}

