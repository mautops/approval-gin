package api

import (
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// DeprecatedVersionInfo 废弃版本信息
type DeprecatedVersionInfo struct {
	Version        string
	DeprecationDate time.Time
	SunsetDate     time.Time
	MigrationPath  string
}

// 版本兼容性配置
var (
	deprecatedVersions = make(map[string]DeprecatedVersionInfo)
	deprecatedMu       sync.RWMutex
)

// VersionMiddleware API 版本中间件
// 支持两种版本控制方式：
// 1. URL 路径版本控制: /api/v1/..., /api/v2/...
// 2. 请求头版本控制: API-Version: v1, API-Version: v2
func VersionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		version := "v1" // 默认版本

		// 方式 1: 从 URL 路径提取版本
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/v") {
			parts := strings.Split(path, "/")
			for i, part := range parts {
				if part == "api" && i+1 < len(parts) {
					nextPart := parts[i+1]
					if strings.HasPrefix(nextPart, "v") && len(nextPart) > 1 {
						version = nextPart
						break
					}
				}
			}
		}

		// 方式 2: 从请求头获取版本（优先级高于 URL 路径）
		if headerVersion := c.GetHeader("API-Version"); headerVersion != "" {
			version = headerVersion
		}

		// 检查版本是否已废弃
		deprecatedMu.RLock()
		deprecationInfo, isDeprecated := deprecatedVersions[version]
		deprecatedMu.RUnlock()

		if isDeprecated {
			c.Header("X-API-Deprecated", "true")
			c.Header("X-API-Deprecation-Date", deprecationInfo.DeprecationDate.Format("2006-01-02"))
			c.Header("X-API-Sunset-Date", deprecationInfo.SunsetDate.Format("2006-01-02"))
			if deprecationInfo.MigrationPath != "" {
				c.Header("X-API-Migration-Path", deprecationInfo.MigrationPath)
			}
		}

		// 将版本信息存储到上下文
		c.Set("api_version", version)

		c.Next()
	}
}

// GetAPIVersion 从上下文获取 API 版本
func GetAPIVersion(c *gin.Context) string {
	if version, exists := c.Get("api_version"); exists {
		if v, ok := version.(string); ok {
			return v
		}
	}
	return "v1" // 默认版本
}

// RegisterDeprecatedVersion 注册废弃版本信息
func RegisterDeprecatedVersion(info DeprecatedVersionInfo) {
	deprecatedMu.Lock()
	defer deprecatedMu.Unlock()
	deprecatedVersions[info.Version] = info
}
