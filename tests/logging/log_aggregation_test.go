package logging_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/api"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestLogAggregation_JSONFormat 测试日志 JSON 格式（用于日志聚合）
func TestLogAggregation_JSONFormat(t *testing.T) {
	logger := api.NewLogger()
	
	var buf strings.Builder
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
	})
	
	logger.WithFields(map[string]interface{}{
		"level":   "info",
		"message": "test message",
		"timestamp": time.Now().Format(time.RFC3339),
	}).Info("test")
	
	output := buf.String()
	assert.NotEmpty(t, output)
	
	// 验证 JSON 格式
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(output), &logEntry)
	assert.NoError(t, err, "log should be valid JSON")
	
	// 验证必需字段
	assert.Contains(t, logEntry, "level")
	assert.Contains(t, logEntry, "msg")
	assert.Contains(t, logEntry, "time")
}

// TestLogAggregation_FileOutput 测试日志文件输出
func TestLogAggregation_FileOutput(t *testing.T) {
	// 创建临时日志文件
	logFile := filepath.Join(t.TempDir(), "test.log")
	
	logger := api.NewLogger()
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	assert.NoError(t, err)
	defer file.Close()
	
	logger.SetOutput(file)
	logger.Info("test log message")
	
	// 验证日志文件存在且有内容
	content, err := os.ReadFile(logFile)
	assert.NoError(t, err)
	assert.NotEmpty(t, content)
	assert.Contains(t, string(content), "test log message")
}

// TestLogAggregation_StructuredFields 测试结构化日志字段（用于日志聚合）
func TestLogAggregation_StructuredFields(t *testing.T) {
	logger := api.NewLogger()
	
	var buf strings.Builder
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
	})
	
	// 记录结构化日志
	logger.WithFields(map[string]interface{}{
		"service":     "approval-gin",
		"version":     "1.0.0",
		"environment": "test",
		"request_id":  "req-123",
		"user_id":     "user-456",
		"action":      "create_task",
		"resource":    "task",
		"resource_id": "task-789",
	}).Info("task created")
	
	output := buf.String()
	assert.NotEmpty(t, output)
	
	// 验证 JSON 格式和字段
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(output), &logEntry)
	assert.NoError(t, err)
	
	// 验证结构化字段
	assert.Equal(t, "approval-gin", logEntry["service"])
	assert.Equal(t, "1.0.0", logEntry["version"])
	assert.Equal(t, "test", logEntry["environment"])
	assert.Equal(t, "req-123", logEntry["request_id"])
	assert.Equal(t, "user-456", logEntry["user_id"])
	assert.Equal(t, "create_task", logEntry["action"])
}

// TestLogAggregation_TimestampFormat 测试时间戳格式（ISO 8601）
func TestLogAggregation_TimestampFormat(t *testing.T) {
	logger := api.NewLogger()
	
	var buf strings.Builder
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
	})
	
	logger.Info("test message")
	
	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(output), &logEntry)
	assert.NoError(t, err)
	
	// 验证时间戳格式（ISO 8601）
	timestamp, ok := logEntry["time"].(string)
	assert.True(t, ok, "timestamp should be string")
	
	_, err = time.Parse(time.RFC3339, timestamp)
	assert.NoError(t, err, "timestamp should be in RFC3339 format")
}

// TestLogAggregation_LogLevels 测试不同日志级别
func TestLogAggregation_LogLevels(t *testing.T) {
	logger := api.NewLogger()
	
	var buf strings.Builder
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
	})
	
	// 测试不同日志级别
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")
	
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	
	// 验证不同级别的日志
	hasInfo := false
	hasWarn := false
	hasError := false
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &logEntry); err == nil {
			level := logEntry["level"].(string)
			switch level {
			case "info":
				hasInfo = true
			case "warning":
				hasWarn = true
			case "error":
				hasError = true
			}
		}
	}
	
	assert.True(t, hasInfo, "should have info log")
	assert.True(t, hasWarn, "should have warn log")
	assert.True(t, hasError, "should have error log")
}

