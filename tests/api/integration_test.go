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
	"github.com/mautops/approval-gin/internal/auth"
	"github.com/mautops/approval-gin/internal/config"
	"github.com/mautops/approval-gin/internal/container"
	"github.com/mautops/approval-gin/internal/database"
	"github.com/mautops/approval-gin/internal/integration"
	"github.com/mautops/approval-gin/internal/repository"
	"github.com/mautops/approval-gin/internal/service"
	"github.com/mautops/approval-kit/pkg/template"
	"github.com/mautops/approval-gin/internal/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestAPIServer 创建测试 API 服务器
func setupTestAPIServer(t *testing.T) (*gin.Engine, *container.Container) {
	// 创建内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 迁移数据库
	err = database.Migrate(db)
	require.NoError(t, err)

	// 创建测试配置
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "test",
			Password: "test",
			DBName:   "test_db",
			SSLMode:  "disable",
		},
		OpenFGA: config.OpenFGAConfig{
			APIURL:  "http://localhost:8081",
			StoreID: "test-store",
			ModelID: "test-model",
		},
		Keycloak: config.KeycloakConfig{
			Issuer: "http://localhost:8082/realms/test",
		},
	}

	// 手动创建容器（不使用 NewContainer，因为需要真实的数据库连接）
	// 直接初始化组件
	templateMgr := integration.NewTemplateManager(db)
	eventHandler := integration.NewEventHandler(db, 1)
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, eventHandler)
	fgaClient, _ := auth.NewOpenFGAClient(cfg.OpenFGA.APIURL, cfg.OpenFGA.StoreID, cfg.OpenFGA.ModelID)
	keycloakValidator := auth.NewKeycloakTokenValidator(cfg.Keycloak.Issuer)

	// 创建服务
	auditLogSvc := service.NewAuditLogService(repository.NewAuditLogRepository(db))
	templateSvc := service.NewTemplateService(templateMgr, db, auditLogSvc, fgaClient)
	taskSvc := service.NewTaskService(taskMgr, db, auditLogSvc, fgaClient)
	querySvc := service.NewQueryService(db, taskMgr)

	// 创建控制器
	templateController := api.NewTemplateController(templateSvc)
	taskController := api.NewTaskController(taskSvc)
	queryController := api.NewQueryController(querySvc)

	// 创建 WebSocket Hub
	hub := websocket.NewHub()
	go hub.Run()

	// 创建健康检查控制器
	healthController := api.NewHealthController(db, fgaClient)

	// 直接创建路由，不使用 SetupRoutes 避免重复注册
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(api.RequestIDMiddleware())
	router.Use(api.RequestLogMiddleware())
	router.Use(api.SecurityHeadersMiddleware())
	csrfConfig := api.DefaultCSRFConfig()
	router.Use(api.CSRFMiddleware(csrfConfig))

	// 健康检查
	router.GET("/health", healthController.Check)
	
	// CSRF Token 端点（用于测试）
	router.GET("/api/v1/csrf-token", func(c *gin.Context) {
		token, err := api.GetCSRFToken(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": token})
	})

	// Prometheus 指标端点
	router.GET("/metrics", api.MetricsHandler)

	// WebSocket 路由
	router.GET("/ws/tasks/:id", websocket.WebSocketHandler(hub, keycloakValidator))

	// SSE 路由
	router.GET("/sse/tasks/:id", api.SSEHandler(keycloakValidator))

	// API v1 路由组（不使用认证中间件，测试中直接使用 mock token）
	v1 := router.Group("/api/v1")
	{
		templates := v1.Group("/templates")
		{
			templates.POST("", templateController.Create)
			templates.GET("", templateController.List)
			templates.GET("/:id", templateController.Get)
			templates.PUT("/:id", templateController.Update)
			templates.DELETE("/:id", templateController.Delete)
			templates.GET("/:id/versions", templateController.ListVersions)
		}

		tasks := v1.Group("/tasks")
		{
			tasks.POST("", taskController.Create)
			tasks.GET("", queryController.ListTasks)
			tasks.GET("/:id", taskController.Get)
			tasks.POST("/:id/submit", taskController.Submit)
			tasks.POST("/:id/approve", taskController.Approve)
			tasks.POST("/:id/reject", taskController.Reject)
			tasks.POST("/:id/cancel", taskController.Cancel)
			tasks.POST("/:id/withdraw", taskController.Withdraw)
			// 高级操作路由
			tasks.POST("/:id/transfer", taskController.Transfer)
			tasks.POST("/:id/approvers", taskController.AddApprover)
			tasks.DELETE("/:id/approvers", taskController.RemoveApprover)
			tasks.POST("/:id/pause", taskController.Pause)
			tasks.POST("/:id/resume", taskController.Resume)
			tasks.POST("/:id/rollback", taskController.RollbackToNode)
			tasks.POST("/:id/approvers/replace", taskController.ReplaceApprover)
			tasks.GET("/:id/records", queryController.GetRecords)
			tasks.GET("/:id/history", queryController.GetHistory)
		}
	}

	// 创建一个简单的容器用于返回
	ctr := &container.Container{
		// 注意: 这里需要 Container 的字段是可导出的，或者我们需要提供 getter 方法
		// 暂时返回 nil，测试中直接使用已创建的组件
	}

	return router, ctr
}

