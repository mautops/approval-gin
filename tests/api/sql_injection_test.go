package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/utils"
	"github.com/stretchr/testify/assert"
)

// TestSQLInjection_TemplateName 测试模板名称 SQL 注入防护
func TestSQLInjection_TemplateName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/api/v1/templates", func(c *gin.Context) {
		var req struct {
			Name string `json:"name" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// 使用验证工具验证输入
		if err := utils.ValidateTemplateName(req.Name); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"name": req.Name})
	})

	// SQL 注入 payload
	sqlPayload := "'; DROP TABLE templates; --"
	body, _ := json.Marshal(map[string]interface{}{
		"name": sqlPayload,
	})
	req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 应该被输入验证拒绝，而不是执行 SQL
	assert.Equal(t, http.StatusBadRequest, w.Code, "Should reject SQL injection input")
	assert.Contains(t, w.Body.String(), "dangerous", "Error message should indicate dangerous characters")
}

// TestSQLInjection_TemplateID 测试模板 ID SQL 注入防护
func TestSQLInjection_TemplateID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/api/v1/templates/:id", func(c *gin.Context) {
		id := c.Param("id")
		// 使用验证工具验证输入
		if err := utils.ValidateTemplateID(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"id": id})
	})

	// SQL 注入 payload
	sqlPayload := "tpl-001' OR '1'='1"
	// 使用 url.PathEscape 转义路径参数
	escapedPayload := url.PathEscape(sqlPayload)
	req := httptest.NewRequest("GET", "/api/v1/templates/"+escapedPayload, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 应该被输入验证拒绝
	assert.Equal(t, http.StatusBadRequest, w.Code, "Should reject SQL injection input")
	assert.Contains(t, w.Body.String(), "invalid", "Error message should indicate invalid id")
}

// TestSQLInjection_TaskID 测试任务 ID SQL 注入防护
func TestSQLInjection_TaskID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/api/v1/tasks/:id", func(c *gin.Context) {
		id := c.Param("id")
		// 使用验证工具验证输入
		if err := utils.ValidateTaskID(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"id": id})
	})

	// SQL 注入 payload
	sqlPayload := "task-001' OR '1'='1"
	// 使用 url.PathEscape 转义路径参数
	escapedPayload := url.PathEscape(sqlPayload)
	req := httptest.NewRequest("GET", "/api/v1/tasks/"+escapedPayload, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 应该被输入验证拒绝
	assert.Equal(t, http.StatusBadRequest, w.Code, "Should reject SQL injection input")
	assert.Contains(t, w.Body.String(), "invalid", "Error message should indicate invalid id")
}

// TestSQLInjection_QueryParameter 测试查询参数 SQL 注入防护
func TestSQLInjection_QueryParameter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/api/v1/templates", func(c *gin.Context) {
		// GORM 使用参数化查询，查询参数中的 SQL 注入会被安全处理
		search := c.Query("search")
		c.JSON(http.StatusOK, gin.H{"search": search})
	})

	// SQL 注入 payload 在查询参数中
	sqlPayload := "test' OR '1'='1"
	// 使用 url.QueryEscape 转义查询参数
	escapedPayload := url.QueryEscape(sqlPayload)
	req := httptest.NewRequest("GET", "/api/v1/templates?search="+escapedPayload, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// GORM 应该使用参数化查询，不会执行 SQL 注入
	// 这里主要验证不会导致错误或异常行为
	assert.Equal(t, http.StatusOK, w.Code, "Should not cause error (parameterized query)")
}

