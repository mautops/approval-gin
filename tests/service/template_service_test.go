package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/mautops/approval-gin/internal/integration"
	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-gin/internal/service"
	"github.com/mautops/approval-kit/pkg/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDBForService 创建服务测试数据库
func setupTestDBForService(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 迁移数据库
	err = db.AutoMigrate(&model.TemplateModel{})
	require.NoError(t, err)

	return db
}

// TestTemplateService_Create 测试创建模板
func TestTemplateService_Create(t *testing.T) {
	db := setupTestDBForService(t)
	templateMgr := integration.NewTemplateManager(db)
	auditLogSvc := service.NewAuditLogService(nil) // 测试中不使用真实的审计日志仓储
	templateService := service.NewTemplateService(templateMgr, db, auditLogSvc)

	req := &service.CreateTemplateRequest{
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	}

	template, err := templateService.Create(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, template)
	assert.Equal(t, "测试模板", template.Name)
	assert.Equal(t, 1, template.Version)
}

// TestTemplateService_Get 测试获取模板
func TestTemplateService_Get(t *testing.T) {
	db := setupTestDBForService(t)
	templateMgr := integration.NewTemplateManager(db)
	auditLogSvc := service.NewAuditLogService(nil)
	templateService := service.NewTemplateService(templateMgr, db, auditLogSvc)

	// 先创建模板
	req := &service.CreateTemplateRequest{
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	}
	created, err := templateService.Create(context.Background(), req)
	require.NoError(t, err)

	// 获取模板
	got, err := templateService.Get(created.ID, 0)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "测试模板", got.Name)
}

// TestTemplateService_Update 测试更新模板
func TestTemplateService_Update(t *testing.T) {
	db := setupTestDBForService(t)
	templateMgr := integration.NewTemplateManager(db)
	auditLogSvc := service.NewAuditLogService(nil)
	templateService := service.NewTemplateService(templateMgr, db, auditLogSvc)

	// 先创建模板
	req := &service.CreateTemplateRequest{
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	}
	created, err := templateService.Create(context.Background(), req)
	require.NoError(t, err)

	// 更新模板
	updateReq := &service.UpdateTemplateRequest{
		Name:        "更新后的模板",
		Description: "这是更新后的模板",
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	}
	updated, err := templateService.Update(context.Background(), created.ID, updateReq)
	assert.NoError(t, err)
	assert.NotNil(t, updated)
	assert.Equal(t, created.ID, updated.ID)
	assert.Equal(t, "更新后的模板", updated.Name)
	assert.Equal(t, 2, updated.Version) // 版本号应该递增
}

// TestTemplateService_Delete 测试删除模板
func TestTemplateService_Delete(t *testing.T) {
	db := setupTestDBForService(t)
	templateMgr := integration.NewTemplateManager(db)
	auditLogSvc := service.NewAuditLogService(nil)
	templateService := service.NewTemplateService(templateMgr, db, auditLogSvc)

	// 先创建模板
	req := &service.CreateTemplateRequest{
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	}
	created, err := templateService.Create(context.Background(), req)
	require.NoError(t, err)

	// 删除模板
	err = templateService.Delete(context.Background(), created.ID)
	assert.NoError(t, err)

	// 验证模板已删除
	_, err = templateService.Get(created.ID, 0)
	assert.Error(t, err)
}

// TestTemplateService_ListVersions 测试列出模板版本
func TestTemplateService_ListVersions(t *testing.T) {
	db := setupTestDBForService(t)
	templateMgr := integration.NewTemplateManager(db)
	auditLogSvc := service.NewAuditLogService(nil)
	templateService := service.NewTemplateService(templateMgr, db, auditLogSvc)

	// 先创建模板
	req := &service.CreateTemplateRequest{
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	}
	created, err := templateService.Create(context.Background(), req)
	require.NoError(t, err)

	// 创建多个版本
	for i := 0; i < 2; i++ {
		updateReq := &service.UpdateTemplateRequest{
			Name:        "测试模板",
			Description: "这是一个测试模板",
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
			Config:      nil,
		}
		_, err = templateService.Update(context.Background(), created.ID, updateReq)
		require.NoError(t, err)
	}

	// 列出版本
	versions, err := templateService.ListVersions(created.ID)
	assert.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, versions)
}

// TestTemplateService_Get_NotFound 测试获取不存在的模板
func TestTemplateService_Get_NotFound(t *testing.T) {
	db := setupTestDBForService(t)
	templateMgr := integration.NewTemplateManager(db)
	auditLogSvc := service.NewAuditLogService(nil)
	templateService := service.NewTemplateService(templateMgr, db, auditLogSvc)

	// 获取不存在的模板
	_, err := templateService.Get("non-existent", 0)
	assert.Error(t, err)
}

// TestTemplateService_Update_NotFound 测试更新不存在的模板
func TestTemplateService_Update_NotFound(t *testing.T) {
	db := setupTestDBForService(t)
	templateMgr := integration.NewTemplateManager(db)
	auditLogSvc := service.NewAuditLogService(nil)
	templateService := service.NewTemplateService(templateMgr, db, auditLogSvc)

	// 更新不存在的模板
	updateReq := &service.UpdateTemplateRequest{
		Name:        "更新后的模板",
		Description: "这是更新后的模板",
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	}
	_, err := templateService.Update(context.Background(), "non-existent", updateReq)
	assert.Error(t, err)
}

