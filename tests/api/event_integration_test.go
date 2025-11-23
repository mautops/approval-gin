package api_test

import (
	"encoding/json"
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
	"github.com/mautops/approval-kit/pkg/event"
	"github.com/mautops/approval-kit/pkg/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestAPIServerWithEvents 创建带事件处理的测试 API 服务器
func setupTestAPIServerWithEvents(t *testing.T) (*gin.Engine, *container.Container, event.EventHandler) {
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
	_ = auth.NewKeycloakTokenValidator(cfg.Keycloak.Issuer)

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

	// 创建路由
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(api.RequestIDMiddleware())
	router.Use(api.RequestLogMiddleware())

	// 健康检查
	router.GET("/health", healthController.Check)

	// API v1 路由组（不使用认证中间件，测试中直接使用 mock token）
	v1 := router.Group("/api/v1")
	{
		templates := v1.Group("/templates")
		{
			templates.POST("", templateController.Create)
			templates.GET("", templateController.List)
			templates.GET("/:id", templateController.Get)
		}

		tasks := v1.Group("/tasks")
		{
			tasks.POST("", taskController.Create)
			tasks.GET("", queryController.ListTasks)
			tasks.GET("/:id", taskController.Get)
			tasks.POST("/:id/submit", taskController.Submit)
			tasks.POST("/:id/approve", taskController.Approve)
			tasks.POST("/:id/reject", taskController.Reject)
		}
	}

	ctr := &container.Container{}

	return router, ctr, eventHandler
}

// TestAPIIntegration_Event_Generation 测试事件生成
func TestAPIIntegration_Event_Generation(t *testing.T) {
	_, _, _ = setupTestAPIServerWithEvents(t)

	// 注意: 事件生成测试需要验证事件是否在任务操作时自动生成
	// 由于事件处理是异步的，这里先跳过详细验证
	// 等实现事件查询 API 后再完善测试
	t.Skip("Event generation test requires event query API, skipping for now")
}

// TestAPIIntegration_Event_Persistence 测试事件持久化
func TestAPIIntegration_Event_Persistence(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 迁移数据库
	err = database.Migrate(db)
	require.NoError(t, err)

	// 创建事件处理器
	eventHandler := integration.NewEventHandler(db, 1)

	// 创建测试事件
	testEvent := &event.Event{
		Type: event.EventTypeTaskCreated,
		Task: &event.TaskInfo{
			ID:    "task-001",
			State: "pending",
		},
		Time: time.Now(),
	}

	// 处理事件
	err = eventHandler.Handle(testEvent)
	require.NoError(t, err)

	// 等待事件处理
	time.Sleep(100 * time.Millisecond)

	// 验证事件已持久化
	eventRepo := repository.NewEventRepository(db)
	events, err := eventRepo.FindByTaskID("task-001")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(events), 1)

	// 验证事件数据
	eventModel := events[0]
	assert.Equal(t, "task-001", eventModel.TaskID)
	assert.Equal(t, string(event.EventTypeTaskCreated), eventModel.Type)
	assert.Contains(t, []string{"pending", "success", "failed"}, eventModel.Status)
}

// TestAPIIntegration_Event_WebhookPush 测试事件 Webhook 推送
func TestAPIIntegration_Event_WebhookPush(t *testing.T) {
	// 注意: Webhook 推送需要真实的 HTTP 服务器
	// 这里先跳过，等实现 mock HTTP 服务器后再完善测试
	t.Skip("Webhook push test requires mock HTTP server, skipping for now")
}

// TestAPIIntegration_Event_FullFlow 测试事件处理完整流程
func TestAPIIntegration_Event_FullFlow(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 迁移数据库
	err = database.Migrate(db)
	require.NoError(t, err)

	// 创建模板管理器
	templateMgr := integration.NewTemplateManager(db)

	// 创建带 Webhook 配置的模板
	tpl := &template.Template{
		ID:          "tpl-001",
		Name:        "测试模板",
		Description: "这是一个测试模板",
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
		Config: &template.TemplateConfig{
			Webhooks: []*template.WebhookConfig{
				{
					URL:    "http://localhost:8080/webhook",
					Method: "POST",
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
				},
			},
		},
	}
	err = templateMgr.Create(tpl)
	require.NoError(t, err)

	// 创建事件处理器
	eventHandler := integration.NewEventHandler(db, 1)

	// 创建任务
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, eventHandler)
	tsk, err := taskMgr.Create("tpl-001", "biz-001", json.RawMessage(`{}`))
	require.NoError(t, err)

	// 创建任务创建事件
	taskEvent := &event.Event{
		Type: event.EventTypeTaskCreated,
		Task: &event.TaskInfo{
			ID:    tsk.ID,
			State: string(tsk.State),
		},
		Time: time.Now(),
	}

	// 处理事件
	err = eventHandler.Handle(taskEvent)
	require.NoError(t, err)

	// 等待事件处理
	time.Sleep(200 * time.Millisecond)

	// 验证事件已持久化
	eventRepo := repository.NewEventRepository(db)
	events, err := eventRepo.FindByTaskID(tsk.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(events), 1)

	// 验证事件状态（由于 Webhook 服务器不存在，应该失败或 pending）
	eventModel := events[0]
	assert.Contains(t, []string{"pending", "success", "failed"}, eventModel.Status)
}

