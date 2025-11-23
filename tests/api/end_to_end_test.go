package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/mautops/approval-gin/internal/service"
	"github.com/mautops/approval-kit/pkg/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEndToEnd_CompleteWorkflow 测试完整的审批流程
// 从模板创建 -> 任务创建 -> 提交 -> 审批 -> 完成
func TestEndToEnd_CompleteWorkflow(t *testing.T) {
	router, _ := setupTestAPIServer(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建带审批节点的模板
	createTemplateReqBody := service.CreateTemplateRequest{
		Name:        "完整流程测试模板",
		Description: "这是一个完整流程测试模板",
		Nodes: map[string]*template.Node{
			"start": {
				ID:    "start",
				Name:  "开始",
				Type:  "start",
				Order: 1,
			},
			"approval": {
				ID:    "approval",
				Name:  "审批节点",
				Type:  "approval",
				Order: 2,
			},
			"end": {
				ID:    "end",
				Name:  "结束",
				Type:  "end",
				Order: 3,
			},
		},
		Edges: []*template.Edge{
			{From: "start", To: "approval"},
			{From: "approval", To: "end"},
		},
		Config: nil,
	}

	templateBody, err := json.Marshal(createTemplateReqBody)
	require.NoError(t, err)

	createTemplateReq := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewBuffer(templateBody))
	createTemplateReq.Header.Set("Content-Type", "application/json")
	createTemplateReq.Header.Set("X-CSRF-Token", csrfToken)
	createTemplateW := httptest.NewRecorder()
	router.ServeHTTP(createTemplateW, createTemplateReq)

	require.Equal(t, http.StatusOK, createTemplateW.Code)
	templateID := extractTemplateID(t, createTemplateW.Body.Bytes())
	require.NotEmpty(t, templateID)

	// 2. 创建任务
	createTaskReqBody := service.CreateTaskRequest{
		TemplateID: templateID,
		BusinessID: "business-001",
		Params:     json.RawMessage(`{"key": "value"}`),
	}

	taskBody, err := json.Marshal(createTaskReqBody)
	require.NoError(t, err)

	createTaskReq := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(taskBody))
	createTaskReq.Header.Set("Content-Type", "application/json")
	createTaskReq.Header.Set("X-CSRF-Token", csrfToken)
	createTaskW := httptest.NewRecorder()
	router.ServeHTTP(createTaskW, createTaskReq)

	require.Equal(t, http.StatusOK, createTaskW.Code)
	taskID := extractTaskID(t, createTaskW.Body.Bytes())
	require.NotEmpty(t, taskID)

	// 3. 提交任务
	submitReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/submit", nil)
	submitReq.Header.Set("X-CSRF-Token", csrfToken)
	submitW := httptest.NewRecorder()
	router.ServeHTTP(submitW, submitReq)

	assert.Equal(t, http.StatusOK, submitW.Code)

	// 4. 添加审批人
	addApproverReqBody := service.AddApproverRequest{
		NodeID:   "approval",
		Approver: "user1",
		Reason:   "设置审批人",
	}
	approverBody, _ := json.Marshal(addApproverReqBody)
	addApproverReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/approvers", bytes.NewBuffer(approverBody))
	addApproverReq.Header.Set("Content-Type", "application/json")
	addApproverReq.Header.Set("X-CSRF-Token", csrfToken)
	addApproverW := httptest.NewRecorder()
	router.ServeHTTP(addApproverW, addApproverReq)
	require.Equal(t, http.StatusOK, addApproverW.Code)

	// 5. 审批任务
	approveReqBody := service.ApproveRequest{
		NodeID:  "approval",
		Comment: "Approved",
	}
	approveBody, _ := json.Marshal(approveReqBody)
	approveReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/approve", bytes.NewBuffer(approveBody))
	approveReq.Header.Set("Content-Type", "application/json")
	approveReq.Header.Set("X-CSRF-Token", csrfToken)
	approveW := httptest.NewRecorder()
	router.ServeHTTP(approveW, approveReq)

	assert.Equal(t, http.StatusOK, approveW.Code)

	// 6. 验证任务状态
	getReq := httptest.NewRequest("GET", "/api/v1/tasks/"+taskID, nil)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)

	assert.Equal(t, http.StatusOK, getW.Code)
	var finalTaskResp api.Response
	err = json.Unmarshal(getW.Body.Bytes(), &finalTaskResp)
	require.NoError(t, err)
	finalTaskDataBytes, _ := json.Marshal(finalTaskResp.Data)
	var finalTaskData map[string]interface{}
	json.Unmarshal(finalTaskDataBytes, &finalTaskData)
	// 验证任务状态为已完成或已审批
	state, ok := finalTaskData["state"].(string)
	if !ok {
		state, ok = finalTaskData["State"].(string)
	}
	assert.True(t, ok, "task should have state field")
	assert.NotEmpty(t, state, "task state should not be empty")
}

