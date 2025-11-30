package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mautops/approval-gin/internal/auth"
	"github.com/mautops/approval-gin/internal/metrics"
	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-kit/pkg/task"
	"gorm.io/gorm"
)

// TaskService 任务服务接口
type TaskService interface {
	Create(ctx context.Context, req *CreateTaskRequest) (*task.Task, error)
	Get(id string) (*task.Task, error)
	Submit(ctx context.Context, id string) error
	Approve(ctx context.Context, id string, req *ApproveRequest) error
	Reject(ctx context.Context, id string, req *RejectRequest) error
	Cancel(ctx context.Context, id string, reason string) error
	Withdraw(ctx context.Context, id string, reason string) error
	// 高级操作方法
	Transfer(ctx context.Context, id string, req *TransferRequest) error
	AddApprover(ctx context.Context, id string, req *AddApproverRequest) error
	RemoveApprover(ctx context.Context, id string, req *RemoveApproverRequest) error
	Pause(ctx context.Context, id string, reason string) error
	Resume(ctx context.Context, id string, reason string) error
	RollbackToNode(ctx context.Context, id string, req *RollbackRequest) error
	ReplaceApprover(ctx context.Context, id string, req *ReplaceApproverRequest) error
	HandleTimeout(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
	// 批量操作方法
	BatchApprove(ctx context.Context, req *BatchApproveRequest) ([]BatchOperationResult, error)
	BatchTransfer(ctx context.Context, req *BatchTransferRequest) ([]BatchOperationResult, error)
}

// CreateTaskRequest 创建任务请求
// @Description 创建审批任务的请求参数
type CreateTaskRequest struct {
	TemplateID string `json:"template_id" example:"tpl-001" binding:"required"` // 模板 ID
	BusinessID string `json:"business_id" example:"biz-001" binding:"required"` // 业务 ID
	Params     json.RawMessage `json:"params" swaggertype:"object" example:"{\"amount\":1000}"` // 任务参数(JSON 格式)
}

// ApproveRequest 审批同意请求
// @Description 审批同意的请求参数
type ApproveRequest struct {
	NodeID      string `json:"node_id" example:"node-001" binding:"required"` // 节点 ID
	Comment     string `json:"comment" example:"同意"` // 审批意见
	Attachments []string `json:"attachments" example:"[\"file1.pdf\",\"file2.pdf\"]"` // 附件列表
}

// RejectRequest 审批拒绝请求
// @Description 审批拒绝的请求参数
type RejectRequest struct {
	NodeID      string `json:"node_id" example:"node-001" binding:"required"` // 节点 ID
	Comment     string `json:"comment" example:"拒绝"` // 审批意见
	Attachments []string `json:"attachments" example:"[\"file1.pdf\",\"file2.pdf\"]"` // 附件列表
}

// TransferRequest 转交请求
// @Description 转交审批的请求参数
type TransferRequest struct {
	NodeID      string `json:"node_id" example:"node-001" binding:"required"` // 节点 ID
	ToApprover  string `json:"to_approver" example:"user-002" binding:"required"` // 新审批人 ID
	Reason      string `json:"reason" example:"转交原因"` // 转交原因
}

// AddApproverRequest 加签请求
// @Description 加签的请求参数
type AddApproverRequest struct {
	NodeID   string `json:"node_id" example:"node-001" binding:"required"` // 节点 ID
	Approver string `json:"approver" example:"user-003" binding:"required"` // 新审批人 ID
	Reason   string `json:"reason" example:"加签原因"` // 加签原因
}

// RemoveApproverRequest 减签请求
// @Description 减签的请求参数
type RemoveApproverRequest struct {
	NodeID   string `json:"node_id" example:"node-001" binding:"required"` // 节点 ID
	Approver string `json:"approver" example:"user-003" binding:"required"` // 要移除的审批人 ID
	Reason   string `json:"reason" example:"减签原因"` // 减签原因
}

// RollbackRequest 回退请求
// @Description 回退到指定节点的请求参数
type RollbackRequest struct {
	NodeID string `json:"node_id" example:"node-001" binding:"required"` // 目标节点 ID
	Reason string `json:"reason" example:"回退原因"` // 回退原因
}

// ReplaceApproverRequest 替换审批人请求
// @Description 替换审批人的请求参数
type ReplaceApproverRequest struct {
	NodeID      string `json:"node_id" example:"node-001" binding:"required"` // 节点 ID
	OldApprover string `json:"old_approver" example:"user-001" binding:"required"` // 原审批人 ID
	NewApprover string `json:"new_approver" example:"user-002" binding:"required"` // 新审批人 ID
	Reason      string `json:"reason" example:"替换原因"` // 替换原因
}

// BatchApproveRequest 批量审批请求
// @Description 批量审批的请求参数
type BatchApproveRequest struct {
	TaskIDs []string `json:"task_ids" binding:"required"` // 任务 ID 列表
	NodeID  string   `json:"node_id" binding:"required"` // 节点 ID
	Comment string   `json:"comment"` // 审批意见
}

// BatchTransferRequest 批量转交请求
// @Description 批量转交的请求参数
type BatchTransferRequest struct {
	TaskIDs     []string `json:"task_ids" binding:"required"` // 任务 ID 列表
	NodeID      string   `json:"node_id" binding:"required"` // 节点 ID
	OldApprover string   `json:"old_approver" binding:"required"` // 原审批人 ID
	NewApprover string   `json:"new_approver" binding:"required"` // 新审批人 ID
	Comment     string   `json:"comment"` // 转交原因
}

// BatchOperationResult 批量操作结果
// @Description 批量操作的结果
type BatchOperationResult struct {
	TaskID  string `json:"task_id"` // 任务 ID
	Success bool   `json:"success"` // 是否成功
	Error   string `json:"error,omitempty"` // 错误信息(如果失败)
}

type taskService struct {
	taskMgr    task.TaskManager
	db         *gorm.DB
	fgaClient  *auth.OpenFGAClient
	auditLogSvc AuditLogService
}

// NewTaskService 创建任务服务
func NewTaskService(taskMgr task.TaskManager, db *gorm.DB, auditLogSvc AuditLogService, fgaClient ...*auth.OpenFGAClient) TaskService {
	var fga *auth.OpenFGAClient
	if len(fgaClient) > 0 && fgaClient[0] != nil {
		fga = fgaClient[0]
	}
	return &taskService{
		taskMgr:     taskMgr,
		db:          db,
		fgaClient:   fga,
		auditLogSvc: auditLogSvc,
	}
}

// Create 创建任务
func (s *taskService) Create(ctx context.Context, req *CreateTaskRequest) (*task.Task, error) {
	// 调用 TaskManager 创建任务
	task, err := s.taskMgr.Create(req.TemplateID, req.BusinessID, req.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// 记录业务指标
	metrics.RecordTaskCreated()

	// 记录审计日志
	if s.auditLogSvc != nil {
		userID := getUserIDFromContext(ctx)
		if userID != "" {
			details := fmt.Sprintf(`{"task_id":"%s","template_id":"%s","business_id":"%s"}`, task.ID, task.TemplateID, task.BusinessID)
			_ = s.auditLogSvc.RecordAction(ctx, userID, "create", "task", task.ID, details)
		}
	}

	return task, nil
}

// Get 获取任务详情
func (s *taskService) Get(id string) (*task.Task, error) {
	return s.taskMgr.Get(id)
}

// Submit 提交任务
func (s *taskService) Submit(ctx context.Context, id string) error {
	if err := s.taskMgr.Submit(id); err != nil {
		return err
	}

	// 记录审计日志
	if s.auditLogSvc != nil {
		userID := getUserIDFromContext(ctx)
		if userID != "" {
			details := fmt.Sprintf(`{"task_id":"%s"}`, id)
			_ = s.auditLogSvc.RecordAction(ctx, userID, "submit", "task", id, details)
		}
	}

	return nil
}

// Approve 审批同意
func (s *taskService) Approve(ctx context.Context, id string, req *ApproveRequest) error {
	// 根据是否有附件选择不同的方法
	if len(req.Attachments) > 0 {
		if err := s.taskMgr.ApproveWithAttachments(id, req.NodeID, getUserIDFromContext(ctx), req.Comment, req.Attachments); err != nil {
			return err
		}
	} else {
		if err := s.taskMgr.Approve(id, req.NodeID, getUserIDFromContext(ctx), req.Comment); err != nil {
			return err
		}
	}

	// 记录审计日志
	if s.auditLogSvc != nil {
		userID := getUserIDFromContext(ctx)
		if userID != "" {
			details := fmt.Sprintf(`{"task_id":"%s","node_id":"%s","comment":"%s"}`, id, req.NodeID, req.Comment)
			_ = s.auditLogSvc.RecordAction(ctx, userID, "approve", "task", id, details)
		}
	}

	return nil
}

// Reject 审批拒绝
func (s *taskService) Reject(ctx context.Context, id string, req *RejectRequest) error {
	// 根据是否有附件选择不同的方法
	if len(req.Attachments) > 0 {
		if err := s.taskMgr.RejectWithAttachments(id, req.NodeID, getUserIDFromContext(ctx), req.Comment, req.Attachments); err != nil {
			return err
		}
	} else {
		if err := s.taskMgr.Reject(id, req.NodeID, getUserIDFromContext(ctx), req.Comment); err != nil {
			return err
		}
	}

	// 记录审计日志
	if s.auditLogSvc != nil {
		userID := getUserIDFromContext(ctx)
		if userID != "" {
			details := fmt.Sprintf(`{"task_id":"%s","node_id":"%s","comment":"%s"}`, id, req.NodeID, req.Comment)
			_ = s.auditLogSvc.RecordAction(ctx, userID, "reject", "task", id, details)
		}
	}

	return nil
}

// Cancel 取消任务
func (s *taskService) Cancel(ctx context.Context, id string, reason string) error {
	if err := s.taskMgr.Cancel(id, reason); err != nil {
		return err
	}

	// 记录审计日志
	if s.auditLogSvc != nil {
		userID := getUserIDFromContext(ctx)
		if userID != "" {
			details := fmt.Sprintf(`{"task_id":"%s","reason":"%s"}`, id, reason)
			_ = s.auditLogSvc.RecordAction(ctx, userID, "cancel", "task", id, details)
		}
	}

	return nil
}

// Withdraw 撤回任务
func (s *taskService) Withdraw(ctx context.Context, id string, reason string) error {
	if err := s.taskMgr.Withdraw(id, reason); err != nil {
		return err
	}

	// 记录审计日志
	if s.auditLogSvc != nil {
		userID := getUserIDFromContext(ctx)
		if userID != "" {
			details := fmt.Sprintf(`{"task_id":"%s","reason":"%s"}`, id, reason)
			_ = s.auditLogSvc.RecordAction(ctx, userID, "withdraw", "task", id, details)
		}
	}

	return nil
}

// Transfer 转交审批
func (s *taskService) Transfer(ctx context.Context, id string, req *TransferRequest) error {
	userID := getUserIDFromContext(ctx)
	if err := s.taskMgr.Transfer(id, req.NodeID, userID, req.ToApprover, req.Reason); err != nil {
		return err
	}

	// 记录审计日志
	if s.auditLogSvc != nil {
		if userID != "" {
			details := fmt.Sprintf(`{"task_id":"%s","node_id":"%s","to_approver":"%s","reason":"%s"}`, id, req.NodeID, req.ToApprover, req.Reason)
			_ = s.auditLogSvc.RecordAction(ctx, userID, "transfer", "task", id, details)
		}
	}

	return nil
}

// AddApprover 加签
func (s *taskService) AddApprover(ctx context.Context, id string, req *AddApproverRequest) error {
	if err := s.taskMgr.AddApprover(id, req.NodeID, req.Approver, req.Reason); err != nil {
		return err
	}

	// 记录审计日志
	if s.auditLogSvc != nil {
		userID := getUserIDFromContext(ctx)
		if userID != "" {
			details := fmt.Sprintf(`{"task_id":"%s","node_id":"%s","approver":"%s","reason":"%s"}`, id, req.NodeID, req.Approver, req.Reason)
			_ = s.auditLogSvc.RecordAction(ctx, userID, "add_approver", "task", id, details)
		}
	}

	return nil
}

// RemoveApprover 减签
func (s *taskService) RemoveApprover(ctx context.Context, id string, req *RemoveApproverRequest) error {
	if err := s.taskMgr.RemoveApprover(id, req.NodeID, req.Approver, req.Reason); err != nil {
		return err
	}

	// 记录审计日志
	if s.auditLogSvc != nil {
		userID := getUserIDFromContext(ctx)
		if userID != "" {
			details := fmt.Sprintf(`{"task_id":"%s","node_id":"%s","approver":"%s","reason":"%s"}`, id, req.NodeID, req.Approver, req.Reason)
			_ = s.auditLogSvc.RecordAction(ctx, userID, "remove_approver", "task", id, details)
		}
	}

	return nil
}

// Pause 暂停任务
func (s *taskService) Pause(ctx context.Context, id string, reason string) error {
	if err := s.taskMgr.Pause(id, reason); err != nil {
		return err
	}

	// 记录审计日志
	if s.auditLogSvc != nil {
		userID := getUserIDFromContext(ctx)
		if userID != "" {
			details := fmt.Sprintf(`{"task_id":"%s","reason":"%s"}`, id, reason)
			_ = s.auditLogSvc.RecordAction(ctx, userID, "pause", "task", id, details)
		}
	}

	return nil
}

// Resume 恢复任务
func (s *taskService) Resume(ctx context.Context, id string, reason string) error {
	if err := s.taskMgr.Resume(id, reason); err != nil {
		return err
	}

	// 记录审计日志
	if s.auditLogSvc != nil {
		userID := getUserIDFromContext(ctx)
		if userID != "" {
			details := fmt.Sprintf(`{"task_id":"%s","reason":"%s"}`, id, reason)
			_ = s.auditLogSvc.RecordAction(ctx, userID, "resume", "task", id, details)
		}
	}

	return nil
}

// RollbackToNode 回退到指定节点
func (s *taskService) RollbackToNode(ctx context.Context, id string, req *RollbackRequest) error {
	if err := s.taskMgr.RollbackToNode(id, req.NodeID, req.Reason); err != nil {
		return err
	}

	// 记录审计日志
	if s.auditLogSvc != nil {
		userID := getUserIDFromContext(ctx)
		if userID != "" {
			details := fmt.Sprintf(`{"task_id":"%s","node_id":"%s","reason":"%s"}`, id, req.NodeID, req.Reason)
			_ = s.auditLogSvc.RecordAction(ctx, userID, "rollback", "task", id, details)
		}
	}

	return nil
}

// ReplaceApprover 替换审批人
func (s *taskService) ReplaceApprover(ctx context.Context, id string, req *ReplaceApproverRequest) error {
	if err := s.taskMgr.ReplaceApprover(id, req.NodeID, req.OldApprover, req.NewApprover, req.Reason); err != nil {
		return err
	}

	// 记录审计日志
	if s.auditLogSvc != nil {
		userID := getUserIDFromContext(ctx)
		if userID != "" {
			details := fmt.Sprintf(`{"task_id":"%s","node_id":"%s","old_approver":"%s","new_approver":"%s","reason":"%s"}`, id, req.NodeID, req.OldApprover, req.NewApprover, req.Reason)
			_ = s.auditLogSvc.RecordAction(ctx, userID, "replace_approver", "task", id, details)
		}
	}

	return nil
}

// HandleTimeout 处理任务超时
func (s *taskService) HandleTimeout(ctx context.Context, id string) error {
	if err := s.taskMgr.HandleTimeout(id); err != nil {
		return err
	}
	if s.auditLogSvc != nil {
		userID := getUserIDFromContext(ctx)
		if userID != "" {
			details := fmt.Sprintf(`{"task_id":"%s"}`, id)
			_ = s.auditLogSvc.RecordAction(ctx, userID, "handle_timeout", "task", id, details)
		}
	}
	return nil
}

// Delete 删除任务
// 只允许删除特定状态的任务(pending、cancelled),且不能有审批记录
func (s *taskService) Delete(ctx context.Context, id string) error {
	// 1. 检查任务是否存在
	var taskModel model.TaskModel
	if err := s.db.Where("id = ?", id).First(&taskModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("task not found")
		}
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 2. 获取任务
	tsk, err := s.taskMgr.Get(id)
	if err != nil {
		// 如果 taskMgr.Get 也失败，说明数据不一致，但我们已经确认任务存在，所以返回错误
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 3. 检查任务状态,只允许删除 pending 或 cancelled 状态的任务
	if tsk.State != "pending" && tsk.State != "cancelled" {
		return fmt.Errorf("无法删除任务: 只能删除待审批或已取消状态的任务,当前状态为 %s", tsk.State)
	}

	// 4. 检查是否有审批记录,如果有则不允许删除
	if len(tsk.Records) > 0 {
		return fmt.Errorf("无法删除任务: 该任务已有 %d 条审批记录,不允许删除", len(tsk.Records))
	}

	// 5. 权限检查: 只有创建者可以删除
	if s.fgaClient != nil {
		userID := getUserIDFromContext(ctx)
		if userID != "" {
			hasPermission, err := s.fgaClient.CheckPermission(ctx, userID, "operator", "task", id)
			if err != nil {
				return fmt.Errorf("failed to check permission: %w", err)
			}
			if !hasPermission {
				// 使用已经获取的 taskModel 中的 CreatedBy
				if taskModel.CreatedBy != userID {
					return fmt.Errorf("权限不足: 只有任务创建者可以删除任务")
				}
			}
		}
	}

	// 6. 删除任务及相关数据(使用事务)
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 6.1 删除审批记录
		if err := tx.Where("task_id = ?", id).Delete(&model.ApprovalRecordModel{}).Error; err != nil {
			return fmt.Errorf("failed to delete approval records: %w", err)
		}

		// 6.2 删除状态历史
		if err := tx.Where("task_id = ?", id).Delete(&model.StateHistoryModel{}).Error; err != nil {
			return fmt.Errorf("failed to delete state history: %w", err)
		}

		// 6.3 删除事件
		if err := tx.Where("task_id = ?", id).Delete(&model.EventModel{}).Error; err != nil {
			return fmt.Errorf("failed to delete events: %w", err)
		}

		// 6.4 删除任务
		if err := tx.Where("id = ?", id).Delete(&model.TaskModel{}).Error; err != nil {
			return fmt.Errorf("failed to delete task: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	// 7. 记录审计日志
	if s.auditLogSvc != nil {
		userID := getUserIDFromContext(ctx)
		if userID != "" {
			details := fmt.Sprintf(`{"task_id":"%s","state":"%s","business_id":"%s"}`, id, tsk.State, tsk.BusinessID)
			_ = s.auditLogSvc.RecordAction(ctx, userID, "delete", "task", id, details)
		}
	}

	return nil
}

// BatchApprove 批量审批
func (s *taskService) BatchApprove(ctx context.Context, req *BatchApproveRequest) ([]BatchOperationResult, error) {
	results := make([]BatchOperationResult, 0, len(req.TaskIDs))

	for _, taskID := range req.TaskIDs {
		approveReq := &ApproveRequest{
			NodeID:  req.NodeID,
			Comment: req.Comment,
		}

		err := s.Approve(ctx, taskID, approveReq)
		result := BatchOperationResult{
			TaskID:  taskID,
			Success: err == nil,
		}
		if err != nil {
			result.Error = err.Error()
		}
		results = append(results, result)
	}

	return results, nil
}

// BatchTransfer 批量转交
func (s *taskService) BatchTransfer(ctx context.Context, req *BatchTransferRequest) ([]BatchOperationResult, error) {
	results := make([]BatchOperationResult, 0, len(req.TaskIDs))
	userID := getUserIDFromContext(ctx)

	for _, taskID := range req.TaskIDs {
		// 使用当前用户作为 fromApprover,或者使用请求中的 OldApprover
		fromApprover := userID
		if req.OldApprover != "" {
			fromApprover = req.OldApprover
		}

		err := s.taskMgr.Transfer(taskID, req.NodeID, fromApprover, req.NewApprover, req.Comment)
		result := BatchOperationResult{
			TaskID:  taskID,
			Success: err == nil,
		}
		if err != nil {
			result.Error = err.Error()
		}
		results = append(results, result)
	}

	return results, nil
}
