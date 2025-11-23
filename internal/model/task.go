package model

import (
	"errors"
	"time"
)

// TaskModel 任务数据模型
type TaskModel struct {
	ID             string     `gorm:"primaryKey;type:varchar(64)"`
	TemplateID     string     `gorm:"type:varchar(64);not null;index"`
	TemplateVersion int       `gorm:"type:int;not null"`
	BusinessID     string     `gorm:"type:varchar(64);index"` // 业务 ID
	State          string     `gorm:"type:varchar(32);not null;index"` // 任务状态
	CurrentNode    string     `gorm:"type:varchar(64)"` // 当前节点 ID
	Data           []byte     `gorm:"type:jsonb;not null"` // 序列化后的 Task 对象
	CreatedAt      time.Time  `gorm:"not null;index"`
	UpdatedAt      time.Time  `gorm:"not null;index"`
	SubmittedAt    *time.Time `gorm:"index"` // 提交时间
	CreatedBy      string     `gorm:"type:varchar(64);index"` // 创建人 ID
}

// TableName 指定表名
func (TaskModel) TableName() string {
	return "tasks"
}

// Validate 验证任务模型
func (tm *TaskModel) Validate() error {
	if tm.ID == "" {
		return errors.New("task ID is required")
	}
	if tm.TemplateID == "" {
		return errors.New("template ID is required")
	}
	if tm.State == "" {
		return errors.New("task state is required")
	}
	if len(tm.Data) == 0 {
		return errors.New("task data is required")
	}
	return nil
}

