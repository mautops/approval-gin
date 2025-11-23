package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	gorillaWS "github.com/gorilla/websocket"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/mautops/approval-gin/internal/auth"
	"github.com/mautops/approval-gin/internal/config"
	"github.com/mautops/approval-gin/internal/container"
	"github.com/mautops/approval-gin/internal/database"
	"github.com/mautops/approval-gin/internal/integration"
	"github.com/mautops/approval-gin/internal/repository"
	"github.com/mautops/approval-gin/internal/service"
	"github.com/mautops/approval-gin/internal/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestAPIServerWithWebSocket 创建带 WebSocket 的测试 API 服务器
func setupTestAPIServerWithWebSocket(t *testing.T) (*gin.Engine, *websocket.Hub, *auth.KeycloakTokenValidator) {
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

	// 创建控制器
	templateController := api.NewTemplateController(templateSvc)
	taskController := api.NewTaskController(taskSvc)
	_ = api.NewQueryController(querySvc)

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

	// WebSocket 路由
	router.GET("/ws/tasks/:id", websocket.WebSocketHandler(hub, keycloakValidator))

	// API v1 路由组
	v1 := router.Group("/api/v1")
	{
		templates := v1.Group("/templates")
		{
			templates.POST("", templateController.Create)
		}

		tasks := v1.Group("/tasks")
		{
			tasks.POST("", taskController.Create)
		}
	}

	_ = &container.Container{}

	return router, hub, keycloakValidator
}

// TestAPIIntegration_WebSocket_Connection 测试 WebSocket 连接
func TestAPIIntegration_WebSocket_Connection(t *testing.T) {
	router, hub, _ := setupTestAPIServerWithWebSocket(t)

	// 创建测试服务器
	server := httptest.NewServer(router)
	defer server.Close()

	// 连接到 WebSocket（不带 token，应该失败）
	wsURL := "ws" + server.URL[4:] + "/ws/tasks/task-001"
	_, resp, err := gorillaWS.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		resp.Body.Close()
	}

	// 没有 token 时应该失败
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// 验证 Hub 中没有客户端
	assert.Equal(t, 0, hub.GetClientCount())
}

// TestAPIIntegration_WebSocket_InvalidToken 测试 WebSocket 无效 Token
func TestAPIIntegration_WebSocket_InvalidToken(t *testing.T) {
	router, hub, _ := setupTestAPIServerWithWebSocket(t)

	// 创建测试服务器
	server := httptest.NewServer(router)
	defer server.Close()

	// 连接到 WebSocket（带无效 token，应该失败）
	wsURL := "ws" + server.URL[4:] + "/ws/tasks/task-001?token=invalid-token"
	_, resp, err := gorillaWS.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		resp.Body.Close()
	}

	// 无效 token 时应该失败
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// 验证 Hub 中没有客户端
	assert.Equal(t, 0, hub.GetClientCount())
}

// TestAPIIntegration_WebSocket_MessagePush 测试 WebSocket 消息推送
func TestAPIIntegration_WebSocket_MessagePush(t *testing.T) {
	router, hub, _ := setupTestAPIServerWithWebSocket(t)

	// 注意: WebSocket 消息推送测试需要有效的 token 和真实的 WebSocket 连接
	// 由于 Keycloak 认证需要真实的 JWKS 端点，这里先跳过
	// 等实现 mock Keycloak 服务器后再完善测试
	_ = router
	_ = hub
	t.Skip("WebSocket message push test requires valid token, skipping for now")
}

// TestAPIIntegration_WebSocket_Broadcast 测试 WebSocket 广播
func TestAPIIntegration_WebSocket_Broadcast(t *testing.T) {
	router, hub, _ := setupTestAPIServerWithWebSocket(t)

	// 注意: WebSocket 广播测试需要多个有效的 WebSocket 连接
	// 由于 Keycloak 认证需要真实的 JWKS 端点，这里先跳过
	// 等实现 mock Keycloak 服务器后再完善测试
	_ = router
	_ = hub
	t.Skip("WebSocket broadcast test requires valid tokens, skipping for now")
}

// TestAPIIntegration_WebSocket_UserSpecificBroadcast 测试 WebSocket 用户特定广播
func TestAPIIntegration_WebSocket_UserSpecificBroadcast(t *testing.T) {
	router, hub, _ := setupTestAPIServerWithWebSocket(t)

	// 测试 BroadcastToUser 方法
	testMessage := []byte(`{"type":"test","data":"test message"}`)
	hub.BroadcastToUser("user-001", testMessage)

	// 验证方法可调用（由于没有客户端连接，消息不会被发送）
	// 这里只验证方法存在且可调用
	_ = router
	assert.Equal(t, 0, hub.GetClientCount())
}

// TestAPIIntegration_WebSocket_ClientCount 测试 WebSocket 客户端计数
func TestAPIIntegration_WebSocket_ClientCount(t *testing.T) {
	_, hub, _ := setupTestAPIServerWithWebSocket(t)

	// 验证初始客户端数量为 0
	assert.Equal(t, 0, hub.GetClientCount())

	// 验证 HasClient 方法
	assert.False(t, hub.HasClient("non-existent-client"))
}

