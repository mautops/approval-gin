package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-gin/internal/repository"
	"github.com/mautops/approval-gin/internal/utils"
	"github.com/mautops/approval-kit/pkg/task"
	"github.com/mautops/approval-kit/pkg/types"
	"gorm.io/gorm"
)

// QueryService 查询服务接口
type QueryService interface {
	ListTasks(filter *ListTasksFilter) ([]*task.Task, int64, error)
	GetRecords(taskID string) ([]*ApprovalRecord, error)
	GetHistory(taskID string) ([]*StateHistory, error)
}

// ListTasksFilter 任务列表查询过滤器
type ListTasksFilter struct {
	State      *types.TaskState
	TemplateID *string
	BusinessID *string
	Approver   *string
	StartTime  *string
	EndTime    *string
	Page       int
	PageSize   int
	SortBy     string
	Order      string
}

// ApprovalRecord 审批记录
type ApprovalRecord struct {
	ID          string
	TaskID      string
	NodeID      string
	Approver    string
	Result      string
	Comment     string
	Attachments []string
	CreatedAt   string
}

// StateHistory 状态历史
type StateHistory struct {
	ID        string
	TaskID    string
	FromState string
	ToState   string
	Reason    string
	Operator  string
	CreatedAt string
}

// queryService 查询服务实现
type queryService struct {
	db         *gorm.DB
	taskMgr    task.TaskManager
	recordRepo repository.ApprovalRecordRepository
	historyRepo repository.StateHistoryRepository
}

// NewQueryService 创建查询服务
func NewQueryService(db *gorm.DB, taskMgr task.TaskManager) QueryService {
	return &queryService{
		db:         db,
		taskMgr:    taskMgr,
		recordRepo: repository.NewApprovalRecordRepository(db),
		historyRepo: repository.NewStateHistoryRepository(db),
	}
}

// ListTasks 列出任务
func (s *queryService) ListTasks(filter *ListTasksFilter) ([]*task.Task, int64, error) {
	// 构建查询
	query := s.db.Model(&model.TaskModel{})

	// 应用过滤条件
	if filter.State != nil {
		query = query.Where("state = ?", string(*filter.State))
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

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	// 应用排序（验证并清理排序字段，防止 SQL 注入）
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	// 验证排序字段
	if err := utils.ValidateSortField(sortBy); err != nil {
		return nil, 0, fmt.Errorf("invalid sort field: %w", err)
	}
	
	order := filter.Order
	if order == "" {
		order = "desc"
	}
	// 验证排序方向
	if err := utils.ValidateSortOrder(order); err != nil {
		return nil, 0, fmt.Errorf("invalid sort order: %w", err)
	}
	query = query.Order(fmt.Sprintf("%s %s", sortBy, strings.ToUpper(order)))

	// 应用分页
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	query = query.Offset((page - 1) * pageSize).Limit(pageSize)

	// 执行查询
	var models []model.TaskModel
	if err := query.Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to query tasks: %w", err)
	}

	// 转换为 Task 对象（直接反序列化，避免 N+1 查询）
	tasks := make([]*task.Task, 0, len(models))
	for _, tm := range models {
		var tsk task.Task
		if err := json.Unmarshal(tm.Data, &tsk); err != nil {
			continue // 跳过无法反序列化的任务
		}
		tasks = append(tasks, &tsk)
	}

	return tasks, total, nil
}

// GetRecords 获取审批记录
func (s *queryService) GetRecords(taskID string) ([]*ApprovalRecord, error) {
	models, err := s.recordRepo.FindByTaskID(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get records: %w", err)
	}

	records := make([]*ApprovalRecord, 0, len(models))
	for _, m := range models {
		var attachments []string
		if len(m.Attachments) > 0 {
			_ = json.Unmarshal(m.Attachments, &attachments)
		}
		records = append(records, &ApprovalRecord{
			ID:          m.ID,
			TaskID:      m.TaskID,
			NodeID:      m.NodeID,
			Approver:    m.Approver,
			Result:      m.Result,
			Comment:     m.Comment,
			Attachments: attachments,
			CreatedAt:   m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return records, nil
}

// GetHistory 获取状态历史
func (s *queryService) GetHistory(taskID string) ([]*StateHistory, error) {
	models, err := s.historyRepo.FindByTaskID(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}

	histories := make([]*StateHistory, 0, len(models))
	for _, m := range models {
		histories = append(histories, &StateHistory{
			ID:        m.ID,
			TaskID:    m.TaskID,
			FromState: m.FromState,
			ToState:   m.ToState,
			Reason:    m.Reason,
			Operator:  m.Operator,
			CreatedAt: m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return histories, nil
}