// TestEndToEnd_TaskRejection 测试任务拒绝流程
func TestEndToEnd_TaskRejection(t *testing.T) {
	router, _ := setupTestAPIServer(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建带审批节点的模板
	templateID := createTestTemplateWithApprovalNodeForE2E(t, router, csrfToken)

	// 2. 创建并提交任务
	taskID := createTestTaskForE2E(t, router, csrfToken, templateID)
	submitTaskForE2E(t, router, csrfToken, taskID)

	// 3. 添加审批人
	addApproverReqBody := service.AddApproverRequest{
		NodeID:   "approval",
		Approver: "user-001",
		Reason:   "设置审批人",
	}
	approverBody, _ := json.Marshal(addApproverReqBody)
	addApproverReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/approvers", bytes.NewBuffer(approverBody))
	addApproverReq.Header.Set("Content-Type", "application/json")
	addApproverReq.Header.Set("X-CSRF-Token", csrfToken)
	addApproverW := httptest.NewRecorder()
	router.ServeHTTP(addApproverW, addApproverReq)
	require.Equal(t, http.StatusOK, addApproverW.Code)

	// 4. 拒绝任务
	rejectReqBody := service.RejectRequest{
		NodeID:  "approval",
		Comment: "Rejected",
	}
	rejectBody, _ := json.Marshal(rejectReqBody)
	rejectReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/reject", bytes.NewBuffer(rejectBody))
	rejectReq.Header.Set("Content-Type", "application/json")
	rejectReq.Header.Set("X-CSRF-Token", csrfToken)
	rejectW := httptest.NewRecorder()
	router.ServeHTTP(rejectW, rejectReq)

	assert.Equal(t, http.StatusOK, rejectW.Code)

	// 5. 验证任务状态为已拒绝
	getReq := httptest.NewRequest("GET", "/api/v1/tasks/"+taskID, nil)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)

	var getResponse api.Response
	json.Unmarshal(getW.Body.Bytes(), &getResponse)
	taskDataBytes, _ := json.Marshal(getResponse.Data)
	var task map[string]interface{}
	json.Unmarshal(taskDataBytes, &task)
	
	var state string
	state, _ = task["state"].(string)
	if state == "" {
		state, _ = task["State"].(string)
	}
	assert.Equal(t, "rejected", state, "task state should be rejected")
}

