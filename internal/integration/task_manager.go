package integration

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-gin/internal/repository"
	"github.com/mautops/approval-kit/pkg/event"
	pkgSM "github.com/mautops/approval-kit/pkg/statemachine"
	"github.com/mautops/approval-kit/pkg/task"
	"github.com/mautops/approval-kit/pkg/template"
	"github.com/mautops/approval-kit/pkg/types"
	"gorm.io/gorm"
)

// taskAdapter 适配器,让 task.Task 实现 pkgSM.TransitionableTask 接口
type taskAdapter struct {
	task *task.Task
}

func (a *taskAdapter) GetState() types.TaskState {
	return a.task.GetState()
}

func (a *taskAdapter) SetState(state types.TaskState) {
	a.task.SetState(state)
}

func (a *taskAdapter) GetUpdatedAt() time.Time {
	return a.task.GetUpdatedAt()
}

func (a *taskAdapter) SetUpdatedAt(t time.Time) {
	a.task.SetUpdatedAt(t)
}

func (a *taskAdapter) GetStateHistory() []*pkgSM.StateChange {
	history := a.task.GetStateHistory()
	result := make([]*pkgSM.StateChange, len(history))
	for i, sc := range history {
		result[i] = &pkgSM.StateChange{
			From:   sc.From,
			To:     sc.To,
			Reason: sc.Reason,
			Time:   sc.Time,
		}
	}
	return result
}

func (a *taskAdapter) AddStateChange(change *pkgSM.StateChange) {
	a.task.AddStateChangeRecord(change.From, change.To, change.Reason, change.Time)
	// 注意: 状态历史会通过 taskAdapter 的 manager 保存到数据库
	// 这里只更新内存中的任务对象
}

func (a *taskAdapter) Clone() pkgSM.TransitionableTask {
	return &taskAdapter{task: a.task.Clone()}
}

// dbTaskManager 基于数据库的任务管理器
type dbTaskManager struct {
	db          *gorm.DB
	templateMgr template.TemplateManager
	stateMachine pkgSM.StateMachine
	eventHandler event.EventHandler
	recordRepo  repository.ApprovalRecordRepository
	historyRepo repository.StateHistoryRepository
}

// NewTaskManager 创建任务管理器
// 返回 pkg/task.TaskManager 接口实现
func NewTaskManager(db *gorm.DB, templateMgr template.TemplateManager, stateMachine pkgSM.StateMachine, eventHandler event.EventHandler) task.TaskManager {
	// 如果没有提供状态机,创建默认实例
	if stateMachine == nil {
		stateMachine = pkgSM.NewStateMachine()
	}
	
	return &dbTaskManager{
		db:          db,
		templateMgr: templateMgr,
		stateMachine: stateMachine,
		eventHandler: eventHandler,
		recordRepo:  repository.NewApprovalRecordRepository(db),
		historyRepo: repository.NewStateHistoryRepository(db),
	}
}

// Create 创建任务
func (m *dbTaskManager) Create(templateID string, businessID string, params json.RawMessage) (*task.Task, error) {
	// 1. 获取模板
	tpl, err := m.templateMgr.Get(templateID, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// 2. 查找开始节点
	startNodeID := findStartNode(tpl)
	
	// 3. 创建任务对象
	tsk := &task.Task{
		ID:              generateTaskID(),
		TemplateID:      templateID,
		TemplateVersion: tpl.Version,
		BusinessID:      businessID,
		Params:          params,
		State:           types.TaskStatePending,
		CurrentNode:     startNodeID,
		PausedAt:        nil,
		PausedState:     types.TaskStatePending,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		SubmittedAt:     nil,
		NodeOutputs:     make(map[string]json.RawMessage),
		Approvers:       make(map[string][]string),
		Approvals:       make(map[string]map[string]*task.Approval),
		CompletedNodes:  []string{},
		Records:         []*task.Record{},
		StateHistory:    []*task.StateChange{},
	}

	// 3. 序列化任务数据
	data, err := json.Marshal(tsk)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task: %w", err)
	}

	// 4. 保存到数据库
	taskModel := &model.TaskModel{
		ID:             tsk.ID,
		TemplateID:     tsk.TemplateID,
		TemplateVersion: tsk.TemplateVersion,
		BusinessID:     tsk.BusinessID,
		State:          string(tsk.State),
		CurrentNode:    tsk.CurrentNode,
		Data:           data,
		CreatedAt:      tsk.CreatedAt,
		UpdatedAt:      tsk.UpdatedAt,
		SubmittedAt:    tsk.SubmittedAt,
	}

	if err := m.db.Create(taskModel).Error; err != nil {
		return nil, fmt.Errorf("failed to save task: %w", err)
	}

	return tsk, nil
}

// Get 获取任务
func (m *dbTaskManager) Get(id string) (*task.Task, error) {
	var tm model.TaskModel
	if err := m.db.Where("id = ?", id).First(&tm).Error; err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	// 反序列化
	var tsk task.Task
	if err := json.Unmarshal(tm.Data, &tsk); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &tsk, nil
}

// generateTaskID 生成任务 ID
func generateTaskID() string {
	return fmt.Sprintf("task-%d", time.Now().UnixNano())
}

// generateRecordID 生成审批记录 ID
func generateRecordID() string {
	return fmt.Sprintf("record-%d", time.Now().UnixNano())
}

