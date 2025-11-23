package api

import (
	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/auth"
	"github.com/mautops/approval-gin/internal/websocket"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "github.com/mautops/approval-gin/docs" // 导入生成的 docs 包
	"gorm.io/gorm"
)

// SetupRoutes 配置路由
func SetupRoutes(hub *websocket.Hub, validator *auth.KeycloakTokenValidator, db *gorm.DB, fgaClient *auth.OpenFGAClient) *gin.Engine {
	router := gin.Default()

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
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
		ginSwagger.URL("http://localhost:8080/swagger/doc.json"), // Swagger JSON URL
	))

	// API v1 路由组
	v1 := router.Group("/api/v1")
	{
		// 模板管理路由
		templates := v1.Group("/templates")
		{
			templates.POST("", nil)                    // Create
			templates.GET("", nil)                     // List
			templates.GET("/:id", nil)                 // Get
			templates.PUT("/:id", nil)                 // Update
			templates.DELETE("/:id", nil)              // Delete
			templates.GET("/:id/versions", nil)        // ListVersions
		}

		// 任务管理路由
		tasks := v1.Group("/tasks")
		{
			tasks.POST("", nil)                        // Create
			tasks.GET("", nil)                         // List
			tasks.GET("/:id", nil)                     // Get
			tasks.POST("/:id/submit", nil)             // Submit
			tasks.POST("/:id/approve", nil)            // Approve
			tasks.POST("/:id/reject", nil)             // Reject
			tasks.POST("/:id/cancel", nil)             // Cancel
			tasks.POST("/:id/withdraw", nil)            // Withdraw
			tasks.GET("/:id/records", nil)             // GetRecords
			tasks.GET("/:id/history", nil)             // GetHistory
		}
	}

	return router
}

