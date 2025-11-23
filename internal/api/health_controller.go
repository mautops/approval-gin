package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/auth"
	"gorm.io/gorm"
)

// HealthController 健康检查控制器
type HealthController struct {
	db       *gorm.DB
	fgaClient *auth.OpenFGAClient
}

// NewHealthController 创建健康检查控制器
func NewHealthController(db *gorm.DB, fgaClient *auth.OpenFGAClient) *HealthController {
	return &HealthController{
		db:       db,
		fgaClient: fgaClient,
	}
}

// Check 健康检查
func (c *HealthController) Check(ctx *gin.Context) {
	status := "healthy"
	checks := make(map[string]string)
	
	// 检查数据库连接
	if c.db != nil {
		if err := c.checkDatabase(ctx.Request.Context()); err != nil {
			status = "unhealthy"
			checks["database"] = "unhealthy: " + err.Error()
		} else {
			checks["database"] = "healthy"
		}
	} else {
		checks["database"] = "not configured"
	}
	
	// 检查 OpenFGA 连接
	if c.fgaClient != nil {
		if err := c.checkOpenFGA(ctx.Request.Context()); err != nil {
			status = "unhealthy"
			checks["openfga"] = "unhealthy: " + err.Error()
		} else {
			checks["openfga"] = "healthy"
		}
	} else {
		checks["openfga"] = "not configured"
	}
	
	httpStatus := http.StatusOK
	if status == "unhealthy" {
		httpStatus = http.StatusServiceUnavailable
	}
	
	ctx.JSON(httpStatus, gin.H{
		"status":    status,
		"timestamp": time.Now().Unix(),
		"checks":    checks,
	})
}

// checkDatabase 检查数据库连接
func (c *HealthController) checkDatabase(ctx context.Context) error {
	sqlDB, err := c.db.DB()
	if err != nil {
		return err
	}
	
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	return sqlDB.PingContext(ctx)
}

// checkOpenFGA 检查 OpenFGA 连接
func (c *HealthController) checkOpenFGA(ctx context.Context) error {
	// 通过执行一个简单的权限检查来测试 OpenFGA 连接
	// 使用一个不存在的资源，这样不会影响实际权限
	_, err := c.fgaClient.CheckPermission(ctx, "health-check-user", "viewer", "template", "health-check-resource")
	// 如果错误是权限相关的（资源不存在），说明连接正常
	// 如果是网络错误，说明连接失败
	if err != nil {
		// 检查是否是网络错误
		if err.Error() == "failed to check permission: context deadline exceeded" ||
			err.Error() == "failed to check permission: connection refused" {
			return err
		}
		// 其他错误（如权限错误）说明连接正常，只是资源不存在
		return nil
	}
	return nil
}