// Submit 提交任务进入审批流程
// 使用状态机进行状态转换,从 pending 转换为 submitted
func (m *dbTaskManager) Submit(id string) error {
	// 1. 获取任务
	tsk, err := m.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 2. 验证当前状态允许提交
	if !m.stateMachine.CanTransition(tsk.GetState(), types.TaskStateSubmitted) {
		return fmt.Errorf("invalid state transition: cannot submit task in state %q", tsk.GetState())
	}

	// 3. 使用状态机执行状态转换
	adapter := &taskAdapter{task: tsk}
	oldState := tsk.GetState()
	newTaskAdapter, err := m.stateMachine.Transition(adapter, types.TaskStateSubmitted, "task submitted")
	if err != nil {
		return fmt.Errorf("state transition failed: %w", err)
	}

	// 4. 获取转换后的任务对象
	newTask := newTaskAdapter.(*taskAdapter).task

	// 保存状态历史到数据库
	if err := m.saveStateHistory(id, oldState, newTask.GetState(), "task submitted", "system"); err != nil {
		return fmt.Errorf("failed to save state history: %w", err)
	}

	// 5. 设置提交时间
	now := time.Now()
	newTask.SubmittedAt = &now

	// 6. 执行开始节点逻辑(如果当前节点是开始节点,找到下一个节点)
	if newTask.CurrentNode != "" {
		tpl, err := m.templateMgr.Get(newTask.TemplateID, newTask.TemplateVersion)
		if err == nil {
			if node, exists := tpl.Nodes[newTask.CurrentNode]; exists && node.Type == template.NodeTypeStart {
				// 查找从 start 节点出发的下一个节点
				nextNodeID := findNextNode(tpl, newTask.CurrentNode)
				if nextNodeID != "" {
					// 更新当前节点为下一个节点
					newTask.CurrentNode = nextNodeID
					// 保存开始节点的输出(任务参数)
					if newTask.NodeOutputs == nil {
						newTask.NodeOutputs = make(map[string]json.RawMessage)
					}
					if newTask.Params != nil && len(newTask.Params) > 0 {
						newTask.NodeOutputs[node.ID] = newTask.Params
					} else {
						newTask.NodeOutputs[node.ID] = json.RawMessage("{}")
					}
					// 将开始节点标记为已完成
					newTask.CompletedNodes = append(newTask.CompletedNodes, node.ID)
				}
			}
		}
	}

	// 7. 序列化并保存到数据库
	data, err := json.Marshal(newTask)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	// 7. 更新数据库
	taskModel := &model.TaskModel{
		ID:             newTask.ID,
		TemplateID:     newTask.TemplateID,
		TemplateVersion: newTask.TemplateVersion,
		BusinessID:     newTask.BusinessID,
		State:          string(newTask.State),
		CurrentNode:    newTask.CurrentNode,
		Data:           data,
		CreatedAt:      newTask.CreatedAt,
		UpdatedAt:      newTask.UpdatedAt,
		SubmittedAt:    newTask.SubmittedAt,
	}

	if err := m.db.Model(&model.TaskModel{}).Where("id = ?", id).Updates(taskModel).Error; err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

// Approve 审批人进行同意操作
func (m *dbTaskManager) Approve(id string, nodeID string, approver string, comment string) error {
	// 1. 获取任务
	tsk, err := m.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 2. 验证任务状态(只有 submitted 或 approving 状态才能审批)
	currentState := tsk.GetState()
	if currentState != types.TaskStateSubmitted && currentState != types.TaskStateApproving {
		return fmt.Errorf("task state %q cannot be approved", currentState)
	}

	// 3. 获取模板和节点配置,验证审批意见必填
	tpl, err := m.templateMgr.Get(tsk.TemplateID, 0)
	if err != nil {
		return fmt.Errorf("failed to get template %q: %w", tsk.TemplateID, err)
	}

	node, exists := tpl.Nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %q not found in template", nodeID)
	}

	if node.Type == template.NodeTypeApproval {
		approvalConfig, ok := node.Config.(template.ApprovalNodeConfigAccessor)
		if ok && approvalConfig.RequireComment() {
			if comment == "" {
				return fmt.Errorf("comment is required for approval node %q", nodeID)
			}
		}
	}

	// 4. 更新任务状态为 approving(如果还是 submitted)
	if currentState == types.TaskStateSubmitted {
		adapter := &taskAdapter{task: tsk}
		newTaskAdapter, err := m.stateMachine.Transition(adapter, types.TaskStateApproving, "task approved")
		if err != nil {
			return fmt.Errorf("state transition failed: %w", err)
		}
		tsk = newTaskAdapter.(*taskAdapter).task
	}

	// 5. 记录审批结果
	// 初始化节点审批记录
	if tsk.Approvals == nil {
		tsk.Approvals = make(map[string]map[string]*task.Approval)
	}
	if tsk.Approvals[nodeID] == nil {
		tsk.Approvals[nodeID] = make(map[string]*task.Approval)
	}

	// 记录审批结果
	tsk.Approvals[nodeID][approver] = &task.Approval{
		Result:    "approve",
		Comment:   comment,
		CreatedAt: time.Now(),
	}

	// 6. 生成审批记录
	record := &task.Record{
		ID:         generateRecordID(),
		TaskID:     id,
		NodeID:     nodeID,
		Approver:   approver,
		Result:     "approve",
		Comment:    comment,
		CreatedAt:  time.Now(),
		Attachments: []string{},
	}

	// 添加到记录列表
	tsk.Records = append(tsk.Records, record)

	// 保存审批记录到数据库
	attachmentsJSON, _ := json.Marshal(record.Attachments)
	recordModel := &model.ApprovalRecordModel{
		ID:          record.ID,
		TaskID:      record.TaskID,
		NodeID:      record.NodeID,
		Approver:    record.Approver,
		Result:      record.Result,
		Comment:     record.Comment,
		Attachments: attachmentsJSON,
		CreatedAt:   record.CreatedAt,
	}
	if err := m.recordRepo.Save(recordModel); err != nil {
		return fmt.Errorf("failed to save approval record: %w", err)
	}

	// 7. 检查审批是否完成(简化处理: 对于单人审批模式,审批人同意后立即完成)
	// 获取审批人列表
	approvers := tsk.Approvers[nodeID]
	shouldTransition := false

	// 简化处理: 如果只有一个审批人且就是当前审批人,且已同意,将任务状态转换为已通过
	if len(approvers) == 1 && approvers[0] == approver {
		// 单人审批模式,审批人已同意
		if m.stateMachine.CanTransition(tsk.GetState(), types.TaskStateApproved) {
			shouldTransition = true
		}
	} else if len(approvers) > 1 {
		// 多人审批模式: 检查是否所有审批人都已同意(简化处理,假设是会签模式)
		allApproved := true
		for _, approverID := range approvers {
			approval, exists := tsk.Approvals[nodeID][approverID]
			if !exists || approval == nil || approval.Result != "approve" {
				allApproved = false
				break
			}
		}
		if allApproved && m.stateMachine.CanTransition(tsk.GetState(), types.TaskStateApproved) {
			shouldTransition = true
		}
	} else if len(approvers) == 0 {
		// 如果没有审批人列表,假设是单人审批模式(当前审批人就是唯一审批人)
		if m.stateMachine.CanTransition(tsk.GetState(), types.TaskStateApproved) {
			shouldTransition = true
		}
	}

	// 8. 如果需要转换状态,执行状态转换
	if shouldTransition {
		adapter := &taskAdapter{task: tsk}
		oldState := tsk.GetState()
		newTaskAdapter, err := m.stateMachine.Transition(adapter, types.TaskStateApproved, "all approvers approved")
		if err != nil {
			return fmt.Errorf("state transition failed: %w", err)
		}
		tsk = newTaskAdapter.(*taskAdapter).task

		// 保存状态历史到数据库
		if err := m.saveStateHistory(id, oldState, tsk.GetState(), "all approvers approved", approver); err != nil {
			return fmt.Errorf("failed to save state history: %w", err)
		}

		// 节点完成,添加到已完成节点列表
		if tsk.CompletedNodes == nil {
			tsk.CompletedNodes = []string{}
		}
		// 检查节点是否已在列表中
		found := false
		for _, completedNodeID := range tsk.CompletedNodes {
			if completedNodeID == nodeID {
				found = true
				break
			}
		}
		if !found {
			tsk.CompletedNodes = append(tsk.CompletedNodes, nodeID)
		}

		// 查找下一个节点
		nextNodeID := findNextNode(tpl, nodeID)
		if nextNodeID != "" {
			tsk.CurrentNode = nextNodeID
			// 保存审批节点的输出
			if tsk.NodeOutputs == nil {
				tsk.NodeOutputs = make(map[string]json.RawMessage)
			}
			output := json.RawMessage(`{"result":"approve"}`)
			tsk.NodeOutputs[nodeID] = output
		} else {
			// 如果没有下一个节点,任务完成
			tsk.CurrentNode = ""
		}
	}

	// 9. 更新任务更新时间
	tsk.UpdatedAt = time.Now()

	// 10. 序列化并保存到数据库
	data, err := json.Marshal(tsk)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	taskModel := &model.TaskModel{
		ID:             tsk.ID,
		TemplateID:     tsk.TemplateID,
		TemplateVersion: tsk.TemplateVersion,
		BusinessID:     tsk.BusinessID,
		State:          string(tsk.State),
		CurrentNode:    tsk.CurrentNode,
		Data:           data,
		CreatedAt:      tsk.CreatedAt,
		UpdatedAt:      tsk.UpdatedAt,
		SubmittedAt:    tsk.SubmittedAt,
	}

	if err := m.db.Model(&model.TaskModel{}).Where("id = ?", id).Updates(taskModel).Error; err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// 11. 生成审批事件
	if m.eventHandler != nil {
		// 事件生成逻辑将在后续实现
	}

	return nil
}

