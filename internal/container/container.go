package container

import (
	"fmt"
	"time"

	"github.com/mautops/approval-gin/internal/auth"
	"github.com/mautops/approval-gin/internal/config"
	"github.com/mautops/approval-gin/internal/database"
	"github.com/mautops/approval-gin/internal/integration"
	"github.com/mautops/approval-gin/internal/service"
	"github.com/mautops/approval-kit/pkg/event"
	"github.com/mautops/approval-kit/pkg/task"
	"github.com/mautops/approval-kit/pkg/template"
	"gorm.io/gorm"
)

// Container 依赖注入容器
// 管理所有应用依赖,包括数据库、服务、客户端等
type Container struct {
	db                *gorm.DB
	templateMgr       template.TemplateManager
	taskMgr           task.TaskManager
	fgaClient         *auth.OpenFGAClient
	eventHandler      event.EventHandler
	keycloakValidator *auth.KeycloakTokenValidator
	backupService     *service.BackupService
}

// NewContainer 创建依赖注入容器
// 根据配置初始化所有依赖组件
func NewContainer(cfg *config.Config) (*Container, error) {
	// 1. 初始化数据库（带重试机制）
	// 默认重试 3 次，初始间隔 1 秒，指数退避
	db, err := database.ConnectWithRetry(cfg.Database, 3, time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// 执行数据库迁移
	if err := database.Migrate(db); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// 2. 初始化 TemplateManager
	templateMgr := integration.NewTemplateManager(db)

	// 3. 初始化 EventHandler
	// 默认使用 5 个 worker
	eventWorkers := 5
	eventHandler := integration.NewEventHandler(db, eventWorkers)

	// 4. 初始化 TaskManager
	// 注意: 状态机已集成(任务 3.5),传递 nil 时会自动创建默认状态机实例
	// 节点执行引擎尚未实现(任务 3.6),相关方法会返回 "not implemented" 错误
	taskMgr := integration.NewTaskManager(db, templateMgr, nil, eventHandler)

	// 5. 初始化 OpenFGA 客户端（带重试机制）
	// 默认重试 3 次，初始间隔 1 秒，指数退避
	fgaClient, err := auth.NewOpenFGAClientWithRetry(cfg.OpenFGA.APIURL, cfg.OpenFGA.StoreID, cfg.OpenFGA.ModelID, 3, time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OpenFGA client: %w", err)
	}

	// 6. 初始化 Keycloak Token 验证器
	keycloakValidator := auth.NewKeycloakTokenValidator(cfg.Keycloak.Issuer)

	// 7. 初始化备份服务
	// 默认备份目录为 ./backups，可以通过环境变量配置
	backupDir := "./backups"
	backupService := service.NewBackupService(db, backupDir)

	return &Container{
		db:                db,
		templateMgr:       templateMgr,
		taskMgr:           taskMgr,
		fgaClient:         fgaClient,
		eventHandler:      eventHandler,
		keycloakValidator: keycloakValidator,
		backupService:     backupService,
	}, nil
}

// DB 获取数据库连接
func (c *Container) DB() *gorm.DB {
	return c.db
}

// TemplateManager 获取模板管理器
func (c *Container) TemplateManager() template.TemplateManager {
	return c.templateMgr
}

// TaskManager 获取任务管理器
func (c *Container) TaskManager() task.TaskManager {
	return c.taskMgr
}

// OpenFGAClient 获取 OpenFGA 客户端
func (c *Container) OpenFGAClient() *auth.OpenFGAClient {
	return c.fgaClient
}

// EventHandler 获取事件处理器
func (c *Container) EventHandler() event.EventHandler {
	return c.eventHandler
}

// KeycloakValidator 获取 Keycloak Token 验证器
func (c *Container) KeycloakValidator() *auth.KeycloakTokenValidator {
	return c.keycloakValidator
}

// BackupService 获取备份服务
func (c *Container) BackupService() *service.BackupService {
	return c.backupService
}

// Close 关闭容器,清理资源
func (c *Container) Close() error {
	if c.db != nil {
		sqlDB, err := c.db.DB()
		if err == nil {
			sqlDB.Close()
		}
	}

	// 关闭事件处理器
	// 注意: EventHandler 接口可能没有 Close 方法
	// 如果需要清理资源,可以在 EventHandler 接口中添加 Close 方法

	return nil
}