// TestTemplateService_Delete_NotFound 测试删除不存在的模板
// 注意: GORM 的 Delete 方法在删除不存在的记录时不会返回错误(幂等操作)
func TestTemplateService_Delete_NotFound(t *testing.T) {
	db := setupTestDBForService(t)
	templateMgr := integration.NewTemplateManager(db)
	auditLogSvc := service.NewAuditLogService(nil)
	templateService := service.NewTemplateService(templateMgr, db, auditLogSvc)

	// 删除不存在的模板(应该不报错,幂等操作)
	err := templateService.Delete(context.Background(), "non-existent")
	assert.NoError(t, err)
}

// TestTemplateService_ListVersions_NotFound 测试列出不存在模板的版本
// 注意: GORM 的 Pluck 方法在查询不存在记录时返回空列表而不是错误
func TestTemplateService_ListVersions_NotFound(t *testing.T) {
	db := setupTestDBForService(t)
	templateMgr := integration.NewTemplateManager(db)
	auditLogSvc := service.NewAuditLogService(nil)
	templateService := service.NewTemplateService(templateMgr, db, auditLogSvc)

	// 列出不存在模板的版本(应该返回空列表)
	versions, err := templateService.ListVersions("non-existent")
	assert.NoError(t, err)
	assert.Empty(t, versions)
}

// TestTemplateService_Get_SpecificVersion 测试获取特定版本的模板
func TestTemplateService_Get_SpecificVersion(t *testing.T) {
	db := setupTestDBForService(t)
	templateMgr := integration.NewTemplateManager(db)
	auditLogSvc := service.NewAuditLogService(nil)
	templateService := service.NewTemplateService(templateMgr, db, auditLogSvc)

	// 先创建模板
	req := &service.CreateTemplateRequest{
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	}
	created, err := templateService.Create(context.Background(), req)
	require.NoError(t, err)

	// 创建第二个版本
	updateReq := &service.UpdateTemplateRequest{
		Name:        "更新后的模板",
		Description: "这是更新后的模板",
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	}
	_, err = templateService.Update(context.Background(), created.ID, updateReq)
	require.NoError(t, err)

	// 获取版本 1
	version1, err := templateService.Get(created.ID, 1)
	assert.NoError(t, err)
	assert.NotNil(t, version1)
	assert.Equal(t, 1, version1.Version)
	assert.Equal(t, "测试模板", version1.Name)

	// 获取版本 2
	version2, err := templateService.Get(created.ID, 2)
	assert.NoError(t, err)
	assert.NotNil(t, version2)
	assert.Equal(t, 2, version2.Version)
	assert.Equal(t, "更新后的模板", version2.Name)
}

// TestTemplateService_List_Basic 测试列表功能基础场景
func TestTemplateService_List_Basic(t *testing.T) {
	db := setupTestDBForService(t)
	templateMgr := integration.NewTemplateManager(db)
	auditLogSvc := service.NewAuditLogService(nil)
	templateService := service.NewTemplateService(templateMgr, db, auditLogSvc)

	// 创建多个模板
	for i := 1; i <= 5; i++ {
		req := &service.CreateTemplateRequest{
			Name:        fmt.Sprintf("测试模板%d", i),
			Description: "这是一个测试模板",
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
			Config:      nil,
		}
		_, err := templateService.Create(context.Background(), req)
		require.NoError(t, err)
	}

	// 测试列表查询
	filter := &service.TemplateListFilter{
		Page:     1,
		PageSize: 10,
	}
	result, err := templateService.List(filter)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, len(result.Data), 5)
	assert.Equal(t, 1, result.Pagination.Page)
	assert.Equal(t, 10, result.Pagination.PageSize)
}

// TestTemplateService_List_Pagination 测试分页功能
func TestTemplateService_List_Pagination(t *testing.T) {
	db := setupTestDBForService(t)
	templateMgr := integration.NewTemplateManager(db)
	auditLogSvc := service.NewAuditLogService(nil)
	templateService := service.NewTemplateService(templateMgr, db, auditLogSvc)

	// 创建多个模板
	for i := 1; i <= 10; i++ {
		req := &service.CreateTemplateRequest{
			Name:        fmt.Sprintf("测试模板%d", i),
			Description: "这是一个测试模板",
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
			Config:      nil,
		}
		_, err := templateService.Create(context.Background(), req)
		require.NoError(t, err)
	}

	// 测试第一页
	filter := &service.TemplateListFilter{
		Page:     1,
		PageSize: 3,
	}
	result, err := templateService.List(filter)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.LessOrEqual(t, len(result.Data), 3)
	assert.Equal(t, 1, result.Pagination.Page)
	assert.Equal(t, 3, result.Pagination.PageSize)
	assert.GreaterOrEqual(t, result.Pagination.Total, int64(10))
}

// TestTemplateService_List_Empty 测试空列表
func TestTemplateService_List_Empty(t *testing.T) {
	db := setupTestDBForService(t)
	templateMgr := integration.NewTemplateManager(db)
	auditLogSvc := service.NewAuditLogService(nil)
	templateService := service.NewTemplateService(templateMgr, db, auditLogSvc)

	// 测试空列表
	filter := &service.TemplateListFilter{
		Page:     1,
		PageSize: 10,
	}
	result, err := templateService.List(filter)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Data)
	assert.Equal(t, int64(0), result.Pagination.Total)
}

