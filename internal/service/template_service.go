package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mautops/approval-gin/internal/auth"
	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-gin/internal/utils"
	"github.com/mautops/approval-kit/pkg/template"
	"gorm.io/gorm"
)

// TemplateService 模板服务接口
type TemplateService interface {
	Create(ctx context.Context, req *CreateTemplateRequest) (*template.Template, error)
	Get(id string, version int) (*template.Template, error)
	Update(ctx context.Context, id string, req *UpdateTemplateRequest) (*template.Template, error)
	Delete(ctx context.Context, id string) error
	List(filter *TemplateListFilter) (*TemplateListResponse, error)
	ListVersions(id string) ([]int, error)
}

// CreateTemplateRequest 创建模板请求
// @Description 创建审批模板的请求参数
type CreateTemplateRequest struct {
	Name        string `json:"name" example:"请假审批" binding:"required"` // 模板名称
	Description string `json:"description" example:"员工请假审批流程"` // 模板描述
	Nodes       map[string]*template.Node `json:"nodes" binding:"required"` // 节点定义
	Edges       []*template.Edge `json:"edges" binding:"required"` // 节点连接关系
	Config      *template.TemplateConfig `json:"config"` // 模板配置
}

// UpdateTemplateRequest 更新模板请求
// @Description 更新审批模板的请求参数
type UpdateTemplateRequest struct {
	Name        string `json:"name" example:"请假审批"` // 模板名称
	Description string `json:"description" example:"员工请假审批流程"` // 模板描述
	Nodes       map[string]*template.Node `json:"nodes"` // 节点定义
	Edges       []*template.Edge `json:"edges"` // 节点连接关系
	Config      *template.TemplateConfig `json:"config"` // 模板配置
}

// TemplateListFilter 模板列表查询过滤器
type TemplateListFilter struct {
	Page     int
	PageSize int
	Search   string
	SortBy   string
	Order    string // asc/desc
}

// TemplateListResponse 模板列表响应
type TemplateListResponse struct {
	Data       []*template.Template
	Pagination PaginationInfo
}

// PaginationInfo 分页信息
type PaginationInfo struct {
	Page      int
	PageSize  int
	Total     int64
	TotalPage int
}

// templateCacheEntry 模板缓存条目
type templateCacheEntry struct {
	template  *template.Template
	expiresAt time.Time
}

// templateService 模板服务实现
type templateService struct {
	templateMgr  template.TemplateManager
	db           *gorm.DB
	fgaClient    *auth.OpenFGAClient
	auditLogSvc  AuditLogService
	cache        *sync.Map
	cacheTTL     time.Duration
}

// NewTemplateService 创建模板服务
func NewTemplateService(templateMgr template.TemplateManager, db *gorm.DB, auditLogSvc AuditLogService, fgaClient ...*auth.OpenFGAClient) TemplateService {
	var fga *auth.OpenFGAClient
	if len(fgaClient) > 0 {
		fga = fgaClient[0]
	}
	return &templateService{
		templateMgr: templateMgr,
		db:          db,
		fgaClient:   fga,
		auditLogSvc: auditLogSvc,
		cache:       &sync.Map{},
		cacheTTL:    5 * time.Minute, // 默认缓存 5 分钟
	}
}

// generateTemplateID 生成模板 ID
func generateTemplateID() string {
	return fmt.Sprintf("tpl-%d", time.Now().UnixNano())
}

// Create 创建模板
func (s *templateService) Create(ctx context.Context, req *CreateTemplateRequest) (*template.Template, error) {
	// 1. 构建模板对象
	tpl := &template.Template{
		ID:          generateTemplateID(),
		Name:        req.Name,
		Description: req.Description,
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Nodes:       req.Nodes,
		Edges:       req.Edges,
		Config:      req.Config,
	}

	// 2. 调用 TemplateManager 创建
	if err := s.templateMgr.Create(tpl); err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	// 3. 记录审计日志
	if s.auditLogSvc != nil {
		userID := getUserIDFromContext(ctx)
		if userID != "" {
			details := fmt.Sprintf(`{"template_id":"%s","name":"%s"}`, tpl.ID, tpl.Name)
			_ = s.auditLogSvc.RecordAction(ctx, userID, "create", "template", tpl.ID, details)
		}
	}

	// 4. 设置权限关系（如果有 OpenFGA 客户端和用户ID）
	// 注意: 用户ID需要从 context 中获取，这里暂时跳过
	// 后续在 API 层从 context 获取用户ID并设置权限关系

	return tpl, nil
}

// Get 获取模板（带缓存）
func (s *templateService) Get(id string, version int) (*template.Template, error) {
	// 生成缓存 key
	cacheKey := fmt.Sprintf("%s:%d", id, version)

	// 从缓存获取
	if val, found := s.cache.Load(cacheKey); found {
		entry := val.(*templateCacheEntry)
		if time.Now().Before(entry.expiresAt) {
			// 缓存未过期，直接返回
			return entry.template, nil
		}
		// 缓存已过期，删除
		s.cache.Delete(cacheKey)
	}

	// 缓存未命中或已过期，从数据库查询
	template, err := s.templateMgr.Get(id, version)
	if err != nil {
		return nil, err
	}

	// 写入缓存
	entry := &templateCacheEntry{
		template:  template,
		expiresAt: time.Now().Add(s.cacheTTL),
	}
	s.cache.Store(cacheKey, entry)

	return template, nil
}

