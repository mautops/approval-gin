package integration

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-kit/pkg/template"
	"gorm.io/gorm"
)

// dbTemplateManager 基于数据库的模板管理器
type dbTemplateManager struct {
	db *gorm.DB
}

// NewTemplateManager 创建模板管理器
// 返回 pkg/template.TemplateManager 接口实现
func NewTemplateManager(db *gorm.DB) template.TemplateManager {
	return &dbTemplateManager{db: db}
}

// Create 创建模板
func (m *dbTemplateManager) Create(tpl *template.Template) error {
	// 1. 序列化模板数据
	data, err := json.Marshal(tpl)
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	// 2. 保存到数据库
	model := &model.TemplateModel{
		ID:          tpl.ID,
		Name:        tpl.Name,
		Description: tpl.Description,
		Version:     tpl.Version,
		Data:        data,
		CreatedAt:   tpl.CreatedAt,
		UpdatedAt:   tpl.UpdatedAt,
	}

	return m.db.Create(model).Error
}

// Get 获取模板
func (m *dbTemplateManager) Get(id string, version int) (*template.Template, error) {
	var tm model.TemplateModel
	query := m.db.Where("id = ?", id)

	if version > 0 {
		query = query.Where("version = ?", version)
	} else {
		// 获取最新版本
		query = query.Order("version DESC").Limit(1)
	}

	if err := query.First(&tm).Error; err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}

	// 反序列化
	var tpl template.Template
	if err := json.Unmarshal(tm.Data, &tpl); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template: %w", err)
	}

	return &tpl, nil
}

// Update 更新模板(创建新版本)
func (m *dbTemplateManager) Update(id string, tpl *template.Template) error {
	// 1. 获取当前最新版本
	current, err := m.Get(id, 0)
	if err != nil {
		return fmt.Errorf("failed to get current template: %w", err)
	}

	// 2. 版本号递增
	tpl.Version = current.Version + 1
	tpl.UpdatedAt = time.Now()

	// 3. 保存新版本
	return m.Create(tpl)
}

// Delete 删除模板
func (m *dbTemplateManager) Delete(id string) error {
	return m.db.Where("id = ?", id).Delete(&model.TemplateModel{}).Error
}

// ListVersions 列出模板版本
func (m *dbTemplateManager) ListVersions(id string) ([]int, error) {
	var versions []int
	err := m.db.Model(&model.TemplateModel{}).
		Where("id = ?", id).
		Order("version ASC").
		Pluck("version", &versions).Error
	return versions, err
}
