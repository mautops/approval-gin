package metrics

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/gorm"
)

var (
	// API 请求计数器
	apiRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_requests_total",
			Help: "Total number of API requests",
		},
		[]string{"method", "path", "status"},
	)

	// API 请求响应时间
	apiRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "api_request_duration_seconds",
			Help:    "API request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// 任务创建数
	tasksCreatedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "tasks_created_total",
			Help: "Total number of tasks created",
		},
	)

	// 审批操作数
	approvalsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "approvals_total",
			Help: "Total number of approval operations",
		},
		[]string{"action"}, // approve, reject, etc.
	)

	// 数据库连接数
	databaseConnectionsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "database_connections_active",
			Help: "Number of active database connections",
		},
	)

	databaseConnectionsIdle = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "database_connections_idle",
			Help: "Number of idle database connections",
		},
	)

	databaseConnectionsMax = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "database_connections_max",
			Help: "Maximum number of database connections",
		},
	)

	// 任务状态分布
	tasksByState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tasks_by_state",
			Help: "Number of tasks by state",
		},
		[]string{"state"},
	)
)

var (
	once sync.Once
)

func init() {
	// 注册指标
	prometheus.MustRegister(apiRequestsTotal)
	prometheus.MustRegister(apiRequestDuration)
	prometheus.MustRegister(tasksCreatedTotal)
	prometheus.MustRegister(approvalsTotal)
	prometheus.MustRegister(databaseConnectionsActive)
	prometheus.MustRegister(databaseConnectionsIdle)
	prometheus.MustRegister(databaseConnectionsMax)
	prometheus.MustRegister(tasksByState)

	// 注册 Go 运行时指标（只注册一次）
	once.Do(func() {
		// 尝试注册 Go 运行时指标，如果已注册则忽略错误
		_ = prometheus.Register(prometheus.NewGoCollector())
		_ = prometheus.Register(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	})
}

// Handler 返回 Prometheus 指标处理器
func Handler() http.Handler {
	return promhttp.Handler()
}

// RecordAPIRequest 记录 API 请求
func RecordAPIRequest(method, path string, status int, duration float64) {
	statusText := http.StatusText(status)
	if statusText == "" {
		statusText = fmt.Sprintf("%d", status)
	}
	apiRequestsTotal.WithLabelValues(method, path, statusText).Inc()
	apiRequestDuration.WithLabelValues(method, path).Observe(duration)
}

// RecordTaskCreated 记录任务创建
func RecordTaskCreated() {
	tasksCreatedTotal.Inc()
}

// RecordApproval 记录审批操作
func RecordApproval(action string) {
	approvalsTotal.WithLabelValues(action).Inc()
}

// UpdateDatabaseConnections 更新数据库连接数指标
func UpdateDatabaseConnections(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	stats := sqlDB.Stats()
	databaseConnectionsActive.Set(float64(stats.OpenConnections - stats.Idle))
	databaseConnectionsIdle.Set(float64(stats.Idle))
	databaseConnectionsMax.Set(float64(stats.MaxOpenConnections))

	return nil
}

// UpdateTasksByState 更新任务状态分布指标
func UpdateTasksByState(state string, count float64) {
	tasksByState.WithLabelValues(state).Set(count)
}