// ApproveWithAttachments 审批人进行同意操作(带附件)
func (m *dbTaskManager) ApproveWithAttachments(id string, nodeID string, approver string, comment string, attachments []string) error {
	// 1. 获取任务
	tsk, err := m.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 2. 验证任务状态(只有 submitted 或 approving 状态才能审批)
	currentState := tsk.GetState()
	if currentState != types.TaskStateSubmitted && currentState != types.TaskStateApproving {
		return fmt.Errorf("task state %q cannot be approved", currentState)
	}

	// 3. 获取模板和节点配置,验证审批意见和附件要求
	tpl, err := m.templateMgr.Get(tsk.TemplateID, 0)
	if err != nil {
		return fmt.Errorf("failed to get template %q: %w", tsk.TemplateID, err)
	}

	node, exists := tpl.Nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %q not found in template", nodeID)
	}

	if node.Type == template.NodeTypeApproval {
		approvalConfig, ok := node.Config.(template.ApprovalNodeConfigAccessor)
		if ok {
			if approvalConfig.RequireComment() && comment == "" {
				return fmt.Errorf("comment is required for approval node %q", nodeID)
			}
			if approvalConfig.RequireAttachments() && len(attachments) == 0 {
				return fmt.Errorf("attachments are required for approval node %q", nodeID)
			}
		}
	}

	// 4. 更新任务状态为 approving(如果还是 submitted)
	if currentState == types.TaskStateSubmitted {
		adapter := &taskAdapter{task: tsk}
		newTaskAdapter, err := m.stateMachine.Transition(adapter, types.TaskStateApproving, "task approved")
		if err != nil {
			return fmt.Errorf("state transition failed: %w", err)
		}
		tsk = newTaskAdapter.(*taskAdapter).task
	}

	// 5. 记录审批结果
	// 初始化节点审批记录
	if tsk.Approvals == nil {
		tsk.Approvals = make(map[string]map[string]*task.Approval)
	}
	if tsk.Approvals[nodeID] == nil {
		tsk.Approvals[nodeID] = make(map[string]*task.Approval)
	}

	// 记录审批结果
	tsk.Approvals[nodeID][approver] = &task.Approval{
		Result:    "approve",
		Comment:   comment,
		CreatedAt: time.Now(),
	}

	// 6. 生成审批记录
	record := &task.Record{
		ID:         generateRecordID(),
		TaskID:     id,
		NodeID:     nodeID,
		Approver:   approver,
		Result:     "approve",
		Comment:    comment,
		CreatedAt:  time.Now(),
		Attachments: attachments,
	}

	// 添加到记录列表
	tsk.Records = append(tsk.Records, record)

	// 保存审批记录到数据库
	attachmentsJSON, _ := json.Marshal(record.Attachments)
	recordModel := &model.ApprovalRecordModel{
		ID:          record.ID,
		TaskID:      record.TaskID,
		NodeID:      record.NodeID,
		Approver:    record.Approver,
		Result:      record.Result,
		Comment:     record.Comment,
		Attachments: attachmentsJSON,
		CreatedAt:   record.CreatedAt,
	}
	if err := m.recordRepo.Save(recordModel); err != nil {
		return fmt.Errorf("failed to save approval record: %w", err)
	}

	// 7. 检查审批是否完成(与 Approve 方法相同的逻辑)
	approvers := tsk.Approvers[nodeID]
	shouldTransition := false

	if len(approvers) == 1 && approvers[0] == approver {
		if m.stateMachine.CanTransition(tsk.GetState(), types.TaskStateApproved) {
			shouldTransition = true
		}
	} else if len(approvers) > 1 {
		allApproved := true
		for _, approverID := range approvers {
			approval, exists := tsk.Approvals[nodeID][approverID]
			if !exists || approval == nil || approval.Result != "approve" {
				allApproved = false
				break
			}
		}
		if allApproved && m.stateMachine.CanTransition(tsk.GetState(), types.TaskStateApproved) {
			shouldTransition = true
		}
	} else if len(approvers) == 0 {
		if m.stateMachine.CanTransition(tsk.GetState(), types.TaskStateApproved) {
			shouldTransition = true
		}
	}

	// 8. 如果需要转换状态,执行状态转换
	if shouldTransition {
		adapter := &taskAdapter{task: tsk}
		oldState := tsk.GetState()
		newTaskAdapter, err := m.stateMachine.Transition(adapter, types.TaskStateApproved, "all approvers approved")
		if err != nil {
			return fmt.Errorf("state transition failed: %w", err)
		}
		tsk = newTaskAdapter.(*taskAdapter).task

		// 保存状态历史到数据库
		if err := m.saveStateHistory(id, oldState, tsk.GetState(), "all approvers approved", approver); err != nil {
			return fmt.Errorf("failed to save state history: %w", err)
		}

		// 节点完成,添加到已完成节点列表
		if tsk.CompletedNodes == nil {
			tsk.CompletedNodes = []string{}
		}
		found := false
		for _, completedNodeID := range tsk.CompletedNodes {
			if completedNodeID == nodeID {
				found = true
				break
			}
		}
		if !found {
			tsk.CompletedNodes = append(tsk.CompletedNodes, nodeID)
		}

		// 查找下一个节点
		nextNodeID := findNextNode(tpl, nodeID)
		if nextNodeID != "" {
			tsk.CurrentNode = nextNodeID
			if tsk.NodeOutputs == nil {
				tsk.NodeOutputs = make(map[string]json.RawMessage)
			}
			output := json.RawMessage(`{"result":"approve"}`)
			tsk.NodeOutputs[nodeID] = output
		} else {
			tsk.CurrentNode = ""
		}
	}

	// 9. 更新任务更新时间
	tsk.UpdatedAt = time.Now()

	// 10. 序列化并保存到数据库
	data, err := json.Marshal(tsk)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	taskModel := &model.TaskModel{
		ID:             tsk.ID,
		TemplateID:     tsk.TemplateID,
		TemplateVersion: tsk.TemplateVersion,
		BusinessID:     tsk.BusinessID,
		State:          string(tsk.State),
		CurrentNode:    tsk.CurrentNode,
		Data:           data,
		CreatedAt:      tsk.CreatedAt,
		UpdatedAt:      tsk.UpdatedAt,
		SubmittedAt:    tsk.SubmittedAt,
	}

	if err := m.db.Model(&model.TaskModel{}).Where("id = ?", id).Updates(taskModel).Error; err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// 11. 生成审批事件
	if m.eventHandler != nil {
		// 事件生成逻辑将在后续实现
	}

	return nil
}

// Reject 审批人进行拒绝操作
func (m *dbTaskManager) Reject(id string, nodeID string, approver string, comment string) error {
	// 1. 获取任务
	tsk, err := m.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 2. 验证任务状态(只有 submitted 或 approving 状态才能拒绝)
	currentState := tsk.GetState()
	if currentState != types.TaskStateSubmitted && currentState != types.TaskStateApproving {
		return fmt.Errorf("task state %q cannot be rejected", currentState)
	}

	// 3. 获取模板和节点配置,验证审批意见必填
	tpl, err := m.templateMgr.Get(tsk.TemplateID, 0)
	if err != nil {
		return fmt.Errorf("failed to get template %q: %w", tsk.TemplateID, err)
	}

	node, exists := tpl.Nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %q not found in template", nodeID)
	}

	if node.Type == template.NodeTypeApproval {
		approvalConfig, ok := node.Config.(template.ApprovalNodeConfigAccessor)
		if ok && approvalConfig.RequireComment() {
			if comment == "" {
				return fmt.Errorf("comment is required for approval node %q", nodeID)
			}
		}
	}

	// 4. 更新任务状态为 approving(如果还是 submitted)
	if currentState == types.TaskStateSubmitted {
		adapter := &taskAdapter{task: tsk}
		newTaskAdapter, err := m.stateMachine.Transition(adapter, types.TaskStateApproving, "task rejected")
		if err != nil {
			return fmt.Errorf("state transition failed: %w", err)
		}
		tsk = newTaskAdapter.(*taskAdapter).task
	}

	// 5. 记录审批结果
	// 初始化节点审批记录
	if tsk.Approvals == nil {
		tsk.Approvals = make(map[string]map[string]*task.Approval)
	}
	if tsk.Approvals[nodeID] == nil {
		tsk.Approvals[nodeID] = make(map[string]*task.Approval)
	}

	// 记录审批结果
	tsk.Approvals[nodeID][approver] = &task.Approval{
		Result:    "reject",
		Comment:   comment,
		CreatedAt: time.Now(),
	}

	// 6. 生成审批记录
	record := &task.Record{
		ID:         generateRecordID(),
		TaskID:     id,
		NodeID:     nodeID,
		Approver:   approver,
		Result:     "reject",
		Comment:    comment,
		CreatedAt:  time.Now(),
		Attachments: []string{},
	}

	// 添加到记录列表
	tsk.Records = append(tsk.Records, record)

	// 7. 拒绝后,将任务状态转换为 rejected
	if m.stateMachine.CanTransition(tsk.GetState(), types.TaskStateRejected) {
		adapter := &taskAdapter{task: tsk}
		newTaskAdapter, err := m.stateMachine.Transition(adapter, types.TaskStateRejected, "task rejected")
		if err != nil {
			return fmt.Errorf("state transition failed: %w", err)
		}
		tsk = newTaskAdapter.(*taskAdapter).task

		// 节点完成,添加到已完成节点列表
		if tsk.CompletedNodes == nil {
			tsk.CompletedNodes = []string{}
		}
		found := false
		for _, completedNodeID := range tsk.CompletedNodes {
			if completedNodeID == nodeID {
				found = true
				break
			}
		}
		if !found {
			tsk.CompletedNodes = append(tsk.CompletedNodes, nodeID)
		}

		// 保存拒绝节点的输出
		if tsk.NodeOutputs == nil {
			tsk.NodeOutputs = make(map[string]json.RawMessage)
		}
		output := json.RawMessage(`{"result":"reject"}`)
		tsk.NodeOutputs[nodeID] = output
	}

	// 8. 更新任务更新时间
	tsk.UpdatedAt = time.Now()

	// 9. 序列化并保存到数据库
	data, err := json.Marshal(tsk)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	taskModel := &model.TaskModel{
		ID:             tsk.ID,
		TemplateID:     tsk.TemplateID,
		TemplateVersion: tsk.TemplateVersion,
		BusinessID:     tsk.BusinessID,
		State:          string(tsk.State),
		CurrentNode:    tsk.CurrentNode,
		Data:           data,
		CreatedAt:      tsk.CreatedAt,
		UpdatedAt:      tsk.UpdatedAt,
		SubmittedAt:    tsk.SubmittedAt,
	}

	if err := m.db.Model(&model.TaskModel{}).Where("id = ?", id).Updates(taskModel).Error; err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// 10. 生成拒绝事件
	if m.eventHandler != nil {
		// 事件生成逻辑将在后续实现
	}

	return nil
}

