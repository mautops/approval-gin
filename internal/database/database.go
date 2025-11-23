package database

import (
	"context"
	"fmt"
	"time"

	"github.com/mautops/approval-gin/internal/config"
	"github.com/mautops/approval-gin/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// PoolConfig 连接池配置
type PoolConfig struct {
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime int // 秒
	ConnMaxIdleTime int // 秒
}

// BuildDSN 构建 PostgreSQL DSN
func BuildDSN(cfg config.DatabaseConfig) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)
}

// GetPoolConfig 获取连接池配置
func GetPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: 3600, // 1 小时
		ConnMaxIdleTime: 600,  // 10 分钟
	}
}

// GetProductionPoolConfig 获取生产环境连接池配置
func GetProductionPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxIdleConns:    20,   // 生产环境增加空闲连接数
		MaxOpenConns:    200,  // 生产环境增加最大连接数
		ConnMaxLifetime: 3600, // 1 小时
		ConnMaxIdleTime: 300,  // 5 分钟（生产环境缩短空闲时间）
	}
}

// Connect 连接数据库
func Connect(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := BuildDSN(cfg)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}
	
	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}
	
	// 从配置中读取连接池参数，如果没有配置则使用默认值
	var poolConfig *PoolConfig
	if cfg.MaxIdleConns > 0 || cfg.MaxOpenConns > 0 {
		// 使用配置中的值
		poolConfig = &PoolConfig{
			MaxIdleConns:    cfg.MaxIdleConns,
			MaxOpenConns:    cfg.MaxOpenConns,
			ConnMaxLifetime: cfg.ConnMaxLifetime,
			ConnMaxIdleTime: cfg.ConnMaxIdleTime,
		}
		// 如果某些值未设置，使用默认值
		if poolConfig.MaxIdleConns == 0 {
			poolConfig.MaxIdleConns = 10
		}
		if poolConfig.MaxOpenConns == 0 {
			poolConfig.MaxOpenConns = 100
		}
		if poolConfig.ConnMaxLifetime == 0 {
			poolConfig.ConnMaxLifetime = 3600
		}
		if poolConfig.ConnMaxIdleTime == 0 {
			poolConfig.ConnMaxIdleTime = 600
		}
	} else {
		// 使用默认配置
		poolConfig = GetPoolConfig()
	}
	
	sqlDB.SetMaxIdleConns(poolConfig.MaxIdleConns)
	sqlDB.SetMaxOpenConns(poolConfig.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(poolConfig.ConnMaxLifetime) * time.Second)
	sqlDB.SetConnMaxIdleTime(time.Duration(poolConfig.ConnMaxIdleTime) * time.Second)
	
	return db, nil
}

// ConnectProduction 连接数据库（生产环境配置）
func ConnectProduction(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := BuildDSN(cfg)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}
	
	// 配置连接池（生产环境）
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}
	
	// 从配置中读取连接池参数，如果没有配置则使用生产环境默认值
	var poolConfig *PoolConfig
	if cfg.MaxIdleConns > 0 || cfg.MaxOpenConns > 0 {
		// 使用配置中的值
		poolConfig = &PoolConfig{
			MaxIdleConns:    cfg.MaxIdleConns,
			MaxOpenConns:    cfg.MaxOpenConns,
			ConnMaxLifetime: cfg.ConnMaxLifetime,
			ConnMaxIdleTime: cfg.ConnMaxIdleTime,
		}
		// 如果某些值未设置，使用生产环境默认值
		if poolConfig.MaxIdleConns == 0 {
			poolConfig.MaxIdleConns = 20
		}
		if poolConfig.MaxOpenConns == 0 {
			poolConfig.MaxOpenConns = 200
		}
		if poolConfig.ConnMaxLifetime == 0 {
			poolConfig.ConnMaxLifetime = 3600
		}
		if poolConfig.ConnMaxIdleTime == 0 {
			poolConfig.ConnMaxIdleTime = 300
		}
	} else {
		// 使用生产环境默认配置
		poolConfig = GetProductionPoolConfig()
	}
	
	sqlDB.SetMaxIdleConns(poolConfig.MaxIdleConns)
	sqlDB.SetMaxOpenConns(poolConfig.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(poolConfig.ConnMaxLifetime) * time.Second)
	sqlDB.SetConnMaxIdleTime(time.Duration(poolConfig.ConnMaxIdleTime) * time.Second)
	
	return db, nil
}

