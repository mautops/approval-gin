package api

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/service"
	"github.com/mautops/approval-gin/internal/utils"
)

// TaskController 任务控制器
type TaskController struct {
	taskService service.TaskService
}

// NewTaskController 创建任务控制器
func NewTaskController(taskService service.TaskService) *TaskController {
	return &TaskController{
		taskService: taskService,
	}
}

// validateTaskID 验证任务 ID 并返回错误响应（如果无效）
func (c *TaskController) validateTaskID(ctx *gin.Context, id string) bool {
	if err := utils.ValidateTaskID(id); err != nil {
		Error(ctx, http.StatusBadRequest, "invalid task ID", err.Error())
		return false
	}
	return true
}

// handleServiceError 统一处理服务层错误
func (c *TaskController) handleServiceError(ctx *gin.Context, err error, operation string) bool {
	if err != nil {
		Error(ctx, http.StatusInternalServerError, "failed to "+operation, err.Error())
		return false
	}
	return true
}

// Create 创建任务
// @Summary      创建审批任务
// @Description  基于模板创建新的审批任务
// @Tags         任务管理
// @Accept       json
// @Produce      json
// @Param        request body service.CreateTaskRequest true "任务信息"
// @Success      200  {object}  Response
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /tasks [post]
// @Security    BearerAuth
func (c *TaskController) Create(ctx *gin.Context) {
	var req service.CreateTaskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		Error(ctx, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	task, err := c.taskService.Create(ctx.Request.Context(), &req)
	if err != nil {
		Error(ctx, http.StatusInternalServerError, "failed to create task", err.Error())
		return
	}

	Success(ctx, task)
}

// Get 获取任务
// @Summary      获取任务详情
// @Description  根据 ID 获取任务详情
// @Tags         任务管理
// @Accept       json
// @Produce      json
// @Param        id path string true "任务 ID"
// @Success      200  {object}  Response
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /tasks/{id} [get]
// @Security     BearerAuth
func (c *TaskController) Get(ctx *gin.Context) {
	id := ctx.Param("id")
	if !c.validateTaskID(ctx, id) {
		return
	}

	task, err := c.taskService.Get(id)
	if err != nil {
		Error(ctx, http.StatusNotFound, "task not found", err.Error())
		return
	}

	Success(ctx, task)
}

// Submit 提交任务
// @Summary      提交审批任务
// @Description  提交任务进入审批流程
// @Tags         任务管理
// @Accept       json
// @Produce      json
// @Param        id path string true "任务 ID"
// @Success      200  {object}  Response
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /tasks/{id}/submit [post]
// @Security     BearerAuth
func (c *TaskController) Submit(ctx *gin.Context) {
	id := ctx.Param("id")
	if !c.validateTaskID(ctx, id) {
		return
	}

	if !c.handleServiceError(ctx, c.taskService.Submit(ctx.Request.Context(), id), "submit task") {
		return
	}

	Success(ctx, nil)
}

// Approve 审批同意
// @Summary      审批同意
// @Description  审批人同意审批任务
// @Tags         任务管理
// @Accept       json
// @Produce      json
// @Param        id path string true "任务 ID"
// @Param        request body service.ApproveRequest true "审批信息"
// @Success      200  {object}  Response
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /tasks/{id}/approve [post]
// @Security     BearerAuth
func (c *TaskController) Approve(ctx *gin.Context) {
	id := ctx.Param("id")
	if !c.validateTaskID(ctx, id) {
		return
	}

	var req service.ApproveRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		Error(ctx, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	if !c.handleServiceError(ctx, c.taskService.Approve(ctx.Request.Context(), id, &req), "approve task") {
		return
	}

	Success(ctx, nil)
}

// Reject 审批拒绝
func (c *TaskController) Reject(ctx *gin.Context) {
	id := ctx.Param("id")
	if !c.validateTaskID(ctx, id) {
		return
	}

	var req service.RejectRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		Error(ctx, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	if !c.handleServiceError(ctx, c.taskService.Reject(ctx.Request.Context(), id, &req), "reject task") {
		return
	}

	Success(ctx, nil)
}

// Cancel 取消任务
func (c *TaskController) Cancel(ctx *gin.Context) {
	id := ctx.Param("id")
	if !c.validateTaskID(ctx, id) {
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		Error(ctx, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	if !c.handleServiceError(ctx, c.taskService.Cancel(ctx.Request.Context(), id, req.Reason), "cancel task") {
		return
	}

	Success(ctx, nil)
}

// Withdraw 撤回任务
func (c *TaskController) Withdraw(ctx *gin.Context) {
	id := ctx.Param("id")
	if !c.validateTaskID(ctx, id) {
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		Error(ctx, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	if !c.handleServiceError(ctx, c.taskService.Withdraw(ctx.Request.Context(), id, req.Reason), "withdraw task") {
		return
	}

	Success(ctx, nil)
}

// Transfer 转交审批
// @Summary      转交审批
// @Description  将审批任务从原审批人转交给新审批人
// @Tags         任务管理
// @Accept       json
// @Produce      json
// @Param        id path string true "任务 ID"
// @Param        request body service.TransferRequest true "转交信息"
// @Success      200  {object}  Response
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /tasks/{id}/transfer [post]
// @Security     BearerAuth
func (c *TaskController) Transfer(ctx *gin.Context) {
	id := ctx.Param("id")
	if !c.validateTaskID(ctx, id) {
		return
	}

	var req service.TransferRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		Error(ctx, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	if !c.handleServiceError(ctx, c.taskService.Transfer(ctx.Request.Context(), id, &req), "transfer task") {
		return
	}

	Success(ctx, nil)
}

// AddApprover 加签
// @Summary      加签
// @Description  在审批人列表中添加新的审批人
// @Tags         任务管理
// @Accept       json
// @Produce      json
// @Param        id path string true "任务 ID"
// @Param        request body service.AddApproverRequest true "加签信息"
// @Success      200  {object}  Response
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /tasks/{id}/approvers [post]
// @Security     BearerAuth
func (c *TaskController) AddApprover(ctx *gin.Context) {
	id := ctx.Param("id")
	if !c.validateTaskID(ctx, id) {
		return
	}

	var req service.AddApproverRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		Error(ctx, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	if !c.handleServiceError(ctx, c.taskService.AddApprover(ctx.Request.Context(), id, &req), "add approver") {
		return
	}

	Success(ctx, nil)
}

// RemoveApprover 减签
// @Summary      减签
// @Description  从审批人列表中移除指定的审批人
// @Tags         任务管理
// @Accept       json
// @Produce      json
// @Param        id path string true "任务 ID"
// @Param        request body service.RemoveApproverRequest true "减签信息"
// @Success      200  {object}  Response
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /tasks/{id}/approvers [delete]
// @Security     BearerAuth
func (c *TaskController) RemoveApprover(ctx *gin.Context) {
	id := ctx.Param("id")
	if !c.validateTaskID(ctx, id) {
		return
	}

	var req service.RemoveApproverRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		Error(ctx, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	if !c.handleServiceError(ctx, c.taskService.RemoveApprover(ctx.Request.Context(), id, &req), "remove approver") {
		return
	}

	Success(ctx, nil)
}

// Pause 暂停任务
// @Summary      暂停任务
// @Description  暂停任务,只有 pending、submitted、approving 状态可以暂停
// @Tags         任务管理
// @Accept       json
// @Produce      json
// @Param        id path string true "任务 ID"
// @Param        request body object{reason=string} true "暂停原因"
// @Success      200  {object}  Response
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /tasks/{id}/pause [post]
// @Security     BearerAuth
func (c *TaskController) Pause(ctx *gin.Context) {
	id := ctx.Param("id")
	if !c.validateTaskID(ctx, id) {
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		Error(ctx, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	if !c.handleServiceError(ctx, c.taskService.Pause(ctx.Request.Context(), id, req.Reason), "pause task") {
		return
	}

	Success(ctx, nil)
}

// Resume 恢复任务
// @Summary      恢复任务
// @Description  恢复任务,只有 paused 状态可以恢复
// @Tags         任务管理
// @Accept       json
// @Produce      json
// @Param        id path string true "任务 ID"
// @Param        request body object{reason=string} true "恢复原因"
// @Success      200  {object}  Response
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /tasks/{id}/resume [post]
// @Security     BearerAuth
func (c *TaskController) Resume(ctx *gin.Context) {
	id := ctx.Param("id")
	if !c.validateTaskID(ctx, id) {
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		Error(ctx, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	if !c.handleServiceError(ctx, c.taskService.Resume(ctx.Request.Context(), id, req.Reason), "resume task") {
		return
	}

	Success(ctx, nil)
}

// RollbackToNode 回退到指定节点
// @Summary      回退到指定节点
// @Description  回退到指定节点,只能回退到已完成的节点
// @Tags         任务管理
// @Accept       json
// @Produce      json
// @Param        id path string true "任务 ID"
// @Param        request body service.RollbackRequest true "回退信息"
// @Success      200  {object}  Response
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /tasks/{id}/rollback [post]
// @Security     BearerAuth
func (c *TaskController) RollbackToNode(ctx *gin.Context) {
	id := ctx.Param("id")
	if !c.validateTaskID(ctx, id) {
		return
	}

	var req service.RollbackRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		Error(ctx, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	if !c.handleServiceError(ctx, c.taskService.RollbackToNode(ctx.Request.Context(), id, &req), "rollback task") {
		return
	}

	Success(ctx, nil)
}

// ReplaceApprover 替换审批人
// @Summary      替换审批人
// @Description  替换审批人,只能替换尚未审批的审批人
// @Tags         任务管理
// @Accept       json
// @Produce      json
// @Param        id path string true "任务 ID"
// @Param        request body service.ReplaceApproverRequest true "替换信息"
// @Success      200  {object}  Response
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /tasks/{id}/approvers/replace [post]
// @Security     BearerAuth
func (c *TaskController) ReplaceApprover(ctx *gin.Context) {
	id := ctx.Param("id")
	if !c.validateTaskID(ctx, id) {
		return
	}

	var req service.ReplaceApproverRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		Error(ctx, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	if !c.handleServiceError(ctx, c.taskService.ReplaceApprover(ctx.Request.Context(), id, &req), "replace approver") {
		return
	}

	Success(ctx, nil)
}

// BatchApprove 批量审批
// @Summary      批量审批任务
// @Description  批量审批多个任务
// @Tags         任务管理
// @Accept       json
// @Produce      json
// @Param        request body service.BatchApproveRequest true "批量审批请求"
// @Success      200  {object}  Response{data=[]service.BatchOperationResult}
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /tasks/batch/approve [post]
// @Security    BearerAuth
func (c *TaskController) BatchApprove(ctx *gin.Context) {
	var req service.BatchApproveRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		Error(ctx, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	results, err := c.taskService.BatchApprove(ctx.Request.Context(), &req)
	if err != nil {
		Error(ctx, http.StatusInternalServerError, "failed to batch approve", err.Error())
		return
	}

	Success(ctx, results)
}

// BatchTransfer 批量转交
// @Summary      批量转交任务
// @Description  批量转交多个任务
// @Tags         任务管理
// @Accept       json
// @Produce      json
// @Param        request body service.BatchTransferRequest true "批量转交请求"
// @Success      200  {object}  Response{data=[]service.BatchOperationResult}
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /tasks/batch/transfer [post]
// @Security    BearerAuth
func (c *TaskController) BatchTransfer(ctx *gin.Context) {
	var req service.BatchTransferRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		Error(ctx, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	results, err := c.taskService.BatchTransfer(ctx.Request.Context(), &req)
	if err != nil {
		Error(ctx, http.StatusInternalServerError, "failed to batch transfer", err.Error())
		return
	}

	Success(ctx, results)
}

// HandleTimeout 处理任务超时
// @Summary      处理任务超时
// @Description  处理任务超时,如果任务已超时,将任务状态转换为 timeout
// @Tags         任务管理
// @Accept       json
// @Produce      json
// @Param        id path string true "任务 ID"
// @Success      200  {object}  Response
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /tasks/{id}/timeout [post]
// @Security     BearerAuth
func (c *TaskController) HandleTimeout(ctx *gin.Context) {
	id := ctx.Param("id")
	if !c.validateTaskID(ctx, id) {
		return
	}

	if !c.handleServiceError(ctx, c.taskService.HandleTimeout(ctx.Request.Context(), id), "handle timeout") {
		return
	}

	Success(ctx, nil)
}

// Delete 删除任务
// @Summary      删除任务
// @Description  删除审批任务,只允许删除待审批或已取消状态的任务,且不能有审批记录
// @Tags         任务管理
// @Accept       json
// @Produce      json
// @Param        id path string true "任务 ID"
// @Success      200  {object}  Response
// @Failure      400  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /tasks/{id} [delete]
// @Security     BearerAuth
func (c *TaskController) Delete(ctx *gin.Context) {
	id := ctx.Param("id")
	// 调试日志
	log.Printf("[DEBUG] Delete method called with id: %s, path: %s", id, ctx.Request.URL.Path)
	if !c.validateTaskID(ctx, id) {
		return
	}

	if err := c.taskService.Delete(ctx.Request.Context(), id); err != nil {
		// 检查是否是任务不存在的错误
		if strings.Contains(err.Error(), "task not found") {
			Error(ctx, http.StatusNotFound, "task not found", err.Error())
			return
		}
		// 检查是否是权限或状态错误
		if strings.Contains(err.Error(), "无法删除任务") || strings.Contains(err.Error(), "权限不足") {
			Error(ctx, http.StatusForbidden, "无法删除任务", err.Error())
			return
		}
		c.handleServiceError(ctx, err, "delete task")
		return
	}

	Success(ctx, nil)
}