// RejectWithAttachments 审批人进行拒绝操作(带附件)
func (m *dbTaskManager) RejectWithAttachments(id string, nodeID string, approver string, comment string, attachments []string) error {
	// 1. 获取任务
	tsk, err := m.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 2. 验证任务状态(只有 submitted 或 approving 状态才能拒绝)
	currentState := tsk.GetState()
	if currentState != types.TaskStateSubmitted && currentState != types.TaskStateApproving {
		return fmt.Errorf("task state %q cannot be rejected", currentState)
	}

	// 3. 获取模板和节点配置,验证审批意见和附件要求
	tpl, err := m.templateMgr.Get(tsk.TemplateID, 0)
	if err != nil {
		return fmt.Errorf("failed to get template %q: %w", tsk.TemplateID, err)
	}

	node, exists := tpl.Nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %q not found in template", nodeID)
	}

	if node.Type == template.NodeTypeApproval {
		approvalConfig, ok := node.Config.(template.ApprovalNodeConfigAccessor)
		if ok {
			if approvalConfig.RequireComment() && comment == "" {
				return fmt.Errorf("comment is required for approval node %q", nodeID)
			}
			if approvalConfig.RequireAttachments() && len(attachments) == 0 {
				return fmt.Errorf("attachments are required for approval node %q", nodeID)
			}
		}
	}

	// 4. 更新任务状态为 approving(如果还是 submitted)
	if currentState == types.TaskStateSubmitted {
		adapter := &taskAdapter{task: tsk}
		newTaskAdapter, err := m.stateMachine.Transition(adapter, types.TaskStateApproving, "task rejected")
		if err != nil {
			return fmt.Errorf("state transition failed: %w", err)
		}
		tsk = newTaskAdapter.(*taskAdapter).task
	}

	// 5. 记录审批结果
	// 初始化节点审批记录
	if tsk.Approvals == nil {
		tsk.Approvals = make(map[string]map[string]*task.Approval)
	}
	if tsk.Approvals[nodeID] == nil {
		tsk.Approvals[nodeID] = make(map[string]*task.Approval)
	}

	// 记录审批结果
	tsk.Approvals[nodeID][approver] = &task.Approval{
		Result:    "reject",
		Comment:   comment,
		CreatedAt: time.Now(),
	}

	// 6. 生成审批记录
	record := &task.Record{
		ID:         generateRecordID(),
		TaskID:     id,
		NodeID:     nodeID,
		Approver:   approver,
		Result:     "reject",
		Comment:    comment,
		CreatedAt:  time.Now(),
		Attachments: attachments,
	}

	// 添加到记录列表
	tsk.Records = append(tsk.Records, record)

	// 7. 拒绝后,将任务状态转换为 rejected
	if m.stateMachine.CanTransition(tsk.GetState(), types.TaskStateRejected) {
		adapter := &taskAdapter{task: tsk}
		newTaskAdapter, err := m.stateMachine.Transition(adapter, types.TaskStateRejected, "task rejected")
		if err != nil {
			return fmt.Errorf("state transition failed: %w", err)
		}
		tsk = newTaskAdapter.(*taskAdapter).task

		// 节点完成,添加到已完成节点列表
		if tsk.CompletedNodes == nil {
			tsk.CompletedNodes = []string{}
		}
		found := false
		for _, completedNodeID := range tsk.CompletedNodes {
			if completedNodeID == nodeID {
				found = true
				break
			}
		}
		if !found {
			tsk.CompletedNodes = append(tsk.CompletedNodes, nodeID)
		}

		// 保存拒绝节点的输出
		if tsk.NodeOutputs == nil {
			tsk.NodeOutputs = make(map[string]json.RawMessage)
		}
		output := json.RawMessage(`{"result":"reject"}`)
		tsk.NodeOutputs[nodeID] = output
	}

	// 8. 更新任务更新时间
	tsk.UpdatedAt = time.Now()

	// 9. 序列化并保存到数据库
	data, err := json.Marshal(tsk)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	taskModel := &model.TaskModel{
		ID:             tsk.ID,
		TemplateID:     tsk.TemplateID,
		TemplateVersion: tsk.TemplateVersion,
		BusinessID:     tsk.BusinessID,
		State:          string(tsk.State),
		CurrentNode:    tsk.CurrentNode,
		Data:           data,
		CreatedAt:      tsk.CreatedAt,
		UpdatedAt:      tsk.UpdatedAt,
		SubmittedAt:    tsk.SubmittedAt,
	}

	if err := m.db.Model(&model.TaskModel{}).Where("id = ?", id).Updates(taskModel).Error; err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// 10. 生成拒绝事件
	if m.eventHandler != nil {
		// 事件生成逻辑将在后续实现
	}

	return nil
}

