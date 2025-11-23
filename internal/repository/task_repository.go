package repository

import (
	"github.com/mautops/approval-gin/internal/model"
	"gorm.io/gorm"
)

// TaskRepository 任务仓储接口
type TaskRepository interface {
	Save(task *model.TaskModel) error
	FindByID(id string) (*model.TaskModel, error)
	FindAll() ([]*model.TaskModel, error)
	FindByFilter(filter *TaskFilter) ([]*model.TaskModel, error)
}

// TaskFilter 任务查询过滤器
type TaskFilter struct {
	State      *string
	TemplateID *string
	BusinessID *string
	Approver   *string
	StartTime  *string
	EndTime    *string
}

// taskRepository 任务仓储实现
type taskRepository struct {
	db *gorm.DB
}

// NewTaskRepository 创建任务仓储
func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepository{db: db}
}

// Save 保存任务
func (r *taskRepository) Save(task *model.TaskModel) error {
	return r.db.Save(task).Error
}

// FindByID 根据 ID 查找任务
func (r *taskRepository) FindByID(id string) (*model.TaskModel, error) {
	var task model.TaskModel
	if err := r.db.Where("id = ?", id).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// FindAll 查找所有任务
func (r *taskRepository) FindAll() ([]*model.TaskModel, error) {
	var tasks []*model.TaskModel
	err := r.db.Order("created_at DESC").Find(&tasks).Error
	return tasks, err
}

// FindByFilter 根据过滤器查找任务
func (r *taskRepository) FindByFilter(filter *TaskFilter) ([]*model.TaskModel, error) {
	var tasks []*model.TaskModel
	query := r.db.Model(&model.TaskModel{})

	if filter != nil {
		if filter.State != nil {
			query = query.Where("state = ?", *filter.State)
		}
		if filter.TemplateID != nil {
			query = query.Where("template_id = ?", *filter.TemplateID)
		}
		if filter.BusinessID != nil {
			query = query.Where("business_id = ?", *filter.BusinessID)
		}
		if filter.StartTime != nil {
			query = query.Where("created_at >= ?", *filter.StartTime)
		}
		if filter.EndTime != nil {
			query = query.Where("created_at <= ?", *filter.EndTime)
		}
	}

	err := query.Order("created_at DESC").Find(&tasks).Error
	return tasks, err
}


