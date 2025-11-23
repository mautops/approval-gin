package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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

// setupTestAPIServerWithAuth 创建带认证的测试 API 服务器
func setupTestAPIServerWithAuth(t *testing.T) (*gin.Engine, *container.Container, *auth.KeycloakTokenValidator, *auth.OpenFGAClient) {
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

	// API v1 路由组（使用认证中间件）
	v1 := router.Group("/api/v1")
	{
		// 使用 Keycloak 认证中间件
		v1.Use(auth.KeycloakAuthMiddleware(keycloakValidator))

		templates := v1.Group("/templates")
		{
			templates.POST("", templateController.Create)
			templates.GET("", templateController.List)
			templates.GET("/:id", templateController.Get)
			templates.PUT("/:id", templateController.Update)
			templates.DELETE("/:id", templateController.Delete)
		}

		tasks := v1.Group("/tasks")
		{
			tasks.POST("", taskController.Create)
			tasks.GET("", queryController.ListTasks)
			tasks.GET("/:id", taskController.Get)
		}
	}

	ctr := &container.Container{}

	return router, ctr, keycloakValidator, fgaClient
}

// createMockKeycloakValidator 创建 mock Keycloak 验证器（用于测试）
// 注意: 这是一个简化的实现，仅用于测试
func createMockKeycloakValidator(issuer string) *auth.KeycloakTokenValidator {
	return auth.NewKeycloakTokenValidator(issuer)
}

// generateTestToken 生成测试用的 JWT Token
// 注意: 由于 Keycloak 使用 RSA 签名，这里生成的 token 无法通过真实的验证
// 仅用于测试无效 token 的场景
func generateTestToken(t *testing.T, issuer string, userID string, username string) string {
	claims := auth.KeycloakClaims{
		Sub:               userID,
		PreferredUsername: username,
		Email:             username + "@example.com",
		Name:              username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// 使用测试私钥签名（在实际测试中，应该使用真实的 Keycloak 签名）
	// 这里简化处理，使用 HS256 算法
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("test-secret-key"))
	require.NoError(t, err)

	return tokenString
}

// TestAPIIntegration_Auth_Keycloak_ValidToken 测试 Keycloak 认证 - 有效 Token
func TestAPIIntegration_Auth_Keycloak_ValidToken(t *testing.T) {
	_, _, _, _ = setupTestAPIServerWithAuth(t)

	// 注意: 由于 Keycloak 认证需要真实的 JWKS 端点，这里先跳过
	// 等实现 mock Keycloak 服务器后再完善测试
	t.Skip("Keycloak authentication requires real JWKS endpoint, skipping for now")
}

// TestAPIIntegration_Auth_Keycloak_InvalidToken 测试 Keycloak 认证 - 无效 Token
func TestAPIIntegration_Auth_Keycloak_InvalidToken(t *testing.T) {
	router, _, _, _ := setupTestAPIServerWithAuth(t)

	// 测试无效 Token
	req := httptest.NewRequest("GET", "/api/v1/templates", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// 应该返回 401 Unauthorized
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, float64(401), response["code"])
}

// TestAPIIntegration_Auth_Keycloak_MissingToken 测试 Keycloak 认证 - 缺少 Token
func TestAPIIntegration_Auth_Keycloak_MissingToken(t *testing.T) {
	router, _, _, _ := setupTestAPIServerWithAuth(t)

	// 测试缺少 Token
	req := httptest.NewRequest("GET", "/api/v1/templates", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// 应该返回 401 Unauthorized
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, float64(401), response["code"])
	assert.Contains(t, response["message"].(string), "missing authorization header")
}

// TestAPIIntegration_Auth_OpenFGA_PermissionCheck 测试 OpenFGA 权限检查
func TestAPIIntegration_Auth_OpenFGA_PermissionCheck(t *testing.T) {
	_, _, _, _ = setupTestAPIServerWithAuth(t)

	// 注意: 由于 OpenFGA 权限检查需要真实的 OpenFGA 服务器，这里先跳过
	// 等实现 mock OpenFGA 服务器后再完善测试
	t.Skip("OpenFGA permission check requires real OpenFGA server, skipping for now")
}

// TestAPIIntegration_Auth_UserContext 测试用户上下文信息
func TestAPIIntegration_Auth_UserContext(t *testing.T) {
	_, _, _, _ = setupTestAPIServerWithAuth(t)

	// 注意: 需要有效的 Token 才能测试用户上下文
	// 等实现 mock Keycloak 服务器后再完善测试
	t.Skip("User context test requires valid token, skipping for now")
}