// Cancel 取消任务
// 使用状态机进行状态转换,从当前状态转换为 cancelled
func (m *dbTaskManager) Cancel(id string, reason string) error {
	// 1. 获取任务
	tsk, err := m.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 2. 验证当前状态允许取消
	if !m.stateMachine.CanTransition(tsk.GetState(), types.TaskStateCancelled) {
		return fmt.Errorf("invalid state transition: cannot cancel task in state %q", tsk.GetState())
	}

	// 3. 使用状态机执行状态转换
	adapter := &taskAdapter{task: tsk}
	newTaskAdapter, err := m.stateMachine.Transition(adapter, types.TaskStateCancelled, reason)
	if err != nil {
		return fmt.Errorf("state transition failed: %w", err)
	}

	// 4. 获取转换后的任务对象
	newTask := newTaskAdapter.(*taskAdapter).task

	// 5. 序列化并保存到数据库
	data, err := json.Marshal(newTask)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	// 6. 更新数据库
	taskModel := &model.TaskModel{
		ID:             newTask.ID,
		TemplateID:     newTask.TemplateID,
		TemplateVersion: newTask.TemplateVersion,
		BusinessID:     newTask.BusinessID,
		State:          string(newTask.State),
		CurrentNode:    newTask.CurrentNode,
		Data:           data,
		CreatedAt:      newTask.CreatedAt,
		UpdatedAt:      newTask.UpdatedAt,
		SubmittedAt:    newTask.SubmittedAt,
	}

	if err := m.db.Model(&model.TaskModel{}).Where("id = ?", id).Updates(taskModel).Error; err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

// Withdraw 撤回任务
// 撤回会将任务从 submitted 或 approving 状态撤回回 pending 状态
// 如果任务已有审批记录,不允许撤回
func (m *dbTaskManager) Withdraw(id string, reason string) error {
	// 1. 获取任务
	tsk, err := m.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 2. 检查当前状态是否允许撤回
	currentState := tsk.GetState()
	if currentState != types.TaskStateSubmitted && currentState != types.TaskStateApproving {
		return fmt.Errorf("cannot withdraw task in state %q, only submitted or approving tasks can be withdrawn", currentState)
	}

	// 3. 检查是否有审批记录(如果有,不允许撤回)
	records := tsk.GetRecords()
	if len(records) > 0 {
		return fmt.Errorf("cannot withdraw task with approval records")
	}

	// 4. 验证当前状态允许转换为 pending
	if !m.stateMachine.CanTransition(currentState, types.TaskStatePending) {
		return fmt.Errorf("invalid state transition: cannot withdraw task from state %q", currentState)
	}

	// 5. 使用状态机执行状态转换
	adapter := &taskAdapter{task: tsk}
	newTaskAdapter, err := m.stateMachine.Transition(adapter, types.TaskStatePending, reason)
	if err != nil {
		return fmt.Errorf("state transition failed: %w", err)
	}

	// 6. 获取转换后的任务对象
	newTask := newTaskAdapter.(*taskAdapter).task

	// 7. 清空提交时间
	newTask.SubmittedAt = nil
	newTask.UpdatedAt = time.Now()

	// 8. 序列化并保存到数据库
	data, err := json.Marshal(newTask)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	taskModel := &model.TaskModel{
		ID:             newTask.ID,
		TemplateID:     newTask.TemplateID,
		TemplateVersion: newTask.TemplateVersion,
		BusinessID:     newTask.BusinessID,
		State:          string(newTask.State),
		CurrentNode:    newTask.CurrentNode,
		Data:           data,
		CreatedAt:      newTask.CreatedAt,
		UpdatedAt:      newTask.UpdatedAt,
		SubmittedAt:    newTask.SubmittedAt,
	}

	if err := m.db.Model(&model.TaskModel{}).Where("id = ?", id).Updates(taskModel).Error; err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// 9. 生成撤回事件
	if m.eventHandler != nil {
		// 事件生成逻辑将在后续实现
	}

	return nil
}

// Transfer 转交审批
// 将审批任务从原审批人转交给新审批人
// 转交需要节点配置允许转交,且原审批人必须是当前审批人
func (m *dbTaskManager) Transfer(id string, nodeID string, fromApprover string, toApprover string, reason string) error {
	// 1. 获取任务
	tsk, err := m.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 2. 获取模板
	tpl, err := m.templateMgr.Get(tsk.TemplateID, 0)
	if err != nil {
		return fmt.Errorf("failed to get template %q: %w", tsk.TemplateID, err)
	}

	// 3. 获取节点配置
	node, exists := tpl.Nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %q not found in template", nodeID)
	}

	// 4. 检查节点类型是否为审批节点
	if node.Type != template.NodeTypeApproval {
		return fmt.Errorf("node %q is not an approval node", nodeID)
	}

	// 5. 获取审批节点配置(如果配置为 nil,允许转交)
	if node.Config != nil {
		approvalConfig, ok := node.Config.(template.ApprovalNodeConfigAccessor)
		if ok {
			// 6. 检查是否允许转交
			perms, ok := approvalConfig.GetPermissions().(template.OperationPermissionsAccessor)
			if ok && !perms.AllowTransfer() {
				return fmt.Errorf("transfer is not allowed for node %q", nodeID)
			}
		}
	}

	// 7. 检查原审批人是否是任务的审批人
	approvers, exists := tsk.Approvers[nodeID]
	if !exists {
		return fmt.Errorf("approvers not found for node %q", nodeID)
	}

	// 检查原审批人是否在审批人列表中
	found := false
	for _, approver := range approvers {
		if approver == fromApprover {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("user %q is not an approver for node %q", fromApprover, nodeID)
	}

	// 8. 更新审批人列表(移除原审批人,添加新审批人)
	newApprovers := make([]string, 0, len(approvers))
	for _, approver := range approvers {
		if approver != fromApprover {
			newApprovers = append(newApprovers, approver)
		}
	}
	// 检查新审批人是否已在列表中
	toApproverExists := false
	for _, approver := range newApprovers {
		if approver == toApprover {
			toApproverExists = true
			break
		}
	}
	if !toApproverExists {
		newApprovers = append(newApprovers, toApprover)
	}
	tsk.Approvers[nodeID] = newApprovers

	// 9. 更新审批记录(如果有原审批人的审批记录,需要更新)
	// 如果原审批人已有审批记录,将其保留但标记为转交
	if tsk.Approvals != nil && tsk.Approvals[nodeID] != nil {
		// 保留原审批记录,但会在 Records 中创建转交记录
		_ = tsk.Approvals[nodeID][fromApprover]
	}

	// 10. 生成转交记录
	record := &task.Record{
		ID:         generateRecordID(),
		TaskID:     id,
		NodeID:     nodeID,
		Approver:   fromApprover,
		Result:     "transfer",
		Comment:    reason,
		CreatedAt:  time.Now(),
		Attachments: []string{},
	}

	// 添加到记录列表
	tsk.Records = append(tsk.Records, record)

	// 11. 更新任务更新时间
	tsk.UpdatedAt = time.Now()

	// 12. 序列化并保存到数据库
	data, err := json.Marshal(tsk)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	taskModel := &model.TaskModel{
		ID:             tsk.ID,
		TemplateID:     tsk.TemplateID,
		TemplateVersion: tsk.TemplateVersion,
		BusinessID:     tsk.BusinessID,
		State:          string(tsk.State),
		CurrentNode:    tsk.CurrentNode,
		Data:           data,
		CreatedAt:      tsk.CreatedAt,
		UpdatedAt:      tsk.UpdatedAt,
		SubmittedAt:    tsk.SubmittedAt,
	}

	if err := m.db.Model(&model.TaskModel{}).Where("id = ?", id).Updates(taskModel).Error; err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// 13. 生成转交事件
	if m.eventHandler != nil {
		// 事件生成逻辑将在后续实现
	}

	return nil
}

// AddApprover 加签
// 在审批人列表中添加新的审批人
// 加签需要节点配置允许加签
func (m *dbTaskManager) AddApprover(id string, nodeID string, approver string, reason string) error {
	// 1. 获取任务
	tsk, err := m.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 2. 获取模板
	tpl, err := m.templateMgr.Get(tsk.TemplateID, 0)
	if err != nil {
		return fmt.Errorf("failed to get template %q: %w", tsk.TemplateID, err)
	}

	// 3. 获取节点配置
	node, exists := tpl.Nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %q not found in template", nodeID)
	}

	// 4. 检查节点类型是否为审批节点
	if node.Type != template.NodeTypeApproval {
		return fmt.Errorf("node %q is not an approval node", nodeID)
	}

	// 5. 获取审批节点配置(如果配置为 nil,允许加签)
	if node.Config != nil {
		approvalConfig, ok := node.Config.(template.ApprovalNodeConfigAccessor)
		if ok {
			// 6. 检查是否允许加签
			perms, ok := approvalConfig.GetPermissions().(template.OperationPermissionsAccessor)
			if ok && !perms.AllowAddApprover() {
				return fmt.Errorf("add approver is not allowed for node %q", nodeID)
			}
		}
	}

	// 7. 更新审批人列表
	if tsk.Approvers == nil {
		tsk.Approvers = make(map[string][]string)
	}

	// 获取当前审批人列表
	approvers, exists := tsk.Approvers[nodeID]
	if !exists {
		approvers = []string{}
	}

	// 检查新审批人是否已在列表中
	for _, existingApprover := range approvers {
		if existingApprover == approver {
			return fmt.Errorf("approver %q already exists in node %q", approver, nodeID)
		}
	}

	// 添加新审批人
	approvers = append(approvers, approver)
	tsk.Approvers[nodeID] = approvers

	// 8. 生成加签记录
	record := &task.Record{
		ID:         generateRecordID(),
		TaskID:     id,
		NodeID:     nodeID,
		Approver:   approver,
		Result:     "add_approver",
		Comment:    reason,
		CreatedAt:  time.Now(),
		Attachments: []string{},
	}

	// 添加到记录列表
	tsk.Records = append(tsk.Records, record)

	// 9. 更新任务更新时间
	tsk.UpdatedAt = time.Now()

	// 10. 序列化并保存到数据库
	data, err := json.Marshal(tsk)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	taskModel := &model.TaskModel{
		ID:             tsk.ID,
		TemplateID:     tsk.TemplateID,
		TemplateVersion: tsk.TemplateVersion,
		BusinessID:     tsk.BusinessID,
		State:          string(tsk.State),
		CurrentNode:    tsk.CurrentNode,
		Data:           data,
		CreatedAt:      tsk.CreatedAt,
		UpdatedAt:      tsk.UpdatedAt,
		SubmittedAt:    tsk.SubmittedAt,
	}

	if err := m.db.Model(&model.TaskModel{}).Where("id = ?", id).Updates(taskModel).Error; err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// 11. 生成加签事件
	if m.eventHandler != nil {
		// 事件生成逻辑将在后续实现
	}

	return nil
}

