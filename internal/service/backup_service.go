package service

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
)

// BackupService 备份服务
type BackupService struct {
	db          *gorm.DB
	backupDir   string
	compression bool
}

// BackupInfo 备份信息
type BackupInfo struct {
	Filename    string    `json:"filename"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	DatabaseType string   `json:"database_type"`
}

// NewBackupService 创建备份服务
func NewBackupService(db *gorm.DB, backupDir string) *BackupService {
	// 确保备份目录存在
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		// 如果创建失败，使用临时目录
		backupDir = os.TempDir()
	}

	return &BackupService{
		db:          db,
		backupDir:   backupDir,
		compression: true, // 默认启用压缩
	}
}

// CreateBackup 创建备份
func (s *BackupService) CreateBackup(ctx context.Context) (string, error) {
	// 获取数据库类型
	dialector := s.db.Dialector.Name()
	
	// 生成备份文件名
	timestamp := time.Now().Format("20060102_150405")
	var ext string
	if s.compression {
		ext = ".tar.gz"
	} else {
		ext = ".sql"
	}
	filename := fmt.Sprintf("backup_%s_%s%s", dialector, timestamp, ext)
	backupPath := filepath.Join(s.backupDir, filename)

	// 根据数据库类型执行备份
	switch dialector {
	case "postgres":
		return s.createPostgreSQLBackup(ctx, backupPath)
	case "sqlite", "sqlite3":
		return s.createSQLiteBackup(ctx, backupPath)
	default:
		return "", fmt.Errorf("unsupported database type: %s", dialector)
	}
}

// createPostgreSQLBackup 创建 PostgreSQL 备份
func (s *BackupService) createPostgreSQLBackup(ctx context.Context, backupPath string) (string, error) {
	// 对于 PostgreSQL，我们使用 SQL 导出方式
	// 在实际生产环境中，应该使用 pg_dump 命令
	// 这里使用简化的 SQL 导出方式
	return s.exportSQLBackup(ctx, backupPath)
}

// createSQLiteBackup 创建 SQLite 备份
func (s *BackupService) createSQLiteBackup(ctx context.Context, backupPath string) (string, error) {
	// SQLite 可以直接复制数据库文件
	// 获取数据库文件路径
	sqlDB, err := s.db.DB()
	if err != nil {
		return "", fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// 尝试从 DSN 获取文件路径
	dsn := sqlDB.Driver()
	_ = dsn // SQLite 文件路径在 DSN 中

	// 使用 SQL 导出方式（简化实现）
	return s.exportSQLBackup(ctx, backupPath)
}

// exportSQLBackup 导出 SQL 备份
func (s *BackupService) exportSQLBackup(ctx context.Context, backupPath string) (string, error) {
	// 创建备份文件
	var writer io.Writer
	var file *os.File
	var err error

	if s.compression {
		// 创建压缩文件
		file, err = os.Create(backupPath)
		if err != nil {
			return "", fmt.Errorf("failed to create backup file: %w", err)
		}
		defer file.Close()

		gzWriter := gzip.NewWriter(file)
		defer gzWriter.Close()

		tarWriter := tar.NewWriter(gzWriter)
		defer tarWriter.Close()

		// 创建 SQL 文件条目
		sqlFilename := filepath.Base(backupPath[:len(backupPath)-len(filepath.Ext(backupPath))]) + ".sql"
		header := &tar.Header{
			Name: sqlFilename,
			Mode: 0644,
			Size: 0, // 将在写入时更新
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return "", fmt.Errorf("failed to write tar header: %w", err)
		}

		writer = tarWriter
	} else {
		file, err = os.Create(backupPath)
		if err != nil {
			return "", fmt.Errorf("failed to create backup file: %w", err)
		}
		defer file.Close()
		writer = file
	}

	// 导出所有表的数据
	if err := s.exportTables(ctx, writer); err != nil {
		return "", fmt.Errorf("failed to export tables: %w", err)
	}

	return backupPath, nil
}

// exportTables 导出所有表的数据
func (s *BackupService) exportTables(ctx context.Context, writer io.Writer) error {
	// 获取所有表名
	var tables []string
	dialector := s.db.Dialector.Name()
	
	if dialector == "sqlite" || dialector == "sqlite3" {
		if err := s.db.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&tables).Error; err != nil {
			return fmt.Errorf("failed to get table names: %w", err)
		}
	} else if dialector == "postgres" {
		if err := s.db.Raw("SELECT tablename FROM pg_tables WHERE schemaname='public'").Scan(&tables).Error; err != nil {
			return fmt.Errorf("failed to get table names: %w", err)
		}
	} else {
		return fmt.Errorf("unsupported database type: %s", dialector)
	}

	// 导出每个表
	for _, table := range tables {
		if err := s.exportTable(ctx, writer, table); err != nil {
			return fmt.Errorf("failed to export table %s: %w", table, err)
		}
	}

	return nil
}

// exportTable 导出单个表
func (s *BackupService) exportTable(ctx context.Context, writer io.Writer, tableName string) error {
	// 这里简化实现，实际应该使用数据库特定的导出命令
	// 对于生产环境，应该使用 pg_dump 或 sqlite3 .dump
	_, err := fmt.Fprintf(writer, "-- Table: %s\n", tableName)
	if err != nil {
		return err
	}

	// 导出表结构
	dialector := s.db.Dialector.Name()
	var createTableSQL string
	
	if dialector == "sqlite" || dialector == "sqlite3" {
		if err := s.db.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&createTableSQL).Error; err != nil {
			return fmt.Errorf("failed to get table schema: %w", err)
		}
	} else {
		// PostgreSQL 方式 - 简化实现
		createTableSQL = fmt.Sprintf("-- CREATE TABLE %s (...);", tableName)
	}

	_, err = fmt.Fprintf(writer, "%s\n\n", createTableSQL)
	return err
}

// RestoreBackup 恢复备份
func (s *BackupService) RestoreBackup(ctx context.Context, backupPath string) error {
	// 检查备份文件是否存在
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backupPath)
	}

	// 打开备份文件
	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	var reader io.Reader = file

	// 如果是压缩文件，解压
	if filepath.Ext(backupPath) == ".gz" {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()

		tarReader := tar.NewReader(gzReader)
		
		// 读取 tar 文件中的 SQL 文件
		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("failed to read tar: %w", err)
			}

			if filepath.Ext(header.Name) == ".sql" {
				reader = tarReader
				break
			}
		}
	}

	// 读取并执行 SQL
	sqlBytes, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read SQL: %w", err)
	}

	// 执行 SQL 恢复
	if err := s.db.Exec(string(sqlBytes)).Error; err != nil {
		return fmt.Errorf("failed to restore database: %w", err)
	}

	return nil
}

// ListBackups 列出所有备份
func (s *BackupService) ListBackups(ctx context.Context) ([]BackupInfo, error) {
	var backups []BackupInfo

	// 读取备份目录
	entries, err := os.ReadDir(s.backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// 检查是否是备份文件
		if !isBackupFile(entry.Name()) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		backupInfo := BackupInfo{
			Filename:     entry.Name(),
			Path:         filepath.Join(s.backupDir, entry.Name()),
			Size:         info.Size(),
			CreatedAt:    info.ModTime(),
			DatabaseType: detectDatabaseType(entry.Name()),
		}

		backups = append(backups, backupInfo)
	}

	return backups, nil
}

// BackupDir 获取备份目录
func (s *BackupService) BackupDir() string {
	return s.backupDir
}

// DeleteBackup 删除备份
func (s *BackupService) DeleteBackup(ctx context.Context, filename string) error {
	backupPath := filepath.Join(s.backupDir, filename)
	
	// 安全检查：确保文件在备份目录内
	absBackupDir, err := filepath.Abs(s.backupDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute backup directory: %w", err)
	}

	absBackupPath, err := filepath.Abs(backupPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute backup path: %w", err)
	}

	if !filepath.HasPrefix(absBackupPath, absBackupDir) {
		return fmt.Errorf("invalid backup path: %s", filename)
	}

	// 删除文件
	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	return nil
}

// isBackupFile 检查是否是备份文件
func isBackupFile(filename string) bool {
	// 检查 .tar.gz 扩展名（需要先检查，因为 filepath.Ext 只返回最后一个扩展名）
	if strings.HasSuffix(filename, ".tar.gz") {
		return true
	}
	// 检查其他扩展名
	ext := filepath.Ext(filename)
	return ext == ".sql" || ext == ".gz" || strings.HasPrefix(filename, "backup_")
}

// detectDatabaseType 检测数据库类型
func detectDatabaseType(filename string) string {
	if contains(filename, "postgres") {
		return "postgres"
	}
	if contains(filename, "sqlite") {
		return "sqlite"
	}
	return "unknown"
}

// contains 检查字符串是否包含子字符串
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

