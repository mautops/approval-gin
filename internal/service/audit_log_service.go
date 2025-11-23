package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-gin/internal/repository"
)

// AuditLogService 审计日志服务
type AuditLogService interface {
	RecordAction(ctx context.Context, userID string, action string, resourceType string, resourceID string, details interface{}) error
}

// auditLogService 审计日志服务实现
type auditLogService struct {
	auditRepo repository.AuditLogRepository
}

// NewAuditLogService 创建审计日志服务
func NewAuditLogService(auditRepo repository.AuditLogRepository) AuditLogService {
	return &auditLogService{
		auditRepo: auditRepo,
	}
}

// RecordAction 记录操作审计日志
func (s *auditLogService) RecordAction(
	ctx context.Context,
	userID string,
	action string,
	resourceType string,
	resourceID string,
	details interface{},
) error {
	// 序列化详情
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return err
	}

	// 获取请求信息
	requestID := ""
	if req := ctx.Value("request_id"); req != nil {
		requestID = req.(string)
	}

	ip := ""
	if req := ctx.Value("ip"); req != nil {
		ip = req.(string)
	}

	userAgent := ""
	if req := ctx.Value("user_agent"); req != nil {
		userAgent = req.(string)
	}

	// 创建审计日志
	auditLog := &model.AuditLogModel{
		ID:           uuid.New().String(),
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:  resourceID,
		RequestID:    requestID,
		IP:           ip,
		UserAgent:    userAgent,
		Details:      detailsJSON,
		CreatedAt:    time.Now(),
	}

	return s.auditRepo.Save(auditLog)
}

// GetClientIP 从 context 获取客户端 IP
func GetClientIP(ctx context.Context) string {
	if req := ctx.Value("ip"); req != nil {
		return req.(string)
	}
	return ""
}

// GetUserAgent 从 context 获取 User Agent
func GetUserAgent(ctx context.Context) string {
	if req := ctx.Value("user_agent"); req != nil {
		return req.(string)
	}
	return ""
}

