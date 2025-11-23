package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/stretchr/testify/assert"
)

func TestAPIVersionMiddleware_URLPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 注册版本中间件
	router.Use(api.VersionMiddleware())

	// 测试路由
	router.GET("/api/v1/test", func(c *gin.Context) {
		version := c.GetString("api_version")
		c.JSON(200, gin.H{"version": version})
	})

	router.GET("/api/v2/test", func(c *gin.Context) {
		version := c.GetString("api_version")
		c.JSON(200, gin.H{"version": version})
	})

	// 测试 v1
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "v1", resp["version"])

	// 测试 v2
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v2/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "v2", resp["version"])
}

func TestAPIVersionMiddleware_Header(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 注册版本中间件
	router.Use(api.VersionMiddleware())

	// 测试路由
	router.GET("/api/test", func(c *gin.Context) {
		version := c.GetString("api_version")
		c.JSON(200, gin.H{"version": version})
	})

	// 测试通过请求头指定版本
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	req.Header.Set("API-Version", "v2")
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "v2", resp["version"])
}

func TestAPIVersionMiddleware_DefaultVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 注册版本中间件
	router.Use(api.VersionMiddleware())

	// 测试路由
	router.GET("/api/test", func(c *gin.Context) {
		version := c.GetString("api_version")
		c.JSON(200, gin.H{"version": version})
	})

	// 测试默认版本（无版本信息时）
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "v1", resp["version"]) // 默认版本应该是 v1
}