// Update 更新模板
func (s *templateService) Update(ctx context.Context, id string, req *UpdateTemplateRequest) (*template.Template, error) {
	// 1. 获取当前模板
	current, err := s.templateMgr.Get(id, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get current template: %w", err)
	}

	// 2. 构建更新后的模板对象
	updated := &template.Template{
		ID:          current.ID,
		Name:        req.Name,
		Description: req.Description,
		Version:     current.Version, // 版本号由 TemplateManager 递增
		CreatedAt:   current.CreatedAt,
		UpdatedAt:   time.Now(),
		Nodes:       req.Nodes,
		Edges:       req.Edges,
		Config:      req.Config,
	}

	// 3. 调用 TemplateManager 更新
	if err := s.templateMgr.Update(id, updated); err != nil {
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	// 4. 清除缓存（更新后版本号变化，需要清除旧版本缓存）
	s.clearTemplateCache(id, 0) // 清除所有版本的缓存

	// 5. 获取更新后的模板
	result, err := s.templateMgr.Get(id, 0)
	if err != nil {
		return nil, err
	}

	// 6. 记录审计日志
	if s.auditLogSvc != nil {
		userID := getUserIDFromContext(ctx)
		if userID != "" {
			details := fmt.Sprintf(`{"template_id":"%s","name":"%s","version":%d}`, result.ID, result.Name, result.Version)
			_ = s.auditLogSvc.RecordAction(ctx, userID, "update", "template", id, details)
		}
	}

	return result, nil
}

// Delete 删除模板
func (s *templateService) Delete(ctx context.Context, id string) error {
	// 1. 获取模板信息（用于审计日志）
	template, _ := s.templateMgr.Get(id, 0)

	// 2. 清除缓存
	s.clearTemplateCache(id, 0) // 清除所有版本的缓存

	// 3. 删除模板
	if err := s.templateMgr.Delete(id); err != nil {
		return err
	}

	// 4. 记录审计日志
	if s.auditLogSvc != nil {
		userID := getUserIDFromContext(ctx)
		if userID != "" {
			name := ""
			if template != nil {
				name = template.Name
			}
			details := fmt.Sprintf(`{"template_id":"%s","name":"%s"}`, id, name)
			_ = s.auditLogSvc.RecordAction(ctx, userID, "delete", "template", id, details)
		}
	}

	return nil
}

// List 查询模板列表
func (s *templateService) List(filter *TemplateListFilter) (*TemplateListResponse, error) {
	if filter == nil {
		filter = &TemplateListFilter{
			Page:     1,
			PageSize: 20,
			SortBy:   "created_at",
			Order:    "desc",
		}
	}

	// 设置默认值
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.SortBy == "" {
		filter.SortBy = "created_at"
	}
	if filter.Order == "" {
		filter.Order = "desc"
	}

	// 构建查询
	query := s.db.Model(&model.TemplateModel{})

	// 搜索条件
	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", searchPattern, searchPattern)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count templates: %w", err)
	}

	// 排序（验证并清理排序字段，防止 SQL 注入）
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	// 验证排序字段
	if err := utils.ValidateSortField(sortBy); err != nil {
		return nil, fmt.Errorf("invalid sort field: %w", err)
	}
	
	order := filter.Order
	if order == "" {
		order = "desc"
	}
	// 验证排序方向
	if err := utils.ValidateSortOrder(order); err != nil {
		return nil, fmt.Errorf("invalid sort order: %w", err)
	}
	query = query.Order(fmt.Sprintf("%s %s", sortBy, strings.ToUpper(order)))

	// 分页
	offset := (filter.Page - 1) * filter.PageSize
	query = query.Offset(offset).Limit(filter.PageSize)

	// 查询数据
	var models []model.TemplateModel
	if err := query.Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to find templates: %w", err)
	}

	// 转换为 Template 对象（直接反序列化，避免 N+1 查询）
	templates := make([]*template.Template, 0, len(models))
	for _, m := range models {
		var tpl template.Template
		if err := json.Unmarshal(m.Data, &tpl); err != nil {
			continue // 跳过无法反序列化的模板
		}
		templates = append(templates, &tpl)
	}

	// 计算总页数
	totalPage := int(total) / filter.PageSize
	if int(total)%filter.PageSize > 0 {
		totalPage++
	}

	return &TemplateListResponse{
		Data: templates,
		Pagination: PaginationInfo{
			Page:      filter.Page,
			PageSize:  filter.PageSize,
			Total:     total,
			TotalPage: totalPage,
		},
	}, nil
}

// ListVersions 列出模板版本
func (s *templateService) ListVersions(id string) ([]int, error) {
	return s.templateMgr.ListVersions(id)
}

// getUserIDFromContext 从 context 中获取用户ID
func getUserIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	// 从 context 中获取用户ID（由认证中间件设置）
	if userID, ok := ctx.Value("user_id").(string); ok {
		return userID
	}
	return ""
}

// clearTemplateCache 清除模板缓存
func (s *templateService) clearTemplateCache(id string, version int) {
	if version > 0 {
		// 清除指定版本的缓存
		cacheKey := fmt.Sprintf("%s:%d", id, version)
		s.cache.Delete(cacheKey)
	} else {
		// 清除所有版本的缓存
		s.cache.Range(func(key, value interface{}) bool {
			keyStr := key.(string)
			if len(keyStr) > len(id) && keyStr[:len(id)] == id && keyStr[len(id)] == ':' {
				s.cache.Delete(key)
			}
			return true
		})
	}
}

