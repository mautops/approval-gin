package service

import (
	"fmt"

	"github.com/mautops/approval-gin/internal/model"
	"gorm.io/gorm"
)

// StatisticsService 统计服务接口
type StatisticsService interface {
	GetTaskStatisticsByState() ([]*TaskStatisticsByState, error)
	GetTaskStatisticsByTemplate() ([]*TaskStatisticsByTemplate, error)
	GetTaskStatisticsByTime() ([]*TaskStatisticsByTime, error)
	GetApprovalStatistics() (*ApprovalStatistics, error)
}

// TaskStatisticsByState 按状态统计
type TaskStatisticsByState struct {
	State string
	Count int64
}

// TaskStatisticsByTemplate 按模板统计
type TaskStatisticsByTemplate struct {
	TemplateID   string
	TemplateName string
	Count        int64
}

// TaskStatisticsByTime 按时间统计
type TaskStatisticsByTime struct {
	Date  string
	Count int64
}

// ApprovalStatistics 审批统计
type ApprovalStatistics struct {
	TotalApprovals    int64
	ApprovedCount     int64
	RejectedCount     int64
	ApprovalRate      float64
	AverageApprovalTime float64 // 单位：秒
}

// statisticsService 统计服务实现
type statisticsService struct {
	db *gorm.DB
}

// NewStatisticsService 创建统计服务
func NewStatisticsService(db *gorm.DB) StatisticsService {
	return &statisticsService{db: db}
}

// GetTaskStatisticsByState 按状态统计任务
func (s *statisticsService) GetTaskStatisticsByState() ([]*TaskStatisticsByState, error) {
	var results []struct {
		State string
		Count int64
	}

	err := s.db.Model(&model.TaskModel{}).
		Select("state, COUNT(*) as count").
		Group("state").
		Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get task statistics by state: %w", err)
	}

	stats := make([]*TaskStatisticsByState, 0, len(results))
	for _, r := range results {
		stats = append(stats, &TaskStatisticsByState{
			State: r.State,
			Count: r.Count,
		})
	}

	return stats, nil
}

// GetTaskStatisticsByTemplate 按模板统计任务
func (s *statisticsService) GetTaskStatisticsByTemplate() ([]*TaskStatisticsByTemplate, error) {
	var results []struct {
		TemplateID   string
		TemplateName string
		Count        int64
	}

	err := s.db.Model(&model.TaskModel{}).
		Select("template_id, COUNT(*) as count").
		Group("template_id").
		Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get task statistics by template: %w", err)
	}

	// 获取模板名称
	stats := make([]*TaskStatisticsByTemplate, 0, len(results))
	for _, r := range results {
		var template model.TemplateModel
		if err := s.db.Where("id = ?", r.TemplateID).
			Order("version DESC").
			First(&template).Error; err == nil {
			// 解析模板名称（从 Data 字段中提取，简化处理）
			stats = append(stats, &TaskStatisticsByTemplate{
				TemplateID:   r.TemplateID,
				TemplateName: template.Name,
				Count:        r.Count,
			})
		} else {
			stats = append(stats, &TaskStatisticsByTemplate{
				TemplateID:   r.TemplateID,
				TemplateName: "未知模板",
				Count:        r.Count,
			})
		}
	}

	return stats, nil
}

// GetTaskStatisticsByTime 按时间统计任务
func (s *statisticsService) GetTaskStatisticsByTime() ([]*TaskStatisticsByTime, error) {
	var results []struct {
		Date  string
		Count int64
	}

	err := s.db.Model(&model.TaskModel{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Group("DATE(created_at)").
		Order("date DESC").
		Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get task statistics by time: %w", err)
	}

	stats := make([]*TaskStatisticsByTime, 0, len(results))
	for _, r := range results {
		stats = append(stats, &TaskStatisticsByTime{
			Date:  r.Date,
			Count: r.Count,
		})
	}

	return stats, nil
}

// GetApprovalStatistics 获取审批统计
func (s *statisticsService) GetApprovalStatistics() (*ApprovalStatistics, error) {
	var totalCount int64
	err := s.db.Model(&model.ApprovalRecordModel{}).Count(&totalCount).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count approval records: %w", err)
	}

	var approvedCount int64
	err = s.db.Model(&model.ApprovalRecordModel{}).
		Where("result = ?", "approve").
		Count(&approvedCount).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count approved records: %w", err)
	}

	var rejectedCount int64
	err = s.db.Model(&model.ApprovalRecordModel{}).
		Where("result = ?", "reject").
		Count(&rejectedCount).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count rejected records: %w", err)
	}

	approvalRate := 0.0
	if totalCount > 0 {
		approvalRate = float64(approvedCount) / float64(totalCount) * 100
	}

	// TODO: 计算平均审批时间（需要从审批记录中计算时间差）
	averageApprovalTime := 0.0

	return &ApprovalStatistics{
		TotalApprovals:      totalCount,
		ApprovedCount:       approvedCount,
		RejectedCount:        rejectedCount,
		ApprovalRate:         approvalRate,
		AverageApprovalTime: averageApprovalTime,
	}, nil
}


