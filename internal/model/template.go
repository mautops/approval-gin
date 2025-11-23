package model

import (
	"errors"
	"time"
)

// TemplateModel 模板数据模型
type TemplateModel struct {
	ID          string    `gorm:"primaryKey;type:varchar(64)"`
	Version     int       `gorm:"primaryKey;type:int;not null;default:1"` // 主键组合 (id, version)
	Name        string    `gorm:"type:varchar(255);not null"`
	Description string    `gorm:"type:text"`
	Data        []byte    `gorm:"type:jsonb;not null"` // 序列化后的 Template 对象
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
	CreatedBy   string    `gorm:"type:varchar(64)"` // 创建人 ID
	UpdatedBy   string    `gorm:"type:varchar(64)"` // 更新人 ID
}

// TableName 指定表名
func (TemplateModel) TableName() string {
	return "templates"
}

// Validate 验证模板模型
func (tm *TemplateModel) Validate() error {
	if tm.ID == "" {
		return errors.New("template ID is required")
	}
	if tm.Name == "" {
		return errors.New("template name is required")
	}
	if len(tm.Data) == 0 {
		return errors.New("template data is required")
	}
	return nil
}

