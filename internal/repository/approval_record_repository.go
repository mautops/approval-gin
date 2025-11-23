package repository

import (
	"github.com/mautops/approval-gin/internal/model"
	"gorm.io/gorm"
)

// ApprovalRecordRepository 审批记录仓储接口
type ApprovalRecordRepository interface {
	Save(record *model.ApprovalRecordModel) error
	FindByTaskID(taskID string) ([]*model.ApprovalRecordModel, error)
	FindByApprover(approver string) ([]*model.ApprovalRecordModel, error)
}

// approvalRecordRepository 审批记录仓储实现
type approvalRecordRepository struct {
	db *gorm.DB
}

// NewApprovalRecordRepository 创建审批记录仓储
func NewApprovalRecordRepository(db *gorm.DB) ApprovalRecordRepository {
	return &approvalRecordRepository{db: db}
}

// Save 保存审批记录
func (r *approvalRecordRepository) Save(record *model.ApprovalRecordModel) error {
	return r.db.Save(record).Error
}

// FindByTaskID 根据任务 ID 查找审批记录
func (r *approvalRecordRepository) FindByTaskID(taskID string) ([]*model.ApprovalRecordModel, error) {
	var records []*model.ApprovalRecordModel
	err := r.db.Where("task_id = ?", taskID).Order("created_at ASC").Find(&records).Error
	return records, err
}

// FindByApprover 根据审批人查找审批记录
func (r *approvalRecordRepository) FindByApprover(approver string) ([]*model.ApprovalRecordModel, error) {
	var records []*model.ApprovalRecordModel
	err := r.db.Where("approver = ?", approver).Order("created_at DESC").Find(&records).Error
	return records, err
}