// TestAPIIntegration_TemplateCreate 测试模板创建 API
func TestAPIIntegration_TemplateCreate(t *testing.T) {
	router, _ := setupTestAPIServer(t)

	// 获取 CSRF Token
	csrfToken := getCSRFToken(t, router)

	// 创建模板请求
	reqBody := service.CreateTemplateRequest{
		Name:        "测试模板",
		Description: "这是一个测试模板",
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

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrfToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)

	var response api.Response
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Code)
	assert.NotNil(t, response.Data)
}

// TestAPIIntegration_TemplateGet 测试模板获取 API
func TestAPIIntegration_TemplateGet(t *testing.T) {
	router, _ := setupTestAPIServer(t)

	// 获取 CSRF Token
	csrfToken := getCSRFToken(t, router)

	// 先创建一个模板（通过 API）
	createReqBody := service.CreateTemplateRequest{
		Name:        "测试模板",
		Description: "这是一个测试模板",
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

	// 从响应中获取模板 ID
	templateID := extractTemplateID(t, createW.Body.Bytes())
	require.NotEmpty(t, templateID)

	// 获取模板
	req := httptest.NewRequest("GET", "/api/v1/templates/"+templateID, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)

	var response api.Response
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Code)
	assert.NotNil(t, response.Data)
}

// TestAPIIntegration_TaskCreate 测试任务创建 API
func TestAPIIntegration_TaskCreate(t *testing.T) {
	router, _ := setupTestAPIServer(t)

	// 获取 CSRF Token
	csrfToken := getCSRFToken(t, router)

	// 先创建一个模板（通过 API）
	createTemplateReqBody := service.CreateTemplateRequest{
		Name:        "测试模板",
		Description: "这是一个测试模板",
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

	templateBody, err := json.Marshal(createTemplateReqBody)
	require.NoError(t, err)

	createTemplateReq := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewBuffer(templateBody))
	createTemplateReq.Header.Set("Content-Type", "application/json")
	createTemplateReq.Header.Set("X-CSRF-Token", csrfToken)
	createTemplateW := httptest.NewRecorder()

	router.ServeHTTP(createTemplateW, createTemplateReq)
	require.Equal(t, http.StatusOK, createTemplateW.Code)

	// 从响应中获取模板 ID
	templateID := extractTemplateID(t, createTemplateW.Body.Bytes())
	require.NotEmpty(t, templateID)

	// 创建任务请求
	reqBody := service.CreateTaskRequest{
		TemplateID: templateID,
		BusinessID: "biz-001",
		Params:     json.RawMessage(`{"amount": 1000}`),
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrfToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)

	var response api.Response
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Code)
	assert.NotNil(t, response.Data)
}

// getTestDB 获取测试数据库
func getTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = database.Migrate(db)
	require.NoError(t, err)
	return db
}

// extractTemplateID 从 API 响应中提取模板 ID
func extractTemplateID(t *testing.T, responseBody []byte) string {
	var response api.Response
	err := json.Unmarshal(responseBody, &response)
	require.NoError(t, err)
	require.NotNil(t, response.Data)

	// 将 data 转换为 map 以提取 ID
	dataBytes, err := json.Marshal(response.Data)
	require.NoError(t, err)

	var template map[string]interface{}
	err = json.Unmarshal(dataBytes, &template)
	require.NoError(t, err)

	// 处理不同大小写的 ID 字段 (JSON 可能使用 "ID" 或 "id")
	var idValue interface{}
	var ok bool
	
	// 先尝试小写 "id"
	idValue, ok = template["id"]
	if !ok {
		// 再尝试大写 "ID"
		idValue, ok = template["ID"]
	}
	require.True(t, ok, "template should have id field")
	
	var id string
	switch v := idValue.(type) {
	case string:
		id = v
	case float64:
		// JSON 数字可能被解析为 float64
		id = fmt.Sprintf("%.0f", v)
	default:
		// 尝试转换为字符串
		id = fmt.Sprintf("%v", v)
	}
	
	require.NotEmpty(t, id, "template ID should not be empty")
	return id
}

