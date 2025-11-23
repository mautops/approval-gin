package repository

import (
	"github.com/mautops/approval-gin/internal/model"
	"gorm.io/gorm"
)

// TemplateRepository 模板仓储接口
type TemplateRepository interface {
	Save(template *model.TemplateModel) error
	FindByID(id string, version int) (*model.TemplateModel, error)
	FindAll() ([]*model.TemplateModel, error)
	Delete(id string) error
}

// templateRepository 模板仓储实现
type templateRepository struct {
	db *gorm.DB
}

// NewTemplateRepository 创建模板仓储
func NewTemplateRepository(db *gorm.DB) TemplateRepository {
	return &templateRepository{db: db}
}

// Save 保存模板
func (r *templateRepository) Save(template *model.TemplateModel) error {
	return r.db.Save(template).Error
}

// FindByID 根据 ID 查找模板
func (r *templateRepository) FindByID(id string, version int) (*model.TemplateModel, error) {
	var template model.TemplateModel
	query := r.db.Where("id = ?", id)

	if version > 0 {
		query = query.Where("version = ?", version)
	} else {
		// 获取最新版本
		query = query.Order("version DESC").Limit(1)
	}

	if err := query.First(&template).Error; err != nil {
		return nil, err
	}

	return &template, nil
}

// FindAll 查找所有模板
func (r *templateRepository) FindAll() ([]*model.TemplateModel, error) {
	var templates []*model.TemplateModel
	err := r.db.Order("created_at DESC").Find(&templates).Error
	return templates, err
}

// Delete 删除模板
func (r *templateRepository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&model.TemplateModel{}).Error
}

