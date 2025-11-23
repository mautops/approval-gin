package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/mautops/approval-gin/internal/service"
	"github.com/mautops/approval-kit/pkg/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchApprove(t *testing.T) {
	// 创建测试用的 mock service
	mockTaskService := &mockTaskService{}
	taskController := api.NewTaskController(mockTaskService)

	router := gin.New()
	router.POST("/api/v1/tasks/batch/approve", taskController.BatchApprove)

	// 测试请求
	reqBody := map[string]interface{}{
		"task_ids": []string{"task-1", "task-2", "task-3"},
		"node_id":  "node-1",
		"comment":  "批量审批",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/batch/approve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response api.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Code)

	// 验证批量操作结果
	results, ok := response.Data.([]interface{})
	require.True(t, ok)
	assert.Len(t, results, 3)
}

func TestBatchTransfer(t *testing.T) {
	// 创建测试用的 mock service
	mockTaskService := &mockTaskService{}
	taskController := api.NewTaskController(mockTaskService)

	router := gin.New()
	router.POST("/api/v1/tasks/batch/transfer", taskController.BatchTransfer)

	// 测试请求
	reqBody := map[string]interface{}{
		"task_ids":      []string{"task-1", "task-2"},
		"node_id":       "node-1",
		"old_approver":  "user-1",
		"new_approver":  "user-2",
		"comment":       "批量转交",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/batch/transfer", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response api.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Code)
}

// mockTaskService 用于测试的 mock service
type mockTaskService struct{}

func (m *mockTaskService) Create(ctx context.Context, req *service.CreateTaskRequest) (*task.Task, error) {
	return nil, nil
}

func (m *mockTaskService) Get(id string) (*task.Task, error) {
	return nil, nil
}

func (m *mockTaskService) Submit(ctx context.Context, id string) error {
	return nil
}

func (m *mockTaskService) Approve(ctx context.Context, id string, req *service.ApproveRequest) error {
	return nil
}

func (m *mockTaskService) Reject(ctx context.Context, id string, req *service.RejectRequest) error {
	return nil
}

func (m *mockTaskService) Cancel(ctx context.Context, id string, reason string) error {
	return nil
}

func (m *mockTaskService) Withdraw(ctx context.Context, id string, reason string) error {
	return nil
}

func (m *mockTaskService) Transfer(ctx context.Context, id string, req *service.TransferRequest) error {
	return nil
}

func (m *mockTaskService) AddApprover(ctx context.Context, id string, req *service.AddApproverRequest) error {
	return nil
}

func (m *mockTaskService) RemoveApprover(ctx context.Context, id string, req *service.RemoveApproverRequest) error {
	return nil
}

func (m *mockTaskService) Pause(ctx context.Context, id string, reason string) error {
	return nil
}

func (m *mockTaskService) Resume(ctx context.Context, id string, reason string) error {
	return nil
}

func (m *mockTaskService) RollbackToNode(ctx context.Context, id string, req *service.RollbackRequest) error {
	return nil
}

func (m *mockTaskService) ReplaceApprover(ctx context.Context, id string, req *service.ReplaceApproverRequest) error {
	return nil
}

func (m *mockTaskService) BatchApprove(ctx context.Context, req *service.BatchApproveRequest) ([]service.BatchOperationResult, error) {
	results := make([]service.BatchOperationResult, len(req.TaskIDs))
	for i, taskID := range req.TaskIDs {
		results[i] = service.BatchOperationResult{
			TaskID:  taskID,
			Success: true,
		}
	}
	return results, nil
}

func (m *mockTaskService) BatchTransfer(ctx context.Context, req *service.BatchTransferRequest) ([]service.BatchOperationResult, error) {
	results := make([]service.BatchOperationResult, len(req.TaskIDs))
	for i, taskID := range req.TaskIDs {
		results[i] = service.BatchOperationResult{
			TaskID:  taskID,
			Success: true,
		}
	}
	return results, nil
}

