/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/mautops/approval-gin/internal/config"
	"github.com/mautops/approval-gin/internal/container"
	"github.com/mautops/approval-gin/internal/repository"
	"github.com/mautops/approval-gin/internal/service"
	"github.com/mautops/approval-gin/internal/websocket"
	"github.com/spf13/cobra"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the API server",
	Long: `Start the Approval Gin API server.
The server will listen on the configured host and port,
and provide REST API interfaces for approval workflow management.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. 加载配置
		configPath, _ := cmd.Flags().GetString("config")
		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// 2. 初始化容器
		ctr, err := container.NewContainer(cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize container: %w", err)
		}
		defer ctr.Close()

		// 3. 初始化服务
		auditLogSvc := service.NewAuditLogService(repository.NewAuditLogRepository(ctr.DB()))
		templateSvc := service.NewTemplateService(ctr.TemplateManager(), ctr.DB(), auditLogSvc, ctr.OpenFGAClient())
		taskSvc := service.NewTaskService(ctr.TaskManager(), ctr.DB(), auditLogSvc, ctr.OpenFGAClient())
		querySvc := service.NewQueryService(ctr.DB(), ctr.TaskManager())

		// 4. 初始化控制器
		templateController := api.NewTemplateController(templateSvc)
		taskController := api.NewTaskController(taskSvc)
		queryController := api.NewQueryController(querySvc)
		backupController := api.NewBackupController(ctr.BackupService())

		// 5. 初始化 WebSocket Hub
		hub := websocket.NewHub()
		go hub.Run()

		// 6. 设置路由
		router := setupRoutesWithControllers(hub, ctr, templateController, taskController, queryController, backupController)

		// 7. 启动服务器
		addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		srv := &http.Server{
			Addr:    addr,
			Handler: router,
		}

		// 启动服务器（在 goroutine 中）
		go func() {
			log.Printf("Server starting on %s", addr)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Failed to start server: %v", err)
			}
		}()

		// 等待中断信号
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		log.Println("Shutting down server...")

		// 优雅关闭
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("Server forced to shutdown: %v", err)
		}

		log.Println("Server exited")
		return nil
	},
}

// setupRoutesWithControllers 设置路由并绑定控制器
func setupRoutesWithControllers(
	hub *websocket.Hub,
	ctr *container.Container,
	templateController *api.TemplateController,
	taskController *api.TaskController,
	queryController *api.QueryController,
	backupController *api.BackupController,
) *gin.Engine {
	router := api.SetupRoutes(hub, ctr.KeycloakValidator(), ctr.DB(), ctr.OpenFGAClient())

	// API v1 路由组
	v1 := router.Group("/api/v1")
	{
		// 模板管理路由
		templates := v1.Group("/templates")
		{
			templates.POST("", templateController.Create)
			templates.GET("", templateController.List)
			templates.GET("/:id", templateController.Get)
			templates.PUT("/:id", templateController.Update)
			templates.DELETE("/:id", templateController.Delete)
			templates.GET("/:id/versions", templateController.ListVersions)
		}

		// 任务管理路由
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
			// 批量操作路由
			tasks.POST("/batch/approve", taskController.BatchApprove)
			tasks.POST("/batch/transfer", taskController.BatchTransfer)
			tasks.GET("/:id/records", queryController.GetRecords)
			tasks.GET("/:id/history", queryController.GetHistory)
		}

		// 备份管理路由
		backups := v1.Group("/backups")
		{
			backups.POST("", backupController.CreateBackup)
			backups.GET("", backupController.ListBackups)
			backups.POST("/:filename/restore", backupController.RestoreBackup)
			backups.DELETE("/:filename", backupController.DeleteBackup)
		}
	}

	return router
}

func init() {
	rootCmd.AddCommand(serverCmd)

	// 服务器配置标志
	serverCmd.Flags().String("config", "", "Config file path (default: config.yaml)")
	serverCmd.Flags().String("host", "0.0.0.0", "Server host")
	serverCmd.Flags().Int("port", 8080, "Server port")
}

// LoadConfig 加载配置
func LoadConfig(configPath string) (*config.Config, error) {
	return config.Load(configPath)
}
