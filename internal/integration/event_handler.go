package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-gin/internal/repository"
	"github.com/mautops/approval-kit/pkg/event"
	"github.com/mautops/approval-kit/pkg/template"
	"gorm.io/gorm"
)

// dbEventHandler 基于数据库的事件处理器
// 实现 approval-kit 的 EventHandler 接口
type dbEventHandler struct {
	db          *gorm.DB
	eventRepo   repository.EventRepository
	templateMgr template.TemplateManager
	httpClient  *http.Client
	queue       chan *event.Event
	workers     int
	stop        chan struct{}
}

// NewEventHandler 创建事件处理器
func NewEventHandler(db *gorm.DB, workers int) event.EventHandler {
	if workers <= 0 {
		workers = 1
	}

	handler := &dbEventHandler{
		db:          db,
		eventRepo:   repository.NewEventRepository(db),
		templateMgr: NewTemplateManager(db),
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		queue:       make(chan *event.Event, 1000),
		workers:     workers,
		stop:        make(chan struct{}),
	}

	// 启动 worker goroutines
	for i := 0; i < workers; i++ {
		go handler.worker()
	}

	return handler
}

// Handle 处理事件
func (h *dbEventHandler) Handle(evt *event.Event) error {
	// 1. 持久化事件到数据库
	eventData, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	eventModel := &model.EventModel{
		ID:        uuid.New().String(),
		TaskID:    evt.Task.ID,
		Type:      string(evt.Type),
		Data:      eventData,
		Status:    "pending",
		RetryCount: 0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.eventRepo.Save(eventModel); err != nil {
		return fmt.Errorf("failed to save event: %w", err)
	}

	// 2. 异步推送到 Webhook
	select {
	case h.queue <- evt:
		// 事件成功入队
	default:
		// 队列满时记录日志,不阻塞
		// TODO: 使用日志库记录
		fmt.Printf("event queue full, dropping event: type=%q, task=%q\n", evt.Type, evt.Task.ID)
	}

	return nil
}

// worker 事件处理 worker
func (h *dbEventHandler) worker() {
	for {
		select {
		case evt := <-h.queue:
			h.pushToWebhook(evt)
		case <-h.stop:
			return
		}
	}
}

// pushToWebhook 推送到 Webhook
func (h *dbEventHandler) pushToWebhook(evt *event.Event) {
	// 1. 查找事件模型
	var eventModel model.EventModel
	err := h.db.Where("task_id = ? AND type = ?", evt.Task.ID, string(evt.Type)).
		Order("created_at DESC").
		First(&eventModel).Error
	if err != nil {
		// 如果找不到事件模型，记录错误但不重试
		fmt.Printf("failed to find event model: %v\n", err)
		return
	}

	// 2. 获取模板配置（包含 Webhook 配置）
	// 如果 Task 中没有 TemplateID，尝试从事件数据中获取
	templateID := evt.Task.TemplateID
	if templateID == "" {
		// 如果 TemplateID 为空，记录错误但不重试
		fmt.Printf("template ID is empty in event task: %v\n", evt.Task)
		// 标记为成功（无需推送，因为没有模板配置）
		eventModel.Status = "success"
		eventModel.UpdatedAt = time.Now()
		h.eventRepo.Save(&eventModel)
		return
	}

	template, err := h.templateMgr.Get(templateID, 0)
	if err != nil {
		// 如果找不到模板，记录错误但不重试
		fmt.Printf("failed to get template: %v\n", err)
		// 标记为成功（无需推送，因为找不到模板）
		eventModel.Status = "success"
		eventModel.UpdatedAt = time.Now()
		h.eventRepo.Save(&eventModel)
		return
	}

	// 3. 如果没有 Webhook 配置，直接返回
	if template.Config == nil || len(template.Config.Webhooks) == 0 {
		// 没有 Webhook 配置，标记为成功（无需推送）
		eventModel.Status = "success"
		eventModel.UpdatedAt = time.Now()
		h.eventRepo.Save(&eventModel)
		return
	}

	// 4. 尝试推送到所有 Webhook（使用重试机制）
	maxRetries := 3
	backoff := time.Second

	for i := 0; i < maxRetries; i++ {
		success := true
		for _, webhook := range template.Config.Webhooks {
			if err := h.sendWebhookRequest(webhook, evt); err != nil {
				success = false
				fmt.Printf("failed to send webhook request: %v\n", err)
			}
		}

		if success {
			// 推送成功，更新事件状态
			eventModel.Status = "success"
			eventModel.UpdatedAt = time.Now()
			h.eventRepo.Save(&eventModel)
			return
		}

		// 推送失败，增加重试计数
		eventModel.RetryCount++
		eventModel.UpdatedAt = time.Now()
		h.eventRepo.Save(&eventModel)

		// 如果还有重试机会，等待后重试
		if i < maxRetries-1 {
			time.Sleep(backoff)
			backoff *= 2 // 指数退避
		}
	}

	// 所有重试都失败，更新事件状态为失败
	eventModel.Status = "failed"
	eventModel.UpdatedAt = time.Now()
	h.eventRepo.Save(&eventModel)
}

// sendWebhookRequest 发送 Webhook 请求
func (h *dbEventHandler) sendWebhookRequest(webhook *template.WebhookConfig, evt *event.Event) error {
	// 1. 序列化事件数据
	eventData, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// 2. 创建 HTTP 请求
	method := webhook.Method
	if method == "" {
		method = "POST"
	}

	req, err := http.NewRequest(method, webhook.URL, bytes.NewBuffer(eventData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 3. 设置请求头
	req.Header.Set("Content-Type", "application/json")
	for key, value := range webhook.Headers {
		req.Header.Set(key, value)
	}

	// 4. 设置认证信息
	if webhook.Auth != nil {
		switch webhook.Auth.Type {
		case "bearer":
			req.Header.Set("Authorization", "Bearer "+webhook.Auth.Token)
		case "basic":
			req.SetBasicAuth(webhook.Auth.Key, webhook.Auth.Token)
		case "header":
			req.Header.Set(webhook.Auth.Key, webhook.Auth.Token)
		}
	}

	// 5. 发送请求
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 6. 检查响应状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status code: %d", resp.StatusCode)
	}

	return nil
}

// Stop 停止事件处理器
func (h *dbEventHandler) Stop() {
	close(h.stop)
}

