package service

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// BackupScheduler 备份调度器
type BackupScheduler struct {
	backupService *BackupService
	config        *BackupScheduleConfig
	stopChan      chan struct{}
}

// BackupScheduleConfig 备份计划配置
type BackupScheduleConfig struct {
	// 全量备份配置
	FullBackupEnabled       bool   // 是否启用全量备份
	FullBackupSchedule      string // 全量备份计划 (cron 格式，如 "0 0 * * *" 表示每天凌晨)
	FullBackupRetentionDays int    // 全量备份保留天数

	// 增量备份配置
	IncrementalBackupEnabled       bool          // 是否启用增量备份
	IncrementalBackupInterval      time.Duration // 增量备份间隔
	IncrementalBackupRetentionDays int           // 增量备份保留天数

	// 备份验证
	VerifyBackup bool // 是否验证备份文件完整性
}

// NewBackupScheduler 创建备份调度器
func NewBackupScheduler(backupService *BackupService, config *BackupScheduleConfig) *BackupScheduler {
	if config == nil {
		config = &BackupScheduleConfig{
			FullBackupEnabled:              true,
			FullBackupSchedule:             "0 0 * * *", // 每天凌晨
			FullBackupRetentionDays:        30,
			IncrementalBackupEnabled:       false,
			IncrementalBackupInterval:      time.Hour,
			IncrementalBackupRetentionDays: 7,
			VerifyBackup:                   true,
		}
	}

	return &BackupScheduler{
		backupService: backupService,
		config:        config,
		stopChan:      make(chan struct{}),
	}
}

// Start 启动备份调度器
func (s *BackupScheduler) Start(ctx context.Context) error {
	// 启动全量备份调度
	if s.config.FullBackupEnabled {
		go s.scheduleFullBackup(ctx)
	}

	// 启动增量备份调度
	if s.config.IncrementalBackupEnabled {
		go s.scheduleIncrementalBackup(ctx)
	}

	// 启动备份清理调度
	go s.scheduleBackupCleanup(ctx)

	return nil
}

// Stop 停止备份调度器
func (s *BackupScheduler) Stop() {
	close(s.stopChan)
}

// Config 获取备份配置
func (s *BackupScheduler) Config() *BackupScheduleConfig {
	return s.config
}

// scheduleFullBackup 调度全量备份
func (s *BackupScheduler) scheduleFullBackup(ctx context.Context) {
	// 简化实现：使用 time.Ticker 而不是 cron
	// 实际生产环境应该使用 cron 库
	ticker := time.NewTicker(24 * time.Hour) // 每天执行一次
	defer ticker.Stop()

	// 立即执行一次
	s.performFullBackup(ctx)

	for {
		select {
		case <-ticker.C:
			s.performFullBackup(ctx)
		case <-s.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// scheduleIncrementalBackup 调度增量备份
func (s *BackupScheduler) scheduleIncrementalBackup(ctx context.Context) {
	ticker := time.NewTicker(s.config.IncrementalBackupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.performIncrementalBackup(ctx)
		case <-s.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// scheduleBackupCleanup 调度备份清理
func (s *BackupScheduler) scheduleBackupCleanup(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour) // 每天执行一次
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.CleanupOldBackups(ctx)
		case <-s.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// performFullBackup 执行全量备份
func (s *BackupScheduler) performFullBackup(ctx context.Context) {
	backupPath, err := s.backupService.CreateBackup(ctx)
	if err != nil {
		fmt.Printf("Failed to create full backup: %v\n", err)
		return
	}

	fmt.Printf("Full backup created: %s\n", backupPath)

	// 验证备份（如果启用）
	if s.config.VerifyBackup {
		// 这里可以添加备份验证逻辑
		// 例如：检查文件大小、校验和等
	}
}

// performIncrementalBackup 执行增量备份
func (s *BackupScheduler) performIncrementalBackup(ctx context.Context) {
	// 增量备份实现：只备份自上次备份以来的变更
	// 这里简化实现，实际应该跟踪上次备份时间
	backupPath, err := s.backupService.CreateBackup(ctx)
	if err != nil {
		fmt.Printf("Failed to create incremental backup: %v\n", err)
		return
	}

	fmt.Printf("Incremental backup created: %s\n", backupPath)
}

// CleanupOldBackups 清理旧备份（公开方法，用于测试）
func (s *BackupScheduler) CleanupOldBackups(ctx context.Context) {
	backups, err := s.backupService.ListBackups(ctx)
	if err != nil {
		fmt.Printf("Failed to list backups: %v\n", err)
		return
	}

	now := time.Now()
	fullRetention := time.Duration(s.config.FullBackupRetentionDays) * 24 * time.Hour
	incrementalRetention := time.Duration(s.config.IncrementalBackupRetentionDays) * 24 * time.Hour

	for _, backup := range backups {
		age := now.Sub(backup.CreatedAt)

		// 判断是全量备份还是增量备份
		// 如果增量备份未启用，所有备份都当作全量备份处理
		// 否则通过文件名判断（包含 "full" 或 "incremental"）
		var retention time.Duration
		if !s.config.IncrementalBackupEnabled {
			// 增量备份未启用，所有备份都当作全量备份
			retention = fullRetention
		} else {
			// 增量备份已启用，通过文件名判断
			isFullBackup := strings.Contains(backup.Filename, "full")
			isIncremental := strings.Contains(backup.Filename, "incremental")
			
			if isFullBackup {
				retention = fullRetention
			} else if isIncremental {
				retention = incrementalRetention
			} else {
				// 无法判断类型，默认使用全量备份保留期
				retention = fullRetention
			}
		}

		if age > retention {
			if err := s.backupService.DeleteBackup(ctx, backup.Filename); err != nil {
				fmt.Printf("Failed to delete old backup %s: %v\n", backup.Filename, err)
			} else {
				fmt.Printf("Deleted old backup: %s\n", backup.Filename)
			}
		}
	}
}