// Migrate 执行数据库迁移
func Migrate(db *gorm.DB) error {
	// 检测数据库类型
	dialector := db.Dialector.Name()
	
	// SQLite 不支持 jsonb，需要手动创建表
	// GORM SQLite dialector 的名称可能是 "sqlite" 或 "sqlite3"
	if dialector == "sqlite" || dialector == "sqlite3" {
		// 手动创建 SQLite 表（使用 TEXT 替代 jsonb）
		if err := createSQLiteTables(db); err != nil {
			return fmt.Errorf("failed to create SQLite tables: %w", err)
		}
	} else {
		// PostgreSQL 等其他数据库使用 AutoMigrate
		if err := db.AutoMigrate(
			&model.TemplateModel{},
			&model.TaskModel{},
			&model.ApprovalRecordModel{},
			&model.StateHistoryModel{},
			&model.EventModel{},
			&model.AuditLogModel{},
		); err != nil {
			return fmt.Errorf("failed to auto migrate: %w", err)
		}
	}
	
	// 创建索引
	if err := CreateIndexes(db); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}
	
	return nil
}

// createSQLiteTables 为 SQLite 手动创建表（使用 TEXT 替代 jsonb）
func createSQLiteTables(db *gorm.DB) error {
	// 创建 templates 表 (使用组合主键 id, version)
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS templates (
			id VARCHAR(64) NOT NULL,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			version INTEGER NOT NULL DEFAULT 1,
			data TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			created_by VARCHAR(64),
			updated_by VARCHAR(64),
			PRIMARY KEY (id, version)
		)
	`).Error; err != nil {
		return fmt.Errorf("failed to create templates table: %w", err)
	}

	// 创建 tasks 表
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id VARCHAR(64) PRIMARY KEY,
			template_id VARCHAR(64) NOT NULL,
			template_version INTEGER NOT NULL,
			business_id VARCHAR(64),
			state VARCHAR(32) NOT NULL,
			current_node VARCHAR(64),
			data TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			submitted_at DATETIME,
			created_by VARCHAR(64)
		)
	`).Error; err != nil {
		return fmt.Errorf("failed to create tasks table: %w", err)
	}

	// 创建 approval_records 表
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS approval_records (
			id VARCHAR(64) PRIMARY KEY,
			task_id VARCHAR(64) NOT NULL,
			node_id VARCHAR(64) NOT NULL,
			approver VARCHAR(64) NOT NULL,
			result VARCHAR(32) NOT NULL,
			comment TEXT,
			attachments TEXT,
			created_at DATETIME NOT NULL
		)
	`).Error; err != nil {
		return fmt.Errorf("failed to create approval_records table: %w", err)
	}

	// 创建 state_history 表
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS state_history (
			id VARCHAR(64) PRIMARY KEY,
			task_id VARCHAR(64) NOT NULL,
			from_state VARCHAR(32),
			to_state VARCHAR(32) NOT NULL,
			reason TEXT,
			operator VARCHAR(64) NOT NULL,
			created_at DATETIME NOT NULL
		)
	`).Error; err != nil {
		return fmt.Errorf("failed to create state_history table: %w", err)
	}

	// 创建 events 表
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			id VARCHAR(64) PRIMARY KEY,
			task_id VARCHAR(64) NOT NULL,
			type VARCHAR(32) NOT NULL,
			data TEXT NOT NULL,
			status VARCHAR(32) NOT NULL DEFAULT 'pending',
			retry_count INTEGER DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)
	`).Error; err != nil {
		return fmt.Errorf("failed to create events table: %w", err)
	}

	// 创建 audit_logs 表
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS audit_logs (
			id VARCHAR(64) PRIMARY KEY,
			user_id VARCHAR(64) NOT NULL,
			action VARCHAR(64) NOT NULL,
			resource_type VARCHAR(32) NOT NULL,
			resource_id VARCHAR(64) NOT NULL,
			request_id VARCHAR(64),
			ip VARCHAR(45),
			user_agent TEXT,
			details TEXT,
			created_at DATETIME NOT NULL
		)
	`).Error; err != nil {
		return fmt.Errorf("failed to create audit_logs table: %w", err)
	}

	return nil
}