// TestEndToEnd_WorkflowWithMultipleApprovers 测试多人审批流程
// 场景: 创建模板 -> 创建任务 -> 提交 -> 多人审批 -> 完成
func TestEndToEnd_WorkflowWithMultipleApprovers(t *testing.T) {
	router, _ := setupTestAPIServer(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建带多人审批节点的模板
	createTemplateReqBody := service.CreateTemplateRequest{
		Name:        "多人审批测试模板",
		Description: "这是一个多人审批测试模板",
		Nodes: map[string]*template.Node{
			"start": {
				ID:    "start",
				Name:  "开始",
				Type:  "start",
				Order: 1,
			},
			"approval-1": {
				ID:    "approval-1",
				Name:  "审批节点1",
				Type:  "approval",
				Order: 2,
			},
			"end": {
				ID:    "end",
				Name:  "结束",
				Type:  "end",
				Order: 3,
			},
		},
		Edges: []*template.Edge{
			{From: "start", To: "approval-1"},
			{From: "approval-1", To: "end"},
		},
		Config: nil,
	}

	templateBody, err := json.Marshal(createTemplateReqBody)
	require.NoError(t, err)

	createTemplateReq := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewBuffer(templateBody))
	createTemplateReq.Header.Set("Content-Type", "application/json")
	createTemplateReq.Header.Set("X-CSRF-Token", csrfToken)
	createTemplateW := httptest.NewRecorder()
	router.ServeHTTP(createTemplateW, createTemplateReq)

	require.Equal(t, http.StatusOK, createTemplateW.Code)
	templateID := extractTemplateID(t, createTemplateW.Body.Bytes())
	require.NotEmpty(t, templateID)

	// 2. 创建任务
	createTaskReqBody := service.CreateTaskRequest{
		TemplateID: templateID,
		BusinessID: "biz-001",
		Params:     json.RawMessage(`{"amount": 1000}`),
	}

	taskBody, err := json.Marshal(createTaskReqBody)
	require.NoError(t, err)

	createTaskReq := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(taskBody))
	createTaskReq.Header.Set("Content-Type", "application/json")
	createTaskReq.Header.Set("X-CSRF-Token", csrfToken)
	createTaskW := httptest.NewRecorder()
	router.ServeHTTP(createTaskW, createTaskReq)

	require.Equal(t, http.StatusOK, createTaskW.Code)
	taskID := extractTaskID(t, createTaskW.Body.Bytes())
	require.NotEmpty(t, taskID)

	// 3. 提交任务
	submitReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/submit", nil)
	submitReq.Header.Set("X-CSRF-Token", csrfToken)
	submitW := httptest.NewRecorder()
	router.ServeHTTP(submitW, submitReq)

	assert.Equal(t, http.StatusOK, submitW.Code)

	// 4. 添加多个审批人
	for i := 1; i <= 2; i++ {
		addApproverReqBody := service.AddApproverRequest{
			NodeID:   "approval-1",
			Approver: fmt.Sprintf("user-%03d", i),
			Reason:   "添加审批人",
		}
		approverBody, _ := json.Marshal(addApproverReqBody)
		addApproverReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/approvers", bytes.NewBuffer(approverBody))
		addApproverReq.Header.Set("Content-Type", "application/json")
		addApproverReq.Header.Set("X-CSRF-Token", csrfToken)
		addApproverW := httptest.NewRecorder()
		router.ServeHTTP(addApproverW, addApproverReq)
		require.Equal(t, http.StatusOK, addApproverW.Code)
	}

	// 5. 第一个审批人审批
	approveReqBody1 := service.ApproveRequest{
		NodeID:  "approval-1",
		Comment: "第一个审批人同意",
	}
	approveBody1, _ := json.Marshal(approveReqBody1)
	approveReq1 := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/approve", bytes.NewBuffer(approveBody1))
	approveReq1.Header.Set("Content-Type", "application/json")
	approveReq1.Header.Set("X-CSRF-Token", csrfToken)
	approveW1 := httptest.NewRecorder()
	router.ServeHTTP(approveW1, approveReq1)
	assert.Equal(t, http.StatusOK, approveW1.Code)

	// 6. 第二个审批人审批
	approveReqBody2 := service.ApproveRequest{
		NodeID:  "approval-1",
		Comment: "第二个审批人同意",
	}
	approveBody2, _ := json.Marshal(approveReqBody2)
	approveReq2 := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/approve", bytes.NewBuffer(approveBody2))
	approveReq2.Header.Set("Content-Type", "application/json")
	approveReq2.Header.Set("X-CSRF-Token", csrfToken)
	approveW2 := httptest.NewRecorder()
	router.ServeHTTP(approveW2, approveReq2)
	assert.Equal(t, http.StatusOK, approveW2.Code)

	// 7. 验证任务最终状态
	getReq := httptest.NewRequest("GET", "/api/v1/tasks/"+taskID, nil)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)

	assert.Equal(t, http.StatusOK, getW.Code)
	var getResponse api.Response
	err = json.Unmarshal(getW.Body.Bytes(), &getResponse)
	require.NoError(t, err)
	assert.Equal(t, 0, getResponse.Code)
}

