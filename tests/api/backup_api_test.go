package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/mautops/approval-gin/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBackupAPI_CreateBackup(t *testing.T) {
	// 创建测试数据库
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	// 创建备份服务
	backupService := service.NewBackupService(db, t.TempDir())

	// 创建备份控制器
	backupController := api.NewBackupController(backupService)

	router := gin.New()
	router.POST("/api/v1/backups", backupController.CreateBackup)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backups", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response api.Response
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Code)
}

func TestBackupAPI_ListBackups(t *testing.T) {
	// 创建测试数据库
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	// 创建备份服务
	backupService := service.NewBackupService(db, t.TempDir())

	// 创建备份
	_, err = backupService.CreateBackup(context.Background())
	require.NoError(t, err)

	// 创建备份控制器
	backupController := api.NewBackupController(backupService)

	router := gin.New()
	router.GET("/api/v1/backups", backupController.ListBackups)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/backups", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response api.Response
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Code)
}

