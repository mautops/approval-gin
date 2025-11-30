package integration

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-kit/pkg/template"
	"gorm.io/gorm"
)

// DBTemplateManager 基于数据库的模板管理器(导出以便服务层调用)
type DBTemplateManager struct {
	db *gorm.DB
}

// dbTemplateManager 基于数据库的模板管理器(内部别名)
type dbTemplateManager = DBTemplateManager

// NewTemplateManager 创建模板管理器
// 返回 pkg/template.TemplateManager 接口实现
func NewTemplateManager(db *gorm.DB) template.TemplateManager {
	return &DBTemplateManager{db: db}
}

// Create 创建模板
func (m *DBTemplateManager) Create(tpl *template.Template) error {
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

func (m *DBTemplateManager) CreateWithNodePositions(tpl *template.Template, rawNodesJSON json.RawMessage) error {
	data, err := json.Marshal(tpl)
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	if len(rawNodesJSON) > 0 {
		var templateMap map[string]interface{}
		if err := json.Unmarshal(data, &templateMap); err != nil {
			return fmt.Errorf("failed to unmarshal template: %w", err)
		}

		var rawNodesMap map[string]interface{}
		if err := json.Unmarshal(rawNodesJSON, &rawNodesMap); err != nil {
			return fmt.Errorf("failed to unmarshal raw nodes: %w", err)
		}

		if nodes, ok := templateMap["nodes"].(map[string]interface{}); ok {
			for nodeID, rawNode := range rawNodesMap {
				if rawNodeMap, ok := rawNode.(map[string]interface{}); ok {
					if position, ok := rawNodeMap["position"]; ok {
						if node, exists := nodes[nodeID]; exists {
							if nodeMap, ok := node.(map[string]interface{}); ok {
								nodeMap["position"] = position
							}
						}
					}
				}
			}
		}

		data, err = json.Marshal(templateMap)
		if err != nil {
			return fmt.Errorf("failed to remarshal template: %w", err)
		}
	}

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

func (m *DBTemplateManager) UpdateWithNodePositions(id string, tpl *template.Template, rawNodesJSON json.RawMessage) error {
	current, err := m.Get(id, 0)
	if err != nil {
		return fmt.Errorf("failed to get current template: %w", err)
	}

	tpl.Version = current.Version + 1
	tpl.UpdatedAt = time.Now()

	return m.CreateWithNodePositions(tpl, rawNodesJSON)
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

// DeleteVersion 删除指定版本的模板
func (m *dbTemplateManager) DeleteVersion(id string, version int) error {
	// 检查是否存在该版本
	var count int64
	if err := m.db.Model(&model.TemplateModel{}).
		Where("id = ? AND version = ?", id, version).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check template version: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("template version not found")
	}

	// 检查是否还有其他版本
	var totalCount int64
	if err := m.db.Model(&model.TemplateModel{}).
		Where("id = ?", id).
		Count(&totalCount).Error; err != nil {
		return fmt.Errorf("failed to count template versions: %w", err)
	}
	if totalCount <= 1 {
		return fmt.Errorf("cannot delete the last version of template")
	}

	// 删除指定版本
	return m.db.Where("id = ? AND version = ?", id, version).Delete(&model.TemplateModel{}).Error
}
