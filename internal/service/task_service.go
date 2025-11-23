package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mautops/approval-gin/internal/auth"
	"github.com/mautops/approval-gin/internal/metrics"
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

// TransferRequest 转交审批请求
// @Description 转交审批的请求参数
type TransferRequest struct {
	NodeID      string `json:"node_id" example:"node-001" binding:"required"` // 节点 ID
	FromApprover string `json:"from_approver" example:"user-001" binding:"required"` // 原审批人 ID
	ToApprover   string `json:"to_approver" example:"user-002" binding:"required"` // 新审批人 ID
	Reason       string `json:"reason" example:"转交原因"` // 转交原因
}

// AddApproverRequest 加签请求
// @Description 加签的请求参数
type AddApproverRequest struct {
	NodeID   string `json:"node_id" example:"node-001" binding:"required"` // 节点 ID
	Approver string `json:"approver" example:"user-002" binding:"required"` // 新审批人 ID
	Reason   string `json:"reason" example:"加签原因"` // 加签原因
}

// RemoveApproverRequest 减签请求
// @Description 减签的请求参数
type RemoveApproverRequest struct {
	NodeID   string `json:"node_id" example:"node-001" binding:"required"` // 节点 ID
	Approver string `json:"approver" example:"user-001" binding:"required"` // 要移除的审批人 ID
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
	TaskIDs []string `json:"task_ids" binding:"required,min=1"` // 任务 ID 列表
	NodeID  string   `json:"node_id" binding:"required"`        // 节点 ID
	Comment string   `json:"comment"`                          // 审批意见
}

// BatchTransferRequest 批量转交请求
// @Description 批量转交的请求参数
type BatchTransferRequest struct {
	TaskIDs     []string `json:"task_ids" binding:"required,min=1"` // 任务 ID 列表
	NodeID      string   `json:"node_id" binding:"required"`         // 节点 ID
	OldApprover string   `json:"old_approver" binding:"required"`   // 原审批人 ID
	NewApprover string   `json:"new_approver" binding:"required"`   // 新审批人 ID
	Comment     string   `json:"comment"`                            // 转交说明
}

// BatchOperationResult 批量操作结果
// @Description 批量操作的结果项
type BatchOperationResult struct {
	TaskID  string `json:"task_id"`  // 任务 ID
	Success bool   `json:"success"`  // 是否成功
	Error   string `json:"error,omitempty"` // 错误信息(如果失败)
}

// taskService 任务服务实现
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

	// 设置权限关系（如果有 OpenFGA 客户端和用户ID）
	// 注意: 用户ID需要从 context 中获取，这里暂时跳过
	// 后续在 API 层从 context 获取用户ID并设置权限关系
	_ = s.fgaClient

	return task, nil
}

// Get 获取任务
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
	var err error
	if len(req.Attachments) > 0 {
		err = s.taskMgr.ApproveWithAttachments(id, req.NodeID, "", req.Comment, req.Attachments)
	} else {
		err = s.taskMgr.Approve(id, req.NodeID, "", req.Comment)
	}
	if err != nil {
		return err
	}

	// 记录业务指标
	metrics.RecordApproval("approve")

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
	var err error
	if len(req.Attachments) > 0 {
		err = s.taskMgr.RejectWithAttachments(id, req.NodeID, "", req.Comment, req.Attachments)
	} else {
		err = s.taskMgr.Reject(id, req.NodeID, "", req.Comment)
	}
	if err != nil {
		return err
	}

	// 记录业务指标
	metrics.RecordApproval("reject")

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
	if err := s.taskMgr.Transfer(id, req.NodeID, req.FromApprover, req.ToApprover, req.Reason); err != nil {
		return err
	}

	// 记录审计日志
	if s.auditLogSvc != nil {
		userID := getUserIDFromContext(ctx)
		if userID != "" {
			details := fmt.Sprintf(`{"task_id":"%s","node_id":"%s","from_approver":"%s","to_approver":"%s","reason":"%s"}`, id, req.NodeID, req.FromApprover, req.ToApprover, req.Reason)
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

	for _, taskID := range req.TaskIDs {
		transferReq := &TransferRequest{
			NodeID:      req.NodeID,
			FromApprover: req.OldApprover,
			ToApprover:  req.NewApprover,
			Reason:      req.Comment,
		}

		err := s.Transfer(ctx, taskID, transferReq)
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