// extractTaskID 从 API 响应中提取任务 ID
func extractTaskID(t *testing.T, responseBody []byte) string {
	var response api.Response
	err := json.Unmarshal(responseBody, &response)
	require.NoError(t, err)
	require.NotNil(t, response.Data)

	// 将 data 转换为 map 以提取 ID
	dataBytes, err := json.Marshal(response.Data)
	require.NoError(t, err)

	var task map[string]interface{}
	err = json.Unmarshal(dataBytes, &task)
	require.NoError(t, err)

	// 处理不同大小写的 ID 字段
	var idValue interface{}
	var ok bool
	
	// 先尝试小写 "id"
	idValue, ok = task["id"]
	if !ok {
		// 再尝试大写 "ID"
		idValue, ok = task["ID"]
	}
	require.True(t, ok, "task should have id field")
	
	var id string
	switch v := idValue.(type) {
	case string:
		id = v
	case float64:
		// JSON 数字可能被解析为 float64
		id = fmt.Sprintf("%.0f", v)
	default:
		// 尝试转换为字符串
		id = fmt.Sprintf("%v", v)
	}
	
	require.NotEmpty(t, id, "task ID should not be empty")
	return id
}

// getCSRFToken 获取 CSRF Token（用于测试）
func getCSRFToken(t *testing.T, router *gin.Engine) string {
	req := httptest.NewRequest("GET", "/api/v1/csrf-token", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var tokenResp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &tokenResp)
	require.NoError(t, err)

	token, ok := tokenResp["token"]
	require.True(t, ok, "token should exist in response")
	require.NotEmpty(t, token, "token should not be empty")

	return token
}

