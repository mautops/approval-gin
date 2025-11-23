package metrics

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// Collector 指标收集器
type Collector struct {
	db        *gorm.DB
	interval  time.Duration
	ctx       context.Context
	cancel    context.CancelFunc
	done      chan struct{}
}

// NewCollector 创建指标收集器
func NewCollector(db *gorm.DB, interval time.Duration) *Collector {
	ctx, cancel := context.WithCancel(context.Background())
	return &Collector{
		db:       db,
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
		done:     make(chan struct{}),
	}
}

// Start 启动指标收集器
func (c *Collector) Start() {
	go c.collect()
}

// Stop 停止指标收集器
func (c *Collector) Stop() {
	c.cancel()
	<-c.done
}

// collect 定期收集指标
func (c *Collector) collect() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	defer close(c.done)

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			// 更新数据库连接数指标
			_ = UpdateDatabaseConnections(c.db)
		}
	}
}

