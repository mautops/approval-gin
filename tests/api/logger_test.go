package api_test

import (
	"bytes"
	"testing"

	"github.com/mautops/approval-gin/internal/api"
	"github.com/stretchr/testify/assert"
)

// TestLogger_Initialization 测试日志初始化
func TestLogger_Initialization(t *testing.T) {
	logger := api.NewLogger()
	assert.NotNil(t, logger, "logger should be initialized")
}

// TestLogger_LogLevel 测试日志级别
func TestLogger_LogLevel(t *testing.T) {
	logger := api.NewLogger()
	
	// 测试不同日志级别
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	
	logger.Info("test info message")
	logger.Warn("test warn message")
	logger.Error("test error message")
	
	// 验证日志输出不为空
	assert.NotEmpty(t, buf.String(), "logger should output messages")
}

// TestLogger_StructuredLogging 测试结构化日志
func TestLogger_StructuredLogging(t *testing.T) {
	logger := api.NewLogger()
	
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	
	// 测试结构化日志（带字段）
	logger.WithField("key", "value").Info("structured log message")
	
	// 验证日志输出不为空
	output := buf.String()
	assert.NotEmpty(t, output, "logger should output messages")
	assert.Contains(t, output, "structured log message", "log should contain message")
}

