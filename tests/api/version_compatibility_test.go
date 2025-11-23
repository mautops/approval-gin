package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/stretchr/testify/assert"
)

func TestAPIVersionCompatibility_BackwardCompatible(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 注册版本中间件
	router.Use(api.VersionMiddleware())

	// v1 API（旧版本）
	router.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"version": "v1", "data": "old format"})
	})

	// v2 API（新版本，向后兼容）
	router.GET("/api/v2/test", func(c *gin.Context) {
		// v2 应该能够处理 v1 格式的请求
		version := api.GetAPIVersion(c)
		c.JSON(200, gin.H{"version": version, "data": "new format", "compatible": true})
	})

	// 测试 v1 仍然可用
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "v1", resp["version"])

	// 测试 v2 可用
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v2/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "v2", resp["version"])
}

func TestAPIVersionCompatibility_DeprecatedVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 注册版本中间件
	router.Use(api.VersionMiddleware())

	// 标记 v1 为已废弃，但仍可用
	router.GET("/api/v1/deprecated", func(c *gin.Context) {
		c.Header("X-API-Deprecated", "true")
		c.Header("X-API-Deprecation-Date", "2025-12-31")
		c.Header("X-API-Sunset-Date", "2026-12-31")
		c.JSON(200, gin.H{"message": "This API version is deprecated"})
	})

	// 测试废弃版本仍然可用，但返回警告头
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/deprecated", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "true", w.Header().Get("X-API-Deprecated"))
	assert.Equal(t, "2025-12-31", w.Header().Get("X-API-Deprecation-Date"))
	assert.Equal(t, "2026-12-31", w.Header().Get("X-API-Sunset-Date"))
}

func TestAPIVersionCompatibility_VersionMigration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 注册废弃版本信息（包含迁移路径）
	api.RegisterDeprecatedVersion(api.DeprecatedVersionInfo{
		Version:        "v1",
		DeprecationDate: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		SunsetDate:     time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		MigrationPath:  "/api/v2/migrate",
	})

	// 注册版本中间件
	router.Use(api.VersionMiddleware())

	// v1 API
	router.GET("/api/v1/migrate", func(c *gin.Context) {
		c.JSON(200, gin.H{"version": "v1", "message": "Please migrate to v2"})
	})

	// v2 API（迁移目标）
	router.GET("/api/v2/migrate", func(c *gin.Context) {
		c.JSON(200, gin.H{"version": "v2", "message": "Current version"})
	})

	// 测试 v1 返回迁移路径
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/migrate", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "/api/v2/migrate", w.Header().Get("X-API-Migration-Path"))
}