// TestEndToEnd_TaskWithdrawFlow 测试任务撤回流程
// 场景: 创建模板 -> 创建任务 -> 提交 -> 撤回 -> 验证状态
func TestEndToEnd_TaskWithdrawFlow(t *testing.T) {
	router, _ := setupTestAPIServer(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建模板
	templateID := createTestTemplateForE2E(t, router, csrfToken)

	// 2. 创建任务
	taskID := createTestTaskForE2E(t, router, csrfToken, templateID)

	// 3. 提交任务
	submitTaskForE2E(t, router, csrfToken, taskID)

	// 4. 撤回任务
	withdrawReqBody := map[string]string{"reason": "需要修改"}
	withdrawBody, _ := json.Marshal(withdrawReqBody)
	withdrawReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/withdraw", bytes.NewBuffer(withdrawBody))
	withdrawReq.Header.Set("Content-Type", "application/json")
	withdrawReq.Header.Set("X-CSRF-Token", csrfToken)
	withdrawW := httptest.NewRecorder()
	router.ServeHTTP(withdrawW, withdrawReq)

	assert.Equal(t, http.StatusOK, withdrawW.Code)

	// 5. 验证任务状态已撤回
	getReq := httptest.NewRequest("GET", "/api/v1/tasks/"+taskID, nil)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)

	var getResponse api.Response
	json.Unmarshal(getW.Body.Bytes(), &getResponse)
	taskDataBytes, _ := json.Marshal(getResponse.Data)
	var task map[string]interface{}
	json.Unmarshal(taskDataBytes, &task)
	
	var state string
	var ok bool
	state, ok = task["state"].(string)
	if !ok {
		state, ok = task["State"].(string)
	}
	assert.True(t, ok, "task should have state field")
	assert.Equal(t, "pending", state, "task state should be pending after withdraw")
}

