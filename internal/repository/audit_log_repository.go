package repository

import (
	"github.com/mautops/approval-gin/internal/model"
	"gorm.io/gorm"
)

// AuditLogRepository 审计日志仓储接口
type AuditLogRepository interface {
	Save(log *model.AuditLogModel) error
	FindByUserID(userID string) ([]*model.AuditLogModel, error)
	FindByResource(resourceType string, resourceID string) ([]*model.AuditLogModel, error)
}

// auditLogRepository 审计日志仓储实现
type auditLogRepository struct {
	db *gorm.DB
}

// NewAuditLogRepository 创建审计日志仓储
func NewAuditLogRepository(db *gorm.DB) AuditLogRepository {
	return &auditLogRepository{db: db}
}

// Save 保存审计日志
func (r *auditLogRepository) Save(log *model.AuditLogModel) error {
	return r.db.Save(log).Error
}

// FindByUserID 根据用户 ID 查找审计日志
func (r *auditLogRepository) FindByUserID(userID string) ([]*model.AuditLogModel, error) {
	var logs []*model.AuditLogModel
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&logs).Error
	return logs, err
}

// FindByResource 根据资源查找审计日志
func (r *auditLogRepository) FindByResource(resourceType string, resourceID string) ([]*model.AuditLogModel, error) {
	var logs []*model.AuditLogModel
	err := r.db.Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Order("created_at DESC").
		Find(&logs).Error
	return logs, err
}


