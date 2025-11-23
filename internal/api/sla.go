package api

import (
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SLAConfig SLA 配置
type SLAConfig struct {
	TaskCreationMaxTime    time.Duration // 任务创建最大响应时间
	TaskApprovalMaxTime    time.Duration // 任务审批最大响应时间
	TemplateQueryMaxTime   time.Duration // 模板查询最大响应时间
	TaskQueryMaxTime       time.Duration // 任务查询最大响应时间
}

// DefaultSLAConfig 返回默认 SLA 配置
func DefaultSLAConfig() *SLAConfig {
	return &SLAConfig{
		TaskCreationMaxTime:  1 * time.Second,
		TaskApprovalMaxTime:  2 * time.Second,
		TemplateQueryMaxTime: 500 * time.Millisecond,
		TaskQueryMaxTime:     500 * time.Millisecond,
	}
}

// getOperation 从请求路径和方法获取操作类型
func getOperation(c *gin.Context) string {
	method := c.Request.Method
	path := c.Request.URL.Path

	// 根据路径和方法判断操作类型
	if path == "/api/v1/tasks" && method == "POST" {
		return "task_creation"
	}
	if path == "/api/v1/tasks" && method == "GET" {
		return "task_query"
	}
	if strings.Contains(path, "/approve") || strings.Contains(path, "/reject") {
		return "task_approval"
	}
	if strings.Contains(path, "/templates") && method == "GET" {
		return "template_query"
	}

	return "unknown"
}

// CheckSLA 检查 SLA
func CheckSLA(operation string, duration time.Duration, config *SLAConfig) bool {
	switch operation {
	case "task_creation":
		return duration <= config.TaskCreationMaxTime
	case "task_approval":
		return duration <= config.TaskApprovalMaxTime
	case "template_query":
		return duration <= config.TemplateQueryMaxTime
	case "task_query":
		return duration <= config.TaskQueryMaxTime
	default:
		return true // 未知操作不检查 SLA
	}
}

// SLAMonitorMiddleware SLA 监控中间件
func SLAMonitorMiddleware(config *SLAConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultSLAConfig()
	}

	return func(c *gin.Context) {
		start := time.Now()
		operation := getOperation(c)

		c.Next()

		duration := time.Since(start)
		if !CheckSLA(operation, duration, config) {
			// 记录 SLA 违反
			c.Header("X-SLA-Violation", "true")
			c.Header("X-SLA-Operation", operation)
			c.Header("X-SLA-Duration", duration.String())
			c.Header("X-SLA-Expected", getExpectedDuration(operation, config).String())
		}
	}
}

// getExpectedDuration 获取期望的响应时间
func getExpectedDuration(operation string, config *SLAConfig) time.Duration {
	switch operation {
	case "task_creation":
		return config.TaskCreationMaxTime
	case "task_approval":
		return config.TaskApprovalMaxTime
	case "template_query":
		return config.TemplateQueryMaxTime
	case "task_query":
		return config.TaskQueryMaxTime
	default:
		return 0
	}
}

// SLAViolation SLA 违反记录
type SLAViolation struct {
	Operation string
	Duration  time.Duration
	Expected  time.Duration
	Timestamp time.Time
	Path      string
	Method    string
}

// SLAAlertManager SLA 告警管理器
type SLAAlertManager struct {
	violations     map[string][]SLAViolation
	thresholds     map[string]int
	alertCallbacks []func(string, []SLAViolation)
	mu             sync.RWMutex
}

// NewSLAAlertManager 创建 SLA 告警管理器
func NewSLAAlertManager() *SLAAlertManager {
	return &SLAAlertManager{
		violations:     make(map[string][]SLAViolation),
		thresholds:     make(map[string]int),
		alertCallbacks: make([]func(string, []SLAViolation), 0),
	}
}

// RecordViolation 记录 SLA 违反
func (m *SLAAlertManager) RecordViolation(operation string, violation SLAViolation) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.violations[operation] = append(m.violations[operation], violation)

	// 检查是否达到告警阈值
	threshold := m.thresholds[operation]
	if threshold > 0 && len(m.violations[operation]) >= threshold {
		// 触发告警
		m.triggerAlert(operation)
	}
}

// SetAlertThreshold 设置告警阈值
func (m *SLAAlertManager) SetAlertThreshold(operation string, threshold int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.thresholds[operation] = threshold
}

// OnAlert 注册告警回调
func (m *SLAAlertManager) OnAlert(callback func(string, []SLAViolation)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alertCallbacks = append(m.alertCallbacks, callback)
}

// GetViolations 获取违反记录
func (m *SLAAlertManager) GetViolations(operation string) []SLAViolation {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.violations[operation]
}

// triggerAlert 触发告警
func (m *SLAAlertManager) triggerAlert(operation string) {
	violations := m.violations[operation]
	for _, callback := range m.alertCallbacks {
		callback(operation, violations)
	}
}

// SLAMonitorMiddlewareWithAlert SLA 监控中间件（带告警）
func SLAMonitorMiddlewareWithAlert(config *SLAConfig, alertManager *SLAAlertManager) gin.HandlerFunc {
	if config == nil {
		config = DefaultSLAConfig()
	}

	return func(c *gin.Context) {
		start := time.Now()
		operation := getOperation(c)

		c.Next()

		duration := time.Since(start)
		if !CheckSLA(operation, duration, config) {
			// 记录 SLA 违反
			violation := SLAViolation{
				Operation: operation,
				Duration:  duration,
				Expected:  getExpectedDuration(operation, config),
				Timestamp: time.Now(),
				Path:      c.Request.URL.Path,
				Method:    c.Request.Method,
			}

			if alertManager != nil {
				alertManager.RecordViolation(operation, violation)
			}

			// 设置响应头
			c.Header("X-SLA-Violation", "true")
			c.Header("X-SLA-Operation", operation)
			c.Header("X-SLA-Duration", duration.String())
			c.Header("X-SLA-Expected", violation.Expected.String())
		}
	}
}

