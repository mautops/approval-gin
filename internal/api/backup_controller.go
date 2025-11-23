package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/service"
)

// BackupController 备份控制器
type BackupController struct {
	backupService *service.BackupService
}

// NewBackupController 创建备份控制器
func NewBackupController(backupService *service.BackupService) *BackupController {
	return &BackupController{
		backupService: backupService,
	}
}

// CreateBackup 创建备份
// @Summary      创建数据备份
// @Description  创建数据库备份文件
// @Tags         系统管理
// @Accept       json
// @Produce      json
// @Success      200  {object}  Response{data=service.BackupInfo}
// @Failure      500  {object}  ErrorResponse
// @Router       /backups [post]
// @Security    BearerAuth
func (c *BackupController) CreateBackup(ctx *gin.Context) {
	backupPath, err := c.backupService.CreateBackup(ctx.Request.Context())
	if err != nil {
		Error(ctx, http.StatusInternalServerError, "failed to create backup", err.Error())
		return
	}

	// 获取备份信息
	backups, err := c.backupService.ListBackups(ctx.Request.Context())
	if err != nil {
		Error(ctx, http.StatusInternalServerError, "failed to list backups", err.Error())
		return
	}

	// 找到刚创建的备份
	var backupInfo *service.BackupInfo
	for _, b := range backups {
		if b.Path == backupPath {
			backupInfo = &b
			break
		}
	}

	if backupInfo == nil {
		Error(ctx, http.StatusInternalServerError, "backup created but not found", "")
		return
	}

	Success(ctx, backupInfo)
}

// ListBackups 列出所有备份
// @Summary      列出所有备份
// @Description  获取所有备份文件列表
// @Tags         系统管理
// @Accept       json
// @Produce      json
// @Success      200  {object}  Response{data=[]service.BackupInfo}
// @Failure      500  {object}  ErrorResponse
// @Router       /backups [get]
// @Security    BearerAuth
func (c *BackupController) ListBackups(ctx *gin.Context) {
	backups, err := c.backupService.ListBackups(ctx.Request.Context())
	if err != nil {
		Error(ctx, http.StatusInternalServerError, "failed to list backups", err.Error())
		return
	}

	Success(ctx, backups)
}

// RestoreBackup 恢复备份
// @Summary      恢复数据备份
// @Description  从备份文件恢复数据库
// @Tags         系统管理
// @Accept       json
// @Produce      json
// @Param        filename path string true "备份文件名"
// @Success      200  {object}  Response
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /backups/{filename}/restore [post]
// @Security    BearerAuth
func (c *BackupController) RestoreBackup(ctx *gin.Context) {
	filename := ctx.Param("filename")
	if filename == "" {
		Error(ctx, http.StatusBadRequest, "invalid filename", "filename is required")
		return
	}

	// 获取备份文件路径
	backups, err := c.backupService.ListBackups(ctx.Request.Context())
	if err != nil {
		Error(ctx, http.StatusInternalServerError, "failed to list backups", err.Error())
		return
	}

	var backupPath string
	for _, b := range backups {
		if b.Filename == filename {
			backupPath = b.Path
			break
		}
	}

	if backupPath == "" {
		Error(ctx, http.StatusNotFound, "backup not found", "")
		return
	}

	if err := c.backupService.RestoreBackup(ctx.Request.Context(), backupPath); err != nil {
		Error(ctx, http.StatusInternalServerError, "failed to restore backup", err.Error())
		return
	}

	Success(ctx, nil)
}

// DeleteBackup 删除备份
// @Summary      删除备份
// @Description  删除指定的备份文件
// @Tags         系统管理
// @Accept       json
// @Produce      json
// @Param        filename path string true "备份文件名"
// @Success      200  {object}  Response
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /backups/{filename} [delete]
// @Security    BearerAuth
func (c *BackupController) DeleteBackup(ctx *gin.Context) {
	filename := ctx.Param("filename")
	if filename == "" {
		Error(ctx, http.StatusBadRequest, "invalid filename", "filename is required")
		return
	}

	if err := c.backupService.DeleteBackup(ctx.Request.Context(), filename); err != nil {
		Error(ctx, http.StatusInternalServerError, "failed to delete backup", err.Error())
		return
	}

	Success(ctx, nil)
}