// RemoveApprover 减签
// 从审批人列表中移除指定的审批人
// 减签需要节点配置允许减签,且审批人必须在审批人列表中
func (m *dbTaskManager) RemoveApprover(id string, nodeID string, approver string, reason string) error {
	// 1. 获取任务
	tsk, err := m.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 2. 获取模板
	tpl, err := m.templateMgr.Get(tsk.TemplateID, 0)
	if err != nil {
		return fmt.Errorf("failed to get template %q: %w", tsk.TemplateID, err)
	}

	// 3. 获取节点配置
	node, exists := tpl.Nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %q not found in template", nodeID)
	}

	// 4. 检查节点类型是否为审批节点
	if node.Type != template.NodeTypeApproval {
		return fmt.Errorf("node %q is not an approval node", nodeID)
	}

	// 5. 获取审批节点配置(如果配置为 nil,允许减签)
	if node.Config != nil {
		approvalConfig, ok := node.Config.(template.ApprovalNodeConfigAccessor)
		if ok {
			// 6. 检查是否允许减签
			perms, ok := approvalConfig.GetPermissions().(template.OperationPermissionsAccessor)
			if ok && !perms.AllowRemoveApprover() {
				return fmt.Errorf("remove approver is not allowed for node %q", nodeID)
			}
		}
	}

	// 7. 检查审批人是否在审批人列表中
	approvers, exists := tsk.Approvers[nodeID]
	if !exists {
		return fmt.Errorf("approvers not found for node %q", nodeID)
	}

	found := false
	for _, existingApprover := range approvers {
		if existingApprover == approver {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("user %q is not an approver for node %q", approver, nodeID)
	}

	// 8. 检查审批人是否已审批(如果已审批,不允许减签)
	if tsk.Approvals != nil && tsk.Approvals[nodeID] != nil {
		if approval, exists := tsk.Approvals[nodeID][approver]; exists && approval != nil {
			return fmt.Errorf("user %q has already approved, cannot remove", approver)
		}
	}

	// 9. 更新审批人列表(移除指定审批人)
	newApprovers := make([]string, 0, len(approvers)-1)
	for _, existingApprover := range approvers {
		if existingApprover != approver {
			newApprovers = append(newApprovers, existingApprover)
		}
	}
	tsk.Approvers[nodeID] = newApprovers

	// 10. 生成减签记录
	record := &task.Record{
		ID:         generateRecordID(),
		TaskID:     id,
		NodeID:     nodeID,
		Approver:   approver,
		Result:     "remove_approver",
		Comment:    reason,
		CreatedAt:  time.Now(),
		Attachments: []string{},
	}

	// 添加到记录列表
	tsk.Records = append(tsk.Records, record)

	// 11. 更新任务更新时间
	tsk.UpdatedAt = time.Now()

	// 12. 序列化并保存到数据库
	data, err := json.Marshal(tsk)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	taskModel := &model.TaskModel{
		ID:             tsk.ID,
		TemplateID:     tsk.TemplateID,
		TemplateVersion: tsk.TemplateVersion,
		BusinessID:     tsk.BusinessID,
		State:          string(tsk.State),
		CurrentNode:    tsk.CurrentNode,
		Data:           data,
		CreatedAt:      tsk.CreatedAt,
		UpdatedAt:      tsk.UpdatedAt,
		SubmittedAt:    tsk.SubmittedAt,
	}

	if err := m.db.Model(&model.TaskModel{}).Where("id = ?", id).Updates(taskModel).Error; err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// 13. 生成减签事件
	if m.eventHandler != nil {
		// 事件生成逻辑将在后续实现
	}

	return nil
}

// Query 查询任务列表(支持多条件组合)
func (m *dbTaskManager) Query(filter *task.TaskFilter) ([]*task.Task, error) {
	if filter == nil {
		filter = &task.TaskFilter{}
	}

	// 1. 构建查询条件
	query := m.db.Model(&model.TaskModel{})

	// 按状态过滤
	if filter.State != types.TaskState("") {
		query = query.Where("state = ?", string(filter.State))
	}

	// 按模板 ID 过滤
	if filter.TemplateID != "" {
		query = query.Where("template_id = ?", filter.TemplateID)
	}

	// 按业务 ID 过滤
	if filter.BusinessID != "" {
		query = query.Where("business_id = ?", filter.BusinessID)
	}

	// 按时间范围过滤
	if !filter.StartTime.IsZero() {
		query = query.Where("created_at >= ?", filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		query = query.Where("created_at <= ?", filter.EndTime)
	}

	// 2. 查询数据库
	var taskModels []model.TaskModel
	if err := query.Find(&taskModels).Error; err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}

	// 3. 反序列化任务并应用审批人过滤
	var results []*task.Task
	for _, taskModel := range taskModels {
		var tsk task.Task
		if err := json.Unmarshal(taskModel.Data, &tsk); err != nil {
			// 跳过无法反序列化的任务
			continue
		}

		// 按审批人过滤(如果指定了审批人)
		if filter.Approver != "" {
			found := false
			if tsk.Approvers != nil {
				for _, approvers := range tsk.Approvers {
					for _, approver := range approvers {
						if approver == filter.Approver {
							found = true
							break
						}
					}
					if found {
						break
					}
				}
			}
			if !found {
				// 该任务不包含指定的审批人,跳过
				continue
			}
		}

		results = append(results, &tsk)
	}

	return results, nil
}

// findStartNode 查找模板中的开始节点
// 从 pkg/template 导入节点类型,实现节点查找逻辑
func findStartNode(tpl *template.Template) string {
	for _, node := range tpl.Nodes {
		if node.Type == template.NodeTypeStart {
			return node.ID
		}
	}
	// 如果找不到开始节点,返回空字符串
	// 这应该不会发生,因为模板验证已经确保有开始节点
	return ""
}

// findNextNode 查找指定节点的下一个节点
// 从 pkg/template 导入边类型,实现节点查找逻辑
// saveStateHistory 保存状态历史到数据库
func (m *dbTaskManager) saveStateHistory(taskID string, fromState types.TaskState, toState types.TaskState, reason string, operator string) error {
	historyModel := &model.StateHistoryModel{
		ID:        generateHistoryID(),
		TaskID:    taskID,
		FromState: string(fromState),
		ToState:   string(toState),
		Reason:    reason,
		Operator:  operator,
		CreatedAt: time.Now(),
	}
	return m.historyRepo.Save(historyModel)
}

// generateHistoryID 生成状态历史 ID
func generateHistoryID() string {
	return fmt.Sprintf("hist-%d", time.Now().UnixNano())
}

func findNextNode(tpl *template.Template, nodeID string) string {
	for _, edge := range tpl.Edges {
		if edge.From == nodeID {
			return edge.To
		}
	}
	return ""
}

