package model

import (
	"errors"
	"time"
)

// AuditLogModel 审计日志数据模型
type AuditLogModel struct {
	ID           string    `gorm:"primaryKey;type:varchar(64)"`
	UserID       string    `gorm:"type:varchar(64);not null;index"`
	Action       string    `gorm:"type:varchar(64);not null;index"` // create/update/delete/approve/reject
	ResourceType string    `gorm:"type:varchar(32);not null"`      // template/task
	ResourceID   string    `gorm:"type:varchar(64);not null;index"`
	RequestID    string    `gorm:"type:varchar(64);index"`
	IP           string    `gorm:"type:varchar(45)"` // IPv4 或 IPv6
	UserAgent    string    `gorm:"type:text"`
	Details      []byte    `gorm:"type:jsonb"` // 操作详情
	CreatedAt    time.Time `gorm:"not null;index"`
}

// TableName 指定表名
func (AuditLogModel) TableName() string {
	return "audit_logs"
}

// Validate 验证审计日志模型
func (alm *AuditLogModel) Validate() error {
	if alm.ID == "" {
		return errors.New("audit log ID is required")
	}
	if alm.UserID == "" {
		return errors.New("user ID is required")
	}
	if alm.Action == "" {
		return errors.New("action is required")
	}
	if alm.ResourceType == "" {
		return errors.New("resource type is required")
	}
	if alm.ResourceID == "" {
		return errors.New("resource ID is required")
	}
	return nil
}

