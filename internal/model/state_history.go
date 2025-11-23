package model

import (
	"errors"
	"time"
)

// StateHistoryModel 状态变更历史数据模型
type StateHistoryModel struct {
	ID        string    `gorm:"primaryKey;type:varchar(64)"`
	TaskID    string    `gorm:"type:varchar(64);not null;index"`
	FromState string    `gorm:"type:varchar(32)"`
	ToState   string    `gorm:"type:varchar(32);not null"`
	Reason    string    `gorm:"type:text"`
	Operator  string    `gorm:"type:varchar(64);not null"`
	CreatedAt time.Time `gorm:"not null;index"`
}

// TableName 指定表名
func (StateHistoryModel) TableName() string {
	return "state_history"
}

// Validate 验证状态历史模型
func (shm *StateHistoryModel) Validate() error {
	if shm.ID == "" {
		return errors.New("history ID is required")
	}
	if shm.TaskID == "" {
		return errors.New("task ID is required")
	}
	if shm.ToState == "" {
		return errors.New("to state is required")
	}
	if shm.Operator == "" {
		return errors.New("operator is required")
	}
	return nil
}