// CreateIndexes 创建数据库索引
func CreateIndexes(db *gorm.DB) error {
	// 检测数据库类型
	dialector := db.Dialector.Name()
	
	// templates 表索引
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_templates_name ON templates(name)").Error; err != nil {
		return fmt.Errorf("failed to create idx_templates_name: %w", err)
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_templates_created_at ON templates(created_at)").Error; err != nil {
		return fmt.Errorf("failed to create idx_templates_created_at: %w", err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_templates_id_version ON templates(id, version)").Error; err != nil {
		return fmt.Errorf("failed to create idx_templates_id_version: %w", err)
	}
	
	// tasks 表索引
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_tasks_state_business ON tasks(state, business_id)").Error; err != nil {
		return fmt.Errorf("failed to create idx_tasks_state_business: %w", err)
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_tasks_template_id ON tasks(template_id)").Error; err != nil {
		return fmt.Errorf("failed to create idx_tasks_template_id: %w", err)
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_tasks_created_by ON tasks(created_by)").Error; err != nil {
		return fmt.Errorf("failed to create idx_tasks_created_by: %w", err)
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_tasks_updated_at ON tasks(updated_at)").Error; err != nil {
		return fmt.Errorf("failed to create idx_tasks_updated_at: %w", err)
	}
	
	// approval_records 表索引
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_records_task_id ON approval_records(task_id)").Error; err != nil {
		return fmt.Errorf("failed to create idx_records_task_id: %w", err)
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_records_approver ON approval_records(approver)").Error; err != nil {
		return fmt.Errorf("failed to create idx_records_approver: %w", err)
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_records_created_at ON approval_records(created_at)").Error; err != nil {
		return fmt.Errorf("failed to create idx_records_created_at: %w", err)
	}
	
	// state_history 表索引
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_history_task_id ON state_history(task_id)").Error; err != nil {
		return fmt.Errorf("failed to create idx_history_task_id: %w", err)
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_history_created_at ON state_history(created_at)").Error; err != nil {
		return fmt.Errorf("failed to create idx_history_created_at: %w", err)
	}
	
	// events 表索引
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_events_status ON events(status)").Error; err != nil {
		return fmt.Errorf("failed to create idx_events_status: %w", err)
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_events_task_id ON events(task_id)").Error; err != nil {
		return fmt.Errorf("failed to create idx_events_task_id: %w", err)
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at)").Error; err != nil {
		return fmt.Errorf("failed to create idx_events_created_at: %w", err)
	}
	
	// audit_logs 表索引
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_audit_resource ON audit_logs(resource_type, resource_id)").Error; err != nil {
		return fmt.Errorf("failed to create idx_audit_resource: %w", err)
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_audit_user_id ON audit_logs(user_id)").Error; err != nil {
		return fmt.Errorf("failed to create idx_audit_user_id: %w", err)
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_audit_created_at ON audit_logs(created_at)").Error; err != nil {
		return fmt.Errorf("failed to create idx_audit_created_at: %w", err)
	}
	
	// PostgreSQL 特定的 GIN 索引
	if dialector == "postgres" {
		// JSONB 字段的 GIN 索引
		if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_templates_data_gin ON templates USING GIN (data)").Error; err != nil {
			return fmt.Errorf("failed to create idx_templates_data_gin: %w", err)
		}
		if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_tasks_data_gin ON tasks USING GIN (data)").Error; err != nil {
			return fmt.Errorf("failed to create idx_tasks_data_gin: %w", err)
		}
	}
	
	return nil
}

// ConnectWithRetry 带重试的数据库连接
func ConnectWithRetry(cfg config.DatabaseConfig, maxRetries int, retryInterval time.Duration) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	for i := 0; i < maxRetries; i++ {
		db, err = Connect(cfg)
		if err == nil {
			return db, nil
		}

		// 如果不是最后一次重试，等待后重试
		if i < maxRetries-1 {
			time.Sleep(retryInterval)
			retryInterval *= 2 // 指数退避
		}
	}

	return nil, fmt.Errorf("failed to connect database after %d retries: %w", maxRetries, err)
}

// CheckHealth 检查数据库连接健康状态
func CheckHealth(db *gorm.DB) bool {
	if db == nil {
		return false
	}

	sqlDB, err := db.DB()
	if err != nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return false
	}

	return true
}

// Reconnect 重新连接数据库
func Reconnect(cfg config.DatabaseConfig, oldDB *gorm.DB) (*gorm.DB, error) {
	// 关闭旧连接
	if oldDB != nil {
		if sqlDB, err := oldDB.DB(); err == nil {
			sqlDB.Close()
		}
	}

	// 重新连接
	return Connect(cfg)
}

