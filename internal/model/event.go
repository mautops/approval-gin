package model

import (
	"errors"
	"time"
)

// EventModel 事件数据模型
type EventModel struct {
	ID         string    `gorm:"primaryKey;type:varchar(64)"`
	TaskID     string    `gorm:"type:varchar(64);not null;index"`
	Type       string    `gorm:"type:varchar(32);not null;index"`
	Data       []byte    `gorm:"type:jsonb;not null"` // 序列化后的事件数据
	Status     string    `gorm:"type:varchar(32);not null;default:'pending'"` // pending/success/failed
	RetryCount int       `gorm:"type:int;default:0"`
	CreatedAt  time.Time `gorm:"not null;index"`
	UpdatedAt  time.Time `gorm:"not null"`
}

// TableName 指定表名
func (EventModel) TableName() string {
	return "events"
}

// Validate 验证事件模型
func (em *EventModel) Validate() error {
	if em.ID == "" {
		return errors.New("event ID is required")
	}
	if em.TaskID == "" {
		return errors.New("task ID is required")
	}
	if em.Type == "" {
		return errors.New("event type is required")
	}
	if len(em.Data) == 0 {
		return errors.New("event data is required")
	}
	if em.Status == "" {
		em.Status = "pending"
	}
	return nil
}

