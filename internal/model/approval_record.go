package model

import (
	"errors"
	"time"
)

// ApprovalRecordModel 审批记录数据模型
type ApprovalRecordModel struct {
	ID          string    `gorm:"primaryKey;type:varchar(64)"`
	TaskID      string    `gorm:"type:varchar(64);not null;index"`
	NodeID      string    `gorm:"type:varchar(64);not null"`
	Approver    string    `gorm:"type:varchar(64);not null;index"`
	Result      string    `gorm:"type:varchar(32);not null"` // approve/reject/transfer
	Comment     string    `gorm:"type:text"`
	Attachments []byte    `gorm:"type:jsonb"` // 附件列表
	CreatedAt   time.Time `gorm:"not null;index"`
}

// TableName 指定表名
func (ApprovalRecordModel) TableName() string {
	return "approval_records"
}

// Validate 验证审批记录模型
func (arm *ApprovalRecordModel) Validate() error {
	if arm.ID == "" {
		return errors.New("record ID is required")
	}
	if arm.TaskID == "" {
		return errors.New("task ID is required")
	}
	if arm.Result == "" {
		return errors.New("approval result is required")
	}
	return nil
}

