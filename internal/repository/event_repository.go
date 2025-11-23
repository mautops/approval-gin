package repository

import (
	"github.com/mautops/approval-gin/internal/model"
	"gorm.io/gorm"
)

// EventRepository 事件仓储接口
type EventRepository interface {
	Save(event *model.EventModel) error
	FindByTaskID(taskID string) ([]*model.EventModel, error)
	FindPending() ([]*model.EventModel, error)
}

// eventRepository 事件仓储实现
type eventRepository struct {
	db *gorm.DB
}

// NewEventRepository 创建事件仓储
func NewEventRepository(db *gorm.DB) EventRepository {
	return &eventRepository{db: db}
}

// Save 保存事件
func (r *eventRepository) Save(event *model.EventModel) error {
	return r.db.Save(event).Error
}

// FindByTaskID 根据任务 ID 查找事件
func (r *eventRepository) FindByTaskID(taskID string) ([]*model.EventModel, error) {
	var events []*model.EventModel
	err := r.db.Where("task_id = ?", taskID).Order("created_at ASC").Find(&events).Error
	return events, err
}

// FindPending 查找待处理的事件
func (r *eventRepository) FindPending() ([]*model.EventModel, error) {
	var events []*model.EventModel
	err := r.db.Where("status = ?", "pending").Order("created_at ASC").Find(&events).Error
	return events, err
}