// HandleTimeout 处理任务超时
func (m *dbTaskManager) HandleTimeout(id string) error {
	// 1. 获取任务
	tsk, err := m.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 2. 检查任务是否超时
	// 只有 submitted 或 approving 状态的任务才需要检查超时
	currentState := tsk.GetState()
	if currentState != types.TaskStateSubmitted && currentState != types.TaskStateApproving {
		// 任务不在需要检查超时的状态,直接返回
		return nil
	}

	// 3. 获取模板和当前节点
	tpl, err := m.templateMgr.Get(tsk.TemplateID, 0)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	currentNodeID := tsk.CurrentNode
	if currentNodeID == "" {
		// 没有当前节点,无法检查超时
		return nil
	}

	node, exists := tpl.Nodes[currentNodeID]
	if !exists {
		// 节点不存在,无法检查超时
		return nil
	}

	// 4. 检查节点类型和配置
	if node.Type != template.NodeTypeApproval {
		// 只有审批节点才需要检查超时
		return nil
	}

	// 5. 获取审批节点配置
	config, ok := node.Config.(template.ApprovalNodeConfigAccessor)
	if !ok {
		// 节点配置不是审批节点配置,无法检查超时
		return nil
	}

	// 6. 检查是否配置了超时
	timeout := config.GetTimeout()
	if timeout == nil {
		// 未配置超时,直接返回
		return nil
	}

	// 7. 计算开始时间(使用提交时间或创建时间)
	startTime := tsk.CreatedAt
	if tsk.SubmittedAt != nil {
		startTime = *tsk.SubmittedAt
	}

	// 8. 检查是否超时
	if time.Since(startTime) <= *timeout {
		// 未超时,直接返回
		return nil
	}

	// 9. 任务已超时,验证当前状态允许转换为超时状态
	if !m.stateMachine.CanTransition(currentState, types.TaskStateTimeout) {
		return fmt.Errorf("invalid state transition: cannot timeout task in state %q", currentState)
	}

	// 10. 使用状态机执行状态转换
	adapter := &taskAdapter{task: tsk}
	newTaskAdapter, err := m.stateMachine.Transition(adapter, types.TaskStateTimeout, "task timeout")
	if err != nil {
		return fmt.Errorf("state transition failed: %w", err)
	}

	// 11. 获取转换后的任务对象
	newTask := newTaskAdapter.(*taskAdapter).task
	newTask.UpdatedAt = time.Now()

	// 12. 序列化并保存到数据库
	data, err := json.Marshal(newTask)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	taskModel := &model.TaskModel{
		ID:             newTask.ID,
		TemplateID:     newTask.TemplateID,
		TemplateVersion: newTask.TemplateVersion,
		BusinessID:     newTask.BusinessID,
		State:          string(newTask.State),
		CurrentNode:    newTask.CurrentNode,
		Data:           data,
		CreatedAt:      newTask.CreatedAt,
		UpdatedAt:      newTask.UpdatedAt,
		SubmittedAt:    newTask.SubmittedAt,
	}

	if err := m.db.Model(&model.TaskModel{}).Where("id = ?", id).Updates(taskModel).Error; err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// 13. 生成超时事件
	if m.eventHandler != nil {
		// 事件生成逻辑将在后续实现
	}

	return nil
}

// Pause 暂停任务
// 只有 pending、submitted、approving 状态可以暂停
// 暂停时会记录暂停前的状态,用于恢复时恢复到正确状态
func (m *dbTaskManager) Pause(id string, reason string) error {
	// 1. 获取任务
	tsk, err := m.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 2. 检查当前状态是否允许暂停
	currentState := tsk.GetState()
	if !m.stateMachine.CanTransition(currentState, types.TaskStatePaused) {
		return fmt.Errorf("cannot pause task in state %q", currentState)
	}

	// 3. 记录暂停前的状态
	pausedState := currentState

	// 4. 使用状态机执行状态转换
	adapter := &taskAdapter{task: tsk}
	newTaskAdapter, err := m.stateMachine.Transition(adapter, types.TaskStatePaused, reason)
	if err != nil {
		return fmt.Errorf("state transition failed: %w", err)
	}

	// 5. 获取转换后的任务对象
	newTask := newTaskAdapter.(*taskAdapter).task

	// 6. 设置暂停相关字段
	now := time.Now()
	newTask.PausedAt = &now
	newTask.PausedState = pausedState
	newTask.UpdatedAt = now

	// 7. 序列化并保存到数据库
	data, err := json.Marshal(newTask)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	taskModel := &model.TaskModel{
		ID:             newTask.ID,
		TemplateID:     newTask.TemplateID,
		TemplateVersion: newTask.TemplateVersion,
		BusinessID:     newTask.BusinessID,
		State:          string(newTask.State),
		CurrentNode:    newTask.CurrentNode,
		Data:           data,
		CreatedAt:      newTask.CreatedAt,
		UpdatedAt:      newTask.UpdatedAt,
		SubmittedAt:    newTask.SubmittedAt,
	}

	if err := m.db.Model(&model.TaskModel{}).Where("id = ?", id).Updates(taskModel).Error; err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// 8. 生成暂停事件
	if m.eventHandler != nil {
		// 事件生成逻辑将在后续实现
	}

	return nil
}

// Resume 恢复任务
// 只有 paused 状态可以恢复
// 恢复时会恢复到暂停前的状态(pending、submitted 或 approving)
func (m *dbTaskManager) Resume(id string, reason string) error {
	// 1. 获取任务
	tsk, err := m.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 2. 检查当前状态是否是 paused
	currentState := tsk.GetState()
	if currentState != types.TaskStatePaused {
		return fmt.Errorf("cannot resume task in state %q, only paused tasks can be resumed", currentState)
	}

	// 3. 获取暂停前的状态
	targetState := tsk.PausedState
	if targetState == "" {
		// 如果没有记录暂停前状态,默认恢复到 pending
		targetState = types.TaskStatePending
	}

	// 4. 验证恢复状态转换的合法性
	if !m.stateMachine.CanTransition(types.TaskStatePaused, targetState) {
		return fmt.Errorf("cannot resume task to state %q from paused state", targetState)
	}

	// 5. 使用状态机执行状态转换
	adapter := &taskAdapter{task: tsk}
	newTaskAdapter, err := m.stateMachine.Transition(adapter, targetState, reason)
	if err != nil {
		return fmt.Errorf("state transition failed: %w", err)
	}

	// 6. 获取转换后的任务对象
	newTask := newTaskAdapter.(*taskAdapter).task

	// 7. 清除暂停相关字段
	newTask.PausedAt = nil
	newTask.PausedState = types.TaskState("") // 清除暂停前状态
	newTask.UpdatedAt = time.Now()

	// 8. 序列化并保存到数据库
	data, err := json.Marshal(newTask)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	taskModel := &model.TaskModel{
		ID:             newTask.ID,
		TemplateID:     newTask.TemplateID,
		TemplateVersion: newTask.TemplateVersion,
		BusinessID:     newTask.BusinessID,
		State:          string(newTask.State),
		CurrentNode:    newTask.CurrentNode,
		Data:           data,
		CreatedAt:      newTask.CreatedAt,
		UpdatedAt:      newTask.UpdatedAt,
		SubmittedAt:    newTask.SubmittedAt,
	}

	if err := m.db.Model(&model.TaskModel{}).Where("id = ?", id).Updates(taskModel).Error; err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// 9. 生成恢复事件
	if m.eventHandler != nil {
		// 事件生成逻辑将在后续实现
	}

	return nil
}

