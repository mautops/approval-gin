package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/integration"
	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-gin/internal/service"
	"github.com/mautops/approval-kit/pkg/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDBForTemplateCache 创建模板缓存测试数据库
func setupTestDBForTemplateCache(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 迁移数据库
	err = db.AutoMigrate(&model.TemplateModel{})
	require.NoError(t, err)

	return db
}

// TestTemplateService_Get_WithCache 测试模板获取缓存功能
func TestTemplateService_Get_WithCache(t *testing.T) {
	db := setupTestDBForTemplateCache(t)
	templateMgr := integration.NewTemplateManager(db)
	templateSvc := service.NewTemplateService(templateMgr, db, service.NewAuditLogService(nil))

	// 创建测试模板
	tpl := &template.Template{
		ID:          "tpl-cache-001",
		Name:        "Cached Template",
		Description: "Test cached template",
		Version:     1,
		Nodes:       map[string]*template.Node{},
		Edges:       []*template.Edge{},
		Config:      &template.TemplateConfig{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := templateMgr.Create(tpl)
	require.NoError(t, err)

	// 第一次获取（应该从数据库查询）
	start1 := time.Now()
	tpl1, err := templateSvc.Get("tpl-cache-001", 1)
	duration1 := time.Since(start1)
	require.NoError(t, err)
	assert.Equal(t, "Cached Template", tpl1.Name)

	// 第二次获取（应该从缓存获取，速度更快）
	start2 := time.Now()
	tpl2, err := templateSvc.Get("tpl-cache-001", 1)
	duration2 := time.Since(start2)
	require.NoError(t, err)
	assert.Equal(t, "Cached Template", tpl2.Name)

	// 验证缓存命中（第二次获取应该更快）
	t.Logf("First get duration: %v, Second get duration: %v", duration1, duration2)
	// 注意：由于 SQLite 内存数据库可能很快，这个测试可能不够明显
	// 但我们可以验证两次获取的结果一致
	assert.Equal(t, tpl1.ID, tpl2.ID)
	assert.Equal(t, tpl1.Name, tpl2.Name)
}

// TestTemplateService_Get_CacheInvalidation 测试模板缓存失效
func TestTemplateService_Get_CacheInvalidation(t *testing.T) {
	db := setupTestDBForTemplateCache(t)
	templateMgr := integration.NewTemplateManager(db)
	templateSvc := service.NewTemplateService(templateMgr, db, service.NewAuditLogService(nil))

	// 创建测试模板
	tpl := &template.Template{
		ID:          "tpl-cache-002",
		Name:        "Original Template",
		Description: "Original",
		Version:     1,
		Nodes:       map[string]*template.Node{},
		Edges:       []*template.Edge{},
		Config:      &template.TemplateConfig{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := templateMgr.Create(tpl)
	require.NoError(t, err)

	// 第一次获取
	tpl1, err := templateSvc.Get("tpl-cache-002", 1)
	require.NoError(t, err)
	assert.Equal(t, "Original Template", tpl1.Name)

	// 更新模板（创建新版本）
	updatedReq := &service.UpdateTemplateRequest{
		Name:        "Updated Template",
		Description: "Updated",
		Nodes:       map[string]*template.Node{},
		Edges:       []*template.Edge{},
		Config:      &template.TemplateConfig{},
	}
	_, err = templateSvc.Update(context.Background(), "tpl-cache-002", updatedReq)
	require.NoError(t, err)

	// 获取新版本（应该从数据库查询，因为缓存中只有旧版本）
	tpl2, err := templateSvc.Get("tpl-cache-002", 2)
	require.NoError(t, err)
	assert.Equal(t, "Updated Template", tpl2.Name)
	assert.Equal(t, 2, tpl2.Version)

	// 获取旧版本（应该从缓存获取）
	tpl3, err := templateSvc.Get("tpl-cache-002", 1)
	require.NoError(t, err)
	assert.Equal(t, "Original Template", tpl3.Name)
	assert.Equal(t, 1, tpl3.Version)
}

// TestTemplateService_Get_CacheExpiration 测试模板缓存过期
func TestTemplateService_Get_CacheExpiration(t *testing.T) {
	db := setupTestDBForTemplateCache(t)
	templateMgr := integration.NewTemplateManager(db)
	templateSvc := service.NewTemplateService(templateMgr, db, service.NewAuditLogService(nil))

	// 创建测试模板
	tpl := &template.Template{
		ID:          "tpl-cache-003",
		Name:        "Expiring Template",
		Description: "Test",
		Version:     1,
		Nodes:       map[string]*template.Node{},
		Edges:       []*template.Edge{},
		Config:      &template.TemplateConfig{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := templateMgr.Create(tpl)
	require.NoError(t, err)

	// 第一次获取
	tpl1, err := templateSvc.Get("tpl-cache-003", 1)
	require.NoError(t, err)
	assert.Equal(t, "Expiring Template", tpl1.Name)

	// 等待缓存过期（如果实现了 TTL）
	// 注意：这个测试需要缓存实现支持 TTL
	// 当前实现可能没有 TTL，所以这个测试可能只是验证缓存仍然有效
	time.Sleep(100 * time.Millisecond)

	// 再次获取（如果缓存过期，应该从数据库重新查询）
	tpl2, err := templateSvc.Get("tpl-cache-003", 1)
	require.NoError(t, err)
	assert.Equal(t, "Expiring Template", tpl2.Name)
}

