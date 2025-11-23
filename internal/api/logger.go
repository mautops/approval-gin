package api

import (
	"io"
	"os"
	"path/filepath"

	"github.com/mautops/approval-gin/internal/config"
	"github.com/sirupsen/logrus"
)

var defaultLogger *logrus.Logger

// JSONFormatter JSON 格式化器（用于测试）
type JSONFormatter = logrus.JSONFormatter

// NewLogger 创建新的日志记录器
func NewLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "time",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "msg",
		},
	})
	logger.SetLevel(logrus.InfoLevel)
	logger.SetOutput(os.Stdout)
	return logger
}

// NewLoggerFromConfig 根据配置创建日志记录器
func NewLoggerFromConfig(cfg *config.LogConfig) (*logrus.Logger, error) {
	logger := logrus.New()
	
	// 设置日志格式
	if cfg.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "time",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "msg",
			},
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
			FullTimestamp:   true,
		})
	}
	
	// 设置日志级别
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)
	
	// 设置日志输出
	var writers []io.Writer
	if cfg.Output == "stdout" || cfg.Output == "both" {
		writers = append(writers, os.Stdout)
	}
	if cfg.Output == "file" || cfg.Output == "both" {
		// 创建日志目录
		logDir := "logs"
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, err
		}
		
		// 打开日志文件
		logFile := filepath.Join(logDir, "approval-gin.log")
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		writers = append(writers, file)
	}
	
	if len(writers) == 0 {
		writers = []io.Writer{os.Stdout}
	}
	
	logger.SetOutput(io.MultiWriter(writers...))
	
	// 添加默认字段（用于日志聚合）
	logger.AddHook(&defaultFieldsHook{
		fields: map[string]interface{}{
			"service": "approval-gin",
		},
	})
	
	return logger, nil
}

// defaultFieldsHook 添加默认字段的 Hook
type defaultFieldsHook struct {
	fields map[string]interface{}
}

func (h *defaultFieldsHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *defaultFieldsHook) Fire(entry *logrus.Entry) error {
	for k, v := range h.fields {
		entry.Data[k] = v
	}
	return nil
}

// GetLogger 获取默认日志记录器
func GetLogger() *logrus.Logger {
	if defaultLogger == nil {
		defaultLogger = NewLogger()
	}
	return defaultLogger
}

// SetLoggerOutput 设置日志输出
func SetLoggerOutput(w io.Writer) {
	logger := GetLogger()
	logger.SetOutput(w)
}

// SetLoggerLevel 设置日志级别
func SetLoggerLevel(level logrus.Level) {
	logger := GetLogger()
	logger.SetLevel(level)
}


