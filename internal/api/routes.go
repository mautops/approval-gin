package api

import (
	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/auth"
	"github.com/mautops/approval-gin/internal/config"
	"github.com/mautops/approval-gin/internal/websocket"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "github.com/mautops/approval-gin/docs" // 导入生成的 docs 包
	"gorm.io/gorm"
)

// SetupRoutes 配置路由
func SetupRoutes(hub *websocket.Hub, validator *auth.KeycloakTokenValidator, db *gorm.DB, fgaClient *auth.OpenFGAClient) *gin.Engine {
	return SetupRoutesWithConfig(hub, validator, db, fgaClient, "", 0, nil)
}

// SetupRoutesWithConfig 配置路由(带配置参数)
func SetupRoutesWithConfig(hub *websocket.Hub, validator *auth.KeycloakTokenValidator, db *gorm.DB, fgaClient *auth.OpenFGAClient, host string, port int, corsConfig *config.CORSConfig) *gin.Engine {
	router := gin.Default()

	// CORS 中间件(必须在其他中间件之前)
	if corsConfig != nil && len(corsConfig.AllowedOrigins) > 0 {
		router.Use(CORSMiddleware(corsConfig.AllowedOrigins))
	} else {
		// 默认允许所有源(开发环境)
		router.Use(CORSMiddleware([]string{"*"}))
	}

	// 中间件
	router.Use(RequestIDMiddleware())
	router.Use(RequestLogMiddleware())

	// 健康检查
	healthController := NewHealthController(db, fgaClient)
	router.GET("/health", healthController.Check)

	// Prometheus 指标端点
	router.GET("/metrics", MetricsHandler)

	// WebSocket 路由
	if hub != nil && validator != nil {
		router.GET("/ws/tasks/:id", websocket.WebSocketHandler(hub, validator))
	}

	// SSE 路由
	if validator != nil {
		router.GET("/sse/tasks/:id", SSEHandler(validator))
	}

	// Swagger UI 路由
	// 使用相对路径,自动使用当前请求的 origin,避免跨域问题
	// 这是最佳实践,因为 Swagger UI 和 API 在同一服务器上
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 注意: 业务路由(模板、任务等)在 setupRoutesWithControllers 中注册
	// 这里只设置基础路由和中间件

	return router
}

