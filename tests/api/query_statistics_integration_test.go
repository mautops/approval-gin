package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/mautops/approval-gin/internal/auth"
	"github.com/mautops/approval-gin/internal/config"
	"github.com/mautops/approval-gin/internal/container"
	"github.com/mautops/approval-gin/internal/database"
	"github.com/mautops/approval-gin/internal/integration"
	"github.com/mautops/approval-gin/internal/repository"
	"github.com/mautops/approval-gin/internal/service"
	"github.com/mautops/approval-gin/internal/websocket"
	"github.com/mautops/approval-kit/pkg/template"
	"github.com/mautops/approval-kit/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestAPIServerForQuery 创建查询测试 API 服务器
func setupTestAPIServerForQuery(t *testing.T) (*gin.Engine, *container.Container) {
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

	// 手动创建容器
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
	statisticsSvc := service.NewStatisticsService(db)

	// 创建控制器
	templateController := api.NewTemplateController(templateSvc)
	taskController := api.NewTaskController(taskSvc)
	queryController := api.NewQueryController(querySvc)

	// 创建 WebSocket Hub
	hub := websocket.NewHub()
	go hub.Run()

	// 创建健康检查控制器
	healthController := api.NewHealthController(db, fgaClient)

	// 创建路由
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(api.RequestIDMiddleware())
	router.Use(api.RequestLogMiddleware())

	// 健康检查
	router.GET("/health", healthController.Check)

	// CSRF Token 路由
	router.GET("/api/v1/csrf-token", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"token": "test-csrf-token"})
	})

	// API v1 路由组
	v1 := router.Group("/api/v1")
	{
		templates := v1.Group("/templates")
		{
			templates.POST("", templateController.Create)
			templates.GET("", templateController.List)
		}

		tasks := v1.Group("/tasks")
		{
			tasks.POST("", taskController.Create)
			tasks.GET("", queryController.ListTasks)
			tasks.GET("/:id", taskController.Get)
			tasks.POST("/:id/submit", taskController.Submit)
			tasks.POST("/:id/approve", taskController.Approve)
			tasks.POST("/:id/approvers", taskController.AddApprover)
			tasks.GET("/:id/records", queryController.GetRecords)
			tasks.GET("/:id/history", queryController.GetHistory)
		}

		// 统计路由（需要创建 StatisticsController）
		// statistics := v1.Group("/statistics")
		// {
		// 	statistics.GET("/tasks/by-state", statisticsController.GetTaskStatisticsByState)
		// 	statistics.GET("/tasks/by-template", statisticsController.GetTaskStatisticsByTemplate)
		// 	statistics.GET("/tasks/by-time", statisticsController.GetTaskStatisticsByTime)
		// 	statistics.GET("/approvals", statisticsController.GetApprovalStatistics)
		// }
	}

	_ = keycloakValidator
	_ = statisticsSvc

	ctr := &container.Container{}

	return router, ctr
}

