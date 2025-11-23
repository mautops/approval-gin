package repository

import (
	"github.com/mautops/approval-gin/internal/model"
	"gorm.io/gorm"
)

// StateHistoryRepository 状态历史仓储接口
type StateHistoryRepository interface {
	Save(history *model.StateHistoryModel) error
	FindByTaskID(taskID string) ([]*model.StateHistoryModel, error)
}

// stateHistoryRepository 状态历史仓储实现
type stateHistoryRepository struct {
	db *gorm.DB
}

// NewStateHistoryRepository 创建状态历史仓储
func NewStateHistoryRepository(db *gorm.DB) StateHistoryRepository {
	return &stateHistoryRepository{db: db}
}

// Save 保存状态历史
func (r *stateHistoryRepository) Save(history *model.StateHistoryModel) error {
	return r.db.Save(history).Error
}

// FindByTaskID 根据任务 ID 查找状态历史
func (r *stateHistoryRepository) FindByTaskID(taskID string) ([]*model.StateHistoryModel, error) {
	var histories []*model.StateHistoryModel
	err := r.db.Where("task_id = ?", taskID).Order("created_at ASC").Find(&histories).Error
	return histories, err
}