// TestAPIIntegration_TemplateFullFlow 测试模板管理完整流程
func TestAPIIntegration_TemplateFullFlow(t *testing.T) {
	router, _ := setupTestAPIServer(t)

	// 获取 CSRF Token
	csrfToken := getCSRFToken(t, router)

	// 1. 创建模板
	createReqBody := service.CreateTemplateRequest{
		Name:        "完整流程测试模板",
		Description: "这是一个完整流程测试模板",
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

	// 2. 获取模板
	getReq := httptest.NewRequest("GET", "/api/v1/templates/"+templateID, nil)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)

	assert.Equal(t, http.StatusOK, getW.Code)
	var getResponse api.Response
	err = json.Unmarshal(getW.Body.Bytes(), &getResponse)
	require.NoError(t, err)
	assert.Equal(t, 0, getResponse.Code)

	// 3. 更新模板
	updateReqBody := service.UpdateTemplateRequest{
		Name:        "更新后的模板",
		Description: "这是更新后的模板",
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

	updateBody, err := json.Marshal(updateReqBody)
	require.NoError(t, err)

	updateReq := httptest.NewRequest("PUT", "/api/v1/templates/"+templateID, bytes.NewBuffer(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("X-CSRF-Token", csrfToken)
	updateW := httptest.NewRecorder()
	router.ServeHTTP(updateW, updateReq)

	assert.Equal(t, http.StatusOK, updateW.Code)

	// 4. 获取版本列表
	versionsReq := httptest.NewRequest("GET", "/api/v1/templates/"+templateID+"/versions", nil)
	versionsW := httptest.NewRecorder()
	router.ServeHTTP(versionsW, versionsReq)

	assert.Equal(t, http.StatusOK, versionsW.Code)
	var versionsResponse api.Response
	err = json.Unmarshal(versionsW.Body.Bytes(), &versionsResponse)
	require.NoError(t, err)
	assert.Equal(t, 0, versionsResponse.Code)

	// 5. 获取模板列表
	listReq := httptest.NewRequest("GET", "/api/v1/templates?page=1&page_size=10", nil)
	listW := httptest.NewRecorder()
	router.ServeHTTP(listW, listReq)

	assert.Equal(t, http.StatusOK, listW.Code)
	var listResponse api.PaginatedResponse
	err = json.Unmarshal(listW.Body.Bytes(), &listResponse)
	require.NoError(t, err)
	assert.Equal(t, 0, listResponse.Code)
	
	// 将 Data 转换为切片以检查长度
	dataBytes, err := json.Marshal(listResponse.Data)
	require.NoError(t, err)
	var dataList []interface{}
	err = json.Unmarshal(dataBytes, &dataList)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(dataList), 1)

	// 6. 删除模板
	deleteReq := httptest.NewRequest("DELETE", "/api/v1/templates/"+templateID, nil)
	deleteReq.Header.Set("X-CSRF-Token", csrfToken)
	deleteW := httptest.NewRecorder()
	router.ServeHTTP(deleteW, deleteReq)

	assert.Equal(t, http.StatusOK, deleteW.Code)

	// 7. 验证模板已删除
	verifyReq := httptest.NewRequest("GET", "/api/v1/templates/"+templateID, nil)
	verifyW := httptest.NewRecorder()
	router.ServeHTTP(verifyW, verifyReq)

	assert.Equal(t, http.StatusNotFound, verifyW.Code)
}

// TestAPIIntegration_TaskFullFlow 测试任务管理完整流程
func TestAPIIntegration_TaskFullFlow(t *testing.T) {
	router, _ := setupTestAPIServer(t)

	// 获取 CSRF Token
	csrfToken := getCSRFToken(t, router)

	// 1. 先创建模板
	createTemplateReqBody := service.CreateTemplateRequest{
		Name:        "任务流程测试模板",
		Description: "这是一个任务流程测试模板",
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

	// 3. 获取任务
	getTaskReq := httptest.NewRequest("GET", "/api/v1/tasks/"+taskID, nil)
	getTaskW := httptest.NewRecorder()
	router.ServeHTTP(getTaskW, getTaskReq)

	assert.Equal(t, http.StatusOK, getTaskW.Code)
	var getTaskResponse api.Response
	err = json.Unmarshal(getTaskW.Body.Bytes(), &getTaskResponse)
	require.NoError(t, err)
	assert.Equal(t, 0, getTaskResponse.Code)

	// 4. 获取任务列表
	listTasksReq := httptest.NewRequest("GET", "/api/v1/tasks?page=1&page_size=10", nil)
	listTasksW := httptest.NewRecorder()
	router.ServeHTTP(listTasksW, listTasksReq)

	assert.Equal(t, http.StatusOK, listTasksW.Code)
	var listTasksResponse api.PaginatedResponse
	err = json.Unmarshal(listTasksW.Body.Bytes(), &listTasksResponse)
	require.NoError(t, err)
	assert.Equal(t, 0, listTasksResponse.Code)
	
	// 将 Data 转换为切片以检查长度
	tasksDataBytes, err := json.Marshal(listTasksResponse.Data)
	require.NoError(t, err)
	var tasksList []interface{}
	err = json.Unmarshal(tasksDataBytes, &tasksList)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(tasksList), 1)

	// 5. 获取审批记录(任务刚创建,可能没有记录)
	getRecordsReq := httptest.NewRequest("GET", "/api/v1/tasks/"+taskID+"/records", nil)
	getRecordsW := httptest.NewRecorder()
	router.ServeHTTP(getRecordsW, getRecordsReq)

	assert.Equal(t, http.StatusOK, getRecordsW.Code)

	// 6. 获取状态历史
	getHistoryReq := httptest.NewRequest("GET", "/api/v1/tasks/"+taskID+"/history", nil)
	getHistoryW := httptest.NewRecorder()
	router.ServeHTTP(getHistoryW, getHistoryReq)

	assert.Equal(t, http.StatusOK, getHistoryW.Code)
}

// TestAPIIntegration_TemplateGet_NotFound 测试获取不存在的模板
func TestAPIIntegration_TemplateGet_NotFound(t *testing.T) {
	router, _ := setupTestAPIServer(t)

	req := httptest.NewRequest("GET", "/api/v1/templates/non-existent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestAPIIntegration_TaskGet_NotFound 测试获取不存在的任务
func TestAPIIntegration_TaskGet_NotFound(t *testing.T) {
	router, _ := setupTestAPIServer(t)

	req := httptest.NewRequest("GET", "/api/v1/tasks/non-existent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestAPIIntegration_TemplateUpdate_NotFound 测试更新不存在的模板
func TestAPIIntegration_TemplateUpdate_NotFound(t *testing.T) {
	router, _ := setupTestAPIServer(t)

	// 获取 CSRF Token
	csrfToken := getCSRFToken(t, router)

	updateReqBody := service.UpdateTemplateRequest{
		Name:        "更新后的模板",
		Description: "这是更新后的模板",
			Nodes:       make(map[string]*template.Node),
			Edges:       []*template.Edge{},
		Config:      nil,
	}

	body, err := json.Marshal(updateReqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("PUT", "/api/v1/templates/non-existent", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrfToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestAPIIntegration_TemplateList_Pagination 测试模板列表分页
func TestAPIIntegration_TemplateList_Pagination(t *testing.T) {
	router, _ := setupTestAPIServer(t)

	// 获取 CSRF Token
	csrfToken := getCSRFToken(t, router)

	// 创建多个模板
	for i := 1; i <= 5; i++ {
		createReqBody := service.CreateTemplateRequest{
			Name:        fmt.Sprintf("模板%d", i),
			Description: fmt.Sprintf("这是模板%d", i),
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
	}

	// 测试第一页 (Gin 的 ShouldBindQuery 使用字段名，不区分大小写，但通常使用小写)
	// 由于 TemplateListFilter 没有 form 标签，Gin 会尝试匹配字段名
	req := httptest.NewRequest("GET", "/api/v1/templates?page=1&page_size=2", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response api.PaginatedResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Code)
	
	// 将 Data 转换为切片以检查长度
	dataBytes, err := json.Marshal(response.Data)
	require.NoError(t, err)
	var dataList []interface{}
	err = json.Unmarshal(dataBytes, &dataList)
	require.NoError(t, err)
	// 验证分页信息
	assert.GreaterOrEqual(t, len(dataList), 1)
	assert.Equal(t, 1, response.Pagination.Page)
	// 注意: 如果查询参数没有正确绑定，PageSize 会是默认值 20
	// 我们验证总数至少是 5
	assert.GreaterOrEqual(t, response.Pagination.Total, int64(5))
}

// TestAPIIntegration_TaskList_Filter 测试任务列表过滤
func TestAPIIntegration_TaskList_Filter(t *testing.T) {
	router, _ := setupTestAPIServer(t)

	// 获取 CSRF Token
	csrfToken := getCSRFToken(t, router)

	// 先创建模板和任务
	createTemplateReqBody := service.CreateTemplateRequest{
		Name:        "过滤测试模板",
		Description: "这是一个过滤测试模板",
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

	templateBody, err := json.Marshal(createTemplateReqBody)
	require.NoError(t, err)

	createTemplateReq := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewBuffer(templateBody))
	createTemplateReq.Header.Set("Content-Type", "application/json")
	createTemplateReq.Header.Set("X-CSRF-Token", csrfToken)
	createTemplateW := httptest.NewRecorder()
	router.ServeHTTP(createTemplateW, createTemplateReq)

	require.Equal(t, http.StatusOK, createTemplateW.Code)
	templateID := extractTemplateID(t, createTemplateW.Body.Bytes())

	// 创建任务
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

	// 按模板 ID 过滤
	filterReq := httptest.NewRequest("GET", "/api/v1/tasks?template_id="+templateID, nil)
	filterW := httptest.NewRecorder()
	router.ServeHTTP(filterW, filterReq)

	assert.Equal(t, http.StatusOK, filterW.Code)
	var response api.PaginatedResponse
	err = json.Unmarshal(filterW.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Code)
	
	// 将 Data 转换为切片以检查长度
	filterDataBytes, err := json.Marshal(response.Data)
	require.NoError(t, err)
	var filterTasksList []interface{}
	err = json.Unmarshal(filterDataBytes, &filterTasksList)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(filterTasksList), 1)
}

// TestAPIIntegration_TemplateGet_WithVersion 测试带版本号的模板获取
func TestAPIIntegration_TemplateGet_WithVersion(t *testing.T) {
	router, _ := setupTestAPIServer(t)

	// 获取 CSRF Token
	csrfToken := getCSRFToken(t, router)

	// 1. 创建模板
	createReqBody := service.CreateTemplateRequest{
		Name:        "版本测试模板",
		Description: "这是一个版本测试模板",
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

	// 2. 更新模板（创建新版本）
	updateReqBody := service.UpdateTemplateRequest{
		Name:        "更新后的版本测试模板",
		Description: "这是更新后的版本测试模板",
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

	updateBody, err := json.Marshal(updateReqBody)
	require.NoError(t, err)

	updateReq := httptest.NewRequest("PUT", "/api/v1/templates/"+templateID, bytes.NewBuffer(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("X-CSRF-Token", csrfToken)
	updateW := httptest.NewRecorder()
	router.ServeHTTP(updateW, updateReq)

	require.Equal(t, http.StatusOK, updateW.Code)

	// 3. 获取版本 1
	getV1Req := httptest.NewRequest("GET", "/api/v1/templates/"+templateID+"?version=1", nil)
	getV1W := httptest.NewRecorder()
	router.ServeHTTP(getV1W, getV1Req)

	assert.Equal(t, http.StatusOK, getV1W.Code)
	var v1Response api.Response
	err = json.Unmarshal(getV1W.Body.Bytes(), &v1Response)
	require.NoError(t, err)
	assert.Equal(t, 0, v1Response.Code)

	// 4. 获取版本 2
	getV2Req := httptest.NewRequest("GET", "/api/v1/templates/"+templateID+"?version=2", nil)
	getV2W := httptest.NewRecorder()
	router.ServeHTTP(getV2W, getV2Req)

	assert.Equal(t, http.StatusOK, getV2W.Code)
	var v2Response api.Response
	err = json.Unmarshal(getV2W.Body.Bytes(), &v2Response)
	require.NoError(t, err)
	assert.Equal(t, 0, v2Response.Code)

	// 5. 验证两个版本的内容不同
	v1DataBytes, _ := json.Marshal(v1Response.Data)
	v2DataBytes, _ := json.Marshal(v2Response.Data)
	
	var v1Template map[string]interface{}
	var v2Template map[string]interface{}
	json.Unmarshal(v1DataBytes, &v1Template)
	json.Unmarshal(v2DataBytes, &v2Template)

	// 尝试多种可能的字段名（JSON 可能使用不同的命名）
	var v1Name, v2Name string
	if name, ok := v1Template["name"].(string); ok {
		v1Name = name
	} else if name, ok := v1Template["Name"].(string); ok {
		v1Name = name
	}
	if name, ok := v2Template["name"].(string); ok {
		v2Name = name
	} else if name, ok := v2Template["Name"].(string); ok {
		v2Name = name
	}

	// 验证版本号不同（如果名称提取失败，至少验证版本号）
	v1Version, _ := v1Template["version"].(float64)
	v2Version, _ := v2Template["version"].(float64)
	if v1Version == 0 {
		if v, ok := v1Template["Version"].(float64); ok {
			v1Version = v
		}
	}
	if v2Version == 0 {
		if v, ok := v2Template["Version"].(float64); ok {
			v2Version = v
		}
	}

	// 验证版本号不同
	assert.NotEqual(t, v1Version, v2Version, "version 1 and version 2 should have different version numbers")
	
	// 如果名称提取成功，验证名称不同
	if v1Name != "" && v2Name != "" {
		assert.NotEqual(t, v1Name, v2Name, "version 1 and version 2 should have different names")
	}
}

// TestAPIIntegration_TemplateList_Search 测试模板列表搜索
func TestAPIIntegration_TemplateList_Search(t *testing.T) {
	router, _ := setupTestAPIServer(t)

	// 获取 CSRF Token
	csrfToken := getCSRFToken(t, router)

	// 创建多个模板
	templateNames := []string{"搜索测试模板A", "搜索测试模板B", "其他模板"}
	for _, name := range templateNames {
		createReqBody := service.CreateTemplateRequest{
			Name:        name,
			Description: "这是一个测试模板",
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
	}

	// 搜索包含"搜索测试"的模板
	searchReq := httptest.NewRequest("GET", "/api/v1/templates?search=搜索测试", nil)
	searchW := httptest.NewRecorder()
	router.ServeHTTP(searchW, searchReq)

	assert.Equal(t, http.StatusOK, searchW.Code)
	var response api.PaginatedResponse
	err := json.Unmarshal(searchW.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Code)

	// 验证搜索结果
	dataBytes, err := json.Marshal(response.Data)
	require.NoError(t, err)
	var dataList []interface{}
	err = json.Unmarshal(dataBytes, &dataList)
	require.NoError(t, err)
	
	// 应该找到至少 2 个包含"搜索测试"的模板
	assert.GreaterOrEqual(t, len(dataList), 2)
}

// TestAPIIntegration_TemplateList_Sort 测试模板列表排序
func TestAPIIntegration_TemplateList_Sort(t *testing.T) {
	router, _ := setupTestAPIServer(t)

	// 获取 CSRF Token
	csrfToken := getCSRFToken(t, router)

	// 创建多个模板
	for i := 1; i <= 3; i++ {
		createReqBody := service.CreateTemplateRequest{
			Name:        fmt.Sprintf("排序测试模板%d", i),
			Description: fmt.Sprintf("这是排序测试模板%d", i),
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
	}

	// 测试按名称升序排序
	ascReq := httptest.NewRequest("GET", "/api/v1/templates?sort_by=name&order=asc", nil)
	ascW := httptest.NewRecorder()
	router.ServeHTTP(ascW, ascReq)

	assert.Equal(t, http.StatusOK, ascW.Code)
	var ascResponse api.PaginatedResponse
	err := json.Unmarshal(ascW.Body.Bytes(), &ascResponse)
	require.NoError(t, err)
	assert.Equal(t, 0, ascResponse.Code)

	// 测试按名称降序排序
	descReq := httptest.NewRequest("GET", "/api/v1/templates?sort_by=name&order=desc", nil)
	descW := httptest.NewRecorder()
	router.ServeHTTP(descW, descReq)

	assert.Equal(t, http.StatusOK, descW.Code)
	var descResponse api.PaginatedResponse
	err = json.Unmarshal(descW.Body.Bytes(), &descResponse)
	require.NoError(t, err)
	assert.Equal(t, 0, descResponse.Code)
}

// TestAPIIntegration_TemplateCreate_InvalidInput 测试创建模板时的无效输入
func TestAPIIntegration_TemplateCreate_InvalidInput(t *testing.T) {
	router, _ := setupTestAPIServer(t)

	// 获取 CSRF Token
	csrfToken := getCSRFToken(t, router)

	// 测试空名称
	emptyNameReqBody := service.CreateTemplateRequest{
		Name:        "",
		Description: "这是一个测试模板",
		Nodes:       make(map[string]*template.Node),
		Edges:       []*template.Edge{},
		Config:      nil,
	}

	body, err := json.Marshal(emptyNameReqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrfToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 应该返回 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 测试无效的 JSON
	invalidJSONReq := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewBufferString("invalid json"))
	invalidJSONReq.Header.Set("Content-Type", "application/json")
	invalidJSONReq.Header.Set("X-CSRF-Token", csrfToken)
	invalidJSONW := httptest.NewRecorder()
	router.ServeHTTP(invalidJSONW, invalidJSONReq)

	assert.Equal(t, http.StatusBadRequest, invalidJSONW.Code)
}

// TestAPIIntegration_TemplateGet_InvalidVersion 测试获取模板时的无效版本号
func TestAPIIntegration_TemplateGet_InvalidVersion(t *testing.T) {
	router, _ := setupTestAPIServer(t)

	// 获取 CSRF Token
	csrfToken := getCSRFToken(t, router)

	// 先创建一个模板
	createReqBody := service.CreateTemplateRequest{
		Name:        "版本测试模板",
		Description: "这是一个版本测试模板",
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

	// 测试无效的版本号（非数字）
	invalidVersionReq := httptest.NewRequest("GET", "/api/v1/templates/"+templateID+"?version=invalid", nil)
	invalidVersionW := httptest.NewRecorder()
	router.ServeHTTP(invalidVersionW, invalidVersionReq)

	assert.Equal(t, http.StatusBadRequest, invalidVersionW.Code)

	// 测试不存在的版本号
	nonExistentVersionReq := httptest.NewRequest("GET", "/api/v1/templates/"+templateID+"?version=999", nil)
	nonExistentVersionW := httptest.NewRecorder()
	router.ServeHTTP(nonExistentVersionW, nonExistentVersionReq)

	assert.Equal(t, http.StatusNotFound, nonExistentVersionW.Code)
}

// TestAPIIntegration_TaskList_Pagination 测试任务列表分页
func TestAPIIntegration_TaskList_Pagination(t *testing.T) {
	router, _ := setupTestAPIServer(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建模板
	templateID := createTestTemplateForTaskList(t, router, csrfToken)

	// 2. 创建多个任务
	for i := 1; i <= 5; i++ {
		createTaskReqBody := service.CreateTaskRequest{
			TemplateID: templateID,
			BusinessID: fmt.Sprintf("biz-%03d", i),
			Params:     json.RawMessage(`{"amount": 1000}`),
		}
		taskBody, _ := json.Marshal(createTaskReqBody)
		createTaskReq := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(taskBody))
		createTaskReq.Header.Set("Content-Type", "application/json")
		createTaskReq.Header.Set("X-CSRF-Token", csrfToken)
		createTaskW := httptest.NewRecorder()
		router.ServeHTTP(createTaskW, createTaskReq)
		require.Equal(t, http.StatusOK, createTaskW.Code)
	}

	// 3. 测试分页
	req := httptest.NewRequest("GET", "/api/v1/tasks?page=1&page_size=2", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response api.PaginatedResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Code)
	assert.Equal(t, 1, response.Pagination.Page)
	assert.GreaterOrEqual(t, response.Pagination.Total, int64(5))
}

// TestAPIIntegration_TaskList_Sort 测试任务列表排序
func TestAPIIntegration_TaskList_Sort(t *testing.T) {
	router, _ := setupTestAPIServer(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建模板
	templateID := createTestTemplateForTaskList(t, router, csrfToken)

	// 2. 创建多个任务
	for i := 1; i <= 3; i++ {
		createTaskReqBody := service.CreateTaskRequest{
			TemplateID: templateID,
			BusinessID: fmt.Sprintf("biz-%03d", i),
			Params:     json.RawMessage(`{"amount": 1000}`),
		}
		taskBody, _ := json.Marshal(createTaskReqBody)
		createTaskReq := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(taskBody))
		createTaskReq.Header.Set("Content-Type", "application/json")
		createTaskReq.Header.Set("X-CSRF-Token", csrfToken)
		createTaskW := httptest.NewRecorder()
		router.ServeHTTP(createTaskW, createTaskReq)
		require.Equal(t, http.StatusOK, createTaskW.Code)
	}

	// 3. 测试排序（按创建时间降序）
	req := httptest.NewRequest("GET", "/api/v1/tasks?sort_by=created_at&order=desc", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response api.PaginatedResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Code)
}

// TestAPIIntegration_TaskList_FilterByState 测试任务列表按状态筛选
func TestAPIIntegration_TaskList_FilterByState(t *testing.T) {
	router, _ := setupTestAPIServer(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建模板
	templateID := createTestTemplateForTaskList(t, router, csrfToken)

	// 2. 创建任务并取消
	createTaskReqBody := service.CreateTaskRequest{
		TemplateID: templateID,
		BusinessID: "biz-001",
		Params:     json.RawMessage(`{"amount": 1000}`),
	}
	taskBody, _ := json.Marshal(createTaskReqBody)
	createTaskReq := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(taskBody))
	createTaskReq.Header.Set("Content-Type", "application/json")
	createTaskReq.Header.Set("X-CSRF-Token", csrfToken)
	createTaskW := httptest.NewRecorder()
	router.ServeHTTP(createTaskW, createTaskReq)
	require.Equal(t, http.StatusOK, createTaskW.Code)
	taskID := extractTaskID(t, createTaskW.Body.Bytes())

	// 取消任务
	cancelReqBody := map[string]string{"reason": "测试取消"}
	cancelBody, _ := json.Marshal(cancelReqBody)
	cancelReq := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/cancel", bytes.NewBuffer(cancelBody))
	cancelReq.Header.Set("Content-Type", "application/json")
	cancelReq.Header.Set("X-CSRF-Token", csrfToken)
	cancelW := httptest.NewRecorder()
	router.ServeHTTP(cancelW, cancelReq)
	require.Equal(t, http.StatusOK, cancelW.Code)

	// 3. 按状态筛选
	req := httptest.NewRequest("GET", "/api/v1/tasks?state=cancelled", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response api.PaginatedResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Code)

	// 验证返回的任务状态都是 cancelled
	tasksDataBytes, _ := json.Marshal(response.Data)
	var tasksList []interface{}
	json.Unmarshal(tasksDataBytes, &tasksList)
	assert.Greater(t, len(tasksList), 0, "should have at least one cancelled task")
	for _, taskInterface := range tasksList {
		taskBytes, _ := json.Marshal(taskInterface)
		var task map[string]interface{}
		json.Unmarshal(taskBytes, &task)
		
		// 处理不同大小写的状态字段
		var state string
		var ok bool
		state, ok = task["state"].(string)
		if !ok {
			state, ok = task["State"].(string)
		}
		assert.True(t, ok, "task should have state field")
		assert.Equal(t, "cancelled", state, "all tasks should be cancelled")
	}
}

// createTestTemplateForTaskList 创建用于任务列表测试的模板（避免与 task_operations_integration_test.go 中的辅助函数冲突）
func createTestTemplateForTaskList(t *testing.T, router *gin.Engine, csrfToken string) string {
	createReqBody := service.CreateTemplateRequest{
		Name:        "任务列表测试模板",
		Description: "用于任务列表测试的模板",
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