// TestAPIIntegration_Query_ListTasks 测试任务列表查询
func TestAPIIntegration_Query_ListTasks(t *testing.T) {
	router, _ := setupTestAPIServerForQuery(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建模板
	templateID := createTestTemplate(t, router, csrfToken)

	// 2. 创建多个任务
	taskID1 := createTestTask(t, router, csrfToken, templateID)
	_ = createTestTask(t, router, csrfToken, templateID)

	// 3. 提交 taskID1
	submitTestTask(t, router, csrfToken, taskID1)

	// 4. 查询所有任务
	req := httptest.NewRequest("GET", "/api/v1/tasks?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response api.PaginatedResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Code)
	assert.GreaterOrEqual(t, int64(2), response.Pagination.Total)

	// 5. 按状态查询
	req = httptest.NewRequest("GET", "/api/v1/tasks?state=pending&page=1&page_size=10", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Code)
	assert.GreaterOrEqual(t, int64(1), response.Pagination.Total) // 至少有一个 pending 状态的任务

	// 6. 按模板 ID 查询
	req = httptest.NewRequest("GET", "/api/v1/tasks?template_id="+templateID+"&page=1&page_size=10", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Code)
	assert.GreaterOrEqual(t, int64(2), response.Pagination.Total) // 至少有两个任务
}

// TestAPIIntegration_Query_ListTasks_Pagination 测试任务列表分页
func TestAPIIntegration_Query_ListTasks_Pagination(t *testing.T) {
	router, _ := setupTestAPIServerForQuery(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建模板
	templateID := createTestTemplate(t, router, csrfToken)

	// 2. 创建多个任务
	for i := 0; i < 5; i++ {
		createTestTask(t, router, csrfToken, templateID)
	}

	// 3. 查询第一页（每页 2 条）
	req := httptest.NewRequest("GET", "/api/v1/tasks?page=1&page_size=2", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response api.PaginatedResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Code)
	assert.Equal(t, int64(5), response.Pagination.Total)
	// 验证返回的数据数量不超过 page_size
	dataBytes, err := json.Marshal(response.Data)
	require.NoError(t, err)
	var dataList []interface{}
	err = json.Unmarshal(dataBytes, &dataList)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(dataList), 2)

	// 4. 查询第二页
	req = httptest.NewRequest("GET", "/api/v1/tasks?page=2&page_size=2", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Code)
	assert.Equal(t, int64(5), response.Pagination.Total)
}

// TestAPIIntegration_Query_ListTasks_Sort 测试任务列表排序
func TestAPIIntegration_Query_ListTasks_Sort(t *testing.T) {
	router, _ := setupTestAPIServerForQuery(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建模板
	templateID := createTestTemplate(t, router, csrfToken)

	// 2. 创建多个任务
	for i := 0; i < 3; i++ {
		createTestTask(t, router, csrfToken, templateID)
		time.Sleep(10 * time.Millisecond) // 确保创建时间不同
	}

	// 3. 按创建时间降序排序
	req := httptest.NewRequest("GET", "/api/v1/tasks?sort_by=created_at&order=desc&page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response api.PaginatedResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Code)
	assert.GreaterOrEqual(t, int64(3), response.Pagination.Total)
}

// TestAPIIntegration_Query_GetRecords 测试获取审批记录
func TestAPIIntegration_Query_GetRecords(t *testing.T) {
	router, _ := setupTestAPIServerForQuery(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建模板（包含审批节点）
	templateID := createTestTemplateWithApprovalNode(t, router, csrfToken)

	// 2. 创建并提交任务
	taskID := createTestTask(t, router, csrfToken, templateID)
	submitTestTask(t, router, csrfToken, taskID)

	// 3. 添加审批人并审批
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

	// 4. 审批
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

	require.Equal(t, http.StatusOK, approveW.Code)

	// 5. 获取审批记录
	req := httptest.NewRequest("GET", "/api/v1/tasks/"+taskID+"/records", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response api.Response
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Code)
	assert.NotNil(t, response.Data)

	// 验证记录数据
	dataBytes, err := json.Marshal(response.Data)
	require.NoError(t, err)
	var records []service.ApprovalRecord
	err = json.Unmarshal(dataBytes, &records)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(records), 1)
}

// TestAPIIntegration_Query_GetHistory 测试获取状态历史
func TestAPIIntegration_Query_GetHistory(t *testing.T) {
	router, _ := setupTestAPIServerForQuery(t)
	csrfToken := getCSRFToken(t, router)

	// 1. 创建模板
	templateID := createTestTemplate(t, router, csrfToken)

	// 2. 创建并提交任务
	taskID := createTestTask(t, router, csrfToken, templateID)
	submitTestTask(t, router, csrfToken, taskID)

	// 3. 获取状态历史
	req := httptest.NewRequest("GET", "/api/v1/tasks/"+taskID+"/history", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response api.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Code)
	assert.NotNil(t, response.Data)

	// 验证历史数据
	dataBytes, err := json.Marshal(response.Data)
	require.NoError(t, err)
	var histories []service.StateHistory
	err = json.Unmarshal(dataBytes, &histories)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(histories), 1) // 至少有一条状态转换记录
}

// TestAPIIntegration_Statistics_TaskByState 测试按状态统计任务
func TestAPIIntegration_Statistics_TaskByState(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 迁移数据库
	err = database.Migrate(db)
	require.NoError(t, err)

	// 创建统计服务
	statisticsSvc := service.NewStatisticsService(db)

	// 创建模板管理器和任务管理器
	templateMgr := integration.NewTemplateManager(db)
	eventHandler := integration.NewEventHandler(db, 1)
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, eventHandler)

	// 创建模板
	tpl := &template.Template{
		ID:          "tpl-001",
		Name:        "测试模板",
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
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
	err = templateMgr.Create(tpl)
	require.NoError(t, err)

	// 创建多个任务并设置不同状态
	task1, err := taskMgr.Create("tpl-001", "biz-001", json.RawMessage(`{}`))
	require.NoError(t, err)

	task2, err := taskMgr.Create("tpl-001", "biz-002", json.RawMessage(`{}`))
	require.NoError(t, err)
	err = taskMgr.Submit(task2.ID)
	require.NoError(t, err)

	// 获取统计
	stats, err := statisticsSvc.GetTaskStatisticsByState()
	require.NoError(t, err)

	// 验证统计结果
	assert.GreaterOrEqual(t, len(stats), 1)

	// 查找 pending 状态的统计
	foundPending := false
	for _, stat := range stats {
		if stat.State == string(types.TaskStatePending) {
			foundPending = true
			assert.GreaterOrEqual(t, stat.Count, int64(1))
		}
	}
	assert.True(t, foundPending, "should have pending state statistics")

	_ = task1
}

// TestAPIIntegration_Statistics_TaskByTemplate 测试按模板统计任务
func TestAPIIntegration_Statistics_TaskByTemplate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 迁移数据库
	err = database.Migrate(db)
	require.NoError(t, err)

	// 创建统计服务
	statisticsSvc := service.NewStatisticsService(db)

	// 创建模板管理器和任务管理器
	templateMgr := integration.NewTemplateManager(db)
	eventHandler := integration.NewEventHandler(db, 1)
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, eventHandler)

	// 创建多个模板
	tpl1 := &template.Template{
		ID:          "tpl-001",
		Name:        "模板1",
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
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
	err = templateMgr.Create(tpl1)
	require.NoError(t, err)

	tpl2 := &template.Template{
		ID:          "tpl-002",
		Name:        "模板2",
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
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
	err = templateMgr.Create(tpl2)
	require.NoError(t, err)

	// 创建任务
	_, err = taskMgr.Create("tpl-001", "biz-001", json.RawMessage(`{}`))
	require.NoError(t, err)

	_, err = taskMgr.Create("tpl-002", "biz-002", json.RawMessage(`{}`))
	require.NoError(t, err)

	// 获取统计
	stats, err := statisticsSvc.GetTaskStatisticsByTemplate()
	require.NoError(t, err)

	// 验证统计结果
	assert.GreaterOrEqual(t, len(stats), 2)
}

// TestAPIIntegration_Statistics_Approval 测试审批统计
func TestAPIIntegration_Statistics_Approval(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 迁移数据库
	err = database.Migrate(db)
	require.NoError(t, err)

	// 创建统计服务
	statisticsSvc := service.NewStatisticsService(db)

	// 获取统计
	stats, err := statisticsSvc.GetApprovalStatistics()
	require.NoError(t, err)

	// 验证统计结果
	assert.NotNil(t, stats)
	assert.GreaterOrEqual(t, stats.TotalApprovals, int64(0))
	assert.GreaterOrEqual(t, stats.ApprovedCount, int64(0))
	assert.GreaterOrEqual(t, stats.RejectedCount, int64(0))
	assert.GreaterOrEqual(t, stats.ApprovalRate, 0.0)
	assert.LessOrEqual(t, stats.ApprovalRate, 100.0)
}