// RollbackToNode 回退到指定节点
// 只能回退到已完成的节点
// 回退时会清理回退节点之后的审批记录和状态
func (m *dbTaskManager) RollbackToNode(id string, nodeID string, reason string) error {
	// 1. 获取任务
	tsk, err := m.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 2. 获取模板
	tpl, err := m.templateMgr.Get(tsk.TemplateID, tsk.TemplateVersion)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	// 3. 验证节点存在
	node, exists := tpl.Nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %q not found in template", nodeID)
	}

	// 4. 验证节点已完成
	completed := false
	for _, completedNodeID := range tsk.CompletedNodes {
		if completedNodeID == nodeID {
			completed = true
			break
		}
	}
	if !completed {
		return fmt.Errorf("node %q is not completed, cannot rollback", nodeID)
	}

	// 5. 找到回退节点在已完成节点列表中的位置
	rollbackIndex := -1
	for i, completedNodeID := range tsk.CompletedNodes {
		if completedNodeID == nodeID {
			rollbackIndex = i
			break
		}
	}
	if rollbackIndex == -1 {
		return fmt.Errorf("node %q is not in completed nodes list", nodeID)
	}

	// 6. 构建需要保留的节点集合(回退节点及之前的节点)
	keepNodes := make(map[string]bool)
	for i := 0; i <= rollbackIndex; i++ {
		keepNodes[tsk.CompletedNodes[i]] = true
	}

	// 7. 清理回退节点之后的审批记录和状态
	// 7.1 移除回退节点之后的审批记录
	var filteredRecords []*task.Record
	for _, record := range tsk.Records {
		if keepNodes[record.NodeID] {
			filteredRecords = append(filteredRecords, record)
		}
	}
	tsk.Records = filteredRecords

	// 7.2 清除回退节点之后的节点输出数据
	newNodeOutputs := make(map[string]json.RawMessage)
	for k, v := range tsk.NodeOutputs {
		if keepNodes[k] {
			newNodeOutputs[k] = v
		}
	}
	tsk.NodeOutputs = newNodeOutputs

	// 7.3 清除回退节点之后的审批人列表和审批结果
	newApprovers := make(map[string][]string)
	newApprovals := make(map[string]map[string]*task.Approval)
	for k, v := range tsk.Approvers {
		if keepNodes[k] {
			newApprovers[k] = v
			if approvals, exists := tsk.Approvals[k]; exists {
				newApprovals[k] = approvals
			}
		}
	}
	tsk.Approvers = newApprovers
	tsk.Approvals = newApprovals

	// 8. 更新当前节点为回退的目标节点
	tsk.CurrentNode = nodeID

	// 9. 更新已完成节点列表,移除回退节点之后的节点
	tsk.CompletedNodes = tsk.CompletedNodes[:rollbackIndex+1]

	// 10. 更新任务状态
	// 根据回退的节点类型确定新的状态
	var targetState types.TaskState
	switch node.Type {
	case template.NodeTypeStart:
		targetState = types.TaskStatePending
	case template.NodeTypeApproval:
		targetState = types.TaskStateApproving
	case template.NodeTypeCondition:
		targetState = types.TaskStateApproving
	case template.NodeTypeEnd:
		return fmt.Errorf("cannot rollback to end node")
	default:
		targetState = types.TaskStateApproving
	}

	// 11. 更新任务状态
	// 回退操作允许从终态回退,所以需要特殊处理
	currentState := tsk.GetState()
	
	// 如果当前是终态(approved/rejected/cancelled/timeout),允许回退
	// 这种情况下不通过状态机,直接设置状态
	if currentState == types.TaskStateApproved || 
		currentState == types.TaskStateRejected || 
		currentState == types.TaskStateCancelled || 
		currentState == types.TaskStateTimeout {
		tsk.SetState(targetState)
		tsk.AddStateChangeRecord(currentState, targetState, reason, time.Now())
	} else {
		// 如果当前不是终态,使用状态机进行状态转换
		if !m.stateMachine.CanTransition(currentState, targetState) {
			return fmt.Errorf("cannot rollback from state %q to state %q", currentState, targetState)
		}
		adapter := &taskAdapter{task: tsk}
		newTaskAdapter, err := m.stateMachine.Transition(adapter, targetState, reason)
		if err != nil {
			return fmt.Errorf("state transition failed: %w", err)
		}
		tsk = newTaskAdapter.(*taskAdapter).task
	}

	// 12. 更新任务更新时间
	tsk.UpdatedAt = time.Now()

	// 13. 序列化并保存到数据库
	data, err := json.Marshal(tsk)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	taskModel := &model.TaskModel{
		ID:             tsk.ID,
		TemplateID:     tsk.TemplateID,
		TemplateVersion: tsk.TemplateVersion,
		BusinessID:     tsk.BusinessID,
		State:          string(tsk.State),
		CurrentNode:    tsk.CurrentNode,
		Data:           data,
		CreatedAt:      tsk.CreatedAt,
		UpdatedAt:      tsk.UpdatedAt,
		SubmittedAt:    tsk.SubmittedAt,
	}

	if err := m.db.Model(&model.TaskModel{}).Where("id = ?", id).Updates(taskModel).Error; err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// 14. 生成回退事件
	if m.eventHandler != nil {
		// 事件生成逻辑将在后续实现
	}

	return nil
}

// ReplaceApprover 替换审批人
// 只能替换尚未审批的审批人
// 替换后会保留原审批人的审批记录(如果有),新审批人可以继续审批
func (m *dbTaskManager) ReplaceApprover(id string, nodeID string, oldApprover string, newApprover string, reason string) error {
	// 1. 获取任务
	tsk, err := m.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 2. 获取模板
	tpl, err := m.templateMgr.Get(tsk.TemplateID, 0)
	if err != nil {
		return fmt.Errorf("failed to get template %q: %w", tsk.TemplateID, err)
	}

	// 3. 获取节点配置
	node, exists := tpl.Nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %q not found in template", nodeID)
	}

	// 4. 检查节点类型是否为审批节点
	if node.Type != template.NodeTypeApproval {
		return fmt.Errorf("node %q is not an approval node", nodeID)
	}

	// 5. 检查节点是否已激活(当前节点或已完成节点)
	nodeActivated := false
	if tsk.CurrentNode == nodeID {
		nodeActivated = true
	} else {
		// 检查节点是否在已完成节点列表中
		for _, completedNodeID := range tsk.CompletedNodes {
			if completedNodeID == nodeID {
				nodeActivated = true
				break
			}
		}
	}
	if !nodeActivated {
		return fmt.Errorf("node %q is not activated (current node: %q)", nodeID, tsk.CurrentNode)
	}

	// 6. 检查原审批人是否在审批人列表中
	approvers, exists := tsk.Approvers[nodeID]
	if !exists {
		return fmt.Errorf("approvers not found for node %q", nodeID)
	}

	found := false
	for _, approver := range approvers {
		if approver == oldApprover {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("user %q is not an approver for node %q", oldApprover, nodeID)
	}

	// 7. 检查原审批人是否尚未审批
	if tsk.Approvals != nil && tsk.Approvals[nodeID] != nil {
		if approval, exists := tsk.Approvals[nodeID][oldApprover]; exists && approval != nil {
			return fmt.Errorf("user %q has already approved, cannot replace", oldApprover)
		}
	}

	// 8. 替换审批人(移除原审批人,添加新审批人)
	newApprovers := make([]string, 0, len(approvers))
	for _, approver := range approvers {
		if approver != oldApprover {
			newApprovers = append(newApprovers, approver)
		}
	}
	// 检查新审批人是否已在列表中
	newApproverExists := false
	for _, approver := range newApprovers {
		if approver == newApprover {
			newApproverExists = true
			break
		}
	}
	if !newApproverExists {
		newApprovers = append(newApprovers, newApprover)
	}
	tsk.Approvers[nodeID] = newApprovers

	// 9. 生成替换审批人记录
	record := &task.Record{
		ID:         generateRecordID(),
		TaskID:     id,
		NodeID:     nodeID,
		Approver:   oldApprover,
		Result:     "replace",
		Comment:    fmt.Sprintf("替换为 %s: %s", newApprover, reason),
		CreatedAt:  time.Now(),
		Attachments: []string{},
	}

	// 添加到记录列表
	tsk.Records = append(tsk.Records, record)

	// 10. 更新任务更新时间
	tsk.UpdatedAt = time.Now()

	// 11. 序列化并保存到数据库
	data, err := json.Marshal(tsk)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	taskModel := &model.TaskModel{
		ID:             tsk.ID,
		TemplateID:     tsk.TemplateID,
		TemplateVersion: tsk.TemplateVersion,
		BusinessID:     tsk.BusinessID,
		State:          string(tsk.State),
		CurrentNode:    tsk.CurrentNode,
		Data:           data,
		CreatedAt:      tsk.CreatedAt,
		UpdatedAt:      tsk.UpdatedAt,
		SubmittedAt:    tsk.SubmittedAt,
	}

	if err := m.db.Model(&model.TaskModel{}).Where("id = ?", id).Updates(taskModel).Error; err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// 12. 生成替换审批人事件
	if m.eventHandler != nil {
		// 事件生成逻辑将在后续实现
	}

	return nil
}