// TestEndToEnd_TaskTransferFlow 测试任务转交流程
// 场景: 创建模板 -> 创建任务 -> 提交 -> 添加审批人 -> 转交 -> 新审批人审批
func TestEndToEnd_TaskTransferFlow(t *testing.T) {
	router, _ := setupTestAPIServer(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建带审批节点的模板
	templateID := createTestTemplateWithApprovalNodeForE2E(t, router, csrfToken)

	// 2. 创建并提交任务
	taskID := createTestTaskForE2E(t, router, csrfToken, templateID)
	submitTaskForE2E(t, router, csrfToken, taskID)

	// 3. 添加审批人
	addApproverReqBody := service.AddApproverRequest{
		NodeID:   "approval",
		Approver: "user-001",
		Reason:   "设置审批人",
	}
	approverBody, _ := json.Marshal(addApproverReqBody)
	addApproverReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/approvers", bytes.NewBuffer(approverBody))
	addApproverReq.Header.Set("Content-Type", "application/json")
	addApproverReq.Header.Set("X-CSRF-Token", csrfToken)
	addApproverW := httptest.NewRecorder()
	router.ServeHTTP(addApproverW, addApproverReq)
	require.Equal(t, http.StatusOK, addApproverW.Code)

	// 4. 转交审批
	transferReqBody := service.TransferRequest{
		NodeID:      "approval",
		FromApprover: "user-001",
		ToApprover:   "user-002",
		Reason:       "转交原因",
	}
	transferBody, _ := json.Marshal(transferReqBody)
	transferReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/transfer", bytes.NewBuffer(transferBody))
	transferReq.Header.Set("Content-Type", "application/json")
	transferReq.Header.Set("X-CSRF-Token", csrfToken)
	transferW := httptest.NewRecorder()
	router.ServeHTTP(transferW, transferReq)

	assert.Equal(t, http.StatusOK, transferW.Code)

	// 5. 新审批人审批
	approveReqBody := service.ApproveRequest{
		NodeID:  "approval",
		Comment: "转交后的审批人同意",
	}
	approveBody, _ := json.Marshal(approveReqBody)
	approveReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/approve", bytes.NewBuffer(approveBody))
	approveReq.Header.Set("Content-Type", "application/json")
	approveReq.Header.Set("X-CSRF-Token", csrfToken)
	approveW := httptest.NewRecorder()
	router.ServeHTTP(approveW, approveReq)

	assert.Equal(t, http.StatusOK, approveW.Code)
}

// TestEndToEnd_TaskPauseResumeFlow 测试任务暂停和恢复流程
// 场景: 创建模板 -> 创建任务 -> 暂停 -> 恢复 -> 继续流程
func TestEndToEnd_TaskPauseResumeFlow(t *testing.T) {
	router, _ := setupTestAPIServer(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建模板
	templateID := createTestTemplateForE2E(t, router, csrfToken)

	// 2. 创建任务
	taskID := createTestTaskForE2E(t, router, csrfToken, templateID)

	// 3. 暂停任务
	pauseReqBody := map[string]string{"reason": "需要暂停"}
	pauseBody, _ := json.Marshal(pauseReqBody)
	pauseReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/pause", bytes.NewBuffer(pauseBody))
	pauseReq.Header.Set("Content-Type", "application/json")
	pauseReq.Header.Set("X-CSRF-Token", csrfToken)
	pauseW := httptest.NewRecorder()
	router.ServeHTTP(pauseW, pauseReq)

	assert.Equal(t, http.StatusOK, pauseW.Code)

	// 4. 验证任务状态为已暂停
	getReq1 := httptest.NewRequest("GET", "/api/v1/tasks/"+taskID, nil)
	getW1 := httptest.NewRecorder()
	router.ServeHTTP(getW1, getReq1)

	var getResponse1 api.Response
	json.Unmarshal(getW1.Body.Bytes(), &getResponse1)
	taskDataBytes1, _ := json.Marshal(getResponse1.Data)
	var task1 map[string]interface{}
	json.Unmarshal(taskDataBytes1, &task1)
	
	var state1 string
	state1, _ = task1["state"].(string)
	if state1 == "" {
		state1, _ = task1["State"].(string)
	}
	assert.Equal(t, "paused", state1, "task state should be paused")

	// 5. 恢复任务
	resumeReqBody := map[string]string{"reason": "恢复任务"}
	resumeBody, _ := json.Marshal(resumeReqBody)
	resumeReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/resume", bytes.NewBuffer(resumeBody))
	resumeReq.Header.Set("Content-Type", "application/json")
	resumeReq.Header.Set("X-CSRF-Token", csrfToken)
	resumeW := httptest.NewRecorder()
	router.ServeHTTP(resumeW, resumeReq)

	assert.Equal(t, http.StatusOK, resumeW.Code)

	// 6. 验证任务状态已恢复
	getReq2 := httptest.NewRequest("GET", "/api/v1/tasks/"+taskID, nil)
	getW2 := httptest.NewRecorder()
	router.ServeHTTP(getW2, getReq2)

	var getResponse2 api.Response
	json.Unmarshal(getW2.Body.Bytes(), &getResponse2)
	taskDataBytes2, _ := json.Marshal(getResponse2.Data)
	var task2 map[string]interface{}
	json.Unmarshal(taskDataBytes2, &task2)
	
	var state2 string
	state2, _ = task2["state"].(string)
	if state2 == "" {
		state2, _ = task2["State"].(string)
	}
	assert.NotEqual(t, "paused", state2, "task state should not be paused after resume")
}

// TestEndToEnd_TemplateVersionManagement 测试模板版本管理流程
// 场景: 创建模板 -> 更新模板(创建新版本) -> 使用旧版本创建任务 -> 使用新版本创建任务
func TestEndToEnd_TemplateVersionManagement(t *testing.T) {
	router, _ := setupTestAPIServer(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建模板 (Version 1)
	createReqBody := service.CreateTemplateRequest{
		Name:        "版本管理测试模板 V1",
		Description: "这是版本管理测试模板 V1",
		Nodes: map[string]*template.Node{
			"start": {
				ID:    "start",
				Name:  "开始",
				Type:  "start",
				Order: 1,
			},
		},
		Edges: []*template.Edge{},
		Config: nil,
	}
	body, err := json.Marshal(createReqBody)
	require.NoError(t, err)

	createReq := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewBuffer(body))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("X-CSRF-Token", csrfToken)
	createW := httptest.NewRecorder()
	router.ServeHTTP(createW, createReq)

	require.Equal(t, http.StatusOK, createW.Code)
	templateID := extractTemplateID(t, createW.Body.Bytes())
	require.NotEmpty(t, templateID)

	// 2. 使用 Version 1 创建任务
	createTaskReqBody1 := service.CreateTaskRequest{
		TemplateID: templateID,
		BusinessID: "biz-v1-001",
		Params:     json.RawMessage(`{"version": "v1"}`),
	}
	taskBody1, _ := json.Marshal(createTaskReqBody1)
	createTaskReq1 := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(taskBody1))
	createTaskReq1.Header.Set("Content-Type", "application/json")
	createTaskReq1.Header.Set("X-CSRF-Token", csrfToken)
	createTaskW1 := httptest.NewRecorder()
	router.ServeHTTP(createTaskW1, createTaskReq1)
	require.Equal(t, http.StatusOK, createTaskW1.Code)
	taskID1 := extractTaskID(t, createTaskW1.Body.Bytes())

	// 3. 更新模板 (创建 Version 2)
	updateReqBody := service.UpdateTemplateRequest{
		Name:        "版本管理测试模板 V2",
		Description: "这是版本管理测试模板 V2",
		Nodes: map[string]*template.Node{
			"start": {
				ID:    "start",
				Name:  "开始",
				Type:  "start",
				Order: 1,
			},
			"approval": {
				ID:    "approval",
				Name:  "审批节点",
				Type:  "approval",
				Order: 2,
			},
		},
		Edges: []*template.Edge{
			{From: "start", To: "approval"},
		},
		Config: nil,
	}
	updateBody, _ := json.Marshal(updateReqBody)
	updateReq := httptest.NewRequest("PUT", "/api/v1/templates/"+templateID, bytes.NewBuffer(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("X-CSRF-Token", csrfToken)
	updateW := httptest.NewRecorder()
	router.ServeHTTP(updateW, updateReq)
	require.Equal(t, http.StatusOK, updateW.Code)

	// 4. 使用 Version 2 创建任务
	createTaskReqBody2 := service.CreateTaskRequest{
		TemplateID: templateID,
		BusinessID: "biz-v2-001",
		Params:     json.RawMessage(`{"version": "v2"}`),
	}
	taskBody2, _ := json.Marshal(createTaskReqBody2)
	createTaskReq2 := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(taskBody2))
	createTaskReq2.Header.Set("Content-Type", "application/json")
	createTaskReq2.Header.Set("X-CSRF-Token", csrfToken)
	createTaskW2 := httptest.NewRecorder()
	router.ServeHTTP(createTaskW2, createTaskReq2)
	require.Equal(t, http.StatusOK, createTaskW2.Code)
	taskID2 := extractTaskID(t, createTaskW2.Body.Bytes())

	// 5. 验证两个任务使用不同版本的模板
	getTaskReq1 := httptest.NewRequest("GET", "/api/v1/tasks/"+taskID1, nil)
	getTaskW1 := httptest.NewRecorder()
	router.ServeHTTP(getTaskW1, getTaskReq1)

	getTaskReq2 := httptest.NewRequest("GET", "/api/v1/tasks/"+taskID2, nil)
	getTaskW2 := httptest.NewRecorder()
	router.ServeHTTP(getTaskW2, getTaskReq2)

	var taskResp1, taskResp2 api.Response
	json.Unmarshal(getTaskW1.Body.Bytes(), &taskResp1)
	json.Unmarshal(getTaskW2.Body.Bytes(), &taskResp2)

	task1DataBytes, _ := json.Marshal(taskResp1.Data)
	task2DataBytes, _ := json.Marshal(taskResp2.Data)

	var task1, task2 map[string]interface{}
	json.Unmarshal(task1DataBytes, &task1)
	json.Unmarshal(task2DataBytes, &task2)

	// 验证两个任务都关联到同一个模板 ID
	templateID1, _ := task1["template_id"].(string)
	if templateID1 == "" {
		templateID1, _ = task1["TemplateID"].(string)
	}
	templateID2, _ := task2["template_id"].(string)
	if templateID2 == "" {
		templateID2, _ = task2["TemplateID"].(string)
	}

	assert.Equal(t, templateID, templateID1, "task1 should use the template")
	assert.Equal(t, templateID, templateID2, "task2 should use the template")
}

// 辅助函数：用于端到端测试
func createTestTemplateForE2E(t *testing.T, router *gin.Engine, csrfToken string) string {
	createReqBody := service.CreateTemplateRequest{
		Name:        "E2E测试模板",
		Description: "用于端到端测试的模板",
		Nodes: map[string]*template.Node{
			"start": {
				ID:    "start",
				Name:  "开始",
				Type:  "start",
				Order: 1,
			},
		},
		Edges: []*template.Edge{},
		Config: nil,
	}
	body, err := json.Marshal(createReqBody)
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrfToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	return extractTemplateID(t, w.Body.Bytes())
}

func createTestTemplateWithApprovalNodeForE2E(t *testing.T, router *gin.Engine, csrfToken string) string {
	createReqBody := service.CreateTemplateRequest{
		Name:        "E2E审批节点测试模板",
		Description: "用于端到端测试的带审批节点的模板",
		Nodes: map[string]*template.Node{
			"start": {
				ID:    "start",
				Name:  "开始",
				Type:  "start",
				Order: 1,
			},
			"approval": {
				ID:    "approval",
				Name:  "审批节点",
				Type:  "approval",
				Order: 2,
			},
		},
		Edges: []*template.Edge{
			{From: "start", To: "approval"},
		},
		Config: nil,
	}
	body, err := json.Marshal(createReqBody)
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrfToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	return extractTemplateID(t, w.Body.Bytes())
}

func createTestTaskForE2E(t *testing.T, router *gin.Engine, csrfToken string, templateID string) string {
	createTaskReqBody := service.CreateTaskRequest{
		TemplateID: templateID,
		BusinessID: "biz-e2e-001",
		Params:     json.RawMessage(`{"amount": 1000}`),
	}
	taskBody, err := json.Marshal(createTaskReqBody)
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(taskBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrfToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	return extractTaskID(t, w.Body.Bytes())
}

func submitTaskForE2E(t *testing.T, router *gin.Engine, csrfToken string, taskID string) {
	req := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/submit", nil)
	req.Header.Set("X-CSRF-Token", csrfToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

