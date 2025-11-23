package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/mautops/approval-gin/internal/integration"
	"github.com/mautops/approval-gin/internal/service"
	"github.com/mautops/approval-kit/pkg/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTemplateService_List 测试模板列表查询
func TestTemplateService_List(t *testing.T) {
	db := setupTestDBForService(t)
	templateMgr := integration.NewTemplateManager(db)
	templateService := service.NewTemplateService(templateMgr, db, service.NewAuditLogService(nil))

	// 创建多个模板
	for i := 1; i <= 5; i++ {
		req := &service.CreateTemplateRequest{
			Name:        "模板" + string(rune(i+'0')),
			Description: "模板描述" + string(rune(i+'0')),
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
		SortBy:   "created_at",
		Order:    "desc",
	}

	response, err := templateService.List(filter)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Len(t, response.Data, 5)
	assert.Equal(t, int64(5), response.Pagination.Total)
}

// TestTemplateService_ListWithPagination 测试分页查询
func TestTemplateService_ListWithPagination(t *testing.T) {
	db := setupTestDBForService(t)
	templateMgr := integration.NewTemplateManager(db)
	templateService := service.NewTemplateService(templateMgr, db, service.NewAuditLogService(nil))

	// 创建10个模板
	for i := 1; i <= 10; i++ {
		req := &service.CreateTemplateRequest{
			Name:        fmt.Sprintf("模板%d", i),
			Description: fmt.Sprintf("模板描述%d", i),
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
		PageSize: 5,
		SortBy:   "created_at",
		Order:    "desc",
	}

	response, err := templateService.List(filter)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Len(t, response.Data, 5)
	assert.Equal(t, int64(10), response.Pagination.Total)
	assert.Equal(t, 2, response.Pagination.TotalPage)
}

// TestTemplateService_ListWithSearch 测试搜索查询
func TestTemplateService_ListWithSearch(t *testing.T) {
	db := setupTestDBForService(t)
	templateMgr := integration.NewTemplateManager(db)
	templateService := service.NewTemplateService(templateMgr, db, service.NewAuditLogService(nil))

	// 创建模板
	req1 := &service.CreateTemplateRequest{
		Name:        "请假审批模板",
		Description: "员工请假审批",
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	}
	_, err := templateService.Create(context.Background(), req1)
	require.NoError(t, err)

	req2 := &service.CreateTemplateRequest{
		Name:        "报销审批模板",
		Description: "费用报销审批",
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	}
	_, err = templateService.Create(context.Background(), req2)
	require.NoError(t, err)

	// 测试搜索
	filter := &service.TemplateListFilter{
		Page:     1,
		PageSize: 10,
		Search:   "请假",
		SortBy:   "created_at",
		Order:    "desc",
	}

	response, err := templateService.List(filter)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.GreaterOrEqual(t, len(response.Data), 1)
}

// TestTemplateService_ListWithSorting 测试排序查询
func TestTemplateService_ListWithSorting(t *testing.T) {
	db := setupTestDBForService(t)
	templateMgr := integration.NewTemplateManager(db)
	templateService := service.NewTemplateService(templateMgr, db, service.NewAuditLogService(nil))

	// 创建模板
	for i := 1; i <= 3; i++ {
		req := &service.CreateTemplateRequest{
			Name:        fmt.Sprintf("模板%d", i),
			Description: fmt.Sprintf("模板描述%d", i),
			Nodes:       make(map[string]*template.Node),
			Edges:       []*template.Edge{},
			Config:      nil,
		}
		_, err := templateService.Create(context.Background(), req)
		require.NoError(t, err)
	}

	// 测试升序排序
	filter := &service.TemplateListFilter{
		Page:     1,
		PageSize: 10,
		SortBy:   "name",
		Order:    "asc",
	}

	response, err := templateService.List(filter)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.GreaterOrEqual(t, len(response.Data), 3)
}
