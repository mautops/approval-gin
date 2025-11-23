package api_test

import (
	"bytes"
	"encoding/json"
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

// TestAPIIntegration_TaskSubmit 测试任务提交 API
func TestAPIIntegration_TaskSubmit(t *testing.T) {
	router, _ := setupTestAPIServer(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建模板
	templateID := createTestTemplate(t, router, csrfToken)

	// 2. 创建任务
	taskID := createTestTask(t, router, csrfToken, templateID)

	// 3. 提交任务
	submitReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/submit", nil)
	submitReq.Header.Set("X-CSRF-Token", csrfToken)
	submitW := httptest.NewRecorder()
	router.ServeHTTP(submitW, submitReq)

	assert.Equal(t, http.StatusOK, submitW.Code)
	var submitResponse api.Response
	err := json.Unmarshal(submitW.Body.Bytes(), &submitResponse)
	require.NoError(t, err)
	assert.Equal(t, 0, submitResponse.Code)

	// 4. 验证任务状态已更新
	getTaskReq := httptest.NewRequest("GET", "/api/v1/tasks/"+taskID, nil)
	getTaskW := httptest.NewRecorder()
	router.ServeHTTP(getTaskW, getTaskReq)

	assert.Equal(t, http.StatusOK, getTaskW.Code)
	var getTaskResponse api.Response
	err = json.Unmarshal(getTaskW.Body.Bytes(), &getTaskResponse)
	require.NoError(t, err)
	assert.Equal(t, 0, getTaskResponse.Code)
}

// TestAPIIntegration_TaskApprove 测试任务审批 API
func TestAPIIntegration_TaskApprove(t *testing.T) {
	router, _ := setupTestAPIServer(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建模板(包含审批节点)
	templateID := createTestTemplateWithApprovalNode(t, router, csrfToken)

	// 2. 创建并提交任务
	taskID := createTestTask(t, router, csrfToken, templateID)
	submitTestTask(t, router, csrfToken, taskID)

	// 3. 添加审批人
	addApproverReqBody := service.AddApproverRequest{
		NodeID:   "approval",
		Approver: "user-001",
		Reason:   "设置审批人",
	}
	approverBody, err := json.Marshal(addApproverReqBody)
	require.NoError(t, err)

	addApproverReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/approvers", bytes.NewBuffer(approverBody))
	addApproverReq.Header.Set("Content-Type", "application/json")
	addApproverReq.Header.Set("X-CSRF-Token", csrfToken)
	addApproverW := httptest.NewRecorder()
	router.ServeHTTP(addApproverW, addApproverReq)

	require.Equal(t, http.StatusOK, addApproverW.Code)

	// 4. 审批同意
	approveReqBody := service.ApproveRequest{
		NodeID:  "approval",
		Comment: "同意",
	}
	approveBody, err := json.Marshal(approveReqBody)
	require.NoError(t, err)

	approveReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/approve", bytes.NewBuffer(approveBody))
	approveReq.Header.Set("Content-Type", "application/json")
	approveReq.Header.Set("X-CSRF-Token", csrfToken)
	approveW := httptest.NewRecorder()
	router.ServeHTTP(approveW, approveReq)

	assert.Equal(t, http.StatusOK, approveW.Code)
	var approveResponse api.Response
	err = json.Unmarshal(approveW.Body.Bytes(), &approveResponse)
	require.NoError(t, err)
	assert.Equal(t, 0, approveResponse.Code)
}

// TestAPIIntegration_TaskReject 测试任务拒绝 API
func TestAPIIntegration_TaskReject(t *testing.T) {
	router, _ := setupTestAPIServer(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建模板(包含审批节点)
	templateID := createTestTemplateWithApprovalNode(t, router, csrfToken)

	// 2. 创建并提交任务
	taskID := createTestTask(t, router, csrfToken, templateID)
	submitTestTask(t, router, csrfToken, taskID)

	// 3. 添加审批人
	addApproverReqBody := service.AddApproverRequest{
		NodeID:   "approval",
		Approver: "user-001",
		Reason:   "设置审批人",
	}
	approverBody, err := json.Marshal(addApproverReqBody)
	require.NoError(t, err)

	addApproverReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/approvers", bytes.NewBuffer(approverBody))
	addApproverReq.Header.Set("Content-Type", "application/json")
	addApproverReq.Header.Set("X-CSRF-Token", csrfToken)
	addApproverW := httptest.NewRecorder()
	router.ServeHTTP(addApproverW, addApproverReq)

	require.Equal(t, http.StatusOK, addApproverW.Code)

	// 4. 审批拒绝
	rejectReqBody := service.RejectRequest{
		NodeID:  "approval",
		Comment: "拒绝",
	}
	rejectBody, err := json.Marshal(rejectReqBody)
	require.NoError(t, err)

	rejectReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/reject", bytes.NewBuffer(rejectBody))
	rejectReq.Header.Set("Content-Type", "application/json")
	rejectReq.Header.Set("X-CSRF-Token", csrfToken)
	rejectW := httptest.NewRecorder()
	router.ServeHTTP(rejectW, rejectReq)

	assert.Equal(t, http.StatusOK, rejectW.Code)
	var rejectResponse api.Response
	err = json.Unmarshal(rejectW.Body.Bytes(), &rejectResponse)
	require.NoError(t, err)
	assert.Equal(t, 0, rejectResponse.Code)
}

// TestAPIIntegration_TaskCancel 测试任务取消 API
func TestAPIIntegration_TaskCancel(t *testing.T) {
	router, _ := setupTestAPIServer(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建模板
	templateID := createTestTemplate(t, router, csrfToken)

	// 2. 创建任务
	taskID := createTestTask(t, router, csrfToken, templateID)

	// 3. 取消任务
	cancelReqBody := map[string]string{
		"reason": "取消原因",
	}
	cancelBody, err := json.Marshal(cancelReqBody)
	require.NoError(t, err)

	cancelReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/cancel", bytes.NewBuffer(cancelBody))
	cancelReq.Header.Set("Content-Type", "application/json")
	cancelReq.Header.Set("X-CSRF-Token", csrfToken)
	cancelW := httptest.NewRecorder()
	router.ServeHTTP(cancelW, cancelReq)

	assert.Equal(t, http.StatusOK, cancelW.Code)
	var cancelResponse api.Response
	err = json.Unmarshal(cancelW.Body.Bytes(), &cancelResponse)
	require.NoError(t, err)
	assert.Equal(t, 0, cancelResponse.Code)
}

// TestAPIIntegration_TaskWithdraw 测试任务撤回 API
func TestAPIIntegration_TaskWithdraw(t *testing.T) {
	router, _ := setupTestAPIServer(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建模板
	templateID := createTestTemplate(t, router, csrfToken)

	// 2. 创建并提交任务
	taskID := createTestTask(t, router, csrfToken, templateID)
	submitTestTask(t, router, csrfToken, taskID)

	// 3. 撤回任务
	withdrawReqBody := map[string]string{
		"reason": "撤回原因",
	}
	withdrawBody, err := json.Marshal(withdrawReqBody)
	require.NoError(t, err)

	withdrawReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/withdraw", bytes.NewBuffer(withdrawBody))
	withdrawReq.Header.Set("Content-Type", "application/json")
	withdrawReq.Header.Set("X-CSRF-Token", csrfToken)
	withdrawW := httptest.NewRecorder()
	router.ServeHTTP(withdrawW, withdrawReq)

	assert.Equal(t, http.StatusOK, withdrawW.Code)
	var withdrawResponse api.Response
	err = json.Unmarshal(withdrawW.Body.Bytes(), &withdrawResponse)
	require.NoError(t, err)
	assert.Equal(t, 0, withdrawResponse.Code)
}

// TestAPIIntegration_TaskPause 测试任务暂停 API
func TestAPIIntegration_TaskPause(t *testing.T) {
	router, _ := setupTestAPIServer(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建模板
	templateID := createTestTemplate(t, router, csrfToken)

	// 2. 创建并提交任务
	taskID := createTestTask(t, router, csrfToken, templateID)
	submitTestTask(t, router, csrfToken, taskID)

	// 3. 暂停任务
	pauseReqBody := map[string]string{
		"reason": "暂停原因",
	}
	pauseBody, err := json.Marshal(pauseReqBody)
	require.NoError(t, err)

	pauseReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/pause", bytes.NewBuffer(pauseBody))
	pauseReq.Header.Set("Content-Type", "application/json")
	pauseReq.Header.Set("X-CSRF-Token", csrfToken)
	pauseW := httptest.NewRecorder()
	router.ServeHTTP(pauseW, pauseReq)

	assert.Equal(t, http.StatusOK, pauseW.Code)
	var pauseResponse api.Response
	err = json.Unmarshal(pauseW.Body.Bytes(), &pauseResponse)
	require.NoError(t, err)
	assert.Equal(t, 0, pauseResponse.Code)
}

// TestAPIIntegration_TaskResume 测试任务恢复 API
func TestAPIIntegration_TaskResume(t *testing.T) {
	router, _ := setupTestAPIServer(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建模板
	templateID := createTestTemplate(t, router, csrfToken)

	// 2. 创建并提交任务
	taskID := createTestTask(t, router, csrfToken, templateID)
	submitTestTask(t, router, csrfToken, taskID)

	// 3. 暂停任务
	pauseReqBody := map[string]string{
		"reason": "暂停原因",
	}
	pauseBody, err := json.Marshal(pauseReqBody)
	require.NoError(t, err)

	pauseReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/pause", bytes.NewBuffer(pauseBody))
	pauseReq.Header.Set("Content-Type", "application/json")
	pauseReq.Header.Set("X-CSRF-Token", csrfToken)
	pauseW := httptest.NewRecorder()
	router.ServeHTTP(pauseW, pauseReq)

	require.Equal(t, http.StatusOK, pauseW.Code)

	// 4. 恢复任务
	resumeReqBody := map[string]string{
		"reason": "恢复原因",
	}
	resumeBody, err := json.Marshal(resumeReqBody)
	require.NoError(t, err)

	resumeReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/resume", bytes.NewBuffer(resumeBody))
	resumeReq.Header.Set("Content-Type", "application/json")
	resumeReq.Header.Set("X-CSRF-Token", csrfToken)
	resumeW := httptest.NewRecorder()
	router.ServeHTTP(resumeW, resumeReq)

	assert.Equal(t, http.StatusOK, resumeW.Code)
	var resumeResponse api.Response
	err = json.Unmarshal(resumeW.Body.Bytes(), &resumeResponse)
	require.NoError(t, err)
	assert.Equal(t, 0, resumeResponse.Code)
}

// createTestTemplate 创建测试模板的辅助函数
func createTestTemplate(t *testing.T, router *gin.Engine, csrfToken string) string {
	createReqBody := service.CreateTemplateRequest{
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Nodes: map[string]*template.Node{
			"start": {
				ID:    "start",
				Name:  "开始",
				Type:  template.NodeTypeStart,
				Order: 1,
				Config: nil,
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

// createTestTemplateWithApprovalNode 创建包含审批节点的测试模板
func createTestTemplateWithApprovalNode(t *testing.T, router *gin.Engine, csrfToken string) string {
	createReqBody := service.CreateTemplateRequest{
		Name:        "审批节点测试模板",
		Description: "这是一个包含审批节点的测试模板",
		Nodes: map[string]*template.Node{
			"start": {
				ID:    "start",
				Name:  "开始",
				Type:  template.NodeTypeStart,
				Order: 1,
				Config: nil,
			},
			"approval": {
				ID:    "approval",
				Name:  "审批节点",
				Type:  template.NodeTypeApproval,
				Order: 2,
				Config: nil,
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

// createTestTask 创建测试任务的辅助函数
func createTestTask(t *testing.T, router *gin.Engine, csrfToken string, templateID string) string {
	createReqBody := service.CreateTaskRequest{
		TemplateID: templateID,
		BusinessID: "biz-001",
		Params:     json.RawMessage(`{}`),
	}

	body, err := json.Marshal(createReqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrfToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	return extractTaskID(t, w.Body.Bytes())
}

// submitTestTask 提交测试任务的辅助函数
func submitTestTask(t *testing.T, router *gin.Engine, csrfToken string, taskID string) {
	req := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/submit", nil)
	req.Header.Set("X-CSRF-Token", csrfToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

// TestAPIIntegration_TaskTransfer 测试任务转交 API
func TestAPIIntegration_TaskTransfer(t *testing.T) {
	router, _ := setupTestAPIServer(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建模板(包含审批节点)
	templateID := createTestTemplateWithApprovalNode(t, router, csrfToken)

	// 2. 创建并提交任务
	taskID := createTestTask(t, router, csrfToken, templateID)
	submitTestTask(t, router, csrfToken, taskID)

	// 3. 添加审批人
	addApproverReqBody := service.AddApproverRequest{
		NodeID:   "approval",
		Approver: "user-001",
		Reason:   "设置审批人",
	}
	approverBody, err := json.Marshal(addApproverReqBody)
	require.NoError(t, err)

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
	transferBody, err := json.Marshal(transferReqBody)
	require.NoError(t, err)

	transferReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/transfer", bytes.NewBuffer(transferBody))
	transferReq.Header.Set("Content-Type", "application/json")
	transferReq.Header.Set("X-CSRF-Token", csrfToken)
	transferW := httptest.NewRecorder()
	router.ServeHTTP(transferW, transferReq)

	assert.Equal(t, http.StatusOK, transferW.Code)
	var transferResponse api.Response
	err = json.Unmarshal(transferW.Body.Bytes(), &transferResponse)
	require.NoError(t, err)
	assert.Equal(t, 0, transferResponse.Code)
}

// TestAPIIntegration_TaskAddApprover 测试任务加签 API
func TestAPIIntegration_TaskAddApprover(t *testing.T) {
	router, _ := setupTestAPIServer(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建模板(包含审批节点)
	templateID := createTestTemplateWithApprovalNode(t, router, csrfToken)

	// 2. 创建并提交任务
	taskID := createTestTask(t, router, csrfToken, templateID)
	submitTestTask(t, router, csrfToken, taskID)

	// 3. 加签
	addApproverReqBody := service.AddApproverRequest{
		NodeID:   "approval",
		Approver: "user-001",
		Reason:   "加签原因",
	}
	approverBody, err := json.Marshal(addApproverReqBody)
	require.NoError(t, err)

	addApproverReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/approvers", bytes.NewBuffer(approverBody))
	addApproverReq.Header.Set("Content-Type", "application/json")
	addApproverReq.Header.Set("X-CSRF-Token", csrfToken)
	addApproverW := httptest.NewRecorder()
	router.ServeHTTP(addApproverW, addApproverReq)

	assert.Equal(t, http.StatusOK, addApproverW.Code)
	var addApproverResponse api.Response
	err = json.Unmarshal(addApproverW.Body.Bytes(), &addApproverResponse)
	require.NoError(t, err)
	assert.Equal(t, 0, addApproverResponse.Code)
}

// TestAPIIntegration_TaskRemoveApprover 测试任务减签 API
func TestAPIIntegration_TaskRemoveApprover(t *testing.T) {
	router, _ := setupTestAPIServer(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建模板(包含审批节点)
	templateID := createTestTemplateWithApprovalNode(t, router, csrfToken)

	// 2. 创建并提交任务
	taskID := createTestTask(t, router, csrfToken, templateID)
	submitTestTask(t, router, csrfToken, taskID)

	// 3. 先添加审批人
	addApproverReqBody := service.AddApproverRequest{
		NodeID:   "approval",
		Approver: "user-001",
		Reason:   "设置审批人",
	}
	approverBody, err := json.Marshal(addApproverReqBody)
	require.NoError(t, err)

	addApproverReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/approvers", bytes.NewBuffer(approverBody))
	addApproverReq.Header.Set("Content-Type", "application/json")
	addApproverReq.Header.Set("X-CSRF-Token", csrfToken)
	addApproverW := httptest.NewRecorder()
	router.ServeHTTP(addApproverW, addApproverReq)

	require.Equal(t, http.StatusOK, addApproverW.Code)

	// 4. 减签
	removeApproverReqBody := service.RemoveApproverRequest{
		NodeID:   "approval",
		Approver: "user-001",
		Reason:   "减签原因",
	}
	removeApproverBody, err := json.Marshal(removeApproverReqBody)
	require.NoError(t, err)

	removeApproverReq := httptest.NewRequest("DELETE", "/api/v1/tasks/"+taskID+"/approvers", bytes.NewBuffer(removeApproverBody))
	removeApproverReq.Header.Set("Content-Type", "application/json")
	removeApproverReq.Header.Set("X-CSRF-Token", csrfToken)
	removeApproverW := httptest.NewRecorder()
	router.ServeHTTP(removeApproverW, removeApproverReq)

	assert.Equal(t, http.StatusOK, removeApproverW.Code)
	var removeApproverResponse api.Response
	err = json.Unmarshal(removeApproverW.Body.Bytes(), &removeApproverResponse)
	require.NoError(t, err)
	assert.Equal(t, 0, removeApproverResponse.Code)
}

// TestAPIIntegration_TaskReplaceApprover 测试任务替换审批人 API
func TestAPIIntegration_TaskReplaceApprover(t *testing.T) {
	router, _ := setupTestAPIServer(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建模板(包含审批节点)
	templateID := createTestTemplateWithApprovalNode(t, router, csrfToken)

	// 2. 创建并提交任务
	taskID := createTestTask(t, router, csrfToken, templateID)
	submitTestTask(t, router, csrfToken, taskID)

	// 3. 先添加审批人
	addApproverReqBody := service.AddApproverRequest{
		NodeID:   "approval",
		Approver: "user-001",
		Reason:   "设置审批人",
	}
	approverBody, err := json.Marshal(addApproverReqBody)
	require.NoError(t, err)

	addApproverReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/approvers", bytes.NewBuffer(approverBody))
	addApproverReq.Header.Set("Content-Type", "application/json")
	addApproverReq.Header.Set("X-CSRF-Token", csrfToken)
	addApproverW := httptest.NewRecorder()
	router.ServeHTTP(addApproverW, addApproverReq)

	require.Equal(t, http.StatusOK, addApproverW.Code)

	// 4. 替换审批人
	replaceApproverReqBody := service.ReplaceApproverRequest{
		NodeID:      "approval",
		OldApprover: "user-001",
		NewApprover: "user-002",
		Reason:      "替换原因",
	}
	replaceApproverBody, err := json.Marshal(replaceApproverReqBody)
	require.NoError(t, err)

	replaceApproverReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/approvers/replace", bytes.NewBuffer(replaceApproverBody))
	replaceApproverReq.Header.Set("Content-Type", "application/json")
	replaceApproverReq.Header.Set("X-CSRF-Token", csrfToken)
	replaceApproverW := httptest.NewRecorder()
	router.ServeHTTP(replaceApproverW, replaceApproverReq)

	assert.Equal(t, http.StatusOK, replaceApproverW.Code)
	var replaceApproverResponse api.Response
	err = json.Unmarshal(replaceApproverW.Body.Bytes(), &replaceApproverResponse)
	require.NoError(t, err)
	assert.Equal(t, 0, replaceApproverResponse.Code)
}

